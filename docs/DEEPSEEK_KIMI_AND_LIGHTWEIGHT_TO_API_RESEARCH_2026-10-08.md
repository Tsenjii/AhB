# AhB deeper To-API discovery — DeepSeek, Kimi and lightweight account gateways
**Research date: 2026-10-08. Status: research, not a bundled release.**

## 1. Correct existing integration status FIRST

**Do not present all visible UI presets as bundled services.**

| Source | Actually shipped in Android ARM64 archive? | Runs under hubd? | Why it matters |
| --- | --- | --- | --- |
| OpenCode (opencode2api) | Yes | enabled | Real-phone OpenCode inference and SSE validated |
| FreeBuff | Yes | enabled | User reports 0 accounts, currently DEGRADED |
| Agent2API | Yes | typically enabled by installer | User reports 2/2 credentialed but HTTP 503 on actual Qwen inference because balance is under threshold |
| **DeepSeek Web** | **Yes**: `deepseek2api` Go binary + static WebUI from `zengtao227/Deepseek2API`, pinned commit `7a0925fa9bd83e36838b4d3762c81292feb344c2` | Disabled until `enable-deepseek2api.sh` | **Not a connector-only stub!** It has its own account configuration, API paths and admin UI. Needs real-phone account/inference tests; don't add a second duplicate by default. |
| Grok2API | Yes, binary + UI | Disabled until enabled | Needs real-phone inference |
| Kiro-Go | Yes, binary + UI | Disabled until enabled | Needs real-phone inference |
| **Kimi Web**, LMArena, Windsurf, Qwen, Gemini Web, Claude Web | **No**: only `connect-bridge.sh` presets and generic external routing | Not launched or managed by AhB | Connection requires independently running local OpenAI-compatible upstream |

Sources of integration truth: [`.github/workflows/build-android-arm64.yml`](../.github/workflows/build-android-arm64.yml), [`config.example.json`](../config.example.json), [`docs/PROVIDERS.md`](PROVIDERS.md), plus firsthand device [acceptance report](DEVICE_ACCEPTANCE_2026-10-08.md).

## 2. DeepSeek: finish existing backend before replacing it

**Current main implementation**: https://github.com/zengtao227/Deepseek2API — Go backend and management UI already pinned and built into AhB. Its pinned source was still the upstream HEAD as of this research. Current installation is DISABLED by default and untested with real accounts; DO NOT claim it is a working DeepSeek Web inference source yet.

**Candidate alternate**: https://github.com/Sakura520222/deepseek2api — Node >=18, *no third-party npm runtime dependencies* in `package.json`; includes management console, multiple DeepSeek accounts, OpenAI/Anthropic/Responses, and heuristic tool-call parsing. Last observed commit 2026-04-27. Adds little new value vs packaged Go backend, but is a useful regression/fallback candidate if the Go backend fails live. Heuristic tool calls should not be assumed fully native or correct.

**Other redundant alternative**: https://github.com/iidamie/deepseek2api — multi-account DeepSeek bridge, Docker-oriented and not a clear Android advantage. Avoid redundant packaging.

**DeepSeek test gate**:
1. Without deleting user data, enable the *already bundled* Go service through its existing script; verify localhost binding, admin console, and no exposed secrets.
2. Using properly authorized accounts, verify real `/v1/models`, nonstreamed chat, SSE completion, error propagation when expired/exhausted, true tool-call support only if claimed.
3. If test fails, capture sanitized diagnostics and compare against Node alternate before replacing the pinned Go build.

## 3. Kimi Web: highest missing first-class component

| Upstream | Evidence & features | Mobile implications | Decision |
| --- | --- | --- | --- |
| **https://github.com/chopper1026/kimi2api** | Python >=3.8 FastAPI/httpx/uvicorn + itsdangerous/python-multipart; React admin panel; OpenAI Models/Chat/Completions/Responses, streaming, multi-account health & concurrency scheduling, refresh-token handling, API keys, logs; last observed commit 2026-05-14 | Can potentially run under Termux Python but creates package/runtime overhead. Kimi Web account access and endpoint changes not tested. README explicitly states **not optimized for coding Tool Calling** | **P1 for Kimi Web; optional install/sidecar, not a forced base-pack addition** |
| https://github.com/XxxXTeam/kimi2api | Smaller Python FastAPI/httpx gateway, Chat + Responses, status/usage methods; last observed commit 2026-05-04; simpler package dependencies and not a documented full account-pool WebUI | Better size but account management and control UX weaker. README warns usage accuracy is limited | P2 lightweight fallback |
| https://github.com/lorsque-sir/kimi2api | Kimi Web multiple token round-robin and OpenAI basic endpoints | Less complete operational controls, no verified Android build | P3 |
| https://github.com/PruhaNLP/web-to-api | Node >=20 plus Playwright Core **and Chrome/Chromium** for DeepSeek+Kimi+Qwen Web. Provides dashboard, basic chat SSE and pseudo tool calls | Browser requirement makes it much heavier on Termux. Pseudo tool calls aren't native and source includes external session-state dependency | **Research only for desktop/VPS, not a lightweight Android base** |
| https://github.com/delmacy/webtoapi-gateway | Bun + Playwright/CDP with broad provider catalogue | Browser runtime, duplicates provider aggregation & fragile website sessions | Not a phone core sidecar |

**Recommendation:** Kimi Web should be built as an *optional* `kimiweb/`-namespaced managed adapter using an upstream admin UI, or a dependency-managed Termux Python helper, **only after** a successful on-device Python startup, model list, real inference, concurrent account selection and token lifetime test. Do not conflate Kimi Web free chat access with separately licensed Kimi Code/API entitlements.

## 4. Kimi Code: a different (coding-oriented) account source

| Upstream | Evidence | Caveat | Priority |
| --- | --- | --- | --- |
| **https://github.com/xaionaro-go/kimi-oauth-proxy** | Small Go localhost OAuth forwarding bridge using existing Kimi Code CLI login, Chat Completions/Responses/Anthropic, advertises tool calls; 2 commits observed, last 2026-07-18 | Requires user-initiated supported OAuth login/CLI credential state. Little project history, correctness and token confidentiality audits needed. Android Kimi CLI credential lifecycle not tested. | **P2 Go candidate after Kimi Web** |
| https://github.com/PixelMelt/kimi-proxy | Bun-based OpenAI and Anthropic proxy for Kimi coding subscription; API key or OAuth, models and health, last commit 2026-07-24 | Bun runtime + one-account focus. Coding subscription **not the free Kimi Web tier** | P2 backup |
| https://github.com/MoonshotAI/kimi-code | Official Kimi Code CLI; has managed user-initiated OAuth login and quota/usage endpoints documented in its server API, actively developed | The official CLI server API is **not automatically** a generic OpenAI-compatible proxy. Need separate compatibility layer; Android packaging is separate effort | Strong reference for correct auth/quota interpretation, not yet an AhB sidecar |

Preferred architecture for coding: choose **one** legitimate Kimi Code protocol adapter with explicit account/quota observations rather than running Kimi Web and Kimi Code under one ambiguous `kimi/` identifier.

## 5. Other new account sources worth watching

| Candidate | URL and features | Decision |
| --- | --- | --- |
| GLM Web | https://github.com/XxxXTeam/glm2api — Python, Chat/Responses/Images, multiple token entries, optional valid account access; last observed commit 2026-08-23 | **P2** optional, potentially high incremental value, but no native binary. Security: debug modes can log requests/response bodies, disable by default |
| Xiaomi MiMo desktop/OAuth | https://github.com/IMROVOID/MiMo2API — Node >=22, OpenAI chat/models/SSE, account routing documented, last observed commit 2026-09-20 | **P2 desktop/local bridge**, not phone-first: requires preexisting MiMo Desktop login/auth file, and platform access not tested |
| Xiaomi MiMo Web | https://github.com/Fly143/MiMo2API — Python FastAPI + WebUI, account management & encrypted persistent credentials | **P2** optional bridge, verify auth support and Android Python cryptography compatibility |
| Qwen | https://github.com/XxxXTeam/Qwen2API_Go — **already present as external connector**, potential self-hosted Go sidecar if legitimate account/protocol works | P2: verify before duplicating |
| Qwen CLI alternative | https://github.com/aptdnfapt/qwen-code-oai-proxy — upstream README self-identifies as **DEPRECATED/DEAD** due to Qwen account access errors, and last commit `deprecated dead proxy` on 2026-04-16 | **REJECT**; do not advertise as new usable provider |
| GitHub Copilot | https://github.com/whtsky/copilot2api (Go 1.26, small) / https://github.com/StarryKira/copilot2api-go (multi-account WebUI, bigger SDK dependencies) | Previous shortlist remains promising; select exactly one after footprint and Android support audit. Copilot entitlement required |
| AutoClaw GLM | https://github.com/eequaled/GLM_proxy — Node >=18, no npm deps, OpenAI/Anthropic, last observed commit 2026-09-22 | **Desktop-only external** in practice: upstream README explicitly requires AutoClaw running and logged in on Windows/macOS; doesn't produce free independent GLM accounts on Termux |
| Broad account pool | https://github.com/router-for-me/CLIProxyAPI, https://github.com/Ken-Chy129/llm-proxy | Powerful but *overlaps AhB's control plane*. Do not bundle wholesale as another ModelAtlas-style aggregator |
| MiMo Code | https://github.com/Water008/MiMo2API — Python implementation, multi-account UI; basis for further comparison | HOLD: select one MiMo implementation instead of multiple overlapping versions |

## 6. Practical prioritized next sprint (without installing blind)

1. **P0**: real-phone inference on existing bundled `deepseek/`; fix actual failing account/middleware states, do not build a duplicate before evidence.
2. **P1**: make `chopper1026/kimi2api` first independently work on Termux with its native admin, then integrate an optional launcher/backup into AhB. Check Python footprint, session lifecycle, account availability, SSE and Tool Calling limitations.
3. **P1 alternative**: if Android can't run full Python WebUI reliably, trial the smaller XxxXTeam Python Kimi adapter or a user-run localhost Kimi gateway as an *external* source instead of pretending it is bundled.
4. **P2**: trial only one Copilot Go adapter, and optionally GLM Web for additional coverage; Kimi Code Go is separate optional source only for eligible authenticated accounts.
5. **P3**: defer desktop-bound AutoClaw, browser/CDP multi-web gateways, MiMo Desktop-bound components and legacy/deprecated Qwen adapters.

## 7. Release gates & privacy

- Never claim that a source is **integrated** merely because `connect-bridge.sh --list` contains it; require installed runnable service, valid models response and a real Chat Completions success.
- Prefer Go/Rust native ARM64; accept Node pure built-ins only when it materially improves account capacity; Python packages only as *optional* Termux installs.
- Respect providers' quotas, logins, account entitlements and terms. No automated CAPTCHA/challenge bypass or account-registration scaling.
- Preserve original upstream settings and account state under ignored data; avoid sending secrets to HTML/JS UI. Keep network loopback only and keep `config.json` private.
- Display three health layers **Process/Endpoint**, **Authenticated Accounts**, **Quota/Inferences Available**. Agent2API real-device finding (2/2 credentials but 503 low balance) proves these layers cannot be merged.
- Test healthy inference, SSE, full client-tool-client continuation, real 401/429/503 errors, graceful sidecar shutdown, and upgrade state/backups before declaring Android VERIFIED.

**Method:** Primary GitHub READMEs, project manifests and latest commit info inspected 2026-10-08, compared against current AhB Android assembly/config. Public repo features are claims until runtime tested.
