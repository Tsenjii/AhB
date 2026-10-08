#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/AhB/scripts" "$TMP/AhB/bin" "$TMP/AhB/data/grok2api/frontend/dist" "$TMP/AhB/logs" "$TMP/shims"
cp "$ROOT/scripts/enable-grok2api.sh" "$TMP/AhB/scripts/"
cat > "$TMP/AhB/scripts/prepare-configs.sh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
SH
cat > "$TMP/AhB/bin/grok2api" <<'SH'
#!/usr/bin/env bash
exit 0
SH
cat > "$TMP/shims/curl" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
last="${*: -1}"
case "$last" in
  */healthz) printf '200' ;;
  */v1/models) printf '%s' "${GROK_KEY_HTTP:-503}" ;;
  */api/admin/v1/auth/login)
    printf '{"data":{"tokens":{"accessToken":"test-admin-session"}}}'
    ;;
  */api/admin/v1/client-keys)
    printf '{"data":{"secret":"new-grok-client-key"}}'
    ;;
  *) echo "unexpected curl URL: $last" >&2; exit 2 ;;
esac
SH
chmod +x "$TMP/AhB/bin/grok2api" "$TMP/AhB/scripts/"* "$TMP/shims/curl"
printf 'fake UI\n' > "$TMP/AhB/data/grok2api/frontend/dist/index.html"
printf 'fake-password\n' > "$TMP/AhB/data/grok2api/admin-password.txt"
printf 'old-grok-client-key\n' > "$TMP/AhB/data/grok2api/client-key.txt"
cat > "$TMP/AhB/config.json" <<'JSON'
{"providers":[{"id":"grok","kind":"sidecar","enabled":false,"headers":{"Authorization":"Bearer old-grok-client-key"}}]}
JSON
cd "$TMP/AhB"
export PATH="$TMP/shims:$PATH"
cp config.json "$TMP/before.json"
if GROK_KEY_HTTP=503 bash ./scripts/enable-grok2api.sh >/dev/null 2>&1; then
  echo "error: transient 503 should fail closed without altering credentials" >&2
  exit 1
fi
cmp "$TMP/before.json" config.json
test "$(cat data/grok2api/client-key.txt)" = "old-grok-client-key"
test ! -e data/grok2api/config-before-enable.json

GROK_KEY_HTTP=200 bash ./scripts/enable-grok2api.sh >/dev/null
jq -e '.providers[] | select(.id=="grok" and .enabled==true and .headers.Authorization=="Bearer old-grok-client-key")' config.json >/dev/null
cmp "$TMP/before.json" data/grok2api/config-before-enable.json
test "$(stat -c '%a' config.json)" = "600"

cp "$TMP/before.json" config.json
GROK_KEY_HTTP=401 bash ./scripts/enable-grok2api.sh >/dev/null 2>&1
test "$(cat data/grok2api/client-key.txt)" = "new-grok-client-key"
jq -e '.providers[] | select(.id=="grok" and .enabled==true and .headers.Authorization=="Bearer new-grok-client-key")' config.json >/dev/null
echo "Grok2API client key recovery fixture passed (503 preserve, 200 reuse, 401 recover)"
