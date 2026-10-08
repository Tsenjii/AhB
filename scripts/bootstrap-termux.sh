#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BUILD_DIR="$ROOT/.build"
BIN_DIR="$ROOT/bin"

echo "== Android AI Hub / Termux bootstrap =="
echo "repo: $ROOT"

if ! command -v pkg >/dev/null 2>&1; then
  echo "This script must run inside Termux."
  exit 1
fi

pkg install -y git curl golang rust clang make pkg-config coreutils tar

mkdir -p "$BUILD_DIR" "$BIN_DIR" "$ROOT/data/opencode" "$ROOT/data/freebuff" "$ROOT/data/agent2api" "$ROOT/logs"

build_or_update() {
  local dir="$1"
  local repo_url="$2"
  local ref="$3"
  if [ ! -d "$dir/.git" ]; then
    git clone --depth 1 --branch "$ref" "$repo_url" "$dir"
  else
    git -C "$dir" fetch --depth 1 origin "refs/tags/$ref:refs/tags/$ref"
    git -C "$dir" checkout --detach "$ref"
    git -C "$dir" reset --hard "$ref"
  fi
}

echo "== Building hubd =="
cd "$ROOT"
go build -trimpath -ldflags="-s -w" -o "$BIN_DIR/hubd" ./cmd/hubd

echo "== Installing opencode2api =="
OPENCODE_VERSION="v1.3.7"
case "$(uname -m)" in
  aarch64|arm64)
    OC_PACKAGE="opencode2api_${OPENCODE_VERSION}_linux_arm64"
    OC_TMP="$BUILD_DIR/opencode-release"
    rm -rf "$OC_TMP"
    mkdir -p "$OC_TMP"
    curl -fL --retry 3 -o "$OC_TMP/$OC_PACKAGE.tar.gz" \
      "https://github.com/jasonxu114514/opencode2api/releases/download/$OPENCODE_VERSION/$OC_PACKAGE.tar.gz"
    curl -fL --retry 3 -o "$OC_TMP/$OC_PACKAGE.tar.gz.sha256" \
      "https://github.com/jasonxu114514/opencode2api/releases/download/$OPENCODE_VERSION/$OC_PACKAGE.tar.gz.sha256"
    (
      cd "$OC_TMP"
      sha256sum -c "$OC_PACKAGE.tar.gz.sha256"
      tar -xzf "$OC_PACKAGE.tar.gz"
    )
    cp "$OC_TMP/$OC_PACKAGE/opencode2api" "$BIN_DIR/opencode2api"
    chmod 700 "$BIN_DIR/opencode2api"
    if ! "$BIN_DIR/opencode2api" -h >/dev/null 2>&1; then
      echo "Pinned Linux ARM64 binary cannot execute in Termux; building from source instead."
      rm -f "$BIN_DIR/opencode2api"
      build_or_update "$BUILD_DIR/opencode2api" "https://github.com/jasonxu114514/opencode2api.git" "$OPENCODE_VERSION"
      cd "$BUILD_DIR/opencode2api"
      go build -trimpath -ldflags="-s -w" -o "$BIN_DIR/opencode2api" ./cmd/opencode2api
    fi
    ;;
  *)
    echo "No pinned prebuilt opencode2api for $(uname -m); building from source."
    build_or_update "$BUILD_DIR/opencode2api" "https://github.com/jasonxu114514/opencode2api.git" "$OPENCODE_VERSION"
    cd "$BUILD_DIR/opencode2api"
    go build -trimpath -ldflags="-s -w" -o "$BIN_DIR/opencode2api" ./cmd/opencode2api
    ;;
esac

echo "== Building Freebuff2API =="
build_or_update "$BUILD_DIR/freebuff2api" "https://github.com/lza6/Freebuff-2API.git" "v0.10.3"
cd "$BUILD_DIR/freebuff2api"
CARGO_BUILD_JOBS="${AIHUB_CARGO_JOBS:-2}" \
CARGO_PROFILE_RELEASE_LTO=false \
CARGO_PROFILE_RELEASE_CODEGEN_UNITS=8 \
cargo build --locked --release
cp target/release/freebuff2api "$BIN_DIR/freebuff2api"
strip "$BIN_DIR/freebuff2api" 2>/dev/null || true

cd "$ROOT"
chmod +x scripts/prepare-configs.sh
./scripts/prepare-configs.sh
chmod 700 "$BIN_DIR/hubd" "$BIN_DIR/opencode2api" "$BIN_DIR/freebuff2api"

echo
echo "Bootstrap complete."
echo "Hub UI:       http://127.0.0.1:8317/ui"
echo "OpenCode UI:  http://127.0.0.1:8404/"
echo "FreeBuff UI:  http://127.0.0.1:8402/ui"
echo "OpenCode UI password is stored in: data/opencode/webui-password.txt"
echo
if [ "${AIHUB_WITH_AGENT2API:-0}" = "1" ]; then
  echo "== Installing optional Agent2API provider pack =="
  "$ROOT/scripts/install-agent2api-termux.sh"
fi

echo "Optional multi-provider sidecar:"
echo "  ./scripts/install-agent2api-termux.sh"
echo "or install everything in one pass:"
echo "  AIHUB_WITH_AGENT2API=1 ./scripts/bootstrap-termux.sh"
echo
echo "Start:"
echo "  ./scripts/run-termux.sh"