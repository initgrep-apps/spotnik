# Changelog

All notable changes to Spotnik are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0-rc4] - 2026-05-21

### Added
- Universal polling infra with per-pane backoff, error state, and auto-recovery
- Centralized API error mapper and two-column toast layout
- Overlay self-sufficiency (survives parent pane failures)
- OAuth error code → user-friendly message mapping
- Mono theme variants; page labels renamed Music→Spotify, Stats→Developer

### Fixed
- Volume bar snap-back race after debounce
- Gateway Health/Live panes: eager drain on resize, per-slot dot bar
- Polling Traffic pane: missing Stats row
- User profile not fetched for returning authenticated sessions
- Silent gap fixes in error paths (nil guards, missing fallbacks)

### Chores
- Polling infra test robustness and backoff guard
- Worktree-based feature branch workflow
- Bump `golang.org/x/term`

## [0.1.0-rc3] - 2026-05-08

### Added
- Onboarding screen with permissions messaging, InfoBox, and overlay
- Consolidated auth screens into single flow

## [0.1.0-rc2] - 2026-05-07

### Added
- Rustup-style PATH integration for installer scripts

### Changed
- OSC 52 cross-platform clipboard (replaces `pbcopy`/`xclip` shelling)

### Chores
- Bump `actions/setup-go`, `actions/checkout`, `release-please-action`
- Bump Go minor-patch group

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

[0.1.0-rc4]: https://github.com/initgrep-apps/spotnik/releases/tag/v0.1.0-rc4
[0.1.0-rc3]: https://github.com/initgrep-apps/spotnik/releases/tag/v0.1.0-rc3
[0.1.0-rc2]: https://github.com/initgrep-apps/spotnik/releases/tag/v0.1.0-rc2
[0.1.0-rc1]: https://github.com/initgrep-apps/spotnik/releases/tag/v0.1.0-rc1
