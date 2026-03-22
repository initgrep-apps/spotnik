# Feature Overview — spotnik

> This file is the feature map. It lists all planned features, their implementation order,
> dependencies, and links to their detailed specs.
> Do not start implementing a feature until its spec file is read.

---

## Implementation Order

Features must be built in order. Each depends on the previous being stable and tested.

| # | Feature | Spec | Status | Depends On |
|---|---|---|---|---|
| 1  | Theme System | `01-theme-system.md` | 🔲 Not started | — |
| 2 | Authentication | `02-auth.md` | 🔲 Not started | — |
| 3 | Playback Controls | `03-playback.md` | 🔲 Not started | Theme System, Auth |
| 4 | Library Browser | `04-library.md` | 🔲 Not started | Auth, Playback |
| 5 | Search | `05-search.md` | 🔲 Not started | Auth |
| 6 | Queue Management | `06-queue.md` | 🔲 Not started | Playback |
| 7 | Device Switcher | `07-devices.md` | 🔲 Not started | Playback |
| 8 | Stats Dashboard | `08-stats.md` | 🔲 Not started | Auth |
| 9 | Playlist Manager | `09-playlists.md` | 🔲 Not started | Library |

> **Note on 1:** Theme System (01) and Auth (02) have no dependencies on each other and can be
> built in parallel by separate agents. Both must be complete before Feature 03 begins.

> **Note on views:** Features 08 (Stats) and 09 (Playlists) use alternative views that
> temporarily replace the three-pane layout. Pressing `1` always returns to the main
> Library | Player | Queue layout. This does not violate the three-pane freeze — the freeze
> means the three-pane layout itself is never modified, not that it must be the only view.

---

## Testing Convention

All features use two test tiers: **Unit** and **Integration**.

- **Unit tests** live in standard `*_test.go` files. They test individual functions, model handlers, and API methods in isolation.
- **Integration tests** live in `*_integration_test.go` files tagged with `//go:build integration`. They test multi-component flows.
- `make test` runs unit tests only. `make ci` runs both.
- See `docs/ARCHITECTURE.md` → "Integration Test Convention" for full details.

Each task in every feature spec lists its required tests by category.

---

## Feature Scope Summary

### Feature 1: Theme System
Pure infrastructure — no user-facing UI. Defines the `Theme` interface, implements all five
themes (True Black, Monokai, Catppuccin, Nord, Light), and wires the active theme from
config into the app at startup. Every UI component depends on this existing first.

### Feature 2: Authentication
First-run OAuth PKCE flow. Token storage in OS keychain. Token refresh. No app function works without this.

### Feature 3: Playback Controls
The heart of the app. Display currently playing track. Play/pause, skip, seek, volume, shuffle, repeat.
Polling loop for live state updates. This is what users see every second.

### Feature 4: Library Browser
Left pane. Navigate playlists, albums, liked songs. Select a playlist to load its tracks.
Plays a playlist/album from selection. Recently played list.

### Feature 5: Search
The `/` search overlay. Live search as user types (debounced). Results grouped by type.
Play directly, add to queue, or open in library.

### Feature 6: Queue Management
Right pane. Show upcoming queue. Add items from library/search.
Current queue sourced from Spotify's own queue endpoint. Queue removal is not supported
by the Spotify Web API.

### Feature 7: Device Switcher
`d` key overlay. List all Spotify Connect devices. One-keypress transfer of playback.
Show active device in header bar.

### Feature 8: Stats Dashboard
`2` view. Top tracks and artists by time range. Recently played history.
This is the differentiating feature — makes Spotnik more than a player.

### Feature 9: Playlist Manager
`3` view. Create playlists. Add/remove tracks. Rename. Reorder.
Power user feature for curating music libraries from the terminal.

---

## Versioning

| Version | Includes |
|---|---|
| v0.1.0 | Features 1 + 2 + 3 (theme system + auth + basic playback) |
| v0.2.0 | Features 4 + 5 (library + search) |
| v0.3.0 | Features 6 + 7 (queue + devices) |
| v0.4.0 | Feature 8 (stats) |
| v1.0.0 | Feature 9 (playlist manager) + polish |

---

*Last updated: 2026-02-21*
