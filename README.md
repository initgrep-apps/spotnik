# Spotnik

A terminal Spotify client for developers. Keyboard-driven, single binary, beautiful in a terminal.

Not a Spotify clone — a developer-first music environment. Target user: developer with Spotify
Premium who lives in the terminal all day.

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

## Install from Source

### Prerequisites

- Go 1.22 or later
- `golangci-lint` (for linting only — not required to run)
- A Spotify Premium account
- A Spotify Developer app (see [DEV-SETUP.md](docs/DEV-SETUP.md))

### Build

```bash
git clone https://github.com/initgrep-apps/spotnik.git
cd spotnik

# Set your Spotify client ID (from the developer dashboard)
export SPOTIFY_CLIENT_ID=<your-client-id>

make build
```

This produces `bin/spotnik`.

### Authenticate

```bash
./bin/spotnik auth
```

Opens your browser for the Spotify PKCE OAuth flow. Tokens are stored in the OS keychain.

### Run

```bash
./bin/spotnik
```

---

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `/` | Open search overlay |
| `d` | Open device switcher |
| `t` | Open theme switcher |
| `?` | Open help overlay |
| `q` | Quit |
| `0` | Toggle Page A / Page B |
| `1`–`8` | Toggle pane visibility (Page A only) |
| `p` | Cycle preset layout |

### Playback

Playback keys are always active regardless of which pane has focus.

| Key | Action |
|-----|--------|
| `Space` | Play / pause |
| `n` | Next track |
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
| `j` / `k` | Scroll down / up |
| `Esc` | Close overlay or filter |

### Pane Actions

| Key | Action | Context |
|-----|--------|---------|
| `Enter` | Select / play item | Focused pane |
| `f` | Toggle filter | List panes |
| `A` | Add to queue | Search overlay, list panes |
| `i` | Like / unlike track | LikedSongs pane |
| `x` | Remove track from playlist | Playlists pane track sub-view |
| `Shift+↑` / `Shift+↓` | Reorder track | Playlists pane |

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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for branch naming, commit conventions, test
requirements, and the PR process.

## Dev Setup

See [docs/DEV-SETUP.md](docs/DEV-SETUP.md) for prerequisites, Spotify app setup, environment
variables, and all make targets.

## Testing

See [docs/TESTING.md](docs/TESTING.md) for the test architecture, patterns, and how to run
each category of tests.
