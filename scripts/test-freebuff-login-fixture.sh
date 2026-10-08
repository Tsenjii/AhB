#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

cat > "$TMP/curl" <<'MOCK'
#!/usr/bin/env bash
printf '%s' "${FAKE_FREEBUFF_STATUS:-200}"
if [ -n "${FAKE_FREEBUFF_BODY:-}" ]; then
  for ((i=1;i<=$#;i++)); do
    if [ "${!i}" = "-o" ]; then
      n=$((i+1))
      printf '%s' "$FAKE_FREEBUFF_BODY" > "${!n}"
    fi
  done
fi
MOCK
chmod +x "$TMP/curl"
export PATH="$TMP:$PATH" AIHUB_FREEBUFF_BASE="http://127.0.0.1:8402"

export FAKE_FREEBUFF_STATUS=200
export FAKE_FREEBUFF_BODY='{"status":"critical","accounts":0,"alive_accounts":0,"unknown_accounts":0}'
if bash "$ROOT/scripts/check-freebuff-login.sh" >"$TMP/out" 2>&1; then
  echo "Expected no-accounts to fail" >&2; exit 1
fi
grep -q 'NO ACCOUNTS' "$TMP/out"
! grep -qE 'session-token|password=' "$TMP/out"

export FAKE_FREEBUFF_BODY='{"status":"degraded","accounts":1,"alive_accounts":0,"unknown_accounts":1,"account_details":[{"email":"private@example.com","token":"supersecret"}]}'
bash "$ROOT/scripts/check-freebuff-login.sh" >"$TMP/out"
grep -q 'accounts=1' "$TMP/out"
! grep -qE 'private@example.com|supersecret' "$TMP/out"

export FAKE_FREEBUFF_BODY='{"status":"critical","accounts":[],"alive_accounts":0,"unknown_accounts":0}'
if bash "$ROOT/scripts/check-freebuff-login.sh" >"$TMP/out" 2>&1; then
  echo "Expected invalid health response to fail" >&2; exit 1
fi
echo "FreeBuff metadata-only onboarding fixtures passed"
