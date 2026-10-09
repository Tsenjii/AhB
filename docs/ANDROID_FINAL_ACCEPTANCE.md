# AhB Android one-time release acceptance (0f48b1165b46)

**Status:** the published Android ARM64 bundle and checksum-verified Linux AMD64/ARM64 native release are built from source commit `0f48b1165b468658e34ea273f3a9b02ed147b2af`. GitHub CI/race and package tests passed. This **does not** assert that any real phone has installed it or that personal OAuth/quota, SSE/tool calls, proxy or all upstream WebUI versions work.

## 1. Before updating (user initiates locally)

Wait until any live OpenCode/Muse/Agent2API inference, SSE stream, login or other work is finished. Keep the phone charged and enough free storage for a complete backup. **Never share tokens, passwords, OAuth URLs, or credential files in screenshots.**

In Termux, from the device that already has `~/AhB` installed:

```bash
cd ~
curl -fL --retry 3 \
  -o upgrade-ahb.sh \
  https://raw.githubusercontent.com/Tsenjii/AhB/main/scripts/upgrade-prebuilt-termux.sh
bash upgrade-ahb.sh
```

The updater checks Android ARM64 and SHA-256, validates the new package, stops old AhB-owned processes before copying databases, and saves `~/AhB.backup-YYYYMMDD-HHMMSS` for rollback. A detected unknown/corrupted installation fails closed rather than overwriting accounts. It does **not** automatically start the upgraded Hub.

After `Upgrade installed`:

```bash
cd ~/AhB
./scripts/start-termux.sh
./scripts/doctor-termux.sh
./scripts/provider-status-termux.sh
curl -fsS http://127.0.0.1:8317/healthz
```

If anything fails, **stop before repeating the updater**; inspect non-secret errors. Do not delete backups or apply random reinstall scripts.

## 2. Dashboard: verify one-stop account management

Open [AhB local Dashboard](http://127.0.0.1:8317/ui) **on the same Android phone**. Under **帳號與登入**:

- FreeBuff → `以 Google 登入 FreeBuff` opens the Codebuff official website. Approve on that site, wait for a status of `done`, confirm count increased, then `重新載入 FreeBuff 帳號`. Do not share the one-time URL or Bearer token. No real account is tested by CI.
- Native console center → OpenCode, DeepSeek, Gemini Web, Grok and Kiro show a native management link and `複製管理密碼` when an AhB-owned private secret exists. An asleep, enabled on-demand Gateway should wake before the original native UI is opened. Inactive or capacity-blocked sources should explain why; they must not pretend to be logged in.
- Agent2API continues to use its original management/login mechanism, and existing account/session files should still be present. AhB does **not** claim cross-service passwordless SSO or accept `admin/0000` as a universal built-in default. Browsers may remember native sessions normally.

## 3. Gateway stability and resource control

In **總覽**, confirm OpenCode and Agent2API still work **before** enabling more sources. Verify actual answer content and complete SSE finish, and two-turn structured tool calls only if your account's source supports them. Metadata model lists, HTTP 200 or account counts alone cannot prove quota.

Test a safe **single-Gateway restart** using an idle on-demand source that needs recovery; it must refuse to interrupt active inference/SSE and enforce a 45-second cooldown. A 503 caused by exhausted quota, expired account auth or provider-side failures may not be fixed by restarting; no inference is automatically retried.

In **進階與診斷**, adjust `max_running_sidecars` (1–16) and idle stop seconds (30–86400) only after checking real Android RAM usage. Proxy is a per-process best-effort `HTTP_PROXY`/etc override; account-specific proxy pools in Agent2API remain native. Verify a proxy with a real permitted source; do not rely on settings alone.

## 4. Linux/Docker parity and limitations

The matching native Linux release is [`linux-0f48b1165b46`](https://github.com/Tsenjii/AhB/releases/tag/linux-0f48b1165b46), with AMD64 and ARM64 checksum assets. Generic [Docker Compose instructions](DOCKER.md) use the same published Linux release. In Docker, upstream native loopback UI links are intentionally unavailable to remote browsers; use a trusted host shell/private connection if original upstream account login is needed.

Mac/Windows via Docker Desktop runs a **Linux container**, not native installers. Native Darwin and Windows porting still needs dependent Gateway builds and process management; it is **not** included in this version.

## 5. Reporting a problem

Send only the sanitized `./scripts/provider-status-termux.sh` table, the exact button/action, and a **redacted** error message. Do not send `config.json`, tokens, API keys, original admin passwords, OAuth links or raw login logs.
