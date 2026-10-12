#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/AhB/scripts" "$TMP/AhB/data" "$TMP/shims"
cp "$ROOT/scripts/connect-bridge.sh" "$TMP/AhB/scripts/"

cat > "$TMP/AhB/scripts/prepare-configs.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p data
test -f config.json || cp config.example.json config.json
EOF
cat > "$TMP/AhB/config.example.json" <<'EOF'
{
  "listen":"127.0.0.1:8317",
  "providers":[
    {"id":"opencode","kind":"sidecar","enabled":false,"binary":"./bin/opencode2api"},
    {"id":"lmarena","kind":"external","enabled":false,"base_url":"http://127.0.0.1:8406","models_path":"/v1/models","health_path":"/v1/models"}
  ],
  "routing":{"same_model_fallback":{"enabled":false,"providers":["opencode","lmarena"]}}
}
EOF
cat > "$TMP/shims/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "${AHB_STUB_CURL_FAIL:-0}" = "1" ]; then
  exit 22
fi
if [ "${AHB_STUB_CURL_INVALID:-0}" = "1" ]; then
  printf '{"not_a_models_list":true}\n'
else
  printf '{"object":"list","data":[{"id":"local-test-model"}]}\n'
fi
EOF
chmod +x "$TMP/shims/curl" "$TMP/AhB/scripts/"*.sh
export PATH="$TMP/shims:$PATH"

cd "$TMP/AhB"
AIHUB_BRIDGE_API_KEY="private-key" bash ./scripts/connect-bridge.sh lmarena http://127.0.0.1:5102
jq -e '.providers[] | select(.id=="lmarena" and .enabled==true and .base_url=="http://127.0.0.1:5102")' config.json >/dev/null
jq -e '.providers[] | select(.id=="lmarena").headers.Authorization=="Bearer private-key"' config.json >/dev/null
AIHUB_BRIDGE_API_KEY="" bash ./scripts/connect-bridge.sh lmarena http://127.0.0.1:5102
test "$(jq '[.providers[] | select(.id=="lmarena")] | length' config.json)" -eq 1
jq -e '.providers[] | select(.id=="lmarena").headers.Authorization=="Bearer private-key"' config.json >/dev/null

# CLIProxyAPI is opt-in, uses separate 8416 (not AhB's 8317) and must
# preserve locally supplied API authentication; no account data is imported.
AIHUB_BRIDGE_API_KEY="cliproxy-test-local-key" bash ./scripts/connect-bridge.sh cliproxy http://127.0.0.1:8416
jq -e '.providers[] | select(.id=="cliproxy" and .kind=="external" and .enabled==true and .base_url=="http://127.0.0.1:8416" and .headers.Authorization=="Bearer cliproxy-test-local-key" and .docs_url=="https://github.com/router-for-me/CLIProxyAPI")' config.json >/dev/null
# New web sources remain separate external providers, never silently replacing
# the existing DeepSeek/Duck sidecars or Kimi account records.
bash ./scripts/connect-bridge.sh qwenweb http://127.0.0.1:3000
jq -e '.providers[] | select(.id=="qwenweb" and .kind=="external" and .docs_url=="https://github.com/XxxXTeam/Qwen2API_Go")' config.json >/dev/null
bash ./scripts/connect-bridge.sh deepseekweb http://127.0.0.1:3456
jq -e '.providers[] | select(.id=="deepseekweb" and .kind=="external" and .docs_url=="https://github.com/PruhaNLP/web-to-api")' config.json >/dev/null
bash ./scripts/connect-bridge.sh kimi http://127.0.0.1:8000
jq -e '.providers[] | select(.id=="kimi" and .kind=="external" and .docs_url=="https://github.com/chopper1026/kimi2api")' config.json >/dev/null
# New lightweight optional connectors remain independent local IDs.
for entry in 'mimo 8090 https://github.com/rong6/mimo-2api' \
             'geminioauth 10551 https://github.com/kimws/gemini-oauth' \
             'claudecli 8080 https://github.com/kpod13/claude-openai-proxy'; do
  read -r preset port docs <<< "$entry"
  bash ./scripts/connect-bridge.sh "$preset" "http://127.0.0.1:$port"
  jq -e --arg id "$preset" --arg docs "$docs" \
    '.providers[] | select(.id==$id and .kind=="external" and .enabled==true and .docs_url==$docs)' config.json >/dev/null
done
bash ./scripts/connect-bridge.sh mybridge http://127.0.0.1:8560
jq -e '.providers[] | select(.id=="mybridge" and .kind=="external" and .enabled==true)' config.json >/dev/null
test "$(jq '[.routing.same_model_fallback.providers[] | select(.=="mybridge")] | length' config.json)" -eq 1

cp config.json "$TMP/before.json"
if bash ./scripts/connect-bridge.sh opencode http://127.0.0.1:9999; then
 echo "expected managed sidecar rejection" >&2; exit 1
fi
if bash ./scripts/connect-bridge.sh bad http://example.com:3000; then
 echo "expected non-loopback rejection" >&2; exit 1
fi
if AHB_STUB_CURL_FAIL=1 bash ./scripts/connect-bridge.sh unreachable http://127.0.0.1:9112; then
 echo "expected probe failure" >&2; exit 1
fi
if AHB_STUB_CURL_INVALID=1 bash ./scripts/connect-bridge.sh invalid http://127.0.0.1:9113; then
 echo "expected malformed model-list rejection" >&2; exit 1
fi
cmp config.json "$TMP/before.json"
test "$(stat -c %a config.json)" = "600"
echo "bridge connector fixture tests passed"
