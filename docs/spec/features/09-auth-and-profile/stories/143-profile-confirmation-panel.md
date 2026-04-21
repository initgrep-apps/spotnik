---
title: "Profile Overlay — Confirmation UX: Replace !! With Warning Toast"
feature: 09-auth-and-profile
status: done
---

## Background

The profile overlay's logout/forget confirmation currently shows:

```
!! Press l again to confirm logout   (Warning colour)
!! Press f again to confirm forget   (Warning colour)
```

The `!!` prefix was added as a quick visual signal but it reads as aggressive and inconsistent
with the rest of the app's notification style. All other confirmations and warnings in Spotnik
go through the `a.alerts` toast system — this should too.

**Decision: emit a warning toast on first press + simplify inline text.** When the user first
presses `l` or `f`, the `Update()` method:
1. Sets `pendingAction` (existing logic — unchanged).
2. Returns a new `ProfileConfirmToastMsg` that the app-level handler converts into a
   `a.alerts.NewAlertCmd("warning", ...)` toast.

The inline text in `View()` is kept (it serves as a reminder while the overlay is open) but
simplified — `!!` is removed, and the line is indented cleanly.

**Depends on:** Story 140 (profile overlay with `pendingAction` and `renderActions`).

## Design

### `internal/ui/panes/messages.go`

Add one new message type:

```go
// ProfileConfirmToastMsg is emitted when the user arms a logout or forget action
// (first keypress). The app converts it into a warning alert via the notifications system.
type ProfileConfirmToastMsg struct {
    Text string // e.g. "Press l again to confirm logout"
}
```

### `internal/ui/panes/profile.go` — `Update()`

On the first `l` press, emit `ProfileConfirmToastMsg` alongside setting `pendingAction`:

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
        return ProfileConfirmToastMsg{Text: "Press f again to confirm forget"}
    }
```

### `internal/ui/panes/profile.go` — `renderActions()`

Remove the `!!` prefix from the confirmation lines. Keep warning colour and indentation:

```go
// Before
lines = append(lines, warnStyle.Render("!! Press l again to confirm logout"))
// ...
lines = append(lines, warnStyle.Render("!! Press f again to confirm forget"))

// After
lines = append(lines, warnStyle.Render("  Press l again to confirm logout"))
// ...
lines = append(lines, warnStyle.Render("  Press f again to confirm forget"))
```

### `internal/app/handlers.go`

Add handler for `ProfileConfirmToastMsg`:

```go
case panes.ProfileConfirmToastMsg:
    return a, a.alerts.NewAlertCmd("warning", m.Text)
```

Place this case alongside the other `panes.*Msg` handlers (near `ProfileLogoutMsg`,
`ProfileForgetMsg`).

### Tests — `internal/ui/panes/profile_test.go`

Add tests for the new toast emission:

```go
func TestProfileOverlay_logoutFirstPress_emitsToastMsg(t *testing.T) {
    store := state.New()
    pane := NewProfileOverlay(store, theme.NewBlack())

    _, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
    require.NotNil(t, cmd)
    msg := cmd()
    toast, ok := msg.(ProfileConfirmToastMsg)
    require.True(t, ok)
    assert.Contains(t, toast.Text, "confirm logout")
}

func TestProfileOverlay_forgetFirstPress_emitsToastMsg(t *testing.T) {
    store := state.New()
    pane := NewProfileOverlay(store, theme.NewBlack())

    _, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
    require.NotNil(t, cmd)
    msg := cmd()
    toast, ok := msg.(ProfileConfirmToastMsg)
    require.True(t, ok)
    assert.Contains(t, toast.Text, "confirm forget")
}

func TestProfileOverlay_confirmationView_noDoubleExclamation(t *testing.T) {
    store := state.New()
    pane := NewProfileOverlay(store, theme.NewBlack())

    updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
    view := updated.(*ProfileOverlay).View()
    assert.NotContains(t, view, "!!")
    assert.Contains(t, view, "Press l again to confirm")
}
```

Update existing confirmation tests to assert `cmd != nil` (they now return a toast cmd):

```go
func TestProfileOverlay_logoutFirstPress_showsConfirmation(t *testing.T) {
    store := state.New()
    pane := NewProfileOverlay(store, theme.NewBlack())

    updated, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
    model := updated.(*ProfileOverlay)

    // cmd is now non-nil (emits ProfileConfirmToastMsg) — this is expected.
    assert.NotNil(t, cmd)
    assert.Contains(t, model.View(), "Press l again to confirm")
    assert.NotContains(t, model.View(), "!!")
}
```

## Acceptance Criteria

- [ ] `ProfileConfirmToastMsg{Text string}` added to `internal/ui/panes/messages.go`
- [ ] First `l` press emits `ProfileConfirmToastMsg{Text: "Press l again to confirm logout"}`
- [ ] First `f` press emits `ProfileConfirmToastMsg{Text: "Press f again to confirm forget"}`
- [ ] App handler converts `ProfileConfirmToastMsg` to `a.alerts.NewAlertCmd("warning", m.Text)`
- [ ] Inline confirmation lines in `View()` no longer start with `!!`
- [ ] Warning colour is preserved for inline confirmation lines
- [ ] `TestProfileOverlay_logoutFirstPress_emitsToastMsg` passes
- [ ] `TestProfileOverlay_forgetFirstPress_emitsToastMsg` passes
- [ ] `TestProfileOverlay_confirmationView_noDoubleExclamation` passes
- [ ] Existing `TestProfileOverlay_logout*` / `TestProfileOverlay_forget*` tests still pass
- [ ] `make ci` passes

## Tasks

- [ ] Add `ProfileConfirmToastMsg` to `internal/ui/panes/messages.go`
      - test: `go build ./internal/ui/panes/...` → clean
- [ ] Write failing tests in `profile_test.go` for `*_emitsToastMsg` and `*_noDoubleExclamation`
      - test: `go test ./internal/ui/panes/... -run "TestProfileOverlay" -v` → FAIL (cmd is nil)
- [ ] Update `Update()` in `profile.go` to return `ProfileConfirmToastMsg` on first press
      - test: `TestProfileOverlay_logoutFirstPress_emitsToastMsg`,
        `TestProfileOverlay_forgetFirstPress_emitsToastMsg` → PASS
- [ ] Remove `!!` from `renderActions()` in `profile.go`
      - test: `TestProfileOverlay_confirmationView_noDoubleExclamation` → PASS
- [ ] Add `ProfileConfirmToastMsg` handler in `internal/app/handlers.go`
      - test: `go build ./...` → clean
- [ ] `make ci` → PASS
