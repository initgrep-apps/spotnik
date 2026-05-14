---
title: "Fix: Polling infra test robustness + backoff guard"
feature: 15-error-resilience
status: done
---

## Background

Five follow-up items from PR #278 (story 199) and PR #279 (story 200) reviews.
None blocked merge; all are small, isolated fixes in `internal/app/`.

Root causes / observations:

**From story 199 review (PR #278):**
- `collectAllMsgs` in `poll_test.go` only resolves one level of batch nesting.
  Currently safe (TickMsg handler returns flat batches), but fragile if a future
  story wraps fetch commands in an inner batch. `collectInitMsgs` in `app_test.go`
  already handles recursive nesting — `collectAllMsgs` should mirror it.
- `hasLibraryMsg` uses a disjunctive check: the test passes if *any one* of the
  five library message types appears. Four panes could silently fail to dispatch.
  Change to assert all five types are represented.

**From story 200 review (PR #279):**
- `calcBackoffTicks(0)` returns 0 (`1 << uint(-1)` shifts to 0 in Go). Currently
  safe because all callers increment `errorCount` before calling, but a future
  caller could pass 0. Add a guard clause and a test case.
- `QueueLoadedMsg` error path emits a toast on every failure with no suppression.
  All other panes gate the toast at `errorCount == 1` via a `pollState`. Queue
  has no `queuePoll` and thus no backoff, so during persistent failures it spams
  toasts at the queue's polling interval. Wire a `queuePoll pollState` and gate
  the queue error toast.
- `PlaybackStateFetchedMsg` error handler has no backoff intentionally (playback
  is the most important data stream and always polls at 1s). This decision is
  undocumented; a future maintainer will wonder why. Add a `// NOTE:` comment.

## Design

### `internal/app/poll_test.go` — `collectAllMsgs` recursive

Replace the flat implementation with the same recursive pattern used by
`collectInitMsgs` in `app_test.go`:

```go
func collectAllMsgs(cmd tea.Cmd) []tea.Msg {
    if cmd == nil {
        return nil
    }
    msg := cmd()
    if msg == nil {
        return nil
    }
    if batch, ok := msg.(tea.BatchMsg); ok {
        var msgs []tea.Msg
        for _, c := range batch {
            if c != nil {
                msgs = append(msgs, collectAllMsgs(c)...)
            }
        }
        return msgs
    }
    return []tea.Msg{msg}
}
```

### `internal/app/poll_test.go` — `hasLibraryMsg` → check all five types

Replace the disjunctive helper with an assertion that all five types are present:

```go
// assertAllLibraryMsgs fails the test if any of the five library fetch message
// types is absent from msgs.
func assertAllLibraryMsgs(t *testing.T, msgs []tea.Msg) {
    t.Helper()
    types := map[string]bool{
        "LibraryLoadedMsg":       false,
        "AlbumsLoadedMsg":        false,
        "LikedTracksLoadedMsg":   false,
        "RecentlyPlayedLoadedMsg": false,
        "StatsLoadedMsg":         false,
    }
    for _, m := range msgs {
        switch m.(type) {
        case panes.LibraryLoadedMsg:
            types["LibraryLoadedMsg"] = true
        case panes.AlbumsLoadedMsg:
            types["AlbumsLoadedMsg"] = true
        case panes.LikedTracksLoadedMsg:
            types["LikedTracksLoadedMsg"] = true
        case panes.RecentlyPlayedLoadedMsg:
            types["RecentlyPlayedLoadedMsg"] = true
        case panes.StatsLoadedMsg:
            types["StatsLoadedMsg"] = true
        }
    }
    for name, found := range types {
        assert.True(t, found, "TickMsg at tick 0 must dispatch %s", name)
    }
}
```

Update `TestApp_TickMsg_LibraryPollDispatchesAtTick0` to call
`assertAllLibraryMsgs(t, msgs)` instead of the `hasLibraryMsg` check.
Remove the now-unused `hasLibraryMsg` function.

### `internal/app/app.go` — `calcBackoffTicks` guard

```go
func calcBackoffTicks(errorCount int) int {
    if errorCount <= 0 {
        return 5 // minimum backoff interval
    }
    if ticks := 5 * (1 << uint(errorCount-1)); ticks < 60 {
        return ticks
    }
    return 60
}
```

Add test case to `TestCalcBackoffTicks` in `poll_internal_test.go`:
```go
{0, 5},  // guard: errorCount <= 0 returns minimum
```

### `internal/app/app.go` — `queuePoll pollState`

Add a `queuePoll pollState` field alongside the existing poll state fields:

```go
queuePoll pollState
```

### `internal/app/handlers.go` — queue error toast gating

In the `QueueLoadedMsg` error path, mirror the pattern used by all other library
panes (e.g., `LibraryLoadedMsg`):

```go
case panes.QueueLoadedMsg:
    if m.Err != nil {
        if errors.Is(m.Err, errNilClient) {
            return a, nil
        }
        a.queuePoll.errorCount++
        a.queuePoll.backoffTicks = calcBackoffTicks(a.queuePoll.errorCount)
        a.store.SetQueueError(m.Err)
        if a.queuePoll.errorCount == 1 {
            return a, a.toasts.Cmd(uikit.Toast{
                Intent: uikit.ToastError,
                Title:  "Queue update failed",
                Body:   string(uikit.RecoveryCheckConnection),
            })
        }
        return a, nil
    }
    a.queuePoll.errorCount = 0
    a.store.ClearQueueError()
    a.store.SetQueue(m.Tracks)
    if qp := a.queuePane(); qp != nil {
        qp.RefreshRows()
    }
    return a, nil
```

Note: queue has no per-tick backoff dispatch (unlike library panes) because its
polling interval is driven by `pollIntervals()` in the tick handler, not a
`backoffTicks` countdown. The `backoffTicks` field on `queuePoll` is populated
here for consistency but the queue tick path does not currently read it. This is
intentional — a follow-up story can wire queue backoff if needed.

### `internal/app/handlers.go` — playback NOTE comment

In the `PlaybackStateFetchedMsg` error handler, after the `errNilClient` early
return, add:

```go
// NOTE: Playback intentionally has no per-pane backoff. It is the most
// important data stream (transport state, progress bar) and must always
// poll at the 1s interval regardless of consecutive errors.
```

## Acceptance Criteria

- [ ] `collectAllMsgs` in `poll_test.go` recursively resolves nested batches
- [ ] `TestApp_TickMsg_LibraryPollDispatchesAtTick0` asserts all five library msg types
- [ ] `calcBackoffTicks(0)` returns `5`; test case `{0, 5}` added to `TestCalcBackoffTicks`
- [ ] `QueueLoadedMsg` error path gates toast at `errorCount == 1`; `queuePoll` field exists on `App`
- [ ] `PlaybackStateFetchedMsg` error handler has `// NOTE:` comment explaining no-backoff decision
- [ ] `make ci` passes

## Tasks

- [ ] Make `collectAllMsgs` recursive in `poll_test.go`; update
      `TestApp_TickMsg_LibraryPollDispatchesAtTick0` to use `assertAllLibraryMsgs`;
      remove `hasLibraryMsg`
      - test: `go test ./internal/app/ -run TestApp_TickMsg -v` → all PASS

- [ ] Add guard `errorCount <= 0` to `calcBackoffTicks` in `app.go`;
      add `{0, 5}` case to `TestCalcBackoffTicks` in `poll_internal_test.go`
      - test: `go test ./internal/app/ -run TestCalcBackoffTicks -v` → PASS

- [ ] Add `queuePoll pollState` to `App` struct in `app.go`;
      update `QueueLoadedMsg` handler in `handlers.go` to gate toast at `errorCount == 1`
      - test: `go test ./internal/app/ -run TestApp.*Queue -v` → all PASS

- [ ] Add `// NOTE:` comment to `PlaybackStateFetchedMsg` error handler in `handlers.go`
      - test: `make lint` passes

- [ ] `make ci` passes
