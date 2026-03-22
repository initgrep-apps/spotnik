# PRODUCT.md — Vision, Research & Baseline

> This document is the source of truth for *why* we are building Spotnik and *what* the product is.
> It is research-locked. Changes here require explicit decision to pivot, not casual edits.

---

## The One-Line Pitch

**Spotnik** is the Spotify client for developers who live in their terminal — keyboard-driven, fast, beautiful, and built to stay out of your way.

---

## Problem

Developers spend 8+ hours a day in a terminal. Music is essential to focus. The current options are:

- **Alt-tab to Spotify desktop app** — breaks flow, heavy Electron app (~300MB RAM)
- **Existing TUI clients (ncspot, spotify-player, spotify-tui)** — functional but ugly, no developer identity, replication of the Spotify app in ASCII
- **No solution at all** — use keyboard media keys and never see what's playing

None of these feel like they were built *for* developers. They feel like ports of a consumer app.

---

## Solution

A terminal music environment that feels as native to the developer's workspace as `lazygit` or `k9s`. Key design decisions:

- **Vim-style navigation** — developers already know these keys
- **Three-pane layout** — like a file manager (library / now playing / queue)
- **Your music as data** — surface statistics, listening history, patterns
- **Device-aware** — control any Spotify Connect device from the terminal
- **Config-driven theming** — Catppuccin, Nord, Dracula built-in
- **Single binary** — `brew install spotnik`, done

---

## Competitive Landscape

| Tool | Language | Status | What's Missing |
|---|---|---|---|
| spotify-tui | Rust | Unmaintained (2022) | Dead project, dated design |
| ncspot | Rust | Active | Basic ncurses look, no stats |
| spotify-player | Rust | Active | Feature-rich but design-last |

**Our differentiation:** Design-first, developer-identity-forward, data/stats layer, modern TUI aesthetic using Charmbracelet ecosystem.

---

## Target User

- Software developer (any stack)
- Uses terminal daily (tmux, vim/neovim, zsh/fish)
- Spotify Premium subscriber
- Appreciates tools with visual polish (thinks btop > top, lazygit > git CLI)
- Has 1-3 Spotify Connect devices (laptop, phone, smart speaker)

---

## Spotify Web API — Full Capability Inventory

### What We Have (Available Endpoints)

#### Player / Playback
| Endpoint | What It Does |
|---|---|
| `GET /me/player` | Full playback state: track, device, progress, shuffle, repeat, volume |
| `GET /me/player/currently-playing` | Currently playing item only |
| `GET /me/player/devices` | All available Spotify Connect devices |
| `GET /me/player/queue` | Current queue (what's coming up) |
| `GET /me/player/recently-played` | Last ~50 played tracks |
| `PUT /me/player/play` | Start or resume playback (specific track, album, playlist, or context) |
| `PUT /me/player/pause` | Pause |
| `POST /me/player/next` | Skip forward |
| `POST /me/player/previous` | Skip back |
| `PUT /me/player/seek` | Seek to position in ms |
| `PUT /me/player/volume` | Set volume 0-100 |
| `PUT /me/player/shuffle` | Toggle shuffle |
| `PUT /me/player/repeat` | Set repeat (off / context / track) |
| `PUT /me/player` | Transfer playback to a device |
| `POST /me/player/queue` | Add item to queue |

> **Important:** All player endpoints require **Spotify Premium**. Free users cannot use playback control. Show a clear error message on first run if Premium is not detected.

#### User & Library
| Endpoint | What It Does |
|---|---|
| `GET /me` | User profile (name, image, country, product tier) |
| `GET /me/top/tracks` | Top tracks (short_term / medium_term / long_term) |
| `GET /me/top/artists` | Top artists (short_term / medium_term / long_term) |
| `GET /me/following` | Followed artists |
| `GET /me/playlists` | User's playlists |
| `GET /me/albums` | Saved albums |
| `GET /me/tracks` | Liked songs |
| `PUT /me/tracks` | Save (like) a track |
| `DELETE /me/tracks` | Remove (unlike) a track |
| `GET /me/shows` | Saved podcasts |
| `GET /me/episodes` | Saved podcast episodes |
| `GET /me/audiobooks` | Saved audiobooks |

#### Playlists
| Endpoint | What It Does |
|---|---|
| `GET /playlists/{id}/items` | Tracks in a playlist (paginated) |
| `POST /me/playlists` | Create a new playlist |
| `PUT /playlists/{id}` | Rename / change description |
| `POST /playlists/{id}/items` | Add tracks |
| `DELETE /playlists/{id}/items` | Remove tracks |
| `PUT /playlists/{id}/items` | Reorder tracks |
| `PUT /playlists/{id}/followers` | Follow a playlist |
| `DELETE /playlists/{id}/followers` | Unfollow a playlist |

#### Search
| Endpoint | What It Does |
|---|---|
| `GET /search` | Search: track, album, artist, playlist, show, episode, audiobook |

#### Albums / Artists / Tracks
| Endpoint | What It Does |
|---|---|
| `GET /albums/{id}` | Album details + tracks |
| `GET /artists/{id}` | Artist details |
| `GET /artists/{id}/albums` | Artist's albums |

---

### What We Do NOT Have (Critical Constraints)

These endpoints are **gone as of 2024–2026** for Development Mode apps:

| Removed Endpoint | Impact on Spotnik |
|---|---|
| Audio features (BPM, energy, danceability, key) | No smart DJ, no "mood" filters, no audio analysis |
| Recommendations (`/recommendations`) | No "you might also like" feature |
| `GET /browse/new-releases` | No new releases browser |
| `GET /markets` | No market/region filtering |
| `GET /artists/{id}/top-tracks` | Cannot show artist's popular tracks |
| `GET /tracks` (bulk) | Must fetch tracks one at a time or use playlist endpoints |

> **Extended Quota Mode** restores these endpoints but requires applying to Spotify with a detailed use case. Apply when we have 25 real users actively using the app. Until then, design features only around available endpoints.

---

## OAuth Scopes Required

Request these scopes during auth. Do not request scopes you don't use.

```
user-read-playback-state
user-modify-playback-state
user-read-currently-playing
user-read-recently-played
user-top-read
user-library-read
user-library-modify
playlist-read-private
playlist-read-collaborative
playlist-modify-private
playlist-modify-public
user-follow-read
user-read-private
user-read-email
```

---

## API Rate Limiting

Spotify does not publish exact rate limits but the practical guidelines are:

- **Polling playback state**: max every 1 second
- **Search**: debounce 300ms, max 1 request per search
- **On 429**: read `Retry-After` header, wait that many seconds, show user feedback
- **On multiple 429s**: implement exponential backoff up to 30s

---

## Distribution Strategy

1. **Phase 1** (MVP): GitHub releases with pre-compiled binaries for:
   - `linux/amd64`, `linux/arm64`
   - `darwin/amd64`, `darwin/arm64` (Apple Silicon)
   - `windows/amd64`

2. **Phase 2** (Traction): Homebrew tap (`brew install spotnik`), AUR package

3. **Spotify Quota**: Dev mode supports 25 users. Apply for extended quota after reaching 25 active users with documented usage.

---

## Success Metrics (MVP)

- First `make run` to music playing: under 60 seconds (including auth)
- Playback state refresh latency: under 100ms UI response
- Binary size: under 15MB
- Memory usage: under 30MB resident
- GitHub stars: 100+ within 30 days of HN launch

---

## What Spotnik Is NOT

- Not a music discovery app (no recommendations engine)
- Not a social app (no sharing, no friend activity)
- Not a podcast manager (podcasts visible but not the focus)
- Not a free Spotify client (Premium required for all playback features)
- Not a last.fm scrobbler (out of scope)
- Not a lyrics app (out of scope)

---

*Last updated: 2026-02-21*
