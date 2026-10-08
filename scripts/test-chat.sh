#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077

# Explicit model inference smoke test, not merely a process/HTTP health probe.
# Intentionally does not retry: 429, 503 and account quota problems must remain visible.
BASE="${AIHUB_BASE:-http://127.0.0.1:8317}"
MODEL="${AIHUB_TEST_MODEL:-}"
PROMPT="${AIHUB_TEST_PROMPT:-Reply with exactly: AIHUB_OK}"

for tool in jq curl; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Missing command: $tool (Termux: pkg install jq curl)." >&2
    exit 2
  fi
done
case "$BASE" in
  http://127.0.0.1:*|http://localhost:*|http://\[::1\]:*) ;;
  *) echo "Refusing a non-loopback AhB diagnostic endpoint." >&2; exit 2 ;;
esac
if [ -z "$MODEL" ]; then
  echo "Set AIHUB_TEST_MODEL to a provider-prefixed model id."
  echo "Available models:"
  curl --fail --silent --show-error --max-time 15 "$BASE/v1/models" |
    jq -r '.data[]?.id // empty'
  echo
  echo "Example:"
  echo "  AIHUB_TEST_MODEL='opencode/<model-id>' ./scripts/test-chat.sh"
  exit 2
fi
case "$MODEL" in
  */?*) ;;
  *) echo "AIHUB_TEST_MODEL must be provider/model." >&2; exit 2 ;;
esac

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
payload="$(jq -nc --arg model "$MODEL" --arg prompt "$PROMPT" '{
  model:$model,
  stream:false,
  messages:[{role:"user",content:$prompt}]
}')"

status="$(curl --silent --show-error --max-time 120 -o "$tmp" \
  -w '%{http_code}' \
  -H 'Content-Type: application/json' \
  -d "$payload" "$BASE/v1/chat/completions" || true)"
if [ "$status" != "200" ]; then
  echo "FAIL: $MODEL HTTP $status (check quota/account/provider log)." >&2
  jq -r '(.error.code // .error.message // .detail // "no structured error") |
    tostring | .[0:200]' "$tmp" 2>/dev/null || true
  exit 1
fi
if ! jq -e '
  (.choices[0].message.content | type == "string" and length > 0)
  and (.choices[0].finish_reason != "tool_calls")
' "$tmp" >/dev/null 2>&1; then
  echo "FAIL: HTTP 200 without an actual assistant text answer." >&2
  exit 1
fi
echo "PASS: HTTP 200 with non-empty assistant answer from $MODEL"
jq -r '.choices[0].message.content' "$tmp"
