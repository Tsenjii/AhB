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
               'id="models"', 'id="refresh"', 'id="optional"', 'id="diagnostics"', 'id="freebuffLogin"', 'id="copyCopilotInstall"'):
    assert needle in html, f"missing UI element: {needle}"
assert "Node.js" in html and "FreeBuff" in html and "freebuff-login-termux.sh" in html
js=html.split("<script>",1)[1].split("</script>",1)[0]
Path(sys.argv[1]).write_text(js,encoding="utf-8")
PY

node --check "$TMP/dashboard.js"
echo "dashboard markup and JavaScript syntax passed"
