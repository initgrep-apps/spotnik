# Feature Overview — spotnik

> This file is the feature map. It lists all planned features, their implementation order,
> dependencies, and links to their detailed specs.
> Do not start implementing a feature until its spec file is read.

---

## Implementation Order

Features must be built in order. Each depends on the previous being stable and tested.

| # | Feature | Spec | Status | Depends On | PR |
|---|---|---|---|---|---|
| 1  | Theme System | `01-theme-system.md` | ✅ Complete | — | — |
| 2 | Authentication | `02-auth.md` | ✅ Complete | — | — |
| 3 | Playback Controls | `03-playback.md` | ✅ Complete | Theme System, Auth | — |
| 4 | Library Browser | `04-library.md` | ✅ Complete | Auth, Playback | #6 |
| 5 | Search | `05-search.md` | ✅ Complete | Auth | #8 |
| 6 | Queue Management | `06-queue.md` | ✅ Complete | Playback | #9 |
| 7 | Device Switcher | `07-devices.md` | ✅ Complete | Playback | #10 |
| 8 | Stats Dashboard | `08-stats.md` | ✅ Complete | Auth | #11 |
| 9 | Playlist Manager | `09-playlists.md` | ✅ Complete | Library | #12 |
| 10 | Fix Library Display | `10-fix-library-display.md` | ✅ Complete | — | #15 |
| 11 | Fix Playback UX | `11-fix-playback-ux.md` | ✅ Complete | — | #16 |
| 12 | Fix Queue Overflow | `12-fix-queue-overflow.md` | ✅ Complete | — | #17 |
| 13 | Fix Devices Errors | `13-fix-devices-errors.md` | ✅ Complete | 18 | #19 |
| 14 | Fix Views Rendering | `14-fix-views-rendering.md` | ✅ Complete | 18 | #18 |
| 15 | Fix UX Polish | `15-fix-ux-polish.md` | ✅ Complete | — | #18 |
| 16 | Fix Search Results | `16-fix-search-results.md` | ✅ Complete | — | #20 |
| 17 | Fix Auth UX | `17-fix-auth-ux.md` | ✅ Complete | — | #21 |
| 18 | Fix Error Architecture | `18-fix-error-architecture.md` | ✅ Complete | — | #14 |
| 19 | P0 Correctness Fixes | `19-p0-correctness-fixes.md` | ✅ Complete | — | #24 |
| 20 | Elm Architecture Purity | `20-elm-architecture-purity.md` | ✅ Complete | 19 | #26 |
| 21 | Import Boundary Fixes | `21-import-boundary-fixes.md` | ✅ Partial | 19 | #29 |
| 22 | app.go Decomposition | `22-app-decomposition.md` | ✅ Complete | 20 | #27 |
| 23 | API Interfaces & Mocks | `23-api-interfaces-mocks.md` | ✅ Complete | — | #25 |
| 24 | Typed Errors & TokenProvider | `24-typed-errors-token-provider.md` | ✅ Partial | 23 | #30 |
| 25 | API DRY Refactoring | `25-api-dry-refactoring.md` | ✅ Complete | 23, 24 | #31 |
| 26 | View Height Enforcement | `26-view-height-enforcement.md` | ✅ Complete | — | #28 |
| 27 | Error Resilience | `27-error-resilience.md` | ✅ Complete | 24 | #32 |

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

## Bug Fix Execution Order

Bug fix features should be implemented in this order:

1. `18-fix-error-architecture.md` — Foundation: reusable error pattern (all other fixes depend on this)
2. `10-fix-library-display.md` — Quick, isolated fix
3. `11-fix-playback-ux.md` — Player UX: centering, icons, volume config, error feedback
4. `12-fix-queue-overflow.md` — Scroll + repeat label
5. `13-fix-devices-errors.md` — Error handling in overlay (depends on 18)
6. `14-fix-views-rendering.md` — Stats + Playlists view fixes (depends on 18)
7. `15-fix-ux-polish.md` — Status bar hints
8. `16-fix-search-results.md` — Full search pipeline investigation + fix
9. `17-fix-auth-ux.md` — Splash screen + auth TUI

---

## Architecture Improvement Execution Order

Improvements from the architecture review (2026-03-23). Implement in this order:

1. `19-p0-correctness-fixes.md` — Focus restoration, View() purity, dead code
2. `20-elm-architecture-purity.md` — Store mutation fixes, hardcoded hex (depends on 19)
3. `21-import-boundary-fixes.md` — Remove ui/→api/ imports (depends on 19)
4. `22-app-decomposition.md` — Split app.go into commands.go + render.go (depends on 20)
5. `23-api-interfaces-mocks.md` — Define interfaces, create mocks (independent)
6. `24-typed-errors-token-provider.md` — Typed errors, TokenProvider (depends on 23)
7. `25-api-dry-refactoring.md` — BaseClient, fetchAll, Get prefix (depends on 23, 24)
8. `26-view-height-enforcement.md` — Height capping for 3 panes (independent)
9. `27-error-resilience.md` — 401 retry, 429/403 extension (depends on 24)

> **Parallelism:** Features 19, 23, 26 have no dependencies and can start in parallel.
> After 19 completes, 20+21 can run in parallel. After 23, 24 can start.
> After 24, 25+27 can run in parallel.

---

## Versioning

| Version | Includes |
|---|---|
| v0.1.0 | Features 1 + 2 + 3 (theme system + auth + basic playback) |
| v0.2.0 | Features 4 + 5 (library + search) |
| v0.3.0 | Features 6 + 7 (queue + devices) |
| v0.4.0 | Feature 8 (stats) |
| v1.0.0 | Feature 9 (playlist manager) + polish |
| v1.1.0 | Features 10-18 (bug fixes + error architecture + UX polish) |
| v2.0.0 | Features 19-27 (architecture improvements from review) |

---

## Lessons & Future Improvement Ideas

Brief insights from implementing all 9 features:

**Architecture:**
- app.go is growing large (~700+ lines) — split into app_routing.go, app_commands.go, app_views.go
- Overlays (search, devices) share a pattern — extract a generic OverlayModel to reduce boilerplate
- Store mutex contention will increase with more pollers — consider event-based notifications
- Every new message type needs a case in app.go Update() — a message registry would scale better

**Performance:**
- Queue + playback state poll every 1s independently — batch into a single tick to halve API calls
- Search debounce is 300ms but API latency isn't tracked — add tracking to tune the delay
- StatsView fetches on first open — prefetch after auth to make `2` feel instant

**UX:**
- Never intercept single-char rune keys as action keys in overlays with text input — use Ctrl+key or arrows
- All non-KeyMsg messages must route to the active overlay — forgetting this silently breaks timers
- Status bar hints should always include overlay triggers (/, d) so users discover features
- Check lipgloss API availability (e.g. PlaceOverlay) before planning overlay rendering

**Testing:**
- Test helpers must accept *testing.T and call t.Helper() — nil masks failures
- sendKey-style helpers are reusable across pane tests — extract to a shared testutil package
- Coverage holds at ~81-82% across all features — focus new tests on edge cases, not line count

---

*Last updated: 2026-03-23*
