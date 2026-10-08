#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077

# Optional, explicitly requested install: Kimi Web Python + React UI.
# Not part of the base Android archive; may require several native Termux
# packages and requires an independently authorized Kimi account.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
if [ "$(uname -m)" != "aarch64" ]; then
  echo "This helper is intended for Termux Android ARM64." >&2
  exit 1
fi
if ! command -v pkg >/dev/null 2>&1; then
  echo "Termux pkg is required." >&2; exit 1
fi
for pkg_name in git python nodejs jq; do
  if ! command -v "${pkg_name/nodejs/node}" >/dev/null 2>&1; then
    pkg install -y "$pkg_name"
  fi
done
./scripts/prepare-configs.sh

BASE="$ROOT/data/kimiweb"
SRC="$BASE/source"
PIN="7f046d8627f275432f82788a6547bc905038738c"
mkdir -p "$BASE"
chmod 700 "$BASE"

if [ ! -d "$SRC/.git" ]; then
  if [ -e "$SRC" ]; then
    echo "Kimi source path exists but is not a Git repository: $SRC" >&2
    exit 1
  fi
  mkdir -p "$SRC"
  git -C "$SRC" init -q
  git -C "$SRC" remote add origin https://github.com/chopper1026/kimi2api.git
  git -C "$SRC" fetch --depth 1 origin "$PIN"
  git -C "$SRC" checkout --detach FETCH_HEAD
fi
actual="$(git -C "$SRC" rev-parse HEAD)"
if [ "$actual" != "$PIN" ]; then
  echo "Kimi source is not the audited pinned revision ($PIN)." >&2
  echo "Existing files untouched. Review changes or move the old source directory before reinstalling." >&2
  exit 1
fi
if [ ! -f "$SRC/pyproject.toml" ] || [ ! -f "$SRC/web/package-lock.json" ]; then
  echo "Incomplete Kimi source checkout." >&2; exit 1
fi

if [ ! -x "$BASE/venv/bin/python" ]; then
  python -m venv "$BASE/venv"
fi
"$BASE/venv/bin/python" -m pip install --no-cache-dir -e "$SRC"
if [ ! -f "$SRC/app/static/dist/index.html" ]; then
  npm ci --prefix "$SRC/web"
  npm run build --prefix "$SRC/web"
fi

generate_secret(){ od -An -N24 -tx1 /dev/urandom | tr -d ' \n'; }
for item in admin-password client-key; do
  if [ ! -s "$BASE/$item.txt" ]; then
    generate_secret > "$BASE/$item.txt"
    printf '\n' >> "$BASE/$item.txt"
  fi
  chmod 600 "$BASE/$item.txt"
done
mkdir -p "$BASE/state"
chmod 700 "$BASE/state"

# Create configuration only once: never overwrite existing user's Kimi state.
if [ ! -f "$SRC/.env" ]; then
  admin_pass="$(tr -d '\r\n' < "$BASE/admin-password.txt")"
  client_key="$(tr -d '\r\n' < "$BASE/client-key.txt")"
  cat > "$SRC/.env" <<EOF
HOST=127.0.0.1
PORT=8412
ADMIN_PASSWORD=$admin_pass
OPENAI_API_KEY=$client_key
SECURE_COOKIES=false
DATA_DIR=$BASE/state
TIMEZONE=Asia/Taipei
REQUEST_LOG_RETENTION=100
REQUEST_LOG_BODY_LIMIT=0
EOF
  chmod 600 "$SRC/.env"
  unset admin_pass client_key
fi

echo
echo "Optional Kimi Web code and dependencies installed locally."
echo "Accounts, tokens and request logs stay under $BASE (mode-restricted local files)."
echo "You still need your own authorized Kimi Web account."
echo "Enable it with: cd $ROOT && ./scripts/enable-kimiweb-termux.sh"
echo "If Python package installation fails on Android, report the exact error; do not run on a public listener."
