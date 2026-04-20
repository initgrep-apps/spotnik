---
title: "Profile Overlay — Logout / Forget Actions with Confirmation + Keybinding Docs"
feature: 09-auth-and-profile
status: open
---

## Background

The profile overlay currently only displays user information and closes on Esc. The onboarding
redesign adds two session management actions:

- **Logout (`l`)** — clears tokens only; client ID remains in config. On next launch: straight
  to `viewAuth`.
- **Forget (`f`)** — clears tokens *and* removes `client_id` from config. On next launch: back
  to `viewOnboarding` (stepRegister).

Both require **double-key confirmation**: first press arms the action and shows a warning; a
second press of the *same* key executes it. Pressing any other key while armed cancels the
pending action silently.

Both are new keybindings — all three required locations must be updated in the **same commit**
per `CLAUDE.md`.

**Depends on:** Stories 134 (`ClearClientID`), 137 (profile overlay exists with current structure).

## Design

### `internal/ui/panes/messages.go`

Add two new message types:

```go
// ProfileLogoutMsg is emitted when the user confirms logout from the profile overlay.
// The app clears tokens and quits.
type ProfileLogoutMsg struct{}

// ProfileForgetMsg is emitted when the user confirms forget from the profile overlay.
// The app clears tokens and client_id from config, then quits.
type ProfileForgetMsg struct{}
```

### `internal/ui/panes/profile.go`

**Add `profileAction` type and constants**:

```go
type profileAction int

const (
    profileActionNone   profileAction = iota
    profileActionLogout               // awaiting second 'l'
    profileActionForget               // awaiting second 'f'
)
```

**Add `pendingAction profileAction` field** to `ProfileOverlay` struct.

**Update `Update()`** — handle `l` and `f` with double-key confirmation:

- `Esc`: reset `pendingAction`; emit `ProfileOverlayClosedMsg{}`
- `l`:
  - if `pendingAction == profileActionLogout` → second press: reset, emit `ProfileLogoutMsg{}`
  - else → set `pendingAction = profileActionLogout`
- `f`:
  - if `pendingAction == profileActionForget` → second press: reset, emit `ProfileForgetMsg{}`
  - else → set `pendingAction = profileActionForget`
- Any other key → reset `pendingAction = profileActionNone`

**Update `View()`** — add a separator and action section below the profile info:

```
────────────────────

  l  Logout
     ends session · keeps Client ID

  f  Forget
     removes session + Client ID

  Esc  close
```

When `pendingAction == profileActionLogout`, replace the action section with:

```
!! Press l again to confirm logout   (Warning() colour)
   f  Forget
      removes session + Client ID
```

When `pendingAction == profileActionForget`:

```
   l  Logout
      ends session · keeps Client ID
!! Press f again to confirm forget   (Warning() colour)
```

`!!` warning lines use `theme.Warning()` colour. Action labels use `theme.TextPrimary()`. Sub-labels
use `theme.TextMuted()`.

### `internal/app/handlers.go`

Add handlers for the two new message types:

```go
case panes.ProfileLogoutMsg:
    _ = a.tokenStore.Delete()
    return a, tea.Quit

case panes.ProfileForgetMsg:
    _ = a.tokenStore.Delete()
    _ = config.ClearClientID(config.DefaultConfigPath())
    return a, tea.Quit
```

Add import `"github.com/initgrep-apps/spotnik/internal/config"` if not already present.

### Keybinding documentation — update all three locations in one commit

**`docs/keybinding.md`** — add under the Profile Overlay section:

```markdown
| l | Profile overlay | Logout — ends session, keeps Client ID. Press twice to confirm. |
| f | Profile overlay | Forget — removes session + Client ID. Press twice to confirm. |
```

**`docs/DESIGN.md §17`** — add the same two rows to the keybinding table under the profile
overlay section.

**`internal/ui/panes/help_overlay.go` `helpContent` var** — add entries in the profile section:

```
l  logout        end session (keeps Client ID) — press twice
f  forget        remove session + Client ID — press twice
```

### Tests — `internal/ui/panes/profile_test.go`

Write **failing** tests first (TDD):

```go
func TestProfileOverlay_logoutFirstPress_showsConfirmation(t *testing.T) {
    // Act: pane.Update(keyMsg("l")).
    // Assert: cmd == nil; model.View() contains "Press l again to confirm".
}

func TestProfileOverlay_logoutSecondPress_emitsLogoutMsg(t *testing.T) {
    // Act: two presses of "l".
    // Assert: second cmd() returns ProfileLogoutMsg{}.
}

func TestProfileOverlay_forgetFirstPress_showsConfirmation(t *testing.T) {
    // Act: pane.Update(keyMsg("f")).
    // Assert: model.View() contains "Press f again to confirm".
}

func TestProfileOverlay_forgetSecondPress_emitsForgetMsg(t *testing.T) {
    // Act: two presses of "f".
    // Assert: second cmd() returns ProfileForgetMsg{}.
}

func TestProfileOverlay_differentKeyAfterFirstPress_cancelsAndArmsNew(t *testing.T) {
    // Act: press "l" then "f".
    // Assert: after second press, View() shows forget confirmation (not logout); cmd == nil.
}
```

## Acceptance Criteria

- [ ] `ProfileLogoutMsg` and `ProfileForgetMsg` added to `messages.go`
- [ ] `ProfileOverlay.pendingAction` tracks confirmation state
- [ ] First `l` press: `pendingAction = profileActionLogout`; `View()` shows logout warning
- [ ] Second `l` press: emits `ProfileLogoutMsg`; `pendingAction` reset
- [ ] First `f` press: `pendingAction = profileActionForget`; `View()` shows forget warning
- [ ] Second `f` press: emits `ProfileForgetMsg`; `pendingAction` reset
- [ ] Any key other than the confirming key resets `pendingAction` and arms for the new key
- [ ] `ProfileLogoutMsg` handler: clears tokens, quits
- [ ] `ProfileForgetMsg` handler: clears tokens, calls `config.ClearClientID`, quits
- [ ] `docs/keybinding.md`, `docs/DESIGN.md §17`, and `help_overlay.go` `helpContent` all
      updated with `l` and `f` bindings in the **same commit**
- [ ] All 5 `TestProfileOverlay_*` tests pass; `make ci` passes

## Tasks

- [ ] Write failing tests in `internal/ui/panes/profile_test.go` for
      `TestProfileOverlay_logout*` and `TestProfileOverlay_forget*` and
      `TestProfileOverlay_differentKeyAfterFirstPress_cancelsAndArmsNew`
      - test: `go test ./internal/ui/panes/... -run "TestProfileOverlay_logout|TestProfileOverlay_forget|TestProfileOverlay_different" -v`
        → compile errors (`pendingAction` undefined)
- [ ] Add `ProfileLogoutMsg` and `ProfileForgetMsg` to `internal/ui/panes/messages.go`
      - test: `go build ./internal/ui/panes/...` → clean
- [ ] Add `profileAction` type + constants, `pendingAction` field, updated `Update()`, updated
      `View()` to `internal/ui/panes/profile.go`
      - test: all 5 `TestProfileOverlay_*` tests → PASS
- [ ] Add `ProfileLogoutMsg` and `ProfileForgetMsg` handlers in `internal/app/handlers.go`
      - test: `go build ./...` → clean
- [ ] Update `docs/keybinding.md`, `docs/DESIGN.md §17`, and
      `internal/ui/panes/help_overlay.go` `helpContent` with `l` and `f` bindings
      - test: `grep -r "logout\|forget" docs/keybinding.md docs/DESIGN.md internal/ui/panes/help_overlay.go` → matches in all three
- [ ] Commit all keybinding doc changes **together** with the profile overlay changes in a
      single commit
      - test: `make ci` passes
