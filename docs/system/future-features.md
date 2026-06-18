# Future Features — Spotnik

> Status: analysis complete | 2026-06-18
>
> Based on Spotify Web API OpenAPI Schema v1.0.0.
> Only active (non-deprecated) endpoints. 48 active endpoints available.

---

## Active API Surface

| Category | Endpoints |
|---|---|
| **Playback** | get state, play, pause, next, prev, seek, volume, shuffle, repeat, queue (get+add), transfer, devices, currently-playing, recently-played |
| **Library** | `PUT/DELETE/GET /me/library` (unified save/remove/check), `GET /me/tracks`, `GET /me/albums`, `GET /me/shows`, `GET /me/episodes`, `GET /me/audiobooks`, `GET /me/playlists`, `GET /me/following` |
| **Playlists** | create, get, get items, add items, remove items, reorder items, change details, get/upload cover image |
| **Catalog Lookup** | get track, get album + tracks, get artist + albums, get show + episodes, get episode, get audiobook + chapters, get chapter |
| **Search** | search (all types) |
| **User** | get current user profile, get top items |
| **Audiobooks** | get audiobook, get chapters, get chapter, saved audiobooks |

---

## Feature Candidates

### TIER 1 — Very Low Effort (existing patterns, small UI)

| # | Feature | Endpoint | Notes |
|---|---------|----------|-------|
| 1 | **Save/Unsave track from NowPlaying** | `PUT/DELETE /me/library` | Add keybinding (`Ctrl+S`) + heart indicator. Unified library endpoint handles any URI type. |
| 2 | **Check item saved status** | `GET /me/library/contains?uris=...` | Unified check for any URI type. Show heart/bookmark indicator on current item. |
| 3 | **Create playlist from selection** | `POST /me/playlists` | API client exists. Need: "Save as playlist..." prompt overlay from search/queue. |
| 4 | **Rename playlist** | `PUT /playlists/{id}` | API client exists. Need: inline edit or prompt overlay in Playlists pane. |
| 5 | **Remove track from playlist** | `DELETE /playlists/{id}/items` | API client exists. Wire `x` key (currently no-op). |
| 6 | **Reorder playlist items** | `PUT /playlists/{id}/items` | API client exists. Cut/paste or `Ctrl+Up/Down` keybindings. |
| 7 | **Add track to playlist** | `POST /playlists/{id}/items` | API client exists. "Add to playlist" overlay with playlist picker. |
| 8 | **Save/Unsave album** | `PUT/DELETE /me/library` | Keybinding in Albums pane. Save by album URI. |
| 9 | **Save/Unsave show** | `PUT/DELETE /me/library` | Keybinding in FollowedShows pane. |
| 10 | **Save/Unsave episode** | `PUT/DELETE /me/library` | Keybinding in episode lists. |
| 11 | **Get Followed Artists** | `GET /me/following?type=artist` | New pane or section: artists you follow (different from Top Artists). |

**11 features.** ~3-5 days.

---

### TIER 2 — Medium Effort (new API clients, follows existing design patterns)

| # | Feature | Endpoint | Notes |
|---|---------|----------|-------|
| 12 | **Artist Detail overlay** | `GET /artists/{id}` | Genres, popularity, follower count, images. Trigger from NowPlaying artist, search results, top artists. |
| 13 | **Artist's Albums drill-down** | `GET /artists/{id}/albums` | From artist detail → list albums (album, single, appears_on, compilation). Play directly. |
| 14 | **Track Detail overlay** | `GET /tracks/{id}` | Full track metadata: explicit flag, ISRC, popularity, album art URL. Trigger from any track list. |
| 15 | **Album Detail overlay** | `GET /albums/{id}` | Release date, label, copyrights, full track list. Trigger from Albums pane, NowPlaying. |
| 16 | **Get Playlist Cover Image** | `GET /playlists/{id}/images` | Display cover URL in playlist detail. |
| 17 | **Upload Custom Playlist Cover** | `PUT /playlists/{id}/images` | JPEG upload from local file for owned playlists. |
| 18 | **Audiobooks Pane** | `GET /me/audiobooks` | List saved audiobooks. New pane in library preset. |
| 19 | **Audiobook Detail + Chapters** | `GET /audiobooks/{id}`, `GET /audiobooks/{id}/chapters` | Drill-down: audiobook → chapter list with durations. |
| 20 | **Chapter Detail** | `GET /chapters/{id}` | Chapter metadata, duration, audio preview URL. |
| 21 | **Lightweight Playback Poll** | `GET /me/player/currently-playing` | Lighter than full playback state. Use when paused/idle to reduce bandwidth. |

**10 features.** ~2-3 weeks.

---

### TIER 3 — Higher Effort (new panes, significant UI)

| # | Feature | Endpoints | Notes |
|---|---------|-----------|-------|
| 22 | **Artist Pane** (dedicated) | artist + albums + search | Full artist experience: genres, popularity, discography, play all. Replaces overlay approach. |
| 23 | **Album Pane** (dedicated) | album + album tracks | Full album view: cover URL, release info, track list, durations, play all. |
| 24 | **Full Audiobook Experience** | audiobooks + chapters + chapter + library | Dedicated pane: progress tracking, chapter navigation, save/unsave, play from position. |
| 25 | **Followed Artists Pane** | `GET /me/following?type=artist` + artist detail | Artists you follow with drill-down to their albums. Different from Top Artists. |

**4 features.** ~4-6 weeks.

---

## Recommended Trajectory

```
Phase 1: Library + Playlist CRUD (Week 1-2)
  └─ Save/unsave track with heart indicator
  └─ Create, rename, remove, reorder, add-to-playlist
  └─ "Add to playlist" overlay
  └─ Save/unsave albums, shows, episodes

Phase 2: Catalog Detail Overlays (Week 3-4)
  └─ Artist detail, artist albums
  └─ Track detail, album detail
  └─ Playlist cover image

Phase 3: Discovery + Audiobooks (Week 5-8)
  └─ Followed Artists pane
  └─ Audiobooks support (pane + detail + chapters)
  └─ Lightweight playback poll optimization

Phase 4: Dedicated Panes (Week 9+)
  └─ Artist pane, Album pane
  └─ Full audiobook UX
```

---

## Key Design Decisions

1. **One save/unsave keybinding for all types** (`Ctrl+S`). The unified `/me/library` endpoint makes type-specific bindings unnecessary.
2. **Overlays first, panes later.** Start with overlays for artist/album/track detail. Promote to panes if usage warrants.
3. **Audiobooks are optional.** Gated behind market availability (US, UK, CA, IE, NZ, AU only). Add a config flag or auto-detect.
4. **Discovery is search-driven.** Without recommendations, browse, or related-artists APIs, discovery relies on search + followed content. This aligns with Spotnik's terminal power-user focus.
