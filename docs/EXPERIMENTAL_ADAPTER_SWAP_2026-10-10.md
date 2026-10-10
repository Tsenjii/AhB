# Experimental DeepSeek / Duck.ai replacement (2026-10-10)

## What changed
- **DeepSeek Web** binary name and AhB provider prefix remain `deepseek2api` / `deepseek/`; upstream changed from zengtao227/Deepseek2API to the pinned [0xgetz/deepseek2api](https://github.com/0xgetz/deepseek2api) commit `41b924e7741e1619433469900625dca132e00d6c`. Pure Go, direct OpenAI Chat Completions, no native admin dashboard.
- **Duck.ai** binary name/prefix remain `duck2api` / `duckai/`; upstream changed from aurora-develop/Duck2api to pinned [desktop-tools-which-may-be-useful/duckai2api](https://github.com/desktop-tools-which-may-be-useful/duckai2api) commit `d8c6888eeb11daedb71a6ad589095f0a315626eb`. Only pure-HTTP Rust build: no browser/Playwright binary. Still disabled by default due real 418.
- **Agent2API** remains upstream official v2.9.9 commit `cd97bce9912225e055cc79761d96a3f0b77e26f8` already pinned on both platforms; revalidated 2026-10-10.

## Compatibility and protection
These are **experimental source swaps, not a guarantee of upstream availability**. Both remain disabled on fresh Android/Linux installs. Normal installed `config.json` preserves user-selected enabled flags and all account files; `prepare-configs.sh` migrates only the two sidecars' runtime contract without touching obsolete DeepSeek admin keys or account databases. Upgrading keeps the backup.

DeepSeek now requires a real user's authorized Web account token in `data/deepseek2api/accounts.txt`; it does not reuse old `data/deepseek2api/config.json`, legacy admin password or session data. Login is manual: **no new OAuth integration is claimed**. The enable script rejects empty/nonexistent account files, masks all tokens and enforces file mode 0600. The new adapter only implements Chat Completions and model listing. Existing `Responses`, `Messages`, and tool-call compatibility must be tested separately and must not be advertised if not supported.

For Duck, this source does not make HTTP 418 go away automatically. It is only a candidate to be tested; respect upstream limits and don't treat `/health` or listed models as usable quota. No CAPTCHA evasion or account-farming feature is added by AhB.

## Build and release gate
Both Android ARM64 and Linux AMD64/ARM64 bundle workflows fetch **immutable SHA** source revisions and compile against their native targets. The Rust Duck build uses `--no-default-features --features http` to exclude Chrome/Playwright. Do **not** ask the user to upgrade until both exact-commit build jobs succeed and `prebuilt/source-commit.txt` points to the merged revision. Once released, test via AhB's `/playground` direct per-Gateway endpoint with **no silent fallback**. Real user accounts and live provider HTTP 418/401 remain outside CI.

If the Rust build fails on Android/low memory, do not ship a mismatched or half-updated archive. Retain the previous working package and leave the experimental providers disabled. OpenCode, FreeBuff, Agent2API and all other gateways are unchanged by this experiment.

## Android Rust build and legacy upgrade compatibility
The pinned Duck Rust HTTP-only build indirectly resolves dlopen2 0.9.0, whose Android Unix import references an undeclared once_cell crate. The native ARM64 build performs a strictly version-matched, ephemeral import substitution to Rust 1.95 std::sync::LazyLock so Cargo.lock stays pinned and no browser runtime is introduced. This does not fix any upstream HTTP 418 challenge. The CI must pass before release.

Existing Android updaters require the old DeepSeek static/admin directory to exist in the new tarball. We include a nonfunctional placeholder only for upgrade compatibility; the new Go service does not provide a /admin page. Newer upgrade scripts stop requiring this path. Credentials remain private and must be reauthorized according to the replacement's documented format.
