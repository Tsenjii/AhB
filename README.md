# Android AI Hub

A lightweight Android/Termux AI gateway that supervises mature provider adapters and exposes one local API.

## What V1 does

Default providers:

- **OpenCode Free** via `opencode2api v1.3.7`
- **FreeBuff** via `Freebuff2API v0.10.3`

Optional provider pack:

- **Agent2API v2.9.5** — adds a mature UI and adapters for personal CodeArts, Qoder, Cline, Trae, Loomy and other supported accounts without reimplementing those protocols in this repo.
- **DeepSeek2API** — optional DeepSeek Web multi-account sidecar with its original admin UI, OpenAI/Anthropic/Responses compatibility, tool calling and provider-internal account pooling.
- **External localhost bridges** — optional `kind: external` providers that AhB health-checks and routes without supervising a process. A disabled `lmarena/` slot is included for a user-supplied local OpenAI-compatible bridge.

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
lmarena/<model>   # only when the external localhost slot is enabled
```

Supported proxy endpoints:

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/completions`
- `POST /v1/embeddings`
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

The Hub intentionally does not duplicate the upstream management consoles.

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

The public ARM64 bundle is rebuilt by GitHub Actions from pinned upstream versions and currently contains:

- `hubd`
- `opencode2api`
- `Freebuff2API`
- `agent2api-server`
- Agent2API's original management UI
- `deepseek2api` (disabled by default)
- DeepSeek2API's original management UI

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

AhB can attach an already-running localhost OpenAI-compatible bridge without
bundling or supervising it. The included LMArena slot defaults to
`http://127.0.0.1:8406` and is disabled by default:

```sh
./scripts/enable-lmarena-external.sh
```

The bridge must provide `GET /v1/models` and the OpenAI-compatible POST
endpoints you intend to use. AhB intentionally does not bundle browser-session,
Cloudflare-clearance, CAPTCHA, fingerprint-evasion, or anti-bot bypass logic.
