---
title: "Reject Interactive Requests During Backoff"
feature: 27-gateway-rate-protection
status: done
---

## Background

### What F26 left behind

Story 125 removed the `interactiveDebounce` — correct, because the debounce
was silently dropping semantically independent playback commands. But the
gateway's backoff policy for Interactive requests was not re-examined at the
same time. That policy has a serious flaw that only manifests at scale: when a
user *holds* a playback key and hits a 429.

### The failing code path — annotated

```go
// internal/api/gateway.go  Do()  lines 334–344  (CURRENT — BROKEN)

if priority == Interactive {
    // Reads backoffUntil under lock, then IMMEDIATELY drops the lock.
    g.mu.Lock()
    waited = time.Now().Before(g.backoffUntil)
    g.mu.Unlock()

    // *** THIS BLOCKS THE GOROUTINE ***
    // Every Interactive goroutine that enters Do() while backoffUntil is in
    // the future calls waitForBackoff and parks here.
    if err := g.waitForBackoff(ctx); err != nil {
        g.emitEvent(domain.EventRequestWaited, ...)
        return nil, err
    }
}

// waitForBackoff — lines 558–581
func (g *Gateway) waitForBackoff(ctx context.Context) error {
    for {  // ← FOR LOOP — re-checks after each timer fires
        g.mu.Lock()
        until := g.backoffUntil
        g.mu.Unlock()

        remaining := time.Until(until)
        if remaining <= 0 {
            return nil
        }
        timer := time.NewTimer(remaining)
        select {
        case <-ctx.Done():
            timer.Stop()
            return ctx.Err()
        case <-timer.C:
            // Loops back — if a NEW 429 extended backoffUntil while this
            // goroutine was sleeping, it will sleep AGAIN.
        }
    }
}
```

### Proof by simulation — BEFORE fix

The user holds `-` (volume down) for 3 seconds. OS key-repeat fires at ~15
events/second. Token bucket is BYPASSED for Interactive (Bug 2, Story 127).

```
t=0s:     User presses and holds '-'
t=0ms:    PUT #1 fires immediately — no backoff active          → Spotify: 200
t=66ms:   PUT #2 fires immediately                               → Spotify: 200
t=133ms:  PUT #3 fires                                           → Spotify: 200
...       (bucket bypass: all requests fire without rate check)
t=900ms:  PUT #14 fires                                          → Spotify: 429
          backoffUntil = now + 10s

t=901ms:  PUT #15 enters Do() → Interactive → waitForBackoff()   ← GOROUTINE PARKS
t=967ms:  PUT #16 enters Do() → Interactive → waitForBackoff()   ← GOROUTINE PARKS
t=1033ms: PUT #17 enters Do() → Interactive → waitForBackoff()   ← GOROUTINE PARKS
...
t=3000ms: User releases key.  PUTs #15–#46 all parked in waitForBackoff.

t=10s:    backoffUntil expires.  All 32 goroutines wake simultaneously.
          Token bucket BYPASSED → 32 PUTs fire as a burst against Spotify.
          Spotify returns 429 on request #11 of the burst.
          backoffUntil = now + 10s  (RESET)

          waitForBackoff's for loop: remaining goroutines see new deadline.
          They sleep AGAIN.

t=20s:    Second backoff expires.  Remaining goroutines burst again → another 429.
          Cycle continues.

Observable in requestflow pane: PUT /v1/me/player/volume in "wait" state for
MINUTES even though user released the key seconds ago.
```

The `for` loop in `waitForBackoff` is what turns a 10-second event into an
indefinite cascade.

### Root cause in one sentence

Interactive requests during backoff should be rejected, not queued — stale
volume PUTs from a key-hold that ended seconds ago have zero value once the
backoff expires.

---

### Second problem: incomplete `RateLimitError` propagation in command layer

After the gateway fix, every Interactive request rejected during backoff returns
`*api.RateLimitError` from `gw.Do()`. The command layer must convert this to
`panes.RateLimitedMsg` so the existing `RateLimitedMsg` handler in `handlers.go`
can run its required side effects:

- emit the friendly "Rate limited, retrying in Ns" toast
- update `store.SetThrottle()` for UI observability
- call `clearAllFetchingSentinels()` so stale fetching flags do not permanently
  block re-fetches after backoff expires
- schedule `throttleExpiredMsg` to clear throttle state after Retry-After seconds

The existing pattern used by most Interactive commands is correct:

```go
// Pattern A — command-level intercept (CORRECT)
if secs := parse429RetryAfter(err); secs > 0 {
    return panes.RateLimitedMsg{RetryAfterSecs: secs}
}
```

`parse429RetryAfter` uses `errors.As` which unwraps the `fmt.Errorf("sending
request: %w", err)` wrapper applied by `BaseClient.doJSON`/`doNoContent`, so it
correctly catches both Spotify 429 responses and gateway-level rejections
(both produce `*api.RateLimitError`).

Auditing every Interactive command in `internal/app/commands.go`:

| Command | Result message | `parse429RetryAfter`? |
|---|---|---|
| `buildPlaybackAPICmd` | `PlaybackCmdSentMsg` | ✓ yes |
| `buildPlayContextCmd` | `PlaybackCmdSentMsg` | ✓ yes |
| `buildPlayTrackListCmd` | `PlaybackCmdSentMsg` | ✓ yes |
| `buildAddToQueueCmd` | `AddToQueueResultMsg` | ✓ yes |
| `buildFetchDevicesCmd` | `DevicesLoadedMsg` | ✓ yes |
| `buildFetchCurrentUserCmd` | `userProfileLoadedMsg` | ✓ yes |
| `buildFetchPlaylistTracksCmd` | `PlaylistTracksLoadedMsg` | ✓ yes |
| `buildFetchAlbumTracksCmd` | `AlbumTracksLoadedMsg` | ✓ yes |
| `buildSearchPageCmd` | `SearchPageLoadedMsg` | ✓ yes |
| **`buildTransferPlaybackCmd`** | `DeviceTransferredMsg` | **✗ missing** |
| **`buildRemovePlaylistTrackCmd`** | `PlaylistRemoveResultMsg` | **✗ missing** |

The two missing commands pass the raw `*api.RateLimitError` through to their
result messages. Their handlers surface a raw internal error string to the user
and take incorrect actions:

**`buildTransferPlaybackCmd` gap:**
```go
// internal/app/commands.go  buildTransferPlaybackCmd  (CURRENT — INCOMPLETE)
err := devices.TransferPlayback(api.WithPriority(..., api.Interactive), deviceID, true)
return panes.DeviceTransferredMsg{DeviceID: deviceID, Err: err}
// ↑ RateLimitError is not intercepted — it leaks into the message Err field
```

`DeviceTransferredMsg` handler (`handlers.go:876`):
```go
case panes.DeviceTransferredMsg:
    if m.Err != nil {
        ...
        return a, tea.Batch(
            fetchPlaybackStateCmd(a.player, api.Background),  // ← WRONG: no HTTP was made
            a.alerts.NewAlertCmd("error", m.Err.Error()),     // ← raw "rate limit: retry after 10s"
        )
    }
```

Two problems:
1. Raw internal error string shown to user instead of friendly toast.
2. `fetchPlaybackStateCmd(a.player, api.Background)` dispatched even though the
   request never reached Spotify — there is no state change to reconcile. This
   Background fetch will itself be immediately rejected (backoff active), wasting
   a token and adding noise to the network log.

**`buildRemovePlaylistTrackCmd` gap:**
```go
// internal/app/commands.go  buildRemovePlaylistTrackCmd  (CURRENT — INCOMPLETE)
err := playlistsAPI.RemoveTracksFromPlaylist(api.WithPriority(..., api.Interactive), ...)
return panes.PlaylistRemoveResultMsg{PlaylistID: playlistID, TrackURI: trackURI, Err: err}
// ↑ RateLimitError leaks into Err field
```

`PlaylistRemoveResultMsg` handler (`routing.go:366`):
```go
case panes.PlaylistRemoveResultMsg:
    ...
    if m.Err != nil {
        return a, tea.Batch(cmd, a.alerts.NewAlertCmd("error", m.Err.Error()))
        // ← raw "rate limit: retry after 10s" shown as generic error toast
    }
```

In both cases `clearAllFetchingSentinels()` is never called, so any fetching
sentinel active at the time of the 429 remains set after backoff expires —
permanently blocking re-fetches until the user navigates away and back.

---

## Design

### `internal/api/gateway.go` — Phase 1 Interactive branch

Replace the `waitForBackoff` call with an immediate rejection, identical in
shape to the Background rejection that already exists.

```go
// BEFORE — lines 334–344
if priority == Interactive {
    g.mu.Lock()
    waited = time.Now().Before(g.backoffUntil)
    g.mu.Unlock()
    if err := g.waitForBackoff(ctx); err != nil {
        g.emitEvent(domain.EventRequestWaited, reqID, key.Method, key.Path, domainPriority, 0, 0)
        return nil, err
    }
}

// AFTER
if priority == Interactive {
    g.mu.Lock()
    throttled := time.Now().Before(g.backoffUntil)
    retryAfter := g.retryAfter
    if throttled {
        g.emitEventLocked(domain.EventRequestBlocked, reqID, key.Method, key.Path, domainPriority, 0, 0)
    }
    g.mu.Unlock()
    if throttled {
        return nil, &RateLimitError{RetryAfter: retryAfter}
    }
}
```

The `waited bool` field at the top of `Do()` and both `EventRequestWaited`
emission sites (lines 342 and 535–538) are removed. The final event for all
non-blocked Interactive requests becomes `EventRequestAllowed` unconditionally.

### `internal/api/gateway.go` — remove dead code

```go
// DELETE: waitForBackoff function (lines 558–581)
// DELETE: waited bool variable (line 330)
// DELETE: EventRequestWaited emission on context cancel (lines 341–343)
// DELETE: EventRequestWaited emission at end of Do() (lines 535–538)

// UPDATE Gateway doc comment — remove:
//   "429 backoff with priority bypass for Interactive requests"
// REPLACE with:
//   "429 backoff: both priorities are rejected immediately; Interactive
//    requests are not queued so stale commands do not pile up"
```

### `internal/api/gateway_hardening_test.go` — remove waitForBackoff tests

```go
// DELETE: TestWaitForBackoff_ContextCancelReturnsImmediately (lines 82–101)
// DELETE: TestWaitForBackoff_CompletesAfterDuration (lines 103–119)
// These tested a function that no longer exists.
```

### `internal/app/commands.go` — add `parse429RetryAfter` to the two incomplete commands

Both commands follow the same pattern as every other Interactive command.
`parse429RetryAfter` catches `*api.RateLimitError` via `errors.As`, unwrapping
the `fmt.Errorf("sending request: %w", ...)` wrapper applied by `BaseClient`.

```go
// BEFORE — buildTransferPlaybackCmd
func (a *App) buildTransferPlaybackCmd(deviceID string) tea.Cmd {
    devices := a.devices
    return func() tea.Msg {
        if devices == nil {
            return panes.DeviceTransferredMsg{Err: errNilClient, DeviceID: deviceID}
        }
        err := devices.TransferPlayback(api.WithPriority(context.Background(), api.Interactive), deviceID, true)
        return panes.DeviceTransferredMsg{DeviceID: deviceID, Err: err}
    }
}

// AFTER — buildTransferPlaybackCmd
func (a *App) buildTransferPlaybackCmd(deviceID string) tea.Cmd {
    devices := a.devices
    return func() tea.Msg {
        if devices == nil {
            return panes.DeviceTransferredMsg{Err: errNilClient, DeviceID: deviceID}
        }
        err := devices.TransferPlayback(api.WithPriority(context.Background(), api.Interactive), deviceID, true)
        if err != nil {
            if secs := parse429RetryAfter(err); secs > 0 {
                return panes.RateLimitedMsg{RetryAfterSecs: secs}
            }
            if isUnauthorizedError(err) {
                return unauthorizedMsg{}
            }
        }
        return panes.DeviceTransferredMsg{DeviceID: deviceID, Err: err}
    }
}
```

```go
// BEFORE — buildRemovePlaylistTrackCmd
func (a *App) buildRemovePlaylistTrackCmd(playlistID, trackURI string) tea.Cmd {
    playlistsAPI := a.playlistsAPI
    return func() tea.Msg {
        if playlistsAPI == nil {
            return panes.PlaylistRemoveResultMsg{Err: errNilClient, PlaylistID: playlistID, TrackURI: trackURI}
        }
        err := playlistsAPI.RemoveTracksFromPlaylist(api.WithPriority(context.Background(), api.Interactive), playlistID, []string{trackURI})
        return panes.PlaylistRemoveResultMsg{PlaylistID: playlistID, TrackURI: trackURI, Err: err}
    }
}

// AFTER — buildRemovePlaylistTrackCmd
func (a *App) buildRemovePlaylistTrackCmd(playlistID, trackURI string) tea.Cmd {
    playlistsAPI := a.playlistsAPI
    return func() tea.Msg {
        if playlistsAPI == nil {
            return panes.PlaylistRemoveResultMsg{Err: errNilClient, PlaylistID: playlistID, TrackURI: trackURI}
        }
        err := playlistsAPI.RemoveTracksFromPlaylist(api.WithPriority(context.Background(), api.Interactive), playlistID, []string{trackURI})
        if err != nil {
            if secs := parse429RetryAfter(err); secs > 0 {
                return panes.RateLimitedMsg{RetryAfterSecs: secs}
            }
            if isUnauthorizedError(err) {
                return unauthorizedMsg{}
            }
        }
        return panes.PlaylistRemoveResultMsg{PlaylistID: playlistID, TrackURI: trackURI, Err: err}
    }
}
```

By returning `panes.RateLimitedMsg`, both commands route through the existing
`RateLimitedMsg` handler which calls `clearAllFetchingSentinels()`, updates
`store.SetThrottle()`, emits the friendly toast, and schedules `throttleExpiredMsg`.
The incorrect `fetchPlaybackStateCmd` dispatch in the `DeviceTransferredMsg` handler
is automatically avoided because that handler is never reached.

### `internal/app/handlers.go` — defense-in-depth for `PlaybackCmdSentMsg`

All current `Play*Cmd` functions already call `parse429RetryAfter` and return
`panes.RateLimitedMsg` — so `PlaybackCmdSentMsg.Err` should never be a
`RateLimitError` under normal operation. This handler fix is defense-in-depth:
if a future command variant omits the intercept, the handler shows a friendly
message instead of the raw internal error string.

```go
// BEFORE — lines 501–515
case panes.PlaybackCmdSentMsg:
    if m.Err != nil {
        if errors.Is(m.Err, errNilClient) {
            return a, nil
        }
        var forbiddenErr *api.ForbiddenError
        if errors.As(m.Err, &forbiddenErr) {
            return a, tea.Batch(
                fetchPlaybackStateCmd(a.player, api.Background),
                a.alerts.NewAlertCmd("warning", "Spotify Premium required"),
            )
        }
        return a, tea.Batch(
            fetchPlaybackStateCmd(a.player, api.Background),
            a.alerts.NewAlertCmd("error", m.Err.Error()),
        )
    }

// AFTER — add RateLimitError branch between ForbiddenError and generic
case panes.PlaybackCmdSentMsg:
    if m.Err != nil {
        if errors.Is(m.Err, errNilClient) {
            return a, nil
        }
        var forbiddenErr *api.ForbiddenError
        if errors.As(m.Err, &forbiddenErr) {
            return a, tea.Batch(
                fetchPlaybackStateCmd(a.player, api.Background),
                a.alerts.NewAlertCmd("warning", "Spotify Premium required"),
            )
        }
        var rateLimitErr *api.RateLimitError
        if errors.As(m.Err, &rateLimitErr) {
            // NOTE: no fetchPlaybackStateCmd — the request never reached Spotify,
            // no state change to reconcile.
            // NOTE: this branch is defense-in-depth only. All current Play*Cmd
            // functions intercept RateLimitError at the command level via
            // parse429RetryAfter, returning panes.RateLimitedMsg instead. If
            // a future variant omits the intercept, this branch prevents the raw
            // "rate limit: retry after Ns" string from reaching the user.
            return a, a.alerts.NewAlertCmd("warning",
                fmt.Sprintf("Rate limited — wait %ds before retrying", rateLimitErr.RetryAfter))
        }
        return a, tea.Batch(
            fetchPlaybackStateCmd(a.player, api.Background),
            a.alerts.NewAlertCmd("error", m.Err.Error()),
        )
    }
```

### Simulation — AFTER fix

Same scenario: user holds `-` for 3 seconds at ~15 events/s, hits 429 at t=900ms.

```
t=0ms:    PUT #1 fires → Spotify: 200     (token bucket still bypassed — Story 127)
...
t=900ms:  PUT #14 fires → Spotify: 429
          gateway: sets backoffUntil = now + 10s, returns *RateLimitError
          buildPlaybackAPICmd: parse429RetryAfter catches it
          → panes.RateLimitedMsg{RetryAfterSecs: 10}
          RateLimitedMsg handler: clearAllFetchingSentinels(), SetThrottle(true,10),
          emits "Rate limited, retrying in 10s" toast, schedules throttleExpiredMsg

t=901ms:  PUT #15 enters Do() → Interactive → throttled=true → RateLimitError returned
          buildPlaybackAPICmd: parse429RetryAfter → panes.RateLimitedMsg
          RateLimitedMsg handler: no-ops (sentinel already cleared, throttle already set)
          GOROUTINE EXITS IMMEDIATELY.  No parking.
t=967ms:  PUT #16 → same → exits immediately
t=1033ms: PUT #17 → same → exits immediately
...
t=3000ms: User releases key. All rejected PUTs already returned.

t=10s:    backoffUntil expires.  throttleExpiredMsg fires → store.SetThrottle(false).
          No goroutines waiting. No burst. No cascade. System returns to normal.

Observable: requestflow pane shows PUTs as "blocked" (EventRequestBlocked)
during the backoff window. No "wait" entries. No minutes-long stuck requests.
```

---

## Acceptance Criteria

- [ ] Interactive requests arriving when `backoffUntil` is in the future return
  `&RateLimitError{RetryAfter: g.retryAfter}` immediately — no goroutine parks.
- [ ] `waitForBackoff` function is deleted from `gateway.go`.
- [ ] `waited bool` variable and both `EventRequestWaited` emission sites are
  removed from `Do()`.
- [ ] `TestWaitForBackoff_*` tests are deleted from `gateway_hardening_test.go`.
- [ ] `buildTransferPlaybackCmd` intercepts `RateLimitError` via `parse429RetryAfter`
  and returns `panes.RateLimitedMsg` — `DeviceTransferredMsg.Err` is never a
  `RateLimitError`.
- [ ] `buildRemovePlaylistTrackCmd` intercepts `RateLimitError` via `parse429RetryAfter`
  and returns `panes.RateLimitedMsg` — `PlaylistRemoveResultMsg.Err` is never a
  `RateLimitError`.
- [ ] `PlaybackCmdSentMsg` handler has defense-in-depth `RateLimitError` branch:
  emits friendly toast, no `fetchPlaybackStateCmd`.
- [ ] Background reject-during-backoff behaviour is unchanged.
- [ ] `make ci` passes.

---

## Tasks

- [ ] In `internal/api/gateway.go` `Do()`, replace the Interactive backoff block
  (lines 334–344) with an immediate rejection — read `throttled` and `retryAfter`
  under `g.mu`, emit `EventRequestBlocked` via `emitEventLocked`, unlock, return
  `&RateLimitError{RetryAfter: retryAfter}` if throttled.
  - test: `go build ./internal/api/...` compiles cleanly

- [ ] Remove `waited bool` variable (line ~330) and both `EventRequestWaited`
  emission sites from `Do()` (lines ~342 and ~535–538). Replace the final event
  block with unconditional `EventRequestAllowed`.
  - test: `go build ./internal/api/...` compiles cleanly

- [ ] Delete `waitForBackoff` function from `gateway.go` (lines 558–581).
  - test: `go build ./internal/api/...` compiles cleanly (no references remain)

- [ ] Update `Gateway` doc comment: replace "429 backoff with priority bypass
  for Interactive requests" with the revised wording from the Design section.

- [ ] Delete `TestWaitForBackoff_ContextCancelReturnsImmediately` and
  `TestWaitForBackoff_CompletesAfterDuration` from
  `internal/api/gateway_hardening_test.go`.
  - test: `go test ./internal/api/... -run TestWaitForBackoff` — no tests run,
    zero failures

- [ ] Add `TestGateway_InteractiveRejectedDuringBackoff` to
  `internal/api/gateway_test.go`:
  ```go
  func TestGateway_InteractiveRejectedDuringBackoff(t *testing.T) {
      gw := NewGateway()
      // Set an active backoff.
      gw.mu.Lock()
      gw.retryAfter = 10
      gw.backoffUntil = time.Now().Add(10 * time.Second)
      gw.mu.Unlock()

      key := RequestKey{Method: http.MethodPut, Path: "/v1/me/player/volume", Priority: Interactive}
      calls := 0
      _, err := gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
          calls++
          return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
      })

      require.Error(t, err)
      var rlErr *RateLimitError
      require.ErrorAs(t, err, &rlErr, "must return RateLimitError, not block")
      assert.Equal(t, 10, rlErr.RetryAfter)
      assert.Equal(t, 0, calls, "fn must not be called — request rejected before HTTP")
  }
  ```
  - test: `go test ./internal/api/... -run TestGateway_InteractiveRejectedDuringBackoff` PASS

- [ ] Add `TestGateway_InteractiveAllowedAfterBackoffExpires` to
  `internal/api/gateway_test.go`:
  ```go
  func TestGateway_InteractiveAllowedAfterBackoffExpires(t *testing.T) {
      gw := NewGateway()
      // Set a backoff that has already expired.
      gw.mu.Lock()
      gw.retryAfter = 1
      gw.backoffUntil = time.Now().Add(-1 * time.Millisecond)
      gw.mu.Unlock()

      key := RequestKey{Method: http.MethodPut, Path: "/v1/me/player/volume", Priority: Interactive}
      calls := 0
      resp, err := gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
          calls++
          return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
      })

      require.NoError(t, err)
      assert.Equal(t, 204, resp.StatusCode)
      assert.Equal(t, 1, calls, "fn must be called once when backoff has expired")
  }
  ```
  - test: `go test ./internal/api/... -run TestGateway_InteractiveAllowedAfterBackoffExpires` PASS

- [ ] In `internal/app/commands.go`, add `parse429RetryAfter` + `isUnauthorizedError`
  intercept to `buildTransferPlaybackCmd` between the `errNilClient` guard and the
  final `return panes.DeviceTransferredMsg{...}`, as shown in the Design section.
  - test: `go build ./internal/app/...` compiles cleanly

- [ ] Add `TestBuildTransferPlaybackCmd_RateLimitReturnsRateLimitedMsg` to
  `internal/app/command_safety_test.go`:
  ```go
  func TestBuildTransferPlaybackCmd_RateLimitReturnsRateLimitedMsg(t *testing.T) {
      a := newTestApp(t)
      // Wire a devices client that returns *api.RateLimitError.
      a.devices = &stubDevices{
          transferErr: &api.RateLimitError{RetryAfter: 10},
      }
      cmd := a.buildTransferPlaybackCmd("device-1")
      require.NotNil(t, cmd)

      msg := cmd()

      rlMsg, ok := msg.(panes.RateLimitedMsg)
      require.True(t, ok, "RateLimitError must be converted to panes.RateLimitedMsg, got %T", msg)
      assert.Equal(t, 10, rlMsg.RetryAfterSecs)
  }
  ```
  - test: `go test ./internal/app/... -run TestBuildTransferPlaybackCmd_RateLimitReturnsRateLimitedMsg` PASS

- [ ] In `internal/app/commands.go`, add `parse429RetryAfter` + `isUnauthorizedError`
  intercept to `buildRemovePlaylistTrackCmd` between the `errNilClient` guard and the
  final `return panes.PlaylistRemoveResultMsg{...}`, as shown in the Design section.
  - test: `go build ./internal/app/...` compiles cleanly

- [ ] Add `TestBuildRemovePlaylistTrackCmd_RateLimitReturnsRateLimitedMsg` to
  `internal/app/command_safety_test.go`:
  ```go
  func TestBuildRemovePlaylistTrackCmd_RateLimitReturnsRateLimitedMsg(t *testing.T) {
      a := newTestApp(t)
      // Wire a playlists client that returns *api.RateLimitError.
      a.playlistsAPI = &stubPlaylistsAPI{
          removeErr: &api.RateLimitError{RetryAfter: 5},
      }
      cmd := a.buildRemovePlaylistTrackCmd("pl-1", "spotify:track:abc")
      require.NotNil(t, cmd)

      msg := cmd()

      rlMsg, ok := msg.(panes.RateLimitedMsg)
      require.True(t, ok, "RateLimitError must be converted to panes.RateLimitedMsg, got %T", msg)
      assert.Equal(t, 5, rlMsg.RetryAfterSecs)
  }
  ```
  - test: `go test ./internal/app/... -run TestBuildRemovePlaylistTrackCmd_RateLimitReturnsRateLimitedMsg` PASS

- [ ] In `internal/app/handlers.go`, add `var rateLimitErr *api.RateLimitError` /
  `errors.As` branch in `PlaybackCmdSentMsg` between the `ForbiddenError` and
  generic branches, as shown in the Design section. Import `fmt` if not already
  present.
  - test: `go build ./internal/app/...` compiles cleanly

- [ ] Add `TestHandlers_PlaybackCmdSentMsg_RateLimitToast` to
  `internal/app/handlers_test.go` (or the nearest existing test file for
  `PlaybackCmdSentMsg`):
  ```go
  func TestHandlers_PlaybackCmdSentMsg_RateLimitToast(t *testing.T) {
      a := newTestApp(t)
      msg := panes.PlaybackCmdSentMsg{Err: &api.RateLimitError{RetryAfter: 10}}
      _, cmd := a.Update(msg)
      require.NotNil(t, cmd)

      // Execute the returned command and collect messages.
      msgs := collectMsgs(t, cmd)

      // Must contain exactly one alert with the rate-limit wording.
      found := false
      for _, m := range msgs {
          if alert, ok := m.(alerts.AlertMsg); ok {
              assert.Contains(t, alert.Text, "Rate limited")
              assert.Contains(t, alert.Text, "10s")
              found = true
          }
      }
      assert.True(t, found, "must emit a rate-limit toast")

      // Must NOT contain a fetchPlaybackStateCmd result (no reconcile needed).
      for _, m := range msgs {
          _, isPlayback := m.(panes.PlaybackStateFetchedMsg)
          assert.False(t, isPlayback, "must not dispatch reconcile fetch for rate-limit error")
      }
  }
  ```
  - test: `go test ./internal/app/... -run TestHandlers_PlaybackCmdSentMsg_RateLimitToast` PASS

- [ ] `make ci` passes
