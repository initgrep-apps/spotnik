---
title: "Profile Overlay — Confirmation UX: Replace !! With Warning Toast"
feature: 09-auth-and-profile
status: open
---

## Background

The profile overlay's logout/forget confirmation shows:

```
!! Press l again to confirm logout   (Warning colour)
```

The `!!` prefix was added as a visual signal but it reads as overly aggressive and inconsistent
with the rest of the UI. The user wants a cleaner confirmation experience. Two valid approaches
were considered:

1. **Warning toast** — emit a `NewAlertCmd(warning, "Press l again to confirm logout")` which
   renders via the existing `a.alerts` notification system on top of the full app view.
2. **Inline bordered panel** — replace the raw text line with a small rounded-border panel
   inside the profile overlay card.

**Decision: Use the warning toast system.** The profile overlay is rendered as an overlay on top
of the main view, and `a.alerts.Render()` is applied as the outermost layer in `View()`. This
means a toast emitted by the profile overlay will appear correctly above everything. Using the
existing notification system avoids changing the overlay layout and is consistent with how all
other warnings in the app are surfaced.

**Implementation change**: When the user presses `l` (first press — arming confirmation),
instead of just setting `pendingAction = profileActionLogout` and relying on `View()` to show
the inline warning text, the `Update()` method also emits a warning alert command via a new
`ProfileConfirmToastMsg` message type that the app-level handler converts into an
`a.alerts.NewAlertCmd` call.

The inline warning text in `View()` is simplified: remove the `!!` prefix. Keep the warning
colour and the confirmation text, but present it cleanly:

```
  Press l again to confirm logout    (Warning colour, no !!)
```

The `!!` is removed from both the logout and forget confirmation lines.

## Design

### `internal/ui/panes/messages.go`

Add a new message type for the confirmation toast:

```go
// ProfileConfirmToastMsg is emitted when the user arms a logout or forget action.
// The app converts it into a warning alert notification.
// Text is the confirmation prompt to display.
type ProfileConfirmToastMsg struct {
    Text string
}
```

### `internal/ui/panes/profile.go`

**Update `Update()`** — on first `l` or `f` press, emit `ProfileConfirmToastMsg` alongside
setting `pendingAction`:

```go
case 'l':
    if p.pendingAction == profileActionLogout {
        p.pendingAction = profileActionNone
        return p, func() tea.Msg { return ProfileLogoutMsg{} }
    }
    p.pendingAction = profileActionLogout
    return p, func() tea.Msg {
        return ProfileConfirmToastMsg{Text: "Press l again to confirm logout"}
    }
case 'f':
    if p.pendingAction == profileActionForget {
        p.pendingAction = profileActionNone
        return p, func() tea.Msg { return ProfileForgetMsg{} }
    }
    p.pendingAction = profileActionForget
    return p, func() tea.Msg {
        return ProfileConfirmToastMsg{Text: "Press f again to confirm — removes session + Client ID"}
    }
```

**Update `renderActions()`** — remove `!!` prefix, keep warning colour:

```go
// Logout row.
if p.pendingAction == profileActionLogout {
    lines = append(lines, warnStyle.Render("Press l again to confirm logout"))
} else {
    ...
}

// Forget row.
if p.pendingAction == profileActionForget {
    lines = append(lines, warnStyle.Render("Press f again to confirm forget"))
} else {
    ...
}
```

### `internal/app/handlers.go`

Add handler for `ProfileConfirmToastMsg`:

```go
case panes.ProfileConfirmToastMsg:
    return a, a.alerts.NewAlertCmd(components.AlertWarning, m.Text)
```

### Tests — `internal/ui/panes/profile_test.go`

Update the existing confirmation tests to verify the new behaviour:

```go
func TestProfileOverlay_logoutFirstPress_emitsToast(t *testing.T) {
    // Act: one press of "l".
    // Assert: cmd() returns ProfileConfirmToastMsg{Text: "Press l again..."}.
    // Assert: model.View() does NOT contain "!!".
}

func TestProfileOverlay_forgetFirstPress_emitsToast(t *testing.T) {
    // Act: one press of "f".
    // Assert: cmd() returns ProfileConfirmToastMsg{Text: "Press f again..."}.
}
```

Update the existing `TestProfileOverlay_logoutFirstPress_showsConfirmation` to check that
`View()` contains "Press l again" (not "!! Press l again"):

```go
assert.Contains(t, model.View(), "Press l again to confirm logout")
assert.NotContains(t, model.View(), "!!")
```

## Acceptance Criteria

- [ ] `ProfileConfirmToastMsg` type added to `messages.go`
- [ ] First `l` press emits `ProfileConfirmToastMsg{Text: "Press l again to confirm logout"}`
- [ ] First `f` press emits `ProfileConfirmToastMsg{Text: "Press f again to confirm — removes session + Client ID"}`
- [ ] `ProfileConfirmToastMsg` handler in `handlers.go` calls `a.alerts.NewAlertCmd(AlertWarning, m.Text)`
- [ ] `View()` confirmation lines no longer contain `!!`
- [ ] Warning colour is preserved on the confirmation text in `View()`
- [ ] `TestProfileOverlay_logoutFirstPress_emitsToast` passes
- [ ] `TestProfileOverlay_forgetFirstPress_emitsToast` passes
- [ ] Existing `TestProfileOverlay_logout*` / `TestProfileOverlay_forget*` tests updated and passing
- [ ] `make ci` passes

## Tasks

- [ ] Add `ProfileConfirmToastMsg` to `internal/ui/panes/messages.go`
      - test: `go build ./...` → clean
- [ ] Update `Update()` in `internal/ui/panes/profile.go` to emit `ProfileConfirmToastMsg`
      on first `l`/`f` press; update `renderActions()` to remove `!!`
      - test: write failing tests first; then: all `TestProfileOverlay_*` → PASS
- [ ] Add `ProfileConfirmToastMsg` handler to `internal/app/handlers.go`
      - test: `go build ./...` → clean
- [ ] `make ci` passes
