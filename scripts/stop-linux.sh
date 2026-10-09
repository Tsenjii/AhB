#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT"
PIDFILE="$ROOT/data/hubd.pid"
[ -f "$PIDFILE" ] || { echo "No Linux hub PID file; nothing stopped"; exit 0; }
pid="$(cat "$PIDFILE")"
if ! [[ "$pid" =~ ^[0-9]+$ ]] || [ "$(readlink "/proc/$pid/exe" 2>/dev/null || :)" != "$ROOT/bin/hubd" ]; then
  echo "Stale PID file; refusing to stop any unrelated process" >&2
  exit 1
fi
kill -TERM "$pid"
for i in $(seq 1 60); do
  if ! kill -0 "$pid" 2>/dev/null || [ "$(readlink "/proc/$pid/exe" 2>/dev/null || :)" != "$ROOT/bin/hubd" ]; then
    rm -f "$PIDFILE"
    echo "AhB Linux gracefully stopped"
    exit 0
  fi
  sleep .5
done
echo "Hub did not stop cleanly; leaving process and account data intact" >&2
exit 1
