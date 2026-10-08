#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
BASE="$ROOT/data/kimiweb"
if [ ! -x "$BASE/venv/bin/python" ] || [ ! -f "$BASE/source/run.py" ] ||
   [ ! -f "$BASE/source/app/static/dist/index.html" ] ||
   [ ! -s "$BASE/client-key.txt" ] || [ ! -s "$BASE/source/.env" ]; then
  echo "Kimi Web not installed. First run: ./scripts/install-kimiweb-termux.sh" >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then pkg install -y jq; fi
./scripts/prepare-configs.sh
key="$(tr -d '\r\n' < "$BASE/client-key.txt")"
tmp="$(mktemp "$ROOT/.kimiweb-enable.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
jq -e --arg key "$key" '
  if any(.providers[]; .id == "kimiweb" and .kind == "sidecar") then
    (.providers[] | select(.id == "kimiweb") | .enabled) = true
    | (.providers[] | select(.id == "kimiweb") | .headers.Authorization) = ("Bearer " + $key)
  else error("kimiweb provider missing: upgrade AhB") end
' config.json > "$tmp"
cp -p config.json "$BASE/config-before-enable.json"
mv "$tmp" config.json
chmod 600 config.json
unset key
echo "Kimi Web enabled under 'kimiweb/' (separate from external 'kimi/' preset)."
echo "Restart: cd $ROOT && ./scripts/stop-termux.sh && ./scripts/start-termux.sh"
echo "Dashboard: http://127.0.0.1:8412/admin"
echo "Admin password is stored privately at $BASE/admin-password.txt"
echo "No-account / invalid-account health must still be treated as unable to infer."
