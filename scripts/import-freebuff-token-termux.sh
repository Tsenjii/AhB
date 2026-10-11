#!/usr/bin/env bash
set -euo pipefail
umask 077
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
command -v python3 >/dev/null || { echo "Python is required (Termux: pkg install python)" >&2; exit 2; }
mkdir -p data/freebuff/credentials
chmod 700 data/freebuff data/freebuff/credentials
echo "This imports an EXISTING, authorized Codebuff CLI authToken."
echo "It cannot repair a failed Google login or upstream HTTP 502."
echo "Never paste the token into ChatGPT, logs, GitHub, or shell command history."
printf 'Paste existing authToken (hidden input): '
IFS= read -rs token
printf '\n'
if [ "${#token}" -lt 9 ] || [ "${#token}" -gt 16384 ]; then
 echo "Invalid token length; nothing was saved." >&2; exit 1
fi
# Transfer through a private inherited environment variable only. The token
# never appears in process argv, command history, stdout or config.json.
AHB_FREEBUFF_IMPORT_TOKEN="$token" python3 - <<'PY'
import hashlib, json, os, stat, tempfile
from pathlib import Path
root=Path("data/freebuff/credentials")
token=os.environ.pop("AHB_FREEBUFF_IMPORT_TOKEN","").strip()
if not (9 <= len(token) <= 16384): raise SystemExit("Invalid bearer token.")
for d in (Path("data"),Path("data/freebuff"),root):
    info=d.lstat()
    if not stat.S_ISDIR(info.st_mode): raise SystemExit("Unsafe credential directory.")
name=hashlib.sha256(token.encode()).hexdigest()[:18]+".json"
target=root/name
if target.is_symlink(): raise SystemExit("Refusing credential symlink.")
fd,path=tempfile.mkstemp(prefix=".ahb-import-",suffix=".tmp",dir=root)
try:
    os.fchmod(fd,0o600)
    with os.fdopen(fd,"w") as f:
        json.dump({"authToken":token},f)
        f.flush();os.fsync(f.fileno())
    os.replace(path,target)
    os.chmod(target,0o600)
finally:
    if os.path.exists(path):os.remove(path)
print("Imported one authorized CLI credential privately (not verified against upstream).")
PY
unset token
echo "If FreeBuff is running, restart only that gateway from the AhB Dashboard."
echo "Then verify account readiness. This import does not guarantee free quota or working inference."
