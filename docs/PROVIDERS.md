# AhB providers — current integration matrix

This page describes provider **integration**, not proof of live account or inference success. Read [AI_HANDOFF.md](../AI_HANDOFF.md) for the latest source/CI/prebuilt revision and [ON_DEVICE_CHECKLIST.md](ON_DEVICE_CHECKLIST.md) for real Android test instructions.

| AhB prefix | Upstream / version | Packaged in ARM64 prebuilt | Initial state | Original management UI |
|---|---|---|---|---|
| `opencode/` | [opencode2api v1.3.7](https://github.com/jasonxu114514/opencode2api) | yes | enabled | `127.0.0.1:8404/` |
| `freebuff/` | [yutian81/freebuff2api](https://github.com/yutian81/freebuff2api) pinned e0d8c9d (Node >=20) | yes | enabled; requires fresh CLI/Bearer login | no management UI; `scripts/freebuff-login-termux.sh` |
| `agent2api/` | [Agent2API v2.9.9 pinned source](https://github.com/aimod-cc/agent2api) | yes | enabled by prebuilt installer when complete | `127.0.0.1:8403/` |
| `deepseek/` | [0xgetz/deepseek2api](https://github.com/0xgetz/deepseek2api) experimental pure-Go source | yes | disabled on fresh installs; needs real chat.deepseek.com Web account/session; admin key alone is insufficient | none; private `data/deepseek2api/accounts.txt` instead |
| `grok/` | [Grok2API v3.1.6](https://github.com/chenyme/grok2api/releases/tag/v3.1.6), pinned to 2026-09-30 upstream SHA | yes | configured enabled/on demand; **P0 to test account readiness** | `127.0.0.1:8407/` |
| `kiro/` | [Kiro-Go](https://github.com/Quorinex/Kiro-Go) pinned source | yes | configured enabled/on demand; real account inference unverified | `127.0.0.1:8408/admin` |
| `copilot/` | [copilot2api pinned](https://github.com/whtsky/copilot2api) | yes; no verified account-backed phone inference | configured enabled, authorized OAuth required | device login via AhB dashboard |
| `geminiweb/` | [zexadev/gemini-web2api-go](https://github.com/zexadev/gemini-web2api-go) | yes, native Go ARM64 | on demand; account auth/inference unverified | upstream-dependent |
| `duckai/` | [desktop-tools-which-may-be-useful/duckai2api](https://github.com/desktop-tools-which-may-be-useful/duckai2api) Rust HTTP-only | yes, native Rust ARM64 | disabled on fresh installs; current upstream may return HTTP 418, so live chat is unverified | upstream-dependent |
| `kimiweb/` | [chopper1026/kimi2api pinned](https://github.com/chopper1026/kimi2api) | **installer scripts only** (Python/React not bundled) | disabled; use `install-kimiweb-termux.sh` then `enable-kimiweb-termux.sh` | `127.0.0.1:8412/admin` |
| `lmarena/` | separate user-supplied localhost bridge | **no bridge bundled** | disabled external slot | depends on bridge |

Run optional enable scripts from `~/AhB`: `scripts/enable-deepseek2api.sh`, `scripts/enable-grok2api.sh`, `scripts/enable-kiro-go.sh`, or `scripts/enable-lmarena-external.sh`; restart AhB after an enable operation. Never expose the upstream management UIs or an unauthenticated Agent2API port to a LAN or the public internet.

## FreeBuff login on Android — new Node adapter

The old Rust Freebuff2API v0.10.3 Web Cookie bridge produced 409 `chat_moved` (wrapped as 502). The new package replaces the executable implementation with the [pinned yutian81 Node gateway](https://github.com/yutian81/freebuff2api), still listening only at 127.0.0.1:8402 using the existing `freebuff/` namespace. It exposes no old management UI, and uses authenticated CLI/Bearer tokens, **not** the previous Web Cookie sessions. All old tokens, SQLite, JSON, configs and backups are preserved without automatic reinterpretation.

Android upgrade installs Termux Node.js. The next step is explicit user-initiated device-code authorization: `./scripts/freebuff-login-termux.sh`, then restart via `./scripts/stop-termux.sh && ./scripts/start-termux.sh`. Check with `./scripts/check-freebuff-login.sh` (account count only) and a real opt-in `AIHUB_TEST_MODEL='freebuff/<actual-listed-model>' ./scripts/test-chat.sh`. See [new Android login guide](FREEBUFF_ANDROID_LOGIN.md). Until live inference is confirmed, this remains **packaged and CI-verified, not phone-verified**.

## Agent2API release pin

The compiled Android/Linux source at `15b2a47dea22a964755e6ed97ef0769b2baa1cda` pins upstream Agent2API v2.9.9 source `cd97bce9912225e055cc79761d96a3f0b77e26f8`, built headless with the original UI. It is packaged, but real account quota and full Android tool continuation for this release remain unverified. The historical v2.9.6/v2.9.7 notes apply to older archives only.

## What the upstream management UIs own

- Account import/login, provider-internal account pool and rotation
- Account balances, quota usage, authentication and refresh
- Provider-local proxy configuration, rate limits and retries
- Provider-specific model visibility, testing, Playground, diagnostics

AhB intentionally **does not** replicate that internal management logic. It supervises provider processes, exposes a unified `/v1` API with namespaced model IDs, and shows process/readiness/account states, model lists and memory usage in a phone-friendly foyer.

Agent2API v2.9.7 manages `workbuddy` (domestic) and `workbuddy-intl` (international) as separate account/provider identities inside Agent2API. AhB presents them through its `agent2api/` service prefix, not as separately launched processes.

## Verified vs unverified

- AhB Go unit/integration tests cover routing, model rewriting, credential header stripping, forwarding, simulated failures, account health, restarts and related control logic.
- CI builds the Android ARM64 bundle and checks installation/upgrade fixture behavior; it cannot prove the external account API works on a real phone.
- DeepSeek2API, Grok2API and Kiro-Go still require **real Android launch, account import, live model discovery, inference, streaming and tool-calling tests** before they are called real-device verified.
- No automatic signup, CAPTCHA avoidance, or restriction circumvention is part of AhB.

For actual versions, use the pinned commits in [the Android workflow](../.github/workflows/build-android-arm64.yml), not floating upstream branches.

**Account/Quota caveat:** process HEALTHY and a credential count (e.g. Agent2API 2/2) do not prove adequate quota; the real phone test saw HTTP 503 from Agent2API with both credentialed accounts exhausted. The published dual-platform release also includes Copilot, Gemini Web and Duck.ai binaries; Kimi Web still ships only optional installer scripts. Neither has passed authenticated Android inference tests. See [the living source registry](TO_API_REGISTRY.json) and [installation/optimization plan](TO_API_INSTALL_AND_OPTIMIZATION_PLAN.md).


## Grok2API community audit and on-device acceptance

The pinned Go backend reports `VERSION=v3.1.6`; see [Grok2API community reliability study](GROK2API_COMMUNITY_RELIABILITY_2026-10-08.md). The author and community report good compatibility with Grok Build, Codex and Claude Code, but negative reports include Web/Console sync, rate-limit cooldown, long-running tool calls and responses that return HTTP 200 but no useful content. **Grok Web chat credits are not automatically equal to Grok Build/Codex model entitlements or capacity**. AhB marks Grok as high priority but *not* live-verified.

When the new script is present in your published bundle, and you have imported an authorized account using the native UI:

```sh
cd ~/AhB
./scripts/enable-grok2api.sh
./scripts/stop-termux.sh && ./scripts/start-termux.sh

# Safe metadata-only diagnostics (no model calls)
./scripts/test-grok2api-termux.sh

# Only when a real model is listed and your account may use it:
AIHUB_TEST_MODEL='grok/REPLACE_WITH_LISTED_MODEL_ID' ./scripts/test-grok2api-termux.sh
```

The opt-in live test checks ordinary chat, SSE termination and **two-step** structured Tool Calling, not just an emitted tool call. Do not expose your Grok admin key, client key or session/cookies in screenshots or bug logs. Enabled-provider accounts may still be rate-limited; process liveness does not prove available quota. Transient failures during client key validation now leave the saved local key and config untouched instead of provisioning duplicates.
