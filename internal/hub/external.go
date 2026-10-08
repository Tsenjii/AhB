package hub

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
	"github.com/Tsenjii/AhB/internal/provider"
	"github.com/Tsenjii/AhB/internal/sidecar"
)

type externalProbeState struct {
	mu   sync.RWMutex
	snap sidecar.Snapshot
}

func newExternalProbe(cfg config.ProviderConfig) *externalProbeState {
	return &externalProbeState{snap: sidecar.Snapshot{ID: cfg.ID, State: provider.StateStarting}}
}

func (s *externalProbeState) snapshot() sidecar.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.snap
	out.HealthBody = append([]byte(nil), s.snap.HealthBody...)
	return out
}

func (s *externalProbeState) set(snap sidecar.Snapshot) {
	s.mu.Lock()
	s.snap = snap
	s.mu.Unlock()
}

func (h *Hub) runExternalProbe(ctx context.Context, p *runtimeProvider) {
	interval := time.Duration(p.cfg.HealthIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 15 * time.Second
	}
	refresh := func() {
		p.external.set(h.probeExternal(ctx, p.cfg))
	}
	refresh()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refresh()
		}
	}
}

func (h *Hub) probeExternal(parent context.Context, cfg config.ProviderConfig) sidecar.Snapshot {
	snap := sidecar.Snapshot{ID: cfg.ID, State: provider.StateDead}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	endpoint, err := joinURL(cfg.BaseURL, cfg.HealthPath)
	if err != nil {
		snap.LastError = err.Error()
		return snap
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		snap.LastError = err.Error()
		return snap
	}
	applyProviderHeaders(req.Header, cfg.Headers)
	resp, err := h.client.Do(req)
	if err != nil {
		snap.LastError = fmt.Sprintf("external health unreachable: %v", err)
		return snap
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if readErr != nil {
		snap.LastError = fmt.Sprintf("external health read failed: %v", readErr)
		return snap
	}
	snap.HealthHTTPStatus = resp.StatusCode
	snap.HealthBody = body
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		snap.State = provider.StateHealthy
		return snap
	}
	snap.State = provider.StateDegraded
	snap.LastError = fmt.Sprintf("external health returned HTTP %d", resp.StatusCode)
	return snap
}
