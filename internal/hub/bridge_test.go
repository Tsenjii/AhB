package hub

import (
  "bytes"
  "encoding/json"
  "net/http"
  "net/http/httptest"
  "testing"

  "github.com/Tsenjii/AhB/internal/config"
  "github.com/Tsenjii/AhB/internal/provider"
  "github.com/Tsenjii/AhB/internal/sidecar"
)

func TestExternalBridgePrefixAndAuthRouting(t *testing.T) {
  gotPath := ""
  gotAuth := ""
  upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case "/openai/v1/models":
      if r.Header.Get("Authorization") != "Bearer local-secret" { http.Error(w, "bad authorization", http.StatusUnauthorized); return }
      _ = json.NewEncoder(w).Encode(map[string]any{"data":[]map[string]any{{"id":"test"}}})
    case "/openai/v1/chat/completions":
      gotPath = r.URL.Path
      gotAuth = r.Header.Get("Authorization")
      var body map[string]any
      _ = json.NewDecoder(r.Body).Decode(&body)
      _ = json.NewEncoder(w).Encode(map[string]any{"model":body["model"]})
    default:
      http.NotFound(w,r)
    }
  }))
  defer upstream.Close()

  cfg := config.Config{
    Listen:"127.0.0.1:8317",
    Providers:[]config.ProviderConfig{{
      ID:"gemini", Kind:"external", Enabled:true, BaseURL:upstream.URL,
      ModelsPath:"/openai/v1/models", HealthPath:"/openai/v1/models",
      APIPathPrefix:"/openai/v1",
      Headers:map[string]string{"Authorization":"Bearer local-secret"},
    }},
  }
  if err := cfg.Validate(); err != nil { t.Fatal(err) }
  h := New(cfg)
  h.providers["gemini"].external.set(sidecar.Snapshot{
    ID:"gemini",State:provider.StateHealthy,HealthHTTPStatus:200,
  })
  resp := httptest.NewRecorder()
  h.Handler().ServeHTTP(resp, httptest.NewRequest("GET","/v1/models",nil))
  if resp.Code != 200 || !bytes.Contains(resp.Body.Bytes(),[]byte("gemini/test")) {
    t.Fatalf("model list failed %d: %s",resp.Code,resp.Body.String())
  }

  req := httptest.NewRequest(http.MethodPost,"/v1/chat/completions",
    bytes.NewBufferString(`{"model":"gemini/test","messages":[]}`))
  req.Header.Set("Authorization","Bearer caller-key")
  answer := httptest.NewRecorder()
  h.Handler().ServeHTTP(answer,req)
  if answer.Code != 200 || !bytes.Contains(answer.Body.Bytes(),[]byte(`"model":"test"`)) {
    t.Fatalf("bridge response status %d body %s",answer.Code,answer.Body.String())
  }
  if gotPath != "/openai/v1/chat/completions" || gotAuth != "Bearer local-secret" {
    t.Fatalf("upstream path=%s authorization=%s",gotPath,gotAuth)
  }
}
