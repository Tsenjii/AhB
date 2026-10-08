# Android AI Hub

A lightweight Android/Termux AI gateway that supervises mature provider adapters and exposes one local API.

## What V1 does

Default providers:

- **OpenCode Free** via `opencode2api v1.3.7`
- **FreeBuff** via `Freebuff2API v0.10.3`

Optional provider pack:

- **Agent2API v2.9.7 (next build; previously published 91c8da4 remains v2.9.6)** — adds a mature UI and adapters for personal WorkBuddy domestic/international, CodeArts, Qoder, Cline, Trae, Loomy, KukuAI and other supported accounts without reimplementing those protocols in this repo.
- **DeepSeek2API** — optional DeepSeek Web multi-account sidecar with its original admin UI, OpenAI/Anthropic/Responses compatibility, tool calling and provider-internal account pooling.
- **Grok2API** — optional Grok Build / Web / Console multi-account sidecar with its original management UI, quota/model sync, OpenAI/Anthropic/Responses support and media features.
- **Kiro-Go** — optional Kiro multi-account sidecar with its original Web admin, automatic token refresh, OpenAI/Anthropic/Responses endpoints and account-level proxy support.
- **GitHub Copilot (new opt-in Go sidecar)** — native ARM64 build wired to the bundle; use your own authorized GitHub Copilot Device Flow login, then enable. The binary has no built-in API-key authentication, so it remains localhost only. Real Android inference verification still required.
- **Kimi Web (new optional Termux installer)** — `chopper1026/kimi2api` account-pool backend and WebUI with local-only credentials and explicit opt-in; Python/Node dependencies are installed *only on request*, not in the base archive. Android installation/login/inference unverified.
- **External localhost bridges** — a universal attach tool and phone-first connection wizard for **LMArena, WindsurfAPI, Qwen2API, Kimi2API, Gemini2API, Claude2API**, and any custom local OpenAI-compatible service. These run separately; AhB connects to their local HTTP APIs but does not claim to install or operate the external bridge.

The Hub itself stays small. Provider-specific login, account pools, quota logic, proxy settings and diagnostics remain inside the upstream adapters.

## Unified API

```text
http://127.0.0.1:8317/v1
```

Models use explicit provider prefixes:

```text
opencode/<model>
freebuff/<model>
agent2api/<model>
deepseek/<model>
grok/<model>
kiro/<model>
copilot/<model>  # once enabled
kimiweb/<model>  # after optional Termux install and enable
lmarena/<model>
windsurf/<model>
qwen/<model>
kimi/<model>
gemini/<model>
claude/<model>
<custom-id>/<model> # only after connecting that local bridge
```

Supported proxy endpoints:

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/completions`
- `POST /v1/embeddings`
- `POST /v1/images/generations` (JSON pass-through, upstream must support)
- `POST /v1/audio/speech` (JSON pass-through, upstream must support)
- `POST /v1/responses`
- `POST /v1/messages`
- `POST /v1/messages/count_tokens`
- `POST /v1/systemone`

## UI

Open the Hub home page:

```text
http://127.0.0.1:8317/ui
```

It shows provider status, process RSS, restart counts, model counts and the unified model list.

Each healthy provider has a **管理原本 UI** button:

- OpenCode: `http://127.0.0.1:8404/`
- FreeBuff: `http://127.0.0.1:8402/ui`
- Agent2API when installed: `http://127.0.0.1:8403/`
- DeepSeek2API when enabled: `http://127.0.0.1:8405/admin`
- Grok2API when enabled: `http://127.0.0.1:8407/`
- Kiro-Go when enabled: `http://127.0.0.1:8408/admin`

The Hub intentionally does not duplicate the upstream management consoles.

### FreeBuff migration (Oct 9, 2026)

The old Rust `lza6/Freebuff-2API v0.10.3` was replaced by a pinned, dependency-free Node.js gateway from [yutian81/freebuff2api](https://github.com/yutian81/freebuff2api). This is **not** proof the upstream accepts every model. The old Web Cookie 409 `chat_moved` path is no longer used, but you must complete a **fresh authorized CLI/Bearer login** after upgrading. Old cookies, account files and SQLite data are preserved intact in the backup; old Cookie tokens are **never** converted or exposed. The freebuff/ prefix and localhost port 8402 remain. The new gateway does **not** offer the old /ui management dashboard.

```sh
cd ~/AhB
./scripts/check-freebuff-login.sh
./scripts/freebuff-login-termux.sh    # only if there are no new CLI tokens
./scripts/stop-termux.sh && ./scripts/start-termux.sh
```

Termux Node.js >=20 is installed by the prebuilt install/upgrade scripts. Python 3 is required for the optional interactive device-code login. See [login and migration guide](docs/FREEBUFF_ANDROID_LOGIN.md).

### All bundled sources: read-only readiness inventory

After the new package has been published and safely installed on your own Termux device, run:

```sh
cd ~/AhB
./scripts/provider-status-termux.sh
```

This reports OpenCode, FreeBuff, Agent2API, DeepSeek Web, Grok, Kiro, GitHub Copilot and separately the optional Kimi Web, including enabled status, process health, ready/account-count layers, model count and recent upstream HTTP code. It **never** reads or prints individual account details/tokens and does **not** spend inference quota. Unknown states remain UNKNOWN; HTTP 200 is not evidence of completed streaming/tools or actual available balance. See [provider stability rollout](docs/PROVIDER_STABILITY_2026-10-09.md).

**FreeBuff on Android:** the bundled Rust gateway needs no extra runtime to launch, but the desktop one-click login does not work natively inside Termux: upstream Windows embed uses WebView2, and desktop Chrome/Edge web login requires an extension on the same computer as the gateway. Use the upstream http://127.0.0.1:8402/ui Accounts import workflow on your phone instead of expecting a desktop extension to control Android localhost. After importing, run ./scripts/check-freebuff-login.sh (metadata only, prints no credentials). See [FreeBuff Android login guide](docs/FREEBUFF_ANDROID_LOGIN.md).

## Fastest Android install

For an ARM64 Android phone, no GitHub login and no on-device Rust/Go compilation are required:

```sh
pkg update
pkg install -y curl
curl -fsSL https://raw.githubusercontent.com/Tsenjii/AhB/main/scripts/install-prebuilt-termux.sh | bash
cd ~/AhB
./scripts/run-termux.sh
```

The installer downloads the public `prebuilt` branch bundle, verifies its SHA-256 checksum, prepares local secrets, and enables Agent2API when its prebuilt binary is present.

### Updating an existing Termux installation

Do **not** delete `~/AhB` or re-run the first-install script over an existing installation. For an existing prebuilt install, run:

```sh
curl -fsSL https://raw.githubusercontent.com/Tsenjii/AhB/main/scripts/upgrade-prebuilt-termux.sh | bash
```

The upgrade first verifies the downloaded archive, then preserves `config.json`, account data, databases, local secrets, and logs. It refreshes bundled upstream WebUI assets, stops the old AhB processes, and swaps the installation. The original directory is retained as `~/AhB.backup-YYYYMMDD-HHMMSS` for rollback. No user data is intentionally deleted.

Start the upgraded Hub with `cd ~/AhB && ./scripts/run-termux.sh`. In a second Termux session, run `./scripts/doctor-termux.sh` and `./scripts/smoke.sh`. New optional providers still require setup and real-device account/inference tests; a green CI build does not prove those paths work on Android.

**Release verified 2026-10-08:** Android ARM64 prebuilt source commit `91c8da481d455d3df6ef888b18a8e3001b2eb83c` built successfully. This does **not** mean the user has upgraded their existing phone or authenticated newly bundled providers. The latest inventory and evidence levels are in [Current Features and Platforms](docs/CURRENT_FEATURES_AND_PLATFORMS_2026-10-08.md).

The public ARM64 bundle is rebuilt by GitHub Actions from pinned upstream versions and currently contains:

- `hubd`
- `opencode2api`
- `Freebuff2API`
- `agent2api-server`
- Agent2API's original management UI
- `deepseek2api` (disabled by default)
- DeepSeek2API's original management UI
- `grok2api` (disabled by default)
- Grok2API's original management UI
- `kiro-go` (disabled by default)
- Kiro-Go's original management UI

## Termux install from source

```sh
chmod +x scripts/*.sh
./scripts/bootstrap-termux.sh
./scripts/run-termux.sh
```

The bootstrap generates local-only secrets into ignored files under `data/`; no real key or password belongs in Git.

OpenCode's generated UI password is stored at:

```text
data/opencode/webui-password.txt
```

Optional Agent2API pack:

```sh
./scripts/install-agent2api-termux.sh
```

Then restart the Hub.

Optional DeepSeek2API provider (the Android prebuilt already contains it):

```sh
./scripts/enable-deepseek2api.sh
```

Its local admin key is generated into `data/deepseek2api/admin-key.txt`. The AhB build applies a loopback-only listen patch to the pinned upstream source so this sidecar does not expose itself to the LAN.

## Reliability

`hubd` provides:

- child-process supervision
- startup readiness checks
- periodic health checks
- bounded restart with backoff
- independent provider failure domains
- graceful shutdown before forced kill
- streaming pass-through
- model aggregation
- client credential stripping before forwarding
- loopback-only V1 listener
- live Hub + sidecar RSS reporting

See:

- `docs/ARCHITECTURE.md`
- `docs/V1_SPEC.md`
- `docs/ON_DEVICE_CHECKLIST.md`
- `docs/NETWORK_EGRESS.md`
- `docs/PROVIDERS.md`
- `docs/DEPLOY_VPS_512MB.md`

## External To-API bridges

**LMArena and arbitrary local gateways:** AhB now includes a shared `connect-bridge.sh` helper, a clean mobile connection wizard, and upstream API-prefix rewriting for gateways such as Gemini2API.

Start your third-party bridge separately on the same phone, then run a single command:

```sh
cd ~/AhB
./scripts/connect-bridge.sh lmarena http://127.0.0.1:5102
```

It privately asks for an optional local bridge API key, verifies `GET /v1/models`, and only then enables the local route. Restart AhB to apply it. Supported named presets include `windsurf`, `qwen`, `kimi`, `gemini`, and `claude`; custom IDs work for other OpenAI-compatible localhost services.

```sh
./scripts/connect-bridge.sh --list
./scripts/connect-bridge.sh windsurf http://127.0.0.1:3003
./scripts/connect-bridge.sh kimi http://127.0.0.1:8000
./scripts/connect-bridge.sh gemini http://127.0.0.1:5918
```

The bridge must itself be installed, running, and legitimately usable. The presets are **connectors, not bundled upstream implementations**. No browser-session bypass, CAPTCHA solving, anti-bot evasion, or bulk account creation is included. See [External bridge guide](docs/BRIDGES.md) for project links, custom API paths, precise limits and troubleshooting.


### Optional Grok2API provider

The Android prebuilt contains the pinned Grok2API backend and its original Web
UI, but the provider is disabled until it is initialized:

```sh
./scripts/enable-grok2api.sh
```

The helper generates local Grok2API secrets, starts the sidecar temporarily on
loopback, creates a dedicated AhB Client Key through its local admin API, stores
the key under ignored `data/grok2api/`, enables `grok/`, then returns normal
process lifecycle control to hubd.


### Optional Kiro-Go provider

The Android bundle also contains a pinned Kiro-Go binary and its original Web
admin. It stays disabled until requested:

```sh
./scripts/enable-kiro-go.sh
```

AhB binds it to `127.0.0.1:8408`, supplies a local API key, and reads its
authenticated `/v1/stats` account totals so an empty account pool is not
treated as routable.

## New optional account-backed adapters (build/package status must be checked)

```sh
cd ~/AhB

# Copilot: lightweight native Go sidecar; requires a valid Copilot entitlement.
./scripts/login-copilot2api.sh
# Authorize with GitHub Device Flow, then Ctrl+C.
./scripts/enable-copilot2api.sh
./scripts/stop-termux.sh && ./scripts/start-termux.sh

# Kimi Web: separate optional Python/React install; not part of base archive.
./scripts/install-kimiweb-termux.sh
./scripts/enable-kimiweb-termux.sh
./scripts/stop-termux.sh && ./scripts/start-termux.sh
```

The Kimi Web native admin is `http://127.0.0.1:8412/admin`. Its password is stored privately in `data/kimiweb/admin-password.txt`. The `kimiweb/` managed adapter is separate from the original `kimi/` connection preset. Avoid adding duplicate account sources or auto-starting every service; optional providers are disabled by default. See [curated installation and optimization plan](docs/TO_API_INSTALL_AND_OPTIMIZATION_PLAN.md).

**Source status is distinct from usage entitlement:** a HEALTHY process, two configured credentials or a listed model do not prove quota availability. In the user's Android smoke test Agent2API returned HTTP 503 from two exhausted accounts. Treat provider quota and real inference as separate checks.

All previously found and future To-API candidates belong in the [living registry](docs/TO_API_REGISTRY.json), with explicit statuses, direct project URLs, validation steps and a history-preserving update policy.

### GPT / Codex bridge and real tool validation

The **new `codex` option is connector-only** (not a bundled GPT backend). If you have *already started and authenticated* your own localhost OpenAI-compatible Codex OAuth gateway such as [Codex OAuth Proxy](https://github.com/dvcrn/codex-oauth-proxy), you can attach it with:

```sh
cd ~/AhB
./scripts/connect-bridge.sh codex http://127.0.0.1:9879
./scripts/stop-termux.sh && ./scripts/start-termux.sh
```

The script requests only the **local gateway's client API key** and verifies its models endpoint; it does not log into ChatGPT or install/launch a third-party backend. The provider must have authorized Codex access and its own functional model endpoint. GPT Codex OAuth is not the same as controlling the ChatGPT **web browser** via MCP/Playwright. See [GPT Web vs Codex API vs MCP](docs/GPT_WEB_CODEX_MCP_STABILITY_2026-10-08.md).

To qualify any source for reliable function use, run a **full two-request tool-call test**, which verifies that an API can accept a function schema, return structured tool calls, accept a matching tool result and produce a final answer:

```sh
AIHUB_TEST_MODEL='opencode/YOUR_MODEL_ID' ./scripts/test-tool-roundtrip.sh
```

This test consumes upstream account quota. Do not call a provider tool-compatible based only on `/v1/models` or an initial single tool-call response. Kimi Web is experimental and disabled by default until real Android long-run/tool-continuation tests pass.

