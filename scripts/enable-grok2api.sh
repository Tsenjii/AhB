#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

for cmd in curl jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    pkg install -y "$cmd"
  fi
done

if [ ! -x bin/grok2api ]; then
  echo "bin/grok2api is missing. Install the current Android prebuilt bundle first."
  exit 1
fi
if [ ! -f data/grok2api/frontend/dist/index.html ]; then
  echo "Grok2API frontend assets are missing."
  exit 1
fi

./scripts/prepare-configs.sh

ADMIN_PASSWORD="$(tr -d '\r\n' < data/grok2api/admin-password.txt)"
CLIENT_KEY_FILE="data/grok2api/client-key.txt"
BOOT_LOG="logs/grok2api-bootstrap.log"
BASE="http://127.0.0.1:8407"
started_here=0
pid=""

cleanup() {
  if [ "$started_here" = "1" ] && [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 1 "$BASE/healthz" 2>/dev/null || true)"
if [[ ! "$code" =~ ^2 ]]; then
  : > "$BOOT_LOG"
  "$ROOT/bin/grok2api" --config "$ROOT/data/grok2api/config.yaml" --listen 127.0.0.1:8407 >>"$BOOT_LOG" 2>&1 &
  pid=$!
  started_here=1
  ready=0
  for _ in $(seq 1 45); do
    if ! kill -0 "$pid" 2>/dev/null; then
      echo "Grok2API exited during bootstrap. See $BOOT_LOG"
      exit 1
    fi
    code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 1 "$BASE/healthz" 2>/dev/null || true)"
    if [[ "$code" =~ ^2 ]]; then ready=1; break; fi
    sleep 1
  done
  if [ "$ready" != "1" ]; then
    echo "Grok2API did not become reachable. See $BOOT_LOG"
    exit 1
  fi
fi

client_key=""
if [ -s "$CLIENT_KEY_FILE" ]; then
  candidate="$(tr -d '\r\n' < "$CLIENT_KEY_FILE")"
  auth_code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5     -H "Authorization: Bearer $candidate" "$BASE/v1/models" 2>/dev/null || true)"
  if [ "$auth_code" = "200" ]; then
    client_key="$candidate"
  fi
fi

if [ -z "$client_key" ]; then
  login_body="$(jq -nc --arg p "$ADMIN_PASSWORD" '{username:"admin",password:$p}')"
  login="$(curl -fsS --max-time 10 -H 'Content-Type: application/json'     -d "$login_body" "$BASE/api/admin/v1/auth/login")"
  access="$(printf '%s' "$login" | jq -r '.data.tokens.accessToken // empty')"
  if [ -z "$access" ]; then
    echo "Could not obtain Grok2API admin session."
    exit 1
  fi

  key_body='{"name":"AhB Local Hub","enabled":true,"rpmLimit":0,"maxConcurrent":0,"billingLimitUsdTicks":0,"allowModelAliases":true,"allowedModelIds":[],"providerScope":["all"],"tierScope":["all"]}'
  created="$(curl -fsS --max-time 10 -H 'Content-Type: application/json'     -H "Authorization: Bearer $access" -d "$key_body" "$BASE/api/admin/v1/client-keys")"
  client_key="$(printf '%s' "$created" | jq -r '.data.secret // empty')"
  if [ -z "$client_key" ]; then
    echo "Could not create AhB Grok2API client key."
    exit 1
  fi
  printf '%s\n' "$client_key" > "$CLIENT_KEY_FILE"
  chmod 600 "$CLIENT_KEY_FILE"
fi

tmp="$(mktemp)"
jq --arg key "$client_key" '
  (.providers[] | select(.id == "grok") | .enabled) = true
  | (.providers[] | select(.id == "grok") | .headers.Authorization) = ("Bearer " + $key)
' config.json > "$tmp"
mv "$tmp" config.json
chmod 600 config.json

echo "Grok2API enabled."
echo "Restart AhB, then open: http://127.0.0.1:8407/"
echo "Admin password is stored locally at: $ROOT/data/grok2api/admin-password.txt"
