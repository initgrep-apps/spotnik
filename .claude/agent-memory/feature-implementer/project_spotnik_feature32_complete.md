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
- Accessor methods: `PlaylistsFetchedAt/AlbumsFetchedAt/LikedTracksFetchedAt/RecentPlayedFetchedAt/StatsFetchedAt(range)/DevicesFetchedAt`
- `SetDevicesFetchedAt(t time.Time)` — called by DeviceOverlay.Update() after successful load
- 6 convenience stale methods: `PlaylistsStale/AlbumsStale/LikedTracksStale/RecentlyPlayedStale/StatsStale(range)/DevicesStale`
- `albumsLoaded` and `likedLoaded` boolean sentinel struct fields removed; `AlbumsLoaded()`/`LikedLoaded()` now derived from fetchedAt
- Staleness gates in `app.go handleMsg` for 5 fetch request types
- `internal/app/staleness_test.go` with 14 tests
- `docs/ARCHITECTURE.md` updated with Staleness Tracking section and updated Polling Architecture

**Key files:**
- `internal/state/store.go` — all staleness infrastructure (constants, IsStale, fields, accessors, stale methods)
- `internal/app/app.go` — 5 staleness gates in handleMsg (stats, playlists, albums, likedTracks, recentlyPlayed)
- `internal/ui/panes/library.go` — handleExpandSection updated to use AlbumsStale()/LikedTracksStale()
- `internal/app/staleness_test.go` — integration tests for all 5 gated fetch types

**Patterns established:**
- Staleness gate pattern in app.go: `if m.Offset == 0 && !a.store.DomainStale() { return a, nil }`
- Paginated requests (offset > 0) always bypass the staleness gate to avoid incomplete data
- `SetDevicesFetchedAt` is called explicitly by `DeviceOverlay.Update()` in the `devicesLoadedMsg` success path — all other domain timestamps are stamped implicitly by their `Set*()` data writers
- `AlbumsLoaded()`/`LikedLoaded()` kept as public methods (backward compat) but now derived from `!fetchedAt.IsZero()`

**Gotchas:**
- `devicesLoadedMsg` is unexported — the fetchedAt stamp goes through `DeviceOverlay.Update()` calling `store.SetDevicesFetchedAt(time.Now())` on the success path; the staleness gate lives in `app.go` (`FetchDevicesRequestMsg` handler checks `!a.store.DevicesStale()`)
- PR #37 review found that `SetDevicesFetchedAt` was not being called in `devices.go` (dead code) and `FetchDevicesRequestMsg` had no staleness gate in `app.go` — both were fixed in a follow-up commit
- `StatsStale()` checks just tracks vs. artists separately via `SetTopTracks`/`SetTopArtists` both stamping the same range key — both stamps happen so the last write wins (fine since they're called in the same cmd)
- `statsFetchedAt` map needs nil check in `StatsFetchedAt()` and `StatsStale()` accessors (uninitialized store)
- The existing `TestFetchStatsMsg_WhenFresh_SkipsFetch` test needs BOTH `SetTopTracks` AND `SetTopArtists` because either one stamps the range as fresh — setting just one is enough but setting both is cleaner

**Testing notes:**
- Coverage: 82.7% total, state package 99.7%
- TDD: 19 store tests + 14 app integration tests, all written before implementation
- Tests for TTL constants as a regression guard (TestStalenessTTLConstants) — useful catch if someone accidentally changes a value
- Can't fast-forward time in tests without exposing internal timestamps; the zero-time "never fetched" path is used to test the stale dispatch path
