package hub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		{"agent2api legacy degraded", "agent2api", `{"status":"degraded","unavailableReason":"no login"}`, provider.StateDegraded, true, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := assessProviderHealth(tc.id, sidecar.Snapshot{State: healthy, PID: 123, HealthHTTPStatus: 200, HealthBody: []byte(tc.body)})
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
