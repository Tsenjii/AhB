#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p data/opencode data/freebuff data/agent2api logs bin

if [ ! -f config.json ]; then
  cp config.example.json config.json
fi
if [ ! -f data/opencode/config.json ]; then
  cp configs/opencode2api.json data/opencode/config.json
fi
if [ ! -f data/freebuff/config.json ]; then
  cp configs/freebuff2api.json data/freebuff/config.json
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

for file in config.json data/opencode/config.json; do
  if [ -f "$file" ]; then
    sed -i "s/__AIHUB_SERVER_KEY__/$LOCAL_KEY/g" "$file"
    sed -i "s/hub-local-opencode/$LOCAL_KEY/g" "$file"
  fi
done

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

chmod 600 config.json data/opencode/config.json data/freebuff/config.json 2>/dev/null || true