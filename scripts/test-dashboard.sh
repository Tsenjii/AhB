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
assert "'/api/control/resources'" in html and "'/api/runtime'" in html
assert "x_cached" in html and "休眠快取" in html
assert "'/api/control/wake/'" in html and "scanAllModels" in html
assert "'/api/control/console-access'" in html and "nativeConsoleRequest" in html
# Mobile OAuth: never open a blank tab before obtaining the official link.
assert "window.open('about:blank'" not in html
assert "freebuffLink.href=url" in html
assert "freebuffLink.removeAttribute('href')" in html
assert "signal:controller.signal" in html
assert "開啟 FreeBuff 官方授權頁" in html
assert "visit.addEventListener('click',async event=>" in html and "started=await fetch('/api/control/wake/'" in html
js=html.split("<script>",1)[1].split("</script>",1)[0]
Path(sys.argv[1]).write_text(js,encoding="utf-8")
PY

node --check "$TMP/dashboard.js"
echo "dashboard markup and JavaScript syntax passed"
