#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ ! -x bin/deepseek2api ]; then
  echo "Missing bin/deepseek2api; install the latest Android prebuilt first." >&2
  exit 1
fi
# This replacement has no WebUI or admin key. Never convert previous
# passwords, account JSON, cookies, or other upstream secrets automatically.
ACCOUNT_FILE="$ROOT/data/deepseek2api/accounts.txt"
if [ ! -f "$ACCOUNT_FILE" ] || ! grep -Eq '^[[:space:]]*[^#[:space:]]' "$ACCOUNT_FILE"; then
  echo "No authorized DeepSeek Web account configured." >&2
  echo "Add your own authorized account token to private file:" >&2
  echo "  $ACCOUNT_FILE" >&2
  echo "One token per line; chmod 600; never paste credentials in chat." >&2
  exit 1
fi
chmod 600 "$ACCOUNT_FILE"
./scripts/prepare-configs.sh
tmp="$(mktemp)"
jq '(.providers[] | select(.id == "deepseek") | .enabled) = true' config.json > "$tmp"
chmod 600 "$tmp"
mv "$tmp" config.json
echo "Experimental DeepSeek Web adapter enabled. Restart AhB and test deepseek/deepseek-chat in /playground."
echo "The old /admin dashboard and admin-key are NOT part of this replacement."
