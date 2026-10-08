# Network egress

By default, every provider sidecar uses the phone/VPS network directly, so all direct providers share the same public egress IP.

V1 supports **a fixed egress proxy per provider** using the upstream gateways' own supported settings. This is useful for legitimate network separation, routing around an unreliable path, or keeping one provider on a different network. Do not use proxy rotation to bypass provider limits or anti-abuse controls.

## OpenCode

`opencode2api` natively supports:

- direct
- HTTP / HTTPS proxy
- SOCKS5 / SOCKS5H proxy
- proxy file

For a single fixed egress, keep exactly one proxy entry in `data/opencode/config.json`:

```json
{
  "proxies": ["socks5h://HOST:PORT"]
}
```

Use `["direct"]` to use the phone/VPS public IP.

## FreeBuff

`Freebuff2API` natively supports one HTTP/SOCKS5 egress proxy through `http_proxy`.

In `data/freebuff/config.json`:

```json
{
  "http_proxy": "socks5://HOST:PORT"
}
```

Leave it empty for direct egress.

## Hybrid layout

A later hub release can support a remote-provider mode so one sidecar can live on a VPS while hubd stays on the phone. Until then, the simplest split-egress setup is:

```text
phone hubd
├─ opencode2api -> fixed proxy/VPS egress
└─ freebuff2api -> direct phone egress
```

or the reverse.

Provider credentials remain owned by the provider sidecar; the hub should not contain proxy credentials in its own public config.