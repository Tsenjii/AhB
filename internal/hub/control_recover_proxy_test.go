package hub

import (
 "context"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "testing"

 "github.com/Tsenjii/AhB/internal/config"
)

func localControlRequest(h *Hub, method,path,body string)*httptest.ResponseRecorder{
 req:=httptest.NewRequest(method,"http://127.0.0.1:8317"+path,strings.NewReader(body))
 req.RemoteAddr="127.0.0.1:13337"
 req.Header.Set("Origin","http://127.0.0.1:8317")
 req.Header.Set("X-AhB-Control-Token",h.controlToken)
 req.Header.Set("Content-Type","application/json")
 rec:=httptest.NewRecorder()
 h.Handler().ServeHTTP(rec,req)
 return rec
}

func TestSingleGatewayRecoveryProtectsStreamsAndHasCooldown(t *testing.T){
 if _,err:=exec.LookPath("sleep");err!=nil{t.Skip("requires sleep")}
 upstream:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/healthz"{http.NotFound(w,r);return}
  w.Header().Set("Content-Type","application/json")
  _,_=w.Write([]byte(`{"keys":{"anonymous":true,"total":0}}`))
 }))
 defer upstream.Close()
 h:=New(config.Config{Listen:"127.0.0.1:8317",
  Resources:config.ResourceConfig{MaxRunningSidecars:1,IdleStopSeconds:60},
  Providers:[]config.ProviderConfig{{ID:"opencode",Enabled:true,Kind:"sidecar",StartMode:"on_demand",
   Binary:"sleep",Args:[]string{"35"},BaseURL:upstream.URL,HealthPath:"/healthz",
   StartupTimeoutSeconds:5,HealthIntervalSeconds:1,MaxRestarts:1}}})
 h.SetRestartHandler(func()error{return nil})
 ctx,cancel:=context.WithCancel(context.Background())
 h.Start(ctx);defer func(){cancel();h.Wait()}()
 if r:=localControlRequest(h,"POST","/api/control/recover/opencode","{}");r.Code!=409{
  t.Fatalf("sleeping source should not be woken by recovery: %d",r.Code)
 }
 lease,err:=h.acquireOnDemand(ctx,h.providers["opencode"])
 if err!=nil{t.Fatal(err)}
 oldPID:=h.providers["opencode"].snapshot().PID
 if oldPID==0{t.Fatal("missing original PID")}
 if r:=localControlRequest(h,"POST","/api/control/recover/opencode","{}");r.Code!=409{
  t.Fatalf("recovery evicted active inference lease: HTTP %d",r.Code)
 }
 if h.providers["opencode"].snapshot().PID!=oldPID{t.Fatal("active gateway killed")}
 lease()
 blocked:=localControlRequest(h,"POST","/api/control/recover/opencode","{}")
 if blocked.Code!=200{t.Fatalf("safe recovery: HTTP %d, %s",blocked.Code,blocked.Body.String())}
 var result map[string]any
 if err:=json.Unmarshal(blocked.Body.Bytes(),&result);err!=nil{t.Fatal(err)}
 newPID:=h.providers["opencode"].snapshot().PID
 if newPID==0||newPID==oldPID{t.Fatalf("expected new process PID: %d -> %d",oldPID,newPID)}
 if result["status"]!="restarted"{t.Fatal(result)}
 if again:=localControlRequest(h,"POST","/api/control/recover/opencode","{}");again.Code!=429{
  t.Fatalf("recovery loop not cooled down: %d %s",again.Code,again.Body.String())
 }
 denied:=httptest.NewRequest("POST","http://127.0.0.1:8317/api/control/recover/opencode",strings.NewReader("{}"))
 denied.RemoteAddr="127.0.0.1:14444"
 denied.Header.Set("Origin","http://evil.invalid")
 denied.Header.Set("X-AhB-Control-Token",h.controlToken)
 unauthorized:=httptest.NewRecorder();h.Handler().ServeHTTP(unauthorized,denied)
 if unauthorized.Code!=403{t.Fatalf("cross-origin recovery bypass: %d",unauthorized.Code)}
}

func TestProviderProxySettingPreservesCredentialsAndRejectsURLSecrets(t *testing.T){
 root:=t.TempDir()
 path:=filepath.Join(root,"config.json")
 original:=`{"listen":"127.0.0.1:8317","account_note":"keep","providers":[{"id":"opencode","enabled":true,"kind":"sidecar","binary":"./bin/opencode","base_url":"http://127.0.0.1:8401","headers":{"Authorization":"Bearer secret-still-here"}},{"id":"agent2api","enabled":false,"kind":"sidecar","proxy_url":"http://127.0.0.1:7890"}]}`
 if err:=os.WriteFile(path,[]byte(original),0600);err!=nil{t.Fatal(err)}
 h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{{ID:"opencode",Enabled:true,Kind:"sidecar"}}})
 h.SetControlConfigPath(path)
 restarts:=0
 h.SetRestartHandler(func()error{restarts++;return nil})
 invalid:=[]string{
  `{"proxy_url":"http://alice:password@localhost:8080"}`,
  `{"proxy_url":"ftp://localhost:1000"}`,
  `{"proxy_url":"http://localhost:0"}`,
  `{"proxy_url":"http://localhost:8080/private"}`,
  `{"proxy_url":"http://localhost:8080?secret=bad"}`,
  `{"proxy_url":"http://localhost:8080","unknown":true}`,
 }
 for _,body:=range invalid{
  rec:=localControlRequest(h,"POST","/api/control/proxy/opencode",body)
  if rec.Code!=400{t.Fatalf("unsafe URL allowed: %s, HTTP %d",body,rec.Code)}
 }
 untouched,_:=os.ReadFile(path)
 if string(untouched)!=original||restarts!=0{t.Fatal("malformed proxy request changed saved credentials")}
 rec:=localControlRequest(h,"POST","/api/control/proxy/opencode",`{"proxy_url":"socks5://127.0.0.1:1080"}`)
 if rec.Code!=202{t.Fatalf("valid proxy rejected: HTTP %d %s",rec.Code,rec.Body.String())}
 if restarts!=1 {t.Fatalf("expected controlled restart, got %d",restarts)}
 raw,err:=os.ReadFile(path)
 if err!=nil{t.Fatal(err)}
 if !strings.Contains(string(raw),"secret-still-here")||!strings.Contains(string(raw),`"account_note": "keep"`){
  t.Fatal("proxy setting erased unknown metadata or account credentials")
 }
 var result map[string]any
 if err:=json.Unmarshal(raw,&result);err!=nil{t.Fatal(err)}
 items:=result["providers"].([]any)
 first:=items[0].(map[string]any)
 if first["proxy_url"]!="socks5://127.0.0.1:1080" {t.Fatalf("proxy missing: %v",first["proxy_url"])}
 if st,err:=os.Stat(path);err!=nil||st.Mode().Perm()!=0600{
  t.Fatalf("config mode not owner-only %v %v",st,err)
 }
 if again:=localControlRequest(h,"POST","/api/control/proxy/opencode",`{"proxy_url":""}`);again.Code!=409{
  t.Fatalf("two simultaneous configuration restarts allowed: %d",again.Code)
 }
}
