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
const activeModelMaxBytes = 2 << 20
const activeModelMaxItems = 1000

type activeModelEntry struct {
	updated time.Time
	models []map[string]any
	pending chan struct{}
}

type activeModelCatalog struct {
	mu sync.Mutex
	entries map[string]*activeModelEntry
}

func newActiveModelCatalog() *activeModelCatalog {
	return &activeModelCatalog{entries:make(map[string]*activeModelEntry)}
}

func cloneModelList(src []map[string]any) []map[string]any {
	out:=make([]map[string]any,0,len(src))
	for _,m:=range src {
		copied:=make(map[string]any,len(m))
		for k,v:=range m {copied[k]=v}
		out=append(out,copied)
	}
	return out
}

func (h *Hub) fetchActiveModels(ctx context.Context, cfg config.ProviderConfig) ([]map[string]any,error) {
	cache:=h.activeModelCache
	if cache==nil{return h.fetchModels(ctx,cfg)}
	for {
		cache.mu.Lock()
		entry:=cache.entries[cfg.ID]
		if entry!=nil && entry.pending==nil && entry.models!=nil &&
			time.Since(entry.updated)<activeModelTTL {
			result:=cloneModelList(entry.models)
			cache.mu.Unlock()
			return result,nil
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
			entry.updated=time.Now()
		}else{
			// Never return previous successful metadata when a fresh probe
			// failed; visibility should reflect the current upstream state.
			entry.models=nil
			entry.updated=time.Time{}
		}
		entry.pending=nil
		close(done)
		cache.mu.Unlock()
		return models,err
	}
}
