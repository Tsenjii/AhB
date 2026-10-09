#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

# Upgrade an existing AhB prebuilt install without deleting local accounts,
# credentials, settings, or provider databases. The old tree remains as a backup.
REPO="Tsenjii/AhB"
BASE="${AIHUB_PREBUILT_BASE:-https://raw.githubusercontent.com/$REPO/prebuilt}"
ASSET="AhB_android_arm64.tar.gz"
DEST="${AIHUB_INSTALL_DIR:-$HOME/AhB}"

if [ ! -d "$DEST" ] || [ ! -f "$DEST/config.json" ] || [ ! -d "$DEST/data" ]; then
  echo "Existing AhB install not found at $DEST"
  echo "For a first install use scripts/install-prebuilt-termux.sh instead."
  exit 1
fi
if [ -L "$DEST" ]; then
  echo "Refusing to upgrade a symlinked install directory: $DEST"
  exit 1
fi
# Very old/non-AhB layouts must never be snapshotted while services may still
# be writing account databases. The legacy stop helper was present even in
# AhB's earliest public ARM64 installer; unknown layouts fail closed.
if [ ! -f "$DEST/scripts/stop-termux.sh" ] || [ ! -r "$DEST/scripts/stop-termux.sh" ]; then
  echo "Cannot safely upgrade: missing readable $DEST/scripts/stop-termux.sh" >&2
  echo "Old installation left untouched. Check whether this is an older assistant-new project rather than AhB." >&2
  exit 1
fi
# Early AhB Android releases accidentally published a syntactically corrupted
# stop helper. Recognize ONLY that known signature: unknown corruption still
# fails closed, and we never snapshot a live SQLite database.
LEGACY_STOP_REPAIR=0
if ! bash -n "$DEST/scripts/stop-termux.sh" >/dev/null 2>&1; then
  if grep -Fq '      "$ROOT/data/kimiweb/venv/bin/python"' "$DEST/scripts/stop-termux.sh" &&
     grep -Fq 'collect_ahb_pids()' "$DEST/scripts/stop-termux.sh"; then
    LEGACY_STOP_REPAIR=1
    echo "Detected known corrupted AhB stop helper; will recover it ONLY after package verification."
  else
    echo "Cannot safely upgrade: unknown invalid legacy stop script; old installation left untouched." >&2
    exit 1
  fi
fi
case "$(uname -m)" in
  aarch64|arm64) ;;
  *) echo "Android ARM64 prebuilt required; detected $(uname -m)"; exit 1 ;;
esac

pkg install -y curl coreutils tar jq nodejs

PARENT="$(cd "$(dirname "$DEST")" && pwd)"
NAME="$(basename "$DEST")"
DEST="$PARENT/$NAME"
TMP="$(mktemp -d "$PARENT/.ahb-upgrade.XXXXXX")"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

echo "Downloading and verifying the new bundle..."
curl -fL --retry 3 -o "$TMP/$ASSET" "$BASE/$ASSET"
curl -fL --retry 3 -o "$TMP/$ASSET.sha256" "$BASE/$ASSET.sha256"
(
  cd "$TMP"
  sha256sum -c "$ASSET.sha256"
)
mkdir -p "$TMP/stage"
tar -xzf "$TMP/$ASSET" -C "$TMP/stage"
NEW="$TMP/stage/AhB"

for path in bin/hubd bin/opencode2api bin/freebuff2api data/freebuff/gateway/server.js data/freebuff/gateway/worker.js scripts/prepare-configs.sh config.example.json; do
  if [ ! -e "$NEW/$path" ]; then
    echo "Incomplete prebuilt archive: missing $path"
    exit 1
  fi
done

# Validate the incoming package and existing config before stopping services.
# In particular, never copy a live SQLite database or account state: providers
# may be writing to their databases until stop-termux.sh finishes.
jq -e '.providers | type == "array"' "$DEST/config.json" > /dev/null
jq -e '.providers | type == "array"' "$NEW/config.example.json" > /dev/null
bash -n "$NEW/scripts/prepare-configs.sh"
# A package without a valid stop helper must never replace a working install.
if [ ! -f "$NEW/scripts/stop-termux.sh" ] || ! bash -n "$NEW/scripts/stop-termux.sh"; then
  echo "Downloaded AhB package contains an invalid stop helper; no old files changed." >&2
  exit 1
fi
# On the known broken legacy release, restore only the known-good stop helper
# from the checksum-verified archive. Keep the corrupted original for audit
# and rollback; preserve config, keys, accounts and other user files.
if [ "$LEGACY_STOP_REPAIR" = 1 ]; then
  if ! grep -Fq 'collect_ahb_pids()' "$NEW/scripts/stop-termux.sh"; then
    echo "New package stop helper cannot safely repair this legacy install." >&2
    exit 1
  fi
fi

assets=(
  AhB/data/freebuff/gateway
  AhB/data/agent2api/ui
  AhB/data/deepseek2api/static/admin
  AhB/data/grok2api/frontend/dist
  AhB/data/kiro-go/web
)
for path in "${assets[@]}"; do
  if [ ! -d "$TMP/stage/$path" ]; then
    echo "Incomplete prebuilt archive: missing $path"
    exit 1
  fi
done

BACKUP="$PARENT/$NAME.backup-$(date +%Y%m%d-%H%M%S)"
if [ -e "$BACKUP" ]; then
  echo "Backup directory already exists: $BACKUP"
  exit 1
fi

echo "Stopping AhB before taking a consistent account/database snapshot..."
if [ "$LEGACY_STOP_REPAIR" = 1 ]; then
  broken_backup="$DEST/scripts/stop-termux.sh.corrupt-$(date +%Y%m%d-%H%M%S)"
  if [ -e "$broken_backup" ]; then
    echo "Existing damaged-helper backup; aborting: $broken_backup" >&2
    exit 1
  fi
  cp -p "$DEST/scripts/stop-termux.sh" "$broken_backup"
  install -m 700 "$NEW/scripts/stop-termux.sh" "$DEST/scripts/stop-termux.sh"
  echo "Repaired the known corrupted stop helper; preserved original as $(basename "$broken_backup")."
fi
# Invoke with bash so even very old installs with missing executable bit
# are cleanly stopped before account databases are copied.
bash "$DEST/scripts/stop-termux.sh"

# Old stop helpers sometimes checked only hubd.pid; foreground processes
# could continue running after the helper returned success. Check TCP listeners
# before copying any account/SQLite data. External bridges are excluded.
mapfile -t stop_ports < <(
  jq -r '
    [(.listen // "127.0.0.1:8317"),
     (.providers[]? | select(.enabled == true and .kind == "sidecar") | .base_url // empty),
     "127.0.0.1:8404"] | .[] | strings
    | (capture(":(?<port>[0-9]+)(/|$)")? | .port)
  ' "$DEST/config.json" | sort -nu
)
still_bound=""
for attempt in 1 2 3 4 5; do
  still_bound=""
  for port in "${stop_ports[@]}"; do
    if (exec 3<>"/dev/tcp/127.0.0.1/$port") 2>/dev/null; then
      still_bound="$still_bound $port"
    fi
  done
  [ -z "$still_bound" ] && break
  sleep 1
done
if [ -n "$still_bound" ]; then
  echo "ERROR: old AhB listeners still active on local port(s):$still_bound" >&2
  echo "No account data was copied, moved or replaced." >&2
  echo "Stop foreground AhB sessions before retrying." >&2
  exit 1
fi

# Only after the sidecars stop is it safe to copy on-disk account databases.
# The full original install is retained unchanged for rollback.
cp -a "$DEST/data/." "$NEW/data/"
cp -a "$DEST/config.json" "$NEW/config.json"
if [ -d "$DEST/logs" ]; then
  mkdir -p "$NEW/logs"
  cp -a "$DEST/logs/." "$NEW/logs/"
fi

# A historical UI directory from user data must never shadow new bundled UI.
for path in "${assets[@]}"; do
  rm -rf "$TMP/stage/$path"
done
tar -xzf "$TMP/$ASSET" -C "$TMP/stage" "${assets[@]}"

chmod +x "$NEW"/scripts/*.sh "$NEW"/bin/*
(
  cd "$NEW"
  ./scripts/prepare-configs.sh
  jq -e '.providers | type == "array"' config.json > /dev/null
)

mv "$DEST" "$BACKUP"
if ! mv "$NEW" "$DEST"; then
  echo "Failed to activate the new version. Restoring previous installation..."
  mv "$BACKUP" "$DEST"
  exit 1
fi

echo
echo "Upgrade installed: $DEST"
echo "Previous installation retained: $BACKUP"
echo "Local accounts, configuration and databases were copied."
echo "Start and check:"
echo "  cd $DEST"
echo "  ./scripts/run-termux.sh"
echo "Then in a second Termux session:"
echo "  cd $DEST && ./scripts/doctor-termux.sh && ./scripts/smoke.sh"
