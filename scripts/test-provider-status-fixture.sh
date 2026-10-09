#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/bin"
cat > "$tmp/bin/curl" <<'SH'
#!/usr/bin/env bash
out=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "-o" ]; then out="$arg"; fi
  prev="$arg"
done
case "${!#}" in
  */api/providers) cp "$FIXTURE_DIR/providers.json" "$out" ;;
  */v1/models) cp "$FIXTURE_DIR/models.json" "$out" ;;
  *) exit 22 ;;
esac
SH
chmod +x "$tmp/bin/curl"
cat >"$tmp/providers.json" <<'JSON'
{"providers":[
 {"id":"opencode","enabled":true,"state":"HEALTHY","provider_ready":true,"last_request_http_status":200,"account_total":null},
 {"id":"agent2api","enabled":true,"state":"HEALTHY","provider_ready":true,"account_total":2,"account_usable_count":2,"last_request_http_status":503,"private_key":"MUST_NOT_LEAK"},
 {"id":"freebuff","enabled":false,"state":"DISABLED","provider_ready":false,"account_total":null},
 {"id":"geminiweb","enabled":true,"start_mode":"on_demand","process_alive":true,"state":"HEALTHY","provider_ready":false,"account_total":1,"account_usable_count":0,"last_request_http_status":429},
 {"id":"duckai","enabled":true,"start_mode":"on_demand","process_alive":false,"state":"HEALTHY","provider_ready":true,"account_total":null,"last_request_http_status":200}
]}
JSON
cat >"$tmp/models.json" <<'JSON'
{"data":[{"id":"opencode/alpha","x_provider":"opencode"},{"id":"agent2api/alpha","x_provider":"agent2api"},{"id":"geminiweb/alpha","x_provider":"geminiweb"},{"id":"duckai/alpha","x_provider":"duckai","x_cached":true}]}
JSON
export PATH="$tmp/bin:$PATH" FIXTURE_DIR="$tmp"
bash "$ROOT/scripts/provider-status-termux.sh" > "$tmp/output"
grep -q 'agent2api.*503' "$tmp/output"
grep -q 'freebuff.*OFF' "$tmp/output"
grep -q 'grok.*MISSING' "$tmp/output"
grep -q 'geminiweb.*429' "$tmp/output"
grep -q 'duckai.*200' "$tmp/output"
grep -q 'geminiweb.*1.*429' "$tmp/output"
grep -q 'duckai.*CACHE:1.*200' "$tmp/output"
! grep -q 'MUST_NOT_LEAK' "$tmp/output"
echo "provider inventory fixture passed; nine bundled sources including Gemini Web and Duck.ai"
