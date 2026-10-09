# AhB — Infrlo 512 MiB cloud trial

## Alternative: editable Build Command + Run Command (Infrlo Step 3)

The signed-in Infrlo interface may show **only two editable commands**, not a Dockerfile selector. In that case, on `main` after PR merge (or use branch `deploy/infrlo-custom-commands` for preview):

- **Build Command:** `python3 deploy/infrlo/install-command.py`
- **Run Command:** `python3 deploy/infrlo/run-command.py`
- Repository: `https://github.com/Tsenjii/AhB`; branch as chosen.

The Build Command uses only Python's standard library to download and SHA-256 verify the existing native Linux AMD64/ARM64 release. It uses **neither pip requirements nor the Dockerfile**. The Run Command publishes a small **Python-only authenticated** listener on the platform-provided `PORT` (fallback 5000 when Infrlo injects no PORT; the published Infrlo Python and Node sample projects use port 5000), with the original Go Hub and OpenCode on localhost. This **proof of concept enables OpenCode only**; the other eight installed providers remain disabled. No Node.js is needed for this smoke test.

To activate external API use, configure `AHB_PUBLIC_TOKEN` as a **private** environment variable containing at least 32 random characters, using an Infrlo application secret/settings page if offered, then restart. `AHB_PUBLIC_USER` defaults to `ahb`. Never put an API secret in the Build/Run Command, repo URL, Git commit, public chat, or screenshot. **Until the secret exists, the app is deliberately locked:** `GET /healthz` responds `{"status":"setup_required"}` and all other API requests return 503. If Infrlo provides no private variable support, use this only to test that an app can be started; do not expose upstream accounts.

This source-host path is experimental and still depends on Infrlo's **actual** OS, GitHub release download access, runtime persistence, and the port it forwards. It needs glibc compatible with the published Ubuntu 24.04 native Linux release. A GitHub CI success on Ubuntu cannot guarantee Infrlo's runtime, disk limits, or idle/ephemeral storage support. Avoid importing credentials until those facts are verified.

---


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