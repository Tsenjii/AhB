package config

import "testing"

func TestValidateRejectsPublicListenByDefault(t *testing.T) {
	cfg := Config{Listen: "0.0.0.0:8317"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected public listen to be rejected")
	}
}

func TestValidateAcceptsLoopbackSidecar(t *testing.T) {
	cfg := Config{
		Listen: "127.0.0.1:8317",
		Providers: []ProviderConfig{{
			ID: "opencode", Enabled: true, Kind: "sidecar",
			Binary: "/tmp/opencode2api", BaseURL: "http://127.0.0.1:8401",
			UIURL: "http://127.0.0.1:8404/",
		}},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsDuplicateProviderIDs(t *testing.T) {
	cfg := Config{
		Listen: "127.0.0.1:8317",
		Providers: []ProviderConfig{
			{ID: "x", Enabled: false},
			{ID: "x", Enabled: false},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected duplicate provider id to be rejected")
	}
}

func TestValidateRejectsCredentialsInUIURL(t *testing.T) {
	cfg := Config{
		Listen: "127.0.0.1:8317",
		Providers: []ProviderConfig{{
			ID: "x", Enabled: false,
			UIURL: "http://user:pass@127.0.0.1:9000/",
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected credentials in ui_url to be rejected")
	}
}

func TestValidateAcceptsExplicitCrossProviderRoute(t *testing.T) {
	cfg := Config{
		Listen: "127.0.0.1:8317",
		Providers: []ProviderConfig{
			{ID: "opencode", Enabled: false},
			{ID: "agent2api", Enabled: false},
		},
		Routes: []RouteConfig{{
			ID: "coding",
			Targets: []string{
				"opencode/muse-spark-1.3-contributor-free",
				"agent2api/Qwen3.8-Flash",
			},
		}},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsUnknownRouteProvider(t *testing.T) {
	cfg := Config{
		Listen: "127.0.0.1:8317",
		Providers: []ProviderConfig{{ID: "opencode", Enabled: false}},
		Routes: []RouteConfig{{ID: "coding", Targets: []string{"missing/model"}}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unknown route provider to be rejected")
	}
}

func TestValidateRejectsReservedRouteProviderID(t *testing.T) {
	cfg := Config{
		Listen: "127.0.0.1:8317",
		Providers: []ProviderConfig{{ID: "route", Enabled: false}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected reserved provider id to be rejected")
	}
}
