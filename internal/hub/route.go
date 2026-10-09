package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
)

const maxRouteErrorBody = 2 << 20

type bufferedRouteResponse struct {
	status   int
	header   http.Header
	body     []byte
	provider string
}

type providerLoad struct {
	inflight atomic.Int64
}

func (l *providerLoad) acquire() { l.inflight.Add(1) }
func (l *providerLoad) release() { l.inflight.Add(-1) }
func (l *providerLoad) current() int64 { return l.inflight.Load() }

type fallbackBalancer struct {
	mu sync.Mutex
}

func requestedModel(raw []byte) (string, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("body must be a JSON object: %w", err)
	}
	modelRaw, ok := payload["model"]
	if !ok {
		return "", fmt.Errorf("model is required")
	}
	var model string
	if err := json.Unmarshal(modelRaw, &model); err != nil || strings.TrimSpace(model) == "" {
		return "", fmt.Errorf("model must be a non-empty string")
	}
	return model, nil
}

func rewriteModelTo(raw []byte, model string) ([]byte, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("body must be a JSON object: %w", err)
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		return nil, err
	}
	payload["model"] = encoded
	return json.Marshal(payload)
}

func (h *Hub) routeConfig(id string) (config.RouteConfig, bool) {
	if !h.cfg.Routing.RouteAliasesEnabled {
		return config.RouteConfig{}, false
	}
	for _, route := range h.cfg.Routes {
		if route.ID == id {
			return route, true
		}
	}
	return config.RouteConfig{}, false
}

func routeRetryableStatus(status int) bool {
	switch status {
	case http.StatusPaymentRequired,
		http.StatusNotFound,
		http.StatusRequestTimeout,
		http.StatusTooEarly,
		http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func (h *Hub) fallbackProviderIDs(primary string) []string {
	seen := map[string]struct{}{primary: {}}
	out := []string{primary}
	configured := h.cfg.Routing.SameModelFallback.Providers
	if len(configured) == 0 {
		for _, p := range h.cfg.Providers {
			if _, ok := seen[p.ID]; ok {
				continue
			}
			seen[p.ID] = struct{}{}
			out = append(out, p.ID)
		}
		return out
	}
	for _, id := range configured {
		id = strings.TrimSpace(id)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// usableFallbackProviders only considers an alternate if its live model
// catalog advertises the exact upstream ID. Sending a nominally equal ID to
// an unrelated service is not a valid same-model fallback. The originally
// requested provider remains eligible without an extra model-list round trip.
func (h *Hub) usableFallbackProviders(ctx context.Context, primary, model string) []*runtimeProvider {
	ids := h.fallbackProviderIDs(primary)
	out := make([]*runtimeProvider, 0, len(ids))
	for _, id := range ids {
		p, ok := h.providers[id]
		if !ok || !p.cfg.Enabled || !p.hasRuntime() {
			continue
		}
		if !providerUsableForRouting(id, p.cfg.Kind, p.snapshot(), p.accounts.snapshot()) {
			continue
		}
		if id != primary {
			probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			models, err := h.fetchModels(probeCtx, p.cfg)
			cancel()
			if err != nil {
				continue // Fail closed: unverified model identity must not route.
			}
			found := false
			for _, entry := range models {
				if upstream, _ := entry["x_upstream_id"].(string); upstream == model {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

func (h *Hub) handleSameModelFallback(w http.ResponseWriter, r *http.Request, raw []byte, primary, model string) {
	if p:=h.providers[primary];p!=nil && p.cfg.Enabled && h.onDemand(p) {
		release,err:=h.acquireOnDemand(r.Context(),p)
		if err!=nil {writeError(w,http.StatusServiceUnavailable,"provider_start_unavailable",err.Error());return}
		defer release()
	}
	mode := strings.ToLower(strings.TrimSpace(h.cfg.Routing.SameModelFallback.Mode))
	if mode == "" || mode == "balanced" || mode == "parallel" {
		h.handleSameModelBalanced(w, r, raw, primary, model)
		return
	}
	h.handleSameModelSequential(w, r, raw, primary, model)
}

func (h *Hub) handleSameModelSequential(w http.ResponseWriter, r *http.Request, raw []byte, primary, model string) {
	rewritten, err := rewriteModelTo(raw, model)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
		return
	}
	providers := h.usableFallbackProviders(r.Context(), primary, model)
	if len(providers) == 0 {
		writeError(w, http.StatusServiceUnavailable, "provider_unavailable", "no usable provider for same-model fallback")
		return
	}

	failures := make([]string, 0, len(providers))
	var last *bufferedRouteResponse
	attempts := 0
	for _, p := range providers {
		p.load.acquire()
		attempts++
		resp, reqErr := h.doProviderRequest(r, p.cfg, rewritten)
		if reqErr != nil {
			p.load.release()
			failures = append(failures, p.cfg.ID+": transport error")
			continue
		}
		if !routeRetryableStatus(resp.StatusCode) {
			h.writeFallbackResponse(w, resp, primary, p.cfg.ID, "same-model-sequential", attempts)
			p.load.release()
			return
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxRouteErrorBody))
		_ = resp.Body.Close()
		p.load.release()
		if readErr == nil {
			last = &bufferedRouteResponse{status: resp.StatusCode, header: resp.Header.Clone(), body: body, provider: p.cfg.ID}
		}
		failures = append(failures, fmt.Sprintf("%s: HTTP %d", p.cfg.ID, resp.StatusCode))
	}

	h.writeFallbackFailure(w, last, primary, model, "same-model-sequential", attempts, failures)
}

func (h *Hub) pickBalancedProvider(providers []*runtimeProvider, attempted map[string]struct{}, primary string) *runtimeProvider {
	h.balance.mu.Lock()
	defer h.balance.mu.Unlock()

	var best *runtimeProvider
	for _, p := range providers {
		if _, done := attempted[p.cfg.ID]; done {
			continue
		}
		if best == nil {
			best = p
			continue
		}
		pLoad, bestLoad := p.load.current(), best.load.current()
		if pLoad < bestLoad {
			best = p
			continue
		}
		if pLoad == bestLoad && p.cfg.ID == primary && best.cfg.ID != primary {
			best = p
		}
	}
	if best != nil {
		best.load.acquire()
	}
	return best
}

func (h *Hub) handleSameModelBalanced(w http.ResponseWriter, r *http.Request, raw []byte, primary, model string) {
	rewritten, err := rewriteModelTo(raw, model)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
		return
	}
	providers := h.usableFallbackProviders(r.Context(), primary, model)
	if len(providers) == 0 {
		writeError(w, http.StatusServiceUnavailable, "provider_unavailable", "no usable provider for same-model balanced routing")
		return
	}

	attempted := make(map[string]struct{}, len(providers))
	failures := make([]string, 0, len(providers))
	var last *bufferedRouteResponse
	attempts := 0

	for len(attempted) < len(providers) {
		p := h.pickBalancedProvider(providers, attempted, primary)
		if p == nil {
			break
		}
		attempted[p.cfg.ID] = struct{}{}
		attempts++

		resp, reqErr := h.doProviderRequest(r, p.cfg, rewritten)
		if reqErr != nil {
			p.load.release()
			failures = append(failures, p.cfg.ID+": transport error")
			continue
		}
		if !routeRetryableStatus(resp.StatusCode) {
			h.writeFallbackResponse(w, resp, primary, p.cfg.ID, "same-model-balanced", attempts)
			p.load.release()
			return
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxRouteErrorBody))
		_ = resp.Body.Close()
		p.load.release()
		if readErr == nil {
			last = &bufferedRouteResponse{status: resp.StatusCode, header: resp.Header.Clone(), body: body, provider: p.cfg.ID}
		}
		failures = append(failures, fmt.Sprintf("%s: HTTP %d", p.cfg.ID, resp.StatusCode))
	}

	h.writeFallbackFailure(w, last, primary, model, "same-model-balanced", attempts, failures)
}

func (h *Hub) writeFallbackResponse(w http.ResponseWriter, resp *http.Response, primary, provider, mode string, attempts int) {
	defer resp.Body.Close()
	w.Header().Set("X-AhB-Provider", provider)
	w.Header().Set("X-AhB-Routing", mode)
	w.Header().Set("X-AhB-Attempts", fmt.Sprintf("%d", attempts))
	if provider != primary {
		w.Header().Set("X-AhB-Fallback", "same-model")
	}
	copyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	copyStreaming(w, resp.Body)
}

func (h *Hub) writeFallbackFailure(
	w http.ResponseWriter,
	last *bufferedRouteResponse,
	primary, model, mode string,
	attempts int,
	failures []string,
) {
	if last != nil {
		w.Header().Set("X-AhB-Provider", last.provider)
		w.Header().Set("X-AhB-Routing", mode)
		w.Header().Set("X-AhB-Attempts", fmt.Sprintf("%d", attempts))
		if last.provider != primary {
			w.Header().Set("X-AhB-Fallback", "same-model")
		}
		copyResponseHeaders(w.Header(), last.header)
		w.WriteHeader(last.status)
		_, _ = w.Write(last.body)
		return
	}
	writeError(w, http.StatusBadGateway, "same_model_fallback_failed",
		fmt.Sprintf("model %q failed across providers: %s", model, strings.Join(failures, "; ")))
}

func (h *Hub) handleRouteProxy(w http.ResponseWriter, r *http.Request, raw []byte, routeID string) {
	route, ok := h.routeConfig(routeID)
	if !ok {
		writeError(w, http.StatusBadRequest, "route_alias_disabled_or_unknown", fmt.Sprintf("route %q is not enabled or configured", routeID))
		return
	}

	failures := make([]string, 0, len(route.Targets))
	var last *bufferedRouteResponse
	attempted := false
	for _, target := range route.Targets {
		providerID, upstreamModel, err := splitModel(target)
		if err != nil {
			failures = append(failures, target+": invalid target")
			continue
		}
		p, ok := h.providers[providerID]
		if !ok || !p.cfg.Enabled || !p.hasRuntime() {
			failures = append(failures, target+": provider disabled")
			continue
		}
		release, startErr := h.acquireOnDemand(r.Context(), p)
		if startErr != nil {
			failures = append(failures, target+": startup unavailable")
			continue
		}
		if !providerUsableForRouting(providerID, p.cfg.Kind, p.snapshot(), p.accounts.snapshot()) {
			release()
			failures = append(failures, target+": provider unavailable")
			continue
		}
		rewritten, err := rewriteModelTo(raw, upstreamModel)
		if err != nil {
			release()
			writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
			return
		}
		resp, err := h.doProviderRequest(r, p.cfg, rewritten)
		attempted = true
		if err != nil {
			release()
			failures = append(failures, target+": transport error")
			continue
		}
		if !routeRetryableStatus(resp.StatusCode) {
			w.Header().Set("X-AhB-Route", routeID)
			w.Header().Set("X-AhB-Provider", providerID)
			copyResponseHeaders(w.Header(), resp.Header)
			w.WriteHeader(resp.StatusCode)
			copyStreaming(w, resp.Body)
			_ = resp.Body.Close()
			release()
			return
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxRouteErrorBody))
		_ = resp.Body.Close()
		release()
		if readErr == nil {
			last = &bufferedRouteResponse{status: resp.StatusCode, header: resp.Header.Clone(), body: body, provider: providerID}
		}
		failures = append(failures, fmt.Sprintf("%s: HTTP %d", target, resp.StatusCode))
	}
	if last != nil {
		w.Header().Set("X-AhB-Route", routeID)
		w.Header().Set("X-AhB-Provider", last.provider)
		copyResponseHeaders(w.Header(), last.header)
		w.WriteHeader(last.status)
		_, _ = w.Write(last.body)
		return
	}
	status, kind := http.StatusServiceUnavailable, "route_unavailable"
	if attempted {
		status, kind = http.StatusBadGateway, "route_failed"
	}
	message := fmt.Sprintf("route %q has no usable targets", routeID)
	if len(failures) > 0 {
		message += ": " + strings.Join(failures, "; ")
	}
	writeError(w, status, kind, message)
}

func (h *Hub) doProviderRequest(r *http.Request, cfg config.ProviderConfig, body []byte) (*http.Response, error) {
	return h.doProviderRequestContext(r.Context(), r, cfg, body)
}

func (h *Hub) doProviderRequestContext(ctx context.Context, r *http.Request, cfg config.ProviderConfig, body []byte) (*http.Response, error) {
	upstreamPath := r.URL.Path
	if cfg.APIPathPrefix != "" && strings.HasPrefix(upstreamPath, "/v1/") {
		upstreamPath = cfg.APIPathPrefix + strings.TrimPrefix(upstreamPath, "/v1")
	}
	endpoint, err := joinURL(cfg.BaseURL, upstreamPath)
	if err != nil {
		return nil, err
	}
	if r.URL.RawQuery != "" {
		endpoint += "?" + r.URL.RawQuery
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	copyRequestHeaders(req.Header, r.Header)
	applyProviderHeaders(req.Header, cfg.Headers)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if p := h.providers[cfg.ID]; p != nil {
		if err != nil {
			p.lastRequest.record(0, true)
		} else {
			// Status is recorded when headers arrive. A 200 streaming response
			// could still fail before [DONE]; this is NOT proof of successful tokens.
			p.lastRequest.record(resp.StatusCode, false)
		}
	}
	return resp, err
}
