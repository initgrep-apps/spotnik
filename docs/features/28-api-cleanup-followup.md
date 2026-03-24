# Feature 28 — API Cleanup Follow-up

> **Refactoring:** Complete three items left incomplete from features 21, 24, and 25:
> remove Get prefix from API getters, introduce TokenProvider interface, and fix
> the search.go import boundary violation.

## Context

Features 19-27 addressed the architecture review findings. Three items were deferred
due to scope/risk:

1. **Get prefix on API getters** (Feature 25 Task 3) — Go convention violation. 10 methods
   named `GetX` should be `X`. Mechanical rename across 17 files, ~64 edits.
2. **TokenProvider interface** (Feature 24 Task 3) — Token stored as string in BaseClient.
   Architecture spec calls for per-request resolution via interface. Needed for future
   401 token refresh.
3. **search.go imports api/** (Feature 21 Task 2) — The only remaining `ui/ -> api/`
   import boundary violation. SearchOverlay reads `*api.SearchResult` from store and
   uses api types in clamped* render helpers.

---

## Task 1: Remove Get prefix from API getters

**Scope:** 10 methods across 4 client files + 4 interface files + 1 mock file + 10 call
sites in commands.go + ~27 call sites in test files. Total: 17 files, ~64 line edits.

**Renames:**

| Current | New | Client |
|---|---|---|
| `GetPlaybackState` | `PlaybackState` | Player |
| `GetQueue` | `Queue` | Player |
| `GetPlaylists` | `Playlists` | LibraryClient |
| `GetPlaylistTracks` | `PlaylistTracks` | LibraryClient |
| `GetSavedAlbums` | `SavedAlbums` | LibraryClient |
| `GetLikedTracks` | `LikedTracks` | LibraryClient |
| `GetRecentlyPlayed` | `RecentlyPlayed` | LibraryClient, UserClient |
| `GetDevices` | `Devices` | DevicesClient |
| `GetTopTracks` | `TopTracks` | UserClient |
| `GetTopArtists` | `TopArtists` | UserClient |

Note: `GetRecentlyPlayed` exists on both LibraryClient and UserClient — rename both.

**Files to update (complete list):**

Production code:
- `internal/api/player.go` — 2 method definitions
- `internal/api/library.go` — 5 method definitions
- `internal/api/devices.go` — 1 method definition
- `internal/api/user.go` — 2 method definitions (GetTopTracks, GetTopArtists, GetRecentlyPlayed)
- `internal/api/player_interfaces.go` — 2 interface methods
- `internal/api/library_interfaces.go` — 5 interface methods
- `internal/api/devices_interfaces.go` — 1 interface method
- `internal/api/user_interfaces.go` — 2 interface methods
- `internal/api/apitest/mock.go` — 10 mock method implementations
- `internal/app/commands.go` — 10 call sites

Test code:
- `internal/api/player_test.go` — ~5 call sites
- `internal/api/library_test.go` — ~13 call sites
- `internal/api/devices_test.go` — ~4 call sites
- `internal/api/user_test.go` — ~6 call sites

**Approach:** Use find-and-replace per method. Verify with `go build ./...` after each
batch. This is a mechanical rename — no behavioral change.

**Tests:**
- All existing tests pass with updated method names
- Compile-time interface checks still pass (interfaces renamed in sync)
- `go vet ./...` clean

---

## Task 2: Introduce TokenProvider interface

**Problem:** BaseClient stores `accessToken string` as a field set at construction time.
If the token expires mid-session, all requests fail until the app restarts. The
architecture spec calls for per-request token resolution via an interface.

**Current code (`internal/api/base.go`):**
```go
type BaseClient struct {
    baseURL     string
    accessToken string    // ← static string
    http        *http.Client
}

func (b *BaseClient) newRequest(...) {
    req.Header.Set("Authorization", "Bearer "+b.accessToken)  // ← uses stored string
}
```

**Fix:**

1. Create `internal/api/token.go`:
```go
// TokenProvider resolves an access token for each API request.
type TokenProvider interface {
    AccessToken(ctx context.Context) (string, error)
}

// StaticTokenProvider returns a fixed token. Used in tests and initial construction.
type StaticTokenProvider struct {
    Token string
}

func (s *StaticTokenProvider) AccessToken(_ context.Context) (string, error) {
    return s.Token, nil
}
```

2. Update `BaseClient` to use `TokenProvider`:
```go
type BaseClient struct {
    baseURL string
    token   TokenProvider    // ← interface instead of string
    http    *http.Client
}
```

3. Update `newRequest()` to call `b.token.AccessToken(ctx)` per-request.

4. Update `NewBaseClient(baseURL, accessToken string)` to wrap the string in
   `StaticTokenProvider` — preserves all existing constructor signatures:
```go
func NewBaseClient(baseURL, accessToken string) BaseClient {
    return BaseClient{
        baseURL: baseURL,
        token:   &StaticTokenProvider{Token: accessToken},
        http:    &http.Client{},
    }
}
```

5. Add `NewBaseClientWithProvider(baseURL string, tp TokenProvider)` for future use
   by RefreshableTokenProvider (Feature 27 follow-up).

**Files:**
- `internal/api/token.go` (new) — TokenProvider interface + StaticTokenProvider
- `internal/api/token_test.go` (new) — Tests for StaticTokenProvider
- `internal/api/base.go` — Change `accessToken string` to `token TokenProvider`, update `newRequest()`
- `internal/api/base_test.go` — Update tests (may need StaticTokenProvider in test setup)

No changes needed to client constructors — they all go through `NewBaseClient` which
wraps the string internally.

**Tests:**
- Unit test: StaticTokenProvider returns the fixed token
- Unit test: newRequest() calls TokenProvider.AccessToken and sets the Authorization header
- All existing client tests pass unchanged (NewBaseClient wraps string automatically)
- `make ci` passes

---

## Task 3: Fix search.go import boundary violation

**Problem:** `internal/ui/panes/search.go` imports `internal/api` to use
`api.SearchResult`, `api.Track`, `api.SearchArtist`, `api.SearchAlbum`,
`api.SearchPlaylist` in the clamped* render helper functions. This violates the
CLAUDE.md rule: "ui/ never imports api/."

**Current data flow:**
```
buildSearchCmd → api.Search() → store.SetSearchResults(*api.SearchResult)
SearchResultsMsg (empty signal) → SearchOverlay reads store.SearchResults()
SearchOverlay.clamped*() → uses api.* types for rendering
```

**Fix — carry pre-converted data on the message:**

1. Define UI-facing search types in `internal/ui/panes/messages.go`:
```go
type SearchResultData struct {
    Tracks    []SearchTrackItem
    Artists   []SearchArtistItem
    Albums    []SearchAlbumItem
    Playlists []SearchPlaylistItem
}

type SearchTrackItem struct {
    URI    string
    Name   string
    Artist string  // pre-formatted: first artist name
}

type SearchArtistItem struct {
    URI  string
    Name string
}

type SearchAlbumItem struct {
    URI    string
    Name   string
    Artist string  // pre-formatted: first artist name
}

type SearchPlaylistItem struct {
    URI   string
    Name  string
    Owner string  // pre-formatted: owner display name
}
```

2. Update `SearchResultsMsg` to carry the converted data:
```go
type SearchResultsMsg struct {
    Results *SearchResultData  // pre-converted from api.SearchResult
    Err     error
}
```

3. In `buildSearchCmd` (commands.go), convert `*api.SearchResult` to
   `*SearchResultData` before returning the message. The conversion extracts
   only the fields the UI needs (Name, URI, first artist name, owner display name).

4. Update `SearchOverlay` to:
   - Store `results *SearchResultData` as a local field (instead of reading from store)
   - On `SearchResultsMsg`, save `msg.Results` to the local field
   - Update all clamped* functions to use `SearchResultData` types
   - Remove the `api` import

5. The store can still hold `*api.SearchResult` for other consumers — the overlay
   just doesn't read it directly anymore. Alternatively, remove `SearchResults` from
   the store entirely if no other consumer reads it. Check for other callers of
   `store.SearchResults()` — if only search.go reads it, remove it from the store.

**Files:**
- `internal/ui/panes/messages.go` — Add SearchResultData types, update SearchResultsMsg
- `internal/ui/panes/search.go` — Store results locally, rewrite clamped* functions, remove api import
- `internal/app/commands.go` — Convert api.SearchResult to SearchResultData in buildSearchCmd
- `internal/app/app.go` — Update SearchResultsMsg handler (pass results from message)
- `internal/state/store.go` — Remove SearchResults/SetSearchResults if no longer needed
- `internal/ui/panes/search_test.go` — Update tests with new types

**Tests:**
- Unit test: SearchOverlay renders correctly with SearchResultData
- Unit test: conversion from api.SearchResult to SearchResultData is correct
- Unit test: SearchResultsMsg carries results to overlay
- Verify: `grep -r 'internal/api' internal/ui/panes/search.go` returns nothing
- All existing search tests pass
- `make ci` passes

---

## Verification

After all three tasks, run:
```bash
grep -r '"github.com/initgrep-apps/spotnik/internal/api"' internal/ui/
```
This must return zero results (devices.go was already fixed in Feature 21).

---

## Acceptance Criteria

- [ ] Zero API getter methods start with `Get`
- [ ] All interfaces, mocks, and callers updated with new names
- [ ] `TokenProvider` interface exists in `api/token.go`
- [ ] `BaseClient` uses `TokenProvider` per-request, not stored string
- [ ] `StaticTokenProvider` provides backward compatibility
- [ ] `search.go` has zero imports from `internal/api`
- [ ] `SearchResultData` UI types carry data via messages, not store reads
- [ ] `grep -r 'internal/api' internal/ui/` returns zero results
- [ ] All existing tests pass
- [ ] `make ci` passes
