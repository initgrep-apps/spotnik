package app

// poll_internal_test.go — White-box unit tests for per-pane polling helpers.
// Must be package app (not app_test) because pollState and libraryIntervals
// are unexported types defined in app.go.

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/stretchr/testify/assert"
)

// --- calcBackoffTicks ---

// TestCalcBackoffTicks verifies the exponential backoff formula:
// min(5 * 2^(errorCount-1), 60).
func TestCalcBackoffTicks(t *testing.T) {
	tests := []struct {
		errorCount int
		want       int
	}{
		{1, 5},   // 5 * 2^0 = 5
		{2, 10},  // 5 * 2^1 = 10
		{3, 20},  // 5 * 2^2 = 20
		{4, 40},  // 5 * 2^3 = 40
		{5, 60},  // 5 * 2^4 = 80 → capped at 60
		{6, 60},  // 5 * 2^5 = 160 → capped at 60
		{10, 60}, // well past cap
	}
	for _, tt := range tests {
		got := calcBackoffTicks(tt.errorCount)
		assert.Equal(t, tt.want, got, "calcBackoffTicks(%d)", tt.errorCount)
	}
}

// --- libraryInterval ---

// newTestAppInternal creates a minimal App for internal (white-box) polling tests.
func newTestAppInternal() *App {
	cfg := &config.Config{}
	return New(cfg, AppOptions{})
}

// TestLibraryInterval_RetryMode verifies that when hasData is false, the interval
// is always 5 (retry mode) regardless of playback state or idle state.
func TestLibraryInterval_RetryMode(t *testing.T) {
	a := newTestAppInternal()
	p := &pollState{hasData: false}

	// Active + playing
	a.store.SetPlaybackState(&api.PlaybackState{IsPlaying: true})
	a.lastInteraction = time.Now()
	assert.Equal(t, 5, a.libraryInterval(p, playlistsIntervals), "retry mode: playing should return 5")

	// Idle + paused
	a.store.SetPlaybackState(&api.PlaybackState{IsPlaying: false})
	a.lastInteraction = time.Now().Add(-61 * time.Second)
	assert.Equal(t, 5, a.libraryInterval(p, playlistsIntervals), "retry mode: idle+paused should return 5")
}

// TestLibraryInterval_Playing verifies that when hasData is true and music is playing,
// the playing interval is returned regardless of idle state.
func TestLibraryInterval_Playing(t *testing.T) {
	a := newTestAppInternal()
	p := &pollState{hasData: true}

	a.store.SetPlaybackState(&api.PlaybackState{IsPlaying: true})
	a.lastInteraction = time.Now() // active
	got := a.libraryInterval(p, recentPlayedIntervals)
	assert.Equal(t, recentPlayedIntervals.playing, got, "active+playing should return playing interval")
}

// TestLibraryInterval_Paused verifies that when hasData is true, music is paused,
// and the user is active, the paused interval is returned.
func TestLibraryInterval_Paused(t *testing.T) {
	a := newTestAppInternal()
	p := &pollState{hasData: true}

	a.store.SetPlaybackState(&api.PlaybackState{IsPlaying: false})
	a.lastInteraction = time.Now() // active (not idle)
	got := a.libraryInterval(p, recentPlayedIntervals)
	assert.Equal(t, recentPlayedIntervals.paused, got, "active+paused should return paused interval")
}

// TestLibraryInterval_Idle_OnlyWhenPaused verifies that the idle interval is returned
// only when music is paused AND the user is idle.
func TestLibraryInterval_Idle_OnlyWhenPaused(t *testing.T) {
	a := newTestAppInternal()
	p := &pollState{hasData: true}

	a.store.SetPlaybackState(&api.PlaybackState{IsPlaying: false})
	a.lastInteraction = time.Now().Add(-61 * time.Second) // idle
	got := a.libraryInterval(p, recentPlayedIntervals)
	assert.Equal(t, recentPlayedIntervals.idle, got, "idle+paused should return idle interval")
}

// TestLibraryInterval_PlayingOverridesIdle verifies that when music is playing,
// the playing interval is returned even when the user is idle.
func TestLibraryInterval_PlayingOverridesIdle(t *testing.T) {
	a := newTestAppInternal()
	p := &pollState{hasData: true}

	a.store.SetPlaybackState(&api.PlaybackState{IsPlaying: true})
	a.lastInteraction = time.Now().Add(-61 * time.Second) // idle
	got := a.libraryInterval(p, recentPlayedIntervals)
	assert.Equal(t, recentPlayedIntervals.playing, got, "idle+playing should return playing interval (not idle)")
}

// TestLibraryInterval_Albums verifies album intervals with hasData=true.
func TestLibraryInterval_Albums(t *testing.T) {
	a := newTestAppInternal()
	p := &pollState{hasData: true}

	// playing
	a.store.SetPlaybackState(&api.PlaybackState{IsPlaying: true})
	a.lastInteraction = time.Now()
	assert.Equal(t, albumsIntervals.playing, a.libraryInterval(p, albumsIntervals), "albums: playing interval")

	// paused + active
	a.store.SetPlaybackState(&api.PlaybackState{IsPlaying: false})
	a.lastInteraction = time.Now()
	assert.Equal(t, albumsIntervals.paused, a.libraryInterval(p, albumsIntervals), "albums: active+paused interval")

	// paused + idle
	a.lastInteraction = time.Now().Add(-61 * time.Second)
	assert.Equal(t, albumsIntervals.idle, a.libraryInterval(p, albumsIntervals), "albums: idle+paused interval")
}

// TestLibraryInterval_Stats verifies that statsIntervals are uniform (3600/3600/3600).
func TestLibraryInterval_Stats(t *testing.T) {
	a := newTestAppInternal()
	p := &pollState{hasData: true}

	// playing
	a.store.SetPlaybackState(&api.PlaybackState{IsPlaying: true})
	a.lastInteraction = time.Now()
	assert.Equal(t, statsIntervals.playing, a.libraryInterval(p, statsIntervals), "stats: playing interval")

	// paused + active
	a.store.SetPlaybackState(&api.PlaybackState{IsPlaying: false})
	a.lastInteraction = time.Now()
	assert.Equal(t, statsIntervals.paused, a.libraryInterval(p, statsIntervals), "stats: paused interval")

	// idle + paused
	a.lastInteraction = time.Now().Add(-61 * time.Second)
	assert.Equal(t, statsIntervals.idle, a.libraryInterval(p, statsIntervals), "stats: idle interval")
}
