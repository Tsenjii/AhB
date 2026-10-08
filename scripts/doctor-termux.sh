#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail=0

check_file() {
  local path="$1"
  if [ -e "$path" ]; then
    echo "OK   $path"
  else
    echo "MISS $path"
    fail=1
  fi
}

provider_enabled() {
  local id="$1"
  command -v jq >/dev/null 2>&1 &&
    [ -f config.json ] &&
    jq -e --arg id "$id" '.providers[] | select(.id == $id and .enabled == true)' config.json >/dev/null 2>&1
}

check_url() {
  local name="$1"
  local url="$2"
  local code
  code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 3 "$url" 2>/dev/null || true)"
  case "$code" in
    2??|3??) echo "UP       $name  HTTP $code  $url" ;;
    4??|5??) echo "DEGRADED $name  HTTP $code  $url" ;;
    *)       echo "DOWN     $name  $url" ;;
  esac
}

echo "== system =="
uname -a
echo "arch: $(uname -m)"
echo

echo "== required files =="
check_file "config.json"
check_file "data/opencode/config.json"
check_file "data/opencode/webui-password.txt"
check_file "data/freebuff/config.json"
check_file "bin/hubd"
check_file "bin/opencode2api"
check_file "bin/freebuff2api"
if [ -x bin/agent2api-server ]; then
  check_file "data/agent2api/ui"
fi
if provider_enabled deepseek; then
  check_file "bin/deepseek2api"
  check_file "data/deepseek2api/config.json"
  check_file "data/deepseek2api/static/admin"
  check_file "data/deepseek2api/admin-key.txt"
fi
if provider_enabled grok; then
  check_file "bin/grok2api"
  check_file "data/grok2api/config.yaml"
  check_file "data/grok2api/frontend/dist/index.html"
  check_file "data/grok2api/client-key.txt"
fi
if provider_enabled kiro; then
  check_file "bin/kiro-go"
  check_file "data/kiro-go/config.json"
  check_file "data/kiro-go/web/index.html"
  check_file "data/kiro-go/admin-password.txt"
fi
echo

echo "== local endpoints =="
check_url "hub-api" "http://127.0.0.1:8317/healthz"
check_url "hub-ui" "http://127.0.0.1:8317/ui"
check_url "opencode-api" "http://127.0.0.1:8401/healthz"
check_url "opencode-ui" "http://127.0.0.1:8404/"
check_url "freebuff-api" "http://127.0.0.1:8402/healthz"
check_url "freebuff-ui" "http://127.0.0.1:8402/ui"
if [ -x bin/agent2api-server ]; then
  check_url "agent2api" "http://127.0.0.1:8403/health"
  check_url "agent2api-ui" "http://127.0.0.1:8403/"
fi
if provider_enabled deepseek; then
  check_url "deepseek-api" "http://127.0.0.1:8405/healthz"
  check_url "deepseek-ready" "http://127.0.0.1:8405/readyz"
  check_url "deepseek-ui" "http://127.0.0.1:8405/admin"
fi
# Read the actual configured bridge URLs. Hard-coded LMArena ports are stale,
# and authenticated external bridges must be checked through hubd instead of
# incorrectly treating an unauthenticated direct 401 as a failed service.
if command -v jq >/dev/null 2>&1 && [ -f config.json ]; then
  while IFS=
if provider_enabled grok; then
  check_url "grok-health" "http://127.0.0.1:8407/healthz"
  check_url "grok-ready" "http://127.0.0.1:8407/readyz"
  check_url "grok-ui" "http://127.0.0.1:8407/"
fi
if provider_enabled kiro; then
  check_url "kiro-health" "http://127.0.0.1:8408/health"
  check_url "kiro-ui" "http://127.0.0.1:8408/admin"
fi
echo

echo "== hub runtime =="
curl -fsS --max-time 3 http://127.0.0.1:8317/api/runtime 2>/dev/null || true
echo
echo "== providers =="
curl -fsS --max-time 3 http://127.0.0.1:8317/api/providers 2>/dev/null || true
echo

exit "$fail"\t' read -r ext_id ext_base ext_health ext_auth; do
    [ -n "$ext_id" ] || continue
    if [ "$ext_auth" = "true" ]; then
      echo "HUB CHECK external-$ext_id (bridge has private auth; see /api/providers)"
    else
      check_url "external-$ext_id" "${ext_base%/}$ext_health"
    fi
  done < <(jq -r '
    .providers[] | select(.kind == "external" and .enabled == true) |
    [.id, .base_url, (.health_path // .models_path // "/v1/models"),
     ((.headers.Authorization // "") != "")] | @tsv' config.json)
fi
if provider_enabled grok; then
  check_url "grok-health" "http://127.0.0.1:8407/healthz"
  check_url "grok-ready" "http://127.0.0.1:8407/readyz"
  check_url "grok-ui" "http://127.0.0.1:8407/"
fi
if provider_enabled kiro; then
  check_url "kiro-health" "http://127.0.0.1:8408/health"
  check_url "kiro-ui" "http://127.0.0.1:8408/admin"
fi
echo

echo "== hub runtime =="
curl -fsS --max-time 3 http://127.0.0.1:8317/api/runtime 2>/dev/null || true
echo
echo "== providers =="
curl -fsS --max-time 3 http://127.0.0.1:8317/api/providers 2>/dev/null || true
echo

exit "$fail"