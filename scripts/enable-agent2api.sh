#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ ! -x bin/agent2api-server ]; then
  echo "bin/agent2api-server is missing."
  echo "Install/build Agent2API first, or use the prebuilt Android bundle."
  exit 1
fi
if [ ! -d data/agent2api/ui ]; then
  echo "data/agent2api/ui is missing."
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  pkg install -y jq
fi
if [ ! -f config.json ]; then
  cp config.example.json config.json
fi

tmp="$(mktemp)"
jq '(.providers[] | select(.id == "agent2api") | .enabled) = true' config.json > "$tmp"
mv "$tmp" config.json
chmod 600 config.json

echo "Agent2API enabled."
echo "Restart AhB, then open http://127.0.0.1:8403/"
