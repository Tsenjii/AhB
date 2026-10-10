#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p data/geminiweb data/duckai data/opencode data/freebuff/credentials data/agent2api data/deepseek2api data/grok2api/frontend data/grok2api/data data/kiro-go/web data/copilot2api logs bin

if [ ! -f config.json ]; then
  cp config.example.json config.json
fi
if [ ! -f data/opencode/config.json ]; then
  cp configs/opencode2api.json data/opencode/config.json
fi
if [ ! -f data/freebuff/config.json ]; then
  cp configs/freebuff2api.json data/freebuff/config.json
fi
if [ ! -f data/geminiweb/config.json ]; then
  cp configs/geminiweb.json data/geminiweb/config.json
fi
if [ ! -f data/deepseek2api/config.json ]; then
  cp configs/deepseek2api.json data/deepseek2api/config.json
fi
if [ ! -f data/grok2api/config.yaml ]; then
  cp configs/grok2api.yaml data/grok2api/config.yaml
fi
if [ ! -f data/kiro-go/config.json ]; then
  cp configs/kiro-go.json data/kiro-go/config.json
fi

# Migrate existing installs without overwriting user settings.
# Newly bundled providers are appended from config.example.json without
# replacing existing provider-specific settings.
if command -v jq >/dev/null 2>&1 && [ -f config.json ]; then
  # Keep the old config intact on failure, never stream a partial jq result
  # back into a file containing real client keys and custom user routes.
  tmp="$(mktemp "$ROOT/.ahb-config-merge.XXXXXX")"
  kimi_installed=false
  if [ -x "$ROOT/data/kimiweb/venv/bin/python" ] && [ -f "$ROOT/data/kimiweb/source/run.py" ]; then
    kimi_installed=true
  fi
  if jq --slurpfile example config.example.json --argjson kimiInstalled "$kimi_installed" '
      reduce $example[0].providers[] as $p (.;
        if any(.providers[]?; .id == $p.id) then . else .providers += [$p] end
      )
      # Only real bundled ARM64 adapters belong to the stock install.
      # Preserve any manually attached bridge and any installed legacy Kimi.
      | ([.routes[]?.targets[]? | split("/")[0]]) as $routeRefs
      | .providers |= map(select(
          ((
            (.id == "lmarena" and (.enabled == false) and .base_url == "http://127.0.0.1:5102" and ($routeRefs | index("lmarena")) == null)
            or
            (.id == "kimiweb" and ($kimiInstalled | not) and (.enabled == false) and ($routeRefs | index("kimiweb")) == null)
          ) | not)
        ))
      # Existing user-selected on/off flags stay untouched. Newly installed
      # providers are enabled but sleep until used, respecting this bundle\x27s profile.
      | .providers |= map(
          if (.id as $id | ["opencode","freebuff","agent2api","deepseek","grok","kiro","copilot","geminiweb","duckai"] | index($id)) != null
          then .start_mode = (.start_mode // "on_demand")
          else . end
        )
      | .resources = ((.resources // {}) + {
          "max_running_sidecars": ((.resources.max_running_sidecars // $example[0].resources.max_running_sidecars // 3)),
          "idle_stop_seconds": ((.resources.idle_stop_seconds // $example[0].resources.idle_stop_seconds // 900))
        })
      | (.providers[] | select(.id == "opencode") | .env) =
          (((.providers[] | select(.id == "opencode") | .env) // {}) + {"GODEBUG":"netdns=cgo"})
      | (.providers[] | select(.id == "grok") | .env) =
          (((.providers[] | select(.id == "grok") | .env) // {}) + {"GODEBUG":"netdns=cgo"})
      | (.providers[] | select(.id == "copilot") | .env) =
          (((.providers[] | select(.id == "copilot") | .env) // {}) + {"GODEBUG":"netdns=cgo"})
      # Switch existing FreeBuff installs to the pinned Node service without
      # touching old Cookie, SQLite or token stores under data/freebuff/.
      | .providers |= map(
          if .id == "freebuff" then
            .description = "Freebuff Node.js CLI/Bearer gateway (device login required)"
            | .ui_url = ""
            | .docs_url = "https://github.com/yutian81/freebuff2api"
            | .env = ((.env // {}) + {
                "HOST":"127.0.0.1", "PORT":"8402",
                "FREEBUFF_CREDENTIALS_DIR":"./credentials",
                "FREEBUFF_API_KEY":"__AIHUB_SERVER_KEY__",
                "FREEBUFF_DEBUG":"false"
              })
            | .headers = ((.headers // {}) + {"Authorization":"Bearer __AIHUB_SERVER_KEY__"})
          else . end
        )
      # Replace old DeepSeek and Duck sidecar contracts without importing,
      # printing or reusing obsolete account/admin secrets.
      | .providers |= map(
          if .id == "deepseek" then
            .display_name = "DeepSeek Web (experimental Go)"
            | .description = "Experimental Oct 2026 pure-Go Web adapter; requires own account in private accounts.txt"
            | .ui_url = ""
            | .docs_url = "https://github.com/0xgetz/deepseek2api"
            | .health_path = "/health"
            | .env = {"PORT":"8405","PROXY_API_KEY":"__AIHUB_SERVER_KEY__",
                       "DEEPSEEK_ACCOUNTS_FILE":"accounts.txt","GODEBUG":"netdns=cgo",
                       "NO_PROXY":"127.0.0.1,localhost"}
          elif .id == "duckai" then
            .display_name = "Duck.ai (experimental Rust)"
            | .description = "Experimental Rust HTTP-only adapter; upstream 418 remains unresolved"
            | .docs_url = "https://github.com/desktop-tools-which-may-be-useful/duckai2api"
            | .health_path = "/health"
            | .env = {"PORT":"8414","DUCKAI_BIND":"127.0.0.1",
                       "DUCKAI_DEFAULT_API_KEY":"__AIHUB_SERVER_KEY__",
                       "DUCKAI_DB_PATH":"duckai.db","DUCKAI_MAX_CONCURRENCY":"2"}
          else . end
        )
      | (.providers | map(.id)) as $validIDs
      | .routing.same_model_fallback.providers =
          ((.routing.same_model_fallback.providers // []) | map(select(. as $id | $validIDs | index($id))))
      | reduce ($example[0].routing.same_model_fallback.providers // [])[] as $id (.;
          if (.routing.same_model_fallback.providers | index($id)) == null
          then .routing.same_model_fallback.providers += [$id]
          else .
          end
        )
    ' config.json > "$tmp" && jq -e '.providers | type == "array"' "$tmp" >/dev/null; then
    chmod 600 "$tmp"
    mv -f "$tmp" config.json
  else
    rm -f "$tmp"
    echo "ERROR: AhB provider config migration failed. Original config.json left untouched." >&2
    exit 1
  fi
fi
make_secret() {
  od -An -N24 -tx1 /dev/urandom | tr -d ' \n'
}

make_base64_32() {
  dd if=/dev/urandom bs=32 count=1 2>/dev/null | base64 | tr -d '\r\n'
}

KEY_FILE="data/hub-local-key.txt"
if [ ! -s "$KEY_FILE" ]; then
  make_secret > "$KEY_FILE"
  printf '\n' >> "$KEY_FILE"
  chmod 600 "$KEY_FILE"
fi
LOCAL_KEY="$(tr -d '\r\n' < "$KEY_FILE")"

for file in config.json data/opencode/config.json data/deepseek2api/config.json data/kiro-go/config.json; do
  if [ -f "$file" ]; then
    sed -i "s/__AIHUB_SERVER_KEY__/$LOCAL_KEY/g" "$file"
    sed -i "s/hub-local-opencode/$LOCAL_KEY/g" "$file"
  fi
done

DEEPSEEK_ADMIN_KEY_FILE="data/deepseek2api/admin-key.txt"
if [ ! -s "$DEEPSEEK_ADMIN_KEY_FILE" ]; then
  make_secret > "$DEEPSEEK_ADMIN_KEY_FILE"
  printf '\n' >> "$DEEPSEEK_ADMIN_KEY_FILE"
  chmod 600 "$DEEPSEEK_ADMIN_KEY_FILE"
fi
DEEPSEEK_ADMIN_KEY="$(tr -d '\r\n' < "$DEEPSEEK_ADMIN_KEY_FILE")"
if [ -f config.json ]; then
  sed -i "s/__AIHUB_DEEPSEEK_ADMIN_KEY__/$DEEPSEEK_ADMIN_KEY/g" config.json
fi

GROK_JWT_FILE="data/grok2api/jwt-secret.txt"
if [ ! -s "$GROK_JWT_FILE" ]; then
  make_secret > "$GROK_JWT_FILE"
  printf '\n' >> "$GROK_JWT_FILE"
  chmod 600 "$GROK_JWT_FILE"
fi
GROK_JWT="$(tr -d '\r\n' < "$GROK_JWT_FILE")"

GROK_CRED_FILE="data/grok2api/credential-key.txt"
if [ ! -s "$GROK_CRED_FILE" ]; then
  make_base64_32 > "$GROK_CRED_FILE"
  printf '\n' >> "$GROK_CRED_FILE"
  chmod 600 "$GROK_CRED_FILE"
fi
GROK_CRED="$(tr -d '\r\n' < "$GROK_CRED_FILE")"

GROK_ADMIN_FILE="data/grok2api/admin-password.txt"
if [ ! -s "$GROK_ADMIN_FILE" ]; then
  make_secret > "$GROK_ADMIN_FILE"
  printf '\n' >> "$GROK_ADMIN_FILE"
  chmod 600 "$GROK_ADMIN_FILE"
fi
GROK_ADMIN="$(tr -d '\r\n' < "$GROK_ADMIN_FILE")"

if [ -f data/grok2api/config.yaml ]; then
  sed -i "s|__AIHUB_GROK_JWT_SECRET__|$GROK_JWT|g" data/grok2api/config.yaml
  sed -i "s|__AIHUB_GROK_CREDENTIAL_KEY__|$GROK_CRED|g" data/grok2api/config.yaml
  sed -i "s|__AIHUB_GROK_ADMIN_PASSWORD__|$GROK_ADMIN|g" data/grok2api/config.yaml
fi

GROK_CLIENT_FILE="data/grok2api/client-key.txt"
if [ -s "$GROK_CLIENT_FILE" ] && [ -f config.json ]; then
  GROK_CLIENT="$(tr -d '\r\n' < "$GROK_CLIENT_FILE")"
  sed -i "s|__AIHUB_GROK_CLIENT_KEY__|$GROK_CLIENT|g" config.json
fi

KIRO_ADMIN_FILE="data/kiro-go/admin-password.txt"
if [ ! -s "$KIRO_ADMIN_FILE" ]; then
  make_secret > "$KIRO_ADMIN_FILE"
  printf '\n' >> "$KIRO_ADMIN_FILE"
  chmod 600 "$KIRO_ADMIN_FILE"
fi
KIRO_ADMIN="$(tr -d '\r\n' < "$KIRO_ADMIN_FILE")"
if [ -f data/kiro-go/config.json ]; then
  sed -i "s|__AIHUB_KIRO_ADMIN_PASSWORD__|$KIRO_ADMIN|g" data/kiro-go/config.json
fi
if [ -f config.json ]; then
  sed -i "s|__AIHUB_KIRO_ADMIN_PASSWORD__|$KIRO_ADMIN|g" config.json
fi

WEB_PASS_FILE="data/opencode/webui-password.txt"
if grep -q "__AIHUB_WEB_PASSWORD__" data/opencode/config.json; then
  if [ ! -s "$WEB_PASS_FILE" ]; then
    make_secret > "$WEB_PASS_FILE"
    printf '\n' >> "$WEB_PASS_FILE"
    chmod 600 "$WEB_PASS_FILE"
  fi
  WEB_PASS="$(tr -d '\r\n' < "$WEB_PASS_FILE")"
  sed -i "s/__AIHUB_WEB_PASSWORD__/$WEB_PASS/g" data/opencode/config.json
fi

chmod 600 config.json data/opencode/config.json data/freebuff/config.json data/deepseek2api/config.json data/deepseek2api/admin-key.txt data/grok2api/config.yaml data/grok2api/jwt-secret.txt data/grok2api/credential-key.txt data/grok2api/admin-password.txt data/grok2api/client-key.txt data/kiro-go/config.json data/kiro-go/admin-password.txt 2>/dev/null || true