---
title: "Device Switcher"
description: "Lets users view all available Spotify Connect devices and transfer playback to any of them with a single keypress, with the active device always visible in the header bar."
status: done
stories: [07]
---

# Device Switcher

## Background

Spotnik is a terminal Spotify client for developers, and users often have multiple Spotify Connect devices available -- a laptop, phone, smart speaker, or TV. The Device Switcher feature lets users see all available devices at a glance and transfer playback between them without leaving the terminal.

The active device name is always displayed in the header bar so the user knows where audio is currently playing. Pressing `d` opens a floating overlay listing all available devices; the user navigates with `j`/`k` and confirms with `Enter` to transfer playback. The overlay follows the same compositing pattern established by the Search overlay (Feature 05).

Device data comes from two sources: the active device is extracted from the `GET /me/player` response that the playback polling loop (Feature 03) already fetches every second, while the full device list is fetched on-demand via `GET /me/player/devices` only when the overlay opens, avoiding unnecessary API calls.

---

## Story: Device Switcher (spec 07)

### Background

This story built the complete device switching flow: an API client for listing devices and transferring playback, a DeviceOverlay model rendered as a floating panel, a header bar indicator showing the active device at all times, and the transfer UX wiring through the root app model. The overlay pattern reuses the same approach as the search overlay -- root app renders it above a dimmed main view when `m.showDeviceSwitcher == true`.

### Acceptance Criteria
- [ ] Active device name visible in header bar at all times
- [ ] `d` opens device overlay with current devices loaded
- [ ] `Enter` transfers playback, header updates within 2 seconds
- [ ] `Esc` closes overlay without any change
- [ ] "No devices found" shown when Spotify returns empty list
- [ ] Device type icons render correctly (with `○` fallback for unsupported terminals)
- [ ] All API and pane handlers tested

### Implementation Context

**Store fields:**
```go
Devices        []api.Device  // all available playback devices
ActiveDevice   *api.Device   // currently active device (full struct, not just ID)
```

**Message types:**
```go
type devicesLoadedMsg      struct{ devices []api.Device }
type transferPlaybackMsg   struct{ deviceID string }
type deviceTransferredMsg  struct{ deviceID string }
```

**Data source:** The active device comes from the `GET /me/player` response (part of the playback state polling loop owned by Feature 03). The full device list is only fetched when the device overlay opens via `GET /me/player/devices`.

**Design tokens:** `theme.DeviceActive()`, `theme.SurfaceAlt()`, `theme.ActiveBorder()`, `theme.TextPrimary()`, `theme.TextMuted()`, `theme.SelectedBg()`, `theme.Success()`

**Device Switcher Overlay Layout:**
```
╭──────────────────────────────╮
│  DEVICES                     │
│  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄  │
│  ◉ MacBook Pro   [active]    │  ← DeviceActive() token bullet for active
│  ○ iPhone 14                 │  ← TextSecondary() token bullet for others
│  ○ Kitchen Speaker           │
│  ○ Living Room TV            │
╰──────────────────────────────╯
```

- Overlay positioned in top-right area (below device indicator)
- Width: 32 chars minimum, expands to fit longest device name + 4 padding
- `[active]` label: `Success()` token
- Active device: `◉` in `DeviceActive()` token
- Other devices: `○` in `InactiveBorder()` token
- Active device is selectable but pressing Enter on it shows status message: "Already playing on this device". Cursor does NOT skip over active device.

**Header Bar -- Device Indicator:**
```
│  Spotnik                                              ◉ MacBook Pro Speakers   │
```

- Right-aligned
- `◉` in `DeviceActive()` token when active
- `○` in `TextMuted()` token when no device / nothing playing
- Max 25 chars for device name (truncate with `…` if longer)
- Updates on every playback state poll

**Device Types:**

| Type | Prefix |
|---|---|
| `Computer` | `⊡` |
| `Smartphone` | `⊞` |
| `Speaker` | `⊟` |
| `TV` | `⊠` |
| Other | `○` |

If unicode icons cause rendering issues, fall back to plain `○` for all types.

**API Usage:**
- **Load devices**: `GET /me/player/devices`
- **Transfer playback**: `PUT /me/player` with `{ "device_ids": ["{id}"], "play": true }`
- **Refresh**: fetch devices when overlay opens (not on every tick -- too expensive)

**Keymap (Device Overlay):**

| Key | Action |
|---|---|
| `j` / `↓` | Move selection down |
| `k` / `↑` | Move selection up |
| `Enter` | Transfer playback to selected device |
| `Esc` | Close overlay |

### Tasks

1. **Device API calls** — Implement the Spotify API client methods for listing available devices and transferring playback. The `Device` struct must capture id, name, type, is_active, and volume_percent.
   - Files: `internal/api/devices.go`, `internal/api/devices_test.go`
   - Steps:
     - [ ] Define `Device` struct: id, name, type, is_active, volume_percent
     - [ ] `GetDevices(ctx) ([]Device, error)` — calls `GET /me/player/devices`, parses response
     - [ ] `TransferPlayback(ctx context.Context, deviceID string, play bool) error` — sends `PUT /me/player` with `{ "device_ids": ["{id}"], "play": <bool> }`
     - [ ] Test with fixture JSON via `httptest.NewServer`
   - Acceptance criteria:
     - `GetDevices` returns a correctly parsed slice of `Device` from the API response
     - `TransferPlayback` sends the correct PUT body including both `device_ids` and `play` fields
     - All errors are wrapped with context (`fmt.Errorf`)
   - Tests:
     - `TestGetDevices_Success` — returns parsed device list
     - `TestGetDevices_Empty` — returns empty slice, no error
     - `TestGetDevices_ServerError` — returns descriptive error
     - `TestTransferPlayback_Success` — sends correct PUT body with device_ids and play
     - `TestTransferPlayback_Error` — returns error with context
     - `TestDevice_Unmarshal` — parses Device JSON (id, name, type, is_active, volume_percent)

2. **DeviceOverlay model** — Implement the `DeviceOverlay` as a `tea.Model`. It fetches devices on init, renders them as a navigable list, and dispatches a transfer command when the user selects a device. The active device is selectable but pressing Enter on it shows a status message instead of triggering a transfer.
   - Files: `internal/ui/panes/devices.go`, `internal/ui/panes/devices_test.go`
   - Steps:
     - [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
     - [ ] `Init()` returns `fetchDevices` command
     - [ ] Navigate list with j/k, select with Enter
     - [ ] Active device marked with `◉` in `DeviceActive()` token and `[active]` label in `Success()` token
     - [ ] Other devices marked with `○` in `InactiveBorder()` token
     - [ ] Pressing Enter on active device shows "Already playing on this device" status message
     - [ ] Empty list renders "No devices found" message
     - [ ] Esc closes overlay by returning a close message
   - Acceptance criteria:
     - Overlay renders device list with correct symbols and theme tokens
     - j/k navigation moves cursor up and down through all devices (including active)
     - Enter on inactive device returns a transfer command
     - Enter on active device shows "Already playing on this device"
     - Esc returns a close message without side effects
     - Empty device list shows "No devices found"
   - Tests:
     - `TestDeviceOverlay_Init_FetchesDevices` — returns fetchDevices command
     - `TestDeviceOverlay_View_DeviceList` — renders all devices with correct symbols
     - `TestDeviceOverlay_View_ActiveDevice` — active device shows ◉ and [active] label
     - `TestDeviceOverlay_View_EmptyList` — shows "No devices found" message
     - `TestDeviceOverlay_Update_J_MovesDown` — cursor moves down
     - `TestDeviceOverlay_Update_K_MovesUp` — cursor moves up
     - `TestDeviceOverlay_Update_Enter_TransfersPlayback` — returns transfer command
     - `TestDeviceOverlay_Update_Enter_OnActiveDevice` — shows "already playing" message
     - `TestDeviceOverlay_Update_Esc` — returns close message

3. **Header device indicator** — Extend the header bar component to display the active device name, right-aligned. The indicator updates on every `playbackStateFetchedMsg` by reading `ActiveDevice` from the store. Long device names are truncated.
   - Files: `internal/ui/components/header.go` (or wherever header lives)
   - Steps:
     - [ ] Extend header bar component to show active device
     - [ ] `◉` in `DeviceActive()` token + device name when a device is active
     - [ ] `○` in `TextMuted()` token + "No active device" when no device is present
     - [ ] Update on every `playbackStateFetchedMsg`
     - [ ] Truncate device names longer than 25 characters with `…`
   - Acceptance criteria:
     - Active device name and `◉` symbol visible in header at all times
     - No device state shows `○` + "No active device"
     - Names longer than 25 characters are truncated with `…`
     - Indicator updates reactively from store on each playback poll
   - Tests:
     - `TestHeaderDeviceIndicator_ActiveDevice` — shows ◉ + device name
     - `TestHeaderDeviceIndicator_NoDevice` — shows ○ + "No active device"
     - `TestHeaderDeviceIndicator_LongName` — truncates to 25 chars with …

4. **Transfer UX** — Wire up the full transfer flow: pressing Enter fires a transfer command, the overlay closes, the status bar shows a transitional message, and the next playback poll updates the header with the new active device.
   - Files: `internal/app/app.go` (root model routing), `internal/ui/panes/devices.go`, `internal/ui/components/statusbar.go`
   - Steps:
     - [ ] On Enter: fire transfer command, close overlay
     - [ ] Show status bar: "Switching to {device name}..."
     - [ ] After next poll: status clears, header updates with new device
     - [ ] Handle transfer errors with status bar error message
   - Acceptance criteria:
     - Transfer command fires and overlay closes on Enter
     - Status bar shows "Switching to {device name}..." during transfer
     - Header updates with the new active device after the next playback poll
     - Transfer errors display in the status bar
   - Tests:
     - `TestApp_DeviceTransfer_UpdatesHeader` — transfer command -> deviceTransferredMsg -> next poll shows new device in header
     - `TestApp_DKeyOpensOverlay` — d key opens device overlay, background dimmed
     - `TestApp_DeviceOverlay_EscCloses` — Esc closes overlay, restores previous focus

### Files Created

| File | Purpose |
|---|---|
| `internal/api/devices.go` | Device list + transfer API calls |
| `internal/api/devices_test.go` | Tests with mock server |
| `internal/ui/panes/devices.go` | DeviceOverlay model |
| `internal/ui/panes/devices_test.go` | Update tests |
