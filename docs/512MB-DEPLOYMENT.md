# AhB 512 MiB deployment and resource safeguards

## What is actually bundled

Android ARM64 base package contains **nine** real provider processes: OpenCode
(Go), FreeBuff (Node 22), Agent2API (Rust), DeepSeek (Go), Grok (Go), Kiro
(Go), Copilot (Go), Gemini Web (Go), and experimental Duck.ai (Rust HTTP-only). The AhB Hub is a separate Go process.

LMArena is an **external** third-party bridge and is not installed by the
bundle. Kimi Web is **not bundled**; the legacy Python installer remains only
for existing users until a reliable replacement is validated. Windsurf, Qwen,
Claude and any other imported local gateways must be installed and
run independently; no such gateway is silently bundled.

The default `config.example.json` no longer pretends uninstalled bridges are
running. Existing user-selected providers and already-installed Kimi data are
preserved by migration. No credentials, database or user-specific custom routes
are removed in the course of cleanup.

## UI resource controls and RAM measurement

The dashboard **Advanced / Resources** tab now provides an authenticated
`max_running_sidecars` selector (**1..16**) and `idle_stop_seconds`
(**30..86400 seconds**). Saving them writes `config.json` atomically,
validates the full configuration, preserves unknown fields and credentials,
and restarts only AhB. If the restart cannot be scheduled, the original file
is restored. In-flight requests can be interrupted by this requested restart.

The RAM panel differentiates:
- **Host**: `/proc/meminfo` MemTotal and MemAvailable, including reclaimable
  cached pages rather than counting MemFree as the only available RAM.
- **cgroup v2**: `memory.max` and `memory.current` when the container limit
  is tighter than host RAM; this includes unrelated processes and cache.
- **AhB RSS**: Hub process RSS plus the known child gateway process RSS,
  which may double-count shared memory pages.

For a 512 MiB VPS start at **one** active gateway and **120 seconds** idle.
For Android Termux default to **three** and **900 seconds** idle. These are
preferences adjustable from the local authenticated UI; neither is a hard
operating-system memory cap. RAM telemetry refreshes every ten seconds
without fetching every provider's models.

## Low RAM mode

On a **fresh** installation, seven bundled gateways default to `enabled=true`
and `start_mode=on_demand`; the experimental DeepSeek Web and Duck.ai sources
are installed but **disabled** until explicitly opted in. Enabled means
*permitted to start*, not seven simultaneous running background services.

- `resources.max_running_sidecars=1` caps concurrently resident managed
  on-demand processes. A second request to a different provider while a
  stream is active receives HTTP 503, rather than killing that stream or
  silently starting another large runtime.
- A direct `provider/model` inference request starts its sidecar if sleeping.
  The gateway holds a lease until the complete HTTP body/SSE is sent.
- Inactive on-demand services shut down after
  `resources.idle_stop_seconds=120` (when zero leases remain). Switching
  platforms first evicts any unused resident sidecar.
- Model discovery now has a **Hub-wide** concurrency limit: one live
  upstream probe on the one-resident-sidecar (512 MiB) profile, or four on
  larger configurations. Multiple browser tabs or API clients share the
  same limit. Waiting probes acquire their sidecar lease only after obtaining
  a slot, so they do not unnecessarily keep a gateway resident.
- Each managed gateway's private diagnostic log is capped at **4 MiB**, so
  long-running output cannot grow its on-disk log without bound. Logs wrap
  in place; old log contents are dropped when the cap is reached.
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

Android/Termux and native Linux AMD64/ARM64 have independent verified release
bundles and startup scripts. Always use the correct platform package; Android
ELF files cannot run on regular Linux. Check the newest GitHub Release and
matching Android prebuilt commit before deploying.

## Experimental DeepSeek and Duck.ai behavior

The previously bundled DeepSeek adapter was replaced by a lightweight
experimental 0xgetz/deepseek2api-derived **Go** Web adapter. The earlier
`/admin` UI does not exist in this version; only properly authorized private
Web account credentials belong in `data/deepseek2api/accounts.txt` (0600).
Its installation/build status is not proof that upstream chat or quotas work.

Duck.ai uses an experimental Rust HTTP-only adapter and remains disabled by
default. A successful local health probe does **not** establish inference
availability; the observed upstream HTTP 418 requires real service validation.
Do not count either experimental provider toward tested production capacity.

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
- Newly proposed source changes must pass CI, native Android ARM64 and Linux
  AMD64/ARM64 builds before upgrading an existing 512 MiB deployment.
