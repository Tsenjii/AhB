#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Private UI restart worker. Invoked only by the already running Android Hub.
# Never kill all Termux processes: terminate only the verified AhB Hub PID,
# let the Go supervisor shut down its own sidecars, then start AhB again.
if [ "${1:-}" = "--restart-from-pid" ]; then
  old_pid="${2:-}"
  if ! [[ "$old_pid" =~ ^[1-9][0-9]*$ ]]; then
    echo "Invalid AhB restart PID" >&2
    exit 1
  fi
  sleep 2  # allow the browser's 202 response to finish before shutdown
  old_exe="$(readlink "/proc/$old_pid/exe" 2>/dev/null || true)"
  if [ "$old_exe" != "$ROOT/bin/hubd" ]; then
    echo "Refusing restart: PID no longer belongs to this AhB" >&2
    exit 1
  fi
  kill -TERM "$old_pid"
  stopped=0
  for _ in $(seq 1 80); do
    if ! kill -0 "$old_pid" 2>/dev/null; then
      stopped=1
      break
    fi
    sleep .25
  done
  if [ "$stopped" != 1 ]; then
    echo "AhB did not stop cleanly; refusing a duplicate Hub start" >&2
    exit 1
  fi
  echo "Old AhB stopped; starting AhB with existing config and accounts"
elif [ "$#" -gt 0 ]; then
  echo "Unknown argument" >&2
  exit 2
fi

mkdir -p data logs

if [ -f data/hubd.pid ]; then
  old_pid="$(cat data/hubd.pid 2>/dev/null || true)"
  if [ -n "$old_pid" ] && kill -0 "$old_pid" 2>/dev/null; then
    exe="$(readlink "/proc/$old_pid/exe" 2>/dev/null || true)"
    if [ "$exe" = "$ROOT/bin/hubd" ]; then
      echo "hubd already running as PID $old_pid"
      exit 0
    fi
  fi
fi

# A foreground run, killed shell, or stale pid file can leave sidecars alive.
# Clean only AhB's own binaries before launching a fresh supervisor.
./scripts/stop-termux.sh >/dev/null 2>&1 || true

if command -v termux-wake-lock >/dev/null 2>&1; then
  termux-wake-lock || true
fi

nohup ./scripts/run-termux.sh >> logs/hubd.log 2>&1 &
pid=$!
echo "$pid" > data/hubd.pid

sleep 1
if ! kill -0 "$pid" 2>/dev/null; then
  echo "hubd failed to stay running; recent log:"
  tail -n 40 logs/hubd.log 2>/dev/null || true
  rm -f data/hubd.pid
  exit 1
fi

echo "hubd started as PID $pid"
echo "log: $ROOT/logs/hubd.log"
echo "check: curl http://127.0.0.1:8317/healthz"
