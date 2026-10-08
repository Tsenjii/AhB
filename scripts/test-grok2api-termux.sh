#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077

# Grok2API on-device readiness and optional real model acceptance.
# Inspect independent layers: process, startup readiness, authenticated
# model inventory, then explicit quota-consuming chat/SSE/tool continuation.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
for tool in curl jq; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Missing $tool; install with: pkg install -y curl jq" >&2
    exit 2
  fi
done
BASE="${AIHUB_GROK_BASE:-http://127.0.0.1:8407}"
HUB="${AIHUB_BASE:-http://127.0.0.1:8317}"
for url in "$BASE" "$HUB"; do
  case "$url" in
    http://127.0.0.1:*|http://localhost:*|http://\[::1\]:*) ;;
    *) echo "Grok diagnostics allow loopback HTTP only." >&2; exit 2 ;;
  esac
done
model="${AIHUB_TEST_MODEL:-}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
status(){
  local url="$1" dest="$2"
  curl --silent --show-error --max-time 8 -o "$dest" -w '%{http_code}' "$url" 2>/dev/null || true
}
if [ ! -x bin/grok2api ]; then
  echo "NOT INSTALLED: bin/grok2api (upgrade the published Android prebuilt)." >&2
  exit 2
fi
h="$(status "$BASE/healthz" "$tmp/health")"
if [ "$h" != 200 ] || ! jq -e '.ok == true' "$tmp/health" >/dev/null 2>&1; then
  echo "FAIL: Grok2API process health (HTTP $h)." >&2
  echo "Enable from Termux with ./scripts/enable-grok2api.sh then restart AhB." >&2
  exit 1
fi
echo "PASS: Grok2API process HTTP 200."
r="$(status "$BASE/readyz" "$tmp/ready")"
ready="$(jq -r '.ready // false' "$tmp/ready" 2>/dev/null || printf false)"
if [ "$r" = 200 ] && [ "$ready" = true ]; then
  echo "PASS: Grok2API startup ready."
else
  echo "NOT READY: Grok2API startup HTTP $r (readiness=$ready)."
  echo "Use the Grok2API native admin UI to inspect account/startup state; do not infer account quotas."
fi
if [ -s "$tmp/ready" ]; then
  jq -r '(.components // {}) | to_entries[] | "  \(.key): \(.value.state // "unknown")"' "$tmp/ready" 2>/dev/null || true
fi

keyfile="data/grok2api/client-key.txt"
if [ ! -s "$keyfile" ]; then
  echo "NO CLIENT KEY: enable Grok2API first; model access not verified."
  exit 1
fi
key="$(tr -d '\r\n' < "$keyfile")"
m="$(curl --silent --show-error --max-time 12 \
  -H "Authorization: Bearer $key" -o "$tmp/models" -w '%{http_code}' "$BASE/v1/models" 2>/dev/null || true)"
unset key
if [ "$m" != 200 ] || ! jq -e '.data | type == "array"' "$tmp/models" >/dev/null 2>&1; then
  echo "FAIL: authenticated Grok models inventory HTTP $m." >&2
  exit 1
fi
count="$(jq -r '.data | length' "$tmp/models")"
echo "PASS: authenticated /v1/models, $count models listed (NOT proof of usable quota)."
if [ "$count" = 0 ]; then
  echo "NO MODELS: initialize an authorized Grok account and sync models in the upstream admin."
  exit 1
fi

if [ -z "$model" ]; then
  echo "Metadata-only checks finished. This does NOT prove inference works."
  echo "For real Chat/SSE/function-call verification, use:"
  echo "  AIHUB_TEST_MODEL='grok/<exact-listed-model>' ./scripts/test-grok2api-termux.sh"
  exit 0
fi
case "$model" in
  grok/*) ;;
  *) echo "AIHUB_TEST_MODEL must start with grok/." >&2; exit 2 ;;
esac
if [ "$ready" != true ]; then
  echo "FAIL: won't spend quota while Grok2API readiness is false." >&2
  exit 1
fi
echo "LIVE TEST: using your authorized Grok account quota."
payload="$(jq -nc --arg model "$model" '{
  model:$model,stream:false,
  messages:[{role:"user",content:"Please reply with exactly GROK_AHB_OK."}]
}')"
chat="$(curl --silent --show-error --max-time 120 -o "$tmp/chat" -w '%{http_code}' \
  -H 'Content-Type: application/json' -d "$payload" "$HUB/v1/chat/completions" 2>/dev/null || true)"
if [ "$chat" != 200 ] || ! jq -e '
  (.choices[0].message.content // "") | type == "string" and length > 0
' "$tmp/chat" >/dev/null 2>&1; then
  echo "FAIL: Grok chat HTTP $chat; check upstream quota and authorization." >&2
  exit 1
fi
echo "PASS: Grok actual chat."
payload="$(jq -nc --arg model "$model" '{
  model:$model,stream:true,messages:[{role:"user",content:"Reply in three short sentences about Taiwan."}]
}')"
sse="$(curl --silent --show-error -N --max-time 120 -o "$tmp/sse" -w '%{http_code}' \
  -H 'Content-Type: application/json' -d "$payload" "$HUB/v1/chat/completions" 2>/dev/null || true)"
events="$(grep -c '^data:' "$tmp/sse" || true)"
if [ "$sse" != 200 ] || [ "$events" -lt 2 ] ||
  ! grep -Eq '^data: *\[DONE\]\r?$' "$tmp/sse"; then
  echo "FAIL: Grok SSE stream HTTP $sse, events=$events, missing/incomplete DONE." >&2
  exit 1
fi
echo "PASS: SSE event format, events=$events, [DONE] seen."
echo "Checking two-turn structured tool-call protocol..."
AIHUB_BASE="$HUB" AIHUB_TEST_MODEL="$model" bash ./scripts/test-tool-roundtrip.sh
echo "GROK ACCEPTANCE PASS: chat, SSE event formatting, structured tool-call and tool-result continuation."
echo "This is a short test, not long-term reliability or every Grok provider's entitlement."
