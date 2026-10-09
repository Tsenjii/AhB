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
func copilotHasLocalCredentials(root string) bool {
 f,err:=os.Stat(filepath.Join(root,"data","copilot2api","credentials.json"))
 return err==nil && f.Mode().IsRegular() && f.Size()>0
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
  writeJSON(w,200,map[string]string{"state":"done","detail":"本機 Copilot 授權資料已存在，可以使用首頁開關啟用。"})
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
 if p,ok:=h.providers["copilot"]; !ok || p.cfg.Enabled {
  http.Error(w,"Please disable Copilot before interactive device login",409);return
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
 if err:=cmd.Start();err!=nil {job.update("failed","","","Copilot 無法啟動");return}

 done:=make(chan struct{})
 go func(){
  defer close(done)
  scan:=bufio.NewScanner(pipe)
  scan.Buffer(make([]byte,4096),64*1024)
  for scan.Scan() {
   line:=strings.TrimSpace(scan.Text())
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
    job.update("failed","","","登入程序提早結束。請確認網路／Copilot 帳號資格後再試。")
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
