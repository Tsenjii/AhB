#!/usr/bin/env bash
set -euo pipefail
umask 077
: "${AHB_PUBLIC_TOKEN:?Set AHB_PUBLIC_TOKEN in private Docker environment or .env}"
[ "${#AHB_PUBLIC_TOKEN}" -ge 32 ] || { echo 'AHB_PUBLIC_TOKEN must contain at least 32 characters' >&2; exit 1; }
ROOT="${AHB_STATE_DIR:-/state}/AhB"
mkdir -p "$ROOT"
if [ ! -f "$ROOT/config.example.json" ]; then
  echo 'Initializing private AhB state from pinned Linux release'
  cp -a /opt/AhB/. "$ROOT/"
else
  cp -a /opt/AhB/bin/. "$ROOT/bin/"
  cp -a /opt/AhB/scripts/. "$ROOT/scripts/"
  cp -a /opt/AhB/configs/. "$ROOT/configs/"
  cp /opt/AhB/config.example.json "$ROOT/config.example.json"
  for item in \
    freebuff/gateway \
    agent2api/ui \
    deepseek2api/static/admin \
    grok2api/frontend/dist \
    kiro-go/web; do
    if [ -d "/opt/AhB/data/$item" ]; then
      mkdir -p "$ROOT/data/$(dirname "$item")"
      rm -rf "$ROOT/data/$item"
      cp -a "/opt/AhB/data/$item" "$ROOT/data/$item"
    fi
  done
fi
cd "$ROOT"
bash ./scripts/prepare-configs.sh
if ! jq -e '.listen == "127.0.0.1:8317" and (.allow_lan == false) and (.resources.max_running_sidecars >= 1) and (.resources.max_running_sidecars <= 16)' config.json >/dev/null; then
  echo 'Unsafe config: Docker requires loopback Hub and valid process ceiling' >&2
  exit 1
fi
env -u AHB_PUBLIC_TOKEN -u AHB_PUBLIC_USER ./bin/hubd -config ./config.json &
hubpid=$!
trap 'kill -TERM "$hubpid" 2>/dev/null || :; wait "$hubpid" 2>/dev/null || :' EXIT
node /usr/local/bin/ahb-auth-gateway.mjs &
proxy_pid=$!
set +e
wait -n "$hubpid" "$proxy_pid"
result=$?
set -e
kill -TERM "$proxy_pid" 2>/dev/null || :
wait "$proxy_pid" 2>/dev/null || :
exit "$result"
