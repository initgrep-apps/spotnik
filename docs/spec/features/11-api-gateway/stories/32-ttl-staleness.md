---
title: "Staleness Tracking"
feature: 11-api-gateway
status: done
---

## Background
The Store had no concept of data age. Library data (albums, liked tracks) was fetched once per session and never refreshed. Stats data was cached per time-range forever. Boolean sentinels like `albumsLoaded` and `likedLoaded` tracked whether data had been fetched at all, but didn't support re-fetching after a TTL. After Feature 29 established that Update() owns all Store writes, this story added fetchedAt timestamps and TTL-based staleness checks so Update() can make informed decisions about when to re-fetch versus reuse cached data.

Gap reference: G4 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

## Design

### Staleness TTLs

| Domain | TTL | Rationale |
|---|---|---|
| Playback state | N/A | Always polled, overwritten each tick cycle |
| Queue | N/A | Always polled, overwritten each tick cycle |
| Playlists list | 5 min | Changes infrequently |
| Albums | 5 min | Changes infrequently |
| Liked tracks | 5 min | Changes infrequently |
| Recently played | 2 min | Changes with playback |
| Stats (per range) | 10 min | Spotify updates these slowly |
| Devices | 5 sec | Volatile -- short cooldown; user-initiated fetches use Interactive priority |

### fetchedAt Fields
```go
playlistsFetchedAt    time.Time
albumsFetchedAt       time.Time
likedTracksFetchedAt  time.Time
recentPlayedFetchedAt time.Time
statsFetchedAt        map[string]time.Time // keyed by time range
devicesFetchedAt      time.Time
```

### IsStale Helper
```go
func IsStale(fetchedAt time.Time, ttl time.Duration) bool {
    return fetchedAt.IsZero() || time.Since(fetchedAt) > ttl
}
```

### Convenience Methods
`PlaylistsStale() bool`, `AlbumsStale() bool`, etc. -- each acquires RLock and delegates to IsStale().

### Verification
```bash
grep -r 'albumsLoaded\|likedLoaded' internal/state/store.go
# Expected: ZERO matches
grep -r 'Stale()' internal/app/
# Expected: multiple matches
make ci
```

## Acceptance Criteria
- [ ] Every data domain has a `fetchedAt` timestamp set on successful write
- [ ] `IsStale(fetchedAt, ttl)` helper returns true for zero time or elapsed > TTL
- [ ] Boolean sentinels `albumsLoaded` and `likedLoaded` are removed
- [ ] Convenience `*Stale()` methods exist for all domains
- [ ] Library/stats data re-fetches after TTL expires; uses cached data within TTL
- [ ] `make ci` passes

## Tasks
- [ ] Add fetchedAt fields + IsStale() helper to internal/state/store.go
      - test: IsStale returns true for zero time; true when elapsed > TTL; false when elapsed < TTL
      - test: SetPlaylists() updates playlistsFetchedAt; SetTopTracks() updates statsFetchedAt[range]
- [ ] TTL constants + replace boolean sentinels with convenience *Stale() methods
      - test: PlaylistsStale() returns true when never fetched; true after TTL; false within TTL
      - test: removing boolean sentinels doesn't break existing tests
- [ ] Wire staleness checks into Update() + docs -- gate library and stats fetch commands
      - test: library data re-fetches after TTL; NOT re-fetched within TTL
      - test: stats re-fetch on stale re-open; NOT within TTL
      - test: switching away from stats and back within TTL uses cached data
