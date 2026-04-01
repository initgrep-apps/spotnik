# Spotify Web API Capability Reference for Spotnik

> **Purpose:** Comprehensive reference of all Spotify Web API endpoints, their capabilities,
> limitations, device restrictions, deprecation status, and Spotnik's current usage.
> Use this document as the single source of truth when planning new features.
>
> **Last researched:** 2026-04-01
> **API base URL:** `https://api.spotify.com/v1`
> **Verified against:** Spotify Web API docs (Context7 + firecrawl), codebase audit of `internal/api/`

---

## Table of Contents

1. [Current Spotnik OAuth Scopes](#1-current-spotnik-oauth-scopes)
2. [Spotnik API Coverage Summary](#2-spotnik-api-coverage-summary)
3. [End-to-End Wiring Audit](#3-end-to-end-wiring-audit)
4. [February 2026 API Breaking Changes](#4-february-2026-api-breaking-changes)
5. [Player API](#5-player-api)
6. [Library API](#6-library-api)
7. [Playlists API](#7-playlists-api)
8. [Search API](#8-search-api)
9. [Albums API](#9-albums-api)
10. [Artists API](#10-artists-api)
11. [Tracks API](#11-tracks-api)
12. [Users API](#12-users-api)
13. [Audiobooks API](#13-audiobooks-api)
14. [Shows API](#14-shows-api)
15. [Episodes API](#15-episodes-api)
16. [Categories API](#16-categories-api)
17. [Chapters API](#17-chapters-api)
18. [Genres API](#18-genres-api)
19. [Deprecated & Removed Endpoints](#19-deprecated--removed-endpoints)
20. [Device Restrictions & Limitations](#20-device-restrictions--limitations)
21. [Premium vs Free Tier Restrictions](#21-premium-vs-free-tier-restrictions)
22. [Scope Reference](#22-scope-reference)
23. [Feature Opportunity Matrix](#23-feature-opportunity-matrix)

---

## 1. Current Spotnik OAuth Scopes

Defined in `internal/api/auth.go`:

```
user-read-playback-state    user-modify-playback-state
user-read-currently-playing playlist-read-private
playlist-read-collaborative playlist-modify-public
playlist-modify-private     user-library-read
user-library-modify         user-read-private
user-read-email             user-top-read
user-follow-read            user-read-recently-played
```

### Scopes NOT Currently Requested

| Scope | What It Unlocks |
|-------|-----------------|
| `user-follow-modify` | Follow/unfollow artists and users |
| `user-read-playback-position` | Resume position for episodes/audiobooks |
| `ugc-image-upload` | Upload custom playlist cover images |
| `streaming` | Web Playback SDK (not applicable for TUI) |
| `app-remote-control` | Spotify Connect remote control (iOS/Android SDK only) |

### Unused Scope Warning

`user-follow-read` is requested but **zero API methods** use it. No `GetFollowedArtists`, `CheckFollowing`, or similar methods exist in any API client. This scope should either be used (implement follow features) or removed.

**Action needed:** Add `user-follow-modify` if we want follow/unfollow. Add `user-read-playback-position` if we want episode/podcast support.

---

## 2. Spotnik API Coverage Summary

### Currently Implemented

| Client | File | Endpoints Used |
|--------|------|----------------|
| `Player` | `player.go` | `GET /me/player`, `PUT /me/player/play`, `PUT /me/player/pause`, `POST /me/player/next`, `POST /me/player/previous`, `PUT /me/player/seek`, `PUT /me/player/volume`, `PUT /me/player/shuffle`, `PUT /me/player/repeat`, `POST /me/player/queue`, `GET /me/player/queue` |
| `LibraryClient` | `library.go` | `GET /me/playlists`, `GET /playlists/{id}/tracks`, `GET /me/albums`, `GET /me/tracks`, `GET /me/player/recently-played`, `PUT /me/tracks`, `DELETE /me/tracks` |
| `PlaylistsClient` | `playlists.go` | `POST /me/playlists`, `PUT /playlists/{id}`, `POST /playlists/{id}/tracks`, `DELETE /playlists/{id}/tracks`, `PUT /playlists/{id}/tracks` (reorder) |
| `SearchClient` | `search.go` | `GET /search` |
| `DevicesClient` | `devices.go` | `GET /me/player/devices`, `PUT /me/player` (transfer) |
| `UserClient` | `user.go` | `GET /me/top/tracks`, `GET /me/top/artists`, `GET /me/player/recently-played` |

### Not Implemented (Opportunity)

| Category | Key Missing Endpoints | Effort | Value |
|----------|----------------------|--------|-------|
| Library: Albums | Save/remove/check albums | Low | Medium |
| Library: Shows/Episodes | Full podcast support | Medium | Low (music-focused app) |
| Library: Audiobooks | Full audiobook support | Medium | Low (music-focused app) |
| Artists | Get artist, albums, top tracks, related artists | Low | High |
| Artists | Follow/unfollow artists | Low | Medium |
| Albums | Get album details, album tracks | Low | Medium |
| Users | Get current user profile (`GET /me`) | Low | High (Premium detection) |
| Browse | Categories, category playlists | Low | Medium |
| Playlist | Follow/unfollow from search results | Low | Medium |
| Tracks | Get track details | Low | Low |

### Deprecated APIs Used

**None.** Spotnik does not use any deprecated or removed endpoints.

---

## 3. End-to-End Wiring Audit

Every API method was traced from keybinding → command builder → API call → response → store → UI.

### Wiring Status

| API Method | Keybinding | Command Builder | Store Write | UI Display | Status |
|-----------|------------|-----------------|-------------|------------|--------|
| `Player.PlaybackState` | 1s tick poll | `fetchPlaybackStateCmd` | Yes | NowPlaying pane | **Full** — but missing `supports_volume`, `actions`, `context`, `currently_playing_type` fields |
| `Player.Play` | `Space` | `buildPlaybackAPICmd` | N/A | Toast on error | **Full** |
| `Player.Pause` | `Space` | `buildPlaybackAPICmd` | N/A | Toast on error | **Full** |
| `Player.Next` | `n` | `buildPlaybackAPICmd` | N/A | Toast on error | **Full** |
| `Player.Previous` | N/A (no binding) | `buildPlaybackAPICmd` | N/A | Toast on error | **Full** |
| `Player.Seek` | `←`/`→` | `buildPlaybackAPICmd` | N/A | Progress bar updates | **Full** |
| `Player.SetVolume` | `+`/`-` | `buildPlaybackAPICmd` | N/A | Volume bar updates | **Broken on some devices** — no `supports_volume` guard |
| `Player.SetShuffle` | `s` | `buildPlaybackAPICmd` | N/A | Shuffle indicator | **Full** |
| `Player.SetRepeat` | `r` | `buildPlaybackAPICmd` | N/A | Repeat indicator | **Full** |
| `Player.AddToQueue` | `a` (in search/panes) | `buildAddToQueueCmd` | N/A | Success/error toast | **Full** |
| `Player.Queue` | Adaptive tick | `fetchQueueCmd` | Yes | QueuePane | **Full** |
| `Library.Playlists` | Init/Tab | `buildFetchPlaylistsCmd` | Yes | PlaylistsPane | **Full** |
| `Library.PlaylistTracks` | Playlist select | `buildFetchPlaylistTracksCmd` | Yes | Track list | **Full** |
| `Library.SavedAlbums` | Init/Tab | `buildFetchAlbumsCmd` | Yes | AlbumsPane | **Full** |
| `Library.LikedTracks` | Init/Tab | `buildFetchLikedTracksCmd` | Yes | LikedSongsPane | **Full** |
| `Library.RecentlyPlayed` | Init/Tab | `buildFetchRecentlyPlayedCmd` | Yes | RecentlyPlayedPane | **Full** |
| `Library.LikeTrack` | `l` | `buildToggleLikeCmd` | Yes | **No UI indicator** | **Partial** — API works but no heart icon |
| `Library.UnlikeTrack` | `l` | `buildToggleLikeCmd` | Yes | **No UI indicator** | **Partial** — same |
| `Playlists.CreatePlaylist` | Overlay | `buildCreatePlaylistCmd` | Yes | Toast + list refresh | **Full** |
| `Playlists.UpdatePlaylist` | Overlay | `buildRenamePlaylistCmd` | Yes | Toast + list refresh | **Full** |
| `Playlists.AddTracks` | Overlay | `buildAddTracksToPlaylistCmd` | Yes | Toast | **Full** |
| `Playlists.RemoveTracks` | `x` | `buildRemovePlaylistTrackCmd` | Yes | Toast + list refresh | **Full** |
| `Playlists.ReorderTracks` | `Shift+↑/↓` | `buildReorderPlaylistTracksCmd` | Yes | Optimistic UI update | **Full** |
| `Search.Search` | `/` (type query) | `buildSearchCmd` | Yes | SearchOverlay | **Full** — hardcoded: 4 types, limit=5, `market=from_token` |
| `Devices.Devices` | `d` (open overlay) | `buildFetchDevicesCmd` | Yes | DeviceOverlay | **Partial** — `supports_volume`, `is_restricted` fetched but NOT sent to UI |
| `Devices.TransferPlayback` | `Enter` (on device) | `buildTransferPlaybackCmd` | N/A | Toast | **Full** |
| `User.TopTracks` | Stats tab | `buildFetchStatsCmd` | Yes | TopTracksPane | **Full** |
| `User.TopArtists` | Stats tab | `buildFetchStatsCmd` | Yes | TopArtistsPane | **Full** |

### Missing Response Fields (Domain Type Gaps)

These fields exist in API responses but are **not parsed** into our domain types:

| Field | Missing From | Impact | Priority |
|-------|-------------|--------|----------|
| `Device.supports_volume` | `domain.Device` | Volume commands fail silently on unsupported devices | **Critical** |
| `PlaybackState.actions` | `domain.PlaybackState` | Can't disable unavailable controls per device | **Critical** |
| `PlaybackState.context` | `domain.PlaybackState` | Can't show what playlist/album is currently playing | **High** |
| `PlaybackState.currently_playing_type` | `domain.PlaybackState` | Can't distinguish tracks from episodes | **Medium** |
| `Track.explicit` | `domain.Track` | Can't show explicit content indicator | **Low** |
| `Track.track_number` | `domain.Track` | Can't show position in album | **Low** |
| `Track.disc_number` | `domain.Track` | Can't show disc number for multi-disc albums | **Low** |
| `SimplePlaylist.description` | `domain.SimplePlaylist` | Can't show playlist description | **Low** |
| `SimplePlaylist.public` | `domain.SimplePlaylist` | Can't show public/private status | **Low** |

### Partially Wired Features

1. **Like/Unlike (no UI indicator):** `LikeTrack`/`UnlikeTrack` API methods work and are wired to `buildToggleLikeCmd`. The `l` key triggers them. But the result is never shown in the UI — no heart icon in NowPlaying, no "liked" state in any pane. The `CheckSavedTracks` endpoint (`GET /me/tracks/contains`) is not implemented, so we can't even check the current state.

2. **Device overlay (missing capabilities):** `DevicesClient.Devices()` fetches the full device object including `supports_volume`, `is_restricted`, and `volume_percent`. But the conversion to `panes.DeviceInfo` only extracts `ID`, `Name`, `Type`, `IsActive`. The overlay shows no indication of restricted devices or volume-unsupported devices.

3. **Search (hardcoded params):** Search always uses 4 types, limit=5 per type, no pagination, no field filters. The API supports year/genre/tag filters and pagination up to offset 1000. (**Note:** Search limit has been reduced from 50 to 10 per request as of Feb 2026.)

---

## 4. February 2026 API Breaking Changes

Spotify made significant breaking changes in February 2026. These affect our capability doc:

### Endpoints Removed

| Endpoint | Status |
|----------|--------|
| `GET /markets` | **REMOVED** — no longer exists |
| `GET /browse/new-releases` | **Likely removed** — may still work but no longer documented |

### Response Fields Removed

| Field | Previously In | Impact on Spotnik |
|-------|--------------|-------------------|
| `Album.label` | Album objects | Not parsed — no impact |
| `Album.popularity` | Album objects | Not parsed — no impact |
| `Artist.followers` | Artist objects | Not parsed — no impact |
| `Artist.popularity` | Artist + FullArtist objects | **`FullArtist.Popularity` field in `domain.FullArtist` now receives zero value** |

### Search Limit Reduced

- **Old:** `limit` param accepted 0-50 per type
- **New:** `limit` param max is **10** per type (must paginate with `offset`)
- **Spotnik impact:** We use `limit=5` which is under the new cap, so no immediate breakage. But the doc must reflect the correct max.

### Playlist Items Restriction

Playlist items endpoint (`GET /playlists/{id}/tracks`) now only returns data for playlists the user **owns or collaborates on**. Third-party playlist track listing may fail.

### New Generic Library Endpoints

Spotify is introducing unified endpoints to replace entity-specific save/remove:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `PUT /me/library` | PUT | Save/follow any item by Spotify URI |
| `DELETE /me/library` | DELETE | Remove/unfollow any item by URI |
| `GET /me/library/contains` | GET | Check if items are saved by URI |

Entity-specific endpoints (`/me/tracks`, `/me/albums`, etc.) still work but may be sunset. **Recommendation:** Continue using entity-specific endpoints for now; plan migration to generic endpoints when Spotify announces deprecation dates.

### Recommendations API

- `GET /recommendations` — **Restricted** since Nov 27, 2024. Returns 403 for apps created after that date. Not fully deleted — existing apps with extended access can still use it.
- `GET /recommendations/available-genre-seeds` — **Restricted** alongside recommendations.

---

## 5. Player API

All write endpoints require **Spotify Premium**. Read endpoints work on Free tier.

### 5.1 Get Playback State

| | |
|---|---|
| **Endpoint** | `GET /me/player` |
| **Scope** | `user-read-playback-state` |
| **Premium** | No |
| **Spotnik** | **Implemented** in `Player.PlaybackState()` |

**Query Params:**
- `market` (string, optional) -- ISO 3166-1 alpha-2
- `additional_types` (string, optional) -- `track`, `episode`

**Response (200):**
- `device` -- object with `id`, `is_active`, `is_private_session`, `is_restricted`, `name`, `type`, `volume_percent`, **`supports_volume`**
- `repeat_state` -- `"off"`, `"track"`, `"context"`
- `shuffle_state` -- boolean
- `context` -- object: `type`, `href`, `external_urls`, `uri`
- `timestamp` -- Unix ms
- `progress_ms` -- integer
- `is_playing` -- boolean
- `item` -- Track or Episode object
- `currently_playing_type` -- `"track"`, `"episode"`, `"ad"`, `"unknown"`
- **`actions`** -- disallowed actions object (see [Device Restrictions](#20-device-restrictions--limitations))

**Returns 204** with empty body if no active device.

**Spotnik gaps:**
- `supports_volume` not in `domain.Device` struct — volume commands sent blindly to unsupported devices
- `actions` not parsed — can't know which controls are available on current device
- `currently_playing_type` not parsed — can't distinguish tracks from episodes
- `context` not parsed — can't show what playlist/album is currently playing

### 5.2 Get Currently Playing Track

| | |
|---|---|
| **Endpoint** | `GET /me/player/currently-playing` |
| **Scope** | `user-read-currently-playing` OR `user-read-playback-state` |
| **Premium** | No |
| **Spotnik** | **Not implemented** (we use `GET /me/player` which returns more data) |

Lighter-weight alternative to 5.1 — not needed since we poll the full playback state.

### 5.3 Start/Resume Playback

| | |
|---|---|
| **Endpoint** | `PUT /me/player/play` |
| **Scope** | `user-modify-playback-state` |
| **Premium** | **Yes** |
| **Spotnik** | **Implemented** in `Player.Play()` |

**Query Params:**
- `device_id` (string, optional) -- target device

**Body (JSON, all optional):**
- `context_uri` -- album/artist/playlist URI to play
- `uris` -- array of track URIs
- `offset` -- `{position: int}` or `{uri: string}` to start at specific track
- `position_ms` -- position within track to start from

**Note:** Empty body resumes current playback. If both `context_uri` and `uris` provided, behavior is undefined.

### 5.4 Pause Playback

| | |
|---|---|
| **Endpoint** | `PUT /me/player/pause` |
| **Scope** | `user-modify-playback-state` |
| **Premium** | **Yes** |
| **Spotnik** | **Implemented** in `Player.Pause()` |

**Query Params:** `device_id` (optional)

### 5.5 Skip to Next Track

| | |
|---|---|
| **Endpoint** | `POST /me/player/next` |
| **Scope** | `user-modify-playback-state` |
| **Premium** | **Yes** |
| **Spotnik** | **Implemented** in `Player.Next()` |

**Query Params:** `device_id` (optional)

### 5.6 Skip to Previous Track

| | |
|---|---|
| **Endpoint** | `POST /me/player/previous` |
| **Scope** | `user-modify-playback-state` |
| **Premium** | **Yes** |
| **Spotnik** | **Implemented** in `Player.Previous()` |

**Query Params:** `device_id` (optional)

### 5.7 Seek to Position

| | |
|---|---|
| **Endpoint** | `PUT /me/player/seek` |
| **Scope** | `user-modify-playback-state` |
| **Premium** | **Yes** |
| **Spotnik** | **Implemented** in `Player.Seek()` |

**Query Params:**
- `position_ms` (integer, **required**) -- must be positive; beyond track length seeks to end
- `device_id` (optional)

### 5.8 Set Repeat Mode

| | |
|---|---|
| **Endpoint** | `PUT /me/player/repeat` |
| **Scope** | `user-modify-playback-state` |
| **Premium** | **Yes** |
| **Spotnik** | **Implemented** in `Player.SetRepeat()` |

**Query Params:**
- `state` (string, **required**) -- `"track"`, `"context"`, or `"off"`
- `device_id` (optional)

### 5.9 Set Volume

| | |
|---|---|
| **Endpoint** | `PUT /me/player/volume` |
| **Scope** | `user-modify-playback-state` |
| **Premium** | **Yes** |
| **Spotnik** | **Implemented** in `Player.SetVolume()` |

**Query Params:**
- `volume_percent` (integer, **required**) -- 0 to 100
- `device_id` (optional)

**IMPORTANT — Device restriction:** Volume control is **not supported on all devices**. The `Device.supports_volume` field (boolean) indicates whether this endpoint will work. When `false`, the API call will fail. See [Device Restrictions](#20-device-restrictions--limitations).

**Spotnik gap:** Volume commands are sent blindly without checking `supports_volume`. This causes 403 errors on devices like smart speakers, Chromecast, and TV integrations. The `buildPlaybackAPICmd` in `commands.go` should check device capability before dispatching volume commands.

### 5.10 Toggle Shuffle

| | |
|---|---|
| **Endpoint** | `PUT /me/player/shuffle` |
| **Scope** | `user-modify-playback-state` |
| **Premium** | **Yes** |
| **Spotnik** | **Implemented** in `Player.SetShuffle()` |

**Query Params:**
- `state` (boolean, **required**) -- `true`/`false`
- `device_id` (optional)

### 5.11 Get Recently Played Tracks

| | |
|---|---|
| **Endpoint** | `GET /me/player/recently-played` |
| **Scope** | `user-read-recently-played` |
| **Premium** | No |
| **Spotnik** | **Implemented** in `LibraryClient.RecentlyPlayed()` and `UserClient.RecentlyPlayed()` |

**Query Params:**
- `limit` (integer, optional) -- 1-50, default 20
- `after` (integer, optional) -- Unix ms cursor, items after this time
- `before` (integer, optional) -- Unix ms cursor, items before this time

**Note:** Uses cursor-based pagination (`after`/`before`), not offset. Does not currently support podcast episodes.

### 5.12 Get User's Queue

| | |
|---|---|
| **Endpoint** | `GET /me/player/queue` |
| **Scope** | `user-read-currently-playing` OR `user-read-playback-state` |
| **Premium** | No |
| **Spotnik** | **Implemented** in `Player.Queue()` |

**Response:** `currently_playing` (Track/Episode) + `queue` (array of Track/Episode)

### 5.13 Add Item to Queue

| | |
|---|---|
| **Endpoint** | `POST /me/player/queue` |
| **Scope** | `user-modify-playback-state` |
| **Premium** | **Yes** |
| **Spotnik** | **Implemented** in `Player.AddToQueue()` |

**Query Params:**
- `uri` (string, **required**) -- track or episode URI
- `device_id` (optional)

### 5.14 Get Available Devices

| | |
|---|---|
| **Endpoint** | `GET /me/player/devices` |
| **Scope** | `user-read-playback-state` |
| **Premium** | No |
| **Spotnik** | **Implemented** in `DevicesClient.Devices()` |

**Response:** Array of device objects with: `id`, `is_active`, `is_private_session`, `is_restricted`, `name`, `type`, `volume_percent`, **`supports_volume`**

**Spotnik gap:** `supports_volume` is not in the `domain.Device` struct. The device overlay receives only `ID`, `Name`, `Type`, `IsActive` — the `is_restricted`, `supports_volume`, and `volume_percent` fields are fetched from the API but discarded during conversion to `panes.DeviceInfo`.

### 5.15 Transfer Playback

| | |
|---|---|
| **Endpoint** | `PUT /me/player` |
| **Scope** | `user-modify-playback-state` |
| **Premium** | **Yes** |
| **Spotnik** | **Implemented** in `DevicesClient.TransferPlayback()` |

**Body (JSON):**
- `device_ids` (array, **required**) -- single device ID
- `play` (boolean, optional) -- default `false`

---

## 6. Library API

All Library endpoints are available on **Free tier** (no Premium required).
Scopes: `user-library-read` for GET, `user-library-modify` for PUT/DELETE.

### 6.1 Saved Albums

| Endpoint | Method | Path | Max IDs | Spotnik |
|----------|--------|------|---------|---------|
| Get Saved Albums | `GET` | `/me/albums` | N/A (paginated) | **Implemented** |
| Save Albums | `PUT` | `/me/albums` | 20 (query) / 50 (body) | **Not implemented** |
| Remove Saved Albums | `DELETE` | `/me/albums` | 20 (query) / 50 (body) | **Not implemented** |
| Check Saved Albums | `GET` | `/me/albums/contains` | 20 | **Not implemented** |

**Opportunity:** Save/unsave albums from the Albums pane. Check if albums are saved to show a heart icon.

### 6.2 Saved Tracks (Liked Songs)

| Endpoint | Method | Path | Max IDs | Spotnik |
|----------|--------|------|---------|---------|
| Get Saved Tracks | `GET` | `/me/tracks` | N/A (paginated) | **Implemented** |
| Save Tracks | `PUT` | `/me/tracks` | 50 | **Implemented** (`LikeTrack`) |
| Remove Saved Tracks | `DELETE` | `/me/tracks` | 50 | **Implemented** (`UnlikeTrack`) |
| Check Saved Tracks | `GET` | `/me/tracks/contains` | 50 | **Not implemented** |

**Wiring gap:** `LikeTrack`/`UnlikeTrack` are fully wired to `buildToggleLikeCmd` (triggered by `l` key), but the result is never shown in the UI. No heart icon in NowPlaying or any other pane. `CheckSavedTracks` is not implemented, so we can't query the current like state.

**Special feature:** `PUT /me/tracks` supports `timestamped_ids` in body to set custom `added_at` per track.

### 6.3 Saved Episodes

| Endpoint | Method | Path | Max IDs | Spotnik |
|----------|--------|------|---------|---------|
| Get Saved Episodes | `GET` | `/me/episodes` | N/A | **Not implemented** |
| Save Episodes | `PUT` | `/me/episodes` | 50 | **Not implemented** |
| Remove Saved Episodes | `DELETE` | `/me/episodes` | 50 | **Not implemented** |
| Check Saved Episodes | `GET` | `/me/episodes/contains` | 50 | **Not implemented** |

**Scope needed:** `user-read-playback-position` for GET. Low priority — Spotnik is music-focused.

### 6.4 Saved Shows (Podcasts)

| Endpoint | Method | Path | Max IDs | Spotnik |
|----------|--------|------|---------|---------|
| Get Saved Shows | `GET` | `/me/shows` | N/A | **Not implemented** |
| Save Shows | `PUT` | `/me/shows` | 50 | **Not implemented** |
| Remove Saved Shows | `DELETE` | `/me/shows` | 50 | **Not implemented** |
| Check Saved Shows | `GET` | `/me/shows/contains` | 50 | **Not implemented** |

Low priority — Spotnik is music-focused.

### 6.5 Saved Audiobooks

| Endpoint | Method | Path | Max IDs | Spotnik |
|----------|--------|------|---------|---------|
| Get Saved Audiobooks | `GET` | `/me/audiobooks` | N/A | **Not implemented** |
| Save Audiobooks | `PUT` | `/me/audiobooks` | 50 | **Not implemented** |
| Remove Saved Audiobooks | `DELETE` | `/me/audiobooks` | 50 | **Not implemented** |
| Check Saved Audiobooks | `GET` | `/me/audiobooks/contains` | 50 | **Not implemented** |

Low priority. Audiobooks only available in US, UK, Canada, Ireland, New Zealand, Australia.

### 6.6 Generic Library Endpoints (New — Feb 2026)

| Endpoint | Method | Path | Purpose |
|----------|--------|------|---------|
| Save Items | `PUT` | `/me/library` | Save/follow any item by Spotify URI |
| Remove Items | `DELETE` | `/me/library` | Remove/unfollow any item by URI |
| Check Items | `GET` | `/me/library/contains` | Check if items are saved by URI |

These generic endpoints accept any Spotify URI and replace entity-specific save/remove endpoints. Entity-specific endpoints still work. **Recommendation:** Continue using entity-specific endpoints until Spotify announces deprecation dates for them.

---

## 7. Playlists API

### 7.1 Current User's Playlists

| Endpoint | Method | Path | Scope | Spotnik |
|----------|--------|------|-------|---------|
| Get Current User's Playlists | `GET` | `/me/playlists` | `playlist-read-private` | **Implemented** |
| Create Playlist | `POST` | `/users/{user_id}/playlists` | `playlist-modify-public/private` | **Implemented** (via `POST /me/playlists`) |

### 7.2 Playlist Operations

| Endpoint | Method | Path | Scope | Spotnik |
|----------|--------|------|-------|---------|
| Get Playlist | `GET` | `/playlists/{id}` | `playlist-read-private` (for private) | **Not implemented** |
| Change Playlist Details | `PUT` | `/playlists/{id}` | `playlist-modify-public/private` | **Implemented** (`UpdatePlaylist`) |
| Get Playlist Items | `GET` | `/playlists/{id}/tracks` | `playlist-read-private` | **Implemented** (`PlaylistTracks`) |
| Add Items to Playlist | `POST` | `/playlists/{id}/tracks` | `playlist-modify-public/private` | **Implemented** (`AddTracksToPlaylist`) |
| Remove Items from Playlist | `DELETE` | `/playlists/{id}/tracks` | `playlist-modify-public/private` | **Implemented** (`RemoveTracksFromPlaylist`) |
| Reorder Playlist Items | `PUT` | `/playlists/{id}/tracks` | `playlist-modify-public/private` | **Implemented** (`ReorderPlaylistTracks`) |

**Feb 2026 change:** Playlist items endpoint now only returns data for playlists the user **owns or collaborates on**. Third-party playlist track listing may return limited results.

### 7.3 Other Playlist Endpoints

| Endpoint | Method | Path | Scope | Spotnik |
|----------|--------|------|-------|---------|
| Get User's Playlists | `GET` | `/users/{id}/playlists` | `playlist-read-private/collaborative` | **Not implemented** |
| Get Playlist Cover Image | `GET` | `/playlists/{id}/images` | None | **Not implemented** |
| Upload Playlist Cover | `PUT` | `/playlists/{id}/images` | `ugc-image-upload` + modify | **Not implemented** |
| Follow Playlist | `PUT` | `/playlists/{id}/followers` | `playlist-modify-public/private` | **Not implemented** |
| Unfollow Playlist | `DELETE` | `/playlists/{id}/followers` | `playlist-modify-public/private` | **Not implemented** |
| Check If User Follows Playlist | `GET` | `/playlists/{id}/followers/contains` | `playlist-read-private` | **Not implemented** |

**Note on cover images:** Upload requires base64-encoded JPEG, max 256 KB. Requires `ugc-image-upload` scope. Not practical for a TUI.

**Note on follows/contains:** The `ids` query param on `GET /playlists/{id}/followers/contains` is **deprecated** — the endpoint now only checks the current authenticated user.

**Opportunity:** Follow/unfollow playlist is useful — "save to my library" for playlists from search results.

### 7.4 Browse Playlists

| Endpoint | Method | Path | Status | Spotnik |
|----------|--------|------|--------|---------|
| Get Featured Playlists | `GET` | `/browse/featured-playlists` | **DEPRECATED** | **Not implemented** |
| Get Category's Playlists | `GET` | `/browse/categories/{id}/playlists` | Active | **Not implemented** |

---

## 8. Search API

| | |
|---|---|
| **Endpoint** | `GET /search` |
| **Scope** | None required |
| **Spotnik** | **Implemented** in `SearchClient.Search()` |

**Query Params:**
- `q` (string, **required**) -- supports field filters: `album:`, `artist:`, `track:`, `year:`, `upc:`, `isrc:`, `genre:`, `tag:hipster`, `tag:new`
- `type` (string, **required**) -- comma-separated: `album`, `artist`, `playlist`, `track`, `show`, `episode`, `audiobook`
- `market` (string, optional)
- `limit` (integer, optional) -- **1-10** (reduced from 50 in Feb 2026), default 10, per type
- `offset` (integer, optional) -- 0-1000
- `include_external` (string, optional) -- `"audio"` for externally hosted

**Spotnik currently:** searches 4 types (`track`, `artist`, `album`, `playlist`), limit=5, `market=from_token`. No pagination, no field filters.

**Spotnik is within the new limit** (5 < 10), but pagination is now more important since results are capped lower.

**Advanced search features not exposed:**
- Year range filtering: `year:2020-2024`
- `tag:new` for albums released in past 2 weeks
- `tag:hipster` for low-popularity albums
- Genre filtering: `genre:rock`
- ISRC/UPC lookup

---

## 9. Albums API

No scopes required. Available on Free tier.

| Endpoint | Method | Path | Spotnik |
|----------|--------|------|---------|
| Get Album | `GET` | `/albums/{id}` | **Not implemented** |
| Get Several Albums | `GET` | `/albums` (max 20 IDs) | **Not implemented** |
| Get Album Tracks | `GET` | `/albums/{id}/tracks` | **Not implemented** |
| Get New Releases | `GET` | `/browse/new-releases` | **Not implemented** — **likely removed in Feb 2026** |

**Feb 2026 changes:** `Album.label` and `Album.popularity` fields have been **removed** from API responses.

**Opportunity:**
- **Get Album Tracks** would let users browse tracks within an album from the Albums pane
- **Get Album** provides full album details (release date, total tracks, artists)

---

## 10. Artists API

No scopes required for read endpoints. Follow endpoints require `user-follow-read`/`user-follow-modify`.

### 10.1 Artist Information

| Endpoint | Method | Path | Spotnik |
|----------|--------|------|---------|
| Get Artist | `GET` | `/artists/{id}` | **Not implemented** |
| Get Several Artists | `GET` | `/artists` (max 50) | **Not implemented** |
| Get Artist's Albums | `GET` | `/artists/{id}/albums` | **Not implemented** |
| Get Artist's Top Tracks | `GET` | `/artists/{id}/top-tracks` | **Not implemented** |
| Get Related Artists | `GET` | `/artists/{id}/related-artists` | **Not implemented** |

**Key notes:**
- **Top Tracks** requires `market` (mandatory), returns max 10 tracks, no pagination
- **Related Artists** returns max 20 artists, no pagination
- **Artist's Albums** supports `include_groups` filter: `album`, `single`, `appears_on`, `compilation`

**Feb 2026 changes:** `Artist.followers` and `Artist.popularity` fields have been **removed** from API responses. Our `domain.FullArtist` struct has a `Popularity int` field that will now always receive zero.

**High-value opportunity:** Artist detail view — when selecting an artist, show their top tracks, albums, related artists.

### 10.2 Follow/Unfollow Artists

| Endpoint | Method | Path | Scope | Spotnik |
|----------|--------|------|-------|---------|
| Get Followed Artists | `GET` | `/me/following?type=artist` | `user-follow-read` | **Not implemented** |
| Follow Artists | `PUT` | `/me/following?type=artist` | `user-follow-modify` | **Not implemented** |
| Unfollow Artists | `DELETE` | `/me/following?type=artist` | `user-follow-modify` | **Not implemented** |
| Check If Following | `GET` | `/me/following/contains?type=artist` | `user-follow-read` | **Not implemented** |

**Note:** Follow uses cursor-based pagination (`after` param), not offset. Max 50 IDs per follow/unfollow request. These endpoints also support `type=user` for following/unfollowing users.

**Scope needed:** `user-follow-modify` must be added to `SpotifyScopes` in `auth.go`.

---

## 11. Tracks API

| Endpoint | Method | Path | Status | Spotnik |
|----------|--------|------|--------|---------|
| Get Track | `GET` | `/tracks/{id}` | Active | **Not implemented** |
| Get Several Tracks | `GET` | `/tracks` (max 50) | Active | **Not implemented** |
| Get Audio Features | `GET` | `/audio-features/{id}` | **Deprecation signaled** | **Not implemented** |
| Get Several Audio Features | `GET` | `/audio-features` (max 100) | **Deprecation signaled** | **Not implemented** |
| Get Audio Analysis | `GET` | `/audio-analysis/{id}` | **Deprecation signaled** | **Not implemented** |

**Audio features/analysis:** Not formally marked as deprecated in indexed docs, but Spotify has signaled intent to deprecate since late 2024. These endpoints may stop working for new apps at any time. **Do not build new features on them.**

**Opportunity:** `Get Track` could be useful for enriching track details (e.g., when a track is selected).

---

## 12. Users API

| Endpoint | Method | Path | Scope | Spotnik |
|----------|--------|------|-------|---------|
| Get Current User's Profile | `GET` | `/me` | `user-read-private`, `user-read-email` | **Not implemented** |
| Get User's Profile | `GET` | `/users/{id}` | None | **Not implemented** |
| Get User's Top Items | `GET` | `/me/top/{type}` | `user-top-read` | **Implemented** (via `UserClient`) |

**Get Current User's Profile** response includes:
- `country` (string) -- useful for `market` param
- `display_name` (string)
- `product` (string) -- `"premium"` or `"free"` -- **critical for knowing Premium status**
- `images` (array)

**Spotnik gap:** `GET /me` is never called. Premium status is unknown until a 403 error is received. The 403 handler shows a generic "Playback control not available on this device" toast rather than "Spotify Premium required". Implementing this endpoint would enable:
- Proactive Premium detection at startup
- Better error messages
- Hiding/dimming Premium-only controls for free users
- Using `country` for the `market` param instead of `from_token`

### Top Items Time Ranges

| Value | Period |
|-------|--------|
| `short_term` | Last 4 weeks |
| `medium_term` | Last 6 months (default) |
| `long_term` | Several years |

---

## 13. Audiobooks API

**Market restriction:** Only available in US, UK, Canada, Ireland, New Zealand, Australia.

| Endpoint | Method | Path | Spotnik |
|----------|--------|------|---------|
| Get Audiobook | `GET` | `/audiobooks/{id}` | **Not implemented** |
| Get Several Audiobooks | `GET` | `/audiobooks` (max 50) | **Not implemented** |
| Get Audiobook Chapters | `GET` | `/audiobooks/{id}/chapters` | **Not implemented** |

Low priority for a music-focused TUI.

---

## 14. Shows API (Podcasts)

| Endpoint | Method | Path | Scope | Spotnik |
|----------|--------|------|-------|---------|
| Get Show | `GET` | `/shows/{id}` | None | **Not implemented** |
| Get Several Shows | `GET` | `/shows` (max 50) | None | **Not implemented** |
| Get Show Episodes | `GET` | `/shows/{id}/episodes` | `user-read-playback-position` | **Not implemented** |

Low priority for a music-focused TUI.

---

## 15. Episodes API

| Endpoint | Method | Path | Scope | Spotnik |
|----------|--------|------|-------|---------|
| Get Episode | `GET` | `/episodes/{id}` | `user-read-playback-position` | **Not implemented** |
| Get Several Episodes | `GET` | `/episodes` | `user-read-playback-position` | **Not implemented** |

Low priority for a music-focused TUI.

---

## 16. Categories API

| Endpoint | Method | Path | Spotnik |
|----------|--------|------|---------|
| Get Browse Categories | `GET` | `/browse/categories` | **Not implemented** |
| Get Single Category | `GET` | `/browse/categories/{id}` | **Not implemented** |

**Opportunity:** Category browsing could power a "Browse" or "Discover" feature — browse by mood, genre, etc. May be restricted for new apps post-Nov 2024.

---

## 17. Chapters API

**Market restriction:** Only available in US, UK, Canada, Ireland, New Zealand, Australia.

| Endpoint | Method | Path | Spotnik |
|----------|--------|------|---------|
| Get Chapter | `GET` | `/chapters/{id}` | **Not implemented** |
| Get Several Chapters | `GET` | `/chapters` (max 20) | **Not implemented** |

Low priority — audiobook-specific.

---

## 18. Genres API

| Endpoint | Method | Path | Status | Spotnik |
|----------|--------|------|--------|---------|
| Get Available Genre Seeds | `GET` | `/recommendations/available-genre-seeds` | **RESTRICTED** | **Not implemented** |

Restricted alongside `GET /recommendations` since Nov 2024. Returns 403 for apps created after Nov 27, 2024.

---

## 19. Deprecated & Removed Endpoints

### Removed (no longer exist)

| Endpoint | Path | Removed |
|----------|------|---------|
| Get Available Markets | `GET /markets` | Feb 2026 |
| Get New Releases | `GET /browse/new-releases` | Likely Feb 2026 |

### Restricted (403 for new apps)

| Endpoint | Path | Since |
|----------|------|-------|
| Get Recommendations | `GET /recommendations` | Nov 2024 |
| Get Genre Seeds | `GET /recommendations/available-genre-seeds` | Nov 2024 |

### Deprecated (still works but may be removed)

| Endpoint | Path | Notes |
|----------|------|-------|
| Get Featured Playlists | `GET /browse/featured-playlists` | May return empty results |
| Check Playlist Followers (`ids` param) | `GET /playlists/{id}/followers/contains` | `ids` param deprecated — now checks current user only |

### Deprecation Signaled (no formal notice but intent communicated)

| Endpoint | Path | Notes |
|----------|------|-------|
| Get Audio Features (single) | `GET /audio-features/{id}` | Signaled since late 2024 |
| Get Audio Features (bulk) | `GET /audio-features` | Signaled since late 2024 |
| Get Audio Analysis | `GET /audio-analysis/{id}` | Signaled since late 2024 |

### Response Fields Removed (Feb 2026)

| Field | Previously In | Notes |
|-------|--------------|-------|
| `Album.label` | Album objects | No longer returned |
| `Album.popularity` | Album objects | No longer returned |
| `Artist.followers` | Artist objects | No longer returned |
| `Artist.popularity` | Artist objects | No longer returned; `domain.FullArtist.Popularity` now always zero |

**Key takeaway:** Spotnik uses **zero** deprecated or removed endpoints. The `FullArtist.Popularity` field in our domain types references a removed API field — harmless (just zeros) but should be noted.

---

## 20. Device Restrictions & Limitations

### The `actions` Object

The playback state response includes an `actions.disallows` object that lists controls the current device does not support:

```json
{
  "actions": {
    "disallows": {
      "interrupting_playback": true,
      "pausing": false,
      "resuming": false,
      "seeking": false,
      "skipping_next": false,
      "skipping_prev": false,
      "toggling_repeat_context": false,
      "toggling_shuffle": false,
      "toggling_repeat_track": false,
      "transferring_playback": false
    }
  }
}
```

**When a field is `true`, that action is NOT available on the current device.** This is the authoritative way to know which controls to enable/disable in the UI.

### The `supports_volume` Field

Each device object includes:
- `supports_volume` (boolean) -- `false` on certain smart speakers, TV integrations, and cast devices
- `volume_percent` -- may be `null` when volume is not supported

**When `supports_volume` is `false`:**
- `PUT /me/player/volume` will fail
- The UI should hide or disable volume controls

### The `is_restricted` Field

- `is_restricted` (boolean) -- when `true`, **no Web API commands** will be accepted by this device
- Restricted devices should be shown as non-selectable in the device list

### Device Types

Common values for `device.type`:
- `Computer` -- desktop app, supports all controls
- `Smartphone` -- mobile app, supports all controls
- `Speaker` -- smart speakers, may not support volume
- `TV` -- smart TV, may have restrictions
- `CastVideo` / `CastAudio` -- Chromecast, often no volume support
- `Automobile` -- car systems, may have restrictions
- `GameConsole` -- limited control support

### Spotnik Gaps (Codebase Audit)

1. **`supports_volume` not in `domain.Device` struct** — must be added
2. **`actions` not parsed from `PlaybackState`** — must be added to `domain.PlaybackState`
3. **Device overlay discards capability data** — `DevicesClient.Devices()` returns full device objects but the conversion to `panes.DeviceInfo` only keeps `ID`, `Name`, `Type`, `IsActive`. The `is_restricted`, `supports_volume`, and `volume_percent` fields are lost.
4. **Volume commands sent blindly** — `buildPlaybackAPICmd` in `commands.go` dispatches volume changes without checking `supports_volume`. The 403 error is caught reactively via toast notification.
5. **No UI indication** when volume/controls are unavailable on the current device

---

## 21. Premium vs Free Tier Restrictions

### Premium Required (write/control operations)

All Player write operations require Premium:
- Play, Pause, Next, Previous
- Seek, Volume, Shuffle, Repeat
- Transfer Playback
- Add to Queue

### Free Tier (read operations)

These work without Premium:
- Get Playback State, Currently Playing, Queue
- Get Devices
- Get Recently Played
- All Library read/write operations (save/remove tracks, albums, etc.)
- All Search operations
- All Playlist operations (create, modify, etc.)
- All Artist/Album/Track read operations
- All User profile operations

### Detecting Premium Status

`GET /me` returns `product: "premium"` or `product: "free"`. Use this to:
- Show/hide playback controls
- Show warnings when attempting Premium-only actions
- Adjust the UI layout for Free tier users

### Spotnik's Current Premium Handling (Codebase Audit)

- **No `GET /me` call** — Premium status is completely unknown until an operation fails
- **Reactive 403 handling only** — two places catch `ForbiddenError`:
  - `PlaybackCmdSentMsg` handler: shows "Playback control not available on this device" (generic message, doesn't mention Premium)
  - `AddToQueueResultMsg` handler: shows the API error message directly
- **No proactive detection** — all playback controls are shown regardless of account type
- **Free users see all controls** and hit 403 errors when they try to use them

---

## 22. Scope Reference

### All Available Scopes

| Scope | Description | Spotnik Requests | Spotnik Uses |
|-------|-------------|:------:|:------:|
| `user-read-playback-state` | Read playback state, devices | Yes | Yes |
| `user-modify-playback-state` | Control playback | Yes | Yes |
| `user-read-currently-playing` | Read currently playing | Yes | Yes |
| `playlist-read-private` | Read private playlists | Yes | Yes |
| `playlist-read-collaborative` | Read collaborative playlists | Yes | Yes |
| `playlist-modify-public` | Modify public playlists | Yes | Yes |
| `playlist-modify-private` | Modify private playlists | Yes | Yes |
| `user-library-read` | Read saved content | Yes | Yes |
| `user-library-modify` | Save/remove content | Yes | Yes |
| `user-read-private` | Read user profile (country, product) | Yes | **No** (no `GET /me` call) |
| `user-read-email` | Read user email | Yes | **No** (no `GET /me` call) |
| `user-top-read` | Read top artists/tracks | Yes | Yes |
| `user-follow-read` | Read followed artists | Yes | **No** (zero API methods) |
| `user-follow-modify` | Follow/unfollow artists | **No** | — |
| `user-read-recently-played` | Read recently played | Yes | Yes |
| `user-read-playback-position` | Read episode/audiobook position | **No** | — |
| `ugc-image-upload` | Upload playlist cover images | **No** | — |
| `streaming` | Web Playback SDK | **No** | — |
| `app-remote-control` | iOS/Android remote control | **No** | — |

**3 scopes requested but unused:** `user-read-private`, `user-read-email`, `user-follow-read`

---

## 23. Feature Opportunity Matrix

Ranked by value to Spotnik users and implementation effort.

### Critical Fixes (Should Do)

| Feature | Endpoints Needed | New Scopes | Notes |
|---------|-----------------|------------|-------|
| **Device capability awareness** | Parse `supports_volume` + `actions` from existing responses | None | Stop sending unsupported commands; show disabled states in UI |
| **Premium detection** | `GET /me` | None (have `user-read-private`) | Proactive detection; better error messages; hide controls for free users |

### High Value, Low Effort

| Feature | Endpoints Needed | New Scopes | Notes |
|---------|-----------------|------------|-------|
| **Track like status** | `GET /me/tracks/contains` | None | Show heart icon in NowPlaying for currently playing track |
| **Save/unsave albums** | `PUT /me/albums`, `DELETE /me/albums`, `GET /me/albums/contains` | None | Complete the library CRUD story |
| **Playback context display** | Parse `context` from existing `GET /me/player` response | None | Show "Playing from: My Playlist" in NowPlaying |

### High Value, Medium Effort

| Feature | Endpoints Needed | New Scopes | Notes |
|---------|-----------------|------------|-------|
| **Artist detail view** | `GET /artists/{id}`, `/top-tracks`, `/albums`, `/related-artists` | None | Deep-dive into any artist; note: `popularity` field removed |
| **Album track browsing** | `GET /albums/{id}/tracks` | None | View tracks within an album, play from position |
| **Follow/unfollow artists** | `PUT/DELETE /me/following`, `GET /me/following/contains` | `user-follow-modify` | Show follow state, toggle from artist view |
| **Category browsing** | `GET /browse/categories`, `/categories/{id}/playlists` | None | Browse by mood/genre; may be restricted for new apps |

### Medium Value, Low Effort

| Feature | Endpoints Needed | New Scopes | Notes |
|---------|-----------------|------------|-------|
| **Follow/unfollow playlist** | `PUT/DELETE /playlists/{id}/followers` | None | "Save playlist" from search results |
| **Get followed artists list** | `GET /me/following?type=artist` | None (have `user-follow-read`) | Show in library; uses cursor pagination |
| **Advanced search filters** | Existing `GET /search` | None | Expose year, genre, tag filters in search UI |
| **Search pagination** | Existing `GET /search` with `offset` | None | Navigate beyond first 10 results |

### Low Value (Out of Scope)

| Feature | Endpoints Needed | New Scopes |
|---------|-----------------|------------|
| Podcast browsing | Shows + Episodes APIs | `user-read-playback-position` |
| Audiobook browsing | Audiobooks + Chapters APIs | `user-read-playback-position` |
| Episode library | Library Episodes APIs | `user-read-playback-position` |

---

## API Rate Limiting

All endpoints share the same rate limiting:
- **429 Too Many Requests** with `Retry-After` header (seconds)
- No documented per-endpoint rate limits — it's a global rolling window
- Spotnik already handles this via the API Gateway (token bucket + 429 backoff)

## Pagination Patterns

| Pattern | Used By | Params |
|---------|---------|--------|
| **Offset** | Most endpoints | `limit` (max varies by endpoint), `offset` |
| **Cursor** | Recently Played, Followed Artists | `after`/`before` (timestamps or IDs) |

**Search limit:** Max 10 per type (reduced from 50 in Feb 2026).
**Max offset:** Playlists cap at 100,000. Most others don't document a max.

---

*This document should be updated when Spotify changes their API. Check https://developer.spotify.com/documentation/web-api for updates.*
*Sources: Spotify Web API Reference, Feb 2026 Changelog, Nov 2024 Changes Blog, codebase audit of internal/api/ and internal/app/*
