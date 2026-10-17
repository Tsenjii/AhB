package hub

import (
	"reflect"
	"testing"

	"github.com/Tsenjii/AhB/internal/config"
)

// Explicitly empty fallback allowlists must not enroll other installed gateways.
// New external bridges only participate when individually authorized.
func TestFallbackProviderIDsExplicitAllowlist(t *testing.T) {
	cases := []struct {
		name      string
		allowlist []string
		want      []string
	}{
		{name: "empty means primary only", allowlist: []string{}, want: []string{"primary"}},
		{name: "nil means primary only", allowlist: nil, want: []string{"primary"}},
		{name: "explicit alternate", allowlist: []string{"alternate"}, want: []string{"primary", "alternate"}},
		{name: "primary not duplicated", allowlist: []string{"primary", "alternate"}, want: []string{"primary", "alternate"}},
		{name: "trim provider ID", allowlist: []string{" alternate "}, want: []string{"primary", "alternate"}},
		{name: "deduplicate alternate", allowlist: []string{"alternate", "alternate"}, want: []string{"primary", "alternate"}},
		{name: "custom only", allowlist: []string{"newbridge"}, want: []string{"primary", "newbridge"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := New(config.Config{
				Providers: []config.ProviderConfig{
					{ID: "primary", Kind: "sidecar"},
					{ID: "alternate", Kind: "sidecar"},
					{ID: "newbridge", Kind: "external"},
				},
				Routing: config.RoutingConfig{SameModelFallback: config.SameModelFallbackConfig{
					Enabled: true, Mode: "sequential", Providers: tc.allowlist,
				}},
			})
			if got := h.fallbackProviderIDs("primary"); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("fallback provider IDs = %v; want %v", got, tc.want)
			}
		})
	}
}
