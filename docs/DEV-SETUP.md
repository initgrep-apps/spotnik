# Spotnik Developer Setup

This guide walks you from a clean machine to a running local build.

---

## Prerequisites

### Go 1.22+

```bash
go version   # must be ≥ go1.22
```

Download from <https://go.dev/dl/> if needed.

### golangci-lint

Required for `make lint` and `make ci`. Install via:

```bash
# macOS / Linux (recommended)
brew install golangci-lint

# Or via the official script
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b $(go env GOPATH)/bin

golangci-lint --version
```

See <https://golangci-lint.run/usage/install/> for other platforms.

---

## Spotify App Setup

Spotnik uses PKCE OAuth 2.0. You need a Spotify Developer application with the correct
redirect URI registered.

1. Go to <https://developer.spotify.com/dashboard>
2. Log in and click **Create app**
3. Fill in any name and description
4. Add the redirect URI: `http://localhost:8888/callback`
5. Save, then open the app settings to copy your **Client ID**

The Client ID is a 32-character hex string. You do **not** need the Client Secret.

---

## Environment Variables

Create a `.env` file in the project root (it is `.gitignore`d):

```bash
SPOTIFY_CLIENT_ID=<your-32-char-client-id>
```

The Makefile automatically loads `.env` on every target via `-include .env && export`.

Alternatively, export the variable in your shell:

```bash
export SPOTIFY_CLIENT_ID=<your-client-id>
```

Without a Client ID the binary will build but authentication will fail.

---

## Build and Run

```bash
# Build binary to bin/spotnik
make build

# Build and immediately run
make run

# First-time auth (opens browser)
./bin/spotnik auth

# Run the app
./bin/spotnik
```

---

## Make Targets

| Target | What it does |
|--------|-------------|
| `make build` | Compile to `bin/spotnik` |
| `make run` | Build + run |
| `make test` | Unit tests (`-race -count=1`) |
| `make test-integration` | Integration tests (requires `//go:build integration` tag) |
| `make test-coverage` | Unit tests + coverage report; fails below 80% |
| `make lint` | Run `golangci-lint ./...` |
| `make fmt` | Format all Go files with `gofmt` |
| `make fmt-check` | Verify formatting (fails if files would change) — used by CI |
| `make tidy-check` | Verify `go.mod`/`go.sum` are tidy |
| `make ci` | Full pre-commit check: `fmt-check → tidy-check → lint → test-coverage → build` |
| `make clean` | Remove `bin/`, `coverage.out`, `coverage.html` |
| `make install` | Install binary to `$GOPATH/bin` |
| `make release` | Cross-compile for all target platforms |

---

## Linting

```bash
make lint
```

Uses default `golangci-lint` rules — no custom `.golangci.yml`. Reviewers will reject
PRs with lint failures.

---

## Debugging Tips

### Enable Bubble Tea debug logging

Set `DEBUG=1` before running to enable Bubble Tea's log output:

```bash
DEBUG=1 ./bin/spotnik
```

Then in a second terminal:

```bash
tail -f debug.log
```

### Race detector

All test targets pass `-race`. Run manually with:

```bash
go test -race ./...
```

### Page B (Nerd Status)

Press `0` to toggle to Page B, which shows the live API gateway request flow and
network event log. This is useful for diagnosing rate-limit or connectivity issues
without leaving the app.

### Auth troubleshooting

If auth fails, delete stored tokens and retry:

```bash
./bin/spotnik auth logout
```

Tokens are stored in the OS keychain (macOS Keychain, Linux Secret Service, Windows
Credential Manager) under the `spotnik` service name.

---

## Project Layout

```
spotnik/
├── main.go              ← entry point only
├── cmd/root.go          ← CLI flags, auth check, app launch
├── internal/
│   ├── app/             ← root Bubble Tea model
│   ├── api/             ← Spotify HTTP clients, gateway, rate limiting
│   ├── domain/          ← shared types bridging api/ and ui/
│   ├── ui/
│   │   ├── panes/       ← 10 panes + overlays
│   │   ├── components/  ← visualizer, gradient bars, filter, table wrapper
│   │   ├── layout/      ← LayoutManager, preset system, focus rotation
│   │   └── theme/       ← Theme interface + 11 implementations
│   ├── state/           ← central Store (single source of truth)
│   ├── config/          ← config loading + defaults
│   ├── prefs/           ← runtime preference persistence
│   └── keychain/        ← token storage abstraction
├── testdata/fixtures/   ← JSON fixtures for API mock tests
└── docs/                ← architecture, design, and spec documentation
```

For the full architecture reference see [ARCHITECTURE.md](ARCHITECTURE.md).
