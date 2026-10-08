#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

REPO="Tsenjii/AhB"
BRANCH="prebuilt"
ASSET="AhB_android_arm64.tar.gz"
BASE="https://raw.githubusercontent.com/$REPO/$BRANCH"
DEST="${AIHUB_INSTALL_DIR:-$HOME/AhB}"
TMP="$(mktemp -d)"

cleanup(){ rm -rf "$TMP"; }
trap cleanup EXIT

case "$(uname -m)" in
  aarch64|arm64) ;;
  *)
    echo "This prebuilt bundle is ARM64-only. Detected: $(uname -m)"
    exit 1
    ;;
esac

pkg install -y curl coreutils tar nodejs

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

if [ -x ./bin/agent2api-server ] && [ -d ./data/agent2api/ui ]; then
  ./scripts/enable-agent2api.sh
fi

echo
echo "Installed to: $DEST"
echo "Start with:"
echo "  cd $DEST && ./scripts/run-termux.sh"
echo
echo "Hub UI:"
echo "  http://127.0.0.1:8317/ui"
echo
echo "OpenCode UI:"
echo "  http://127.0.0.1:8404/"
echo
echo "FreeBuff CLI login (new Node gateway):"
echo "  cd $DEST && ./scripts/freebuff-login-termux.sh"
echo
echo "Agent2API UI:"
echo "  http://127.0.0.1:8403/"

echo "Optional DeepSeek2API provider:"
echo "  cd $DEST && ./scripts/enable-deepseek2api.sh"
echo "  UI: http://127.0.0.1:8405/admin"
