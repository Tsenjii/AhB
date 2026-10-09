package hub

import (
 "bytes"
 "context"
 "crypto/rand"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "os"
 "path/filepath"
 "strings"
 "sync"
 "time"
)

// FreeBuff's documented CLI authorization flow is a browser-approved login,
// not a password form or a direct Google token handoff. The temporary URL can
// go to the local UI; the returned authToken must stay on the Hub filesystem.
const freebuffOAuthOrigin = "https://www.codebuff.com"

type freebuffLoginSession struct {
 mu sync.RWMutex
 state string
 link string
 detail string
 cancel context.CancelFunc
}

func (s *freebuffLoginSession) set(state, link, detail string) {
 s.mu.Lock()
 defer s.mu.Unlock()
 s.state=state
 s.link=link
 s.detail=detail
}
func (s *freebuffLoginSession) view(accountCount int) map[string]any {
 s.mu.RLock()
 defer s.mu.RUnlock()
 return map[string]any{"state":s.state,"url":s.link,"detail":s.detail,"accounts":accountCount}
}

func (h *Hub) handleFreebuffLogin(w http.ResponseWriter,r *http.Request) {
 w.Header().Set("Cache-Control","no-store")
 w.Header().Set("Referrer-Policy","no-referrer")
 if r.Method!=http.MethodPost {w.Header().Set("Allow","POST");http.Error(w,"POST required",405);return}
 if !h.authorizeLocalControl(w,r){return}
 var payload map[string]string
 dec:=json.NewDecoder(io.LimitReader(r.Body,1024))
 if err:=dec.Decode(&payload);err!=nil{http.Error(w,"Invalid JSON",400);return}
 if err:=dec.Decode(new(any));err!=io.EOF{http.Error(w,"Trailing JSON",400);return}
 if len(payload)!=1{http.Error(w,"Expected an action only",400);return}
 action:=payload["action"]
 if action!="start"&&action!="status"&&action!="cancel"{
  http.Error(w,"Unknown action",400);return
 }
 if h.controlConfigPath==""{http.Error(w,"Local AhB installation required",403);return}
 root,err:=filepath.Abs(filepath.Dir(h.controlConfigPath))
 if err!=nil{http.Error(w,"Invalid local installation",500);return}
 count:=countFreebuffCredentials(root)

 h.freebuffLoginMu.Lock()
 defer h.freebuffLoginMu.Unlock()
 if action=="status"{
  if h.freebuffLogin==nil{
   writeJSON(w,200,map[string]any{"state":"idle","accounts":count});return
  }
  writeJSON(w,200,h.freebuffLogin.view(count));return
 }
 if action=="cancel" {
  if h.freebuffLogin!=nil{
   if h.freebuffLogin.cancel!=nil{h.freebuffLogin.cancel()}
   h.freebuffLogin.set("cancelled","","已取消這次授權，之前的帳號不受影響。")
  }
  writeJSON(w,200,map[string]any{"state":"cancelled","accounts":count});return
 }
 h.controlMu.Lock()
 pending:=h.controlPending
 h.controlMu.Unlock()
 if pending{http.Error(w,"AhB restart pending",409);return}
 if p,ok:=h.providers["freebuff"];!ok||p.cfg.Kind!="sidecar"{
  http.Error(w,"FreeBuff gateway is not configured",409);return
 }
 if h.freebuffLogin!=nil{
  snap:=h.freebuffLogin.view(count)
  if snap["state"]=="starting"||snap["state"]=="waiting"{
   writeJSON(w,200,snap);return
  }
 }
 h.demandMu.Lock()
 baseCtx:=h.demandContext
 h.demandMu.Unlock()
 if baseCtx==nil||baseCtx.Err()!=nil{
  http.Error(w,"AhB is not running",503);return
 }
 ctx,cancel:=context.WithTimeout(baseCtx,5*time.Minute)
 session:=&freebuffLoginSession{state:"starting",detail:"正在向 FreeBuff 官方要求授權網址…",cancel:cancel}
 h.freebuffLogin=session
 go h.runFreebuffLogin(ctx,root,session)
 writeJSON(w,http.StatusAccepted,session.view(count))
}

// OAuth transport has a private fixed production origin. Tests can point the
// unexported Hub field at a local fake upstream; it is never user-configurable.
func (h *Hub) freebuffRequest(ctx context.Context,method,path string,body any) (int,map[string]json.RawMessage,error) {
 var data io.Reader
 if body!=nil{
  b,err:=json.Marshal(body)
  if err!=nil{return 0,nil,err}
  data=bytes.NewReader(b)
 }
 origin:=h.freebuffAuthBase
 if origin==""{origin=freebuffOAuthOrigin}
 req,err:=http.NewRequestWithContext(ctx,method,origin+path,data)
 if err!=nil{return 0,nil,err}
 req.Header.Set("Accept","application/json")
 if body!=nil{req.Header.Set("Content-Type","application/json")}
 client:=h.freebuffAuthClient
 if client==nil{client=&http.Client{Timeout:12*time.Second}}
 response,err:=client.Do(req)
 if err!=nil{return 0,nil,err}
 defer response.Body.Close()
 raw,err:=io.ReadAll(io.LimitReader(response.Body,64*1024+1))
 if err!=nil{return response.StatusCode,nil,err}
 if len(raw)>64*1024{return response.StatusCode,nil,errors.New("oversized upstream response")}
 var parsed map[string]json.RawMessage
 if len(raw)>0&&json.Unmarshal(raw,&parsed)!=nil{
  return response.StatusCode,nil,errors.New("invalid upstream JSON")
 }
 return response.StatusCode,parsed,nil
}

func upstreamString(data map[string]json.RawMessage,key string)string{
 var s string
 _=json.Unmarshal(data[key],&s)
 return s
}
func validFreebuffLink(link string)bool {
 if len(link)<10||len(link)>4096{return false}
 parsed,err:=url.Parse(link)
 if err!=nil||parsed.Scheme!="https"||parsed.User!=nil{return false}
 host:=strings.ToLower(parsed.Host)
 return host=="www.codebuff.com"||host=="codebuff.com"
}
func (h *Hub) runFreebuffLogin(ctx context.Context,root string,session *freebuffLoginSession){
 defer session.cancel()
 bytesID:=make([]byte,12)
 if _,err:=rand.Read(bytesID);err!=nil{session.set("failed","","無法產生授權識別碼");return}
 fingerprint:="codebuff-cli-"+hex.EncodeToString(bytesID)
 codeStatus,data,err:=h.freebuffRequest(ctx,"POST","/api/auth/cli/code",map[string]string{"fingerprintId":fingerprint})
 if err!=nil||codeStatus!=200 {
  session.set("failed","","無法連線 FreeBuff 官方授權服務，請稍後重試。");return
 }
 loginURL:=upstreamString(data,"loginUrl")
 hash:=upstreamString(data,"fingerprintHash")
 var expires string
 if err:=json.Unmarshal(data["expiresAt"],&expires);err!=nil {
  expires=string(data["expiresAt"])
 }
 if !validFreebuffLink(loginURL)||len(hash)==0||len(hash)>1024||len(expires)==0||len(expires)>128 {
  session.set("failed","","官方授權回應格式不正確，已拒絕不安全的網址。");return
 }
 session.set("waiting",loginURL,"請在官方頁面完成 Google 授權；AhB 會自動檢查結果，最多等待五分鐘。")
 interval:=h.freebuffPollInterval
 if interval<=0{interval=5*time.Second}
 ticker:=time.NewTicker(interval)
 defer ticker.Stop()
 invalidReplies:=0
 for {
  select{
  case <-ctx.Done():
   if errors.Is(ctx.Err(),context.DeadlineExceeded){
    session.set("expired","","授權已逾時，請重新開始。")
   }else{
    session.set("cancelled","","授權已取消或 AhB 正在結束。")
   }
   return
  case <-ticker.C:
  }
  query:=url.Values{}
  query.Set("fingerprintId",fingerprint)
  query.Set("fingerprintHash",hash)
  query.Set("expiresAt",expires)
  status,data,err:=h.freebuffRequest(ctx,"GET","/api/auth/cli/status?"+query.Encode(),nil)
  if err!=nil {invalidReplies++;if invalidReplies>=4{
    session.set("failed","","官方授權狀態查詢失敗，請確認網路後重試。");return
   };continue
  }
  switch status {
  case 200:
   var user map[string]json.RawMessage
   if err:=json.Unmarshal(data["user"],&user);err!=nil||user==nil{
    invalidReplies++
    if invalidReplies>=4{session.set("failed","","官方授權回應無效。");return}
    continue
   }
   token:=upstreamString(user,"authToken")
   if len(token)<9||len(token)>16*1024{
    session.set("failed","","授權成功但未取得有效的帳號憑證。");return
   }
   if err:=saveFreebuffToken(root,token);err!=nil{
    session.set("failed","","無法安全保存帳號到手機；請檢查私人儲存空間。");return
   }
   session.set("done","","登入成功！憑證只儲存在手機。若 FreeBuff 正在運作，請在總覽單獨重啟；休眠時下次啟動會自動載入。")
   return
  case 401:
   invalidReplies=0
  case 400,403:
   session.set("failed","","官方授權要求已失效或被拒絕，請重新開始。");return
  default:
   invalidReplies++
   if invalidReplies>=4{session.set("failed","","官方授權服務暫時不可用，請稍後重試。");return}
  }
 }
}

func freebuffCredentialsDir(root string) string {
 return filepath.Join(root,"data","freebuff","credentials")
}
func countFreebuffCredentials(root string)int {
 dir:=freebuffCredentialsDir(root)
 entries,err:=os.ReadDir(dir)
 if err!=nil{return 0}
 count:=0
 for _,item:=range entries{
  if !strings.HasSuffix(item.Name(),".json")||!item.Type().IsRegular(){continue}
  raw,err:=os.ReadFile(filepath.Join(dir,item.Name()))
  if err!=nil||len(raw)>32*1024{continue}
  var fields map[string]json.RawMessage
  if json.Unmarshal(raw,&fields)==nil&&len(upstreamString(fields,"authToken"))>=9{count++}
 }
 return count
}
func saveFreebuffToken(root,token string)error {
 if len(token)<9||len(token)>16*1024{return errors.New("invalid bearer token length")}
 dir:=freebuffCredentialsDir(root)
 if err:=os.MkdirAll(dir,0700);err!=nil{return err}
 for _,part:=range []string{filepath.Join(root,"data"),filepath.Join(root,"data","freebuff"),dir}{
  info,err:=os.Lstat(part)
  if err!=nil{return err}
  if !info.IsDir()||info.Mode()&os.ModeSymlink!=0{return fmt.Errorf("unsafe credential directory")}
 }
 sum:=sha256.Sum256([]byte(token))
 target:=filepath.Join(dir,hex.EncodeToString(sum[:9])+".json")
 content,err:=json.Marshal(map[string]string{"authToken":token})
 if err!=nil{return err}
 file,err:=os.CreateTemp(dir,".ahb-login-*.tmp")
 if err!=nil{return err}
 defer os.Remove(file.Name())
 if err:=file.Chmod(0600);err!=nil{_ =file.Close();return err}
 if _,err:=file.Write(content);err!=nil{_ =file.Close();return err}
 if err:=file.Sync();err!=nil{_ =file.Close();return err}
 if err:=file.Close();err!=nil{return err}
 return os.Rename(file.Name(),target)
}
