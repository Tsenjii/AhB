package hub

import (
 "bufio"
 "context"
 "encoding/json"
 "io"
 "net/http"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "sync"
 "time"
)

// Copilot's upstream binary owns the OAuth device flow and token storage.
// AhB only displays the *public authorization URL* and temporary user code.
// No GitHub tokens, upstream HTTP bodies or credential files enter the UI.
type copilotLoginSession struct {
 mu sync.RWMutex
 State string
 URL string
 UserCode string
 Detail string
}

func (j *copilotLoginSession) snapshot() map[string]string {
 j.mu.RLock()
 defer j.mu.RUnlock()
 return map[string]string{
  "state":j.State,"url":j.URL,"user_code":j.UserCode,"detail":j.Detail,
 }
}
func (j *copilotLoginSession) update(state,url,code,detail string) {
 j.mu.Lock()
 defer j.mu.Unlock()
 if state!="" { j.State=state }
 if url!="" {j.URL=url}
 if code!="" {j.UserCode=code}
 if detail!="" {j.Detail=detail}
}
// Map known startup errors to bounded, user-facing hints. Never expose
// raw upstream logs or GitHub credentials to the local dashboard.
func copilotSafeFailureHint(line string) string {
 lower:=strings.ToLower(line)
 switch {
 case strings.Contains(lower,"exec format error"):
  return "Copilot 執行檔架構不符，請檢查 Android ARM64 安裝包。"
 case strings.Contains(lower,"no such host")||strings.Contains(lower,"lookup github.com")||strings.Contains(lower,"network is unreachable"):
  return "Termux 無法連上 GitHub（DNS／網路），請先檢查手機網路。"
 case strings.Contains(lower,"failed to request device code")||strings.Contains(lower,"failed to initiate device flow"):
  return "GitHub 裝置授權請求失敗，請檢查 Termux 的網路。"
 case strings.Contains(lower,"access_denied")||strings.Contains(lower,"authorization declined"):
  return "GitHub 裝置授權被拒絕，請重新登入。"
 case strings.Contains(lower,"expired_token")||strings.Contains(lower,"polling timeout"):
  return "一次性授權碼已過期，請重新取得。"
 case strings.Contains(lower,"failed to get copilot token")||strings.Contains(lower,"copilot_internal")||strings.Contains(lower,"failed to refresh copilot token"):
  return "GitHub 已授權，但無法取得 Copilot Token；請檢查帳號使用資格。"
 case strings.Contains(lower,"failed to get access token")||strings.Contains(lower,"token error"):
  return "GitHub 裝置授權未完成或已失效，請重新登入。"
 case strings.Contains(lower,"connection refused")||strings.Contains(lower,"i/o timeout"):
  return "無法連接 GitHub，請檢查手機網路。"
 }
 return ""
}
func copilotHasLocalCredentials(root string) bool {
 path:=filepath.Join(root,"data","copilot2api","credentials.json")
 st,err:=os.Lstat(path)
 if err!=nil||!st.Mode().IsRegular() {return false}
 raw,err:=os.ReadFile(path)
 if err!=nil||len(raw)>256*1024 {return false}
 var doc map[string]json.RawMessage
 if json.Unmarshal(raw,&doc)!=nil{return false}
 var githubToken string
 if json.Unmarshal(doc["github_token"],&githubToken)!=nil{return false}
 // Nonempty valid-format field only; actual upstream entitlement is checked
 // by Copilot at startup and by a real model request.
 return len(strings.TrimSpace(githubToken))>12
}

func (h *Hub) handleCopilotLogin(w http.ResponseWriter,r *http.Request) {
 w.Header().Set("Cache-Control","no-store")
 if r.Method!=http.MethodPost { http.Error(w,"POST required",405);return }
 if !h.authorizeLocalControl(w,r) {return}
 if h.controlConfigPath=="" {http.Error(w,"Local login unavailable",403);return}
 var request struct {Action string `json:"action"`}
 dec:=json.NewDecoder(io.LimitReader(r.Body,1024))
 dec.DisallowUnknownFields()
 if err:=dec.Decode(&request);err!=nil {http.Error(w,"Invalid action",400);return}
 if request.Action!="start" && request.Action!="status" {
  http.Error(w,"Unknown action",400);return
 }
 root:=filepath.Dir(h.controlConfigPath)
 if !filepath.IsAbs(root) {
  var err error
  root,err=filepath.Abs(root)
  if err!=nil {http.Error(w,"Invalid local installation",500);return}
 }
 h.copilotLoginMu.Lock()
 defer h.copilotLoginMu.Unlock()
 if copilotHasLocalCredentials(root) {
  writeJSON(w,200,map[string]string{"state":"done","detail":"本機已有 GitHub 授權資料；Copilot 模型與額度仍須實際驗證。"})
  return
 }
 if request.Action=="status" {
  if h.copilotLogin==nil {writeJSON(w,200,map[string]string{"state":"idle"});return}
  writeJSON(w,200,h.copilotLogin.snapshot())
  return
 }
 // During a pending Hub restart, never start another login process.
 h.controlMu.Lock()
 busy:=h.controlPending
 h.controlMu.Unlock()
 if busy {http.Error(w,"AhB restart pending",409);return}
 if p,ok:=h.providers["copilot"]; !ok || (p.cfg.Enabled && (!h.onDemand(p) || p.snapshot().PID>0)) {
  http.Error(w,"Copilot 程序正在執行，請先停用此來源再開始裝置授權。",409);return
 }
 if h.copilotLogin!=nil {
  state:=h.copilotLogin.snapshot()["state"]
  if state=="starting" || state=="waiting" {
   writeJSON(w,200,h.copilotLogin.snapshot());return
  }
 }
 binary:=filepath.Join(root,"bin","copilot2api")
 if f,err:=os.Stat(binary);err!=nil||f.IsDir()||f.Mode()&0111==0 {
  http.Error(w,"Bundled Copilot binary unavailable",409);return
 }
 credsDir:=filepath.Join(root,"data","copilot2api")
 if err:=os.MkdirAll(credsDir,0700);err!=nil {
  http.Error(w,"Cannot create private credential directory",500);return
 }
 job:=&copilotLoginSession{State:"starting",Detail:"正在啟動 GitHub 裝置授權…"}
 h.copilotLogin=job
 go runCopilotLogin(root,binary,credsDir,job)
 writeJSON(w,http.StatusAccepted,job.snapshot())
}

func runCopilotLogin(root,binary,credsDir string,job *copilotLoginSession) {
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Minute)
 defer cancel()
 cmd:=exec.CommandContext(ctx,binary,"-host","127.0.0.1","-port","8411","-token-dir",credsDir)
 cmd.Dir=root
 cmd.Env=append(os.Environ(),"GODEBUG=netdns=cgo")
 pipe,err:=cmd.StdoutPipe()
 if err!=nil {job.update("failed","","","無法啟動登入流程");return}
 // Deliberately discard stderr and do not record stdout. The token never
 // appears in the dashboard, terminal logs or AhB's JSON status endpoints.
 cmd.Stderr=io.Discard
 if err:=cmd.Start();err!=nil {
  hint:=copilotSafeFailureHint(err.Error())
  if hint==""{hint="Copilot 執行檔無法啟動，請確認 Android ARM64 版本及執行權限。"}
  job.update("failed","","",hint);return
 }

 var hintMu sync.Mutex
 failureHint:=""
 observe:=func(line string) {
  if hint:=copilotSafeFailureHint(line);hint!=""{
   hintMu.Lock();failureHint=hint;hintMu.Unlock()
  }
 }
 safeHint:=func() string {
  hintMu.Lock();defer hintMu.Unlock()
  if failureHint!=""{return failureHint}
  return "登入程序提早結束，請確認裝置授權、Termux 網路與 Copilot 資格。"
 }
 done:=make(chan struct{})
 go func(){
  defer close(done)
  scan:=bufio.NewScanner(pipe)
  scan.Buffer(make([]byte,4096),64*1024)
  for scan.Scan() {
   line:=strings.TrimSpace(scan.Text())
   observe(line)
   if strings.HasPrefix(line,"Please visit: ") {
    url:=strings.TrimSpace(strings.TrimPrefix(line,"Please visit: "))
    if url=="https://github.com/login/device" || url=="https://github.com/login/device/" {
     job.update("waiting",url,"","使用自己的 GitHub 帳號完成授權。")
    }
   }
   if strings.HasPrefix(line,"Enter code: ") {
    code:=strings.TrimSpace(strings.TrimPrefix(line,"Enter code: "))
    if len(code)>=5 && len(code)<=16 {
     valid:=true
     for _,c:=range code {if !((c>='A'&&c<='Z')||(c>='0'&&c<='9')||c=='-'){valid=false;break}}
     if valid {job.update("waiting","",code,"授權後會自動保存到手機，不用貼給任何人。")}
    }
   }
  }
 }()
 // The upstream persists credentials *after* receiving and validating the
 // OAuth token. Never claim success merely because a code was displayed.
 ticker:=time.NewTicker(400*time.Millisecond)
 defer ticker.Stop()
 for {
  select{
  case <-ticker.C:
   if copilotHasLocalCredentials(root) {
    job.update("done","","","GitHub 已授權，憑證僅儲存在手機。請在首頁啟用 Copilot。")
    _=cmd.Process.Signal(os.Interrupt)
    wait:=make(chan struct{})
    go func(){_ =cmd.Wait();close(wait)}()
    select{case <-wait:case <-time.After(3*time.Second):_ =cmd.Process.Kill();<-wait}
    <-done
    return
   }
  case <-done:
   // Upstream may exit immediately on DNS, quota, or OAuth failures.
   _=cmd.Wait()
   if copilotHasLocalCredentials(root) {
    job.update("done","","","GitHub 授權資料已儲存在手機。")
   } else {
    job.update("failed","","",safeHint())
   }
   return
  case <-ctx.Done():
   _=cmd.Process.Kill()
   _=cmd.Wait()
   job.update("failed","","","登入逾時，請重新開始官方 GitHub 授權。")
   return
  }
 }
}
