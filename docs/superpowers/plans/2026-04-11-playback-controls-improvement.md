# Playback Controls Improvement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix playback icon lag (optimistic UI), reduce volume step to 1%, add device capability gating for all playback controls (replacing wrong "Spotify Premium required" errors), give controls a three-state visual (active/inactive/disabled), and fix the repeat-one glyph.

**Architecture:** Pane-local pending state mirrors the existing `localProgressMs` precedent for optimistic UI. A new `checkCapability` helper in `internal/app/` is the single pre-flight gate for all playback key actions. `Store.IsTargetDeviceRestricted` covers device-transfer gating. Domain types gain `SupportsVolume` on `Device` and a new `PlaybackActions` struct wired into `PlaybackState`.

**Tech Stack:** Go 1.22, Bubble Tea v0.27, Lip Gloss, testify

**Spec:** `docs/superpowers/specs/2026-04-11-playback-controls-improvement-design.md`

---

## File Map

| File | What changes |
|---|---|
| `internal/domain/types.go` | Add `SupportsVolume` to `Device`; add `PlaybackActions`, `PlaybackActionsWrapper`; add `Actions` to `PlaybackState` |
| `internal/ui/panes/messages.go` | Add `IsRestricted`, `SupportsVolume` to `DeviceInfo`; add `TargetVolume` to `PlaybackRequestMsg` |
| `internal/state/store.go` | Add `IsTargetDeviceRestricted(deviceID string) bool` |
| `internal/state/store_test.go` | Tests for `IsTargetDeviceRestricted` |
| `internal/app/capability.go` | **New file** — `checkCapability(ps *domain.PlaybackState, action panes.PlaybackAction) (bool, string)` |
| `internal/app/capability_test.go` | **New file** — table-driven tests for all capability cases |
| `internal/app/routing.go` | Add capability gate after Premium gate |
| `internal/app/handlers.go` | Fix 403 message in `PlaybackCmdSentMsg`; extend `TransferPlaybackMsg` gate; fix `DeviceInfo` conversions |
| `internal/app/commands.go` | Fix `DeviceInfo` forward-conversion; change `buildPlaybackAPICmd` signature to accept `targetVolume int`; remove `volStep`/`currentVolume` snapshot |
| `internal/app/app.go` | Change `volumeStep: 5` → `volumeStep: 1` |
| `internal/ui/panes/nowplaying.go` | Add pending fields; update `handleKey`, `handlePlaybackFetched`, `View()` |
| `internal/ui/panes/nowplaying_test.go` | Tests for pending state, accumulation, clear-on-fetch |
| `internal/ui/components/controls.go` | Add `disallows`, `supportsVolume`, `disabledStyle`; extend `NewControls`; three-state `Render()`; `↻¹` glyph |
| `internal/ui/components/controls_test.go` | Update helper; update repeat-track assertion; add disabled-state tests |
| `internal/ui/panes/devices.go` | Render `IsRestricted`/`!SupportsVolume` in `renderDevice` |
| `docs/ARCHITECTURE.md` | Four update locations (see Task 11) |

---

## Task 1: Domain type additions

**Files:**
- Modify: `internal/domain/types.go`

- [ ] **Step 1: Add `SupportsVolume` to `Device` struct**

In `internal/domain/types.go`, the `Device` struct ends at line ~235. Add the new field after `VolumePercent`:

```go
// Device represents a Spotify Connect playback device.
type Device struct {
	ID               string `json:"id"`
	IsActive         bool   `json:"is_active"`
	IsPrivateSession bool   `json:"is_private_session"`
	IsRestricted     bool   `json:"is_restricted"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	VolumePercent    int    `json:"volume_percent"`
	// SupportsVolume indicates whether PUT /me/player/volume is accepted by this device.
	// false for speakers, Chromecast, TV, Automobile, etc. that manage their own volume.
	SupportsVolume bool `json:"supports_volume"`
}
```

- [ ] **Step 2: Add `PlaybackActions`, `PlaybackActionsWrapper`, wire into `PlaybackState`**

After the `Device` struct, add:

```go
// PlaybackActions lists which player operations Spotify is currently disallowing.
// A field set to true means that operation is NOT available right now.
// This reflects subscription tier, active content type (ads, DRM, radio), and device capability.
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

// PlaybackActionsWrapper wraps the disallows object in the Spotify API response.
// Spotify sends: { "actions": { "disallows": { "pausing": true, ... } } }
type PlaybackActionsWrapper struct {
	Disallows PlaybackActions `json:"disallows"`
}
```

In the `PlaybackState` struct (line ~34), add the `Actions` field after `Device`:

```go
type PlaybackState struct {
	IsPlaying    bool   `json:"is_playing"`
	ProgressMs   int    `json:"progress_ms"`
	ShuffleState bool   `json:"shuffle_state"`
	RepeatState  string `json:"repeat_state"`
	Item         *Track  `json:"item"`
	Device       *Device `json:"device"`
	// Actions lists operations currently disallowed by Spotify (content, device, or tier).
	Actions PlaybackActionsWrapper `json:"actions"`
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd /Users/irshadsheikh/dev/github/apps/spotnik && go build ./internal/domain/...
```

Expected: no output (clean build).

- [ ] **Step 4: Commit**

```bash
git add internal/domain/types.go
git commit -m "feat(domain): add SupportsVolume to Device and PlaybackActions to PlaybackState"
```

---

## Task 2: Message type extensions

**Files:**
- Modify: `internal/ui/panes/messages.go`

- [ ] **Step 1: Extend `DeviceInfo`**

`DeviceInfo` is at line ~195. Replace it:

```go
// DeviceInfo is the UI-facing representation of a Spotify device.
// It mirrors the fields needed for rendering without importing api/.
type DeviceInfo struct {
	ID             string
	Name           string
	Type           string
	IsActive       bool
	IsRestricted   bool // true = no Web API commands accepted by this device
	SupportsVolume bool // false = PUT /me/player/volume will fail
}
```

- [ ] **Step 2: Add `TargetVolume` to `PlaybackRequestMsg`**

`PlaybackRequestMsg` is at line ~62. Replace it:

```go
// PlaybackRequestMsg is emitted by the player pane when the user presses a
// playback control key. The root app model receives it and dispatches the
// appropriate Spotify API command.
type PlaybackRequestMsg struct {
	Action PlaybackAction
	// TargetVolume is the resolved absolute volume to set (0–100).
	// Only meaningful for ActionVolumeUp and ActionVolumeDown — ignored for all other actions.
	// The pane computes this from its pending state so the command never re-reads the store.
	TargetVolume int
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./internal/ui/panes/...
```

Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/panes/messages.go
git commit -m "feat(panes): extend DeviceInfo with capability fields; add TargetVolume to PlaybackRequestMsg"
```

---

## Task 3: Store `IsTargetDeviceRestricted`

**Files:**
- Modify: `internal/state/store.go`
- Modify: `internal/state/store_test.go`

- [ ] **Step 1: Write the failing test**

In `internal/state/store_test.go`, add at the end of the file:

```go
func TestStore_IsTargetDeviceRestricted(t *testing.T) {
	tests := []struct {
		name     string
		devices  []domain.Device
		deviceID string
		want     bool
	}{
		{
			name:     "no devices cached — returns false (safe default)",
			devices:  nil,
			deviceID: "abc",
			want:     false,
		},
		{
			name: "device found, not restricted",
			devices: []domain.Device{
				{ID: "abc", IsRestricted: false},
			},
			deviceID: "abc",
			want:     false,
		},
		{
			name: "device found, is restricted",
			devices: []domain.Device{
				{ID: "abc", IsRestricted: true},
			},
			deviceID: "abc",
			want:     true,
		},
		{
			name: "device ID not in list — returns false",
			devices: []domain.Device{
				{ID: "xyz", IsRestricted: true},
			},
			deviceID: "abc",
			want:     false,
		},
		{
			name: "multiple devices, target is restricted",
			devices: []domain.Device{
				{ID: "aaa", IsRestricted: false},
				{ID: "bbb", IsRestricted: true},
				{ID: "ccc", IsRestricted: false},
			},
			deviceID: "bbb",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			if tt.devices != nil {
				s.SetDevices(tt.devices)
			}
			got := s.IsTargetDeviceRestricted(tt.deviceID)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/state/... -run TestStore_IsTargetDeviceRestricted -v
```

Expected: `FAIL — undefined: Store.IsTargetDeviceRestricted`

- [ ] **Step 3: Implement `IsTargetDeviceRestricted`**

In `internal/state/store.go`, add after `Devices()`:

```go
// IsTargetDeviceRestricted returns true if the device with the given ID has
// IsRestricted set in the cached device list. Returns false when the device is
// not found (safe default — let the API decide). Used by the TransferPlaybackMsg
// handler to block transfers to restricted devices before making an API call.
func (s *Store) IsTargetDeviceRestricted(deviceID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.devices {
		if d.ID == deviceID {
			return d.IsRestricted
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/state/... -run TestStore_IsTargetDeviceRestricted -v
```

Expected: all 5 sub-tests PASS.

- [ ] **Step 5: Run full state test suite**

```bash
go test ./internal/state/...
```

Expected: PASS, no regressions.

- [ ] **Step 6: Commit**

```bash
git add internal/state/store.go internal/state/store_test.go
git commit -m "feat(state): add IsTargetDeviceRestricted capability lookup"
```

---

## Task 4: App capability helper

**Files:**
- Create: `internal/app/capability.go`
- Create: `internal/app/capability_test.go`

This function lives in `internal/app/` (not `state/`) to avoid an import cycle —
`state/` must not import `ui/panes/`, but `app/` already imports both.

- [ ] **Step 1: Write the failing tests**

Create `internal/app/capability_test.go`:

```go
package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
)

func TestCheckCapability(t *testing.T) {
	tests := []struct {
		name        string
		ps          *domain.PlaybackState
		action      panes.PlaybackAction
		wantAllowed bool
		wantReason  string
	}{
		{
			name:        "nil playback state — always allowed (no state yet)",
			ps:          nil,
			action:      panes.ActionVolumeUp,
			wantAllowed: true,
		},
		{
			name: "device is_restricted — blocks all actions",
			ps: &domain.PlaybackState{
				Device: &domain.Device{IsRestricted: true, SupportsVolume: true},
			},
			action:      panes.ActionPlay,
			wantAllowed: false,
			wantReason:  "Device not controllable via API",
		},
		{
			name: "volume up — device supports volume — allowed",
			ps: &domain.PlaybackState{
				Device: &domain.Device{IsRestricted: false, SupportsVolume: true},
			},
			action:      panes.ActionVolumeUp,
			wantAllowed: true,
		},
		{
			name: "volume down — device does not support volume — blocked",
			ps: &domain.PlaybackState{
				Device: &domain.Device{IsRestricted: false, SupportsVolume: false},
			},
			action:      panes.ActionVolumeDown,
			wantAllowed: false,
			wantReason:  "Volume not available on this device",
		},
		{
			name: "skip next — disallowed by actions",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{SkippingNext: true}},
			},
			action:      panes.ActionNext,
			wantAllowed: false,
			wantReason:  "Skip not available in this context",
		},
		{
			name: "skip previous — disallowed by actions",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{SkippingPrev: true}},
			},
			action:      panes.ActionPrevious,
			wantAllowed: false,
			wantReason:  "Skip not available in this context",
		},
		{
			name: "toggle shuffle — disallowed",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{TogglingShuffle: true}},
			},
			action:      panes.ActionToggleShuffle,
			wantAllowed: false,
			wantReason:  "Shuffle not available in this context",
		},
		{
			name: "cycle repeat — both repeat modes disallowed",
			ps: &domain.PlaybackState{
				Device: &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{
					TogglingRepeatContext: true,
					TogglingRepeatTrack:   true,
				}},
			},
			action:      panes.ActionCycleRepeat,
			wantAllowed: false,
			wantReason:  "Repeat not available in this context",
		},
		{
			name: "cycle repeat — only repeat-context disallowed (track still available) — allowed",
			ps: &domain.PlaybackState{
				Device: &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{
					TogglingRepeatContext: true,
					TogglingRepeatTrack:   false,
				}},
			},
			action:      panes.ActionCycleRepeat,
			wantAllowed: true,
		},
		{
			name: "play — resuming disallowed",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{Resuming: true}},
			},
			action:      panes.ActionPlay,
			wantAllowed: false,
			wantReason:  "Playback not available in this context",
		},
		{
			name: "pause — pausing disallowed",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{Pausing: true}},
			},
			action:      panes.ActionPause,
			wantAllowed: false,
			wantReason:  "Playback not available in this context",
		},
		{
			name: "all disallows false — action allowed",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{},
			},
			action:      panes.ActionNext,
			wantAllowed: true,
		},
		{
			name: "nil device — volume check skipped (safe default)",
			ps: &domain.PlaybackState{
				Device: nil,
			},
			action:      panes.ActionVolumeUp,
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := checkCapability(tt.ps, tt.action)
			assert.Equal(t, tt.wantAllowed, got, "allowed mismatch")
			if !tt.wantAllowed {
				assert.Equal(t, tt.wantReason, reason, "reason mismatch")
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/app/... -run TestCheckCapability -v
```

Expected: `FAIL — undefined: checkCapability`

- [ ] **Step 3: Implement `checkCapability`**

Create `internal/app/capability.go`:

```go
package app

import (
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
)

// checkCapability is the single pre-flight gate for all playback key actions.
// It returns (true, "") when the action is permitted and (false, reason) when blocked.
//
// Three signal sources, checked in order:
//  1. device.IsRestricted — total lockout; no Web API commands accepted by this device.
//  2. device.SupportsVolume — volume-specific; only for ActionVolumeUp/Down.
//  3. playbackState.Actions.Disallows — runtime signal reflecting subscription tier,
//     content type (ads, DRM, radio), and device capability in one object.
//
// Returns (true, "") when ps is nil — no state yet; let the API respond.
// This function lives in app/ (not state/) to avoid importing ui/panes/ from state/.
func checkCapability(ps *domain.PlaybackState, action panes.PlaybackAction) (bool, string) {
	if ps == nil {
		return true, ""
	}

	// Total device lockout — no Web API commands accepted.
	if ps.Device != nil && ps.Device.IsRestricted {
		return false, "Device not controllable via API"
	}

	d := ps.Actions.Disallows

	switch action {
	case panes.ActionVolumeUp, panes.ActionVolumeDown:
		if ps.Device != nil && !ps.Device.SupportsVolume {
			return false, "Volume not available on this device"
		}
	case panes.ActionNext:
		if d.SkippingNext {
			return false, "Skip not available in this context"
		}
	case panes.ActionPrevious:
		if d.SkippingPrev {
			return false, "Skip not available in this context"
		}
	case panes.ActionToggleShuffle:
		if d.TogglingShuffle {
			return false, "Shuffle not available in this context"
		}
	case panes.ActionCycleRepeat:
		// Only blocked when BOTH repeat modes are disallowed — if either is available,
		// the cycle can still advance to that mode.
		if d.TogglingRepeatContext && d.TogglingRepeatTrack {
			return false, "Repeat not available in this context"
		}
	case panes.ActionPlay:
		if d.Resuming {
			return false, "Playback not available in this context"
		}
	case panes.ActionPause:
		if d.Pausing {
			return false, "Playback not available in this context"
		}
	}

	return true, ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/app/... -run TestCheckCapability -v
```

Expected: all 13 sub-tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/capability.go internal/app/capability_test.go
git commit -m "feat(app): add checkCapability pre-flight gate for playback actions"
```

---

## Task 5: Routing capability gate

**Files:**
- Modify: `internal/app/routing.go`

- [ ] **Step 1: Add the capability gate**

In `routing.go`, find the playback key block starting at line ~207:

```go
if isPlaybackKey(m) {
    // Gate: free-tier users are blocked from Premium-only API operations.
    // 'v' (visualizer cycle) is exempt — it is a local UI action, not an API call.
    if isPremiumOnlyPlaybackKey(m) && !a.store.IsPremium() {
        return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required")
    }
    np := a.nowPlayingPane()
```

Replace with:

```go
if isPlaybackKey(m) {
    // Gate 1: free-tier users are blocked from Premium-only API operations.
    // 'v' (visualizer cycle) is exempt — it is a local UI action, not an API call.
    if isPremiumOnlyPlaybackKey(m) && !a.store.IsPremium() {
        return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required")
    }
    // Gate 2: device capability check — blocks actions the active device or current
    // Spotify context disallows (wrong device type, DRM content, ad playback, etc.).
    // 'v' is exempt for the same reason as above.
    if isPremiumOnlyPlaybackKey(m) {
        action := playbackKeyToAction(m)
        if ok, reason := checkCapability(a.store.PlaybackState(), action); !ok {
            return a, a.alerts.NewAlertCmd("warning", reason)
        }
    }
    np := a.nowPlayingPane()
```

- [ ] **Step 2: Add `playbackKeyToAction` helper in `routing.go`**

Add below `isPremiumOnlyPlaybackKey`:

```go
// playbackKeyToAction maps a Premium-only playback key to its PlaybackAction.
// Only called after isPremiumOnlyPlaybackKey returns true, so the switch is exhaustive.
func playbackKeyToAction(m tea.KeyMsg) panes.PlaybackAction {
    if m.Type == tea.KeyRunes {
        switch string(m.Runes) {
        case "+":
            return panes.ActionVolumeUp
        case "-":
            return panes.ActionVolumeDown
        case "s":
            return panes.ActionToggleShuffle
        case "r":
            return panes.ActionCycleRepeat
        }
    }
    switch m.Type {
    case tea.KeyLeft:
        return panes.ActionPrevious
    case tea.KeyRight:
        return panes.ActionNext
    default: // tea.KeySpace
        // Space toggles play/pause; map to Pause as the conservative choice.
        // checkCapability will use ps.IsPlaying context — but routing only needs
        // to identify the key; the pane will send the correct ActionPause/ActionPlay.
        return panes.ActionPause
    }
}
```

- [ ] **Step 3: Build to verify no compile errors**

```bash
go build ./internal/app/...
```

Expected: clean.

- [ ] **Step 4: Run lint**

```bash
make lint
```

Expected: no new lint errors.

- [ ] **Step 5: Commit**

```bash
git add internal/app/routing.go
git commit -m "feat(routing): add device capability gate after Premium gate for playback keys"
```

---

## Task 6: Handler updates

**Files:**
- Modify: `internal/app/handlers.go`

Three separate changes in one file.

### 6a: Fix `PlaybackCmdSentMsg` 403 message

- [ ] **Step 1: Fix the hardcoded forbidden message**

Find the `PlaybackCmdSentMsg` handler at line ~500:

```go
var forbiddenErr *api.ForbiddenError
if errors.As(m.Err, &forbiddenErr) {
    return a, tea.Batch(
        fetchPlaybackStateCmd(a.player),
        a.alerts.NewAlertCmd("warning", "Spotify Premium required"),
    )
}
```

Replace with:

```go
var forbiddenErr *api.ForbiddenError
if errors.As(m.Err, &forbiddenErr) {
    msg := forbiddenErr.Message
    if msg == "" {
        msg = "Spotify Premium required"
    }
    return a, tea.Batch(
        fetchPlaybackStateCmd(a.player),
        a.alerts.NewAlertCmd("warning", msg),
    )
}
```

### 6b: Extend `TransferPlaybackMsg` gate

- [ ] **Step 2: Add capability and device-restriction gates to transfer handler**

Find the `TransferPlaybackMsg` handler at line ~862:

```go
case panes.TransferPlaybackMsg:
    a.deviceOverlayOpen = false
    if !a.store.IsPremium() {
        return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required")
    }
    return a, tea.Batch(
        a.buildTransferPlaybackCmd(m.DeviceID),
        a.alerts.NewAlertCmd("info", fmt.Sprintf("Switching to %s...", m.DeviceName)),
    )
```

Replace with:

```go
case panes.TransferPlaybackMsg:
    a.deviceOverlayOpen = false
    if !a.store.IsPremium() {
        return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required")
    }
    if ps := a.store.PlaybackState(); ps != nil {
        if ps.Actions.Disallows.TransferringPlayback {
            return a, a.alerts.NewAlertCmd("warning", "Playback transfer not available")
        }
    }
    if a.store.IsTargetDeviceRestricted(m.DeviceID) {
        return a, a.alerts.NewAlertCmd("warning", "Device not controllable via API")
    }
    return a, tea.Batch(
        a.buildTransferPlaybackCmd(m.DeviceID),
        a.alerts.NewAlertCmd("info", fmt.Sprintf("Switching to %s...", m.DeviceName)),
    )
```

### 6c: Fix `DevicesLoadedMsg` reverse-conversion (DeviceInfo → domain.Device)

- [ ] **Step 3: Fix reverse-conversion to carry `IsRestricted` and `SupportsVolume`**

Find the reverse-conversion in the `DevicesLoadedMsg` handler at line ~843:

```go
rawDevices := make([]domain.Device, 0, len(m.Devices))
for _, info := range m.Devices {
    rawDevices = append(rawDevices, domain.Device{
        ID:       info.ID,
        Name:     info.Name,
        Type:     info.Type,
        IsActive: info.IsActive,
    })
}
```

Replace with:

```go
rawDevices := make([]domain.Device, 0, len(m.Devices))
for _, info := range m.Devices {
    rawDevices = append(rawDevices, domain.Device{
        ID:             info.ID,
        Name:           info.Name,
        Type:           info.Type,
        IsActive:       info.IsActive,
        IsRestricted:   info.IsRestricted,
        SupportsVolume: info.SupportsVolume,
    })
}
```

- [ ] **Step 4: Build to verify**

```bash
go build ./internal/app/...
```

Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/app/handlers.go
git commit -m "fix(handlers): use actual Spotify error message for 403; extend transfer gate; fix DeviceInfo reverse-conversion"
```

---

## Task 7: Commands + volume step

**Files:**
- Modify: `internal/app/commands.go`
- Modify: `internal/app/app.go`

### 7a: Fix DeviceInfo forward-conversion in `buildFetchDevicesCmd`

- [ ] **Step 1: Fix forward-conversion (domain.Device → DeviceInfo)**

Find the conversion in `buildFetchDevicesCmd` at line ~385:

```go
for _, d := range devList {
    infos = append(infos, panes.DeviceInfo{
        ID:       d.ID,
        Name:     d.Name,
        Type:     d.Type,
        IsActive: d.IsActive,
    })
}
```

Replace with:

```go
for _, d := range devList {
    infos = append(infos, panes.DeviceInfo{
        ID:             d.ID,
        Name:           d.Name,
        Type:           d.Type,
        IsActive:       d.IsActive,
        IsRestricted:   d.IsRestricted,
        SupportsVolume: d.SupportsVolume,
    })
}
```

### 7b: Change `buildPlaybackAPICmd` to accept `targetVolume`

- [ ] **Step 2: Update `buildPlaybackAPICmd` signature and volume handling**

Find the method signature at line ~41:

```go
func (a *App) buildPlaybackAPICmd(action panes.PlaybackAction) tea.Cmd {
```

Change to:

```go
func (a *App) buildPlaybackAPICmd(action panes.PlaybackAction, targetVolume int) tea.Cmd {
```

Find the snapshot block at lines ~46-60 — remove `volStep` and `currentVolume` since they are
no longer needed (the pane now sends the resolved target in the message):

```go
// Before — remove these two lines:
volStep := a.volumeStep
// ...
currentVolume := 65
if ps != nil {
    if ps.Device != nil {
        currentVolume = ps.Device.VolumePercent
    }
    ...
}
```

The final snapshot block should look like (only shuffle and repeat remain):

```go
ps := a.store.PlaybackState()
isShuffled := false
repeatMode := "off"
if ps != nil {
    isShuffled = ps.ShuffleState
    repeatMode = ps.RepeatState
}
```

Find the volume cases at lines ~76-87:

```go
case panes.ActionVolumeUp:
    newVol := currentVolume + volStep
    if newVol > 100 {
        newVol = 100
    }
    err = player.SetVolume(ctx, newVol)
case panes.ActionVolumeDown:
    newVol := currentVolume - volStep
    if newVol < 0 {
        newVol = 0
    }
    err = player.SetVolume(ctx, newVol)
```

Replace with:

```go
case panes.ActionVolumeUp, panes.ActionVolumeDown:
    err = player.SetVolume(ctx, targetVolume)
```

- [ ] **Step 3: Update the call site in `handlers.go`**

Find the `PlaybackRequestMsg` handler at line ~519:

```go
case panes.PlaybackRequestMsg:
    return a, a.buildPlaybackAPICmd(m.Action)
```

Replace with:

```go
case panes.PlaybackRequestMsg:
    return a, a.buildPlaybackAPICmd(m.Action, m.TargetVolume)
```

### 7c: Change volume step to 1%

- [ ] **Step 4: Change `volumeStep: 5` → `volumeStep: 1` in `app.go`**

In `internal/app/app.go` at line ~300:

```go
volumeStep: 5,
```

Change to:

```go
volumeStep: 1,
```

Note: `volumeStep` is now only used for the `+`/`-` step size in `NowPlayingPane.handleKey`
(Task 8). The `buildPlaybackAPICmd` no longer reads it — the pane sends the resolved
`TargetVolume` directly. Confirm `volumeStep` is still passed to `NowPlayingPane` or read
through the store when needed.

Actually, looking at the design: `NowPlayingPane` computes `pendingVolume` using `+1`/`-1`
steps directly (hardcoded 1%). `volumeStep` on `App` is no longer read anywhere useful after
this task. Leave it in place but its value (1) is now just a documentation of intent. The
pane uses hardcoded `+1`/`-1` in Task 8.

- [ ] **Step 5: Build to verify**

```bash
go build ./internal/app/...
```

Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add internal/app/commands.go internal/app/handlers.go internal/app/app.go
git commit -m "feat(commands): pass TargetVolume through message; remove volume snapshot from buildPlaybackAPICmd; volume step 1%"
```

---

## Task 8: NowPlaying optimistic UI

**Files:**
- Modify: `internal/ui/panes/nowplaying.go`
- Modify: `internal/ui/panes/nowplaying_test.go`

- [ ] **Step 1: Write failing tests first**

In `internal/ui/panes/nowplaying_test.go`, find the test file and add these tests. Read the
existing file first to understand the test helper setup, then add:

```go
func TestNowPlayingPane_OptimisticShuffle(t *testing.T) {
	store := state.New()
	store.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:    true,
		ShuffleState: false,
		RepeatState:  "off",
		Device:       &domain.Device{VolumePercent: 50, SupportsVolume: true},
		Item: &domain.Track{
			Name:       "Test",
			DurationMs: 200000,
			Artists:    []domain.Artist{{Name: "Artist"}},
		},
	})
	p := NewNowPlayingPane(store, theme.Load("black"))
	p.SetSize(80, 20)

	// Press 's' — should set pendingShuffleOn immediately.
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	p.SetFocused(true)
	updated, _ := p.Update(keyMsg)
	np := updated.(*NowPlayingPane)

	require.NotNil(t, np.pendingShuffleOn, "pendingShuffleOn should be set after keypress")
	assert.True(t, *np.pendingShuffleOn, "shuffle was off, pending should be true")
}

func TestNowPlayingPane_OptimisticShuffleClearedOnFetch(t *testing.T) {
	store := state.New()
	store.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:    true,
		ShuffleState: false,
		RepeatState:  "off",
		Device:       &domain.Device{VolumePercent: 50, SupportsVolume: true},
		Item: &domain.Track{
			Name:       "Test",
			DurationMs: 200000,
			Artists:    []domain.Artist{{Name: "Artist"}},
		},
	})
	p := NewNowPlayingPane(store, theme.Load("black"))
	p.SetSize(80, 20)
	p.SetFocused(true)

	// Set pending state via keypress.
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated, _ := p.Update(keyMsg)
	np := updated.(*NowPlayingPane)
	require.NotNil(t, np.pendingShuffleOn)

	// Simulate server poll arriving — pending should clear.
	updated2, _ := np.Update(PlaybackStateFetchedMsg{State: store.PlaybackState()})
	np2 := updated2.(*NowPlayingPane)
	assert.Nil(t, np2.pendingShuffleOn, "pendingShuffleOn should clear after PlaybackStateFetchedMsg")
}

func TestNowPlayingPane_VolumeAccumulation(t *testing.T) {
	store := state.New()
	store.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:   true,
		RepeatState: "off",
		Device:      &domain.Device{VolumePercent: 50, SupportsVolume: true},
		Item: &domain.Track{
			Name:       "Test",
			DurationMs: 200000,
			Artists:    []domain.Artist{{Name: "Artist"}},
		},
	})
	p := NewNowPlayingPane(store, theme.Load("black"))
	p.SetSize(80, 20)
	p.SetFocused(true)

	// Press '+' five times — pendingVolume should accumulate to 55.
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")}
	var updated tea.Model = p
	for i := 0; i < 5; i++ {
		updated, _ = updated.(*NowPlayingPane).Update(keyMsg)
	}
	np := updated.(*NowPlayingPane)

	require.NotNil(t, np.pendingVolume, "pendingVolume should be set")
	assert.Equal(t, 55, *np.pendingVolume, "five +1 presses from 50 should reach 55")
}

func TestNowPlayingPane_VolumeClearedOnFetch(t *testing.T) {
	store := state.New()
	store.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:   true,
		RepeatState: "off",
		Device:      &domain.Device{VolumePercent: 50, SupportsVolume: true},
		Item: &domain.Track{
			Name:       "Test",
			DurationMs: 200000,
			Artists:    []domain.Artist{{Name: "Artist"}},
		},
	})
	p := NewNowPlayingPane(store, theme.Load("black"))
	p.SetSize(80, 20)
	p.SetFocused(true)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")}
	updated, _ := p.Update(keyMsg)
	np := updated.(*NowPlayingPane)
	require.NotNil(t, np.pendingVolume)

	updated2, _ := np.Update(PlaybackStateFetchedMsg{State: store.PlaybackState()})
	np2 := updated2.(*NowPlayingPane)
	assert.Nil(t, np2.pendingVolume, "pendingVolume should clear after PlaybackStateFetchedMsg")
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/panes/... -run "TestNowPlayingPane_Optimistic|TestNowPlayingPane_Volume" -v
```

Expected: compile error — `pendingShuffleOn` undefined.

- [ ] **Step 3: Add pending fields to `NowPlayingPane` struct**

In `nowplaying.go`, the struct is at line ~27. Add after `localProgressMs`:

```go
// Optimistic state — set on keypress to update the UI immediately.
// Cleared by handlePlaybackFetched() when the server poll arrives (server truth wins).
// This mirrors the localProgressMs pattern for smooth seek-bar interpolation.
pendingIsPlaying  *bool   // nil = no pending state
pendingShuffleOn  *bool
pendingRepeatMode *string // "off" | "context" | "track"
pendingVolume     *int    // absolute volume to set, accumulated across rapid keypresses
```

- [ ] **Step 4: Add `resolvedVolume` helper**

After the `emitPlaybackRequest` function (~line 411), add:

```go
// resolvedVolume returns the current pending volume if set, otherwise the server-confirmed
// volume from the store. Used to correctly accumulate rapid +/- keypresses.
func (p *NowPlayingPane) resolvedVolume() int {
	if p.pendingVolume != nil {
		return *p.pendingVolume
	}
	ps := p.store.PlaybackState()
	if ps != nil && ps.Device != nil {
		return ps.Device.VolumePercent
	}
	return 65 // safe default when no device info available
}
```

- [ ] **Step 5: Update `handleKey` to set pending state**

In `handleKey` (~line 358), update the play/pause, shuffle, repeat, and volume cases:

```go
func (p *NowPlayingPane) handleKey(msg tea.KeyMsg) (*NowPlayingPane, tea.Cmd) {
	switch {
	case msg.Type == tea.KeySpace:
		ps := p.store.PlaybackState()
		if ps != nil && ps.IsPlaying {
			f := false
			p.pendingIsPlaying = &f
			return p, emitPlaybackRequest(ActionPause)
		}
		t := true
		p.pendingIsPlaying = &t
		return p, emitPlaybackRequest(ActionPlay)

	case msg.Type == tea.KeyRight:
		return p, emitPlaybackRequest(ActionNext)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "p",
		msg.Type == tea.KeyLeft:
		return p, emitPlaybackRequest(ActionPrevious)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "+":
		newVol := p.resolvedVolume() + 1
		if newVol > 100 {
			newVol = 100
		}
		p.pendingVolume = &newVol
		return p, func() tea.Msg {
			return PlaybackRequestMsg{Action: ActionVolumeUp, TargetVolume: newVol}
		}

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "-":
		newVol := p.resolvedVolume() - 1
		if newVol < 0 {
			newVol = 0
		}
		p.pendingVolume = &newVol
		return p, func() tea.Msg {
			return PlaybackRequestMsg{Action: ActionVolumeDown, TargetVolume: newVol}
		}

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "s":
		ps := p.store.PlaybackState()
		next := true
		if ps != nil {
			next = !ps.ShuffleState
		}
		p.pendingShuffleOn = &next
		return p, emitPlaybackRequest(ActionToggleShuffle)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "r":
		ps := p.store.PlaybackState()
		mode := "context"
		if ps != nil {
			mode = nextRepeatMode(ps.RepeatState)
		}
		p.pendingRepeatMode = &mode
		return p, emitPlaybackRequest(ActionCycleRepeat)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "v":
		p.engine.CyclePattern()
		return p, func() tea.Msg {
			return VisualizerPatternChangedMsg{PatternIndex: p.engine.Pattern()}
		}
	}

	return p, nil
}
```

Note: `nextRepeatMode` is already defined in `commands.go` in the `app` package. Since `nowplaying.go` is in `panes` package, define a local copy:

```go
// nextRepeatMode returns the next repeat mode in the cycle off→context→track→off.
// Mirrors the same function in app/commands.go — kept local to avoid cross-package dependency.
func nextRepeatMode(current string) string {
	switch current {
	case "off":
		return "context"
	case "context":
		return "track"
	default:
		return "off"
	}
}
```

(Check if this function already exists in the panes package — if so skip adding it.)

- [ ] **Step 6: Update `handlePlaybackFetched` to clear pending state**

Replace the existing `handlePlaybackFetched` (~line 344):

```go
func (p *NowPlayingPane) handlePlaybackFetched() (*NowPlayingPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil {
		p.localProgressMs = ps.ProgressMs
		p.engine.SetPlaying(ps.IsPlaying)
	} else {
		p.localProgressMs = 0
		p.engine.SetPlaying(false)
	}
	// Clear all optimistic state — server truth wins.
	p.pendingIsPlaying = nil
	p.pendingShuffleOn = nil
	p.pendingRepeatMode = nil
	p.pendingVolume = nil
	return p, nil
}
```

- [ ] **Step 7: Update `View()` to use pending values**

In `View()`, after reading `ps` at line ~187, add resolution of pending state:

```go
// Resolve optimistic state: pending values take precedence until the server poll clears them.
isPlaying := ps.IsPlaying
if p.pendingIsPlaying != nil {
    isPlaying = *p.pendingIsPlaying
}
shuffleOn := ps.ShuffleState
if p.pendingShuffleOn != nil {
    shuffleOn = *p.pendingShuffleOn
}
repeatMode := ps.RepeatState
if p.pendingRepeatMode != nil {
    repeatMode = *p.pendingRepeatMode
}
volume := 0
supportsVolume := true
if ps.Device != nil {
    volume = ps.Device.VolumePercent
    supportsVolume = ps.Device.SupportsVolume
}
if p.pendingVolume != nil {
    volume = *p.pendingVolume
}
```

Then change the `NewControls` call (currently at line ~207):

```go
// Before:
ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)

// After:
ctrl := components.NewControls(p.theme, isPlaying, shuffleOn, repeatMode, ps.Actions.Disallows, supportsVolume)
```

And change the `p.volumeBar.Render(volume)` lines — when volume is not supported, show a
muted placeholder instead:

```go
var volLine string
if supportsVolume {
    volLine = p.volumeBar.Render(volume)
} else {
    volLine = lipgloss.NewStyle().Foreground(p.theme.TextMuted()).Render("♪ volume unavailable")
}
```

Replace all four occurrences of `p.volumeBar.Render(volume)` in the `infoLines` switch with
`volLine`.

Also remove the old `volume` computation at line ~203:

```go
// Remove these lines (now computed above with supportsVolume):
volume := 0
if ps.Device != nil {
    volume = ps.Device.VolumePercent
}
```

- [ ] **Step 8: Run the new tests**

```bash
go test ./internal/ui/panes/... -run "TestNowPlayingPane_Optimistic|TestNowPlayingPane_Volume" -v
```

Expected: all 4 tests PASS.

- [ ] **Step 9: Run full pane test suite**

```bash
go test ./internal/ui/panes/...
```

Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add internal/ui/panes/nowplaying.go internal/ui/panes/nowplaying_test.go
git commit -m "feat(nowplaying): optimistic UI for play/pause/shuffle/repeat/volume with pending state"
```

---

## Task 9: Controls three-state rendering + repeat-one glyph

**Files:**
- Modify: `internal/ui/components/controls.go`
- Modify: `internal/ui/components/controls_test.go`

- [ ] **Step 1: Write failing tests**

In `controls_test.go`, update the helper and add new tests. Replace the existing
`newTestControls` helper and add disabled-state tests:

```go
import "github.com/initgrep-apps/spotnik/internal/domain"

func newTestControls(isPlaying, shuffleOn bool, repeatMode string) Controls {
	t := theme.Load("black")
	return NewControls(t, isPlaying, shuffleOn, repeatMode, domain.PlaybackActions{}, true)
}

func newTestControlsDisallows(isPlaying, shuffleOn bool, repeatMode string, disallows domain.PlaybackActions, supportsVolume bool) Controls {
	t := theme.Load("black")
	return NewControls(t, isPlaying, shuffleOn, repeatMode, disallows, supportsVolume)
}

func TestControls_RepeatTrack_SuperscriptOne(t *testing.T) {
	c := newTestControls(false, false, "track")
	out := c.Render()
	assert.Contains(t, out, "↻¹", "repeat-track should use superscript one (U+00B9)")
	assert.NotContains(t, out, "↻1", "repeat-track should not use ASCII 1")
}

func TestControls_ShuffleDisabled(t *testing.T) {
	c := newTestControlsDisallows(false, false, "off",
		domain.PlaybackActions{TogglingShuffle: true}, true)
	out := c.Render()
	// Shuffle icon still present but in disabled style (muted color).
	assert.Contains(t, out, "⇄", "disabled shuffle should still render the icon")
}

func TestControls_RepeatDisabled_BothModesDisallowed(t *testing.T) {
	c := newTestControlsDisallows(false, false, "off",
		domain.PlaybackActions{TogglingRepeatContext: true, TogglingRepeatTrack: true}, true)
	out := c.Render()
	assert.Contains(t, out, "↻", "disabled repeat should still render the icon")
}

func TestControls_RepeatNotDisabled_OnlyContextDisallowed(t *testing.T) {
	// If only context-repeat is disallowed, repeat is not fully disabled.
	c := newTestControlsDisallows(false, false, "off",
		domain.PlaybackActions{TogglingRepeatContext: true, TogglingRepeatTrack: false}, true)
	out := c.Render()
	assert.Contains(t, out, "↻")
}

func TestControls_PlayDisabled_ResumingDisallowed(t *testing.T) {
	c := newTestControlsDisallows(false, false, "off",
		domain.PlaybackActions{Resuming: true}, true)
	out := c.Render()
	assert.Contains(t, out, "▷", "disabled play should still render the icon")
}
```

Also update the existing `TestControls_RepeatTrack` test to match the new glyph:

```go
func TestControls_RepeatTrack(t *testing.T) {
	c := newTestControls(false, false, "track")
	out := c.Render()
	assert.Contains(t, out, "↻¹") // was "↻1"
}
```

And update `TestControls_RepeatOff` to remove the `↻1` NotContains assertion (now `↻¹`):

```go
func TestControls_RepeatOff(t *testing.T) {
	c := newTestControls(false, false, "off")
	out := c.Render()
	assert.Contains(t, out, "↻")
	assert.NotContains(t, out, "↻¹", "off state should not show superscript one")
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/components/... -run TestControls -v
```

Expected: compile error — `NewControls` wrong argument count.

- [ ] **Step 3: Rewrite `controls.go`**

Replace the entire file:

```go
package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Controls renders the transport controls row: ⇄ ▷/⏸ ≡ ↻
// Three visual states per icon:
//   - active:   PlayingIndicator color (on and available)
//   - inactive: TextSecondary color   (off but available)
//   - disabled: TextMuted color       (unavailable per device/context/subscription)
type Controls struct {
	isPlaying      bool
	shuffleOn      bool
	repeatMode     string
	disallows      domain.PlaybackActions
	supportsVolume bool

	activeStyle   lipgloss.Style
	inactiveStyle lipgloss.Style
	disabledStyle lipgloss.Style
}

// NewControls creates a Controls renderer with the given state, capability context, and theme.
// repeatMode must be one of "off", "context", or "track".
// disallows reflects the current Spotify actions.disallows object from PlaybackState.
// supportsVolume is device.SupportsVolume from the active device.
func NewControls(t theme.Theme, isPlaying, shuffleOn bool, repeatMode string, disallows domain.PlaybackActions, supportsVolume bool) Controls {
	return Controls{
		isPlaying:      isPlaying,
		shuffleOn:      shuffleOn,
		repeatMode:     repeatMode,
		disallows:      disallows,
		supportsVolume: supportsVolume,
		activeStyle:    lipgloss.NewStyle().Foreground(t.PlayingIndicator()),
		inactiveStyle:  lipgloss.NewStyle().Foreground(t.TextSecondary()),
		disabledStyle:  lipgloss.NewStyle().Foreground(t.TextMuted()),
	}
}

// Render returns the controls row as a string.
func (c Controls) Render() string {
	// Shuffle: active (on), inactive (off), disabled (Spotify disallows toggling).
	var shuffle string
	switch {
	case c.disallows.TogglingShuffle:
		shuffle = c.disabledStyle.Render("⇄")
	case c.shuffleOn:
		shuffle = c.activeStyle.Render("⇄")
	default:
		shuffle = c.inactiveStyle.Render("⇄")
	}

	// Play/Pause: disabled when the current state's action is disallowed.
	var playPause string
	switch {
	case c.isPlaying && c.disallows.Pausing:
		playPause = c.disabledStyle.Render("⏸")
	case !c.isPlaying && c.disallows.Resuming:
		playPause = c.disabledStyle.Render("▷")
	case c.isPlaying:
		playPause = c.activeStyle.Render("⏸")
	default:
		playPause = c.inactiveStyle.Render("▷")
	}

	queue := c.inactiveStyle.Render("≡")

	// Repeat: disabled only when BOTH modes are disallowed.
	// ↻¹ uses superscript one (U+00B9) for visual balance with the arrow glyph.
	repeatDisabled := c.disallows.TogglingRepeatContext && c.disallows.TogglingRepeatTrack
	var repeat string
	switch {
	case repeatDisabled:
		if c.repeatMode == "track" {
			repeat = c.disabledStyle.Render("↻¹")
		} else {
			repeat = c.disabledStyle.Render("↻")
		}
	case c.repeatMode == "track":
		repeat = c.activeStyle.Render("↻¹")
	case c.repeatMode == "context":
		repeat = c.activeStyle.Render("↻")
	default:
		repeat = c.inactiveStyle.Render("↻")
	}

	return shuffle + "  " + playPause + "  " + queue + "  " + repeat
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/ui/components/... -run TestControls -v
```

Expected: all PASS.

- [ ] **Step 5: Run full component test suite**

```bash
go test ./internal/ui/components/...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/ui/components/controls.go internal/ui/components/controls_test.go
git commit -m "feat(controls): three-state rendering (active/inactive/disabled); fix ↻¹ repeat-one glyph"
```

---

## Task 10: DeviceOverlay visual capability indicators

**Files:**
- Modify: `internal/ui/panes/devices.go`

- [ ] **Step 1: Update `renderDevice` to show restricted and no-volume indicators**

In `devices.go`, find `renderDevice` at line ~198. The current function computes `bullet`,
`bulletStyle`, `nameStyle`, and `label`. Add capability indicators to the label construction.

Find where `label` is built (it shows `[active]` text for the active device). After the
existing label logic, add the capability suffix:

```go
// After the existing label computation, add capability indicators:
var capSuffix string
if dev.IsRestricted {
    capSuffix = lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Render(" [restricted]")
} else if !dev.SupportsVolume {
    capSuffix = lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Render(" (no vol)")
}
```

Then append `capSuffix` to the final row assembly. Find the return statement that joins
`bullet`, device name, and label, and append `capSuffix` at the end.

Read `renderDevice` in full before editing to understand exactly where the return assembles the
row. The typical pattern ends with something like:

```go
row := bulletStyle.Render(bullet) + " " + typeIcon + " " + nameStyle.Render(dev.Name) + labelPart
```

Append `+ capSuffix` to this final assembly.

- [ ] **Step 2: Build and run device pane tests**

```bash
go build ./internal/ui/panes/...
go test ./internal/ui/panes/... -run TestDevice -v
```

Expected: clean build, existing tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/panes/devices.go
git commit -m "feat(devices): show [restricted] and (no vol) capability indicators in device list"
```

---

## Task 11: ARCHITECTURE.md updates

**Files:**
- Modify: `docs/ARCHITECTURE.md`

Four targeted updates. Read each section before editing.

- [ ] **Step 1: Update routing table — playback key entry (priority 8)**

Find the "Overlay Routing Precedence" table. The row for priority 8 currently reads:

```
| 8 | Playback keys | `Space`, `n`, `+`, `-`, `s`, `r`, `v`, `←`, `→` → always NowPlayingPane |
```

Replace with (note: `n` was removed in Story 118, correct to `←`/`→`):

```
| 8 | Playback keys | Two pre-flight gates run before forwarding to NowPlayingPane:          |
|   |               | **Gate 1 — Premium:** `!store.IsPremium()` → "Spotify Premium required" |
|   |               | **Gate 2 — Capability:** `checkCapability(ps, action)` → reason string |
|   |               | If both pass: key forwarded to NowPlayingPane regardless of focus      |
|   |               | `v` (visualizer) is exempt from both gates — local UI action only      |
```

- [ ] **Step 2: Add "Capability Gating" subsection**

Find the "Staleness Tracking" subsection in the State Management section. Add a new subsection
directly after it:

```markdown
### Capability Gating

`checkCapability(ps *domain.PlaybackState, action panes.PlaybackAction) (bool, string)` in
`internal/app/capability.go` is the single decision point for operation capability checks.
It returns `(true, "")` when allowed and `(false, reason)` when blocked.

**Three signal sources, checked in order:**

1. `device.IsRestricted` — total lockout; no Web API commands accepted by this device.
2. `device.SupportsVolume` — volume-specific; `PUT /me/player/volume` fails on this device.
3. `PlaybackState.Actions.Disallows.*` — runtime signal from Spotify reflecting subscription
   tier, content type (ads, DRM, radio), and device capability in one object. Fields:
   `Pausing`, `Resuming`, `Seeking`, `SkippingNext`, `SkippingPrev`, `TogglingRepeatContext`,
   `TogglingRepeatTrack`, `TogglingShuffle`, `TransferringPlayback`.

**Two call sites:**

- `routing.go` `handleKeyMsg` — gate 2 for all keyboard-triggered playback actions.
- `handlers.go` `TransferPlaybackMsg` handler — inline check for
  `Actions.Disallows.TransferringPlayback` and `IsTargetDeviceRestricted(deviceID)`.

**Rule for future operations:** Any new playback operation that maps to a Spotify
`actions.disallows` field must be added to `checkCapability`'s switch statement. Never add
an ad-hoc capability check in routing or a handler.

`Store.IsTargetDeviceRestricted(deviceID string) bool` — used only for device transfer where
the *target* device's `IsRestricted` must be checked independently of current playback state.
```

- [ ] **Step 3: Update error handling table — 403 row**

Find the error handling table. The current 403 row reads:

```
| 403 (no premium) | `"warning"` | `Spotify Premium required for playback` |
```

Replace with:

```
| 403 (capability blocked) | `"warning"` | Proactively prevented by capability gate before the |
|                          |             | API call fires. If a 403 arrives anyway (race), the |
|                          |             | actual Spotify error body is shown; fallback is     |
|                          |             | "Spotify Premium required" when body is empty.      |
```

- [ ] **Step 4: Update domain types list**

Find the Domain Package section, the `types.go` bullet:

```
- `types.go` — Core types: `PlaybackState`, `Track`, `Artist`, `Album`, `Device`, ...
```

Add `PlaybackActions`, `PlaybackActionsWrapper` to the list.

- [ ] **Step 5: Build and lint**

```bash
make ci
```

Expected: PASS — lint, tests, 80% coverage, build.

- [ ] **Step 6: Commit**

```bash
git add docs/ARCHITECTURE.md
git commit -m "docs(architecture): document capability gating pattern, routing two-gate, error table, domain types"
```

---

## Task 12: Full CI verification

- [ ] **Step 1: Run full CI**

```bash
make ci
```

Expected: lint PASS, all tests PASS, coverage ≥ 80%, build PASS.

- [ ] **Step 2: Smoke test locally**

```bash
make run
```

Verify:
- Volume `+`/`-` updates bar immediately (no lag)
- Play/pause icon flips immediately on `Space`
- Shuffle icon flips immediately on `s`
- Repeat icon cycles immediately on `r`
- Repeat-one shows `↻¹` not `↻1`
- Device overlay shows `(no vol)` for phone/speaker devices

- [ ] **Step 3: Final commit (if any loose ends)**

If any files were missed, stage and commit now. Otherwise proceed to PR.

---

## Self-Review Notes

**Spec coverage check:**
- ✅ Domain additions (§3.1) — Task 1
- ✅ DeviceInfo + PlaybackRequestMsg extensions (§3.1) — Task 2
- ✅ `IsTargetDeviceRestricted` (§3.2) — Task 3
- ✅ `checkCapability` / `ActionAllowed` pattern (§3.2) — Task 4
- ✅ Routing gate (§3.3) — Task 5
- ✅ 403 message fix (§3.5) — Task 6a
- ✅ TransferPlaybackMsg gate (§3.4) — Task 6b
- ✅ DeviceInfo conversion fix (§3.1) — Tasks 6c + 7a
- ✅ `buildPlaybackAPICmd` TargetVolume (§3.6) — Task 7b
- ✅ Volume step 1% (§3.7) — Task 7c
- ✅ NowPlaying pending state + accumulation (§3.6) — Task 8
- ✅ Controls three-state + ↻¹ glyph (§3.8) — Task 9
- ✅ DeviceOverlay visual indicators (§3.9) — Task 10
- ✅ ARCHITECTURE.md updates (§5) — Task 11
