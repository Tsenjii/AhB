#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

chmod +x scripts/prepare-configs.sh
./scripts/prepare-configs.sh

if [ ! -x ./bin/hubd ]; then
  echo "bin/hubd not found. Run ./scripts/bootstrap-termux.sh first."
  exit 1
fi

exec ./bin/hubd -config ./config.json