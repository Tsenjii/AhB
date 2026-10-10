package hub

import (
 "context"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"

 "github.com/Tsenjii/AhB/internal/config"
 "github.com/Tsenjii/AhB/internal/provider"
 "github.com/Tsenjii/AhB/internal/sidecar"
)

func TestCatalogModelsMarkedUnverified(t *testing.T) {
 upstream:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/v1/models" {http.NotFound(w,r);return}
  w.Header().Set("Content-Type","application/json")
  _,_=w.Write([]byte(`{"data":[{"id":"deepseek-reasoner","object":"model"}]}`))
 }))
 defer upstream.Close()
 cfg:=config.ProviderConfig{ID:"test",Kind:"external",BaseURL:upstream.URL,ModelsPath:"/v1/models"}
 h:=New(config.Config{Providers:[]config.ProviderConfig{cfg}})
 models,err:=h.fetchModels(context.Background(),cfg)
 if err!=nil||len(models)!=1{t.Fatalf("catalog err=%v models=%v",err,models)}
 if models[0]["id"]!="test/deepseek-reasoner" || models[0]["x_catalog_only"]!=true {
  t.Fatalf("unverified model advertised without caveat: %v",models[0])
 }
}

func TestDeepseekTokenCandidatesAreNotInferenceVerification(t *testing.T) {
 s:=sidecar.Snapshot{ID:"deepseek",State:provider.StateHealthy,PID:4242,HealthHTTPStatus:200}
 assessment:=assessProviderHealth("deepseek","sidecar",s,accountProbe{Known:true,Total:2,Usable:2})
 if assessment.AccountUsable!=nil || !strings.Contains(assessment.Detail,"unverified") {
  t.Fatalf("unverified token lines look usable: %+v",assessment)
 }
 missing:=assessProviderHealth("deepseek","sidecar",s,accountProbe{Known:true,Total:0,Usable:0})
 if missing.AccountUsable==nil || *missing.AccountUsable {
  t.Fatalf("zero accounts incorrectly usable: %+v",missing)
 }
}

func TestFreebuffAuth502IsDiagnosedWithoutLeakingUpstreamBody(t *testing.T) {
 root:=t.TempDir()
 if err:=os.WriteFile(filepath.Join(root,"config.json"),[]byte("{}"),0600);err!=nil{t.Fatal(err)}
 const hidden="do-not-print-upstream-token-or-cookie"
 upstream:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
  w.Header().Set("Content-Type","application/json")
  w.WriteHeader(http.StatusBadGateway)
  _,_=w.Write([]byte(`{"error":"`+hidden+`"}`))
 }))
 defer upstream.Close()
 h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{{
  ID:"freebuff",Kind:"sidecar",Enabled:true,StartMode:"on_demand",Binary:"sleep",BaseURL:"http://127.0.0.1:8402",
 }}})
 h.SetControlConfigPath(filepath.Join(root,"config.json"))
 h.SetRestartHandler(func()error{return nil})
 h.freebuffAuthBase=upstream.URL
 ctx,cancel:=context.WithCancel(context.Background())
 h.Start(ctx)
 defer func(){cancel();h.Wait()}()
 start:=localControlRequest(h,"POST","/api/control/login/freebuff",`{"action":"start"}`)
 if start.Code!=http.StatusAccepted {t.Fatalf("start: %d %s",start.Code,start.Body.String())}
 deadline:=time.After(3*time.Second)
 for{
  resp:=localControlRequest(h,"POST","/api/control/login/freebuff",`{"action":"status"}`)
  var state struct{State string `json:"state"`;Detail string `json:"detail"`}
  if err:=json.Unmarshal(resp.Body.Bytes(),&state);err!=nil{t.Fatal(err)}
  if strings.Contains(resp.Body.String(),hidden) {t.Fatal("upstream credentials leaked into UI")}
  if state.State=="failed"{
   if !strings.Contains(state.Detail,"502")||!strings.Contains(state.Detail,"授權網址"){
    t.Fatalf("missing actionable stage/status: %+v",state)
   }
   break
  }
  select {case <-deadline:t.Fatalf("OAuth error status timed out: %+v",state);default:time.Sleep(10*time.Millisecond)}
 }
}
