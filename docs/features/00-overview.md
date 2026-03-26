# Feature Overview — spotnik

> Feature index. Lists all features, status, dependencies, and links to specs.
> Read the spec file before implementing any feature.

---

## Features

| # | Feature | Spec | Status | Depends On | PR |
|---|---|---|---|---|---|
| 1 | Theme System | `01-theme-system.md` | ✅ Complete | — | — |
| 2 | Authentication | `02-auth.md` | ✅ Complete | — | — |
| 3 | Playback Controls | `03-playback.md` | ✅ Complete | 1, 2 | — |
| 4 | Library Browser | `04-library.md` | ✅ Complete | 2, 3 | #6 |
| 5 | Search | `05-search.md` | ✅ Complete | 2 | #8 |
| 6 | Queue Management | `06-queue.md` | ✅ Complete | 3 | #9 |
| 7 | Device Switcher | `07-devices.md` | ✅ Complete | 3 | #10 |
| 8 | Stats Dashboard | `08-stats.md` | ✅ Complete | 2 | #11 |
| 9 | Playlist Manager | `09-playlists.md` | ✅ Complete | 4 | #12 |
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
| 40 | Theme Enhancement | `40-theme-enhancement.md` | ✅ Complete | — | #45 |
| 41 | Layout Infrastructure | `41-layout-infrastructure.md` | ✅ Complete | — | #46 |
| 42 | Custom Border Renderer | `42-custom-border-renderer.md` | ✅ Complete | 40 | #47 |
| 43 | Reusable Components | `43-reusable-components.md` | ✅ Complete | 40 | #48 |
| 44 | Visualizer + Gradient Bars | `44-visualizer-gradient-bars.md` | ✅ Complete | 40 | #49 |
| 45 | NowPlaying Pane | `45-nowplaying-pane.md` | ✅ Complete | 41, 42, 44 | #50 |
| 46 | Queue Pane Migration | `46-queue-pane-migration.md` | ✅ Complete | 41, 43 | #51 |
| 47 | Library Split | `47-library-split.md` | ✅ Complete | 41, 43 | #52 |
| 48 | Stats Split | `48-stats-split.md` | ✅ Complete | 41, 43 | #53 |
| 49 | App Migration | `49-app-migration.md` | ✅ Complete | 40-48 | #54 |
| 50 | Header + Status Bar + Overlays | `50-header-statusbar-overlays.md` | ✅ Complete | 42, 49 | #55 |
| 51 | Page B: Nerd Status | `51-page-b-nerd-status.md` | ✅ Complete | 41-43, 49 | #56 |
| 52 | Mouse Scroll + Responsive | `52-mouse-scroll-responsive.md` | ✅ Complete | 41, 49 | #57 |
| 53 | Cleanup | `53-cleanup.md` | ✅ Complete | 40-52 | #58 |

---

## Versioning

| Version | Includes |
|---|---|
| v0.1.0 | Features 1-3 (theme + auth + playback) |
| v0.2.0 | Features 4-5 (library + search) |
| v0.3.0 | Features 6-7 (queue + devices) |
| v0.4.0 | Feature 8 (stats) |
| v1.0.0 | Feature 9 (playlist manager) |
| v1.1.0 | Features 10-18 (bug fixes + error architecture) |
| v2.0.0 | Features 19-27 (architecture improvements) |
| v3.0.0 | Features 29-33 (architecture baseline) |
| v3.1.0 | Features 34-39 (issues cleanup) |
| v4.0.0 | Features 40-53 (btop-inspired UI redesign) |

---

*Last updated: 2026-03-26*
