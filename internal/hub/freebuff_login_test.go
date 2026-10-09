package hub

import (
 "context"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "sync/atomic"
 "testing"
 "time"

 "github.com/Tsenjii/AhB/internal/config"
)

func TestFreebuffDashboardOAuthStoresCredentialsPrivatelyAndPreservesOthers(t *testing.T){
 root:=t.TempDir()
 if err:=os.WriteFile(filepath.Join(root,"config.json"),[]byte("{}"),0600);err!=nil{t.Fatal(err)}
 folder:=freebuffCredentialsDir(root)
 if err:=os.MkdirAll(folder,0700);err!=nil{t.Fatal(err)}
 const existing="existing-credential-value"
 if err:=os.WriteFile(filepath.Join(folder,"old.json"),[]byte("{\"authToken\":\""+existing+"\"}"),0600);err!=nil{t.Fatal(err)}

 const token="private-bearer-token-never-in-ui-123456789"
 var polls atomic.Int32
 auth:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Content-Type","application/json")
  switch r.URL.Path{
  case "/api/auth/cli/code":
   if r.Method!=http.MethodPost{t.Errorf("wrong method %s",r.Method)}
   var b map[string]string
   if err:=json.NewDecoder(r.Body).Decode(&b);err!=nil||!strings.HasPrefix(b["fingerprintId"],"codebuff-cli-"){t.Errorf("invalid code payload: %v, err=%v",b,err)}
   _,_=w.Write([]byte("{\"loginUrl\":\"https://www.codebuff.com/auth/cli?auth_code=one-time-secret\",\"fingerprintHash\":\"fixture-hash\",\"expiresAt\":1999999999}"))
  case "/api/auth/cli/status":
   if r.URL.Query().Get("fingerprintHash")!="fixture-hash"||r.URL.Query().Get("expiresAt")!="1999999999"{
    t.Errorf("invalid authorization poll %v",r.URL.Query())
   }
   if polls.Add(1)==1{w.WriteHeader(401);_,_=w.Write([]byte("{}"));return}
   _,_=w.Write([]byte("{\"user\":{\"authToken\":\""+token+"\",\"email\":\"never-share@example.com\"}}"))
  default: http.NotFound(w,r)
  }
 }))
 defer auth.Close()
 h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{{ID:"freebuff",Enabled:true,Kind:"sidecar",StartMode:"on_demand",Binary:"sleep",BaseURL:"http://127.0.0.1:8402"}}})
 h.SetControlConfigPath(filepath.Join(root,"config.json"))
 h.SetRestartHandler(func()error{return nil})
 h.freebuffAuthBase=auth.URL
 h.freebuffAuthClient=&http.Client{Timeout:time.Second}
 h.freebuffPollInterval=10*time.Millisecond
 ctx,cancel:=context.WithCancel(context.Background())
 h.Start(ctx)
 defer func(){cancel();h.Wait()}()

 csrf:=httptest.NewRequest(http.MethodPost,"http://127.0.0.1:8317/api/control/login/freebuff",strings.NewReader("{\"action\":\"start\"}"))
 csrf.RemoteAddr="127.0.0.1:9919"
 csrf.Header.Set("Origin","https://not-ahb.invalid")
 csrf.Header.Set("X-AhB-Control-Token",h.controlToken)
 denied:=httptest.NewRecorder()
 h.Handler().ServeHTTP(denied,csrf)
 if denied.Code!=403{t.Fatalf("cross origin OAuth start allowed: %d",denied.Code)}

 start:=localControlRequest(h,"POST","/api/control/login/freebuff","{\"action\":\"start\"}")
 if start.Code!=202{t.Fatalf("start HTTP %d: %s",start.Code,start.Body.String())}
 // Repeated start while polling must not launch a second authorization.
 again:=localControlRequest(h,"POST","/api/control/login/freebuff","{\"action\":\"start\"}")
 if again.Code!=200{t.Fatalf("second start HTTP %d",again.Code)}
 var state map[string]any
 deadline:=time.After(3*time.Second)
 for{
  response:=localControlRequest(h,"POST","/api/control/login/freebuff","{\"action\":\"status\"}")
  if response.Code!=200{t.Fatalf("status HTTP %d",response.Code)}
  if strings.Contains(response.Body.String(),token)||strings.Contains(response.Body.String(),"never-share@example.com"){
   t.Fatal("UI status leaked long-term token or account identity")
  }
  if err:=json.Unmarshal(response.Body.Bytes(),&state);err!=nil{t.Fatal(err)}
  if state["state"]=="done"{break}
  if state["state"]=="failed"{t.Fatalf("OAuth failed: %v",state)}
  select{case <-deadline:t.Fatalf("OAuth did not complete: %v",state);default:time.Sleep(8*time.Millisecond)}
 }
 if got:=countFreebuffCredentials(root);got!=2{t.Fatalf("expected 2 authorized accounts, got %d",got)}
 if data,err:=os.ReadFile(filepath.Join(folder,"old.json"));err!=nil||!strings.Contains(string(data),existing){t.Fatal("existing CLI login overwritten")}
 entries,_:=os.ReadDir(folder)
 found:=false
 for _,item:=range entries {
  if !strings.HasSuffix(item.Name(),".json")||item.Name()=="old.json"{continue}
  file:=filepath.Join(folder,item.Name())
  data,err:=os.ReadFile(file)
  if err!=nil{t.Fatal(err)}
  if !strings.Contains(string(data),token){t.Fatalf("new auth token not stored: %s",file)}
  st,err:=os.Stat(file)
  if err!=nil||st.Mode().Perm()!=0600{t.Fatal("new credentials are not owner-only")}
  found=true
 }
 if !found{t.Fatal("authorized token file missing")}
 if state["accounts"]!=float64(2){t.Fatalf("account count not reported safely: %v",state)}
}

func TestFreebuffDashboardRejectsUnsafeUpstreamURLAndKeepsOldFiles(t *testing.T) {
 root:=t.TempDir()
 if err:=os.WriteFile(filepath.Join(root,"config.json"),[]byte("{}"),0600);err!=nil{t.Fatal(err)}
 upstream:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Content-Type","application/json")
  _,_=w.Write([]byte("{\"loginUrl\":\"https://attacker.invalid/oauth?code=x\",\"fingerprintHash\":\"safe\",\"expiresAt\":1999999999}"))
 }))
 defer upstream.Close()
 h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{{ID:"freebuff",Kind:"sidecar",Enabled:true,StartMode:"on_demand",Binary:"sleep",BaseURL:"http://127.0.0.1:8402"}}})
 h.SetControlConfigPath(filepath.Join(root,"config.json"))
 h.SetRestartHandler(func()error{return nil})
 h.freebuffAuthBase=upstream.URL
 h.freebuffPollInterval=10*time.Millisecond
 ctx,cancel:=context.WithCancel(context.Background())
 h.Start(ctx)
 defer func(){cancel();h.Wait()}()
 first:=localControlRequest(h,"POST","/api/control/login/freebuff","{\"action\":\"start\"}")
 if first.Code!=202{t.Fatalf("start failed: %d",first.Code)}
 deadline:=time.After(2*time.Second)
 for{
  response:=localControlRequest(h,"POST","/api/control/login/freebuff","{\"action\":\"status\"}")
  if strings.Contains(response.Body.String(),"attacker.invalid"){t.Fatalf("untrusted OAuth link exposed: %s",response.Body.String())}
  var state map[string]any
  _=json.Unmarshal(response.Body.Bytes(),&state)
  if state["state"]=="failed"{break}
  select {case <-deadline:t.Fatal("unsafe URL not rejected");default:time.Sleep(10*time.Millisecond)}
 }
 if countFreebuffCredentials(root)!=0{t.Fatal("rejected URL wrote credentials")}
}


func TestFreebuffCancelNeverTurnsLateAuthorizationIntoSavedCredential(t *testing.T) {
 root:=t.TempDir()
 ctx,cancel:=context.WithCancel(context.Background())
 session:=&freebuffLoginSession{state:"waiting",link:"https://www.codebuff.com/auth/cli?auth_code=temp",cancel:cancel}
 session.stop()
 // A response might have been received immediately before cancellation.
 // It must not create credentials or turn the UI back to done.
 session.commitToken(ctx,root,"late-valid-bearer-token-from-upstream")
 session.set("done","","late network response")
 snapshot:=session.view(countFreebuffCredentials(root))
 if snapshot["state"]!="cancelled"||snapshot["url"]!=""{
  t.Fatalf("late callback changed cancelled OAuth state: %+v",snapshot)
 }
 if countFreebuffCredentials(root)!=0 {
  t.Fatal("cancelled OAuth wrote credentials")
 }
 session.stop() // idempotent
 if session.view(0)["state"]!="cancelled" {t.Fatal("second cancel changed terminal status")}
}
func TestFreebuffSuccessfulLoginCannotBeCancelledAfterPrivateCommit(t *testing.T) {
 root:=t.TempDir()
 ctx,cancel:=context.WithCancel(context.Background())
 defer cancel()
 session:=&freebuffLoginSession{state:"waiting",cancel:cancel}
 session.commitToken(ctx,root,"authorized-client-token-xyz-01234567")
 session.stop()
 if session.view(countFreebuffCredentials(root))["state"]!="done"{
  t.Fatal("cancelled a completed and persisted authorization")
 }
 if countFreebuffCredentials(root)!=1{t.Fatal("completed authorization was not persisted")}
}
