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
| 28 | API Cleanup Follow-up | `28-api-cleanup-followup.md` | ✅ Complete | 21, 24, 25 | #33 |
| 29 | Elm Purity: Data-Carrying Msgs | `29-elm-purity-data-carrying-msgs.md` | ✅ Complete | — | #34 |
| 30 | API Gateway | `30-api-gateway.md` | ✅ Complete | — | #35 |
| 31 | Notifications + Error Routing | `31-notifications-error-routing.md` | ✅ Complete | 29 | #36 |
| 32 | Staleness Tracking | `32-staleness-tracking.md` | ✅ Complete | 29 | #37 |
| 33 | Idle Polling Backoff | `33-idle-polling-backoff.md` | ✅ Complete | 29 | #38 |
| 34 | Docs, Dead Code & Init | `34-docs-dead-code-init.md` | ✅ Complete | — | #39 |
| 35 | Type Design Alignment | `35-type-design-alignment.md` | ✅ Complete | 34 | #40 |
| 36 | Command Safety & Errors | `36-command-safety-errors.md` | ✅ Complete | 35 | #41 |
| 37 | Gateway Hardening | `37-gateway-hardening.md` | ✅ Complete | — | #42 |
| 38 | Notification & Staleness Hardening | `38-notification-staleness-hardening.md` | ✅ Complete | 36 | #43 |
| 39 | Idle Polish & Test Gaps | `39-idle-polish-test-gaps.md` | ✅ Complete | 38 | #44 |

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

## Architecture Baseline Execution Order

Architecture baseline features from gap analysis (2026-03-24). See
`docs/superpowers/plans/2026-03-24-architecture-baseline.md` for the full analysis.

```
F29: Elm Purity ──────┬──→ F31: Notifications + Error Routing
                      ├──→ F32: Staleness Tracking
                      └──→ F33: Idle Polling Backoff

F30: API Gateway ─────────→ (independent, can parallel with F29)
```

1. `29-elm-purity-data-carrying-msgs.md` — Data-carrying Msgs, Store writes only in Update() (independent)
2. `30-api-gateway.md` — Rate limiting, dedup, concurrency, priority (independent, parallel with 29)
3. `31-notifications-error-routing.md` — BubbleUp toasts, replace statusMsg, error routing (depends on 29)
4. `32-staleness-tracking.md` — fetchedAt timestamps, TTL-based refresh (depends on 29)
5. `33-idle-polling-backoff.md` — Adaptive polling intervals based on idle/playback state (depends on 29)

> **Parallelism:** Features 29 and 30 have no dependencies and can start in parallel.
> After 29 completes, 31+32+33 can run in parallel.

---

## Issues Cleanup Execution Order

Fixes from PR reviews of features 29-33 (2026-03-25). See
`docs/superpowers/plans/2026-03-25-issues-cleanup.md` for the full analysis.

```
F34: Docs/Init ──→ F35: Type Alignment ──→ F36: Command Safety ──→ F38: Notif/Staleness ──→ F39: Idle/Tests

F37: Gateway Hardening ──→ (independent, can parallel with F34-F36)
```

1. `34-docs-dead-code-init.md` — Stale docs, dead unmarshalJSON, statsFetchedAt init (independent)
2. `35-type-design-alignment.md` — SearchResult→domain, StatsLoadedMsg move, AlbumsLoadedMsg Offset, DevicesLoadedMsg export (depends on 34)
3. `36-command-safety-errors.md` — Data race fix, nil-client errors, playback error toast (depends on 35)
4. `37-gateway-hardening.md` — Thread safety, timer leaks, nil response, 429 cleanup (independent)
5. `38-notification-staleness-hardening.md` — Alert safety, fetchedAt guards, stats stamp, TOCTOU, cached data (depends on 36)
6. `39-idle-polish-test-gaps.md` — WindowSizeMsg idle, backoff toast, nil state, test gaps (depends on 38)

> **Parallelism:** Features 34 and 37 have no dependencies and can start in parallel.
> After 34, 35→36→38→39 are sequential. However, all features should be implemented sequentially per project rules.

---

## UI Redesign Execution Order

Features from the btop-inspired UI redesign (2026-03-26). See `docs/DESIGN.md` for
the full specification. All features are sequential per project rules.

```
F40 (Theme) ──┐
F41 (Layout) ─┤
F42 (Border) ─┼──→ F45 (NowPlaying) ──┐
F43 (Components)┘   F46 (Queue) ──────┤
                     F47 (LibSplit) ───┼──→ F49 (App Migration) ──→ F50 (Header/Overlay)
F44 (Visualizer) → F45                ┤                             F51 (Page B)
                     F48 (StatsSplit) ─┘                            F52 (Mouse)
                                                                    F53 (Cleanup)
```

| # | Feature | Spec | Status | Depends On | PR |
|---|---------|------|--------|-----------|-----|
| 40 | Theme Enhancement | `40-theme-enhancement.md` | | — | |
| 41 | Layout Infrastructure | `41-layout-infrastructure.md` | | — | |
| 42 | Custom Border Renderer | `42-custom-border-renderer.md` | | 40 | |
| 43 | Reusable Components | `43-reusable-components.md` | | 40 | |
| 44 | Visualizer + Gradient Bars | `44-visualizer-gradient-bars.md` | | 40 | |
| 45 | NowPlaying Pane | `45-nowplaying-pane.md` | | 41,42,44 | |
| 46 | Queue Pane Migration | `46-queue-pane-migration.md` | | 41,43 | |
| 47 | Library Split | `47-library-split.md` | | 41,43 | |
| 48 | Stats Split | `48-stats-split.md` | | 41,43 | |
| 49 | App Migration | `49-app-migration.md` | | 40-48 | |
| 50 | Header + Status Bar + Overlays | `50-header-statusbar-overlays.md` | | 42,49 | |
| 51 | Page B: Nerd Status | `51-page-b-nerd-status.md` | | 41-43,49 | |
| 52 | Mouse Scroll + Responsive | `52-mouse-scroll-responsive.md` | | 41,49 | |
| 53 | Cleanup | `53-cleanup.md` | | 40-52 | |

> **New dependencies:** `github.com/evertras/bubble-table` (dense tables), `github.com/rmhubbert/bubbletea-overlay` (overlay compositing). Both approved by owner 2026-03-26.

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
| v3.0.0 | Features 29-33 (architecture baseline: Elm purity, gateway, notifications, staleness, idle backoff) |
| v3.1.0 | Features 34-39 (issues cleanup: docs, types, safety, gateway, staleness, tests) |
| v4.0.0 | Features 40-53 (btop-inspired UI redesign: grid layout, 10 panes, 2 pages, presets) |

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

*Last updated: 2026-03-26*
