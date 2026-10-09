#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
if [ ! -x bin/copilot2api ]; then
  echo "Copilot ARM64 binary not installed. Upgrade to an AhB package that includes it first." >&2
  exit 1
fi
mkdir -p data/copilot2api
chmod 700 data/copilot2api

if curl -fsS --max-time 2 http://127.0.0.1:8410/v1/models >/dev/null 2>&1; then
  echo "Copilot sidecar appears to be running. Stop AhB before interactive authentication." >&2
  exit 1
fi

echo "GitHub Copilot: authorize your own eligible account using the on-screen GitHub Device Flow."
echo "The login token remains in AhB/data/copilot2api/ and is not committed or sent to the Hub UI."
echo "When authentication succeeds and the local HTTP server starts, press Ctrl+C."
echo
# Native Android libc DNS avoids the unusable [::1]:53 fallback seen in
# pure-Go Android binaries. A cgo-enabled prebuilt is also required.
GODEBUG=netdns=cgo exec "$ROOT/bin/copilot2api" -host 127.0.0.1 -port 8411 -token-dir "$ROOT/data/copilot2api"
