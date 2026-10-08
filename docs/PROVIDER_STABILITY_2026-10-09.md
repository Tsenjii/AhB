# AhB seven bundled providers — stability and release matrix (2026-10-09)

Source evidence: [provider config](../config.example.json), [Android build workflow](../.github/workflows/build-android-arm64.yml), [on-device acceptance](NEXT_PHONE_ACCEPTANCE.md). This is **source-level and CI coverage**, not proof of every user's live quotas.

## Bundled providers and one optional installer

| ID | Upstream | Packaging / startup default | Real Android user account evidence before this pass | Main blockers |
|---|---|---|---|---|
| `opencode/` | [opencode2api v1.3.7](https://github.com/jasonxu114514/opencode2api) | Go ARM64, ON | Actual chat, SSE + [DONE], first structured tool emission on previous phone version | Confirm regression-free new bundle, full tool-result continuation |
| `freebuff/` | [Freebuff2API v0.10.3](https://github.com/lza6/Freebuff-2API) | Rust ARM64, ON | Account pool empty on last phone run | Use original FreeBuff account import UI; no native non-Windows WebView login; verify model quota |
| `agent2api/` | [Agent2API v2.9.7](https://github.com/aimod-cc/agent2api/releases/tag/v2.9.7) | Rust ARM64, first-install helper enables, previous accounts preserved | Two credentialed accounts, real model returned quota-related HTTP 503 on previous build | Credentials != quota. Confirm v2.9.7 initial launch and any usable account entitlement |
| `deepseek/` | [Deepseek2API](https://github.com/zengtao227/Deepseek2API) | Go ARM64, OFF | No live authenticated inference | Check account, real chat, SSE, native tool continuation individually |
| `grok/` | [Grok2API v3.1.6](https://github.com/chenyme/grok2api) | Go ARM64, OFF | No live authenticated inference | Build/Web/Console may differ; 429 + tool-loop acceptance after local account setup |
| `kiro/` | [Kiro-Go](https://github.com/Quorinex/Kiro-Go) | Go ARM64, OFF | No live authenticated inference | Ensure proper account refresh, authenticated models/chat/SSE/tools |
| `copilot/` | [copilot2api](https://github.com/whtsky/copilot2api) | Go ARM64, OFF | No device OAuth/chat proof | One-account OAuth and actual model entitlement; local-only service |
| `kimiweb/` | [kimi2api](https://github.com/chopper1026/kimi2api) | Optional Python installer **only**, OFF | Not installed/tested on phone | Large optional runtime; keep experimental |

The following are **NOT** counted as bundled working providers: LMArena, Windsurf, Qwen Web, Kimi external, Gemini Web, Claude Web, Codex OAuth, CLIProxyAPI. They are connector presets and require independently running, legitimately usable local gateways. Agent2API's WorkBuddy CN/Intl, CodeArts, Qoder, Cline, Trae, Loomy, KukuAI etc are its internal subplatforms, **not** extra binaries in AhB.

## This source stabilization batch

1. **Account-state correctness**: FreeBuff account circuit breakers must be known `closed`, `half_open`, or `open` with an elapsed cooldown to be counted as candidates. The Hub no longer counts missing/unknown breaker states as healthy.
2. **Bounded model discovery**: querying multiple enabled providers is concurrent, limited to 4 and to 10 seconds per provider (instead of serially consuming up to 7 times the timeout). The Hub still excludes unready providers, returns warning fields on failure and sorts model IDs; no stale cache, no protocol translation claims.
3. **Full tool-loop test correctness**: `test-tool-roundtrip.sh` now validates all tool function IDs/arguments (up to 8 per round) and supplies matching results before the follow-up answer. Mock fixture tests both single/multiple and malformed tool outputs. It never executes commands proposed by the model.
4. **Minimal read-only phone inventory**: `scripts/provider-status-termux.sh` reads local `/api/providers` and `/v1/models`; prints each bundled source's enabled, process state, ready, account count, listed model count and last upstream HTTP. No authentication material or individual account records are read or printed. A metadata fixture tests safe output.
5. **Safety**: no account/token migration changes, no opportunistic enabling of optional adapters, no browser-cookie extraction, no remote/public endpoints. Agent2API, Grok, Kiro etc still own account refresh, quotas and platform-specific retry logic.

## Phone acceptance after exact new release publication

Before user upgrades: require source CI **success** plus Android ARM64 job **success**, AND `prebuilt/source-commit.txt` points to the matching new main commit. Device users retain their existing `~/AhB.backup-20261008-194847`, any newer known-good backups, config and `data/`.

After non-destructive upgrade via existing `upgrade-prebuilt-termux.sh`:

```sh
cd ~/AhB
./scripts/start-termux.sh
./scripts/doctor-termux.sh
./scripts/provider-status-termux.sh

# Identify actual advertised model ID, then test OpenCode first:
curl -fsS http://127.0.0.1:8317/v1/models | jq -r '.data[].id'
AIHUB_TEST_MODEL='opencode/ACTUAL_LISTED_MODEL' ./scripts/test-provider-acceptance.sh
```

The provider inventory and `doctor-termux.sh` do **not** verify actual user-account model inference; chat/SSE/native structured tools require opt-in live requests, and every model/source may vary. Only report non-sensitive provider status, codes and pass/fail. Never paste cookies, API keys, original config or account exports.

## Later priorities

- P0: phone regression-free upgrade; original OpenCode chat/SSE/two-turn tools; truthful Agent2API quota states; initial FreeBuff login.
- P1: real Grok/DeepSeek/Kiro/Copilot per-model Chat/SSE/tools; account refresh and cooldown evidence; resource memory monitoring over hours.
- P1: capacity-aware fallback after repeated device evidence, not by assuming identical model IDs imply compatible tools or image capability. Cross-source fallback stays default OFF.
- P2: alternative authenticated CLI adapters, optional Kimi Web and external bridges only after real need, low-memory feasibility and non-overlap are demonstrated.
