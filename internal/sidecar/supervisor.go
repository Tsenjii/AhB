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

	"github.com/Tsenjii/AhB/internal/config"
	"github.com/Tsenjii/AhB/internal/provider"
)

type Snapshot struct {
	ID               string         `json:"id"`
	State            provider.State `json:"state"`
	PID              int            `json:"pid,omitempty"`
	RSSBytes         int64          `json:"rss_bytes,omitempty"`
	Restarts         int            `json:"restarts"`
	LastError        string         `json:"last_error,omitempty"`
	HealthHTTPStatus int            `json:"-"`
	HealthBody       []byte         `json:"-"`
}

type Supervisor struct {
	spec config.ProviderConfig

	mu        sync.RWMutex
	state     provider.State
	cmd       *exec.Cmd
	lastError        string
	restarts         int
	lastHealthStatus int
	lastHealthBody   []byte

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
	healthBody := append([]byte(nil), s.lastHealthBody...)
	return Snapshot{
		ID: s.spec.ID, State: s.state, PID: pid, RSSBytes: rss,
		Restarts: s.restarts, LastError: s.lastError,
		HealthHTTPStatus: s.lastHealthStatus, HealthBody: healthBody,
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

		// A provider may expose its HTTP listener before upstream model discovery
		// is ready. Keep the live process running in DEGRADED instead of killing it
		// and resetting slow startup work (notably opencode2api on Android).
		if err != nil {
			s.setState(provider.StateDegraded, err.Error())
		} else {
			s.setState(provider.StateHealthy, "")
		}
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
				status, err := s.probeHealth(ctx)
				if err != nil {
					failures++
					s.setState(provider.StateDegraded, err.Error())
					if failures >= 3 {
						healthTicker.Stop()
						cleanup()
						s.incrementRestart(fmt.Errorf("health endpoint unreachable 3 times: %w", err))
						restart = true
					}
					continue
				}
				// An HTTP response proves the sidecar is alive. 4xx/5xx health
				// statuses mean "not ready" and should not cause a restart loop.
				failures = 0
				if status < 200 || status >= 300 {
					s.setState(provider.StateDegraded, fmt.Sprintf("health returned HTTP %d", status))
					continue
				}
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
	logFile, err := openCappedLogFile(logPath)
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
	// Best-effort process-wide egress default. Native provider account proxy
	// pools still take precedence if their implementation supports them.
	// Explicitly exclude local Hub/admin ports and OAuth callback addresses.
	if s.spec.ProxyURL != "" {
		overrides := map[string]string{
			"HTTP_PROXY":s.spec.ProxyURL, "HTTPS_PROXY":s.spec.ProxyURL, "ALL_PROXY":s.spec.ProxyURL,
			"http_proxy":s.spec.ProxyURL, "https_proxy":s.spec.ProxyURL, "all_proxy":s.spec.ProxyURL,
		}
		noProxy := "localhost,127.0.0.1,::1,[::1]"
		for _,entry:=range cmd.Env {
			if strings.HasPrefix(strings.ToLower(entry),"no_proxy=") {
				parts:=strings.SplitN(entry,"=",2)
				if len(parts)==2&&parts[1]!="" {noProxy+=","+parts[1]}
				break
			}
		}
		overrides["NO_PROXY"]=noProxy
		overrides["no_proxy"]=noProxy
		cmd.Env = mergedEnv(cmd.Env, overrides)
	}
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
			status, err := s.probeHealth(ctx)
			if err != nil {
				continue
			}
			if status < 200 || status >= 300 {
				return true, fmt.Errorf("health returned HTTP %d", status)
			}
			return true, nil
		}
	}
}

func (s *Supervisor) checkHealth(ctx context.Context) error {
	status, err := s.probeHealth(ctx)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("health returned HTTP %d", status)
	}
	return nil
}

func (s *Supervisor) probeHealth(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.spec.BaseURL, "/")+"/"+strings.TrimLeft(s.spec.HealthPath, "/"), nil)
	if err != nil {
		return 0, err
	}
	for k, v := range s.spec.Headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		s.recordHealth(0, nil)
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	s.recordHealth(resp.StatusCode, body)
	return resp.StatusCode, nil
}

func (s *Supervisor) recordHealth(status int, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastHealthStatus = status
	s.lastHealthBody = append(s.lastHealthBody[:0], body...)
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