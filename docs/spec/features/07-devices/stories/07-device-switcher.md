---
title: "Device Switcher"
feature: 07-devices
status: done
---

## Background
This story built the complete device switching flow: an API client for listing devices and transferring playback, a DeviceOverlay model rendered as a floating panel, a header bar indicator showing the active device at all times, and the transfer UX wiring through the root app model. The overlay pattern reuses the same approach as the search overlay -- root app renders it above a dimmed main view when `m.showDeviceSwitcher == true`.

## Design

### Store fields
```go
Devices        []api.Device  // all available playback devices
ActiveDevice   *api.Device   // currently active device
```

### Message types
```go
type devicesLoadedMsg      struct{ devices []api.Device }
type transferPlaybackMsg   struct{ deviceID string }
type deviceTransferredMsg  struct{ deviceID string }
```

### Design tokens
`theme.DeviceActive()`, `theme.SurfaceAlt()`, `theme.ActiveBorder()`, `theme.TextPrimary()`, `theme.TextMuted()`, `theme.SelectedBg()`, `theme.Success()`

### Device Switcher Overlay Layout
```
+------------------------------+
|  DEVICES                     |
|  .........................    |
|  * MacBook Pro   [active]    |  <- DeviceActive() token bullet
|  o iPhone 14                 |  <- TextSecondary() token bullet
|  o Kitchen Speaker           |
|  o Living Room TV            |
+------------------------------+
```

- Overlay positioned in top-right area (below device indicator)
- Width: 32 chars minimum, expands to fit longest device name + 4 padding
- `[active]` label: `Success()` token
- Active device: bullet in `DeviceActive()` token
- Other devices: bullet in `InactiveBorder()` token
- Active device is selectable but Enter shows "Already playing on this device"

### Header Bar -- Device Indicator
```
|  Spotnik                                              * MacBook Pro Speakers   |
```

- Right-aligned
- `*` in `DeviceActive()` token when active
- `o` in `TextMuted()` token when no device
- Max 25 chars for device name (truncate with `...` if longer)
- Updates on every playback state poll

### Device Types

| Type | Prefix |
|---|---|
| `Computer` | Computer icon |
| `Smartphone` | Phone icon |
| `Speaker` | Speaker icon |
| `TV` | TV icon |
| Other | `o` |

### API Usage
- **Load devices**: `GET /me/player/devices`
- **Transfer playback**: `PUT /me/player` with `{ "device_ids": ["{id}"], "play": true }`
- **Refresh**: fetch devices when overlay opens (not on every tick)

### Keymap (Device Overlay)

| Key | Action |
|---|---|
| `j` / `down` | Move selection down |
| `k` / `up` | Move selection up |
| `Enter` | Transfer playback to selected device |
| `Esc` | Close overlay |

### Files Created

| File | Purpose |
|---|---|
| `internal/api/devices.go` | Device list + transfer API calls |
| `internal/api/devices_test.go` | Tests with mock server |
| `internal/ui/panes/devices.go` | DeviceOverlay model |
| `internal/ui/panes/devices_test.go` | Update tests |

## Acceptance Criteria
- [ ] Active device name visible in header bar at all times
- [ ] `d` opens device overlay with current devices loaded
- [ ] `Enter` transfers playback, header updates within 2 seconds
- [ ] `Esc` closes overlay without any change
- [ ] "No devices found" shown when Spotify returns empty list
- [ ] Device type icons render correctly (with fallback for unsupported terminals)
- [ ] All API and pane handlers tested

## Tasks
- [ ] Device API calls -- Implement GetDevices and TransferPlayback methods
      - test: `TestGetDevices_Success`, `TestGetDevices_Empty`, `TestGetDevices_ServerError`, `TestTransferPlayback_Success`, `TestTransferPlayback_Error`, `TestDevice_Unmarshal`
- [ ] DeviceOverlay model -- Implement floating overlay with device list and navigation
      - test: `TestDeviceOverlay_Init_FetchesDevices`, `TestDeviceOverlay_View_DeviceList`, `TestDeviceOverlay_View_ActiveDevice`, `TestDeviceOverlay_View_EmptyList`, `TestDeviceOverlay_Update_J_MovesDown`, `TestDeviceOverlay_Update_K_MovesUp`, `TestDeviceOverlay_Update_Enter_TransfersPlayback`, `TestDeviceOverlay_Update_Enter_OnActiveDevice`, `TestDeviceOverlay_Update_Esc`
- [ ] Header device indicator -- Extend header bar with active device display
      - test: `TestHeaderDeviceIndicator_ActiveDevice`, `TestHeaderDeviceIndicator_NoDevice`, `TestHeaderDeviceIndicator_LongName`
- [ ] Transfer UX -- Wire transfer flow with status bar feedback
      - test: `TestApp_DeviceTransfer_UpdatesHeader`, `TestApp_DKeyOpensOverlay`, `TestApp_DeviceOverlay_EscCloses`
