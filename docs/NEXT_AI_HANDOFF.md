# NEXT AI HANDOFF — AhB (2026-10-09 release checkpoint)

> **Canonical long-form history:** [AI_HANDOFF.md](../AI_HANDOFF.md). Historical notes there remain for audit only; use the latest dated release gate below, not older v2.9.6 / v2.9.7 / Rust FreeBuff snapshots. Recheck GitHub Actions and release pointers if HEAD changes.

## Repository and objective

- Repository: [Tsenjii/AhB](https://github.com/Tsenjii/AhB), default branch `main`.
- Aim: lightweight account-backed API Hub, original upstream gateway management UIs, strict `provider/model` routes, login/account pools and quota logic delegated to each supported sidecar. No ModelAtlas-style discovery product or duplicate CLIProxyAPI implementation.
- Hub UI: `http://127.0.0.1:8317/ui`; local API: `http://127.0.0.1:8317/v1`. Loopback-only by default.
- Two **native** distributions from one codebase: Android Termux ARM64 (`prebuilt` branch) and GNU/Linux AMD64/ARM64 (independent release). Never interchange packages.

## Exact published baseline

- Merged main commit: `5841235cde0465fc29bec06fc584d4526d971c47` (2026-10-09).
- [Source CI success](https://github.com/Tsenjii/AhB/actions/runs/37921364712).
- [Android ARM64 build success](https://github.com/Tsenjii/AhB/actions/runs/37921364600); `prebuilt/source-commit.txt` matches this exact commit.
- [Linux AMD64/ARM64 build success](https://github.com/Tsenjii/AhB/actions/runs/37921364637); [native Linux release](https://github.com/Tsenjii/AhB/releases/tag/linux-5841235cde04) is published, with checksums.
- Device state remains **UNVERIFIED** for this exact release: no evidence the user's phone has already upgraded or the 512 MiB VPS has been deployed and survived OOM/peak-RSS tests. Build green ≠ account eligible ≠ real chat/SSE/tool completion.

## Actual packaging / architecture

Nine bundled gateway implementations, each configured enabled but started on demand:

| Prefix | Gateway | Platform/runtime | Account / inference status |
| --- | --- | --- | --- |
| `opencode/` | OpenCode2API | Go | Earlier Android real chat and SSE confirmed; new release and complete tool roundtrip need retest |
| `freebuff/` | yutian81/freebuff2api | Node.js >=20 | **Replaces Rust v0.10.3**; new CLI/Bearer login required; no old Rust WebUI |
| `agent2api/` | Agent2API pinned v2.9.8 | Rust, unchanged upstream | Previously saw account credentials but insufficient quota; new release untested |
| `deepseek/` | DeepSeek2API | Go | Live Android auth/chat/tools not verified |
| `grok/` | Grok2API | Go | Live Android auth/chat/tools not verified |
| `kiro/` | Kiro-Go | Go | Live Android auth/chat/tools not verified |
| `copilot/` | Copilot2API | Go | Requires explicit authorized GitHub device login; inference unverified |
| `geminiweb/` | Gemini Web2API | Go | Bundled, inference unverified |
| `duckai/` | Duck2api | Go | Bundled, inference unverified |

Do **not** call optional Kimi Web, LMArena, Windsurf, Qwen or generic `gemini/` bridges installed. They are older optional installers or connector presets, not extra native binaries. Agent2API subplatform names are not separate installed services.

Resource defaults: Android up to **3** resident on-demand sidecars, **900 s** idle; Linux up to **1**, **120 s** idle. The Hub settings can change process limits and show system/cgroup RAM plus approximate Hub + child RSS. A process count limit is *not* a hard RAM cap.

## Safe next action

1. Preserve user credentials, `config.json`, `data/`, and existing backups (including `~/AhB.backup-20261008-194847`). Do not re-run fresh installation over an existing installation.
2. Android: follow [README](../README.md) and run the published safe `scripts/upgrade-prebuilt-termux.sh` in Termux. Check `doctor-termux.sh`, `provider-status-termux.sh`, `smoke.sh` and local dashboard before account-backed tests. The bundle source SHA must match the release checked above.
3. FreeBuff: check `check-freebuff-login.sh`; for no new CLI accounts, authorize via `freebuff-login-termux.sh`. **Do not** reuse old Rust cookie sessions. Then test one truly listed, authorized model.
4. Linux 512 MiB: deploy **only** the Linux-native release using [VPS instructions](DEPLOY_VPS_512MB.md); capture `/api/runtime`, peak RSS, idle stop, cold wake, SSE lease, and OOM evidence. No public port `8317`.
5. Verify OpenCode known-good chat → complete SSE `[DONE]` → two-turn structured tool result first; then add one new source at a time. Real model tests consume account quota and must be explicitly initiated by the user.
6. If failures appear, patch **AhB** only, open our own issue/PR, rerun CI and release gate; do not contact or change upstream projects without request.

## Reference files

- [Main handoff and chronological changes](../AI_HANDOFF.md)
- [Current README and update commands](../README.md)
- [Android acceptance](NEXT_PHONE_ACCEPTANCE.md)
- [Linux 512 MiB guide](DEPLOY_VPS_512MB.md)
- [FreeBuff CLI login](FREEBUFF_ANDROID_LOGIN.md)
- [Upstream bridge vs bundled boundaries](BRIDGES.md)
