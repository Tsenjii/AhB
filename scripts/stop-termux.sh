#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ ! -f data/hubd.pid ]; then
  echo "hubd pid file not found"
  exit 0
fi

pid="$(cat data/hubd.pid 2>/dev/null || true)"
if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
  kill "$pid"
  for _ in 1 2 3 4 5; do
    if ! kill -0 "$pid" 2>/dev/null; then
      break
    fi
    sleep 1
  done
  if kill -0 "$pid" 2>/dev/null; then
    kill -9 "$pid" 2>/dev/null || true
  fi
fi
rm -f data/hubd.pid

if command -v termux-wake-unlock >/dev/null 2>&1; then
  termux-wake-unlock || true
fi

echo "hubd stopped"