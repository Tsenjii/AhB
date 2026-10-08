#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

BASE="${AIHUB_BASE:-http://127.0.0.1:8317}"

echo "== health =="
curl -fsS "$BASE/healthz"
echo

echo "== runtime =="
curl -fsS "$BASE/api/runtime"
echo

echo "== providers =="
curl -fsS "$BASE/api/providers"
echo

echo "== models =="
curl -fsS "$BASE/v1/models"
echo

echo "== ui =="
ui_html="$(curl -fsS "$BASE/ui")"
if [[ "$ui_html" != *'<title>AhB</title>'* ]]; then
  echo "UI HTML FAIL: AhB dashboard title missing" >&2
  exit 1
fi
if [[ "$ui_html" != *'id="bridgePreset"'* ]]; then
  echo "UI HTML FAIL: bridge connector wizard missing" >&2
  exit 1
fi
echo "UI HTML OK (AhB dashboard + bridge wizard)"