# NEXT AI HANDOFF — AhB

> **Single source of truth:** [../AI_HANDOFF.md](../AI_HANDOFF.md). Read that file and current GitHub Actions before changing the project. This page is a short index, **not** an independent state snapshot. Do not use older v2.9.5 or pre-Grok/Kiro notes as the project baseline.

## Repository and goal

- Canonical repository: `Tsenjii/AhB` (public), branch `main`.
- Public `prebuilt` branch distributes the Android ARM64/Termux bundle.
- Project goal: thin local supervisor and unified OpenAI/Anthropic-compatible API for authorized personal accounts. Account pooling, quotas, proxy configuration, login, and original management UIs remain with mature upstream sidecars. **Do not recreate a ModelAtlas-style model-provider aggregator.**
- Hub UI: `http://127.0.0.1:8317/ui`.
- Unified API: `http://127.0.0.1:8317/v1`.
- Android/Termux first; small VPS later. Keep loopback-only bindings.

## Current integration status (2026-10-08)

| Provider | Model prefix | Packaging | Real Android verification |
|---|---|---|---|
| OpenCode Free / opencode2api v1.3.7 | `opencode/` | default | previous inference evidence |
| FreeBuff / Freebuff2API v0.10.3 | `freebuff/` | default | retest accounts/inference |
| Agent2API v2.9.6 (including separate WorkBuddy domestic/international) | `agent2api/` | included in prebuilt | prior Qoder/tool-calling evidence; retest v2.9.6 |
| DeepSeek2API | `deepseek/` | optional, bundled | not yet established |
| Grok2API | `grok/` | optional, bundled | not yet established |
| Kiro-Go | `kiro/` | optional, bundled | not yet established |
| LMArena external | `lmarena/` | disabled localhost slot, no bridge bundled | not yet established |

Other candidates (WindsurfAPI, Qwen2API_Go, Kimi2API, Gemini2API and others) are a **research backlog**, not implemented providers. See `../AI_HANDOFF.md`.

The known-good previous Android packaging baseline was source commit `ea52255a8a1c2093e045cb0835aa9f6f838891c5`, with CI and Android build successful. On 2026-10-08 a non-destructive prebuilt upgrader and corresponding packaging changes were added. **Check the newest Actions run and `prebuilt/source-commit.txt` before declaring that newer bundle verified.** Never equate successful compilation with real-device execution.

## What to do before the user tests

1. Check `main` HEAD, CI, Android ARM64 build, and public `prebuilt/source-commit.txt` against the source commit that triggered the build.
2. Review `scripts/install-prebuilt-termux.sh` for first installs and `scripts/upgrade-prebuilt-termux.sh` for existing installs. Never delete user account data or local secrets to upgrade.
3. On the user's real phone, verify the installer/upgrade, `./scripts/run-termux.sh`, `./scripts/doctor-termux.sh`, `./scripts/smoke.sh`, Hub UI and provider-original management UIs.
4. Verify model discovery, real streaming inference, full tool-calling loop, and sidecar failure/restart. First retest existing OpenCode, FreeBuff, Agent2API/Qoder; then enable optional sidecars one by one.
5. Record real device RSS, provider readiness, account usability, model routing, success/failure cases and logs (with credentials redacted).
6. Fix only reproduced defects. Maintain a clear distinction between code written, CI green, Android bundle built, and real-device verified.

## Stable architecture rules

- Preserve each upstream's original account management UI. Hub is a compact mobile-first status foyer, not a full replacement control panel.
- Model IDs are always `provider/model`. Cross-provider balancing/fallback is **off by default**; when enabled, same-model matching is strict and a single request does not race multiple providers.
- Local secrets belong under ignored `data/` and must not enter Git or public diagnostic output.
- Keep explicit dependency versions / pinned revisions and automated Go, shell, JSON and Android packaging checks.

## References

- [Main handoff](../AI_HANDOFF.md)
- [README and install / upgrade instructions](../README.md)
- [Android on-device checklist](ON_DEVICE_CHECKLIST.md)
- [Provider notes](PROVIDERS.md)
- [Android ARM64 workflow](../.github/workflows/build-android-arm64.yml)
