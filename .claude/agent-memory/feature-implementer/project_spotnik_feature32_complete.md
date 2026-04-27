---
name: project_spotnik_feature32_complete
description: Feature 32 (Staleness Tracking): fetchedAt timestamps, IsStale helper, TTL constants, boolean sentinel removal, staleness-gated fetches
type: project
---

## Feature 32 — Staleness Tracking

**What was built:**
- `IsStale(fetchedAt, ttl)` package-level helper in `internal/state/store.go`
- 6 TTL constants: PlaylistsTTL/AlbumsTTL/LikedTracksTTL (5m), RecentlyPlayedTTL (2m), StatsTTL (10m), DevicesTTL (30s)
- `fetchedAt` timestamp fields in Store for all 6 domains
- All `Set*()` methods stamp `fetchedAt = time.Now()` on write
- Accessors: `PlaylistsFetchedAt/AlbumsFetchedAt/LikedTracksFetchedAt/RecentPlayedFetchedAt/StatsFetchedAt(range)/DevicesFetchedAt`
- `SetDevicesFetchedAt(t time.Time)` — called by DeviceOverlay.Update() after successful load
- 6 stale methods: `PlaylistsStale/AlbumsStale/LikedTracksStale/RecentlyPlayedStale/StatsStale(range)/DevicesStale`
- `albumsLoaded`/`likedLoaded` boolean sentinels removed; `AlbumsLoaded()`/`LikedLoaded()` derived from fetchedAt
- Staleness gates in `app.go handleMsg` for 5 fetch request types
- `internal/app/staleness_test.go` with 14 tests
- `docs/ARCHITECTURE.md` updated: Staleness Tracking section + updated Polling Architecture

**Key files:**
- `internal/state/store.go` — staleness infra (constants, IsStale, fields, accessors, stale methods)
- `internal/app/app.go` — 5 staleness gates in handleMsg (stats, playlists, albums, likedTracks, recentlyPlayed)
- `internal/ui/panes/library.go` — handleExpandSection uses AlbumsStale()/LikedTracksStale()
- `internal/app/staleness_test.go` — integration tests for 5 gated fetch types

**Patterns established:**
- Staleness gate pattern in app.go: `if m.Offset == 0 && !a.store.DomainStale() { return a, nil }`
- Paginated requests (offset > 0) bypass staleness gate — avoid incomplete data
- `SetDevicesFetchedAt` called explicitly by `DeviceOverlay.Update()` in `devicesLoadedMsg` success path; other domain timestamps stamped implicitly by `Set*()` data writers
- `AlbumsLoaded()`/`LikedLoaded()` kept public (backward compat), derived from `!fetchedAt.IsZero()`

**Gotchas:**
- `devicesLoadedMsg` unexported — fetchedAt stamp goes via `DeviceOverlay.Update()` calling `store.SetDevicesFetchedAt(time.Now())` on success; staleness gate in `app.go` (`FetchDevicesRequestMsg` handler checks `!a.store.DevicesStale()`)
- PR #37 review caught: `SetDevicesFetchedAt` not called in `devices.go` (dead code) + `FetchDevicesRequestMsg` had no staleness gate in `app.go` — both fixed in follow-up commit
- `StatsStale()` checks tracks vs artists separately via `SetTopTracks`/`SetTopArtists` both stamping same range key — both stamps happen, last write wins (fine, called in same cmd)
- `statsFetchedAt` map needs nil check in `StatsFetchedAt()` + `StatsStale()` accessors (uninitialized store)
- `TestFetchStatsMsg_WhenFresh_SkipsFetch` needs BOTH `SetTopTracks` AND `SetTopArtists` because either stamps range as fresh — one enough, both cleaner

**Testing notes:**
- Coverage: 82.7% total, state package 99.7%
- TDD: 19 store tests + 14 app integration tests, all written before implementation
- TTL constants regression guard (TestStalenessTTLConstants) — catches accidental value changes
- Can't fast-forward time without exposing internal timestamps; zero-time "never fetched" path tests stale dispatch