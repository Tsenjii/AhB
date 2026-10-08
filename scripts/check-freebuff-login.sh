#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077
# Metadata-only account readiness for yutian81 Node gateway /healthz.
# Old Rust Web Cookie accounts and cookies are never read or exported here.
BASE="${AIHUB_FREEBUFF_BASE:-http://127.0.0.1:8402}"
case "$BASE" in
  http://127.0.0.1:*|http://localhost:*|http://\[::1\]:*) ;;
  *) echo "Only local FreeBuff diagnostics are supported." >&2; exit 2 ;;
esac
for cmd in curl jq; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "Missing $cmd; pkg install curl jq" >&2; exit 2; }
done
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
code="$(curl --silent --show-error --max-time 8 -o "$tmp" -w '%{http_code}' "$BASE/healthz" 2>/dev/null || true)"
if [ "$code" != 200 ]; then
  echo "FreeBuff Node gateway is not responding (HTTP $code)." >&2
  exit 1
fi
if ! jq -e '(.accounts | type == "number" and . >= 0)
  and (.alive_accounts | type == "number" and . >= 0)
  and (.unknown_accounts | type == "number" and . >= 0)
  and (.alive_accounts + .unknown_accounts <= .accounts)' "$tmp" >/dev/null 2>&1; then
  echo "Unknown or invalid FreeBuff health schema. Not treating it as logged in." >&2; exit 1
fi
jq -r '"FreeBuff Node gateway: accounts="+(.accounts|tostring)+
" observed_alive="+(.alive_accounts|tostring)+
" untested="+(.unknown_accounts|tostring)' "$tmp"
total="$(jq -r '.accounts' "$tmp")"
if [ "$total" -eq 0 ]; then
  echo "NO ACCOUNTS: old Rust Web Cookie logins cannot automatically migrate."
  echo "Run: cd ~/AhB && ./scripts/freebuff-login-termux.sh"
  exit 1
fi
echo "Metadata only: unknown accounts and HTTP 200 are NOT proof of inference quota."
echo "Next: test a real listed model via AhB's test-chat.sh (opt-in)."
