package hub

import (
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"

 "github.com/Tsenjii/AhB/internal/config"
)

func adminFixture(t *testing.T) (*Hub,string) {
 t.Helper()
 root:=t.TempDir()
 write:=func(name,value string){
  t.Helper()
  path:=filepath.Join(root,name)
  if err:=os.MkdirAll(filepath.Dir(path),0700);err!=nil{t.Fatal(err)}
  if err:=os.WriteFile(path,[]byte(value),0600);err!=nil{t.Fatal(err)}
 }
 write("config.json",`{"providers":[{"id":"geminiweb","env":{"ADMIN_TOKEN":"gemini-secret"}}]}`)
 write("data/opencode/config.json",`{"webui":{"username":"admin","password":"opencode-secret"}}`)
 write("data/deepseek2api/admin-key.txt","deepseek-secret\n")
 write("data/grok2api/admin-password.txt","grok-bootstrap-secret\n")
 write("data/kiro-go/config.json",`{"password":"kiro-secret"}`)
 h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{
  {ID:"opencode",Enabled:true,Kind:"sidecar"},
  {ID:"deepseek",Enabled:true,Kind:"sidecar"},
  {ID:"geminiweb",Enabled:true,Kind:"sidecar"},
  {ID:"grok",Enabled:true,Kind:"sidecar"},
  {ID:"kiro",Enabled:true,Kind:"sidecar"},
  {ID:"agent2api",Enabled:true,Kind:"sidecar"},
 }})
 h.SetControlConfigPath(filepath.Join(root,"config.json"))
 h.SetRestartHandler(func()error{return nil})
 return h,root
}

func TestConsoleAccessExplicitLocalCopyNeverLeaksByDefault(t *testing.T) {
 h,_:=adminFixture(t)
 got:=localControlRequest(h,"POST","/api/control/console-access",`{"action":"list"}`)
 if got.Code!=200{t.Fatalf("list HTTP %d: %s",got.Code,got.Body.String())}
 for _,secret:=range []string{"opencode-secret","deepseek-secret","gemini-secret","grok-bootstrap-secret","kiro-secret"}{
  if strings.Contains(got.Body.String(),secret){t.Fatalf("list disclosed private secret")}
 }
 var payload struct{Consoles []struct{
  ID string `json:"id"`
  Username string `json:"username"`
  LocalAvailable bool `json:"local_available"`
 } `json:"consoles"` }
 if err:=json.Unmarshal(got.Body.Bytes(),&payload);err!=nil{t.Fatal(err)}
 if len(payload.Consoles)!=6{t.Fatalf("expected six console rows: %+v",payload)}
 values:=map[string]string{
  "opencode":"opencode-secret","deepseek":"deepseek-secret",
  "geminiweb":"gemini-secret","grok":"grok-bootstrap-secret","kiro":"kiro-secret",
 }
 for id,expected:=range values{
  reqBody:=`{"action":"copy","provider":"`+id+`"} `
  rec:=localControlRequest(h,"POST","/api/control/console-access",reqBody)
  if rec.Code!=200{t.Fatalf("%s: HTTP %d, %s",id,rec.Code,rec.Body.String())}
  var body map[string]string
  if err:=json.Unmarshal(rec.Body.Bytes(),&body);err!=nil{t.Fatal(err)}
  if body["secret"]!=expected{t.Fatalf("%s returned incorrect local admin password",id)}
  if rec.Header().Get("Cache-Control")!="no-store"{t.Fatal("missing no-store on secret response")}
  if strings.Contains(rec.Body.String(),"github_token"){t.Fatal("OAuth token unexpectedly returned")}
 }
 missing:=localControlRequest(h,"POST","/api/control/console-access",`{"action":"copy","provider":"agent2api"}`)
 if missing.Code!=404{t.Fatalf("upstream-native-only credentials unexpectedly exposed: %d",missing.Code)}
}

func TestConsoleAccessRejectsCrossOriginAndUnsafeFilesystem(t *testing.T){
 h,root:=adminFixture(t)
 cross:=httptest.NewRequest(http.MethodPost,"http://127.0.0.1:8317/api/control/console-access",strings.NewReader(`{"action":"copy","provider":"opencode"}`))
 cross.RemoteAddr="127.0.0.1:9000"
 cross.Header.Set("Origin","https://attack.example")
 cross.Header.Set("X-AhB-Control-Token",h.controlToken)
 denied:=httptest.NewRecorder()
 h.Handler().ServeHTTP(denied,cross)
 if denied.Code!=403{t.Fatalf("origin bypass on secret read: %d",denied.Code)}
 for _,invalid:=range []string{
  `{"action":"copy","provider":"../../etc/passwd"}`,
  `{"action":"copy","provider":"agent2api"}`,
  `{"action":"copy","provider":"kiro","other":"field"}`,
 } {
  rec:=localControlRequest(h,"POST","/api/control/console-access",invalid)
  if rec.Code==200{t.Fatalf("invalid credential read allowed: %s",invalid)}
 }
 target:=filepath.Join(root,"data","opencode","config.json")
 if err:=os.Chmod(target,0644);err!=nil{t.Fatal(err)}
 if rec:=localControlRequest(h,"POST","/api/control/console-access",`{"action":"copy","provider":"opencode"}`);rec.Code!=409{
  t.Fatalf("world-readable private credential file unexpectedly accepted: %d",rec.Code)
 }
 if err:=os.Remove(target);err!=nil{t.Fatal(err)}
 if err:=os.Symlink(filepath.Join(root,"config.json"),target);err!=nil{t.Skip("symlinks unsupported")}
 if rec:=localControlRequest(h,"POST","/api/control/console-access",`{"action":"copy","provider":"opencode"}`);rec.Code!=409{
  t.Fatalf("symlinked admin config unexpectedly accepted: %d",rec.Code)
 }
}
