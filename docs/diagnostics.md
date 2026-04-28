# Diagnostics

This fork includes an initial debug mode for iPad, Tailscale, Cloudflare Tunnel, and WebSocket investigation.

Enable it with:

```bash
gotty --debug -w -p 9090 bash
```

or:

```bash
gotty -d -w -p 9090 bash
```

## What is logged

When debug mode is enabled, GoTTY logs additional information for:

- HTTP requests and responses.
- WebSocket upgrade requests.
- WebSocket upgrade success/failure.
- WebSocket subprotocol.
- WebSocket initialization message metadata.
- WebTTY initialization options.
- input message sizes.
- ping messages.
- resize events and terminal dimensions.
- connection close reason.

The debug logs include proxy and browser-related request headers when present:

- `User-Agent`
- `Origin`
- `Host`
- `X-Forwarded-For`
- `X-Forwarded-Proto`
- `X-Forwarded-Host`
- `CF-Connecting-IP`

This is intended to make iPad Safari/Chrome behavior observable when connecting through Tailscale or Cloudflare Tunnel.

## Example workflow

Start GoTTY on the Mac mini:

```bash
gotty --debug -w -p 9090 bash
```

Then connect from:

- iPad Safari over Tailscale.
- iPad Chrome over Tailscale.
- iPad Safari over Cloudflare Tunnel.
- iPad Chrome over Cloudflare Tunnel.

Compare logs for:

- user agent differences,
- missing or unexpected forwarded headers,
- failed WebSocket upgrades,
- wrong origin/host values,
- repeated reconnects,
- resize events caused by software keyboard changes.

## Notes

Debug logging is intentionally verbose. Use it for investigation, not as a normal production setting.
