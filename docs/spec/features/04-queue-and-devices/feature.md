---
title: "Queue & Device Switching"
status: in-progress
---

## Description

Two related overlays that extend playback control. The Queue pane renders the upcoming track queue in a dense bubble-table with filter support (f key) and live tick-loop refresh. The Devices overlay lists all Spotify Connect devices the user has available and lets them transfer playback to any device with Enter. Both are accessed from Page A via keyboard shortcuts (q for queue, d for devices).

## Acceptance Criteria

- [ ] Queue pane shows upcoming tracks in dense table; refreshes on each playback poll tick
- [ ] Queue filter (f) narrows tracks by name without API calls
- [ ] Device overlay lists all available Spotify Connect devices
- [ ] Selecting a device (Enter) transfers playback within 200ms with optimistic feedback
- [ ] Both panes handle empty state (no queue, no devices) gracefully
- [ ] Open: stories 12 (queue overflow), 13 (device errors)
