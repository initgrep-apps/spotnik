---
title: "Profile UI — Overlay Pane, Header Chip, App Wiring"
feature: 23-user-profile-subscription
status: open
---

## Background

After Story 114 the full user profile lives in the Store. This story surfaces it in the UI:
a `u`-triggered floating overlay and a profile chip in the header right side. The overlay
follows the same pattern as `DeviceOverlay` — composited via `bubbletea-overlay`, key routing
intercepted while open.

**Depends on:** Story 114 (store must have `UserProfile()` and `IsPremium()`)

## Design

### Overlay layout

```
╭─ Profile ──────────────────────╮
│                                │
│   Irshad Sheikh                │
│   ────────────────────────     │
│   ♛  Premium                   │
│   ◎  Germany (DE)              │
│                                │
│   esc  close                   │
╰────────────────────────────────╯
```

When profile not yet loaded: render `"Loading profile..."` inside the same border.

### `internal/ui/panes/messages.go`

Add after `DeviceOverlayClosedMsg`:

```go
// ProfileOverlayClosedMsg is emitted by ProfileOverlay when the user presses Esc.
type ProfileOverlayClosedMsg struct{}
```

### `internal/ui/panes/profile.go` — new file

```go
// ProfileOverlay renders the authenticated user's profile as a floating overlay.
// Reads directly from the Store — no local state beyond width/height.
// Triggered by the 'u' key; closed by Esc.
type ProfileOverlay struct {
    store  state.StateReader
    theme  theme.Theme
    width  int
    height int
}
```

Constructor: `NewProfileOverlay(store state.StateReader, t theme.Theme) *ProfileOverlay`

`Init()` returns `nil` — data is already in the store before the user can press `u`.

`Update()` handles only `tea.KeyEsc` → emits `ProfileOverlayClosedMsg{}`. All other messages
are ignored.

`View()` is pure — reads `store.UserProfile()` and `store.IsPremium()`:
- Empty profile (`ID == ""`): render `"Loading profile..."` placeholder
- Name: `TextPrimary()` + Bold, truncated to 20 runes
- Separator: `"────"` in `TextMuted()`
- `♛  Premium` in `Info()` color; `○  Free` in `TextMuted()`
- `◎  CountryCode` — icon in `TextMuted()`, code in `TextFg()`
- `esc  close` hint in `TextMuted()`
- Border: `RoundedBorder()` with `BorderFocused()` colour (same as device overlay)

`SetSize(width, height int)` called by App on resize.

**Note:** Use `state.StateReader` (not `*state.Store`) for the store field — same pattern as
all other panes after Story 110.

### `internal/app/app.go`

Add to `App` struct after `deviceOverlayOpen`:

```go
profileOverlayOpen bool
profilePane        *panes.ProfileOverlay
```

In `New()`, initialise alongside `devicePane`:

```go
profilePane: panes.NewProfileOverlay(store, t),
```

In the `tea.WindowSizeMsg` handler where `devicePane.SetSize` is called, add:

```go
a.profilePane.SetSize(40, 12) // fixed size — profile card is not resizable
```

### `internal/app/routing.go`

**Key handler — `u` shortcut** (add after `d` global shortcut):

```go
if m.Type == tea.KeyRunes && string(m.Runes) == "u" {
    a.profileOverlayOpen = true
    return a, nil
}
```

**Overlay guard** (add after device overlay guard):

```go
if a.profileOverlayOpen {
    updated, cmd := a.profilePane.Update(m)
    if pp, ok := updated.(*panes.ProfileOverlay); ok {
        a.profilePane = pp
    }
    return a, cmd
}
```

**`ProfileOverlayClosedMsg` handler** (add after `DeviceOverlayClosedMsg` case):

```go
case panes.ProfileOverlayClosedMsg:
    a.profileOverlayOpen = false
    return a, nil, true
```

**Mouse guard** — add `a.profileOverlayOpen` to the existing condition in `handleMouseMsg`:

```go
if a.deviceOverlayOpen || a.searchOpen || a.profileOverlayOpen {
    return nil
}
```

### `internal/app/render.go`

**`renderProfileChip()`** — returns `""` if profile not yet loaded (graceful startup); name
truncated to 20 runes; `♛` in `Info()` for Premium, `○` in `TextMuted()` for Free.

**`renderHeader()`** — update the right side to compose device chip then profile chip
(profile is rightmost):

```
right := deviceChip + a.renderProfileChip()
```

**`renderWithProfileOverlay()`** — mirrors `renderWithDeviceOverlay`; composited top-right via
`btoverlay.Composite(fg, dimmed, btoverlay.Right, btoverlay.Top, 0, 0)`.

**`buildView()`** — add profile overlay compositing after device overlay check:

```go
if a.profileOverlayOpen {
    return a.renderWithProfileOverlay(body)
}
```

## Acceptance Criteria

- [ ] `ProfileOverlayClosedMsg` added to `messages.go`
- [ ] `panes.ProfileOverlay` exists with `NewProfileOverlay`, `Init`, `Update`, `View`, `SetSize`
- [ ] `View()` renders display name, `♛ Premium` or `○ Free`, country code
- [ ] `View()` renders `"Loading profile..."` when `profile.ID == ""`
- [ ] `Esc` emits `ProfileOverlayClosedMsg`; no other key closes the overlay
- [ ] `u` key opens overlay from any page/pane; `Esc` closes it
- [ ] All keys are intercepted while overlay is open (no pass-through to panes)
- [ ] Mouse events suppressed while overlay is open
- [ ] Header right side shows `device chip + profile chip` (profile chip absent if not loaded)
- [ ] Profile chip: `♛` in `Info()` for Premium, `○` in `TextMuted()` for Free
- [ ] Profile overlay composited top-right via `bubbletea-overlay`
- [ ] `make ci` passes

## Tasks

- [ ] Add `ProfileOverlayClosedMsg` to `internal/ui/panes/messages.go`
      - test: `go build ./internal/ui/panes/...` clean
- [ ] Write failing tests in `internal/ui/panes/profile_test.go`:
      `TestProfileOverlay_View_ShowsDisplayName`, `_PremiumBadge`, `_FreeBadge`,
      `_ShowsCountry`, `_LoadingState`, `_EscEmitsClosedMsg`
      - test: `go test ./internal/ui/panes/... -run TestProfileOverlay -v` → compile error
- [ ] Create `internal/ui/panes/profile.go` with `ProfileOverlay`, `NewProfileOverlay`,
      `Init`, `Update`, `View`, `SetSize`
      - test: all `TestProfileOverlay_*` tests → PASS
- [ ] Add `profileOverlayOpen bool` and `profilePane *panes.ProfileOverlay` to `App` struct
      in `app.go`; initialise in `New()`; add `SetSize` call in window resize handler
      - test: `go build ./...` clean
- [ ] Add `u` key shortcut, overlay routing guard, `ProfileOverlayClosedMsg` handler, and
      mouse suppression to `routing.go`
      - test: `go test ./internal/app/... -v` → PASS
- [ ] Add `renderProfileChip()`, update `renderHeader()` right side, add
      `renderWithProfileOverlay()`, update `buildView()` in `render.go`
      - test: `go test ./internal/app/... -run "TestRender\|TestHeader" -v` → PASS (update any
        test that checks exact header output to include the chip format)
- [ ] `make ci` passes
