#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/AhB/scripts" "$TMP/AhB/configs" "$TMP/AhB/data/grok2api/data"
cp "$ROOT/scripts/prepare-configs.sh" "$TMP/AhB/scripts/"
cp "$ROOT/config.example.json" "$TMP/AhB/"
cp "$ROOT/configs/"* "$TMP/AhB/configs/"
cat > "$TMP/AhB/config.json" <<'JSON'
{
  "listen":"127.0.0.1:8317",
  "allow_lan":false,
  "providers":[
    {"id":"opencode","enabled":true,"kind":"sidecar","headers":{"Authorization":"Bearer preserved-user-secret"},"env":{"CUSTOM":"preserved"}},
    {"id":"custom-provider","enabled":false,"kind":"external","base_url":"http://127.0.0.1:9001","headers":{"Authorization":"Bearer preserved-bridge-secret"}}
  ],
  "routing":{"same_model_fallback":{"enabled":false,"providers":["opencode"]}},
  "custom_user_metadata":{"important":"must-survive"}
}
JSON
printf 'account database must survive\n' > "$TMP/AhB/data/grok2api/data/backend.db"
cd "$TMP/AhB"
bash scripts/prepare-configs.sh
jq -e '
  (.providers | map(.id) | unique | length) == (.providers | length)
  and (any(.providers[]; .id == "copilot" and .enabled == false))
  and (any(.providers[]; .id == "kimiweb" and .enabled == false))
  and (any(.providers[]; .id == "grok" and .enabled == false))
  and (any(.providers[]; .id == "custom-provider" and .headers.Authorization == "Bearer preserved-bridge-secret"))
  and (any(.providers[]; .id == "opencode" and .headers.Authorization == "Bearer preserved-user-secret" and .env.CUSTOM == "preserved" and .env.GODEBUG == "netdns=cgo"))
  and (.custom_user_metadata.important == "must-survive")
' config.json >/dev/null
test "$(cat data/grok2api/data/backend.db)" = "account database must survive"
test "$(stat -c '%a' config.json)" = "600"
sha256sum config.json > "$TMP/config-before"
bash scripts/prepare-configs.sh
sha256sum -c "$TMP/config-before" >/dev/null
printf '{"providers":"not-an-array","unique":"unchanged"}\n' > config.json
cp config.json "$TMP/broken-original"
if bash scripts/prepare-configs.sh > "$TMP/broken.log" 2>&1; then
  echo "ERROR: malformed old config silently succeeded" >&2
  exit 1
fi
cmp config.json "$TMP/broken-original"
echo "prepare-configs fixture passed (atomic preserve, new disabled providers, idempotence, malformed config rejection)"
