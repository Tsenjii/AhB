# FreeBuff Node gateway: Android / Termux device login (2026-10-09)

AhB replaced the legacy Rust `lza6/Freebuff-2API v0.10.3` with the independently reviewed source [yutian81/freebuff2api](https://github.com/yutian81/freebuff2api), pinned to commit `e0d8c9d4d1a955f46fbeb6841ea192bd8b3f2109` (Oct 8, 2026). The upstream repository's local Node entrypoints are `server.js` and `worker.js`, requiring Node.js >=20 but no npm dependencies.

**This replacement is not evidence of real FreeBuff inference success.** The previous user's one Web-Cookie account received `web chat HTTP 409 chat_moved` (wrapped in HTTP 502) across three models. The new gateway uses its own CLI/Bearer endpoint instead of the broken legacy Web Cookie relay. Still confirm official account eligibility and real, user-authorized model inference after installing the new package.

## Account and data preservation

- Existing `~/AhB/data/freebuff/` is kept, including Rust's `tokens.json`, `freebuff2api.sqlite`, `web_threads.json`, config and any credential data; the entire pre-upgrade installation is also saved to a timestamped backup.
- Legacy Web Cookie credentials **cannot** be assumed equivalent to FreeBuff CLI Bearer credentials. We **do not** extract, convert, delete, sync or submit the old cookies.
- The new gateway is copied into `data/freebuff/gateway/`. During upgrading, only this replaceable source directory is refreshed from the new package; `data/freebuff/credentials/` and local saved authorized CLI logins survive.
- AhB's old `freebuff/` model namespace and loopback port **8402** remain; the gateway has **no legacy Rust management UI**.

## One-tap local Dashboard OAuth (new, pending release)

AhB now integrates the existing authorized FreeBuff/Codebuff CLI device flow into **Dashboard → 帳號與登入 → FreeBuff**. Tap **以 Google 登入 FreeBuff** once. AhB requests a one-time link from the official `https://www.codebuff.com/api/auth/cli/code` endpoint and opens that *official site* in the browser; Google sign-in itself stays on the official browser flow. A background process checks `/api/auth/cli/status` (up to five minutes) and saves only the completed account's `authToken` in a private owner-only file under `data/freebuff/credentials/`. It never sends that token to Dashboard, GitHub, Telegram or an AhB log. Existing authorized accounts are preserved and multi-account login can be repeated. This is a CLI-device OAuth flow, **not** a new Google OAuth client that accepts Google passwords or a way to skip official login.

After success, press **重新載入 FreeBuff 帳號** in the same panel; the Hub wakes a sleeping Gateway or safely restarts an idle running one. If the current Gateway has a live chat or SSE request, it will refuse to interrupt it; retry the reload after those requests finish.

The original Termux script remains as an offline/manual fallback only. The panel is limited to local authenticated AhB controls. One-time authorization links must not be copied into logs or shared with other users. A successful OAuth link merely means a token was saved; use live model inference and quota checks to verify usable account entitlement.

## Steps on Android after a verified published release

Upgrade with `scripts/upgrade-prebuilt-termux.sh`, which safely installs Termux Node.js and takes a backup before replacing the binaries and gateway source. Then:

```sh
cd ~/AhB
./scripts/start-termux.sh
./scripts/check-freebuff-login.sh
```

If accounts=0, authorize the new CLI account **on your phone**:

```sh
cd ~/AhB
./scripts/freebuff-login-termux.sh
```

The helper uses the upstream device-code login and asks you to open a one-time link for your own account. It stores the resulting Bearer token in owner-only files under `data/freebuff/credentials/` without echoing it to the terminal, and never uploads private credentials to GitHub or AhB. Do **not** publish the login URL or any account files. The login requires Python 3 standard library (Termux: `pkg install python` if absent). The bundled script suppresses upstream token echo. The CLI credential file is saved separately as `data/freebuff/freebuff_credentials.json` and the importer writes owner-only `*.json` files containing `authToken` to `data/freebuff/credentials/`. The Node gateway reads them at startup.

After authorized login:

```sh
cd ~/AhB
./scripts/stop-termux.sh
./scripts/start-termux.sh
./scripts/check-freebuff-login.sh
./scripts/provider-status-termux.sh
curl -fsS http://127.0.0.1:8317/v1/models | jq -r '.data[].id | select(startswith("freebuff/"))'
```

If a model is listed **and is authorized for that account**, opt into one real inference (consumes quota):

```sh
AIHUB_TEST_MODEL='freebuff/ACTUAL_LISTED_MODEL' ./scripts/test-chat.sh
```

Only after chat passes, separately test SSE and tool-result continuation. `/healthz` returns **observed alive / unknown / unhealthy candidates**, not guaranteed working quotas. Unknown accounts can be tried for first acceptance but are not claimed to have succeeded.

## Important boundaries

- New adapter's `/healthz` is public **only on phone loopback** and provides metadata without original tokens. `/v1` has the generated local AhB API key. No `0.0.0.0` exposure.
- The upstream Node app's CLI credential importer replaces the earlier non-Windows Rust WebView onboarding. No browser extension or third-party hosted panel is necessary.
- Account credentials are loaded from `data/freebuff/credentials/`, not via a process command-line flag. Do not pass secrets in shell history or chats.
- Never run endless 429/403/409 retries; no guarantee that a provider/model currently accepts your account.
