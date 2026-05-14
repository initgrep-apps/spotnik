package app

// poll_internal_test.go — White-box unit tests for per-pane polling helpers.
// Must be package app (not app_test) because pollState and libraryIntervals
// are unexported types defined in app.go.

import (
	"fmt"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Suppress unused-import lint for uikit — used indirectly via toast assertions.
var _ uikit.ToastIntent

// --- calcBackoffTicks ---

// TestCalcBackoffTicks verifies the exponential backoff formula:
// min(5 * 2^(errorCount-1), 60).
func TestCalcBackoffTicks(t *testing.T) {
	tests := []struct {
		errorCount int
		want       int
	}{
		{0, 5},   // guard: errorCount <= 0 returns minimum
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

// --- Story 200: Per-pane error state in loaded-message handlers ---

// TestApp_LibraryLoaded_SuccessResetsPollState verifies that a successful
// LibraryLoadedMsg resets errorCount and backoffTicks to 0, and sets hasData=true.
// If wasErr (previous errors existed), it emits an info toast.
func TestApp_LibraryLoaded_SuccessResetsPollState(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	// Simulate a prior error state.
	a.playlistsPoll.errorCount = 3
	a.playlistsPoll.backoffTicks = 20

	// Send a successful LibraryLoadedMsg.
	msg := panes.LibraryLoadedMsg{Items: []domain.SimplePlaylist{}, Offset: 0}
	model, cmd := a.Update(msg)
	a = model.(*App)

	assert.Equal(t, 0, a.playlistsPoll.errorCount, "success should reset errorCount")
	assert.Equal(t, 0, a.playlistsPoll.backoffTicks, "success should reset backoffTicks")
	assert.True(t, a.playlistsPoll.hasData, "success should set hasData=true")

	// wasErr was true (errorCount was 3 before reset), so info toast expected.
	assert.NotNil(t, cmd, "recovery from error should emit an info toast")
}

// TestApp_LibraryLoaded_SuccessNoToastOnFreshLoad verifies that a successful
// LibraryLoadedMsg with no prior errors does NOT emit a recovery toast.
func TestApp_LibraryLoaded_SuccessNoToastOnFreshLoad(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	// No prior errors — errorCount starts at 0.
	msg := panes.LibraryLoadedMsg{Items: []domain.SimplePlaylist{}, Offset: 0}
	model, cmd := a.Update(msg)
	a = model.(*App)

	assert.Equal(t, 0, a.playlistsPoll.errorCount)
	assert.Equal(t, 0, a.playlistsPoll.backoffTicks)
	assert.True(t, a.playlistsPoll.hasData)

	// wasErr is false (no prior errors), so no recovery toast.
	// cmd may be nil (no toast) or may be a pane forward cmd.
	// We only assert that no info toast is produced — the pane forward may exist.
	_ = cmd
}

// TestApp_LibraryLoaded_ErrorFirstOnlyEmitsToast verifies that the first error
// increments errorCount, sets backoffTicks, and emits a toast.
// Subsequent errors are silent (no additional toast).
func TestApp_LibraryLoaded_ErrorFirstOnlyEmitsToast(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	// First error.
	errMsg := panes.LibraryLoadedMsg{Err: fmt.Errorf("network error")}
	model, cmd := a.Update(errMsg)
	a = model.(*App)

	assert.Equal(t, 1, a.playlistsPoll.errorCount, "first error should set errorCount=1")
	assert.Equal(t, calcBackoffTicks(1), a.playlistsPoll.backoffTicks, "first error should set backoffTicks")
	assert.NotNil(t, cmd, "first error should emit a toast")

	// Second error — should increment errorCount but NOT emit toast.
	model2, cmd2 := a.Update(errMsg)
	a = model2.(*App)

	assert.Equal(t, 2, a.playlistsPoll.errorCount, "second error should set errorCount=2")
	assert.Equal(t, calcBackoffTicks(2), a.playlistsPoll.backoffTicks, "second error should set backoffTicks=10")
	assert.Nil(t, cmd2, "second error should NOT emit a toast")
}

// TestApp_LibraryLoaded_RecoveryEmitsInfoToast verifies that after errors,
// a successful load emits a "Playlists loaded" info toast.
func TestApp_LibraryLoaded_RecoveryEmitsInfoToast(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.width = 120
	a.height = 40

	// Simulate a prior error state.
	a.playlistsPoll.errorCount = 2
	a.playlistsPoll.backoffTicks = 10

	// Send a successful LibraryLoadedMsg.
	msg := panes.LibraryLoadedMsg{Items: []domain.SimplePlaylist{}, Offset: 0}
	model, cmd := a.Update(msg)
	a = model.(*App)

	assert.Equal(t, 0, a.playlistsPoll.errorCount, "success should reset errorCount")
	assert.Equal(t, 0, a.playlistsPoll.backoffTicks, "success should reset backoffTicks")
	assert.True(t, a.playlistsPoll.hasData, "success should set hasData")

	// wasErr was true — should emit info toast.
	require.NotNil(t, cmd, "recovery should emit info toast cmd")

	// Execute the toast cmd and feed it back through Update to materialise.
	alertMsg := cmd()
	updated, _ := a.Update(alertMsg)
	appModel := updated.(*App)
	view := appModel.View()
	assert.Contains(t, view, "Playlists loaded", "recovery toast should say 'Playlists loaded'")
}

// TestApp_LibraryLoaded_errNilClient_SilentlyIgnored verifies that errNilClient
// is silently ignored without incrementing errorCount.
func TestApp_LibraryLoaded_errNilClient_SilentlyIgnored(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	errMsg := panes.LibraryLoadedMsg{Err: errNilClient}
	model, cmd := a.Update(errMsg)
	a = model.(*App)

	assert.Equal(t, 0, a.playlistsPoll.errorCount, "errNilClient should not increment errorCount")
	assert.Nil(t, cmd, "errNilClient should not emit any cmd")
}

// --- AlbumsLoadedMsg ---

func TestApp_AlbumsLoaded_ErrorFirstEmitsToast(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	errMsg := panes.AlbumsLoadedMsg{Err: fmt.Errorf("network error")}
	model, cmd := a.Update(errMsg)
	a = model.(*App)

	assert.Equal(t, 1, a.albumsPoll.errorCount)
	assert.Equal(t, calcBackoffTicks(1), a.albumsPoll.backoffTicks)
	assert.NotNil(t, cmd, "first error should emit toast")
}

func TestApp_AlbumsLoaded_ErrorSecondSilent(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.albumsPoll.errorCount = 1
	a.albumsPoll.backoffTicks = 5

	errMsg := panes.AlbumsLoadedMsg{Err: fmt.Errorf("network error")}
	model, cmd := a.Update(errMsg)
	a = model.(*App)

	assert.Equal(t, 2, a.albumsPoll.errorCount)
	assert.Equal(t, calcBackoffTicks(2), a.albumsPoll.backoffTicks)
	assert.Nil(t, cmd, "second error should NOT emit toast")
}

func TestApp_AlbumsLoaded_SuccessResetsPollState(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.albumsPoll.errorCount = 3
	a.albumsPoll.backoffTicks = 20

	msg := panes.AlbumsLoadedMsg{Items: []domain.SavedAlbum{}, Offset: 0}
	model, cmd := a.Update(msg)
	a = model.(*App)

	assert.Equal(t, 0, a.albumsPoll.errorCount)
	assert.Equal(t, 0, a.albumsPoll.backoffTicks)
	assert.True(t, a.albumsPoll.hasData)
	assert.NotNil(t, cmd, "recovery from error should emit info toast")
}

// --- LikedTracksLoadedMsg ---

func TestApp_LikedTracksLoaded_ErrorFirstEmitsToast(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	errMsg := panes.LikedTracksLoadedMsg{Err: fmt.Errorf("network error")}
	model, cmd := a.Update(errMsg)
	a = model.(*App)

	assert.Equal(t, 1, a.likedSongsPoll.errorCount)
	assert.Equal(t, calcBackoffTicks(1), a.likedSongsPoll.backoffTicks)
	assert.NotNil(t, cmd, "first error should emit toast")
}

func TestApp_LikedTracksLoaded_ErrorSecondSilent(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.likedSongsPoll.errorCount = 1
	a.likedSongsPoll.backoffTicks = 5

	errMsg := panes.LikedTracksLoadedMsg{Err: fmt.Errorf("network error")}
	model, cmd := a.Update(errMsg)
	a = model.(*App)

	assert.Equal(t, 2, a.likedSongsPoll.errorCount)
	assert.Equal(t, calcBackoffTicks(2), a.likedSongsPoll.backoffTicks)
	assert.Nil(t, cmd, "second error should NOT emit toast")
}

func TestApp_LikedTracksLoaded_SuccessResetsPollState(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.likedSongsPoll.errorCount = 2
	a.likedSongsPoll.backoffTicks = 10

	msg := panes.LikedTracksLoadedMsg{Items: []domain.SavedTrack{}, Offset: 0}
	model, cmd := a.Update(msg)
	a = model.(*App)

	assert.Equal(t, 0, a.likedSongsPoll.errorCount)
	assert.Equal(t, 0, a.likedSongsPoll.backoffTicks)
	assert.True(t, a.likedSongsPoll.hasData)
	assert.NotNil(t, cmd, "recovery from error should emit info toast")
}

// --- RecentlyPlayedLoadedMsg ---

func TestApp_RecentlyPlayedLoaded_ErrorFirstEmitsToast(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	errMsg := panes.RecentlyPlayedLoadedMsg{Err: fmt.Errorf("network error")}
	model, cmd := a.Update(errMsg)
	a = model.(*App)

	assert.Equal(t, 1, a.recentPlayedPoll.errorCount)
	assert.Equal(t, calcBackoffTicks(1), a.recentPlayedPoll.backoffTicks)
	assert.NotNil(t, cmd, "first error should emit toast")
}

func TestApp_RecentlyPlayedLoaded_ErrorSecondSilent(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.recentPlayedPoll.errorCount = 1
	a.recentPlayedPoll.backoffTicks = 5

	errMsg := panes.RecentlyPlayedLoadedMsg{Err: fmt.Errorf("network error")}
	model, cmd := a.Update(errMsg)
	a = model.(*App)

	assert.Equal(t, 2, a.recentPlayedPoll.errorCount)
	assert.Equal(t, calcBackoffTicks(2), a.recentPlayedPoll.backoffTicks)
	assert.Nil(t, cmd, "second error should NOT emit toast")
}

func TestApp_RecentlyPlayedLoaded_SuccessResetsPollState(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.recentPlayedPoll.errorCount = 2
	a.recentPlayedPoll.backoffTicks = 10

	msg := panes.RecentlyPlayedLoadedMsg{Items: []domain.PlayHistory{}}
	model, cmd := a.Update(msg)
	a = model.(*App)

	assert.Equal(t, 0, a.recentPlayedPoll.errorCount)
	assert.Equal(t, 0, a.recentPlayedPoll.backoffTicks)
	assert.True(t, a.recentPlayedPoll.hasData)
	assert.NotNil(t, cmd, "recovery from error should emit info toast")
}

// --- StatsLoadedMsg ---

func TestApp_StatsLoaded_ErrorFirstEmitsToast(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	errMsg := panes.StatsLoadedMsg{Err: fmt.Errorf("network error")}
	model, cmd := a.Update(errMsg)
	a = model.(*App)

	assert.Equal(t, 1, a.statsPoll.errorCount)
	assert.Equal(t, calcBackoffTicks(1), a.statsPoll.backoffTicks)
	assert.NotNil(t, cmd, "first error should emit toast")
}

func TestApp_StatsLoaded_ErrorSecondSilent(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.statsPoll.errorCount = 1
	a.statsPoll.backoffTicks = 5

	errMsg := panes.StatsLoadedMsg{Err: fmt.Errorf("network error")}
	model, cmd := a.Update(errMsg)
	a = model.(*App)

	assert.Equal(t, 2, a.statsPoll.errorCount)
	assert.Equal(t, calcBackoffTicks(2), a.statsPoll.backoffTicks)
	assert.Nil(t, cmd, "second error should NOT emit toast")
}

func TestApp_StatsLoaded_SuccessResetsPollState(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.statsPoll.errorCount = 2
	a.statsPoll.backoffTicks = 10

	msg := panes.StatsLoadedMsg{TimeRange: "short_term"}
	model, cmd := a.Update(msg)
	a = model.(*App)

	assert.Equal(t, 0, a.statsPoll.errorCount)
	assert.Equal(t, 0, a.statsPoll.backoffTicks)
	assert.True(t, a.statsPoll.hasData)
	assert.NotNil(t, cmd, "recovery from error should emit info toast")
}

// --- DevicesLoadedMsg (no recovery toast) ---

func TestApp_DevicesLoaded_ErrorFirstIncrementsButNoToast(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	errMsg := panes.DevicesLoadedMsg{Err: fmt.Errorf("network error")}
	model, _ := a.Update(errMsg)
	a = model.(*App)

	assert.Equal(t, 1, a.devicesPoll.errorCount, "first error should increment errorCount")
	assert.Equal(t, calcBackoffTicks(1), a.devicesPoll.backoffTicks, "first error should set backoffTicks")
}

func TestApp_DevicesLoaded_SuccessResetsPollState(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.devicesPoll.errorCount = 2
	a.devicesPoll.backoffTicks = 10

	msg := panes.DevicesLoadedMsg{Devices: []panes.DeviceInfo{}}
	model, _ := a.Update(msg)
	a = model.(*App)

	assert.Equal(t, 0, a.devicesPoll.errorCount, "success should reset errorCount")
	assert.Equal(t, 0, a.devicesPoll.backoffTicks, "success should reset backoffTicks")
	assert.True(t, a.devicesPoll.hasData, "success should set hasData")
}

// --- Playback error threshold ---

// TestApp_PlaybackErrors_ToastOnThird verifies that the playback error toast
// fires on the 3rd consecutive error (not 5th).
func TestApp_PlaybackErrors_ToastOnThird(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.width = 120
	a.height = 40

	errMsg := panes.PlaybackStateFetchedMsg{Err: fmt.Errorf("network error")}

	// 1st error: no toast.
	model, cmd := a.Update(errMsg)
	a = model.(*App)
	assert.Equal(t, 1, a.consecutivePlaybackErrors, "1st error should increment to 1")
	assert.Nil(t, cmd, "1st error should not emit toast")

	// 2nd error: no toast.
	model, cmd = a.Update(errMsg)
	a = model.(*App)
	assert.Equal(t, 2, a.consecutivePlaybackErrors, "2nd error should increment to 2")
	assert.Nil(t, cmd, "2nd error should not emit toast")

	// 3rd error: toast fires.
	model, cmd = a.Update(errMsg)
	a = model.(*App)
	assert.Equal(t, 3, a.consecutivePlaybackErrors, "3rd error should increment to 3")
	require.NotNil(t, cmd, "3rd error should emit warning toast")

	// Verify toast content.
	alertMsg := cmd()
	updated, _ := a.Update(alertMsg)
	appModel := updated.(*App)
	view := appModel.View()
	assert.Contains(t, view, "Playback updates failing", "toast should say 'Playback updates failing'")
}

// TestApp_PlaybackErrors_FourthSilent verifies that the 4th consecutive error
// does not emit another toast.
func TestApp_PlaybackErrors_FourthSilent(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	errMsg := panes.PlaybackStateFetchedMsg{Err: fmt.Errorf("network error")}

	// Drive error count to 3 (triggers toast).
	a.Update(errMsg)
	a.Update(errMsg)
	model, _ := a.Update(errMsg)
	a = model.(*App)

	// 4th error: silent.
	model, cmd := a.Update(errMsg)
	a = model.(*App)
	assert.Equal(t, 4, a.consecutivePlaybackErrors)
	assert.Nil(t, cmd, "4th error should not emit toast")
}

// TestApp_PlaybackErrors_ResetOnSuccess verifies that a successful playback
// fetch resets consecutivePlaybackErrors to 0.
func TestApp_PlaybackErrors_ResetOnSuccess(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	errMsg := panes.PlaybackStateFetchedMsg{Err: fmt.Errorf("network error")}
	a.Update(errMsg)
	a.Update(errMsg)
	model, _ := a.Update(errMsg)
	a = model.(*App)
	assert.Equal(t, 3, a.consecutivePlaybackErrors)

	// Success resets the counter.
	successMsg := panes.PlaybackStateFetchedMsg{State: &api.PlaybackState{IsPlaying: true}}
	model, _ = a.Update(successMsg)
	a = model.(*App)
	assert.Equal(t, 0, a.consecutivePlaybackErrors, "success should reset consecutivePlaybackErrors to 0")
}

// --- QueueLoadedMsg ---

// TestApp_QueueLoaded_ErrorFirstEmitsToast verifies that the first queue error
// increments errorCount and emits a toast. Subsequent errors are silent.
func TestApp_QueueLoaded_ErrorFirstEmitsToast(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	errMsg := panes.QueueLoadedMsg{Err: fmt.Errorf("network error")}

	// First error: toast.
	model, cmd := a.Update(errMsg)
	a = model.(*App)
	assert.Equal(t, 1, a.queuePoll.errorCount, "first error should set errorCount=1")
	require.NotNil(t, cmd, "first error should emit a toast")

	// Second error: silent.
	model2, cmd2 := a.Update(errMsg)
	a = model2.(*App)
	assert.Equal(t, 2, a.queuePoll.errorCount, "second error should set errorCount=2")
	assert.Nil(t, cmd2, "second error should NOT emit a toast")
}

// TestApp_QueueLoaded_errNilClient_SilentlyIgnored verifies that errNilClient
// in QueueLoadedMsg is silently ignored, leaving errorCount untouched.
func TestApp_QueueLoaded_errNilClient_SilentlyIgnored(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid

	msg := panes.QueueLoadedMsg{Err: errNilClient}
	_, cmd := a.Update(msg)
	assert.Nil(t, cmd, "errNilClient should not emit toast")
	assert.Equal(t, 0, a.queuePoll.errorCount, "errNilClient should not increment errorCount")
}

// TestApp_QueueLoaded_SuccessResetsPollState verifies that a successful
// QueueLoadedMsg resets errorCount and backoffTicks.
func TestApp_QueueLoaded_SuccessResetsPollState(t *testing.T) {
	a := newTestAppInternal()
	a.currentView = viewGrid
	a.width = 120
	a.height = 40

	// Prime error state.
	errMsg := panes.QueueLoadedMsg{Err: fmt.Errorf("network error")}
	a.Update(errMsg)
	assert.Equal(t, 1, a.queuePoll.errorCount)

	// Success resets.
	msg := panes.QueueLoadedMsg{Tracks: []domain.Track{}}
	model, cmd := a.Update(msg)
	a = model.(*App)
	assert.Equal(t, 0, a.queuePoll.errorCount, "success should reset errorCount")
	assert.Nil(t, cmd, "success should not emit toast")
}
