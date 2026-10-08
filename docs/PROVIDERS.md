# AhB providers — current integration matrix

This page describes provider **integration**, not proof of live account or inference success. Read [AI_HANDOFF.md](../AI_HANDOFF.md) for the latest source/CI/prebuilt revision and [ON_DEVICE_CHECKLIST.md](ON_DEVICE_CHECKLIST.md) for real Android test instructions.

| AhB prefix | Upstream / version | Packaged in ARM64 prebuilt | Initial state | Original management UI |
|---|---|---|---|---|
| `opencode/` | [opencode2api v1.3.7](https://github.com/jasonxu114514/opencode2api) | yes | enabled | `127.0.0.1:8404/` |
| `freebuff/` | [Freebuff2API v0.10.3](https://github.com/lza6/Freebuff-2API) | yes | enabled | `127.0.0.1:8402/ui` |
| `agent2api/` | [Agent2API v2.9.6](https://github.com/aimod-cc/agent2api) | yes | enabled by prebuilt installer when complete | `127.0.0.1:8403/` |
| `deepseek/` | [Deepseek2API](https://github.com/zengtao227/Deepseek2API) pinned source | yes | disabled | `127.0.0.1:8405/admin` |
| `grok/` | [Grok2API](https://github.com/chenyme/grok2api) pinned source | yes | disabled | `127.0.0.1:8407/` |
| `kiro/` | [Kiro-Go](https://github.com/Quorinex/Kiro-Go) pinned source | yes | disabled | `127.0.0.1:8408/admin` |
| `lmarena/` | separate user-supplied localhost bridge | **no bridge bundled** | disabled external slot | depends on bridge |

Run optional enable scripts from `~/AhB`: `scripts/enable-deepseek2api.sh`, `scripts/enable-grok2api.sh`, `scripts/enable-kiro-go.sh`, or `scripts/enable-lmarena-external.sh`; restart AhB after an enable operation. Never expose the upstream management UIs or an unauthenticated Agent2API port to a LAN or the public internet.

## What the upstream management UIs own

- Account import/login, provider-internal account pool and rotation
- Account balances, quota usage, authentication and refresh
- Provider-local proxy configuration, rate limits and retries
- Provider-specific model visibility, testing, Playground, diagnostics

AhB intentionally **does not** replicate that internal management logic. It supervises provider processes, exposes a unified `/v1` API with namespaced model IDs, and shows process/readiness/account states, model lists and memory usage in a phone-friendly foyer.

Agent2API v2.9.6 manages `workbuddy` (domestic) and `workbuddy-intl` (international) as separate account/provider identities inside Agent2API. AhB presents them through its `agent2api/` service prefix, not as separately launched processes.

## Verified vs unverified

- AhB Go unit/integration tests cover routing, model rewriting, credential header stripping, forwarding, simulated failures, account health, restarts and related control logic.
- CI builds the Android ARM64 bundle and checks installation/upgrade fixture behavior; it cannot prove the external account API works on a real phone.
- DeepSeek2API, Grok2API and Kiro-Go still require **real Android launch, account import, live model discovery, inference, streaming and tool-calling tests** before they are called real-device verified.
- No automatic signup, CAPTCHA avoidance, or restriction circumvention is part of AhB.

For actual versions, use the pinned commits in [the Android workflow](../.github/workflows/build-android-arm64.yml), not floating upstream branches.
