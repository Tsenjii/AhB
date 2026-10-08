#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077

# Attach any already-running local OpenAI-compatible gateway. No account
# credentials are copied out of the upstream bridge.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

usage() {
  cat <<'HELP'
AhB bridge connector
  ./scripts/connect-bridge.sh <preset-or-id> <local-base-url> [models-path]
  ./scripts/connect-bridge.sh --list

Examples:
  ./scripts/connect-bridge.sh lmarena http://127.0.0.1:5102
  ./scripts/connect-bridge.sh windsurf http://127.0.0.1:3003
  ./scripts/connect-bridge.sh kimi http://127.0.0.1:8000
  ./scripts/connect-bridge.sh mybridge http://127.0.0.1:8560

Set AIHUB_BRIDGE_API_KEY via a private environment (not in shell history)
when the local bridge requires an API key. Restart AhB after changing config.
Only loopback OpenAI-compatible services are accepted.
HELP
}

if [ "${1:-}" = "--list" ]; then
  printf '%s\n' "lmarena  LMArena (bring your own local bridge)" "windsurf WindsurfAPI" \
    "qwen     Qwen2API_Go" "kimi     Kimi2API" "gemini   Gemini2API" \
    "claude   Claude2API" "custom   Any localhost OpenAI-compatible API"
  exit 0
fi

id="${1:-}"
base="${2:-}"
models_path="${3:-/v1/models}"
if [ -z "$id" ] || [ -z "$base" ]; then usage; exit 2; fi
if [[ ! "$id" =~ ^[a-z][a-z0-9_-]{1,30}$ ]] || [ "$id" = "route" ]; then
  echo "Invalid provider ID; use lowercase letters, digits, hyphen or underscore." >&2
  exit 2
fi
# Exclude credentials, query, fragment, local-path tricks, or SSRF via domain
# aliases. Net/http also independently validates loopback provider URLs.
if [[ ! "$base" =~ ^http://(127\.0\.0\.1|localhost|\[::1\]):([0-9]{1,5})/?$ ]]; then
  echo "Use a localhost URL such as http://127.0.0.1:5102 (no paths or credentials)." >&2
  exit 2
fi
port="${BASH_REMATCH[2]}"
if ((10#$port < 1 || 10#$port > 65535)); then
  echo "Invalid local TCP port." >&2
  exit 2
fi
base="${base%/}"
if [[ ! "$models_path" =~ ^/[a-zA-Z0-9/_-]+$ ]]; then
  echo "Invalid models path." >&2
  exit 2
fi

if ! command -v jq >/dev/null 2>&1; then
  if command -v pkg >/dev/null 2>&1; then pkg install -y jq; else echo "Install jq first" >&2; exit 1; fi
fi
./scripts/prepare-configs.sh

case "$id" in
  lmarena) title="LMArena Bridge"; docs="https://github.com/Lianues/LMArenaBridge" ;;
  windsurf) title="WindsurfAPI"; docs="https://github.com/dwgx/WindsurfAPI" ;;
  qwen) title="Qwen2API"; docs="https://github.com/XxxXTeam/Qwen2API_Go" ;;
  kimi) title="Kimi2API"; docs="https://github.com/chopper1026/kimi2api" ;;
  gemini) title="Gemini2API"; docs="https://github.com/xwteam/gemini2api" ;;
  claude) title="Claude2API"; docs="https://github.com/yushangxiao/claude2api" ;;
  *) title="$id Bridge"; docs="" ;;
esac

# Do not overwrite existing managed sidecars: only external slots are editable.
if ! jq -e --arg id "$id" 'all(.providers[] | select(.id == $id); .kind == "external")' config.json >/dev/null; then
  echo "Provider $id is a managed sidecar and cannot be replaced." >&2
  exit 1
fi

key="${AIHUB_BRIDGE_API_KEY:-}"
headers='{}'
if [ -n "$key" ]; then
  headers="$(jq -nc --arg key "$key" '{Authorization:("Bearer " + $key)}')"
fi

tmp="$(mktemp "$ROOT/.bridge-config.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
jq --arg id "$id" --arg title "$title" --arg docs "$docs" \
  --arg base "$base" --arg models "$models_path" --argjson headers "$headers" '
  .providers = (
    if any(.providers[]; .id == $id) then
      [.providers[] | if .id == $id then
        .enabled = true | .base_url = $base | .models_path = $models
        | .health_path = $models | .headers = (if ($headers|length) > 0 then $headers else (.headers // {}) end)
      else . end]
    else
      .providers + [{
        id:$id, display_name:$title, description:"Local OpenAI-compatible bridge",
        enabled:true, kind:"external", base_url:$base,
        ui_url:"", docs_url:$docs, models_path:$models,
        health_path:$models, health_interval_seconds:15,
        headers:$headers
      }]
    end
  )
  | .routing.same_model_fallback.providers = (
      (.routing.same_model_fallback.providers // []) |
      if index($id) == null then . + [$id] else . end
    )
' config.json > "$tmp"
jq -e '.providers | type == "array"' "$tmp" >/dev/null
cp -p config.json "$ROOT/data/bridge-config-before-last-change.json"
mv "$tmp" config.json
chmod 600 config.json
echo "Connected configuration: $id -> $base (model list $models_path)"
echo "Restart AhB to apply the configuration. No upstream service was installed or launched."
