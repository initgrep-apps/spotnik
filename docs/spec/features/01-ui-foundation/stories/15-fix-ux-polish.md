---
title: "Fix UX Polish"
feature: 12-layout
status: done
---

## Background
Status bar in all focus states doesn't mention `1`/`2`/`3` for view switching. DESIGN.md lists these keys in the help overlay but they're not in the status bar context hints. B6 (Tab order L->Q->P feels wrong) -- current order P->L->Q is kept per owner decision.

## Design
Add `2 stats` and `3 playlists` hints to main view status bar. Ensure hints appear in all focus states (library, player, queue). Keep status bar concise -- may abbreviate as `2 stats  3 lists` if space is tight.

## Acceptance Criteria
- [ ] Status bar in all focus states shows `2 stats` and `3 playlists` hints
- [ ] Tests verify view switcher hints are present

## Tasks
- [ ] Add view switcher hints to renderStatusBar() in internal/app/app.go
      - test: status bar shows `2 stats` and `3 playlists` in all focus states
