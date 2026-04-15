---
title: "Playback Control Correctness"
status: open
---

## Description

Fixes two bugs in the gateway pipeline that produce incorrect behaviour after
user-triggered playback actions.

**Bug 1 — Stale reconcile fetch.** After a playback command (volume, shuffle,
repeat, play, pause, next, previous) the gateway fires a reconcile GET to
confirm the new state. Because `RequestKey` carries no priority, this GET can
join an in-flight Background poll that was registered *before* the command was
sent. The poll carries pre-command Spotify state. The UI reverts to the old
value for up to 3 seconds.

**Bug 2 — Gateway debounce drops rapid commands.** A 100ms hold window
(`interactiveDebounce`) is keyed by API path for all Interactive requests. For
volume up/down (both share `/v1/me/player/volume`) a burst of three presses at
60ms intervals means only the last press survives — the first two are silently
dropped. The debounce was designed for search; for playback commands each press
is a semantically independent side-effect that must fire.

## Acceptance Criteria

- `GET /v1/me/player` fired from `PlaybackCmdSentMsg` success path does not
  join an in-flight Background poll for the same path.
- Background polls still deduplicate with other Background polls (existing
  behaviour preserved).
- Interactive GET requests never join any in-flight request — Background or
  Interactive.
- All playback command PUTs/POSTs fire immediately with no 100ms gateway hold.
- Search behaviour is unchanged — last query wins, stale results are dropped.
- `make ci` passes (lint + tests + 80% coverage).
