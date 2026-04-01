# Spec Overview — Spotnik

> Each feature lives in its own directory with a `feature.md` and individual story files.
> Issues from PR reviews are tracked in `issues.md` at the spec root.

---

## Features

| # | Feature | Path | Status | Stories | Description |
|---|---------|------|--------|---------|-------------|
| 00 | Architecture | `features/00-architecture/` | done | 21, 22, 34, 35 | Cross-cutting codebase health: import boundaries, app decomposition, dead code, type alignment |
| 01 | Theme System | `features/01-theme/` | done | 01, 40 | Token-based color theming with five built-in themes and 16 extended tokens |
| 02 | Authentication | `features/02-auth/` | done | 02, 17 | PKCE OAuth flow, token refresh, keychain storage, auth UX |
| 03 | Playback Controls | `features/03-playback/` | done | 03, 11, 36 | Player polling, transport controls, progress/volume, command safety |
| 04 | Library Browser | `features/04-library/` | done | 04, 10, 47 | Browse playlists/albums/liked songs, split into dedicated panes |
| 05 | Search | `features/05-search/` | done | 05, 16 | Keyboard-native search overlay with debounced live results |
| 06 | Queue Management | `features/06-queue/` | done | 06, 12, 46 | Queue viewer pane with bubble-table and layout.Pane interface |
| 07 | Device Switcher | `features/07-devices/` | done | 07, 13 | Spotify Connect device selection overlay |
| 08 | Stats Dashboard | `features/08-stats/` | done | 08, 14, 48, 55 | Top tracks, top artists, recently played — split into dedicated panes |
| 09 | Playlist Manager | `features/09-playlists/` | done | 09 | Create, rename, reorder, delete playlists |
| 10 | Error Resilience | `features/10-error-resilience/` | done | 18, 19, 24, 27 | Token refresh, rate limiting, typed errors, correctness fixes |
| 11 | API Gateway | `features/11-api-gateway/` | done | 20, 23, 25, 28, 29-33, 37-39, 65 | Data-carrying messages, centralized gateway, notifications, staleness, idle backoff |
| 12 | Layout System | `features/12-layout/` | done | 15, 26, 41-44, 49-50, 52-54 | Grid layout manager, btop borders, reusable components, responsive design |
| 13 | NowPlaying | `features/13-nowplaying/` | done | 45, 58-60 | Real-time playback display with visualizer engine and split layout |
| 14 | Nerd Status | `features/14-nerd-status/` | done | 51, 56, 61-64, 66-69 | Page B developer visibility: request flow, network log, gateway events |
| 15 | CI/CD | `features/15-cicd/` | open | 57 | GitHub Actions, GoReleaser, multi-platform distribution |
| 16 | Vivid Theme System | `features/16-vivid-themes/` | in-progress | 70, 71, 72, 73, 74, 75, 77, 78, 79 | Config-driven TOML themes, colorful borders, per-column colors, 6 new themes, runtime switcher overlay |
| 17 | Bootstrap | `features/17-bootstrap/` | done | 76, 79, 80 | First-launch config bootstrap, embedded client ID, preference persistence engine |
| 18 | Search Overlay Redesign | `features/18-search-redesign/` | done | 81, 82 | Wide tabbed search overlay with rich metadata columns, per-tab theme colors, contextual help bar |

---

## Unresolved Issues

See `issues.md` for untriaged issues from PR reviews. Triage into feature stories when ready to fix.

---

*Last updated: 2026-04-01*
