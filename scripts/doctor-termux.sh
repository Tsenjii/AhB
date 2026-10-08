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

check_url() {
  local name="$1"
  local url="$2"
  if curl -fsS --max-time 3 "$url" >/dev/null 2>&1; then
    echo "UP   $name  $url"
  else
    echo "DOWN $name  $url"
  fi
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
echo

echo "== hub runtime =="
curl -fsS --max-time 3 http://127.0.0.1:8317/api/runtime 2>/dev/null || true
echo
echo "== providers =="
curl -fsS --max-time 3 http://127.0.0.1:8317/api/providers 2>/dev/null || true
echo

exit "$fail"