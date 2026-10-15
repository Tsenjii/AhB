# AHB AI handoff — 2026-10-15 21:38 Asia/Taipei

## Goal and scope
Maintain the lightweight Go account/API hub, authorized sidecars, quota/health visibility and direct model routing. No ModelAtlas-style provider discovery or authentication bypass.

## Verified repository state
- Main: `42d4c7c46b3bed139e4abef23a204b06cc3abe35`.
- UI work branch: `fix/playground-reload-errors-20261015-1933` at `56f963202d302a241f090c6f2206ca5af2245ce1` before this documentation commit.
- Separate proxy-watch Draft PR #54: https://github.com/Tsenjii/AhB/pull/54
- Proxy-watch CI #38736194274: completed **SUCCESS** on `512d910fd3cdf0607c314d401f528aa88a1c99a1`. This verifies documentation-branch CI only, not a Playground fix: https://github.com/Tsenjii/AhB/actions/runs/38736194274
- UI branch CI #38728310707 passed on `56f9632`, prior to the still-uncommitted reload/CSS changes.

## Reproduced Playground defects
Read `internal/hub/playground_ui.go` on the work branch and executed its embedded JavaScript with a mock DOM and deterministic `/v1/models` responses:
1. Initial models request: HTTP 200.
2. User selects “啟動並載入此來源模型”; second models request: HTTP 503.
3. **Original:** two requests; UI incorrectly says “已刷新模型清單”; CSS class `status`.
4. **Candidate (not committed):** two requests; UI says “來源清單讀取失敗：HTTP 503”; CSS class `status bad`.
5. **Candidate success control:** 200 then 200; success message retained.
6. Original CSS `button,.btn` rule lacks closing `}`; candidate CSS braces balanced at 109/109, embedded JavaScript parses.
7. Optional narrow-screen input/select/textarea 16px font change is proposed but not real-device verified.

Candidate edits: close the button CSS rule; make `reload()` return a boolean and set `status bad` on failure; have `loadModels` display success only when reload returned true; add 16px input sizing under 600px; add regression test to `scripts/test-playground-batch.mjs` and CSS guard to `scripts/test-dashboard.sh`.

## Actual changes and blockers
- Code update through connected GitHub `update_file` was attempted and failed (connector returned an error without a usable message). **No source fix commit, PR, or fix CI has been created.**
- Local `git ls-remote` failed: `Could not resolve host: github.com`. Full checkout/build and real browser/Android testing are not available locally.
- Installed skill inventory was empty; reviewed official UI/UX Pro Max skill from https://github.com/nextlevelbuilder/ui-ux-pro-max-skill and avoided duplicating existing touch/focus/safe-area/reduced-motion support.
- Existing 20 KiB Playground guardrails are on `56f9632`; do not claim they are merged to main.
- The above JavaScript test is a **mock-DOM interaction smoke**, not full Go CI or device acceptance.

## Next AI: prioritized execution
1. Re-read current branch HEAD and UI file before editing. Apply only the missing targeted CSS/reload fix and its regression test.
2. Run `bash scripts/test-dashboard.sh`, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/hubd`; verify exact-SHA Actions result.
3. Browser smoke at 375, 768, 1440px; verify 200→503, 200→200, keyboard and focus states; Android ARM64 remains unverified.
4. Address OpenCode intermittent 503, FreeBuff authorized login, and safe SSE/429/502/503 account-pool failover using mock upstreams before live credentials.
5. Do not merge unrelated Draft PR #54 into this UI branch.

## Test status
- Mock-DOM reload 200→503 and 200→200: **PASS for candidate, regression reproduced on original**.
- Candidate CSS brace count and JS parse: **PASS**.
- New Go build/tests/race, full browser, phone, 512 MiB VPS RSS: **NOT RUN**.
