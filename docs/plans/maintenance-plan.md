# GoTTY fork maintenance plan

## Purpose

This fork is maintained for a specific use case:

> Stable browser terminal access from iPadOS Safari/Chrome to a macOS host, especially a Mac mini, over Tailscale or Cloudflare Tunnel.

Although `ttyd` is actively maintained and should be used as an important reference implementation, this fork should not blindly become ttyd. The current patched GoTTY implementation works comfortably from iPad where ttyd does not, so the main goal is to preserve the properties that make GoTTY reliable on iPad while modernizing dependencies and selectively importing ttyd ideas.

## Guiding principles

1. Keep the iPad Safari/Chrome path working at every step.
2. Prefer small, testable changes over large rewrites.
3. Fix old dependencies and known vulnerabilities first.
4. Preserve GoTTY's simple text WebSocket protocol until there is evidence that changing it is safe on iPad.
5. Do not make WebGL the default renderer.
6. Treat Tailscale, Cloudflare Tunnel, and reverse proxy deployments as first-class scenarios.
7. Import ttyd features selectively, especially operational and proxy-friendly features.

## Current baseline

The sibling `gotty-original` tree currently contains local fixes that make the original project build and run on modern macOS:

- Go Modules introduced.
- `github.com/codegangsta/cli` replaced with `github.com/urfave/cli` v1-compatible import path.
- `github.com/kr/pty` replaced with `github.com/creack/pty`.
- macOS PTY startup adjusted to avoid the `Setctty set but Ctty not valid in child` failure.

The first implementation step for this fork should be to reproduce that working baseline here without committing generated binaries.

## Why not simply use ttyd?

`ttyd` is actively maintained and has many desirable features, but in the target environment it currently fails or behaves poorly from iPad Safari/Chrome. The target environment is:

- iPadOS Safari/Chrome as the browser client.
- Mac mini as the development host.
- Access over Tailscale and/or Cloudflare Tunnel.
- Interactive SSH/development workflow.

The local patched GoTTY baseline works well in that scenario. Therefore this fork exists to keep that practical compatibility while gradually modernizing the codebase.

## Important differences from ttyd

GoTTY current frontend/protocol (after Phase 4):

- `@xterm/xterm` v5 with fit and web-links addons.
- TypeScript/webpack 5 frontend (no framework).
- WebSocket subprotocol `webtty`.
- Text WebSocket messages.
- Server output is base64 encoded text.
- `Uint8Array` passed directly to xterm for stateful UTF-8 handling.
- JavaScript-side ping every 30 seconds (configurable via `--ping-interval`).
- No WebGL renderer (addon not loaded).
- No Preact lifecycle.
- No fetch-based token endpoint.
- No hterm/libapps (removed).

Current ttyd frontend/protocol:

- `@xterm/xterm` v5 and multiple addons.
- WebSocket subprotocol `tty`.
- Binary WebSocket messages using `ArrayBuffer`.
- `TextEncoder` / `TextDecoder` based protocol handling.
- Default client renderer is WebGL.
- Token is fetched from `/token` before WebSocket connection.
- Flow control using PAUSE/RESUME messages.
- Preact-based frontend.

These differences are likely relevant to the iPad issue. This fork should therefore avoid adopting ttyd's frontend/protocol wholesale until the failure mode is understood.

## Phased plan

### Phase 0: establish a modern Go baseline

Goal: make `gotty-fork` match the currently working patched baseline.

Tasks:

- Add `go.mod` and `go.sum`.
- Replace `github.com/codegangsta/cli` with `github.com/urfave/cli` v1 import path.
- Replace `github.com/kr/pty` with `github.com/creack/pty`.
- Keep the macOS PTY fix from the patched original tree.
- Run `go mod tidy`.
- Remove unnecessary module requirements such as `github.com/kr/pty` and `github.com/urfave/cli/v2` if they are not used.
- Do not commit generated local binaries.

Open decision:

- Prefer deleting old `Godeps/` and `vendor/` and using Go Modules as the single source of truth.

### Phase 1: add iPad/Tailscale/Cloudflare diagnostics

Goal: make connection failures observable before changing frontend behavior.

Add debug logging for:

- HTTP request path.
- WebSocket upgrade success/failure.
- WebSocket close/error details where available.
- `Origin` and `Host`.
- `User-Agent`.
- `X-Forwarded-For`.
- `X-Forwarded-Proto`.
- `X-Forwarded-Host`.
- `CF-Connecting-IP`.
- Remote address.
- Ping/pong activity.
- Resize events and terminal dimensions.
- Reconnect attempts.

This phase should include comparison notes against ttyd on the same iPad/Tailscale/Cloudflare paths.

### Phase 2: security baseline without protocol churn — ✅ Done (d2608e9)

Goal: remove obvious risk while keeping the working iPad behavior.

Tasks:

- Change `PermitArguments` default from `true` to `false`.
- Add `http.Server.ReadHeaderTimeout`.
- Set TLS minimum version to TLS 1.2 or newer.
- Add basic response headers such as `X-Content-Type-Options: nosniff` and `Referrer-Policy`.
- Strengthen warnings for `--permit-write`, especially with public bind and no authentication.
- Add `govulncheck ./...` to local checks and eventually CI.
- Document npm audit findings for the old frontend. → See `docs/investigations/dependency-security-audit.md`.

Done:
- d2608e9 — all 7 tasks completed, `go build ./...` and `go vet ./...` pass.
- `govulncheck ./...` found 5 stdlib vulnerabilities (GO-2026-4865/4866/4870/4946/4947), all fixed in go1.26.2.
- `make vulncheck` target added.

Avoid in this phase:

- Binary WebSocket migration.
- WebGL default renderer.
- Large frontend rewrite.

### Phase 2.5: Go standard library cleanup — ✅ Done (f6bc85a)

Goal: mechanical, low-risk modernization that reduces diff noise for later phases.

Why now: `ioutil` is deprecated since Go 1.16, `pkg/errors` is superseded by
stdlib `errors` (Go 1.13), and `go-bindata` is superseded by `embed` (Go 1.16).
These are the smallest possible changes — no behavioral impact, no protocol risk.

Tasks:

- Replace `ioutil.ReadFile` with `os.ReadFile`, drop `io/ioutil` import.
- Replace `github.com/pkg/errors` with standard `errors` and `fmt.Errorf("%w")`.
- Replace `go-bindata`/`go-bindata-assetfs` with Go `embed` (`server/embed.go` + `server/static/`).
- Run `go mod tidy` after each step.

Avoid: `urfave/cli` v2 migration, HCL config replacement (deferred to Phase 6).

### Phase 3: iPad frontend fixes — ✅ Done (19e426f)

Goal: address real observed issues without introducing a CLI abstraction.

Renamed from the original "iPad compatibility profile" plan. A `--profile ipad`
flag would be over-engineering at this stage — the flags already exist
(`--reconnect`, `--debug`). Instead, fix the actual browser-side problems:

1. **Viewport meta tag** — add `<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">`
   to `resources/index.html`. Zero risk, one line.

2. **Resize debounce** — add 150ms debounce in the frontend resize handler.
   Prevents flicker when the iPad software keyboard changes viewport height.
   Small JS change, no protocol impact.

3. **Configurable ping interval** — expose the hardcoded 30-second ping as a
   `--ping-interval` CLI flag (Go side) and `gotty_ping_interval` JS variable
   (via `config.js` endpoint). Improves reconnection reliability after
   iPad sleep/background.

Each task is independent, testable on iPad immediately, and touches only one
layer (HTML, JS, or Go config — never all three at once).

Done:
- 19e426f — viewport meta tag, 150ms resize debounce, `--ping-interval` flag with `gotty_ping_interval` JS variable.

### Phase 4: modernize frontend carefully — ✅ Done (ac7be45)

Goal: remove known frontend vulnerabilities while preserving the working protocol.

Tasks:

1. Replace old `xterm` v2 with `@xterm/xterm` v5.
2. Use `@xterm/addon-fit`.
3. Optionally use `@xterm/addon-web-links`.
4. Keep WebGL optional and off by default.
5. Keep GoTTY's text WebSocket protocol initially.
6. Remove hterm/libapps if xterm v5 is stable enough on iPad.
7. Upgrade TypeScript and webpack.
8. Test on iPad Safari/Chrome after each step.

Done:
- ac7be45 — replaced xterm v2 with `@xterm/xterm` v5.5.0, added fit and web-links addons.
- Removed hterm/libapps entirely. hterm.ts, typings/libapps, and `--term hterm` path deleted.
- Upgraded TypeScript 2.3→5.4, webpack 2.5→5.91, ts-loader 2.0→9.5.
- Removed deprecated uglifyjs-webpack-plugin (webpack 5 has built-in terser).
- iPadOS Ctrl+C fix ported to `attachCustomKeyEventHandler` (xterm v5 public API).
- UTF-8 decoding: pass `Uint8Array` directly to `term.write()` so xterm's stateful
  parser handles multi-byte sequences split across WebSocket message boundaries
  (fixes box-drawing character corruption with React Ink / pi agent).
- `--debug` keyboard diagnostics preserved via `term.textarea` (public API in v5).
- WebSocket text protocol, base64 encoding, `webtty` subprotocol, ping/reconnect
  all unchanged.
- CSS: xterm v5 `.xterm` class selectors (was `.terminal` in v2).

Avoided:
- Binary WebSocket migration.
- WebGL renderer (not loaded).
- Preact, zmodem/trzsz, image addon.
- CLI abstraction / Preact rewrite.

Do not import all ttyd frontend features at once. Features such as Preact, binary protocol, WebGL, zmodem/trzsz, image addon, and complex flow control should be evaluated separately.

### Phase 5: proxy-friendly features from ttyd

Goal: make the fork first-class behind Tailscale, Cloudflare Tunnel, and reverse proxies.

Features to consider importing or adapting from ttyd:

- `--base-path` for mounting under a path such as `/terminal/`.
- `--auth-header` for Cloudflare Access, OAuth2 Proxy, or other reverse-proxy authentication.
- Forwarded header awareness in logs and displayed URLs.
- Configurable WebSocket ping interval and timeout.
- Clear origin-check behavior.

### Phase 6: remaining Go modernization

Goal: final cleanup of pre-module-Go patterns deferred until the core is stable.

Tasks already moved to Phase 2.5:
- `ioutil` → `os`/`io`.
- `pkg/errors` → stdlib `errors`.
- `go-bindata` → `embed`.

Remaining tasks for this phase:

- Consider `urfave/cli` v2 migration only after frontend modernization (Phase 4)
  is validated on iPad.
- Revisit the old HCL config dependency (currently unused after the JSON
  migration; may be removable).

## ttyd features to adopt selectively

Good candidates:

- `--base-path`.
- `--auth-header`.
- xterm v5.
- fit addon.
- Unicode/CJK/IME improvements.
- Client option mechanism.
- Canvas renderer as an option.
- Flow control after iPad testing.
- TLS hardening ideas.

Features to treat carefully:

- WebGL renderer as default.
- Binary WebSocket protocol as default.
- Fetch-token connection model.
- Full Preact frontend rewrite.
- zmodem/trzsz/file transfer.
- Sixel/image output.

## Validation matrix

Every meaningful frontend/protocol change should be tested against:

- iPad Safari over Tailscale.
- iPad Chrome over Tailscale.
- iPad Safari over Cloudflare Tunnel.
- iPad Chrome over Cloudflare Tunnel.
- macOS Safari direct/local.
- macOS Chrome direct/local.

Important observations:

- Does the page load?
- Does WebSocket upgrade return 101?
- Does terminal render?
- Does input work?
- Does software keyboard resize break layout?
- Does reconnect work after sleep/background?
- Are close codes/reasons logged?

