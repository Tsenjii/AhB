# Grok2API stability: upstream, community and AhB Android audit
**Research date: 2026-10-08; device state last reported by user 2026-10-08.** This is a differentiated stability assessment, not a claim that new Grok accounts were tested.

## Executive conclusion

**Promising, actively fixed, and currently one of the better account-backed Grok gateways for AhB**, especially **Grok Build** and CLI clients. But "stable Grok2API" is NOT synonymous with "every Grok Web chat allowance can be used in every API/tool flow": Web, Build and Console use distinct upstream behavior, entitlement, error modes and rate limits. Quota must be treated as the service reports it. Some reports of 200-with-empty-response, 403, 429 and tools failing remain legitimate. Do not declare production-device VERIFIED from social comments alone.

## Primary-source version confirmation

- Upstream: https://github.com/chenyme/grok2api
- Latest official release identified 2026-10-08: **v3.1.6**, released 2026-09-16. Official changelog: https://github.com/chenyme/grok2api/releases/tag/v3.1.6
- AhB source build pins **`7c889a960e2638341b4dae9a5c81af0e0f38c87f`**, upstream dated **2026-09-30**, and the upstream `VERSION` at that commit is **v3.1.6**. The pinned SHA was the GitHub latest source HEAD at check time. This is not an outdated v2/FastAPI version.
- It is the native **Go backend plus React admin UI**, built for Android ARM64 in `.github/workflows/build-android-arm64.yml`; Grok2API sidecar is **disabled by default**, and real Android model inference has **NOT** yet been observed by user.
- Upstream `/healthz` only checks the web process (`{"ok":true}`); `/readyz` reports `ready`, startup state and component status. AhB `config.example.json` correctly points Grok to **`/readyz`**, but that is still not proof of a specific model/account having spendable quota.
- Changelog v3.1.6 covers failed/leased account isolation, cool-down corrections, empty-stream and quality guard behavior, correct billing/503 auditing. Earlier v3.1.5 and v3.1.4 covered post-tool agent turn robustness, Web media quotas, refreshed Build credentials and more.

## Verifiable community reports (do not cherry-pick)

| Source/date | Evidence | Appropriate interpretation |
|---|---|---|
| Developer **Chenyme**, Linux.do, **2026-07-20** | Claimed v3.0.6 works quite stably with Grok Build, Codex and Claude Code, but Anthropic Messages prompt caching still had issues. https://linux.do/t/topic/2619785/1 | Positive firsthand engineering signal for **Build/CLI**, not proof all web users/tool calls work |
| Linux.do user thread, **2026-07-14** | Grok2API recommended after updates resumed; comparisons of Sub2API Responses vs Completions and disproportionate quota use. https://linux.do/t/topic/2581801 | Other toolchains/bridges have protocol and usage problems; Grok2API often recommended |
| GitHub **#793**, **2026-07-27** | Grok Build tool web_search/web_fetch and agent/tool listing failure on v3.0.9. https://github.com/chenyme/grok2api/issues/793 | Specific tool implementations can fail independently of basic chat |
| GitHub **#975**, **2026-08-19** | Same authorized Build account worked in native Sub2API route but 429/cooldown through Grok2API 3.1.4; linked fix #999. https://github.com/chenyme/grok2api/issues/975 | Avoid treating transient transport failure as exhausted quota |
| GitHub **#893**, **2026-08-12** | User reported Grok Build working while Web/Console sync failed. https://github.com/chenyme/grok2api/issues/893 | Provider family/capability matters for model eligibility |
| Linux.do debug, **2026-08-19–20** | High performance reported but long reasoning/tool loop sometimes yielded “200·error”; later fork worked for one reported test. https://linux.do/t/topic/2775254?page=4 | Long agent workflows require repeated end-to-end tests; status 200 alone is not enough |
| Linux.do mixed early experience **2026-02 to June** | Several users reported API stable on their setups while others encountered 403/502 (especially old v2). https://linux.do/t/topic/1576878 and https://linux.do/t/topic/1939934 | Historical v2 breakages should not automatically discredit v3.1.6, but Web fragility is real |
| NodeLoc **2026-08-30** | Universal browser Web-to-API project advertised GPT/Grok/DeepSeek use, with another commenter calling it potentially unstable. https://www.nodeloc.com/t/topic/105857 | Generic browser automation is NOT interchangeable with mature Grok2API Go; no basis to replace it |
| GitHub releases, **2026-08–09** | Extensive continuous fixes through v3.1.6. https://github.com/chenyme/grok2api/releases | Active fixes are a stronger signal than an isolated success story, not a substitute for live device tests |

**Historical context that can mislead:** author announced an end to development on Linux.do **2026-06-09** (https://linux.do/t/topic/2352950), but development resumed in July and v3.1.6 shipped in September. Therefore do not classify the current Go v3.1.6 as dead merely from the earlier notice.

## What AhB now implements to reduce uncertainty

- **Native installed and pinned**: optional Grok2API binary+UI, loopback port **8407**, SQLite state, login through upstream native admin, client key provisioned by `scripts/enable-grok2api.sh`.
- **Grok key safety improvement (2026-10-08)**: enable script now preserves the existing saved client key and configuration when a validation attempt fails for a transient 5xx/429/network response, rather than creating a replacement; only explicit 401/403 triggers local admin-based key recovery. It backs up `config.json` before enabling and saves private state with restrictive permissions. No bypass is implemented.
- **Grok phone acceptance script**: `scripts/test-grok2api-termux.sh` checks actual `/healthz`, `/readyz`, component summary, authenticated `/v1/models` and then, **only with explicit `AIHUB_TEST_MODEL='grok/model'`**, sends real Chat, SSE (checks final `[DONE]`), structured OpenAI function-call and a second tool-result turn. It does **not** execute arbitrary tool-generated shell commands.
- **Mock fixture** `scripts/test-grok2api-fixture.sh` exercises both success and no-tool/empty-model failures in CI. Mock test pass is not genuine Grok authentication or phone runtime evidence.
- Existing optional UI opens at **http://127.0.0.1:8407/**. AhB integrated endpoint at **http://127.0.0.1:8317/v1**, with model prefix `grok/`. Authenticated account is provisioned by the upstream original UI using an account the user owns.
- No new account registrations, CAPTCHA solving, provider risk-controls circumvention, quota bypass or credit duplication.

## Suggested exact, low-risk phone verification

After confirmed publication of an Android bundle containing `scripts/test-grok2api-termux.sh` and with the user's backup preserved:

```sh
cd ~/AhB
./scripts/enable-grok2api.sh
./scripts/stop-termux.sh
./scripts/start-termux.sh

# Admin UI: http://127.0.0.1:8407/
# Add your own eligible Grok account using upstream's permitted login flow.
./scripts/test-grok2api-termux.sh
curl -fsS http://127.0.0.1:8317/v1/models | jq -r '.data[].id | select(startswith("grok/"))'

# Replace MODEL_ID with the actual listed ID, if you have quota.
AIHUB_TEST_MODEL='grok/MODEL_ID' ./scripts/test-grok2api-termux.sh
```

Store only a summary of `HTTP status / model / type of failure / stream events / tools pass` in acceptance. Never upload admin access tokens, cookies, exported account credentials, full request transcripts or private keys. Before marking VERIFIED, repeat low-rate tests over user-approved time windows and observe independently Web/Build/Console entitlements. Account web quotas may update separately and should never be inferred from 200 /healthz.

## Current judgement

**Grok2API v3.1.6 = P0, prioritized** for completion of actual AhB inference/tool acceptance. **Grok Build** has better documented CLI/tool compatibility than generic Grok Web adapters. **Grok Web** may access a different web subscription allowance but its entitlement, capacity and end-to-end tool support must be demonstrated independently. Keep Kimi Web experimental; keep working OpenCode as stable fallback. Do not replace Grok2API with another random browser Web proxy without comparative test evidence.
