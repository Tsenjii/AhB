package hub

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type runtimeStats struct {
	GoVersion   string `json:"go_version"`
	GOOS        string `json:"goos"`
	GOARCH      string `json:"goarch"`
	Goroutines  int    `json:"goroutines"`
	HeapBytes   uint64 `json:"heap_bytes"`
	ProcessRSS  int64  `json:"process_rss_bytes,omitempty"`
	MaxRunningSidecars int `json:"max_running_sidecars,omitempty"`
	IdleStopSeconds int `json:"idle_stop_seconds,omitempty"`
	RunningOnDemand int `json:"running_on_demand"`
}

func collectRuntimeStats() runtimeStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return runtimeStats{
		GoVersion:  runtime.Version(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		Goroutines: runtime.NumGoroutine(),
		HeapBytes:  m.HeapAlloc,
		ProcessRSS: selfRSSBytes(),
	}
}

func selfRSSBytes() int64 {
	f, err := os.Open("/proc/self/status")
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