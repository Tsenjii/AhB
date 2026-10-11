#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/AhB/scripts"
cp "$root/scripts/import-freebuff-token-termux.sh" "$tmp/AhB/scripts/"
cd "$tmp/AhB"
token="fixture-authorized-bearer-token-123456789"
printf '%s\n' "$token" | bash scripts/import-freebuff-token-termux.sh > "$tmp/out"
! grep -Fq "$token" "$tmp/out"
python3 - <<'PY'
import json, os
from pathlib import Path
files=list(Path("data/freebuff/credentials").glob("*.json"))
assert len(files)==1,files
item=files[0]
assert json.loads(item.read_text())=={"authToken":"fixture-authorized-bearer-token-123456789"}
assert item.stat().st_mode & 0o777 == 0o600
PY
# Duplicate imports are idempotent and may not overwrite an unrelated account.
printf '%s\n' "$token" | bash scripts/import-freebuff-token-termux.sh >/dev/null
test "$(find data/freebuff/credentials -maxdepth 1 -name '*.json' | wc -l)" -eq 1
if printf 'short\n' | bash scripts/import-freebuff-token-termux.sh >"$tmp/invalid" 2>&1; then
 echo "short credential accepted" >&2; exit 1
fi
test "$(find data/freebuff/credentials -maxdepth 1 -name '*.json' | wc -l)" -eq 1
echo 'FreeBuff offline credential import fixture passed (private, idempotent, no stdout tokens)'
