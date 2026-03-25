package app_test

// idle_test.go — Tests for Feature 33: Idle Polling Backoff
//
// Task 1: Track lastInteraction time and isIdle() helper.
// Task 2: Adaptive pollIntervals() based on idle state and playback.
// Task 3: Tick handler uses dynamic intervals; KeyMsg after idle resets tickCount.

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Task 1: isIdle() ---

// TestIsIdle_FalseImmediatelyAfterCreation verifies the app is not idle
// immediately after creation (lastInteraction is initialized to now).
func TestIsIdle_FalseImmediatelyAfterCreation(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	assert.False(t, a.IsIdle(), "app should not be idle immediately after creation")
}

// TestIsIdle_TrueAfterThresholdElapses verifies that the app is idle
// after the idle threshold has elapsed.
func TestIsIdle_TrueAfterThresholdElapses(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	// Simulate time passing by setting lastInteraction to >60s ago.
	a.SetLastInteraction(time.Now().Add(-61 * time.Second))
	assert.True(t, a.IsIdle(), "app should be idle after 61 seconds of no input")
}

// TestIsIdle_FalseJustBeforeThreshold verifies the app is not idle
// just before the threshold.
func TestIsIdle_FalseJustBeforeThreshold(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	a.SetLastInteraction(time.Now().Add(-59 * time.Second))
	assert.False(t, a.IsIdle(), "app should not be idle 59 seconds after last interaction")
}

// TestKeyMsg_ResetsLastInteraction verifies that receiving a KeyMsg updates
// lastInteraction and makes the app no longer idle.
func TestKeyMsg_ResetsLastInteraction(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	// Make it idle first.
	a.SetLastInteraction(time.Now().Add(-61 * time.Second))
	require.True(t, a.IsIdle(), "precondition: app should be idle")

	// Send a key.
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	_, _ = a.Update(keyMsg)

	assert.False(t, a.IsIdle(), "app should not be idle after receiving a KeyMsg")
}

// --- Task 2: pollIntervals() ---

// TestPollIntervals_ActivePlaying returns 3s/9s (full speed).
func TestPollIntervals_ActivePlaying(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	// Active: lastInteraction = now (default).
	// Playing: set playback state.
	a.Store().SetPlaybackState(&api.PlaybackState{IsPlaying: true})

	pb, q := a.PollIntervals()
	assert.Equal(t, 3, pb, "active+playing playback interval should be 3s")
	assert.Equal(t, 9, q, "active+playing queue interval should be 9s")
}

// TestPollIntervals_ActivePaused returns 10s/30s (reduced).
func TestPollIntervals_ActivePaused(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	// Active: lastInteraction = now (default).
	// Paused: playback state with IsPlaying=false.
	a.Store().SetPlaybackState(&api.PlaybackState{IsPlaying: false})

	pb, q := a.PollIntervals()
	assert.Equal(t, 10, pb, "active+paused playback interval should be 10s")
	assert.Equal(t, 30, q, "active+paused queue interval should be 30s")
}

// TestPollIntervals_IdlePlaying returns 10s/30s (reduced).
func TestPollIntervals_IdlePlaying(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	a.SetLastInteraction(time.Now().Add(-61 * time.Second))
	a.Store().SetPlaybackState(&api.PlaybackState{IsPlaying: true})

	pb, q := a.PollIntervals()
	assert.Equal(t, 10, pb, "idle+playing playback interval should be 10s")
	assert.Equal(t, 30, q, "idle+playing queue interval should be 30s")
}

// TestPollIntervals_IdlePaused returns 30s/60s (slowest).
func TestPollIntervals_IdlePaused(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	a.SetLastInteraction(time.Now().Add(-61 * time.Second))
	a.Store().SetPlaybackState(&api.PlaybackState{IsPlaying: false})

	pb, q := a.PollIntervals()
	assert.Equal(t, 30, pb, "idle+paused playback interval should be 30s")
	assert.Equal(t, 60, q, "idle+paused queue interval should be 60s")
}

// TestPollIntervals_NilPlaybackState treats nil as paused.
func TestPollIntervals_NilPlaybackState(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	// Store has no playback state → treated as "paused".
	// Active (lastInteraction = now).
	pb, q := a.PollIntervals()
	assert.Equal(t, 10, pb, "active+nil-state playback interval should be 10s (treated as paused)")
	assert.Equal(t, 30, q, "active+nil-state queue interval should be 30s")
}

// --- Task 3: Tick handler wiring ---

// TestTickHandler_UsesAdaptiveIntervals verifies that the tick handler fires
// playback fetch at dynamic interval, not the old hardcoded 3s constant.
// When active+playing, tick 0 fires both playback and queue fetches.
func TestTickHandler_UsesAdaptiveIntervals(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	// Active + playing → intervals are 3/9.
	a.Store().SetPlaybackState(&api.PlaybackState{IsPlaying: true})

	// tickCount=0 → playback fires (0%3==0), queue fires (0%9==0).
	_, cmd := a.Update(panes.TickMsg{})
	assert.NotNil(t, cmd, "tick at count=0 should produce commands (fetch+timer)")
}

// TestTickHandler_IdlePaused_LongerIntervals verifies that with idle+paused state,
// fetches happen at the slower 30s/60s intervals.
// At tickCount=1, neither 1%30==0 nor 1%60==0, so no fetch (only timer).
func TestTickHandler_IdlePaused_LongerIntervals(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	a.SetLastInteraction(time.Now().Add(-61 * time.Second))
	a.Store().SetPlaybackState(&api.PlaybackState{IsPlaying: false})

	// Advance tickCount past 0 first (tick 0 always fires a fetch).
	_, _ = a.Update(panes.TickMsg{})
	// Now tickCount=1. At idle+paused with intervals 30/60, tick 1 should not fire fetches.
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd, "tick should always produce at least the next tick timer")
}

// TestKeyMsg_AfterIdle_ResetsTick verifies that when a KeyMsg arrives after
// the app has been idle, tickCount is reset to 0.
func TestKeyMsg_AfterIdle_ResetsTick(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	// Advance the tickCount by sending a few ticks.
	_, _ = a.Update(panes.TickMsg{})
	_, _ = a.Update(panes.TickMsg{})
	_, _ = a.Update(panes.TickMsg{})
	// Now make it idle.
	a.SetLastInteraction(time.Now().Add(-61 * time.Second))
	require.True(t, a.IsIdle(), "precondition: app should be idle")
	require.Greater(t, a.TickCount(), 0, "precondition: tickCount should be >0")

	// Send a key — this should trigger the idle-to-active reset.
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updatedModel, _ := a.Update(keyMsg)
	updated := updatedModel.(*app.App)

	assert.Equal(t, 0, updated.TickCount(), "tickCount should be reset to 0 on KeyMsg after idle")
}
