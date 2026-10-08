#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
if [ ! -x bin/copilot2api ]; then
  echo "Copilot2API is not bundled. Install the latest AhB Android prebuilt first." >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  pkg install -y jq
fi
./scripts/prepare-configs.sh
if [ ! -s data/copilot2api/credentials.json ]; then
  echo "Copilot account login is required first:" >&2
  echo "  cd $ROOT && ./scripts/login-copilot2api.sh" >&2
  echo "After user-initiated device authorization, press Ctrl+C and retry." >&2
  exit 1
fi
tmp="$(mktemp "$ROOT/.copilot-enable.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
jq -e 'if any(.providers[]; .id == "copilot" and .kind == "sidecar")
        then (.providers[] | select(.id == "copilot") | .enabled) = true
        else error("copilot provider not in config.example.json: update AhB first") end' config.json > "$tmp"
cp -p config.json data/copilot2api/config-before-enable.json
mv "$tmp" config.json
chmod 600 config.json
echo "GitHub Copilot is enabled as a supervised optional sidecar."
echo "Restart: cd $ROOT && ./scripts/stop-termux.sh && ./scripts/start-termux.sh"
echo "Then inspect /api/providers and /v1/models, and test an authorized model."
echo "NOTE: source credentials are not proof of remaining Copilot quota."
