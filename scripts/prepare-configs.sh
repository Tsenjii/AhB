#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p data/opencode data/freebuff data/agent2api data/deepseek2api data/grok2api/frontend data/grok2api/data data/kiro-go/web logs bin

if [ ! -f config.json ]; then
  cp config.example.json config.json
fi
if [ ! -f data/opencode/config.json ]; then
  cp configs/opencode2api.json data/opencode/config.json
fi
if [ ! -f data/freebuff/config.json ]; then
  cp configs/freebuff2api.json data/freebuff/config.json
fi
if [ ! -f data/deepseek2api/config.json ]; then
  cp configs/deepseek2api.json data/deepseek2api/config.json
fi
if [ ! -f data/grok2api/config.yaml ]; then
  cp configs/grok2api.yaml data/grok2api/config.yaml
fi
if [ ! -f data/kiro-go/config.json ]; then
  cp configs/kiro-go.json data/kiro-go/config.json
fi

# Migrate existing installs without overwriting user settings.
# Newly bundled providers are appended from config.example.json without
# replacing existing provider-specific settings.
if command -v jq >/dev/null 2>&1 && [ -f config.json ]; then
  tmp="$(mktemp)"
  if jq --slurpfile example config.example.json '
      reduce $example[0].providers[] as $p (.;
        if any(.providers[]?; .id == $p.id) then . else .providers += [$p] end
      )
      | (.providers[] | select(.id == "opencode") | .env) =
          (((.providers[] | select(.id == "opencode") | .env) // {}) + {"GODEBUG":"netdns=cgo"})
      | (.providers[] | select(.id == "grok") | .env) =
          (((.providers[] | select(.id == "grok") | .env) // {}) + {"GODEBUG":"netdns=cgo"})
      | .routing.same_model_fallback.providers =
          (.routing.same_model_fallback.providers // [])
      | reduce ($example[0].routing.same_model_fallback.providers // [])[] as $id (.;
          if (.routing.same_model_fallback.providers | index($id)) == null
          then .routing.same_model_fallback.providers += [$id]
          else .
          end
        )
    ' config.json > "$tmp"; then
    cat "$tmp" > config.json
  fi
  rm -f "$tmp"
fi
make_secret() {
  od -An -N24 -tx1 /dev/urandom | tr -d ' \n'
}

make_base64_32() {
  dd if=/dev/urandom bs=32 count=1 2>/dev/null | base64 | tr -d '\r\n'
}

KEY_FILE="data/hub-local-key.txt"
if [ ! -s "$KEY_FILE" ]; then
  make_secret > "$KEY_FILE"
  printf '\n' >> "$KEY_FILE"
  chmod 600 "$KEY_FILE"
fi
LOCAL_KEY="$(tr -d '\r\n' < "$KEY_FILE")"

for file in config.json data/opencode/config.json data/deepseek2api/config.json data/kiro-go/config.json; do
  if [ -f "$file" ]; then
    sed -i "s/__AIHUB_SERVER_KEY__/$LOCAL_KEY/g" "$file"
    sed -i "s/hub-local-opencode/$LOCAL_KEY/g" "$file"
  fi
done

DEEPSEEK_ADMIN_KEY_FILE="data/deepseek2api/admin-key.txt"
if [ ! -s "$DEEPSEEK_ADMIN_KEY_FILE" ]; then
  make_secret > "$DEEPSEEK_ADMIN_KEY_FILE"
  printf '\n' >> "$DEEPSEEK_ADMIN_KEY_FILE"
  chmod 600 "$DEEPSEEK_ADMIN_KEY_FILE"
fi
DEEPSEEK_ADMIN_KEY="$(tr -d '\r\n' < "$DEEPSEEK_ADMIN_KEY_FILE")"
if [ -f config.json ]; then
  sed -i "s/__AIHUB_DEEPSEEK_ADMIN_KEY__/$DEEPSEEK_ADMIN_KEY/g" config.json
fi

GROK_JWT_FILE="data/grok2api/jwt-secret.txt"
if [ ! -s "$GROK_JWT_FILE" ]; then
  make_secret > "$GROK_JWT_FILE"
  printf '\n' >> "$GROK_JWT_FILE"
  chmod 600 "$GROK_JWT_FILE"
fi
GROK_JWT="$(tr -d '\r\n' < "$GROK_JWT_FILE")"

GROK_CRED_FILE="data/grok2api/credential-key.txt"
if [ ! -s "$GROK_CRED_FILE" ]; then
  make_base64_32 > "$GROK_CRED_FILE"
  printf '\n' >> "$GROK_CRED_FILE"
  chmod 600 "$GROK_CRED_FILE"
fi
GROK_CRED="$(tr -d '\r\n' < "$GROK_CRED_FILE")"

GROK_ADMIN_FILE="data/grok2api/admin-password.txt"
if [ ! -s "$GROK_ADMIN_FILE" ]; then
  make_secret > "$GROK_ADMIN_FILE"
  printf '\n' >> "$GROK_ADMIN_FILE"
  chmod 600 "$GROK_ADMIN_FILE"
fi
GROK_ADMIN="$(tr -d '\r\n' < "$GROK_ADMIN_FILE")"

if [ -f data/grok2api/config.yaml ]; then
  sed -i "s|__AIHUB_GROK_JWT_SECRET__|$GROK_JWT|g" data/grok2api/config.yaml
  sed -i "s|__AIHUB_GROK_CREDENTIAL_KEY__|$GROK_CRED|g" data/grok2api/config.yaml
  sed -i "s|__AIHUB_GROK_ADMIN_PASSWORD__|$GROK_ADMIN|g" data/grok2api/config.yaml
fi

GROK_CLIENT_FILE="data/grok2api/client-key.txt"
if [ -s "$GROK_CLIENT_FILE" ] && [ -f config.json ]; then
  GROK_CLIENT="$(tr -d '\r\n' < "$GROK_CLIENT_FILE")"
  sed -i "s|__AIHUB_GROK_CLIENT_KEY__|$GROK_CLIENT|g" config.json
fi

KIRO_ADMIN_FILE="data/kiro-go/admin-password.txt"
if [ ! -s "$KIRO_ADMIN_FILE" ]; then
  make_secret > "$KIRO_ADMIN_FILE"
  printf '\n' >> "$KIRO_ADMIN_FILE"
  chmod 600 "$KIRO_ADMIN_FILE"
fi
KIRO_ADMIN="$(tr -d '\r\n' < "$KIRO_ADMIN_FILE")"
if [ -f data/kiro-go/config.json ]; then
  sed -i "s|__AIHUB_KIRO_ADMIN_PASSWORD__|$KIRO_ADMIN|g" data/kiro-go/config.json
fi
if [ -f config.json ]; then
  sed -i "s|__AIHUB_KIRO_ADMIN_PASSWORD__|$KIRO_ADMIN|g" config.json
fi

WEB_PASS_FILE="data/opencode/webui-password.txt"
if grep -q "__AIHUB_WEB_PASSWORD__" data/opencode/config.json; then
  if [ ! -s "$WEB_PASS_FILE" ]; then
    make_secret > "$WEB_PASS_FILE"
    printf '\n' >> "$WEB_PASS_FILE"
    chmod 600 "$WEB_PASS_FILE"
  fi
  WEB_PASS="$(tr -d '\r\n' < "$WEB_PASS_FILE")"
  sed -i "s/__AIHUB_WEB_PASSWORD__/$WEB_PASS/g" data/opencode/config.json
fi

chmod 600 config.json data/opencode/config.json data/freebuff/config.json data/deepseek2api/config.json data/deepseek2api/admin-key.txt data/grok2api/config.yaml data/grok2api/jwt-secret.txt data/grok2api/credential-key.txt data/grok2api/admin-password.txt data/grok2api/client-key.txt data/kiro-go/config.json data/kiro-go/admin-password.txt 2>/dev/null || true