package hub

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
	"github.com/Tsenjii/AhB/internal/provider"
	"github.com/Tsenjii/AhB/internal/sidecar"
)

const maxRequestBody = 32 << 20

type Hub struct {
	cfg       config.Config
	providers map[string]*runtimeProvider
	client    *http.Client
	balance   fallbackBalancer
	runWG     sync.WaitGroup
	controlToken string
	restartFn func() error
	updateFn func() error
	controlConfigPath string
	controlMu sync.Mutex
	controlPending bool
	copilotLoginMu sync.Mutex
	copilotLogin *copilotLoginSession
	demandMu sync.Mutex
	demandContext context.Context
}

type runtimeProvider struct {
	cfg      config.ProviderConfig
	sup      *sidecar.Supervisor
	external *externalProbeState
	accounts accountProbeState
	lastRequest requestResultState
	load     providerLoad
	demand *demandState
}

func (p *runtimeProvider) hasRuntime() bool {
	return p.sup != nil || p.external != nil
}

func (p *runtimeProvider) snapshot() sidecar.Snapshot {
	if p.sup != nil {
		return p.sup.Snapshot()
	}
	if p.external != nil {
		return p.external.snapshot()
	}
	return sidecar.Snapshot{ID: p.cfg.ID, State: provider.StateDisabled}
}

type providerView struct {
	ID             string         `json:"id"`
	DisplayName    string         `json:"display_name"`
	Description    string         `json:"description,omitempty"`
	Enabled        bool           `json:"enabled"`
	Kind           string         `json:"kind"`
	UIURL          string         `json:"ui_url,omitempty"`
	DocsURL        string         `json:"docs_url,omitempty"`
	State          provider.State `json:"state"`
	ProcessAlive   bool           `json:"process_alive"`
	ProviderReady  bool           `json:"provider_ready"`
	AccountUsable  *bool          `json:"account_usable"`
	AccountTotal   *int           `json:"account_total,omitempty"`
	AccountUsableCount *int       `json:"account_usable_count,omitempty"`
	ReportedStatus string         `json:"reported_status,omitempty"`
	PID            int            `json:"pid,omitempty"`
	RSSBytes       int64          `json:"rss_bytes,omitempty"`
	Restarts       int            `json:"restarts"`
	StartMode string `json:"start_mode,omitempty"`
	LastError      string         `json:"last_error,omitempty"`
	LastRequestStatus int         `json:"last_request_http_status,omitempty"`
	LastRequestAt string          `json:"last_request_at,omitempty"`
	LastRequestTransportError bool `json:"last_request_transport_error,omitempty"`
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
	// Ephemeral page-bound CSRF token. Never persist or log the token.
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err == nil {
		h.controlToken = hex.EncodeToString(nonce[:])
	}
	for _, p := range cfg.Providers {
		rp := &runtimeProvider{cfg: p}
		if p.Enabled {
			switch p.Kind {
			case "sidecar":
				rp.sup = sidecar.New(p)
			case "external":
				rp.external = newExternalProbe(p)
			}
		}
		h.providers[p.ID] = rp
	}
	return h
}

// Start launches the provider workers. The caller must cancel ctx and call
// Wait before exiting so child processes are reaped during clean shutdown.
func (h *Hub) Start(ctx context.Context) {
	h.demandMu.Lock()
	h.demandContext = ctx
	h.demandMu.Unlock()
	h.runWG.Add(1)
	go func(){defer h.runWG.Done(); h.reapDemand(ctx)}()
	for _, p := range h.providers {
		if p.sup != nil && !h.onDemand(p) {
			h.runWG.Add(1)
			go func(p *runtimeProvider) {
				defer h.runWG.Done()
				p.sup.Run(ctx)
			}(p)
		}
		if p.external != nil {
			h.runWG.Add(1)
			go func(p *runtimeProvider) {
				defer h.runWG.Done()
				h.runExternalProbe(ctx, p)
			}(p)
		}
		if p.hasRuntime() && (p.cfg.ID == "agent2api" || p.cfg.ID == "freebuff" || p.cfg.ID == "deepseek" || p.cfg.ID == "kiro") {
			h.runWG.Add(1)
			go func(p *runtimeProvider) {
				defer h.runWG.Done()
				h.pollProviderAccounts(ctx, p)
			}(p)
		}
	}
}

func (h *Hub) Wait() {
	h.runWG.Wait()
}

func (h *Hub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleRoot)
	mux.HandleFunc("/ui", h.handleUI)
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/api/providers", h.handleProviders)
	mux.HandleFunc("/api/runtime", h.handleRuntime)
	mux.HandleFunc("/api/control/restart", h.handleControlRestart)
	mux.HandleFunc("/api/control/update", h.handleControlUpdate)
	mux.HandleFunc("/api/control/provider/", h.handleControlProvider)
	mux.HandleFunc("/api/control/login/copilot", h.handleCopilotLogin)
	mux.HandleFunc("/api/control/wake/", h.wakeProvider)
	mux.HandleFunc("/v1/models", h.handleModels)
	mux.HandleFunc("/v1/chat/completions", h.handleProxy)
	mux.HandleFunc("/v1/completions", h.handleProxy)
	mux.HandleFunc("/v1/embeddings", h.handleProxy)
	mux.HandleFunc("/v1/images/generations", h.handleProxy)
	mux.HandleFunc("/v1/audio/speech", h.handleProxy)
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
			StartMode: p.cfg.StartMode,
			UIURL:       p.cfg.UIURL,
			DocsURL:     p.cfg.DocsURL,
			State:       provider.StateDisabled,
		}
		last := p.lastRequest.snapshot()
		if !last.At.IsZero() {
			view.LastRequestStatus = last.Status
			view.LastRequestAt = last.At.Format(time.RFC3339)
			view.LastRequestTransportError = last.TransportError
		}
		if p.cfg.Enabled {
			if !p.hasRuntime() {
				view.State = provider.StateDegraded
				view.LastError = "provider enabled without a runtime"
			} else {
				snap := p.snapshot()
				account := p.accounts.snapshot()
				assessment := assessProviderHealth(id, p.cfg.Kind, snap, account)
				view.State = assessment.State
				view.ProcessAlive = assessment.ProcessAlive
				view.ProviderReady = assessment.ProviderReady
				view.AccountUsable = assessment.AccountUsable
				if account.Known {
					total, usable := account.Total, account.Usable
					view.AccountTotal = &total
					view.AccountUsableCount = &usable
				}
				view.ReportedStatus = assessment.ReportedStatus
				view.PID = snap.PID
				view.RSSBytes = snap.RSSBytes
				view.Restarts = snap.Restarts
				view.LastError = snap.LastError
				if view.LastError == "" {
					view.LastError = assessment.Detail
				}
			}
		}
		out = append(out, view)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

type providerAssessment struct {
	State          provider.State
	ProcessAlive   bool
	ProviderReady  bool
	AccountUsable  *bool
	ReportedStatus string
	Detail         string
}

func boolPtr(v bool) *bool { return &v }

func assessProviderHealth(id, kind string, snap sidecar.Snapshot, account accountProbe) providerAssessment {
	external := kind == "external"
	a := providerAssessment{
		State: snap.State, ProcessAlive: !external && snap.PID > 0,
		ProviderReady: snap.State == provider.StateHealthy,
	}
	if !external && !a.ProcessAlive {
		a.ProviderReady = false
		return a
	}
	if external && snap.HealthHTTPStatus == 0 {
		a.ProviderReady = false
		return a
	}
	if snap.HealthHTTPStatus >= 200 && snap.HealthHTTPStatus < 300 {
		a.ProviderReady = true
	} else if snap.HealthHTTPStatus != 0 {
		a.ProviderReady = false
	}
	// Health endpoints sometimes return empty bodies or temporarily invalid JSON.
	// Account-backed providers must still be assessed using the independent
	// account probe, and must not become routable while that probe is unknown.
	var doc map[string]any
	_ = json.Unmarshal(snap.HealthBody, &doc)
	if status, _ := doc["status"].(string); status != "" {
		a.ReportedStatus = strings.ToLower(strings.TrimSpace(status))
	}
	if ready, ok := doc["ready"].(bool); ok {
		a.ProviderReady = ready
	}
	if okValue, ok := doc["ok"].(bool); ok && !okValue {
		a.ProviderReady = false
	}

	switch id {
	case "opencode":
		anonymous, total := false, 0
		if keys, ok := doc["keys"].(map[string]any); ok {
			anonymous, _ = keys["anonymous"].(bool)
			if n, ok := keys["total"].(float64); ok {
				total = int(n)
			}
		}
		usable := a.ProviderReady && (anonymous || total > 0)
		a.AccountUsable = boolPtr(usable)
	case "grok":
		if components, ok := doc["components"].(map[string]any); ok {
			readyAccount := false
			knownAccountState := false
			for _, key := range []string{"grok_build", "grok_web", "grok_console"} {
				component, _ := components[key].(map[string]any)
				state, _ := component["state"].(string)
				switch strings.ToLower(strings.TrimSpace(state)) {
				case "ready":
					readyAccount = true
					knownAccountState = true
				case "unavailable":
					knownAccountState = true
				}
			}
			if readyAccount {
				a.AccountUsable = boolPtr(true)
			} else if knownAccountState {
				a.AccountUsable = boolPtr(false)
				a.Detail = "no usable Grok accounts"
			}
		}
	case "kiro":
		if account.Known {
			usable := account.Usable > 0
			a.AccountUsable = boolPtr(usable)
			if !usable {
				a.Detail = "no usable Kiro accounts"
			}
		}
	case "deepseek":
		if account.Known {
			usable := account.Usable > 0
			a.AccountUsable = boolPtr(usable)
			if !usable {
				a.Detail = "no configured DeepSeek accounts"
			}
		}
	case "freebuff":
		if account.Known {
			usable := account.Usable > 0
			a.AccountUsable = boolPtr(usable)
			if !usable {
				a.Detail = "no usable FreeBuff accounts"
			}
		}
	case "agent2api":
		// /health is a legacy WorkBuddy-only summary in Agent2API v2.9.5.
		// The aggregate truth comes from /api/accounts, which includes every
		// built-in/custom provider and exposes enabled/credentials/chat support.
		if account.Known {
			usable := account.Usable > 0
			a.AccountUsable = boolPtr(usable)
			if !usable {
				a.Detail = "no usable Agent2API accounts"
			}
		} else if a.ReportedStatus == "degraded" {
			if reason, _ := doc["unavailableReason"].(string); strings.TrimSpace(reason) != "" {
				a.Detail = reason
			} else {
				a.Detail = "provider self-reported degraded"
			}
		}
	}

	if accountStatusRequired(id) && !account.Known {
		a.Detail = "account availability not yet verified"
		if account.LastError != "" {
			a.Detail = "account availability probe failed"
		}
	}

	if a.State == provider.StateHealthy {
		if accountStatusRequired(id) && !account.Known {
			a.State = provider.StateDegraded
		} else if !a.ProviderReady {
			a.State = provider.StateDegraded
		} else if a.AccountUsable != nil && !*a.AccountUsable {
			a.State = provider.StateDegraded
		} else if id == "agent2api" && !account.Known && a.ReportedStatus == "degraded" {
			a.State = provider.StateDegraded
		}
	}
	return a
}

// These backends require accounts and expose an account-health probe.
func accountStatusRequired(id string) bool {
	switch id {
	case "agent2api", "freebuff", "deepseek", "kiro":
		return true
	default:
		return false
	}
}

func providerUsableForRouting(id, kind string, snap sidecar.Snapshot, account accountProbe) bool {
	if snap.State != provider.StateHealthy {
		return false
	}
	a := assessProviderHealth(id, kind, snap, account)
	if a.State != provider.StateHealthy || !a.ProviderReady {
		return false
	}
	return a.AccountUsable == nil || *a.AccountUsable
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
	// Independent providers should not stall each other's model discovery.
	// Four concurrent probes cap phone resource usage while preserving the
	// existing per-provider 10s deadline, fail-closed readiness check, and
	// deterministic final sort. No stale models are returned on errors.
	type modelFetchResult struct {
		id string
		models []map[string]any
		err error
	}
	results := make(chan modelFetchResult, len(h.providers))
	slots := make(chan struct{}, 4)
	var probes sync.WaitGroup
	for id, p := range h.providers {
		if !p.cfg.Enabled || !p.hasRuntime() {
			continue
		}
		if h.onDemand(p) && p.snapshot().PID == 0 {
			result.Warnings[id] = "sleeping in 512 MB mode; wake from dashboard or send provider/model request"
			continue
		}
		if !providerUsableForRouting(id, p.cfg.Kind, p.snapshot(), p.accounts.snapshot()) {
			result.Warnings[id] = "provider not currently usable"
			continue
		}
		probes.Add(1)
		go func(id string, p *runtimeProvider) {
			defer probes.Done()
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-r.Context().Done():
				results <- modelFetchResult{id: id, err: r.Context().Err()}
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			models, err := h.fetchModels(ctx, p.cfg)
			cancel()
			results <- modelFetchResult{id: id, models: models, err: err}
		}(id, p)
	}
	probes.Wait()
	close(results)
	for item := range results {
		if item.err != nil {
			result.Warnings[item.id] = item.err.Error()
			continue
		}
		result.Data = append(result.Data, item.models...)
	}
	sort.Slice(result.Data, func(i, j int) bool {
		a, _ := result.Data[i]["id"].(string)
		b, _ := result.Data[j]["id"].(string)
		return a < b
	})
	available := make(map[string]struct{}, len(result.Data))
	for _, model := range result.Data {
		if id, _ := model["id"].(string); id != "" {
			available[id] = struct{}{}
		}
	}
	if h.cfg.Routing.RouteAliasesEnabled {
		for _, route := range h.cfg.Routes {
			usableTargets := make([]string, 0, len(route.Targets))
			for _, target := range route.Targets {
				if _, ok := available[target]; ok {
					usableTargets = append(usableTargets, target)
				}
			}
			if len(usableTargets) == 0 {
				continue
			}
			result.Data = append(result.Data, map[string]any{
				"id":               "route/" + route.ID,
				"object":           "model",
				"x_provider":       "route",
				"x_provider_name":  "AhB Route",
				"x_route_targets":  route.Targets,
				"x_usable_targets": usableTargets,
			})
		}
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

	requested, err := requestedModel(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
		return
	}
	requestedProvider, requestedUpstream, err := splitModel(requested)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
		return
	}
	if requestedProvider == "route" {
		h.handleRouteProxy(w, r, raw, requestedUpstream)
		return
	}
	if h.cfg.Routing.SameModelFallback.Enabled {
		if _, ok := h.providers[requestedProvider]; !ok {
			writeError(w, http.StatusBadRequest, "unknown_provider", "provider is not configured")
			return
		}
		h.handleSameModelFallback(w, r, raw, requestedProvider, requestedUpstream)
		return
	}

	providerID, _, rewritten, err := rewriteModelBody(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
		return
	}
	p, ok := h.providers[providerID]
	if !ok || !p.cfg.Enabled || !p.hasRuntime() {
		writeError(w, http.StatusBadRequest, "unknown_provider", "provider is not enabled")
		return
	}
	lease, startupErr := h.acquireOnDemand(r.Context(),p)
	if startupErr!=nil {
		writeError(w,http.StatusServiceUnavailable,"provider_start_unavailable",startupErr.Error())
		return
	}
	defer lease() // Keep sidecar alive until the final SSE byte is copied.
	snap := p.snapshot()
	if !providerUsableForRouting(providerID, p.cfg.Kind, snap, p.accounts.snapshot()) {
		assessment := assessProviderHealth(providerID, p.cfg.Kind, snap, p.accounts.snapshot())
		writeError(w, http.StatusServiceUnavailable, "provider_unavailable", fmt.Sprintf("%s is %s", providerID, assessment.State))
		return
	}

	resp, err := h.doProviderRequest(r, p.cfg, rewritten)
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
	drop := connectionHeaderTokens(src)
	for k, values := range src {
		if shouldDropRequestHeader(k) {
			continue
		}
		if _, nominated := drop[strings.ToLower(k)]; nominated {
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
	drop := connectionHeaderTokens(src)
	for k, values := range src {
		if isHopByHop(k) {
			continue
		}
		if _, nominated := drop[strings.ToLower(k)]; nominated {
			continue
		}
		for _, v := range values {
			dst.Add(k, v)
		}
	}
}

// RFC 9110: a Connection header also names additional hop-by-hop fields.
// Forwarding one of those headers can leak a private upstream control field.
func connectionHeaderTokens(h http.Header) map[string]struct{} {
	out := make(map[string]struct{})
	for _, value := range h.Values("Connection") {
		for _, token := range strings.Split(value, ",") {
			if field := strings.ToLower(strings.TrimSpace(token)); field != "" {
				out[field] = struct{}{}
			}
		}
	}
	return out
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
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return // Caller disconnected; stop consuming the upstream stream.
			}
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
 // SetUpdateHandler enables the fixed bundled prebuilt updater only on Android.
func (h *Hub) SetUpdateHandler(fn func() error) {
 h.updateFn = fn
}

func (h *Hub) handleControlUpdate(w http.ResponseWriter, r *http.Request) {
 w.Header().Set("Cache-Control", "no-store")
 if r.Method != http.MethodPost {
  w.Header().Set("Allow", "POST")
  http.Error(w, "POST required", http.StatusMethodNotAllowed)
  return
 }
 if !h.authorizeLocalControl(w,r){return}
 h.controlMu.Lock()
 defer h.controlMu.Unlock()
 if h.controlPending {
  http.Error(w, "Another AhB operation is pending", http.StatusConflict)
  return
 }
 if h.updateFn == nil {
  http.Error(w, "Automatic upgrades unavailable", http.StatusForbidden)
  return
 }
 if err:=h.updateFn(); err!=nil {
  http.Error(w, "Could not launch AhB updater; original service is unchanged", http.StatusInternalServerError)
  return
 }
 h.controlPending=true
 writeJSON(w,http.StatusAccepted,map[string]string{
  "status":"scheduled","message":"Checking published Android package; upgrades keep backups and start AhB again.",
 })
}

// SetRestartHandler installs the local Android/Termux restart hook before
 // the server starts. In tests and on non-Termux systems it remains disabled.
func (h *Hub) SetRestartHandler(fn func() error) {
	h.restartFn = fn
}

func (h *Hub) authorizeLocalControl(w http.ResponseWriter, r *http.Request) bool {
	// Local-only management: never permit controls from a LAN-exposed Hub.
	if h.cfg.AllowLAN || h.controlToken == "" || h.restartFn == nil {
		http.Error(w, "Local Termux controls unavailable", http.StatusForbidden)
		return false
	}
	listenHost, _, err := net.SplitHostPort(h.cfg.Listen)
	if err != nil || !(listenHost == "localhost" || (net.ParseIP(listenHost) != nil && net.ParseIP(listenHost).IsLoopback())) {
		http.Error(w, "Local controls require a loopback listener", http.StatusForbidden)
		return false
	}
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil || !(host == "localhost" || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())) {
		http.Error(w, "Local controls require a loopback host", http.StatusForbidden)
		return false
	}
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || net.ParseIP(remoteHost) == nil || !net.ParseIP(remoteHost).IsLoopback() {
		http.Error(w, "Local controls require a loopback client", http.StatusForbidden)
		return false
	}
	// An Origin check plus an unguessable, page-local header protects against
	// a hostile website submitting blind POSTs to the localhost API.
	if r.Header.Get("Origin") != "http://"+r.Host ||
		subtle.ConstantTimeCompare([]byte(r.Header.Get("X-AhB-Control-Token")), []byte(h.controlToken)) != 1 {
		http.Error(w, "Invalid local UI authorization", http.StatusForbidden)
		return false
	}
	return true
}

func (h *Hub) handleControlRestart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if !h.authorizeLocalControl(w, r) {
		return
	}
	h.controlMu.Lock()
	defer h.controlMu.Unlock()
	if h.controlPending {
		http.Error(w, "Restart already scheduled", http.StatusConflict)
		return
	}
	if err := h.restartFn(); err != nil {
		// Do not expose filesystem paths or any internal details to the UI.
		http.Error(w, "Restart could not be scheduled; original AhB remains running", http.StatusInternalServerError)
		return
	}
	h.controlPending = true
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "scheduled",
		"message": "AhB will shut down cleanly and restart; the Termux app remains open.",
	})
}
