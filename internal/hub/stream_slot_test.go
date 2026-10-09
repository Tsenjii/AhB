package hub

import (
 "bytes"
 "context"
 "errors"
 "net/http"
 "net/http/httptest"
 "os/exec"
 "sync"
 "testing"
 "time"

 "github.com/Tsenjii/AhB/internal/config"
)

// A streaming chat keeps the on-demand lease until the final response byte.
// Under a one-provider cap, concurrent requests cannot evict it mid-stream.
func TestNativeSSEHoldsOneResidentSidecarUntilDone(t *testing.T) {
 if _,err:=exec.LookPath("sleep");err!=nil{t.Skip("requires sleep")}
 entered:=make(chan struct{},1)
 finish:=make(chan struct{})
 var releaseStream sync.Once
 defer releaseStream.Do(func(){close(finish)})
 upstream:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  switch r.URL.Path {
  case "/healthz": w.WriteHeader(http.StatusOK)
  case "/v1/chat/completions":
   w.Header().Set("Content-Type","text/event-stream")
   w.WriteHeader(http.StatusOK)
   _,_=w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"))
   if flush,ok:=w.(http.Flusher);ok{flush.Flush()}
   select {case entered<-struct{}{}:default:}
   select {
   case <-finish:
    _,_=w.Write([]byte("data: [DONE]\n\n"))
    if flush,ok:=w.(http.Flusher);ok{flush.Flush()}
   case <-r.Context().Done(): return
   }
  default: http.NotFound(w,r)
  }
 }))
 defer upstream.Close()
 mk:=func(id string)config.ProviderConfig {
  return config.ProviderConfig{ID:id,Enabled:true,Kind:"sidecar",StartMode:"on_demand",
   Binary:"sleep",Args:[]string{"35"},BaseURL:upstream.URL,HealthPath:"/healthz",
   StartupTimeoutSeconds:5,HealthIntervalSeconds:1,MaxRestarts:1}
 }
 h:=New(config.Config{Listen:"127.0.0.1:8317",
  Resources:config.ResourceConfig{MaxRunningSidecars:1,IdleStopSeconds:1},
  Providers:[]config.ProviderConfig{mk("first"),mk("second")}})
 ctx,cancel:=context.WithCancel(context.Background())
 h.Start(ctx)
 defer func(){cancel();h.Wait()}()
 done:=make(chan *httptest.ResponseRecorder,1)
 go func(){
  rec:=httptest.NewRecorder()
  req:=httptest.NewRequest(http.MethodPost,"/v1/chat/completions",
   bytes.NewBufferString("{\"model\":\"first/test\",\"stream\":true,\"messages\":[]}"))
  h.Handler().ServeHTTP(rec,req)
  done<-rec
 }()
 select {
 case <-entered:
 case <-time.After(7*time.Second):t.Fatal("SSE did not reach first chunk")
 }
 if _,err:=h.acquireOnDemand(ctx,h.providers["second"]);!errors.Is(err,errOnDemandBusy){
  t.Fatalf("second provider evicted an active SSE response: %v",err)
 }
 if h.providers["first"].snapshot().PID==0{t.Fatal("active stream's provider disappeared")}
 releaseStream.Do(func(){close(finish)})
 select {
 case rec:=<-done:
  if rec.Code!=http.StatusOK||!bytes.Contains(rec.Body.Bytes(),[]byte("data: [DONE]\n\n")){
   t.Fatalf("SSE did not complete: HTTP %d body %s",rec.Code,rec.Body.String())
  }
 case <-time.After(7*time.Second):t.Fatal("stream lease did not complete")
 }
 second,err:=h.acquireOnDemand(ctx,h.providers["second"])
 if err!=nil{t.Fatalf("sidecar capacity stayed locked after [DONE]: %v",err)}
 second()
}
