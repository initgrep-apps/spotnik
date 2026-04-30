# Spotnik

[![CI](https://github.com/initgrep-apps/spotnik/actions/workflows/ci.yml/badge.svg)](https://github.com/initgrep-apps/spotnik/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/initgrep-apps/spotnik)](https://github.com/initgrep-apps/spotnik/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/initgrep-apps/spotnik)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A terminal Spotify client . Keyboard-driven, single binary, beautiful in a terminal.

---

## Features

- **10-pane btop-style grid** — Now Playing, Queue, Playlists, Albums, Liked Songs,
  Recently Played, Top Tracks, Top Artists across two pages
- **Braille visualizer** — animated waveform with gradient seek bar and volume bar
- **Search overlay** — full-text search across tracks, artists, albums, playlists with
  debounced input and prefix autocomplete
- **Device switcher** — transfer playback to any Spotify Connect device
- **Preset layouts** — `p` cycles through curated layouts; `1`–`8` toggle individual panes
  (btop-style)
- **Playlist management** — create, rename, reorder tracks, remove from playlist
- **11 themes** — black (default), monokai, catppuccin, nord, light, dracula, gruvbox,
  rosepine, solarized, synthwave, tokyonight
- **Gateway observability** — Page B shows live request flow, token bucket state, and
  API event log
- **Adaptive polling** — polling intervals back off when idle; rate-limit responses trigger
  automatic retry-after cooldown
- **PKCE auth** — no client secret required; tokens stored in OS keychain

---

## Installation

### Homebrew (macOS / Linux)

```bash
brew install initgrep-apps/tap/spotnik
```

### Scoop (Windows)

```powershell
scoop bucket add spotnik https://github.com/initgrep-apps/scoop-bucket
scoop install spotnik
```

### DEB package (Ubuntu / Debian)

```bash
# Replace <version> with the latest release, e.g. 0.1.0
wget https://github.com/initgrep-apps/spotnik/releases/latest/download/spotnik_<version>_linux_amd64.deb
sudo dpkg -i spotnik_<version>_linux_amd64.deb
```

### RPM package (Fedora / RHEL)

```bash
# Replace <version> with the latest release, e.g. 0.1.0
wget https://github.com/initgrep-apps/spotnik/releases/latest/download/spotnik_<version>_linux_amd64.rpm
sudo rpm -i spotnik_<version>_linux_amd64.rpm
```

### Go install

```bash
go install github.com/initgrep-apps/spotnik@latest
```

### Binary download

Download a pre-built binary for your platform from the
[Releases page](https://github.com/initgrep-apps/spotnik/releases/latest).

---

## Prerequisites

- **Spotify Premium** — playback controls require a Premium subscription
- **Spotify Developer app** — needed for the PKCE OAuth flow:
  1. Go to [developer.spotify.com/dashboard](https://developer.spotify.com/dashboard)
  2. Create an app and add `http://127.0.0.1` as a Redirect URI
  3. Copy the Client ID — you'll need it in the config file (see [Configuration](#configuration))

---

## Quick Start

```bash
# First launch — Spotnik opens your browser for Spotify authorization
spotnik

# Or run the auth flow explicitly
spotnik auth
```

Tokens are stored securely in your OS keychain. After the first auth, `spotnik` launches
directly into the TUI.

---

## Keybindings

> When changing any keybinding, update this section, `docs/system/design.md §17`, and
> the `helpContent` var in `internal/ui/panes/help_overlay.go` in the same commit.

### Global

| Key | Action |
|-----|--------|
| `/` | Open search overlay |
| `d` | Open device switcher |
| `u` | Open user profile overlay |
| `t` | Open theme switcher |
| `?` | Open help overlay |
| `q` | Quit |
| `0` | Toggle Page A / Page B |
| `1`–`8` | Toggle pane visibility on Page A |
| `1`–`5` | Toggle pane visibility on Page B |
| `p` | Cycle preset layout |

### Playback

Playback keys are always active regardless of which pane has focus.

| Key | Action |
|-----|--------|
| `Space` | Play / pause |
| `←` / `→` | Previous / next track |
| `+` / `-` | Volume up / down |
| `s` | Toggle shuffle |
| `r` | Cycle repeat mode |
| `v` | Cycle visualizer pattern |

### Navigation

| Key | Action |
|-----|--------|
| `Tab` | Next pane focus |
| `Shift+Tab` | Previous pane focus |
| `↑` / `k` | Scroll up |
| `↓` / `j` | Scroll down |
| `Esc` | Close overlay · clear filter · scroll top |

### Pane Actions

| Key | Action | Context |
|-----|--------|---------|
| `Enter` | Select / play item | Focused pane |
| `f` | Toggle filter | List panes |
| `g` | Cycle time range | TopTracks / TopArtists |
| `A` | Add to queue | Search overlay, list panes |
| `i` | Like / unlike track | LikedSongs pane |
| `x` | Remove track from playlist | Playlists pane track sub-view |
| `Shift+↑` / `Shift+↓` | Reorder track | Playlists pane |

### Profile Overlay

| Key | Action |
|-----|--------|
| `l` | Logout — ends session, keeps Client ID. Press twice to confirm. |
| `f` | Forget — removes session + Client ID. Press twice to confirm. |

### Search Overlay

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle search category |
| `Enter` | Play selected result |
| `Ctrl+A` | Add result to queue |
| `Ctrl+U` | Clear search input |
| `PgDn` / `PgUp` | Next / previous result page |
| `Esc` | Close search overlay |

---

## Configuration

Config file location: `~/.config/spotnik/config.toml`

Spotnik bootstraps this file on first launch. Key options:

```toml
[spotify]
# Your Spotify Developer app client ID.
# Required only if you did not use a pre-built binary with an embedded client ID.
client_id = "your-client-id-here"

[preferences]
# Active theme. Options: black (default), monokai, catppuccin, nord, light,
# dracula, gruvbox, rosepine, solarized, synthwave, tokyonight
theme = "black"
```

---

## Development

### Prerequisites

- **Go 1.26+** (see `go.mod`) — `brew install go` or download from <https://go.dev/dl/>
- **golangci-lint** — required by `make lint` and `make ci`:
  ```bash
  brew install golangci-lint
  # or
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
  ```

### Build and run

```bash
git clone https://github.com/initgrep-apps/spotnik.git
cd spotnik
make build       # binary at bin/spotnik
make run         # build + run
```

### Make targets

| Target | What it does |
|--------|-------------|
| `make build` | Compile to `bin/spotnik` |
| `make run` | Build + run |
| `make test` | Unit tests (`-race -count=1`) |
| `make test-integration` | Integration tests (build tag `integration`) |
| `make test-coverage` | Unit tests + coverage; fails below 80% |
| `make lint` | Run `golangci-lint ./...` |
| `make fmt` / `make fmt-check` | Format / verify formatting |
| `make tidy-check` | Verify `go.mod` / `go.sum` are tidy |
| `make ci` | Full pre-commit gate: `fmt-check → tidy-check → lint → test-coverage → build` |
| `make clean` | Remove `bin/` and coverage artifacts |
| `make install` | Install binary to `$GOPATH/bin` |
| `make release` | Cross-compile for all release targets |

### Debugging

```bash
DEBUG=1 ./bin/spotnik           # enables Bubble Tea debug log
tail -f debug.log               # in another terminal

go test -race ./...             # race detector

./bin/spotnik auth logout       # remove tokens, keep client_id
./bin/spotnik auth forget       # remove tokens AND client_id
```

Press `0` inside the app to switch to Page B — live API gateway request flow and network
event log, useful for diagnosing rate-limit or connectivity issues.

### Architecture

See [docs/system/architecture.md](docs/system/architecture.md) for the full reference.
The `docs/system/` folder also holds [design.md](docs/system/design.md) (UI layout spec),
[tui.md](docs/system/tui.md) (primitive contracts), [cli.md](docs/system/cli.md) (CLI
output spec), and [api-guide.md](docs/system/api-guide.md) (Spotify Web API capability
inventory).

---

## License

MIT — see [LICENSE](LICENSE).
