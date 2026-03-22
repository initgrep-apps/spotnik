# Feature 07 — Device Switcher

> **Depends on:** Feature 03 (Playback) complete and committed.

## Implementation Context

### Store fields this feature uses
```go
Devices        []api.Device // all available playback devices
ActiveDeviceID string       // ID of the currently active device
```

### Overlay pattern (same as search)
Device switcher is a floating overlay. Root app renders it above the dimmed main view
when `m.showDeviceSwitcher == true`. Press `d` to open, `Esc` to close.

### Message types for this feature
```go
type devicesLoadedMsg     struct{ devices []api.Device }
type transferPlaybackMsg  struct{ deviceID string }
type devicesSwitchedMsg   struct{ deviceID string }
```

### Design tokens used in this feature
`theme.DeviceActive()` · `theme.SurfaceAlt()` · `theme.ActiveBorder()` ·
`theme.TextPrimary()` · `theme.TextMuted()` · `theme.SelectedBg()`

---

---

## Goal

Show all available Spotify Connect devices and allow the user to transfer playback to any
of them with a single keypress. The active device is always visible in the header bar.

---

## User Stories

- **As a user**, I see the active device name in the top-right of the header bar at all times.
- **As a user**, I press `d` to open the device switcher overlay.
- **As a user**, I see all available devices listed with their type.
- **As a user**, I press `Enter` to transfer playback to the selected device.
- **As a user**, I press `Esc` to close the overlay without changing devices.
- **As a user**, if there are no other devices, I see "No other devices available".

---

## Device Switcher Overlay Layout (from DESIGN.md)

```
╭──────────────────────────────╮
│  DEVICES                     │
│  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄  │
│  ◉ MacBook Pro   [active]    │  ← Teal bullet for active
│  ○ iPhone 14                 │  ← Subtext bullet for others
│  ○ Kitchen Speaker           │
│  ○ Living Room TV            │
╰──────────────────────────────╯
```

- Overlay positioned in top-right area (below device indicator)
- Width: 32 chars minimum, expands to fit longest device name + 4 padding
- `[active]` label: `Green` color
- Active device: `◉` in `Teal`
- Other devices: `○` in `Overlay1`

---

## Header Bar — Device Indicator

```
│  Spotnik                                              ◉ MacBook Pro Speakers   │
```

- Right-aligned
- `◉` in `Teal` when active
- `○` in `Overlay0` when no device / nothing playing
- Max 25 chars for device name (truncate with `…` if longer)
- Updates on every playback state poll

---

## Device Types

Spotify returns device types. Display with an icon prefix:

| Type | Prefix |
|---|---|
| `Computer` | `⊡` |
| `Smartphone` | `⊞` |
| `Speaker` | `⊟` |
| `TV` | `⊠` |
| Other | `○` |

> If unicode icons cause rendering issues, fall back to plain `○` for all types.

---

## API Usage

- **Load devices**: `GET /me/player/devices`
- **Transfer playback**: `PUT /me/player` with `{ "device_ids": ["{id}"], "play": true }`
- **Refresh**: fetch devices when overlay opens (not on every tick — too expensive)

---

## Keymap (Device Overlay)

| Key | Action |
|---|---|
| `j` / `↓` | Move selection down |
| `k` / `↑` | Move selection up |
| `Enter` | Transfer playback to selected device |
| `Esc` | Close overlay |

---

## Files to Create

| File | Purpose |
|---|---|
| `internal/api/devices.go` | Device list + transfer API calls |
| `internal/api/devices_test.go` | Tests with mock server |
| `internal/ui/panes/devices.go` | DeviceOverlay model |
| `internal/ui/panes/devices_test.go` | Update tests |

---

## Task Breakdown

### Task 6.1 — Device API calls
- [ ] `GetDevices(ctx) ([]Device, error)`
- [ ] `TransferPlayback(ctx, deviceID string, play bool) error`
- [ ] `Device` struct: id, name, type, is_active, volume_percent
- [ ] Test with fixture JSON

### Task 6.2 — DeviceOverlay model
- [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
- [ ] `Init()` returns `fetchDevices` command
- [ ] Navigate list with j/k, select with Enter
- [ ] Active device marked, non-selectable (already active)
- [ ] Test: navigation, selection, empty list

### Task 6.3 — Header device indicator
- [ ] Extend header bar component to show active device
- [ ] Update on every `playbackStateFetchedMsg`
- [ ] Truncate long names with `…`

### Task 6.4 — Transfer UX
- [ ] On Enter: fire transfer command, close overlay
- [ ] Show status bar: "Switching to {device name}..."
- [ ] After next poll: status clears, header updates with new device
- [ ] Test: transfer success and error

---

## Acceptance Criteria

- [ ] Active device name visible in header at all times
- [ ] `d` opens overlay with current devices loaded
- [ ] `Enter` transfers playback, header updates within 2 seconds
- [ ] `Esc` closes without any change
- [ ] "No devices found" shown when Spotify returns empty list
- [ ] All API and pane handlers tested

---

*Last updated: 2026-02-21*
