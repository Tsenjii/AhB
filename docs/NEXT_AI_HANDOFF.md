# NEXT AI HANDOFF — AhB (2026-10-09 release checkpoint)

> **Canonical long-form history:** [AI_HANDOFF.md](../AI_HANDOFF.md). Historical notes there remain for audit only; use the latest dated release gate below, not older v2.9.6 / v2.9.7 / Rust FreeBuff snapshots. Recheck GitHub Actions and release pointers if HEAD changes.

## Repository and objective

- Repository: [Tsenjii/AhB](https://github.com/Tsenjii/AhB), default branch `main`.
- Aim: lightweight account-backed API Hub, original upstream gateway management UIs, strict `provider/model` routes, login/account pools and quota logic delegated to each supported sidecar. No ModelAtlas-style discovery product or duplicate CLIProxyAPI implementation.
- Hub UI: `http://127.0.0.1:8317/ui`; local API: `http://127.0.0.1:8317/v1`. Loopback-only by default.
- Two **native** distributions from one codebase: Android Termux ARM64 (`prebuilt` branch) and GNU/Linux AMD64/ARM64 (independent release). Never interchange packages.

## Verified release checkpoint — 2026-10-09

- [PR #24](https://github.com/Tsenjii/AhB/pull/24) merged into main `570ef0e7af7eceb06bf428f71a5d76259782acb8`: local single-Gateway recovery with 45s cooldown and active-SSE rejection; credentialless best-effort per-sidecar egress Proxy setting and mobile UI. CI, Android ARM64, native Linux AMD64/ARM64 all passed; Android `prebuilt/source-commit.txt` matches exactly; [Linux release](https://github.com/Tsenjii/AhB/releases/tag/linux-570ef0e7af7e) includes checksums.
- Agent2API is pinned at **v2.9.8** `f82308a9549ee4760f27e393058c3482056ec514` in both native builds, currently matching upstream main/release. Do not replace native per-account proxy pools with a Hub-level duplicate.
- Portable Docker/Compose is Linux AMD64/ARM64 using this published release, private ingress credentials and persistent `/state`. See [Docker guide](DOCKER.md). macOS and Windows are Docker Desktop compatible for the Linux image; there are **no verified native Darwin or Windows AhB binaries**.
- GitHub source/packaging checks are green; the user's actual Android phone update, account quota, end-to-end OpenCode/Muse, FreeBuff login and full real-tool SSE/RSS remain **unverified**.

## Current workstream: core reliability and portable Docker

- All temporary platform-specific Python runners and workflows are removed; generic Docker Compose with verified Linux release, local ingress auth and persistent state is the supported portable container path. See [Docker guide](DOCKER.md).
- Core reliability change: plain `GET /v1/models` must **not wake** a sleeping gateway, but probing an already running gateway must hold an existing-process lease until its model response completes; this prevents idle reaping or one-provider-cap eviction during a list request. See `internal/hub/on_demand.go` and `internal/hub/model_probe_lease_test.go`.
- Added Hub-level synthetic regression for a **native two-turn structured Tool Calling flow**, preserving nested upstream model IDs, tool call IDs and matching tool result; and synthetic full SSE `[DONE]` under `max_running_sidecars=1`, which must keep the active source alive. These tests are protocol/lease fixtures, **not proof that upstream OpenCode/Muse quotas or device authentication work**.
- Continue to prioritize actual OpenCode on-device chat, SSE finish and two-turn native tools first, FreeBuff fresh authorized CLI/Bearer login second, other nine bundled sources later. Preserve existing phone credentials and backups. A cloud build/test is not a real phone or 512 MiB VPS RSS acceptance test.

## Latest development checkpoint — 2026-10-09

- [PR #17](https://github.com/Tsenjii/AhB/pull/17) was merged as `4e31217b06d780e9e47e10b6e06ca7433816855d`. Main CI, Android ARM64 and native Linux build passed; the Android `prebuilt/source-commit.txt` moved to that SHA and Linux release `linux-4e31217b06d7` was published.
- [PR #18](https://github.com/Tsenjii/AhB/pull/18) adds bounded, private **model metadata only** caching to support sleeping on-demand providers. Its explicit dashboard **逐一載入全部模型** action wakes providers one-by-one, discovers models inside the lease, and leaves the 512 MiB one-process ceiling intact. `GET /v1/models` does NOT wake everything; a sleeping provider reports `x_cached: true`, `x_cache_state: sleeping_unverified` if a successful model listing is cached within six hours, otherwise remains unavailable to discovery. Models marked cached must still pass runtime/account checks on inference and cannot make an unverified `route/*` alias routable. Read-only Termux status prints `CACHE:n` versus `SLEEP`. Cache file `data/hub-model-catalog.json` is owner-only, atomic, capped at 2 MiB, with allowlisted metadata and no credentials.
- **Release gate for PR #18:** Check the latest *exact head* GitHub CI and both ARM64 + Linux build workflows. Only after merger verify new `main`, Android prebuilt source SHA and matching Linux release tag. Do not confuse PR packaging, deployed phone version, available Muse upstream models, authenticated FreeBuff or 512 MiB real OOM/SSE/tool testing.
- Never delete previous AhB backups, change upstream repositories or expose credentials. Keep OpenCode at its pinned, separately tested upstream commit unless a deliberate migration is approved.

## Previous published baseline (before PR #18)

- Previously verified merged main commit: `4e31217b06d780e9e47e10b6e06ca7433816855d` (2026-10-09).
- [Source CI success](https://github.com/Tsenjii/AhB/actions/runs/37929299383).
- [Android ARM64 build success](https://github.com/Tsenjii/AhB/actions/runs/37929299453); `prebuilt/source-commit.txt` matches this exact commit.
- [Linux AMD64/ARM64 build success](https://github.com/Tsenjii/AhB/actions/runs/37929299387); [native Linux release](https://github.com/Tsenjii/AhB/releases/tag/linux-4e31217b06d7) is published, with checksums.
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
