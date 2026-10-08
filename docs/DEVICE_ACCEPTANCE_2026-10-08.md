# AhB Android / Termux device acceptance report — 2026-10-08

> Source: user's firsthand test report (quoted facts summarized, not independent access to device logs).
> The app reports `0.2.0-dev`. Exact installed source commit was not separately recorded in this test report.
> Device backup retained: `~/AhB.backup-20261008-194847` — **do not delete before all account sources are re-verified**.

## Device checklist results

| Item | User-tested result | Acceptance interpretation |
|---|---|---|
| Hub startup | `GET /healthz` responded `{"status":"ok"}` | **PASS** — Hub is running |
| Provider status | OpenCode HEALTHY; Agent2API HEALTHY, 2/2 credentials; FreeBuff DEGRADED, 0 accounts | **PASS** — health reporting; **see quota caveat below** |
| Model listing | 15 IDs, Agent2API 1 and OpenCode 14 | **PASS** — advertised models, not real-inference proof for each |
| Real inference | OpenCode `nemotron-3.5-lightning-free` returned `AIHUB_OK`, HTTP 200 | **PASS** — successful OpenCode chat |
| Streaming | 147 `data:` events and `data: [DONE]` | **PASS** — actual streaming |
| Upgrade persistence | Previously configured accounts/settings/databases retained | **PASS** — observed on real device |
| External bridge UI/setup | New dark UI has Connect wizard with LMArena, Providers and Models; `connect-bridge.sh` presets shown | **PASS for UI/preset visibility**; live LMArena and other third-party bridges **NOT CONNECTED OR INFERRED** |
| Tool calling | Model returned a `get_time` tool call, `finish_reason: tool_calls` | **PASS for tool-call emission**; no actual tool execution + follow-up response recorded, so the full tool round trip is **NOT VERIFIED** |

### Known account-level limitations, not Hub regressions

1. **Agent2API quota exhausted:** `agent2api/Qwen3.8-Flash` live inference returned **HTTP 503**; both of the two accounts were skipped because their remaining balances were below the upstream threshold. This must be reported as **INFERENCE UNAVAILABLE — LOW QUOTA**, even though the Hub's health view said **HEALTHY, 2/2 accounts**.
2. **FreeBuff:** no imported or usable account (0), hence DEGRADED. Account login/import must be completed before live inference can be verified.

### Important product UX finding

`Agent2API HEALTHY` and `2/2` currently describe **running backend and configured, enabled accounts with usable credentials**, not **sufficient quota to serve a request**. The current account probe at `internal/hub/accounts.go` counts `enabled && hasCredentials && chatSupported`; it does **not** probe balance, model-specific spendability, or cooldown/exhaustion state. Avoid presenting `2/2` as `2 accounts ready to infer`.

**Suggested follow-up enhancement:** distinguish `Backend alive`, `Credentials present`, and `Inference capacity` in the dashboard. Surface an explicit `LOW_QUOTA` / model-inference-unavailable indication only when supported by the upstream's own safe quota/usage endpoints or by a meaningful upstream error. Do not attempt to bypass account limits or reinterpret 503 as an AhB software outage.

### Still required before marking all features full-pass

- Agent2API: wait for legitimately replenished quota or a valid account, then verify successful actual chat and streaming; do not bypass upstream quota.
- FreeBuff: import/use a valid authorized account, then perform live inference.
- LMArena and other custom localhost bridges: test actual bridge startup, authenticated `/v1/models`, live inference, streaming, and relevant model paths. UI/preset visibility alone is not bridge integration runtime acceptance.
- Tool calling: after receiving the `get_time` tool call, execute the local tool, submit its output, and confirm the model produces a final response.
- DeepSeek2API, Grok2API, Kiro-Go: not covered in this reported run and remain real-device unverified.

## Overall decision

**Device smoke test: PASS for eight reported basic checkpoints, with explicitly limited scope.**
**Full real-device acceptance across all providers / LMArena / full tool cycle: NOT YET COMPLETE.**

No account token, email, private key, cookie or diagnostic log is reproduced in this document.
