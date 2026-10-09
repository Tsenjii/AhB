package hub

import (
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "os"
 "path/filepath"

 "github.com/Tsenjii/AhB/internal/config"
)

// handleControlResources changes the on-demand process ceiling and idle time
// through the same strict local UI authorization and atomic restart policy as
// the provider toggle. Existing secret values and unknown JSON fields survive.
func (h *Hub) handleControlResources(w http.ResponseWriter, r *http.Request) {
 w.Header().Set("Cache-Control","no-store")
 if r.Method!=http.MethodPost {w.Header().Set("Allow","POST");http.Error(w,"POST required",http.StatusMethodNotAllowed);return}
 if !h.authorizeLocalControl(w,r) {return}
 var input struct{
  MaxRunningSidecars *int `json:"max_running_sidecars"`
  IdleStopSeconds *int `json:"idle_stop_seconds"`
 }
 dec:=json.NewDecoder(io.LimitReader(r.Body,4096))
 dec.DisallowUnknownFields()
 if dec.Decode(&input)!=nil || input.MaxRunningSidecars==nil || input.IdleStopSeconds==nil {
  http.Error(w,"Expected max_running_sidecars and idle_stop_seconds",http.StatusBadRequest);return
 }
 if err:=dec.Decode(new(any));err!=io.EOF {http.Error(w,"Trailing JSON",http.StatusBadRequest);return}
 if *input.MaxRunningSidecars<1 || *input.MaxRunningSidecars>16 ||
    *input.IdleStopSeconds<30 || *input.IdleStopSeconds>86400 {
  http.Error(w,"Limits: 1-16 providers, 30-86400 seconds idle",http.StatusBadRequest);return
 }
 h.controlMu.Lock()
 defer h.controlMu.Unlock()
 if h.controlPending {http.Error(w,"AhB restart is already scheduled",http.StatusConflict);return}
 if h.controlConfigPath=="" {http.Error(w,"No local configuration",http.StatusForbidden);return}
 path,err:=filepath.Abs(h.controlConfigPath)
 if err!=nil {http.Error(w,"Invalid config location",500);return}
 st,err:=os.Lstat(path)
 if err!=nil || !st.Mode().IsRegular() {http.Error(w,"Missing regular configuration",http.StatusConflict);return}
 oldRaw,err:=os.ReadFile(path)
 if err!=nil {http.Error(w,"Cannot read config",500);return}
 var doc map[string]json.RawMessage
 if err:=json.Unmarshal(oldRaw,&doc);err!=nil {http.Error(w,"Invalid config",http.StatusConflict);return}
 var resource map[string]json.RawMessage
 if raw:=doc["resources"];len(raw)>0 {
  if err:=json.Unmarshal(raw,&resource);err!=nil {http.Error(w,"Invalid resources",http.StatusConflict);return}
 }
 if resource==nil {resource=make(map[string]json.RawMessage)}
 var current struct{
  MaxRunningSidecars int `json:"max_running_sidecars"`
  IdleStopSeconds int `json:"idle_stop_seconds"`
 }
 _=json.Unmarshal(doc["resources"],&current)
 if current.MaxRunningSidecars==0 {current.MaxRunningSidecars=1}
 if current.IdleStopSeconds==0 {current.IdleStopSeconds=120}
 if current.MaxRunningSidecars==*input.MaxRunningSidecars &&
    current.IdleStopSeconds==*input.IdleStopSeconds {
  writeJSON(w,http.StatusOK,map[string]any{"status":"unchanged"});return
 }
 resource["max_running_sidecars"],_=json.Marshal(*input.MaxRunningSidecars)
 resource["idle_stop_seconds"],_=json.Marshal(*input.IdleStopSeconds)
 doc["resources"],_=json.Marshal(resource)
 newRaw,err:=json.MarshalIndent(doc,"","  ")
 if err!=nil {http.Error(w,"Cannot encode config",500);return}
 newRaw=append(newRaw,'\n')
 // Load from the same directory so relative executable paths remain correct.
 tmp,err:=os.CreateTemp(filepath.Dir(path),".ahb-resource-validate-*.json")
 if err!=nil {http.Error(w,"Cannot validate settings",500);return}
 validationPath:=tmp.Name()
 _=tmp.Close()
 defer os.Remove(validationPath)
 if err:=os.WriteFile(validationPath,newRaw,0600);err!=nil {http.Error(w,"Cannot validate settings",500);return}
 if _,err:=config.Load(validationPath);err!=nil {
  http.Error(w,fmt.Sprintf("Invalid settings: %s",err.Error()),http.StatusConflict);return
 }
 if err:=atomicControlConfig(path,newRaw);err!=nil {http.Error(w,"Unable to save settings",500);return}
 if err:=h.restartFn();err!=nil {
  if atomicControlConfig(path,oldRaw)!=nil {http.Error(w,"Restart failed; manual configuration recovery required",500)} else {
   http.Error(w,"Restart failed; original settings restored",500)
  }
  return
 }
 h.controlPending=true
 writeJSON(w,http.StatusAccepted,map[string]any{
  "status":"scheduled","max_running_sidecars":*input.MaxRunningSidecars,
  "idle_stop_seconds":*input.IdleStopSeconds,
  "message":"Resource policy saved; AhB will restart its own services.",
 })
}
