---
title: "Wire Preset & Visualizer Preference Persistence"
feature: 17-bootstrap
status: done
---

## Background

Story 76 adds `preset` and `visualizer` fields to the config. Story 79 builds the
PreferenceStore engine. This story wires everything together: loading saved preferences
at startup, emitting messages when preferences change, and persisting changes through
the PreferenceStore.

Two preferences are wired:
1. **Layout preset** (Page A) — cycled via `p` key in `routing.go`
2. **Visualizer pattern** — cycled via `v` key in `NowPlayingPane`

## Design

### Loading Saved Preferences at Startup

In `app.New()`, after creating the layout manager and viz engine, apply saved values:

```go
// Apply saved layout preset (Page A only).
// SetPreset is a no-op for out-of-range indices, so no extra validation needed.
if cfg.Preferences.Preset > 0 {
    a.layout.SetPreset(cfg.Preferences.Preset)
    a.propagateSizes()
    a.syncFocus()
}

// Apply saved visualizer pattern.
// SetPattern wraps with modulo, so out-of-range values are safe.
if cfg.Preferences.Visualizer > 0 {
    a.nowPlayingPane.SetVisualizerPattern(cfg.Preferences.Visualizer)
}
```

### Visualizer: New SetPattern Method

The viz engine currently only has `CyclePattern()`. We need a setter for loading
a saved index:

```go
// internal/ui/components/viz/engine.go

// SetPattern sets the active pattern to the given index.
// If index is out of range, it wraps with modulo (same as CyclePattern).
// Regenerates frames if the engine has been sized.
func (e *Engine) SetPattern(index int) {
    if len(e.patterns) == 0 {
        return
    }
    e.patternIdx = index % len(e.patterns)
    if e.patternIdx < 0 {
        e.patternIdx = 0
    }
    e.frameIdx = 0
    if e.width > 0 && e.height > 0 {
        e.frames = e.generateFrames()
    }
}
```

### NowPlayingPane: Expose Visualizer Control

The NowPlayingPane needs a method for the app to set the pattern on startup,
and it needs to emit a message when the user cycles the pattern so the app can
persist it:

```go
// internal/ui/panes/nowplaying.go

// SetVisualizerPattern sets the visualizer to a specific pattern index.
// Used at startup to restore the saved preference.
func (p *NowPlayingPane) SetVisualizerPattern(index int) {
    p.engine.SetPattern(index)
}

// VisualizerPatternChangedMsg is emitted when the user cycles the visualizer
// pattern via the 'v' key. The root app handles this to persist the preference.
type VisualizerPatternChangedMsg struct {
    PatternIndex int
}
```

Update the `v` key handler to emit the message:

```go
// Before:
case msg.Type == tea.KeyRunes && string(msg.Runes) == "v":
    p.engine.CyclePattern()
    return p, nil

// After:
case msg.Type == tea.KeyRunes && string(msg.Runes) == "v":
    p.engine.CyclePattern()
    return p, func() tea.Msg {
        return VisualizerPatternChangedMsg{PatternIndex: p.engine.Pattern()}
    }
```

### Preset Cycling: Emit Message for Persistence

Currently preset cycling in `routing.go` is fire-and-forget:

```go
// Before:
if m.Type == tea.KeyRunes && string(m.Runes) == "p" {
    a.layout.CyclePreset()
    a.propagateSizes()
    a.syncFocus()
    return a, nil
}

// After:
if m.Type == tea.KeyRunes && string(m.Runes) == "p" {
    a.layout.CyclePreset()
    a.propagateSizes()
    a.syncFocus()
    a.prefs.Set("preset", a.layout.ActivePresetIndex())
    return a, a.schedulePrefsFlush()
}
```

### Visualizer Message Handling in app.go

```go
// In Update():
case panes.VisualizerPatternChangedMsg:
    a.prefs.Set("visualizer", msg.PatternIndex)
    return a, a.schedulePrefsFlush()
```

### Coalescing Example

User rapidly presses `p` three times and `v` once within 500ms:

```
p pressed → prefs.Set("preset", 1), gen=1, timer #1 starts
p pressed → prefs.Set("preset", 2), gen=2, timer #2 starts
v pressed → prefs.Set("visualizer", 3), gen=3, timer #3 starts
p pressed → prefs.Set("preset", 0), gen=4, timer #4 starts

  ...500ms...

Timer #1 fires → gen 1 ≠ 4 → ignored
Timer #2 fires → gen 2 ≠ 4 → ignored
Timer #3 fires → gen 3 ≠ 4 → ignored
Timer #4 fires → gen 4 = 4 → FlushCmd()
    → writes: preset=0, visualizer=3 (both from pending map, one disk write)
```

### Validation at Boundaries

The validation chain for persisted preferences:

| Stage | What happens |
|---|---|
| **config.Load()** (story 76) | Clamps negatives to 0, unknown theme to `"black"` |
| **layout.SetPreset()** | No-op if index >= len(presets) — falls back to current preset |
| **viz.SetPattern()** | Wraps with modulo — always produces a valid index |
| **prefs.writeToDisk()** | Writes whatever is in pending — values come from runtime state, already valid |

Invalid config values from manual edits are caught at load time. Runtime values
are always valid because they come from `ActivePresetIndex()` and `Pattern()`.

### Files Changed

| Action | File | Purpose |
|---|---|---|
| Modify | `internal/ui/components/viz/engine.go` | Add `SetPattern(index int)` method |
| Modify | `internal/ui/components/viz/engine_test.go` | Test SetPattern |
| Modify | `internal/ui/panes/nowplaying.go` | Add `SetVisualizerPattern()`, `VisualizerPatternChangedMsg`; update `v` key handler to emit message |
| Modify | `internal/ui/panes/nowplaying_test.go` | Test SetVisualizerPattern, test v key emits message |
| Modify | `internal/app/app.go` | Apply saved preset and visualizer in `New()`; handle `VisualizerPatternChangedMsg` |
| Modify | `internal/app/routing.go` | Add `prefs.Set("preset", ...)` + `schedulePrefsFlush()` to preset cycle handler |
| Modify | `internal/app/app_test.go` | Test startup with saved preferences, test persistence wiring |

## Acceptance Criteria

- [ ] App loads `preset` from config and applies it to layout manager at startup
- [ ] App loads `visualizer` from config and applies it to viz engine at startup
- [ ] Out-of-range `preset` in config (e.g. 99) gracefully falls back (no crash)
- [ ] Out-of-range `visualizer` in config (e.g. 99) wraps to valid index (no crash)
- [ ] Pressing `p` persists new preset index via PreferenceStore
- [ ] Pressing `v` persists new visualizer index via PreferenceStore
- [ ] Rapid `p` presses debounce to a single disk write
- [ ] Saved preset survives app restart (start → cycle preset → restart → same preset)
- [ ] Saved visualizer survives app restart (start → cycle viz → restart → same pattern)
- [ ] `make ci` passes

## Tasks

- [ ] Add `SetPattern(index int)` to `internal/ui/components/viz/engine.go`. Wraps with modulo, regenerates frames if sized.
      - test: `TestSetPattern_ValidIndex`, `TestSetPattern_OutOfRange_Wraps`, `TestSetPattern_Negative_ClampsToZero`, `TestSetPattern_RegeneratesFrames`
- [ ] Add `SetVisualizerPattern(index int)` to `internal/ui/panes/nowplaying.go`. Delegates to `engine.SetPattern()`.
      - test: `TestNowPlayingPane_SetVisualizerPattern`
- [ ] Add `VisualizerPatternChangedMsg` type to `nowplaying.go`. Update `v` key handler to emit the message after calling `CyclePattern()`.
      - test: `TestNowPlayingPane_VKey_EmitsVisualizerChangedMsg`
- [ ] In `app.New()`, apply `cfg.Preferences.Preset` via `layout.SetPreset()` and `cfg.Preferences.Visualizer` via `nowPlayingPane.SetVisualizerPattern()` after initial setup.
      - test: `TestAppNew_AppliesSavedPreset`, `TestAppNew_AppliesSavedVisualizer`, `TestAppNew_InvalidPreset_NoOp`, `TestAppNew_InvalidVisualizer_Wraps`
- [ ] Handle `VisualizerPatternChangedMsg` in `app.go Update()`: call `prefs.Set("visualizer", msg.PatternIndex)` + `schedulePrefsFlush()`.
      - test: `TestApp_VisualizerChanged_PersistsPreference`
- [ ] Update preset cycle handler in `routing.go`: add `prefs.Set("preset", a.layout.ActivePresetIndex())` + `schedulePrefsFlush()` after `CyclePreset()`.
      - test: `TestApp_PresetCycle_PersistsPreference`
- [ ] End-to-end round-trip test: set preferences, flush, reload config, verify values match.
      - test: `TestPreferenceRoundTrip_PresetAndVisualizer`
