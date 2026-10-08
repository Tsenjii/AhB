#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

# Explicit quota-consuming, end-to-end Chat Completions compatibility test.
# It does NOT alter accounts or run arbitrary model-generated tools.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
model="${AIHUB_TEST_MODEL:-}"
if [ -z "$model" ]; then
  echo "Usage: AIHUB_TEST_MODEL='opencode/real-listed-model' ./scripts/test-provider-acceptance.sh" >&2
  echo "Tests: real chat, content-bearing SSE, structured tool call and tool-result continuation." >&2
  exit 2
fi
echo "=== 1/3 chat completion: $model ==="
bash ./scripts/test-chat.sh
echo
echo "=== 2/3 complete SSE stream ==="
bash ./scripts/test-stream.sh
echo
echo "=== 3/3 tool call -> tool response -> final answer ==="
bash ./scripts/test-tool-roundtrip.sh
echo
echo "PASS: full basic AhB provider acceptance: $model"
echo "NOTE: long-term quota, concurrency and upstream fidelity still require separate monitoring."
