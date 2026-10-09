#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
umask 077

# Read-only inventory for nine bundled providers, plus optional Kimi Web.
# Never prints credentials, model prompts, provider errors or token fields.
BASE="${AIHUB_BASE:-http://127.0.0.1:8317}"
case "$BASE" in
  http://127.0.0.1:*|http://localhost:*|http://\[::1\]:*) ;;
  *) echo "Provider inventory accepts only loopback AhB URLs." >&2; exit 2 ;;
esac
for cmd in jq curl; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "Missing $cmd (Termux: pkg install -y curl jq)" >&2; exit 2; }
done

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
if ! curl -fsS --max-time 15 -o "$tmp/providers" "$BASE/api/providers"; then
  echo "AhB local provider API unavailable; start your existing AhB installation first." >&2
  exit 1
fi
if ! jq -e '(.providers | type == "array")' "$tmp/providers" >/dev/null 2>&1; then
  echo "AhB provider status response invalid." >&2
  exit 1
fi
# Missing /v1/models does not prevent account/process status diagnostics.
models_known=true
if ! curl -fsS --max-time 30 -o "$tmp/models" "$BASE/v1/models" 2>/dev/null ||
   ! jq -e '(.data | type == "array")' "$tmp/models" >/dev/null 2>&1; then
  models_known=false
fi

printf '%-11s %-8s %-10s %-9s %-10s %-8s %-10s\n' "SOURCE" "ENABLED" "PROCESS" "READY" "ACCOUNTS" "MODELS" "LAST_HTTP"
for id in opencode freebuff agent2api deepseek grok kiro copilot geminiweb duckai kimiweb; do
  status="$(jq -cr --arg id "$id" '.providers[] | select(.id == $id)' "$tmp/providers")"
  if [ -z "$status" ]; then
    printf '%-11s %-8s %-10s %-9s %-10s %-8s %-10s\n' "$id" "-" "MISSING" "-" "-" "-" "-"
    continue
  fi
  enabled="$(jq -r 'if .enabled then "YES" else "OFF" end' <<< "$status")"
  state="$(jq -r '.state // "UNKNOWN"' <<< "$status")"
  ready="$(jq -r 'if .provider_ready then "YES" else "NO" end' <<< "$status")"
  accounts="$(jq -r 'if .account_total == null then "UNKNOWN" else ((.account_usable_count // 0 | tostring) + "/" + (.account_total|tostring)) end' <<< "$status")"
  last="$(jq -r 'if .last_request_transport_error then "NETWORK" elif (.last_request_http_status // 0) > 0 then (.last_request_http_status|tostring) else "NOT_TESTED" end' <<< "$status")"
  count="UNKNOWN"
  if [ "$models_known" = true ]; then
    # Model discovery deliberately skips sleeping on-demand processes. Zero
    # advertised entries while asleep is NOT evidence the models were removed.
    sleeping="$(jq -r '(.start_mode == "on_demand") and (.process_alive != true)' <<< "$status")"
    if [ "$enabled" = "YES" ] && [ "$sleeping" = "true" ]; then
      count="SLEEP"
    else
      count="$(jq -r --arg id "$id" '[.data[] | select(.x_provider == $id)] | length' "$tmp/models")"
    fi
  fi
  printf '%-11s %-8s %-10s %-9s %-10s %-8s %-10s\n' "$id" "$enabled" "$state" "$ready" "$accounts" "$count" "$last"
done
echo "NOTE: HEALTHY, account counts and listed models are not proof of remaining quota, SSE completion or working tool calls."
echo "NOTE: No live model requests, log exports, credentials or account details were used."
