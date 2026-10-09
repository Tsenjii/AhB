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
	AhBRSSBytes int64 `json:"ahb_rss_bytes"`
	SidecarRSSBytes int64 `json:"sidecar_rss_bytes"`
	// Total system RAM or the tighter cgroup quota; avoids advertising host
	// memory as available on a small containerized VPS.
	SystemTotalBytes uint64 `json:"system_total_bytes,omitempty"`
	SystemUsedBytes uint64 `json:"system_used_bytes,omitempty"`
	SystemAvailableBytes uint64 `json:"system_available_bytes,omitempty"`
	SystemMemorySource string `json:"system_memory_source,omitempty"`
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
type memorySnapshot struct {
	total, used, available uint64
	source string
}

// memInfoSnapshot uses MemAvailable (not MemFree) so Linux filesystem
// caches are counted as reclaimable; values are bytes.
func memInfoSnapshot(path string) memorySnapshot {
	raw,err:=os.ReadFile(path)
	if err!=nil {return memorySnapshot{}}
	var total,available uint64
	for _,line:=range strings.Split(string(raw),"\n") {
		fields:=strings.Fields(line)
		if len(fields)<2 {continue}
		n,err:=strconv.ParseUint(fields[1],10,64)
		if err!=nil {continue}
		switch fields[0] {
		case "MemTotal:": total=n*1024
		case "MemAvailable:": available=n*1024
		}
	}
	if total==0 {return memorySnapshot{}}
	if available>total {available=total}
	return memorySnapshot{total:total,used:total-available,available:available,source:"host"}
}

// cgroup v2 current includes cache and kernel use; this is intentionally
// labeled cgroup, not "RAM consumed by AhB". It captures other processes
// sharing the VPS container as well.
func cgroupSnapshot(limitPath,currentPath string) memorySnapshot {
	limitBytes,e1:=os.ReadFile(limitPath)
	currentBytes,e2:=os.ReadFile(currentPath)
	if e1!=nil||e2!=nil {return memorySnapshot{}}
	limit,err1:=strconv.ParseUint(strings.TrimSpace(string(limitBytes)),10,64)
	used,err2:=strconv.ParseUint(strings.TrimSpace(string(currentBytes)),10,64)
	if err1!=nil||err2!=nil||limit==0 {return memorySnapshot{}}
	if used>limit {used=limit}
	return memorySnapshot{total:limit,used:used,available:limit-used,source:"cgroup_v2"}
}

func effectiveMemorySnapshot() memorySnapshot {
	host:=memInfoSnapshot("/proc/meminfo")
	group:=cgroupSnapshot("/sys/fs/cgroup/memory.max","/sys/fs/cgroup/memory.current")
	if group.total>0 && (host.total==0||group.total<host.total) {return group}
	return host
}
