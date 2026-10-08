# Deploy on a 512 MB VPS

A 512 MB VPS is viable for V1 if you deploy prebuilt binaries. Do not compile FreeBuff on the VPS.

## Recommended runtime

Use a minimal Debian/Ubuntu image.

Processes:

- hubd
- opencode2api
- freebuff2api

Keep all three bound to loopback. Access the hub through an SSH tunnel first. This avoids spending RAM on a reverse proxy while V1 is still being validated.

## Estimated memory budget

These are planning estimates until measured on the target VPS:

- hubd: **~7–10 MB RSS measured** on a local Linux/amd64 idle build (the current code reports its own live RSS)
- opencode2api: roughly 20–70 MB RSS
- freebuff2api: roughly 60–180 MB RSS
- minimal OS + sshd: roughly 70–140 MB

Expected idle total: roughly **150–290 MB**. OpenCode and FreeBuff values remain planning estimates until the first real deployment; the dashboard will replace them with live RSS immediately.

Under active streaming, model-catalog refreshes, SQLite activity, and larger buffers, plan for roughly 250–420 MB total system use.

512 MB should work, but it is tight enough that a small swap file is recommended.

The hub dashboard and /api/providers report each sidecar's live RSS from /proc so the estimate can be replaced with real measurements immediately after deployment.

## Install a prebuilt bundle

Unpack the release bundle into:

```text
/opt/android-ai-hub
```

Create an unprivileged service account:

```sh
sudo useradd --system --home /opt/android-ai-hub --shell /usr/sbin/nologin aihub || true
sudo chown -R aihub:aihub /opt/android-ai-hub
```

Install the service:

```sh
sudo cp /opt/android-ai-hub/deploy/android-ai-hub.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now android-ai-hub
```

Check:

```sh
sudo systemctl status android-ai-hub
curl -s http://127.0.0.1:8317/healthz
```

## Remote access without exposing the API

From another machine:

```sh
ssh -N -L 8317:127.0.0.1:8317 USER@VPS
```

Then use:

```text
http://127.0.0.1:8317/v1
```

The dashboard is available at:

```text
http://127.0.0.1:8317/ui
```

## Why not compile on 512 MB

The hub and OpenCode gateway are small, but a Rust release build of FreeBuff can consume much more than 512 MB during compilation. Build it on GitHub Actions, a PC, or another larger machine, then copy the resulting binary to the VPS.