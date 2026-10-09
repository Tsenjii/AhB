#!/usr/bin/env bash
set -euo pipefail
umask 077
# Native Linux upgrade: stage and checksum first, then stop safely, take an
# offline account/SQLite snapshot, and preserve the old tree for rollback.
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
DEST="${AHB_LINUX_INSTALL_DIR:-$ROOT}"
[ "$(uname -s)" = Linux ] || { echo "Native Linux required" >&2; exit 1; }
[ -d "$DEST" ] && [ ! -L "$DEST" ] && [ -f "$DEST/config.json" ] && [ -d "$DEST/data" ] || {
  echo "Existing Linux AhB account install required; refusing first-install upgrade" >&2
  exit 1
}
for file in scripts/install-prebuilt-linux.sh scripts/stop-linux.sh scripts/prepare-configs.sh; do
  [ -r "$DEST/$file" ] || { echo "Missing original service helper: $file" >&2; exit 1; }
  bash -n "$DEST/$file"
done
for cmd in curl tar sha256sum jq node python3 readlink; do
  command -v "$cmd" >/dev/null || { echo "Missing $cmd" >&2; exit 1; }
done
jq -e '.providers | type == "array"' "$DEST/config.json" >/dev/null
PARENT="$(cd "$(dirname "$DEST")" && pwd -P)"
NAME="$(basename "$DEST")"
DEST="$PARENT/$NAME"
TMP="$(mktemp -d "$PARENT/.ahb-linux-upgrade.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading and checksum-verifying new native Linux release..."
# Only the bundled, reviewed installer can select a Linux-only release. This
# runs before old Hub shutdown, so network failures never touch live accounts.
AHB_LINUX_INSTALL_DIR="$TMP/new/AhB" bash "$DEST/scripts/install-prebuilt-linux.sh"
NEW="$TMP/new/AhB"
for file in bin/hubd scripts/prepare-configs.sh scripts/stop-linux.sh scripts/start-linux.sh config.example.json; do
  [ -e "$NEW/$file" ] || { echo "Incomplete Linux bundle: $file" >&2; exit 1; }
done
jq -e '.resources.max_running_sidecars >= 1 and (.providers|length) >= 7' "$NEW/config.example.json" >/dev/null
bash -n "$NEW/scripts/"*.sh
jq -e '.providers | type == "array"' "$DEST/config.json" >/dev/null

echo "Stopping old AhB before snapshotting account databases..."
bash "$DEST/scripts/stop-linux.sh"
# Stop-linux refuses unrelated/stale PID files. Never guess or kill a process
# just because it happens to share a binary basename.
for attempt in 1 2 3 4 5 6 7 8 9 10; do
  busy=0
  for path in /proc/[0-9]*/exe; do
    [ -e "$path" ] || continue
    case "$(readlink "$path" 2>/dev/null || :)" in
      "$DEST"/bin/*) busy=1; break;;
    esac
  done
  # Node FreeBuff has a generic 'node' executable and must be checked via its
  # unique installed script path, not by matching every Node.js process.
  if [ "$busy" -eq 0 ]; then
    for path in /proc/[0-9]*/cmdline; do
      [ -r "$path" ] || continue
      if tr '\0' ' ' < "$path" 2>/dev/null | grep -Fq "$DEST/data/freebuff/gateway/server.js"; then
        busy=1
        break
      fi
    done
  fi
  [ "$busy" -eq 0 ] && break
  sleep 1
done
if [ "$busy" -ne 0 ]; then
  echo "Old AhB processes still running; refusing to snapshot live accounts" >&2
  exit 1
fi
# Local TCP bound by an orphan sidecar/Hub? Fail closed, even if the listener
# belongs to a different process: do not snapshot while ownership is uncertain.
mapfile -t ports < <(jq -r '
  [(.listen // "127.0.0.1:8317"),
   (.providers[]? | select(.enabled == true and .kind == "sidecar") | .base_url // empty)] | .[] | strings
  | (capture(":(?<port>[0-9]+)(/|$)")? | .port)
 ' "$DEST/config.json" | sort -nu)
for port in "${ports[@]}"; do
  if (exec 3<>"/dev/tcp/127.0.0.1/$port") 2>/dev/null; then
    echo "Port $port still bound after shutdown. Old files remain untouched." >&2
    exit 1
  fi
done

# Keep the new version's packaged frontend/client code, not previous bundled
# assets masquerading as account data. Preserve all credential DBs and user
# configuration by copying their old data tree into the new staging tree.
ASSETS=(
 data/freebuff/gateway
 data/agent2api/ui
 data/deepseek2api/static/admin
 data/grok2api/frontend/dist
 data/kiro-go/web
)
mkdir -p "$TMP/static"
for path in "${ASSETS[@]}"; do
  if [ -d "$NEW/$path" ]; then
    mkdir -p "$TMP/static/$(dirname "$path")"
    cp -a "$NEW/$path" "$TMP/static/$path"
  fi
done
mkdir -p "$NEW/data" "$NEW/logs"
cp -a "$DEST/data/." "$NEW/data/"
cp -a "$DEST/config.json" "$NEW/config.json"
if [ -d "$DEST/logs" ]; then cp -a "$DEST/logs/." "$NEW/logs/"; fi
for path in "${ASSETS[@]}"; do
  if [ -d "$TMP/static/$path" ]; then
    rm -rf "$NEW/$path"
    mkdir -p "$NEW/$(dirname "$path")"
    cp -a "$TMP/static/$path" "$NEW/$path"
  fi
done
(
  cd "$NEW"
  bash scripts/prepare-configs.sh
  jq -e '.providers | type == "array"' config.json >/dev/null
)

BACKUP="$PARENT/$NAME.backup-$(date +%Y%m%d-%H%M%S)"
[ ! -e "$BACKUP" ] || { echo "Backup path exists, refusing upgrade" >&2; exit 1; }
mv "$DEST" "$BACKUP"
if ! mv "$NEW" "$DEST"; then
  echo "Activation failed. Restoring previous AhB..." >&2
  mv "$BACKUP" "$DEST"
  exit 1
fi
echo "Linux AhB upgraded successfully. Existing accounts preserved."
echo "Rollback backup: $BACKUP"
echo "To start: cd $DEST && bash scripts/start-linux.sh"
