---
name: Feature 28 (API Cleanup Follow-up)
description: Get prefix removal, TokenProvider interface, search.go import boundary fix
type: project
---

Feature 28: 3 deferred cleanup items from prior arch reviews.

**Built:**

**Task 1 — Get prefix removal:**
- Renamed 10 methods: PlaybackState, Queue (player.go), Playlists, PlaylistTracks, SavedAlbums, LikedTracks, RecentlyPlayed (library.go), Devices (devices.go), TopTracks, TopArtists, RecentlyPlayed (user.go)
- Order: method defs → interface files → apitest/mock.go → commands.go call sites → tests
- Verified `go build ./...` per batch before next

**Task 2 — TokenProvider interface:**
- Made `api/token.go` w/ `TokenProvider` interface + `StaticTokenProvider`
- `BaseClient.accessToken string` → `BaseClient.token TokenProvider`
- `newRequest()` calls `b.token.AccessToken(ctx)` per request
- `NewBaseClient(url, string)` wraps string in `StaticTokenProvider` — constructors unchanged
- Added `NewBaseClientWithProvider(url, TokenProvider)` for future RefreshableTokenProvider
- `base_test.go` `newTestBaseClient` made `BaseClient{accessToken: ...}` direct — swapped to `NewBaseClient()` call
- `devices_test.go` had `client.accessToken` access — changed to `client.token`

**Task 3 — search.go import boundary:**
- Added `SearchResultData` + 4 item types to `panes/messages.go`
- `SearchResultsMsg` now carries `*SearchResultData` (was just `Err`)
- `SearchOverlay` now has `results *SearchResultData` field; set in `SearchResultsMsg` case
- Dropped all `o.store.SearchResults()` calls from search.go — swapped w/ `o.results`
- `commands.go` has `convertSearchResult(*api.SearchResult) *panes.SearchResultData` as sole crossing point
- `store.SetSearchResults`/`SearchResults` kept — `app.go` still uses for `SearchClearedMsg` handler, `app_test.go` tests it
- Search tests rewritten: inject via `SearchResultsMsg` vs `store.SetSearchResults(&api.SearchResult{...})`
- `search_test.go` no longer imports `api` pkg

**Lessons:**
- Test helpers using `api.SearchResult` → pass `SearchResultsMsg{Results: sampleData}` to overlay direct — same Update() path real app uses
- `gofmt` failed on search_test.go post-Write — always `gofmt -w` or `make ci` pre-commit
- Store's `SearchResults`/`SetSearchResults` kept: `app.go` uses `SetSearchResults(nil)` in SearchClearedMsg handler (not removing from store, just not reading in overlay)