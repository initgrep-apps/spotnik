# Changelog

All notable changes to Spotnik are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0-rc1] - 2026-05-01


### Spotify (Music page)

- Now Playing pane: current track, braille visualizer, gradient seek bar, volume bar
- Queue: upcoming tracks
- Playlists: saved playlists with reorder, add to queue, and remove-from-playlist
- Albums: saved albums with track view
- Liked Songs: liked tracks with like / unlike toggle
- Recently Played: listening history
- Top Tracks and Top Artists with time-range cycling

### Search and Discovery

- Full-screen search across tracks, artists, albums, and playlists
- Prefix autocomplete with tab completion
- Paginated results with category cycling
- Add to queue from search results

### Devices and Playback

- Spotify Connect device switcher
- Play, pause, skip, shuffle, repeat, and volume controls
- Volume bar with 1% steps and partial-block rendering
- Optimistic playback updates
- API gateway with rate limiting, request dedup, adaptive polling, and automatic retry-after

### Developer view (Stats page)

- Gateway Health pane: token bucket, backoff, and rate-limit state
- Polling Traffic pane: per-pane request volume
- Live Request Flow pane: in-flight requests
- Network Log pane: recent API events

### Theming

- 11 built-in themes: black, monokai, catppuccin, nord, light, dracula,
  gruvbox, rosepine, solarized, synthwave, tokyonight
- Runtime theme switcher (`t`)

### Authentication and Profile

- PKCE OAuth flow (no client secret required)
- Token storage in OS keychain
- `spotnik auth {register, login, logout, forget, status}` CLI subcommands
- Profile overlay with logout and forget actions

### Configuration

- TOML config bootstrapped at `~/.config/spotnik/config.toml` on first launch
- Persistent preferences: theme, layout preset, visualizer pattern
- Configurable CLI palette: `auto`, `fixed`, or `theme`
- Configurable glyph rendering: `auto`, `unicode`, or `ascii`

### Layout

- Two-page btop-style pane layout (Music page: Spotify, Stats page: Developer)
- Per-pane visibility toggles (`1`–`8`)
- Preset layouts cycled with `p`
- Universal filter on list panes (`f`, `Esc` to clear)

### Installation

- macOS and Linux installer via `curl ... | bash`
- Windows installer via `irm ... | iex`
- DEB and RPM packages
- Pre-built binaries for linux, darwin, and windows on amd64 and arm64

[0.1.0]: https://github.com/initgrep-apps/spotnik/releases/tag/v0.1.0
