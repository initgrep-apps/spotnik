---
title: "Fix: Volume bar review polish"
feature: 03-playback
status: open
---

## Background

Eight follow-up items from the PR #271 multi-agent review of stories 197–198
(volume debounce queue cleanup and snap-back fix). None blocked merge.

Root causes / observations:
- Two app-level `VolumeAppliedMsg` error paths (401, generic) have no test
  coverage — only the happy path and 429 are tested.
- `TestApp_VolumeDebounce_FiveRapidPresses_SendsOneCall` uses a boolean
  `MockPlayer.SetVolumeCalled` flag, so the test proves "at least one call" not
  "exactly one call."
- `TestApp_VolumeAppliedMsg_429_ClearsPendingAndBacksOff` never inspects
  `NowPlayingPane` to confirm `hasPending` was cleared.
- No `NowPlayingPane`-level test covers the stale `VolumeAppliedMsg` scenario
  (seq mismatch: first burst's applied msg arrives after a second burst started).
- `TestApp_VolumeAppliedMsg_Success_DispatchesInteractivePoll` only asserts
  `NotNil` on the returned command — it never executes it.
- The `VolumeAppliedMsg` handler in `handlers.go` discards the pane command with
  `updated, _ := np.Update(m)`. Today the pane returns nil, but if a future
  story adds a pane command here the discard will silently drop it.
- `VolumeAppliedMsg{}` zero-value reads as "volume successfully set to 0%", which
  is ambiguous. Constructor functions make the success/failure duality explicit.

## Design

### `internal/app/volume_test.go` — new + improved tests

**1. Test `VolumeAppliedMsg` 401 → `unauthorizedMsg` path:**
```go
func TestApp_VolumeAppliedMsg_401_RoutesToUnauthorized(t *testing.T) {
    a := newVolumeTestApp(&apitest.MockPlayer{})
    unauthorizedErr := &api.UnauthorizedError{}

    _, cmd := a.Update(panes.VolumeAppliedMsg{Err: unauthorizedErr, Seq: 1})
    require.NotNil(t, cmd)

    // The 401 path re-routes through handleMsg(unauthorizedMsg{}) which
    // returns a token-refresh command.
    msgs := collectAllMsgsVolume(cmd)
    hasUnauthorized := false
    for _, m := range msgs {
        if _, ok := m.(unauthorizedMsg); ok {
            hasUnauthorized = true
            break
        }
    }
    assert.True(t, hasUnauthorized, "VolumeAppliedMsg with 401 must route to unauthorizedMsg")
}
```

**2. Test `VolumeAppliedMsg` generic error → `tea.Batch` with toast + poll:**
```go
func TestApp_VolumeAppliedMsg_GenericError_BatchesPollandToast(t *testing.T) {
    a := newVolumeTestApp(&apitest.MockPlayer{})

    _, cmd := a.Update(panes.VolumeAppliedMsg{Err: errors.New("network timeout"), Seq: 1})
    require.NotNil(t, cmd)

    // Execute the returned command: should be a BatchMsg containing
    // a fetchPlaybackStateCmd and a toast cmd.
    msg := cmd()
    batch, ok := msg.(tea.BatchMsg)
    require.True(t, ok, "generic error must return a BatchMsg, got %T", msg)
    assert.Len(t, batch, 2, "batch must contain fetchPlaybackStateCmd + toast cmd")
}
```

**3. Add `SetVolumeCallCount` to `MockPlayer`; assert exact count in debounce test:**

In `internal/api/apitest/mock.go` add:
```go
SetVolumeCallCount int
```

In `SetVolume` method:
```go
m.SetVolumeCallCount++
```

In `TestApp_VolumeDebounce_FiveRapidPresses_SendsOneCall` replace:
```go
assert.True(t, mock.SetVolumeCalled, ...)
```
with:
```go
assert.Equal(t, 1, mock.SetVolumeCallCount, "5 rapid presses must result in exactly one SetVolume call")
```

**4. Fix 429 test to assert `NowPlayingPane` pending state cleared:**

In `TestApp_VolumeAppliedMsg_429_ClearsPendingAndBacksOff`, after feeding the
`VolumeAppliedMsg` back, call `a.View()` and verify the volume bar is no longer
in pending state. Pending state renders a `~` prefix on the bar; confirmed state
does not:
```go
view := a.View()
assert.NotContains(t, view, "~", "volume bar must not be in pending state after 429")
```
(Adjust based on what the actual pending-state rendering looks like.)

**5. New `NowPlayingPane` test for stale `VolumeAppliedMsg` in `nowplaying_test.go`:**
```go
func TestNowPlayingPane_VolumeAppliedMsg_StaleSeq_BarStaysInSecondBurst(t *testing.T) {
    st := state.New()
    st.SetPlaybackState(&api.PlaybackState{Device: &domain.Device{VolumePercent: 50}})
    p := newTestNowPlayingPane(st)

    // First burst: prime seq to 1.
    p.volumeBar.HandleKey(1, 50)
    // Second burst: advance seq to 2 (first burst is now stale).
    p.volumeBar.HandleKey(1, 51)

    // Feed first burst's VolumeAppliedMsg (seq=1 — stale).
    p.Update(VolumeAppliedMsg{Vol: 51, Seq: 1})

    // Bar must still show the second burst's pending value (52), not snap to 51.
    view := p.View()
    assert.Contains(t, view, "52", "stale applied msg must not override second burst's pending value")
}
```

**6. Fix `Success` test to execute cmd and verify it's non-trivial:**
```go
func TestApp_VolumeAppliedMsg_Success_DispatchesInteractivePoll(t *testing.T) {
    a := newVolumeTestApp(&apitest.MockPlayer{})

    _, cmd := a.Update(panes.VolumeAppliedMsg{Vol: 72, Seq: 1})
    require.NotNil(t, cmd, "VolumeAppliedMsg success must dispatch a cmd")

    // Execute the cmd — must produce PlaybackStateFetchedMsg (interactive poll).
    result := cmd()
    _, ok := result.(panes.PlaybackStateFetchedMsg)
    assert.True(t, ok, "success cmd must produce PlaybackStateFetchedMsg, got %T", result)
}
```

### `internal/app/handlers.go` — capture pane command

In the `VolumeAppliedMsg` handler, replace the discard with a capture:

```go
// Before:
updated, _ := np.Update(m)

// After:
updated, paneCmd := np.Update(m)
```

Then batch `paneCmd` with any downstream effects. For the success path:
```go
return a, tea.Batch(paneCmd, fetchPlaybackStateCmd(a.player, api.Interactive))
```
(Use `tea.Batch` even when `paneCmd` is nil — `tea.Batch` ignores nil entries.)

### `internal/ui/panes/messages.go` — constructor functions (optional hardening)

```go
// NewVolumeAppliedMsg constructs a success VolumeAppliedMsg.
func NewVolumeAppliedMsg(vol, seq int) VolumeAppliedMsg {
    return VolumeAppliedMsg{Vol: vol, Seq: seq}
}

// NewVolumeAppliedError constructs an error VolumeAppliedMsg.
func NewVolumeAppliedError(seq int, err error) VolumeAppliedMsg {
    return VolumeAppliedMsg{Seq: seq, Err: err}
}
```

Update call sites in `internal/app/commands.go` to use the constructors.

## Acceptance Criteria

- [ ] `TestApp_VolumeAppliedMsg_401_RoutesToUnauthorized` passes
- [ ] `TestApp_VolumeAppliedMsg_GenericError_BatchesPollandToast` passes
- [ ] `MockPlayer.SetVolumeCallCount` exists; debounce test asserts `== 1`
- [ ] `TestApp_VolumeAppliedMsg_429_ClearsPendingAndBacksOff` asserts via `View()` that pending cleared
- [ ] `TestNowPlayingPane_VolumeAppliedMsg_StaleSeq_BarStaysInSecondBurst` passes
- [ ] `TestApp_VolumeAppliedMsg_Success_DispatchesInteractivePoll` executes cmd and asserts `PlaybackStateFetchedMsg`
- [ ] `handlers.go` `VolumeAppliedMsg` handler captures pane cmd and batches it
- [ ] (Optional) Constructor functions added to `messages.go`; commands.go updated
- [ ] `make ci` passes

## Tasks

- [ ] Add `SetVolumeCallCount int` to `MockPlayer` in `internal/api/apitest/mock.go`;
      increment in `SetVolume`
      - test: `go build ./internal/api/apitest/...` compiles cleanly

- [ ] Update `TestApp_VolumeDebounce_FiveRapidPresses_SendsOneCall` to assert
      `SetVolumeCallCount == 1`
      - test: `go test ./internal/app/ -run TestApp_VolumeDebounce -v` → PASS

- [ ] Add `TestApp_VolumeAppliedMsg_401_RoutesToUnauthorized` and
      `TestApp_VolumeAppliedMsg_GenericError_BatchesPollandToast` to `volume_test.go`
      - test: `go test ./internal/app/ -run TestApp_VolumeAppliedMsg -v` → both PASS

- [ ] Fix `TestApp_VolumeAppliedMsg_429_ClearsPendingAndBacksOff` to assert
      pending state cleared via `View()`
      - test: `go test ./internal/app/ -run TestApp_VolumeAppliedMsg_429 -v` → PASS

- [ ] Add `TestNowPlayingPane_VolumeAppliedMsg_StaleSeq_BarStaysInSecondBurst`
      to `nowplaying_test.go`
      - test: `go test ./internal/ui/panes/ -run TestNowPlayingPane_VolumeApplied -v` → all PASS

- [ ] Fix `TestApp_VolumeAppliedMsg_Success_DispatchesInteractivePoll` to execute
      cmd and assert `PlaybackStateFetchedMsg`
      - test: `go test ./internal/app/ -run TestApp_VolumeAppliedMsg_Success -v` → PASS

- [ ] Capture pane cmd in `handlers.go` `VolumeAppliedMsg` handler; batch with
      downstream effects
      - test: `go build ./internal/app/...` compiles cleanly; existing volume tests pass

- [ ] (Optional) Add `NewVolumeAppliedMsg` / `NewVolumeAppliedError` constructors
      to `messages.go`; update `commands.go` call sites
      - test: `go build ./...` compiles cleanly

- [ ] `make ci` passes
