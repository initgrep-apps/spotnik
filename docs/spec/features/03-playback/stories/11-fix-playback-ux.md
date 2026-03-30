---
title: "Fix Playback UX"
feature: 03-playback
status: open
---

## Background
Multiple playback UX issues were identified: volume errors shown raw, empty state not centered, emoji icons rendering poorly in terminals, and silent playback failures.

**Bugs addressed:**
- B2: Volume 403 error shown raw (some devices disallow volume control; raw JSON shown)
- B3: Volume up doesn't work / 5% too coarse (device limitation + hardcoded step size)
- B4: Space play/pause fails silently (device doesn't support web API control; no feedback)
- B5: Shuffle/repeat fail silently (same as B4)
- B7: "Nothing playing" not centered (hardcoded spaces in `renderEmpty()`)
- B15: Player emoji icons look weird (Unicode emoji, not TUI-friendly)

## Design

1. **Replace emoji with terminal-friendly symbols or text labels**
   - `|<` (prev), `||` / `>` (pause/play), `>|` (next), `~` (shuffle), `=>` (repeat)
   - Or use text: `Prev  Pause  Next  Shuf  Rep`

2. **Center "Nothing playing" message**
   - Use `lipgloss.Place()` with `p.width` and `p.height` for proper centering

3. **Add `volume_step` to config**
   - New field `VolumeStep int` in `UIConfig` with default `5`
   - Read from `config.toml`: `volume_step = 5`

4. **Catch `VOLUME_CONTROL_DISALLOW`**
   - Parse 403 response body for restriction reason
   - Show "Volume control not supported on this device" in status bar

5. **Surface playback command errors**
   - If play/pause/shuffle/repeat returns an error, show it in status bar
   - Message: "Playback control not available on this device"

### Files
- `internal/ui/panes/player.go` -- Icon replacement, empty state centering, error surfacing
- `internal/app/app.go` -- Playback command error handling in Update()
- `internal/config/config.go` -- Add `VolumeStep` field
- Tests for all modified functions

## Acceptance Criteria
- [ ] Empty state "Nothing playing" is centered in the pane
- [ ] Player icons are ASCII/text, not Unicode emoji
- [ ] Volume step is configurable in config.toml (default 5%)
- [ ] VOLUME_CONTROL_DISALLOW shows user-friendly message in status bar
- [ ] Failed playback commands show error in status bar
- [ ] All modified functions have tests

## Tasks
- [ ] Replace emoji icons with terminal-friendly ASCII symbols
      - test: Controls render correct ASCII symbols for all states
- [ ] Center "Nothing playing" using `lipgloss.Place()` with dynamic width/height
      - test: Empty state is centered within pane dimensions
- [ ] Add `VolumeStep` config field with default 5
      - test: Config loading parses volume_step; default is 5 when not specified
- [ ] Catch `VOLUME_CONTROL_DISALLOW` 403 and show user-friendly status bar message
      - test: 403 with restriction reason shows friendly message; other 403s show generic error
- [ ] Surface playback command errors in status bar
      - test: Failed play/pause/shuffle/repeat shows error in status bar
