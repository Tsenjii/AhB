package hub

import (
 "encoding/json"
 "errors"
 "io"
 "net/http"
 "os"
 "path/filepath"
 "strings"
)

// AhB is a localhost-only account console. A user may open the original
// WebUI without manually searching Termux for each native admin password.
// This endpoint is an explicitly clicked *copy* operation, NOT SSO, and it
// never creates, weakens, resets, or forwards upstream credentials.
type adminConsoleDefinition struct {
 id string
 title string
 link string
 mode string
 username string
 help string
}
var adminConsoles = []adminConsoleDefinition{
 {"opencode","OpenCode Free","http://127.0.0.1:8404/","username_password","admin","讀取 OpenCode 目前本機 WebUI 設定。"},
 {"geminiweb","Gemini Web","http://127.0.0.1:8413/admin","admin_token","","讀取 AhB 啟動 Gemini Web 時使用的 ADMIN_TOKEN；來源可能有自己的登入規則。"},
 {"grok","Grok2API","http://127.0.0.1:8407/","bootstrap_password","admin","僅為初次建立管理員時使用的密碼；若在上游 UI 更改過，請使用新的密碼。"},
 {"kiro","Kiro-Go","http://127.0.0.1:8408/admin","admin_password","","Kiro 管理密碼；無須更改原本的帳號登入。"},
 {"agent2api","Agent2API","http://127.0.0.1:8403/","native","","使用上游原生管理方式，AhB 不會假裝有通用管理密碼。"},
}
func consoleDefinition(id string) (adminConsoleDefinition,bool) {
 for _,c:=range adminConsoles{if c.id==id{return c,true}}
 return adminConsoleDefinition{},false
}

func (h *Hub) handleConsoleAccess(w http.ResponseWriter,r *http.Request) {
 w.Header().Set("Cache-Control","no-store")
 w.Header().Set("Pragma","no-cache")
 w.Header().Set("Referrer-Policy","no-referrer")
 w.Header().Set("X-Content-Type-Options","nosniff")
 if r.Method!=http.MethodPost {w.Header().Set("Allow","POST");http.Error(w,"POST required",405);return}
 if !h.authorizeLocalControl(w,r){return}
 var input struct {
  Action string `json:"action"`
  Provider string `json:"provider,omitempty"`
 }
 dec:=json.NewDecoder(io.LimitReader(r.Body,2048))
 dec.DisallowUnknownFields()
 if err:=dec.Decode(&input);err!=nil || dec.Decode(new(any))!=io.EOF {
  http.Error(w,"Invalid console action",400);return
 }
 if input.Action!="list" && input.Action!="copy" {
  http.Error(w,"Unsupported console action",400);return
 }
 if input.Action=="list"&&input.Provider!=""{
  http.Error(w,"Provider not expected for list",400);return
 }
 if h.controlConfigPath==""||h.cfg.AllowLAN{
  http.Error(w,"Private local dashboard unavailable",403);return
 }
 root,err:=filepath.Abs(filepath.Dir(h.controlConfigPath))
 if err!=nil{http.Error(w,"Invalid local configuration",500);return}

 if input.Action=="list"{
  type row struct{
   ID string `json:"id"`
   Name string `json:"name"`
   URL string `json:"url"`
   Mode string `json:"mode"`
   Username string `json:"username,omitempty"`
   Help string `json:"help"`
   LocalAvailable bool `json:"local_available"`
  }
  entries:=make([]row,0,len(adminConsoles))
  for _,c:=range adminConsoles {
   _,exists:=h.providers[c.id]
   if !exists {continue}
   // Availability reports whether the expected private credential file is
   // accessible. A native-managed UI never has an AhB-managed password.
   available:=false
   if c.mode!="native"{
    if secret,_,err:=readLocalAdminSecret(root,c.id);err==nil&&secret!=""{
     available=true
    }
   }
   entries=append(entries,row{ID:c.id,Name:c.title,URL:c.link,Mode:c.mode,
    Username:c.username,Help:c.help,LocalAvailable:available})
  }
  writeJSON(w,200,map[string]any{"consoles":entries});return
 }
 c,ok:=consoleDefinition(input.Provider)
 if !ok||c.mode=="native"{
  http.Error(w,"No AhB-managed admin secret for this source",404);return
 }
 if _,ok:=h.providers[c.id];!ok{
  http.Error(w,"Gateway is not configured",404);return
 }
 secret,username,err:=readLocalAdminSecret(root,c.id)
 if err!=nil||secret==""{
  http.Error(w,"Local admin secret unavailable; check the source's native UI setup",409);return
 }
 // This is returned ONLY after a first-party button POST with strict local
 // origin+CSRF verification. It is never in the default HTML or /api/providers.
 writeJSON(w,200,map[string]string{"id":c.id,"username":username,"secret":secret})
}

func readLocalAdminSecret(root,id string) (secret,username string,err error) {
 username=""
 switch id {
 case "opencode":
  var doc struct {WebUI struct {
   Username string `json:"username"`
   Password string `json:"password"`
  } `json:"webui"` }
  raw,e:=readPrivateAdminFile(root,"data","opencode","config.json")
  if e!=nil{return "","",e}
  if e=json.Unmarshal(raw,&doc);e!=nil{return "","",e}
  if doc.WebUI.Password==""||strings.Contains(doc.WebUI.Password,"__AIHUB_"){return "","",errors.New("uninitialized password")}
  return doc.WebUI.Password,doc.WebUI.Username,nil
 case "geminiweb":
  raw,e:=readPrivateAdminFile(root,"config.json")
  if e!=nil{return "","",e}
  var doc struct {Providers []struct{
   ID string `json:"id"`
   Env map[string]string `json:"env"`
  } `json:"providers"`}
  if e=json.Unmarshal(raw,&doc);e!=nil{return "","",e}
  for _,p:=range doc.Providers{
   if p.ID=="geminiweb"{
    v:=p.Env["ADMIN_TOKEN"]
    if v==""||strings.Contains(v,"__AIHUB_"){return "","",errors.New("missing Gemini admin token")}
    return v,"",nil
   }
  }
  return "","",errors.New("no Gemini configuration")
 case "grok":
  raw,e:=readPrivateAdminFile(root,"data","grok2api","admin-password.txt")
  if e!=nil{return "","",e}
  return strings.TrimSpace(string(raw)),"admin",nil
 case "kiro":
  raw,e:=readPrivateAdminFile(root,"data","kiro-go","config.json")
  if e!=nil{return "","",e}
  var doc struct {Password string `json:"password"`}
  if e=json.Unmarshal(raw,&doc);e!=nil{return "","",e}
  return doc.Password,"",nil
 default:
  return "","",errors.New("source not supported")
 }
}

// Fixed relative paths only; reject all symlink path components to avoid
// turning this read-only private helper into a generic local-file viewer.
func readPrivateAdminFile(root string,parts ...string)([]byte,error){
 path:=root
 for _,part:=range parts {
  if part==""||part=="."||part==".."||strings.ContainsAny(part,"/\\"){
   return nil,errors.New("invalid private path")
  }
  path=filepath.Join(path,part)
  info,err:=os.Lstat(path)
  if err!=nil{return nil,err}
  if info.Mode()&os.ModeSymlink!=0{return nil,errors.New("symlink not permitted")}
  if part!=parts[len(parts)-1] {
   if !info.IsDir(){return nil,errors.New("non-directory credential path")}
  }
 }
 info,err:=os.Lstat(path)
 if err!=nil{return nil,err}
 if !info.Mode().IsRegular()||info.Size()>256*1024||info.Mode().Perm()&0077!=0{
  return nil,errors.New("unsafe or non-private admin file")
 }
 return os.ReadFile(path)
}
