---
title: "Fix: Volume bar snaps back during rapid presses after debounce fires"
feature: 03-playback
status: open
---

## Background

### Observed bug

After story 197 (volume debounce), users see the volume bar flicker back and
forth when pressing a key once and then immediately holding down a rapid burst.

Concretely: press `+` from vol=49 → bar shows 50 → continue holding → bar
accumulates to 55 → bar briefly snaps back to 49 → bar jumps back to 55.

### Root cause

`HandleDebounce` clears `hasPending = false` as soon as the debounce timer
fires. After that, the 1 s regular Spotify poll can call `SetConfirmed(49)`
(the old Spotify-side value, which Spotify hasn't updated yet) and snap the
bar back. When the Interactive reconcile poll (triggered by the API success)
later returns `55`, the bar jumps forward again — producing the visible
back-and-forth.

The race window is the entire volume API round-trip:
**debounce fires → PUT /v1/me/player/volume → 200 OK → Interactive GET → store update**
(~200 ms – 700 ms, regular poll interval = 1 s, so this race fires
~20–70 % of the time).

### Why "just patch the store on 200 OK" is not enough

Even if the store were patched the moment the API succeeds, there is still a
window between the debounce clearing `hasPending` and the 200 OK arriving (the
PUT latency itself). Patching the store would only close the second half of the
window. The fix must keep `hasPending = true` through the entire round-trip.

### Fix summary

1. `HandleDebounce` no longer clears `hasPending` — the bar stays in pending
   mode until the API call itself confirms (or fails).
2. `HandleDebounce` returns a third value — `intentSeq` (the seq value that
   matched) — so callers can forward it through the message chain.
3. Two new methods on `GradientVolumeBar`:
   - `ConfirmFromAPI(intentSeq, vol int)` — confirms the value if no newer burst
     has started (seq check prevents clobbering a concurrent burst).
   - `CancelPending(intentSeq int)` — clears `hasPending` without changing
     `currentVol` when an API error occurs; same seq guard.
4. `VolumeIntentMsg` gains a `Seq int` field so the seq flows all the way from
   debounce through the intent message into `buildSetVolumeCmd`.
5. `buildSetVolumeCmd` now returns `VolumeAppliedMsg` (success or error) instead
   of `PlaybackCmdSentMsg`, so all outcomes — including 429 and 401 — route
   through the new handler that calls `ConfirmFromAPI` / `CancelPending` before
   dispatching downstream effects.

---

## Design

### Change 1 — `GradientVolumeBar` updates

**File:** `internal/ui/components/gradient.go`

#### `HandleDebounce` — remove `hasPending = false`, add third return value

```go
// Before:
func (b *GradientVolumeBar) HandleDebounce(m VolumeDebounceTickMsg) (matched bool, targetVol int) {
    if m.Seq != b.seq {
        return false, 0
    }
    b.hasPending = false     // ← THIS IS THE BUG SOURCE — remove it
    b.seq++
    return true, m.TargetVol
}

// After:
// HandleDebounce checks whether the debounce tick is current.
// Returns (true, targetVol, intentSeq) when matched — the caller must forward
// intentSeq through VolumeIntentMsg so ConfirmFromAPI/CancelPending can guard
// against concurrent bursts. hasPending stays true until the API call returns.
// Returns (false, 0, 0) when the tick is stale (newer keypress superseded it).
func (b *GradientVolumeBar) HandleDebounce(m VolumeDebounceTickMsg) (matched bool, targetVol, intentSeq int) {
    if m.Seq != b.seq {
        return false, 0, 0
    }
    b.seq++ // double-fire guard: any future tick with this same seq is now stale
    return true, m.TargetVol, m.Seq
}
```

**Why `intentSeq = m.Seq` (not `b.seq` after increment):**
After `b.seq++`, `b.seq = m.Seq + 1`. `ConfirmFromAPI` checks
`b.seq == intentSeq + 1` — i.e., no new keypress happened. Passing `m.Seq`
(the pre-increment value) is the correct operand for that check.

#### New `ConfirmFromAPI(intentSeq, vol int)`

```go
// ConfirmFromAPI sets currentVol to the API-confirmed value and clears hasPending,
// but only when no newer burst has started. If the user pressed again while the
// API call was in flight (b.seq > intentSeq+1), the confirmation is discarded
// and the new burst's debounce will produce its own VolumeAppliedMsg.
func (b *GradientVolumeBar) ConfirmFromAPI(intentSeq, vol int) {
    if b.seq == intentSeq+1 {
        b.currentVol = vol
        b.hasPending = false
    }
}
```

#### New `CancelPending(intentSeq int)`

```go
// CancelPending clears hasPending without changing currentVol, only when no
// newer burst has started. Call this on API error so the next Spotify poll can
// reconcile the bar via SetConfirmed. If a newer burst is in flight the guard
// fires and the bar correctly stays pending for that burst.
func (b *GradientVolumeBar) CancelPending(intentSeq int) {
    if b.seq == intentSeq+1 {
        b.hasPending = false
    }
}
```

---

### Change 2 — `VolumeIntentMsg` and `VolumeAppliedMsg`

**File:** `internal/ui/panes/messages.go`

```go
// VolumeIntentMsg is emitted by NowPlayingPane after the volume debounce
// resolves. TargetVol is the exact percentage to set. Seq is the debounce
// sequence number; it is threaded through to VolumeAppliedMsg so the bar
// can guard ConfirmFromAPI / CancelPending against concurrent bursts.
type VolumeIntentMsg struct {
    TargetVol int
    Seq       int // intentSeq returned by HandleDebounce
}

// VolumeAppliedMsg is returned by buildSetVolumeCmd after the Spotify volume
// API call completes (success or failure). It replaces PlaybackCmdSentMsg for
// the volume-specific path so the bar's pending state is managed correctly.
//
// On success: Vol holds the confirmed volume, Err is nil.
// On error:   Vol is 0, Err holds the underlying error (may be *api.RateLimitError,
//             *api.UnauthorizedError, or a generic error).
type VolumeAppliedMsg struct {
    Vol int
    Seq int
    Err error
}
```

---

### Change 3 — `NowPlayingPane` wiring

**File:** `internal/ui/panes/nowplaying.go`

#### `Update()` — forward `intentSeq` in the debounce match arm

```go
// Before:
case components.VolumeDebounceTickMsg:
    if matched, vol := p.volumeBar.HandleDebounce(m); matched {
        return p, func() tea.Msg { return VolumeIntentMsg{TargetVol: vol} }
    }
    return p, nil

// After:
case components.VolumeDebounceTickMsg:
    if matched, vol, seq := p.volumeBar.HandleDebounce(m); matched {
        return p, func() tea.Msg { return VolumeIntentMsg{TargetVol: vol, Seq: seq} }
    }
    return p, nil
```

#### `Update()` — add `VolumeAppliedMsg` case

```go
case VolumeAppliedMsg:
    if m.Err != nil {
        p.volumeBar.CancelPending(m.Seq)
    } else {
        p.volumeBar.ConfirmFromAPI(m.Seq, m.Vol)
    }
    return p, nil
```

---

### Change 4 — App layer

**File:** `internal/app/commands.go`

Change `buildSetVolumeCmd` to accept `intentSeq` and return `VolumeAppliedMsg`
instead of `PlaybackCmdSentMsg` for all non-nil-player outcomes:

```go
// buildSetVolumeCmd creates a command that calls player.SetVolume with the
// exact target volume. On completion it returns VolumeAppliedMsg so the bar's
// pending state is confirmed or cancelled before downstream effects fire.
// intentSeq must match the seq value from the VolumeIntentMsg that triggered
// this command — it is forwarded to ConfirmFromAPI / CancelPending.
func (a *App) buildSetVolumeCmd(targetVol, intentSeq int) tea.Cmd {
    player := a.player
    return func() tea.Msg {
        if player == nil {
            return panes.PlaybackCmdSentMsg{Err: errNilClient} // keep: startup sentinel
        }
        ctx := api.WithPriority(context.Background(), api.Interactive)
        err := player.SetVolume(ctx, targetVol)
        if err != nil {
            return panes.VolumeAppliedMsg{Seq: intentSeq, Err: err}
        }
        return panes.VolumeAppliedMsg{Vol: targetVol, Seq: intentSeq}
    }
}
```

Note: the nil-player branch still returns `PlaybackCmdSentMsg{Err: errNilClient}`.
The existing handler silently ignores `errNilClient` — no change needed there.

**File:** `internal/app/handlers.go`

#### `VolumeIntentMsg` handler — pass `m.Seq` to `buildSetVolumeCmd`

```go
case panes.VolumeIntentMsg:
    if !a.store.IsPremium() {
        return a, a.toasts.Cmd(uikit.Toast{
            Intent: uikit.ToastWarning,
            Title:  "Spotify Premium required",
        })
    }
    return a, a.buildSetVolumeCmd(m.TargetVol, m.Seq)
```

#### New `VolumeAppliedMsg` handler

Add this case to `handleMsg`, near the existing `VolumeIntentMsg` case:

```go
case panes.VolumeAppliedMsg:
    // Always confirm or cancel the bar's pending state first, before dispatching
    // downstream effects. This prevents concurrent polls from overriding the bar.
    if np := a.nowPlayingPane(); np != nil {
        updated, _ := np.Update(m)
        if pp, ok := updated.(*panes.NowPlayingPane); ok {
            a.panes[layout.PaneNowPlaying] = pp
        }
    }
    if m.Err != nil {
        // Re-route typed errors to their existing handlers after bar is cleared.
        var rateLimitErr *api.RateLimitError
        if errors.As(m.Err, &rateLimitErr) {
            return a.handleMsg(panes.RateLimitedMsg{RetryAfterSecs: rateLimitErr.RetryAfter})
        }
        if isUnauthorizedError(m.Err) {
            return a.handleMsg(unauthorizedMsg{})
        }
        return a, tea.Batch(
            fetchPlaybackStateCmd(a.player, api.Interactive),
            a.toasts.Cmd(uikit.Toast{
                Intent: uikit.ToastError,
                Title:  "Volume change failed",
                Body:   m.Err.Error(),
            }),
        )
    }
    return a, fetchPlaybackStateCmd(a.player, api.Interactive)
```

---

### Seq guard — worked example

```
t=0ms    HandleKey(+1, 49)           b.seq=1, hasPending=true, currentVol=50
t=100ms  HandleKey(+1, -)            b.seq=2, currentVol=51   (accumulates)
t=300ms  HandleDebounce({Seq:1})     stale (b.seq=2≠1) → discard
t=400ms  HandleDebounce({Seq:2})     match → b.seq=3, hasPending=true (NOT cleared)
         VolumeIntentMsg{50, seq=2}
         buildSetVolumeCmd(50, 2) → PUT in flight

t=450ms  Regular poll fires          SetConfirmed(49) → hasPending=true → no-op ✓
t=500ms  HandleKey(+1, -)            b.seq=4, currentVol=52   (new burst!)

t=700ms  PUT 200 OK                  VolumeAppliedMsg{Vol:50, Seq:2}
         ConfirmFromAPI(2, 50):      b.seq=4 ≠ 2+1=3 → discard ✓ (new burst in flight)
         Interactive poll dispatched

t=800ms  HandleDebounce({Seq:4})     match → b.seq=5, hasPending=true
         VolumeIntentMsg{52, seq=4}
         buildSetVolumeCmd(52, 4) → PUT in flight

t=1100ms PUT 200 OK                  VolumeAppliedMsg{Vol:52, Seq:4}
         ConfirmFromAPI(4, 52):      b.seq=5 == 4+1=5 → hasPending=false, currentVol=52 ✓
         Interactive poll dispatched

t=1200ms Interactive poll returns 52 SetConfirmed(52) → hasPending=false → currentVol=52 (no change) ✓
```

---

## Acceptance Criteria

- [ ] Pressing `+` once from vol=49, then immediately holding, never causes the
      bar to visually snap back to 49 during the API round-trip
- [ ] `ConfirmFromAPI(intentSeq, vol)` with a matching seq updates `currentVol`
      and clears `hasPending`
- [ ] `ConfirmFromAPI(intentSeq, vol)` with a mismatched seq (new burst in
      flight) is a no-op — bar stays pending with the burst's accumulated value
- [ ] `CancelPending(intentSeq)` with matching seq clears `hasPending` without
      changing `currentVol`
- [ ] `CancelPending(intentSeq)` with mismatched seq is a no-op
- [ ] `HandleDebounce` no longer clears `hasPending`; after a matched call the
      bar remains in pending mode until `ConfirmFromAPI` or `CancelPending` runs
- [ ] `HandleDebounce` returns `(matched bool, targetVol, intentSeq int)`; the
      `intentSeq` value equals `m.Seq` from the tick that matched
- [ ] `VolumeIntentMsg` carries `Seq int`; `NowPlayingPane` forwards it from
      the `HandleDebounce` return
- [ ] `buildSetVolumeCmd` returns `VolumeAppliedMsg` (not `PlaybackCmdSentMsg`)
      for success and non-nil-client errors
- [ ] A 429 from `SetVolume` clears the bar's pending state and then triggers
      the existing `RateLimitedMsg` backoff/toast path
- [ ] A 401 from `SetVolume` clears the bar's pending state and then triggers
      the existing `unauthorizedMsg` token-refresh path
- [ ] `make ci` passes (lint + tests + ≥ 80% coverage)

---

## Tasks

### Task 1 — `GradientVolumeBar` new methods + `HandleDebounce` signature change

**Files:** `internal/ui/components/gradient.go`, `internal/ui/components/gradient_test.go`

- [ ] In `gradient_test.go`, rewrite `TestVolumeBar_HandleDebounce_ClearsPending`
      as `TestVolumeBar_HandleDebounce_DoesNotClearPending`:
      after `HandleDebounce` matches, assert that a subsequent `SetConfirmed(30)`
      is still a **no-op** (hasPending is still true), verifying via `Render()`
      that currentVol has not changed to 30.
      Also update every `matched, vol := b.HandleDebounce(...)` call in the
      test file to `matched, vol, _ := b.HandleDebounce(...)`.
      Add table-driven tests for `ConfirmFromAPI` and `CancelPending`:
      - `TestVolumeBar_ConfirmFromAPI_ConfirmsOnSeqMatch` — seq matches → hasPending=false, currentVol=vol
      - `TestVolumeBar_ConfirmFromAPI_NoOpOnSeqMismatch` — new burst (seq+2) → no-op
      - `TestVolumeBar_CancelPending_ClearsOnSeqMatch` — seq matches → hasPending=false, currentVol unchanged
      - `TestVolumeBar_CancelPending_NoOpOnSeqMismatch` — new burst → no-op
      - test: `go test ./internal/ui/components/... -run "TestVolumeBar_HandleDebounce|TestVolumeBar_Confirm|TestVolumeBar_Cancel" -v` → compile error (undefined)
- [ ] In `gradient.go`, change `HandleDebounce` return signature to
      `(matched bool, targetVol, intentSeq int)`; remove `b.hasPending = false`.
      Add `ConfirmFromAPI(intentSeq, vol int)` and `CancelPending(intentSeq int)`.
      - test: same run → PASS; full `./internal/ui/components/...` → PASS
- [ ] `make ci` passes

### Task 2 — Messages: `VolumeIntentMsg.Seq` + new `VolumeAppliedMsg`

**Files:** `internal/ui/panes/messages.go`

- [ ] Add `Seq int` to `VolumeIntentMsg`; update its doc comment.
- [ ] Add `VolumeAppliedMsg` struct with `Vol, Seq int` and `Err error` fields;
      add doc comment per the Design section.
- [ ] `go build ./...` → no errors (existing callers still compile because `Seq`
      is zero-valued by default — the seq check in ConfirmFromAPI will catch mismatches)
- [ ] `make ci` passes

### Task 3 — `NowPlayingPane` wiring

**Files:** `internal/ui/panes/nowplaying.go`, `internal/ui/panes/nowplaying_test.go`

- [ ] In `nowplaying_test.go`:
      - Update `TestNowPlayingPane_VolumeDebounceMsg_EmitsVolumeIntent` to also
        assert `assert.Equal(t, 1, intent.Seq)`.
      - Add `TestNowPlayingPane_VolumeAppliedMsg_Success_ConfirmsBar`: send
        `VolumeAppliedMsg{Vol: 55, Seq: 1}` to a pane where `HandleKey` was
        called once (so bar seq=2 after HandleDebounce would set it to 2 — but
        we need to prime it correctly). Use `newPaneWithVolume(49)`, call `HandleKey`,
        call `HandleDebounce` (discarding cmd), then send the `VolumeAppliedMsg`.
        Assert bar renders 55 and that `SetConfirmed(0)` now changes the bar
        (hasPending=false after confirm).
      - Add `TestNowPlayingPane_VolumeAppliedMsg_Error_CancelsPending`: similar
        setup, send `VolumeAppliedMsg{Err: errors.New("fail"), Seq: 1}`.
        Assert bar still shows optimistic value (currentVol unchanged), and
        `SetConfirmed(0)` is now accepted (hasPending=false after cancel).
      - test: `go test ./internal/ui/panes/... -run "TestNowPlayingPane_Volume" -v` → compile/FAIL
- [ ] In `nowplaying.go`, update the `VolumeDebounceTickMsg` case to use three-return
      `HandleDebounce` and set `Seq: seq` in the `VolumeIntentMsg`.
      Add `VolumeAppliedMsg` case per the Design section.
      - test: same run → PASS; full pane suite → PASS
- [ ] `make ci` passes

### Task 4 — App layer: `buildSetVolumeCmd` + `VolumeAppliedMsg` handler

**Files:** `internal/app/commands.go`, `internal/app/handlers.go`,
`internal/app/volume_test.go`, `internal/app/volume_internal_test.go`

- [ ] In `volume_test.go`:
      - Update `TestApp_VolumeIntentMsg_CallsSetVolume`: the `cmd()` result is now
        `VolumeAppliedMsg` (not `PlaybackCmdSentMsg`); assert `assert.NoError(t, sent.Err)`
        and `assert.Equal(t, 72, mock.LastSetVolume)`.
      - Update `TestApp_VolumeDebounce_FiveRapidPresses_SendsOneCall`: the final
        `sentMsg` is `VolumeAppliedMsg`; fix type assertion accordingly.
      - Update `TestBuildSetVolumeCmd_429_EmitsRateLimitedMsg`: `buildSetVolumeCmd`
        now returns `VolumeAppliedMsg{Err: &api.RateLimitError{...}}`; assert type
        is `VolumeAppliedMsg` and `errors.As(sent.Err, &rl)` matches.
      - Add `TestApp_VolumeAppliedMsg_Success_DispatchesInteractivePoll`: send
        `VolumeAppliedMsg{Vol: 72, Seq: 1}` to app; assert the returned cmd is
        non-nil (the Interactive fetch cmd); `mock.FetchPlaybackStateCalled` is
        not directly observable, so just assert `cmd != nil`.
      - Add `TestApp_VolumeAppliedMsg_429_ClearsPendingAndBacksOff`: inject
        `VolumeAppliedMsg{Seq: 1, Err: &api.RateLimitError{RetryAfter: 5}}`; the
        handler calls `handleMsg(RateLimitedMsg{5})` — assert the returned batch
        contains a cmd (the backoff tick) and that a toast was queued.
        (Use the existing `newVolumeTestApp` helper.)
      - test: `go test ./internal/app/... -run "TestApp_Volume|TestBuildSetVolume" -v` → compile/FAIL
- [ ] In `volume_internal_test.go`:
      - Update `TestBuildSetVolumeCmd_401_EmitsUnauthorized`: result is now
        `VolumeAppliedMsg{Err: <unauthorized error>}`.
- [ ] In `commands.go`, change `buildSetVolumeCmd` signature to
      `(targetVol, intentSeq int)` and update the return types per the Design
      section. Keep the nil-player branch returning `PlaybackCmdSentMsg{Err: errNilClient}`.
- [ ] In `handlers.go`, update `VolumeIntentMsg` case to pass `m.Seq` to
      `buildSetVolumeCmd`. Add `VolumeAppliedMsg` case per the Design section;
      import `errors` if not already imported.
  - test: `go build ./...` → clean; volume tests → PASS
- [ ] `make ci` passes
