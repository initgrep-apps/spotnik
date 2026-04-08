# Phase 2 — Structural Refactors Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract `StateReader` interface, split `gateway.go` and `app.go` into focused files, and standardise API tests to table-driven style — all with zero behaviour change.

**Architecture:** Pure restructuring. No new logic, no API surface changes. All existing tests must pass unchanged after each task.

**Tech Stack:** Go 1.22+, `internal/state`, `internal/api`, `internal/app`.

**Prerequisite:** Phase 1 branch merged to `main` before starting.

---

## File Map

| Action | Path | Purpose |
|--------|------|---------|
| Create | `internal/state/reader.go` | `StateReader` read-only interface |
| Create | `internal/api/gateway_bucket.go` | `tokenBucket` extracted from gateway.go |
| Create | `internal/api/gateway_dedup.go` | `inflightEntry` extracted from gateway.go |
| Create | `internal/app/handlers.go` | `handleMsg()` extracted from app.go |
| Create | `internal/app/prefs.go` | Preference flush logic extracted from app.go |
| Modify | `internal/api/gateway.go` | Remove bucket and dedup types (moved to new files) |
| Modify | `internal/app/app.go` | Remove handleMsg and prefs logic (moved to new files) |
| Modify | `internal/ui/panes/*.go` (8 files) | Accept `state.StateReader` instead of `*state.Store` |
| Modify | `internal/api/apitest/mock.go` | No change needed (mocks implement API interfaces, not Store) |
| Modify | `internal/api/player_test.go` | Refactor to table-driven |
| Modify | `internal/api/library_test.go` | Refactor to table-driven |
| Modify | `internal/api/playlists_test.go` | Refactor to table-driven |

---

### Task 1: Create branch

- [ ] **Step 1: Create and switch to feature branch**

```bash
git checkout main && git pull origin main
git checkout -b refactor/audit-phase2-structure
```

---

### Task 2: Extract StateReader interface

**Files:**
- Create: `internal/state/reader.go`

The `StateReader` interface covers every **read-only** method on `*Store` that panes
actually call. Write-only methods (`Set*`, `Clear*`, `Stamp*`) and sentinel setters
stay on the concrete `*Store` only — `app.go` holds `*Store` directly.

- [ ] **Step 1: Identify which methods panes actually call**

```bash
grep -r "\.store\." internal/ui/panes/ | grep -v "_test.go" | grep -oP '\.\w+\(' | sort -u
```

Record the output — these are the methods to include in StateReader.

- [ ] **Step 2: Write the interface**

Create `internal/state/reader.go`:

```go
// Package state provides the central Store and StateReader interface.
package state

import (
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
)

// StateReader is the read-only view of the Store. Panes and components depend on
// this interface instead of *Store directly, making unit tests lighter — a test
// only needs to provide the data the pane actually reads.
//
// app.go holds a *Store (which satisfies StateReader) and remains the sole writer.
type StateReader interface {
	// Playback
	PlaybackState() *domain.PlaybackState
	ActiveDevice() *domain.Device
	UserID() string

	// Queue
	Queue() []domain.Track

	// Library
	Playlists() []domain.SimplePlaylist
	PlaylistsTotal() int
	PlaylistTracks(playlistID string) []domain.Track
	PlayingPlaylistID() string
	SavedAlbums() []domain.SavedAlbum
	AlbumsLoaded() bool
	LikedTracks() []domain.SavedTrack
	LikedTotal() int
	LikedLoaded() bool
	RecentlyPlayed() []domain.PlayHistory

	// Stats
	TopTracks(timeRange string) []domain.Track
	TopArtists(timeRange string) []domain.FullArtist

	// Devices
	Devices() []domain.Device

	// Staleness (panes read these to decide whether to show a loading indicator)
	PlaylistsStale() bool
	AlbumsStale() bool
	LikedTracksStale() bool
	RecentlyPlayedStale() bool
	StatsStale(timeRange string) bool
	DevicesStale() bool

	// Gateway event log (read by NetworkLogPane and RequestFlowPane)
	ReadEventsFrom(cursor uint64) ([]domain.GatewayEvent, uint64)

	// Throttle observability (read by status bar)
	IsThrottled() bool
	ThrottleRetryAfterSecs() int
}

// Compile-time assertion: *Store must satisfy StateReader.
// This fails to compile if a method is added to StateReader but not implemented on Store.
var _ StateReader = (*Store)(nil)
```

> **Note:** If `go build` fails after adding the compile-time assertion, it means
> `*Store` is missing one of the listed methods. Check the method name spelling
> against `store.go` and fix the interface.

- [ ] **Step 3: Build to verify**

```bash
go build ./internal/state/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/state/reader.go
git commit -m "refactor(state): add StateReader read-only interface with compile-time assertion"
```

---

### Task 3: Update pane constructors to accept StateReader

**Files:**
- Modify: `internal/ui/panes/nowplaying.go`, `queue.go`, `playlists_pane.go`, `albums_pane.go`, `likedsongs_pane.go`, `recentlyplayed_pane.go`, `toptracks_pane.go`, `topartists_pane.go`

For **each** of the 8 pane files:

- [ ] **Step 1: Change the struct field type**

Find:
```go
store   *state.Store
```

Replace with:
```go
store   state.StateReader
```

- [ ] **Step 2: Change the constructor signature**

Find (example for QueuePane):
```go
func NewQueuePane(store *state.Store, th theme.Theme, focused bool) *QueuePane {
```

Replace with:
```go
func NewQueuePane(store state.StateReader, th theme.Theme, focused bool) *QueuePane {
```

Apply the same change for all 8 pane constructors.

- [ ] **Step 3: Build to check for cascading errors**

```bash
go build ./internal/...
```

If `app.go` fails: it passes a `*state.Store` to pane constructors. Since `*Store`
satisfies `StateReader`, this should compile without changes to `app.go`.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/ui/panes/... -race -count=1
```

Expected: all tests pass. Pane test helpers construct `state.New()` which returns
`*Store`, which satisfies `StateReader` — no test changes needed.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/panes/
git commit -m "refactor(panes): accept state.StateReader instead of *state.Store in all 8 pane constructors"
```

---

### Task 4: Verify import cleanliness

- [ ] **Step 1: Confirm no pane imports state for writes**

```bash
grep -r '"github.com/initgrep-apps/spotnik/internal/state"' internal/ui/panes/ | grep -v "_test.go"
```

Each pane file that imports `state` should only use `state.StateReader` (the interface),
not `state.New()` or `state.Store{}`. Test files may still use `state.New()` to build
a concrete store for test setup — that is correct.

- [ ] **Step 2: Run full test suite**

```bash
make test
```

Expected: all tests pass.

- [ ] **Step 3: Commit (if any cleanup was needed)**

```bash
git add -p
git commit -m "refactor(panes): clean up state import usage after StateReader migration"
```

---

### Task 5: Split gateway.go

**Files:**
- Create: `internal/api/gateway_bucket.go`
- Create: `internal/api/gateway_dedup.go`
- Modify: `internal/api/gateway.go`

First, read the current `gateway.go` to identify the `tokenBucket` and `inflightEntry` types:

```bash
grep -n "^type tokenBucket\|^type inflightEntry\|^func (b \*tokenBucket\|^func (e \*inflightEntry" internal/api/gateway.go
```

- [ ] **Step 1: Move tokenBucket to gateway_bucket.go**

Create `internal/api/gateway_bucket.go` containing:
- The `tokenBucket` struct definition
- All methods with receiver `*tokenBucket`

The file header:
```go
// Package api — token bucket rate limiter for the API gateway.
// A classic token-bucket (10 tokens/second, burst of 10) limits total request
// throughput. Background requests drain the bucket; Interactive requests bypass it.
package api
```

- [ ] **Step 2: Move inflightEntry to gateway_dedup.go**

Create `internal/api/gateway_dedup.go` containing:
- The `inflightEntry` struct definition
- All methods with receiver `*inflightEntry` (if any)

The file header:
```go
// Package api — in-flight request deduplication for the API gateway.
// When two goroutines issue a GET to the same (Method, Path) key simultaneously,
// only one HTTP call is made. The second waiter receives a copy of the response body.
package api
```

- [ ] **Step 3: Remove moved types from gateway.go**

Delete the `tokenBucket` struct, its methods, `inflightEntry` struct, and its methods
from `gateway.go`. Keep everything else (Gateway struct, `New()`, `Do()`, priority
context helpers, `emitEvent*` functions).

- [ ] **Step 4: Build**

```bash
go build ./internal/api/...
```

Expected: no errors. All types are still in the same package — Go sees them all.

- [ ] **Step 5: Run gateway tests**

```bash
go test ./internal/api/... -run TestGateway -race -count=1 -v
```

Expected: all gateway tests pass. No changes to test files needed.

- [ ] **Step 6: Commit**

```bash
git add internal/api/gateway.go internal/api/gateway_bucket.go internal/api/gateway_dedup.go
git commit -m "refactor(api): split gateway.go into gateway, gateway_bucket, gateway_dedup"
```

---

### Task 6: Split app.go

**Files:**
- Create: `internal/app/handlers.go`
- Create: `internal/app/prefs.go`
- Modify: `internal/app/app.go`

First, identify the boundary points:

```bash
grep -n "^func (a \*App) handleMsg\|^func (a \*App) schedule\|^func (a \*App) Prefs\|^func (a \*App) flushPrefs" internal/app/app.go
```

- [ ] **Step 1: Move handleMsg to handlers.go**

Create `internal/app/handlers.go`:

```go
// Package app — message handlers for the root Bubble Tea model.
// handleMsg is the central dispatch function called by Update() for every
// non-key, non-mouse message. It routes data-carrying Msg payloads to Store
// writes and returns any follow-up commands.
package app
```

Move the entire `handleMsg(msg tea.Msg) (App, tea.Cmd)` function (and any private
helper functions it calls that are not used elsewhere in app.go) into this file.

- [ ] **Step 2: Move preference logic to prefs.go**

Create `internal/app/prefs.go`:

```go
// Package app — preference persistence for the root model.
// Preferences (theme, preset, visualizer) are flushed to disk with a debounce
// to avoid excessive writes during rapid key presses. A generation counter
// discards stale timers if the preference changes again before flush fires.
package app
```

Move to this file:
- `schedulePrefsFlush()` method
- `PrefsDirtyGen()` method (if it exists as a standalone method)
- The `prefsFlushMsg` type (if defined locally in app.go)
- The `flushPrefsCmd` or equivalent flush function

- [ ] **Step 3: Remove moved functions from app.go**

Delete `handleMsg`, `schedulePrefsFlush`, and related helpers from `app.go`.
Keep in `app.go`: struct definition, `New()`, `Init()`, `Update()` entry point,
tick handling, and `isIdle()` / `pollIntervals()`.

- [ ] **Step 4: Build**

```bash
go build ./internal/app/...
```

Expected: no errors.

- [ ] **Step 5: Run all app tests**

```bash
go test ./internal/app/... -race -count=1
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/app/app.go internal/app/handlers.go internal/app/prefs.go
git commit -m "refactor(app): split app.go into app, handlers, prefs (no behaviour change)"
```

---

### Task 7: Refactor player_test.go to table-driven

**Files:**
- Modify: `internal/api/player_test.go`

- [ ] **Step 1: Identify candidates**

```bash
grep -n "^func Test" internal/api/player_test.go
```

Look for test functions that test the same function under multiple conditions
(different options, status codes, or response shapes) using separate `TestX_YYY`
names. These are the candidates for table consolidation.

- [ ] **Step 2: Consolidate into table-driven tests**

For each group of related tests (e.g., `TestPlayer_Play_WithContext`,
`TestPlayer_Play_WithURI`, `TestPlayer_Play_WithOffset`), merge into one table:

```go
func TestPlayer_Play(t *testing.T) {
    tests := []struct {
        name        string
        opts        PlayOptions
        wantBody    string // substring to assert in request body
        wantStatus  int
        wantErr     bool
    }{
        {
            name:       "play with context URI",
            opts:       PlayOptions{ContextURI: "spotify:playlist:abc"},
            wantBody:   `"context_uri":"spotify:playlist:abc"`,
            wantStatus: http.StatusNoContent,
        },
        {
            name:       "play with track URI",
            opts:       PlayOptions{URIs: []string{"spotify:track:xyz"}},
            wantBody:   `"uris":["spotify:track:xyz"]`,
            wantStatus: http.StatusNoContent,
        },
        {
            name:       "server error returns error",
            opts:       PlayOptions{},
            wantStatus: http.StatusInternalServerError,
            wantErr:    true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                body, _ := io.ReadAll(r.Body)
                if tt.wantBody != "" {
                    assert.Contains(t, string(body), tt.wantBody)
                }
                w.WriteHeader(tt.wantStatus)
            }))
            defer srv.Close()

            client := NewPlayer(srv.URL, "test-token")
            err := client.Play(context.Background(), tt.opts)
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

Apply this pattern to all candidate test groups in `player_test.go`.

- [ ] **Step 3: Run tests**

```bash
go test ./internal/api/... -run TestPlayer -race -count=1 -v
```

Expected: all player tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/api/player_test.go
git commit -m "test(api): refactor player_test.go to table-driven style"
```

---

### Task 8: Refactor library_test.go to table-driven

**Files:**
- Modify: `internal/api/library_test.go`

- [ ] **Step 1: Identify candidates**

```bash
grep -n "^func Test" internal/api/library_test.go
```

- [ ] **Step 2: Consolidate pagination and error cases**

For functions tested with multiple offset/limit combinations or error status codes,
merge into one table. Example for `GetSavedAlbums`:

```go
func TestLibrary_GetSavedAlbums(t *testing.T) {
    fixture := testhelpers.LoadFixture(t, "saved_albums_response.json")

    tests := []struct {
        name       string
        offset     int
        limit      int
        handler    http.HandlerFunc
        wantLen    int
        wantErr    bool
    }{
        {
            name:   "returns albums from fixture",
            offset: 0, limit: 50,
            handler: func(w http.ResponseWriter, r *http.Request) {
                assert.Equal(t, "0", r.URL.Query().Get("offset"))
                assert.Equal(t, "50", r.URL.Query().Get("limit"))
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                _, _ = w.Write(fixture)
            },
            wantLen: 1, // based on fixture contents
        },
        {
            name:   "401 returns UnauthorizedError",
            offset: 0, limit: 50,
            handler: func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusUnauthorized)
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            srv := httptest.NewServer(tt.handler)
            defer srv.Close()
            client := newTestLibrary(srv.URL, "test-token")
            albums, err := client.GetSavedAlbums(context.Background(), tt.offset, tt.limit)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Len(t, albums, tt.wantLen)
        })
    }
}
```

Apply this pattern to all candidate groups.

- [ ] **Step 3: Run tests**

```bash
go test ./internal/api/... -run TestLibrary -race -count=1 -v
```

Expected: all library tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/api/library_test.go
git commit -m "test(api): refactor library_test.go to table-driven style"
```

---

### Task 9: Refactor playlists_test.go to table-driven

**Files:**
- Modify: `internal/api/playlists_test.go`

- [ ] **Step 1: Identify candidates**

```bash
grep -n "^func Test" internal/api/playlists_test.go
```

- [ ] **Step 2: Consolidate variant tests**

Apply the same table-driven pattern. For create/rename/remove that vary by request
body or status code:

```go
func TestPlaylists_CreatePlaylist(t *testing.T) {
    tests := []struct {
        name       string
        userID     string
        playlistName string
        wantStatus int
        wantErr    bool
    }{
        {
            name: "creates playlist successfully",
            userID: "user1", playlistName: "My Playlist",
            wantStatus: http.StatusCreated,
        },
        {
            name: "403 returns ForbiddenError",
            userID: "user1", playlistName: "My Playlist",
            wantStatus: http.StatusForbidden,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(tt.wantStatus)
                if tt.wantStatus == http.StatusCreated {
                    _, _ = w.Write([]byte(`{"id":"new-id","name":"My Playlist"}`))
                }
            }))
            defer srv.Close()
            client := newTestPlaylists(srv.URL, "test-token")
            _, err := client.CreatePlaylist(context.Background(), tt.userID, tt.playlistName)
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/api/... -run TestPlaylists -race -count=1 -v
```

Expected: all playlist tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/api/playlists_test.go
git commit -m "test(api): refactor playlists_test.go to table-driven style"
```

---

### Task 10: Final verification and PR

- [ ] **Step 1: Full CI gate**

```bash
make ci
```

Expected: all steps pass.

- [ ] **Step 2: Confirm no *state.Store in pane files**

```bash
grep -r "\*state\.Store" internal/ui/panes/ | grep -v "_test.go"
```

Expected: no output.

- [ ] **Step 3: Confirm gateway tests still pass**

```bash
go test ./internal/api/... -run TestGateway -race -count=1
```

Expected: all pass.

- [ ] **Step 4: Push and open PR**

```bash
git push origin refactor/audit-phase2-structure
```

Open PR with title: `refactor: phase 2 — StateReader interface, gateway/app splits, table-driven tests`

Body:
```
## Changes

- Add `state.StateReader` read-only interface; 8 pane constructors now accept it instead of *Store
- Split gateway.go (747 LOC) → gateway.go + gateway_bucket.go + gateway_dedup.go
- Split app.go (1807 LOC) → app.go + handlers.go + prefs.go
- Refactor player_test.go, library_test.go, playlists_test.go to table-driven style

## No Behaviour Changes

All existing tests pass unchanged. This is pure structural reorganisation.

## Test Summary

- make ci passes
- grep -r "*state.Store" internal/ui/panes/ → zero hits
- All gateway tests pass after split
```
