#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
python3 - <<'PY'
from pathlib import Path
import re
expected="cd97bce9912225e055cc79761d96a3f0b77e26f8"
files=[
    Path(".github/workflows/build-android-arm64.yml"),
    Path(".github/workflows/build-linux.yml"),
    Path("scripts/install-agent2api-termux.sh"),
]
for path in files:
    s=path.read_text()
    key="AGENT_SHA" if "workflows" in str(path) else "EXPECTED_SHA"
    values=re.findall(rf'{key}="([a-f0-9]{{40}})"',s)
    assert values==[expected],f"{path}: expected one reviewed Agent2API v2.9.9 SHA"
    assert "2.9.9" in s,f"{path}: stale Agent2API version marker"
    print(path,"Agent2API SHA pinned consistently")
bridge=Path("scripts/connect-bridge.sh").read_text()
assert 'cliproxy) title="CLIProxyAPI"' in bridge
assert 'http://127.0.0.1:8416' in bridge
assert "curl_args=(--fail --silent --show-error --max-time 8)" in bridge
ui=Path("internal/hub/ui.go").read_text()
assert '<option value="cliproxy">' in ui
assert 'cliproxy:8416' in ui
print("Optional CLIProxyAPI preset verified (not bundled)")
PY
