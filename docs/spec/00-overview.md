# Spec Overview — Spotnik

> Each feature lives in its own directory with a `feature.md` and individual story files.
> Issues from PR reviews are tracked in `issues.md` at the spec root.

---

## Features

| # | Feature | Path | Status | Stories | Description |
|---|---------|------|--------|---------|-------------|
| 01 | UI Layout & Components | `features/01-ui-foundation/` | done | 15, 26, 41–44, 49, 50, 52–54, 108 | Grid layout manager, btop borders, reusable table/filter components, help overlay |
| 02 | API Gateway & Reliability | `features/02-api-infrastructure/` | done | 18–35, 37–39, 65, 126–127 | Centralized gateway, rate limiting, dedup, error types |
| 03 | Playback & NowPlaying | `features/03-playback/` | done | 03, 11, 36, 45, 58–60, 105–107, 118–125, 197–198, 205, 224 | Transport controls, NowPlaying display, visualizer, interactive seek bar |
| 04 | Queue & Device Switching | `features/04-queue-and-devices/` | done | 06, 07, 12, 13, 46 | Queue viewer pane, Spotify Connect device selection |
| 05 | Library Browser & Playlists | `features/05-library/` | done | 04, 09, 10, 47 | Browse playlists/albums/liked songs, full playlist management |
| 06 | Search | `features/06-search/` | done | 05, 16, 81–104, 212–213 | Full-screen overlay, multi-tab results, prefix autocomplete, pagination |
| 07 | Stats & Listening History | `features/07-stats/` | done | 08, 14, 48, 55 | Top tracks, top artists, recently played with time-range cycling |
| 08 | Theming & Appearance | `features/08-theming/` | done | 01, 40, 70–75, 77–79 | Token-based themes, TOML config, runtime switcher, 11 built-in themes |
| 09 | Auth, Bootstrap & User Profile | `features/09-auth-and-profile/` | done | 02, 17, 76, 79, 80, 114–117, 134–145, 196, 204, 209 | PKCE OAuth, config-first client ID, TUI onboarding, auth CLI subcommands, profile overlay |
| 10 | Developer Visibility (Stats page) | `features/10-developer-tools/` | done | 51, 56, 61–69, 109–113 | Request flow pane, network log, developer foundations |
| 11 | CI/CD & Release | `features/11-cicd/` | done | 57, 128–133, 194 | GitHub Actions, GoReleaser, release-please, multi-platform distribution, curl/PS1 installer scripts |
| 12 | CLI Output Renderer | `features/12-cli-output/` | done | 146–149 | `internal/cliout` package, typed message taxonomy, palette config, TTY-guarded spinner, validated prompt |
| 13 | TUI Design System | `features/13-tui-design-system/` | done | 150–172, 183–193 | `internal/uikit` package (18 primitives), frozen glyph catalogue with ASCII fallback, role-to-token matrix, glyph-fallback CI guards |
| 14 | Stats Page Redesign | `features/14-page-b-redesign/` | done | 173–182, 210, 211 | Stacked Stats page layout, GatewayHealth/PollingTraffic/GatewayLive panes, universal filter border + Esc-clear, TableBasedPane consolidation |
| 15 | Error Resilience & Universal Polling | `features/15-error-resilience/` | done | 199 ✓, 200 ✓, 201 ✓, 202 ✓, 203 ✓, 206 | Universal tick-driven polling for all data panes, per-pane exponential backoff, silent failure fixes, overlay self-sufficiency, auth error mapping |
| 16 | Mono Themes + Page Rename | `features/16-mono-themes-page-rename/` | done | 207 ✓, 208 ✓ | Rename Page A/B → Music/Stats, add mono-dark/mono-light themes |
| 17 | Album Art & Responsive NowPlaying | `features/17-album-art/` | in-progress | 214 ✓, 215 ✓, 216 ✓, 217 ✓, 218 ✓, 219 ✓, 220 ✓, 221 ✓, 222 ✓, 223 | Phase 1 (shipped): pixterm album art, responsive layout, LayoutManager MinHeight, single-formula layout. Phase 2 (shipped): OverlayBackground token + InfoBox fill (221); remove album art, overlay InfoBox on full-pane visualizer (222). Phase 3 (open): Fix layout — adaptive width, remove solid bg, centering, compact preset MinHeight (223) |

---

## Unresolved Issues

See `issues.md` for untriaged issues from PR reviews. Triage into feature stories when ready to fix.

---

*Last updated: 2026-06-11 — story 224 done, feature 03 done*
