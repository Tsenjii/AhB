package hub

import (
	"bytes"
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

func (h *Hub) handleRouteProxy(w http.ResponseWriter, r *http.Request, raw []byte, routeID string) {
	route, ok := h.routeConfig(routeID)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown_route", fmt.Sprintf("route %q is not configured", routeID))
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
		snap := p.sup.Snapshot()
		if !providerUsableForRouting(providerID, snap, p.accounts.snapshot()) {
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
		if readErr != nil {
			failures = append(failures, fmt.Sprintf("%s: HTTP %d", target, resp.StatusCode))
			continue
		}
		last = &bufferedRouteResponse{
			status:   resp.StatusCode,
			header:   resp.Header.Clone(),
			body:     body,
			provider: providerID,
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

	status := http.StatusServiceUnavailable
	kind := "route_unavailable"
	if attempted {
		status = http.StatusBadGateway
		kind = "route_failed"
	}
	message := fmt.Sprintf("route %q has no usable targets", routeID)
	if len(failures) > 0 {
		message += ": " + strings.Join(failures, "; ")
	}
	writeError(w, status, kind, message)
}

func (h *Hub) doProviderRequest(r *http.Request, cfg config.ProviderConfig, body []byte) (*http.Response, error) {
	endpoint, err := joinURL(cfg.BaseURL, r.URL.Path)
	if err != nil {
		return nil, err
	}
	if r.URL.RawQuery != "" {
		endpoint += "?" + r.URL.RawQuery
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	copyRequestHeaders(req.Header, r.Header)
	applyProviderHeaders(req.Header, cfg.Headers)
	req.Header.Set("Content-Type", "application/json")
	return h.client.Do(req)
}
