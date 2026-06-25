---
title: "Fix: Overlay rate-limit error delivery"
feature: 02-api-infrastructure
status: done
---

## Background

When overlay-initiated API calls hit a 429 rate limit, the command factories convert
the error to `panes.RateLimitedMsg` instead of the overlay-specific message types
(`userProfileLoadedMsg`, `DevicesLoadedMsg`). The global `RateLimitedMsg` handler
sets backoff and shows a toast, but does **not** forward any error information to the
overlays that are waiting for data.

The result: overlays appear stuck or show misleading states:

- **Profile overlay** — `Init()` emits `FetchCurrentUserRequestMsg` → handler calls
  `buildFetchCurrentUserCmd()` → 429 → returns `RateLimitedMsg` instead of
  `userProfileLoadedMsg`. The overlay never receives `UserProfileLoadedMsg`, so
  `p.err` stays nil and `profile.ID` stays empty. `View()` renders "Loading profile..."
  indefinitely. No retry, no timeout, no error message.

- **Device overlay** — `Init()` emits `FetchDevicesRequestMsg` → handler calls
  `buildFetchDevicesCmd()` → 429 → returns `RateLimitedMsg` instead of
  `DevicesLoadedMsg`. The overlay never receives `DevicesLoadedMsg`, so `d.err`
  stays nil and `d.devices` stays empty. `View()` renders "No devices found" (empty
  state) instead of "Failed to load devices" (error state). The overlay eventually
  recovers on the next poll cycle after backoff, but the initial display is
  misleading — the user thinks no devices exist when the real problem is a rate limit.

- **No automatic retry** — After backoff expires, the `TickMsg` handler only
  re-dispatches playback and queue fetches. There is no periodic user profile fetch
  mechanism. The device overlay's next poll cycle may also be blocked by the
  per-pane `devicesPoll.backoffTicks`.

This is a gap in Story 202 (Overlay Self-Sufficiency). Its acceptance criteria
state that "ProfileOverlay.View() shows 'Profile unavailable' when err != nil"
and "'Loading...' never persists indefinitely", but the 429 error path bypasses
these guarantees because `RateLimitedMsg` is never forwarded to overlays.

## Design

### Approach: Forward rate-limit errors to open overlays in `RateLimitedMsg` handler

When a 429 occurs, the `RateLimitedMsg` handler already knows the backoff duration
and clears all fetching sentinels. The fix adds overlay-aware error delivery:

1. In the `RateLimitedMsg` handler (`handlers.go`), after setting backoff and
   clearing sentinels, check if overlays are open and waiting for data:
   - If `profileOverlayOpen && store.UserProfile().ID == ""`, deliver a synthetic
     `UserProfileLoadedMsg{Err: fmt.Errorf("rate limited, retry in %ds", backoff)}`
     to the profile overlay.
   - If `deviceOverlayOpen && !store.DevicesFetched()`, deliver a synthetic
     `DevicesLoadedMsg{Err: fmt.Errorf("rate limited")}` to the device overlay.

2. This is correct because during backoff, ALL requests are blocked (gateway Phase 1
   rejects immediately). So even if the 429 came from a playback fetch, the profile
   and device fetches would also fail.

3. The overlays already handle these error messages correctly — they set their `err`
   field and render the appropriate error state. No overlay code changes are needed.

4. After backoff expires and data is re-fetched successfully, the normal message flow
   delivers fresh data to the overlays, clearing the error state.

### Files affected

- `internal/app/handlers.go` — `RateLimitedMsg` handler: add overlay error delivery
- `internal/app/handlers_test.go` — new test cases for the overlay error delivery

### No overlay code changes needed

The profile and device overlays already handle error messages correctly:
- `ProfileOverlay.Update()` handles `UserProfileLoadedMsg{Err: ...}` by setting `p.err`
- `ProfileOverlay.View()` shows "Profile unavailable" when `p.err != nil`
- `DeviceOverlay.Update()` handles `DevicesLoadedMsg{Err: ...}` by setting `d.err`
- `DeviceOverlay.View()` shows "Failed to load devices" when `d.err != nil`

The gap is purely in the routing layer — `RateLimitedMsg` is never forwarded to
overlays.

## Acceptance Criteria

- [ ] When `/me` returns 429 and the profile overlay is open, the overlay shows
      "Profile unavailable" instead of "Loading profile..."
- [ ] When device fetch returns 429 and the device overlay is open, the overlay shows
      "Failed to load devices" instead of "No devices found"
- [ ] After 429 backoff expires and data is successfully re-fetched, overlays
      automatically recover (profile shows data, devices show list)
- [ ] Closing and reopening an overlay during backoff shows the error state
      immediately (the overlay's `Init()` triggers a new fetch, which is blocked by
      the gateway, which returns `RateLimitError`, which is converted to
      `RateLimitedMsg`, which the handler forwards to the overlay as an error)
- [ ] Non-overlay 429s (playback, queue, library) continue to work as before —
      global backoff, toast, sentinel clearing, no regression
- [ ] `make ci` passes (lint + tests + ≥ 80% coverage)

## Tasks

- [ ] Write failing tests in `internal/app/handlers_test.go`:
      - `TestRateLimitedMsg_DeliversErrorToProfileOverlay`: when `profileOverlayOpen`
        and store has no user profile, verify `profilePane.Err()` returns a rate-limit
        error after `RateLimitedMsg` is processed
      - `TestRateLimitedMsg_DeliversErrorToDeviceOverlay`: when `deviceOverlayOpen`
        and store has no devices, verify `devicePane.Err()` returns a rate-limit
        error after `RateLimitedMsg` is processed
      - `TestRateLimitedMsg_SkipsProfileOverlayWhenClosed`: verify no error is
        delivered to profile pane when `profileOverlayOpen == false`
      - `TestRateLimitedMsg_SkipsDeviceOverlayWhenClosed`: verify no error is
        delivered to device pane when `deviceOverlayOpen == false`
      - `TestRateLimitedMsg_SkipsProfileOverlayWhenDataLoaded`: verify no error is
        delivered to profile pane when `store.UserProfile().ID != ""`
      - `TestRateLimitedMsg_SkipsDeviceOverlayWhenDataLoaded`: verify no error is
        delivered to device pane when `store.DevicesFetched()` is true
  - test: `go test ./internal/app/ -run "TestRateLimitedMsg_Delivers" -v` → FAIL

- [ ] Add overlay error delivery to the `RateLimitedMsg` handler in
      `internal/app/handlers.go`:
      - After `clearAllFetchingSentinels()` and before the return statement:
        - If `a.profileOverlayOpen && a.store.UserProfile().ID == ""`, create
          `panes.UserProfileLoadedMsg{Err: fmt.Errorf("rate limited, retry in %ds", backoff)}`
          and forward to `a.profilePane.Update(msg)`
        - If `a.deviceOverlayOpen && !a.store.DevicesFetched()`, create
          `panes.DevicesLoadedMsg{Err: fmt.Errorf("rate limited")}`
          and forward to `a.devicePane.Update(msg)`
      - Batch any resulting commands with the existing return
  - test: `go test ./internal/app/ -run "TestRateLimitedMsg_Delivers" -v` → PASS

- [ ] `make ci` passes