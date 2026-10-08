#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p data logs

if [ -f data/hubd.pid ]; then
  old_pid="$(cat data/hubd.pid 2>/dev/null || true)"
  if [ -n "$old_pid" ] && kill -0 "$old_pid" 2>/dev/null; then
    echo "hubd already running as PID $old_pid"
    exit 0
  fi
  rm -f data/hubd.pid
fi

if command -v termux-wake-lock >/dev/null 2>&1; then
  termux-wake-lock || true
fi

nohup ./scripts/run-termux.sh >> logs/hubd.log 2>&1 &
pid=$!
echo "$pid" > data/hubd.pid

echo "hubd started as PID $pid"
echo "log: $ROOT/logs/hubd.log"
echo "check: curl http://127.0.0.1:8317/healthz"