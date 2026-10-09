#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
case "$(uname -m)" in
 x86_64|amd64) ARCH=amd64;; aarch64|arm64) ARCH=arm64;; *) exit 1;;
esac
ASSET="AhB_linux_$ARCH.tar.gz"
mkdir -p "$TMP/staged/AhB/bin" "$TMP/staged/AhB/scripts" "$TMP/staged/AhB/data/freebuff/gateway" "$TMP/release"
printf '#!/bin/sh\necho new-release\n' > "$TMP/staged/AhB/bin/hubd"
chmod +x "$TMP/staged/AhB/bin/hubd"
printf 'new-bundled-server\n' > "$TMP/staged/AhB/data/freebuff/gateway/server.js"
printf '#!/usr/bin/env bash\nexit 0\n' > "$TMP/staged/AhB/scripts/prepare-configs.sh"
cp "$ROOT/scripts/start-linux.sh" "$ROOT/scripts/stop-linux.sh" "$ROOT/scripts/install-prebuilt-linux.sh" "$TMP/staged/AhB/scripts/"
python3 - "$TMP/staged/AhB/config.example.json" <<'PY'
import json,sys
json.dump({"resources":{"max_running_sidecars":1,"idle_stop_seconds":120},
 "providers":[{"id":f"fixture{i}"} for i in range(9)]},open(sys.argv[1],"w"))
PY
tar -C "$TMP/staged" -czf "$TMP/release/$ASSET" AhB
(cd "$TMP/release" && sha256sum "$ASSET" > "$ASSET.sha256")

APP="$TMP/AhB"
mkdir -p "$APP/scripts" "$APP/bin" "$APP/data/freebuff/gateway"
printf '#!/bin/sh\necho old-release\n' > "$APP/bin/hubd"
chmod +x "$APP/bin/hubd"
cp "$ROOT/scripts/stop-linux.sh" "$ROOT/scripts/install-prebuilt-linux.sh" "$APP/scripts/"
printf '#!/usr/bin/env bash\nexit 0\n' > "$APP/scripts/prepare-configs.sh"
printf 'old-bundled-server\n' > "$APP/data/freebuff/gateway/server.js"
printf 'do-not-delete-real-accounts\n' > "$APP/data/account.fixture"
printf '%s\n' '{"listen":"127.0.0.1:18477","unknown_settings":{"keep":true},"providers":[]}' > "$APP/config.json"

export AHB_LINUX_PREBUILT_BASE="file://$TMP/release"
export AHB_LINUX_INSTALL_DIR="$APP"
bash "$ROOT/scripts/upgrade-linux.sh"
grep -Fx 'do-not-delete-real-accounts' "$APP/data/account.fixture"
grep -Fx 'new-bundled-server' "$APP/data/freebuff/gateway/server.js"
jq -e '.unknown_settings.keep == true' "$APP/config.json" >/dev/null
grep -Fx 'new-release' <("$APP/bin/hubd")
backup=( "$TMP"/AhB.backup-* )
test "${#backup[@]}" -eq 1
grep -Fx 'old-release' <("${backup[0]}/bin/hubd")
grep -Fx 'do-not-delete-real-accounts' "${backup[0]}/data/account.fixture"

# Corrupt remote release; second upgrade must abort before stopping or
# mutating the working installation, and the original user config stays.
printf '0000000000000000000000000000000000000000000000000000000000000000  %s\n' "$ASSET" > "$TMP/release/$ASSET.sha256"
if bash "$ROOT/scripts/upgrade-linux.sh" >"$TMP/error.log" 2>&1; then
 echo "corrupt native update was accepted" >&2
 exit 1
fi
grep -Fx 'new-release' <("$APP/bin/hubd")
grep -Fx 'do-not-delete-real-accounts' "$APP/data/account.fixture"
jq -e '.unknown_settings.keep == true' "$APP/config.json" >/dev/null
echo "Linux native upgrade fixture: preserved accounts, replaced only bundled static data, kept rollback, rejected corruption"
