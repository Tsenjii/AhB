package hub

import (
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "time"

 "github.com/Tsenjii/AhB/internal/config"
)

func TestRecentRequestStatusDoesNotConflateHTTPWithQuotaOrLeakSecrets(t *testing.T) {
 status := http.StatusServiceUnavailable
 receivedAuthorization := ""
 srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  receivedAuthorization = r.Header.Get("Authorization")
  w.WriteHeader(status)
  _, _ = w.Write([]byte(`{"error":{"message":"quota empty"}}`))
 }))
 defer srv.Close()

 h := New(config.Config{Providers: []config.ProviderConfig{{
  ID:"agent2api",DisplayName:"Agent2API",Enabled:true,
  Kind:"external",BaseURL:srv.URL,
  Headers:map[string]string{"Authorization":"Bearer LOCAL_PRIVATE_KEY"},
 }}})
 cfg := h.providers["agent2api"].cfg
 req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"agent2api/test"}`))
 resp, err := h.doProviderRequest(req,cfg,[]byte(`{"model":"test"}`))
 if err != nil { t.Fatal(err) }
 _ = resp.Body.Close()
 if receivedAuthorization != "Bearer LOCAL_PRIVATE_KEY" {t.Fatal("upstream local authorization missing")}
 view := h.providerViews()[0]
 if view.LastRequestStatus != 503 || view.LastRequestTransportError || view.LastRequestAt == "" {
  t.Fatalf("bad inference failure status: %#v",view)
 }
 if _,err:=time.Parse(time.RFC3339,view.LastRequestAt);err!=nil{t.Fatalf("invalid UTC time: %v",err)}
 raw,_:=json.Marshal(view)
 for _,secret:=range []string{"LOCAL_PRIVATE_KEY","quota empty"} {
  if strings.Contains(string(raw),secret){ t.Fatalf("sensitive value leaked in provider UI response: %s",secret)}
 }

 status = http.StatusOK
 resp,err = h.doProviderRequest(req,cfg,[]byte(`{"model":"test"}`))
 if err!=nil{t.Fatal(err)}
 _ = resp.Body.Close()
 view=h.providerViews()[0]
 if view.LastRequestStatus!=200 || view.LastRequestTransportError{
  t.Fatalf("last status should update on HTTP 200: %#v",view)
 }
 // A 200 only records successful receipt of headers; do not invent quota.
 if view.AccountUsable != nil {t.Fatalf("should not invent account quota: %#v",view)}
}

func TestTransportErrorsAreNotPretendedSuccessful(t *testing.T){
 srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){}))
 u:=srv.URL
 srv.Close()
 h:=New(config.Config{Providers:[]config.ProviderConfig{{ID:"ext", Enabled:true,Kind:"external",BaseURL:u}}})
 req:=httptest.NewRequest(http.MethodPost,"/v1/chat/completions",nil)
 _,err:=h.doProviderRequest(req,h.providers["ext"].cfg,[]byte("{}"))
 if err==nil{t.Fatal("expected transport error")}
 view:=h.providerViews()[0]
 if !view.LastRequestTransportError || view.LastRequestStatus!=0 {
  t.Fatalf("transport error falsely healthy: %#v",view)
 }
}
