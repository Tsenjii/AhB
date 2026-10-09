package hub

import (
 "context"
 "net/http"
 "net/http/httptest"
 "os"
 "os/exec"
 "strings"
 "testing"
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
