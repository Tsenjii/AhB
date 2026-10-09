package hub

import (
 "context"
 "net/http"
 "strings"
 "time"
)

// A transient inference HTTP 503 can mean quota or provider-side throttling,
// not a broken process. Recovery is explicit and scoped to the selected
// on-demand sidecar: never retry user inference, erase account data or
// blindly restart repeatedly.
func (h *Hub) handleControlRecover(w http.ResponseWriter, r *http.Request) {
 w.Header().Set("Cache-Control","no-store")
 if r.Method!=http.MethodPost {
  w.Header().Set("Allow","POST"); http.Error(w,"POST required",http.StatusMethodNotAllowed);return
 }
 if !h.authorizeLocalControl(w,r) {return}
 id:=strings.TrimPrefix(r.URL.Path,"/api/control/recover/")
 if id==""||strings.ContainsAny(id,"/\\"){
  http.Error(w,"Invalid provider",http.StatusNotFound);return
 }
 p,ok:=h.providers[id]
 if !ok || !p.cfg.Enabled || p.sup==nil || !h.onDemand(p) {
  http.Error(w,"Recovery is supported for enabled on-demand gateways only",http.StatusConflict);return
 }
 h.controlMu.Lock()
 pending:=h.controlPending
 h.controlMu.Unlock()
 if pending{http.Error(w,"A whole-Hub restart is scheduled",http.StatusConflict);return}

 h.demandMu.Lock()
 d:=p.demand
 if d==nil||!d.running||d.closing||d.done==nil||p.snapshot().PID==0 {
  h.demandMu.Unlock()
  http.Error(w,"Gateway is already sleeping; use Wake instead",http.StatusConflict);return
 }
 if d.users>0 {
  h.demandMu.Unlock()
  http.Error(w,"Active chat, SSE or model probe; cannot restart a gateway in use",http.StatusConflict);return
 }
 if time.Since(p.lastRecovery)<45*time.Second {
  h.demandMu.Unlock()
  http.Error(w,"Recovery cooldown: wait at least 45 seconds before trying again",http.StatusTooManyRequests);return
 }
 p.lastRecovery=time.Now()
 d.closing=true
 done,cancel:=d.done,d.cancel
 h.demandMu.Unlock()

 h.stopDemand(p,done,cancel)
 restartContext,stop:=context.WithTimeout(r.Context(),90*time.Second)
 defer stop()
 lease,err:=h.acquireOnDemand(restartContext,p)
 if err!=nil {
  http.Error(w,"Gateway stopped; could not verify restart: "+err.Error(),http.StatusServiceUnavailable)
  return
 }
 defer lease()
 snap:=p.snapshot()
 assessment:=assessProviderHealth(id,p.cfg.Kind,snap,p.accounts.snapshot())
 writeJSON(w,http.StatusOK,map[string]any{
  "status":"restarted","provider":id,"ready":assessment.State=="HEALTHY",
  "message":"Gateway process restarted without deleting credentials; verify account quota and make a new inference request manually.",
 })
}
