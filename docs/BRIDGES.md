# AhB External Bridges — 2026-10-08

AhB integrates **already-running local, OpenAI-compatible bridge services** without duplicating their account/login UI. It does not reimplement their upstream protocols, download third-party account data, solve interactive challenges, or start external websites' browser sessions.

This is distinct from a first-class AhB sidecar: an **external bridge** is connected, health-checked and routed, but AhB does not install, spawn, restart or update that bridge. Installation of third-party bridges remains controlled by their own repositories and supported platforms.

## Connect in one command

Once the external service is running on the **same device** and you know its local port:

```sh
cd ~/AhB
./scripts/connect-bridge.sh lmarena http://127.0.0.1:5102
```

The script asks for an optional API key **without echoing it**, validates that the URL is loopback-only, checks the bridge's models endpoint, saves to `config.json`, retains a copy of the previous config under ignored `data/`, and leaves every other source intact. Then restart AhB:

```sh
./scripts/stop-termux.sh
./scripts/start-termux.sh
```

Open `http://127.0.0.1:8317/ui` and check the provider's health and `/v1/models`. New IDs use the `preset/model` prefix. For example, if the connected bridge advertises `foo`, its unified ID becomes `lmarena/foo`.

If the external bridge is not running, does not expose an OpenAI-compatible models list, or rejects the supplied key, **no new route is saved**. It must work independently before attaching to AhB. A reachable model list is not proof of real chat/tool-calling capability.

## Available presets

These are connection presets, **not bundled binaries or independently tested upstream accounts**. Ports below are examples; change them to match your bridge. Direct upstream project URLs are included so you can verify the actual service requirements and feature claims.

| Preset | Upstream implementation / notes | Example |
|---|---|---|
| `lmarena` | [Lianues/LMArenaBridge](https://github.com/Lianues/LMArenaBridge), browser-assisted Python bridge. Requires its own supported browser + backend; on-phone availability is unverified. | `./scripts/connect-bridge.sh lmarena http://127.0.0.1:5102` |
| `windsurf` | [dwgx/WindsurfAPI](https://github.com/dwgx/WindsurfAPI), Node runtime and provider-local state; its LS setup is not guaranteed on Termux. | `./scripts/connect-bridge.sh windsurf http://127.0.0.1:3003` |
| `qwen` | [XxxXTeam/Qwen2API_Go](https://github.com/XxxXTeam/Qwen2API_Go), Go service. | `./scripts/connect-bridge.sh qwen http://127.0.0.1:3000` |
| `kimi` | [chopper1026/kimi2api](https://github.com/chopper1026/kimi2api), Python + own account manager; its upstream requires a separate API key. | `./scripts/connect-bridge.sh kimi http://127.0.0.1:8000` |
| `gemini` | [xwteam/gemini2api](https://github.com/xwteam/gemini2api), Python service with a primary OpenAI-compatible prefix of `/openai/v1`; the script maps `/v1/...` automatically. | `./scripts/connect-bridge.sh gemini http://127.0.0.1:5918` |
| `claude` | [yushangxiao/claude2api](https://github.com/yushangxiao/claude2api), external Go service, maintains its own credentials. | `./scripts/connect-bridge.sh claude http://127.0.0.1:8080` |
| `codex` | [dvcrn/codex-oauth-proxy](https://github.com/dvcrn/codex-oauth-proxy), locally authenticated Codex service that exposes Chat/Responses and separate MCP `/mcp`. A user-authorized Codex CLI login and separately launched API bridge are prerequisites. This is **not ChatGPT browser-Web API**, nor an auto-installer. | `./scripts/connect-bridge.sh codex http://127.0.0.1:9879` |
| `cliproxy` | [router-for-me/CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI), independently installed OAuth-capable multi-account gateway covering Antigravity, Gemini CLI, Codex, Claude Code, Muse Code and other supported sources. **Not bundled**; no guaranteed Termux ARM64 binary. Give it its own local API key; its default port **8317 conflicts with AhB**, so configure `server.host: 127.0.0.1` and `server.port: 8416` in its private `config.yaml` first. Its own management key and provider OAuth stay in CLIProxyAPI. | `./scripts/connect-bridge.sh cliproxy http://127.0.0.1:8416` |
| any local API | Bring your own legitimate OpenAI-compatible gateway. ID must be lowercase ASCII (e.g. `mybridge`), length 2–31. | `./scripts/connect-bridge.sh mybridge http://127.0.0.1:8560` |

The matching connector can also be configured from the **Connect** wizard on the Hub homepage. The wizard produces a safe local terminal command rather than asking for your private API key in browser JavaScript.

### CLIProxyAPI source (optional, no binary bundled)

If CLIProxyAPI is **already installed and running** on the same host, configure it privately with the v8 YAML fields:

```yaml
server:
  host: "127.0.0.1"
  port: 8416
management:
  allow-remote: false
  secret-key: "<your-private-management-key>"
access:
  api-keys: ["<your-private-API-key>"]
```

These are **illustrative fields, not a complete config**. Copy its official `config.example.yaml` and retain its other settings. Do not commit real keys or paste them into chat. CLIProxyAPI's API key is entered **privately in Termux** when running `./scripts/connect-bridge.sh cliproxy http://127.0.0.1:8416`. The connector checks only the already-running source's `/v1/models`; it does not install CLIProxyAPI, handle upstream OAuth, guarantee plan entitlement, or validate real inference. A compatible Android executable must be separately verified; Linux ARM64 packages are **not** assumed to run on Termux.

When you want to check Antigravity or other CLIProxyAPI models, use AhB [Playground](../README.md) or a direct inference request; a model listing is not proof of an account being able to answer. Prefer the original CLIProxyAPI manager for its own login, quota and per-account proxy controls instead of duplicating them in AhB.

### Custom upstream API paths

An upstream whose models endpoint is not `/v1/models` can pass a third argument. AhB automatically maps all `/v1/*` pass-through paths to the same parent prefix:

```sh
./scripts/connect-bridge.sh mybridge http://127.0.0.1:8560 /api/v1/models
```

That attaches `GET /api/v1/models` and routes e.g. `POST /v1/chat/completions` through `/api/v1/chat/completions`. The prefix rewrite is only applied to the external connection that requested it. Native managed sidecars keep their existing paths.

### API keys

The script prompts you for the **bridge's own** downstream API key when a terminal is interactive. You may also provide it through an already-private process environment (`AIHUB_BRIDGE_API_KEY`). The saved key is in local `config.json` (mode 0600), not in Git or web-page source. Do not paste a session cookie, browser secret, or account password into AhB's model API key field. Keep provider-specific credentials inside the provider's own account manager.

## What AhB does and does not promise

- Supports namespace routing, model catalog, JSON Chat Completions, Responses, Anthropic Messages, embeddings, image-generation and text-to-speech pass-through **only when the upstream actually implements the requested endpoint**. AhB is a pass-through, not a translation service.
- Maintains SSE streaming forwarding and strips caller Authorization before applying bridge-specific local auth.
- Rejects LAN/public destination addresses for V1. `localhost` here refers to the device running AhB, not your computer or a remote VPS.
- Enforces the upstream's account/rate-limit/error behavior rather than falsely claiming every adapter can serve every model or tool.
- LMArena and other browser-driven bridges may be unstable or require manual sign-in or verification. AhB does **not** automate or defeat CAPTCHA, fingerprint checks, site restrictions, or account verification.
- The presets let you attach sources that you have **already installed and can legitimately use**. They are not a claim that every To-API repository on the market has been audited or works on Android ARM64.

## Troubleshooting

- **Connection refused**: start the actual bridge locally and verify its port.
- **HTTP 401 / 403**: configure the bridge's own API key/login through its supported methods; the AhB connector cannot repair third-party credentials.
- **Invalid model list**: check the service's models endpoint and optional third path argument.
- **DEGRADED / no models**: check the bridge's own model endpoint and logs; an empty account pool may be intentionally unavailable.
- **Connected but chat fails**: call the bridge's own endpoint directly first; its model list may exist even when inference or tools are not yet supported.
- **Need to undo a change**: the previous configuration is preserved at `data/bridge-config-before-last-change.json`, with local-only permissions. Stop AhB before restoring configuration and restarting.

Refer to the [Android device checklist](ON_DEVICE_CHECKLIST.md) for live model/tool-call verification, not just a successful CI build.


## Native tool-call release gate

A model list and a single `get_time` tool call do not prove a functional tool loop. Before declaring a connected provider tool-compatible, run `AIHUB_TEST_MODEL='provider/model' ./scripts/test-tool-roundtrip.sh`. The script first requires structured OpenAI `tool_calls`, then returns a synthetic tool result and requires a final model answer on the second request. This consumes real model quota. OpenCode's earlier device test confirmed only **initial tool-call emission**, not the completed two-turn sequence. See [GPT Web/Codex/MCP research and stability gates](GPT_WEB_CODEX_MCP_STABILITY_2026-10-08.md).
