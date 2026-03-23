# Feature 11 — Fix Playback UX

> **Bug fix:** Multiple playback UX issues — volume errors, empty state centering,
> emoji icons, silent playback failures.

## Bugs Addressed

| # | Issue | Root Cause |
|---|---|---|
| B2 | Volume 403 error shown raw | Some devices disallow volume control; raw JSON shown |
| B3 | Volume up doesn't work / 5% too coarse | Device limitation + hardcoded step size |
| B4 | Space play/pause fails silently | Device doesn't support web API control; no feedback |
| B5 | Shuffle/repeat fail silently | Same as B4 |
| B7 | "Nothing playing" not centered | Hardcoded spaces in `renderEmpty()` |
| B15 | Player emoji icons (⏮ ⏸ ⏭ 🔀 🔁) look weird | Unicode emoji, not TUI-friendly |

---

## Root Cause Analysis

### B2/B3 — Volume Control
`player.go` uses hardcoded `vol+5` / `vol-5`. Some Spotify Connect devices (e.g., iPhone)
disallow volume control via web API, returning a 403 with `VOLUME_CONTROL_DISALLOW` reason.
The raw JSON error is shown in the status bar — not user-friendly.

### B4/B5 — Playback Commands Fail Silently
Code is correct, but some Spotify Connect devices don't support web API playback control.
Commands fail silently (no error returned, just no effect). If a playback command returns
an error, the user sees nothing.

### B7 — Empty State Not Centered
`renderEmpty()` uses hardcoded spaces (`"        Nothing playing"`). Should use
`lipgloss.Place()` or dynamic centering based on `p.width`.

### B15 — Emoji Icons
`player.go` uses Unicode emoji (⏮ ⏸ ⏭ 🔀 🔁) which render inconsistently across terminals.
Per DESIGN.md note: "If emoji cause rendering issues, fall back to ASCII: `|<`, `||`, `>|`, `>?`, `>>`."

---

## Fix

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

---

## Files

- `internal/ui/panes/player.go` — Icon replacement, empty state centering, error surfacing
- `internal/app/app.go` — Playback command error handling in Update()
- `internal/config/config.go` — Add `VolumeStep` field
- Tests for all modified functions

---

## Acceptance Criteria

- [ ] Empty state "Nothing playing" is centered in the pane
- [ ] Player icons are ASCII/text, not Unicode emoji
- [ ] Volume step is configurable in config.toml (default 5%)
- [ ] VOLUME_CONTROL_DISALLOW shows user-friendly message in status bar
- [ ] Failed playback commands show error in status bar
- [ ] All modified functions have tests
