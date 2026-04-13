---
title: "Optimistic Playback Updates"
status: done
---

## Description

Playback control actions (volume, play/pause, shuffle, repeat) have visible UI lag of
~500ms–1s between key press and the UI reflecting the new state. The physical device
responds immediately (Spotify SDK), but the UI bar waits for a full roundtrip:

```
key press
  → 100ms gateway debounce
  → ~200-400ms HTTP PUT (set volume)
  → PlaybackCmdSentMsg received
  → ~200-400ms HTTP GET (fetch state)
  → PlaybackStateFetchedMsg received
  → store.SetPlaybackState()
  → UI renders updated bar
```

The fix is to write the predicted state to the store immediately when the key is pressed,
before the API call fires. The existing fetch-on-completion path overwrites with authoritative
state. No new types, no new store fields, no new messages.

This feature also fixes two stale entries in `docs/ARCHITECTURE.md` found during design review
and documents the new optimistic update pattern for future maintainers.

## Goals

- Volume bar, play/pause icon, shuffle indicator, and repeat indicator update on the next
  render frame after keypress — not after the API roundtrip.
- Rapid keypresses (hold `+`) stack correctly: each press reads the already-updated store
  value and increments again.
- API error path still corrects state via the existing `fetchPlaybackStateCmd` — no new
  revert logic needed.

## Acceptance Criteria

- [ ] Pressing `+`/`-` visually moves the volume bar immediately (before API response)
- [ ] Pressing `Space` flips the play/pause icon immediately
- [ ] Pressing `s` toggles the shuffle indicator immediately
- [ ] Pressing `r` cycles the repeat indicator immediately
- [ ] `ActionNext` and `ActionPrevious` are explicit no-ops in `applyOptimisticUpdate`
- [ ] Rapid hold of `+` increments the bar on every press; API debounce still fires only the last value
- [ ] On API error, state corrects within ~200–400ms (existing error path — no new revert mechanism)
- [ ] Stale `n` key removed from ARCHITECTURE.md routing table (two locations)
- [ ] Optimistic Update Pattern section added to ARCHITECTURE.md
- [ ] `make ci` passes (lint + tests + ≥ 80% coverage)

## Stories

| # | Title | Status |
|---|-------|--------|
| 124 | Optimistic playback update on key press | open |

## Files Touched (Summary)

| File | Change |
|---|---|
| `internal/app/optimistic.go` | **Create** — `applyOptimisticUpdate` method |
| `internal/app/optimistic_test.go` | **Create** — 13 table-driven test cases + nil-state guard test |
| `internal/app/handlers.go` | Add one-line call before `buildPlaybackAPICmd` in `PlaybackRequestMsg` case |
| `docs/ARCHITECTURE.md` | Remove stale `n` key (two locations); add Optimistic Update Pattern section |
