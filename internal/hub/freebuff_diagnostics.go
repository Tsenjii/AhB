package hub

import (
	"context"
	"path/filepath"
	"strings"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Tsenjii/AhB/internal/provider"
)

// FreeBuff diagnosis never reads or returns token values or calls upstream
// chat. It only reports the local installation, existing credential count,
// process metadata and the gateway's bounded /healthz response. A healthy
// gateway cannot prove authorization or model quota.
func (h *Hub) handleFreebuffDiagnostics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if !h.authorizeLocalControl(w, r) {
		return
	}
	reply := map[string]any{
		"configured": false,
		"enabled": false,
		"process_alive": false,
		"gateway_ready": false,
		"saved_accounts": 0,
		"gateway_health": "not_checked",
		"next_step": "請先從 AhB 安裝原生 FreeBuff Gateway。",
		"proof": "Metadata-only, no inference performed.",
	}
	if h.controlConfigPath != "" {
		root := filepath.Dir(h.controlConfigPath)
		if !strings.HasPrefix(root, "/") { root = "" }
		if root != "" { reply["saved_accounts"] = countFreebuffCredentials(root) }
	}
	p := h.providers["freebuff"]
	if p == nil || p.cfg.Kind != "sidecar" {
		writeJSON(w, http.StatusOK, reply)
		return
	}
	reply["configured"] = true
	reply["enabled"] = p.cfg.Enabled
	snap := p.snapshot()
	reply["process_alive"] = snap.PID > 0
	reply["gateway_ready"] = snap.State == provider.StateHealthy
	last := p.lastRequest.snapshot()
	if !last.At.IsZero() {
		reply["last_api_http_status"] = last.Status
		reply["last_api_transport_error"] = last.TransportError
		reply["last_api_at"] = last.At.Format(time.RFC3339)
	}
	accounts := reply["saved_accounts"].(int)
	switch {
	case !p.cfg.Enabled:
		reply["next_step"] = "FreeBuff 已停用。可在「總覽」啟用，再檢查登入狀態。"
	case accounts == 0:
		reply["next_step"] = "沒有已儲存的 CLI 帳號。按「以 Google 登入 FreeBuff」完成官方授權；舊版 Web Cookie 不會自動轉移。"
	case snap.PID <= 0:
		reply["next_step"] = "本機已存帳號，但 Gateway 目前未執行。到「總覽」啟動來源，再重新診斷。"
	default:
		reply["next_step"] = "已保存帳號候選資料。請先確認 Gateway 健康狀態，再用 Playground 真實測試模型與 Tool Calling；不保證額度。"
	}
	if snap.PID > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		endpoint, err := joinURL(p.cfg.BaseURL, p.cfg.HealthPath)
		if err == nil {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
			if err == nil {
				resp, err := h.client.Do(req)
				if err == nil {
					defer resp.Body.Close()
					reply["gateway_health_http_status"] = resp.StatusCode
					if resp.StatusCode == http.StatusOK {
						var state struct {
							Accounts *int `json:"accounts"`
							Alive *int `json:"alive_accounts"`
							Unknown *int `json:"unknown_accounts"`
						}
						dec := json.NewDecoder(io.LimitReader(resp.Body, 8192))
						if dec.Decode(&state) == nil && state.Accounts != nil && state.Alive != nil && state.Unknown != nil &&
							*state.Accounts >= 0 && *state.Alive >= 0 && *state.Unknown >= 0 && *state.Alive+*state.Unknown <= *state.Accounts {
							reply["gateway_health"] = "metadata_valid"
							reply["gateway_loaded_accounts"] = *state.Accounts
							reply["gateway_observed_alive"] = *state.Alive
							reply["gateway_untested_accounts"] = *state.Unknown
							if *state.Accounts == 0 && accounts > 0 {
								reply["next_step"] = "Gateway 尚未讀取新的 CLI 帳號。請確認沒有推論中的請求，再從總覽重新載入 FreeBuff。"
							}
						} else { reply["gateway_health"] = "invalid_schema" }
					} else { reply["gateway_health"] = "http_error" }
				} else { reply["gateway_health"] = "transport_error" }
			}
		}
	}
	writeJSON(w, http.StatusOK, reply)
}
