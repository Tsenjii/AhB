# AHB AI handoff — 2026-10-14 23:35 Asia/Taipei

## Scope
Keep AHB a lightweight, authorized multi-account API hub. Do not turn it into a general ModelAtlas provider index. Preserve local-only management, original upstream login flows, private keys, quotas and backups.

## Verified current baseline
- Main: `42d4c7c46b3bed139e4abef23a204b06cc3abe35`.
- Main CI: [run 38564569257](https://github.com/Tsenjii/AhB/actions/runs/38564569257), success for *unchanged main*.
- Android prebuilt/source-commit.txt: `3d2e49500d16b808dd7df68bb83d27214a48287b`, not main.
- Linux release: [linux-3d2e49500d16](https://github.com/Tsenjii/AhB/releases/tag/linux-3d2e49500d16).
- Open issues: #3 FreeBuff 409/502, #5 OpenCode intermittent 503, #27 account onboarding, #35 DeepSeek/Duck reliability. Open PRs: none.

## Observed source bugs and proposed focused fix
In `internal/hub/playground_ui.go` main:
1. CSS `button,.btn` rule is missing a closing `}` (107 opens / 106 closes), affecting following rules.
2. `reload()` catches a model-list failure without returning a status; wake/load handler unconditionally displays a success message even after HTTP 503.

Proposed change: close the CSS rule; return explicit booleans from reload; preserve error status and only show refresh success after successful reload. Browser JS syntax and CSS brace-count preflight passed in the 2026-10-14 23:35 local analysis. Full repo CI and Android UI smoke have **not** run on modified source.

## Pending older unmerged local patches
SSE [DONE]/CR/CRLF framing and reader cleanup; bounded nonstream success/error reads; upstream redirect/header isolation. Earlier synthetic fixtures are not full repo CI evidence. Merge and test patches incrementally, checking overlap.

## Next actions
1. Integrate the two focused Playground fixes into a feature branch; verify Go tests, race, vet, JS/browser fixture, 375px and 1440px smoke.
2. Check push and PR permissions; if blocked by tool safety controls, do not claim changes were committed.
3. Validate SSE and bounded-response patches against the current full source, then merge in small revertible units.
4. Rebuild Android and Linux only after modified CI passes; verify release source SHA and checksums.
5. Use authorized accounts for true quota, SSE, tool roundtrip and failover acceptance. Never infer inference availability from model listing.
