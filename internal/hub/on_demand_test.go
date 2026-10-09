package hub

import (
 "context"
 "net/http"
 "net/http/httptest"
 "os"
 "os/exec"
 "strings"
 "testing"
 "sync/atomic"
 "time"

 "github.com/Tsenjii/AhB/internal/config"
)

func TestOnDemandSleepsUntilUsedAndProtectsActiveStreams(t *testing.T) {
 if _,err:=exec.LookPath("sleep");err!=nil {t.Skip("requires sleep executable")}
 health:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path=="/healthz" {w.WriteHeader(http.StatusOK);return}
  http.NotFound(w,r)
 }))
 defer health.Close()
 mk:=func(id string) config.ProviderConfig {
  return config.ProviderConfig{ID:id,Enabled:true,Kind:"sidecar",StartMode:"on_demand",
   Binary:"sleep",Args:[]string{"30"},BaseURL:health.URL,HealthPath:"/healthz",
   StartupTimeoutSeconds:5,HealthIntervalSeconds:2,MaxRestarts:1}
 }
 cfg:=config.Config{Listen:"127.0.0.1:8317",Resources:config.ResourceConfig{MaxRunningSidecars:1,IdleStopSeconds:30},Providers:[]config.ProviderConfig{mk("first"),mk("second")}}
 h:=New(cfg)
 ctx,cancel:=context.WithCancel(context.Background())
 h.Start(ctx)
 defer func(){cancel();h.Wait()}()
 first:=h.providers["first"]
 second:=h.providers["second"]
 if first.snapshot().PID!=0||second.snapshot().PID!=0 {t.Fatal("on-demand sidecar unexpectedly started at boot")}
 lease,err:=h.acquireOnDemand(context.Background(),first)
 if err!=nil {t.Fatalf("start first: %v",err)}
 if first.snapshot().PID<=0 {t.Fatal("first not running")}
 if _,err:=h.acquireOnDemand(context.Background(),second);err==nil || !strings.Contains(err.Error(),"active") {
  t.Fatalf("expected 512 MB capacity warning while stream active, got %v",err)
 }
 lease()
 lease2,err:=h.acquireOnDemand(context.Background(),second)
 if err!=nil {t.Fatalf("idle first did not release capacity: %v",err)}
 if second.snapshot().PID<=0 {t.Fatal("second not started")}
 lease2()
}

func TestOnDemandStopDoesNotEraseCredentials(t *testing.T) {
 dir:=t.TempDir()
 creds:=dir+"/credentials.json"
 if err:=os.WriteFile(creds,[]byte("{\"account\":\"keep\"}"),0600);err!=nil {t.Fatal(err)}
 h:=New(config.Config{Resources:config.ResourceConfig{MaxRunningSidecars:1,IdleStopSeconds:30}})
 if h.maxDemandSidecars()!=1 {t.Fatal("unsafe process ceiling")}
 if h.demandIdle()!=30*time.Second {t.Fatal("unexpected idle delay")}
 raw,err:=os.ReadFile(creds)
 if err!=nil||string(raw)!="{\"account\":\"keep\"}" {t.Fatal("credentials changed")}
}

func TestSleepingFreeBuffFirstRequestRefreshesAccountReadiness(t *testing.T) {
 if _,err:=exec.LookPath("sleep");err!=nil {t.Skip("sleep unavailable")}
 var probes atomic.Int32
 upstream:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/healthz" {http.NotFound(w,r);return}
  probes.Add(1)
  w.Header().Set("Content-Type","application/json")
  _,_=w.Write([]byte(`{"status":"ok","accounts":1,"alive_accounts":1,"unknown_accounts":0}`))
 }))
 defer upstream.Close()
 cfg:=config.Config{
  Listen:"127.0.0.1:8317",
  Resources:config.ResourceConfig{MaxRunningSidecars:1,IdleStopSeconds:30},
  Providers:[]config.ProviderConfig{{
   ID:"freebuff",Enabled:true,Kind:"sidecar",StartMode:"on_demand",
   Binary:"sleep",Args:[]string{"30"},BaseURL:upstream.URL,HealthPath:"/healthz",
   StartupTimeoutSeconds:5,HealthIntervalSeconds:1,MaxRestarts:1,
  }},
 }
 h:=New(cfg)
 ctx,cancel:=context.WithCancel(context.Background())
 h.Start(ctx)
 defer func(){cancel();h.Wait()}()
 p:=h.providers["freebuff"]
 time.Sleep(150*time.Millisecond)
 if n:=probes.Load();n!=0 {t.Fatalf("sleeping sidecar made %d requests",n)}
 if p.accounts.snapshot().Known {t.Fatal("account status should be unknown before wake")}
 release,err:=h.acquireOnDemand(context.Background(),p)
 if err!=nil {t.Fatalf("wake: %v",err)}
 defer release()
 account:=p.accounts.snapshot()
 if !account.Known||account.Usable!=1||account.Total!=1 {
  t.Fatalf("first request still sees stale account state: %+v",account)
 }
 if !providerUsableForRouting("freebuff",p.cfg.Kind,p.snapshot(),account) {
  t.Fatal("first inference should be routable immediately after cold start")
 }
 if probes.Load()<2 {t.Fatal("expected health and credential-count probes after starting")}
}
