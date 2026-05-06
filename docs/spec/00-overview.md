# Spec Overview — Spotnik

> Each feature lives in its own directory with a `feature.md` and individual story files.
> Issues from PR reviews are tracked in `issues.md` at the spec root.

---

## Features

| # | Feature | Path | Status | Stories | Description |
|---|---------|------|--------|---------|-------------|
| 01 | UI Layout & Components | `features/01-ui-foundation/` | done | 15, 26, 41–44, 49, 50, 52–54, 108 | Grid layout manager, btop borders, reusable table/filter components, help overlay |
| 02 | API Gateway & Reliability | `features/02-api-infrastructure/` | done | 18–35, 37–39, 65, 126–127 | Centralized gateway, rate limiting, dedup, error types |
| 03 | Playback & NowPlaying | `features/03-playback/` | done | 03, 11, 36, 45, 58–60, 105–107, 118–125 | Transport controls, NowPlaying display, visualizer |
| 04 | Queue & Device Switching | `features/04-queue-and-devices/` | done | 06, 07, 12, 13, 46 | Queue viewer pane, Spotify Connect device selection |
| 05 | Library Browser & Playlists | `features/05-library/` | done | 04, 09, 10, 47 | Browse playlists/albums/liked songs, full playlist management |
| 06 | Search | `features/06-search/` | done | 05, 16, 81–104 | Full-screen overlay, multi-tab results, prefix autocomplete, pagination |
| 07 | Stats & Listening History | `features/07-stats/` | done | 08, 14, 48, 55 | Top tracks, top artists, recently played with time-range cycling |
| 08 | Theming & Appearance | `features/08-theming/` | done | 01, 40, 70–75, 77–79 | Token-based themes, TOML config, runtime switcher, 11 built-in themes |
| 09 | Auth, Bootstrap & User Profile | `features/09-auth-and-profile/` | in-progress | 02, 17, 76, 79, 80, 114–117, 134–145, 196 | PKCE OAuth, config-first client ID, TUI onboarding, auth CLI subcommands, profile overlay |
| 10 | Developer Visibility (Page B) | `features/10-developer-tools/` | done | 51, 56, 61–69, 109–113 | Request flow pane, network log, developer foundations |
| 11 | CI/CD & Release | `features/11-cicd/` | done | 57, 128–133, 194 | GitHub Actions, GoReleaser, release-please, multi-platform distribution, curl/PS1 installer scripts |
| 12 | CLI Output Renderer | `features/12-cli-output/` | done | 146–149 | `internal/cliout` package, typed message taxonomy, palette config, TTY-guarded spinner, validated prompt |
| 13 | TUI Design System | `features/13-tui-design-system/` | done | 150–172, 183–193 | `internal/uikit` package (18 primitives), frozen glyph catalogue with ASCII fallback, role-to-token matrix, glyph-fallback CI guards |
| 14 | Page B Redesign (Nerd Status) | `features/14-page-b-redesign/` | done | 173–182 | Stacked Page B layout, GatewayHealth/PollingTraffic/GatewayLive panes, universal filter border + Esc-clear, TableBasedPane consolidation |

---

## Unresolved Issues

See `issues.md` for untriaged issues from PR reviews. Triage into feature stories when ready to fix.

---

*Last updated: 2026-05-07*
