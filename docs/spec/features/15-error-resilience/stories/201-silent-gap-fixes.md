---
title: "Silent Gap Fixes"
feature: 15-error-resilience
status: open
---

## Background

Four silent failure modes that produce no user-visible signal:

1. **Prefs flush** — `PreferencesFlushedMsg` handler logs to stderr on error. User never knows their preferences weren't saved.
2. **Search offset limit** — `buildSearchPageCmd` returns `nil` when `offset >= 1000` (Spotify API limit). User presses PgDn on page 20 and nothing happens.
3. **HTTP timeout** — `http.Client` in `internal/api/base.go` has no `Timeout`. Requests can hang indefinitely.
4. **ErrorMapper 403 bodies** — All 403 responses return the generic body `"A Premium subscription is required for this feature."` regardless of operation. Queue and AddToQueue 403s mean "no active device", not Premium.

None of these require Story 199/200 infrastructure. They are independent targeted fixes.

## Design

### `internal/app/prefs.go` — toast on flush error

In the `prefs.FlushedMsg` handler, replace the `fmt.Fprintf(os.Stderr, ...)` path:

```go
// OLD:
case prefs.FlushedMsg:
    if m.Err != nil {
        fmt.Fprintf(os.Stderr, "spotnik: prefs flush failed: %v\n", m.Err)
        return a, a.schedulePrefsFlush(), true
    }

// NEW:
case prefs.FlushedMsg:
    if m.Err != nil {
        return a, tea.Batch(
            a.toasts.Cmd(uikit.Toast{
                Intent: uikit.ToastWarning,
                Title:  "Preferences not saved",
                Body:   "Check available disk space.",
            }),
            a.schedulePrefsFlush(),
        ), true
    }
```

Remove the `fmt.Fprintf` call and remove the `os` import from `prefs.go` if it becomes unused.

### `internal/app/commands.go` — search offset error

In `buildSearchPageCmd`, replace the silent `return nil` with an error message:

```go
// OLD:
if offset >= 1000 {
    return nil
}

// NEW:
if offset >= 1000 {
    return func() tea.Msg {
        return panes.SearchPageLoadedMsg{
            Query: query,
            Page:  page,
            Err:   errors.New("no more results: Spotify limits search to 1000 items per query"),
        }
    }
}
```

Add `"errors"` to imports if not already present. The existing `SearchPageLoadedMsg` error
handler already shows a toast and keeps existing results visible — this fix only plumbs the
signal through.

### `internal/api/base.go` — HTTP client timeout

In `NewBaseClient`, set `Timeout` on the `http.Client`:

```go
// OLD:
http: &http.Client{},

// NEW:
http: &http.Client{Timeout: 30 * time.Second},
```

Add `"time"` to imports if not already present.

### `internal/uikit/error_mapper.go` — per-operation 403 bodies

Add an `opForbiddenBody` map after the existing `opTitle` map:

```go
// opForbiddenBody maps an Operation to its 403-specific recovery hint.
// OpQueue and OpAddToQueue return a "no active device" hint because Spotify
// returns 403 for queue actions when no device is active, not for Premium.
var opForbiddenBody = map[Operation]string{
    OpPlayback:   "Premium required for this action.",
    OpVolume:     "Premium required for volume control.",
    OpSearch:     "Premium required for search.",
    OpQueue:      "No active device. Open Spotify first.",
    OpAddToQueue: "No active device. Open Spotify first.",
    OpTransfer:   "Premium required for device control.",
    OpPlaylistTracks: "No permission to view this playlist.",
}
```

Add a private method:

```go
func (em *ErrorMapper) forbiddenBodyFor(op Operation) string {
    if b, ok := opForbiddenBody[op]; ok {
        return b
    }
    return "A Premium subscription is required for this feature."
}
```

Replace the existing `ForbiddenError` handling block to use the lookup:

```go
// OLD:
var forbiddenErr *api.ForbiddenError
if errors.As(err, &forbiddenErr) {
    body := forbiddenErr.Message
    if body == "" || body == "Spotify Premium required" {
        body = "A Premium subscription is required for this feature."
    }
    return Toast{Intent: ToastWarning, Title: "Spotify Premium required", Body: body}
}

// NEW:
var forbiddenErr *api.ForbiddenError
if errors.As(err, &forbiddenErr) {
    return Toast{
        Intent: ToastWarning,
        Title:  em.titleFor(op),
        Body:   em.forbiddenBodyFor(op),
    }
}
```

## Acceptance Criteria

- [ ] `PreferencesFlushedMsg` error emits `ToastWarning "Preferences not saved"` with body `"Check available disk space."` — no stderr write
- [ ] `buildSearchPageCmd` with `offset >= 1000` returns a cmd that yields `SearchPageLoadedMsg{Err: <non-nil>}` — not `nil`
- [ ] `http.Client` in `NewBaseClient` has `Timeout: 30 * time.Second`
- [ ] `ErrorMapper.Map(OpQueue, &ForbiddenError{})` returns body `"No active device. Open Spotify first."`
- [ ] `ErrorMapper.Map(OpPlayback, &ForbiddenError{})` returns body `"Premium required for this action."`
- [ ] `ErrorMapper.Map(OpPlaylists, &ForbiddenError{})` returns generic fallback body
- [ ] `make ci` passes

## Tasks

- [ ] Write failing test `TestApp_PrefsFlush_ErrorProducesToastWarning` in
      `internal/app/prefs_test.go`
      - test: `go test ./internal/app/ -run "TestApp_PrefsFlush_Error" -v` → FAIL

- [ ] Update `prefs.FlushedMsg` handler in `internal/app/prefs.go`; remove `os` import if unused
      - test: `go test ./internal/app/ -run "TestApp_PrefsFlush_Error" -v` → PASS

- [ ] Write failing test `TestBuildSearchPageCmd_OffsetAtLimit_ReturnsError` in
      `internal/app/commands_test.go` — asserts cmd is non-nil and `SearchPageLoadedMsg.Err` is set
      - test: `go test ./internal/app/ -run "TestBuildSearchPageCmd_OffsetAtLimit" -v` → FAIL

- [ ] Update `buildSearchPageCmd` in `internal/app/commands.go`
      - test: `go test ./internal/app/ -run "TestBuildSearchPageCmd_OffsetAtLimit" -v` → PASS

- [ ] Write failing test `TestNewBaseClient_HasTimeout` in `internal/api/base_test.go` —
      exports `HTTPTimeout()` on `BaseClient` for assertion, asserts `== 30 * time.Second`
      - test: `go test ./internal/api/ -run "TestNewBaseClient_HasTimeout" -v` → FAIL

- [ ] Add `Timeout: 30 * time.Second` to `http.Client` in `NewBaseClient`; add exported
      `HTTPTimeout()` accessor for tests
      - test: `go test ./internal/api/ -run "TestNewBaseClient_HasTimeout" -v` → PASS

- [ ] Write failing tests in `internal/uikit/error_mapper_test.go`:
      `TestErrorMapper_Forbidden_PlaybackBody`, `_QueueBody`, `_AddToQueueBody`,
      `_TransferBody`, `_VolumeBody`, `_SearchBody`, `_GenericFallback`
      - test: `go test ./internal/uikit/ -run "TestErrorMapper_Forbidden" -v` → FAIL

- [ ] Add `opForbiddenBody` map + `forbiddenBodyFor()` to `internal/uikit/error_mapper.go`;
      update `ForbiddenError` block
      - test: `go test ./internal/uikit/ -run "TestErrorMapper_Forbidden" -v` → all PASS

- [ ] `make ci` passes
