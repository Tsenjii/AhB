package hub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
)

// A model catalog is discovery metadata, never an account/quota entitlement.
// Cache only a small allowlist of model metadata: no arbitrary upstream
// payloads, API credentials, prompts, user details, or provider config files.
const modelCacheTTL = 6 * time.Hour
const modelCacheMaxBytes = 2 << 20
const modelCacheMaxModels = 1000

type savedModelCatalog struct {
	UpdatedAt time.Time        `json:"updated_at"`
	Models    []map[string]any `json:"models"`
}
type savedModelCatalogFile struct {
	Version  int                          `json:"version"`
	Entries  map[string]savedModelCatalog `json:"entries"`
}

type modelCatalogCache struct {
	mu         sync.RWMutex
	entries    map[string]savedModelCatalog
	path       string
	lastStored time.Time
}

func newModelCatalogCache() *modelCatalogCache {
	return &modelCatalogCache{entries: make(map[string]savedModelCatalog)}
}

func cacheCatalogAllowed(entry savedModelCatalog, now time.Time) bool {
	if entry.UpdatedAt.IsZero() || entry.UpdatedAt.After(now.Add(5*time.Minute)) {
		return false
	}
	return now.Sub(entry.UpdatedAt) <= modelCacheTTL
}

func safeCachedModel(cfg config.ProviderConfig, original map[string]any) (map[string]any, bool) {
	id, _ := original["id"].(string)
	// Upstream IDs have already been namespaced in fetchModels.
	if id == "" || !strings.HasPrefix(id, cfg.ID+"/") || strings.TrimSpace(strings.TrimPrefix(id, cfg.ID+"/")) == "" {
		return nil, false
	}
	// All cached content is a fixed allowlist. Never persist arbitrary fields
	// returned by a third-party models endpoint.
	out := map[string]any{
		"id": id, "object": "model", "x_provider": cfg.ID,
		"x_provider_name": cfg.DisplayName,
		"x_upstream_id": strings.TrimPrefix(id, cfg.ID+"/"),
		"x_catalog_only": true,
	}
	for _, key := range []string{"context_window", "context_length", "max_input", "max_output"} {
		switch v := original[key].(type) {
		case float64:
			if v >= 0 { out[key] = v }
		case int:
			if v >= 0 { out[key] = v }
		case int64:
			if v >= 0 { out[key] = v }
		}
	}
	return out, true
}

func (c *modelCatalogCache) load(path string, providers map[string]*runtimeProvider) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.path = path
	stat, err := os.Stat(path)
	if err != nil || stat.Size() > modelCacheMaxBytes || stat.Size() < 1 {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil { return }
	var saved savedModelCatalogFile
	if json.Unmarshal(raw, &saved) != nil || saved.Version != 1 { return }
	now := time.Now()
	for id, entry := range saved.Entries {
		p := providers[id]
		if p == nil || !p.cfg.Enabled || !(p.cfg.Kind == "sidecar" && p.cfg.StartMode == "on_demand") || !cacheCatalogAllowed(entry, now) {
			continue
		}
		clean := make([]map[string]any, 0, len(entry.Models))
		for _, model := range entry.Models {
			if m, ok := safeCachedModel(p.cfg, model); ok {
				clean = append(clean, m)
			}
			if len(clean) >= modelCacheMaxModels { break }
		}
		if len(clean) > 0 {
			c.entries[id] = savedModelCatalog{UpdatedAt: entry.UpdatedAt, Models: clean}
		}
	}
	// Avoid rewriting the same catalog on every one-minute dashboard refresh.
	c.lastStored = now
}

// The runtime source of truth is always a live /v1/models response.
// In-memory and persisted entries are used only when an enabled, on-demand
// provider is sleeping; the Hub still validates runtime/account readiness
// for all inference requests.
func (c *modelCatalogCache) record(cfg config.ProviderConfig, models []map[string]any) {
	now := time.Now()
	clean := make([]map[string]any, 0, len(models))
	for _, model := range models {
		if entry, ok := safeCachedModel(cfg, model); ok {
			clean = append(clean, entry)
		}
		if len(clean) >= modelCacheMaxModels { break }
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	previous := c.entries[cfg.ID]
	c.entries[cfg.ID] = savedModelCatalog{UpdatedAt: now, Models: clean}
	if c.path == "" || (reflect.DeepEqual(previous.Models, clean) && now.Sub(c.lastStored) < 10*time.Minute) {
		return
	}
	data, err := json.Marshal(savedModelCatalogFile{Version: 1, Entries: c.entries})
	if err != nil || len(data) > modelCacheMaxBytes { return }
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil { return }
	f, err := os.CreateTemp(dir, ".ahb-model-cache-*")
	if err != nil { return }
	name := f.Name()
	defer os.Remove(name)
	if f.Chmod(0600) != nil { f.Close(); return }
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil { return }
	if os.Rename(name, c.path) == nil { c.lastStored = now }
}

func (c *modelCatalogCache) sleeping(id string) ([]map[string]any, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[id]
	if !ok || !cacheCatalogAllowed(entry, time.Now()) || len(entry.Models) == 0 {
		return nil, time.Time{}
	}
	out := make([]map[string]any, 0, len(entry.Models))
	for _, m := range entry.Models {
		copied := make(map[string]any, len(m)+3)
		for k, v := range m { copied[k] = v }
		copied["x_cached"] = true
		copied["x_cache_state"] = "sleeping_unverified"
		copied["x_cached_at"] = entry.UpdatedAt.UTC().Format(time.RFC3339)
		out = append(out, copied)
	}
	return out, entry.UpdatedAt
}
