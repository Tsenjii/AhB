# AhB — Infrlo 512 MiB cloud trial

The publicly advertised free allocation is 512 MB RAM and 2 GB storage. We could not inspect the authenticated /create/app screen: **use these instructions only if it supports GitHub + Dockerfile builds and an HTTPS public domain**.

## App settings

| Item | Enter |
| --- | --- |
| Name | ahb-test |
| Source | https://github.com/Tsenjii/AhB |
| Branch | main (after PR merge), or deploy/infrlo-512mb for preview |
| Build | Dockerfile at repo root, context . |
| Container port | 8080 |
| RAM | 512 MiB |
| Health path | /healthz |
| Required private secret | AHB_PUBLIC_TOKEN = a random string of 32+ characters |
| Optional | AHB_PUBLIC_USER=ahb; PORT=8080 |
| Persistent volume, if offered | /state, writable by UID 1000 (node) |

Generate a test token with `openssl rand -hex 24`. Enter the result ONLY under private environment/secrets; never share or commit it. Do not reuse an existing provider credential.

## After deploy

Open https://YOUR_DOMAIN/healthz for minimal status, then https://YOUR_DOMAIN/ui. Browser Basic authentication: username **ahb** and password your private **AHB_PUBLIC_TOKEN**.

For OpenAI-compatible clients, Base URL = https://YOUR_DOMAIN/v1, API key = AHB_PUBLIC_TOKEN. Bearer and Anthropic x-api-key are both accepted. All UI/API routes require auth; /healthz does not reveal accounts; public /api/control/* is explicitly denied.

AhB remains on 127.0.0.1:8317 inside the container. The Docker image uses the pinned SHA-256-verified native Linux release linux-16bf4fd6593e, Node.js 22/trixie (sufficient glibc) and one on-demand gateway at a time. It does not build Go/Rust on the small Infrlo host.

Initially /v1/models may show zero models because all nine providers sleep. This is normal; a direct request using a known valid opencode/MODEL_ID wakes it. Once running, live model discovery can populate the bounded cache. The public dashboard's restart/toggle/wake operations are intentionally disabled: management UIs at localhost-only ports cannot be reached over this authenticated ingress. Configure other account-backed providers only through a trusted private console and persistent credentials, if the host supports it.

The startup script seeds /state/AhB on first boot and refreshes **only bundled code/static assets** on an updated image; existing config.json and account data are preserved. **Without a real persistent writable /state mount, every container recreation can lose accounts.** Treat the first run as disposable and do not import your valuable Termux credentials.

No claim is made that Infrlo supports Dockerfile builds, writable volumes, HTTPS, or continuous uptime: its logged-in configuration must verify these. A 512 MiB single-provider process ceiling is NOT a guaranteed memory limit; verify peak usage and do not equate health checks with usable account entitlement or full SSE/tool-call success.