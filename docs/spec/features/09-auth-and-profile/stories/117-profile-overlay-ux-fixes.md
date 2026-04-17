---
title: "Profile Overlay UX Fixes — Remove Duplicate Hint, Add Status Bar Binding"
feature: 23-user-profile-subscription
status: done
---

## Background

Two visual bugs found after Feature 23 shipped:

1. **Duplicate "esc close" hint** — the profile overlay renders `esc  close` in two places:
   the border title area (via `BorderConfig.Actions`) and again inside the pane body as an
   explicit hint line. No other overlay in the app (devices, theme switcher, help) shows this
   hint — Esc-to-close is a universal convention and the border already shows it once. The
   duplicate inside the body is redundant noise, and the one in the border title was also not
   intentional.

2. **`u profile` missing from status bar** — the bottom keyboard navigation bar shows
   `d devices` but not `u profile`, even though `u` is a global keybinding added in Story 115.
   The `appKeyMap` struct in `render.go` has no `Profile` binding, so the hint is never rendered.

## Design

### Fix 1 — Remove both "esc close" occurrences from `internal/ui/panes/profile.go`

In `View()`:

**Remove** the spacer line before the hint (currently `lines = append(lines, "")`).

**Remove** the entire hint line:
```go
// REMOVE THIS:
lines = append(lines, keyStyle.Render("esc")+" "+hintStyle.Render("close"))
```

**Remove** the `Actions` field from `BorderConfig` (set to `nil` / omit):
```go
// REMOVE THIS from BorderConfig:
Actions: []layout.Action{
    {Key: "esc", Label: "close"},
},
```

After the fix the body renders: name → separator → tier badge → country. No hint line.
The border renders: `╭─ Profile ─────╮` with no action annotation.

The `keyStyle` and `hintStyle` local variables become unused — remove them too.

### Fix 2 — Add `u profile` to status bar in `internal/app/render.go`

**Add `Profile` field** to `appKeyMap` struct:
```go
type appKeyMap struct {
    activePage                                                              layout.PageID
    Search, Page, Preset, Toggle, Pane, Devices, Profile, Theme, Help, Quit key.Binding
}
```

**Add binding** in `newAppKeyMap()` after `Devices`:
```go
Profile: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "profile")),
```

**Add `Profile` to `FullHelp()` for both pages** — place it after `Devices` in the same column
(both are overlay shortcuts):

Page A (was 5 columns × 2 rows, becomes 5 columns × 2 rows with `Profile` added to column 3):
```go
// Column 3: Pane, Devices → Pane, Devices, Profile
{k.Pane, k.Devices, k.Profile},
```

Page B (same column 2):
```go
{k.Pane, k.Devices, k.Profile},
```

**Add `Profile` to `ShortHelp()`** for both pages after `Devices`:
```go
// Page A
return []key.Binding{k.Search, k.Page, k.Preset, k.Toggle, k.Pane, k.Devices, k.Profile, k.Theme, k.Help, k.Quit}
// Page B
return []key.Binding{k.Search, k.Page, k.Pane, k.Devices, k.Profile, k.Theme, k.Help, k.Quit}
```

**Update the doc comment** at the top of `render.go` to reflect the new layout:
```
// Page A (10 bindings, 5 columns):
//   / search   p preset   Tab pane    t theme    q quit
//   0 page     1-8 toggle  d devices   u profile  ? help
//
// Page B (8 bindings, 4 columns):
//   / search   Tab pane    t theme    q quit
//   0 page     d devices   u profile  ? help
```

## Acceptance Criteria

- [ ] Profile overlay `View()` body contains no "esc" or "close" text
- [ ] Profile overlay `BorderConfig` has no `Actions` entries
- [ ] Profile overlay renders: name, separator, tier badge, country — nothing else
- [ ] Status bar contains `u • profile` (or `u profile`) on both Page A and Page B
- [ ] `d devices` and `u profile` appear adjacent in the status bar
- [ ] `make ci` passes

## Tasks

- [ ] In `internal/ui/panes/profile.go` `View()`: remove the spacer line, the `keyStyle`/`hintStyle`
      vars and hint line, and the `Actions` field from `BorderConfig`
      - test: `go test ./internal/ui/panes/... -run TestProfileOverlay -v` → PASS;
        assert `view` does not contain `"esc"` or `"close"` in any profile test
- [ ] In `internal/app/render.go`: add `Profile key.Binding` to `appKeyMap`, define it in
      `newAppKeyMap()`, add to `FullHelp()` and `ShortHelp()` for both pages, update doc comment
      - test: add `assert.Contains(t, statusBar, "u")` and `assert.Contains(t, statusBar, "profile")`
        to the relevant render test; `go test ./internal/app/... -run TestRender -v` → PASS
- [ ] `make ci` passes
