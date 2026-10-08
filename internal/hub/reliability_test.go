package hub

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tsenjii/AhB/internal/config"
	"github.com/Tsenjii/AhB/internal/provider"
	"github.com/Tsenjii/AhB/internal/sidecar"
)

func TestAccountBackedProvidersFailClosedWhenProbeUnknown(t *testing.T) {
	for _, id := range []string{"agent2api", "freebuff", "deepseek", "kiro"} {
		t.Run(id, func(t *testing.T) {
			for _, healthBody := range [][]byte{nil, []byte("not-json"), []byte(`{"status":"ok"}`)} {
				snap := sidecar.Snapshot{
					ID: id, State: provider.StateHealthy, PID: 123,
					HealthHTTPStatus: http.StatusOK, HealthBody: healthBody,
				}
				if providerUsableForRouting(id, "sidecar", snap, accountProbe{}) {
					t.Fatalf("provider %s routed without a verified account probe", id)
				}
				assessment := assessProviderHealth(id, "sidecar", snap, accountProbe{})
				if assessment.State != provider.StateDegraded {
					t.Fatalf("unknown accounts were marked %s, not degraded", assessment.State)
				}
				if providerUsableForRouting(id, "sidecar", snap, accountProbe{Known: true, Total: 2, Usable: 0}) {
					t.Fatal("provider with zero usable accounts was routable")
				}
				if !providerUsableForRouting(id, "sidecar", snap, accountProbe{Known: true, Total: 2, Usable: 1}) {
					t.Fatal("provider with known usable account should be routable")
				}
			}
		})
	}
}

func TestSameModelFallbackRejectsUnadvertisedAlternates(t *testing.T) {
	for _, tt := range []struct {
		name string
		advertisedModel string
		wantStatus int
		wantAltCalls int
	}{
		{"unrelated model", "different-model", http.StatusServiceUnavailable, 0},
		{"identical model", "target-model", http.StatusOK, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1/chat/completions":
					http.Error(w, "primary temporarily unavailable", http.StatusServiceUnavailable)
				default:
					http.NotFound(w, r)
				}
			}))
			defer primary.Close()

			alternateCalls := 0
			alternate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1/models":
					_ = json.NewEncoder(w).Encode(map[string]any{
						"data": []map[string]string{{"id": tt.advertisedModel}},
					})
				case "/v1/chat/completions":
					alternateCalls++
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					if payload["model"] != "target-model" {
						t.Errorf("upstream model changed to %v", payload["model"])
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
				default:
					http.NotFound(w, r)
				}
			}))
			defer alternate.Close()

			cfg := config.Config{
				Listen: "127.0.0.1:8317",
				Providers: []config.ProviderConfig{
					{ID: "primary", Enabled: true, Kind: "external", BaseURL: primary.URL, HealthPath: "/healthz", ModelsPath: "/v1/models"},
					{ID: "alternate", Enabled: true, Kind: "external", BaseURL: alternate.URL, HealthPath: "/healthz", ModelsPath: "/v1/models"},
				},
				Routing: config.RoutingConfig{SameModelFallback: config.SameModelFallbackConfig{
					Enabled: true, Mode: "sequential", Providers: []string{"primary", "alternate"},
				}},
			}
			h := New(cfg)
			for _, id := range []string{"primary", "alternate"} {
				h.providers[id].external.set(sidecar.Snapshot{ID: id, State: provider.StateHealthy, HealthHTTPStatus: http.StatusOK})
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
				bytes.NewBufferString(`{"model":"primary/target-model","messages":[]}`))
			res := httptest.NewRecorder()
			h.Handler().ServeHTTP(res, req)

			if res.Code != tt.wantStatus || alternateCalls != tt.wantAltCalls {
				t.Fatalf("status=%d altCalls=%d; want status=%d calls=%d, body=%s",
					res.Code, alternateCalls, tt.wantStatus, tt.wantAltCalls, res.Body.String())
			}
			if tt.wantAltCalls > 0 && res.Header().Get("X-AhB-Fallback") != "same-model" {
				t.Fatal("successful fallback missing routing identification")
			}
		})
	}
}

func TestConnectionNominatedHeadersAreNotForwarded(t *testing.T) {
	requestSource := http.Header{
		"Connection":  []string{"keep-alive, X-Internal-Token"},
		"X-Internal-Token": []string{"sensitive"},
		"Authorization": []string{"Bearer caller-secret"},
		"X-Api-Key": []string{"caller-key"},
		"X-Custom": []string{"allowed"},
	}
	requestDest := http.Header{}
	copyRequestHeaders(requestDest, requestSource)
	if requestDest.Get("X-Internal-Token") != "" || requestDest.Get("Authorization") != "" ||
		requestDest.Get("X-Api-Key") != "" || requestDest.Get("Connection") != "" {
		t.Fatal("request leaked hop-by-hop or caller credential headers")
	}
	if requestDest.Get("X-Custom") != "allowed" {
		t.Fatal("request lost permitted header")
	}

	responseSource := http.Header{
		"Connection": []string{"X-Private-Control, keep-alive"},
		"X-Private-Control": []string{"do-not-forward"},
		"Content-Type": []string{"text/event-stream"},
	}
	responseDest := http.Header{}
	copyResponseHeaders(responseDest, responseSource)
	if responseDest.Get("X-Private-Control") != "" || responseDest.Get("Connection") != "" {
		t.Fatal("response leaked connection-specific headers")
	}
	if responseDest.Get("Content-Type") != "text/event-stream" {
		t.Fatal("response lost streaming content type")
	}
}

type disconnectedWriter struct{}

func (disconnectedWriter) Header() http.Header { return make(http.Header) }
func (disconnectedWriter) WriteHeader(int) {}
func (disconnectedWriter) Write([]byte) (int, error) { return 0, errors.New("client closed") }

type countingReader struct { calls int }
func (r *countingReader) Read(p []byte) (int, error) {
	r.calls++
	if r.calls > 1 { return 0, io.EOF }
	return copy(p, []byte("chunk")), nil
}

func TestStreamingStopsAfterClientDisconnect(t *testing.T) {
	reader := &countingReader{}
	copyStreaming(disconnectedWriter{}, reader)
	if reader.calls != 1 {
		t.Fatalf("stream consumed upstream after downstream disconnected: %d reads", reader.calls)
	}
}
