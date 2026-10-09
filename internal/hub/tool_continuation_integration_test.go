package hub

import (
 "bytes"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "sync/atomic"
 "testing"

 "github.com/Tsenjii/AhB/internal/config"
 "github.com/Tsenjii/AhB/internal/provider"
 "github.com/Tsenjii/AhB/internal/sidecar"
)

// These tests exercise the real Hub transport, not the upstream adapter's
// eligibility. A successful fixture never proves a live Muse account/quota.
func TestHubPreservesNativeToolCallAndToolResultAcrossTwoTurns(t *testing.T){
 var calls atomic.Int32
 upstream:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/v1/chat/completions"{http.NotFound(w,r);return}
  var payload struct{
   Model string `json:"model"`
   Messages []map[string]any `json:"messages"`
   Tools []map[string]any `json:"tools"`
  }
  if err:=json.NewDecoder(r.Body).Decode(&payload);err!=nil{t.Errorf("decode: %v",err);http.Error(w,"bad payload",400);return}
  if payload.Model!="meta/muse" {t.Errorf("nested upstream model changed: %q",payload.Model)}
  w.Header().Set("Content-Type","application/json")
  n:=calls.Add(1)
  if n==1{
   if len(payload.Messages)!=1||len(payload.Tools)!=1{t.Errorf("first turn tools lost: %+v",payload)}
   _=json.NewEncoder(w).Encode(map[string]any{"choices":[]any{map[string]any{
    "message":map[string]any{"role":"assistant","content":nil,"tool_calls":[]any{
     map[string]any{"id":"call-123","type":"function","function":map[string]any{"name":"get_time","arguments":`{"timezone":"Asia/Taipei"}`}},
    }},"finish_reason":"tool_calls"}}})
  }else{
   if len(payload.Messages)!=3{t.Errorf("tool result not forwarded: %+v",payload.Messages)}
   if len(payload.Messages)==3 {
    assistant:=payload.Messages[1]; tool:=payload.Messages[2]
    arr,ok:=assistant["tool_calls"].([]any)
    if !ok||len(arr)!=1{t.Errorf("native assistant call missing: %+v",assistant)}
    if tool["role"]!="tool"||tool["tool_call_id"]!="call-123"||tool["content"]!=`{"timestamp":"2030-01-01"}`{
      t.Errorf("tool_result fields changed: %+v",tool)
    }
   }
   _=json.NewEncoder(w).Encode(map[string]any{"choices":[]any{map[string]any{
    "message":map[string]any{"role":"assistant","content":"TOOL_LOOP_OK"},"finish_reason":"stop"}}})
  }
 }))
 defer upstream.Close()
 h:=New(config.Config{
  Listen:"127.0.0.1:8317",
  Providers:[]config.ProviderConfig{{ID:"opencode",DisplayName:"OpenCode",Enabled:true,Kind:"external",BaseURL:upstream.URL}},
 })
 h.providers["opencode"].external.set(sidecar.Snapshot{ID:"opencode",State:provider.StateHealthy,HealthHTTPStatus:200,HealthBody:[]byte(`{"keys":{"anonymous":true,"total":0}}`)})
 first:=[]byte(`{"model":"opencode/meta/muse","messages":[{"role":"user","content":"Use get_time"}],"tools":[{"type":"function","function":{"name":"get_time","parameters":{"type":"object"}}}]}`)
 serve:=func(body []byte) (map[string]any,int){
  t.Helper()
  req:=httptest.NewRequest(http.MethodPost,"/v1/chat/completions",bytes.NewReader(body))
  rec:=httptest.NewRecorder()
  h.Handler().ServeHTTP(rec,req)
  var result map[string]any
  if err:=json.Unmarshal(rec.Body.Bytes(),&result);err!=nil{t.Fatalf("decode completion: %v body=%s",err,rec.Body.String())}
  return result,rec.Code
 }
 response,status:=serve(first)
 if status!=200{t.Fatalf("first tool turn HTTP %d: %+v",status,response)}
 choices:=response["choices"].([]any)
 message:=choices[0].(map[string]any)["message"].(map[string]any)
 call:=message["tool_calls"].([]any)[0].(map[string]any)
 if call["id"]!="call-123"{t.Fatalf("tool call ID changed: %+v",call)}
 second,_:=json.Marshal(map[string]any{"model":"opencode/meta/muse","messages":[]any{
  map[string]any{"role":"user","content":"Use get_time"},
  message,
  map[string]any{"role":"tool","tool_call_id":"call-123","content":`{"timestamp":"2030-01-01"}`},
 }})
 final,code:=serve(second)
 if code!=200||calls.Load()!=2{t.Fatalf("native tool continuation failed: %d %+v, upstream calls=%d",code,final,calls.Load())}
 finalMessage:=final["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)
 if finalMessage["content"]!="TOOL_LOOP_OK" {t.Fatalf("missing final model answer: %+v",final)}
}
