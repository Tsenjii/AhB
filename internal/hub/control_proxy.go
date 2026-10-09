package hub

import (
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "os"
 "path/filepath"
 "strings"

 "github.com/Tsenjii/AhB/internal/config"
)

// Only a best-effort child-process egress default. Source-native proxies
// (for example Agent2API per-account network pool) stay authoritative.
// The local dashboard never returns proxy endpoints or secrets to clients.
func (h *Hub) handleControlProxy(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Cache-Control","no-store")
 if r.Method!=http.MethodPost {w.Header().Set("Allow","POST");http.Error(w,"POST required",405);return}
 if !h.authorizeLocalControl(w,r){return}
 id:=strings.TrimPrefix(r.URL.Path,"/api/control/proxy/")
 if id==""||strings.ContainsAny(id,"/\\"){http.Error(w,"Unknown provider",404);return}
 p,ok:=h.providers[id]
 if !ok||p.cfg.Kind!="sidecar"{http.Error(w,"Managed sidecar required",404);return}
 var input struct {URL *string `json:"proxy_url"`}
 decoder:=json.NewDecoder(io.LimitReader(r.Body,4096));decoder.DisallowUnknownFields()
 if decoder.Decode(&input)!=nil||input.URL==nil||decoder.Decode(new(any))!=io.EOF{
  http.Error(w,"Expected a single proxy_url string",400);return
 }
 proxy:=strings.TrimSpace(*input.URL)
 if proxy!="" {
  if err:=config.ValidateProxyURL(proxy);err!=nil{
   http.Error(w,"Invalid proxy: "+err.Error(),400);return
  }
 }
 h.controlMu.Lock()
 defer h.controlMu.Unlock()
 if h.controlPending {http.Error(w,"AhB restart already scheduled",409);return}
 if h.controlConfigPath==""{http.Error(w,"Missing configuration path",403);return}
 path,err:=filepath.Abs(h.controlConfigPath)
 if err!=nil{http.Error(w,"Cannot resolve config path",500);return}
 st,err:=os.Lstat(path)
 if err!=nil||!st.Mode().IsRegular(){http.Error(w,"Missing regular config file",409);return}
 old,err:=os.ReadFile(path)
 if err!=nil{http.Error(w,"Cannot read config",500);return}
 var doc map[string]json.RawMessage
 if json.Unmarshal(old,&doc)!=nil {http.Error(w,"Invalid config",409);return}
 var items []json.RawMessage
 if json.Unmarshal(doc["providers"],&items)!=nil {http.Error(w,"Invalid providers",409);return}
 changed:=false
 for i,raw:=range items {
  var item map[string]json.RawMessage
  if json.Unmarshal(raw,&item)!=nil {http.Error(w,"Invalid provider item",409);return}
  var currentID string
  if json.Unmarshal(item["id"],&currentID)!=nil || currentID!=id {continue}
  var previous string
  _=json.Unmarshal(item["proxy_url"],&previous)
  if previous==proxy{
   writeJSON(w,200,map[string]any{"status":"unchanged","proxy_configured":proxy!=""});return
  }
  if proxy==""{delete(item,"proxy_url")}else{item["proxy_url"],_=json.Marshal(proxy)}
  items[i],err=json.Marshal(item)
  if err!=nil{http.Error(w,"Cannot encode provider",500);return}
  changed=true;break
 }
 if !changed {http.Error(w,"Provider missing from configuration",409);return}
 doc["providers"],err=json.Marshal(items)
 if err!=nil{http.Error(w,"Cannot encode providers",500);return}
 updated,err:=json.MarshalIndent(doc,"","  ")
 if err!=nil{http.Error(w,"Cannot encode config",500);return}
 updated=append(updated,'\n')
 tmp,err:=os.CreateTemp(filepath.Dir(path),".ahb-proxy-validate-*.json")
 if err!=nil{http.Error(w,"Cannot validate proxy settings",500);return}
 tmpName:=tmp.Name();_ =tmp.Close();defer os.Remove(tmpName)
 if os.WriteFile(tmpName,updated,0600)!=nil{http.Error(w,"Cannot validate proxy settings",500);return}
 if _,err:=config.Load(tmpName);err!=nil{
  http.Error(w,fmt.Sprintf("Invalid settings: %s",err),409);return
 }
 if atomicControlConfig(path,updated)!=nil{http.Error(w,"Cannot save configuration",500);return}
 if err:=h.restartFn();err!=nil {
  if atomicControlConfig(path,old)!=nil{http.Error(w,"Restart scheduling failed; restore config from backup",500)}else{
   http.Error(w,"Restart scheduling failed; original proxy policy restored",500)
  }
  return
 }
 h.controlPending=true
 writeJSON(w,202,map[string]any{
  "status":"scheduled","id":id,"proxy_configured":proxy!="",
  "message":"Per-gateway egress default saved. Restart AhB to apply; original account-native proxy pools remain unchanged.",
 })
}
