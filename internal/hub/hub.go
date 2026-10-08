package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/Tsenjii/android-ai-hub/internal/config"
	"github.com/Tsenjii/android-ai-hub/internal/provider"
	"github.com/Tsenjii/android-ai-hub/internal/sidecar"
)

const maxRequestBody = 32 << 20

type Hub struct {
	cfg       config.Config
	providers map[string]*runtimeProvider
	client    *http.Client
}

type runtimeProvider struct {
	cfg config.ProviderConfig
	sup *sidecar.Supervisor
}

type providerView struct {
	ID          string         `json:"id"`
	DisplayName string         `json:"display_name"`
	Description string         `json:"description,omitempty"`
	Enabled     bool           `json:"enabled"`
	Kind        string         `json:"kind"`
	UIURL       string         `json:"ui_url,omitempty"`
	DocsURL     string         `json:"docs_url,omitempty"`
	State       provider.State `json:"state"`
	PID         int            `json:"pid,omitempty"`
	RSSBytes    int64          `json:"rss_bytes,omitempty"`
	Restarts    int            `json:"restarts"`
	LastError   string         `json:"last_error,omitempty"`
}

func New(cfg config.Config) *Hub {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          128,
		MaxIdleConnsPerHost:   32,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 0,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	h := &Hub{
		cfg:       cfg,
		providers: make(map[string]*runtimeProvider),
		client:    &http.Client{Transport: transport},
	}
	for _, p := range cfg.Providers {
		rp := &runtimeProvider{cfg: p}
		if p.Enabled && p.Kind == "sidecar" {
			rp.sup = sidecar.New(p)
		}
		h.providers[p.ID] = rp
	}
	return h
}

func (h *Hub) Start(ctx context.Context) {
	for _, p := range h.providers {
		if p.sup != nil {
			go p.sup.Run(ctx)
		}
	}
}

func (h *Hub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleRoot)
	mux.HandleFunc("/ui", h.handleUI)
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/api/providers", h.handleProviders)
	mux.HandleFunc("/api/runtime", h.handleRuntime)
	mux.HandleFunc("/v1/models", h.handleModels)
	mux.HandleFunc("/v1/chat/completions", h.handleProxy)
	mux.HandleFunc("/v1/responses", h.handleProxy)
	mux.HandleFunc("/v1/messages", h.handleProxy)
	mux.HandleFunc("/v1/messages/count_tokens", h.handleProxy)
	mux.HandleFunc("/v1/systemone", h.handleProxy)
	return requestLogMiddleware(mux)
}

func (h *Hub) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", http.StatusTemporaryRedirect)
}

func (h *Hub) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"version":   "0.2.0-dev",
		"providers": h.providerViews(),
	})
}

func (h *Hub) handleRuntime(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, collectRuntimeStats())
}

func (h *Hub) handleProviders(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": h.providerViews(),
	})
}

func (h *Hub) providerViews() []providerView {
	out := make([]providerView, 0, len(h.providers))
	for id, p := range h.providers {
		view := providerView{
			ID:          id,
			DisplayName: p.cfg.DisplayName,
			Description: p.cfg.Description,
			Enabled:     p.cfg.Enabled,
			Kind:        p.cfg.Kind,
			UIURL:       p.cfg.UIURL,
			DocsURL:     p.cfg.DocsURL,
			State:       provider.StateDisabled,
		}
		if p.cfg.Enabled {
			if p.sup == nil {
				view.State = provider.StateDegraded
				view.LastError = "provider enabled without a supervisor"
			} else {
				snap := p.sup.Snapshot()
				view.State = snap.State
				view.PID = snap.PID
				view.RSSBytes = snap.RSSBytes
				view.Restarts = snap.Restarts
				view.LastError = snap.LastError
			}
		}
		out = append(out, view)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (h *Hub) handleModels(w http.ResponseWriter, r *http.Request) {
	type modelList struct {
		Object   string            `json:"object"`
		Data     []map[string]any  `json:"data"`
		Warnings map[string]string `json:"x_provider_warnings,omitempty"`
	}

	result := modelList{
		Object:   "list",
		Data:     []map[string]any{},
		Warnings: map[string]string{},
	}
	for id, p := range h.providers {
		if !p.cfg.Enabled || p.sup == nil {
			continue
		}
		if p.sup.Snapshot().State != provider.StateHealthy {
			result.Warnings[id] = "provider not healthy"
			continue
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		models, err := h.fetchModels(ctx, p.cfg)
		cancel()
		if err != nil {
			result.Warnings[id] = err.Error()
			continue
		}
		result.Data = append(result.Data, models...)
	}
	sort.Slice(result.Data, func(i, j int) bool {
		a, _ := result.Data[i]["id"].(string)
		b, _ := result.Data[j]["id"].(string)
		return a < b
	})
	if len(result.Warnings) == 0 {
		result.Warnings = nil
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Hub) fetchModels(ctx context.Context, cfg config.ProviderConfig) ([]map[string]any, error) {
	endpoint, err := joinURL(cfg.BaseURL, cfg.ModelsPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	applyProviderHeaders(req.Header, cfg.Headers)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("models returned HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}
	out := make([]map[string]any, 0, len(doc.Data))
	for _, m := range doc.Data {
		upstreamID, ok := m["id"].(string)
		if !ok || strings.TrimSpace(upstreamID) == "" {
			continue
		}
		copyModel := make(map[string]any, len(m)+3)
		for k, v := range m {
			copyModel[k] = v
		}
		copyModel["id"] = cfg.ID + "/" + upstreamID
		copyModel["x_provider"] = cfg.ID
		copyModel["x_provider_name"] = cfg.DisplayName
		copyModel["x_upstream_id"] = upstreamID
		out = append(out, copyModel)
	}
	return out, nil
}

func (h *Hub) handleProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}
	raw, err := readLimited(r.Body, maxRequestBody)
	if err != nil {
		var tooLarge *bodyTooLargeError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	providerID, _, rewritten, err := rewriteModelBody(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
		return
	}
	p, ok := h.providers[providerID]
	if !ok || !p.cfg.Enabled || p.sup == nil {
		writeError(w, http.StatusBadRequest, "unknown_provider", "provider is not enabled")
		return
	}
	snap := p.sup.Snapshot()
	if snap.State != provider.StateHealthy {
		writeError(w, http.StatusServiceUnavailable, "provider_unavailable", fmt.Sprintf("%s is %s", providerID, snap.State))
		return
	}

	endpoint, err := joinURL(p.cfg.BaseURL, r.URL.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "routing_error", err.Error())
		return
	}
	if r.URL.RawQuery != "" {
		endpoint += "?" + r.URL.RawQuery
	}

	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, bytes.NewReader(rewritten))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "routing_error", err.Error())
		return
	}
	copyRequestHeaders(upstreamReq.Header, r.Header)
	applyProviderHeaders(upstreamReq.Header, p.cfg.Headers)
	upstreamReq.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(upstreamReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	copyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	copyStreaming(w, resp.Body)
}

func rewriteModelBody(raw []byte) (providerID, upstreamModel string, rewritten []byte, err error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", "", nil, fmt.Errorf("body must be a JSON object: %w", err)
	}
	modelRaw, ok := payload["model"]
	if !ok {
		return "", "", nil, fmt.Errorf("model is required")
	}
	var model string
	if err := json.Unmarshal(modelRaw, &model); err != nil || strings.TrimSpace(model) == "" {
		return "", "", nil, fmt.Errorf("model must be a non-empty string")
	}
	providerID, upstreamModel, err = splitModel(model)
	if err != nil {
		return "", "", nil, err
	}
	payload["model"], err = json.Marshal(upstreamModel)
	if err != nil {
		return "", "", nil, err
	}
	rewritten, err = json.Marshal(payload)
	if err != nil {
		return "", "", nil, err
	}
	return providerID, upstreamModel, rewritten, nil
}

func splitModel(model string) (string, string, error) {
	providerID, upstream, ok := strings.Cut(model, "/")
	if !ok || strings.TrimSpace(providerID) == "" || strings.TrimSpace(upstream) == "" {
		return "", "", fmt.Errorf("model must use provider/model format")
	}
	return providerID, upstream, nil
}

func readLimited(r io.Reader, max int64) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > max {
		return nil, &bodyTooLargeError{max: max}
	}
	return raw, nil
}

type bodyTooLargeError struct{ max int64 }

func (e *bodyTooLargeError) Error() string {
	return fmt.Sprintf("request body exceeds %d bytes", e.max)
}

func joinURL(base, path string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	return u.String(), nil
}

func copyRequestHeaders(dst, src http.Header) {
	for k, values := range src {
		if shouldDropRequestHeader(k) {
			continue
		}
		for _, v := range values {
			dst.Add(k, v)
		}
	}
}

func applyProviderHeaders(dst http.Header, headers map[string]string) {
	for k, v := range headers {
		dst.Set(k, v)
	}
}

func copyResponseHeaders(dst, src http.Header) {
	for k, values := range src {
		if isHopByHop(k) {
			continue
		}
		for _, v := range values {
			dst.Add(k, v)
		}
	}
}

func shouldDropRequestHeader(k string) bool {
	if isHopByHop(k) {
		return true
	}
	switch strings.ToLower(k) {
	case "authorization", "x-api-key", "content-length":
		return true
	default:
		return false
	}
}

func isHopByHop(k string) bool {
	switch strings.ToLower(k) {
	case "connection", "proxy-connection", "keep-alive", "proxy-authenticate",
		"proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func copyStreaming(w http.ResponseWriter, r io.Reader) {
	buf := make([]byte, 32*1024)
	flusher, _ := w.(http.Flusher)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, kind, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"type":    kind,
			"message": message,
		},
	})
}

func requestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}