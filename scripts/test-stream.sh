#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077
BASE="${AIHUB_BASE:-http://127.0.0.1:8317}"
MODEL="${AIHUB_TEST_MODEL:-}"
if [ -z "$MODEL" ]; then
  echo "Usage: AIHUB_TEST_MODEL='provider/model' ./scripts/test-stream.sh" >&2
  exit 2
fi
case "$MODEL" in
  */?*) ;;
  *) echo "Model must use provider/model form." >&2; exit 2 ;;
esac
case "$BASE" in
  http://127.0.0.1:*|http://localhost:*|http://\[::1\]:*) ;;
  *) echo "SSE diagnostic accepts only loopback AhB URLs." >&2; exit 2 ;;
esac
for tool in curl jq; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Missing $tool (Termux: pkg install -y curl jq)" >&2
    exit 2
  fi
done
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
payload="$(jq -nc --arg m "$MODEL" '{
  model:$m,stream:true,
  messages:[{role:"user",content:"Write two short sentences explaining how stars produce light."}]
}')"
status="$(curl --silent --show-error --no-buffer --max-time 120 \
  -o "$tmp" -w '%{http_code}' -H 'Content-Type: application/json' \
  -d "$payload" "$BASE/v1/chat/completions" || true)"
if [ "$status" != 200 ]; then
  echo "FAIL: $MODEL streaming HTTP $status" >&2
  exit 1
fi
# A valid SSE stream must terminate with [DONE] and contain at least one
# structured assistant content chunk. HTTP 200, keepalive-only events or
# a bare [DONE] must NOT be classified as successful inference.
if ! grep -Eq '^data: *\[DONE\]\r?$' "$tmp"; then
  echo "FAIL: $MODEL streamed no [DONE] marker." >&2
  exit 1
fi
chunks="$(sed -n 's/^data: *//p' "$tmp" | \
  jq -R -s '
    [split("\n")[] |
      gsub("\r$";"") |
      select(length>0 and .!="[DONE]") |
      (try fromjson catch null) |
      select(type=="object") |
      (.choices[0].delta.content // "") |
      select(type=="string" and length>0)
    ] | length
  ' 2>/dev/null || true)"
if ! [[ "$chunks" =~ ^[0-9]+$ ]] || [ "$chunks" -lt 1 ]; then
  echo "FAIL: HTTP 200 + [DONE] but no structured assistant text chunks." >&2
  exit 1
fi
echo "PASS: $MODEL Chat Completions SSE, $chunks content chunks and [DONE]"
echo "NOTE: this proves completed SSE framing, not per-token latency or long-run stability."
