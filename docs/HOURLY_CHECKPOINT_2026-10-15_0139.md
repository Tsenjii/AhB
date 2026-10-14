# AHB development handoff — 2026-10-15 01:39 Asia/Taipei

## Product boundary
Lightweight local authorized account/API hub, not a ModelAtlas provider aggregator. Preserve original account flows, private `data/`, backups, opt-in failover, and the existing optional CLIProxyAPI localhost connector.

## Verified repository and release state
- Main source: `42d4c7c46b3bed139e4abef23a204b06cc3abe35`; [main CI 38386659427](https://github.com/Tsenjii/AhB/actions/runs/38386659427) **success**, but does not contain the proposed UI fix.
- Latest verified Android `prebuilt/source-commit.txt`: `3d2e49500d16b808dd7df68bb83d27214a48287b`; [Android workflow 38386539055](https://github.com/Tsenjii/AhB/actions/runs/38386539055) **success**.
- Matching [native Linux release](https://github.com/Tsenjii/AhB/releases/tag/linux-3d2e49500d16) contains amd64 and arm64 archives and checksum assets. No real-device smoke test was performed in this review.
- Main still has four open issues (FreeBuff login/502, OpenCode 503, onboarding, experimental DeepSeek/Duck reliability) and no open PR at review.
- `README.md` currently labels an older `0bf0ba3727fd` as latest native baseline; synchronize after checking exact release parity. Docker's older pin is deliberate.

## Reproduced Playground bugs (main source blob `a07145fff0d6ba32fcac9ba1939fb9c47d8f9150`)
1. `internal/hub/playground_ui.go` CSS `button,.btn` rule lacks a closing brace: **107 opening / 106 closing**. In a minimally modified source, braces balance **107/107**.
2. The `loadModels` click handler always displays “已刷新模型清單” after `reload()`, even if the catalog fetch returned HTTP 503. Executed the **real embedded JavaScript** inside an isolated mock DOM: original shows false success after second `/v1/models` call fails. Proposed `reload()` boolean return plus conditional success shows “來源清單讀取失敗：HTTP 503” with `status bad`. Normal success remains intact.
3. The modified embedded JavaScript passed V8 syntax compilation. These are isolated tests, not Go build, CI, Android emulator, or physical phone tests.

## Proposed small, reversible patch — NOT merged
- Close the missing `button,.btn` CSS rule.
- Return `false` when `batchActive` or `reload()` catches a fetch failure, and `true` on successful reload.
- Preserve `status bad` on failure; only display the post-wake success text if `await reload()` returns true.
- Local patch reference: `09_playground_css_refresh_combined.patch` from the previous handoff. Do not assume that patch is committed.
- GitHub `update_file` for the full `playground_ui.go` returned **“This tool call was blocked by OpenAI's safety checks.”** Do not bypass this restriction or claim a remote code fix.

## Acceptance gates and next steps
1. Recheck branch diff and CI before any PR/merge. A docs-only commit does not fix Playground.
2. Apply the narrowly scoped code patch only when permitted. Add browser-free fixture for HTTP 503 vs success, run `bash scripts/test-dashboard.sh`, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/hubd`.
3. Verify 375px and 1440px real browser layouts and focus/error status; no browser UI smoke performed here. Existing 44px/48px touch targets already implemented; do not duplicate.
4. Later review bounded response and SSE patches against current code; do not bundle unrelated changes into the Playground fix.
5. Before claiming Android/VPS release: exact source SHA, packaging workflow, prebuilt/release parity, device or VPS RSS, actual authorized inference, SSE [DONE], tool continuation, quotas, and failover evidence are separate checks.

UI guidance consulted: https://github.com/nextlevelbuilder/ui-ux-pro-max-skill/blob/main/.claude/skills/ui-ux-pro-max/SKILL.md (visible error feedback, touch targets, accessible status). 
