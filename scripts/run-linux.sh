#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p logs data/opencode data/freebuff

if [ ! -x ./bin/hubd ]; then
  echo "bin/hubd not found"
  exit 1
fi

if [ ! -f config.json ]; then
  cp config.example.json config.json
fi
if [ ! -f data/opencode/config.json ]; then
  cp configs/opencode2api.json data/opencode/config.json
fi
if [ ! -f data/freebuff/config.json ]; then
  cp configs/freebuff2api.json data/freebuff/config.json
fi

exec ./bin/hubd -config ./config.json