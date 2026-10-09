package hub

import (
 "context"
 "errors"
 "fmt"
 "net/http"
 "strings"
 "sync"
 "time"

 "github.com/Tsenjii/AhB/internal/provider"
)

var errOnDemandBusy = errors.New("512 MB lightweight mode: another provider is active; wait for its request to finish or switch it off")
var errHubNotStarted = errors.New("AhB runtime is not started")

// The demand manager retains a whole upstream process only while it is useful.
// Every request holds a lease through its *entire* SSE response. Neither an
// idle timer nor a second provider request may kill an active response.
type demandState struct {
 cancel context.CancelFunc
 done chan struct{}
 running bool
 closing bool
 users int
 last time.Time
}

// demandLifecycle assumes h.demandMu is held unless otherwise documented.
// It never handles credential data and never modifies Agent2API itself.
func (h *Hub) onDemand(p *runtimeProvider) bool {
 return p.cfg.Kind=="sidecar" && p.cfg.StartMode=="on_demand"
}

func (h *Hub) clearFinishedDemandLocked(p *runtimeProvider) {
 d:=p.demand
 if d==nil||!d.running||d.closing||d.done==nil {return}
 select {
 case <-d.done:
  d.running=false
  d.cancel=nil
  d.done=nil
  d.users=0
 default:
 }
}

func (h *Hub) maxDemandSidecars() int {
 n:=h.cfg.Resources.MaxRunningSidecars
 if n<=0{return 1}
 return n
}

func (h *Hub) demandIdle() time.Duration {
 n:=h.cfg.Resources.IdleStopSeconds
 if n<=0 {n=120}
 return time.Duration(n)*time.Second
}

func (h *Hub) finishDemand(p *runtimeProvider, done chan struct{}) {
 h.demandMu.Lock()
 if p.demand!=nil && p.demand.done==done {
  p.demand.running=false
  p.demand.closing=false
  p.demand.cancel=nil
  p.demand.done=nil
  p.demand.users=0
 }
 h.demandMu.Unlock()
}

func (h *Hub) stopDemand(p *runtimeProvider, done chan struct{}, cancel context.CancelFunc) {
 cancel()
 select {
 case <-done:
  h.finishDemand(p,done)
 case <-time.After(6*time.Second):
  // Keep the old process counted until it really exits. If stopping takes
  // longer than six seconds, eventually clear its closing state instead of
  // leaving this provider permanently unavailable until a full Hub restart.
  go func() {
   <-done
   h.finishDemand(p,done)
  }()
 }
}

// leaseRunningDemand protects a model-list probe against idle reap / capacity
// eviction without waking a sleeping sidecar. A plain /v1/models request must
// never use acquireOnDemand, which would consume 512 MiB capacity on polling.
func (h *Hub) leaseRunningDemand(p *runtimeProvider) (func(), bool) {
 if !h.onDemand(p) { return func(){}, true }
 h.demandMu.Lock()
 d:=p.demand
 if d==nil || !d.running || d.closing || d.done==nil {
  h.demandMu.Unlock()
  return nil,false
 }
 select {
 case <-d.done:
  h.demandMu.Unlock()
  return nil,false
 default:
 }
 d.users++
 done:=d.done
 h.demandMu.Unlock()
 var once sync.Once
 return func(){
  once.Do(func(){
   h.demandMu.Lock()
   if p.demand!=nil && p.demand.done==done && p.demand.users>0 {
    p.demand.users--
    p.demand.last=time.Now()
   }
   h.demandMu.Unlock()
  })
 },true
}

func (h *Hub) acquireOnDemand(ctx context.Context, p *runtimeProvider) (func(),error) {
 noop:=func(){}
 if !h.onDemand(p) {return noop,nil}
 if err:=ctx.Err();err!=nil{return nil,err}

 h.demandMu.Lock()
 if h.demandContext==nil || h.demandContext.Err()!=nil {
  h.demandMu.Unlock();return nil,errHubNotStarted
 }
 if p.demand==nil {p.demand=&demandState{}}
 h.clearFinishedDemandLocked(p)
 if p.demand.closing {h.demandMu.Unlock();return nil,errOnDemandBusy}

 coldStarted:=false
 if !p.demand.running {
  coldStarted=true
  count:=0
  var victim *runtimeProvider
  var oldest time.Time
  for _,other:=range h.providers {
   if !h.onDemand(other) || other.demand==nil {continue}
   h.clearFinishedDemandLocked(other)
   d:=other.demand
   if !d.running {continue}
   count++
   if !d.closing && d.users==0 && (victim==nil||d.last.Before(oldest)){
    victim=other
    oldest=d.last
   }
  }
  if count>=h.maxDemandSidecars(){
   if victim==nil {h.demandMu.Unlock();return nil,errOnDemandBusy}
   d:=victim.demand
   d.closing=true
   done,cancel:=d.done,d.cancel
   h.demandMu.Unlock()
   h.stopDemand(victim,done,cancel)
   // Re-check in the next lock acquisition, never exceed resource ceiling.
   return h.acquireOnDemand(ctx,p)
  }
  runCtx,cancel:=context.WithCancel(h.demandContext)
  done:=make(chan struct{})
  p.demand=&demandState{cancel:cancel,done:done,running:true,last:time.Now()}
  h.runWG.Add(1)
  go func(){
   defer h.runWG.Done()
   p.sup.Run(runCtx)
   close(done)
  }()
 }
 d:=p.demand
 d.users++
 done:=d.done
 h.demandMu.Unlock()

 var once sync.Once
 release:=func(){once.Do(func(){
  h.demandMu.Lock()
  if p.demand!=nil && p.demand.done==done && p.demand.users>0 {
   p.demand.users--
   p.demand.last=time.Now()
  }
  h.demandMu.Unlock()
 })}

 startupSeconds:=p.cfg.StartupTimeoutSeconds
 if startupSeconds<=0 {startupSeconds=30}
 deadline:=time.NewTimer(time.Duration(startupSeconds)*time.Second)
 defer deadline.Stop()
 ticker:=time.NewTicker(200*time.Millisecond)
 defer ticker.Stop()
 for{
  snapshot:=p.sup.Snapshot()
  if snapshot.State==provider.StateHealthy ||
   (snapshot.State==provider.StateDegraded && snapshot.PID>0 && snapshot.HealthHTTPStatus>0) {
   // The background probe is intentionally suspended while asleep; perform
   // one immediately after cold startup instead of rejecting the first chat
   // for up to 10 seconds because account readiness has not been refreshed.
   if accountStatusRequired(p.cfg.ID) && (coldStarted || !p.accounts.snapshot().Known) {
    h.refreshProviderAccounts(ctx,p)
   }
   return release,nil
  }
  select{
  case <-ctx.Done():
   release();return nil,ctx.Err()
  case <-h.demandContext.Done():
   release();return nil,errHubNotStarted
  case <-done:
   release();return nil,fmt.Errorf("provider %s stopped before becoming ready",p.cfg.ID)
  case <-deadline.C:
   release();return nil,fmt.Errorf("provider %s startup timed out",p.cfg.ID)
  case <-ticker.C:
  }
 }
}

func (h *Hub) reapDemand(ctx context.Context) {
 ticker:=time.NewTicker(10*time.Second)
 defer ticker.Stop()
 for{
  select{
  case <-ctx.Done():return
  case <-ticker.C:
   var victims []struct{p *runtimeProvider;done chan struct{};cancel context.CancelFunc}
   h.demandMu.Lock()
   for _,p:=range h.providers {
    if !h.onDemand(p)||p.demand==nil {continue}
    h.clearFinishedDemandLocked(p)
    d:=p.demand
    if d.running && !d.closing && d.users==0 && time.Since(d.last)>=h.demandIdle(){
     d.closing=true
     victims=append(victims,struct{p *runtimeProvider;done chan struct{};cancel context.CancelFunc}{p,d.done,d.cancel})
    }
   }
   h.demandMu.Unlock()
   for _,v:=range victims {h.stopDemand(v.p,v.done,v.cancel)}
  }
 }
}

func (h *Hub) wakeProvider(w http.ResponseWriter,r *http.Request) {
 if r.Method!=http.MethodPost {w.Header().Set("Allow","POST");http.Error(w,"POST required",http.StatusMethodNotAllowed);return}
 if !h.authorizeLocalControl(w,r) {return}
 id:=strings.TrimPrefix(r.URL.Path,"/api/control/wake/")
 if id==""||strings.Contains(id,"/"){http.NotFound(w,r);return}
 p:=h.providers[id]
 if p==nil||!p.cfg.Enabled||!h.onDemand(p) {http.Error(w,"Provider unavailable or not on demand",http.StatusConflict);return}
 ctx,cancel:=context.WithTimeout(r.Context(),time.Duration(p.cfg.StartupTimeoutSeconds+3)*time.Second)
 defer cancel()
 release,err:=h.acquireOnDemand(ctx,p)
 if err!=nil {http.Error(w,"Unable to start provider: "+err.Error(),http.StatusServiceUnavailable);return}
 defer release()
 // A dashboard wake is an explicit request to load the model catalog, not
 // just start the process. This uses the safe metadata endpoint only (no
 // chat, no session creation) and holds the process lease until it finishes.
 result:=map[string]any{"status":"started","id":id,"model_discovery":"unavailable","model_count":0}
 if !providerUsableForRouting(p.cfg.ID,p.cfg.Kind,p.snapshot(),p.accounts.snapshot()) {
  result["message"]="Provider running but has no confirmed account/readiness. Log in with its original management UI."
 } else {
  modelsCtx,modelsCancel:=context.WithTimeout(r.Context(),8*time.Second)
  models,modelsErr:=h.fetchModels(modelsCtx,p.cfg)
  modelsCancel()
  if modelsErr!=nil {
   result["model_discovery"]="failed"
   result["message"]="Provider running; model discovery failed. Existing metadata cache, if any, remains marked unverified."
  } else {
   result["model_discovery"]="ok"
   result["model_count"]=len(models)
   result["message"]="Model metadata discovered; availability/quota must still be verified by inference."
  }
 }
 writeJSON(w,http.StatusOK,result)
}
