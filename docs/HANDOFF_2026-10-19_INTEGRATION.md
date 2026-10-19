# AHB integration handoff — 2026-10-19 (Asia/Taipei)

## Scope and immutable guardrails
AHB remains a lightweight authorized-account / local Gateway API Hub for native Android/Termux ARM64 and Linux AMD64/ARM64. It manages its own sidecar lifecycle, health, explicit provider/model routes, streaming, API ingress and optional **same-model, explicitly allowlisted** failover. Existing upstream gateways retain their account pools, sign-in, quota logic and provider-specific proxy configuration. Do **not** broaden the product into ModelAtlas, auto-enroll unknown external providers, attempt unauthorized account access, bypass quotas, or write any user keys, cookies or backups to the repository.

## Integration performed and verified on GitHub
All seven PRs opened against main at the start of this checkpoint were merged with GitHub's own reported `merged:true` response:

| PR | Topic | Merge commit |
| --- | --- | --- |
| [#59](https://github.com/Tsenjii/AhB/pull/59) | Pin-preserving Agent2API moved-origin fix | [fc4400a](https://github.com/Tsenjii/AhB/commit/fc4400a5d7f68185abcb1449416cdb4a25f04410) |
| [#56](https://github.com/Tsenjii/AhB/pull/56) | Fail-closed fallback allowlist + nondestructive config migration tests | [3dcf8fa](https://github.com/Tsenjii/AhB/commit/3dcf8fa750ace3ec77b4192292f277d767d5fc49) |
| [#55](https://github.com/Tsenjii/AhB/pull/55) | Playground missing CSS brace + 16px iOS input + CSS test | [81ee0f5](https://github.com/Tsenjii/AhB/commit/81ee0f5671ab28995987f073a914e7cebfb63cc1) |
| [#57](https://github.com/Tsenjii/AhB/pull/57) | Playground stale catalog, batch and control race guards | [af901d4](https://github.com/Tsenjii/AhB/commit/af901d4afbd1d9d4221dc35c7cc512dbb2483445) |
| [#60](https://github.com/Tsenjii/AhB/pull/60) | Five mobile bottom tabs, direct Playground shortcut | [5ffb809](https://github.com/Tsenjii/AhB/commit/5ffb80903e5780da82eaba33a5f0c7427e2e00a3) |
| [#58](https://github.com/Tsenjii/AhB/pull/58) | Previously verified 3d2e495 published version and source pointers in README + release docs regression | [27f09c5](https://github.com/Tsenjii/AhB/commit/27f09c53fe9b678694b805cdb7b844457b0f50a9) |
| [#54](https://github.com/Tsenjii/AhB/pull/54) | Historical lightweight proxy compatibility research docs | [3f7713f](https://github.com/Tsenjii/AhB/commit/3f7713fe9a224b4d7690242ca71ec7375cb1a4e5) |

Main source HEAD immediately before this handoff file: `3f7713fe9a224b4d7690242ca71ec7375cb1a4e5`.
Verified post-merge main contains the combined dashboard regression harness invoking `test-playground-css.py`, `test-playground-race.mjs`, `test-playground-controls.mjs` and five-tabs assertions. Source also contains the 16px Playground inputs and mobile media rules.
No platform/package release is implied by an ordinary PR merge.

## Test evidence (do not overstate)
- [PR #59 exact-head Go CI](https://github.com/Tsenjii/AhB/actions/runs/39200191092), [Android ARM64](https://github.com/Tsenjii/AhB/actions/runs/39200191076) and [Linux AMD64/ARM64](https://github.com/Tsenjii/AhB/actions/runs/39200191074) all completed **success**, before merging other patches.
- Other exact-head PR Go CI reports success: [#56](https://github.com/Tsenjii/AhB/actions/runs/39041797998), [#55](https://github.com/Tsenjii/AhB/actions/runs/39041977081), [#57](https://github.com/Tsenjii/AhB/actions/runs/39042225697), [#60](https://github.com/Tsenjii/AhB/actions/runs/39200474408), [#58](https://github.com/Tsenjii/AhB/actions/runs/39200970760), [#54](https://github.com/Tsenjii/AhB/actions/runs/39042442286). Native build failures on those older branches predated their integration with #59; do not label them final-main failures.
- [Main CI at 5ffb809](https://github.com/Tsenjii/AhB/actions/runs/39224729605) and [optional Z.ai fixture](https://github.com/Tsenjii/AhB/actions/runs/39224729637) completed **success** after merging the five code PRs.
- As checked during this handoff, [newer main CI at 3f7713f](https://github.com/Tsenjii/AhB/actions/runs/39224865829) was **in progress**. [Native Android](https://github.com/Tsenjii/AhB/actions/runs/39224850716) and [Linux](https://github.com/Tsenjii/AhB/actions/runs/39224850763) jobs on the 27f09c5 integration baseline were **in progress**. Recheck their exact conclusions and published artifacts; do not infer PASS from running status.
- Intermediate native runs on superseded commits were cancelled while new commits arrived. Cancellation is not the same as a compile failure.
- This agent could not clone locally: `git clone` failed with `Could not resolve host: github.com`. **No independent local Go, Node, emulator, real Safari, phone or live authenticated inference test was executed in this turn.**
- Existing released native 3d2e495 artifacts were last verified before this integration. Need verify any subsequent real release against SHA256 and `prebuilt/source-commit.txt`.

## UI/UX review
Reviewed public [UI/UX Pro Max skill](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill/blob/main/.claude/skills/ui-ux-pro-max/SKILL.md) before accepting merged fixes. The relevant checks are <=5 bottom nav items, >=44px touch targets, labels, focus visibility, mobile safe areas, reduced motion and mobile input font >=16px. Static regression and source inspection are evidence only of intended implementation, not real iPhone/Android appearance or assistive-technology acceptance.

## Remaining defects / issues
- [#3 FreeBuff 409 masked as 502](https://github.com/Tsenjii/AhB/issues/3) — still requires authorized account, upstream error provenance and live regression; no stealth bypass.
- [#5 OpenCode intermittent Android 503](https://github.com/Tsenjii/AhB/issues/5) — separate Hub readiness-generated 503 from upstream provider 503; run direct chat/SSE/tools with sanitized diagnostics.
- [#27 one-stop account onboarding](https://github.com/Tsenjii/AhB/issues/27) — existing local UI/nonnative management shortcuts are not real SSO; acceptance remains provider-specific.
- [#35 experimental DeepSeek/Duck reliability](https://github.com/Tsenjii/AhB/issues/35) — disabled by default; do not advertise unverified web inference.
- Android actual login/quota, SSE `[DONE]`, full two-turn tools, low-memory peak RSS/OOM, Linux sidecar failover, account-pool behavior, and real Safari/Android UI smoke **unverified**. Do not mutate production credentials or update user's phone without informed user action.

## Next concrete operations
1. Check `main` HEAD CI and native Actions above to completion; inspect failing **job steps** and fix any actual regressions using a new narrow PR + exact-head CI.
2. Inspect `prebuilt/source-commit.txt` and Linux release asset checksums; publish or install only after packaging and provenance checks.
3. On a credential-safe test host, verify explicit empty and provider-specific allowlists, quota/status surfaces, failover no-stream-replay, SSE framing, tool roundtrip; on a real device, measure RSS and UX.
4. Triage issues #3, #5, #27 and #35 with sanitized evidence, keeping Gateway sign-in and account pools inside the source adapters.
5. Preserve original user `config.json`, private session files, keys, SQLite and backups. Keep integration details here so future agents do not reopen already merged PRs.

Latest handoff is a snapshot, not a self-updating CI badge; verify newer status before claiming completed release or phone acceptance.
