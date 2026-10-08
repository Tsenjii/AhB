# Next Android phone acceptance — AhB (2026-10-08)

**User has not updated the working phone install yet.** OpenCode remains the only account source with real Android chat/SSE and initial tool-call emission evidence. No CodeArts2API duplicate is needed because Agent2API already includes CodeArts. Grok2API v3.1.6 is prebuilt but still needs the user's own authorized Grok model inference. Do not describe other providers as known-working solely from CI or a populated model directory.

## Release gates (before the user touches their phone)

1. The selected `main` source must have passing [GitHub CI](https://github.com/Tsenjii/AhB/actions/workflows/ci.yml), including `test-prepare-configs.sh`, `test-chat-fixture.sh`, `test-stream-fixture.sh`, `test-doctor-termux.sh`, `test-grok2api-fixture.sh`, `test-enable-grok2api-fixture.sh`, `test-tool-roundtrip-fixture.sh`, and `go test -race ./...`.
2. The Android prebuilt workflow must show **success**, and the published `https://raw.githubusercontent.com/Tsenjii/AhB/prebuilt/source-commit.txt` must match that build's source SHA. CI green alone is insufficient.
3. Verify the published bundle includes `scripts/test-chat.sh`, `scripts/test-stream.sh`, `scripts/test-provider-acceptance.sh`, `scripts/test-grok2api-termux.sh`, the repaired `scripts/doctor-termux.sh`, and `bin/grok2api`. Optional Kimi Web Python runtime is not bundled.
4. Keep the older functioning phone backup `~/AhB.backup-20261008-194847` or another independently verified good backup until after the user approves the new version.

## Phone actions AFTER successful publication

In Termux, from the existing installation (do **not** run a fresh installer or delete the account data):

```sh
cd ~/AhB
./scripts/upgrade-prebuilt-termux.sh

# The upgrader preserves account databases/config and retains the old tree.
./scripts/start-termux.sh
./scripts/doctor-termux.sh
./scripts/smoke.sh
```

First verify the existing OpenCode login, ordinary API response, SSE completion, and tool protocol. Obtain the exact model ID from `curl -fsS http://127.0.0.1:8317/v1/models | jq -r '.data[].id'`. Then manually select an OpenCode model, e.g. the previously tested `opencode/nemotron-3.5-lightning-free` **only if still present**:

```sh
AIHUB_TEST_MODEL='opencode/ACTUAL_LISTED_MODEL_ID' ./scripts/test-provider-acceptance.sh
```

The acceptance script makes multiple legitimate upstream requests and consumes actual quota. A full pass requires: nonempty ordinary chat, completed content-bearing SSE, valid structured `tool_calls`, tool-result continuation and a final assistant answer. If a model does not support tools, report it as incomplete; do not fake a pass. Never paste raw account credentials/tokens or private upstream request bodies into chat.

## Grok Web / Build, separately

```sh
cd ~/AhB
./scripts/enable-grok2api.sh
./scripts/stop-termux.sh && ./scripts/start-termux.sh

# Native upstream admin: http://127.0.0.1:8407/
# Authenticate your own legitimately entitled Grok account using its original UI.
./scripts/test-grok2api-termux.sh
curl -fsS http://127.0.0.1:8317/v1/models | jq -r '.data[].id | select(startswith("grok/"))'

# Opt-in live test for the exact model exposed by that account:
AIHUB_TEST_MODEL='grok/ACTUAL_LISTED_MODEL_ID' ./scripts/test-grok2api-termux.sh
```

Grok2API /readyz reports startup/component readiness, not spendable per-model quota. A source can advertise models yet have a rate limit or entitlement failure. Grok Web, Build and Console can have different account capabilities. No credential extraction or CAPTCHA/age/region bypass is part of this workflow.

## Optional dedicated tests and status display

- `./scripts/test-chat.sh`: explicitly fails HTTP 200 with an empty answer, and treats 401/429/503 as failure.
- `./scripts/test-stream.sh`: fails on missing `[DONE]`, missing text content chunks, or non-200 status. Does not claim realtime token latency.
- `./scripts/test-tool-roundtrip.sh`: validates two OpenAI tool requests, without executing model-generated arbitrary commands.
- `/api/providers`: in-memory `last_request_http_status` and `last_request_at` track the last **upstream** response headers, separate from account count and readiness. A 200 doesn't guarantee a successful stream or remaining quota. No request text, model IDs, cookies or secrets stored.
- Never run all optional services automatically on a small phone. Kimi Web stays **P3 experimental/off**; codearts2api remains **duplicate P3 alternative** to Agent2API CodeArts; the GPT Codex entry remains **external connector-only**.

## If an upgrade fails

The upgrader stores the previous installation as a timestamped sibling `~/AhB.backup-YYYYMMDD-HHMMSS`. It preserves original config and data and does not intentionally delete the older tree. Keep both old and new copies until verified. If startup or model routing breaks, stop the new Hub, identify the **most recent** backup containing the known-good account state, and restore it as `~/AhB`; do not overwrite the backup before confirming where the accounts are. Never post backup files publicly.

## What the user should report

Only send: **public prebuilt source SHA, AhB health status, enabled provider names, model ID (if non-sensitive), chat HTTP status, SSE chunk count, tool continuation PASS/FAIL, and any redacted error code**. Do not send secrets, API keys, cookies, account exports, login codes, full upstream traffic, or raw `config.json`.

This is **manual device acceptance**, not a claim that authenticated account inference has been tested in CI.
