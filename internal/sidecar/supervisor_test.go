package sidecar

import (
	"testing"
	"time"
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