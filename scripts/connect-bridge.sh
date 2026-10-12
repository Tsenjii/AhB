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
  ./scripts/connect-bridge.sh codex http://127.0.0.1:9879
  ./scripts/connect-bridge.sh cliproxy http://127.0.0.1:8416
  ./scripts/connect-bridge.sh kimi http://127.0.0.1:8000
  ./scripts/connect-bridge.sh qwenweb http://127.0.0.1:3000
  ./scripts/connect-bridge.sh deepseekweb http://127.0.0.1:3456
  ./scripts/connect-bridge.sh gemini http://127.0.0.1:5918
  ./scripts/connect-bridge.sh mimo http://127.0.0.1:8090
  ./scripts/connect-bridge.sh zaiweb http://127.0.0.1:8460
  ./scripts/connect-bridge.sh geminioauth http://127.0.0.1:10551
  ./scripts/connect-bridge.sh claudecli http://127.0.0.1:8080
  ./scripts/connect-bridge.sh mybridge http://127.0.0.1:8560

The script privately prompts for an optional API key (or accepts
AIHUB_BRIDGE_API_KEY from a private environment). It verifies /v1/models
before saving. Restart AhB after changing config.
Only loopback OpenAI-compatible services are accepted.
HELP
}

if [ "${1:-}" = "--list" ]; then
  printf '%s\n' "lmarena  LMArena (bring your own local bridge)" "windsurf WindsurfAPI" \
    "qwen     Qwen2API_Go" "qwenweb  Qwen Web Go (external)" "kimi     Kimi2API" "deepseekweb  Browser Web-to-API (Chromium needed)" "gemini   Gemini2API" \
    "claude   Claude2API" "codex    Codex OAuth API/MCP (user-run)" \
    "cliproxy CLIProxyAPI (optional, Antigravity/Codex/Claude/Muse)" \
    "mimo     MiMo 2API (Go, separately installed)" \
    "zaiweb   Z.ai Web 2API (Go; authorized account required)" \
    "geminioauth Gemini CLI OAuth (Node.js, separately installed)" \
    "claudecli Claude CLI OpenAI Proxy (Go wrapper + separate Claude CLI)" \
    "custom   Any localhost OpenAI-compatible API"
  exit 0
fi

id="${1:-}"
base="${2:-}"
models_path="${3:-/v1/models}"
if [ "$id" = "gemini" ] && [ "$#" -lt 3 ]; then
  models_path="/openai/v1/models"
fi
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
if [[ ! "$models_path" =~ ^/[a-zA-Z0-9/_-]+$ ]] || [[ "$models_path" != */models ]]; then
  echo "Invalid models path (expected /v1/models or /openai/v1/models)." >&2
  exit 2
fi
api_prefix="${models_path%/models}"

if ! command -v jq >/dev/null 2>&1; then
  if command -v pkg >/dev/null 2>&1; then pkg install -y jq; else echo "Install jq first" >&2; exit 1; fi
fi
./scripts/prepare-configs.sh

case "$id" in
  lmarena) title="LMArena Bridge"; docs="https://github.com/Lianues/LMArenaBridge" ;;
  windsurf) title="WindsurfAPI"; docs="https://github.com/dwgx/WindsurfAPI" ;;
  qwen) title="Qwen2API"; docs="https://github.com/XxxXTeam/Qwen2API_Go" ;;
  qwenweb) title="Qwen Web API (local)"; docs="https://github.com/XxxXTeam/Qwen2API_Go" ;;
  deepseekweb) title="DeepSeek Web (separate Chromium bridge)"; docs="https://github.com/PruhaNLP/web-to-api" ;;
  kimi) title="Kimi2API"; docs="https://github.com/chopper1026/kimi2api" ;;
  gemini) title="Gemini2API"; docs="https://github.com/xwteam/gemini2api" ;;
  claude) title="Claude2API"; docs="https://github.com/yushangxiao/claude2api" ;;
  codex) title="Codex OAuth API Bridge"; docs="https://github.com/dvcrn/codex-oauth-proxy" ;;
  cliproxy) title="CLIProxyAPI"; docs="https://github.com/router-for-me/CLIProxyAPI" ;;
  mimo) title="MiMo 2API"; docs="https://github.com/rong6/mimo-2api" ;;
  zaiweb) title="Z.ai Web (authorized account)"; docs="https://github.com/XxxXTeam/zai2api" ;;
  geminioauth) title="Gemini CLI OAuth Bridge"; docs="https://github.com/kimws/gemini-oauth" ;;
  claudecli) title="Claude CLI OpenAI Proxy"; docs="https://github.com/kpod13/claude-openai-proxy" ;;
  *) title="$id Bridge"; docs="" ;;
esac

# Do not overwrite existing managed sidecars: only external slots are editable.
if ! jq -e --arg id "$id" 'all(.providers[] | select(.id == $id); .kind == "external")' config.json >/dev/null; then
  echo "Provider $id is a managed sidecar and cannot be replaced." >&2
  exit 1
fi

key="${AIHUB_BRIDGE_API_KEY:-}"
existing_auth="$(jq -r --arg id "$id" '.providers[] | select(.id == $id) | .headers.Authorization // empty' config.json | head -n 1)"
existing_base="$(jq -r --arg id "$id" '.providers[] | select(.id == $id) | .base_url // empty' config.json | head -n 1)"
existing_headers_count="$(jq -r --arg id "$id" '[.providers[] | select(.id == $id) | (.headers // {}) | keys[]] | length' config.json)"
# Changing localhost ports must never silently donate old credentials.
if [ -n "$existing_base" ] && [ "${existing_base%/}" != "$base" ] &&
   [ "$existing_headers_count" -gt 0 ]; then
  if [ -z "$key" ] && [ -t 0 ]; then
    printf 'Different bridge destination; enter its NEW local API key: '
    IFS= read -rs key || true
    printf '\n'
  fi
  if [ -z "$key" ]; then
    echo "Refusing to reuse private headers at a changed bridge address. Supply a new key in the private terminal, or attach under a new ID." >&2
    exit 1
  fi
elif [ -z "$key" ] && [ -n "$existing_auth" ]; then
  # Reuse only at the exact same loopback endpoint.
  key="${existing_auth#Bearer }"
elif [ -z "$key" ] && [ -t 0 ]; then
  printf 'Local bridge API key (Enter if not required): '
  IFS= read -rs key || true
  printf '\n'
fi

headers='{}'
# Never allow HTTP_PROXY to relay a private localhost API key.
curl_args=(--fail --silent --show-error --noproxy '*' --proto '=http' \
  --max-redirs 0 --connect-timeout 3 --max-time 8 --max-filesize 1048576)
if [ -n "$key" ]; then
  headers="$(jq -nc --arg key "$key" '{Authorization:("Bearer " + $key)}')"
  curl_args+=(-H "Authorization: Bearer $key")
fi

# Fail fast instead of silently installing a broken route. A model catalog
# with no accounts can be empty, but the response must have a data array.
if ! model_json="$(curl "${curl_args[@]}" "$base$models_path")"; then
  echo "Bridge unavailable or unauthorized at $base$models_path; configuration unchanged." >&2
  exit 1
fi
if ! printf '%s' "$model_json" | jq -e '(.data | type) == "array"' >/dev/null 2>&1; then
  echo "Bridge does not expose a valid OpenAI models list; configuration unchanged." >&2
  exit 1
fi
unset model_json key

tmp="$(mktemp "$ROOT/.bridge-config.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
jq --arg id "$id" --arg title "$title" --arg docs "$docs" \
  --arg base "$base" --arg models "$models_path" --arg api_prefix "$api_prefix" --argjson headers "$headers" '
  .providers = (
    if any(.providers[]; .id == $id) then
      [.providers[] | if .id == $id then
        .enabled = true | .base_url = $base | .models_path = $models
        | .api_path_prefix = $api_prefix
        | .health_path = $models | .headers = (if ($headers|length) > 0 then $headers else (.headers // {}) end)
      else . end]
    else
      .providers + [{
        id:$id, display_name:$title, description:"Local OpenAI-compatible bridge",
        enabled:true, kind:"external", base_url:$base,
        ui_url:"", docs_url:$docs, models_path:$models,
        api_path_prefix:$api_prefix,
        health_path:$models, health_interval_seconds:15,
        headers:$headers
      }]
    end
  )
' config.json > "$tmp"
jq -e '.providers | type == "array"' "$tmp" >/dev/null
cp -p config.json "$ROOT/data/bridge-config-before-last-change.json"
mv "$tmp" config.json
chmod 600 config.json
echo "Connected configuration: $id -> $base (model list $models_path)"
echo "Restart AhB to apply the configuration. No upstream service was installed or launched."
echo "Safety: this connector was not auto-added to failover routing; first verify real inference and tool roundtrip."
