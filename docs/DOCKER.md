# AhB portable Docker / Compose

Supports **Linux AMD64 and Linux ARM64**. Docker Desktop can run this Linux container on macOS/Windows, but this is **not** a native macOS or Windows release.

## Install

On your own trusted machine:

```bash
git clone https://github.com/Tsenjii/AhB.git
cd AhB
printf 'AHB_PUBLIC_TOKEN=%s\n' "$(openssl rand -hex 32)" > .env
chmod 600 .env
docker compose up -d --build
docker compose ps
```

`.env` is ignored by Git. Never put its private key into GitHub, screenshots, chat messages, or public URLs. Docker Compose refuses to start without an `AHB_PUBLIC_TOKEN` secret.

- Dashboard: `http://127.0.0.1:8080/ui` using HTTP Basic (user `ahb`, password from your private `.env`).
- API Base URL: `http://127.0.0.1:8080/v1`, key `AHB_PUBLIC_TOKEN` (Bearer or Anthropic `x-api-key`).
- Network: host port `8080` binds to **127.0.0.1**, not to the public Internet. Use a secure HTTPS reverse proxy or private SSH tunnel if remote access is needed. Never expose the internal Hub or source-account management ports.
- Persistence: `ahb-data:/state` holds accounts, secrets, databases and configuration. `docker compose down` keeps it. **Do not run `docker compose down -v`** unless you want to erase data.
- Auth: every API/UI endpoint requires the token; `/healthz` only returns status. Container ingress rejects `/api/control/*` administrative operations to protect localhost-only controls.

## RAM and Gateway management

Default Linux settings: nine bundled Gateway definitions, all on-demand, only **one** may be resident at a time; idle services stop after 120 seconds. This is a process count policy, not a hard 512 MiB RAM guarantee. Check real peak usage and SSE/tool results before relying on it.

The private `/state/AhB/config.json` allows changing `resources.max_running_sidecars` (1–16), idle timeout and per-provider `proxy_url`; restart container to apply. The original Agent2API account-level proxy pool remains separate and preferred for that service. Docker's local provider toggle/restart UI needs a native supervisor hook, so those controls remain disabled instead of falsely claiming success. Account setup requires each source's original supported headless login procedure through a trusted private container shell (`docker exec -it ahb bash`) or secured tunnel; never publicly expose upstream admin pages.

## Package updates

The image verifies the published Linux release checksums and defaults to `linux-0f48b1165b46`. To use a newer **published** release, set `AHB_RELEASE_TAG=linux-<12-character-revision>` in your local `.env` and rerun `docker compose up -d --build`. Check the Linux assets actually exist. Never substitute Android Termux binaries.

## Platform direction

Native distributions remain Android Termux ARM64 and Linux AMD64/ARM64. macOS native support needs separate Darwin Go/Rust builds and process management. Windows native support additionally needs replacements for Unix shell scripts, signals and process/RSS monitoring. Docker Desktop is the recommended initial compatibility route while those prerequisites remain unfinished.

A green Docker smoke does not prove real account entitlements, tool calling, SSE completeness, or production-grade quotas.
