# Provider layout

## OpenCode Free

Runtime: `opencode2api v1.3.7`

Hub prefix: `opencode/`

Management UI: `http://127.0.0.1:8404/`

The Hub enables the upstream WebUI and generates a random local admin password during first setup. The password is stored in `data/opencode/webui-password.txt`.

Use the upstream UI for:

- Zen / Go keys
- anonymous/free model state
- proxy configuration
- quota
- Playground
- diagnostics
- logs
- model availability
- local gateway keys

## FreeBuff

Runtime: `Freebuff2API v0.10.3`

Hub prefix: `freebuff/`

Management UI: `http://127.0.0.1:8402/ui`

Use the upstream UI for:

- account import
- account pool health
- quotas / usage
- proxy
- model routing
- Playground
- logs and diagnostics

The Hub does not duplicate FreeBuff's SQLite/account logic.

## Agent2API optional pack

Runtime: `agent2api-server v2.9.5`

Hub prefix: `agent2api/`

Management UI: `http://127.0.0.1:8403/`

Install on Termux:

```sh
./scripts/install-agent2api-termux.sh
```

This is intentionally optional because its Rust build is much larger than the default two-provider V1.

The upstream currently contains adapters for personal accounts including CodeArts, Qoder, Cline, Trae, Loomy, AutoClaw, Accio, ZCode and others. The exact supported list is controlled by the upstream project and may change.

For this local V1 integration Agent2API is bound to `127.0.0.1` and `AGENT2API_ALLOW_NO_KEY=1` is used so the Hub can call it without storing another API secret. Do not expose port 8403 to LAN or the Internet in this mode.

This project does not add bulk-account creation or anti-abuse bypass behavior.