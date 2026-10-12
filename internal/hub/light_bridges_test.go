package hub

import (
	"encoding/json"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tsenjii/AhB/internal/config"
)

func TestLightBridgeURLRejectsExternalSelfAndCredentials(t *testing.T) {
	for _, input := range []string{
		"https://127.0.0.1:8100",
		"http://example.com:8100",
		"http://192.168.1.9:8100",
		"http://127.0.0.1:8317",
		"http://user:pass@127.0.0.1:8100",
		"http://127.0.0.1:8100/private",
		"http://127.0.0.1:8100?token=secret",
		"http://127.0.0.1:8100#secret",
		"http://127.0.0.1:8100",
	} {
		_, err := lightBridgeURL(input,"127.0.0.1:8317")
		if strings.HasSuffix(input,":8100") && strings.HasPrefix(input,"http://127.") {
			if err!=nil {t.Fatalf("safe URL rejected: %s: %v", input, err)}
			continue
		}
		if err==nil {t.Fatalf("unsafe URL allowed: %s", input)}
	}
	if got,err:=lightBridgeURL("http://[::1]:8100","127.0.0.1:8317");err!=nil || got!="http://[::1]:8100" {
		t.Fatalf("IPv6 localhost unavailable: %q %v",got,err)
	}
	for _, path:=range []string{"/models","//v1/models","/v1/../models","/v1/models?token=x"} {
		if _,err:=lightBridgeModelsPath(path);err==nil {
			t.Fatalf("unexpected valid model path: %s",path)
		}
	}
	if got,err:=lightBridgeModelsPath("/openai/v1/models");err!=nil || got!="/openai/v1/models" {
		t.Fatalf("custom compatible prefix rejected: %s: %v",got,err)
	}
}

func TestLightBridgeProbeBoundedAndNoRedirect(t *testing.T) {
	h:=New(config.Config{Listen:"127.0.0.1:8317"})
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type","application/json")
			_,_=w.Write([]byte(`{"data":[{"id":"test"}]}`))
		case "/redirect":
			http.Redirect(w,r,"http://example.com",302)
		case "/bad":
			_,_=w.Write([]byte(`{"data":"not a list"}`))
		default:
			http.NotFound(w,r)
		}
	}))
	defer server.Close()
	ctx:=context.Background()
	if err:=h.probeLightBridge(ctx,server.URL,"/v1/models");err!=nil {t.Fatal(err)}
	for _,path:=range []string{"/redirect","/bad","/missing"} {
		if err:=h.probeLightBridge(ctx,server.URL,path);err==nil {t.Fatalf("probe should reject %s",path)}
	}
}

func TestLightBridgeAttachPreservesCredentialsAndIsAtomic(t *testing.T) {
	backend:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		_,_=w.Write([]byte(`{"object":"list","data":[{"id":"local"}]}`))
	}))
	defer backend.Close()
	dir:=t.TempDir()
	path:=filepath.Join(dir,"config.json")
	raw:=`{"listen":"127.0.0.1:8317","providers":[{"id":"native","kind":"sidecar","binary":"/usr/bin/true","base_url":"http://127.0.0.1:8500","enabled":false,"headers":{"Authorization":"Bearer secret-do-not-touch"}},{"id":"existing","kind":"external","base_url":"http://127.0.0.1:8811","enabled":false,"headers":{"Authorization":"Bearer retained-key"}}],"routing":{"same_model_fallback":{"enabled":false,"providers":["native"]}},"opaque_user_field":{"must":"remain"}}`
	if err:=os.WriteFile(path,[]byte(raw),0600);err!=nil {t.Fatal(err)}
	h:=New(config.Config{Listen:"127.0.0.1:8317",Providers:[]config.ProviderConfig{
		{ID:"native",Enabled:false,Kind:"sidecar",BaseURL:"http://127.0.0.1:8500",Binary:"/usr/bin/true"},
		{ID:"existing",Enabled:false,Kind:"external",BaseURL:"http://127.0.0.1:8811"},
	}})
	h.SetRestartHandler(func()error{return nil})
	h.SetControlConfigPath(path)
	request:=func(id,url string) *httptest.ResponseRecorder {
		t.Helper()
		payload,_:=json.Marshal(map[string]string{"id":id,"base_url":url,"models_path":"/v1/models"})
		req:=httptest.NewRequest("POST","http://127.0.0.1:8317/api/control/bridge-attach",strings.NewReader(string(payload)))
		req.RemoteAddr="127.0.0.1:41321"
		req.Header.Set("Origin","http://127.0.0.1:8317")
		req.Header.Set("X-AhB-Control-Token",h.controlToken)
		w:=httptest.NewRecorder()
		h.Handler().ServeHTTP(w,req)
		return w
	}
	if rec:=request("native",backend.URL);rec.Code!=409 {
		t.Fatalf("built-in provider overwritten: %d %s",rec.Code,rec.Body.String())
	}
	if stringMustReadFile(t,path)!=raw {t.Fatal("invalid request mutated config")}
	if rec:=request("mimo",backend.URL);rec.Code!=202 {
		t.Fatalf("good bridge refused: %d %s",rec.Code,rec.Body.String())
	}
	after:=stringMustReadFile(t,path)
	for _,need:=range []string{"retained-key","secret-do-not-touch","opaque_user_field","github.com/rong6/mimo-2api"} {
		if !strings.Contains(after,need){t.Fatalf("lost existing field %q",need)}
	}
	if strings.Contains(after,`"providers":["native","mimo"]`) {
		t.Fatal("auto-enrolled untested gateway into failover route")
	}
	info,err:=os.Stat(path)
	if err!=nil || info.Mode().Perm()!=0600 {t.Fatalf("config file permissions changed: %v",info)}
	if rec:=request("other",backend.URL);rec.Code!=409 {
		t.Fatalf("second attach without daemon restart should be blocked: %d",rec.Code)
	}
}

func TestLightBridgeAttachRollbackOnRestartFailure(t *testing.T) {
	backend:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		_,_=w.Write([]byte(`{"data":[]}`))
	}))
	defer backend.Close()
	dir:=t.TempDir();path:=filepath.Join(dir,"config.json")
	raw:=`{"listen":"127.0.0.1:8317","providers":[]}`
	if err:=os.WriteFile(path,[]byte(raw),0600);err!=nil {t.Fatal(err)}
	h:=New(config.Config{Listen:"127.0.0.1:8317"})
	h.SetRestartHandler(func()error{return errors.New("simulated restart failure")})
	h.SetControlConfigPath(path)
	req:=httptest.NewRequest("POST","http://127.0.0.1:8317/api/control/bridge-attach",strings.NewReader(
		`{"id":"mimo","base_url":"`+backend.URL+`","models_path":"/v1/models"}`))
	req.RemoteAddr="127.0.0.1:22222";req.Header.Set("Origin","http://127.0.0.1:8317")
	req.Header.Set("X-AhB-Control-Token",h.controlToken)
	rec:=httptest.NewRecorder();h.Handler().ServeHTTP(rec,req)
	if rec.Code!=500 {t.Fatalf("restart error not reported: %d: %s",rec.Code,rec.Body.String())}
	if got:=stringMustReadFile(t,path);got!=raw {t.Fatal("rollback failed")}
}
func stringMustReadFile(t *testing.T,path string)string {
	t.Helper();b,err:=os.ReadFile(path);if err!=nil {t.Fatal(err)};return string(b)
}
