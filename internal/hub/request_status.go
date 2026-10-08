package hub

import (
 "sync"
 "time"
)

// RequestResult reflects the last HTTP outcome from a real /v1 request,
// not a liveness poll, account credential probe or inference quota forecast.
// It intentionally stores no URLs, model names, prompts, headers or tokens.
type requestResult struct {
 Status int
 At time.Time
 TransportError bool
}

type requestResultState struct {
 mu sync.RWMutex
 last requestResult
}

func (s *requestResultState) record(status int, transportError bool) {
 s.mu.Lock()
 s.last = requestResult{Status:status, TransportError:transportError, At:time.Now().UTC()}
 s.mu.Unlock()
}

func (s *requestResultState) snapshot() requestResult {
 s.mu.RLock()
 defer s.mu.RUnlock()
 return s.last
}
