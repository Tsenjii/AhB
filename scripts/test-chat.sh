#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

BASE="${AIHUB_BASE:-http://127.0.0.1:8317}"
MODEL="${AIHUB_TEST_MODEL:-}"
PROMPT="${AIHUB_TEST_PROMPT:-Reply with exactly: AIHUB_OK}"

if [ -z "$MODEL" ]; then
  echo "Set AIHUB_TEST_MODEL to a provider-prefixed model id."
  echo
  echo "Available models:"
  curl -fsS "$BASE/v1/models"
  echo
  echo
  echo "Example:"
  echo "  AIHUB_TEST_MODEL='opencode/<model-id>' ./scripts/test-chat.sh"
  exit 2
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

status="$(curl -sS -N -o "$tmp" -w '%{http_code}' \
  -H "content-type: application/json" \
  -X POST "$BASE/v1/chat/completions" \
  -d "$(printf '{"model":"%s","messages":[{"role":"user","content":"%s"}],"stream":false}' "$MODEL" "$PROMPT")" || true)"

cat "$tmp"
echo
echo "HTTP $status"

case "$status" in
  2??) exit 0 ;;
  *) exit 1 ;;
esac