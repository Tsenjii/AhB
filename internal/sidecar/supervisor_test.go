package sidecar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
)

func TestBackoff(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{1, time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{10, 30 * time.Second},
	}
	for _, tc := range cases {
		if got := backoff(tc.attempt); got != tc.want {
			t.Fatalf("attempt %d: got %v want %v", tc.attempt, got, tc.want)
		}
	}
}

func TestWaitReadyTreatsReachable503AsStarted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "model catalog pending", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	s := New(config.ProviderConfig{
		BaseURL:               server.URL,
		HealthPath:            "/healthz",
		StartupTimeoutSeconds: 2,
	})
	ready, err := s.waitReady(context.Background(), make(chan error))
	if !ready {
		t.Fatal("reachable health endpoint must count as process startup")
	}
	if err == nil || !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("expected degraded HTTP 503 result, got %v", err)
	}
}
