# AhB Playground regression handoff — 2026-10-17 00:38 Asia/Taipei

## Verified repository baseline
- Main: `42d4c7c46b3bed139e4abef23a204b06cc3abe35`.
- Playground blob: `a07145fff0d6ba32fcac9ba1939fb9c47d8f9150`.
- Open issues: #3, #5, #27, #35. Open PR #54 is documentation-only.
- Latest main CI checked: https://github.com/Tsenjii/AhB/actions/runs/38924760827 — success on unchanged main.

## Reproduced defect and local candidate
- `internal/hub/playground_ui.go` has a missing CSS brace before `button.primary` and 15px mobile form controls.
- Overlapping `reload()` requests can render stale model catalogs and HTTP 503 errors.
- A stale on-demand wake result can overwrite a newer manual refresh status.
- Proposed change uses latest-request generation checks, busy control guards, and prevents stale wake success/error messages. Manual refresh remains available to supersede a stalled catalog request. Preserve the existing two-column desktop and single-column mobile layout.
- **Not committed:** GitHub connector blocked updating the production file with: `This tool call was blocked by OpenAI's safety checks.` Do not treat this branch as containing the fix.

## Test evidence (this inspection)
- Real Main embedded JS extracted and patched in memory: syntax PASS; CSS braces 107/107; mobile 16px.
- Real-source isolated DOM-mock async tests PASS: stale catalog success ignored; stale HTTP 503 ignored; stale wake success ignored; stale wake error ignored (4/4).
- Separate local synthetic Node fixture PASS. This is not a browser or Go integration test.
- **Not run:** complete Go build/race/vet, Android ARM64, Linux, iPhone Safari, 375px browser smoke, live accounts/quotas.

## Next AI
1. Re-fetch main and inspect diff; apply the small Playground patch only if blob is unchanged and repository write is permitted. Do not bypass safety checks.
2. Add persistent regression tests running the actual embedded script in a DOM mock; wire to CI. Check manual refresh while a previous request is stalled.
3. Run `bash scripts/test-dashboard.sh`, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/hubd`, then desktop/mobile browser smoke.
4. Keep authorized account/API management, original upstream authentication, on-demand sidecars and opt-in failover. Do not expand into ModelAtlas-style provider discovery.
