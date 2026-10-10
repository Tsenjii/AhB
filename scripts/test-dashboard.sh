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
assert "DeepSeek Web 需要先在原生 /admin 設定自己的網頁帳號" in html
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
js=html.split("<script>",1)[1].split("</script>",1)[0]
Path(sys.argv[1]).write_text(js,encoding="utf-8")
PY
node --check "$TMP/playground.js"
echo "dashboard + Playground markup and JavaScript syntax passed"
