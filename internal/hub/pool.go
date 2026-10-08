package hub

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const modelCatalogTTL = 45 * time.Second

type providerLoad struct {
	inflight atomic.Int64
}

func (l *providerLoad) acquire() { l.inflight.Add(1) }
func (l *providerLoad) release() { l.inflight.Add(-1) }
func (l *providerLoad) current() int64 { return l.inflight.Load() }

type poolBalancer struct {
	sequence atomic.Uint64
}

type modelCatalog struct {
	mu        sync.RWMutex
	ids       map[string]struct{}
	updatedAt time.Time
}

func (c *modelCatalog) set(models []map[string]any) {
	ids := make(map[string]struct{}, len(models))
	for _, model := range models {
		if id, _ := model["x_upstream_id"].(string); strings.TrimSpace(id) != "" {
			ids[id] = struct{}{}
		}
	}
	c.mu.Lock()
	c.ids = ids
	c.updatedAt = time.Now()
	c.mu.Unlock()
}

func (c *modelCatalog) freshHas(model string) (known, has bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.updatedAt.IsZero() || time.Since(c.updatedAt) > modelCatalogTTL {
		return false, false
	}
	_, has = c.ids[model]
	return true, has
}

type poolCandidate struct {
	id       string
	provider *runtimeProvider
	inflight int64
}

func (h *Hub) providerSupportsModel(ctx context.Context, p *runtimeProvider, model string) bool {
	if known, has := p.models.freshHas(model); known {
		return has
	}
	probeCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	models, err := h.fetchModels(probeCtx, p.cfg)
	if err != nil {
		return false
	}
	p.models.set(models)
	_, has := p.models.freshHas(model)
	return has
}

func (h *Hub) poolCandidates(ctx context.Context, model string) []poolCandidate {
	out := make([]poolCandidate, 0, len(h.providers))
	for id, p := range h.providers {
		if !p.cfg.Enabled || p.sup == nil {
			continue
		}
		if !providerUsableForRouting(id, p.sup.Snapshot(), p.accounts.snapshot()) {
			continue
		}
		if !h.providerSupportsModel(ctx, p, model) {
			continue
		}
		out = append(out, poolCandidate{
			id: id, provider: p, inflight: p.load.current(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].inflight != out[j].inflight {
			return out[i].inflight < out[j].inflight
		}
		return out[i].id < out[j].id
	})
	if len(out) < 2 {
		return out
	}

	// Rotate only the least-loaded tie group. Combined with the live inflight
	// counter this spreads simultaneous independent requests without sending
	// one request to multiple providers.
	min := out[0].inflight
	tied := 1
	for tied < len(out) && out[tied].inflight == min {
		tied++
	}
	if tied > 1 {
		offset := int((h.pool.sequence.Add(1) - 1) % uint64(tied))
		rotated := append([]poolCandidate{}, out[offset:tied]...)
		rotated = append(rotated, out[:offset]...)
		copy(out[:tied], rotated)
	}
	return out
}

func sameModelPoolEntries(models []map[string]any) []map[string]any {
	providers := map[string]map[string]struct{}{}
	for _, model := range models {
		providerID, _ := model["x_provider"].(string)
		upstreamID, _ := model["x_upstream_id"].(string)
		if providerID == "" || upstreamID == "" || providerID == "route" || providerID == "pool" {
			continue
		}
		if providers[upstreamID] == nil {
			providers[upstreamID] = map[string]struct{}{}
		}
		providers[upstreamID][providerID] = struct{}{}
	}

	out := make([]map[string]any, 0)
	for modelID, set := range providers {
		if len(set) < 2 {
			continue
		}
		ids := make([]string, 0, len(set))
		for providerID := range set {
			ids = append(ids, providerID)
		}
		sort.Strings(ids)
		out = append(out, map[string]any{
			"id":               "pool/" + modelID,
			"object":           "model",
			"x_provider":       "pool",
			"x_provider_name":  "AhB Same-Model Pool",
			"x_upstream_id":    modelID,
			"x_pool_providers": ids,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i]["id"].(string) < out[j]["id"].(string)
	})
	return out
}

func (h *Hub) handlePoolProxy(w http.ResponseWriter, r *http.Request, raw []byte, model string) {
	if !h.cfg.Routing.SameModelPoolEnabled {
		writeError(w, http.StatusBadRequest, "pool_disabled", "same-model provider pool is disabled")
		return
	}

	candidates := h.poolCandidates(r.Context(), model)
	if len(candidates) == 0 {
		writeError(w, http.StatusServiceUnavailable, "pool_unavailable", fmt.Sprintf("no usable provider exposes model %q", model))
		return
	}

	rewritten, err := rewriteModelTo(raw, model)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
		return
	}

	failures := make([]string, 0, len(candidates))
	var last *bufferedRouteResponse
	attempts := 0

	for _, candidate := range candidates {
		p := candidate.provider
		p.load.acquire()
		attempts++

		resp, reqErr := h.doProviderRequest(r, p.cfg, rewritten)
		if reqErr != nil {
			p.load.release()
			failures = append(failures, candidate.id+": transport error")
			continue
		}

		if !routeRetryableStatus(resp.StatusCode) {
			w.Header().Set("X-AhB-Pool", model)
			w.Header().Set("X-AhB-Provider", candidate.id)
			w.Header().Set("X-AhB-Attempts", fmt.Sprintf("%d", attempts))
			copyResponseHeaders(w.Header(), resp.Header)
			w.WriteHeader(resp.StatusCode)
			copyStreaming(w, resp.Body)
			_ = resp.Body.Close()
			p.load.release()
			return
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxRouteErrorBody))
		_ = resp.Body.Close()
		p.load.release()
		if readErr != nil {
			failures = append(failures, fmt.Sprintf("%s: HTTP %d", candidate.id, resp.StatusCode))
			continue
		}
		last = &bufferedRouteResponse{
			status:   resp.StatusCode,
			header:   resp.Header.Clone(),
			body:     body,
			provider: candidate.id,
		}
		failures = append(failures, fmt.Sprintf("%s: HTTP %d", candidate.id, resp.StatusCode))
	}

	if last != nil {
		w.Header().Set("X-AhB-Pool", model)
		w.Header().Set("X-AhB-Provider", last.provider)
		w.Header().Set("X-AhB-Attempts", fmt.Sprintf("%d", attempts))
		copyResponseHeaders(w.Header(), last.header)
		w.WriteHeader(last.status)
		_, _ = w.Write(last.body)
		return
	}

	writeError(
		w,
		http.StatusBadGateway,
		"pool_failed",
		fmt.Sprintf("all providers for model %q failed: %s", model, strings.Join(failures, "; ")),
	)
}
