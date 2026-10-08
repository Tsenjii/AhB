# Android/Termux real-device verification checklist

This checklist supersedes the original source-build-first instructions. Use the pinned ARM64 prebuilt by default; compiling the optional Rust gateways on a phone is not a release-validation requirement.

## 1. First install or upgrade (choose the correct one)

**Fresh ARM64 Android / Termux install** (no existing `~/AhB`):

```sh
pkg update
pkg install -y curl
curl -fsSL https://raw.githubusercontent.com/Tsenjii/AhB/main/scripts/install-prebuilt-termux.sh | bash
```

**Existing prebuilt AhB installation** (retain login data, local secrets and account databases):

```sh
curl -fsSL https://raw.githubusercontent.com/Tsenjii/AhB/main/scripts/upgrade-prebuilt-termux.sh | bash
```

The upgrader verifies the archive checksum *before* touching the existing install. The previous tree is kept in a sibling timestamped `AhB.backup-YYYYMMDD-HHMMSS` directory. Keep it until smoke tests and real account inference pass. Do not delete `~/AhB/data` to troubleshoot a failed upgrade.

## 2. Start and inspect

Start AhB in the foreground in one Termux session:

```sh
cd ~/AhB
./scripts/run-termux.sh
```

From a **second** Termux session:

```sh
cd ~/AhB
./scripts/doctor-termux.sh
./scripts/smoke.sh
```

Open the Hub overview on the same phone at:

```text
http://127.0.0.1:8317/ui
```

Inspect provider state, readiness, account usability, model count and RSS. A provider with **no imported accounts** may correctly appear **DEGRADED**, rather than HEALTHY. Startup/HTTP health alone is not enough to establish inference readiness.

## 3. Provider original UIs

| Provider | Local UI | Optional setup |
|---|---|---|
| OpenCode Free | `http://127.0.0.1:8404/` | account / key setup; username `admin`, local password below |
| FreeBuff | `http://127.0.0.1:8402/ui` | import accounts via its own UI |
| Agent2API | `http://127.0.0.1:8403/` | normally enabled when included in the prebuilt |
| DeepSeek2API | `http://127.0.0.1:8405/admin` | `./scripts/enable-deepseek2api.sh` |
| Grok2API | `http://127.0.0.1:8407/` | `./scripts/enable-grok2api.sh` |
| Kiro-Go | `http://127.0.0.1:8408/admin` | `./scripts/enable-kiro-go.sh` |

OpenCode's generated local password:

```sh
cat ~/AhB/data/opencode/webui-password.txt
```

Do not paste passwords or account tokens into public issue reports. Recheck the Hub after adding accounts.

## 4. Actual inference (not only a successful build)

Get current usable model IDs:

```sh
curl -fsS http://127.0.0.1:8317/v1/models
```

Use an actual returned model ID, for example:

```sh
AIHUB_TEST_MODEL='opencode/<actual-model-id>' ./scripts/test-chat.sh
```

Repeat for configured `freebuff/`, `agent2api/`, and optional sources. Confirm:

1. Model appears in `/v1/models` with the correct provider prefix.
2. Actual chat output succeeds through the unified endpoint.
3. SSE streaming delivers multiple chunks and ends normally.
4. A full tool-call loop works: ask to read a file, provide tool output, then ask to edit/search/test and respond to the test result.
5. API failure paths (bad model, upstream unavailable, exhausted account) produce useful error responses.
6. One crashed sidecar is restarted without taking down healthy providers. Check restart count and RSS.

Cross-provider same-model balancing is disabled by default. Test only after confirming strict `provider/model` routing works.

## 5. Report diagnostics without secrets

```sh
cd ~/AhB
./scripts/doctor-termux.sh
curl -fsS http://127.0.0.1:8317/api/providers
tail -n 100 logs/hubd.log
```

Remove account identifiers, tokens and other secrets from logs before sharing. The following distinctions must be tracked separately: **Go tests**, **Android ARM64 binary built**, **Termux launch**, **real inference**, and **real tool calling**.
