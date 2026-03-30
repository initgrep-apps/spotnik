---
title: "Device Switcher"
status: done
---

## Description
Lets users view all available Spotify Connect devices and transfer playback to any of them with a single keypress, with the active device always visible in the header bar. Pressing `d` opens a floating overlay listing all available devices; the user navigates with `j`/`k` and confirms with `Enter` to transfer playback. Device data comes from two sources: the active device is extracted from `GET /me/player` (polled every second), while the full device list is fetched on-demand via `GET /me/player/devices` only when the overlay opens.

## Acceptance Criteria
- [ ] Active device name visible in header bar at all times
- [ ] `d` opens device overlay with current devices loaded
- [ ] `Enter` transfers playback, header updates within 2 seconds
- [ ] `Esc` closes overlay without any change
- [ ] "No devices found" shown when Spotify returns empty list
- [ ] Device type icons render correctly
- [ ] All API and pane handlers tested
