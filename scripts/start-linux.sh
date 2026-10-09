#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT"
mkdir -p logs data
umask 077
PIDFILE="$ROOT/data/hubd.pid"
if [ -s "$PIDFILE" ]; then
  pid="$(cat "$PIDFILE")"
  if [[ "$pid" =~ ^[0-9]+$ ]] && [ "$(readlink "/proc/$pid/exe" 2>/dev/null || :)" = "$ROOT/bin/hubd" ]; then
    echo "AhB Linux already running: PID $pid"
    exit 0
  fi
fi
nohup bash "$ROOT/scripts/run-linux.sh" >> "$ROOT/logs/hubd.log" 2>&1 < /dev/null &
pid=$!
printf '%s\n' "$pid" > "$PIDFILE"
sleep 2
if [ "$(readlink "/proc/$pid/exe" 2>/dev/null || :)" != "$ROOT/bin/hubd" ]; then
  echo "AhB Linux failed to start. Recent log:" >&2
  tail -n 25 "$ROOT/logs/hubd.log" >&2 || true
  rm -f "$PIDFILE"
  exit 1
fi
echo "AhB Linux started PID $pid; UI on localhost:8317 (use SSH tunnel remotely)."
