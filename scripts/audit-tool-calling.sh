#!/usr/bin/env bash
set -euo pipefail
umask 077

# User-approved, quota-consuming capability test.
# One advertised *live* model per enabled provider. A single actual OpenAI
# tool call followed by its result and the model's second answer is required.
# No generated tool commands are ever executed.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
BASE="${AIHUB_BASE:-http://127.0.0.1:8317}"
case "$BASE" in http://127.0.0.1:*|http://localhost:*|http://\[::1\]:*) ;; *)
 echo "Only localhost AhB is supported" >&2; exit 2;; esac
for tool in curl jq; do command -v "$tool" >/dev/null 2>&1 || {
 echo "Missing required command: $tool" >&2; exit 2; }; done
if [ "${AHB_TOOL_AUDIT_CONFIRM:-}" != YES ]; then
 echo "This sends real requests: up to two inference calls per model."
 echo "Confirm explicitly: AHB_TOOL_AUDIT_CONFIRM=YES bash scripts/audit-tool-calling.sh" >&2
 exit 2
fi
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
curl -fsS --max-time 15 "$BASE/api/providers" > "$tmp/providers.json"
curl -fsS --max-time 35 "$BASE/v1/models" > "$tmp/models.json"
jq -e '.providers|type=="array"' "$tmp/providers.json" >/dev/null
jq -e '.data|type=="array"' "$tmp/models.json" >/dev/null
# Avoid cached-only names, disabled providers, non-local/experimental
# unverified sources and route aliases. Limit candidates per provider to ONE.
jq -nr --slurpfile p "$tmp/providers.json" --slurpfile m "$tmp/models.json" '
  ($p[0].providers | map(select(.enabled==true and .id!="route"
    and .id!="duckai" and .id!="deepseek")) | map(.id)) as $enabled |
  [$m[0].data[] | select((.x_cached // false)==false)
    | select((.x_provider // "") as $id | $enabled | index($id)!=null)
    | select((.x_upstream_id // "")|type=="string" and length>0)
    | {provider:.x_provider,model:.id}] |
  unique_by(.provider) | .[:8] | .[] | [.provider,.model] | @tsv' > "$tmp/candidates"
if [ ! -s "$tmp/candidates" ]; then
 echo "No enabled providers with noncached models. Wake and load a model first."
 exit 1
fi
echo "MODEL / SOURCE                    TOOL LOOP"
pass=0; fail=0
while IFS="$(printf '\t')" read -r id model; do
 [ -n "$model" ] || continue
 if AIHUB_BASE="$BASE" AIHUB_TEST_MODEL="$model" \
   bash scripts/test-tool-roundtrip.sh > "$tmp/result" 2>&1; then
  printf '%-35s PASS (2-turn native tool loop)\n' "$model"
  pass=$((pass+1))
 else
  # Diagnostic output is NOT printed verbatim: providers may echo prompts or
  # private material. Keep result local and show only safe, generic status.
  printf '%-35s FAIL (inspect private result/logs)\n' "$model"
  fail=$((fail+1))
 fi
done < "$tmp/candidates"
echo "Tested: $((pass+fail)) | PASS: $pass | FAIL: $fail"
echo "All requests were sequential, no automatic retries or fallback."
[ "$fail" -eq 0 ]
