# AhB features, platforms and deployment-state inventory

**Snapshot checked 2026-10-08 Asia/Taipei.** This is the consolidated entry point for the **next AI's independent code audit**, *not* a substitute for checking HEAD again. Commit at snapshot: `947f26c9dec44e5daae16cf8b9ab422c2358b2ff` (docs only after published app code). Published `prebuilt/source-commit.txt`: `91c8da481d455d3df6ef888b18a8e3001b2eb83c`. [Android bundle success](https://github.com/Tsenjii/AhB/actions/runs/37792105281); [build-source CI success](https://github.com/Tsenjii/AhB/actions/runs/37792105269). **User has not upgraded their existing Android install**. Never call new runtime features phone-verified until user installs and tests them.

## Product goal and boundaries

**AhB = Android ARM64 Termux local account-backed API hub** that supervises independent To-API services. Core focus: account-backed local API service, opt-in sidecars, status, model/prefix routing, routing options, controlled restarts and low RAM. **NOT another ModelAtlas provider discovery/model catalog, NOT a fresh CLIProxyAPI clone**. The original upstream adapters own actual account login, OAuth, account rotation, credential refresh, quotas, WebUI and service-specific routing. This is critical: do not claim Hub itself manages an independent unified account pool across every adapter.

AhB's current `hubd` binds `http://127.0.0.1:8317` **loopback only**. It does **not** provide a fully authenticated public Internet endpoint, TLS, remote user/API-key isolation, hosted deployment, turnkey browser-UI Web quota reuse, or unlimited subscription entitlement. To expose anything outside localhost will require separate security review and explicit authorized design. Do not casually replace loopback binding.

## A. Actually bundled / prebuilt adapter and phone verification

| AhB prefix | Platform or portal covered | Status in published Android package | Activation and UI | Live phone evidence |
|---|---|---|---|---|
| `opencode/` | OpenCode Zen/Zen Go through [opencode2api](https://github.com/jasonxu114514/opencode2api), pinned v1.3.7 | Go binary, **enabled by default** | Hub-managed; `8404/` original UI; API `8401` | **YES**: actual Android HTTP 200 text/chat, 147 SSE data events + `[DONE]`, *initial* structured `get_time` tool emission. Full two-turn tool continuation still untested on phone. |
| `freebuff/` | FreeBuff/CodeBuff via [Freebuff2API](https://github.com/lza6/Freebuff-2API), v0.10.3 | Rust binary, **enabled by default** | `8402/ui` | On user's last phone report **zero accounts / DEGRADED**, no successful live inference. |
| `agent2api/` | Multi-platform [Agent2API](https://github.com/aimod-cc/agent2api), v2.9.6 headless | Rust binary + original UI; example disabled, prebuilt fresh installer enables where packaged | `8403/`; existing user install had this running | **2 credentialed accounts seen**; actual Qwen3.8-Flash inference returned **503 insufficient remaining balance**, despite 2/2 credential count. NOT verified usable on account quota. |
| `deepseek/` | DeepSeek **Web** via [Deepseek2API](https://github.com/zengtao227/Deepseek2API), pinned Go | Go binary + original admin UI, **disabled by default** | `./scripts/enable-deepseek2api.sh`, `8405/admin` | Not authenticated / inferred on device. |
| `grok/` | Grok **Build, Web and Console** via [Grok2API](https://github.com/chenyme/grok2api), pinned Go v3.1.6 | Go binary + React admin UI, **disabled by default** | `./scripts/enable-grok2api.sh`, `8407/` | Real account chat/tool on phone **not verified**. Author/community report relatively good Grok Build agent stability with occasional 429, tools and Web/Console differences. See [study](GROK2API_COMMUNITY_RELIABILITY_2026-10-08.md). |
| `kiro/` | Kiro accounts via [Kiro-Go](https://github.com/Quorinex/Kiro-Go), pinned Go | Go binary + admin UI, **disabled by default** | `./scripts/enable-kiro-go.sh`, `8408/admin` | Not yet authenticated / inferred on phone. |
| `copilot/` | **GitHub Copilot OAuth** via [copilot2api](https://github.com/whtsky/copilot2api), pinned Go | Go binary compiled for Android in successful package; **disabled by default** | `./scripts/login-copilot2api.sh`, `./scripts/enable-copilot2api.sh`, API `8410` | Independent Go Android compile/test passed; login, actual entitlement and inference **unverified on phone**. NOT ChatGPT Web/GPT Codex quota. |
| `kimiweb/` | **Kimi Web** via [kimi2api](https://github.com/chopper1026/kimi2api) | **Installer scripts only**, no Python runtime/source in base archive; **disabled** | `./scripts/install-kimiweb-termux.sh`, `./scripts/enable-kimiweb-termux.sh`, `8412/admin` if installed | **Experimental, reported unstable upstream category, full native tools not verified, Android pip/Node build unverified. Do NOT auto-install.** |
| `lmarena/` | LMArena Web browser bridge | **External connector slot only**, no gateway executable bundled | User must separately run legitimate service on example `5102` and attach with `connect-bridge.sh` | Never tested real bridge on device. |

**IMPORTANT:** Having a bundled binary != provider activated != valid account != actual model availability != real chat/SSE/tools != web-account quota eligibility. A single provider may have Build/Web/Console pathways with different limits.

### Agent2API subplatforms — **one bundled adapter, not 14 extra binaries**

Verify actual upstream capabilities and versions from [Agent2API README](https://github.com/aimod-cc/agent2api) and platform code before marking any individually verified:

- **WorkBuddy** China and International (distinct account types).
- **小浣熊 / Raccoon**, **CatPaw**, **AutoClaw** domestic/international.
- **Qoder**, **Cline** Free/Pass.
- **Accio** International/China, **ZCode** China/International.
- **CodeArts** Huawei Cloud, **Trae**, **Loomy**.
- **KukuAI** appears in provider modules/project description; independently verify presence in actual pinned v2.9.6 compiled binary and account creation menu.
- Account login, auto-refresh, wallet/benefit support **vary per upstream platform**; e.g. CodeArts supports own existing account benefit route, so [HITZY2002/codearts2api](https://github.com/HITZY2002/codearts2api) is a **redundant P3 alternative**, not a distinct platform to install.

Do not claim each Agent2API upstream subplatform has already been user-tested in AhB. Last two accounts were not spendable.

## B. Connector presets — **not built-in actual services**

The `scripts/connect-bridge.sh` and UI connection wizard offer the following IDs, **only when an independently running local OpenAI-compatible gateway already exists**:

| Prefix after attaching | Candidate external implementation | Caveat |
|---|---|---|
| `lmarena/` | https://github.com/Lianues/LMArenaBridge | Python/browser dependency; not locally bundled |
| `windsurf/` | https://github.com/dwgx/WindsurfAPI | External Node service; not locally bundled |
| `qwen/` | https://github.com/XxxXTeam/Qwen2API_Go | External Qwen Web implementation; not locally bundled, license/ARM64 needs audit |
| `kimi/` | https://github.com/chopper1026/kimi2api | External instance; distinct from optional managed `kimiweb/` installer |
| `gemini/` | https://github.com/xwteam/gemini2api | External Python; special `/openai/v1` prefix mapping |
| `claude/` | https://github.com/yushangxiao/claude2api | External Go; not locally bundled |
| `codex/` | https://github.com/dvcrn/codex-oauth-proxy | **Codex OAuth** local API & optional separate MCP, not ChatGPT browser Web; no binary or login bundled |
| `<user-id>/` | Any separately running localhost service (e.g. authorized CLIProxyAPI) | User-supplied ID; no automatic management of that service |

**CLIProxyAPI** (https://github.com/router-for-me/CLIProxyAPI): sensible umbrella for eligible Codex/Gemini CLI/Claude Code/Kimi Code/etc account-backed CLI services; **NOT currently installed as an AhB sidecar or independently authenticated live provider**. Connect its legitimate local API as external instead of reimplementing the same routes. **CLI-proxy accounts and browser-web chat allotments are not always the same**. Browser-based GPT Web MCP service != native OpenAI Tool Calling.

## C. AhB hub features — check code / tests, avoid hype

| Capability | Current source-level implementation | Owner / limitations |
|---|---|---|
| Android ARM64 public package | GitHub Actions builds Go/Rust adapters, original Web UI assets, bundled scripts, SHA256 checksums, `prebuilt` publication | Built successfully; **not proof all provider accounts work on Android** |
| Local API endpoint and dashboard | `hubd` loopback `8317`, responsive dark `/ui` with original-upstream UI links | No public authenticated API, remote access or global account admin console |
| One consolidated model list | `GET /v1/models`, namespaced `provider/upstream-model`; model warnings, model IDs preserved | Advertised models, not certified available tokens |
| Proxy protocol endpoints | Chat Completions, legacy completions, embeddings, Responses, Anthropic Messages, count_tokens, images, speech and systemone paths | Mainly *passthrough*. Each operation only works if individual upstream implements it; **no universal protocol translation** |
| SSE transport | Streams OpenAI/Anthropic-compatible response bodies from upstream without buffering | Validate each upstream/tool loop; HTTP 200 not proof SSE complete |
| Process lifecycle | Start/stop Termux, sidecar supervision, health checks, bounded restart/backoff, graceful shutdown | External connector providers must be started/stopped separately |
| Availability indicators | Process, Ready, Credentials, model count, RSS/restart; **Last API** (last upstream HTTP code/time, in memory, no payload) | 2/2 credentials or HTTP 200 does not equal quota; no global live per-model quota guarantee |
| Account lists, OAuth, auto-refresh, balances, provider quotas | **Original upstream gateway** UI/API, not a universal AhB account manager | Differs per adapter and per source |
| Routing and failover | Strict provider prefix; configurable same-model fallback and manual virtual route aliases; balanced/sequential options | Both **disabled in example config**. Avoid untested cross-provider tool/session failover claims. |
| Optional gateway attachment | Localhost-only `connect-bridge.sh`, dashboard command wizard, prefixed ID, API path prefix rewrites, private client key support | Connector-only; needs external running service and real `/v1/models` |
| Configuration and secrets | Local random keys, loopback validation, copied `config.json`, sensitive files mode 0600; atomic provider migration | State under ignored `data/`; **never read/commit users' tokens** |
| Safe upgrade/rollback | `upgrade-prebuilt-termux.sh` validates SHA256, stops processes, copies persisted state, swaps dirs, retains prior `~/AhB.backup-*` | User hasn't performed latest upgrade. Keep previously known-good backup. |
| Validation scripts | `doctor-termux.sh`, `smoke.sh`, `test-chat.sh`, `test-stream.sh`, `test-tool-roundtrip.sh`, `test-provider-acceptance.sh`, Grok-specific readiness acceptance | Unit/mock CI green. Genuine account-backed tests run manually and consume quota |
| Persistent request history/usage dashboard | **Not a generalized central metrics database.** Last HTTP status is process-memory-only; provider usage stored by upstream if supported | Do not invent central TPS/TTFT, historical quota, true success rate or every-model capabilities |
| User-facing signup/challenge handling | Not present | Keep authorized sign-in using original official/UI pathways; no abuse of platform protection |

## D. What has actually been demonstrated on a real Android phone?

- Prior tested device installation: Hub 200 and `{"status":"ok"}`, AhB dark UI, 15 listed models (14 OpenCode + 1 Agent2API), OpenCode Nemotron real chat HTTP 200 + content, OpenCode 147 SSE events + `[DONE]`, one generated structured `get_time` call.
- Existing account data survived an earlier update; previous backup `~/AhB.backup-20261008-194847` must not be deleted until the new phone version is verified.
- Agent2API had 2/2 credentialed accounts but actual Qwen3.8-Flash request returned HTTP 503 due insufficient quota; FreeBuff had no accounts. That is a provider/user account usability failure, not a reason to advertise 2/2 as ready for inference.
- **No** verified phone evidence yet for full two-turn OpenCode tool continuation, actual Grok Web/Build/Console, DeepSeek Web, Kiro, Copilot, optional Kimi Web, Codex/GPT Web, LMArena, Qwen, Gemini or Claude external bridges.

## E. Next AI's requested audit deliverables

1. **Read and independently inspect** `README.md`, `AI_HANDOFF.md`, `config.example.json`, `internal/config/`, `internal/hub/`, `internal/sidecar/`, `scripts/`, build workflow, `docs/PROVIDERS.md`, `docs/BRIDGES.md`, `docs/TO_API_REGISTRY.json`, `docs/NEXT_PHONE_ACCEPTANCE.md`, and the original Agent2API upstream provider module list. Do not rely only on README claims; compare actual code and tests.
2. Return **three tables**: (A) actual built Hub features and their verification state; (B) integrations/platforms with separately counted native binary vs Agent2API subplatform vs connector-only vs unintegrated research; (C) overlaps/gaps against CLIProxyAPI, Agent2API and ModelAtlas.
3. Independently verify CI, published Android source SHA, package content and rollback; flag any stale docs or broken script edge cases with exact file locations.
4. Make a ranked **P0 fix now / P1 after phone evidence / P2 hold** plan emphasizing OpenCode regression-free upgrades, actual Grok/DeepSeek tool/stream tests, honest account quota reporting and low RAM. Do not add another CodeArts binary, Kimi Web default service, CLIProxyAPI duplicate bridge or browser Chromium workload absent tangible feature gap.
5. User has **not yet installed the latest prebuilt** and will test later. Do not fabricate account login/inference results, request secrets, delete backups, auto-enable disabled adapters, or expose unauthenticated local API to the public Internet. Changes to existing code should have clear tests and commit references.
6. User's central intent is to use **account-entitled Web/CLI capacity** through unified local API where the source supports it; never assume one platform's paid web chat quota is a transferable API quota or that one successful chat implies native full tool calling.

Current phone guide: [NEXT_PHONE_ACCEPTANCE.md](NEXT_PHONE_ACCEPTANCE.md). Research register: [TO_API_REGISTRY.json](TO_API_REGISTRY.json). Development handoff: [../AI_HANDOFF.md](../AI_HANDOFF.md).
