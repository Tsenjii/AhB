package hub

import (
 "bytes"
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "os"
 "path/filepath"
 "strings"

 "github.com/Tsenjii/AhB/internal/config"
)

// SetControlConfigPath enables controlled on-disk toggles in the Android hub.
// The original file remains untouched on malformed config or failed validation.
func (h *Hub) SetControlConfigPath(path string) {
 h.controlConfigPath = path
 // Persist only non-secret model metadata next to the caller's private
 // config/data tree. The cache is not part of the upstream credentials.
 if root, err := filepath.Abs(filepath.Dir(path)); err == nil {
  h.modelCache.load(filepath.Join(root, "data", "hub-model-catalog.json"), h.providers)
 }
}

func atomicControlConfig(path string, raw []byte) error {
 dir := filepath.Dir(path)
 f, err := os.CreateTemp(dir, ".ahb-control-*.tmp")
 if err != nil { return err }
 name := f.Name()
 defer os.Remove(name)
 if err := f.Chmod(0600); err != nil { f.Close(); return err }
 if _, err := f.Write(raw); err != nil { f.Close(); return err }
 if err := f.Sync(); err != nil { f.Close(); return err }
 if err := f.Close(); err != nil { return err }
 return os.Rename(name, path)
}

func (h *Hub) handleControlProvider(w http.ResponseWriter, r *http.Request) {
 w.Header().Set("Cache-Control", "no-store")
 if r.Method != http.MethodPost {
  w.Header().Set("Allow", "POST")
  http.Error(w, "POST required", http.StatusMethodNotAllowed)
  return
 }
 if !h.authorizeLocalControl(w,r) { return }
 id := strings.TrimPrefix(r.URL.Path, "/api/control/provider/")
 if id == "" || strings.Contains(id, "/") {
  http.Error(w, "Unknown provider", http.StatusNotFound); return
 }
 p, exists := h.providers[id]
 if !exists || (p.cfg.Kind != "sidecar" && p.cfg.Kind != "external") {
  http.Error(w, "Unknown provider", http.StatusNotFound); return
 }
 var input struct { Enabled *bool `json:"enabled"` }
 decoder := json.NewDecoder(io.LimitReader(r.Body, 4096))
 decoder.DisallowUnknownFields()
 if err := decoder.Decode(&input); err != nil || input.Enabled == nil {
  http.Error(w, "Expected {enabled: true/false}", http.StatusBadRequest); return
 }
 if err := decoder.Decode(new(any)); err != io.EOF {
  http.Error(w, "Trailing request body", http.StatusBadRequest); return
 }

 h.controlMu.Lock()
 defer h.controlMu.Unlock()
 if h.controlPending {
  http.Error(w, "AhB restart is already scheduled", http.StatusConflict); return
 }
 if h.controlConfigPath == "" {
  http.Error(w, "No local configuration path", http.StatusForbidden); return
 }
 path, err := filepath.Abs(h.controlConfigPath)
 if err != nil { http.Error(w,"Invalid local config path",500); return }
 st, err := os.Lstat(path)
 if err != nil || !st.Mode().IsRegular() {
  http.Error(w, "Missing regular config file", http.StatusConflict); return
 }
 oldRaw, err := os.ReadFile(path)
 if err != nil { http.Error(w,"Cannot read configuration",500); return }
 var doc map[string]json.RawMessage
 if err := json.Unmarshal(oldRaw, &doc); err != nil {
  http.Error(w,"Malformed configuration",409); return
 }
 var entries []json.RawMessage
 if err := json.Unmarshal(doc["providers"], &entries); err != nil {
  http.Error(w,"Malformed providers",409); return
 }
 changed := false
 for i, raw := range entries {
  var item map[string]json.RawMessage
  if json.Unmarshal(raw, &item) != nil { http.Error(w,"Malformed provider",409);return }
  var existingID string
  if json.Unmarshal(item["id"], &existingID) != nil || existingID != id { continue }
  var existingEnabled bool
  _ = json.Unmarshal(item["enabled"], &existingEnabled)
  if existingEnabled == *input.Enabled {
   writeJSON(w,http.StatusOK,map[string]any{"status":"unchanged","id":id,"enabled":existingEnabled})
   return
  }
  // Avoid spinning a login-blocked Copilot service endlessly. Users should
  // finish an authorized device flow before turning this source on.
  if *input.Enabled && id == "copilot" {
   root := filepath.Dir(path)
   if !copilotHasLocalCredentials(root) {
    http.Error(w,"Copilot: finish GitHub device authorization first",409);return
   }
  }
  if *input.Enabled && id == "grok" {
   root := filepath.Dir(path)
   key, e := os.ReadFile(filepath.Join(root,"data","grok2api","client-key.txt"))
   if e != nil || len(bytes.TrimSpace(key)) == 0 {
    http.Error(w,"Grok: complete initial local key setup first",409);return
   }
   header := map[string]string{}
   _ = json.Unmarshal(item["headers"], &header)
   header["Authorization"] = "Bearer " + string(bytes.TrimSpace(key))
   item["headers"], _ = json.Marshal(header)
  }
  item["enabled"], _ = json.Marshal(*input.Enabled)
  entries[i], err = json.Marshal(item)
  if err != nil {http.Error(w,"Cannot update provider",500);return}
  changed = true
  break
 }
 if !changed { http.Error(w,"Provider missing from config.json",409);return }
 doc["providers"], err = json.Marshal(entries)
 if err != nil {http.Error(w,"Cannot encode providers",500);return}
 newRaw, err := json.MarshalIndent(doc,"","  ")
 if err != nil {http.Error(w,"Cannot encode configuration",500);return}
 newRaw = append(newRaw,'\n')
 // Validate every currently enabled provider *before* replacing config.
 // Paths remain relative to the original config directory.
 tmp, err := os.CreateTemp(filepath.Dir(path),".ahb-validate-*.json")
 if err != nil {http.Error(w,"Cannot validate configuration",500);return}
 validationPath := tmp.Name()
 _ = tmp.Close()
 defer os.Remove(validationPath)
 if err := os.WriteFile(validationPath,newRaw,0600); err != nil {
  http.Error(w,"Cannot validate configuration",500);return
 }
 if _, err := config.Load(validationPath); err != nil {
  http.Error(w,fmt.Sprintf("Invalid provider configuration: %s",err.Error()),409);return
 }
 if err := atomicControlConfig(path,newRaw); err != nil {
  http.Error(w,"Failed to save configuration",500);return
 }
 if err := h.restartFn(); err != nil {
  // The restart is not scheduled: immediately restore the original bytes.
  if rollbackErr := atomicControlConfig(path,oldRaw); rollbackErr != nil {
   http.Error(w,"Restart failed and manual config recovery required",500)
  } else {
   http.Error(w,"Restart failed; original configuration restored",500)
  }
  return
 }
 h.controlPending = true
 writeJSON(w,http.StatusAccepted,map[string]any{
  "status":"scheduled","id":id,"enabled":*input.Enabled,
  "message":"Provider setting saved; AhB will restart its own services. Termux stays open.",
 })
}
