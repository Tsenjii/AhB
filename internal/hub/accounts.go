package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

// refreshProviderAccounts checks authenticated readiness after a sidecar
// starts; a running HTTP endpoint alone is not proof of usable credentials.
func (h *Hub) refreshProviderAccounts(ctx context.Context, p *runtimeProvider) {
 probe, err := h.fetchProviderAccounts(ctx, p.cfg)
 if err != nil {
  p.accounts.set(accountProbe{Known:false,LastError:err.Error(),UpdatedAt:time.Now()})
  return
 }
 probe.UpdatedAt=time.Now()
 p.accounts.set(probe)
}

func (h *Hub) pollProviderAccounts(ctx context.Context, p *runtimeProvider) {
 refresh := func() {
  // On-demand sidecars intentionally have no listening endpoint while
  // sleeping. Do not generate background localhost requests and false
  // failures every ten seconds on a 512 MiB VPS. DeepSeek uses a local
  // on-disk account probe and can still be inspected while asleep.
  if h.onDemand(p) && p.cfg.ID!="deepseek" && p.snapshot().PID==0 {return}
  h.refreshProviderAccounts(ctx,p)
 }
 refresh()
 ticker:=time.NewTicker(10*time.Second)
 defer ticker.Stop()
 for {
  select {
  case <-ctx.Done():return
  case <-ticker.C:refresh()
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

	case "kiro":
		var payload struct {
			Accounts  int `json:"accounts"`
			Available int `json:"available"`
		}
		if err := h.getProviderJSON(ctx, cfg, "/v1/stats", &payload); err != nil {
			return accountProbe{}, err
		}
		if payload.Accounts < 0 || payload.Available < 0 || payload.Available > payload.Accounts {
			return accountProbe{}, fmt.Errorf("kiro stats returned invalid account counts")
		}
		return accountProbe{Known: true, Total: payload.Accounts, Usable: payload.Available}, nil

	case "deepseek":
		return inspectDeepSeekAccounts(cfg)

	case "freebuff":
		// Node gateway exposes local-only summary of configured Bearer/CLI
		// identities in /healthz. It does not expose the Rust API's
		// /api/accounts/health and cannot import Web cookies automatically.
		// "unknown" identities are only candidates for first live test, NOT
		// proven-valid tokens or spendable quota.
		var payload struct {
			Accounts int `json:"accounts"`
			Alive int `json:"alive_accounts"`
			Unknown int `json:"unknown_accounts"`
		}
		if err := h.getProviderJSON(ctx, cfg, "/healthz", &payload); err != nil {
			return accountProbe{}, err
		}
		if payload.Accounts < 0 || payload.Alive < 0 || payload.Unknown < 0 ||
			payload.Alive+payload.Unknown > payload.Accounts {
			return accountProbe{}, fmt.Errorf("freebuff health returned invalid account counts")
		}
		return accountProbe{Known: true, Total: payload.Accounts, Usable: payload.Alive + payload.Unknown}, nil
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


func inspectDeepSeekAccounts(cfg config.ProviderConfig) (accountProbe, error) {
  // New 0xgetz Web adapter: only the private own-account tokens file
  // is a credential candidate inventory. Tokens are NEVER sent to UI.
  // This does not measure quota or prove a live chat response.
  if name := strings.TrimSpace(cfg.Env["DEEPSEEK_ACCOUNTS_FILE"]); name != "" {
    if !filepath.IsAbs(name) {
      base := strings.TrimSpace(cfg.WorkDir)
      if base == "" { base = "." }
      name = filepath.Join(base, name)
    }
    info, err := os.Lstat(name)
    if os.IsNotExist(err) { return accountProbe{Known:true},nil }
    if err != nil { return accountProbe{},fmt.Errorf("stat private DeepSeek account file: %w",err) }
    if !info.Mode().IsRegular() || info.Mode().Perm()&0077!=0 {
      return accountProbe{},fmt.Errorf("DeepSeek account file must be private regular file (0600)")
    }
    raw,err:=os.ReadFile(name)
    if err!=nil{return accountProbe{},fmt.Errorf("read private DeepSeek account file: %w",err)}
    count:=0
    for _,v:=range strings.Split(string(raw),"\n"){
      v=strings.TrimSpace(v)
      if v!="" && !strings.HasPrefix(v,"#"){count++}
    }
    return accountProbe{Known:true,Total:count,Usable:count},nil
  }
	configPath := strings.TrimSpace(cfg.Env["Deepseek2API_CONFIG_PATH"])
	if configPath == "" {
		configPath = "config.json"
	}
	if !filepath.IsAbs(configPath) {
		base := strings.TrimSpace(cfg.WorkDir)
		if base == "" {
			base = "."
		}
		configPath = filepath.Join(base, configPath)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return accountProbe{}, fmt.Errorf("read DeepSeek account config: %w", err)
	}
	var doc struct {
		Accounts []struct {
			Email    string `json:"email"`
			Mobile   string `json:"mobile"`
			Password string `json:"password"`
			Token    string `json:"token"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return accountProbe{}, fmt.Errorf("decode DeepSeek account config: %w", err)
	}
	usable := 0
	for _, account := range doc.Accounts {
		hasID := strings.TrimSpace(account.Email) != "" || strings.TrimSpace(account.Mobile) != ""
		hasCredential := strings.TrimSpace(account.Password) != "" || strings.TrimSpace(account.Token) != ""
		if hasID && hasCredential {
			usable++
		}
	}
	return accountProbe{Known: true, Total: len(doc.Accounts), Usable: usable}, nil
}
