---
title: "Help Overlay"
status: done
---

## Overview

The `?` key is shown in the app's status bar as "help" but pressing it does nothing —
there is no handler and no overlay exists. This feature implements a centered modal
help overlay that displays all app keybindings grouped by category, and establishes
`docs/keybinding.md` as the canonical human-readable keybinding reference with a
CLAUDE.md rule to keep it in sync.

## Goals

- `?` opens a centered floating overlay showing all keybindings grouped into four
  sections: Global, Navigation, Playback, Pane Actions
- The overlay uses the same btop-style border and `bubbletea-overlay` compositing
  pattern as the existing theme and search overlays
- `Esc` closes the overlay; all other keys are consumed (modal)
- `docs/keybinding.md` is created as the canonical keybinding reference
- CLAUDE.md is updated with a rule requiring all three keybinding locations
  (`docs/keybinding.md`, `docs/DESIGN.md §17`, `helpContent` var) to stay in sync
  on every keybinding change

## Out of Scope

- Dynamic sourcing of keybindings from pane `Actions()` methods — these are
  context-sensitive and unsuitable for a stable reference display
- Scrollable content — the two-column layout fits all keybindings without scrolling
- Search within help overlay

## Acceptance Criteria

- [ ] Pressing `?` from any pane on Page A or Page B opens the help overlay centered
      on screen with the full grid dimmed behind it
- [ ] The overlay shows four sections: Global, Navigation, Playback, Pane Actions
- [ ] All keybindings from `docs/DESIGN.md §17` appear in the overlay
- [ ] Pressing `Esc` closes the overlay; the grid resumes with no state change
- [ ] Pressing `?` while another overlay (search, devices, theme) is open does nothing
- [ ] Mouse scroll is blocked while the help overlay is open
- [ ] The overlay re-renders correctly on terminal resize
- [ ] The overlay updates colors immediately when the theme is changed while open
- [ ] `docs/keybinding.md` exists and matches the overlay content
- [ ] CLAUDE.md includes rule #15 and a "Keybinding Maintenance" section

## Stories

| # | Title | Status |
|---|-------|--------|
| 108 | Help overlay implementation and keybinding documentation | open |
