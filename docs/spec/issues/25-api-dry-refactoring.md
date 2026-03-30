# Feature 25 — API DRY Refactoring

> **Refactoring:** Extract shared HTTP helpers into a base client, implement the generic
> `fetchAll[T]` pagination helper, remove `Get` prefix from getters, and add build
> tags to keychain tests.

## Context

The 6 API client files share ~150 lines of duplicated HTTP helper code (`newRequest`,
`doJSON`, `doNoContent`). The architecture spec defines a generic `fetchAll[T]` pagination
helper that was never implemented. Go convention (Effective Go) says getters should not
have a `Get` prefix. Keychain tests need integration build tags.

**Dependency:** This feature should be implemented AFTER Feature 23 (interfaces) and
Feature 24 (typed errors + TokenProvider), since those change the client constructors
and error handling that this feature builds on.

---

## Task 1: Extract shared baseClient

**Problem:** All 6 clients duplicate this pattern:
```go
type Client struct {
    baseURL     string
    accessToken string  // or TokenProvider after Feature 24
    client      *http.Client
}
```

And duplicate `newRequest()`, `doJSON()`, `doNoContent()` helper methods.

**Fix:** Create a shared `BaseClient` struct in `internal/api/base.go`:

```go
// internal/api/base.go

// BaseClient provides shared HTTP functionality for all API clients.
type BaseClient struct {
    BaseURL  string
    Token    TokenProvider
    HTTP     *http.Client
}

// NewBaseClient creates a BaseClient with sensible defaults.
func NewBaseClient(baseURL string, tp TokenProvider) BaseClient {
    if baseURL == "" {
        baseURL = spotifyAPIBaseURL
    }
    return BaseClient{
        BaseURL: baseURL,
        Token:   tp,
        HTTP:    &http.Client{},
    }
}

// NewRequest creates an authenticated HTTP request.
func (b *BaseClient) NewRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
    token, err := b.Token.AccessToken(ctx)
    if err != nil {
        return nil, fmt.Errorf("getting access token: %w", err)
    }
    req, err := http.NewRequestWithContext(ctx, method, b.BaseURL+path, body)
    if err != nil {
        return nil, fmt.Errorf("creating request: %w", err)
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")
    return req, nil
}

// DoJSON executes a request and decodes the JSON response into v.
func (b *BaseClient) DoJSON(ctx context.Context, method, path string, body io.Reader, v interface{}) error {
    // ... shared implementation with typed error returns (429, 403, 401)
}

// DoNoContent executes a request that expects no response body.
func (b *BaseClient) DoNoContent(ctx context.Context, method, path string, body io.Reader) error {
    // ... shared implementation
}
```

Refactor all 6 clients to embed `BaseClient`:
```go
type Player struct {
    BaseClient
}

func NewPlayer(baseURL string, tp TokenProvider) *Player {
    return &Player{BaseClient: NewBaseClient(baseURL, tp)}
}
```

**Files:**
- `internal/api/base.go` (new) — BaseClient struct + shared HTTP methods
- `internal/api/player.go` — Embed BaseClient, remove duplicated helpers
- `internal/api/library.go` — Same
- `internal/api/search.go` — Same
- `internal/api/devices.go` — Same
- `internal/api/user.go` — Same
- `internal/api/playlists.go` — Same

**Tests:**
- Unit test: BaseClient.NewRequest sets correct auth header
- Unit test: BaseClient.DoJSON returns typed errors for 401/403/429
- Existing client tests pass with BaseClient

---

## Task 2: Implement fetchAll[T] pagination helper

**Problem:** The architecture spec defines a generic `fetchAll[T]` but it was never
implemented. Each paginated call uses fixed limit/offset without exhaustive fetching.

**Fix:** Implement in `internal/api/pagination.go`:

```go
// internal/api/pagination.go

// fetchAll fetches all pages of a paginated endpoint.
// fetchPage returns (items, total, error) for a given offset.
// maxItems is a safety cap to prevent runaway loops.
func fetchAll[T any](ctx context.Context, maxItems int, fetchPage func(ctx context.Context, offset int) ([]T, int, error)) ([]T, error) {
    var all []T
    offset := 0
    for {
        items, total, err := fetchPage(ctx, offset)
        if err != nil {
            return nil, err
        }
        all = append(all, items...)
        offset += len(items)
        if offset >= total || offset >= maxItems || len(items) == 0 {
            break
        }
    }
    return all, nil
}
```

Update library client methods that currently use single-page fetches to optionally
use `fetchAll` where appropriate (e.g., `GetPlaylists` could offer a `GetAllPlaylists`
variant). Do NOT change existing method signatures — add new methods if needed.

**Files:**
- `internal/api/pagination.go` (new) — Generic fetchAll helper
- `internal/api/pagination_test.go` (new) — Tests

**Tests:**
- Unit test: fetchAll with 3 pages of data returns all items
- Unit test: fetchAll stops at maxItems cap
- Unit test: fetchAll handles empty first page
- Unit test: fetchAll propagates errors from fetchPage

---

## Task 3: Remove Get prefix from API getters

**Problem:** Go convention (Effective Go) says getters should not have a `Get` prefix.
All API client methods use `GetPlaybackState`, `GetPlaylists`, etc.

**Fix:** Rename all getter methods:
- `GetPlaybackState` → `PlaybackState`
- `GetPlaylists` → `Playlists`
- `GetPlaylistTracks` → `PlaylistTracks`
- `GetSavedAlbums` → `SavedAlbums`
- `GetLikedTracks` → `LikedTracks`
- `GetRecentlyPlayed` → `RecentlyPlayed`
- `GetDevices` → `Devices`
- `GetProfile` → `Profile`
- `GetTopTracks` → `TopTracks`
- `GetTopArtists` → `TopArtists`
- `GetQueue` → `Queue` (if it exists)

This is a mechanical rename. Update all callers in `app.go` / `commands.go`, test files,
and the interface definitions from Feature 23.

**Files:**
- All `internal/api/*.go` — Rename methods
- `internal/app/commands.go` (or `app.go`) — Update callers
- `internal/api/*_test.go` — Update test references
- Interface definitions — Update method names

**Tests:**
- All existing tests updated with new names
- `go build ./...` compiles clean

---

## Task 4: Add //go:build integration to keychain tests

**Problem:** `internal/keychain/keychain_test.go` tests hit the real OS keychain but
has no build tag. Tests use `t.Skipf` at runtime instead of build-tag exclusion.

**Fix:** Add `//go:build integration` as the first line of the file. Verify that
`make test` (which runs without `-tags integration`) skips these tests, and
`make ci` (which should include integration tests) runs them.

**Files:**
- `internal/keychain/keychain_test.go` — Add build tag

**Tests:**
- Verify `go test ./internal/keychain/` skips the file (no output)
- Verify `go test -tags integration ./internal/keychain/` runs the tests

---

## Acceptance Criteria

- [ ] `BaseClient` struct in `api/base.go` with shared HTTP helpers
- [ ] All 6 clients embed `BaseClient`, zero duplicated HTTP code
- [ ] `fetchAll[T]` generic helper in `api/pagination.go`
- [ ] Zero `Get` prefixed getter methods on API clients
- [ ] All callers updated with new method names
- [ ] `keychain_test.go` has `//go:build integration` tag
- [ ] All existing tests pass
- [ ] `make ci` passes
