# AhB 512 MiB deployment plan (staged, not yet a published release)

## What is actually bundled

Android ARM64 base package contains **nine** real provider processes: OpenCode
(Go), FreeBuff (Node 22), Agent2API (Rust), DeepSeek (Go), Grok (Go), Kiro
(Go), Copilot (Go), Gemini Web (Go), and Duck.ai (Go). The AhB Hub is a separate Go process.

LMArena is an **external** third-party bridge and is not installed by the
bundle. Kimi Web is **not bundled**; the legacy Python installer remains only
for existing users until a reliable replacement is validated. Windsurf, Qwen,
Claude and any other imported local gateways must be installed and
run independently; no such gateway is silently bundled.

The default `config.example.json` no longer pretends uninstalled bridges are
running. Existing user-selected providers and already-installed Kimi data are
preserved by migration. No credentials, database or user-specific custom routes
are removed in the course of cleanup.

## Low RAM mode

On a **fresh** installation all nine bundled entries default to
`enabled=true` but `start_mode=on_demand`: "enabled" means *permitted to
start*, not *seven simultaneous running background services*.

- `resources.max_running_sidecars=1` caps concurrently resident managed
  on-demand processes. A second request to a different provider while a
  stream is active receives HTTP 503, rather than killing that stream or
  silently starting another large runtime.
- A direct `provider/model` inference request starts its sidecar if sleeping.
  The gateway holds a lease until the complete HTTP body/SSE is sent.
- Inactive on-demand services shut down after
  `resources.idle_stop_seconds=120` (when zero leases remain). Switching
  platforms first evicts any unused resident sidecar.
- `GET /v1/models` intentionally does **not** wake all sleeping providers.
  Use the UI's `啟動並載入模型` control to inspect the provider's current
  models or specify the known `provider/model` identifier directly.
- The UI switch disables a provider persistently, via the existing graceful
  restart workflow. An enabled-but-sleeping provider does not consume provider
  process RSS; the AhB Hub remains running.
- On **existing installations**, previously selected `enabled` flags are
  preserved, and stock bundled providers gain `start_mode=on_demand`
  unless the user explicitly set a mode. Custom local routes are untouched.

`max_running_sidecars=1` is a process-count guard, **not** a guaranteed
512 MiB total memory bound. Android/Linux host, account data, native libraries,
OS page cache, and Node/Rust/Go process RSS vary. Benchmark on the actual
target and avoid running heavy web browsers on the same 512 MiB host.

This runtime uses Android/Termux-centric bootstrap scripts. A normal Linux VPS
will additionally need a Linux build and startup/service manager; do not
attempt to execute Android ELF files on standard Linux.

## DeepSeek: replacement requirements (DO NOT swap before validation)

Current [zengtao227/Deepseek2API](https://github.com/zengtao227/Deepseek2API)
upstream is no longer maintained. High-priority experimental candidate:
[NIyueeE/ds-free-api](https://github.com/NIyueeE/ds-free-api), actively
maintained Rust with Chat Completions, Responses, Anthropic, tool-call parser,
multi-account management and mobile admin UI.

Before replacing: review GPL-3.0, build Android NDK successfully, measure RSS,
check login/account migration without moving raw secrets through UI, confirm
OpenAI/Anthropic SSE and multi-turn tool calls, test real upstream error
handling and credentials protection, and provide reversible rollback.
It publishes explicit warnings about upstream login/account risk checks.
Do not advertise a working swap without successful real-account authorized
inference and agreement with upstream limits.

## Kimi: replacement requirements (DO NOT ship untested Python)

Current Kimi integration is a non-bundled Python installer for the older
[chopper1026/kimi2api](https://github.com/chopper1026/kimi2api). No sufficiently
recent, lightweight, audited Go/Rust replacement has yet been found.
Prefer an actively maintained, authenticated *official* API where available,
or create a narrow per-account adapter after documenting the provider's
supported login and API capabilities, without bypassing authorization.
A new native adapter must first pass the same Android, streaming, tool, token
persistence, and RSS gates described above.

## Compatibility notes

- Agent2API's existing Rust adapter remains unchanged; don't fork/rewrite it.
- No simultaneous provider fan-out under the default 1-process cap; users who
  need multi-provider failover can raise the cap **after measuring memory**.
- No promised support for background auto-login or inferred remaining quotas.
- Existing active streaming requests are never intentionally stopped by the
  idle reaper.
- Wake and toggle operations require loopback/origin/ephemeral token checks.
- This document describes a development branch. Do not update an existing
  512 MiB deployment until CI and native-target tests pass.
