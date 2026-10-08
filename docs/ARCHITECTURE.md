# Architecture

```text
Client
  |
  v
hubd 127.0.0.1:8317
  |
  +-- opencode2api 127.0.0.1:8401
  |      +-- management UI 127.0.0.1:8404
  |
  +-- Freebuff2API 127.0.0.1:8402
  |      +-- management UI /ui
  |
  +-- agent2api-server 127.0.0.1:8403  (optional)
         +-- management UI /
```

## hubd owns

- sidecar lifecycle
- health state
- bounded restart
- unified model catalog
- provider-prefixed routing
- streaming transport
- basic runtime telemetry
- the lightweight overview page

## hubd deliberately does not own

- provider OAuth/login implementation
- upstream credential refresh
- provider account pools
- quota algorithms
- provider-specific proxies
- provider Playground implementations
- broad account automation

Those stay inside the mature upstream gateway that already implements them.

## UI strategy

The overview page is only a control foyer. It shows status and routes the user into each upstream project's native WebUI.

This prevents duplicated provider settings and makes upstream feature updates immediately useful without rewriting our UI.

## Local secrets

The checked-in configuration contains placeholders only. `scripts/prepare-configs.sh` creates a random Hub-to-OpenCode key and OpenCode WebUI password under ignored `data/` files, then substitutes them into the local runtime config.

## Optional provider growth

Additional mature gateways should be added as sidecars with:

- loopback base URL
- health path
- model path
- management UI URL
- explicit provider prefix

Do not copy large upstream protocol implementations into hubd.