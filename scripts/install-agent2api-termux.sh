#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/.build/agent2api"
BIN_DIR="$ROOT/bin"
VERSION="v2.9.5"

if ! command -v pkg >/dev/null 2>&1; then
  echo "This script must run inside Termux."
  exit 1
fi

pkg install -y git rust clang make pkg-config jq coreutils
mkdir -p "$ROOT/.build" "$BIN_DIR" "$ROOT/data/agent2api"

if [ ! -d "$SRC/.git" ]; then
  git clone --depth 1 --branch "$VERSION" https://github.com/aimod-cc/agent2api.git "$SRC"
else
  git -C "$SRC" fetch --depth 1 origin "refs/tags/$VERSION:refs/tags/$VERSION"
  git -C "$SRC" checkout --detach "$VERSION"
  git -C "$SRC" reset --hard "$VERSION"
fi

echo "== Building Agent2API headless server =="
cd "$SRC/desktop-tauri/src-tauri"
CARGO_BUILD_JOBS="${AIHUB_CARGO_JOBS:-2}" \
CARGO_PROFILE_RELEASE_LTO=false \
CARGO_PROFILE_RELEASE_CODEGEN_UNITS=8 \
cargo build --locked --release -p agent2api-server --bin agent2api-server

cp "$SRC/desktop-tauri/target/release/agent2api-server" "$BIN_DIR/agent2api-server"
chmod 700 "$BIN_DIR/agent2api-server"

rm -rf "$ROOT/data/agent2api/ui"
cp -R "$SRC/desktop-tauri/ui" "$ROOT/data/agent2api/ui"

cd "$ROOT"
chmod +x scripts/prepare-configs.sh
./scripts/prepare-configs.sh

tmp="$(mktemp)"
jq '(.providers[] | select(.id == "agent2api") | .enabled) = true' config.json > "$tmp"
mv "$tmp" config.json
chmod 600 config.json

echo
echo "Agent2API installed and enabled."
echo "Restart hubd, then open:"
echo "  http://127.0.0.1:8403/"
echo
echo "Supported personal channels include CodeArts, Qoder, Cline, Trae, Loomy and others exposed by Agent2API."