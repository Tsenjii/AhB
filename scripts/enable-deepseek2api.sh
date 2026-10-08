#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ ! -x bin/deepseek2api ]; then
  echo "bin/deepseek2api is missing."
  echo "Install the current Android prebuilt bundle first."
  exit 1
fi
if [ ! -d data/deepseek2api/static/admin ]; then
  echo "data/deepseek2api/static/admin is missing."
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  pkg install -y jq
fi

./scripts/prepare-configs.sh

tmp="$(mktemp)"
jq '(.providers[] | select(.id == "deepseek") | .enabled) = true' config.json > "$tmp"
mv "$tmp" config.json
chmod 600 config.json

echo "DeepSeek2API enabled."
echo "Restart AhB, then open http://127.0.0.1:8405/admin"
echo "Admin key is stored locally at: $ROOT/data/deepseek2api/admin-key.txt"
