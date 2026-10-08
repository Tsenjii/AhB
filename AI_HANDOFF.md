# AI HANDOFF — Android AI Hub (AhB)

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
- build the Go backend for Android ARM64 with `CGO_ENABLED=0`
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
