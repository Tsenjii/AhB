package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Tsenjii/AhB/internal/config"
)

const maxRouteErrorBody = 2 << 20

type bufferedRouteResponse struct {
	status   int
	header   http.Header
	body     []byte
	provider string
}

type providerAttempt struct {
	provider string
	resp     *http.Response
	err      error
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

func (h *Hub) usableFallbackProviders(primary string) []*runtimeProvider {
	ids := h.fallbackProviderIDs(primary)
	out := make([]*runtimeProvider, 0, len(ids))
	for _, id := range ids {
		p, ok := h.providers[id]
		if !ok || !p.cfg.Enabled || p.sup == nil {
			continue
		}
		if !providerUsableForRouting(id, p.sup.Snapshot(), p.accounts.snapshot()) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (h *Hub) handleSameModelFallback(w http.ResponseWriter, r *http.Request, raw []byte, primary, model string) {
	if strings.EqualFold(h.cfg.Routing.SameModelFallback.Mode, "parallel") {
		h.handleSameModelParallel(w, r, raw, primary, model)
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
	providers := h.usableFallbackProviders(primary)
	if len(providers) == 0 {
		writeError(w, http.StatusServiceUnavailable, "provider_unavailable", "no usable provider for same-model fallback")
		return
	}

	failures := make([]string, 0, len(providers))
	var last *bufferedRouteResponse
	for _, p := range providers {
		resp, err := h.doProviderRequest(r, p.cfg, rewritten)
		if err != nil {
			failures = append(failures, p.cfg.ID+": transport error")
			continue
		}
		if !routeRetryableStatus(resp.StatusCode) {
			h.writeFallbackResponse(w, resp, primary, p.cfg.ID, false)
			return
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxRouteErrorBody))
		_ = resp.Body.Close()
		if readErr == nil {
			last = &bufferedRouteResponse{status: resp.StatusCode, header: resp.Header.Clone(), body: body, provider: p.cfg.ID}
		}
		failures = append(failures, fmt.Sprintf("%s: HTTP %d", p.cfg.ID, resp.StatusCode))
	}

	if last != nil {
		w.Header().Set("X-AhB-Provider", last.provider)
		w.Header().Set("X-AhB-Fallback", "same-model")
		copyResponseHeaders(w.Header(), last.header)
		w.WriteHeader(last.status)
		_, _ = w.Write(last.body)
		return
	}
	writeError(w, http.StatusBadGateway, "same_model_fallback_failed",
		fmt.Sprintf("model %q failed across providers: %s", model, strings.Join(failures, "; ")))
}

func (h *Hub) handleSameModelParallel(w http.ResponseWriter, r *http.Request, raw []byte, primary, model string) {
	rewritten, err := rewriteModelTo(raw, model)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
		return
	}
	providers := h.usableFallbackProviders(primary)
	if len(providers) == 0 {
		writeError(w, http.StatusServiceUnavailable, "provider_unavailable", "no usable provider for same-model parallel race")
		return
	}

	results := make(chan providerAttempt, len(providers))
	cancels := make([]context.CancelFunc, 0, len(providers))
	for _, p := range providers {
		ctx, cancel := context.WithCancel(r.Context())
		cancels = append(cancels, cancel)
		go func(p *runtimeProvider, ctx context.Context) {
			resp, err := h.doProviderRequestContext(ctx, r, p.cfg, rewritten)
			results <- providerAttempt{provider: p.cfg.ID, resp: resp, err: err}
		}(p, ctx)
	}

	cancelAll := func() {
		for _, cancel := range cancels {
			cancel()
		}
	}
	failures := make([]string, 0, len(providers))
	var last *bufferedRouteResponse
	received := 0
	for received < len(providers) {
		attempt := <-results
		received++
		if attempt.err != nil {
			failures = append(failures, attempt.provider+": transport error")
			continue
		}
		if !routeRetryableStatus(attempt.resp.StatusCode) {
			cancelAll()
			remaining := len(providers) - received
			if remaining > 0 {
				go drainLoserResponses(results, remaining)
			}
			h.writeFallbackResponse(w, attempt.resp, primary, attempt.provider, true)
			return
		}
		body, readErr := io.ReadAll(io.LimitReader(attempt.resp.Body, maxRouteErrorBody))
		_ = attempt.resp.Body.Close()
		if readErr == nil {
			last = &bufferedRouteResponse{
				status: attempt.resp.StatusCode, header: attempt.resp.Header.Clone(),
				body: body, provider: attempt.provider,
			}
		}
		failures = append(failures, fmt.Sprintf("%s: HTTP %d", attempt.provider, attempt.resp.StatusCode))
	}
	cancelAll()

	if last != nil {
		w.Header().Set("X-AhB-Provider", last.provider)
		w.Header().Set("X-AhB-Fallback", "same-model-parallel")
		copyResponseHeaders(w.Header(), last.header)
		w.WriteHeader(last.status)
		_, _ = w.Write(last.body)
		return
	}
	writeError(w, http.StatusBadGateway, "same_model_parallel_failed",
		fmt.Sprintf("model %q failed across providers: %s", model, strings.Join(failures, "; ")))
}

func drainLoserResponses(results <-chan providerAttempt, remaining int) {
	for i := 0; i < remaining; i++ {
		attempt := <-results
		if attempt.resp != nil {
			_ = attempt.resp.Body.Close()
		}
	}
}

func (h *Hub) writeFallbackResponse(w http.ResponseWriter, resp *http.Response, primary, provider string, parallel bool) {
	defer resp.Body.Close()
	w.Header().Set("X-AhB-Provider", provider)
	if provider != primary {
		if parallel {
			w.Header().Set("X-AhB-Fallback", "same-model-parallel")
		} else {
			w.Header().Set("X-AhB-Fallback", "same-model")
		}
	} else if parallel {
		w.Header().Set("X-AhB-Fallback", "same-model-parallel")
	}
	copyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	copyStreaming(w, resp.Body)
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
		if !ok || !p.cfg.Enabled || p.sup == nil {
			failures = append(failures, target+": provider disabled")
			continue
		}
		if !providerUsableForRouting(providerID, p.sup.Snapshot(), p.accounts.snapshot()) {
			failures = append(failures, target+": provider unavailable")
			continue
		}
		rewritten, err := rewriteModelTo(raw, upstreamModel)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
			return
		}
		resp, err := h.doProviderRequest(r, p.cfg, rewritten)
		attempted = true
		if err != nil {
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
			return
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxRouteErrorBody))
		_ = resp.Body.Close()
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
	endpoint, err := joinURL(cfg.BaseURL, r.URL.Path)
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
	return h.client.Do(req)
}
