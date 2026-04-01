---
name: project_spotnik_feature80_complete
description: Story 80 (Wire Preset & Visualizer Persistence): SetPattern on Engine, VisualizerPatternChangedMsg, startup loading, persistence wiring
type: project
---

## Story 80 — Wire Preset & Visualizer Preference Persistence

**What was built:**
- `Engine.SetPattern(index int)` in `internal/ui/components/viz/engine.go` — wraps out-of-range with modulo, clamps negatives to 0, resets frameIdx, regenerates frames if sized
- `VisualizerPatternChangedMsg` struct in `nowplaying.go` — emitted by the `v` key handler after `CyclePattern()` so the app can persist the change
- `NowPlayingPane.SetVisualizerPattern(index int)` — delegates to `engine.SetPattern()`, used at startup
- `NowPlayingPane.VisualizerPattern() int` — read accessor for testing
- `App.New()` now applies `cfg.Preferences.Preset` (via `layout.SetPreset`) and `cfg.Preferences.Visualizer` (via `SetVisualizerPattern`) after constructing the App struct
- `handlePrefsMsg` extended with `VisualizerPatternChangedMsg` case: calls `prefs.Set("visualizer", ...) + schedulePrefsFlush()`
- `routing.go` 'p' key handler updated: calls `prefs.Set("preset", ...) + schedulePrefsFlush()` after `CyclePreset()`
- Exported `App.NowPlayingPane()` and `App.ActivePresetIndex()` test accessors

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/components/viz/engine.go` — SetPattern method
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/panes/nowplaying.go` — SetVisualizerPattern, VisualizerPattern, VisualizerPatternChangedMsg
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/app.go` — startup loading in New(), VisualizerPatternChangedMsg case in handlePrefsMsg
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/routing.go` — preset persistence on p key

**Patterns established:**
- `VisualizerPatternChangedMsg` follows the same `pane emits → app.handlePrefsMsg handles` pattern as `ThemeSwitchMsg`
- `handlePrefsMsg` is the single place where all preference persistence messages are handled
- Exported test accessors (`NowPlayingPane()`, `ActivePresetIndex()`) follow the `RequestFlowPane()` accessor pattern
- `propagateSizes()` + `syncFocus()` called in `New()` after `SetPreset` is safe even before `Resize()` — WindowSizeMsg corrects it

**Gotchas:**
- Go's `%` operator returns negative for negative operands, so `SetPattern` needs the explicit `if e.patternIdx < 0 { e.patternIdx = 0 }` guard after the modulo
- The existing test `TestNowPlayingPane_V_CyclesEnginePattern` expected nil cmd from `v` key — must update it when `v` starts emitting a Cmd
- `propagateSizes()` in `New()` before `Resize()` calls `SetSize(0,0)` on panes but this is harmless — WindowSizeMsg always corrects it
- The `panes` import was already in `app.go`, so `panes.VisualizerPatternChangedMsg` required no new import

**Testing notes:**
- Round-trip test `TestPreferenceRoundTrip_PresetAndVisualizer` uses `prefs.New(cfgPath)` directly, bypassing the app — clean integration test
- `TestAppNew_InvalidPreset_NoOp` verifies `layout.SetPreset` no-ops for index 99 (no crash, stays at 0)
- `TestAppNew_InvalidVisualizer_Wraps` uses `99 % 7 = 1` as the expected wrapped value
- Coverage: 87.0% total after story 80
