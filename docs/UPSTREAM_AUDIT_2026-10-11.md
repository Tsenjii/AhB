# AhB bundled upstream audit — 2026-10-11 (Asia/Taipei)

Source of truth: the **exact SHA pins** in
[Android build](../.github/workflows/build-android-arm64.yml) and
[Linux build](../.github/workflows/build-linux.yml).
This audit compared every bundled upstream's `main` commit with those pins
using its public GitHub compare endpoint on 2026-10-11.
“Latest” means the **latest main revision at audit time**, not a guarantee
of account eligibility, a working unofficial upstream service, or future updates.

| Bundled gateway | Upstream | Pinned SHA after this pass | Finding |
|---|---|---|---|
| OpenCode | [jasonxu114514/opencode2api](https://github.com/jasonxu114514/opencode2api) | `1e2702f6e0562c7656d5bcfb99ce4b47383077a5` | Already latest main; retain the known fixed free-model routing |
| FreeBuff Node | [yutian81/freebuff2api](https://github.com/yutian81/freebuff2api) | `c4db1a352ebfa05f699a9fe1eca2cd96f31f868c` | 6 commits later; only `MODELS.md` changed; latest source pinned without altering credential handling |
| Agent2API | [aimod-cc/agent2api](https://github.com/aimod-cc/agent2api/releases/tag/v3.0.1) | `fd869a4610859f4690c307ad9e8b0e16d51e1058` | Upgrade **v2.9.9 → v3.0.1**; latest signed-off upstream release tag points to exact main SHA (87 commits newer) |
| Copilot | [whtsky/copilot2api](https://github.com/whtsky/copilot2api) | `a4aac95d4a8f430f02121f79ea36aeaaa06daea1` | Already latest main |
| Grok | [chenyme/grok2api](https://github.com/chenyme/grok2api) | `7c889a960e2638341b4dae9a5c81af0e0f38c87f` | Already latest main |
| Kiro | [Quorinex/Kiro-Go](https://github.com/Quorinex/Kiro-Go) | `f8f6071c9298a4266ad3e0c7e483d4a2510cbcaf` | Already latest main |
| DeepSeek Web experimental | [0xgetz/deepseek2api](https://github.com/0xgetz/deepseek2api) | `41b924e7741e1619433469900625dca132e00d6c` | Already latest main; still disabled by default; real request unverified |
| Gemini Web | [zexadev/gemini-web2api-go](https://github.com/zexadev/gemini-web2api-go) | `bc09c7171915deb7406a4b14344692cfa330388e` | Already latest main; real account inference unverified |
| Duck.ai experimental | [desktop-tools-which-may-be-useful/duckai2api](https://github.com/desktop-tools-which-may-be-useful/duckai2api) | `d8c6888eeb11daedb71a6ad589095f0a315626eb` | Already latest main; disabled by default; HTTP 418 upstream remains a possible blocker |

## Critical Agent2API changes

Release v3.0.0 introduced several experimental channels (MonkeyCode,
Command Code, Antigravity) and a six-language UI, including Traditional Chinese.
v3.0.1 fixes headless request cancellation, ZCode token availability and
quota accounting, among other items. This is a **major upstream version bump**:
do not conclude its live account/protocol behavior works from compilation alone.
The same headless binary target (`agent2api-server`) and UI path
(`desktop-tauri/ui`) are retained. The Android and Linux workflows still
build `--locked` and retain exactly the upstream server source, not a fork.

## Upgrade and verification gate

- **Release only after** AhB CI (unit tests, race, vet, frontend) and native
  Android ARM64, Linux AMD64 and ARM64 packaging/smoke tests all pass.
- Protect local account databases, cookies, tokens, private admin state and
  custom config via the existing staged upgrade with rollback backup.
- Verify `/api/accounts` readiness on a configured local Agent2API account,
  model listing, Chat Completions/SSE `[DONE]`, Responses/Anthropic where
  supported, and the full two-turn tool continuation on the actual device.
- Do not automatically enable unverified DeepSeek/Duck or claim new channels
  work on Android/Termux solely because Rust compiled.
- External bridges (CLIProxyAPI, Qwen, Kimi, etc.) are *not* bundled; their
  versions remain owned by the independent service and cannot be upgraded
  by repackaging AhB's nine native gateways.
