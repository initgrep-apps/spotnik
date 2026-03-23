# ARCHITECTURE.md — Technical Reference

> **Reference only.** Feature specs embed the patterns you need inline.
> Consult this document when you need deeper context on a pattern the feature spec points to.
> Do not read cover-to-cover before implementing a feature.

---

## Architectural Overview

Spotnik follows the **Elm Architecture** as enforced by Bubble Tea. The entire application is a pure function of state: `View(State) → UI`. Side effects happen only through commands and messages.

```
┌──────────────────────────────────────────────────────────┐
│                        main.go                           │
│                    (entry point only)                    │
└─────────────────────────┬────────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────────┐
│                      cmd/root.go                         │
│         (flag parsing, config loading, auth check)       │
└─────────────────────────┬────────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────────┐
│                   internal/app/app.go                    │
│              Root Bubble Tea Model (tea.Model)           │
│   - Owns: all pane models, store ref, active pane state  │
│   - Routes: all messages to correct pane                 │
│   - Composes: final view from pane outputs               │
└──────┬────────────────┬──────────────────┬───────────────┘
       │                │                  │
┌──────▼──────┐  ┌──────▼──────┐  ┌───────▼──────┐
│  LibraryPane│  │  PlayerPane │  │  QueuePane   │
│  (tea.Model)│  │  (tea.Model)│  │  (tea.Model) │
└──────┬──────┘  └──────┬──────┘  └───────┬──────┘
       │                │                  │
       └────────────────▼──────────────────┘
                        │
              ┌─────────▼─────────┐
              │   internal/state  │
              │     Store         │
              │  (single source   │
              │   of truth)       │
              └─────────┬─────────┘
                        │
              ┌─────────▼─────────┐
              │   internal/api    │
              │  Spotify Client   │
              │  (HTTP only,      │
              │   no UI imports)  │
              └───────────────────┘
```

---

## Message Flow

```
User Keypress
     │
     ▼
app.Update(keyMsg)
     │
     ├── If global key (Tab, q, ?, d): handle in root
     │
     └── Else: delegate to active pane
              │
              ▼
         pane.Update(msg)
              │
              └── Returns (model, cmd)
                       │
                       ▼ (cmd executes)
                  tea.Cmd runs async
                       │
                       ▼
                  Returns tea.Msg
                       │
                       ▼
              app.Update(resultMsg)
                       │
                       └── Update store, re-render
```

---

## State Management

### The Store

`internal/state/store.go` is the single source of truth. All API data lives here. Panes **read** from the store but **never write** to it directly — they dispatch messages that the root model uses to update the store.

```go
// internal/state/store.go

// Store holds all application state. It is passed by reference to all panes.
// Panes must not hold API data themselves.
type Store struct {
    mu sync.RWMutex

    // Playback
    CurrentTrack    *api.Track        // URI available via CurrentTrack.URI — no separate field needed
    PlaybackState   *api.PlaybackState
    Queue           []api.Track
    RecentlyPlayed  []api.PlayHistory

    // Library
    Playlists        []api.SimplePlaylist
    SavedAlbums      []api.SavedAlbum
    LikedTracks      []api.SavedTrack
    PlaylistTracks   map[string][]api.Track  // keyed by playlist ID

    // User
    Profile          *api.User
    TopTracks        map[string][]api.Track   // keyed by time range
    TopArtists       map[string][]api.Artist  // keyed by time range

    // Devices
    Devices          []api.Device
    ActiveDevice     *api.Device

    // Search
    SearchResults    *api.SearchResult
    SearchQuery      string
    SearchLoading    bool

    // UI state
    ActivePane       PaneID
    ActiveView       ViewID
    ErrorMessage     string
    ErrorExpiry      time.Time
    IsLoading        bool
}

// Read methods — safe for concurrent use
func (s *Store) GetCurrentTrack() *api.Track { ... }
func (s *Store) GetPlaybackState() *api.PlaybackState { ... }
// ... etc

// Write methods — called only from app.Update()
func (s *Store) SetCurrentTrack(t *api.Track) { ... }
func (s *Store) SetPlaybackState(ps *api.PlaybackState) { ... }
// ... etc
```

### Message Types

Every distinct piece of data coming back from async operations has its own message type. Named consistently as `<noun><verb>Msg`.

```go
// internal/app/messages.go

// Playback messages (Feature 03)
type playbackStateFetchedMsg struct{ state *api.PlaybackState }
type playbackStateErrMsg      struct{ err error }
type playPausedMsg            struct{}
type trackSkippedMsg          struct{}
type volumeSetMsg             struct{ volume int }
type seekedMsg                struct{ positionMs int }
type tickMsg                  time.Time

// Auth messages (Feature 02)
type authSuccessMsg struct{}
type authErrMsg     struct{ err error }

// Library messages (Feature 04)
type libraryLoadedMsg         struct{ playlists []api.SimplePlaylist }
type libraryErrMsg            struct{ err error }
type playlistTracksLoadedMsg  struct{ id string; tracks []api.Track }
type playContextMsg           struct{ contextURI string }

// Search messages (Feature 05)
type searchDebounceMsg   struct{ query string }
type searchResultsMsg    struct{ results *api.SearchResult }
type searchErrMsg        struct{ err error }
type searchClosedMsg     struct{}

// Queue messages (Feature 06)
type queueLoadedMsg  struct{ tracks []api.Track }
type queueAddedMsg   struct{ uri string }
type queueAddErrMsg  struct{ err error }

// Device messages (Feature 07)
type devicesLoadedMsg       struct{ devices []api.Device }
type deviceTransferredMsg   struct{ deviceID string }

// Stats messages (Feature 08)
type statsLoadedMsg struct {
    topTracks  []api.Track
    topArtists []api.Artist
    timeRange  string
}
type statsTimeRangeChangedMsg struct{ timeRange string }

// Playlist manager messages (Feature 09)
type playlistCreatedMsg      struct{ playlist api.SimplePlaylist }
type playlistRenamedMsg      struct{ playlistID, newName string }
type playlistTracksAddedMsg  struct{ playlistID string; count int }

// System messages
type errorDismissMsg          struct{}
type terminalSizeMsg          struct{ width, height int }
```

---

## API Client Design

### Client Interface

The API client must be defined as an interface. This enables mocking in tests without any external libraries.

```go
// internal/api/client.go

// SpotifyClient defines all Spotify API operations used by spotnik.
// This interface is the only thing panes and state should depend on.
type SpotifyClient interface {
    // Playback
    GetPlaybackState(ctx context.Context) (*PlaybackState, error)
    GetCurrentlyPlaying(ctx context.Context) (*Track, error)
    Play(ctx context.Context, opts PlayOptions) error
    Pause(ctx context.Context) error
    Next(ctx context.Context) error
    Previous(ctx context.Context) error
    Seek(ctx context.Context, positionMs int) error
    SetVolume(ctx context.Context, volume int) error
    SetShuffle(ctx context.Context, state bool) error
    SetRepeat(ctx context.Context, mode string) error
    GetQueue(ctx context.Context) ([]Track, error)
    AddToQueue(ctx context.Context, uri string) error
    GetRecentlyPlayed(ctx context.Context, limit int) ([]PlayHistory, error)
    GetDevices(ctx context.Context) ([]Device, error)
    TransferPlayback(ctx context.Context, deviceID string, play bool) error

    // Library
    // Pagination methods return (items, error) — the total count is handled
    // internally by the fetchAll helper and not exposed in the interface.
    GetPlaylists(ctx context.Context, limit, offset int) ([]SimplePlaylist, error)
    GetPlaylistTracks(ctx context.Context, id string, limit, offset int) ([]Track, error)
    GetSavedAlbums(ctx context.Context, limit, offset int) ([]SavedAlbum, error)
    GetLikedTracks(ctx context.Context, limit, offset int) ([]SavedTrack, error)
    LikeTrack(ctx context.Context, id string) error
    UnlikeTrack(ctx context.Context, id string) error

    // Search
    Search(ctx context.Context, query string, types []string) (*SearchResult, error)

    // User
    GetProfile(ctx context.Context) (*User, error)
    GetTopTracks(ctx context.Context, timeRange string, limit int) ([]Track, error)
    GetTopArtists(ctx context.Context, timeRange string, limit int) ([]Artist, error)

    // Playlists
    CreatePlaylist(ctx context.Context, name, description string, public bool) (*Playlist, error)
    UpdatePlaylist(ctx context.Context, id, name, description string) error
    AddTracksToPlaylist(ctx context.Context, playlistID string, uris []string) error
    RemoveTracksFromPlaylist(ctx context.Context, playlistID string, uris []string) error
    ReorderPlaylistTracks(ctx context.Context, id string, rangeStart, insertBefore, rangeLength int) error
}
```

### HTTP Client Pattern

```go
// internal/api/client.go

type Client struct {
    baseURL    string
    httpClient *http.Client
    token      TokenProvider  // interface — allows mock in tests
}

// All requests go through this single method
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
    token, err := c.token.AccessToken(ctx)
    if err != nil {
        return nil, fmt.Errorf("getting access token: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
    if err != nil {
        return nil, fmt.Errorf("creating request: %w", err)
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("executing request: %w", err)
    }

    return c.checkResponse(resp)
}
```

### Pagination Pattern

Many Spotify endpoints return paginated results. Use this helper pattern consistently.

```go
// internal/api/pagination.go

// fetchAll fetches all pages of a paginated endpoint.
// fetchPage is a function that fetches one page given offset.
// maxItems is a safety cap (prevent runaway loops on large libraries).
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

---

## Auth Flow

### PKCE OAuth 2.0 (Authorization Code + Proof Key)

```
1. App starts → check keychain for existing valid token
2. Token exists + not expired → proceed to app
3. Token expired → attempt refresh with refresh_token
4. Refresh succeeds → update keychain, proceed
5. No token / refresh fails → start auth flow:
   a. Generate code_verifier (random 43-128 char string)
   b. Compute code_challenge = BASE64URL(SHA256(code_verifier))
   c. Start local HTTP server on random available port
   d. Open browser to Spotify auth URL with redirect_uri=http://localhost:{port}/callback
   e. Wait for callback with ?code= parameter
   f. Exchange code for tokens (with code_verifier)
   g. Store access_token + refresh_token + expiry in OS keychain
   h. Proceed to app
```

### Keychain Keys

| Key | Value |
|---|---|
| `spotnik:access_token` | Spotify access token |
| `spotnik:refresh_token` | Spotify refresh token |
| `spotnik:token_expiry` | Unix timestamp of expiry |

### Token Refresh Strategy

- Refresh **5 minutes before expiry** (proactive)
- On 401 response: refresh immediately, retry once
- On refresh failure: show re-auth prompt, don't crash

---

## Polling Architecture

Playback state must stay fresh. Use `tea.Tick` — never `time.Sleep`.

```go
// internal/app/app.go

// Init returns the initial command: start the polling ticker
func (m Model) Init() tea.Cmd {
    return tea.Batch(
        fetchPlaybackState(m.client),  // immediate first fetch
        tickEvery(time.Second),        // start 1s polling
        fetchLibrary(m.client),        // load library async
    )
}

// tickEvery returns a command that fires tickMsg every d
func tickEvery(d time.Duration) tea.Cmd {
    return tea.Every(d, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

// In Update, on tickMsg, re-fetch playback state and reschedule
case tickMsg:
    return m, tea.Batch(
        fetchPlaybackState(m.client),
        tickEvery(time.Second),
    )
```

### Polling Ownership

The root model's 1-second tick loop is the single polling mechanism in the app.

| Tick Cycle | Endpoint | Owner | Consumers |
|---|---|---|---|
| Every 1s | `GET /me/player` | Feature 03 (Playback) | Features 03, 04, 07, 08 |
| Every 1s | `GET /me/player/queue` | Feature 06 (Queue) | Feature 06 |

**Rules:**
- Feature 03 owns the tick loop and dispatches `fetchPlaybackState` on each `tickMsg`
- Feature 06 extends the tick to also dispatch `fetchQueue` alongside the playback fetch
- Features 04 (Library) and 08 (Stats) fetch their own data on-demand (init, section expand, view open) — they do NOT add to the tick loop
- Feature 07 (Devices) reads device info from the playback state response — it only fetches `GET /me/player/devices` when the device overlay opens
- No feature other than 03 and 06 should add recurring poll commands to the tick cycle

---

## Configuration

### Config Struct

```go
// internal/config/config.go

// Config represents the full user configuration.
// All fields must have sensible defaults so an empty config file works.
type Config struct {
    Spotify     SpotifyConfig     `toml:"spotify"`
    UI          UIConfig          `toml:"ui"`
    Keybindings KeybindingsConfig `toml:"keybindings"`
}

type SpotifyConfig struct {
    ClientID      string `toml:"client_id"`
    RefreshRateMs int    `toml:"refresh_rate_ms"` // default: 1000
}

type UIConfig struct {
    Theme string `toml:"theme"` // default: "black"
}

type KeybindingsConfig struct {
    // Optional overrides for default keybindings
    PlayPause string `toml:"play_pause"` // default: "space"
    // ... etc
}
```

### Config Loading

```go
func Load() (*Config, error) {
    cfg := defaults()

    path, err := configPath()
    if err != nil {
        return cfg, nil  // no config file is fine — use defaults
    }

    if _, err := os.Stat(path); os.IsNotExist(err) {
        return cfg, nil  // file doesn't exist — use defaults
    }

    if _, err := toml.DecodeFile(path, cfg); err != nil {
        return nil, fmt.Errorf("parsing config: %w", err)
    }

    return cfg, nil
}
```

---

## Testing Architecture

### Mock Client

```go
// internal/api/mock_client.go (used in tests only)

// MockClient implements SpotifyClient for testing.
// Set fields to control what each method returns.
type MockClient struct {
    PlaybackStateResult *PlaybackState
    PlaybackStateErr    error
    PlaylistsResult     []SimplePlaylist
    PlaylistsErr        error
    // ... one Result+Err pair per interface method

    // Call tracking
    PlayCalled    bool
    PauseCalled   bool
    NextCalled    bool
}

func (m *MockClient) GetPlaybackState(_ context.Context) (*PlaybackState, error) {
    return m.PlaybackStateResult, m.PlaybackStateErr
}
// ... implement all interface methods
```

### Pane Update Tests

```go
// internal/ui/panes/player_test.go

func TestPlayerPane_SpaceTogglesPlayback(t *testing.T) {
    client := &api.MockClient{
        PlaybackStateResult: &api.PlaybackState{IsPlaying: true},
    }
    store := state.NewStore()
    store.SetPlaybackState(client.PlaybackStateResult)

    pane := NewPlayerPane(store)
    _, cmd := pane.Update(tea.KeyMsg{Type: tea.KeySpace})

    require.NotNil(t, cmd)
    // Execute the command and verify it returns a pause message
    msg := cmd()
    assert.IsType(t, pauseMsg{}, msg)
    assert.True(t, client.PauseCalled)
}
```

### Integration Test Convention

Integration tests verify multi-component interactions: message routing through the root model,
state updates across panes, and end-to-end user workflows with mocked HTTP.

**File naming:** `*_integration_test.go` (e.g., `app_integration_test.go`, `player_integration_test.go`)

**Build tag:** Every integration test file starts with:
```go
//go:build integration
```

**Running tests:**
- `make test` — runs unit tests only (fast, default)
- `make test-integration` — runs integration tests only
- `make ci` — runs both unit and integration tests

**What qualifies as an integration test:**
- Tests that exercise message routing through the root `app.Model`
- Tests that verify state changes propagate from one pane to another
- Tests that combine `httptest.NewServer` with multiple model updates in sequence
- Tests that verify the polling tick produces correct downstream state changes

**What stays as a unit test:**
- Individual API client methods with `httptest.NewServer` (testing one function)
- Store mutation methods (Get/Set)
- Bubble Tea model `Update()` handlers (testing one key → one command)
- `View()` output assertions
- Config loading, PKCE helpers, time formatters

---

## Error Handling Conventions

### Error Handling in build*Cmd Functions

API errors in `build*Cmd` functions MUST be surfaced to the user via status bar or in-pane
error state. **Silent swallowing is prohibited.**

Pattern:
```go
// On failure — MUST set error state
store.SetXxxError(err)
return XxxLoadedMsg{err: err}

// On success — MUST clear error state
store.ClearXxxError()
store.SetXxx(data)
return XxxLoadedMsg{data: data}
```

Every pane that reads data from the store MUST check the corresponding error state in `View()`
and render an error view (using `components.RenderError`) instead of an empty state.

### Pane Rendering Constraints

`View()` output MUST NOT exceed the height set by `SetSize()`. Panes with unbounded content
(queue, library sections, search results) must implement viewport scrolling. Rendering all
items in a loop without height capping is a bug.

### User-Facing Errors

Errors shown in the status bar. Keep them short and actionable.

| Error | User Message |
|---|---|
| 401 (re-auth needed) | `Session expired. Run: spotnik auth` |
| 403 (no premium) | `Spotify Premium required for playback` |
| 429 (rate limited) | `Too many requests. Retrying in 5s...` |
| 503 (Spotify down) | `Spotify is unavailable. Retrying...` |
| Network error | `No connection to Spotify` |

### Error Display Lifecycle

```go
// Show error
m.store.SetError(msg)

// Schedule dismissal (4s)
return m, tea.Tick(4*time.Second, func(t time.Time) tea.Msg {
    return errorDismissMsg{}
})

// On dismiss
case errorDismissMsg:
    m.store.ClearError()
```

---

## Dependency Rules (Import Boundaries)

```
main.go
  └── cmd/
        └── internal/app/
              ├── internal/state/     ← reads store
              ├── internal/ui/        ← renders UI from store
              │     └── internal/ui/theme/
              ├── internal/api/       ← HTTP calls only
              ├── internal/config/    ← reads config
              └── internal/keychain/  ← token storage

FORBIDDEN IMPORTS:
  internal/api/   → internal/ui/    (API must not know about UI)
  internal/ui/    → internal/api/   (UI must not call API directly)
  internal/state/ → internal/ui/    (State must not know about UI)
  internal/state/ → internal/api/   (State must not call API)
```

---

## Build & Release

### Cross-Compilation Targets

```makefile
PLATFORMS = \
  linux/amd64 \
  linux/arm64 \
  darwin/amd64 \
  darwin/arm64 \
  windows/amd64
```

### Binary Size Optimization

```bash
go build -ldflags="-s -w" -trimpath ./...
# -s: omit symbol table
# -w: omit DWARF debug info
# -trimpath: remove build paths from binary
```

Target: under 15MB. Check with `ls -lh bin/spotnik` after each build.

---

*Last updated: 2026-03-23*
