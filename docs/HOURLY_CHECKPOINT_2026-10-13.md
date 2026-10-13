# AHB hourly handoff — 2026-10-13 18:47 (Asia/Taipei)

## Current source of truth
- Repository: https://github.com/Tsenjii/AhB ; default branch `main`.
- Latest inspected main source: `3d2e49500d16b808dd7df68bb83d27214a48287b`.
- Last **confirmed published Android prebuilt**: `prebuilt/source-commit.txt` = `ba3448c98e15b607d5ed2e4c1d4a6945a2d7df03` at this checkpoint. Published Linux release matching that baseline: https://github.com/Tsenjii/AhB/releases/tag/linux-ba3448c98e15
- Do not confuse merged main source, a successful CI, a built artifact, a published package, an installed Android phone, or a real-account chat test. They are separate acceptance levels.
- Current source has **not** yet been confirmed published to `prebuilt`, or installed and exercised on the user's Android phone. `README.md` latest native release section still points to an older `0bf0ba3727fd` snapshot and should be synchronized only after verifying release assets and checksums.

## Completed this inspection
1. [PR #51](https://github.com/Tsenjii/AhB/pull/51) merged as `835fcf465fe82dbea9a8ba9a8fee900360e67258`. Playground batch cancel now aborts pending provider and model-catalog fetches, reports intentional cancellation without a generic error, and restores controls. Branch CI passed: https://github.com/Tsenjii/AhB/actions/runs/38357659122
2. [PR #52](https://github.com/Tsenjii/AhB/pull/52) merged as `03e38ca1b4e2dbad343c4f33939d716819b6cd52`. The GUI and private CLI localhost bridge probes reject malformed or extra JSON, oversize catalogs, invalid model IDs and catalogs above 1,000 entries. Empty `data: []` remains valid; no unknown bridge enters failover automatically. Branch CI passed: https://github.com/Tsenjii/AhB/actions/runs/38386034416 ; merged-main CI for this commit passed: https://github.com/Tsenjii/AhB/actions/runs/38386369936
3. [PR #53](https://github.com/Tsenjii/AhB/pull/53) merged as `3d2e49500d16b808dd7df68bb83d27214a48287b`. Improved Playground touch sizes, labels/paragraph typography, and coarse-pointer targets (48px), with static accessibility regressions inside the embedded dashboard fixture. Branch CI passed: https://github.com/Tsenjii/AhB/actions/runs/38386332762
4. Read official UI/UX Pro Max skill documentation https://github.com/nextlevelbuilder/ui-ux-pro-max-skill and its mobile touch/contrast checklist. No new JS frameworks or always-on services introduced.

## Latest merged-main verification (as of handoff write)
- Main `3d2e49500d16` Go CI: https://github.com/Tsenjii/AhB/actions/runs/38386539043 — **running** when checked, not yet passed.
- Main Android ARM64: https://github.com/Tsenjii/AhB/actions/runs/38386539055 — **pending** when checked.
- Main Linux AMD64/ARM64: https://github.com/Tsenjii/AhB/actions/runs/38386539015 — **pending** when checked.
- Optional Z.ai Web bridge validation: https://github.com/Tsenjii/AhB/actions/runs/38386539158 — **in progress** when checked.
- Branch CI and browser-free DOM fixtures are not equivalent to true Android simulator/e2e testing.

## Unfinished; next AI action order
1. Check the **final** results for the four newest merged-main Actions, resolve any failures and re-run appropriate checks. Confirm exact `head_sha`.
2. After both native workflows publish successfully, confirm `prebuilt/source-commit.txt` matches the built main SHA, verify Android archive checksum and Linux release assets, then synchronize release pointers in `README.md` and this handoff. The Docker Compose default pin is intentionally older until separately verified; do not silently change it.
3. Run available UI smoke tests and—when an Android emulator/device is genuinely available—verify 375px/landscape mobile layout, coarse-pointer controls, Playground cancel during hung catalog fetch, and no accidental inference/quotas.
4. Prioritize real-account onboarding and data-preserving fixes: FreeBuff 502/login, OpenCode 503, DeepSeek incomplete real inference and Duck.ai 418. See issues #3, #5, #27, #35. Do not emulate account entitlements, hide blocked access, bypass upstream auth, or claim health == quota.
5. Verify authorized real inference, tool roundtrip, streaming `[DONE]`, quotas, running memory/RSS and failover behavior individually before declaring any new connector production-ready.
6. Keep both Android Termux ARM64 and native Linux AMD64/ARM64 deployment guidance accurate. Never overwrite `data/`, `config.json`, existing accounts or backups.

## Product scope
AHB is a lightweight authorized account and API hub with optional localhost bridges, not a ModelAtlas-style general provider index. Preserve on-demand sidecars, original upstream login flows, local security boundaries and opt-in fallback.
