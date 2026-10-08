package sidecar

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Tsenjii/android-ai-hub/internal/config"
	"github.com/Tsenjii/android-ai-hub/internal/provider"
)

type Snapshot struct {
	ID        string         `json:"id"`
	State     provider.State `json:"state"`
	PID       int            `json:"pid,omitempty"`
	RSSBytes  int64          `json:"rss_bytes,omitempty"`
	Restarts  int            `json:"restarts"`
	LastError string         `json:"last_error,omitempty"`
}

type Supervisor struct {
	spec config.ProviderConfig

	mu        sync.RWMutex
	state     provider.State
	cmd       *exec.Cmd
	lastError string
	restarts  int

	client *http.Client
}

func New(spec config.ProviderConfig) *Supervisor {
	return &Supervisor{
		spec:  spec,
		state: provider.StateStopped,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *Supervisor) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pid := 0
	if s.cmd != nil && s.cmd.Process != nil {
		pid = s.cmd.Process.Pid
	}
	rss := int64(0)
	if pid > 0 {
		rss = processRSSBytes(pid)
	}
	return Snapshot{
		ID: s.spec.ID, State: s.state, PID: pid, RSSBytes: rss,
		Restarts: s.restarts, LastError: s.lastError,
	}
}

func (s *Supervisor) Run(ctx context.Context) {
	attempt := 0
	for ctx.Err() == nil {
		s.setState(provider.StateStarting, "")
		exitCh, cleanup, err := s.startProcess(ctx)
		if err != nil {
			s.setState(provider.StateDegraded, err.Error())
			attempt++
			if attempt > s.spec.MaxRestarts {
				s.setState(provider.StateDead, err.Error())
				return
			}
			if !sleepContext(ctx, backoff(attempt)) {
				return
			}
			continue
		}

		ready, err := s.waitReady(ctx, exitCh)
		if !ready {
			cleanup()
			attempt++
			if err == nil {
				err = fmt.Errorf("startup failed")
			}
			s.incrementRestart(err)
			if attempt > s.spec.MaxRestarts {
				s.setState(provider.StateDead, err.Error())
				return
			}
			if !sleepContext(ctx, backoff(attempt)) {
				return
			}
			continue
		}

		s.setState(provider.StateHealthy, "")
		stableSince := time.Now()
		healthTicker := time.NewTicker(time.Duration(s.spec.HealthIntervalSeconds) * time.Second)
		failures := 0
		restart := false

		for !restart {
			select {
			case <-ctx.Done():
				healthTicker.Stop()
				cleanup()
				s.setState(provider.StateStopped, "")
				return
			case err := <-exitCh:
				healthTicker.Stop()
				cleanup()
				if err == nil {
					err = fmt.Errorf("process exited")
				}
				s.incrementRestart(err)
				restart = true
			case <-healthTicker.C:
				if err := s.checkHealth(ctx); err != nil {
					failures++
					s.setState(provider.StateDegraded, err.Error())
					if failures >= 3 {
						healthTicker.Stop()
						cleanup()
						s.incrementRestart(fmt.Errorf("health check failed 3 times: %w", err))
						restart = true
					}
					continue
				}
				failures = 0
				s.setState(provider.StateHealthy, "")
			}
		}

		if time.Since(stableSince) >= 5*time.Minute {
			attempt = 0
		}
		attempt++
		if attempt > s.spec.MaxRestarts {
			s.setState(provider.StateDead, s.Snapshot().LastError)
			return
		}
		if !sleepContext(ctx, backoff(attempt)) {
			return
		}
	}
	s.setState(provider.StateStopped, "")
}

func (s *Supervisor) startProcess(ctx context.Context) (<-chan error, func(), error) {
	if err := os.MkdirAll("logs", 0o700); err != nil {
		return nil, nil, fmt.Errorf("create logs directory: %w", err)
	}
	logPath := filepath.Join("logs", s.spec.ID+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open sidecar log: %w", err)
	}

	cmd := exec.Command(s.spec.Binary, s.spec.Args...)
	if s.spec.WorkDir != "" {
		if err := os.MkdirAll(s.spec.WorkDir, 0o700); err != nil {
			_ = logFile.Close()
			return nil, nil, fmt.Errorf("create work directory: %w", err)
		}
		cmd.Dir = s.spec.WorkDir
	}
	cmd.Env = mergedEnv(os.Environ(), s.spec.Env)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, nil, fmt.Errorf("start %s: %w", s.spec.ID, err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()

	exitCh := make(chan error, 1)
	go func() {
		exitCh <- cmd.Wait()
		close(exitCh)
	}()

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(os.Interrupt)
				select {
				case <-exitCh:
				case <-time.After(3 * time.Second):
					_ = cmd.Process.Kill()
					select {
					case <-exitCh:
					case <-time.After(2 * time.Second):
					}
				}
			}
			_ = logFile.Close()
			s.mu.Lock()
			if s.cmd == cmd {
				s.cmd = nil
			}
			s.mu.Unlock()
		})
	}
	return exitCh, cleanup, nil
}

func (s *Supervisor) waitReady(ctx context.Context, exitCh <-chan error) (bool, error) {
	deadline := time.NewTimer(time.Duration(s.spec.StartupTimeoutSeconds) * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case err := <-exitCh:
			if err == nil {
				err = fmt.Errorf("process exited before readiness")
			}
			return false, err
		case <-deadline.C:
			return false, fmt.Errorf("startup timeout after %ds", s.spec.StartupTimeoutSeconds)
		case <-ticker.C:
			if err := s.checkHealth(ctx); err == nil {
				return true, nil
			}
		}
	}
}

func (s *Supervisor) checkHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.spec.BaseURL, "/")+"/"+strings.TrimLeft(s.spec.HealthPath, "/"), nil)
	if err != nil {
		return err
	}
	for k, v := range s.spec.Headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *Supervisor) incrementRestart(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restarts++
	if err != nil {
		s.lastError = err.Error()
	}
	s.state = provider.StateDegraded
}

func (s *Supervisor) setState(state provider.State, lastError string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
	if lastError != "" {
		s.lastError = lastError
	}
	if state == provider.StateHealthy {
		s.lastError = ""
	}
}

func mergedEnv(base []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return base
	}
	out := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		key, _, _ := strings.Cut(item, "=")
		if _, replaced := overrides[key]; replaced {
			continue
		}
		out = append(out, item)
	}
	for key, value := range overrides {
		out = append(out, key+"="+value)
	}
	return out
}

func backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := time.Second << (attempt - 1)
	if d > 30*time.Second {
		return 30 * time.Second
	}
	return d
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func processRSSBytes(pid int) int64 {
	f, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil || kb < 0 {
			return 0
		}
		return kb * 1024
	}
	return 0
}