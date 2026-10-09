package hub

import (
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"

 "github.com/Tsenjii/AhB/internal/config"
)

func TestCopilotDeviceLoginIsLocalAndOnlyExposesDeviceCode(t *testing.T) {
 root:=t.TempDir()
 bin:=filepath.Join(root,"bin")
 if err:=os.MkdirAll(bin,0700);err!=nil{t.Fatal(err)}
 script:="#!/bin/sh\n" +
  "dir=''\n" +
  "while [ $# -gt 0 ]; do if [ \"$1\" = '-token-dir' ]; then dir=\"$2\"; shift 2; else shift; fi; done\n" +
  "echo 'Please visit: https://github.com/login/device'\n" +
  "echo 'Enter code: TEST-1234'\n" +
  "sleep 1\n" +
  "printf '%s\\n' '{\"GitHubToken\":\"TOP_SECRET_FOR_TEST\"}' > \"$dir/credentials.json\"\n" +
  "sleep 3\n"
 if err:=os.WriteFile(filepath.Join(bin,"copilot2api"),[]byte(script),0700);err!=nil{t.Fatal(err)}
 cfg:=filepath.Join(root,"config.json")
 if err:=os.WriteFile(cfg,[]byte("{}"),0600);err!=nil{t.Fatal(err)}
 h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{{ID:"copilot",Kind:"sidecar",Enabled:false}}})
 h.SetControlConfigPath(cfg)
 h.SetRestartHandler(func()error{return nil})
 send:=func(origin,token,body string) (int,string) {
  req:=httptest.NewRequest(http.MethodPost,"http://127.0.0.1:8317/api/control/login/copilot",strings.NewReader(body))
  req.RemoteAddr="127.0.0.1:1002"
  req.Header.Set("Origin",origin)
  req.Header.Set("X-AhB-Control-Token",token)
  rec:=httptest.NewRecorder()
  h.Handler().ServeHTTP(rec,req)
  return rec.Code,rec.Body.String()
 }
 if code,_:=send("https://evil.test",h.controlToken,"{\"action\":\"start\"}");code!=403{t.Fatalf("CSRF got %d",code)}
 if code,_:=send("http://127.0.0.1:8317","invalid","{\"action\":\"start\"}");code!=403{t.Fatalf("invalid token got %d",code)}
 if code,_:=send("http://127.0.0.1:8317",h.controlToken,"{\"action\":\"start\"}");code!=202{t.Fatalf("login start got %d",code)}
 seenCode:=false
 seenDone:=false
 deadline:=time.Now().Add(5*time.Second)
 for time.Now().Before(deadline) {
  code,body:=send("http://127.0.0.1:8317",h.controlToken,"{\"action\":\"status\"}")
  if code!=200{t.Fatalf("login status got %d: %s",code,body)}
  if strings.Contains(body,"TOP_SECRET") {t.Fatal("private credential leaked through login status")}
  if strings.Contains(body,"TEST-1234"){seenCode=true}
  if strings.Contains(body,"\"done\""){seenDone=true;break}
  time.Sleep(120*time.Millisecond)
 }
 if !seenCode || !seenDone {t.Fatalf("expected device code and done: code %v done %v",seenCode,seenDone)}
 if _,err:=os.Stat(filepath.Join(root,"data","copilot2api","credentials.json"));err!=nil{t.Fatalf("local credentials missing: %v",err)}
}
