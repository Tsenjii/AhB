package hub

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "strings"
 "time"
)

// Playground is a local, explicit diagnostic. It bypasses the optional
// same-model fallback/router so HTTP 418/429/503 belongs to the SOURCE
// selected in the UI and a different provider cannot fake a green result.
// It neither adds real accounts nor stores prompts or response text.
func (h *Hub) handlePlayground(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Cache-Control","no-store")
 w.Header().Set("X-Content-Type-Options","nosniff")
 if r.Method!=http.MethodPost {
  w.Header().Set("Allow","POST")
  http.Error(w,"POST required",http.StatusMethodNotAllowed)
  return
 }
 if !h.authorizeLocalControl(w,r) {return}

 var request struct{
  Model string `json:"model"`
  Prompt string `json:"prompt"`
  Stream bool `json:"stream"`
  ToolTest bool `json:"tool_test"`
  MaxTokens int `json:"max_tokens"`
 }
 decoder:=json.NewDecoder(io.LimitReader(r.Body,20*1024))
 decoder.DisallowUnknownFields()
 if err:=decoder.Decode(&request);err!=nil || decoder.Decode(new(any))!=io.EOF {
  writeError(w,400,"invalid_request","Expected one JSON object with model, prompt, stream, tool_test and max_tokens")
  return
 }
 request.Model=strings.TrimSpace(request.Model)
 request.Prompt=strings.TrimSpace(request.Prompt)
 if len(request.Model)>256 || len(request.Prompt)>6000 || request.Prompt==""{
  writeError(w,400,"invalid_request","Prompt required (1–6000 bytes), model at most 256 bytes")
  return
 }
 if request.MaxTokens==0{request.MaxTokens=256}
 if request.MaxTokens<1 || request.MaxTokens>1024 {
  writeError(w,400,"invalid_request","max_tokens must be between 1 and 1024")
  return
 }
 provider,model,err:=splitModel(request.Model)
 if err!=nil || provider=="route" || model==""{
  writeError(w,400,"invalid_model","Use a direct provider/model (not a route alias)")
  return
 }
 p,found:=h.providers[provider]
 if !found||!p.cfg.Enabled||!p.hasRuntime(){
  writeError(w,400,"unknown_provider","Gateway is not configured or enabled")
  return
 }
 payload:=map[string]any{
  "model":model,
  "messages":[]map[string]string{{"role":"user","content":request.Prompt}},
  "stream":request.Stream,
  "max_tokens":request.MaxTokens,
 }
 if request.ToolTest {
  // The assistant may request this mock tool, but the Playground never
  // executes it or fetches real time/accounts/files. Examine tool_calls only.
  payload["tools"]=[]any{map[string]any{
   "type":"function",
   "function":map[string]any{
    "name":"playground_echo",
    "description":"Mock tool for testing structured tool calls. Not executed by AhB.",
    "parameters":map[string]any{
     "type":"object",
     "properties":map[string]any{
      "message":map[string]string{"type":"string","description":"Text to echo"},
     },
     "required":[]string{"message"},
    },
   },
  }}
 }
 body,err:=json.Marshal(payload)
 if err!=nil{writeError(w,500,"encode_error","Unable to encode Playground request");return}
 ctx,cancel:=context.WithTimeout(r.Context(),120*time.Second)
 defer cancel()
 lease,err:=h.acquireOnDemand(ctx,p)
 if err!=nil{
  writeError(w,503,"provider_start_unavailable",err.Error())
  return
 }
 defer lease() // Keep the one-slot sidecar alive through the last SSE byte.
 if !providerUsableForRouting(provider,p.cfg.Kind,p.snapshot(),p.accounts.snapshot()) {
  assessment:=assessProviderHealth(provider,p.cfg.Kind,p.snapshot(),p.accounts.snapshot())
  writeError(w,503,"provider_unavailable",fmt.Sprintf("%s is %s",provider,assessment.State))
  return
 }
 // Prevent the private local-control nonce, browser cookies and other UI
 // headers from ever being sent to an upstream gateway.
 upstream:=r.Clone(ctx)
 upstream.URL.Path="/v1/chat/completions"
 upstream.URL.RawQuery=""
 upstream.Header=make(http.Header)
 if request.Stream {upstream.Header.Set("Accept","text/event-stream")}
 response,err:=h.doProviderRequestContext(ctx,upstream,p.cfg,body)
 if err!=nil{
  message:="Gateway transport failed"
  if errors.Is(ctx.Err(),context.DeadlineExceeded){message="Gateway timed out after 120 seconds"}
  writeError(w,502,"upstream_error",message)
  return
 }
 defer response.Body.Close()
 // Don't forward cookies, authentication headers, redirects or upstream
 // control responses; only transport diagnostics of the explicitly selected
 // provider should be surfaced.
 typ:=response.Header.Get("Content-Type")
 if typ!="" {w.Header().Set("Content-Type",typ)}
 w.Header().Set("X-AhB-Playground-Provider",provider)
 w.Header().Set("X-AhB-Playground-Direct","true")
 w.WriteHeader(response.StatusCode)
 // Avoid unbounded response memory in a browser diagnostic.
 copyStreaming(w,io.LimitReader(response.Body,2<<20))
}
