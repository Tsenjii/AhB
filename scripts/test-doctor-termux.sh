#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

mkdir -p "$TMP/AhB/scripts" "$TMP/AhB/bin" \
  "$TMP/AhB/data/opencode" "$TMP/AhB/data/freebuff/gateway" \
  "$TMP/AhB/data/copilot2api" "$TMP/AhB/data/grok2api/frontend/dist" \
  "$TMP/shims"
cp "$ROOT/scripts/doctor-termux.sh" "$TMP/AhB/scripts/"
for file in config.json data/opencode/config.json data/opencode/webui-password.txt \
  data/freebuff/config.json data/freebuff/gateway/server.js data/freebuff/gateway/worker.js data/copilot2api/credentials.json \
  data/grok2api/config.yaml data/grok2api/frontend/dist/index.html \
  data/grok2api/client-key.txt bin/hubd bin/opencode2api bin/freebuff2api \
  bin/copilot2api bin/grok2api; do
  printf 'fixture\n' > "$TMP/AhB/$file"
done
cat > "$TMP/AhB/config.json" <<'JSON'
{
  "providers": [
    {"id":"opencode","enabled":true,"kind":"sidecar"},
    {"id":"freebuff","enabled":false,"kind":"sidecar"},
    {"id":"grok","enabled":true,"kind":"sidecar"},
    {"id":"copilot","enabled":true,"kind":"sidecar"},
    {"id":"kimiweb","enabled":false,"kind":"sidecar"},
    {"id":"lmarena","enabled":true,"kind":"external","base_url":"http://127.0.0.1:5511","health_path":"/v1/models"},
    {"id":"codex","enabled":true,"kind":"external","base_url":"http://127.0.0.1:5512","health_path":"/v1/models","headers":{"Authorization":"Bearer private-fixture"}}
  ]
}
JSON
cat > "$TMP/shims/curl" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DOCTOR_CURL_MARKER"
printf '200'
SH
chmod +x "$TMP/shims/curl"
export PATH="$TMP/shims:$PATH"
export DOCTOR_CURL_MARKER="$TMP/curl.log"
(
  cd "$TMP/AhB"
  bash scripts/doctor-termux.sh
) > "$TMP/output.txt"

grep -q "UP       external-lmarena" "$TMP/output.txt"
grep -q "HUB CHECK external-codex" "$TMP/output.txt"
grep -q "UP       grok-ready" "$TMP/output.txt"
grep -q "UP       copilot-models" "$TMP/output.txt"
if grep -q "kimiweb-health" "$TMP/output.txt"; then
  echo "doctor incorrectly probed a disabled provider" >&2
  exit 1
fi
if grep -q "127.0.0.1:5512" "$DOCTOR_CURL_MARKER"; then
  echo "doctor leaked a private external bridge via unauthenticated direct probe" >&2
  exit 1
fi
if [ "$(grep -c '^== hub runtime ==$' "$TMP/output.txt")" -ne 1 ] ||
   [ "$(grep -c '^== providers ==$' "$TMP/output.txt")" -ne 1 ]; then
  echo "doctor emitted duplicate or malformed runtime reports" >&2
  exit 1
fi
echo "doctor Termux fixture passed (external bridge, auth, disabled/optional providers, no duplicate sections)"
