---
title: "Fix Error Architecture"
feature: 10-error-resilience
status: done
---

## Background
Multiple features silently swallow API errors, showing empty states instead of error messages. This is a foundational fix -- all other bug fixes depend on it. The current pattern in `build*Cmd` functions: if API returns error, return a result message without writing to store and without propagating the error to the UI. Each pane then shows "No data" instead of "Error loading data."

Affected features: Devices (B10), Stats (B12), Playlists (B16), Search (B11/B16).

## Design

### 1. Reusable Error Renderer
Create `components/errorview.go` with a `RenderError(theme, message, retryHint)` function that returns a styled error box:

```
+------------------------------+
|  X Failed to load devices    |
|                              |
|  Press d to retry            |
+------------------------------+
```

- Uses `theme.Error()` for the X symbol
- Uses `theme.TextPrimary()` for the message
- Uses `theme.TextMuted()` for the retry hint
- Centered within the available width/height

### 2. Error State in Store
Add per-feature error fields to the Store:

```go
// Error state -- one per data-fetching feature
SearchError    error
StatsError     error
DevicesError   error
QueueError     error
LibraryError   error
PlaylistsError error
```

With getter/setter methods following existing patterns:
- `SetXxxError(err error)` -- called from `build*Cmd` on failure
- `GetXxxError() error` -- read by panes in `View()`
- `ClearXxxError()` -- called on successful retry

### 3. Pattern for build*Cmd Functions

```go
// On failure:
store.SetDevicesError(err)
return NewDevicesLoadedMsg(nil, err)

// On success:
store.ClearDevicesError()
store.SetDevices(devices)
return NewDevicesLoadedMsg(devices, nil)
```

### 4. Pattern for Pane View()

```go
func (p *DeviceOverlay) View() string {
    if err := p.store.GetDevicesError(); err != nil {
        return components.RenderError(p.theme, "Failed to load devices", "Press d to retry")
    }
    // ... normal rendering
}
```

## Acceptance Criteria
- [ ] Reusable `RenderError` component exists in `components/`
- [ ] Store has error fields for each data-fetching feature
- [ ] All `build*Cmd` functions set error state on failure
- [ ] All panes/overlays check error state and render error view
- [ ] Error auto-clears on successful retry
- [ ] Tests verify error state rendering for each pane

## Tasks
- [ ] Create reusable error renderer in internal/ui/components/errorview.go
      - test: RenderError produces styled error box with correct theme colors
- [ ] Add per-feature error fields to internal/state/store.go with getter/setter methods
      - test: SetXxxError/GetXxxError/ClearXxxError round-trip correctly
- [ ] Update all build*Cmd functions in internal/app/app.go to set error state on failure
      - test: API error -> store error field set -> pane renders error view
- [ ] Update all panes/overlays to check error state and render error view
      - test: Error auto-clears on successful retry
