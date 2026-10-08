# AhB — live To-API candidate register and staged installation
Last reviewed: 2026-10-08

This is the **human-readable shortlist** for the machine-checked source of truth at [TO_API_REGISTRY.json](TO_API_REGISTRY.json). Every provider found later must be added there with: an exact GitHub URL; current availability; direct login and account-pool support; transport surface (OpenAI/Anthropic, SSE, tools); Android ARM64 requirements; quota truth; redistribution license; known overlaps; testing status; and an evidence-backed next action. The catalog's statuses do not imply real inference availability.

## What is installed vs simply visible

| Source | Present in a new ARM64 bundle? | Available automatically? | Real Android tests |
|---|---|---|---|
| OpenCode | Yes | Yes | Chat + SSE proven |
| FreeBuff | Yes | Yes | No account at last test |
| Agent2API | Yes | Generally | HTTP 503 at last test due exhausted 2 accounts |
| DeepSeek Web Go | Yes | No: opt-in | No live account inference |
| Grok2API Go | Yes | No: opt-in | Not proven |
| Kiro-Go | Yes | No: opt-in | Not proven |
| **GitHub Copilot Go** | **New pinned source in build workflow** | **No: requires self-initiated GitHub login then enable** | Await green ARM64 publication and device test |
| **Kimi Web** | **Only lightweight installer scripts bundled, not Python environment** | **No: requires separate explicit Python/Node install, then enable** | Installer and inference are unverified on Termux |
| LMArena, Windsurf, Qwen, independent Kimi, Gemini, Claude | Only connection presets | No: must start real bridge separately | Not proven |

## Installable shortlist (only the first two have NEW implementation in AhB)

**1. GitHub Copilot — Go, smallest new native option**  
Source: https://github.com/whtsky/copilot2api, pinned `a4aac95d4a8f430f02121f79ea36aeaaa06daea1`, MIT. Native binary will be included *only after successful Android ARM64 workflow and public prebuilt pointer update*. One-account upstream adapter, OpenAI/Anthropic/Responses/embeddings, quota at upstream `/usage`. Does not enforce its own API key; AhB keeps it loopback-only. Credentials are stored locally at `data/copilot2api/credentials.json` with restrictive mode. GitHub OAuth Device Flow must be explicitly completed with an eligible account; a GitHub profile by itself does not grant Copilot capacity.

  ```sh
  cd ~/AhB
  ./scripts/login-copilot2api.sh
  # Follow the official GitHub device-flow screen; press Ctrl+C after success
  ./scripts/enable-copilot2api.sh
  ./scripts/stop-termux.sh
  ./scripts/start-termux.sh
  ```
  Then inspect the real `copilot/` model IDs before making a test request. No account pool in this minimal option: other alternatives remain in registry if multi-account UI proves essential.

**2. Kimi Web — optional Python + React account pool**  
Source: https://github.com/chopper1026/kimi2api, pinned `7f046d8627f275432f82788a6547bc905038738c`, MIT. Provides multi-account WebUI with account health, cooldown, concurrency and logs, unlike `kimi/` which only points at a separately running bridge. The optional Termux installer downloads audited source into `data/kimiweb/source`, creates its own `data/kimiweb/venv`, builds the front-end, writes local strong keys, and retains state under `data/kimiweb`. It is not installed by default and its Python dependencies (especially compiled Pydantic on Termux) have NOT passed real phone tests.

  ```sh
  cd ~/AhB
  ./scripts/install-kimiweb-termux.sh
  ./scripts/enable-kimiweb-termux.sh
  ./scripts/stop-termux.sh
  ./scripts/start-termux.sh
  ```
  Then use `http://127.0.0.1:8412/admin` with the password in `data/kimiweb/admin-password.txt`, import only your own eligible accounts through the native UI, verify `kimiweb/` models and a real chat/streaming response. Client key is in `data/kimiweb/client-key.txt`. The original `kimi/` external connection preset is independent.

  **Privacy warning:** Kimi upstream records request headers and parts of request/response metadata; this optional install limits the recorded body length to zero and retention to 100, but do not use it with sensitive conversations without a review of the upstream logging behavior.

**3. DeepSeek Web — already bundled (do not duplicate)**  
Source: https://github.com/zengtao227/Deepseek2API. Go binary/admin UI included; `./scripts/enable-deepseek2api.sh` after installation, then restart and test authorized DeepSeek access. Secondary pure-Node fallback https://github.com/Sakura520222/deepseek2api only if the current Go route fails real tests.

**4. Qwen Web Go — promising but redistribution BLOCKED pending license audit**  
Source: https://github.com/XxxXTeam/Qwen2API_Go. The upstream README shows Go account management, admin UI, streaming and multiple keys; however an explicit LICENSE file was not found in the repository when checked. Do not redistribute its compiled binary inside AhB without permission. An independently running legitimate instance may already use `connect-bridge.sh qwen ...`; Qwen web access and Android runtime remain unverified.

**5. Other optional research candidates**
- GLM Web (Python): https://github.com/XxxXTeam/glm2api
- MiMo Web (Python): https://github.com/Fly143/MiMo2API
- Kimi Code (Go OAuth, separate from Web): https://github.com/xaionaro-go/kimi-oauth-proxy
- Copilot multi-account alternative: https://github.com/StarryKira/copilot2api-go
- Codex Go small adapter: https://github.com/dvcrn/codex-oauth-proxy
- CLIProxyAPI large optional account proxy: https://github.com/router-for-me/CLIProxyAPI (hold: overlaps AhB and ModelAtlas control plane)

Do not install all similar solutions at once. Select only one per distinct account source to avoid duplicate memory use, ports, credentials and inconsistent quotas. Avoid archived/broken Qwen Code proxy https://github.com/aptdnfapt/qwen-code-oai-proxy.

## Current AhB optimization backlog

**P0 — Correctness & safety**
1. Separate three source dimensions in state: process alive, accounts authenticated, **request-capable models/remaining quota**. Agent2API 2/2 credentials produced HTTP 503 on real Qwen test. Do not label listable models as usable; expose `quota=unknown` where no upstream quota endpoint exists.
2. Explicit no-secret logging: mask Authorization/cookies/query credentials, cap sensitive provider logs, do not display full account IDs. Restrict admin ports to loopback.
3. Pin and audit external dependencies; never auto-upgrade upstream unreviewed source or overwrite installed account state.
4. Preserve non-destructive backups, interrupted installs and error rollback. Keep all optional sidecars **off by default**.

**P1 — Small-device UX and performance**
1. Distinguish `BUNDLED / OPTIONAL DOWNLOAD / CONNECTOR ONLY / ACTIVE / REAL INFERENCE PASSED` on phone UI.
2. Add per-source **one-shot smoke** and clear 401/429/503 messages (authentication, rate limit, exhausted quota), with credentials never submitted through browser JS.
3. Expose resource budget: device RSS for Hub and subprocesses; lazy-run selected optional source, memory cap and stop-idle controls. Count only published models rather than treating all as inference-ready.
4. Show account quota refresh schedules and cooldown **only if upstream explicitly reports them**.
5. UI-generated Termux command is still a copy action, not a true one-click install.

**P2 — Testing and supported adapters**
1. Codify generic adapter capability matrix for Chat/Responses/Anthropic/stream/tools/images so unsupported endpoints aren't shown as a promise.
2. Test SSE client cancellation, full tool-result continuation, IPv4/IPv6 loopback, database consistency on upgrade, midstream failure, pool exhaustion and simultaneous requests on ARM64.
3. Add original-project update watcher against **pinned** versions, changelogs and licenses. Require user review before upgrading an upstream component; do not silently track latest.
4. Evaluate a single multi-account Copilot adapter if the minimal one no longer fits; keep CLIProxyAPI/desktop browser adapters external.

## Onboarding a future candidate

Add a candidate to `docs/TO_API_REGISTRY.json` with a **unique** lowercase ID, the actual GitHub repo URL, one defined `status`, a P0/P1/P2/P3/HOLD/REJECT priority, summary `notes`, `last_reviewed`, and `verified_live_android:false` until the user actually tests a real request. Do not delete a previously known candidate merely because it is not installable now; change its status. Submit a PR, run `bash scripts/test-provider-registry.sh`, and record the acceptance in `docs/DEVICE_ACCEPTANCE_2026-10-08.md` or its successor.

Existing baseline device backup: `~/AhB.backup-20261008-194847`; retain until all desired real-account tests pass.
