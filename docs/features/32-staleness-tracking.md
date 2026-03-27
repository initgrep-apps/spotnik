# Feature 32 — Staleness Tracking

> **Feature:** Add `fetchedAt` timestamps to the Store for each data domain. Provide
> TTL-based staleness checks so `Update()` can decide whether to reuse cached data
> or trigger a refresh.

## Context

The Store has no concept of data age. Library data (albums, liked tracks) is fetched
once per session and never refreshed. Stats data is cached per time-range forever.
Boolean sentinels like `albumsLoaded` (store.go line ~25) and `likedLoaded` (line ~28)
track whether data has been fetched at all, but don't support re-fetching after a TTL.

After Feature 29, `Update()` owns all Store writes. This feature adds staleness metadata
so `Update()` can make informed decisions about when to re-fetch.

**Gap reference:** G4 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

**Depends on:** Feature 29 (Store schema change — Update() must own writes first)

---

## Staleness TTLs

| Domain | TTL | Rationale |
|---|---|---|
| Playback state | N/A | Always polled, overwritten each tick cycle |
| Queue | N/A | Always polled, overwritten each tick cycle |
| Playlists list | 5 min | Changes infrequently |
| Albums | 5 min | Changes infrequently |
| Liked tracks | 5 min | Changes infrequently |
| Recently played | 2 min | Changes with playback |
| Stats (per range) | 10 min | Spotify updates these slowly |
| Devices | 5 sec | Volatile — short cooldown; user-initiated fetches use Interactive priority |

---

## Task 1: Add fetchedAt fields + IsStale() helper

**Problem:** Store has no timestamps for data freshness.

**Fix:**

1. Add `fetchedAt` fields to Store struct (internal/state/store.go):
   ```go
   // Staleness tracking — set to time.Now() on successful data write.
   playlistsFetchedAt    time.Time
   albumsFetchedAt       time.Time
   likedTracksFetchedAt  time.Time
   recentPlayedFetchedAt time.Time
   statsFetchedAt        map[string]time.Time // keyed by time range
   devicesFetchedAt      time.Time
   ```

2. Add `IsStale` package-level helper:
   ```go
   // IsStale returns true if fetchedAt is zero (never fetched) or older than ttl.
   func IsStale(fetchedAt time.Time, ttl time.Duration) bool {
       return fetchedAt.IsZero() || time.Since(fetchedAt) > ttl
   }
   ```

3. Update all `Set*` methods to stamp `fetchedAt = time.Now()` on successful writes:
   - `SetPlaylists()` → sets `playlistsFetchedAt`
   - `SetSavedAlbums()` → sets `albumsFetchedAt`
   - `SetLikedTracks()` → sets `likedTracksFetchedAt`
   - `SetRecentlyPlayed()` → sets `recentPlayedFetchedAt`
   - `SetTopTracks()` / `SetTopArtists()` → sets `statsFetchedAt[timeRange]`
   - Device fetched time set when devices are loaded

4. Add accessor methods:
   ```go
   func (s *Store) PlaylistsFetchedAt() time.Time
   func (s *Store) AlbumsFetchedAt() time.Time
   func (s *Store) LikedTracksFetchedAt() time.Time
   func (s *Store) RecentPlayedFetchedAt() time.Time
   func (s *Store) StatsFetchedAt(timeRange string) time.Time
   func (s *Store) DevicesFetchedAt() time.Time
   ```

**Files:**
- Modify: `internal/state/store.go` — add fields, update Set methods, add accessors
- Create or modify: `internal/state/store_test.go` — test staleness helper

**Tests:**
- Unit: `IsStale` returns true for zero time
- Unit: `IsStale` returns true when elapsed > TTL
- Unit: `IsStale` returns false when elapsed < TTL
- Unit: `SetPlaylists()` updates `playlistsFetchedAt`
- Unit: `SetTopTracks()` updates `statsFetchedAt[range]`

**Commit:** `feat(state): add fetchedAt timestamps and IsStale helper`

---

## Task 2: TTL constants + replace boolean sentinels

**Problem:** `albumsLoaded` and `likedLoaded` booleans (store.go lines ~25, ~28) are
single-use flags that can't support re-fetching.

**Fix:**

1. Add TTL constants to `internal/state/store.go`:
   ```go
   const (
       PlaylistsTTL      = 5 * time.Minute
       AlbumsTTL         = 5 * time.Minute
       LikedTracksTTL    = 5 * time.Minute
       RecentlyPlayedTTL = 2 * time.Minute
       StatsTTL          = 10 * time.Minute
       DevicesTTL        = 5 * time.Second
   )
   ```

2. Add convenience methods:
   ```go
   func (s *Store) PlaylistsStale() bool {
       s.mu.RLock()
       defer s.mu.RUnlock()
       return IsStale(s.playlistsFetchedAt, PlaylistsTTL)
   }
   // ... same for Albums, LikedTracks, RecentlyPlayed, Stats, Devices
   ```

3. Remove `albumsLoaded` and `likedLoaded` boolean fields. Replace their usage:
   - Where `albumsLoaded` was checked → use `!s.AlbumsStale()` or check `albumsFetchedAt.IsZero()`
   - Where `SetSavedAlbums` set `albumsLoaded = true` → timestamp does this implicitly
   - Same for `likedLoaded`

**Files:**
- Modify: `internal/state/store.go` — add constants, convenience methods, remove booleans
- Modify: `internal/ui/panes/library.go` — update any `albumsLoaded`/`likedLoaded` reads
  (if library pane reads these via Store accessors)

**Tests:**
- Unit: `PlaylistsStale()` returns true when never fetched
- Unit: `PlaylistsStale()` returns true after TTL expires
- Unit: `PlaylistsStale()` returns false within TTL
- Unit: removing boolean sentinels doesn't break existing tests

**Commit:** `refactor(state): replace boolean sentinels with TTL-based staleness`

---

## Task 3: Wire staleness checks into Update() + docs

**Problem:** Library data is fetched once and never refreshed. Stats are cached forever
per time range.

**Fix:**

1. In `app.go` / `routing.go` — where library fetch commands are dispatched:
   - Before dispatching `buildFetchPlaylistsCmd`, check `a.store.PlaylistsStale()`
   - Before dispatching `buildFetchAlbumsCmd`, check `a.store.AlbumsStale()`
   - Same for liked tracks, recently played
   - If not stale, skip the fetch and use cached data

2. In stats view: when user switches to stats view or changes time range:
   - Check `a.store.StatsStale(timeRange)` before dispatching `buildFetchStatsCmd`
   - If not stale, skip fetch — cached data is still valid

3. In library pane navigation (Init or section switch):
   - Currently `library.go` line ~350 dispatches fetches unconditionally on Init
   - After this change: `Update()` checks staleness before dispatching

4. Update docs:
   - **`docs/ARCHITECTURE.md`** → "State Management" → "The Store": Add staleness tracking
     documentation, `IsStale()` pattern, TTL constants
   - **`docs/ARCHITECTURE.md`** → "Polling Architecture": Note that library/stats use
     staleness-based refresh, not polling

**Files:**
- Modify: `internal/app/app.go` — add staleness checks before library/stats fetches
- Modify: `internal/app/routing.go` — add staleness checks before playlist fetches
- Modify: `internal/ui/panes/library.go` — update Init/navigation to emit fetch requests
  that `Update()` can gate with staleness checks
- Modify: `docs/ARCHITECTURE.md` — add staleness tracking documentation

**Tests:**
- Unit: library data re-fetches after TTL expires
- Unit: library data NOT re-fetched within TTL
- Unit: stats re-fetch on stale re-open
- Unit: stats NOT re-fetched within TTL
- Integration: switching away from stats and back within TTL uses cached data

**Commit 1:** `feat(app): staleness-gated data fetching`
**Commit 2:** `docs: add staleness tracking to architecture docs`

---

## Verification

```bash
# Boolean sentinels removed
grep -r 'albumsLoaded\|likedLoaded' internal/state/store.go
# Expected: ZERO matches

# Staleness checks exist before fetches
grep -r 'Stale()' internal/app/
# Expected: multiple matches for library/stats domains

make ci
# Expected: Full pass
```

---

*Depends on: Feature 29*
*Blocked by: Feature 29*
*Blocks: Nothing*
