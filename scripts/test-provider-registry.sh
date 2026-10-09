#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
python3 - <<'PY'
from pathlib import Path
import json
src=json.loads(Path("docs/TO_API_REGISTRY.json").read_text())
cfg=json.loads(Path("config.example.json").read_text())
assert src["schema_version"] == 1
assert len(src["candidates"]) >= 30
valid=set(src["status_definitions"])
ids=set()
urls=set()
for c in src["candidates"]:
    assert c["id"] and c["id"] not in ids, f"duplicate registry id {c['id']}"
    ids.add(c["id"])
    assert c["status"] in valid, c
    assert c["priority"] in ("P0","P1","P2","P3","HOLD","REJECT"), c
    assert c["url"].startswith("https://github.com/"),c
    assert c["last_reviewed"] and c["notes"]
for p in cfg["providers"]:
    assert p["id"] in ids, f"config provider missing in source registry: {p['id']}"
# Android installs expose ONLY adapters actually compiled into the
# Android bundle, with all seven enabled but resident only on demand.
installed={"opencode","freebuff","agent2api","deepseek","grok","kiro","copilot"}
assert {p["id"] for p in cfg["providers"]} == installed
for p in cfg["providers"]:
    assert p["kind"]=="sidecar" and p["enabled"] is True
    assert p["start_mode"]=="on_demand"
assert cfg["resources"]["max_running_sidecars"]==3
assert cfg["resources"]["idle_stop_seconds"]==900
linux=json.loads(Path("configs/profiles/linux-512mb.json").read_text())
assert {p["id"] for p in linux["providers"]}==installed
assert linux["resources"]["max_running_sidecars"]==1
assert linux["resources"]["idle_stop_seconds"]==120
assert Path("scripts/login-copilot2api.sh").is_file()
assert Path("scripts/enable-copilot2api.sh").is_file()
assert Path("scripts/install-kimiweb-termux.sh").is_file()
assert Path("scripts/enable-kimiweb-termux.sh").is_file()
workflow=Path(".github/workflows/build-android-arm64.yml").read_text()
for name in ("copilot2api","enable-copilot2api.sh","login-copilot2api.sh","install-kimiweb-termux.sh","enable-kimiweb-termux.sh"):
    assert name in workflow, name
print("registry and optional source wiring verified",len(ids),"candidates")
PY
