package hub

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "github.com/Tsenjii/AhB/internal/config"
)

func TestOnDemandCatalogCacheSurvivesRestartWithoutSecrets(t *testing.T) {
    root:=t.TempDir()
    cfg:=config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{{
        ID:"opencode",DisplayName:"OpenCode",Enabled:true,Kind:"sidecar",StartMode:"on_demand",
        BaseURL:"http://127.0.0.1:8401",
    }}}
    h:=New(cfg)
    h.SetControlConfigPath(filepath.Join(root,"config.json"))
    h.modelCache.record(cfg.Providers[0],[]map[string]any{{
        "id":"opencode/meta/muse", "object":"model","x_provider":"opencode",
        "context_window":float64(128000),"secret_test_token":"do-not-persist-this-secret",
        "authorization":"Bearer do-not-persist-this-secret",
    }})
    path:=filepath.Join(root,"data","hub-model-catalog.json")
    raw,err:=os.ReadFile(path)
    if err!=nil {t.Fatalf("cache not persisted: %v",err)}
    if strings.Contains(string(raw),"do-not-persist-this-secret")||strings.Contains(string(raw),"authorization"){
        t.Fatal("sensitive or unrecognized upstream metadata was persisted")
    }
    info,err:=os.Stat(path)
    if err!=nil||info.Mode().Perm()!=0600 {t.Fatalf("catalog should be owner-only: info=%v err=%v",info,err)}
    second:=New(cfg)
    second.SetControlConfigPath(filepath.Join(root,"config.json"))
    models,stamp:=second.modelCache.sleeping("opencode")
    if stamp.IsZero()||len(models)!=1 {t.Fatalf("catalog not restored: %+v %v",models,stamp)}
    if models[0]["id"]!="opencode/meta/muse"||models[0]["x_cached"]!=true||models[0]["x_cache_state"]!="sleeping_unverified" {
        t.Fatalf("cache result not identified as unverified: %+v",models[0])
    }
    if models[0]["context_window"]!=float64(128000) {t.Fatalf("context metadata lost: %+v",models[0])}
}

func TestExpiredOrDisabledCatalogNeverAdvertised(t *testing.T){
    cfg:=config.ProviderConfig{ID:"opencode",DisplayName:"OpenCode",Enabled:true,Kind:"sidecar",StartMode:"on_demand"}
    cache:=newModelCatalogCache()
    cache.record(cfg,[]map[string]any{{"id":"opencode/muse"}})
    cache.mu.Lock()
    entry:=cache.entries["opencode"]
    entry.UpdatedAt=time.Now().Add(-modelCacheTTL-time.Minute)
    cache.entries["opencode"]=entry
    cache.mu.Unlock()
    if got,_:=cache.sleeping("opencode");len(got)!=0{t.Fatal("expired model cache accepted")}
    // Even a fresh cache is ignored if the source is disabled.
    h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{{
       ID:"opencode",Enabled:false,Kind:"sidecar",StartMode:"on_demand",
    }}})
    h.modelCache.record(cfg,[]map[string]any{{"id":"opencode/muse"}})
    res:=httptest.NewRecorder()
    h.Handler().ServeHTTP(res,httptest.NewRequest(http.MethodGet,"/v1/models",nil))
    var listing struct{Data []map[string]any `json:"data"`}
    if json.Unmarshal(res.Body.Bytes(),&listing)!=nil||len(listing.Data)>0 {
      t.Fatalf("disabled source advertised stale cached model: %s",res.Body.String())
    }
}

func TestSleepingModelsDoNotCreateFakeRouteAliases(t *testing.T){
    providerCfg:=config.ProviderConfig{ID:"opencode",DisplayName:"OpenCode",Enabled:true,Kind:"sidecar",StartMode:"on_demand"}
    cfg:=config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{providerCfg},
      Routing:config.RoutingConfig{RouteAliasesEnabled:true},
      Routes:[]config.RouteConfig{{ID:"coding",Targets:[]string{"opencode/meta/muse"}}},
    }
    h:=New(cfg)
    h.modelCache.record(providerCfg,[]map[string]any{{"id":"opencode/meta/muse"}})
    res:=httptest.NewRecorder()
    h.Handler().ServeHTTP(res,httptest.NewRequest("GET","/v1/models",nil))
    if res.Code!=http.StatusOK {t.Fatal(res.Body.String())}
    var listing struct{
     Data []map[string]any `json:"data"`
     Warnings map[string]string `json:"x_provider_warnings"`
    }
    if err:=json.Unmarshal(res.Body.Bytes(),&listing);err!=nil{t.Fatal(err)}
    if len(listing.Data)!=1||listing.Data[0]["id"]!="opencode/meta/muse"{
     t.Fatalf("cached model/virtual route alias error: %+v",listing.Data)
    }
    if listing.Data[0]["x_cached"]!=true||listing.Warnings["opencode"]==""{
     t.Fatalf("stale cache warning missing: %+v",listing)
    }
}

func TestLiveDiscoveryCachesOnlyAfterRealModelResponse(t *testing.T) {
    if _,err:=exec.LookPath("sleep");err!=nil{t.Skip("requires sleep")}
    modelsSource:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
        switch r.URL.Path {
        case "/healthz": w.WriteHeader(http.StatusOK)
        case "/v1/models":
            _=json.NewEncoder(w).Encode(map[string]any{"data":[]any{
                map[string]any{"id":"meta/muse","object":"model","context_window":12345},
            }})
        default:http.NotFound(w,r)
        }
    }))
    defer modelsSource.Close()
    pCfg:=config.ProviderConfig{
      ID:"opencode",DisplayName:"OpenCode",Enabled:true,Kind:"sidecar",StartMode:"on_demand",
      Binary:"sleep",Args:[]string{"35"},BaseURL:modelsSource.URL,HealthPath:"/healthz",
      StartupTimeoutSeconds:5,HealthIntervalSeconds:1,MaxRestarts:1,
    }
    h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{pCfg},
      Resources:config.ResourceConfig{MaxRunningSidecars:1,IdleStopSeconds:30}})
    ctx,cancel:=context.WithCancel(context.Background())
    h.Start(ctx)
    defer func(){cancel();h.Wait()}()
    get:=func()map[string]any{
      t.Helper()
      rec:=httptest.NewRecorder()
      h.Handler().ServeHTTP(rec,httptest.NewRequest("GET","/v1/models",nil))
      if rec.Code!=http.StatusOK{t.Fatalf("models HTTP %d: %s",rec.Code,rec.Body.String())}
      var doc map[string]any
      if err:=json.NewDecoder(bytes.NewBuffer(rec.Body.Bytes())).Decode(&doc);err!=nil{t.Fatal(err)}
      return doc
    }
    first:=get()
    if len(first["data"].([]any))!=0||h.providers["opencode"].snapshot().PID!=0{
       t.Fatal("background /v1/models unexpectedly woke on-demand service")
    }
    p:=h.providers["opencode"]
    release,err:=h.acquireOnDemand(context.Background(),p)
    if err!=nil {t.Fatal(err)}
    second:=get()
    if len(second["data"].([]any))!=1 {t.Fatalf("live model discovery failed: %+v",second)}
    release()
    h.demandMu.Lock()
    done,cancelRun:=p.demand.done,p.demand.cancel
    p.demand.closing=true
    h.demandMu.Unlock()
    h.stopDemand(p,done,cancelRun)
    third:=get()
    data:=third["data"].([]any)
    if len(data)!=1||data[0].(map[string]any)["x_cached"]!=true{
      t.Fatalf("sleeping cached models not shown: %+v",third)
    }
}
