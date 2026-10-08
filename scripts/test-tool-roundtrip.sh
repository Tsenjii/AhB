#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

# Require a structured function call, then give that call a tool response
# and require a second model answer. This test consumes upstream quota.
BASE="${AIHUB_BASE:-http://127.0.0.1:8317}"
MODEL="${AIHUB_TEST_MODEL:-}"
if [ -z "$MODEL" ]; then
  echo "Usage: AIHUB_TEST_MODEL='provider/model' ./scripts/test-tool-roundtrip.sh" >&2
  exit 2
fi
for cmd in curl jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Missing required command: $cmd" >&2
    exit 2
  fi
done
case "$BASE" in
  http://127.0.0.1:*|http://localhost:*|http://\[::1\]:*) ;;
  *) echo "Only local AhB endpoints are supported by this diagnostic." >&2; exit 2 ;;
esac

first="$(mktemp)"
second="$(mktemp)"
trap 'rm -f "$first" "$second"' EXIT
payload="$(jq -nc --arg model "$MODEL" '{
  model:$model,
  stream:false,
  messages:[{role:"user",content:"Use the get_time function for Asia/Taipei first. After you receive the tool result, finish with the text TOOL_LOOP_OK."}],
  tools:[{type:"function",function:{
    name:"get_time",description:"Return a timezone-specific timestamp.",
    parameters:{type:"object",properties:{timezone:{type:"string"}},required:["timezone"],additionalProperties:false}
  }}],
  tool_choice:"auto"
}')"

status="$(curl -sS --max-time 120 -o "$first" -w '%{http_code}' \
  -H 'Content-Type: application/json' -X POST "$BASE/v1/chat/completions" -d "$payload" || true)"
if [ "$status" != "200" ]; then
  echo "FAIL: initial tool request HTTP $status" >&2
  jq -c '{error:(.error // .detail // "non-200 response")}' "$first" 2>/dev/null || true
  exit 1
fi
if ! jq -e '
  (.choices[0].message.tool_calls | type == "array" and length > 0)
  and (.choices[0].message.tool_calls[0].type == "function")
  and (.choices[0].message.tool_calls[0].function.name == "get_time")
  and ((.choices[0].message.tool_calls[0].id // "") | length > 0)
  and ((.choices[0].message.tool_calls[0].function.arguments | fromjson) | type == "object")
' "$first" >/dev/null 2>&1; then
  echo "FAIL: model did not emit a native structured get_time tool call." >&2
  echo "Text/XML guesses are NOT native OpenAI tool calling." >&2
  exit 1
fi

# Deterministic test result, not actual current clock time.
# No arbitrary shell commands are executed from model arguments.
follow="$(jq -nc --slurpfile response "$first" --arg model "$MODEL" '{
  model:$model,
  stream:false,
  messages:[
    {role:"user",content:"Use the get_time function for Asia/Taipei first. After you receive the tool result, finish with the text TOOL_LOOP_OK."},
    ($response[0].choices[0].message |
      {role:"assistant",content:(.content // null),tool_calls:.tool_calls}),
    {role:"tool",tool_call_id:$response[0].choices[0].message.tool_calls[0].id,
     content:"{\"timezone\":\"Asia/Taipei\",\"timestamp\":\"2030-01-01T12:34:56+08:00\"}"}
  ]
}')"
status="$(curl -sS --max-time 120 -o "$second" -w '%{http_code}' \
  -H 'Content-Type: application/json' -X POST "$BASE/v1/chat/completions" -d "$follow" || true)"
if [ "$status" != "200" ]; then
  echo "FAIL: after tool result, completion HTTP $status" >&2
  jq -c '{error:(.error // .detail // "non-200 response")}' "$second" 2>/dev/null || true
  exit 1
fi
if ! jq -e '
  (.choices[0].message.content | type == "string" and length > 0)
  and (.choices[0].finish_reason != "tool_calls")
' "$second" >/dev/null 2>&1; then
  echo "FAIL: no final assistant answer after submitting the tool result." >&2
  exit 1
fi
echo "PASS: native function call -> tool result -> second model answer"
echo "model: $MODEL"
jq -r '.choices[0].message.content' "$second"
