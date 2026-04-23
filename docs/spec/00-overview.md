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
| 12 | CLI Output Renderer | `features/12-cli-output/` | in-progress | 146(done) 147–149 | Reusable `internal/cliout` package, typed message taxonomy, `docs/CLI-OUTPUT.md` reference, palette config, TTY-guarded spinner, validated prompt |


---

## Unresolved Issues

See `issues.md` for untriaged issues from PR reviews. Triage into feature stories when ready to fix.

---

*Last updated: 2026-04-23* (added feature 12: CLI output renderer)

