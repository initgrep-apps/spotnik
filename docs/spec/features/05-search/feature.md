---
title: "Search"
status: done
---

## Description
A fast, keyboard-native search overlay that lets users find and play tracks, artists, albums, and playlists without leaving the terminal -- press `/`, type, see live results, and act on them instantly. The search overlay layers above the existing pane layout without replacing any pane. Results appear within 400ms of the last keypress thanks to a 300ms debounce mechanism.

## Acceptance Criteria
- [ ] `/` opens search overlay from any pane without disrupting current state
- [ ] Results appear within 400ms of last keypress (300ms debounce + ~100ms API)
- [ ] `Enter` plays a track and closes the overlay
- [ ] `a` adds track to queue, shows status bar confirmation, keeps overlay open
- [ ] `Esc` closes overlay, previous pane is focused and unchanged
- [ ] Typing faster than 300ms between keys fires only one search (debounce works)
- [ ] Empty query shows hint text, no API call fired
- [ ] All search API functions and overlay handlers tested
