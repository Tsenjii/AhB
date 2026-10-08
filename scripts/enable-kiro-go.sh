#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ ! -x bin/kiro-go ]; then
  echo "bin/kiro-go is missing. Install the current Android prebuilt bundle first."
  exit 1
fi
if [ ! -f data/kiro-go/web/index.html ]; then
  echo "Kiro-Go Web UI assets are missing."
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  pkg install -y jq
fi

./scripts/prepare-configs.sh

tmp="$(mktemp)"
jq '(.providers[] | select(.id == "kiro") | .enabled) = true' config.json > "$tmp"
mv "$tmp" config.json
chmod 600 config.json

echo "Kiro-Go enabled."
echo "Restart AhB, then open: http://127.0.0.1:8408/admin"
echo "Admin password is stored locally at: $ROOT/data/kiro-go/admin-password.txt"
