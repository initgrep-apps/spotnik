# Spec Overview — Spotnik

> Consolidated feature specs and issue/bugfix specs.
> Each feature file contains one or more stories with acceptance criteria, tasks, and tests.
> Original spec files are preserved in `archive/` for historical reference.

---

## Features (15 consolidated specs)

| # | Feature | Spec | Status | Stories | Description |
|---|---------|------|--------|---------|-------------|
| 01 | Theme System | `features/01-theme.md` | done | 01, 40 | Token-based color theming with five built-in themes and 16 extended tokens |
| 02 | Authentication | `features/02-auth.md` | done | 02 | PKCE OAuth flow, automatic token refresh, keychain credential storage |
| 03 | Playback Controls | `features/03-playback.md` | done | 03 | Player polling, transport controls, progress/volume, state management |
| 04 | Library Browser | `features/04-library.md` | done | 04, 47 | Browse playlists/albums/liked songs, split into dedicated panes |
| 05 | Search | `features/05-search.md` | done | 05 | Keyboard-native search overlay with debounced live results |
| 06 | Queue Management | `features/06-queue.md` | done | 06, 46 | Queue viewer pane with bubble-table and layout.Pane interface |
| 07 | Device Switcher | `features/07-devices.md` | done | 07 | Spotify Connect device selection overlay |
| 08 | Stats Dashboard | `features/08-stats.md` | done | 08, 48 | Top tracks, top artists, recently played — split into dedicated panes |
| 09 | Playlist Manager | `features/09-playlists.md` | done | 09 | Create, rename, reorder, delete playlists |
| 10 | Error Resilience | `features/10-error-resilience.md` | done | 27 | 401 token refresh, 429/403 backoff across all API calls |
| 11 | API Gateway | `features/11-api-gateway.md` | done | 29-33 | Data-carrying messages, centralized gateway, notifications, staleness, idle backoff |
| 12 | Layout System | `features/12-layout.md` | done | 41-44, 49-50, 52 | Grid layout manager, btop borders, reusable components, responsive design |
| 13 | NowPlaying | `features/13-nowplaying.md` | done | 45, 58-60 | Real-time playback display with visualizer engine and split layout |
| 14 | Nerd Status | `features/14-nerd-status.md` | done | 51, 61-62, 64, 66-69 | Page B developer visibility: request flow, network log, gateway events |
| 15 | CI/CD | `features/15-cicd.md` | open | 57 | GitHub Actions, GoReleaser, multi-platform distribution |

---

## Issues / Bugfixes (30 individual specs)

| # | Name | Spec | Category | Description |
|---|------|------|----------|-------------|
| 10 | Fix Library Display | `issues/10-fix-library-display.md` | Bugfix | Library pane showing only headers, not items |
| 11 | Fix Playback UX | `issues/11-fix-playback-ux.md` | Bugfix | Volume errors, centering, emoji icons |
| 12 | Fix Queue Overflow | `issues/12-fix-queue-overflow.md` | Bugfix | Queue overflow — add scrolling support |
| 13 | Fix Devices Errors | `issues/13-fix-devices-errors.md` | Bugfix | Device overlay showing no devices on API errors |
| 14 | Fix Views Rendering | `issues/14-fix-views-rendering.md` | Bugfix | Stats/playlist views showing empty data |
| 15 | Fix UX Polish | `issues/15-fix-ux-polish.md` | Bugfix | Missing view switcher hints in status bar |
| 16 | Fix Search Results | `issues/16-fix-search-results.md` | Bugfix | Search overlay not showing results |
| 17 | Fix Auth UX | `issues/17-fix-auth-ux.md` | Bugfix | Auth screen not TUI-styled, URL overflow |
| 18 | Fix Error Architecture | `issues/18-fix-error-architecture.md` | Bugfix | Silent API error swallowing across features |
| 19 | P0 Correctness Fixes | `issues/19-p0-correctness-fixes.md` | Bugfix | Overlay focus restoration, Elm purity violations |
| 20 | Elm Architecture Purity | `issues/20-elm-architecture-purity.md` | Refactor | Elm Architecture violations in commands/store |
| 21 | Import Boundary Fixes | `issues/21-import-boundary-fixes.md` | Refactor | Remove ui->api import boundary violations |
| 22 | app.go Decomposition | `issues/22-app-decomposition.md` | Refactor | Split 1730-line app.go into focused files |
| 23 | API Interfaces & Mocks | `issues/23-api-interfaces-mocks.md` | Refactor | SpotifyClient interface, remove nil-guards |
| 24 | Typed Errors & TokenProvider | `issues/24-typed-errors-token-provider.md` | Refactor | Replace string error matching with typed errors |
| 25 | API DRY Refactoring | `issues/25-api-dry-refactoring.md` | Refactor | Shared HTTP helpers, pagination, naming |
| 26 | View Height Enforcement | `issues/26-view-height-enforcement.md` | Refactor | Height-capped rendering for unbounded panes |
| 28 | API Cleanup Follow-up | `issues/28-api-cleanup-followup.md` | Refactor | Deferred items from features 21, 24, 25 |
| 34 | Docs, Dead Code & Init | `issues/34-docs-dead-code-init.md` | Cleanup | Stale docs, dead code removal, defensive init |
| 35 | Type Design Alignment | `issues/35-type-design-alignment.md` | Cleanup | Message type design consistency |
| 36 | Command Safety & Errors | `issues/36-command-safety-errors.md` | Bugfix | Data race in playback commands, error propagation |
| 37 | Gateway Hardening | `issues/37-gateway-hardening.md` | Bugfix | Thread safety, resource leak, panic fixes |
| 38 | Notification & Staleness Hardening | `issues/38-notification-staleness-hardening.md` | Bugfix | Assertion safety fixes in notifications |
| 39 | Idle Polish & Test Gaps | `issues/39-idle-polish-test-gaps.md` | Cleanup | Idle backoff polish, test coverage gaps |
| 53 | Cleanup | `issues/53-cleanup.md` | Cleanup | Dead code from old layout, doc updates |
| 54 | Fix Table Alignment | `issues/54-fix-table-alignment.md` | Bugfix | Tables rendering right-aligned instead of left |
| 55 | Fix Recently Played Empty | `issues/55-fix-recently-played-empty.md` | Bugfix | Recently Played pane empty with no feedback |
| 56 | Fix Request Flow Data | `issues/56-fix-request-flow-data.md` | Bugfix | Request Flow showing static data |
| 63 | Request Flow Boxed Guards | `issues/63-requestflow-boxed-guards.md` | Bugfix | Defensive guards for boxed layout |
| 65 | Gateway-Internal Watermarks | `issues/65-gateway-internal-watermarks.md` | Refactor | Move watermark tracking into Gateway |

---

## Known Issues & Tech Debt

See `issues/00-known-issues-tech-debt.md` for tracked issues from PR reviews that are non-blocking but should be addressed.

---

*Last updated: 2026-03-30*
