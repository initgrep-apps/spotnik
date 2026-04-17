---
title: "UI Layout & Components"
status: done
---

## Description

Grid layout manager, reusable pane borders and components, responsive design presets, and the in-app help overlay. Provides the geometric foundation every other feature builds on — LayoutManager assigns coordinates to each pane, btop-style rounded borders frame content, and the preset system switches between full dashboard, listening, library, and discovery layouts. Reusable table and filter components live here. The help overlay pane renders the full keybinding reference grouped by category.

## Acceptance Criteria

- [ ] LayoutManager assigns pane coordinates without overlap at all terminal sizes
- [ ] Btop-style rounded borders render correctly and resize cleanly
- [ ] Four preset layouts switch without pane rendering artifacts
- [ ] Reusable table component renders dense sortable content across all panes
- [ ] Help overlay opens on `?` and displays all keybindings grouped by category
- [ ] All layout calculations and components covered by unit tests
