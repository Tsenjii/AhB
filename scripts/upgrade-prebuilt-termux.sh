#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

# Upgrade an existing AhB prebuilt install without deleting local accounts,
# credentials, settings, or provider databases. The old tree remains as a backup.
REPO="Tsenjii/AhB"
BASE="https://raw.githubusercontent.com/$REPO/prebuilt"
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
case "$(uname -m)" in
  aarch64|arm64) ;;
  *) echo "Android ARM64 prebuilt required; detected $(uname -m)"; exit 1 ;;
esac

pkg install -y curl coreutils tar jq

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

for path in bin/hubd bin/opencode2api bin/freebuff2api scripts/prepare-configs.sh config.example.json; do
  if [ ! -e "$NEW/$path" ]; then
    echo "Incomplete prebuilt archive: missing $path"
    exit 1
  fi
done

# Preserve local user state. Packaged frontends are then refreshed from the
# downloaded archive so stale WebUI files cannot overwrite the new version.
cp -a "$DEST/data/." "$NEW/data/"
cp -a "$DEST/config.json" "$NEW/config.json"
if [ -d "$DEST/logs" ]; then
  mkdir -p "$NEW/logs"
  cp -a "$DEST/logs/." "$NEW/logs/"
fi

assets=(
  AhB/data/agent2api/ui
  AhB/data/deepseek2api/static/admin
  AhB/data/grok2api/frontend/dist
  AhB/data/kiro-go/web
)
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

BACKUP="$PARENT/$NAME.backup-$(date +%Y%m%d-%H%M%S)"
if [ -e "$BACKUP" ]; then
  echo "Backup directory already exists: $BACKUP"
  exit 1
fi

echo "Stopping the old AhB before switching installations..."
if [ -x "$DEST/scripts/stop-termux.sh" ]; then
  "$DEST/scripts/stop-termux.sh"
fi

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
