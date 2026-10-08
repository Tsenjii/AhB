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