#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
cleanup() {
  for pid in "${node_pid:-}" "${kimi_pid:-}" "${other_node_pid:-}" "${other_py_pid:-}"; do
    [ -n "$pid" ] && kill -TERM "$pid" 2>/dev/null || true
  done
  for pid in "${node_pid:-}" "${kimi_pid:-}" "${other_node_pid:-}" "${other_py_pid:-}"; do
    [ -n "$pid" ] && wait "$pid" 2>/dev/null || true
  done
  rm -rf "$TMP"
}
trap cleanup EXIT
command -v node >/dev/null || { echo "node required for fixture" >&2; exit 1; }
command -v python3 >/dev/null || { echo "python required for fixture" >&2; exit 1; }

APP="$TMP/AhB"
mkdir -p "$APP/scripts" "$APP/data/freebuff/gateway" "$APP/data/kimiweb/source" "$TMP/unrelated"
cp "$ROOT/scripts/stop-termux.sh" "$APP/scripts/stop-termux.sh"
printf '%s\n' 'setInterval(() => {}, 1000);' > "$APP/data/freebuff/gateway/server.js"
printf '%s\n' 'import time; time.sleep(30)' > "$APP/data/kimiweb/source/run.py"
printf '%s\n' 'setInterval(() => {}, 1000);' > "$TMP/unrelated/server.js"

node "$APP/data/freebuff/gateway/server.js" >/dev/null 2>&1 &
node_pid=$!
( cd "$APP/data/kimiweb/source"; exec python3 run.py ) >/dev/null 2>&1 &
kimi_pid=$!
node "$TMP/unrelated/server.js" >/dev/null 2>&1 &
other_node_pid=$!
python3 -c 'import time;time.sleep(30)' >/dev/null 2>&1 &
other_py_pid=$!
sleep 1
for pid in "$node_pid" "$kimi_pid" "$other_node_pid" "$other_py_pid"; do
  kill -0 "$pid" || { echo "fixture failed to start process $pid" >&2; exit 1; }
done

bash "$APP/scripts/stop-termux.sh"
wait "$node_pid" 2>/dev/null || true
wait "$kimi_pid" 2>/dev/null || true
if kill -0 "$node_pid" 2>/dev/null || kill -0 "$kimi_pid" 2>/dev/null; then
  echo "AhB-owned Node/Python processes were not stopped" >&2
  exit 1
fi
if ! kill -0 "$other_node_pid" 2>/dev/null || ! kill -0 "$other_py_pid" 2>/dev/null; then
  echo "stop helper interrupted unrelated Node/Python processes" >&2
  exit 1
fi
echo "stop-termux fixture: AhB processes stopped; unrelated runtimes survived"
