#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

mkdir -p "$TMP/shims" "$TMP/assets" "$TMP/source/AhB"/{bin,scripts,data,logs} "$TMP/installed/AhB"/{bin,scripts,data}

cat > "$TMP/shims/uname" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-}" = "-m" ]; then printf 'aarch64\n'; else /usr/bin/uname "$@"; fi
EOF
cat > "$TMP/shims/pkg" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "$TMP/shims/"*

NEW="$TMP/source/AhB"
for name in hubd opencode2api freebuff2api; do
  printf '#!/usr/bin/env bash\nexit 0\n' > "$NEW/bin/$name"
  chmod +x "$NEW/bin/$name"
done
cat > "$NEW/scripts/prepare-configs.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p data
printf 'ready\n' > data/migration-tested.txt
EOF
chmod +x "$NEW/scripts/prepare-configs.sh"
printf '{"providers":[{"id":"new"}]}\n' > "$NEW/config.example.json"

for path in data/freebuff/gateway data/agent2api/ui data/deepseek2api/static/admin data/grok2api/frontend/dist data/kiro-go/web; do
  mkdir -p "$NEW/$path"
  printf 'new assets\n' > "$NEW/$path/index.html"
  if [ "$path" = "data/freebuff/gateway" ]; then
    printf '// mock node gateway\n' > "$NEW/$path/server.js"
    printf '// mock worker\n' > "$NEW/$path/worker.js"
  fi
done

(
  cd "$TMP/source"
  tar -czf "$TMP/assets/AhB_android_arm64.tar.gz" AhB
)
(
  cd "$TMP/assets"
  sha256sum AhB_android_arm64.tar.gz > AhB_android_arm64.tar.gz.sha256
)

OLD="$TMP/installed/AhB"
printf '{"providers":[{"id":"existing"}]}\n' > "$OLD/config.json"
mkdir -p "$OLD/data/opencode" "$OLD/data/freebuff/gateway" "$OLD/data/agent2api/ui"
printf 'secret account state\n' > "$OLD/data/opencode/account.txt"
printf 'old assets\n' > "$OLD/data/agent2api/ui/index.html"
printf 'old gateway\n' > "$OLD/data/freebuff/gateway/index.html"
printf 'old private cookie\n' > "$OLD/data/freebuff/tokens.json"
cat > "$OLD/scripts/stop-termux.sh" <<'EOF'
#!/usr/bin/env bash
printf 'stopped\n' > "$AIHUB_TEST_STOP_MARKER"
# Simulate a final sidecar database flush during graceful shutdown.
printf 'final transaction\n' > "$AIHUB_INSTALL_DIR/data/opencode/last-transaction.txt"
EOF
chmod +x "$OLD/scripts/stop-termux.sh"

export PATH="$TMP/shims:$PATH"
export AIHUB_PREBUILT_BASE="file://$TMP/assets"
export AIHUB_INSTALL_DIR="$OLD"
export AIHUB_TEST_STOP_MARKER="$TMP/stopped.marker"
bash "$ROOT/scripts/upgrade-prebuilt-termux.sh" > "$TMP/output.txt"

test -e "$AIHUB_TEST_STOP_MARKER"
test "$(cat "$OLD/data/opencode/account.txt")" = "secret account state"
test "$(cat "$OLD/data/opencode/last-transaction.txt")" = "final transaction"
test "$(cat "$OLD/data/agent2api/ui/index.html")" = "new assets"
test "$(cat "$OLD/data/freebuff/gateway/index.html")" = "new assets"
test "$(cat "$OLD/data/freebuff/tokens.json")" = "old private cookie"
test "$(cat "$OLD/data/migration-tested.txt")" = "ready"
test "$(jq -r '.providers[0].id' "$OLD/config.json")" = "existing"

shopt -s nullglob
backups=("$TMP/installed/AhB.backup-"*)
test "${#backups[@]}" -eq 1
test "$(cat "${backups[0]}/data/agent2api/ui/index.html")" = "old assets"
test "$(cat "${backups[0]}/data/freebuff/gateway/index.html")" = "old gateway"
test "$(cat "${backups[0]}/data/opencode/account.txt")" = "secret account state"

# A checksum failure must leave an existing install untouched and running.
rm -f "$AIHUB_TEST_STOP_MARKER"
printf 'corrupted checksum\n' > "$TMP/assets/AhB_android_arm64.tar.gz.sha256"
if bash "$ROOT/scripts/upgrade-prebuilt-termux.sh" > "$TMP/failed.txt" 2>&1; then
  echo "expected invalid checksum to fail" >&2
  exit 1
fi
test ! -e "$AIHUB_TEST_STOP_MARKER"
test "$(cat "$OLD/data/opencode/account.txt")" = "secret account state"
echo "upgrade-prebuilt fixture tests passed"
