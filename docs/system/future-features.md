# Future Features — Spotnik

> Status: analysis complete | 2026-06-18
>
> Based on Spotify Web API OpenAPI Schema v1.0.0.
> Only **active (non-deprecated)** endpoints considered.
> 48 endpoints are active, 48 are deprecated.

---

## Deprecation Context

Spotify has deprecated ~50% of its Web API. Key removals that impact feature planning:

| Deprecated Domain | What's Lost | Replacement |
|---|---|---|
| Audio Features / Analysis | danceability, energy, valence, tempo, key, mode, waveform | **None** |
| Recommendations / Genre Seeds | seed-based recs, tunable attributes, genre list | **None** |
| Browse (Categories, New Releases, Featured Playlists) | discovery content, editorial playlists | **None** |
| Artist Top Tracks / Related Artists | per-artist popularity, similar artists | **None** |
| Follow/Unfollow (Artists, Playlists) | social graph mutations | **None** |
| User Profiles / Other Users' Playlists | cross-user browsing | **None** |
| Type-specific save/remove/check (tracks, albums, shows, episodes, audiobooks) | per-type library CRUD | `PUT/DELETE/GET /me/library` (unified) |
| Old playlist endpoints (`/playlists/{id}/tracks`) | legacy playlist CRUD | `/playlists/{id}/items` (already used) |

### What Survives (Active API Surface)

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

## Feature Candidates (Active APIs Only)

### TIER 1 — Very Low Effort (existing patterns, small UI)

| # | Feature | Endpoint | Notes |
|---|---------|----------|-------|
| 1 | **Save/Unsave track from NowPlaying** | `PUT/DELETE /me/library` | Migrate from deprecated `PUT/DELETE /me/tracks`. Add keybinding + heart indicator. |
| 2 | **Check item saved status** | `GET /me/library/contains?uris=...` | Migrate from deprecated type-specific `contains`. Unified check for any URI. |
| 3 | **Create playlist from selection** | `POST /me/playlists` | API client exists. Need: "Save as playlist..." prompt overlay from search/queue. |
| 4 | **Rename playlist** | `PUT /playlists/{id}` | API client exists. Need: inline edit or prompt overlay in Playlists pane. |
| 5 | **Remove track from playlist** | `DELETE /playlists/{id}/items` | API client exists. Wire `x` key (currently no-op). |
| 6 | **Reorder playlist items** | `PUT /playlists/{id}/items` | API client exists. Cut/paste or `Ctrl+Up/Down` keybindings. |
| 7 | **Add track to playlist** | `POST /playlists/{id}/items` | API client exists. "Add to playlist" overlay with playlist picker. |
| 8 | **Save/Unsave album** | `PUT/DELETE /me/library` | New keybinding in Albums pane. Save by album URI. |
| 9 | **Save/Unsave show** | `PUT/DELETE /me/library` | New keybinding in FollowedShows pane. |
| 10 | **Save/Unsave episode** | `PUT/DELETE /me/library` | New keybinding in episode lists. |
| 11 | **Get Followed Artists** | `GET /me/following?type=artist` | New pane or section: artists you follow (different from Top Artists). |

**TIER 1 total: 11 features.** ~3-5 days.

---

### TIER 2 — Medium Effort (new API clients, follows existing design patterns)

| # | Feature | Endpoint | Notes |
|---|---------|----------|-------|
| 12 | **Artist Detail overlay** | `GET /artists/{id}` | Genres, popularity, follower count, images. Trigger from NowPlaying artist, search results, top artists. |
| 13 | **Artist's Albums drill-down** | `GET /artists/{id}/albums` | From artist detail → list albums (album, single, appears_on, compilation). Play directly. |
| 14 | **Track Detail overlay** | `GET /tracks/{id}` | Full track metadata: explicit flag, ISRC, popularity, album art URL. Trigger from any track list. |
| 15 | **Album Detail overlay** | `GET /albums/{id}` | Release date, label, copyrights, full track list. Trigger from Albums pane, NowPlaying. |
| 16 | **Get Playlist Cover Image** | `GET /playlists/{id}/images` | Display cover URL in playlist detail. Terminal can't render but URL is useful. |
| 17 | **Upload Custom Playlist Cover** | `PUT /playlists/{id}/images` | JPEG upload from local file for owned playlists. |
| 18 | **Audiobooks Pane** | `GET /me/audiobooks` | List saved audiobooks. New pane in library preset. |
| 19 | **Audiobook Detail + Chapters** | `GET /audiobooks/{id}`, `GET /audiobooks/{id}/chapters` | Drill-down: audiobook → chapter list with durations. |
| 20 | **Chapter Detail** | `GET /chapters/{id}` | Chapter metadata, duration, audio preview URL. |
| 21 | **Lightweight Playback Poll** | `GET /me/player/currently-playing` | Lighter than full playback state. Use when paused/idle to reduce bandwidth. Fallback to full state. |

**TIER 2 total: 10 features.** ~2-3 weeks.

---

### TIER 3 — Higher Effort (new panes, significant UI)

| # | Feature | Endpoints | Notes |
|---|---------|-----------|-------|
| 22 | **Artist Pane** (dedicated) | artist + albums + search | Full artist experience: bio, genres, popularity, discography, play all. Replaces overlay approach. |
| 23 | **Album Pane** (dedicated) | album + album tracks + track detail | Full album view: cover URL, release info, track list, durations, play all. |
| 24 | **Full Audiobook Experience** | audiobooks + chapters + chapter + library | Dedicated audiobook pane: progress tracking, chapter navigation, save/unsave, play from position. |
| 25 | **Followed Artists Pane** | `GET /me/following?type=artist` + artist detail | Artists you follow with drill-down to their albums. Different from Top Artists (algorithmic). |
| 26 | **Unified Library Save/Unsave** (all content types) | `PUT/DELETE/GET /me/library` | Single keybinding (`Ctrl+S`) that saves/unsaves ANY item type. Heart/bookmark icon works on tracks, albums, shows, episodes, audiobooks uniformly. |

**TIER 3 total: 5 features.** ~4-6 weeks.

---

### TIER 4 — Speculative / Requires Deprecated APIs (Blocked)

These features are blocked because their APIs are deprecated:

| Feature | Blocked By (Deprecated) |
|---------|------------------------|
| Audio Features (danceability, energy, etc.) | `GET /audio-features/{id}` deprecated |
| Audio Analysis (waveform, beats, sections) | `GET /audio-analysis/{id}` deprecated |
| Track Recommendations | `GET /recommendations` deprecated |
| Genre Seeds list | `GET /recommendations/available-genre-seeds` deprecated |
| Artist's Top Tracks (per-artist) | `GET /artists/{id}/top-tracks` deprecated |
| Related Artists ("Fans also like") | `GET /artists/{id}/related-artists` deprecated |
| Browse Categories (genres/moods) | `GET /browse/categories` deprecated |
| Category Playlists | `GET /browse/categories/{id}/playlists` deprecated |
| Featured Playlists | `GET /browse/featured-playlists` deprecated |
| New Releases | `GET /browse/new-releases` deprecated |
| Follow/Unfollow Artists | `PUT/DELETE /me/following` deprecated |
| Check Artist Follow | `GET /me/following/contains` deprecated |
| Follow/Unfollow Playlist | `PUT/DELETE /playlists/{id}/followers` deprecated |
| Check Playlist Follow | `GET /playlists/{id}/followers/contains` deprecated |
| Other User's Profile | `GET /users/{user_id}` deprecated |
| Other User's Playlists | `GET /users/{user_id}/playlists` deprecated |
| Create Playlist for User | `POST /users/{user_id}/playlists` deprecated |
| Available Markets | `GET /markets` deprecated |

**If Spotify replaces these with new endpoints, re-evaluate.** Until then, these features cannot be built on stable APIs.

---

## Migration Required (Current Spotnik Tech Debt)

Spotnik currently uses deprecated endpoints that **must** migrate:

| Current Code | Deprecated Endpoint | Migration |
|---|---|---|
| `LibraryClient.LikeTracks` | `PUT /me/tracks` | `PUT /me/library?uris=spotify:track:{id}` |
| `LibraryClient.UnlikeTracks` | `DELETE /me/tracks` | `DELETE /me/library?uris=spotify:track:{id}` |
| Any `contains` checks if implemented | `GET /me/{type}/contains` | `GET /me/library/contains?uris=...` |
| Any save/unsave if implemented for albums/shows/episodes | `PUT/DELETE /me/{type}` | `PUT/DELETE /me/library?uris=...` |

The unified library endpoint accepts URIs for **all** content types (track, album, episode, show, audiobook). This simplifies the API client layer — one save/remove/check function handles everything.

---

## Recommended Trajectory

```
Phase 1: Fix Tech Debt (Now)
  └─ Migrate LikeTracks/UnlikeTracks to /me/library
  └─ Wire save/unsave keybinding to UI (heart indicator)

Phase 2: Playlist CRUD UI (Week 1-2)
  └─ Create, rename, remove, reorder, add-to-playlist
  └─ "Add to playlist" overlay

Phase 3: Catalog Detail Overlays (Week 3-4)
  └─ Artist detail, artist albums
  └─ Track detail, album detail
  └─ Playlist cover image

Phase 4: Discovery + Audiobooks (Week 5-8)
  └─ Followed Artists pane
  └─ Audiobooks support (pane + detail + chapters)
  └─ Lightweight playback poll optimization

Phase 5: Dedicated Panes (Week 9+)
  └─ Artist pane, Album pane
  └─ Full audiobook UX
```

---

## Key Design Decisions

1. **One save/unsave keybinding for all types** (`Ctrl+S`). The unified `/me/library` endpoint makes type-specific bindings unnecessary.
2. **Overlays first, panes later.** Start with overlays for artist/album/track detail. Promote to panes if usage warrants.
3. **Audiobooks are optional.** Gated behind market availability (US, UK, CA, IE, NZ, AU only). Add a config flag or auto-detect.
4. **No discovery features (for now).** Without recommendations, browse, related artists, or audio features, discovery is limited to search + followed content. This aligns with Spotnik's "power user" terminal focus.
