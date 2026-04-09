---
title: "StateReader Interface, File Splits & Table-Driven Tests"
feature: 22-developer-foundations
status: open
---

## Background

The 2026-04-08 audit identified three structural issues:

1. **No `StateReader` interface** — all 8 Page A pane constructors accept `*state.Store`
   (the concrete mutable store). This makes pane unit tests heavier than needed (must
   construct a full Store with all fields), and prevents import-boundary enforcement
   between `ui/panes` and `state`.
2. **Oversized files** — `internal/api/gateway.go` at 747 LOC does three distinct
   things (Gateway routing, token bucket rate limiting, in-flight deduplication).
   `internal/app/app.go` at 1807 LOC does four things (struct/init, message dispatch,
   preferences flush, key routing).
3. **Inconsistent test style** — `player_test.go`, `library_test.go`, and
   `playlists_test.go` use ad-hoc per-case functions rather than table-driven style
   consistent with the rest of the codebase.

All changes are pure restructuring. No logic changes, no new behaviour, no API surface
changes. All existing tests must pass unchanged after each task.

**Source:** `docs/code-audit/code-audit-design.md` §1–§2,
`docs/code-audit/phase2-structural-refactors.md`

**Depends on:** Story 109 (for `testhelpers.LoadFixture` in refactored tests).

---

## Design

### Task 1 — `StateReader` read-only interface

**File to create:** `internal/state/reader.go`

Define a `StateReader` interface containing every read-only method on `*Store` that
panes actually call. Write-only methods (`Set*`, `Clear*`, `Stamp*`) stay on the
concrete `*Store` only — `app.go` holds `*Store` directly and remains the sole writer.

Steps to determine the exact method set:
```bash
grep -r "\.store\." internal/ui/panes/ | grep -v "_test.go" | grep -oP '\.\w+\(' | sort -u
```

Include at minimum:
- Playback: `PlaybackState()`, `ActiveDevice()`, `UserID()`
- Queue: `Queue()`
- Library: `Playlists()`, `PlaylistsTotal()`, `PlaylistTracks(id)`, `PlayingPlaylistID()`,
  `SavedAlbums()`, `AlbumsLoaded()`, `LikedTracks()`, `LikedTotal()`, `LikedLoaded()`,
  `RecentlyPlayed()`
- Stats: `TopTracks(timeRange)`, `TopArtists(timeRange)`
- Devices: `Devices()`
- Staleness: `PlaylistsStale()`, `AlbumsStale()`, `LikedTracksStale()`,
  `RecentlyPlayedStale()`, `StatsStale(timeRange)`, `DevicesStale()`
- Gateway log (for NetworkLog/RequestFlow panes): `ReadEventsFrom(cursor)`
- Throttle (for status bar): `IsThrottled()`, `ThrottleRetryAfterSecs()`

Add a compile-time assertion at the bottom of `reader.go`:
```go
var _ StateReader = (*Store)(nil)
```

This fails to compile if `*Store` is missing a method listed in the interface.

### Task 2 — Migrate all 8 pane constructors to `StateReader`

**Files to modify:**
`nowplaying.go`, `queue.go`, `playlists_pane.go`, `albums_pane.go`,
`likedsongs_pane.go`, `recentlyplayed_pane.go`, `toptracks_pane.go`,
`topartists_pane.go` (all in `internal/ui/panes/`)

For each pane:
1. Change the struct field from `store *state.Store` → `store state.StateReader`
2. Change the constructor parameter from `store *state.Store` → `store state.StateReader`

`app.go` passes a `*state.Store` to all constructors. Since `*Store` satisfies
`StateReader`, `app.go` requires no changes.

Pane test helpers construct `state.New()` which returns `*Store`. Since `*Store`
satisfies `StateReader`, no test changes are needed.

Verify after migration:
```bash
grep -r "\*state\.Store" internal/ui/panes/ | grep -v "_test.go"
# Expected: zero hits
```

### Task 3 — Split `gateway.go` into three files

**Current file:** `internal/api/gateway.go` (747 LOC)

Identify the type boundaries:
```bash
grep -n "^type tokenBucket\|^type inflightEntry\|^func (b \*tokenBucket\|^func (i \*inflightEntry" \
    internal/api/gateway.go
```

**Create `internal/api/gateway_bucket.go`:**
Move the `tokenBucket` struct and all its methods (receiver `*tokenBucket`) here.
File header doc: "token bucket rate limiter for the API gateway — 10 tokens/second,
burst 10; background requests drain the bucket; interactive requests bypass it."

**Create `internal/api/gateway_dedup.go`:**
Move the `inflightEntry` struct and all its methods here.
File header doc: "in-flight request deduplication — same (Method, Path) key → one HTTP
call; all concurrent waiters receive a copy of the response body."

**Trim `gateway.go`:**
Remove the moved types. Keep: `Gateway` struct, `New()`, `Do()`, context priority
helpers, `emitEvent*` functions.

All three files remain in `package api` — Go sees them as one package. No import
changes anywhere in the codebase.

Verify:
```bash
go build ./internal/api/...
go test ./internal/api/... -run TestGateway -race -count=1
```

### Task 4 — Split `app.go` into three files

**Current file:** `internal/app/app.go` (1807 LOC)

Identify the boundary functions:
```bash
grep -n "^func (a \*App) handleMsg\|^func (a \*App) schedule\|^func (a \*App) flushPrefs\|^type prefsFlush" \
    internal/app/app.go
```

**Create `internal/app/handlers.go`:**
Move the `handleMsg(msg tea.Msg) (App, tea.Cmd)` function and any private helpers
it calls that are not used elsewhere in `app.go`.
File header doc: "central message dispatch for the root Bubble Tea model — called by
Update() for every non-key, non-mouse message; routes data-carrying Msg payloads to
Store writes and returns follow-up commands."

**Create `internal/app/prefs.go`:**
Move all preference flush logic: `schedulePrefsFlush()`, the `prefsFlushMsg` type
(or equivalent), and the flush command factory.
File header doc: "preference persistence — theme, preset, and visualizer changes are
debounced and flushed to disk via a generation-counter pattern to avoid excessive
writes during rapid key presses."

**Trim `app.go`:**
Keep: App struct definition, `New()`, `Init()`, `Update()` entry point (thin —
delegates to `handleMsg` and key routing), tick handling, `isIdle()` / `pollIntervals()`.

All files remain in `package app`. No import changes elsewhere.

Verify:
```bash
go build ./internal/app/...
go test ./internal/app/... -race -count=1
```

### Task 5 — Table-driven test refactors

**Files to modify:** `internal/api/player_test.go`, `library_test.go`, `playlists_test.go`

For each file, identify test functions that cover multiple input variants or status
codes using separate `TestX_YYY` functions. Merge these into one table-driven function:

```go
func TestPlayer_Play(t *testing.T) {
    tests := []struct {
        name       string
        opts       PlayOptions
        wantStatus int
        wantErr    bool
    }{
        {name: "play with context URI", opts: PlayOptions{ContextURI: "spotify:playlist:abc"}, wantStatus: 204},
        {name: "play with track URI",   opts: PlayOptions{URIs: []string{"spotify:track:xyz"}}, wantStatus: 204},
        {name: "server error",          opts: PlayOptions{}, wantStatus: 500, wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(tt.wantStatus)
            }))
            defer srv.Close()
            client := NewPlayer(srv.URL, "test-token")
            err := client.Play(context.Background(), tt.opts)
            if tt.wantErr { require.Error(t, err) } else { require.NoError(t, err) }
        })
    }
}
```

Use `testhelpers.LoadFixture` (from Story 109) where fixtures are needed.

---

## Acceptance Criteria

- [ ] `internal/state/reader.go` exists; `var _ StateReader = (*Store)(nil)` compiles
- [ ] `grep -r "\*state\.Store" internal/ui/panes/ | grep -v "_test.go"` → zero hits
- [ ] `go test ./internal/ui/panes/... -race -count=1` passes (no test changes required)
- [ ] `gateway.go` split: three files exist, `go build ./internal/api/...` clean,
      all gateway tests pass
- [ ] `app.go` reduced; `handlers.go` and `prefs.go` exist,
      `go test ./internal/app/... -race -count=1` passes
- [ ] `player_test.go`, `library_test.go`, `playlists_test.go` use table-driven style
- [ ] `make ci` passes

## Tasks

- [ ] Create `internal/state/reader.go` with `StateReader` interface and compile-time assertion
      - test: `go build ./internal/state/...` clean; assertion compiles
- [ ] Migrate all 8 pane constructors to accept `state.StateReader`
      - test: `go build ./internal/...` clean; `go test ./internal/ui/panes/... -race` passes;
        `grep -r "\*state\.Store" internal/ui/panes/ | grep -v _test.go` → zero hits
- [ ] Split `gateway.go` → `gateway.go` + `gateway_bucket.go` + `gateway_dedup.go`
      - test: `go build ./internal/api/...` clean;
        `go test ./internal/api/... -run TestGateway -race -count=1` passes
- [ ] Split `app.go` → `app.go` + `handlers.go` + `prefs.go`
      - test: `go build ./internal/app/...` clean;
        `go test ./internal/app/... -race -count=1` passes
- [ ] Refactor `player_test.go` to table-driven style
      - test: `go test ./internal/api/... -run TestPlayer -race -count=1 -v` passes
- [ ] Refactor `library_test.go` to table-driven style
      - test: `go test ./internal/api/... -run TestLibrary -race -count=1 -v` passes
- [ ] Refactor `playlists_test.go` to table-driven style
      - test: `go test ./internal/api/... -run TestPlaylists -race -count=1 -v` passes
