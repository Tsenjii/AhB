# NEXT AI HANDOFF

Canonical repository: **Tsenjii/AhB**  
Visibility: **public**  
Default branch: **main**

This file is the project handoff. Read this first before changing anything.

## User goal

Build a lightweight Android/Termux AI provider hub that reuses mature upstream gateways rather than reimplementing provider protocols.

The user wants:
- one local OpenAI/Anthropic-compatible endpoint;
- provider-prefixed model IDs;
- tool-calling-capable providers;
- low RAM use;
- Android/Termux first;
- provider UIs reused instead of duplicated;
- optional deployment to small VPS later;
- no giant framework rewrite.

## Current architecture

```text
Client
  |
  v
AhB hubd 127.0.0.1:8317
  |
  +-- OpenCode / opencode2api
  |      API 127.0.0.1:8401
  |      UI  127.0.0.1:8404
  |
  +-- FreeBuff / Freebuff2API
  |      API 127.0.0.1:8402
  |      UI  127.0.0.1:8402/ui
  |
  +-- Agent2API (optional provider pack)
         API/UI 127.0.0.1:8403
```

Hub UI:
```text
http://127.0.0.1:8317/ui
```

Unified API:
```text
http://127.0.0.1:8317/v1
```

Model IDs:
```text
opencode/<model>
freebuff/<model>
agent2api/<model>
```

## Implemented in hubd

- provider sidecar supervision
- startup readiness polling
- periodic health checks
- bounded restart/backoff
- graceful stop before forced kill
- independent provider failure domains
- merged GET /v1/models
- POST /v1/chat/completions pass-through
- POST /v1/responses pass-through
- POST /v1/messages pass-through
- POST /v1/messages/count_tokens pass-through
- POST /v1/systemone pass-through
- streaming forwarding
- provider-prefixed routing
- stripping caller Authorization / X-Api-Key before forwarding
- provider-local auth headers from config
- loopback-only V1 listener
- /api/providers
- /api/runtime
- Hub RSS and sidecar RSS reporting from /proc
- embedded mobile-friendly Hub overview UI
- buttons into each provider's original upstream UI

## Provider choices

### OpenCode
Pinned V1 adapter:
- jasonxu114514/opencode2api
- v1.3.7

Enabled by default.

Hub prefix:
`opencode/`

The upstream WebUI is enabled.
AhB generates a random local server key and random WebUI password on first setup.

Generated files:
```text
data/hub-local-key.txt
data/opencode/webui-password.txt
```

These are ignored by Git.

OpenCode supports direct / HTTP / HTTPS / SOCKS5 / SOCKS5H egress configuration through its own config/UI.

### FreeBuff
Pinned V1 adapter:
- lza6/Freebuff-2API
- v0.10.3

Enabled by default.

Hub prefix:
`freebuff/`

Use FreeBuff's existing /ui for:
- account import
- account pool
- quota
- health
- proxy
- models
- Playground
- diagnostics

### Agent2API
Pinned optional provider pack:
- aimod-cc/agent2api
- v2.9.5

Disabled by default in config until installed/enabled.

Hub prefix:
`agent2api/`

It is used to avoid separately reimplementing adapters for providers such as:
- CodeArts
- Qoder
- Cline
- Trae
- Loomy
- AutoClaw
- Accio
- ZCode
- other providers currently supported upstream

Do not assume the provider list is permanent; inspect current upstream before making claims.

Termux install helper:
```sh
./scripts/install-agent2api-termux.sh
```

Enable helper for a prebuilt bundle:
```sh
./scripts/enable-agent2api.sh
```

## UI strategy

Do NOT rebuild provider management features in AhB.

AhB UI is only the overview/control foyer:
- provider state
- model count
- RSS
- restart count
- unified model list
- unified base URL
- direct link to original provider UI

OpenCode and FreeBuff already have mature UIs. Agent2API also ships its own UI.

## Public repository migration

The old private working repository was:
`Tsenjii/assistant-new`

The current canonical repo is:
`Tsenjii/AhB`

AhB is public and contains the entire current source tree plus newer Android build/install work.

Do not continue development in assistant-new unless explicitly asked. Treat AhB as source of truth.

## CI / build status as of 2026-10-08

Public AhB Actions can receive hosted runners.

Verified recent status:
- CI on main: SUCCESS
- Android ARM64 bundle workflow: SUCCESS
- public `prebuilt` branch exists

Important current commits at handoff:
- main: inspect current HEAD before editing; at time this handoff was created it was around `0167ebcc3c16fc6df52f4369cb8c6f20ef884970`
- prebuilt branch: `27f6b5f5f67aed187a153e8dbb1d101ac61a2f7c`

Always re-fetch branch heads before writing.

## Android prebuilt flow

Workflow:
```text
.github/workflows/build-android-arm64.yml
```

It builds:
- AhB hubd for Android ARM64
- opencode2api for Android ARM64
- Freebuff2API for Android ARM64
- Agent2API headless for Android ARM64

It assembles:
```text
AhB_android_arm64.tar.gz
AhB_android_arm64.tar.gz.sha256
AhB_android_arm64_binaries.sha256
```

and publishes a public `prebuilt` branch.

No-login Termux prebuilt installer:
```text
scripts/install-prebuilt-termux.sh
```

This exists specifically so the user's second phone does not need to log into GitHub.

## Source-build Termux flow

Default:
```sh
chmod +x scripts/*.sh
./scripts/bootstrap-termux.sh
./scripts/run-termux.sh
```

Optional all-provider source build:
```sh
AIHUB_WITH_AGENT2API=1 ./scripts/bootstrap-termux.sh
```

Diagnostics:
```sh
./scripts/doctor-termux.sh
./scripts/smoke.sh
```

Background:
```sh
./scripts/start-termux.sh
```

Stop:
```sh
./scripts/stop-termux.sh
```

## Network egress

By default, locally running providers share the phone's public egress IP.

OpenCode and FreeBuff support their own fixed proxy settings, so different providers can legitimately use different fixed egress routes.

Do not implement automatic proxy rotation or anti-abuse evasion.

Future optional architecture:
```text
phone hubd
├─ local provider -> phone egress
└─ remote provider -> VPS egress
```

Remote-provider mode is not yet a required V1 feature.

## 512 MB VPS

A VPS deployment path exists, but Android/Termux is the current priority.

Existing files:
```text
docs/DEPLOY_VPS_512MB.md
deploy/android-ai-hub.service
scripts/run-linux.sh
.github/workflows/build-linux-bundle.yml
```

hubd itself is tiny; provider sidecars dominate memory.

## Important code-review notes

The project previously had several generated-edit mistakes involving literal escaped newlines and malformed JSON. Those were fixed.

When editing through APIs:
- re-fetch the file after update;
- verify JSON is actual JSON;
- verify shell files do not contain accidental literal escape sequences;
- keep tests green.

The current test suite includes:
- model prefix routing
- JSON body rewrite
- large integer preservation
- credential stripping
- config validation
- Hub UI smoke checks
- child sidecar startup
- model aggregation
- request routing
- sidecar kill/restart recovery

## Security rules

- no real credentials in Git
- runtime secrets belong under ignored data/
- Hub remains loopback-only for V1
- do not expose Agent2API with AGENT2API_ALLOW_NO_KEY=1 beyond localhost
- do not expose provider management UIs publicly
- no bulk account farming
- no CAPTCHA/WAF bypass
- no proxy rotation intended to bypass anti-abuse limits
- legitimate personal account import/config is okay

## User preferences for this project

- Traditional Chinese
- concise but concrete
- do the work directly rather than repeatedly asking permission
- inspect current repo before changing code
- prefer mature upstream adapters
- avoid overengineering
- Termux first, native Android later
- if asked for homework: unrelated preference, not relevant here

## Immediate next steps

1. Inspect current AhB main and Actions before editing.
2. Verify the successful Android ARM64 prebuilt artifact / prebuilt branch contents are complete.
3. Make the public no-login install path as simple as possible for tomorrow's phone test.
4. Run/inspect all available CI.
5. Review Android ARM64 workflow assumptions, especially Rust Android builds for FreeBuff and Agent2API.
6. Tomorrow, once the user has the phone, do a real Termux E2E:
   - install prebuilt bundle;
   - start hubd;
   - confirm Hub UI;
   - confirm OpenCode UI;
   - confirm FreeBuff UI;
   - optionally Agent2API UI;
   - test /v1/models;
   - test chat;
   - test streaming;
   - test real tool calling;
   - kill a sidecar and verify restart;
   - record real RAM/RSS.
7. Fix anything found directly in AhB.

## First message for the next AI

The user can say:

> 繼續 AhB 專案。先讀 Tsenjii/AhB 的 docs/NEXT_AI_HANDOFF.md，然後直接檢查 main、Actions、prebuilt branch，照 handoff 繼續做，不要叫我重講背景。
