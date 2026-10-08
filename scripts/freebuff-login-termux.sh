#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
test -f data/freebuff/gateway/extract_freebuff.py || {
  echo "New FreeBuff Node gateway login helper is not installed." >&2; exit 1;
}
command -v python3 >/dev/null 2>&1 || {
  echo "Python is needed for the browser device-code login: pkg install -y python" >&2; exit 2;
}
mkdir -p data/freebuff/credentials
chmod 700 data/freebuff/credentials
# Older Rust FreeBuff Web cookies do not authenticate this new CLI/Bearer
# adapter. Keep ALL legacy token and database files; request a fresh,
# explicit, user-approved device-code login instead.
echo "Starting official FreeBuff browser device-code authorization."
echo "The one-time login URL is private; never post it or your token."
export FREEBUFF_CRED_FILE="$ROOT/data/freebuff/freebuff_credentials.json"
# Avoid incidental token forwarding to Telegram or copying a stale env token.
env -u TG_BOT_TOKEN -u TG_CHAT_ID -u FREEBUFF_TOKEN -u GITHUB_ACTIONS \
  python3 data/freebuff/gateway/extract_freebuff.py login
# Aggregate credentials are persisted outside the replaceable gateway folder.
python3 - <<'PY'
import hashlib
import json
import os
from pathlib import Path

root = Path("data/freebuff")
src = root / "freebuff_credentials.json"
dest = root / "credentials"
data = json.loads(src.read_text())
accounts = data.get("accounts")
if not isinstance(accounts, dict):
    single = data.get("default") or data
    accounts = {"default": single} if isinstance(single, dict) else {}
total = 0
for account in accounts.values():
    if not isinstance(account, dict):
        continue
    token = account.get("authToken")
    if not isinstance(token, str) or len(token.strip()) < 9:
        continue
    name = hashlib.sha256(token.encode()).hexdigest()[:18] + ".json"
    path = dest / name
    # Save atomically with permissions 0600; never print or paste the token.
    temp = dest / (name + ".tmp")
    fd = os.open(temp, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, "w") as stream:
        json.dump({"authToken": token.strip()}, stream)
    os.replace(temp, path)
    os.chmod(path, 0o600)
    total += 1
os.chmod(src, 0o600)
print(f"Imported {total} authorized CLI account(s) locally. No tokens printed.")
PY
echo "Now restart AhB so the Node gateway reloads its account pool:"
echo "  cd ~/AhB && ./scripts/stop-termux.sh && ./scripts/start-termux.sh"
echo "Then run: ./scripts/check-freebuff-login.sh"
