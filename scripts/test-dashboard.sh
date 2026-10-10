#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

python3 - "$TMP/dashboard.js" <<'PY'
from pathlib import Path
import sys
src=Path("internal/hub/ui.go").read_text(encoding="utf-8")
assert src.count("<script>")==1 and src.count("</script>")==1
html=src[src.index("<!doctype html>"):src.index("</html>")+len("</html>")]
for needle in ('id="bridgePreset"', 'id="bridgeURL"', 'id="bridgeCommand"',
               'id="copyBridge"', 'id="customBridgeId"', 'id="providers"',
               'id="models"', 'id="refresh"', 'id="optional"', 'id="diagnostics"', 'id="freebuffLogin"', 'id="copyCopilotInstall"',
               'id="resourceSettings"', 'id="systemRamValue"', 'id="availableRamValue"', 'id="ahbRamValue"',
               'id="maxResident"', 'id="idleTimeout"', 'id="saveResourceSettings"', 'id="scanAllModels"',
               'id="adminConsoles"', 'id="adminConsoleRows"', 'id="reloadAdminConsoles"'):
    assert needle in html, f"missing UI element: {needle}"
assert "Node.js" in html and "FreeBuff" in html and "freebuff-login-termux.sh" in html
assert "HTTP 418 · Duck.ai 拒絕請求" in html
assert "實驗版 DeepSeek Web 已移除舊 /admin 管理台" in html
assert 'data-copilot-auth' in html
assert 'Copilot2API 沒有獨立 WebUI' in html
assert "copilotLoginAction('status').then(showCopilotLogin)" in html
assert 'data-resource-preset="stable"' in html and 'data-resource-preset="lean"' in html and 'data-resource-preset="performance"' in html and 'data-resource-preset="all"' in html
assert 'resourceSettingsDirty=true' in html and 'const values={stable:[2,600],lean:[1,120],performance:[3,900],all:[9,86400]}' in html
assert "'/api/control/resources'" in html and "'/api/runtime'" in html
assert "x_cached" in html and "休眠快取" in html
assert "'/api/control/wake/'" in html and "scanAllModels" in html
assert "'/api/control/console-access'" in html and "nativeConsoleRequest" in html
# Mobile OAuth: never open a blank tab before obtaining the official link.
assert "window.open('about:blank'" not in html
assert "visit.dataset.waking='yes'" in html
assert "已啟動，點此開啟" in html
assert "freebuffLink.href=url" in html
assert "freebuffLink.removeAttribute('href')" in html
assert "signal:controller.signal" in html
assert "開啟 FreeBuff 官方授權頁" in html
assert "visit.addEventListener('click',async event=>" in html and "started=await fetch('/api/control/wake/'" in html
# Responsive desktop/mobile UI and lightweight browser-side collection paging.
for needle in ('id="connectionStatus"', 'id="lastSynced"', 'id="summaryRss"',
               'id="summaryReady"', 'id="showMoreModels"', 'data-provider-filter="problems"',
               'data-provider-filter="asleep"', 'data-provider-filter="enabled"',
               'min-width:1024px', 'grid-template-columns:repeat(5,minmax(0,1fr))'):
    assert needle in html, f"desktop/mobile control center missing: {needle}"
assert 'modelLimit=120' in html and 'rows.slice(0,modelLimit)' in html, 'model page size must be bounded'
assert 'previousCards' in html and 'applyProviderFilter()' in html, 'proxy state/filter must survive refresh'
assert 'providerFilterEmpty' in html and "connection-pill offline" in html, 'visible status required'
assert html.count('<script>') == 1, 'must not introduce a second browser bundle'
js=html.split("<script>",1)[1].split("</script>",1)[0]
Path(sys.argv[1]).write_text(js,encoding="utf-8")
PY

node --check "$TMP/dashboard.js"

# Workbench is independently rendered but uses the same private Hub auth.
# Confirm its JavaScript is parseable and key controls do not silently vanish.
python3 - "$TMP/playground.js" <<'PY'
from pathlib import Path
import sys
src=Path("internal/hub/playground_ui.go").read_text(encoding="utf-8")
html=src[src.index("<!doctype html>"):src.index("</html>")+len("</html>")]
for needle in ('id="provider"', 'id="model"', 'id="prompt"', 'id="stream"',
               'id="toolTest"', 'id="send"', 'id="cancel"', 'id="status"',
               "X-AhB-Control-Token", "'/api/control/playground'", "[DONE]",
               "data:","418","503"):
    assert needle in html, f"Playground missing: {needle}"
assert 'window.open(' not in html, "Playground must not open empty OAuth tabs"
assert "innerHTML=" not in html, "Playground should render model text without HTML injection"
# Playground desktop layout and browser-only conveniences.
for needle in ('workbench-heading', 'results-card', 'result-actions', 'id="copyResponse"',
               'id="clearResponse"', 'id="providerInfo"', 'data-preset="reason"',
               'Ctrl', 'min-width:1600px'):
    if needle == 'Ctrl':
        assert 'e.ctrlKey||e.metaKey' in html, 'keyboard shortcut missing'
    else:
        assert needle in html, f"Playground UI regression: {needle}"
assert 'navigator.clipboard.writeText' in html, 'reply-copy action must be wired'
assert 'id="modelSelect"' in html and 'id="loadModels"' in html, 'Playground must allow selecting and loading models'
js=html.split("<script>",1)[1].split("</script>",1)[0]
Path(sys.argv[1]).write_text(js,encoding="utf-8")
PY
node --check "$TMP/playground.js"
echo "dashboard + Playground markup and JavaScript syntax passed"
