#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077

# Metadata-only FreeBuff account onboarding diagnostic. Never reads or prints
# account cookies, passwords, session tokens, or individual account records.
BASE="${AIHUB_FREEBUFF_BASE:-http://127.0.0.1:8402}"
case "$BASE" in
  http://127.0.0.1:*|http://localhost:*|http://\[::1\]:*) ;;
  *) echo "FreeBuff check accepts only local loopback HTTP." >&2; exit 2 ;;
esac
for cmd in curl jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Missing $cmd (install with: pkg install -y curl jq)" >&2
    exit 2
  fi
done

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
headers=()
if [ -n "${AIHUB_FREEBUFF_API_KEY:-}" ]; then
  headers=(-H "Authorization: Bearer $AIHUB_FREEBUFF_API_KEY")
fi
status="$(curl --silent --show-error --max-time 8 -o "$tmp" -w '%{http_code}' "${headers[@]}" "$BASE/api/accounts/health" 2>/dev/null || true)"
if [ "$status" != 200 ]; then
  echo "FreeBuff accounts endpoint unavailable (HTTP $status). Check AhB/FreeBuff startup and local API key." >&2
  exit 1
fi
if ! jq -e '(.ok == true) and (.accounts | type == "array")' "$tmp" >/dev/null 2>&1; then
  echo "FreeBuff account health response invalid; cannot claim login succeeded." >&2
  exit 1
fi
total="$(jq -r '.accounts | length' "$tmp")"
echo "FreeBuff registered accounts: $total (metadata only; not proof of available quota)."
if [ "$total" -eq 0 ]; then
  echo "NO ACCOUNTS: open http://127.0.0.1:8402/ui and use the original Accounts > Import flow."
  echo "On Android/Termux, Windows embedded login is unavailable."
  echo "Desktop Chrome/Edge one-click login needs FreeBuff's optional browser extension on the SAME computer as its local gateway."
  echo "Use only the upstream-supported import from your own account on the device; never send credentials in chat."
  exit 1
fi
echo "Account records exist. Verify a real model request separately; do not infer inference entitlement from this check."
