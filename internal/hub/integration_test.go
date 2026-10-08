package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
	"github.com/Tsenjii/AhB/internal/provider"
)

func TestSidecarHelperProcess(t *testing.T) {
	if os.Getenv("AIHUB_FAKE_HELPER") != "1" {
		return
	}
	port := os.Getenv("AIHUB_FAKE_PORT")
	name := os.Getenv("AIHUB_FAKE_NAME")
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{{"id": name + "-model", "object": "model"}},
		})
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"provider": name,
			"model":    body["model"],
			"ok":       true,
		})
	})
	server := &http.Server{Addr: "127.0.0.1:" + port, Handler: mux}
	_ = server.ListenAndServe()
}

func TestHubSupervisesAndRoutesSidecars(t *testing.T) {
	portOne := freePort(t)
	portTwo := freePort(t)

	cfg := config.Config{
		Listen: "127.0.0.1:8317",
		Providers: []config.ProviderConfig{
			fakeProvider("one", portOne),
			fakeProvider("two", portTwo),
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := New(cfg)
	h.Start(ctx)

	srv := httptest.NewServer(h.Handler())
	defer srv.Close()

	waitFor(t, 8*time.Second, func() bool {
		return h.providers["one"].sup.Snapshot().State == provider.StateHealthy &&
			h.providers["two"].sup.Snapshot().State == provider.StateHealthy
	})

	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var models struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		t.Fatal(err)
	}
	if len(models.Data) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models.Data))
	}

	body := []byte(`{"model":"one/one-model","messages":[]}`)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	chatResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer chatResp.Body.Close()

	var routed map[string]any
	if err := json.NewDecoder(chatResp.Body).Decode(&routed); err != nil {
		t.Fatal(err)
	}
	if routed["provider"] != "one" || routed["model"] != "one-model" {
		t.Fatalf("unexpected route result: %#v", routed)
	}

	before := h.providers["one"].sup.Snapshot()
	proc, err := os.FindProcess(before.PID)
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatal(err)
	}

	waitFor(t, 8*time.Second, func() bool {
		after := h.providers["one"].sup.Snapshot()
		return after.State == provider.StateHealthy && after.Restarts >= 1 && after.PID != before.PID
	})
}

func fakeProvider(id string, port int) config.ProviderConfig {
	return config.ProviderConfig{
		ID:                    id,
		Enabled:               true,
		Kind:                  "sidecar",
		BaseURL:               "http://127.0.0.1:" + strconv.Itoa(port),
		Binary:                os.Args[0],
		Args:                  []string{"-test.run=TestSidecarHelperProcess"},
		Env:                   map[string]string{"AIHUB_FAKE_HELPER": "1", "AIHUB_FAKE_PORT": strconv.Itoa(port), "AIHUB_FAKE_NAME": id},
		HealthPath:            "/healthz",
		ModelsPath:            "/v1/models",
		MaxRestarts:           2,
		StartupTimeoutSeconds: 5,
		HealthIntervalSeconds: 1,
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func waitFor(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("timeout waiting for condition")
}

func TestHubRoutesExternalLoopbackProvider(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object":"list",
				"data":[]map[string]any{{"id":"arena-model","object":"model"}},
			})
		case "/v1/chat/completions":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"provider":"external-arena",
				"model":body["model"],
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	cfg := config.Config{
		Listen:"127.0.0.1:8317",
		Providers:[]config.ProviderConfig{{
			ID:"lmarena", Enabled:true, Kind:"external",
			BaseURL:upstream.URL, HealthPath:"/v1/models", ModelsPath:"/v1/models",
			HealthIntervalSeconds:1,
		}},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := New(cfg)
	h.Start(ctx)
	srv := httptest.NewServer(h.Handler())
	defer srv.Close()

	waitFor(t, 5*time.Second, func() bool {
		return h.providers["lmarena"].snapshot().State == provider.StateHealthy
	})

	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil { t.Fatal(err) }
	var models struct{ Data []map[string]any `json:"data"` }
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil { t.Fatal(err) }
	_ = resp.Body.Close()
	if len(models.Data) != 1 || models.Data[0]["id"] != "lmarena/arena-model" {
		t.Fatalf("models = %#v", models.Data)
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions",
		bytes.NewBufferString(`{"model":"lmarena/arena-model","messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	chat, err := http.DefaultClient.Do(req)
	if err != nil { t.Fatal(err) }
	defer chat.Body.Close()
	var got map[string]any
	if err := json.NewDecoder(chat.Body).Decode(&got); err != nil { t.Fatal(err) }
	if got["provider"] != "external-arena" || got["model"] != "arena-model" {
		t.Fatalf("route result = %#v", got)
	}
}
