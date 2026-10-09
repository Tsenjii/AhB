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

func TestProviderTogglePersistsUnknownConfigAndSchedulesRestart(t *testing.T) {
 dir:=t.TempDir()
 path:=filepath.Join(dir,"config.json")
 initial := "{\"listen\":\"127.0.0.1:8317\",\"allow_lan\":false,\"custom_metadata\":{\"keep\":\"yes\"},\"providers\":[{\"id\":\"opencode\",\"enabled\":true,\"kind\":\"sidecar\",\"binary\":\"./bin/opencode2api\",\"base_url\":\"http://127.0.0.1:8401\",\"my_private_setting\":\"preserve\"},{\"id\":\"agent2api\",\"enabled\":false,\"kind\":\"sidecar\",\"binary\":\"./bin/agent2api-server\",\"base_url\":\"http://127.0.0.1:8403\"}]}"
 if err:=os.WriteFile(path,[]byte(initial),0600);err!=nil{t.Fatal(err)}
 h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{{ID:"agent2api",Kind:"sidecar"}}})
 h.SetControlConfigPath(path)
 called:=0
 h.SetRestartHandler(func()error{called++;return nil})
 perform:=func(origin,token,id,body string)int{
  req:=httptest.NewRequest(http.MethodPost,"http://127.0.0.1:8317/api/control/provider/"+id,strings.NewReader(body))
  req.RemoteAddr="127.0.0.1:13888"
  req.Header.Set("Origin",origin)
  req.Header.Set("X-AhB-Control-Token",token)
  rec:=httptest.NewRecorder()
  h.Handler().ServeHTTP(rec,req)
  return rec.Code
 }
 if got:=perform("https://untrusted.test",h.controlToken,"agent2api","{\"enabled\":true}");got!=403{t.Fatalf("CSRF got %d",got)}
 if got:=perform("http://127.0.0.1:8317",h.controlToken,"agent2api","{\"enabled\":true}");got!=202{t.Fatalf("enable got %d",got)}
 raw,err:=os.ReadFile(path);if err!=nil{t.Fatal(err)}
 var got map[string]any
 if err:=json.Unmarshal(raw,&got);err!=nil{t.Fatal(err)}
 if got["custom_metadata"].(map[string]any)["keep"]!="yes"{t.Fatal("custom metadata lost")}
 pp:=got["providers"].([]any)
 if pp[0].(map[string]any)["my_private_setting"]!="preserve"{t.Fatal("provider private metadata lost")}
 if pp[1].(map[string]any)["enabled"]!=true{t.Fatal("Agent2API not enabled")}
 if called!=1{t.Fatalf("restarts: %d",called)}
 if got:=perform("http://127.0.0.1:8317",h.controlToken,"agent2api","{\"enabled\":false}");got!=409{t.Fatalf("duplicate control got %d",got)}
 fi,_:=os.Stat(path);if fi.Mode().Perm()!=0600{t.Fatalf("config permission %o",fi.Mode().Perm())}
}

func TestProviderToggleFailsClosedWithoutSecrets(t *testing.T){
 dir:=t.TempDir()
 path:=filepath.Join(dir,"config.json")
 raw:=[]byte("{\"listen\":\"127.0.0.1:8317\",\"providers\":[{\"id\":\"copilot\",\"kind\":\"sidecar\",\"enabled\":false,\"binary\":\"./bin/copilot2api\",\"base_url\":\"http://127.0.0.1:8410\"}]}")
 if err:=os.WriteFile(path,raw,0600);err!=nil{t.Fatal(err)}
 h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{{ID:"copilot",Kind:"sidecar"}}})
 h.SetControlConfigPath(path)
 h.SetRestartHandler(func()error{t.Fatal("unexpected restart");return nil})
 req:=httptest.NewRequest(http.MethodPost,"http://127.0.0.1:8317/api/control/provider/copilot",strings.NewReader("{\"enabled\":true}"))
 req.RemoteAddr="127.0.0.1:1001"
 req.Header.Set("Origin","http://127.0.0.1:8317")
 req.Header.Set("X-AhB-Control-Token",h.controlToken)
 rec:=httptest.NewRecorder()
 h.Handler().ServeHTTP(rec,req)
 if rec.Code!=409{t.Fatalf("Copilot without login got %d",rec.Code)}
 now,_:=os.ReadFile(path)
 if string(now)!=string(raw){t.Fatal("credentials-free login error changed config")}
}
