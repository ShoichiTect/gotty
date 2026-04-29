# iPad and ttyd investigation notes

## Goal

Understand why the patched GoTTY baseline works comfortably from iPad Safari/Chrome while ttyd does not, then preserve that compatibility while selectively adopting ttyd features.

Target scenario:

- iPadOS Safari and Chrome as clients.
- Mac mini as the host.
- Access over Tailscale and/or Cloudflare Tunnel.
- Interactive shell/development workflow.

## Current hypothesis

The failure is likely related to frontend/protocol differences rather than the general concept of a browser terminal.

The patched GoTTY baseline is simpler:

- Text WebSocket messages.
- Base64 text output.
- No WebGL renderer.
- No fetch-before-connect token flow.
- Simple TypeScript frontend.
- Simple reconnect behavior.

Current ttyd is more advanced but also has more moving parts:

- Binary WebSocket messages.
- `ArrayBuffer` and `TextEncoder` / `TextDecoder`.
- `@xterm/xterm` v5.
- WebGL renderer by default.
- Token fetch from `/token` before WebSocket connection.
- Flow control with PAUSE/RESUME.
- Preact lifecycle.
- Multiple xterm addons.

Any one of those may interact poorly with iPad Safari/Chrome, Tailscale, Cloudflare Tunnel, or reverse proxy settings.

## Suspected causes to test

### 1. WebGL renderer on iPad Safari

ttyd defaults to:

```ts
rendererType: 'webgl'
```

iPad Safari supports WebGL, but xterm's WebGL renderer can still be a source of black screens, context loss, rendering stalls, or input/render mismatch.

Test ttyd with:

```bash
ttyd -p 7681 -W -t rendererType=dom bash
```

and:

```bash
ttyd -p 7681 -W -t rendererType=canvas bash
```

If either works better, avoid WebGL as the default in this fork.

### 2. Binary WebSocket protocol

ttyd uses binary WebSocket frames. GoTTY currently uses text frames.

Normally binary WebSocket should work, but the combination of iPad browser, Cloudflare Tunnel, and reverse proxy behavior should be verified before migrating GoTTY's protocol.

Recommendation:

- Keep GoTTY's text WebSocket protocol until iPad tests show a binary protocol is safe.

### 3. Token fetch flow

ttyd fetches `/token` before opening the WebSocket. This may fail or behave differently under:

- Cloudflare Access.
- Basic Auth.
- Reverse proxy path rewriting.
- Base path deployments.
- iPad Safari caching or credential behavior.

GoTTY's current script-based token is less elegant but simpler. It should not be replaced until the proxy/iPad behavior is understood.

### 4. Focus, keyboard, and viewport changes

iPad Safari has special behavior around:

- software keyboard display/hide,
- viewport height changes,
- input focus,
- page backgrounding,
- beforeunload handling.

Any frontend modernization should test these explicitly.

### 5. Cloudflare Tunnel and reverse proxy details

Potential sources of failure:

- WebSocket upgrade not reaching backend.
- Wrong `Host` or `Origin`.
- Wrong base path.
- HTTP/2 or proxy behavior around WebSocket.
- Authentication headers not forwarded as expected.
- `/token` path not mapped correctly.

## Data to collect

For ttyd failures and GoTTY successes, record:

- Browser: Safari or Chrome.
- iPadOS version.
- Access path: Tailscale direct or Cloudflare Tunnel.
- URL shape: root path or mounted base path.
- Whether the HTML page loads.
- Whether `/token` succeeds if testing ttyd.
- Whether `/ws` returns 101 Switching Protocols.
- WebSocket close code.
- JavaScript console errors.
- Network errors.
- Whether terminal renders.
- Whether keyboard input works.
- Whether software keyboard resize breaks layout.
- Whether reconnect works after background/sleep.

## Suggested server-side diagnostics for this fork

Add a debug mode that logs:

- HTTP method and path.
- HTTP status.
- WebSocket upgrade attempt.
- WebSocket upgrade failure reason.
- Remote address.
- `User-Agent`.
- `Origin`.
- `Host`.
- `X-Forwarded-For`.
- `X-Forwarded-Proto`.
- `X-Forwarded-Host`.
- `CF-Connecting-IP`.
- WebSocket close reason.
- Ping/pong events.
- Resize messages.
- Reconnect attempts.

This should be available behind a debug flag so normal output remains clean.

## Suggested client-side diagnostics

Add optional debug logging in the browser for:

- WebSocket URL.
- open/error/close events.
- close code and reason.
- initial terminal cols/rows.
- resize events.
- ping send failures.
- reconnect attempts.
- user agent.
- viewport dimensions.

For iPad Safari, use Safari's remote Web Inspector from macOS when possible.

## ttyd comparison commands

Direct Tailscale test:

```bash
ttyd -d 7 -p 7681 -W bash
```

DOM renderer:

```bash
ttyd -d 7 -p 7681 -W -t rendererType=dom bash
```

Canvas renderer:

```bash
ttyd -d 7 -p 7681 -W -t rendererType=canvas bash
```

Check with Cloudflare Tunnel after direct Tailscale behavior is understood.

## GoTTY comparison commands

Baseline read-only:

```bash
gotty -p 9090 bash
```

Writable interactive session:

```bash
gotty -w -p 9090 bash
```

With Basic Auth:

```bash
gotty -w -c 'user:pass' -p 9090 bash
```

With random URL:

```bash
gotty -w -r -p 9090 bash
```

For Tailscale direct access, bind to all interfaces or the Tailscale address explicitly.

## Compatibility decisions for the fork

Until testing proves otherwise:

- Keep WebGL off by default.
- Keep text WebSocket as the default protocol.
- Keep the frontend small.
- Avoid fetch-token dependency as a hard requirement.
- Add reverse-proxy options before large frontend rewrites.
- Treat iPad Safari as the primary compatibility target.

## Future iPad profile

Consider a profile such as:

```bash
gotty --client-profile ipad -w bash
```

Possible profile behavior:

- DOM/canvas renderer.
- WebGL disabled.
- Resize debounce.
- Mobile viewport CSS.
- Reconnect enabled.
- Configurable ping interval.
- Safer keyboard focus handling.

