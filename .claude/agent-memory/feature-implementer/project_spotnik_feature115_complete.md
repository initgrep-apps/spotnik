---
name: project_spotnik_feature115_complete
description: Story 115 (ProfileOverlay pane, header chip, app wiring): overlay pattern, StateReader extension, Bubble Tea close round-trip, deferred keybinding docs
type: project
---

## Story 115 — Profile UI (Overlay Pane, Header Chip, App Wiring)

**What was built:**
- `internal/ui/panes/profile.go` — new ProfileOverlay pane (fixed 34-char inner width, reads store directly)
- `internal/ui/panes/messages.go` — added ProfileOverlayClosedMsg
- `internal/state/reader.go` — added `UserProfile()` and `IsPremium()` to StateReader interface
- `internal/app/app.go` — profilePane field, profileOverlayOpen flag, ProfileOverlayOpen() accessor
- `internal/app/handlers.go` — SetSize(40,12) on WindowSizeMsg, ProfileOverlayClosedMsg handler, SetTheme() propagation
- `internal/app/routing.go` — profile overlay routing guard, 'u' key handler, mouse suppression
- `internal/app/render.go` — truncateProfileName(), renderProfileChip(), renderWithProfileOverlay(), header right-side update

**Key files:**
- `internal/ui/panes/profile.go` — ProfileOverlay struct, View() reads store.UserProfile()/IsPremium(), fixed inner width
- `internal/app/routing.go` — profile guard placed after device guard, before search guard (lines ~73-80)
- `internal/app/render.go` — renderProfileChip() returns "" when profile.ID=="", renderWithProfileOverlay() uses btoverlay.Composite(Right, Top)

**Patterns established:**
- ProfileOverlay exactly mirrors DeviceOverlay pattern (NewProfileOverlay, SetSize, SetTheme, Init→nil, Update→only Esc, View→reads store)
- Fixed-size overlay cards store width/height but don't use them in View() — documented with "available for future resizable variants"
- Height computation for RenderPaneBorder: `strings.Count(inner, "\n") + 3` (newlines + 1 last line + 2 borders)
- renderWithProfileOverlay uses same btoverlay.Composite(fg, dimmed, Right, Top, 0, 0) as renderWithDeviceOverlay

**Gotchas:**
- StateReader must be extended before ProfileOverlay can compile — panes accept StateReader not *Store
- Theme method names: `ActiveBorder()` (not `BorderFocused()`), `TextPrimary()` (not `TextFg()`)
- Bubble Tea overlay close test: must execute `cmd()` to get ProfileOverlayClosedMsg, then call `a.Update(msg)` — just `a.Update(Esc)` does NOT close the overlay
- 'u' keybinding docs (docs/keybinding.md, DESIGN.md §17, help_overlay.go) are INTENTIONALLY deferred to Story 116 per its spec: "Finally the `u` keybinding is added to all three required locations"
- Mouse suppression: add `|| a.profileOverlayOpen` to the condition in handleMouseMsg alongside deviceOverlayOpen/searchOpen/helpOpen

**Testing notes:**
- 11 unit tests in profile_test.go, 4 routing tests in routing_test.go, 6 render tests in render_test.go
- newTestProfileOverlay() creates *state.Store and passes it directly (Store implements StateReader)
- CI at 88.5% coverage
