# Dependency and security audit notes

This document records the initial dependency and security findings for the GoTTY fork modernization effort.

## Scope

Investigated trees:

- `gotty-original`
- `gotty-fork`
- reference repository `../ttyd`

The current `gotty-fork` is still at the upstream GoTTY baseline. The sibling `gotty-original` contains local modernization fixes and is used as the immediate baseline to port into this fork.

## Original local changes to port

`gotty-original` contains these local changes:

- `main.go`
  - `github.com/codegangsta/cli` -> `github.com/urfave/cli`
- `utils/flags.go`
  - `github.com/codegangsta/cli` -> `github.com/urfave/cli`
- `backend/localcommand/local_command.go`
  - `github.com/kr/pty` -> `github.com/creack/pty`
  - macOS PTY startup adjusted with `pty.StartWithAttrs` and `Setsid: true`
- `go.mod` and `go.sum` added

Do not port the local generated binary from `gotty-original`.

## Go dependency findings

The patched original tree currently has a Go Modules file, but the old `vendor/` directory still exists. This causes inconsistent vendoring unless builds use `-mod=mod` or the vendor directory is regenerated.

Recommended direction:

- Remove `Godeps/` and `vendor/` from the maintained fork.
- Use `go.mod` and `go.sum` as the source of truth.
- Add CI checks for module consistency.

### Dependencies to remove immediately

These were present in the patched `go.mod` but were not needed by the main module:

- `github.com/kr/pty`
- `github.com/urfave/cli/v2`

They should be removed when the Go baseline is ported and `go mod tidy` is run.

### Dependencies to modernize later

| Dependency | Current role | Suggested direction |
| --- | --- | --- |
| `github.com/pkg/errors` | Error wrapping | Replace with standard `errors` and `fmt.Errorf("%w")` |
| `github.com/elazarl/go-bindata-assetfs` | Embedded static assets | Replace with Go `embed` |
| `github.com/yudai/hcl` | Config parsing | Revisit; consider maintained HCL or simpler config |
| `github.com/gorilla/websocket` | WebSocket server | Keep initially; consider alternatives only after baseline stability |
| `github.com/NYTimes/gziphandler` | gzip middleware | Keep initially or replace with local middleware later |
| `github.com/fatih/structs` | Struct/tag reflection | Keep initially; remove only with config/flag refactor |

## Go vulnerability checks

### Phase 2 govulncheck results (2026-04-29)

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
make vulncheck
```

5 standard library vulnerabilities found (all fixed in go1.26.2):

| Vuln ID | Package | Severity | Summary |
|---------|---------|----------|---------|
| GO-2026-4865 | `html/template` | — | JsBraceDepth context tracking bugs (XSS) |
| GO-2026-4866 | `crypto/x509` | — | Case-sensitive excludedSubtrees name constraints → auth bypass |
| GO-2026-4870 | `crypto/tls` | — | Unauthenticated TLS 1.3 KeyUpdate record → DoS |
| GO-2026-4946 | `crypto/x509` | — | Inefficient policy validation |
| GO-2026-4947 | `crypto/x509` | — | Unexpected work during chain building |

All five are in the Go stdlib, not in third-party dependencies. No third-party
vulnerabilities are called by this codebase.

Action: upgrade to go1.26.2+ when available.

`make vulncheck` added.

## Frontend dependency findings

The old frontend is significantly outdated:

```json
{
  "devDependencies": {
    "license-loader": "^0.5.0",
    "ts-loader": "^2.0.3",
    "typescript": "^2.3.2",
    "uglifyjs-webpack-plugin": "^1.0.0-beta.2",
    "webpack": "^2.5.1"
  },
  "dependencies": {
    "libapps": "github:yudai/libapps#release-hterm-1.70",
    "xterm": "^2.7.0"
  }
}
```

Initial `npm audit` against `gotty-original/js` reported:

- 34 total vulnerabilities
- 1 low
- 15 moderate
- 9 high
- 9 critical

Notable findings:

| Package | Severity | Notes |
| --- | --- | --- |
| `xterm` | high | Old `xterm` versions before 3.8.1 include an RCE advisory |
| `webpack` | critical | Old webpack 2 dependency chain has many advisories |
| `loader-utils` | critical | Prototype pollution / ReDoS advisories |
| `lodash` | critical | Multiple prototype pollution / code injection advisories |
| `minimist` | critical | Prototype pollution advisories |
| `pbkdf2` | critical | Cryptographic handling advisories |
| `sha.js` | critical | Hash handling advisory |
| `cipher-base` | critical | Crypto handling advisory |
| `elliptic` | critical | Multiple crypto advisories |

Conclusion: the frontend should not be patched package-by-package. It should be modernized as a focused project.

## Frontend modernization direction

Reference ttyd currently uses:

- `@xterm/xterm` v5
- `@xterm/addon-fit`
- `@xterm/addon-webgl`
- `@xterm/addon-canvas`
- TypeScript 5
- webpack 5

For this fork, adopt only the minimal safe subset first:

1. `@xterm/xterm` v5.
2. `@xterm/addon-fit`.
3. Optional `@xterm/addon-web-links`.
4. WebGL available only as an opt-in feature.
5. Keep text WebSocket protocol initially.
6. Remove hterm/libapps after xterm v5 is validated on iPad.

## Security design findings

### `PermitArguments` default

Current GoTTY has `PermitArguments` defaulting to `true`. This means clients may be able to provide command arguments via URL query parameters unless explicitly disabled.

Recommended change:

- Default to `false`.
- Keep an explicit opt-in flag for compatibility.

This matches ttyd's safer `--url-arg` opt-in model.

### Basic Auth credential exposure

The current implementation writes the configured credential into a JavaScript file as `gotty_auth_token`.

Short-term:

- Keep behavior until compatibility work is complete.
- Avoid making it worse.

Medium-term options:

- Use WebSocket upgrade-time Basic Auth instead of JS token.
- Use a server-generated session token rather than exposing the credential itself.
- Consider a `/token` endpoint only if it works reliably through Cloudflare/Tailscale/iPad Safari.

### HTTP server timeouts

The current `http.Server` does not set timeouts.

Recommended minimum:

- Add `ReadHeaderTimeout`.

Be careful with `WriteTimeout` because WebSocket connections are long-lived.

### TLS minimum version

Set TLS minimum version to TLS 1.2 or newer.

### Security headers

Add basic headers where compatible:

- `X-Content-Type-Options: nosniff`
- `Referrer-Policy`
- Consider CSP after frontend changes are understood.

## Do not break the known-good iPad path

Security fixes should be sequenced so that the current known-good behavior can be tested after each step:

- iPad Safari over Tailscale.
- iPad Chrome over Tailscale.
- iPad Safari over Cloudflare Tunnel.
- iPad Chrome over Cloudflare Tunnel.

Avoid combining dependency, protocol, and frontend renderer changes in the same commit.

