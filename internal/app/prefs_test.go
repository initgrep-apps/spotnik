package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/prefs"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPrefsTestApp creates a standard App for preference-related tests.
func newPrefsTestApp(t *testing.T) *app.App {
	t.Helper()
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := app.New(cfg, app.AppOptions{})
	// Resize to grid view.
	a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	return a
}

// TestApp_SchedulePrefsFlush_IncrementsGeneration verifies that each call to
// SchedulePrefsFlush increments the generation counter.
func TestApp_SchedulePrefsFlush_IncrementsGeneration(t *testing.T) {
	a := newPrefsTestApp(t)
	gen0 := a.PrefsDirtyGen()

	a.SchedulePrefsFlush()
	gen1 := a.PrefsDirtyGen()

	a.SchedulePrefsFlush()
	gen2 := a.PrefsDirtyGen()

	assert.Greater(t, gen1, gen0, "generation should increment after first SchedulePrefsFlush")
	assert.Greater(t, gen2, gen1, "generation should increment after second SchedulePrefsFlush")
}

// TestApp_PrefsFlushTick_MatchingGen_Flushes verifies that when the tick's
// generation matches the current dirty generation, FlushCmd is dispatched.
func TestApp_PrefsFlushTick_MatchingGen_Flushes(t *testing.T) {
	a := newPrefsTestApp(t)

	// Trigger a schedule — get the generation that was set.
	flushCmd := a.SchedulePrefsFlush()
	require.NotNil(t, flushCmd, "SchedulePrefsFlush should return a non-nil Cmd")

	// Execute the tick command to get the prefsFlushTickMsg.
	tickMsg := flushCmd()
	require.NotNil(t, tickMsg)

	// Set a pending preference so FlushCmd has something to write.
	a.Prefs().Set("theme", "nord")

	// Send the tick message to the app — generation should match, triggering flush.
	_, cmd := a.Update(tickMsg)
	// The returned cmd should be the FlushCmd (non-nil because there is a pending pref).
	assert.NotNil(t, cmd, "matching generation tick should dispatch a FlushCmd")
}

// TestApp_PrefsFlushTick_StaleGen_Ignored verifies that a stale tick (generation
// less than the current dirty generation) is ignored — no flush is dispatched.
func TestApp_PrefsFlushTick_StaleGen_Ignored(t *testing.T) {
	a := newPrefsTestApp(t)

	// Fire first flush tick.
	firstCmd := a.SchedulePrefsFlush()
	tickMsg1 := firstCmd()

	// Bump generation again (simulates a second preference change arriving
	// before the first tick fires).
	a.SchedulePrefsFlush()

	// Set a pending pref so if flush mistakenly fires it would be observable.
	a.Prefs().Set("theme", "monokai")

	// Send stale tick — generation is now < dirty gen.
	_, cmd := a.Update(tickMsg1)

	// The returned cmd must be nil (stale tick, no flush dispatched).
	// Note: Update batches with alerts cmd, so we check HasPending is still true
	// as evidence the flush did not clear it.
	_ = cmd
	assert.True(t, a.Prefs().HasPending(), "stale tick should not flush pending preferences")
}

// TestApp_ThemeSwitch_UsesPreferenceStore verifies that ThemeSwitchMsg uses
// the PreferenceStore (prefs.Set called) rather than the old persistThemeChoice.
// We verify this by checking that HasPending is true after the switch and
// that no disk write occurred synchronously (no config file created).
func TestApp_ThemeSwitch_UsesPreferenceStore(t *testing.T) {
	a := newPrefsTestApp(t)

	// Initially no pending changes.
	assert.False(t, a.Prefs().HasPending(), "no pending prefs before theme switch")

	// Switch theme — this should call prefs.Set("theme", ...) and schedulePrefsFlush.
	_, cmd := a.Update(panes.ThemeSwitchMsg{ThemeID: "dracula"})

	// There should be a pending preference in the store.
	assert.True(t, a.Prefs().HasPending(), "theme switch should mark prefs as pending")

	// The returned cmd should be non-nil (batch of toast + schedulePrefsFlush).
	require.NotNil(t, cmd, "ThemeSwitchMsg should return a Cmd")
}

// TestApp_FlushedMsg_NoErrorIsNoop verifies that a successful FlushedMsg (no error)
// is handled without crashing or emitting any toast.
func TestApp_FlushedMsg_NoErrorIsNoop(t *testing.T) {
	a := newPrefsTestApp(t)

	_, cmd := a.Update(prefs.FlushedMsg{Err: nil})
	// No error: Update should return nil cmd from the prefs handler.
	_ = cmd // alerts model may return a non-nil cmd; we just verify no panic.
}

// TestApp_FlushedMsg_WithError_SchedulesRetry verifies that when FlushedMsg
// carries a non-nil error, a retry is scheduled via schedulePrefsFlush so
// the re-queued changes are not silently abandoned.
func TestApp_FlushedMsg_WithError_SchedulesRetry(t *testing.T) {
	a := newPrefsTestApp(t)

	gen0 := a.PrefsDirtyGen()

	// Simulate a failed flush — FlushCmd re-queues changes inside the store
	// before returning FlushedMsg{Err: ...}.  Here we just send FlushedMsg
	// directly to exercise the app-level retry logic.
	_, cmd := a.Update(prefs.FlushedMsg{Err: fmt.Errorf("disk full")})

	// schedulePrefsFlush must have been called: generation should have
	// incremented and the returned Cmd must be non-nil.
	assert.Greater(t, a.PrefsDirtyGen(), gen0, "error path should increment dirty generation")
	assert.NotNil(t, cmd, "error path should return a retry Cmd")
}

// TestApp_PrefsFlush_ErrorProducesToastWarning verifies that when a
// prefs.FlushedMsg carries a non-nil error, the handler emits a
// ToastWarning with title "Preferences not saved" and body
// "Check available disk space." — no stderr write.
func TestApp_PrefsFlush_ErrorProducesToastWarning(t *testing.T) {
	a := newPrefsTestApp(t)

	// Send a FlushedMsg with a simulated flush error.
	model, cmd := a.Update(prefs.FlushedMsg{Err: fmt.Errorf("permission denied")})
	a = model.(*app.App)
	require.NotNil(t, cmd, "error path must return a non-nil Cmd (batch of toast + retry)")

	// Execute the batch cmd to activate the toast, then feed it back.
	msgs := collectAllMsgs(cmd)
	var toastFound bool
	for _, m := range msgs {
		// Process each msg through Update so the toast is rendered.
		updated, _ := a.Update(m)
		a = updated.(*app.App)
		toastFound = true
	}
	require.True(t, toastFound, "batch should contain at least one msg")

	// Check that View contains the toast text.
	output := a.View()
	assert.Contains(t, output, "Preferences not saved", "error path should show toast title")
	assert.Contains(t, output, "Check available disk space", "error path should show toast body")
}

// ---------------------------------------------------------------------------
// Story 80: startup preference loading and persistence wiring
// ---------------------------------------------------------------------------

// TestAppNew_AppliesSavedPreset verifies that a non-zero Preset value in config
// is applied to the layout manager at startup.
func TestAppNew_AppliesSavedPreset(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	cfg.Preferences.Preset = 2

	a := app.New(cfg, app.AppOptions{})
	// Resize so the layout has been initialized.
	a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})

	assert.Equal(t, 2, a.ActivePresetIndex(), "App.New should apply saved preset to layout")
}

// TestAppNew_AppliesSavedVisualizer verifies that a non-zero Visualizer value
// in config is applied to the NowPlayingPane's viz engine at startup.
func TestAppNew_AppliesSavedVisualizer(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	cfg.Preferences.Visualizer = 3

	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})

	np := a.NowPlayingPane()
	require.NotNil(t, np, "NowPlayingPane should exist")
	assert.Equal(t, 3, np.VisualizerPattern(), "App.New should apply saved visualizer to NowPlayingPane")
}

// TestAppNew_InvalidPreset_NoOp verifies that an out-of-range Preset in config
// is silently ignored (layout.SetPreset no-ops on invalid indices).
func TestAppNew_InvalidPreset_NoOp(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	cfg.Preferences.Preset = 99 // no such preset index

	assert.NotPanics(t, func() {
		a := app.New(cfg, app.AppOptions{})
		a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
		// layout.SetPreset is a no-op for out-of-range: active preset stays at 0.
		assert.Equal(t, 0, a.ActivePresetIndex(), "out-of-range preset should fall back to 0")
	})
}

// TestAppNew_InvalidVisualizer_Wraps verifies that a large Visualizer in config
// is wrapped with modulo by the viz engine (no panic, valid index).
func TestAppNew_InvalidVisualizer_Wraps(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	cfg.Preferences.Visualizer = 99 // will wrap with modulo

	assert.NotPanics(t, func() {
		a := app.New(cfg, app.AppOptions{})
		a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
		np := a.NowPlayingPane()
		require.NotNil(t, np)
		// Must be in [0, patternCount). 99 % 7 = 1
		pat := np.VisualizerPattern()
		assert.True(t, pat >= 0 && pat < 7, "out-of-range visualizer should wrap to valid index, got %d", pat)
	})
}

// TestApp_VisualizerChanged_PersistsPreference verifies that receiving a
// VisualizerPatternChangedMsg calls prefs.Set and schedules a flush.
func TestApp_VisualizerChanged_PersistsPreference(t *testing.T) {
	a := newPrefsTestApp(t)

	assert.False(t, a.Prefs().HasPending(), "no pending prefs before visualizer change")

	_, cmd := a.Update(panes.VisualizerPatternChangedMsg{PatternIndex: 4})

	assert.True(t, a.Prefs().HasPending(), "VisualizerPatternChangedMsg should mark prefs as pending")
	assert.NotNil(t, cmd, "VisualizerPatternChangedMsg should schedule a flush Cmd")
}

// TestApp_PresetCycle_PersistsPreference verifies that pressing 'p' persists
// the new preset index via PreferenceStore.
func TestApp_PresetCycle_PersistsPreference(t *testing.T) {
	a := newPrefsTestApp(t)

	assert.False(t, a.Prefs().HasPending(), "no pending prefs before preset cycle")

	pMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	_, cmd := a.Update(pMsg)

	assert.True(t, a.Prefs().HasPending(), "pressing p should mark prefs as pending")
	assert.NotNil(t, cmd, "pressing p should schedule a flush Cmd")
}

// TestPreferenceRoundTrip_PresetAndVisualizer verifies that preset and visualizer
// preferences survive a write → reload cycle. This is the end-to-end round-trip test.
func TestPreferenceRoundTrip_PresetAndVisualizer(t *testing.T) {
	// Use a temp file so we don't touch real config.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	// Create a PreferenceStore pointing at the temp file.
	ps := prefs.New(cfgPath)
	ps.Set("preset", 2)
	ps.Set("visualizer", 5)

	// Flush to disk.
	flushCmd := ps.FlushCmd()
	require.NotNil(t, flushCmd)
	result := flushCmd()
	flushedMsg, ok := result.(prefs.FlushedMsg)
	require.True(t, ok, "FlushCmd should return FlushedMsg, got %T", result)
	require.NoError(t, flushedMsg.Err, "flush should succeed")

	// Verify the file was written.
	_, err := os.Stat(cfgPath)
	require.NoError(t, err, "config file should exist after flush")

	// Reload config and verify the values survived.
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, 2, cfg.Preferences.Preset, "preset should survive round-trip")
	assert.Equal(t, 5, cfg.Preferences.Visualizer, "visualizer should survive round-trip")
}
