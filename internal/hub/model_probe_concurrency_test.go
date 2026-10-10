package hub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
	"github.com/Tsenjii/AhB/internal/provider"
	"github.com/Tsenjii/AhB/internal/sidecar"
)

func TestGlobalModelProbeCapAcrossConcurrentDashboards(t *testing.T) {
	entered := make(chan struct{}, 1)
	gate := make(chan struct{})
	var closeOnce sync.Once
	defer closeOnce.Do(func() { close(gate) })

	var active, maxSeen, calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		n := active.Add(1)
		defer active.Add(-1)
		calls.Add(1)
		for {
			peak := maxSeen.Load()
			if n <= peak || maxSeen.CompareAndSwap(peak, n) {
				break
			}
		}
		select { case entered <- struct{}{}: default: }
		select {
		case <-gate:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": "example"}},
			})
		case <-r.Context().Done():
		}
	}))
	defer upstream.Close()

	providers := []config.ProviderConfig{
		{ID: "alpha", DisplayName: "Alpha", Enabled: true, Kind: "external",
			BaseURL: upstream.URL, ModelsPath: "/v1/models"},
		{ID: "beta", DisplayName: "Beta", Enabled: true, Kind: "external",
			BaseURL: upstream.URL, ModelsPath: "/v1/models"},
	}
	h := New(config.Config{Listen: "127.0.0.1:8317",
		Resources: config.ResourceConfig{MaxRunningSidecars: 1}, Providers: providers})
	if cap(h.modelProbeSlots) != 1 {
		t.Fatalf("512 MiB profile must probe one source globally, got %d", cap(h.modelProbeSlots))
	}
	for _, p := range h.providers {
		p.external.set(sidecar.Snapshot{ID: p.cfg.ID, State: provider.StateHealthy, HealthHTTPStatus: 200})
	}

	done := make(chan *httptest.ResponseRecorder, 2)
	for i := 0; i < 2; i++ {
		go func() {
			rec := httptest.NewRecorder()
			h.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
			done <- rec
		}()
	}
	select {
	case <-entered:
	case <-time.After(3*time.Second):
		t.Fatal("first model probe never started")
	}
	// Two callers each request two providers, but no more than one upstream
	// model-list call may be in flight across the *whole* Hub.
	time.Sleep(150 * time.Millisecond)
	if peak := maxSeen.Load(); peak != 1 {
		t.Fatalf("concurrent dashboards exceeded model-probe cap: max=%d", peak)
	}
	closeOnce.Do(func() { close(gate) })
	for i := 0; i < 2; i++ {
		select {
		case rec := <-done:
			if rec.Code != http.StatusOK {
				t.Fatalf("model listing failed: %d, %s", rec.Code, rec.Body.String())
			}
			var doc struct { Data []map[string]any `json:"data"` }
			if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil || len(doc.Data) != 2 {
				t.Fatalf("missing provider models: data=%v, err=%v", doc.Data, err)
			}
		case <-time.After(5*time.Second):
			t.Fatal("model requests stayed blocked after releasing slot")
		}
	}
	if calls.Load() != 4 || maxSeen.Load() != 1 {
		t.Fatalf("unexpected request count or maximum: calls=%d max=%d", calls.Load(), maxSeen.Load())
	}

	normal := New(config.Config{Resources: config.ResourceConfig{MaxRunningSidecars: 3}})
	if cap(normal.modelProbeSlots) != 4 {
		t.Fatalf("Android/large-host model discovery should retain four slots, got %d", cap(normal.modelProbeSlots))
	}
}
