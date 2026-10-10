# DeepSeek Web / Duck.ai current availability and login (2026-10-10)

**Current compiled source:** [15b2a47](https://github.com/Tsenjii/AhB/commit/15b2a47dea22a964755e6ed97ef0769b2baa1cda). The old DeepSeek admin WebUI and old Go Duck adapter are no longer bundled. Successful compilation is not proof of live inference.

## Experimental DeepSeek Web

- The bundled source is [0xgetz/deepseek2api](https://github.com/0xgetz/deepseek2api) pinned to `41b924e7741e1619433469900625dca132e00d6c`. It is pure Go, serves `deepseek/` at `127.0.0.1:8405`, and does **not** provide the old `/admin` UI.
- Real authorized Web account credentials must exist in a **private** `~/AhB/data/deepseek2api/accounts.txt`, one per line, permission 0600. The old `admin-key.txt`, previous JSON config and session data do not authenticate this new upstream. `./scripts/enable-deepseek2api.sh` rejects missing/empty accounts; it cannot sign in for the user.
- This upstream offers model listing and Chat Completions. Do not assume Responses, Anthropic Messages, native tool calling, or SSE compatibility without real validation. Test via AhB `/playground` using **direct provider routing** and no fallback.
- Fresh installations leave this experimental provider disabled. Existing account files/configs remain preserved for rollback without silently migrating incompatible secrets.

## Experimental Duck.ai

- The bundled source is [desktop-tools-which-may-be-useful/duckai2api](https://github.com/desktop-tools-which-may-be-useful/duckai2api), pinned to `d8c6888eeb11daedb71a6ad589095f0a315626eb`, compiled as Rust HTTP-only (`--no-default-features --features http`), without Chromium/Playwright.
- Android ARM64 and Linux AMD64/ARM64 native builds succeeded, but the user's previous **HTTP 418** has **not** been demonstrated fixed. `/health` only proves the local process is alive; models appearing in `/v1/models` do not prove usable upstream quota.
- New installs leave Duck disabled. Do not attempt to defeat service restrictions or automate repeated rejected requests; select an authorized supported source if necessary.

## Acceptance

1. Verify release SHA, CI, both builds and the Android prebuilt commit.
2. Upgrade non-destructively and preserve existing private accounts and backups.
3. On device, confirm a nonempty real chat response, SSE `[DONE]` where available and full tool-result continuation only when the provider supports it.
4. For Linux 512 MiB, record cold-start, steady and peak RSS and OOM events.

See [adapter migration](EXPERIMENTAL_ADAPTER_SWAP_2026-10-10.md) and [Android acceptance](ANDROID_FINAL_ACCEPTANCE.md).
