# Feature 13 — Fix Devices Error Handling

> **Bug fix:** Devices overlay shows "No devices found" even when actively playing.

## Root Cause

`buildFetchDevicesCmd()` in `app.go`: if `a.devices` is nil OR `GetDevices()` returns an error,
`NewDevicesLoadedMsg(devList, err)` is returned with nil `devList`. The overlay's Update handler
checks `if m.err == nil` but on error it leaves the device list empty and shows "No devices found"
— no error message is displayed.

**Information gap:** Agent didn't implement error display in device overlay.

---

## Fix

1. **Add error state rendering to DeviceOverlay**
   - If API error: show "Failed to load devices. Press d to retry."
   - If devices client is nil: show "Not connected"
   - Use error theme token for styling

2. **Surface the error from the message**
   - DeviceOverlay Update handler should check `err` field and store it for rendering
   - Error state clears on successful retry

---

## Files

- `internal/ui/panes/devices.go` — Error state field, error rendering
- `internal/app/app.go` — Ensure error is propagated in devices loaded message
- Tests for error state rendering

---

## Acceptance Criteria

- [ ] API error shows "Failed to load devices" in overlay, not "No devices found"
- [ ] Nil devices client shows "Not connected"
- [ ] Retry hint shown on error state
- [ ] Tests verify error state rendering
