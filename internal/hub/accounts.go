package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
)

type accountProbe struct {
	Known     bool
	Total     int
	Usable    int
	LastError string
	UpdatedAt time.Time
}

type accountProbeState struct {
	mu   sync.RWMutex
	last accountProbe
}

func (s *accountProbeState) snapshot() accountProbe {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.last
}

func (s *accountProbeState) set(v accountProbe) {
	s.mu.Lock()
	s.last = v
	s.mu.Unlock()
}

func (h *Hub) pollProviderAccounts(ctx context.Context, p *runtimeProvider) {
	refresh := func() {
		probe, err := h.fetchProviderAccounts(ctx, p.cfg)
		if err != nil {
			p.accounts.set(accountProbe{Known: false, LastError: err.Error(), UpdatedAt: time.Now()})
			return
		}
		probe.UpdatedAt = time.Now()
		p.accounts.set(probe)
	}
	refresh()

	ticker := time.NewTicker(10 * time.Second)
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

func (h *Hub) fetchProviderAccounts(parent context.Context, cfg config.ProviderConfig) (accountProbe, error) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()

	switch cfg.ID {
	case "agent2api":
		var envelope struct {
			Success bool `json:"success"`
			Data struct {
				Accounts []struct {
					Enabled        bool `json:"enabled"`
					HasCredentials bool `json:"hasCredentials"`
					ChatSupported  bool `json:"chatSupported"`
				} `json:"accounts"`
			} `json:"data"`
		}
		if err := h.getProviderJSON(ctx, cfg, "/api/accounts", &envelope); err != nil {
			return accountProbe{}, err
		}
		if !envelope.Success {
			return accountProbe{}, fmt.Errorf("agent2api accounts response was not successful")
		}
		usable := 0
		for _, account := range envelope.Data.Accounts {
			if account.Enabled && account.HasCredentials && account.ChatSupported {
				usable++
			}
		}
		return accountProbe{Known: true, Total: len(envelope.Data.Accounts), Usable: usable}, nil

	case "freebuff":
		var payload struct {
			OK       bool `json:"ok"`
			Accounts []struct {
				CircuitState string  `json:"circuit_state"`
				CooldownUntil *string `json:"cooldown_until"`
			} `json:"accounts"`
		}
		if err := h.getProviderJSON(ctx, cfg, "/api/accounts/health", &payload); err != nil {
			return accountProbe{}, err
		}
		if !payload.OK {
			return accountProbe{}, fmt.Errorf("freebuff account health response was not ok")
		}
		usable := 0
		now := time.Now()
		for _, account := range payload.Accounts {
			state := strings.ToLower(strings.TrimSpace(account.CircuitState))
			if state != "open" {
				usable++
				continue
			}
			if account.CooldownUntil != nil {
				if until, err := time.Parse(time.RFC3339, *account.CooldownUntil); err == nil && !until.After(now) {
					usable++
				}
			}
		}
		return accountProbe{Known: true, Total: len(payload.Accounts), Usable: usable}, nil
	default:
		return accountProbe{}, fmt.Errorf("provider %s does not expose an account probe", cfg.ID)
	}
}

func (h *Hub) getProviderJSON(ctx context.Context, cfg config.ProviderConfig, path string, dst any) error {
	endpoint, err := joinURL(cfg.BaseURL, path)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	applyProviderHeaders(req.Header, cfg.Headers)
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s returned HTTP %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(dst); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
