# Android AI Hub

A lightweight Android/Termux AI gateway that supervises mature provider adapters and exposes one local API.

## What V1 does

Default providers:

- **OpenCode Free** via `opencode2api v1.3.7`
- **FreeBuff** via `Freebuff2API v0.10.3`

Optional provider pack:

- **Agent2API v2.9.5** — adds a mature UI and adapters for personal CodeArts, Qoder, Cline, Trae, Loomy and other supported accounts without reimplementing those protocols in this repo.

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
```

Supported proxy endpoints:

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/responses`
- `POST /v1/messages`

## UI

Open the Hub home page:

```text
http://127.0.0.1:8317/ui
```

It shows provider status, process RSS, restart counts, model counts and the unified model list.

Each healthy provider has a **管理原生 UI** button:

- OpenCode: `http://127.0.0.1:8404/`
- FreeBuff: `http://127.0.0.1:8402/ui`
- Agent2API when installed: `http://127.0.0.1:8403/`

The Hub intentionally does not duplicate the upstream management consoles.

## Termux install

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