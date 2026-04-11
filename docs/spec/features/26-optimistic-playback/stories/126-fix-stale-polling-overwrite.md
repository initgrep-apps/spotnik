---
title: "Fix: background polling overwrites optimistic state before API completes"
feature: 26-optimistic-playback
status: open
---

## Background

Even after the order-of-operations fix in story 125, a brief UI flash remains for all
optimistic actions: the background polling tick fires independently and writes stale
Spotify state back to the store while the API command is still in flight.

**Observed bugs:**
- **Play/pause**: paused → optimistic playing → reverts to paused → then playing (device plays correctly throughout)
- **Volume**: 74 → optimistic 75 → flashes 74 → settles at 75
- **Shuffle/Repeat**: same revert-then-correct pattern

**Why it happens:**

```
t=0ms    PlaybackRequestMsg → optimistic written (IsPlaying=true) + API cmd fired
t=100ms  Background poll fires → fetchPlaybackStateCmd started
t=300ms  polling PlaybackStateFetchedMsg arrives
         → Spotify still returns IsPlaying=false (API hasn't completed yet)
         → store.SetPlaybackState(stale) → UI reverts to paused  ← BUG
t=400ms  API cmd completes → PlaybackCmdSentMsg → fires reconcile fetch
t=600ms  reconcile PlaybackStateFetchedMsg → IsPlaying=true → UI correct
```

The polling loop and the in-flight command have no coordination. Any fetch that was
initiated before the API call completes will see the old Spotify state.

## Design

### Fix: `playbackCmdPending` counter

Add `playbackCmdPending int` to `*App`. This tracks the number of in-flight playback
commands — i.e. `PlaybackRequestMsg`s that have been handled but whose `PlaybackCmdSentMsg`
has not yet arrived.

In the `PlaybackStateFetchedMsg` handler, skip the store write if `playbackCmdPending > 0`.
Polling fetches that return during this window are silently discarded. The reconcile fetch
(fired by `PlaybackCmdSentMsg`) arrives when `playbackCmdPending == 0` and writes normally.

This is Elm-compliant: `playbackCmdPending` is model state updated inside `Update()`. The
store write suppression is a conditional write inside `Update()`, the same as the existing
`if m.State != nil` guard.

### New field on `*App`

```go
// playbackCmdPending counts PlaybackRequestMsgs that have fired their API cmd but
// whose PlaybackCmdSentMsg has not yet arrived. While non-zero, PlaybackStateFetchedMsg
// store writes are suppressed to prevent background polling from overwriting optimistic
// state with stale Spotify data.
playbackCmdPending int
```

### Handler changes

`internal/app/handlers.go`:

```go
// PlaybackRequestMsg — increment before returning
case panes.PlaybackRequestMsg:
    cmd := a.buildPlaybackAPICmd(m.Action) // pre-optimistic snapshot (story 125)
    a.applyOptimisticUpdate(m.Action)
    a.playbackCmdPending++
    return a, cmd

// PlaybackCmdSentMsg — decrement when command resolves (success or error)
case panes.PlaybackCmdSentMsg:
    if a.playbackCmdPending > 0 {
        a.playbackCmdPending--
    }
    if m.Err != nil {
        // ... existing error handling unchanged ...
    }
    return a, fetchPlaybackStateCmd(a.player)

// PlaybackStateFetchedMsg — suppress store write while commands are pending
case panes.PlaybackStateFetchedMsg:
    if a.playbackCmdPending > 0 {
        // A command is in flight — this fetch carries pre-command Spotify state.
        // Discard it: the reconcile fetch fired by PlaybackCmdSentMsg will correct
        // the store when the command completes.
        return a, nil
    }
    // ... existing handler body unchanged below this guard ...
```

### Residual limitation

After `PlaybackCmdSentMsg` decrements the counter to 0, the reconcile fetch is fired.
A polling fetch that was started just before `PlaybackCmdSentMsg` arrived may also be
in flight at this moment. If that polling fetch arrives before the reconcile fetch, it
writes (possibly stale) Spotify state. The reconcile fetch corrects it ~200ms later.

This is a narrow timing window (~0–200ms after command completion) and rare in practice.
It is not addressed in this story. The primary flash (300ms, always visible) is eliminated.

### ARCHITECTURE.md update

The "Optimistic Updates" section should document the `playbackCmdPending` guard and why
polling writes are suppressed. Update the pattern section to show the full handler:

```go
// In app.Update():
case panes.PlaybackRequestMsg:
    cmd := a.buildPlaybackAPICmd(m.Action) // snapshot pre-optimistic store state
    a.applyOptimisticUpdate(m.Action)      // sync: store written, UI renders next frame
    a.playbackCmdPending++                 // suppress stale polling writes until cmd completes
    return a, cmd                          // async: API call fires through gateway

case panes.PlaybackCmdSentMsg:
    if a.playbackCmdPending > 0 {
        a.playbackCmdPending--             // cmd resolved — allow next fetch to write store
    }
    // ... fire reconcile fetch ...

case panes.PlaybackStateFetchedMsg:
    if a.playbackCmdPending > 0 {
        return a, nil                      // discard: Spotify state is pre-command
    }
    // ... write store ...
```

Add a note: **Polling suppression**: while `playbackCmdPending > 0`, polling fetches that
return stale Spotify state (before the API command has taken effect) are discarded. The
reconcile fetch fired by `PlaybackCmdSentMsg` writes the authoritative post-command state.

## Acceptance Criteria

- [ ] Press `Space` (pause→play): UI shows playing immediately and does NOT revert to paused before settling
- [ ] Press `+`: UI shows vol+1 immediately and does NOT revert to original before settling
- [ ] Press `s`: shuffle indicator toggles immediately and does NOT revert before settling
- [ ] Press `r`: repeat indicator cycles immediately and does NOT revert before settling
- [ ] Rapid hold of `+`: each press increments the bar; no intermediate revert is visible
- [ ] `playbackCmdPending` never goes negative (guarded with `if a.playbackCmdPending > 0`)
- [ ] All existing `PlaybackStateFetchedMsg` tests pass unmodified
- [ ] `make ci` passes

## Tasks

### Add `playbackCmdPending` field to App

- [ ] In `internal/app/app.go`, add after `nilPlaybackStateTicks int`:

  ```go
  // playbackCmdPending counts PlaybackRequestMsgs that have fired their API cmd but
  // whose PlaybackCmdSentMsg has not yet arrived. While non-zero, PlaybackStateFetchedMsg
  // store writes are suppressed to prevent background polling from overwriting optimistic
  // state with stale Spotify data.
  playbackCmdPending int
  ```

### Update handlers.go

- [ ] **PlaybackRequestMsg**: add `a.playbackCmdPending++` after `applyOptimisticUpdate`:

  ```go
  case panes.PlaybackRequestMsg:
      cmd := a.buildPlaybackAPICmd(m.Action)
      a.applyOptimisticUpdate(m.Action)
      a.playbackCmdPending++
      return a, cmd
  ```

- [ ] **PlaybackCmdSentMsg**: add the decrement guard at the top of the case, before the
  existing `if m.Err != nil` block:

  ```go
  case panes.PlaybackCmdSentMsg:
      if a.playbackCmdPending > 0 {
          a.playbackCmdPending--
      }
      // ... existing error handling unchanged ...
  ```

- [ ] **PlaybackStateFetchedMsg**: add the suppression guard at the very top of the case,
  before the existing `if m.Err != nil` block:

  ```go
  case panes.PlaybackStateFetchedMsg:
      if a.playbackCmdPending > 0 {
          return a, nil
      }
      // ... existing handler body unchanged ...
  ```

  - verify: all existing `PlaybackStateFetchedMsg` tests in `elm_purity_test.go` and
    `command_safety_test.go` still pass (they set no pending commands, so the guard never fires)

### Add tests for the pending guard

- [ ] In `internal/app/optimistic_test.go`, add `TestPlaybackCmdPending_SuppressesPollingWrite`:

  ```go
  func TestPlaybackCmdPending_SuppressesPollingWrite(t *testing.T) {
      a := newTestApp()

      // Set up initial playback state.
      initial := domain.PlaybackState{
          IsPlaying: false,
          Device:    &domain.Device{ID: "d1", VolumePercent: 74},
      }
      a.Store().SetPlaybackState(&initial)

      // Fire PlaybackRequestMsg — this increments playbackCmdPending and applies optimistic.
      a.Update(panes.PlaybackRequestMsg{Action: panes.ActionPlay})

      // Store should reflect the optimistic value.
      got := a.Store().PlaybackState()
      require.NotNil(t, got)
      assert.True(t, got.IsPlaying, "optimistic write should set IsPlaying=true")

      // Simulate a stale polling fetch arriving while the command is still in flight.
      stale := domain.PlaybackState{IsPlaying: false, Device: &domain.Device{ID: "d1", VolumePercent: 74}}
      a.Update(panes.PlaybackStateFetchedMsg{State: &stale})

      // Store write should be suppressed — optimistic value preserved.
      got = a.Store().PlaybackState()
      require.NotNil(t, got)
      assert.True(t, got.IsPlaying, "polling fetch must not overwrite optimistic state while cmd is pending")

      // Simulate PlaybackCmdSentMsg (command completed successfully).
      a.Update(panes.PlaybackCmdSentMsg{Err: nil})

      // Now a fetch should write through.
      authoritative := domain.PlaybackState{IsPlaying: true, Device: &domain.Device{ID: "d1", VolumePercent: 74}}
      a.Update(panes.PlaybackStateFetchedMsg{State: &authoritative})
      got = a.Store().PlaybackState()
      require.NotNil(t, got)
      assert.True(t, got.IsPlaying, "post-cmd fetch should write authoritative state")
  }
  ```

- [ ] Add `TestPlaybackCmdPending_NeverGoesNegative`:

  ```go
  func TestPlaybackCmdPending_NeverGoesNegative(t *testing.T) {
      a := newTestApp()

      // PlaybackCmdSentMsg without a prior PlaybackRequestMsg must not panic or go negative.
      require.NotPanics(t, func() {
          a.Update(panes.PlaybackCmdSentMsg{Err: nil})
      })
  }
  ```

- [ ] `go test ./internal/app/ -run TestPlaybackCmdPending -v` → both tests pass

### Update ARCHITECTURE.md

- [ ] In `docs/ARCHITECTURE.md`, find the `### Optimistic Updates` subsection.
  Replace the existing pattern block and surrounding text with the updated version
  shown in the Design section above (full three-case pattern + polling suppression note).

### Commit

- [ ] `git add internal/app/app.go internal/app/handlers.go internal/app/optimistic_test.go docs/ARCHITECTURE.md`
- [ ] `git commit -m "fix(playback): suppress stale polling writes while playback command is in flight"`
