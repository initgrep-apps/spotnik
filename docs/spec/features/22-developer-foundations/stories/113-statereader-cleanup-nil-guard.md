---
title: "StateReader Interface Cleanup, NowPlaying Test Helper & Nil Guard"
feature: 22-developer-foundations
status: open
---

## Background

Three structural issues were identified during PR reviews of Stories 110–111:

1. **Dead interface surface on `StateReader`** — `internal/state/reader.go`
   includes `PlaylistsStale`, `AlbumsStale`, `LikedTracksStale`,
   `RecentlyPlayedStale`, and `DevicesStale` in the `StateReader` interface.
   Panes never call these methods through the interface — only `handlers.go`
   calls them through the concrete `*Store`. Removing them makes `StateReader`
   minimal and self-documenting: it expresses exactly what panes need to read.

2. **Type-assertion pattern in `nowplaying_test.go`** — four test call sites
   (lines 217, 234, 447, 470) call `pane.store.(*state.Store).SetPlaybackState(...)`
   to set up state, asserting through the read-only `StateReader` interface back
   to `*Store`. If a non-`*Store` `StateReader` is ever injected this panics at
   runtime. A `testStateWriter` helper in the test file exposes write methods
   without an interface-piercing type assertion.

3. **`postTokenRequest` nil-client panic** — `internal/api/auth.go` accepts
   `*http.Client` but has no nil guard. A nil client panics at
   `httpClient.Do(req)` with no attribution. All callers currently pass
   `http.DefaultClient` but the injection pattern makes nil a realistic future
   mistake. A nil guard with a clear error message prevents a confusing panic.

**Source:** `docs/spec/issues.md` (PR review findings from Stories 110, 111)

**Depends on:** Story 110 (StateReader interface must exist), Story 111
(postTokenRequest injection pattern and NowPlayingPane must exist)

---

## Design

### Task 1 — Remove unused staleness methods from `StateReader`

**File:** `internal/state/reader.go`

Read the current interface definition first to see the exact method signatures
and line numbers:
```bash
grep -n "Stale" internal/state/reader.go
```

Remove the following five methods from the `StateReader` interface:
- `PlaylistsStale() bool`
- `AlbumsStale() bool`
- `LikedTracksStale() bool`
- `RecentlyPlayedStale() bool`
- `DevicesStale() bool`

Keep `StatsStale() bool` — it is called through the `StateReader` interface
by the stats pane.

After removal, verify that `handlers.go` still compiles — it calls these
methods on `a.store` (the concrete `*Store`), so removal from the interface
does not break it. The compile-time assertion at the bottom of `reader.go`
(`var _ StateReader = (*Store)(nil)`) still validates that `*Store`
implements the trimmed interface.

Verify:
```bash
go build ./internal/...
go test ./internal/state/... -race -count=1
```

### Task 2 — `testStateWriter` helper in `nowplaying_test.go`

**File:** `internal/ui/panes/nowplaying_test.go`

Read the file first to understand the current constructor signature and the
4 type-assertion call sites before editing:
```bash
grep -n "state.Store\|newTestNowPlaying" internal/ui/panes/nowplaying_test.go
```

Add a `testStateWriter` struct near the top of the file (after imports):

```go
// testStateWriter wraps *state.Store to expose write methods for test setup
// without requiring a type assertion through the StateReader interface.
type testStateWriter struct {
    *state.Store
}
```

Update the test constructor to return `*testStateWriter` alongside the pane:

```go
func newTestNowPlayingPane(th theme.Theme) (*NowPlayingPane, *testStateWriter) {
    store := state.NewStore()
    w := &testStateWriter{store}
    pane := NewNowPlayingPane(store, th, false)
    return pane, w
}
```

If the constructor currently returns only the pane, update all call sites to
accept the second return value. Then replace all 4 type-assertion call sites:

```go
// Before:
pane.store.(*state.Store).SetPlaybackState(newState)

// After:
w.SetPlaybackState(newState)
```

Verify: `go test ./internal/ui/panes/... -run TestNowPlaying -race -count=1` passes
with zero `(*state.Store)` type assertions remaining:
```bash
grep "state.Store)" internal/ui/panes/nowplaying_test.go
# Expected: no output
```

### Task 3 — Nil guard in `postTokenRequest`

**File:** `internal/api/auth.go`

Read the function first to confirm the exact signature and the `Do` call location:
```bash
grep -n "httpClient\|func postToken" internal/api/auth.go
```

Add a nil guard immediately after the function's opening brace:

```go
func postTokenRequest(ctx context.Context, httpClient *http.Client, endpoint string, formData url.Values) (TokenPair, error) {
    if httpClient == nil {
        return TokenPair{}, errors.New("postTokenRequest: httpClient must not be nil")
    }
    // ... rest of function unchanged
}
```

Add `"errors"` to the import block if not already present.

Verify: `go build ./internal/api/...` clean.

### Task 4 — Test the nil guard

**File:** `internal/api/auth_test.go`

Confirm the package declaration of the existing test file first:
```bash
head -3 internal/api/auth_test.go
```

Add a test in `package api` (internal, since `postTokenRequest` is unexported):

```go
func TestPostTokenRequest_NilClientReturnsError(t *testing.T) {
    _, err := postTokenRequest(
        context.Background(),
        nil,
        "http://example.com/token",
        url.Values{"grant_type": {"client_credentials"}},
    )
    assert.ErrorContains(t, err, "httpClient must not be nil")
}
```

Verify: `go test ./internal/api/... -run TestPostTokenRequest_NilClient -race -count=1 -v` → PASS.

---

## Acceptance Criteria

- [ ] `StateReader` interface in `reader.go` no longer contains `PlaylistsStale`,
      `AlbumsStale`, `LikedTracksStale`, `RecentlyPlayedStale`, `DevicesStale`
- [ ] `StatsStale` remains in the interface
- [ ] `go build ./internal/...` passes (compile-time assertion still holds)
- [ ] No `(*state.Store)` type assertions remain in `nowplaying_test.go`
- [ ] `testStateWriter` helper struct exists in `nowplaying_test.go`
- [ ] All nowplaying tests pass: `go test ./internal/ui/panes/... -run TestNowPlaying -race -count=1`
- [ ] `postTokenRequest` returns an error containing `"httpClient must not be nil"` when passed `nil`
- [ ] `TestPostTokenRequest_NilClientReturnsError` passes
- [ ] `make ci` passes

## Tasks

- [ ] Remove 5 staleness methods from `StateReader` interface in `reader.go`
      - test: `go build ./internal/...` clean; `go test ./internal/state/... -race -count=1` passes
- [ ] Add `testStateWriter` helper and update `newTestNowPlayingPane` constructor in `nowplaying_test.go`;
      replace all 4 type-assertion call sites
      - test: `go test ./internal/ui/panes/... -run TestNowPlaying -race -count=1` passes;
        `grep "state.Store)" internal/ui/panes/nowplaying_test.go` → no output
- [ ] Add nil guard to `postTokenRequest` in `auth.go`
      - test: `go build ./internal/api/...` clean
- [ ] Add `TestPostTokenRequest_NilClientReturnsError` to `auth_test.go`
      - test: `go test ./internal/api/... -run TestPostTokenRequest_NilClient -race -count=1 -v` → PASS
- [ ] `make ci` passes
