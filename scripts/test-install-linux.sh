#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

command -v node >/dev/null || { echo "node runtime required" >&2; exit 1; }
mkdir -p "$TMP/stage/AhB/bin" "$TMP/release"
printf '#!/usr/bin/env bash\necho fixture\n' > "$TMP/stage/AhB/bin/hubd"
chmod +x "$TMP/stage/AhB/bin/hubd"
printf '%s\n' '{"resources":{"max_running_sidecars":1,"idle_stop_seconds":120},"providers":[]}' > "$TMP/stage/AhB/config.example.json"
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64;;
  aarch64|arm64) ARCH=arm64;;
  *) echo "unsupported CI architecture" >&2; exit 1;;
esac
ASSET="AhB_linux_$ARCH.tar.gz"
tar -C "$TMP/stage" -czf "$TMP/release/$ASSET" AhB
(cd "$TMP/release" && sha256sum "$ASSET" > "$ASSET.sha256")
export AHB_LINUX_PREBUILT_BASE="file://$TMP/release"
export AHB_LINUX_INSTALL_DIR="$TMP/installed"

bash "$ROOT/scripts/install-prebuilt-linux.sh"
[ -x "$AHB_LINUX_INSTALL_DIR/bin/hubd" ]
[ -r "$AHB_LINUX_INSTALL_DIR/config.example.json" ]

# An already installed copy, including accounts, cannot be silently replaced.
mkdir -p "$AHB_LINUX_INSTALL_DIR/data"
printf 'keep-this-account\n' > "$AHB_LINUX_INSTALL_DIR/data/account.fixture"
if bash "$ROOT/scripts/install-prebuilt-linux.sh" >"$TMP/reinstall.log" 2>&1; then
  echo "installer overwrote an existing installation" >&2
  exit 1
fi
grep -Fx 'keep-this-account' "$AHB_LINUX_INSTALL_DIR/data/account.fixture"

# A damaged archive/checksum must fail before mutating any destination.
export AHB_LINUX_INSTALL_DIR="$TMP/corrupt-target"
printf 'bad sha256  AhB_linux_%s.tar.gz\n' "$ARCH" > "$TMP/release/$ASSET.sha256"
if bash "$ROOT/scripts/install-prebuilt-linux.sh" > "$TMP/corrupt.log" 2>&1; then
  echo "installer accepted a damaged checksum" >&2
  exit 1
fi
[ ! -e "$AHB_LINUX_INSTALL_DIR" ]
(cd "$TMP/release" && sha256sum "$ASSET" > "$ASSET.sha256")

# Reject symlinks/hardlinks or traversal in an otherwise checksum-valid archive.
python3 - "$TMP/release/$ASSET" <<'PY'
import io,sys,tarfile
with tarfile.open(sys.argv[1],"w:gz") as tf:
    good=tarfile.TarInfo("AhB/bin/hubd")
    good.mode=0o755
    payload=b"#!/usr/bin/env bash\n"
    good.size=len(payload)
    tf.addfile(good,io.BytesIO(payload))
    bad=tarfile.TarInfo("AhB/data/credentials")
    bad.type=tarfile.SYMTYPE
    bad.linkname="/tmp/not-owned"
    tf.addfile(bad)
PY
(cd "$TMP/release" && sha256sum "$ASSET" > "$ASSET.sha256")
export AHB_LINUX_INSTALL_DIR="$TMP/link-target"
if bash "$ROOT/scripts/install-prebuilt-linux.sh" >"$TMP/link.log" 2>&1; then
  echo "installer accepted archive symlink" >&2
  exit 1
fi
[ ! -e "$AHB_LINUX_INSTALL_DIR" ]
echo "Linux first-install fixture: checksum, no-overwrite, and unsafe archive checks passed"
