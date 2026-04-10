# Playback Controls Improvement — Design Spec

> **Status:** Draft  
> **Date:** 2026-04-11  
> **Scope:** Optimistic UI for playback icons, volume UX (step + animation), device capability
> gating, repeat-one glyph fix

---

## 1. Background

Five issues were identified through user observation and deep codebase + Spotify API research:

1. **Icon lag** — play/pause, shuffle, repeat icons take 300–1500ms to reflect a keypress because
   the UI waits for two full HTTP round-trips (command + re-poll) before updating.
2. **Volume step too large** — `volumeStep` is hardcoded to 5%. 20 keypresses to go 0→100%.
3. **Volume on phone shows wrong error** — `PUT /me/player/volume` fails with 403 on devices
   where `supports_volume: false`. The app shows "Spotify Premium required" (wrong message,
   wrong cause).
4. **Repeat-one glyph awkward** — `↻1` renders the Unicode arrow at full glyph height next to
   an ASCII `1` at alphanumeric height, creating a visible optical mismatch.
5. **No device capability awareness** — `supports_volume`, `is_restricted`, and
   `actions.disallows` are all returned by the Spotify API but none are parsed or acted on.

---

## 2. Root Cause Analysis

### 2.1 Icon lag

`NowPlayingPane.View()` reads `ps.IsPlaying`, `ps.ShuffleState`, `ps.RepeatState` directly from
the Store. The Store only updates after `PlaybackStateFetchedMsg` arrives, which requires two
sequential async operations:

```
keypress
  → PlaybackRequestMsg
  → buildPlaybackAPICmd (100ms gateway debounce + network)
  → PlaybackCmdSentMsg
  → fetchPlaybackStateCmd (second network call)
  → PlaybackStateFetchedMsg
  → store.SetPlaybackState()
  → View() re-reads → icon updates
```

The 100ms interactive debounce at the gateway is intentional and correct: for volume, it batches
rapid keypresses into one API call (last-wins). For play/pause toggle, it means if the user
presses play/pause/play within 100ms, only the final state (play) reaches Spotify — the
intermediate pause is never sent. This is acceptable because the debounce window is imperceptible.

The lag is not the debounce. The lag is the absence of optimistic local state.

### 2.2 Volume accumulation bug

`buildPlaybackAPICmd` snapshots `ps.Device.VolumePercent` from the store at dispatch time.
With step=5 (or step=1), five rapid `+` presses all read the same snapshot value (e.g. 50) and
each computes `50+1=51`. The pane-local pending volume must accumulate the delta independently
of the store.

### 2.3 Device capability gaps

Three Spotify API fields are returned in existing responses but not parsed:

| Field | API response | `domain` type | Used? |
|---|---|---|---|
| `device.supports_volume` | `GET /me/player`, `GET /me/player/devices` | not in `domain.Device` | no |
| `device.is_restricted` | same | parsed but never checked | no |
| `actions.disallows.*` | `GET /me/player` | not in `domain.PlaybackState` | no |

`actions.disallows` is Spotify's primary runtime signal: it reflects subscription tier, content
type (ads, radio, DRM), and device capabilities in a single object. It is the authoritative
pre-flight check recommended by Spotify's own design guidelines.

---

## 3. Design

### 3.1 Domain additions (`internal/domain/types.go`)

**Add `SupportsVolume` to `Device`:**

```go
type Device struct {
    ID               string `json:"id"`
    IsActive         bool   `json:"is_active"`
    IsPrivateSession bool   `json:"is_private_session"`
    IsRestricted     bool   `json:"is_restricted"`
    Name             string `json:"name"`
    Type             string `json:"type"`
    VolumePercent    int    `json:"volume_percent"`
    SupportsVolume   bool   `json:"supports_volume"` // new
}
```

**Add `PlaybackActions` and wire into `PlaybackState`:**

```go
type PlaybackActions struct {
    Pausing               bool `json:"pausing"`
    Resuming              bool `json:"resuming"`
    Seeking               bool `json:"seeking"`
    SkippingNext          bool `json:"skipping_next"`
    SkippingPrev          bool `json:"skipping_prev"`
    TogglingRepeatContext bool `json:"toggling_repeat_context"`
    TogglingRepeatTrack   bool `json:"toggling_repeat_track"`
    TogglingShuffle       bool `json:"toggling_shuffle"`
    TransferringPlayback  bool `json:"transferring_playback"`
}

type PlaybackActionsWrapper struct {
    Disallows PlaybackActions `json:"disallows"`
}

// PlaybackState — add:
Actions PlaybackActionsWrapper `json:"actions"`
```

`encoding/json` picks up all new fields automatically. No changes to `api/player.go` or
`api/devices.go`.

**Extend `DeviceInfo` (pane message type) to carry capability fields:**

```go
// panes/messages.go
type DeviceInfo struct {
    ID             string
    Name           string
    Type           string
    IsActive       bool
    IsRestricted   bool // new
    SupportsVolume bool // new
}
```

The conversion in `buildFetchDevicesCmd` (`commands.go`) and the reverse conversion in the
`DevicesLoadedMsg` handler must map these fields through. Currently both conversions discard
them.

### 3.2 Store methods (`internal/state/store.go`)

**`ActionAllowed(action PlaybackAction) (bool, string)`**

Single decision point for all capability checks. Returns `(true, "")` when the action is
permitted, `(false, reason)` when blocked.

```
is_restricted   → "Device not controllable via API"         (checked first, all actions)
supports_volume → "Volume not available on this device"     (ActionVolumeUp/Down)
disallows.*     → context-specific reason string            (per action)
```

Mapping of actions to disallow fields:

| Action | Disallow field | Reason string |
|---|---|---|
| ActionVolumeUp/Down | `device.SupportsVolume == false` | "Volume not available on this device" |
| ActionNext | `SkippingNext` | "Skip not available in this context" |
| ActionPrevious | `SkippingPrev` | "Skip not available in this context" |
| ActionShuffle | `TogglingShuffle` | "Shuffle not available in this context" |
| ActionRepeat | `TogglingRepeatContext && TogglingRepeatTrack` | "Repeat not available in this context" |
| ActionSeek | `Seeking` | "Seek not available in this context" |
| ActionTransferPlayback | `TransferringPlayback` | "Playback transfer not available" |

Returns `(true, "")` when `PlaybackState` is nil — no state yet, let the API respond.

**`IsTargetDeviceRestricted(deviceID string) bool`**

Looks up a device by ID in the cached devices list. Returns `true` if `IsRestricted` is set on
that specific device. Used by the `TransferPlaybackMsg` handler to block transfers to restricted
target devices.

Requires `IsRestricted` to flow through `DeviceInfo` (see §3.1).

### 3.3 Routing gate (`internal/app/routing.go`)

Two ordered checks in `handleKeyMsg` before forwarding to `NowPlayingPane`:

```
1. Premium gate (existing)  — !store.IsPremium()              → "Spotify Premium required"
2. Capability gate (new)    — !store.ActionAllowed(action)    → reason from store
```

`DeviceIsRestricted` is folded into `ActionAllowed` (early return before the switch), so no
third check is needed at the routing layer.

### 3.4 `TransferPlaybackMsg` handler (`internal/app/handlers.go`)

Three-layer gate, consistent with keypress routing:

```
1. !store.IsPremium()                          → "Spotify Premium required"
2. !store.ActionAllowed(ActionTransferPlayback) → reason from store
3. store.IsTargetDeviceRestricted(m.DeviceID)  → "Device not controllable via API"
```

If all pass, dispatch `buildTransferPlaybackCmd` as today.

### 3.5 `PlaybackCmdSentMsg` handler — fix 403 message

Replace hardcoded `"Spotify Premium required"` with the actual Spotify error body:

```go
var forbiddenErr *api.ForbiddenError
if errors.As(m.Err, &forbiddenErr) {
    msg := forbiddenErr.Message
    if msg == "" {
        msg = "Spotify Premium required"
    }
    return a, tea.Batch(fetchPlaybackStateCmd(a.player), a.alerts.NewAlertCmd("warning", msg))
}
```

The proactive gate (§3.3) prevents most 403s from reaching this handler. Those that do arrive
(race between poll and keypress) will surface Spotify's actual message.

### 3.6 Optimistic UI in `NowPlayingPane` (`internal/ui/panes/nowplaying.go`)

**New pane-local fields:**

```go
pendingIsPlaying  *bool   // nil = no pending state
pendingShuffleOn  *bool
pendingRepeatMode *string // "off" | "context" | "track"
pendingVolume     *int    // accumulated target (1% steps, clamped 0–100)
```

**`handleKey` — set pending state on keypress:**

Each playback action sets its corresponding pending field before returning the command. For
volume, the pending value accumulates via `resolvedVolume()`:

```go
func (p *NowPlayingPane) resolvedVolume(ps *domain.PlaybackState) int {
    if p.pendingVolume != nil {
        return *p.pendingVolume
    }
    if ps != nil && ps.Device != nil {
        return ps.Device.VolumePercent
    }
    return 65 // default when no device info
}
```

Five rapid `+` presses at store volume 50 correctly accumulate to 55 (not 51×5).

**Volume target passed in message:**

```go
// panes/messages.go
type PlaybackRequestMsg struct {
    Action       PlaybackAction
    TargetVolume int  // new: set for ActionVolumeUp/Down, ignored otherwise
}
```

`buildPlaybackAPICmd` reads `m.TargetVolume` for volume actions instead of snapshotting the
store. Removes the double-snapshot race.

**`handlePlaybackFetched` — clear all pending state:**

```go
p.pendingIsPlaying = nil
p.pendingShuffleOn = nil
p.pendingRepeatMode = nil
p.pendingVolume = nil
```

Server truth always wins when the poll arrives.

**`View()` — resolve display values:**

```go
isPlaying  := resolveOptional(p.pendingIsPlaying,  ps.IsPlaying)
shuffleOn  := resolveOptional(p.pendingShuffleOn,  ps.ShuffleState)
repeatMode := resolveOptional(p.pendingRepeatMode, ps.RepeatState)
vol := ps.Device.VolumePercent
if p.pendingVolume != nil { vol = *p.pendingVolume }
```

`resolveOptional` is a small generic helper: returns the dereferenced pointer if non-nil, else
the fallback value. Volume bar updates immediately on keypress — `pendingVolume` is set in
`handleKey` before returning the command, so the next render cycle reflects the new value
without waiting for the poll.

### 3.7 Volume step size

Change `volumeStep: 5` → `volumeStep: 1` in `internal/app/app.go:300`. Single line.

No config key needed. 1% is the correct default — 100 steps to traverse full range, consistent
with how Spotify's own mobile app behaves when using hardware volume keys.

### 3.8 Controls component — three-state rendering

**`NewControls` signature extended:**

```go
func NewControls(
    isPlaying  bool,
    shuffleOn  bool,
    repeatMode string,
    disallows  domain.PlaybackActions,
    supportsVolume bool,
    theme      theme.Theme,
) Controls
```

**Three style states:**

```go
activeStyle   = lipgloss.NewStyle().Foreground(theme.PlayingIndicator())
inactiveStyle = lipgloss.NewStyle().Foreground(theme.TextSecondary())
disabledStyle = lipgloss.NewStyle().Foreground(theme.TextMuted())
```

`TextMuted()` already exists on the `Theme` interface. No new token required.

**Per-icon state resolution:**

| Icon | Active | Inactive | Disabled condition |
|---|---|---|---|
| `⇄` shuffle | `PlayingIndicator` | `TextSecondary` | `disallows.TogglingShuffle` |
| `▷`/`⏸` play-pause | `PlayingIndicator` | — | `disallows.Pausing` (if playing) or `disallows.Resuming` (if paused) |
| `↻`/`↻¹` repeat | `PlayingIndicator` | `TextSecondary` | both repeat disallows true |
| Volume bar | gradient tier | — | `!supportsVolume` |

**Repeat-one glyph fix:**

```go
// before
activeStyle.Render("↻1")

// after
activeStyle.Render("↻¹")   // U+21BB + U+00B9 (superscript one)
```

Consistent with project's existing use of superscript digits (`¹`–`⁸`) for pane toggle keys.

### 3.9 DeviceOverlay — visual capability indicators

`DeviceOverlay.View()` extended to render capability state per device row:

- `IsRestricted: true` → name in `TextMuted()` + `[restricted]` suffix
- `SupportsVolume: false` → muted speaker symbol appended to name (e.g. `iPhone 󰖁` or `(no vol)`)

No architecture changes — purely visual, reads from `DeviceInfo` fields added in §3.1.

---

## 4. Files Changed

| File | Change |
|---|---|
| `internal/domain/types.go` | Add `SupportsVolume` to `Device`; add `PlaybackActions`, `PlaybackActionsWrapper`; add `Actions` field to `PlaybackState` |
| `internal/ui/panes/messages.go` | Add `IsRestricted`, `SupportsVolume` to `DeviceInfo`; add `TargetVolume` to `PlaybackRequestMsg` |
| `internal/state/store.go` | Add `ActionAllowed()`, `IsTargetDeviceRestricted()` |
| `internal/app/routing.go` | Add capability gate after Premium gate |
| `internal/app/handlers.go` | Fix `PlaybackCmdSentMsg` 403 message; extend `TransferPlaybackMsg` gate; map new `DeviceInfo` fields in conversions |
| `internal/app/commands.go` | Map new `DeviceInfo` fields in `buildFetchDevicesCmd` conversion; read `m.TargetVolume` in `buildPlaybackAPICmd` |
| `internal/app/app.go` | Change `volumeStep: 5` → `volumeStep: 1` |
| `internal/ui/panes/nowplaying.go` | Add pending fields; update `handleKey`, `handlePlaybackFetched`, `View()` |
| `internal/ui/components/controls.go` | Three-state styles; extended `NewControls` signature; `↻¹` glyph |
| `internal/ui/panes/devices.go` | Render capability indicators in device list |

---

## 5. Out of Scope

- Nerd Font icon set — deferred to a future feature
- `actions.disallows.transferring_playback` UI indicator in DeviceOverlay (gate is wired, visual TBD)
- Seek bar disabled state (architecture supports it via `actions.disallows.Seeking` but seek bar
  UI changes are not part of this spec)
- `GET /me` Premium detection (proactive, deferred — reactive 403 handling is sufficient)

---

## 6. Testing Notes

- Table-driven tests for `Store.ActionAllowed` covering: nil state, `is_restricted`, each
  disallow field, `supports_volume: false`
- Table-driven tests for `Store.IsTargetDeviceRestricted` covering: found/not-found/restricted
- Controls component tests: update `↻1` → `↻¹` assertion; add three-state style assertions for
  each icon
- `NowPlayingPane` tests: pending state set on keypress, cleared on `PlaybackStateFetchedMsg`,
  `resolvedVolume` accumulation across rapid keypresses
- Routing gate tests: capability gate fires correct toast, does not forward to pane
