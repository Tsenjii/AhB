# Deploy on Termux

V1 is Termux-first. The runtime is three native processes:

- hubd
- opencode2api
- freebuff2api

Everything binds to loopback by default.

## Build

From the repository root:

```sh
chmod +x scripts/*.sh
./scripts/bootstrap-termux.sh
```

The bootstrap script installs the native build toolchain, builds hubd, builds both upstream sidecars from source, and creates local config files without overwriting existing ones.

## Start

Foreground test run:

```sh
./scripts/run-termux.sh
```

Background run with a Termux wake lock when available:

```sh
./scripts/start-termux.sh
```

Stop:

```sh
./scripts/stop-termux.sh
```

Diagnostics:

```sh
./scripts/doctor-termux.sh
./scripts/smoke.sh
```

The unified endpoint is:

```text
http://127.0.0.1:8317
```

## OpenCode Free

The default OpenCode sidecar config enables anonymous Zen access and disables its Web UI to save resources. It listens only on 127.0.0.1:8401.

Models appear as:

```text
opencode/<upstream-model-id>
```

## FreeBuff

FreeBuff starts on 127.0.0.1:8402. The supplied config allows the process to start before a credential is added.

Add a credential using FreeBuff's existing local management API or UI. The hub does not copy, log, or own the upstream credential.

Models appear as:

```text
freebuff/<upstream-model-id>
```

## Client setup

Use:

```text
Base URL: http://127.0.0.1:8317/v1
```

V1 requires provider-prefixed model IDs.

## Android battery behavior

Android may kill Termux in the background. During initial validation, keep Termux alive while testing. Native Android foreground-service integration comes after the provider runtime is proven stable.

## Security

Keep allow_lan=false. V1 rejects non-loopback hub listening by default.