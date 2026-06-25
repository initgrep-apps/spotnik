---
title: "Cross-cutting integration flows"
feature: 21-test-infrastructure
status: open
---

## Background

Individual pane and overlay tests verify components in isolation. This story adds
integration tests for cross-cutting behaviors that span multiple panes and the root App
model: focus rotation, overlay lifecycle (input capture + Esc close + focus restore),
error→toast routing, and 401/429 resilience. These tests use the full `app.App` model
with mock API backends.

## Design

### Focus rotation: `internal/app/navigation_flow_test.go`

```go
func TestFocusRotation_TabCyclesThroughVisiblePanes(t *testing.T) {
    // Dashboard preset: 8 visible panes
    // 1. App starts with NowPlaying focused
    // 2. Send Tab 8 times → assert focus returns to NowPlaying
    // 3. Send Shift+Tab → focus moves backward
}

func TestFocusRotation_KeysRoutedToFocusedPaneOnly(t *testing.T) {
    // NowPlaying focused, Queue not
    // Send 'f' (filter) → assert QueuePane filter NOT activated (NowPlaying has no filter)
    // Tab to Queue → Send 'f' → assert QueuePane filter activated
}
```

### Overlay lifecycle: `internal/app/overlay_flow_test.go`

```go
func TestOverlayLifecycle_OpenCapturesAllKeys_CloseRestoresFocus(t *testing.T) {
    // 1. Send 't' → ThemeOverlay opens, keys no longer route to panes
    // 2. Send 'j' (scroll down) → ThemeOverlay cursor moves, QueuePane does NOT scroll
    // 3. Send Esc → ThemeOverlay closes, focus returns to previously focused pane
    // 4. Send 'j' → QueuePane scrolls
}

func TestOverlayLifecycle_PlaybackKeysWorkDuringOverlay(t *testing.T) {
    // ThemeOverlay open
    // Send Space → NowPlaying still receives PlaybackRequestMsg (playback keys always route)
}
```

### Error → toast: `internal/app/error_flow_test.go`

```go
func TestErrorToToast_401_TriggersRefresh(t *testing.T) {
    // 1. Send any Msg with Err = 401
    // 2. Assert tokenRefreshedMsg returned as cmd
    // 3. Token refresh succeeds → original request retried
}

func TestErrorToToast_429_ShowsRateLimitToast(t *testing.T) {
    // 1. Send Msg with Err = RateLimitError{RetryAfterSecs: 30}
    // 2. Assert View() contains "rate limit" toast text
    // 3. Assert backoffTicks set in app state
}

func TestErrorToToast_403_ShowsPremiumRequired(t *testing.T) {
    // 1. Free tier user presses Space
    // 2. Assert View() contains "Spotify Premium required" toast
}

func TestErrorToToast_PollingFailure_Throttled(t *testing.T) {
    // 1. 3 consecutive playback poll failures
    // 2. Assert toast appears on 3rd failure (not 1st, not 2nd)
    // 3. 4th failure → no duplicate toast
}
```

### Page toggle: `internal/app/page_flow_test.go`

```go
func TestPageToggle_CyclesPlayerToStats(t *testing.T) {
    // 1. Player page active → Send '0' → Stats page active
    // 2. Assert View() contains Stats preset panes (GatewayHealth, etc.)
    // 3. Send '0' again → Player page active
}
```

## Files

### Create

- `internal/app/navigation_flow_test.go`
- `internal/app/overlay_flow_test.go`
- `internal/app/error_flow_test.go`
- `internal/app/page_flow_test.go`

## Acceptance Criteria

- [ ] Focus rotation: Tab/Shift+Tab cycles through visible panes
- [ ] Focus rotation: keys only route to focused pane (not unfocused ones)
- [ ] Overlay lifecycle: open overlay captures all keys except playback
- [ ] Overlay lifecycle: Esc closes overlay and restores previous focus
- [ ] Error→toast: 401 triggers token refresh
- [ ] Error→toast: 429 shows rate limit toast + backoff
- [ ] Error→toast: 403 shows Premium required toast
- [ ] Error→toast: polling failures throttled (toast on 3rd consecutive)
- [ ] Page toggle: '0' cycles Player→Stats→Player
- [ ] `make ci` passes

## Tasks

- [ ] Create focus rotation integration tests
      - test: `TestFocusRotation_TabCyclesThroughVisiblePanes`, `TestFocusRotation_KeysRoutedToFocusedPaneOnly`
- [ ] Create overlay lifecycle integration tests
      - test: `TestOverlayLifecycle_OpenCapturesAllKeys_CloseRestoresFocus`, `TestOverlayLifecycle_PlaybackKeysWorkDuringOverlay`
- [ ] Create error→toast integration tests
      - test: `TestErrorToToast_401_TriggersRefresh`, `TestErrorToToast_429_ShowsRateLimitToast`, `TestErrorToToast_403_ShowsPremiumRequired`, `TestErrorToToast_PollingFailure_Throttled`
- [ ] Create page toggle integration test
      - test: `TestPageToggle_CyclesPlayerToStats`
- [ ] Verify all tests pass with mock API backends
