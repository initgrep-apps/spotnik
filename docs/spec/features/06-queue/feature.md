---
title: "Queue Management"
status: done
---

## Description
Displays the upcoming playback queue in a dense table pane, supports add-to-queue from library/search, automatic refresh via tick polling, and in-pane filtering. The queue pane reads all data from `state.Store` and never calls the API directly. Queue additions are routed through the app-level command pattern, and all errors surface via toast notifications. The Spotify Web API does not support queue item removal, so no remove functionality is exposed.

## Acceptance Criteria
- [ ] Queue visible on startup within 2 seconds
- [ ] Queue updates within 1 second when track changes (via polling)
- [ ] `a` from library or search adds track and shows status bar confirmation
- [ ] Empty queue shows "Queue is empty" centered message
- [ ] `QueuePane` satisfies `layout.Pane` interface
- [ ] Dense table format with 4 columns: #, Track, Artist, Duration
- [ ] `f` key toggles in-pane filter
- [ ] No remove functionality exposed (Spotify API limitation)
- [ ] All pane `Update()` handlers tested
