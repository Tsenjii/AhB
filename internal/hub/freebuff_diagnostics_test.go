package hub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tsenjii/AhB/internal/config"
)

func TestFreebuffDiagnosticsRequiresLocalControlAndDoesNotRevealTokens(t *testing.T) {
	root:=t.TempDir()
	configFile:=filepath.Join(root,"config.json")
	if err:=os.WriteFile(configFile,[]byte("{}"),0600);err!=nil {t.Fatal(err)}
	h:=New(config.Config{
		Listen:"127.0.0.1:8317",
		Providers:[]config.ProviderConfig{{ID:"freebuff",Kind:"sidecar",Enabled:true,
			BaseURL:"http://127.0.0.1:8402",Binary:"/bin/true",HealthPath:"/healthz"}},
	})
	h.SetRestartHandler(func()error{return nil})
	h.SetControlConfigPath(configFile)
	const fakeToken="fake-test-token-just-for-check"
	if err:=saveFreebuffToken(root,fakeToken);err!=nil {t.Fatal(err)}
	call:=func(remote,token string,method string)*httptest.ResponseRecorder{
		t.Helper()
		req:=httptest.NewRequest(method,"http://127.0.0.1:8317/api/control/freebuff-diagnostics",strings.NewReader("{}"))
		req.RemoteAddr=remote
		req.Header.Set("Origin","http://127.0.0.1:8317")
		req.Header.Set("X-AhB-Control-Token",token)
		rec:=httptest.NewRecorder()
		h.Handler().ServeHTTP(rec,req)
		return rec
	}
	for _, c:=range []struct{remote,token,method string;status int}{
		{"127.0.0.1:12345","invalid","POST",403},
		{"10.0.0.1:12345",h.controlToken,"POST",403},
		{"127.0.0.1:12345",h.controlToken,"GET",405},
		{"127.0.0.1:12345",h.controlToken,"POST",200},
	}{
		r:=call(c.remote,c.token,c.method)
		if r.Code!=c.status{t.Fatalf("%s %s status=%d wanted=%d: %s",c.remote,c.method,r.Code,c.status,r.Body.String())}
		if strings.Contains(r.Body.String(),fakeToken){t.Fatalf("token leaked to UI: %s",r.Body.String())}
		if c.status==200{
			var result struct {
				Configured bool `json:"configured"`
				SavedAccounts int `json:"saved_accounts"`
				GatewayHealth string `json:"gateway_health"`
				NextStep string `json:"next_step"`
			}
			if err:=json.Unmarshal(r.Body.Bytes(),&result);err!=nil{t.Fatal(err)}
			if !result.Configured || result.SavedAccounts!=1 || result.GatewayHealth!="not_checked" || result.NextStep=="" {
				t.Fatalf("unreliable diagnosis: %+v",result)
			}
		}
	}
}
func TestZaiWebPresetRequiresExternalGateway(t *testing.T) {
	p,found:=lookupLightBridgePreset("zaiweb")
	if !found || p.Port!=8460 || p.Runtime!="Go 1.25+" || p.ModelsPath!="/v1/models" ||
		p.Docs!="https://github.com/XxxXTeam/zai2api" {
		t.Fatalf("Z.ai preset missing or changed: %+v",p)
	}
	if _,err:=lightBridgeURL("https://chat.z.ai","127.0.0.1:8317");err==nil {
		t.Fatal("Z.ai bridge must attach a locally running adapter, not the public website")
	}
}
