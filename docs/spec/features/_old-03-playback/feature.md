---
title: "Playback Controls"
status: done
---

## Description
Shows the currently playing track and gives the user full keyboard-driven control over playback, including play/pause, skip, seek, volume, shuffle, and repeat. The PlayerPane lives in the center of the grid layout and renders track metadata, a seek bar, transport controls, and a volume bar. This feature introduces the tick-based polling architecture that keeps Spotnik in sync with Spotify -- a `tea.Tick` loop fires every 1000ms, fetching `GET /me/player` and updating the central Store. Between polls, local progress interpolation increments the seek bar smoothly.

## Acceptance Criteria
- [ ] Currently playing track (name, artist, album) visible within 1 second of app launch
- [ ] `Space` play/pause responds in under 200ms (optimistic update shown immediately)
- [ ] Seek bar updates every 1 second accurately via local interpolation
- [ ] Volume changes reflect immediately in UI (optimistic), confirmed by next poll
- [ ] Shuffle/repeat state accurately reflects Spotify state after each poll
- [ ] "Nothing playing" empty state shown cleanly when Spotify returns 204
- [ ] All API functions tested with httptest mocks; all pane Update() handlers tested
- [ ] No crashes on 204 (nothing playing), 429 (rate limited), or 503 (Spotify down)
