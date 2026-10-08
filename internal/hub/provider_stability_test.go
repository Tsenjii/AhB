package hub

import (
 "context"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "sync"
 "testing"
 "time"

 "github.com/Tsenjii/AhB/internal/config"
 "github.com/Tsenjii/AhB/internal/provider"
 "github.com/Tsenjii/AhB/internal/sidecar"
)

func TestModelsProbeIndependentProvidersConcurrently(t *testing.T) {
 // Both endpoints block until both receive requests. The old serial loop
 // would fail this test and keep the phone dashboard waiting unnecessarily.
 started := make(chan struct{}, 2)
 release := make(chan struct{})
 var once sync.Once
 unblock := func(){ once.Do(func(){close(release)}) }
 defer unblock()
 server := func(name string) *httptest.Server {
  return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
   if r.URL.Path != "/v1/models" { http.NotFound(w,r);return }
   started <- struct{}{}
   <-release
   _ = json.NewEncoder(w).Encode(map[string]any{"data":[]map[string]string{{"id":name}}})
  }))
 }
 a,b := server("alpha"),server("beta")
 defer a.Close()
 defer b.Close()
 h:=New(config.Config{Providers:[]config.ProviderConfig{
  {ID:"alpha", Enabled:true, Kind:"external", BaseURL:a.URL, ModelsPath:"/v1/models"},
  {ID:"beta", Enabled:true, Kind:"external", BaseURL:b.URL, ModelsPath:"/v1/models"},
 }})
 for _,id:=range []string{"alpha","beta"} {
  h.providers[id].external.set(sidecar.Snapshot{ID:id,State:provider.StateHealthy,HealthHTTPStatus:200})
 }
 done:=make(chan *httptest.ResponseRecorder,1)
 go func() {
  w:=httptest.NewRecorder()
  h.Handler().ServeHTTP(w,httptest.NewRequest("GET","/v1/models",nil))
  done<-w
 }()
 for i:=0;i<2;i++ {
  select {
  case <-started:
  case <-time.After(2*time.Second):
   unblock()
   t.Fatalf("only %d provider model endpoint(s) started before release; probes must run concurrently",i)
  }
 }
 unblock()
 select {
 case w:=<-done:
  if w.Code!=200 { t.Fatalf("unexpected status %d: %s",w.Code,w.Body.String()) }
  var body struct{Data []struct{ID string `json:"id"`} `json:"data"`}
  if err:=json.Unmarshal(w.Body.Bytes(),&body);err!=nil{t.Fatal(err)}
  if len(body.Data)!=2 || body.Data[0].ID!="alpha/alpha" || body.Data[1].ID!="beta/beta" {
   t.Fatalf("incorrect model list: %#v",body.Data)
  }
 case <-time.After(3*time.Second): t.Fatal("models handler hung after all upstreams responded")
 }
}

func TestFreeBuffNodeInvalidHealthCountsFailClosed(t *testing.T) {
 for _,payload:=range []string{
  `{"accounts":2,"alive_accounts":1,"unknown_accounts":2}`,
  `{"accounts":1,"alive_accounts":-1,"unknown_accounts":0}`,
  `{"accounts":-1,"alive_accounts":0,"unknown_accounts":0}`,
 } {
  t.Run(payload,func(t *testing.T){
   srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
    if r.URL.Path!="/healthz" {http.NotFound(w,r);return}
    _,_=w.Write([]byte(payload))
   }))
   defer srv.Close()
   h:=New(config.Config{})
   got,err:=h.fetchProviderAccounts(context.Background(),config.ProviderConfig{ID:"freebuff",BaseURL:srv.URL})
   if err==nil||got.Known {t.Fatalf("invalid upstream account numbers accepted: %#v",got)}
  })
 }
}
