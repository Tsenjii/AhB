package hub

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os/exec"
    "sync/atomic"
    "testing"
    "time"

    "github.com/Tsenjii/AhB/internal/config"
)

// Exercise the real demand supervisor, not a mocked ready-only external
// provider. In the 512 MiB profile, each sidecar sleeps until selected.
func TestOnDemandSameModelFallbackWakesAlternateWithOneProcessCap(t *testing.T) {
    if _,err:=exec.LookPath("sleep");err!=nil {t.Skip("requires sleep executable")}
    tests:=[]struct{
        name,mode,alternateAdvertises string
        wantHTTP int
        wantCalls int32
    }{
        {"sequential exact model","sequential","target-model",http.StatusOK,1},
        {"balanced exact model","balanced","target-model",http.StatusOK,1},
        {"unadvertised model is never called","sequential","unrelated-model",http.StatusServiceUnavailable,0},
    }
    for _,tt:=range tests {
        t.Run(tt.name,func(t *testing.T){
            var primaryCalls,alternativeCalls atomic.Int32
            primary:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
                switch r.URL.Path{
                case "/healthz":
                    w.WriteHeader(http.StatusOK)
                case "/v1/chat/completions":
                    primaryCalls.Add(1)
                    http.Error(w,"primary service unavailable",http.StatusServiceUnavailable)
                default:http.NotFound(w,r)
                }
            }))
            defer primary.Close()
            alternative:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
                switch r.URL.Path{
                case "/healthz":
                    w.WriteHeader(http.StatusOK)
                case "/v1/models":
                    _=json.NewEncoder(w).Encode(map[string]any{"data":[]map[string]string{{"id":tt.alternateAdvertises}}})
                case "/v1/chat/completions":
                    alternativeCalls.Add(1)
                    var payload map[string]any
                    if err:=json.NewDecoder(r.Body).Decode(&payload);err!=nil{t.Error(err)}
                    if payload["model"]!="target-model"{t.Errorf("model not rewritten: %v",payload["model"])}
                    _=json.NewEncoder(w).Encode(map[string]any{"choices":[]any{map[string]any{"message":map[string]any{"content":"ok"}}}})
                default:http.NotFound(w,r)
                }
            }))
            defer alternative.Close()
            mk:=func(id,url string)config.ProviderConfig{
                return config.ProviderConfig{
                    ID:id,Enabled:true,Kind:"sidecar",StartMode:"on_demand",
                    Binary:"sleep",Args:[]string{"40"},BaseURL:url,
                    HealthPath:"/healthz",ModelsPath:"/v1/models",
                    StartupTimeoutSeconds:5,HealthIntervalSeconds:1,MaxRestarts:1,
                }
            }
            cfg:=config.Config{
                Listen:"127.0.0.1:8317",
                Providers:[]config.ProviderConfig{mk("primary",primary.URL),mk("alternate",alternative.URL)},
                Resources:config.ResourceConfig{MaxRunningSidecars:1,IdleStopSeconds:30},
                Routing:config.RoutingConfig{SameModelFallback:config.SameModelFallbackConfig{
                    Enabled:true,Mode:tt.mode,Providers:[]string{"primary","alternate"},
                }},
            }
            h:=New(cfg)
            ctx,cancel:=context.WithCancel(context.Background())
            h.Start(ctx)
            defer func(){cancel();h.Wait()}()
            if h.providers["primary"].snapshot().PID!=0 || h.providers["alternate"].snapshot().PID!=0 {
                t.Fatal("a sleeping sidecar started without demand")
            }
            req:=httptest.NewRequest(http.MethodPost,"/v1/chat/completions",
                bytes.NewBufferString(`{"model":"primary/target-model","messages":[{"role":"user","content":"test"}]}`))
            requestCtx,requestCancel:=context.WithTimeout(req.Context(),12*time.Second)
            defer requestCancel()
            recorder:=httptest.NewRecorder()
            h.Handler().ServeHTTP(recorder,req.WithContext(requestCtx))
            if recorder.Code!=tt.wantHTTP{
                t.Fatalf("status=%d want=%d body=%s",recorder.Code,tt.wantHTTP,recorder.Body.String())
            }
            if got:=alternativeCalls.Load();got!=tt.wantCalls{
                t.Fatalf("alternate requests=%d want=%d",got,tt.wantCalls)
            }
            if primaryCalls.Load()!=1{t.Fatalf("primary calls=%d want=1",primaryCalls.Load())}
            if tt.wantCalls>0{
                if recorder.Header().Get("X-AhB-Fallback")!="same-model"{t.Fatal("fallback header missing")}
                if h.providers["primary"].snapshot().PID!=0{
                    t.Fatal("failed primary still holds single-process slot")
                }
            }
        })
    }
}
