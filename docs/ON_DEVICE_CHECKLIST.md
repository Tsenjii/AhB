# First Android / Termux validation

This checklist is for the first real Android run.

## 1. Clone

When the repository is public:

```sh
pkg update
pkg install -y git
git clone https://github.com/Tsenjii/assistant-new.git
cd assistant-new
chmod +x scripts/*.sh
```

## 2. Bootstrap default V1

```sh
./scripts/bootstrap-termux.sh
```

This installs/builds:

- hubd
- opencode2api
- Freebuff2API

It also creates local runtime configuration and random local secrets under `data/`.

If FreeBuff compilation is too heavy:

```sh
AIHUB_CARGO_JOBS=1 ./scripts/bootstrap-termux.sh
```

To build the optional provider pack in the same pass:

```sh
AIHUB_WITH_AGENT2API=1 ./scripts/bootstrap-termux.sh
```

Or install it later:

```sh
./scripts/install-agent2api-termux.sh
```

## 3. Start

First run in the foreground:

```sh
./scripts/run-termux.sh
```

In a second Termux session:

```sh
cd assistant-new
./scripts/doctor-termux.sh
./scripts/smoke.sh
```

Expected default providers:

- OpenCode: HEALTHY
- FreeBuff: HEALTHY
- Agent2API: DISABLED unless installed

## 4. Open the Hub UI

Phone browser:

```text
http://127.0.0.1:8317/ui
```

The page should show:

- live total RSS and Hub RSS
- one card per provider
- status / model count / RSS / restart count
- unified model table
- button to open each provider's original UI

## 5. OpenCode UI

```text
http://127.0.0.1:8404/
```

Username:

```text
admin
```

Generated password:

```sh
cat data/opencode/webui-password.txt
```

Use the upstream UI for OpenCode proxies, keys, model availability, quotas, diagnostics and Playground.

## 6. FreeBuff UI

```text
http://127.0.0.1:8402/ui
```

Use FreeBuff's own account import and management flow.

## 7. Optional Agent2API UI

After installing the optional pack:

```text
http://127.0.0.1:8403/
```

Use its existing UI to add supported personal accounts such as CodeArts, Qoder, Cline, Trae or Loomy.

The local integration is loopback-only. Do not expose port 8403 while `AGENT2API_ALLOW_NO_KEY=1`.

## 8. Test inference

List current models:

```sh
curl -s http://127.0.0.1:8317/v1/models
```

Test one model:

```sh
AIHUB_TEST_MODEL='opencode/<actual-model-id>' ./scripts/test-chat.sh
```

Repeat with `freebuff/...` and `agent2api/...` when those providers are configured.

## 9. Verify restart supervision

```sh
curl -s http://127.0.0.1:8317/api/providers
```

Kill one sidecar PID. Within several seconds it should return to HEALTHY with a new PID and a higher restart count.

## 10. Background mode

```sh
./scripts/start-termux.sh
```

Stop:

```sh
./scripts/stop-termux.sh
```

## Failure report

Capture only diagnostics, never credentials:

```sh
./scripts/doctor-termux.sh
tail -n 120 logs/hubd.log
tail -n 120 logs/opencode.log
tail -n 120 logs/freebuff.log
tail -n 120 logs/agent2api.log 2>/dev/null || true
```