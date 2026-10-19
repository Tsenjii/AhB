# AHB — AI integration handoff (2026-10-17, Asia/Taipei)

## Goals and scope
Maintain AhB as a lightweight, authorized-account/API hub (native Android and Linux, optional local bridges, quota/status visibility and explicit failover). Do not turn it into a ModelAtlas-wide catalog. Never claim local health checks prove inference/quota availability, and preserve user config, backups and private credentials.

## Verified baseline
- Repository: https://github.com/Tsenjii/AhB ; main `42d4c7c46b3bed139e4abef23a204b06cc3abe35` at this checkpoint.
- Current published Android/Linux source baseline (previously verified): `3d2e49500d16`; Linux release https://github.com/Tsenjii/AhB/releases/tag/linux-3d2e49500d16 . This is **not** a release of the PRs below.
- Open user issues: #3 (FreeBuff), #5 (OpenCode 503), #27 (login UX), #35 (DeepSeek/Duck); docs PR #54 is open.

## Code completed and actually pushed (NOT merged to main)
1. [PR #55](https://github.com/Tsenjii/AhB/pull/55) `fix/playground-css-focus-20261017-1235`, head `611ebb354607e45200df8a3211ef66eb5c667a77`: close missing Playground CSS rule, use iOS-friendly 16px form controls, install a static CSS/accessibility regression guard in dashboard tests. [Exact-head Go CI](https://github.com/Tsenjii/AhB/actions/runs/39041977081) SUCCESS. Real Safari rendering unverified.
2. [PR #56](https://github.com/Tsenjii/AhB/pull/56) `fix/fallback-allowlist-20261017-1441`, head `a54ad63eff014b45135bbccea64ed82ac1e8ef48`: runtime respects an explicit empty/listed same-model fallback allowlist; config migration no longer auto-adds providers to explicit lists and rejects malformed values without overwriting config; Go and migration fixtures committed. The first implementation caused an unmatched shell quote in a jq comment; corrected in `a54ad63`. [Exact-head Go CI](https://github.com/Tsenjii/AhB/actions/runs/39041797998) SUCCESS.
3. [PR #57](https://github.com/Tsenjii/AhB/pull/57) `fix/playground-catalog-races-20261017`, head `c0f6bdc43eef93e16bc5f9b9ebf5b1e7ba94b210`: prevent stale model catalog responses/errors overwriting new state; retain provider selection; invalidate old requests across batch tests; prevent wake handler or selection changes from prematurely re-enabling controls. Added two source-extracted Node/V8 fixtures to dashboard CI. [Exact-head Go CI](https://github.com/Tsenjii/AhB/actions/runs/39042225697) SUCCESS. These are DOM fixtures, **not** Safari/browser/device evidence.
- All three PRs are still draft and unmerged. Each branch has actual remote commits; do not reapply earlier checkpoint patch ZIPs to these branches.

## Build and test failures — evidence and limitations
- PR #55, #56 and #57 each have successful exact-head `CI` (Go tests, race, vet, build, existing shell/dashboard fixtures and newly added guards).
- Android ARM64 and Linux AMD64/ARM64 **bundle build workflows FAIL**, e.g. [Android on PR #56](https://github.com/Tsenjii/AhB/actions/runs/39041797925), [Linux on PR #56](https://github.com/Tsenjii/AhB/actions/runs/39041798083). Logs show `git fetch --depth 1 origin "$AGENT_SHA"` for pinned `https://github.com/aimod-cc/agent2api.git` returns `fatal: could not read Username for 'https://github.com': No such device or address`. A separate GitHub API repository check returned HTTP 404. This is an upstream source access blocker, **not proof the patched AhB runtime fails to compile**. Never publish partial packages or silently replace the pinned binary with an unverified fork.
- Local execution environment cannot resolve `github.com` for `git clone`. Exact-head Go CI is GitHub Actions evidence, not a complete local checkout test.
- No real authenticated model inference, usable quota, SSE/tool continuation, physical Android, iPhone Safari, or 512 MiB peak RAM validation was performed. No Android/Linux package based on these PRs was published.

## UI/UX review performed
Reviewed UI/UX Pro Max guidance (accessibility, 44px touch, keyboard focus, iOS form text, responsive viewports, reduced motion). Kept existing navigation and responsive layouts; fixed observable CSS and asynchronous interaction regressions only. Avoid duplicating existing controls.

## Next actions for next AI
1. Recheck PR heads and exact CI evidence. Review #55/#56/#57 together and resolve conflicts if merging sequentially; require truthful reporting of package-build failures.
2. Restore trusted access to pinned Agent2API source or a separately reviewed, verifiable upstream replacement before claiming Android/Linux bundle builds pass. Do not bypass authentication/access controls, change credentials, or ship unvalidated binaries.
3. Run physical Safari/Android UI smoke plus live account/SSE/quota acceptance only in an authorized environment; never report it as completed from fixture tests.
4. Fix any new Go/Dashboard CI regressions at their exact PR heads. Align README release information after verified new packages; keep release SHA and branches distinct.
5. Continue #3/#5/#27/#35 by improving diagnostics and honest account readiness, not by expanding gateway catalogs or bypassing upstream rate limits.
