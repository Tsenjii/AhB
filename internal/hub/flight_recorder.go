package hub

import (
	"crypto/subtle"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// The flight recorder is deliberately ephemeral and bounded. In particular it
// never reads request bodies, response bodies, model IDs, user IPs, credentials,
// headers (except AhB's fixed routing metadata), or arbitrary URL paths.
const flightCapacity = 256
const flightRecentLimit = 48

type flightEvent struct {
	At          time.Time `json:"at"`
	Endpoint    string    `json:"endpoint"`
	Provider    string    `json:"provider"`
	Status      int       `json:"status"`
	Category    string    `json:"category"`
	DurationMS  int64     `json:"duration_ms"`
	FirstByteMS int64     `json:"first_byte_ms"`
	Bytes       int64     `json:"response_bytes"`
}

type flightProviderStats struct {
	Provider   string `json:"provider"`
	Requests   int    `json:"requests"`
	Successes  int    `json:"successes"`
	Failures   int    `json:"failures"`
	LastStatus int    `json:"last_status"`
	P95MS      int64  `json:"p95_ms"`
}

type flightSnapshot struct {
	Since         time.Time             `json:"since"`
	SampleWindow  int                   `json:"sample_window"`
	Requests      int                   `json:"requests"`
	Successes     int                   `json:"successes"`
	Failures      int                   `json:"failures"`
	SuccessRate   float64               `json:"success_rate"`
	P95MS         int64                 `json:"p95_ms"`
	ByProvider    []flightProviderStats `json:"by_provider"`
	Recent        []flightEvent         `json:"recent"`
	Privacy       string                `json:"privacy"`
}

type flightRecorder struct {
	mu      sync.Mutex
	started time.Time
	entries [flightCapacity]flightEvent
	next    int
	count   int
}

func newFlightRecorder() *flightRecorder {
	return &flightRecorder{started: time.Now().UTC()}
}

func (f *flightRecorder) record(e flightEvent) {
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	if e.DurationMS < 0 {
		e.DurationMS = 0
	}
	if e.FirstByteMS < 0 {
		e.FirstByteMS = 0
	}
	if e.Bytes < 0 {
		e.Bytes = 0
	}
	f.mu.Lock()
	f.entries[f.next] = e
	f.next = (f.next + 1) % flightCapacity
	if f.count < flightCapacity {
		f.count++
	}
	f.mu.Unlock()
}

// snapshot copies the small ring under lock, then aggregates without holding
// up live requests. Request counts cover this ring only, not lifetime totals.
func (f *flightRecorder) snapshot() flightSnapshot {
	f.mu.Lock()
	events := make([]flightEvent, 0, f.count)
	for i := 0; i < f.count; i++ {
		index := (f.next - 1 - i + flightCapacity) % flightCapacity
		events = append(events, f.entries[index])
	}
	since := f.started
	f.mu.Unlock()

	result := flightSnapshot{
		Since: since, SampleWindow: flightCapacity, Requests: len(events),
		Recent: make([]flightEvent, 0, flightRecentLimit),
		ByProvider: []flightProviderStats{},
		Privacy: "Volatile, bounded metadata only. No prompts, responses, model names, tokens, IPs or URLs.",
	}
	providerStats := make(map[string]*flightProviderStats)
	times := make([]int64, 0, len(events))
	providerTimes := make(map[string][]int64)
	for _, event := range events {
		if len(result.Recent) < flightRecentLimit {
			result.Recent = append(result.Recent, event)
		}
		if event.Status >= 200 && event.Status < 300 {
			result.Successes++
		} else {
			result.Failures++
		}
		times = append(times, event.DurationMS)
		key := event.Provider
		if key == "" {
			key = "unattributed"
		}
		stats := providerStats[key]
		if stats == nil {
			stats = &flightProviderStats{Provider: key, LastStatus: event.Status}
			providerStats[key] = stats
		}
		stats.Requests++
		if event.Status >= 200 && event.Status < 300 {
			stats.Successes++
		} else {
			stats.Failures++
		}
		providerTimes[key] = append(providerTimes[key], event.DurationMS)
	}
	if result.Requests > 0 {
		result.SuccessRate = float64(result.Successes) * 100 / float64(result.Requests)
	}
	result.P95MS = flightPercentile95(times)
	for key, stats := range providerStats {
		stats.P95MS = flightPercentile95(providerTimes[key])
		result.ByProvider = append(result.ByProvider, *stats)
	}
	sort.Slice(result.ByProvider, func(i, j int) bool {
		return result.ByProvider[i].Provider < result.ByProvider[j].Provider
	})
	return result
}

func flightPercentile95(durations []int64) int64 {
	if len(durations) == 0 {
		return 0
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	idx := (95*len(durations) + 99) / 100 - 1
	return durations[idx]
}

func flightCategory(status int) string {
	switch status {
	case http.StatusTeapot:
		return "upstream_rejected"
	case http.StatusTooManyRequests:
		return "rate_limited"
	case http.StatusUnauthorized, http.StatusForbidden:
		return "authentication"
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return "gateway_unavailable"
	default:
		switch {
		case status >= 200 && status < 300:
			return "success"
		case status >= 400 && status < 500:
			return "request_error"
		case status >= 500:
			return "server_error"
		default:
			return "other"
		}
	}
}

func flightEndpoint(r *http.Request) string {
	if r.Method != http.MethodPost {
		return ""
	}
	switch r.URL.Path {
	case "/v1/chat/completions", "/v1/completions", "/v1/responses",
		"/v1/messages", "/v1/messages/count_tokens", "/v1/embeddings",
		"/v1/images/generations", "/v1/audio/speech", "/v1/systemone",
		"/api/control/playground":
		return r.URL.Path
	}
	return ""
}

type flightWriter struct {
	http.ResponseWriter
	start       time.Time
	status      int
	bytes       int64
	firstByteMS int64
	wroteBody   bool
}

func (w *flightWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *flightWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if len(p) > 0 && !w.wroteBody {
		w.wroteBody = true
		w.firstByteMS = time.Since(w.start).Milliseconds()
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += int64(n)
	return n, err
}

// Preserve streaming semantics: wrapping a ResponseWriter must not turn SSE
// into a buffered response or suppress Flush calls from copyStreaming.
func (w *flightWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *flightWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (h *Hub) flightRecorderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := flightEndpoint(r)
		if endpoint == "" {
			next.ServeHTTP(w, r)
			return
		}
		writer := &flightWriter{ResponseWriter: w, start: time.Now()}
		next.ServeHTTP(writer, r)
		status := writer.status
		if status == 0 {
			status = http.StatusOK
		}
		provider := writer.Header().Get("X-AhB-Provider")
		if endpoint == "/api/control/playground" {
			provider = writer.Header().Get("X-AhB-Playground-Provider")
		}
		if _, known := h.providers[provider]; !known {
			provider = ""
		}
		h.flight.record(flightEvent{
			At: time.Now().UTC(), Endpoint: endpoint, Provider: provider,
			Status: status, Category: flightCategory(status),
			DurationMS: time.Since(writer.start).Milliseconds(),
			FirstByteMS: writer.firstByteMS, Bytes: writer.bytes,
		})
	})
}

// Read-only observability is stricter than the ordinary public status API.
// Even if AhB deliberately serves /v1 over a LAN, this endpoint is local-only.
func (h *Hub) handleInsights(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	listenHost, _, listenErr := net.SplitHostPort(h.cfg.Listen)
	host, _, hostErr := net.SplitHostPort(r.Host)
	remote, _, remoteErr := net.SplitHostPort(r.RemoteAddr)
	if h.cfg.AllowLAN || h.controlToken == "" || listenErr != nil ||
		!flightLoopbackHost(listenHost) || hostErr != nil ||
		!flightLoopbackHost(host) || remoteErr != nil ||
		!flightLoopbackHost(remote) ||
		subtle.ConstantTimeCompare([]byte(r.Header.Get("X-AhB-Control-Token")), []byte(h.controlToken)) != 1 ||
		(r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+r.Host) {
		http.Error(w, "Local read-only diagnostics unavailable", http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, h.flight.snapshot())
}

func flightLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
