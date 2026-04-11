---
title: "Fix: buildPlaybackAPICmd snapshots post-optimistic store state"
feature: 26-optimistic-playback
status: open
---

## Background

Story 124 implemented optimistic playback updates. The handler in `handlers.go` calls
`applyOptimisticUpdate` **before** `buildPlaybackAPICmd`. Inside `buildPlaybackAPICmd`
(`commands.go:50–60`), the store is snapshotted after the optimistic value is already
written. For actions that compute the next API value from the current store state
(shuffle, repeat, volume), this causes the API to receive the wrong value.

**Observed bugs:**
- **Volume**: press `+` at vol=74 → optimistic shows 75 → Spotify receives `SetVolume(76)` → final state 76 (double-increment)
- **Shuffle**: press `s` when off → optimistic shows on → Spotify receives `SetShuffle(false)` → shuffle is undone
- **Repeat**: press `r` when off → optimistic shows context → Spotify receives `SetRepeat("track")` → skips the context step

**Play/pause is NOT affected** — `player.Pause()` / `player.Play()` do not read the store.

## Design

### Root cause

```go
// handlers.go (current — WRONG order)
case panes.PlaybackRequestMsg:
    a.applyOptimisticUpdate(m.Action)         // 1. writes optimistic: shuffle=true
    return a, a.buildPlaybackAPICmd(m.Action) // 2. snapshots store → gets shuffle=true
                                              //    → sends SetShuffle(!true) = false ← BUG
```

`buildPlaybackAPICmd` captures `isShuffled`, `repeatMode`, and `currentVolume` from
`a.store.PlaybackState()` at construction time. When the optimistic value has already
been written, these snapshots are one step ahead, causing every state-derived API call
to compute from the wrong baseline.

### Fix

Swap the call order so `buildPlaybackAPICmd` captures the **pre-optimistic** store state:

```go
// handlers.go (fixed)
case panes.PlaybackRequestMsg:
    cmd := a.buildPlaybackAPICmd(m.Action) // 1. snapshot pre-optimistic (vol=74 → sends 75)
    a.applyOptimisticUpdate(m.Action)      // 2. write optimistic (store → 75, UI renders)
    return a, cmd
```

`buildPlaybackAPICmd` is unchanged — it still builds the same closure; it just reads
the store one line earlier, before the optimistic write.

### ARCHITECTURE.md update

The "Optimistic Updates" section added in story 124 shows the wrong order in its code
example. Update it to reflect the correct order and add a note explaining why
`buildPlaybackAPICmd` must be called first.

## Acceptance Criteria

- [ ] Pressing `+` at vol=74 → UI shows 75, Spotify receives `SetVolume(75)` (not 76)
- [ ] Pressing `s` when shuffle=off → UI shows on, Spotify receives `SetShuffle(true)` (not false)
- [ ] Pressing `r` when repeat=off → UI shows context, Spotify receives `SetRepeat("context")` (not track)
- [ ] Pressing `r` when repeat=track → UI shows off, Spotify receives `SetRepeat("off")` (not context)
- [ ] Play/pause behaviour unchanged
- [ ] `make ci` passes

## Tasks

### Fix the call order in handlers.go

- [ ] In `internal/app/handlers.go`, find the `PlaybackRequestMsg` case (~line 519):

  ```go
  // before
  case panes.PlaybackRequestMsg:
      a.applyOptimisticUpdate(m.Action)
      return a, a.buildPlaybackAPICmd(m.Action)
  ```

  Replace with:

  ```go
  // after
  case panes.PlaybackRequestMsg:
      cmd := a.buildPlaybackAPICmd(m.Action)
      a.applyOptimisticUpdate(m.Action)
      return a, cmd
  ```

  - verify: `go test ./internal/app/ -run TestApplyOptimisticUpdate -v` → all tests still pass
  - verify: `go test ./internal/app/ -v` → all tests pass

### Update ARCHITECTURE.md

- [ ] In `docs/ARCHITECTURE.md`, find the `### Optimistic Updates` section added in story 124.
  Update the code example from:

  ```go
  case panes.PlaybackRequestMsg:
      a.applyOptimisticUpdate(m.Action) // sync: store written, UI renders next frame
      return a, a.buildPlaybackAPICmd(m.Action) // async: API call, result overwrites store
  ```

  To:

  ```go
  case panes.PlaybackRequestMsg:
      cmd := a.buildPlaybackAPICmd(m.Action) // snapshot pre-optimistic store state
      a.applyOptimisticUpdate(m.Action)      // sync: store written, UI renders next frame
      return a, cmd                          // async: API call, result overwrites store
  ```

  Add a note after the example:

  ```markdown
  **Order matters:** `buildPlaybackAPICmd` must be called before `applyOptimisticUpdate`.
  The cmd closure captures store values (volume, shuffle state, repeat mode) at construction
  time. If the optimistic write happens first, the closure captures the already-predicted
  value and computes the wrong next state (e.g. sends `SetShuffle(false)` when the user
  intended to turn shuffle on).
  ```

### Commit

- [ ] `git add internal/app/handlers.go docs/ARCHITECTURE.md`
- [ ] `git commit -m "fix(playback): snapshot store before optimistic write in PlaybackRequestMsg handler"`
