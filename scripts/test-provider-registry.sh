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
# Android bundles still include all nine adapters. Experimental sources
# that have not passed live account/upstream acceptance must start disabled,
# while preserving any existing installed user's private config.json.
installed={"opencode","freebuff","agent2api","deepseek","grok","kiro","copilot","geminiweb","duckai"}
assert {p["id"] for p in cfg["providers"]} == installed
unverified_default_off={"deepseek","duckai"}
for p in cfg["providers"]:
    assert p["kind"]=="sidecar"
    assert p["enabled"] is (p["id"] not in unverified_default_off)
    assert p["start_mode"]=="on_demand"
assert cfg["resources"]["max_running_sidecars"]==3
assert cfg["resources"]["idle_stop_seconds"]==900
linux=json.loads(Path("configs/profiles/linux-512mb.json").read_text())
assert {p["id"] for p in linux["providers"]}==installed
assert linux["resources"]["max_running_sidecars"]==1
assert linux["resources"]["idle_stop_seconds"]==120
for p in linux["providers"]:
    assert p["enabled"] is (p["id"] not in unverified_default_off)
assert Path("scripts/login-copilot2api.sh").is_file()
assert Path("scripts/enable-copilot2api.sh").is_file()
assert Path("scripts/install-kimiweb-termux.sh").is_file()
assert Path("scripts/enable-kimiweb-termux.sh").is_file()
workflow=Path(".github/workflows/build-android-arm64.yml").read_text()
for name in ("copilot2api","enable-copilot2api.sh","login-copilot2api.sh","install-kimiweb-termux.sh","enable-kimiweb-termux.sh"):
    assert name in workflow, name
for path in (".github/workflows/build-android-arm64.yml",".github/workflows/build-linux.yml"):
    build=Path(path).read_text()
    for binary in ("gemini-web2api-go", "duck2api"):
        assert binary in build, (path,binary)
assert Path("configs/geminiweb.json").is_file()
for config in (cfg,linux):
    gemini=next(x for x in config["providers"] if x["id"]=="geminiweb")
    duck=next(x for x in config["providers"] if x["id"]=="duckai")
    assert gemini["base_url"]=="http://127.0.0.1:8413"
    assert gemini["env"]["API_KEY"]=="__AIHUB_SERVER_KEY__"
    assert duck["base_url"]=="http://127.0.0.1:8414"
    assert duck["env"]["DUCKAI_BIND"]=="127.0.0.1"
    assert duck["env"]["PORT"]=="8414"
    assert duck["env"]["DUCKAI_DEFAULT_API_KEY"]=="__AIHUB_SERVER_KEY__"
    assert duck["health_path"]=="/health"
    deepseek=next(x for x in config["providers"] if x["id"]=="deepseek")
    assert deepseek["env"]["PROXY_API_KEY"]=="__AIHUB_SERVER_KEY__"
    assert deepseek["env"]["DEEPSEEK_ACCOUNTS_FILE"]=="accounts.txt"
    assert deepseek["ui_url"]==""
    assert deepseek["health_path"]=="/health"
print("registry and native Android/Linux provider wiring verified",len(ids),"candidates")
PY
