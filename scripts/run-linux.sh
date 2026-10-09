#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT"
[ "$(uname -s)" = Linux ] || { echo "Linux required" >&2; exit 1; }
[ -x ./bin/hubd ] || { echo "Missing native Linux hubd; install the Linux release bundle" >&2; exit 1; }
command -v jq >/dev/null || { echo "Missing jq (apt install jq)" >&2; exit 1; }
command -v node >/dev/null || { echo "Missing Node.js runtime for bundled FreeBuff" >&2; exit 1; }
umask 077
bash ./scripts/prepare-configs.sh
exec ./bin/hubd -config ./config.json
