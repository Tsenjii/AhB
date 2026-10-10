package hub

import (
 "bytes"
 "context"
 "os/exec"
 "sync/atomic"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "time"

 "github.com/Tsenjii/AhB/internal/config"
 "github.com/Tsenjii/AhB/internal/provider"
 "github.com/Tsenjii/AhB/internal/sidecar"
)

func TestUpstreamHeaderDeadlineKeepsStreamingResponses(t *testing.T) {
 h := New(config.Config{})
 transport, ok := h.client.Transport.(*http.Transport)
 if !ok || transport.ResponseHeaderTimeout != 120*time.Second {
  t.Fatalf("expected 120s header-only deadline, got %v", transport)
 }
 // Compress the header timeout for test speed. The body may legally stream
 // long after headers are received; only a stall before headers times out.
 h.client.Transport = &http.Transport{ResponseHeaderTimeout: 50*time.Millisecond}
 t.Cleanup(h.client.CloseIdleConnections)
 upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  if r.URL.Path != "/v1/chat/completions" { http.NotFound(w, r); return }
  if r.Header.Get("X-Test-Hang") == "yes" {
   select { case <-time.After(250*time.Millisecond): case <-r.Context().Done(): }
   return
  }
  w.Header().Set("Content-Type", "text/event-stream")
  w.WriteHeader(http.StatusOK)
  w.(http.Flusher).Flush()
  time.Sleep(140*time.Millisecond)
  _, _ = w.Write([]byte("data: [DONE]\n\n"))
 }))
 defer upstream.Close()
 h.cfg.Providers = []config.ProviderConfig{{ID: "stable", Enabled: true, Kind: "external", BaseURL: upstream.URL}}
 h.providers["stable"] = &runtimeProvider{cfg: h.cfg.Providers[0], external: newExternalProbe(h.cfg.Providers[0])}
 h.providers["stable"].external.set(sidecar.Snapshot{ID: "stable", State: provider.StateHealthy, HealthHTTPStatus: http.StatusOK})
 request := func(hang bool) *httptest.ResponseRecorder {
  req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"stable/demo","messages":[]}`))
  if hang { req.Header.Set("X-Test-Hang", "yes") }
  recorder := httptest.NewRecorder()
  h.Handler().ServeHTTP(recorder, req)
  return recorder
 }
 valid := request(false)
 if valid.Code != http.StatusOK || !strings.Contains(valid.Body.String(), "data: [DONE]") {
  t.Fatalf("slow SSE after headers must complete: %d %q", valid.Code, valid.Body.String())
 }
 stalled := request(true)
 if stalled.Code != http.StatusBadGateway || !strings.Contains(stalled.Body.String(), "upstream_error") {
  t.Fatalf("stalled upstream headers must release request promptly: %d %q", stalled.Code, stalled.Body.String())
 }
}

// The Hub must give active on-demand streaming requests priority over health
// probe failures that may be caused by transient CPU or I/O saturation.
func TestActiveOnDemandLeaseDefersHealthRestart(t *testing.T) {
 if testing.Short() { t.Skip("integration uses real process health polling") }
 // This test uses the system sleep executable, not user account credentials.
 if _, err := exec.LookPath("sleep"); err != nil { t.Skip("sleep unavailable") }
 var fail atomic.Bool
 upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  if r.URL.Path != "/healthz" { http.NotFound(w, r); return }
  if !fail.Load() { w.WriteHeader(http.StatusOK); return }
  conn, _, err := w.(http.Hijacker).Hijack()
  if err == nil { _ = conn.Close() }
 }))
 defer upstream.Close()
 p := config.ProviderConfig{ID: "stable", Enabled: true, Kind: "sidecar", StartMode: "on_demand",
  Binary: "sleep", Args: []string{"30"}, BaseURL: upstream.URL,
  HealthPath: "/healthz", StartupTimeoutSeconds: 5, HealthIntervalSeconds: 1, MaxRestarts: 5}
 h := New(config.Config{Listen: "127.0.0.1:8317",
  Resources: config.ResourceConfig{MaxRunningSidecars: 2, IdleStopSeconds: 300},
  Providers: []config.ProviderConfig{p}})
 ctx, cancel := context.WithCancel(context.Background())
 h.Start(ctx)
 defer func() { cancel(); h.Wait() }()
 lease, err := h.acquireOnDemand(ctx, h.providers[p.ID])
 if err != nil { t.Fatal(err) }
 released := false
 defer func() { if !released { lease() } }()
 pid := h.providers[p.ID].snapshot().PID
 if pid == 0 { t.Fatal("missing running provider") }
 fail.Store(true)
 // At least three failed 1-second health polls occur while the lease is held.
 deadline := time.Now().Add(6*time.Second)
 for time.Now().Before(deadline) {
  if h.providers[p.ID].snapshot().State == provider.StateDegraded { break }
  time.Sleep(100*time.Millisecond)
 }
 time.Sleep(2800*time.Millisecond)
 snap := h.providers[p.ID].snapshot()
 if snap.PID != pid || snap.Restarts != 0 {
  t.Fatalf("transient health failure killed active stream: %+v; original PID %d", snap, pid)
 }
 lease(); released = true
 deadline = time.Now().Add(5*time.Second)
 for time.Now().Before(deadline) {
  if h.providers[p.ID].snapshot().Restarts > 0 { return }
  time.Sleep(100*time.Millisecond)
 }
 t.Fatal("unhealthy unleased process never restarted after stream finished")
}
