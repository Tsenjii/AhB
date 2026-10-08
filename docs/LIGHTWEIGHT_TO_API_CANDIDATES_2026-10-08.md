# Lightweight To-API candidates for AhB — 2026-10-08

**Status: research only; none of these has been bundled, enabled, logged into or live-tested on Android by this review.** This document avoids repeating existing first-class sidecars (OpenCode, FreeBuff, Agent2API, DeepSeek, Grok, Kiro) and existing generic bridge presets (LMArena, Windsurf, Qwen, Kimi, Gemini, Claude).

Objective: expand *legitimately usable account-to-API sources* without making AhB a competing broad provider aggregator. Prefer compact, separately supervised Go/Rust applications, stable model discovery, streamed chat/tool support, upstream account/quota semantics and a localhost address.

## Shortlist and decision

| Candidate | Source | Why it adds value | Runtime and fit | Priority |
|---|---|---|---|---|
| **whtsky/copilot2api** | https://github.com/whtsky/copilot2api | Adds GitHub Copilot; OpenAI Chat/Responses, Anthropic, Gemini, usage endpoint, OAuth Device Flow; 5-min model cache | Very small Go module: `go 1.26.0`, only direct dependency `golang.org/x/sync`; source build should be trialed using AhB's existing Go 1.26 ARM64 job, not assumed to run already | **P1: best minimal Copilot sidecar candidate** |
| **StarryKira/copilot2api-go** | https://github.com/StarryKira/copilot2api-go | Adds Copilot account-pool WebUI, multiple accounts, OAuth Device Flow, load balancing | Go 1.25; uses GitHub Copilot Go SDK, Gin and many packages; determine SDK/subprocess runtime on Android before packaging. Prefer over whtsky only if multi-account UI justifies footprint | P2: account-management alternative |
| **tonghaoch/copilot-proxy-go** | https://github.com/tonghaoch/copilot-proxy-go | Adds Copilot and Chat/Responses/Messages/Embeddings with device login | Go 1.25, additional TUI/CLI deps; smaller account-management scope vs StarryKira | P2 alternative, do not package three Copilot proxies |
| **router-for-me/CLIProxyAPI** | https://github.com/router-for-me/CLIProxyAPI | Broad account-to-API for Gemini CLI, Codex, Claude, Kimi etc.; streaming, tools, multi-account | Go 1.26 but large module with many subdependencies and overlaps existing Account2API/bridge support; evaluate as **one optional external sidecar**, not as a replacement for AhB or as a ModelAtlas-type provider catalog | P2 if user wants these specific CLI sources |
| **10/chatgpt-codex-proxy** | https://github.com/10/chatgpt-codex-proxy | Account-pool + quota-aware Codex adapter, OpenAI/Anthropic, tools, SSE | Go 1.26.8; private/undocumented upstream; needs active authorized Codex accounts, independent quota checks and Android build validation | P3 |
| **thezillo/codex-proxy** | https://github.com/thezillo/codex-proxy | Rust single binary, self-reported approx 3 MiB RSS, Codex Chat/Responses, pool | This **3 MiB is upstream's Linux/container measurement, not verified Android**. Static musl Linux binary cannot be assumed compatible with Android/Termux. Cross-compilation to Android target requires evaluation | P3 research |

## Exclusions for this phase

- `ericc-ch/copilot-api` (https://github.com/ericc-ch/copilot-api): complete but Bun/Node-based and GitHub Copilot only; if a verified Go Copilot option passes it adds less incremental value. Reverse-engineered behavior can break and its README warns about excessive automated use.
- Cursor CLI proxies `code-yeongyu/cursor-proxy` (https://github.com/code-yeongyu/cursor-proxy) and `anyrobert/cursor-api-proxy` (https://github.com/anyrobert/cursor-api-proxy): require running Cursor agent CLI plus Bun/Node runtime; no confirmed compatible Android ARM64 Cursor CLI. Hold as desktop bridge-only.
- `cacaview/iflow2api` (https://github.com/cacaview/iflow2api): archived in March 2026; do not package an unmaintained service.
- New API / LiteLLM / generic multi-provider API-key aggregators: largely duplicate ModelAtlas and are **not** AhB account-backed sidecar goals.
- Browser interception, challenge-solving or access-control circumvention tools: not part of an account-backed, supported local integration.

## Implementation gate

1. **Do not code or enable these by default** during discovery. Select exactly one Copilot approach first; do not build three redundant Copilot sidecars.
2. Audit pinned upstream commit, license, terms, network bindings, account storage paths, credential/log handling, SDK subprocess dependencies and CPU/RSS with realistic sessions. Authenticate through supported user-initiated OAuth/device flows.
3. Prefer `kind: sidecar` only when the candidate compiles/runs on **Android ARM64**, supports localhost binding, offers model discovery and supplies a real health/account capability check. Otherwise attach a separately-running loopback bridge via `scripts/connect-bridge.sh`.
4. Keep original UI/account pool inside the upstream; AhB only exposes namespace routing and telemetry. Avoid duplicating Agent2API features. Enforce exact model compatibility and do not route exhausted accounts as healthy inference-ready.
5. Add pinned binary/UI package only after CI cross-build, Android launch, model listing, actual inference, SSE/tool-call tests, quota-exhaustion/error tests, graceful shutdown, and backup preservation pass.
6. Account access and model availability depend on legitimately accessible subscriptions, upstream quotas and provider terms; do not assume any service grants free inference.

The **last real-device finding** remains important: Agent2API reported HEALTHY for 2/2 credentials while actual Qwen inference returned HTTP 503 due to low balance. The next sidecar needs a stricter separation between process health and quota-available inference.

**Sources:** primary upstream README.md and go.mod inspected 2026-10-08. Toolchain observation: AhB Android workflow has a Go 1.26 setup step, while Hub CI uses Go 1.23 and initial Android compilation uses Go 1.24, so a new Go 1.26 component must use the correct build step instead of altering the Hub's base toolchain.
