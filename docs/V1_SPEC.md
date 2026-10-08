# V1 Scope

## Goal

Expose one low-resource local API while supervising mature provider gateways instead of reimplementing their protocols.

## Default providers

- OpenCode Free via opencode2api
- FreeBuff via Freebuff2API

## Optional provider pack

- Agent2API headless server for additional supported personal-account channels such as CodeArts, Qoder, Cline, Trae and Loomy

Agent2API is disabled in the default configuration and is enabled by its installer script.

## Required API

- GET /healthz
- GET /api/providers
- GET /api/runtime
- GET /v1/models
- POST /v1/chat/completions
- POST /v1/responses
- POST /v1/messages

## UI

- Hub overview: http://127.0.0.1:8317/ui
- OpenCode upstream UI: http://127.0.0.1:8404/
- FreeBuff upstream UI: http://127.0.0.1:8402/ui
- Agent2API upstream UI: http://127.0.0.1:8403/ when installed

## Model IDs

- opencode/<model>
- freebuff/<model>
- agent2api/<model>

No implicit provider switching in V1.

## Acceptance

1. hubd starts in Termux.
2. OpenCode and FreeBuff become HEALTHY.
3. /v1/models merges both catalogs.
4. Hub UI renders provider cards and management links.
5. Chat and streaming succeed through each enabled provider.
6. A real tool-call loop succeeds through each provider used for agent workloads.
7. Killing one sidecar is detected and triggers bounded restart.
8. Failure of one provider does not stop another.
9. Local secrets are generated outside the repository.
10. Optional Agent2API can be installed without changing hubd code.