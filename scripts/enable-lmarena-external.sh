#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v jq >/dev/null 2>&1; then
  pkg install -y jq
fi
./scripts/prepare-configs.sh

base_url="$(jq -r '.providers[] | select(.id == "lmarena") | .base_url' config.json)"
case "$base_url" in
  http://127.0.0.1:*|http://localhost:*|http://[::1]:*) ;;
  *)
    echo "Refusing to enable: LMArena external bridge must use a loopback URL."
    exit 1
    ;;
esac

tmp="$(mktemp)"
jq '(.providers[] | select(.id == "lmarena") | .enabled) = true' config.json > "$tmp"
mv "$tmp" config.json
chmod 600 config.json

echo "LMArena external slot enabled."
echo "AhB expects an OpenAI-compatible bridge at: $base_url"
echo "The bridge itself is intentionally not bundled or started by AhB."
echo "Restart AhB after the local bridge is running."
