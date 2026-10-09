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
  and (any(.providers[]; .id == "freebuff" and .enabled == true
    and .env.FREEBUFF_CREDENTIALS_DIR == "./credentials"
    and .env.FREEBUFF_API_KEY != "__AIHUB_SERVER_KEY__"
    and (.headers.Authorization | startswith("Bearer "))))
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
# Regression: AhB's earliest public ARM64 bundle had only three providers,
# no routing block, and local secret files already populated. Upgrade directly,
# without installing any intermediate version or rotating user credentials.
LEGACY="$TMP/legacy/AhB"
mkdir -p "$LEGACY/scripts" "$LEGACY/configs" "$LEGACY/data/opencode" "$LEGACY/data/freebuff"
cp "$ROOT/scripts/prepare-configs.sh" "$LEGACY/scripts/"
cp "$ROOT/config.example.json" "$LEGACY/"
cp "$ROOT/configs/"* "$LEGACY/configs/"
cat > "$LEGACY/config.json" <<'JSON'
{
  "listen": "127.0.0.1:8317",
  "allow_lan": false,
  "providers": [
    {
      "id": "opencode", "enabled": true, "kind": "sidecar",
      "base_url": "http://127.0.0.1:8401",
      "binary": "./bin/opencode2api", "args": ["-config", "config.json"],
      "work_dir": "./data/opencode",
      "headers": {"Authorization": "Bearer FIRST_RELEASE_SECRET"},
      "health_path": "/healthz", "models_path": "/v1/models"
    },
    {
      "id": "freebuff", "enabled": true, "kind": "sidecar",
      "base_url": "http://127.0.0.1:8402",
      "binary": "./bin/freebuff2api", "work_dir": "./data/freebuff",
      "env": {"RUST_LOG": "info"},
      "ui_url": "http://127.0.0.1:8402/ui",
      "health_path": "/healthz", "models_path": "/v1/models"
    },
    {"id": "agent2api", "enabled": false, "kind": "sidecar",
      "base_url": "http://127.0.0.1:8403", "binary": "./bin/agent2api-server"}
  ]
}
JSON
printf 'FIRST_RELEASE_SECRET\n' > "$LEGACY/data/hub-local-key.txt"
cp "$ROOT/configs/opencode2api.json" "$LEGACY/data/opencode/config.json"
sed -i 's/__AIHUB_SERVER_KEY__/FIRST_RELEASE_SECRET/g' "$LEGACY/data/opencode/config.json"
printf 'OLD_COOKIE_DO_NOT_CONVERT\n' > "$LEGACY/data/freebuff/tokens.json"
(
  cd "$LEGACY"
  bash scripts/prepare-configs.sh
  jq -e '
    ((.routing.same_model_fallback.enabled // false) == false)
    and (any(.providers[]; .id == "opencode" and .enabled == true
      and .headers.Authorization == "Bearer FIRST_RELEASE_SECRET"))
    and (any(.providers[]; .id == "freebuff" and .enabled == true
      and .binary == "./bin/freebuff2api"
      and .ui_url == "" and .env.FREEBUFF_CREDENTIALS_DIR == "./credentials"
      and (.headers.Authorization | startswith("Bearer "))))
    and (any(.providers[]; .id == "copilot" and .enabled == false))
    and (any(.providers[]; .id == "grok" and .enabled == false))
    and (any(.providers[]; .id == "kiro" and .enabled == false))
  ' config.json >/dev/null
  test "$(cat data/hub-local-key.txt)" = "FIRST_RELEASE_SECRET"
  test "$(cat data/freebuff/tokens.json)" = "OLD_COOKIE_DO_NOT_CONVERT"
  jq -e '.server_keys[0] == "FIRST_RELEASE_SECRET"' data/opencode/config.json >/dev/null
  sha256sum config.json > "$TMP/legacy-config-hash"
  bash scripts/prepare-configs.sh
  sha256sum -c "$TMP/legacy-config-hash" >/dev/null
)
echo "first-release 3-provider AhB config -> latest 7-provider migration fixture passed"

echo "prepare-configs fixture passed (atomic preserve, new disabled providers, idempotence, malformed config rejection)"
