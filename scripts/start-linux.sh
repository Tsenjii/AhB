#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT"
# Local dashboard restart worker. Never accept arbitrary target commands.
# Gracefully stop only a PID whose executable is exactly this AhB hub.
if [ "${1:-}" = "--restart-from-pid" ]; then
  pid="${2:-}"
  if ! [[ "$pid" =~ ^[1-9][0-9]*$ ]] ||
     [ "$(readlink "/proc/$pid/exe" 2>/dev/null || :)" != "$ROOT/bin/hubd" ]; then
    echo "Refusing restart: PID does not belong to this Linux AhB" >&2
    exit 1
  fi
  sleep 2
  kill -TERM "$pid"
  stopped=0
  for i in $(seq 1 80); do
    if [ "$(readlink "/proc/$pid/exe" 2>/dev/null || :)" != "$ROOT/bin/hubd" ]; then
      stopped=1
      break
    fi
    sleep .25
  done
  [ "$stopped" = 1 ] || { echo "Hub did not shut down cleanly" >&2; exit 1; }
elif [ "$#" -gt 0 ]; then
  echo "Unknown Linux start argument" >&2
  exit 2
fi
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
