# AI HANDOFF — Android AI Hub (AhB)

> **NEXT AI — READ FIRST (verified 2026-10-08):** [Current feature/platform audit](docs/CURRENT_FEATURES_AND_PLATFORMS_2026-10-08.md) is the authoritative, newly checked inventory. The **public Android ARM64 bundle has succeeded and is published at source `91c8da481d455d3df6ef888b18a8e3001b2eb83c`**, with [build](https://github.com/Tsenjii/AhB/actions/runs/37792105281) and [CI](https://github.com/Tsenjii/AhB/actions/runs/37792105269) both green. The user **has not upgraded their phone yet**; no new account/source authenticated inference tests have happened on-device. Some historical chronology farther below says "build pending"; those statements were correct at the time and are **now superseded** by this notice. Study the source at HEAD and this inventory, not stale past statuses.


This file is the continuity anchor for future AI sessions. Read it before changing code.

## Source of truth

Repository: `Tsenjii/AhB`

Current role:
- **Public repo**
- Default branch: `main`
- Public prebuilt branch: `prebuilt`
- This repo supersedes the old `Tsenjii/assistant-new` project for Android AI Hub work.
- Do not continue implementation in `assistant-new` unless explicitly asked.

## Current verified state

Final stabilization baseline before the next real-phone test:

- Main implementation/bundle source: `ea52255a8a1c2093e045cb0835aa9f6f838891c5`
- CI run `37751642564`: **success**
- Android ARM64 bundle run `37751642532`: **success**
- Public `prebuilt/source-commit.txt`: `ea52255a8a1c2093e045cb0835aa9f6f838891c5`
- Public bundle SHA-256: `cdd633c4f704da57b4e74e6c03caa82ba5e2d23ca77a26ee265e1972117063e1`

The final bundle contains:
- `hubd`
- `opencode2api`
- `freebuff2api`
- `agent2api-server`
- `deepseek2api`
- `grok2api`
- `kiro-go`

Stabilization completed in this baseline:
- Agent2API pinned to v2.9.6
- DeepSeek2API optional sidecar integrated
- Grok2API optional sidecar integrated with original UI and one-time local Client Key bootstrap
- Kiro-Go optional sidecar integrated with original UI and account telemetry
- Grok Android build uses NDK + CGO and `GODEBUG=netdns=cgo` to avoid the loopback DNS failure previously reproduced on Android
- Hub UI refreshed to a restrained, phone-first dashboard; coarse-pointer devices force the mobile layout even when CSS viewport width is large
- optional same-model provider fallback/balancing remains off by default
- CI validates Go code, Termux shell syntax, and JSON configs
- Android prebuilt publishing is serialized with a workflow concurrency group to prevent concurrent force-push ref-lock races

Verification terminology:
- OpenCode and the previously tested Agent2API/Qoder path have real-device inference/tool-calling evidence from the Android phone.
- Grok2API and Kiro-Go are currently **CI/build/prebuilt VERIFIED**, not yet **real-device VERIFIED**.
- DeepSeek2API is also not to be called real-device VERIFIED until the phone test completes.

The public repo contains a `prebuilt` branch for no-login Android/Termux installation.

## Architecture

```text
Client
  |
  v
hubd 127.0.0.1:8317
  |
  +-- opencode2api 127.0.0.1:8401
  |      +-- original management UI 127.0.0.1:8404
  |
  +-- Freebuff2API 127.0.0.1:8402
  |      +-- original management UI /ui
  |
  +-- agent2api-server 127.0.0.1:8403 (optional)
         +-- original management UI /
```

The Hub UI is only a **central foyer/status page**. It must not duplicate the upstream provider UIs.

Hub UI:
```text
http://127.0.0.1:8317/ui
```

Unified API:
```text
http://127.0.0.1:8317/v1
```

## Current providers

### OpenCode Free
- Runtime: `opencode2api v1.3.7`
- Prefix: `opencode/`
- API: `127.0.0.1:8401`
- UI: `127.0.0.1:8404`
- Hub generates a local server key and WebUI password at install time.
- Do not commit real keys/passwords.

### FreeBuff
- Runtime: `Freebuff2API v0.10.3`
- Prefix: `freebuff/`
- API: `127.0.0.1:8402`
- UI: `127.0.0.1:8402/ui`
- Account pool/quota/proxy/account health stay inside upstream FreeBuff.

### Agent2API (optional provider pack)
- Runtime target: `agent2api-server v2.9.6`
- Prefix: `agent2api/`
- API/UI: `127.0.0.1:8403`
- Install script: `scripts/install-agent2api-termux.sh`
- Enable helper: `scripts/enable-agent2api.sh`
- Intended to reuse mature adapters for personal WorkBuddy domestic/international, KukuAI, CodeArts, Qoder, Cline, Trae, Loomy and other upstream-supported channels.
- Keep optional because it is much larger to build than the default V1.

## Hub functionality already implemented

- sidecar process supervision
- startup readiness checks
- periodic health checks
- bounded restart/backoff
- independent provider failure domains
- graceful stop before forced kill
- provider process RSS from `/proc`
- Hub RSS/runtime telemetry
- merged `GET /v1/models`
- provider-prefixed model IDs
- routing by `provider/model`
- streaming pass-through
- request size limit
- client Authorization/X-Api-Key stripping before provider-specific auth headers are applied
- loopback-only V1 binding
- dashboard provider cards
- model search
- links to original provider UIs
- unified Base URL display/copy
- Android/Termux diagnostics
- VPS deployment docs
- provider network egress/proxy docs

Pass-through endpoints currently include at least:
- `POST /v1/chat/completions`
- `POST /v1/responses`
- `POST /v1/messages`
- `POST /v1/messages/count_tokens`
- `POST /v1/systemone`
- other common OpenAI endpoints added in later AhB commits — inspect current `internal/hub/hub.go` before changing routes.

## UI strategy

Do not rewrite provider management features.

Hub UI should show:
- provider name
- HEALTHY / STARTING / DEGRADED / DEAD / DISABLED
- process RSS
- restart count
- model count
- unified model table
- management button that opens the provider's original WebUI

Current Hub UI was designed for phone screens and should remain dependency-free:
- no React
- no Node runtime
- no CDN requirement
- embedded HTML/CSS/JS in Go binary

## Secrets

Public repo must contain placeholders only.

Local runtime secrets belong under ignored `data/`.

Generated by:
```text
scripts/prepare-configs.sh
```

Important files:
```text
data/hub-local-key.txt
data/opencode/webui-password.txt
data/opencode/config.json
data/freebuff/config.json
config.json
```

Never commit live provider credentials.

## Android deployment

Public/no-login route should be preferred on the test phone.

Inspect current README and these files before giving install commands:
- `scripts/install-prebuilt-termux.sh`
- `.github/workflows/build-android-arm64.yml`
- `docs/ON_DEVICE_CHECKLIST.md`

The repository already has a public `prebuilt` branch so a phone should not need the private GitHub account just to install.

If prebuilt install fails, fallback is local Termux build:
```sh
./scripts/bootstrap-termux.sh
```

Optional full provider pack:
```sh
AIHUB_WITH_AGENT2API=1 ./scripts/bootstrap-termux.sh
```

## VPS

512 MB VPS support exists, but phone/Termux is currently the main test target.

VPS guidance:
- use prebuilt binaries
- do not compile FreeBuff on a 512 MB VPS
- keep loopback-only during V1
- use SSH tunnel until authenticated remote access exists

See:
`docs/DEPLOY_VPS_512MB.md`

## Network egress

Running everything directly on one phone means providers normally share the phone's public IP.

Current upstream support:
- OpenCode: direct / HTTP / HTTPS / SOCKS5 / SOCKS5H
- FreeBuff: fixed HTTP/SOCKS5 proxy


See:
`docs/NETWORK_EGRESS.md`

## Immediate next priorities

1. **Do not redesign the architecture.**
2. Inspect current `main` and public `prebuilt` branch before modifying anything.
3. Verify the no-login Termux prebuilt installer on a real Android phone.
4. Start the Hub and open `http://127.0.0.1:8317/ui`.
5. Verify OpenCode original UI and generated password.
6. Verify FreeBuff original UI and normal account import.
7. Test real inference through both providers.
8. Run a real tool-call loop:
   - read file
   - search/grep
   - edit file
   - run test
   - react to result
9. Kill a sidecar and verify automatic restart.
10. Measure real Android RSS.
11. Only after default V1 is stable, install/test Agent2API.
12. Add more providers by reusing mature upstream sidecars, not by copying protocol implementations into hubd.

## Important project preference

Keep implementation minimal and reliable.

Prefer:
```text
mature upstream adapter + thin hub integration
```

over:
```text
rewrite protocol + duplicate UI + duplicate account/quota logic
```

## Before any future AI says "done"

It must distinguish:
- code written
- CI green
- binary built
- real Android launch verified
- real upstream inference verified
- tool calling verified

Do not claim Android deployment success before the real phone test passes.



## Optional cross-provider routing

Cross-provider behavior is **off by default**. Normal requests keep strict `provider/model` routing.

Configuration:

```json
{
  "routing": {
    "route_aliases_enabled": false,
    "same_model_fallback": {
      "enabled": false,
      "mode": "balanced",
      "providers": ["opencode", "agent2api", "freebuff"]
    }
  }
}
```

### Same-model fallback

When enabled, a request such as `opencode/foo` may move only to the exact same upstream model id `foo` on another configured provider. AhB never substitutes a different model.

Modes:
- `sequential`: requested provider first, then configured providers in order after a retryable failure/unavailability.
- `balanced`: independent concurrent requests are distributed toward the provider with the lowest current in-flight count. A single request is sent to only one provider at a time; if that attempt fails with a retryable error, the same request falls back to the next provider.
- Legacy config value `parallel` is accepted as an alias for `balanced`. It does **not** race one request across multiple providers.

This avoids duplicate billing and duplicate tool side effects while still allowing aggregate concurrency across providers.

Retryable statuses remain conservative: 402, 404, 408, 425, 429, 502, 503, 504. Transport errors and provider unavailability also allow fallback.

Responses identify routing decisions with:
- `X-AhB-Provider`
- `X-AhB-Routing`
- `X-AhB-Attempts`
- `X-AhB-Fallback: same-model` when the final provider differs from the requested provider

### Route aliases

The older explicit `route/<id>` cross-model alias feature is retained only as an optional advanced feature and requires `routing.route_aliases_enabled=true`. It is hidden from `/v1/models` and unavailable for requests while disabled.

Provider-internal account routing/failover remains inside mature sidecars such as Agent2API and FreeBuff. AhB only coordinates provider-level behavior when explicitly enabled.


## DeepSeek2API optional provider

AhB integrates pinned upstream `zengtao227/Deepseek2API` at commit
`7a0925fa9bd83e36838b4d3762c81292feb344c2` as an optional sidecar.

AhB-facing identity:
- provider ID / model prefix: `deepseek/`
- API: `127.0.0.1:8405`
- original admin UI: `http://127.0.0.1:8405/admin`
- binary: `bin/deepseek2api`
- work dir: `data/deepseek2api`
- config: `data/deepseek2api/config.json`
- admin secret: `data/deepseek2api/admin-key.txt`
- disabled by default; enable with `scripts/enable-deepseek2api.sh`

The upstream source currently binds `0.0.0.0`. AhB's Android build applies a
single build-time patch to bind it to `127.0.0.1`, preserving AhB V1's
loopback-only security boundary. Do not remove this patch unless upstream gains
a supported bind-host setting.

The Android workflow builds the upstream React admin UI ahead of time and ships
the static output, so Node/npm are not required on the phone. It then builds
the Go backend for Android ARM64 with the Android NDK and libc DNS.

Hub account telemetry reads only the local DeepSeek config to count configured
accounts and credentials. It never emits account identifiers, passwords, or
tokens through `/api/providers`.



## External localhost provider kind

AhB supports `kind: "external"` for an OpenAI-compatible service that is
already running on loopback. External providers:
- must use a localhost / loopback `base_url`; LAN/public addresses are rejected
- are health-polled by hubd but are not spawned, killed, or restarted by hubd
- participate in model aggregation, normal `provider/model` routing, and the
  optional same-model fallback/balancing feature
- show Process=N/A in the Hub UI because AhB does not own their process

A disabled `lmarena` slot is included at `127.0.0.1:8406`, using
`/v1/models` for both health and model discovery. Enable it with
`scripts/enable-lmarena-external.sh` after starting a compatible local bridge.



## Agent2API v2.9.6 update — 2026-10-08

AhB now pins upstream `aimod-cc/agent2api v2.9.6`.

Verified:
- CI: success
- Android ARM64 bundle: success
- public prebuilt source commit: `bce3a6f22d6aa94593a768203ef1bc2b6567a927`

Notable upstream changes:
- new Check-in Center
- KukuAI provider support
- Loomy beginner-task workflow
- per-account automatic balance querying
- low-balance accounts can yield routing automatically
- account-add flow improvements

WorkBuddy is now split upstream into two provider identities:
- `workbuddy` — domestic
- `workbuddy-intl` — international

Treat them as separate provider identities with separate account/catalog state. Do not merge them in AhB.


## Reverse-proxy / To-API research queue — 2026-10-08

Project direction: broad provider coverage via mature sidecars, while keeping hubd thin.

### Priority A — strong first-class sidecar candidates

1. **Grok2API**
   - https://github.com/chenyme/grok2api
   - very active Go + React project
   - Grok Build / Web / Console account pools
   - multi-account, quota/model sync, retry/routing
   - OpenAI Chat Completions, Responses, Anthropic Messages
   - tools/reasoning/multi-turn plus media surfaces
   - linux arm64 images already exist
   - target prefix: `grok/`
   - next checks: listener binding, Android cross-build, static UI packaging, health/account endpoints

2. **Kiro-Go**
   - https://github.com/Quorinex/Kiro-Go
   - Go + Web admin
   - multi-account pool
   - AWS Builder ID / IAM Identity Center / Microsoft SSO / SSO token / Kiro API Key
   - OpenAI Chat Completions + Responses + Anthropic Messages
   - automatic token refresh and outbound proxy support
   - target prefix: `kiro/`
   - next checks: listener binding, storage paths, Android ARM64 build, UI packaging

3. **WindsurfAPI**
   - https://github.com/dwgx/WindsurfAPI
   - high-activity multi-account gateway with dashboard
   - OpenAI + Anthropic endpoints
   - tool-call support, dynamic model discovery, account pool/usage state
   - current canonical repo is `dwgx/WindsurfAPI`; do not use small forks as the source of truth
   - target prefix: `windsurf/`
   - next checks: runtime dependencies, Android feasibility, local UI assets, account-health endpoint

### Priority B — useful but packaging/runtime tradeoffs

4. **Qwen2API_Go**
   - https://github.com/XxxXTeam/Qwen2API_Go
   - Go + management UI/account pool
   - OpenAI + Anthropic compatibility
   - file/image/video surfaces
   - target prefix: `qwen/`
   - next checks: tool-call behavior, Android build, listener binding, account-health API

5. **Kimi2API**
   - https://github.com/chopper1026/kimi2api
   - Python/FastAPI + React admin
   - multi-account Kimi Web pool
   - `/v1/models`, Chat Completions, Responses
   - dynamic Kimi Web model discovery
   - better initial fit as `kind: external` because of Python runtime cost on Termux
   - target prefix: `kimi/`

### Priority C — external/research first

6. **Gemini2API**
   - https://github.com/xwteam/gemini2api
   - Python/FastAPI
   - multi-account management
   - OpenAI / Claude / Gemini protocol compatibility
   - current session behavior is comparatively fragile; keep external-first

7. **Claude2API**
   - https://github.com/yushangxiao/claude2api
   - Go Claude Web compatibility layer
   - historically popular but lower current code activity
   - re-evaluate before integration

8. **LMArena**
   - AhB already includes a disabled external slot: `lmarena/` on `127.0.0.1:8406`
   - keep provider-specific bridge logic outside hubd until a maintainable candidate is selected

### Lower-priority findings

- Mistral Le Chat reverse-proxy projects exist, but current candidates are small and tightly coupled to browser/network-edge behavior; do not prioritize over Grok/Kiro/Windsurf.
- Perplexity/Poe/Meta-AI bridge candidates found so far are much smaller and less mature.
- kRouter is capable but overlaps AhB/ModelAtlas-style aggregation too heavily; do not make it a core AhB dependency.
- KukuAI does not need a separate AhB sidecar while Agent2API v2.9.6 already supports it.

### Recommended implementation order

1. real-device verify DeepSeek2API
2. real-device verify Agent2API v2.9.6
3. integrate Grok2API
4. integrate Kiro-Go
5. integrate WindsurfAPI
6. evaluate Qwen2API_Go
7. keep Kimi/Gemini/LMArena external-first until runtime/session behavior is stable

For each provider, preserve:
- loopback-only listener
- upstream management UI
- local secrets under ignored `data/`
- process / provider-ready / account-usable health semantics
- no credentials in logs or `/api/providers`
- pinned upstream version/commit
- CI build before claiming support
- real Android launch + real inference before marking VERIFIED


## Grok2API first-class sidecar

Integration target:
- upstream: `chenyme/grok2api`
- pinned commit: `7c889a960e2638341b4dae9a5c81af0e0f38c87f`
- prefix: `grok/`
- API/UI: `127.0.0.1:8407`
- health/readiness: `/readyz`
- models: `/v1/models`
- disabled by default

Design:
- preserve the original Grok2API management UI
- build the Go backend for Android ARM64 with Android NDK + `CGO_ENABLED=1` and `GODEBUG=netdns=cgo` (the previous no-CGO Android DNS assumption was superseded)
- build the React frontend in CI; no Node runtime is needed on the phone
- local secrets live only under ignored `data/grok2api/`
- `scripts/enable-grok2api.sh` performs one-time localhost bootstrap and
  creates a dedicated downstream Client Key for AhB rather than sharing an
  administrator credential
- normal process lifecycle is hubd-supervised after bootstrap
- use `/readyz`, not liveness-only `/healthz`, so no-account/model state is
  visible as degraded rather than falsely routable
- Hub interprets Grok readiness components for account usability

Verification ladder:
1. AhB CI green
2. Android ARM64 Grok2API build green
3. public prebuilt contains the backend and frontend assets
4. real Android launch
5. real account model discovery and inference
6. full-path tool calling

Do not mark Grok real-device VERIFIED before steps 4-6 are completed.


## Kiro-Go first-class sidecar

Integration target:
- upstream: `Quorinex/Kiro-Go`
- pinned commit: `f8f6071c9298a4266ad3e0c7e483d4a2510cbcaf`
- prefix: `kiro/`
- API/UI: `127.0.0.1:8408` / `/admin`
- work dir: `data/kiro-go`
- health: `/health`
- account truth: authenticated `/v1/stats` fields `accounts` and `available`
- disabled by default

Android design:
- build with Android NDK + CGO and `GODEBUG=netdns=cgo`
- ship the upstream `web/` directory; no Node runtime is required
- generate a local admin password under ignored `data/kiro-go/`
- reuse AhB's local server key for the Kiro downstream API
- keep provider-internal multi-account rotation/token refresh/proxy logic in Kiro-Go

Verification ladder is the same as Grok: CI/build/prebuilt first, then real
Android model discovery, inference and tool-calling before real-device VERIFIED.

## Hub UI direction

The Hub foyer is intentionally a restrained local-service dashboard rather than
an AI-themed landing page:
- flat neutral background; no decorative glow/gradient
- compact status rows, thin separators and small radii
- clear process/readiness/account layers
- mobile layout also triggers on coarse-pointer devices, not only viewport width
- original provider UIs remain the place for provider-specific management


## Finalization follow-up — 2026-10-08

The user's latest direction is **stabilize and hand off**, not expand the provider catalog further before real Android verification.

- Canonical handoff is **this file**. `docs/NEXT_AI_HANDOFF.md` was consolidated into a pointer to avoid conflicting historical status claims.
- Added `scripts/upgrade-prebuilt-termux.sh` for an **existing** ARM64 prebuilt install. The original `install-prebuilt-termux.sh` remains for first installs only and refuses an existing target directory.
- Upgrader verifies the downloaded bundle checksum before touching an existing installation. It stages the new version on the same filesystem, copies the existing `config.json`, all `data/` state (account databases/secrets), and `logs/`, refreshes the four bundled upstream WebUI asset trees from the archive, prepares/migrates configs, stops the existing AhB processes, and swaps the directory. The entire old install remains as a timestamped sibling `AhB.backup-YYYYMMDD-HHMMSS`.
- Added a Linux CI fixture test (`scripts/test-upgrade-prebuilt.sh`) for state preservation, UI refresh, backup retention, and checksum failure causing **no mutation** to the installed tree. CI now runs it.
- Android ARM64 workflow was updated to include the upgrade helper in the distributed bundle. README now documents the upgrade path.
- **Verification gate:** Do not tell the user the latest upgraded Android bundle is validated until the Android build run succeeds and the public prebuilt source commit advances to the appropriate packaged source commit. CI can verify syntax, Go tests, and upgrade fixture behavior; it cannot prove actual Termux runtime, authentication, inference or tool calling.
- First Android test still must measure real account usability, inference, tool-calling (including an actual tool loop), streaming and sidecar restart on the device, especially for newly bundled DeepSeek2API, Grok2API and Kiro-Go. If a provider remains untested, report it as **build VERIFIED / phone UNVERIFIED**, not finished.

### Upgrade command for an existing Termux installation

```sh
curl -fsSL https://raw.githubusercontent.com/Tsenjii/AhB/main/scripts/upgrade-prebuilt-termux.sh | bash
```

After upgrading, run `cd ~/AhB && ./scripts/run-termux.sh`; from a second Termux session run `./scripts/doctor-termux.sh && ./scripts/smoke.sh`. Keep the backup until account login, model discovery, inference and logs are checked.

Avoid reintroducing speculative features, rewriting provider UIs, or quietly altering fallback defaults during this finalization phase.


## Stability hardening follow-up (2026-10-08, post-release-review)

The final source-level reliability pass after the earlier `ea52255` prebuilt baseline has been committed. Treat `main` and the Actions page as the authoritative HEAD; the prebuilt binary only reflects commits that actually completed the Android workflow.

Corrections covered by Go tests and CI:
- Android upgrader copies local account/SQLite databases **after stopping all AhB provider processes**, rather than while the database may be receiving writes. The upgrade fixture simulates a final shutdown-time write and proves it is copied, preserves backups and also checks checksum-failure behavior.
- Opt-in same-model fallback examines each **alternate's advertised model list** before routing, and never substitutes an unrelated model ID merely because the alternate is healthy. The requested primary provider is not subject to that extra discovery call.
- Account-backed Agent2API, FreeBuff, DeepSeek and Kiro services are not marked routable when independent account health is unknown, including a blank or malformed provider HTTP health body.
- HTTP forwarding strips both fixed hop-by-hop headers and headers nominated by the `Connection` field. Streaming stops reading upstream when the downstream client disconnects.
- Hub now reserves its listener **before** starting any sidecar, avoiding orphaned providers if the Hub port was already occupied. Shutdown cancels workers and waits for all supervised processes and health probes to exit. A new test covers worker teardown.
- CI now runs Go unit/integration tests, race detector, vet/build, shell/JSON validation and the prebuilt upgrade simulation. These are necessary but **not sufficient** for real-device verification.
- `docs/PROVIDERS.md` was updated to match Agent2API v2.9.6 and the included DeepSeek/Grok/Kiro/External provider slots.

**Final validation gate:** Confirm the latest relevant Android Actions bundle run succeeded, and check `prebuilt/source-commit.txt` is the intended build's source commit. Only then offer it for optional real-phone trials. Green CI, arm64 cross-build, and fixture simulations do not establish Android login, streaming inference, original WebUI usability, token refresh, or real tool calling. Those must be exercised with the user's accounts and device. Avoid marking untested providers VERIFIED.

No additional provider expansion or aggregator redesign is required at this stage.


## Universal External Bridges and UI Workbench — 2026-10-08

Latest scope requested: make LMArena and other mature To-API **connections** simple, keep Android light, and improve the phone UI. This addition is a generic connection layer; **do not confuse a connected, already-running upstream with a bundled or launched upstream implementation**.

- Added `scripts/connect-bridge.sh` with presets `lmarena`, `windsurf`, `qwen`, `kimi`, `gemini`, `claude`, plus arbitrary lowercase custom IDs. Any **existing localhost OpenAI-compatible HTTP service** with `GET /v1/models` (or a documented alternate models path) can be connected without new Hub provider code.
- The CLI validates loopback-only URL, optional private API key, and a real JSON `data` model list **before saving**. It refuses replacing managed sidecars; keeps backup of the previous config in ignored `data/`; preserves other providers and adds only a namespaced `provider/model` route. It does not spawn upstream software.
- Added `api_path_prefix` to provider config so e.g. Gemini2API's `/openai/v1/models` and `/openai/v1/chat/completions` work behind normal AhB `/v1` paths without rewriting Gemini2API.
- UI: embedded, dependency-free mobile dashboard now features polished typography, responsive bridge connection wizard (preset, local URL, custom ID), visible generated one-command Termux setup, navigation anchors and correct HEALTHY count (does not count DEGRADED as healthy). Browser does not ask for account tokens or API keys.
- Added JSON pass-through `POST /v1/images/generations` and `POST /v1/audio/speech` for sources whose upstream offers them; unsupported upstream operations remain unsupported.
- Tests: shell connector fixtures (idempotency, local-key retention, sidecar overwrite prevention, non-loopback denial, bad models/connection failure); Go bridge prefix integration test; browser-script syntax validation via `scripts/test-dashboard.sh`; all attached to the CI workflow.
- Android ARM64 bundling includes the new connector script; **it does not bundle Python/Node/browser third-party bridges**. Existing first-class sidecars remain as before.
- Docs: [BRIDGES.md](docs/BRIDGES.md) links upstream projects, explains individual dependencies and honest compatibility/verification limits. Example LMArena preset targets an existing local browser bridge on port 5102; this is not proof the browser bridge functions on a phone.

**Important final gate:** after all related source/build script commits, check the most recent green CI, successful Android ARM64 build and public `prebuilt/source-commit.txt` are aligned. Real LMArena browser bridge, third-party credentials, actual inference and tool-calling still require on-device or independently deployed upstream tests. Avoid promising that every known third-party reverse proxy works. Do not automate CAPTCHA, anti-bot bypass or account-farming.

UI-generated command will restart Hub after successful connection; this **interrupts active in-flight requests**. The standalone connector script only saves config and asks for an explicit restart.


## REAL Android device acceptance (user-tested, 2026-10-08)

**Read [docs/DEVICE_ACCEPTANCE_2026-10-08.md](docs/DEVICE_ACCEPTANCE_2026-10-08.md) before the next release or health-status UI change.** The user's device test reached all eight basic smoke checklist items with scope limits: Hub 0.2.0-dev responds 200 and returns `{"status":"ok"}`; OpenCode real `nemotron-3.5-lightning-free` chat succeeded (HTTP 200, AIHUB_OK); SSE output emitted 147 `data:` events and terminated with `[DONE]`; 15 advertised models (14 OpenCode + 1 Agent2API); prior settings/accounts/databases survived upgrade; the new LMArena/bridge connector wizard and dark Hub UI displayed; a model emitted a `get_time` tool call with `finish_reason=tool_calls`. Keep original backup `~/AhB.backup-20261008-194847` until other provider accounts are verified.

**Critical distinction:** Agent2API reported **HEALTHY** with 2/2 credentialed accounts, yet real `Qwen3.8-Flash` inference returned **HTTP 503** because both accounts were skipped for **below-threshold remaining balance**. The current Hub Agent2API probe counts enabled + credentialed + chat-capable accounts, **not quota-available/inference-usable accounts**. The UI must not present 2/2 as proof of usable quota. FreeBuff showed DEGRADED because the user had 0 accounts. These are account availability/usage findings rather than demonstrated Hub transport regressions.

**Unverified in that report:** actual LMArena/custom bridge connection and inference (UI/preset only); complete tool-call execution + second model request (initial function-call emission only); real inference on Agent2API and FreeBuff while they lack usable quota/accounts; DeepSeek, Grok and Kiro live inference. Do not mark all provider/end-to-end tests 100% complete based on this report.

Next product UX improvement: distinguish provider process health, credential readiness, and account/model quota sufficiency. Obtain genuine quota status from supported upstream APIs or clear 503 error state; never simulate balances or bypass usage limits.


## DeepSeek / Kimi and additional lightweight To-API research — 2026-10-08

**Read [docs/DEEPSEEK_KIMI_AND_LIGHTWEIGHT_TO_API_RESEARCH_2026-10-08.md](docs/DEEPSEEK_KIMI_AND_LIGHTWEIGHT_TO_API_RESEARCH_2026-10-08.md).** This is research, not a new integrated Android binary.

Correction to potential future misunderstanding: **DeepSeek Web is already a real bundled Go sidecar**, pinned from `zengtao227/Deepseek2API`, including static admin UI and `scripts/enable-deepseek2api.sh`, but disabled by default and **not yet verified by real-phone account inference**. Kimi Web, Kimi Code, LMArena, Qwen, Gemini and Windsurf remain *only preconfigured connector names* unless a separate local bridge actually runs. The phone build ships **no Kimi binary/Python service**.

Priority for further integration:
1. **P0**: validate built-in DeepSeek Web on Android with actual authorized account; do not duplicate it merely to add features already present.
2. **P1**: `chopper1026/kimi2api` has Kimi Web multi-account pool, refresh, admin panel, JSON/streaming API, but needs Python FastAPI/httpx; optional managed sidecar feasibility test first, preserve user state, don't falsely claim full native tool calling.
3. **P2**: `xaionaro-go/kimi-oauth-proxy` and `PixelMelt/kimi-proxy` offer **Kimi Code** compatibility, separate from Kimi Web. Need entitled user-initiated OAuth/CLI login and real device tests; Go version is tiny/low-history, Bun version has a runtime cost.
4. **P2**: `XxxXTeam/glm2api` (Python GLM Web), selective MiMo adapter, and the previously shortlisted single Copilot Go sidecar. Reuse existing Qwen2API_Go connector until installed/runnable.
5. **LOW/HOLD**: Node DeepSeek alternate `Sakura520222/deepseek2api` is pure Node with no third-party npm runtime deps, useful fallback if existing Go backend fails. `PruhaNLP/web-to-api` requires Chrome/Playwright and does not belong in lightweight phone base. `eequaled/GLM_proxy` requires AutoClaw on Windows/macOS, not a self-contained Termux GLM source.
6. **REJECT**: `aptdnfapt/qwen-code-oai-proxy` self-identifies as deprecated/dead due to auth errors; do not put in main usable shortlist.

All these are candidate repo claims only; no new runtime has passed device smoke/Chat/SSE/tools. Preserve user's original Android backup, privacy of auth tokens and current build stability. Focus on account availability and quota truth, highlighted by the user's real Agent2API HTTP 503 on 2/2 apparently credentialed accounts.


## Optional lightweight source expansion and registry — 2026-10-08

**Canonical shortlist and future intake:** [docs/TO_API_INSTALL_AND_OPTIMIZATION_PLAN.md](docs/TO_API_INSTALL_AND_OPTIMIZATION_PLAN.md) and [docs/TO_API_REGISTRY.json](docs/TO_API_REGISTRY.json). Registry has **34 unique candidate IDs** with statuses, source links, priorities, reasons and explicit Android live-test evidence. Add all future promising or discarded candidates to this registry instead of losing history. The CI `scripts/test-provider-registry.sh` verifies schema, unique IDs, the configured providers and packaging hooks.

**New native optional Copilot sidecar** from https://github.com/whtsky/copilot2api, pinned source `a4aac95d4a8f430f02121f79ea36aeaaa06daea1`, MIT:
- New `copilot/` disabled provider in `config.example.json`, port 8410, own state dir `data/copilot2api`. Android ARM64 workflow now compiles the Go 1.26 component and includes its LICENSE. No auto-start until explicitly enabled.
- `scripts/login-copilot2api.sh` performs interactive, voluntary user GitHub Device Flow in a temporary foreground server on port 8411, stores state in `data/copilot2api/credentials.json` with restrictive permissions. User presses Ctrl+C after successful auth.
- `scripts/enable-copilot2api.sh` requires the actual local credentials file, makes a config backup and sets enabled. Copilot upstream has no API-key enforcement of its own, so loopback-only is mandatory. No claim of multi-account rotation or remaining Copilot quota.
- Added binary to explicit `stop-termux.sh` owner list, diagnostic checks and UI install copy flow.

**New optional managed Kimi Web** from https://github.com/chopper1026/kimi2api, pinned source `7f046d8627f275432f82788a6547bc905038738c`, MIT:
- New `kimiweb/` disabled provider on port 8412 and native `/admin` UI; separate from legacy external `kimi/` connector. There is **no Python runtime or Kimi implementation in the base archive**. Only optional helpers ship.
- `scripts/install-kimiweb-termux.sh` opt-in downloads pinned source to ignored `data/kimiweb/source`, creates isolated venv, installs FastAPI deps, builds its React admin UI with Termux Node, then removes npm build deps; produces private admin/API keys under `data/kimiweb`, with state underneath. No account token automatically imported.
- `scripts/enable-kimiweb-termux.sh` checks completed installation, sets private AhB-to-Kimi bearer token and marks sidecar enabled. Hub supervises the dedicated venv Python process; exact-source process cleanup added to stop helper.
- **Android pip/package/frontend success, real account login, quota, SSE and full tool calling are NOT YET DEVICE VERIFIED**. Expect possible Python native wheel build failures; do not promise flawless one-click install.

**UI change:** added optional Copilot and Kimi Web command-copy panels. Adjusted Account layer label to **Credentials**, advertised models label to `listed`, and healthy count label to `endpoint healthy`. These are *not proof of inference-available quota*, following the real Agent2API 503 finding.

**Important release gate:** before asking the user to run the upgrader, verify **successful Android ARM64 bundle workflow** for the source change and the published `prebuilt/source-commit.txt` pointer. A green Go/shell/JS CI is not an Android build, and cross-build success is still not authenticated phone inference. Do not imply these new optional providers are already on the user's existing device or live-tested. Earlier user device backup `~/AhB.backup-20261008-194847` should remain intact.

**Follow-up optimization priorities:** quota-availability state (process vs credentials vs actionable inference), safe request logging, one-shot model tests, watchdog/resource limits, capability-aware routes, full tool-call continuation, ARM64 error-path acceptance, and low-memory source selection rather than launching every adapter.


## GPT Web vs Codex OAuth vs MCP; native tool-calling release gate — 2026-10-08

**Read [docs/GPT_WEB_CODEX_MCP_STABILITY_2026-10-08.md](docs/GPT_WEB_CODEX_MCP_STABILITY_2026-10-08.md) before touching GPT adapters.** The living registry [docs/TO_API_REGISTRY.json](docs/TO_API_REGISTRY.json) now tracks **51 candidates**, including lean Codex OAuth Go, multi-account Codex Go, ChatGPT Web MCP/browser implementations, and previously evaluated Kimi/Copilot/GLM/MiMo. Do not conflate browser Web, Codex model API and MCP-to-model one-shot tools.

**New active source code changes in this phase:**
- `scripts/connect-bridge.sh` recognizes **`codex`** (example `127.0.0.1:9879`, upstream reference https://github.com/dvcrn/codex-oauth-proxy) and UI exposes it. This is **EXTERNAL CONNECTOR ONLY**. No Codex/GPT bridge executable, OAuth login, or verified inference has been bundled. Requires a separately started local service with authorized Codex login; still enforces loopback model-list check and private client API key.
- `scripts/test-tool-roundtrip.sh` is a new optional real-inference acceptance script. It sends Chat Completions with an OpenAI function schema and `tool_choice:auto`, rejects a missing/malformed structured `tool_calls` array, submits deterministic `role:tool` output to the second request and requires a final assistant response. It never executes model-proposed commands. `scripts/test-tool-roundtrip-fixture.sh` supplies a local mock for both passing full continuation and rejecting text-only responses. Added to CI and Android bundle.
- The new tool script tests **OpenAI structured protocol compatibility**, not proof of the upstream model's intrinsic native tool encoding or reliability under long load. A Web adapter could emulate tool calls; don't claim true native upstream Tool Calling without additional provenance and repeated device tests. Separate Responses/Anthropic and SSE acceptance remain needed.
- CI initially failed due fixture using `grep -q` in a pipe with `set -o pipefail`, producing an expected `jq broken pipe`; corrected by consuming all output. Check current final CI before claiming green. The source changes remain **not validated with authenticated GPT or Kimi accounts on the actual phone**.
- **Kimi Web** remains **EXPERIMENTAL / DEFAULT OFF / P3** after user reports prior Kimi Web bridge instability and upstream itself says it is not optimized for programmatic tool calling. The optional installer is a research option, not a reliable fallback for OpenCode. Do not mark usable merely from WebUI or model list.
- Codex candidate ranking: (a) https://github.com/10/chatgpt-codex-proxy for multi-account/quota-aware function calls; Go 1.26.8, private upstream API fragility, Android device login pending; (b) https://github.com/dvcrn/codex-oauth-proxy for lightweight Go OpenAI API **and separate one-shot MCP**; requires Codex CLI login; (c) https://github.com/MortalImmortality/CodexProxy as lean alternative. No extra Codex binary bundled; first audit rights/permission, pin source and test genuine Android ARM64 auth/chat/SSE/two-turn tools/usage before packaging.

**Preserve existing stable OpenCode:** real-phone test previously established chat, streaming and initial tool-call **emission**; it did NOT establish second-turn tool continuation. Do not replace a working OpenCode route with a browser-based proxy until exhaustive equivalence testing. HTTP 503 from Agent2API exhausted credentials is account usability, not core transport failure. Prefer quota-truth fields and fail-closed rather than imaginary healthy quota.


## Grok2API independent community audit and acceptance improvements — 2026-10-08

Canonical findings: **[docs/GROK2API_COMMUNITY_RELIABILITY_2026-10-08.md](docs/GROK2API_COMMUNITY_RELIABILITY_2026-10-08.md)**. Findings were checked against Linux.do July–August firsthand accounts, GitHub Issues #793/#893/#975, upstream release v3.1.6 and NodeLoc community. **Important**: Chenyme publicly paused development in June but resumed in July; calling current repo "dead" based only on the June post is wrong. The source pinned by AhB is currently **upstream v3.1.6** at SHA `7c889a960e2638341b4dae9a5c81af0e0f38c87f`, dated 2026-09-30. Positive evidence for Build/Codex/Claude Code API agent loops; negative evidence for Web/Console sync, 429 cooldowns, legacy 403 and long-loop empty responses. No independent Android Grok account inference has been observed. Do not conflate Grok Web chat entitlement with Grok Build/Console quota or tools.

New code:
- `scripts/test-grok2api-termux.sh`: private and local-only `/healthz`, `/readyz`, provider component state, authenticated models inventory (requires user-local `data/grok2api/client-key.txt`). An explicit `AIHUB_TEST_MODEL='grok/actual-model'` additionally triggers real Chat, SSE + `[DONE]`, and two-turn structured tool continuation. Without that opt-in it avoids spending user quota. Never paste auth material into bug reports.
- `scripts/test-grok2api-fixture.sh` mocks success and negative paths (empty-model and no-native-tool), enforced in GitHub CI; this is NOT genuine Grok network authentication.
- `scripts/enable-grok2api.sh` now preserves an existing client credential on transient HTTP errors like 429/503/connection failure instead of repeatedly creating a new client key. 200 reuses, explicit 401/403 can recreate with valid local admin login. Config is backed up to `data/grok2api/config-before-enable.json` before changing with mode 0600. `scripts/test-enable-grok2api-fixture.sh` checks 503 preservation, 200 reuse and 401 recovery, enforced by CI.
- Android bundle manifest now includes `scripts/test-grok2api-termux.sh`; **before upgrading the user device, verify CI and Android build publication, the `prebuilt/source-commit.txt` pointer and the new script's presence**. Old real-device OpenCode chat remains validated; Grok is only P0 candidate until authenticated chat/SSE/tools plus longevity checks pass.
- The source registry now contains **52 candidates**, adding Chromium `lumingya/universal-web-api` found in NodeLoc to HOLD_DESKTOP (generic ChatGPT/Grok/DeepSeek web Chrome automation is not a stable, lightweight Termux substitute for native Grok2API).
- Grok source configuration `health_path=/readyz` already reports startup readiness rather than mere liveness. **Still do not use `HEALTHY` as proof of account quota**; the upstream readiness contract returns process/component state, not guaranteed model-specific spendability.

Phone test, once included in published prebuilt:
```sh
cd ~/AhB
./scripts/enable-grok2api.sh
./scripts/stop-termux.sh && ./scripts/start-termux.sh
./scripts/test-grok2api-termux.sh
AIHUB_TEST_MODEL='grok/<actually-listed-model>' ./scripts/test-grok2api-termux.sh
```

Current device credentials were not accessed, changed or imported by GitHub code edits. No CAPTCHA or upstream-access bypass is part of the integration.


## Release readiness and Termux doctor repair — 2026-10-08 22:00 Asia/Taipei

- The user **has not yet upgraded their phone** from their previously working OpenCode installation. Do not assume new optional adapters have been installed on device. Their last verified OpenCode endpoint provided successful HTTP 200 chat, SSE and initial tool-call emission; no full tool continuation or new provider account validation has occurred on the phone.
- The last confirmed *published prebuilt* at the start of this assessment was source SHA **a19721316ace788f0414e97b381fe175426b91b2**, behind recent Grok safety and diagnostic changes. A build must reach **the latest runtime/scripts source at least 7c30b21c7cbe223d6d5403215e1c8c0813f448dc**, and CI must be green before recommending the next phone upgrade.
- **Critical fix:** \`scripts/doctor-termux.sh\` had a malformed external-bridge \`while IFS=\` block corrupted by optional-provider insertions: duplicated provider reports and a syntactically legal but semantically wrong multiline fragment. Commit **bd813d300269c5be0eded41b2c20ed685da5c2cd** replaced the tail with one correct tab-separated \`while read\` loop, checking authenticated bridges through Hub and optionally probing Copilot/Kimi/Grok/Kiro only when enabled.
- Added \`scripts/test-doctor-termux.sh\` mock runtime fixture, plus CI execution, verifying a public local bridge, a privately authenticated bridge (not unauthenticated direct probe), optional enabled sources, disabled Kimi, and precisely one each of runtime/provider reports. It catches failures missed by \`bash -n\`. Runtime doctor repair is in the Android workflow source.
- **Upgrade behavior inspected:** \`scripts/upgrade-prebuilt-termux.sh\` verifies SHA256, stops owned sidecars, copies user \`data/\`, \`config.json\` and logs, replaces newly packaged UI assets rather than leaving stale copies, moves original directory to \`~/AhB.backup-YYYYMMDD-HHMMSS\` and activates new package. \`scripts/prepare-configs.sh\` appends missing provider configs and routing IDs *without* replacing existing account/provider settings. Caveats: this is a non-destructive copy-and-switch strategy, not a substitute for preserving older phone backup; restart, user consent and legitimate login required for new sidecars.
- **After confirmed latest prebuilt published**: have user use their existing approved upgrade process, then run \`./scripts/start-termux.sh\`, \`./scripts/doctor-termux.sh\`, \`./scripts/smoke.sh\` and OpenCode live test *first*. Only then opt into existing Grok (primary second source) with \`./scripts/enable-grok2api.sh\` and add user-authorized account, followed by \`./scripts/test-grok2api-termux.sh\` metadata and a chosen live model. Do **not** auto-enable Kimi Web or redundant CodeArts2API.
- Desired next product phase: real device account capacity and explicit 429/503 quota handling, source-by-source SSE and full tool-call two-turn tests, opt-in CLIProxyAPI attached as external multi-account CLI bridge rather than building another provider aggregator.


## 2026-10-08 final pre-phone hardening — actual repo changes, release gate

Current completion notes for user who explicitly **has not upgraded Android yet**; **do not overpromise live external provider availability**.

**Source changes tested by green CI** (at code revision `91c8da481d455d3df6ef888b18a8e3001b2eb83c`, CI https://github.com/Tsenjii/AhB/actions/runs/37792105269):
1. `scripts/prepare-configs.sh` now migrates provider definitions atomically in the AhB root with `chmod 600`; a failed `jq` validation fails closed without writing partial config or losing user's custom provider keys/data. `scripts/test-prepare-configs.sh` checks new entries added disabled, same old account state, preserved custom bridge tokens, repeat idempotence, malformed config protection.
2. `scripts/test-chat.sh` now uses `jq` to escape all prompt/model JSON, enforces a real nonempty assistant answer (not just HTTP 200), and preserves legitimate quota 429/503 errors. `scripts/test-chat-fixture.sh` proves real text works and empty/tool-only/503 cannot fake green.
3. In `internal/hub/request_status.go`, `internal/hub/route.go`, `internal/hub/hub.go` and `ui.go`: remember only the last HTTP status/time or a transport failure per provider on **real /v1 forwarded requests**, not account readiness probes. Show it in the phone UI as `Last API` separate from Process, Ready, Credentials. Keep inference protocol metadata only in memory—no chat text, model names, authorization headers, account credentials or full upstream error bodies. A 200 is explicitly **not** proof the stream completed or future quotas. `internal/hub/request_status_test.go` includes 503->200 update, transport failure, absence of leaks and quota-unknown semantics.
4. `scripts/test-stream.sh` validates Chat Completions SSE has structured assistant text and a final `[DONE]`. `scripts/test-stream-fixture.sh` rejects HTTP 503, missing data and missing DONE.
5. `scripts/test-provider-acceptance.sh` lets the user manually set `AIHUB_TEST_MODEL='provider/model'` and runs `test-chat.sh`, `test-stream.sh`, `test-tool-roundtrip.sh` (full tool result continuation). This is explicitly **not part of smoke.sh**, because it consumes model quota. `.github/workflows/build-android-arm64.yml` packages the new scripts for ARM64.
6. Earlier Grok v3.1.6 research/safety, `doctor-termux.sh` repair and tests remain intact. Do not add duplicate CodeArts2API: Agent2API already includes CodeArts. Keep Kimi Web off/experimental; new GPT Codex bridge is connector-only.

**Latest required acceptance instructions**: [docs/NEXT_PHONE_ACCEPTANCE.md](docs/NEXT_PHONE_ACCEPTANCE.md). It explains the precise published ARM64 build vs GitHub HEAD gate, non-destructive `upgrade-prebuilt-termux.sh` procedure, known-good OpenCode first, optional Grok login and SSE/tool tests, and preserving original backup. User's old backup `~/AhB.backup-20261008-194847` must not be deleted.

**CI vs full ARM64 build distinction**: CI for revision 91c8da4 succeeded (https://github.com/Tsenjii/AhB/actions/runs/37792105269); Android bundle build for that revision https://github.com/Tsenjii/AhB/actions/runs/37792105281 was still running at last verification. Last independently verified published source at that instant was `7c30b21c7cbe223d6d5403215e1c8c0813f448dc`, which does **not** contain these latest scripts and API UI. Verify `prebuilt/source-commit.txt` and bundle job conclusion again before recommending a phone upgrade. Subsequent docs-only commits do not change the Android bundle build source.

**Known unverified on-phone**: real Grok Build/Web/Console chat/tools, DeepSeek Web accounts, Copilot eligible login, Kimi Web optional Python install, full OpenCode tool-result continuation, CLIProxyAPI localhost bridge, GPT Codex external bridge, independent per-model quota capacity. Even a green mock fixture does not prove any of these.
