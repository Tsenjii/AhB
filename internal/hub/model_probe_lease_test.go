package hub

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
)

// A model-list request must neither wake a sleeper nor let the 512 MiB
// resource manager evict an actively probed on-demand sidecar.
func TestRunningModelProbeHoldsDemandLeaseUnderOneSidecarCap(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("requires sleep executable")
	}
	entered := make(chan struct{}, 1)
	proceed := make(chan struct{})
	var closeOnce sync.Once
	defer closeOnce.Do(func() { close(proceed) })

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(http.StatusOK)
		case "/v1/models":
			select { case entered <- struct{}{}: default: }
			select {
			case <-proceed:
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []map[string]any{{"id": "model-1", "object": "model"}},
				})
			case <-r.Context().Done():
				return
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	provider := func(id string) config.ProviderConfig {
		return config.ProviderConfig{
			ID: id, Enabled: true, Kind: "sidecar", StartMode: "on_demand",
			Binary: "sleep", Args: []string{"35"}, BaseURL: upstream.URL,
			HealthPath: "/healthz", ModelsPath: "/v1/models",
			StartupTimeoutSeconds: 5, HealthIntervalSeconds: 1, MaxRestarts: 1,
		}
	}
	h := New(config.Config{
		Listen: "127.0.0.1:8317",
		Resources: config.ResourceConfig{MaxRunningSidecars: 1, IdleStopSeconds: 1},
		Providers: []config.ProviderConfig{provider("probe"), provider("other")},
	})
	ctx, cancel := context.WithCancel(context.Background())
	h.Start(ctx)
	defer func() { cancel(); h.Wait() }()

	if _, ok := h.leaseRunningDemand(h.providers["probe"]); ok {
		t.Fatal("model query must not acquire a sleeping sidecar")
	}
	if h.providers["probe"].snapshot().PID != 0 {
		t.Fatal("model query woke a sleeping sidecar")
	}
	firstLease, err := h.acquireOnDemand(ctx, h.providers["probe"])
	if err != nil { t.Fatal(err) }
	firstLease() // Provider stays resident during its idle window.

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rec := httptest.NewRecorder()
		h.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
		done <- rec
	}()
	select {
	case <-entered:
	case <-time.After(6*time.Second):
		t.Fatal("model probe failed to reach upstream")
	}

	if _, err := h.acquireOnDemand(ctx, h.providers["other"]); !errors.Is(err, errOnDemandBusy) {
		t.Fatalf("other source could evict a running model probe: %v", err)
	}
	if h.providers["probe"].snapshot().PID == 0 {
		t.Fatal("running model probe was killed by a second provider")
	}

	closeOnce.Do(func() { close(proceed) })
	select {
	case response := <-done:
		if response.Code != http.StatusOK {
			t.Fatalf("model listing failed: HTTP %d: %s", response.Code, response.Body.String())
		}
		var doc struct { Data []map[string]any `json:"data"` }
		if err := json.Unmarshal(response.Body.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		if len(doc.Data) != 1 || doc.Data[0]["id"] != "probe/model-1" {
			t.Fatalf("model discovery lost the provider result: %+v", doc.Data)
		}
	case <-time.After(6*time.Second):
		t.Fatal("model listing blocked after release")
	}
	// Probe released its lease, so the other service can wake under cap=1.
	secondLease, err := h.acquireOnDemand(ctx, h.providers["other"])
	if err != nil {
		t.Fatalf("completed model probe still occupies one-provider capacity: %v", err)
	}
	secondLease()
}
