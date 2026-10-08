package hub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tsenjii/AhB/internal/config"
	"github.com/Tsenjii/AhB/internal/provider"
	"github.com/Tsenjii/AhB/internal/sidecar"
)

func TestSplitModelPreservesNestedUpstreamID(t *testing.T) {
	providerID, upstream, err := splitModel("freebuff/deepseek/deepseek-v4-flash")
	if err != nil {
		t.Fatal(err)
	}
	if providerID != "freebuff" {
		t.Fatalf("provider = %q", providerID)
	}
	if upstream != "deepseek/deepseek-v4-flash" {
		t.Fatalf("upstream = %q", upstream)
	}
}

func TestRewriteModelBody(t *testing.T) {
	raw := []byte(`{"model":"opencode/mimo-v2.6-flash-free","messages":[{"role":"user","content":"hi"}]}`)
	providerID, upstream, out, err := rewriteModelBody(raw)
	if err != nil {
		t.Fatal(err)
	}
	if providerID != "opencode" || upstream != "mimo-v2.6-flash-free" {
		t.Fatalf("unexpected route: %s %s", providerID, upstream)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "mimo-v2.6-flash-free" {
		t.Fatalf("rewritten model = %v", got["model"])
	}
}

func TestRewriteModelRequiresPrefix(t *testing.T) {
	_, _, _, err := rewriteModelBody([]byte(`{"model":"glm-5.3-flash"}`))
	if err == nil || !strings.Contains(err.Error(), "provider/model") {
		t.Fatalf("expected provider/model error, got %v", err)
	}
}

func TestJoinURL(t *testing.T) {
	got, err := joinURL("http://127.0.0.1:8401/", "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:8401/v1/models" {
		t.Fatalf("got %q", got)
	}
}

func TestRewritePreservesLargeInteger(t *testing.T) {
	raw := []byte(`{"model":"opencode/test","seed":9007199254740993}`)
	_, _, out, err := rewriteModelBody(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"seed":9007199254740993`) {
		t.Fatalf("large integer changed: %s", out)
	}
}

func TestClientCredentialsAreNotForwarded(t *testing.T) {
	for _, h := range []string{"Authorization", "X-Api-Key", "Connection"} {
		if !shouldDropRequestHeader(h) {
			t.Fatalf("%s should be stripped before forwarding", h)
		}
	}
}

func TestDashboardHandlerSmoke(t *testing.T) {
	h := New(config.Config{
		Listen: "127.0.0.1:8317",
		Providers: []config.ProviderConfig{{
			ID: "opencode", DisplayName: "OpenCode Free",
			Enabled: false, Kind: "sidecar",
			UIURL: "http://127.0.0.1:8404/",
		}},
	})
	req := httptest.NewRequest(http.MethodGet, "/ui", nil)
	rec := httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, needle := range []string{
		`<meta name="viewport"`,
		`id="providers"`,
		`id="models"`,
		`id="runtime"`,
		"管理原本 UI",
		"統一 API",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("dashboard missing %q", needle)
		}
	}
}

func TestProviderViewsExposeManagementMetadata(t *testing.T) {
	h := New(config.Config{
		Listen: "127.0.0.1:8317",
		Providers: []config.ProviderConfig{{
			ID: "freebuff", DisplayName: "FreeBuff",
			Description: "test", Enabled: false, Kind: "sidecar",
			UIURL: "http://127.0.0.1:8402/ui",
		}},
	})
	views := h.providerViews()
	if len(views) != 1 {
		t.Fatalf("views = %d", len(views))
	}
	if views[0].DisplayName != "FreeBuff" || views[0].UIURL == "" {
		t.Fatalf("unexpected provider metadata: %#v", views[0])
	}
	if views[0].State != "DISABLED" {
		t.Fatalf("state = %s", views[0].State)
	}
}

func TestAssessProviderHealthLayers(t *testing.T) {
	healthy := provider.StateHealthy
	cases := []struct {
		name string
		id string
		body string
		state provider.State
		ready bool
		usable *bool
	}{
		{"opencode anonymous ready", "opencode", `{"status":"ok","ready":true,"keys":{"anonymous":true,"total":0}}`, healthy, true, boolPtr(true)},
		{"freebuff no accounts", "freebuff", `{"ok":true,"accounts":[]}`, provider.StateDegraded, true, boolPtr(false)},
		{"agent2api aggregate empty", "agent2api", `{"status":"degraded","unavailableReason":"no login"}`, provider.StateDegraded, true, boolPtr(false)},
		{"grok usable account", "grok", `{"ready":true,"state":"ready","components":{"grok_build":{"state":"ready"},"grok_web":{"state":"disabled"},"grok_console":{"state":"disabled"}}}`, healthy, true, boolPtr(true)},
		{"grok no usable account", "grok", `{"ready":false,"state":"not_ready","components":{"grok_build":{"state":"unavailable"},"grok_web":{"state":"disabled"},"grok_console":{"state":"disabled"}}}`, provider.StateDegraded, false, boolPtr(false)},
		{"kiro no accounts", "kiro", `{"status":"ok","version":"1.1.5"}`, provider.StateDegraded, true, boolPtr(false)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			probe := accountProbe{}
			if tc.id == "freebuff" { probe = accountProbe{Known:true, Total:0, Usable:0} }
			if tc.id == "agent2api" { probe = accountProbe{Known:true, Total:0, Usable:0} }
			if tc.id == "kiro" { probe = accountProbe{Known:true, Total:0, Usable:0} }
			a := assessProviderHealth(tc.id, "sidecar", sidecar.Snapshot{State: healthy, PID: 123, HealthHTTPStatus: 200, HealthBody: []byte(tc.body)}, probe)
			if a.State != tc.state || a.ProviderReady != tc.ready {
				t.Fatalf("assessment = %#v", a)
			}
			if tc.usable == nil {
				if a.AccountUsable != nil { t.Fatalf("account usable = %v, want nil", *a.AccountUsable) }
			} else if a.AccountUsable == nil || *a.AccountUsable != *tc.usable {
				t.Fatalf("account usable mismatch: %#v", a.AccountUsable)
			}
		})
	}
}


func TestFetchProviderAccountsAgent2API(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/accounts", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{"accounts": []map[string]any{
				{"enabled": true, "hasCredentials": true, "chatSupported": true},
				{"enabled": false, "hasCredentials": true, "chatSupported": true},
				{"enabled": true, "hasCredentials": false, "chatSupported": true},
			}},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	h := New(config.Config{})
	got, err := h.fetchProviderAccounts(context.Background(), config.ProviderConfig{ID:"agent2api", BaseURL:srv.URL})
	if err != nil { t.Fatal(err) }
	if !got.Known || got.Total != 3 || got.Usable != 1 { t.Fatalf("probe = %#v", got) }
}

func TestFetchProviderAccountsFreeBuffNodeHealth(t *testing.T) {
 mux:=http.NewServeMux()
 mux.HandleFunc("/healthz", func(w http.ResponseWriter,r *http.Request){
  _=json.NewEncoder(w).Encode(map[string]any{
   "status":"degraded","accounts":4,"alive_accounts":1,
   "unknown_accounts":2, "account_details":[]map[string]any{
    {"token":"PRIVATE_TOKEN_DO_NOT_LEAK"},
   },
  })
 })
 srv:=httptest.NewServer(mux); defer srv.Close()
 h:=New(config.Config{})
 got,err:=h.fetchProviderAccounts(context.Background(),config.ProviderConfig{ID:"freebuff",BaseURL:srv.URL})
 if err!=nil {t.Fatal(err)}
 if !got.Known||got.Total!=4||got.Usable!=3 {t.Fatalf("node probe = %#v",got)}
}


func TestRequestedModelAndRewriteModelTo(t *testing.T) {
	raw := []byte(`{"model":"route/coding","messages":[{"role":"user","content":"hi"}],"seed":9007199254740993}`)
	model, err := requestedModel(raw)
	if err != nil {
		t.Fatal(err)
	}
	if model != "route/coding" {
		t.Fatalf("model = %q", model)
	}
	out, err := rewriteModelTo(raw, "Qwen3.8-Flash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"model":"Qwen3.8-Flash"`) {
		t.Fatalf("model not rewritten: %s", out)
	}
	if !strings.Contains(string(out), `"seed":9007199254740993`) {
		t.Fatalf("large integer changed: %s", out)
	}
}

func TestRouteRetryableStatus(t *testing.T) {
	for _, status := range []int{402, 404, 408, 425, 429, 502, 503, 504} {
		if !routeRetryableStatus(status) {
			t.Fatalf("status %d should be retryable", status)
		}
	}
	for _, status := range []int{200, 400, 401, 403, 422, 500} {
		if routeRetryableStatus(status) {
			t.Fatalf("status %d should not be retryable", status)
		}
	}
}


func TestFallbackProviderIDsKeepsPrimaryFirst(t *testing.T) {
	h := New(config.Config{
		Providers: []config.ProviderConfig{
			{ID:"opencode"}, {ID:"agent2api"}, {ID:"freebuff"},
		},
		Routing: config.RoutingConfig{SameModelFallback: config.SameModelFallbackConfig{
			Enabled:true, Mode:"balanced", Providers:[]string{"agent2api","opencode","freebuff"},
		}},
	})
	got := h.fallbackProviderIDs("opencode")
	want := []string{"opencode","agent2api","freebuff"}
	if strings.Join(got,",") != strings.Join(want,",") {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestRouteConfigDisabledByDefault(t *testing.T) {
	h := New(config.Config{
		Routes: []config.RouteConfig{{ID:"coding", Targets:[]string{"opencode/model"}}},
	})
	if _, ok := h.routeConfig("coding"); ok {
		t.Fatal("route aliases must be disabled unless explicitly enabled")
	}
}


func TestBalancedPickerPrefersLowerInflight(t *testing.T) {
	h := New(config.Config{})
	a := &runtimeProvider{cfg: config.ProviderConfig{ID:"a"}}
	b := &runtimeProvider{cfg: config.ProviderConfig{ID:"b"}}
	a.load.acquire()
	defer a.load.release()

	picked := h.pickBalancedProvider([]*runtimeProvider{a, b}, map[string]struct{}{}, "a")
	if picked == nil || picked.cfg.ID != "b" {
		t.Fatalf("picked %#v, want b", picked)
	}
	picked.load.release()
}

func TestBalancedPickerPrefersPrimaryOnTie(t *testing.T) {
	h := New(config.Config{})
	a := &runtimeProvider{cfg: config.ProviderConfig{ID:"a"}}
	b := &runtimeProvider{cfg: config.ProviderConfig{ID:"b"}}

	picked := h.pickBalancedProvider([]*runtimeProvider{b, a}, map[string]struct{}{}, "a")
	if picked == nil || picked.cfg.ID != "a" {
		t.Fatalf("picked %#v, want primary a", picked)
	}
	picked.load.release()
}


func TestInspectDeepSeekAccountsCountsConfiguredCredentials(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := []byte(`{
		"accounts":[
			{"email":"one@example.com","password":"secret"},
			{"mobile":"+886900000000","token":"token"},
			{"email":"missing@example.com"},
			{"password":"no-identifier"}
		]
	}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := inspectDeepSeekAccounts(config.ProviderConfig{
		ID: "deepseek", WorkDir: dir,
		Env: map[string]string{"Deepseek2API_CONFIG_PATH":"config.json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Known || got.Total != 4 || got.Usable != 2 {
		t.Fatalf("probe = %#v", got)
	}
}


func TestFetchProviderAccountsKiroStats(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer local-key" {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"accounts": 4, "available": 3})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	h := New(config.Config{})
	got, err := h.fetchProviderAccounts(context.Background(), config.ProviderConfig{
		ID:"kiro", BaseURL:srv.URL,
		Headers:map[string]string{"Authorization":"Bearer local-key"},
	})
	if err != nil { t.Fatal(err) }
	if !got.Known || got.Total != 4 || got.Usable != 3 { t.Fatalf("probe = %#v", got) }
}
