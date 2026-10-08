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
curl -fsS "$BASE/ui" | grep -q "Android AI Hub"
echo "UI HTML OK"