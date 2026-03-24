# Feature 23 — API Client Interfaces & Mocks

> **Refactoring:** Define a `SpotifyClient` interface (or per-domain interfaces) for the
> API layer, create mock implementations, and remove the nil-guard workaround in
> command builders.

## Context

The architecture spec (`docs/ARCHITECTURE.md`) calls for a `SpotifyClient` interface
but none exists in the codebase. Instead, `app.go` uses concrete client types
(`*api.Player`, `*api.LibraryClient`, etc.) and nil-guards in every `build*Cmd`
function:

```go
if client == nil {
    return XxxResultMsg{}  // empty success — hides errors
}
```

This prevents:
- App-level integration tests with injected fakes
- Testing error handling (401/429/403) at the app level
- Detecting missing client initialization at compile time

---

## Task 1: Define per-domain interfaces in api/

The codebase has 6 distinct API clients. Define an interface for each, co-located
with the client implementation. This follows Go convention of defining interfaces
where they are used, but since multiple consumers exist (app.go, future tests),
placing them in `api/` is appropriate.

**Interfaces to define:**

```go
// internal/api/player.go
type PlayerAPI interface {
    GetPlaybackState(ctx context.Context) (*PlaybackState, error)
    Play(ctx context.Context, opts PlayOptions) error
    Pause(ctx context.Context) error
    Next(ctx context.Context) error
    Previous(ctx context.Context) error
    Seek(ctx context.Context, positionMs int) error
    SetVolume(ctx context.Context, volume int) error
    SetShuffle(ctx context.Context, state bool) error
    SetRepeat(ctx context.Context, mode string) error
}

// internal/api/library.go
type LibraryAPI interface {
    GetPlaylists(ctx context.Context, limit, offset int) ([]SimplePlaylist, int, error)
    GetPlaylistTracks(ctx context.Context, id string, limit, offset int) ([]Track, int, error)
    GetSavedAlbums(ctx context.Context, limit, offset int) ([]SavedAlbum, int, error)
    GetLikedTracks(ctx context.Context, limit, offset int) ([]SavedTrack, int, error)
    GetRecentlyPlayed(ctx context.Context, limit int) ([]PlayHistory, error)
    LikeTrack(ctx context.Context, id string) error
    UnlikeTrack(ctx context.Context, id string) error
}

// internal/api/search.go
type SearchAPI interface {
    Search(ctx context.Context, query string, types []string) (*SearchResult, error)
}

// internal/api/devices.go
type DevicesAPI interface {
    GetDevices(ctx context.Context) ([]Device, error)
    TransferPlayback(ctx context.Context, deviceID string, play bool) error
}

// internal/api/user.go
type UserAPI interface {
    GetProfile(ctx context.Context) (*User, error)
    GetTopTracks(ctx context.Context, timeRange string, limit int) ([]Track, error)
    GetTopArtists(ctx context.Context, timeRange string, limit int) ([]Artist, error)
}

// internal/api/playlists.go
type PlaylistsAPI interface {
    CreatePlaylist(ctx context.Context, name, description string, public bool) (*SimplePlaylist, error)
    RenamePlaylist(ctx context.Context, id, name string) error
    AddTracksToPlaylist(ctx context.Context, playlistID string, uris []string) error
    RemoveTracksFromPlaylist(ctx context.Context, playlistID string, uris []string) error
    ReorderPlaylistTracks(ctx context.Context, id string, rangeStart, insertBefore, rangeLength int) error
}
```

**Important:** Match the exact method signatures of the existing concrete clients.
Read each client file to get the correct signatures — the signatures above are
approximations from the architecture spec and may not match the actual code.

Add compile-time interface satisfaction checks:
```go
var _ PlayerAPI = (*Player)(nil)
```

**Files:**
- `internal/api/player.go` — Add `PlayerAPI` interface + compile check
- `internal/api/library.go` — Add `LibraryAPI` interface + compile check
- `internal/api/search.go` — Add `SearchAPI` interface + compile check
- `internal/api/devices.go` — Add `DevicesAPI` interface + compile check
- `internal/api/user.go` — Add `UserAPI` interface + compile check
- `internal/api/playlists.go` — Add `PlaylistsAPI` interface + compile check

**Tests:**
- Compile-time checks verify interface satisfaction
- Existing tests unchanged

---

## Task 2: Update app.go to use interfaces

**Fix:** Change the `App` struct fields from concrete types to interface types:

```go
type App struct {
    player    api.PlayerAPI       // was *api.Player
    library   api.LibraryAPI      // was *api.LibraryClient
    search    api.SearchAPI       // was *api.SearchClient
    devices   api.DevicesAPI      // was *api.DevicesClient
    user      api.UserAPI         // was *api.UserClient
    playlists api.PlaylistsAPI    // was *api.PlaylistsClient
    // ...
}
```

Update the `Set*` methods (e.g., `SetPlayer`, `SetLibrary`) to accept the interface
types. Update `cmd/root.go` where clients are constructed and passed to the app.

**Files:**
- `internal/app/app.go` — Change struct fields to interface types
- `internal/app/commands.go` — Update type references (if any after Feature 22)
- `cmd/root.go` — Update client construction (concrete types still implement interfaces)

**Tests:**
- All existing tests must pass — concrete types satisfy the interfaces

---

## Task 3: Create mock clients for testing

**Fix:** Create mock implementations of each interface in test files:

```go
// internal/api/mock_player_test.go (or internal/api/mocks_test.go)
type MockPlayer struct {
    PlaybackStateResult *PlaybackState
    PlaybackStateErr    error
    PlayCalled          bool
    PauseCalled         bool
    // ... one result+err pair per method, one Called bool per mutating method
}

func (m *MockPlayer) GetPlaybackState(_ context.Context) (*PlaybackState, error) {
    return m.PlaybackStateResult, m.PlaybackStateErr
}
// ... all interface methods
```

Since these mocks are needed by `app/` tests too, place them in a non-test file
that can be imported: `internal/api/mock_client.go` (as described in ARCHITECTURE.md).

**Files:**
- `internal/api/mock_client.go` (new) — Mock implementations for all 6 interfaces

**Tests:**
- Compile-time checks: `var _ PlayerAPI = (*MockPlayer)(nil)` etc.

---

## Task 4: Remove nil-guard pattern from build*Cmd functions

**Fix:** With interfaces in place, the nil-guard pattern (`if client == nil { return empty }`)
is no longer needed. Remove it from all 18 `build*Cmd` functions.

If a client is nil at runtime, it means initialization failed — this should be caught
at startup, not silently ignored per-request.

**Files:**
- `internal/app/commands.go` (after Feature 22) or `internal/app/app.go`

**Tests:**
- Existing tests pass
- Add a test that verifies a build*Cmd function panics or returns error when client is nil
  (optional — the real fix is ensuring clients are always initialized)

---

## Acceptance Criteria

- [ ] 6 per-domain interfaces defined in `internal/api/`
- [ ] Compile-time interface checks for all 6 concrete clients
- [ ] `App` struct uses interface types, not concrete types
- [ ] `MockPlayer`, `MockLibrary`, etc. exist in `internal/api/mock_client.go`
- [ ] Nil-guard pattern removed from all build*Cmd functions
- [ ] All existing tests pass
- [ ] `make ci` passes
