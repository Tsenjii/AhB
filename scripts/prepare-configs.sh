#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p data/opencode data/freebuff data/agent2api data/deepseek2api logs bin

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

# Migrate existing installs without overwriting user settings.
# DeepSeek is appended disabled; enabling it remains an explicit user action.
# Android/Termux Go sidecars use libc DNS to avoid loopback-resolver failures.
if command -v jq >/dev/null 2>&1 && [ -f config.json ]; then
  tmp="$(mktemp)"
  deepseek_provider="$(jq -c '.providers[] | select(.id == "deepseek")' config.example.json)"
  if jq --argjson deepseek "$deepseek_provider" '
      if any(.providers[]; .id == "deepseek") then . else .providers += [$deepseek] end
      | (.providers[] | select(.id == "opencode") | .env) =
          (((.providers[] | select(.id == "opencode") | .env) // {}) + {"GODEBUG":"netdns=cgo"})
      | (.routing.same_model_fallback.providers // []) as $p
      | if (.routing.same_model_fallback? != null and ($p | index("deepseek")) == null)
        then .routing.same_model_fallback.providers += ["deepseek"]
        else .
        end
    ' config.json > "$tmp"; then
    cat "$tmp" > config.json
  fi
  rm -f "$tmp"
fi

make_secret() {
  od -An -N24 -tx1 /dev/urandom | tr -d ' \n'
}

KEY_FILE="data/hub-local-key.txt"
if [ ! -s "$KEY_FILE" ]; then
  make_secret > "$KEY_FILE"
  printf '\n' >> "$KEY_FILE"
  chmod 600 "$KEY_FILE"
fi
LOCAL_KEY="$(tr -d '\r\n' < "$KEY_FILE")"

for file in config.json data/opencode/config.json data/deepseek2api/config.json; do
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

chmod 600 config.json data/opencode/config.json data/freebuff/config.json data/deepseek2api/config.json data/deepseek2api/admin-key.txt 2>/dev/null || true