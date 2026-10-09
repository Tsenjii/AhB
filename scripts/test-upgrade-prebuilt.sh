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
# Simulate a real released bundle: all published AhB archives include the
# stop helper, even when the old installation was a three-provider release.
cat > "$NEW/scripts/stop-termux.sh" <<'EOF'
#!/usr/bin/env bash
# Fixture implements the allowlisted AhB stop-helper contract.
collect_ahb_pids() { :; }
exit 0
EOF
chmod +x "$NEW/scripts/stop-termux.sh"
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
if ! bash "$ROOT/scripts/upgrade-prebuilt-termux.sh" > "$TMP/output.txt" 2>&1; then
  cat "$TMP/output.txt" >&2
  exit 1
fi

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

# Regression: known 2026-10-09 published AhB release has a corrupted stop
# helper (bash parse error at Kimi case). Verify recovery from a SHA-checked
# replacement, full account preservation, and an independently kept old tree.
RECOVERY="$TMP/recovery/AhB"
mkdir -p "$TMP/recovery"
cp -a "$OLD" "$RECOVERY"
cat > "$RECOVERY/scripts/stop-termux.sh" <<'EOF'
#!/usr/bin/env bash
collect_ahb_pids() {
  case "$cmdline" in
      "$ROOT/data/kimiweb/venv/bin/python"
EOF
if bash -n "$RECOVERY/scripts/stop-termux.sh" >/dev/null 2>&1; then
  echo "expected malformed published stop helper" >&2
  exit 1
fi
if ! AIHUB_INSTALL_DIR="$RECOVERY" bash "$ROOT/scripts/upgrade-prebuilt-termux.sh" > "$TMP/recovery-success.log" 2>&1; then
  cat "$TMP/recovery-success.log" >&2
  exit 1
fi
bash -n "$RECOVERY/scripts/stop-termux.sh"
test "$(cat "$RECOVERY/data/opencode/account.txt")" = "secret account state"
test "$(cat "$RECOVERY/data/freebuff/tokens.json")" = "old private cookie"
shopt -s nullglob
recovery_backups=("$TMP/recovery/AhB.backup-"*)
test "${#recovery_backups[@]}" -eq 1
test "$(cat "${recovery_backups[0]}/data/opencode/account.txt")" = "secret account state"
test "$(find "${recovery_backups[0]}/scripts" -maxdepth 1 -name 'stop-termux.sh.corrupt-*' | wc -l)" -eq 1
# Unknown corruption must fail closed without modifying user config or data.
UNKNOWN="$TMP/unknown/AhB"
mkdir -p "$TMP/unknown"
cp -a "$RECOVERY" "$UNKNOWN"
printf 'case broken in\n' > "$UNKNOWN/scripts/stop-termux.sh"
if AIHUB_INSTALL_DIR="$UNKNOWN" bash "$ROOT/scripts/upgrade-prebuilt-termux.sh" > "$TMP/unknown-rejected.log" 2>&1; then
  echo "expected unknown broken helper to be rejected" >&2
  exit 1
fi
test "$(cat "$UNKNOWN/data/opencode/account.txt")" = "secret account state"
grep -q "unknown invalid legacy" "$TMP/unknown-rejected.log"

# A checksum failure must leave an existing install untouched and running.
rm -f "$AIHUB_TEST_STOP_MARKER"
printf 'corrupted checksum\n' > "$TMP/assets/AhB_android_arm64.tar.gz.sha256"
if bash "$ROOT/scripts/upgrade-prebuilt-termux.sh" > "$TMP/failed.txt" 2>&1; then
  echo "expected invalid checksum to fail" >&2
  exit 1
fi
test ! -e "$AIHUB_TEST_STOP_MARKER"
test "$(cat "$OLD/data/opencode/account.txt")" = "secret account state"
# A pre-AhB or damaged install with no stop helper must abort safely,
# rather than copying account DBs from possibly still-running processes.
(
  cd "$TMP/assets"
  sha256sum AhB_android_arm64.tar.gz > AhB_android_arm64.tar.gz.sha256
)
mv "$OLD/scripts/stop-termux.sh" "$OLD/scripts/stop-termux.sh.temporarily-missing"
rm -f "$AIHUB_TEST_STOP_MARKER"
if bash "$ROOT/scripts/upgrade-prebuilt-termux.sh" > "$TMP/missing-stop.txt" 2>&1; then
  echo "expected unrecognized old layout to fail before replacing original data" >&2
  exit 1
fi
test ! -e "$AIHUB_TEST_STOP_MARKER"
test -f "$OLD/config.json"
test "$(cat "$OLD/data/opencode/account.txt")" = "secret account state"
mv "$OLD/scripts/stop-termux.sh.temporarily-missing" "$OLD/scripts/stop-termux.sh"

# Regression: a legacy stop script can exit 0 while a foreground hub remains
# alive (no PID file). The upgrader must abort before copying live accounts.
# Preserve the checked backup under a different name so another attempt in
# the same CI clock second does not trip duplicate-backup-name protection.
mv "${backups[0]}" "${backups[0]}.verified-previous-backup"
LIVE_PORT=18819
jq --arg listen "127.0.0.1:$LIVE_PORT" '.listen=$listen' "$OLD/config.json" > "$TMP/live-config.json"
mv "$TMP/live-config.json" "$OLD/config.json"
python3 -m http.server "$LIVE_PORT" --bind 127.0.0.1 > "$TMP/live-server.log" 2>&1 &
live_pid=$!
trap 'kill "$live_pid" 2>/dev/null || true; rm -rf "$TMP"' EXIT
listening=0
for attempt in 1 2 3 4 5 6 7 8 9 10; do
  if (exec 3<>"/dev/tcp/127.0.0.1/$LIVE_PORT") 2>/dev/null; then
    listening=1; break
  fi
  sleep .1
done
if [ "$listening" != 1 ]; then
  echo "mock listener did not start" >&2
  exit 1
fi
if bash "$ROOT/scripts/upgrade-prebuilt-termux.sh" > "$TMP/live-rejected.txt" 2>&1; then
  echo "expected the upgrade to refuse a live foreground listener" >&2
  exit 1
fi
if ! grep -q "listeners still active" "$TMP/live-rejected.txt"; then
  cat "$TMP/live-rejected.txt" >&2
  exit 1
fi
test "$(cat "$OLD/data/opencode/account.txt")" = "secret account state"
test "$(jq -r '.listen' "$OLD/config.json")" = "127.0.0.1:$LIVE_PORT"
kill "$live_pid" 2>/dev/null || true
wait "$live_pid" 2>/dev/null || true
trap 'rm -rf "$TMP"' EXIT

echo "upgrade correctly refused active legacy frontend; original data preserved"
echo "upgrade-prebuilt fixture tests passed"
