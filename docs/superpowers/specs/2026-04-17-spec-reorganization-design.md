# Spec Reorganization Design

**Date:** 2026-04-17  
**Status:** approved  

---

## Problem

The `docs/spec/features/` directory grew incrementally — each bug fix, polish pass, and architectural change spawned a new feature directory. After 27+ features and 128+ stories, the spec no longer reflects the actual shape of the product. Related stories are scattered across 5–6 directories, making it hard to reason about what the app does and what remains open.

---

## Goal

Collapse 27 fragmented spec features into 11 high-level features that reflect what Spotnik actually is. Stories keep their original numbers. Old feature directories are deleted after migration. `00-overview.md` is updated to reflect the new structure.

---

## Decisions

| Decision | Choice | Reason |
|---|---|---|
| Output | Analysis doc + physical file reorganization | User wants both |
| Story numbers | Preserved (no renumbering) | Avoids breaking git history and memory references |
| Old feature dirs | Deleted after migration | Clean break, no stubs or archives |
| New feature numbering | Start fresh from `01` | Old numbers go away with old dirs |
| Ordering | Foundation first, features next, CI/CD last | System layers underpin everything |

---

## New Feature Structure

### System / Foundation

| Dir | Title | Absorbs | Stories |
|---|---|---|---|
| `01-ui-foundation` | UI Layout & Components | `12-layout`, `21-help-overlay` | 15, 26, 41–44, 49, 50, 52–54, 108 |
| `02-api-infrastructure` | API Gateway & Reliability | `00-architecture`, `10-error-resilience`, `11-api-gateway`, `27-gateway-rate-protection` | 18–35, 37–39, 65, 126–127 |

### User-Facing Features

| Dir | Title | Absorbs | Stories |
|---|---|---|---|
| `03-playback` | Playback & NowPlaying | `03-playback`, `13-nowplaying`, `20-playback-context`, `24-controls-cleanup`, `25-nowplaying-controls-polish`, `26-playback-correctness` | 03, 11, 36, 45, 58–60, 105–107, 118–125 |
| `04-queue-and-devices` | Queue & Device Switching | `06-queue`, `07-devices` | 06, 07, 12, 13, 46 |
| `05-library` | Library Browser & Playlists | `04-library`, `09-playlists` | 04, 09, 10, 47 |
| `06-search` | Search | `05-search`, `19-search-redesign` | 05, 16, 81–104 |
| `07-stats` | Stats & Listening History | `08-stats` | 08, 14, 48, 55 |
| `08-theming` | Theming & Appearance | `01-theme`, `16-vivid-themes` | 01, 40, 70–75, 77–78, 79* |
| `09-auth-and-profile` | Auth, Bootstrap & User Profile | `02-auth`, `17-bootstrap`, `23-user-profile-subscription` | 02, 17, 76, 80, 114–117 |
| `10-developer-tools` | Developer Visibility (Page B) | `14-nerd-status`, `22-developer-foundations` | 51, 56, 61–69, 109–113 |

### DevOps

| Dir | Title | Absorbs | Stories |
|---|---|---|---|
| `11-cicd` | CI/CD & Release | `15-cicd` | 57 |

---

## Story Conflict

**Story 79** appears in both `16-vivid-themes` and `17-bootstrap` in `00-overview.md`. During migration, place it in `08-theming` (it relates to theme preference persistence, which is a theming concern). Remove it from the bootstrap story list.

---

## Migration Steps

1. Create 11 new feature directories under `docs/spec/features/`
2. For each new feature, write a `feature.md` with consolidated title, status, and story list
3. For each old feature, move its story files into the new feature dir that absorbs it
4. Delete all old feature directories
5. Update `docs/spec/00-overview.md` with the new 11-feature table
6. Commit

---

## New `00-overview.md` Feature Table (after migration)

| # | Feature | Status | Stories | Description |
|---|---------|--------|---------|-------------|
| 01 | UI Layout & Components | done | 15, 26, 41–44, 49, 50, 52–54, 108 | Grid layout manager, btop borders, reusable components, help overlay |
| 02 | API Gateway & Reliability | done | 18–35, 37–39, 65, 126–127 | Centralized gateway, rate limiting, error types, token bucket, architecture health |
| 03 | Playback & NowPlaying | in-progress | 03, 11, 36, 45, 58–60, 105–107, 118–125 | Transport controls, NowPlaying display, visualizer, context-aware playback, polish |
| 04 | Queue & Device Switching | in-progress | 06, 07, 12, 13, 46 | Queue viewer pane, Spotify Connect device selection |
| 05 | Library Browser & Playlists | in-progress | 04, 09, 10, 47 | Browse playlists/albums/liked songs, full playlist management |
| 06 | Search | in-progress | 05, 16, 81–104 | Full-screen search overlay, multi-tab results, prefix autocomplete, pagination |
| 07 | Stats & Listening History | in-progress | 08, 14, 48, 55 | Top tracks, top artists, recently played |
| 08 | Theming & Appearance | done | 01, 40, 70–75, 77–79 | Token-based themes, TOML config themes, runtime switcher, 11 built-in themes |
| 09 | Auth, Bootstrap & User Profile | in-progress | 02, 17, 76, 80, 114–117 | PKCE OAuth, token refresh, first-launch bootstrap, profile overlay, Premium gating |
| 10 | Developer Visibility (Page B) | done | 51, 56, 61–69, 109–113 | Request flow pane, network log, developer foundations |
| 11 | CI/CD & Release | open | 57 | GitHub Actions, GoReleaser, multi-platform distribution |

---

## Open Stories After Migration

Stories not yet done, grouped by new feature:

| New Feature | Open Stories |
|---|---|
| `02-api-infrastructure` | 21, 22, 34, 35 (architecture health) |
| `03-playback` | 11, 36 |
| `04-queue-and-devices` | 12, 13 |
| `05-library` | 10 |
| `06-search` | 16 |
| `07-stats` | 55 |
| `09-auth-and-profile` | 17 |
| `11-cicd` | 57 |
