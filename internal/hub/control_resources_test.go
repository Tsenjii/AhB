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

func TestResourceSettingsSecureAtomicAndPreserveCredentials(t *testing.T) {
 dir:=t.TempDir()
 path:=filepath.Join(dir,"config.json")
 original:=`{"listen":"127.0.0.1:8317","custom_note":{"keep":"yes"},"providers":[{"id":"opencode","enabled":true,"kind":"sidecar","start_mode":"on_demand","binary":"./bin/opencode2api","base_url":"http://127.0.0.1:8401","headers":{"Authorization":"Bearer keep-account-secret"}}],"resources":{"max_running_sidecars":1,"idle_stop_seconds":120,"unrecognized":"keep"}}`
 if err:=os.WriteFile(path,[]byte(original),0600);err!=nil {t.Fatal(err)}
 h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{{ID:"opencode",Kind:"sidecar"}}})
 h.SetControlConfigPath(path)
 restarts:=0
 h.SetRestartHandler(func()error{restarts++;return nil})
 invoke:=func(origin,token,body string)int {
  req:=httptest.NewRequest(http.MethodPost,"http://127.0.0.1:8317/api/control/resources",strings.NewReader(body))
  req.RemoteAddr="127.0.0.1:1024"
  req.Header.Set("Origin",origin)
  req.Header.Set("X-AhB-Control-Token",token)
  w:=httptest.NewRecorder()
  h.Handler().ServeHTTP(w,req)
  return w.Code
 }
 if got:=invoke("https://evil.test",h.controlToken,`{"max_running_sidecars":2,"idle_stop_seconds":900}`);got!=403 {t.Fatalf("csrf %d",got)}
 if got:=invoke("http://127.0.0.1:8317","bad",`{"max_running_sidecars":2,"idle_stop_seconds":900}`);got!=403 {t.Fatalf("token %d",got)}
 for _,raw:=range []string{
  `{"max_running_sidecars":0,"idle_stop_seconds":900}`,
  `{"max_running_sidecars":17,"idle_stop_seconds":900}`,
  `{"max_running_sidecars":2,"idle_stop_seconds":29}`,
  `{"max_running_sidecars":2,"idle_stop_seconds":900,"unknown":1}`,
  `{"max_running_sidecars":2}`,
 } {
  if code:=invoke("http://127.0.0.1:8317",h.controlToken,raw);code!=400 {t.Fatalf("expected 400 for %s got %d",raw,code)}
 }
 unchanged,err:=os.ReadFile(path)
 if err!=nil||string(unchanged)!=original||restarts!=0 {t.Fatal("invalid requests modified config")}
 if code:=invoke("http://127.0.0.1:8317",h.controlToken,`{"max_running_sidecars":3,"idle_stop_seconds":900}`);code!=202 {t.Fatalf("save %d",code)}
 bytes,err:=os.ReadFile(path);if err!=nil {t.Fatal(err)}
 var data map[string]any
 if err:=json.Unmarshal(bytes,&data);err!=nil {t.Fatal(err)}
 resources:=data["resources"].(map[string]any)
 if resources["max_running_sidecars"]!=float64(3)||resources["idle_stop_seconds"]!=float64(900)||resources["unrecognized"]!="keep" {t.Fatal("resource settings not preserved")}
 if data["custom_note"].(map[string]any)["keep"]!="yes" {t.Fatal("unknown config metadata lost")}
 if !strings.Contains(string(bytes),"Bearer keep-account-secret") {t.Fatal("credential field lost")}
 fi,_:=os.Stat(path);if fi.Mode().Perm()!=0600 {t.Fatalf("permissions %o",fi.Mode().Perm())}
 if restarts!=1 {t.Fatalf("scheduled %d restarts",restarts)}
 if code:=invoke("http://127.0.0.1:8317",h.controlToken,`{"max_running_sidecars":2,"idle_stop_seconds":120}`);code!=409 {t.Fatalf("double restart %d",code)}
}

func TestResourceSettingsRollbackOnRestartFailure(t *testing.T) {
 dir:=t.TempDir()
 path:=filepath.Join(dir,"config.json")
 original:=`{"listen":"127.0.0.1:8317","resources":{"max_running_sidecars":1,"idle_stop_seconds":120},"providers":[]}`
 if err:=os.WriteFile(path,[]byte(original),0600);err!=nil {t.Fatal(err)}
 h:=New(config.Config{Listen:"127.0.0.1:8317"})
 h.SetControlConfigPath(path)
 h.SetRestartHandler(func()error{return os.ErrPermission})
 req:=httptest.NewRequest(http.MethodPost,"http://127.0.0.1:8317/api/control/resources",strings.NewReader(`{"max_running_sidecars":2,"idle_stop_seconds":600}`))
 req.RemoteAddr="127.0.0.1:1234"
 req.Header.Set("Origin","http://127.0.0.1:8317")
 req.Header.Set("X-AhB-Control-Token",h.controlToken)
 rec:=httptest.NewRecorder()
 h.Handler().ServeHTTP(rec,req)
 if rec.Code!=500 {t.Fatalf("expected 500 got %d",rec.Code)}
 data,err:=os.ReadFile(path)
 if err!=nil||string(data)!=original {t.Fatal("did not restore original config after restart error")}
}
