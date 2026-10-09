#!/usr/bin/env bash
set -euo pipefail
# First installation only: never overwrite a user's config or accounts.
REPO="Tsenjii/AhB"
BASE="${AHB_LINUX_PREBUILT_BASE:-https://github.com/$REPO/releases/latest/download}"
[ "$(uname -s)" = Linux ] || { echo "Linux required, not Android Termux" >&2; exit 1; }
case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64";;
  aarch64|arm64) ARCH="arm64";;
  *) echo "Unsupported Linux CPU architecture" >&2; exit 1;;
esac
DEST="${AHB_LINUX_INSTALL_DIR:-$HOME/AhB}"
[ ! -e "$DEST" ] || { echo "Existing install detected at $DEST; refusing to overwrite" >&2; exit 1; }
for cmd in curl tar sha256sum jq node python3; do command -v "$cmd" >/dev/null || { echo "Missing $cmd" >&2; exit 1; }; done
ASSET="AhB_linux_$ARCH.tar.gz"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
curl -fsSL --retry 3 "$BASE/$ASSET" -o "$TMP/$ASSET"
curl -fsSL --retry 3 "$BASE/$ASSET.sha256" -o "$TMP/$ASSET.sha256"
(cd "$TMP" && sha256sum -c "$ASSET.sha256")
# Reject symlink/hardlink archive entries, traversal paths and absolute paths
# before extracting as a defense in depth for host filesystem safety.
python3 - "$TMP/$ASSET" <<'PY'
import sys, tarfile, pathlib
with tarfile.open(sys.argv[1], "r:gz") as archive:
    for member in archive:
        path = pathlib.PurePosixPath(member.name)
        if member.name.startswith("/") or ".." in path.parts or path.parts[0] != "AhB":
            raise SystemExit("unsafe tar path")
        if not (member.isfile() or member.isdir()):
            raise SystemExit("unsupported tar entry type")
PY
mkdir -p "$TMP/extract"
tar -xzf "$TMP/$ASSET" -C "$TMP/extract" --no-same-owner
[ -x "$TMP/extract/AhB/bin/hubd" ] || { echo "Incomplete AhB Linux bundle" >&2; exit 1; }
jq -e '.resources.max_running_sidecars==1 and .resources.idle_stop_seconds==120' "$TMP/extract/AhB/config.example.json" >/dev/null
mkdir -p "$(dirname "$DEST")"
mv "$TMP/extract/AhB" "$DEST"
echo "Installed Linux AhB at $DEST; start: cd $DEST && bash scripts/start-linux.sh"
