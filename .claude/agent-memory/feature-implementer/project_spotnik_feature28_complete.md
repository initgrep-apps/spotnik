---
name: Feature 28 (API Cleanup Follow-up)
description: Get prefix removal, TokenProvider interface, search.go import boundary fix
type: project
---

Feature 28 completed three deferred cleanup items from earlier architecture reviews.

**What was built:**

**Task 1 — Get prefix removal:**
- Renamed 10 methods: PlaybackState, Queue (player.go), Playlists, PlaylistTracks, SavedAlbums, LikedTracks, RecentlyPlayed (library.go), Devices (devices.go), TopTracks, TopArtists, RecentlyPlayed (user.go)
- Updated in this order: method definitions → interface files → apitest/mock.go → commands.go call sites → test files
- Verified `go build ./...` compiles after each batch before moving on

**Task 2 — TokenProvider interface:**
- Created `api/token.go` with `TokenProvider` interface + `StaticTokenProvider`
- `BaseClient.accessToken string` → `BaseClient.token TokenProvider`
- `newRequest()` calls `b.token.AccessToken(ctx)` per request
- `NewBaseClient(url, string)` wraps string in `StaticTokenProvider` — all constructors unchanged
- Added `NewBaseClientWithProvider(url, TokenProvider)` for future RefreshableTokenProvider
- `base_test.go` `newTestBaseClient` was creating `BaseClient{accessToken: ...}` directly — replaced with `NewBaseClient()` call
- `devices_test.go` had `client.accessToken` field access — changed to `client.token`

**Task 3 — search.go import boundary:**
- Added `SearchResultData` + 4 item types to `panes/messages.go`
- Updated `SearchResultsMsg` to carry `*SearchResultData` (was carrying just `Err`)
- `SearchOverlay` now has `results *SearchResultData` field; set in `SearchResultsMsg` case
- Removed all `o.store.SearchResults()` calls from search.go — replaced with `o.results`
- `commands.go` has `convertSearchResult(*api.SearchResult) *panes.SearchResultData` as the single crossing point
- `store.SetSearchResults`/`SearchResults` kept because `app.go` still uses it for `SearchClearedMsg` handler and `app_test.go` tests it
- Search tests rewritten to inject results via `SearchResultsMsg` instead of `store.SetSearchResults(&api.SearchResult{...})`
- `search_test.go` no longer imports `api` package

**Key lessons:**
- When rewriting test helpers that used `api.SearchResult`, just pass `SearchResultsMsg{Results: sampleData}` to the overlay directly — same Update() mechanism the real app uses
- `gofmt` failed on search_test.go after a Write — always `gofmt -w` or `make ci` before committing
- The store's `SearchResults`/`SetSearchResults` was kept because `app.go` uses `SetSearchResults(nil)` in the SearchClearedMsg handler (not removing from store, just not reading in overlay)
