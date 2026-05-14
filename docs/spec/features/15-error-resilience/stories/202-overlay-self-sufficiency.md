---
title: "Overlay Self-Sufficiency"
feature: 15-error-resilience
status: done
---

## Background

Two overlays have gaps that leave users stuck:

**Profile overlay** — `Init()` returns `nil`. If the startup user-profile fetch failed, the
overlay shows `"Loading..."` forever. No self-fetch, no error state.

**Devices overlay** — on fetch failure, shows `"No devices found"` — identical to the
legitimate empty state. User cannot tell whether Spotify is offline or simply has no active
devices. Additionally, since Story 199 drives devices polling only while the overlay is open,
this story adds the `err` field to surface fetch errors distinctly.

Two new message types are needed: `FetchCurrentUserRequestMsg` (emitted by `ProfileOverlay.Init()`)
and `UserProfileLoadedMsg` (forwarded to the overlay from the app-level `userProfileLoadedMsg`
handler).

## Design

### `internal/ui/panes/messages.go` — new message types

Add after existing overlay message types:

```go
// FetchCurrentUserRequestMsg is emitted by ProfileOverlay.Init() when the store
// has no user profile loaded, triggering a fetch from the app layer.
type FetchCurrentUserRequestMsg struct{}

// UserProfileLoadedMsg is forwarded to ProfileOverlay after a fetch triggered by
// FetchCurrentUserRequestMsg completes. Err is nil on success; the overlay reads
// the profile from the store.
type UserProfileLoadedMsg struct {
    Err error
}
```

Update `docs/system/tui.md` in the same commit (CLAUDE.md rule 17).

### `internal/ui/panes/profile.go` — self-fetch + error state

Add `err error` field to `ProfileOverlay`:

```go
type ProfileOverlay struct {
    store  state.StateReader
    theme  theme.Theme
    width  int
    height int
    err    error // set when a self-triggered fetch fails; cleared on success
}
```

Update `Init()`:

```go
func (p *ProfileOverlay) Init() tea.Cmd {
    if p.store.UserProfile().ID == "" {
        return func() tea.Msg { return FetchCurrentUserRequestMsg{} }
    }
    return nil
}
```

Update `Update()` — add before existing key handling:

```go
case UserProfileLoadedMsg:
    p.err = m.Err
    return p, nil
```

Update `View()` — add error and loading branches before existing content:

```go
if p.err != nil {
    // error state — distinct from loading
    lines = append(lines, errStyle.Render("Profile unavailable"))
    lines = append(lines, hintStyle.Render("Check your connection."))
} else if profile.ID == "" {
    lines = append(lines, loadingStyle.Render("Loading profile..."))
} else {
    // ... existing content unchanged ...
}
```

### `internal/app/routing.go` — wire FetchCurrentUserRequestMsg + forward to overlay

Add handler for `panes.FetchCurrentUserRequestMsg`:

```go
case panes.FetchCurrentUserRequestMsg:
    return a, a.buildFetchCurrentUserCmd(), true
```

Update the existing `userProfileLoadedMsg` handler to forward to the overlay when open:

```go
case userProfileLoadedMsg:
    var overlayCmd tea.Cmd
    if a.profileOverlayOpen && a.profilePane != nil {
        updated, cmd := a.profilePane.Update(panes.UserProfileLoadedMsg{Err: m.err})
        if pu, ok := updated.(*panes.ProfileOverlay); ok {
            a.profilePane = pu
        }
        overlayCmd = cmd
    }
    // ... existing store write and toast logic unchanged, wrapping with tea.Batch(overlayCmd, ...) ...
```

`tea.Batch` ignores nil commands, so `overlayCmd == nil` when the overlay is closed is safe.

### `internal/ui/panes/devices.go` — distinct error vs empty state

Add `err error` field to `DeviceOverlay`.

Update `Update()` for `DevicesLoadedMsg`:

```go
case DevicesLoadedMsg:
    if m.Err == nil {
        d.err = nil
        d.devices = m.Devices
        // ... existing cursor clamp ...
    } else {
        d.err = m.Err  // preserve last known device list
    }
    return d, nil
```

Update `View()` — add error state before the `len(d.devices) == 0` check:

```go
if d.err != nil {
    // error state — distinct from legitimate empty-devices state
    return d.renderEmptyChrome("Failed to load devices", "Check your connection.")
}
if len(d.devices) == 0 {
    return d.renderEmptyChrome("No devices found", "Open Spotify on another device to see it here.")
}
```

Extract the existing empty-state rendering into a private `renderEmptyChrome(text, hint string)` helper to avoid duplication.

## Acceptance Criteria

- [ ] `FetchCurrentUserRequestMsg` and `UserProfileLoadedMsg` defined in `messages.go`; `docs/system/tui.md` updated in the same commit
- [ ] `ProfileOverlay.Init()` returns `FetchCurrentUserRequestMsg` when `store.UserProfile().ID == ""`; returns `nil` when profile already loaded
- [ ] `ProfileOverlay.Update()` handles `UserProfileLoadedMsg` — stores the error
- [ ] `ProfileOverlay.View()` shows `"Profile unavailable"` + `"Check your connection."` when `err != nil`
- [ ] `ProfileOverlay.View()` never shows `"Loading profile..."` when `err != nil`
- [ ] `routing.go` handles `panes.FetchCurrentUserRequestMsg` by dispatching `buildFetchCurrentUserCmd`
- [ ] `userProfileLoadedMsg` handler forwards `panes.UserProfileLoadedMsg` to overlay when open
- [ ] `DeviceOverlay.View()` shows `"Failed to load devices"` + `"Check your connection."` when fetch failed
- [ ] `DeviceOverlay.View()` shows `"No devices found"` + open-Spotify hint for legitimate empty state
- [ ] Error and empty states are visually distinct (different text)
- [ ] `make ci` passes

## Tasks

- [ ] Write failing tests `TestFetchCurrentUserRequestMsg_Exists`,
      `TestUserProfileLoadedMsg_Exists` in `internal/ui/panes/profile_test.go`
      - test: `go test ./internal/ui/panes/ -run "TestFetchCurrentUser|TestUserProfileLoaded" -v` → compile error

- [ ] Add `FetchCurrentUserRequestMsg` and `UserProfileLoadedMsg` to
      `internal/ui/panes/messages.go`; update `docs/system/tui.md`
      - test: `go test ./internal/ui/panes/ -run "TestFetchCurrentUser|TestUserProfileLoaded" -v` → PASS

- [ ] Write failing tests in `internal/ui/panes/profile_test.go`:
      `TestProfileOverlay_Init_EmitsFetchWhenStoreEmpty`,
      `TestProfileOverlay_Init_NilWhenProfilePresent`,
      `TestProfileOverlay_Update_StoresError`,
      `TestProfileOverlay_View_ErrorState`,
      `TestProfileOverlay_View_NoInfiniteLoading`
      - test: `go test ./internal/ui/panes/ -run "TestProfileOverlay_Init|_Update_Stores|_View_Error|_View_NoInfinite" -v` → FAIL

- [ ] Add `err error` field to `ProfileOverlay`; update `Init()`, `Update()`, `View()`
      per the Design section; add exported `Err() error` accessor for test helpers
      - test: above tests → PASS

- [ ] Write failing tests in `internal/app/routing_test.go`:
      `TestApp_FetchCurrentUserRequestMsg_Dispatches`,
      `TestApp_UserProfileLoaded_ForwardsToOverlayWhenOpen`
      - test: `go test ./internal/app/ -run "TestApp_FetchCurrentUserRequest|TestApp_UserProfileLoaded_Forwards" -v` → FAIL

- [ ] Add `FetchCurrentUserRequestMsg` handler and update `userProfileLoadedMsg` handler
      in `internal/app/routing.go`; add `OpenProfileOverlay`, `InjectUserProfileLoadedErr`,
      `ProfilePaneErr` helpers to `export_test.go`
      - test: above tests → PASS

- [ ] Write failing tests in `internal/ui/panes/devices_test.go`:
      `TestDeviceOverlay_View_ErrorState_Distinct`,
      `TestDeviceOverlay_View_EmptyState_Distinct`,
      `TestDeviceOverlay_View_ErrorClearedOnSuccess`
      - test: `go test ./internal/ui/panes/ -run "TestDeviceOverlay_View_Error|_Empty|_ErrorCleared" -v` → FAIL

- [ ] Add `err error` field to `DeviceOverlay`; extract `renderEmptyChrome`; update
      `Update()` and `View()` in `internal/ui/panes/devices.go`
      - test: above tests → PASS

- [ ] `make ci` passes
