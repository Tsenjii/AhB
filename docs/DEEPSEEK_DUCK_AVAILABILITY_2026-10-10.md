# DeepSeek Web and Duck.ai: availability / login

Updated 2026-10-10. Packaging and local health checks are not proof of live inference or usable account entitlement.

## DeepSeek Web

The bundled [zengtao227/Deepseek2API](https://github.com/zengtao227/Deepseek2API) connects to `chat.deepseek.com` Web service URLs, not the separately paid DeepSeek Platform API.

- The local original admin console is `http://127.0.0.1:8405/admin`; its AhB-generated management token is private at `~/AhB/data/deepseek2api/admin-key.txt`.
- That management token is *not* a DeepSeek Web account. A real authorized DeepSeek Web account/session must also be configured and a content-bearing chat tested in the original console. Empty account pool means it cannot infer.
- AhB does not have FreeBuff-style one-tap Web OAuth for DeepSeek. Original management options and the actual DeepSeek login are separate.
- DeepSeek is disabled by default in fresh-install templates until configured and tested. Existing installed `config.json` and accounts remain untouched by upgrade.

## Duck.ai / 418 I'm a teapot

The bundled [aurora-develop/Duck2api](https://github.com/aurora-develop/Duck2api) unofficial bridge calls Duck.ai private service endpoints. Upstream source `internal/duckgo/request.go` already retries temporary 418/429 responses; this does not guarantee recovery.

- HTTP 418 from a real chat means upstream refused the request. A green local `/ping` only proves the Gateway's process can answer its own health check.
- AhB now shows recent Duck.ai HTTP 418 as DEGRADED with an explicit warning even if the local health endpoint is green. An actual later successful request or an old error clears that recent-error classification; status code 200 alone still cannot guarantee completed streaming output.
- 418 is not evidence by itself that a particular IP, device or account is permanently banned. It can have several upstream causes which need provider-side evidence to distinguish.
- Duck.ai is disabled by default on new installs pending current real inference acceptance. Existing user settings are preserved.
- Avoid repeatedly restarting or flooding a rejected service; allow the upstream issue to resolve or use a supported official integration.

## Validation hierarchy

1. Local process runs, health endpoint works.
2. Provider account or remote access is usable.
3. A real request returns meaningful content, complete streaming and structured tool calls where supported.

No real user account/login or source quota was verified by CI. This change fixes misleading status/defaults; it does not claim to repair Duck.ai's upstream HTTP 418 or add DeepSeek account authorization.
