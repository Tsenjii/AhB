# AHB development handoff — 2026-10-16 02:39 Asia/Taipei

## Scope and goal
Maintain the lightweight, authorized account/API Hub, low-RAM sidecars, SSE, local API routing, quota visibility and safe failover. Do not build a ModelAtlas-style provider discovery system or bypass upstream authentication/anti-abuse controls.

## Source of truth and current status
- Main HEAD: `42d4c7c46b3bed139e4abef23a204b06cc3abe35`, unchanged since previous checkpoint.
- Open issues: [#3](https://github.com/Tsenjii/AhB/issues/3) FreeBuff 409/502, [#5](https://github.com/Tsenjii/AhB/issues/5) OpenCode 503, [#27](https://github.com/Tsenjii/AhB/issues/27) unified account onboarding, [#35](https://github.com/Tsenjii/AhB/issues/35) DeepSeek/Duck.ai stability.
- Open draft PR: [#54](https://github.com/Tsenjii/AhB/pull/54), proxy research documentation, not a runtime fix.
- Last checked Main Actions run [#38777987881](https://github.com/Tsenjii/AhB/actions/runs/38777987881) had Go job SUCCESS, including go test, race, vet, build and shell fixtures; it predates this checkpoint's proposed patches.
- `prebuilt/source-commit.txt` currently contains `3d2e49500d16b808dd7df68bb83d27214a48287b`; README still describes the older `0bf0ba3727fd` release. Docker pin is intentionally older and must not be changed without packaging verification.
- Read current README, `internal/hub/route.go`, `internal/hub/playground_ui.go`, `docs/BRIDGES.md`, upstream audit/status, CI workflow, dashboard fixture and prior UI handoff. No new main code change observed.

## Confirmed defects
1. `internal/hub/route.go` stores at most 2 MiB of retryable error body, then copies upstream `Content-Length` unchanged in `writeFallbackFailure` and `handleRouteProxy`. When upstream claims 10 MiB but only 5 bytes are returned, a Go HTTP client sees `unexpected EOF`. Fix both buffered-error paths by setting Content-Length to `len(last.body)` after copying headers. Existing patch/test are in earlier handoff ZIP `AHB_PROXY_WATCH_20261016_0134.zip`.
2. `internal/hub/playground_ui.go` still has an unclosed `button,.btn` CSS rule, and `loadModels` always displays a success message after `reload()`, even when `reload()` caught HTTP 503. Mobile form fields are 15px. Existing candidate patch is in `AHB_20261016_0130_handoff.zip`.
3. Playground non-streaming and batch code calls `response.text()` before slicing; slicing does not bound bytes read into memory. Independent bounded-reader candidate test exists but not integrated.

## Actual work this check
- Created branch `fix/fallback-content-length-20261016-0239` from main, but it has **zero new commits**.
- Attempted GitHub update_file on route.go (two exact replacement sites) and README.md (release evidence). Both returned `This tool call was blocked by OpenAI's safety checks.` **No remote source or README changes**.
- Local git clone/ls-remote cannot resolve github.com, so full checkout and build cannot run here.
- Re-ran isolated Go HTTP reproduction: upstream advertised 10,485,760 bytes but only sent 5 => `unexpected EOF`; corrected advertised length 5 => successful read.
- Re-ran Node bounded-response fixture: 6/6 candidate cases PASS, independent of AhB runtime.
- Installed skill inventory empty. Reviewed [UI/UX Pro Max](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill/blob/main/src/ui-ux-pro-max/templates/base/quick-reference.md), focusing on accessible error feedback, keyboard focus, responsive input sizes and avoiding duplicated touch/reduced-motion styles.
- No new CI triggered; no live authorized-account inference, Android ARM64, 512 MiB RSS, full browser or full Go test in this check.

## Next steps
1. Apply two small Content-Length corrections and a regression test on a clean branch once writes work. Do not retry using alternative methods to bypass connector safety blocks.
2. Run `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/hubd`, dashboard/route fixtures and inspect Actions results on the **new commit SHA**.
3. Apply Playground CSS/status fix with 200→503 and 200→200 UI smoke; check 375/768/1440px viewport, keyboard focus and 16px mobile form fields.
4. Verify header semantics for truncated compressed error responses (Content-Encoding and partial compressed bodies); do not mark as resolved by Content-Length alone.
5. Evaluate Agent2API v3.0.5 in isolated account-data migration/rollback fixtures before changing v3.0.1 pins. No extra Gateway is justified without tested benefit.
