package hub

import (
 "encoding/json"
 "io"
 "net/http"
 "net/http/httptest"
 "strings"
 "sync/atomic"
 "testing"

 "github.com/Tsenjii/AhB/internal/config"
 "github.com/Tsenjii/AhB/internal/provider"
 "github.com/Tsenjii/AhB/internal/sidecar"
)

func playgroundTestHub(t *testing.T,source http.HandlerFunc) (*Hub,*atomic.Int32){
 t.Helper()
 var seen atomic.Int32
 primary:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  seen.Add(1)
  source(w,r)
 }))
 t.Cleanup(primary.Close)
 other:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  t.Error("Playground illegally fell back to a different provider")
  w.Header().Set("Content-Type","application/json")
  _,_=io.WriteString(w,`{"choices":[{"message":{"content":"wrong fallback"}}]}`)
 }))
 t.Cleanup(other.Close)
 h:=New(config.Config{
  Listen:"127.0.0.1:8317",
  Routing:config.RoutingConfig{SameModelFallback:config.SameModelFallbackConfig{
   Enabled:true,Mode:"balanced",Providers:[]string{"opencode","alternative"},
  }},
  Providers:[]config.ProviderConfig{
   {ID:"opencode",Kind:"external",Enabled:true,BaseURL:primary.URL},
   {ID:"alternative",Kind:"external",Enabled:true,BaseURL:other.URL},
  },
 })
 h.SetRestartHandler(func()error{return nil})
 for _,p:=range h.providers{
  p.external.set(sidecar.Snapshot{ID:p.cfg.ID,State:provider.StateHealthy,HealthHTTPStatus:200,
   HealthBody:[]byte(`{"keys":{"anonymous":true,"total":0}}`)})
 }
 return h,&seen
}

func TestPlaygroundReturnsActualHTTP418NeverCrossProviderFallback(t *testing.T){
 h,seen:=playgroundTestHub(t,func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/v1/chat/completions"{t.Errorf("wrong path %s",r.URL.Path)}
  if r.Header.Get("X-AhB-Control-Token")!=""||r.Header.Get("Cookie")!=""||
    r.Header.Get("Origin")!=""{
   t.Error("local page authorization/browser metadata leaked upstream")
  }
  w.WriteHeader(418)
  _,_=io.WriteString(w,"I'm a teapot")
 })
 body:=`{"model":"opencode/test","prompt":"hello","stream":false,"max_tokens":64}`
 response:=localControlRequest(h,"POST","/api/control/playground",body)
 if response.Code!=418{t.Fatalf("upstream status was disguised: %d %s",response.Code,response.Body.String())}
 if !strings.Contains(response.Body.String(),"teapot"){t.Fatal("real upstream response missing")}
 if response.Header().Get("X-AhB-Playground-Provider")!="opencode"||
 response.Header().Get("X-AhB-Playground-Direct")!="true" {
  t.Fatalf("missing true direct-source indicator: %v",response.Header())
 }
 if seen.Load()!=1{t.Fatalf("unexpected retry/cross-provider attempt: %d",seen.Load())}
 last:=h.providers["opencode"].lastRequest.snapshot()
 if last.Status!=418{t.Fatalf("real error not recorded: %+v",last)}
}

func TestPlaygroundSSEAndMockToolCallingContract(t *testing.T){
 h,_:=playgroundTestHub(t,func(w http.ResponseWriter,r *http.Request){
  if r.Header.Get("Accept")!="text/event-stream"{t.Errorf("missing SSE Accept")}
  var payload struct{
   Model string `json:"model"`
   Stream bool `json:"stream"`
   Messages []map[string]any `json:"messages"`
   Tools []map[string]any `json:"tools"`
  }
  if err:=json.NewDecoder(r.Body).Decode(&payload);err!=nil{t.Error(err);return}
  if payload.Model!="nested/path-model"||!payload.Stream{
   t.Errorf("wrong upstream payload: %+v",payload)
  }
  if len(payload.Messages)!=1||payload.Messages[0]["content"]!="read the time"{
   t.Errorf("missing user prompt: %+v",payload.Messages)
  }
  if len(payload.Tools)!=1||payload.Tools[0]["type"]!="function"{
   t.Errorf("missing mock tool schema: %+v",payload.Tools)
  }
  w.Header().Set("Content-Type","text/event-stream")
  _,_=io.WriteString(w,"data: {\"choices\":[{\"delta\":{\"content\":\"working\"},\"finish_reason\":null}]}\n\n")
  _,_=io.WriteString(w,"data: [DONE]\n\n")
 })
 rec:=localControlRequest(h,"POST","/api/control/playground",`{"model":"opencode/nested/path-model","prompt":"read the time","stream":true,"tool_test":true,"max_tokens":128}`)
 if rec.Code!=200||!strings.Contains(rec.Body.String(),"data: [DONE]"){
  t.Fatalf("SSE response incomplete: %d %s",rec.Code,rec.Body.String())
 }
}

func TestPlaygroundRejectsInvalidAndCrossOriginWithoutSendingPrompts(t *testing.T){
 h,calls:=playgroundTestHub(t,func(w http.ResponseWriter,r *http.Request){
  t.Error("invalid local playground payload reached upstream")
 })
 invalid:=[]string{
  `{"model":"route/x","prompt":"test"}`,
  `{"model":"opencode/x","prompt":""}`,
  `{"model":"opencode/x","prompt":"test","max_tokens":100000}`,
  `{"model":"opencode/x","prompt":"test","secret":"unexpected"}`,
  `{"model":"nonexistent/x","prompt":"test"}`,
 }
 for _,body:=range invalid{
  rec:=localControlRequest(h,"POST","/api/control/playground",body)
  if rec.Code!=400{t.Fatalf("expected 400 for invalid request, got %d: %s",rec.Code,body)}
 }
 forbidden:=httptest.NewRequest(http.MethodPost,"http://127.0.0.1:8317/api/control/playground",
  strings.NewReader(`{"model":"opencode/x","prompt":"test"}`))
 forbidden.RemoteAddr="127.0.0.1:9091"
 forbidden.Header.Set("Origin","https://attacker.example")
 forbidden.Header.Set("X-AhB-Control-Token",h.controlToken)
 rec:=httptest.NewRecorder()
 h.Handler().ServeHTTP(rec,forbidden)
 if rec.Code!=403{t.Fatalf("cross-origin playground request accepted: HTTP %d",rec.Code)}
 if calls.Load()!=0{t.Fatal("invalid requests consumed account quota")}
}

func TestPlaygroundDashboardAndStandaloneWorkbench(t *testing.T){
 h,_:=playgroundTestHub(t,func(w http.ResponseWriter,r *http.Request){t.Fatal("unexpected upstream call")})
 dashboard:=httptest.NewRecorder()
 h.Handler().ServeHTTP(dashboard,httptest.NewRequest(http.MethodGet,"http://127.0.0.1:8317/ui",nil))
 if dashboard.Code!=200||!strings.Contains(dashboard.Body.String(),`href="/playground"`){
  t.Fatal("Dashboard missing Playground navigation")
 }
 workbench:=httptest.NewRecorder()
 h.Handler().ServeHTTP(workbench,httptest.NewRequest(http.MethodGet,"http://127.0.0.1:8317/playground",nil))
 if workbench.Code!=200||!strings.Contains(workbench.Body.String(),"SSE")||
  !strings.Contains(workbench.Body.String(),"418") ||
  !strings.Contains(workbench.Body.String(),h.controlToken) {
  t.Fatal("responsive Playground page missing critical controls")
 }
 if workbench.Header().Get("X-Frame-Options")!="DENY"{
  t.Fatal("local nonce-bearing workbench can be framed")
 }
}
