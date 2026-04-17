---
title: "Header + Status Bar + Overlay Restyle"
feature: 12-layout
status: done
---

## Background
The current header shows spotnik left-aligned and device indicator right-aligned. The status bar shows context-sensitive keybinding hints. Overlays use manual lipgloss.Place() for positioning. The new DESIGN.md specifies a btop-style header with page indicator, preset name, and global action shortcuts; a status bar with global-only shortcuts (pane hints now live in borders); search/device overlays with RenderPaneBorder() borders; bubbletea-overlay for overlay compositing; and toast notifications repositioned to bottom-right.

Design reference: docs/DESIGN.md sections 12, 13, 14, 15.

## Design

### Header
Left: `spotnik` (bold, TextPrimary) + `Page A/B` + `ᐅp preset N` + `ᐅ/ search` + `ᐅd devices`, joined with ` ─ `. Right: `◉ DeviceName` or `○ No device`. Middle filled with `─`.

### Status Bar
Fixed global hints: `/` search, `0` page, `p` preset, `1-8` toggle, `Tab` pane, `d` devices, `?` help, `q` quit. Remove mainHints(), statsHints(), playlistsHints().

### Overlays
Search: centered, btop border, bubbletea-overlay compositing. Device: top-right, btop border. Both use RenderPaneBorder() with BorderConfig.

### Toasts
Repositioned to bottom-right per DESIGN.md.

## Acceptance Criteria
- [ ] bubbletea-overlay dependency added
- [ ] Header shows: spotnik, page, preset index, global action shortcuts, device
- [ ] Status bar shows global-only shortcuts
- [ ] Search overlay uses RenderPaneBorder() with btop-style border
- [ ] Device overlay uses RenderPaneBorder() with btop-style border
- [ ] Both overlays use bubbletea-overlay for compositing
- [ ] Search overlay centered, device overlay top-right
- [ ] Toast notifications positioned bottom-right
- [ ] All ᐅ action prefixes consistent
- [ ] make ci passes

## Tasks
- [ ] Add bubbletea-overlay dependency
      - test: go build ./... succeeds
- [ ] Restyle header bar in internal/app/render.go
      - test: contains spotnik; shows page/preset/device; fits terminal width
- [ ] Restyle status bar to global-only in render.go
      - test: contains all global shortcuts; NO pane-specific hints; fits width
- [ ] Update search overlay with btop borders + bubbletea-overlay
      - test: btop border with title/actions; centered; dimmed background
- [ ] Update device overlay with btop borders + bubbletea-overlay
      - test: btop border; top-right position; active device ◉, inactive ○
- [ ] Reposition toast notifications to bottom-right
      - test: toast in bottom-right; no interference with grid; auto-dismiss 4s
