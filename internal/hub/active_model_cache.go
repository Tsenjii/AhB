package hub

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
)

// Only active providers participate; sleeping providers continue using the
// explicitly marked persistent metadata cache. This live microcache exists to
// coalesce bursts from multiple dashboards and SDK /v1/models polls, not to
// claim that models are inferable or account quotas exist.
const activeModelTTL = 15 * time.Second
// Brief negative caching prevents dozens of simultaneous dashboards from
// repeatedly hammering a 429/503 backend. Never cache cancelled requests.
const activeModelFailureTTL = 3 * time.Second
const activeModelMaxBytes = 2 << 20
const activeModelMaxItems = 1000

type activeModelEntry struct {
	updated time.Time
	models []map[string]any
	lastError error
	pending chan struct{}
}

type activeModelCatalog struct {
	mu sync.Mutex
	entries map[string]*activeModelEntry
}

func newActiveModelCatalog() *activeModelCatalog {
	return &activeModelCatalog{entries:make(map[string]*activeModelEntry)}
}

// Nested metadata (such as "permission" or "capabilities") is mutable JSON;
// copying the outer map alone would let one SDK caller poison another's cache.
func cloneModelValue(v any) any {
	switch x:=v.(type) {
	case map[string]any:
		copied:=make(map[string]any,len(x))
		for k,item:=range x {copied[k]=cloneModelValue(item)}
		return copied
	case []any:
		copied:=make([]any,len(x))
		for i,item:=range x {copied[i]=cloneModelValue(item)}
		return copied
	case []map[string]any:
		copied:=make([]map[string]any,len(x))
		for i,item:=range x {copied[i]=cloneModelValue(item).(map[string]any)}
		return copied
	default:
		return v
	}
}
func cloneModelList(src []map[string]any) []map[string]any {
	if src==nil{return nil}
	out:=make([]map[string]any,len(src))
	for i,m:=range src {out[i]=cloneModelValue(m).(map[string]any)}
	return out
}

func (h *Hub) fetchActiveModels(ctx context.Context, cfg config.ProviderConfig) ([]map[string]any,error) {
	cache:=h.activeModelCache
	if cache==nil{return h.fetchModels(ctx,cfg)}
	for {
		cache.mu.Lock()
		entry:=cache.entries[cfg.ID]
		if entry!=nil && entry.pending==nil && !entry.updated.IsZero() {
			ttl:=activeModelTTL
			if entry.lastError!=nil {ttl=activeModelFailureTTL}
			if time.Since(entry.updated)<ttl {
				if entry.lastError!=nil {
					err:=entry.lastError
					cache.mu.Unlock()
					return nil,err
				}
				if entry.models!=nil {
					result:=cloneModelList(entry.models)
					cache.mu.Unlock()
					return result,nil
				}
			}
		}
		if entry!=nil && entry.pending!=nil {
			done:=entry.pending
			cache.mu.Unlock()
			select {
			case <-done:
				continue
			case <-ctx.Done():
				return nil,ctx.Err()
			}
		}
		if entry==nil {entry=&activeModelEntry{};cache.entries[cfg.ID]=entry}
		done:=make(chan struct{})
		entry.pending=done
		cache.mu.Unlock()

		models,err:=h.fetchModels(ctx,cfg)
		// Hard cap retained objects. Catalogues larger than the cache budget
		// still return normally but are never kept in memory between requests.
		retain:=false
		if err==nil && len(models)<=activeModelMaxItems {
			if raw,e:=json.Marshal(models);e==nil && len(raw)<=activeModelMaxBytes {
				retain=true
			}
		}
		cache.mu.Lock()
		if retain {
			entry.models=cloneModelList(models)
			entry.lastError=nil
			entry.updated=time.Now()
		}else if err!=nil && ctx.Err()==nil {
			// Coalesce bursts of errors just like successes: do not hammer a
			// quota-limited upstream 40 times because 40 dashboards refreshed.
			// No stale models escape after failure, and retry starts in 3 sec.
			entry.models=nil
			entry.lastError=err
			entry.updated=time.Now()
		}else{
			entry.models=nil
			entry.lastError=nil
			entry.updated=time.Time{}
		}
		entry.pending=nil
		close(done)
		cache.mu.Unlock()
		return models,err
	}
}
