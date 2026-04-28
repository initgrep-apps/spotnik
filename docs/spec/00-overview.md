# Spec Overview — Spotnik

> Each feature lives in its own directory with a `feature.md` and individual story files.
> Issues from PR reviews are tracked in `issues.md` at the spec root.

---

## Features

| # | Feature | Path | Status | Stories | Description |
|---|---------|------|--------|---------|-------------|
| 01 | UI Layout & Components | `features/01-ui-foundation/` | done | 15, 26, 41–44, 49, 50, 52–54, 108 | Grid layout manager, btop borders, reusable table/filter components, help overlay |
| 02 | API Gateway & Reliability | `features/02-api-infrastructure/` | in-progress | 18–35, 37–39, 65, 126–127 | Centralized gateway, rate limiting, dedup, error types, architecture health |
| 03 | Playback & NowPlaying | `features/03-playback/` | in-progress | 03, 11, 36, 45, 58–60, 105–107, 118–125 | Transport controls, NowPlaying display, visualizer, context-aware playback, polish |
| 04 | Queue & Device Switching | `features/04-queue-and-devices/` | in-progress | 06, 07, 12, 13, 46 | Queue viewer pane, Spotify Connect device selection |
| 05 | Library Browser & Playlists | `features/05-library/` | in-progress | 04, 09, 10, 47 | Browse playlists/albums/liked songs, full playlist management |
| 06 | Search | `features/06-search/` | in-progress | 05, 16, 81–104 | Full-screen overlay, multi-tab results, prefix autocomplete, pagination |
| 07 | Stats & Listening History | `features/07-stats/` | in-progress | 08, 14, 48, 55 | Top tracks, top artists, recently played with time-range cycling |
| 08 | Theming & Appearance | `features/08-theming/` | done | 01, 40, 70–75, 77–79 | Token-based themes, TOML config, runtime switcher, 11 built-in themes |
| 09 | Auth, Bootstrap & User Profile | `features/09-auth-and-profile/` | done | 02, 17, 76, 79*, 80, 114–117, 134–140, 141–145 | PKCE OAuth, token refresh, config-first client ID, TUI onboarding flow, auth CLI subcommands, profile logout/forget, UX polish |
| 10 | Developer Visibility (Page B) | `features/10-developer-tools/` | done | 51, 56, 61–69, 109–113 | Request flow pane, network log, developer foundations |
| 11 | CI/CD & Release | `features/11-cicd/` | done | 57, 128–133 | GitHub Actions, GoReleaser, release-please, version injection, multi-platform distribution |
| 12 | CLI Output Renderer | `features/12-cli-output/` | done | 146–149 | Reusable `internal/cliout` package, typed message taxonomy, `docs/CLI-OUTPUT.md` reference, palette config, TTY-guarded spinner, validated prompt |
| 13 | TUI Design System | `features/13-tui-design-system/` | done | 150–172 | `internal/uikit` package (18 primitives: PaneChrome, OverlayChrome, Panel, TableChrome, ListRow, LockedRow, SectionLabel, EmptyState, URLBox, HeaderBar, StatusBar, KeyBar, Chip, FormField, Toast, StatusGlyph, ProgressBar, Spinner), frozen glyph catalogue with ascii fallback, role-to-token matrix, `docs/TUI-DESIGN-SYSTEM.md` reference, `⚠`→`◬` swap across cliout + TUI, `ᐅ` removal; 169–172 post-testing regression fixes all merged |
| 14 | Page B Redesign (Nerd Status) | `features/14-page-b-redesign/` | done | 173–182 | Universal Esc scroll-reset across all table panes; replace RequestFlowPane with GatewayHealthPane, PollingTrafficPane, GatewayLivePane; fix NetworkLog decision cross-tick bug; update Page B preset grid; universal filter border label + Esc-clear on all 8 filterable panes (178); Page B toggle keys '1'-'5' (179); stacked 30/70 layout with RowSpan support (180); structural refactor — TableBasedPane consolidates filter routing for 9 panes, graded `f(query)` border shrink, close-notch retired, flat Page B layout, RowSpan deleted (181); GatewayLive multi-column refactor — drop uikit.ListRow, per-row glyph via bubble-table StyledCell, restore full-row selection bg + column padding parity (182) |


---

## Unresolved Issues

See `issues.md` for untriaged issues from PR reviews. Triage into feature stories when ready to fix.

---

*Last updated: 2026-04-28* (feature 14 done: stories 173–182, with story 182 retiring uikit.ListRow inside GatewayLivePane and routing per-row glyph colour through bubble-table StyledCell so row-highlight and column padding match the other eight TableBasedPane consumers)

