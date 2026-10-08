#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

REPO="Tsenjii/AhB"
TAG="dev-latest"
ASSET="AhB_android_arm64.tar.gz"
BASE="https://github.com/$REPO/releases/download/$TAG"
DEST="${AIHUB_INSTALL_DIR:-$HOME/AhB}"
TMP="$(mktemp -d)"

cleanup(){ rm -rf "$TMP"; }
trap cleanup EXIT

pkg install -y curl coreutils tar

echo "Downloading AhB Android ARM64 bundle..."
curl -fL --retry 3 -o "$TMP/$ASSET" "$BASE/$ASSET"
curl -fL --retry 3 -o "$TMP/$ASSET.sha256" "$BASE/$ASSET.sha256"

(
  cd "$TMP"
  sha256sum -c "$ASSET.sha256"
  tar -xzf "$ASSET"
)

if [ -e "$DEST" ]; then
  echo "Destination already exists: $DEST"
  echo "Move/remove it first, or set AIHUB_INSTALL_DIR to another path."
  exit 1
fi

mv "$TMP/AhB" "$DEST"
chmod +x "$DEST"/scripts/*.sh "$DEST"/bin/*

cd "$DEST"
./scripts/prepare-configs.sh

echo
echo "Installed to: $DEST"
echo "Start with:"
echo "  cd $DEST && ./scripts/run-termux.sh"
echo
echo "Hub UI:"
echo "  http://127.0.0.1:8317/ui"
