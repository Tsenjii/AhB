package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
)

// This is a local connection wizard, not an installer or browser automation.
// A candidate must already expose a compatible loopback API and an ordinary
// /models listing. No website cookies, OAuth sessions or passwords are read.
type lightBridgePreset struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Docs string `json:"docs"`
	Port int `json:"port"`
	ModelsPath string `json:"models_path"`
	Runtime string `json:"runtime"`
	Notes string `json:"notes"`
}

var lightBridgePresets = []lightBridgePreset{
	{ID:"cliproxy", Name:"CLIProxyAPI", Docs:"https://github.com/router-for-me/CLIProxyAPI", Port:8416, ModelsPath:"/v1/models", Runtime:"Go", Notes:"已有官方 CLI 授權才連接；需預先改掉與 AhB 衝突的預設埠"},
	{ID:"qwenweb", Name:"Qwen Web · Go", Docs:"https://github.com/XxxXTeam/Qwen2API_Go", Port:3000, ModelsPath:"/v1/models", Runtime:"Go", Notes:"網頁橋接器，先在來源本身完成帳號驗證"},
	{ID:"kimi", Name:"Kimi2API", Docs:"https://github.com/chopper1026/kimi2api", Port:8000, ModelsPath:"/v1/models", Runtime:"Python", Notes:"維護自己的帳號池，並非 Go 單執行檔"},
	{ID:"mimo", Name:"MiMo 2API", Docs:"https://github.com/rong6/mimo-2api", Port:8090, ModelsPath:"/v1/models", Runtime:"Go", Notes:"獨立 Go 網頁橋接器；有自己的認證與服務條款"},
	{ID:"geminioauth", Name:"Gemini CLI OAuth Bridge", Docs:"https://github.com/kimws/gemini-oauth", Port:10551, ModelsPath:"/v1/models", Runtime:"Node.js", Notes:"需合法的 Gemini CLI 登入；Code Assist 私有 API 可能變更"},
	{ID:"claudecli", Name:"Claude CLI OpenAI Proxy", Docs:"https://github.com/kpod13/claude-openai-proxy", Port:8080, ModelsPath:"/v1/models", Runtime:"Go + CLI", Notes:"依賴外部 Claude CLI，可能受使用資格及 CLI 啟動成本影響"},
	{ID:"deepseekweb", Name:"DeepSeek Web · Chromium", Docs:"https://github.com/PruhaNLP/web-to-api", Port:3456, ModelsPath:"/v1/models", Runtime:"Browser", Notes:"需要 Chromium，並非低記憶體替代品"},
	{ID:"custom", Name:"自訂 OpenAI 相容本機 API", Docs:"", Port:8418, ModelsPath:"/v1/models", Runtime:"External", Notes:"只接受同機 localhost，未安裝的服務不會被自動啟動"},
}

func lookupLightBridgePreset(id string) (lightBridgePreset, bool) {
	for _, p := range lightBridgePresets {
		if p.ID == id {
			return p, true
		}
	}
	return lightBridgePreset{}, false
}

func (h *Hub) handleBridgeCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w,"GET required", http.StatusMethodNotAllowed)
		return
	}
	// No credentials in this public metadata document.
	writeJSON(w, http.StatusOK, map[string]any{"presets": lightBridgePresets})
}

var bridgeNameRE = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,30}$`)
var bridgeModelsRE = regexp.MustCompile(`^/[a-zA-Z0-9/_-]+/models$`)

// Reject public destinations, credentials, fragments, URL paths, query
// strings and AhB's own listener. Avoids self-registration and SSRF.
func lightBridgeURL(raw, listen string) (string, error) {
	if len(raw) > 180 || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf("invalid local bridge URL")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" ||
		u.Fragment != "" || (u.Path != "" && u.Path != "/") || u.Opaque != "" {
		return "", fmt.Errorf("bridge URL must be plain loopback HTTP without path, query or credentials")
	}
	host := u.Hostname()
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return "", fmt.Errorf("bridge must use loopback HTTP")
		}
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("bridge must specify a valid TCP port")
	}
	_, listenPort, _ := net.SplitHostPort(listen)
	if u.Port() == listenPort {
		return "", fmt.Errorf("bridge cannot connect back into the AhB listener")
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)), nil
}

func lightBridgeModelsPath(path string) (string, error) {
	if path == "" {
		return "/v1/models", nil
	}
	if len(path) > 100 || !bridgeModelsRE.MatchString(path) ||
		strings.Contains(path,"//") || strings.Contains(path, "..") {
		return "", fmt.Errorf("invalid models path")
	}
	return path, nil
}

func (h *Hub) probeLightBridge(ctx context.Context, base, modelsPath string) error {
	// Do not consult HTTP_PROXY/HTTPS_PROXY for a localhost security probe.
	// No URL redirects are followed, even if the target returns a public Location.
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:nil,
			DialContext: (&net.Dialer{Timeout:2*time.Second}).DialContext,
			DisableKeepAlives:true,
			ResponseHeaderTimeout:4*time.Second,
		},
		Timeout: 5*time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer client.CloseIdleConnections()
	url := base + modelsPath
	req, err := http.NewRequestWithContext(ctx,http.MethodGet,url,nil)
	if err != nil { return fmt.Errorf("invalid bridge health check") }
	resp, err := client.Do(req)
	if err != nil { return fmt.Errorf("local bridge did not respond to models probe") }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("local models endpoint returned HTTP %d; use the private terminal connector for services needing a key", resp.StatusCode)
	}
	var doc struct{ Data []json.RawMessage `json:"data"` }
	dec:=json.NewDecoder(io.LimitReader(resp.Body,1<<20))
	if dec.Decode(&doc)!=nil || doc.Data == nil {
		return fmt.Errorf("bridge does not expose a valid bounded OpenAI models list")
	}
	return nil
}

// Use the same authenticated, atomic and rollback-safe control plane as
// native enable/disable operations. Never overwrite managed native sidecars.
func (h *Hub) handleBridgeAttach(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control","no-store")
	if r.Method!=http.MethodPost {
		w.Header().Set("Allow","POST")
		http.Error(w,"POST required",http.StatusMethodNotAllowed)
		return
	}
	if !h.authorizeLocalControl(w,r) {return}
	var request struct {
		ID string `json:"id"`
		BaseURL string `json:"base_url"`
		ModelsPath string `json:"models_path"`
	}
	dec:=json.NewDecoder(io.LimitReader(r.Body,4096))
	dec.DisallowUnknownFields()
	if err:=dec.Decode(&request);err!=nil || dec.Decode(new(any))!=io.EOF {
		http.Error(w,"Expected id, base_url, models_path",http.StatusBadRequest)
		return
	}
	if !bridgeNameRE.MatchString(request.ID) || request.ID=="route" {
		http.Error(w,"Invalid bridge ID",http.StatusBadRequest);return
	}
	base,err:=lightBridgeURL(request.BaseURL,h.cfg.Listen)
	if err!=nil {http.Error(w,err.Error(),http.StatusBadRequest);return}
	modelsPath,err:=lightBridgeModelsPath(request.ModelsPath)
	if err!=nil {http.Error(w,err.Error(),http.StatusBadRequest);return}
	prefix:=strings.TrimSuffix(modelsPath,"/models")
	if existing:=h.providers[request.ID];existing!=nil && existing.cfg.Kind!="external" {
		http.Error(w,"Cannot replace a built-in managed sidecar",http.StatusConflict);return
	}
	if err:=h.probeLightBridge(r.Context(),base,modelsPath);err!=nil {
		http.Error(w,err.Error(),http.StatusBadGateway);return
	}

	h.controlMu.Lock()
	defer h.controlMu.Unlock()
	if h.controlPending {http.Error(w,"Another AhB operation is pending",409);return}
	if h.controlConfigPath=="" {http.Error(w,"Local config path unavailable",403);return}
	path,err:=filepath.Abs(h.controlConfigPath)
	if err!=nil {http.Error(w,"Invalid local config path",500);return}
	st,err:=os.Lstat(path)
	if err!=nil || !st.Mode().IsRegular() {http.Error(w,"Regular config file required",409);return}
	oldRaw,err:=os.ReadFile(path)
	if err!=nil {http.Error(w,"Cannot read config",500);return}
	var doc map[string]json.RawMessage
	if json.Unmarshal(oldRaw,&doc)!=nil {http.Error(w,"Invalid configuration",409);return}
	var entries []json.RawMessage
	if json.Unmarshal(doc["providers"],&entries)!=nil {http.Error(w,"Invalid provider list",409);return}
	var previous map[string]json.RawMessage
	found:=-1
	for i,raw:=range entries {
		var item map[string]json.RawMessage
		if json.Unmarshal(raw,&item)!=nil {http.Error(w,"Malformed provider",409);return}
		var id,kind string
		_=json.Unmarshal(item["id"],&id)
		_=json.Unmarshal(item["kind"],&kind)
		if id!=request.ID {continue}
		if kind!="external" {http.Error(w,"Managed provider cannot be overwritten",409);return}
		found=i
		previous=item
		break
	}
	preset,known:=lookupLightBridgePreset(request.ID)
	name,docs:=request.ID+" Bridge",""
	if known {name,docs=preset.Name,preset.Docs}
	if previous==nil {previous=make(map[string]json.RawMessage)}
	set:=func(key string,v any){previous[key],_=json.Marshal(v)}
	if found==-1 {
		set("id",request.ID)
		set("kind","external")
		set("display_name",name)
		set("description","User-operated local OpenAI-compatible bridge")
		set("docs_url",docs)
		set("health_interval_seconds",15)
	}
	set("enabled",true)
	set("base_url",base)
	set("models_path",modelsPath)
	set("health_path",modelsPath)
	set("api_path_prefix",prefix)
	// Preserve prior Authorization for known bridges. New bridges start with
	// NO key. Credentials are provided only through the private CLI connector.
	encoded,err:=json.Marshal(previous)
	if err!=nil {http.Error(w,"Cannot encode bridge",500);return}
	if found==-1 {entries=append(entries,encoded)} else {entries[found]=encoded}
	doc["providers"],_=json.Marshal(entries)

	// Do not auto-enroll in failover routing. Probe success does not establish
	// tool support, account quotas, or even inference viability.
	newRaw,err:=json.MarshalIndent(doc,"","  ")
	if err!=nil {http.Error(w,"Cannot encode configuration",500);return}
	newRaw=append(newRaw,'\n')
	temp,err:=os.CreateTemp(filepath.Dir(path),".ahb-bridge-validate-*.json")
	if err!=nil {http.Error(w,"Cannot validate bridge configuration",500);return}
	validationPath:=temp.Name()
	_=temp.Close()
	defer os.Remove(validationPath)
	if os.WriteFile(validationPath,newRaw,0600)!=nil {http.Error(w,"Cannot validate config",500);return}
	if _,err=config.Load(validationPath);err!=nil {
		http.Error(w,"Bridge config validation failed: "+err.Error(),409);return
	}
	if err=atomicControlConfig(path,newRaw);err!=nil {http.Error(w,"Cannot save config",500);return}
	if err=h.restartFn();err!=nil {
		if atomicControlConfig(path,oldRaw)!=nil {
			http.Error(w,"Restart failed; manual config recovery required",500)
		}else{
			http.Error(w,"Restart failed; original configuration restored",500)
		}
		return
	}
	h.controlPending=true
	writeJSON(w,http.StatusAccepted,map[string]string{
		"status":"scheduled","id":request.ID,
		"message":"Verified local models endpoint; bridge configuration saved. AhB restarting. Inference still needs Playground verification.",
	})
}
