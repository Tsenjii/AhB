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
- Both share the same Go Hub and seven pinned upstream gateways.
- **Agent2API is not rewritten**. It is compiled unchanged for each platform.
- Both expose a unified local API at `127.0.0.1:8317/v1`, with a local
  dashboard at `127.0.0.1:8317/ui`.

A one-process on-demand cap prevents loading all seven gateways simultaneously,
but does not guarantee that a given upstream process will always fit in
512 MiB RAM. Avoid local browser automation or extra heavyweight apps.

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
The archive includes Linux-specific `bin/hubd`, OpenCode, FreeBuff,
Agent2API, DeepSeek, Grok, Kiro, and Copilot components.

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
existing account files. The separate Linux release update path is still under
development: do not replace the running installation with a fresh-install
command. Back up accounts and test the update process before upgrading.

## Actual RAM measurement

Use these commands after deployment; do not infer a device's RAM from the
language or binary size:

```bash
free -m
ps -eo pid,rss,comm --sort=-rss | head -n 20
curl -fsS http://127.0.0.1:8317/api/runtime
```

The Linux profile holds a full streaming-response lease and does not
terminate active model requests to make room for a different provider.
Trying to run two different providers simultaneously with a one-process
limit may return `503`; this is an intentional bounded-resource response,
not proof that the upstream model is broken.
