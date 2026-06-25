---
title: "Overlays component tests (Theme, Help, Profile, Devices)"
feature: 21-test-infrastructure
status: open
---

## Background

Four global overlays provide secondary surfaces: Theme switcher (`t`), Help (`?`), Profile
(`u`), and Devices (`d`). Each opens as a modal overlay, captures all input while open, and
closes on Esc. Current unit tests verify Update() behavior but never assert the rendered
overlay output at known dimensions.

## Design

### Golden tests: `internal/ui/panes/themes_golden_test.go`

- `TestThemeOverlay_View_ThemeList` — 13 themes listed, current marked with ✓, 80×24
- `TestThemeOverlay_View_Narrow` — 40×24

### Golden tests: `internal/ui/panes/help_golden_test.go`

- `TestHelpOverlay_View_Keybindings` — all keybinding categories rendered, 80×24
- `TestHelpOverlay_View_Narrow` — 40×24

### Golden tests: `internal/ui/panes/profile_golden_test.go`

- `TestProfileOverlay_View_Premium` — premium user, display name, country, 80×24
- `TestProfileOverlay_View_Free` — free tier user with Free badge
- `TestProfileOverlay_View_Loading` — profile not yet fetched, loading state
- `TestProfileOverlay_View_LogoutConfirmation` — 'l' pressed once, confirmation view

### Golden tests: `internal/ui/panes/devices_golden_test.go`

- `TestDevicesPane_View_Devices` — 3 devices listed, active marked ✓, 80×24
- `TestDevicesPane_View_Empty` — no devices available
- `TestDevicesPane_View_Narrow` — 40×24

## Files

### Create

- `internal/ui/panes/themes_golden_test.go`
- `internal/ui/panes/help_golden_test.go`
- `internal/ui/panes/profile_golden_test.go`
- `internal/ui/panes/devices_golden_test.go`
- `internal/ui/panes/testdata/TestThemeOverlay_View_*.golden` (2 files)
- `internal/ui/panes/testdata/TestHelpOverlay_View_*.golden` (2 files)
- `internal/ui/panes/testdata/TestProfileOverlay_View_*.golden` (4 files)
- `internal/ui/panes/testdata/TestDevicesPane_View_*.golden` (3 files)

## Acceptance Criteria

- [ ] ThemeOverlay: 2 golden snapshots (list, narrow)
- [ ] HelpOverlay: 2 golden snapshots (keybindings, narrow)
- [ ] ProfileOverlay: 4 golden snapshots (premium, free, loading, logout confirmation)
- [ ] DevicesPane: 3 golden snapshots (devices, empty, narrow)
- [ ] `make ci` passes

## Tasks

- [ ] Create ThemeOverlay golden tests (2 snapshots)
      - test: `TestThemeOverlay_View_ThemeList`, `TestThemeOverlay_View_Narrow`
- [ ] Create HelpOverlay golden tests (2 snapshots)
      - test: `TestHelpOverlay_View_Keybindings`, `TestHelpOverlay_View_Narrow`
- [ ] Create ProfileOverlay golden tests (4 snapshots)
      - test: `TestProfileOverlay_View_Premium`, `TestProfileOverlay_View_Free`, `TestProfileOverlay_View_Loading`, `TestProfileOverlay_View_LogoutConfirmation`
- [ ] Create DevicesPane golden tests (3 snapshots)
      - test: `TestDevicesPane_View_Devices`, `TestDevicesPane_View_Empty`, `TestDevicesPane_View_Narrow`
- [ ] Generate golden files and verify all tests pass
