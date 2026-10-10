# AhB Linux VPS: 512 MiB RAM (AMD64 or ARM64)

This guide is for a **native Linux build**, not the Android Termux package.
The Linux GitHub Actions workflow creates both CPU architectures using
Ubuntu 24.04 native runners. Their ABI depends on Linux/glibc compatibility;
Ubuntu 24.04 is the primary initial test target, and Debian/older distros
must pass a real smoke test before support is promised.

## Why two distributions?

- **Android** defaults to up to 3 resident sidecars, idle timeout 15 minutes.
- **Linux 512 MiB** defaults to a single resident on-demand sidecar, idle
  timeout 2 minutes.
- Both share the same Go Hub and nine pinned upstream gateways.
- **Agent2API is not rewritten**. It is compiled unchanged for each platform.
- Both expose a unified local API at `127.0.0.1:8317/v1`, with a local
  dashboard at `127.0.0.1:8317/ui`.

A one-process on-demand cap prevents loading all nine gateways simultaneously,
but does not guarantee that a given upstream process will always fit in
512 MiB RAM. Avoid local browser automation or extra heavyweight apps.

## Stability-first mode (when RAM can exceed 512 MiB)

The strict 512 MiB profile remains available and is **not changed during
upgrade**. If your host has more memory, prefer the control center's
**Settings → Resources → 穩定優先** preset: **2 resident gateways / 600 seconds
idle**. For a larger measured host, **多來源** uses **3 / 900 seconds**.
Both presets only fill the form; the user explicitly saves and confirms the
AhB restart. Existing account files and custom settings are preserved.

Keeping two independently requested gateways warm reduces cold-start churn
and avoids the deliberate HTTP 503 capacity conflict when another Gateway has
an active SSE stream. The tradeoff is greater actual RSS, which depends on the
upstream provider and is not bounded by the numeric process count.
Use `/api/runtime` to inspect real RAM before increasing the limit.

A health probe losing its connection does not forcibly restart an on-demand
Gateway while an inference/model-discovery lease is active. After the request
finishes, repeated failed health probes can restart that process. An upstream
that accepts a connection but never sends response headers is cut off after
120 seconds; after headers arrive, streaming inference may continue longer.


## System dependencies

On a suitable Ubuntu server, install the small *runtime* dependencies
(`jq`, `curl`, `tar`, `python3`, `coreutils`, and Node.js 22 for
FreeBuff). Native Go and Rust gateways are already compiled by GitHub
Actions and **do not need Go or Rust installed on the VPS**.

Use an unprivileged user and keep the `~/AhB/data` directory readable only
by that user. Do not open port 8317 publicly.

## First install, after the GitHub Linux release succeeds

```bash
# The Linux release must exist first. Do not run this before CI is green.
curl -fsSL https://raw.githubusercontent.com/Tsenjii/AhB/main/scripts/install-prebuilt-linux.sh -o "$HOME/ahb-linux-install.sh"
bash "$HOME/ahb-linux-install.sh"
cd "$HOME/AhB"
bash scripts/start-linux.sh
curl -fsS http://127.0.0.1:8317/healthz
```

The installer verifies SHA-256, rejects unsafe archive entries, checks the
Linux config profile and **refuses to overwrite an existing installation**.
The archive includes Linux-native `bin/hubd` plus nine bundled provider adapters:
OpenCode, FreeBuff, Agent2API, DeepSeek, Grok, Kiro, Copilot, Gemini Web
and Duck.ai. Each stays off-process until requested.

If the release does not exist yet, the download will fail and no installation
takes place. That is expected until the dual-edition pull request is merged
and both native CI builds pass.

## Control and diagnosis

```bash
cd "$HOME/AhB"
bash scripts/start-linux.sh
bash scripts/stop-linux.sh
tail -n 40 logs/hubd.log
```

To reach the loopback-only UI remotely, open an SSH tunnel on your computer:

```bash
ssh -N -L 8317:127.0.0.1:8317 USER@VPS
```

Then open `http://127.0.0.1:8317/ui` in your **local computer** browser.
The GUI's on/off toggles trigger a strictly scoped Linux restart, preserving
existing account files. For a **staged Linux upgrade**, run:

```bash
cd "$HOME/AhB"
bash scripts/upgrade-linux.sh
# A timestamped AhB.backup-* directory retains the previous release.
bash scripts/start-linux.sh
```

The Linux upgrader only selects Linux-specific GitHub releases (or a
deliberately pinned `AHB_LINUX_RELEASE_TAG`), checks SHA-256 and archive
safety **before** stopping the old Hub, preserves credentials/config and
newly installed static assets separately, and retains a complete rollback
backup. It refuses to copy account databases if owned processes or listening
ports are still running. The old installation must have an intact Linux
stop helper and installer; older unknown layouts fail closed.

This is tested with synthetic credentials/archives in CI; the first real
512 MiB VPS installation and an account-authorized live upgrade still need
verification. Do not automatically remove the backup.

## Actual RAM measurement

Use these commands after deployment; do not infer a device's RAM from the
language or binary size:

```bash
free -m
ps -eo pid,rss,comm --sort=-rss | head -n 20
curl -fsS http://127.0.0.1:8317/api/runtime
```

Gemini Web and Duck.ai are unofficial gateways. Availability, upstream
limits, model names, and actual permissions must be checked with an
account-authorized request; packaging does not guarantee inference.

The Linux profile holds a full streaming-response lease and does not
terminate active model requests to make room for a different provider.
Trying to run two different providers simultaneously with a one-process
limit may return `503`; this is an intentional bounded-resource response,
not proof that the upstream model is broken.
