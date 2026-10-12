package hub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
)

func TestFlightRecorderBoundedNewestFirstAndRedacted(t *testing.T) {
	f := newFlightRecorder()
	for i := 0; i < flightCapacity+21; i++ {
		f.record(flightEvent{
			At: time.Unix(int64(i+1), 0).UTC(),
			Endpoint: "/v1/chat/completions", Provider: "local",
			Status: http.StatusOK, Category: "success",
			DurationMS: int64(i), Bytes: 3,
		})
	}
	s := f.snapshot()
	if s.Requests != flightCapacity || s.Successes != flightCapacity ||
		len(s.Recent) != flightRecentLimit {
		t.Fatalf("invalid ring count: %+v", s)
	}
	if s.Recent[0].DurationMS != flightCapacity+20 ||
		s.Recent[len(s.Recent)-1].DurationMS != flightCapacity+21-flightRecentLimit {
		t.Fatalf("not newest-first: %+v", s.Recent)
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"prompt", "response_body", "authorization", "remote_addr", "token", "model_id"} {
		// Descriptive privacy notice deliberately mentions several terms.
		// Forbidden keys must never be present.
		if strings.Contains(string(data), "\""+leak+"\":") {
			t.Fatalf("sensitive key %s in snapshot", leak)
		}
	}
}

func TestFlightRecorderMiddlewareClassificationAndStream(t *testing.T) {
	h := New(config.Config{Listen: "127.0.0.1:8317"})
	h.providers["good"] = &runtimeProvider{cfg: config.ProviderConfig{ID: "good"}}
	wrapped := h.flightRecorderMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-AhB-Provider", "good")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, "data: hello\n\n")
		w.(http.Flusher).Flush()
	}))
	req := httptest.NewRequest("POST", "http://127.0.0.1:8317/v1/chat/completions?api_key=do-not-store", strings.NewReader("secret user prompt"))
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	s := h.flight.snapshot()
	if rec.Code != 429 || !rec.Flushed || len(s.Recent) != 1 {
		t.Fatalf("stream or event missing: status=%d flushed=%v events=%d", rec.Code, rec.Flushed, len(s.Recent))
	}
	event := s.Recent[0]
	if event.Status != 429 || event.Category != "rate_limited" ||
		event.Provider != "good" || event.Bytes != int64(len("data: hello\n\n")) {
		t.Fatalf("bad inference metadata: %+v", event)
	}
	if strings.Contains(event.Endpoint, "api_key") || strings.Contains(event.Endpoint, "prompt") {
		t.Fatalf("endpoint leaked private input: %q", event.Endpoint)
	}
	// Control/API discovery GET traffic is not an inference.
	wrapped.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "http://127.0.0.1:8317/v1/models", nil))
	if h.flight.snapshot().Requests != 1 {
		t.Fatal("model inventory must not be counted as inference")
	}
}

func TestFlightRecorderP95AndCategory(t *testing.T) {
	f := newFlightRecorder()
	for i := 1; i <= 20; i++ {
		status := http.StatusOK
		if i%4 == 0 {
			status = http.StatusServiceUnavailable
		}
		f.record(flightEvent{Provider: "test", Status: status, DurationMS: int64(i * 10), Category: flightCategory(status)})
	}
	s := f.snapshot()
	if s.Requests != 20 || s.Successes != 15 || s.Failures != 5 ||
		s.SuccessRate != 75 || s.P95MS != 190 || len(s.ByProvider) != 1 ||
		s.ByProvider[0].P95MS != 190 || s.ByProvider[0].LastStatus != http.StatusServiceUnavailable {
		t.Fatalf("unexpected percentile/stats: %+v", s)
	}
	for status, expected := range map[int]string{
		418: "upstream_rejected", 429: "rate_limited",
		401: "authentication", 502: "gateway_unavailable",
		503: "gateway_unavailable", 400: "request_error",
		500: "server_error", 200: "success",
	} {
		if got := flightCategory(status); got != expected {
			t.Fatalf("%d classified as %s, wanted %s", status, got, expected)
		}
	}
}

func TestFlightInsightsLocalReadOnlyAuth(t *testing.T) {
	h := New(config.Config{Listen: "127.0.0.1:8317"})
	makeRequest := func(remote, host, token, origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8317/api/insights", nil)
		req.Host = host
		req.RemoteAddr = remote
		req.Header.Set("X-AhB-Control-Token", token)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		rec := httptest.NewRecorder()
		h.Handler().ServeHTTP(rec, req)
		return rec
	}
	allowed := makeRequest("127.0.0.1:4567", "127.0.0.1:8317", h.controlToken, "")
	if allowed.Code != http.StatusOK || !strings.Contains(allowed.Body.String(), "\"sample_window\":256") {
		t.Fatalf("loopback authorized read failed: %d %s", allowed.Code, allowed.Body.String())
	}
	for label, args := range map[string][4]string{
		"wrong token": {"127.0.0.1:4567", "127.0.0.1:8317", "wrong", ""},
		"external client": {"10.0.0.50:4567", "127.0.0.1:8317", h.controlToken, ""},
		"external host": {"127.0.0.1:4567", "evil.test:8317", h.controlToken, ""},
		"cross origin": {"127.0.0.1:4567", "127.0.0.1:8317", h.controlToken, "http://evil.test"},
	} {
		if rec := makeRequest(args[0], args[1], args[2], args[3]); rec.Code != http.StatusForbidden {
			t.Fatalf("%s unexpectedly allowed: %d", label, rec.Code)
		}
	}
	h.cfg.AllowLAN = true
	if rec := makeRequest("127.0.0.1:4567", "127.0.0.1:8317", h.controlToken, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("LAN-exposed diagnostics must be disabled: %d", rec.Code)
	}
}

func TestFlightRecorderConcurrent(t *testing.T) {
	f := newFlightRecorder()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				f.record(flightEvent{Provider: "good", Status: http.StatusOK, Category: "success"})
				_ = f.snapshot()
			}
		}()
	}
	wg.Wait()
	if got := f.snapshot(); got.Requests != flightCapacity {
		t.Fatalf("concurrent recorder lost ring bounds: %d", got.Requests)
	}
}
