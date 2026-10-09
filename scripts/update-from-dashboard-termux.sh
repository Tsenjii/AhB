#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077

# Launched detached by the local AhB dashboard. This only upgrades the
# full, signed-off AhB Android prebuilt; individual sidecars are bundled.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
old_pid="${1:-}"
if ! [[ "$old_pid" =~ ^[1-9][0-9]*$ ]]; then
  echo "Refusing update: invalid current AhB PID" >&2
  exit 2
fi
sleep 2
old_exe="$(readlink "/proc/$old_pid/exe" 2>/dev/null || true)"
if [ "$old_exe" != "$ROOT/bin/hubd" ]; then
  echo "Refusing update: target is not this AhB" >&2
  exit 1
fi
BASE="https://raw.githubusercontent.com/Tsenjii/AhB"
PUBLISHED="$(curl -fsSL --max-time 25 "$BASE/prebuilt/source-commit.txt" | tr -d '\r\n')"
if ! [[ "$PUBLISHED" =~ ^[a-f0-9]{40}$ ]]; then
  echo "Invalid published package revision; old installation unchanged" >&2
  exit 1
fi
CURRENT=""
if [ -f "$ROOT/.build-commit" ]; then
  CURRENT="$(tr -d '\r\n' < "$ROOT/.build-commit")"
fi
echo "Installed AhB: ${CURRENT:-unknown}; published: $PUBLISHED"
if [ "$CURRENT" = "$PUBLISHED" ]; then
  echo "Already on latest published AhB. No changes made."
  exit 0
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
# Download the updater from the *exact* published package commit, not from
# the moving main branch. Its normal checksum verification and backup logic
# must complete before we allow a new daemon to start.
curl -fsSL --max-time 25 "$BASE/$PUBLISHED/scripts/upgrade-prebuilt-termux.sh" -o "$TMP/upgrade.sh"
bash -n "$TMP/upgrade.sh"
unset AIHUB_PREBUILT_BASE AIHUB_INSTALL_DIR
echo "Updating AhB package (including bundled Agent2API) with rollback backup."
bash "$TMP/upgrade.sh"
echo "New package installed. Starting only the AhB daemon..."
cd "$ROOT"
bash "$ROOT/scripts/start-termux.sh"
echo "Dashboard update complete: $PUBLISHED"
