# Android AI Hub

A lightweight Android/Termux AI gateway that supervises mature provider adapters and exposes one local API.

## Portable Docker (Linux amd64/arm64)

See [Docker Compose instructions](docs/DOCKER.md) for a private, persistent, Linux-based API container. On macOS/Windows use Docker Desktop; no native binaries are claimed.

## Two editions, one shared codebase

AhB is maintained as **two separate native distributions**. Both build
the same Go Hub and nine pinned upstream gateways. Agent2API remains
**unmodified**.

| Policy | Android Termux | Native Linux VPS |
| --- | --- | --- |
| Platform | Android ARM64 | Linux AMD64 / ARM64 |
| Default providers | Nine bundled; DeepSeek Web / Duck.ai disabled pending live verification; others on demand | Nine bundled; DeepSeek Web / Duck.ai disabled pending live verification; others on demand |
| Startup | On demand | On demand |
| Maximum resident on-demand providers | **3** | **1** |
| Idle cleanup | **900 seconds** | **120 seconds** |
| Release | Android `prebuilt` branch | Independent Linux GitHub Release |
| GUI toggles | Controlled local restart | Controlled local restart |

These are concurrency/process caps, **not guaranteed RAM limits**. Disabled
providers consume no child-process RSS, and switching to a sleeping backend
can release an idle sidecar. Long-running SSE requests hold an active lease.

**Bundled:** OpenCode2API (Go), FreeBuff2API (Node), Agent2API (Rust),
experimental 0xgetz DeepSeek Web (Go), Grok2API (Go), Kiro-Go (Go), Copilot2API (Go),
Gemini Web2API (Go), experimental Duck.ai2API (Rust HTTP-only).
Each requires its own legitimate account/authorization and may have
different quota and model capabilities.

**Not bundled:** Kimi Web, LMArena, Windsurf, standalone Gemini API bridges or any other
generic local connector; those are not installed services. Existing Kimi
data and custom bridges are not discarded by migration. The historical
Termux Python Kimi installer remains for old installations only.

The Linux CI builds separate, native ELF binaries; **Android packages are
not Linux packages**. Linux packages require an appropriate GNU/Linux
environment and Node.js for FreeBuff. See
[512 MB VPS guide](docs/DEPLOY_VPS_512MB.md) and
[technical constraints](docs/512MB-DEPLOYMENT.md).

**Release status (2026-10-10):** main source `0f48b1165b468658e34ea273f3a9b02ed147b2af` passed full CI/race and Android ARM64/native Linux AMD64/ARM64 bundle builds; the Android `prebuilt/source-commit.txt` matches this SHA and the checksum-verified [native Linux release](https://github.com/Tsenjii/AhB/releases/tag/linux-0f48b1165b46) is published. Android upgrades still require user action; packaging success is not evidence of real account quota, WebUI SSO, or live OAuth on the phone.

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
geminiweb/<model>
duckai/<model>
gemini/<model> # only if attached externally
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

**On-demand model discovery (Android and 512 MiB Linux):** The dashboard's **逐一載入全部模型** button wakes each enabled on-demand gateway **sequentially**, fetches only its model-list metadata, and releases the process before moving on. Plain `GET /v1/models` never wakes all gateways just to enumerate choices. For a successfully discovered gateway that is later asleep, the Hub can return its last six hours of model identifiers and numeric context limits from a private `data/hub-model-catalog.json` cache. Cached rows explicitly carry `x_cached: true`, `x_cache_state: "sleeping_unverified"` and `x_cached_at`; a source with no recent discovery remains **SLEEP**, not zero-model verified. The cache contains no credentials or arbitrary upstream metadata, is limited to 2 MiB, and persists safely through normal upgrades. Cached entries never prove current account entitlements, model availability, quota, tool calling or a healthy process. Every real inference request still rechecks on-demand process/account readiness; cached entries do not enable unverified virtual aliases.

Each healthy provider has a **管理原本 UI** button:

- OpenCode: `http://127.0.0.1:8404/`
- FreeBuff: no WebUI in the bundled Node adapter
- Agent2API when installed: `http://127.0.0.1:8403/`
- Experimental DeepSeek Web: no native admin WebUI; authorized account file is required.
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

This reports all nine bundled sources (OpenCode, FreeBuff, Agent2API, DeepSeek Web, Grok, Kiro, GitHub Copilot, Gemini Web and Duck.ai), plus separately the optional Kimi Web, including enabled status, process health, ready/account-count layers, model count and recent upstream HTTP code. **Recent Duck.ai HTTP 418 must be treated as a rejected inference even if local /ping is healthy. DeepSeek Web needs a real Web account; the admin key alone is not an account.** For onboarding details see [DeepSeek/Duck.ai status](docs/DEEPSEEK_DUCK_AVAILABILITY_2026-10-10.md). It **never** reads or prints individual account details/tokens and does **not** spend inference quota. Unknown states remain UNKNOWN; HTTP 200 is not evidence of completed streaming/tools or actual available balance. See [provider stability rollout](docs/PROVIDER_STABILITY_2026-10-09.md).

**FreeBuff on Android:** the Node.js gateway has no old Rust `/ui` account manager. Instead open AhB Dashboard → **帳號與登入 → FreeBuff** and click **以 Google 登入 FreeBuff** to launch its existing official Codebuff CLI browser authorization; after approval, AhB saves only the authorized token to the private credential directory and offers **重新載入 FreeBuff 帳號**. A manual fallback remains `./scripts/freebuff-login-termux.sh`. Old Web Cookies cannot be reused as new CLI credentials. See [FreeBuff guide](docs/FREEBUFF_ANDROID_LOGIN.md) and [Android final acceptance](docs/ANDROID_FINAL_ACCEPTANCE.md).

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

**Release gate verified 2026-10-10:** Android ARM64 `prebuilt/source-commit.txt` matches `0f48b1165b468658e34ea273f3a9b02ed147b2af` ([Android build](https://github.com/Tsenjii/AhB/actions/runs/37973222819)), native Linux AMD64/ARM64 build and release succeeded ([Linux build](https://github.com/Tsenjii/AhB/actions/runs/37973222867)), and [source CI](https://github.com/Tsenjii/AhB/actions/runs/37973223002) passed. This is not a phone installation or real credential/SSO/streaming entitlement test. For the one-time Android update see [final acceptance](docs/ANDROID_FINAL_ACCEPTANCE.md).

The public ARM64 bundle is rebuilt by GitHub Actions from pinned upstream versions and currently contains:

- `hubd`
- `opencode2api`
- `freebuff2api` (Node.js launcher; gateway sources under `data/freebuff/gateway/`)
- `agent2api-server`
- Agent2API's original management UI
- `deepseek2api` (configured on demand; authenticated inference not yet verified)
- DeepSeek2API's original management UI
- `grok2api` (configured on demand; authenticated inference not yet verified)
- Grok2API's original management UI
- `kiro-go` and its original management UI
- `copilot2api` (requires authorized account login)
- `gemini-web2api-go` (Gemini Web route)
- `duck2api` (Duck.ai route)

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

Optional experimental DeepSeek Web provider (once the updated Android bundle is actually published and installed):

- Source: [0xgetz/deepseek2api](https://github.com/0xgetz/deepseek2api), pure Go, pinned at `41b924e7741e1619433469900625dca132e00d6c`.
- This adapter has **no** old DeepSeek admin WebUI or automatic OAuth login. An authorized account token must exist in the **private** `data/deepseek2api/accounts.txt` (one per line, mode 0600). Previous upstream admin keys/config/account data remain preserved, but **are not compatible credentials**.
- Enable with `./scripts/enable-deepseek2api.sh` only after setting up your own authorized Web account. Then check `deepseek/deepseek-chat` directly at `/playground`; process `/health` is not inference proof.
- Experimental Duck.ai now uses the pinned [Rust HTTP adapter](https://github.com/desktop-tools-which-may-be-useful/duckai2api) (`d8c6888eeb11daedb71a6ad589095f0a315626eb`) with browser mode excluded; it remains **disabled by default** and known HTTP 418 is not claimed fixed. Its localhost `/health` does not prove live inference.
- Agent2API remains **v2.9.9**, the newest upstream release checked 2026-10-10; no unnecessary re-pin. See [adapter migration](docs/EXPERIMENTAL_ADAPTER_SWAP_2026-10-10.md).

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

It privately asks for an optional local bridge API key, verifies `GET /v1/models`, and only then enables the local route. Restart AhB to apply it. Supported named presets include `windsurf`, `qwen`, `kimi`, `gemini`, `claude` and **optional `cliproxy`** for a separately running [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) instance with Antigravity/Codex/Claude/Muse OAuth. Configure it to **127.0.0.1:8416** first: its original 8317 listener collides with AhB. Native Android binary/login are not yet validated; this is **only an opt-in connector**, not an extra bundled daemon.

```sh
./scripts/connect-bridge.sh --list
./scripts/connect-bridge.sh windsurf http://127.0.0.1:3003
./scripts/connect-bridge.sh kimi http://127.0.0.1:8000
./scripts/connect-bridge.sh gemini http://127.0.0.1:5918
./scripts/connect-bridge.sh cliproxy http://127.0.0.1:8416
```

The bridge must itself be installed, running, and legitimately usable. The presets are **connectors, not bundled upstream implementations**. No browser-session bypass, CAPTCHA solving, anti-bot evasion, or bulk account creation is included. See [External bridge guide](docs/BRIDGES.md) for project links, custom API paths, precise limits and troubleshooting.

**Agent2API pin:** the reviewed upstream [v2.9.9](https://github.com/aimod-cc/agent2api/releases/tag/v2.9.9) commit `cd97bce9912225e055cc79761d96a3f0b77e26f8` is used by Android ARM64/Linux AMD64/ARM64 native build workflows and the optional Termux source installer. This version fixes crash and SSE-close handling and improves Qoder / account limits. Publishing new binaries and actual phone acceptance require separate successful build gates.


### Optional Grok2API provider

The Android prebuilt contains the pinned Grok2API backend and its original Web
UI. Its account/admin setup is not complete until it is initialized:

```sh
./scripts/enable-grok2api.sh
```

The helper generates local Grok2API secrets, starts the sidecar temporarily on
loopback, creates a dedicated AhB Client Key through its local admin API, stores
the key under ignored `data/grok2api/`, enables `grok/`, then returns normal
process lifecycle control to hubd.


### Optional Kiro-Go provider

The Android bundle also contains a pinned Kiro-Go binary and its original Web
admin. Complete its account initialization before attempting real inference:

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

The Kimi Web native admin is `http://127.0.0.1:8412/admin`. Its password is stored privately in `data/kimiweb/admin-password.txt`. The `kimiweb/` managed adapter is separate from the original `kimi/` connection preset. Avoid adding duplicate account sources or auto-starting every service; unconfigured sources are not usable for inference until authorized; on-demand services stay asleep when idle. See [curated installation and optimization plan](docs/TO_API_INSTALL_AND_OPTIMIZATION_PLAN.md).

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

