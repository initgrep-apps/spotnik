---
title: "Layout System"
status: done
---

## Description
Provides the responsive grid layout engine, btop-style pane borders, reusable table/filter/visualizer components, app migration to the LayoutManager, restyled header/statusbar/overlays, and mouse scroll support -- the full visual infrastructure that makes Spotnik a multi-pane, keyboard-and-mouse-driven terminal dashboard.

Spotnik's original UI was a hardcoded 3-column layout (libraryView | playerView | queueView) assembled with lipgloss.JoinHorizontal(). There was no concept of pages, presets, pane toggling, or responsive grid reflow. The redesign replaces this with a btop-inspired responsive grid system where 10 panes are organized across 2 pages (Page A: Music, Page B: Nerd Status), Page A offers 4 presets (Full Dashboard, Listening, Library, Discovery), keys 1-8 toggle individual pane visibility, hidden panes redistribute space to visible siblings, and the grid recomputes on terminal resize and preset/toggle changes.

## Acceptance Criteria
- [ ] internal/ui/layout/ package compiles independently
- [ ] PaneID enum has 10 values (8 music + 2 nerd status)
- [ ] 4 Page A presets match DESIGN.md exactly
- [ ] RenderPaneBorder() produces btop-style borders
- [ ] Table wraps bubble-table with flex columns and per-column colors
- [ ] Filter wraps textinput with toggle and case-insensitive matching
- [ ] viewMode reduced to viewSplash | viewAuth | viewGrid
- [ ] focusedPane enum deleted, replaced by layout.Manager
- [ ] bubbletea-overlay used for overlay compositing
- [ ] Mouse wheel scrolling on any pane without changing focus
- [ ] Minimum terminal size 120x30
- [ ] make ci passes
