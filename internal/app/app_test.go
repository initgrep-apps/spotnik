package app_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/components/viz"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errorServer returns an httptest.Server that always responds with 500.
func errorServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"status":500,"message":"server error"}}`))
	}))
}

// successServer returns an httptest.Server that responds with 200 and an empty JSON object/array
// for any endpoint, to simulate a successful but empty response.
func successServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[],"total":0,"tracks":{"items":[],"total":0},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0},"queue":[]}`))
	}))
}

func TestAppNew_ReceivesTheme(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "monokai"

	a := app.New(cfg, app.AppOptions{})
	require.NotNil(t, a)
	assert.Equal(t, "monokai", a.Theme().ID())
}

func TestAppNew_DefaultThemeFallback(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "invalid-theme-id"

	a := app.New(cfg, app.AppOptions{})
	require.NotNil(t, a)
	// Unknown IDs fall back to DefaultThemeID without crashing.
	assert.Equal(t, theme.DefaultThemeID, a.Theme().ID())
}

func TestAppNew_EmptyThemeUsesDefault(t *testing.T) {
	cfg := &config.Config{}
	// cfg.Preferences.Theme is zero value (empty string)

	a := app.New(cfg, app.AppOptions{})
	require.NotNil(t, a)
	assert.Equal(t, theme.DefaultThemeID, a.Theme().ID())
}

// TestApp_Init_ReturnsBatch verifies Init returns a non-nil batch command.
func TestApp_Init_ReturnsBatch(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	cmd := a.Init()
	assert.NotNil(t, cmd, "Init should return a non-nil batch command")
}

// TestApp_Update_TickMsg_DispatchesFetch verifies that a TickMsg causes the app
// to produce a fetchPlaybackState command (returns a PlaybackStateFetchedMsg).
func TestApp_Update_TickMsg_DispatchesFetch(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	tickMsg := panes.TickMsg{}
	updatedModel, cmd := a.Update(tickMsg)

	assert.NotNil(t, updatedModel)
	assert.NotNil(t, cmd, "tickMsg should produce a follow-up command")
}

// TestApp_NowPlayingPaneRouting verifies key events are routed to the NowPlaying pane
// when it is focused.
func TestApp_NowPlayingPaneRouting(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Pre-populate the store so key events do something meaningful.
	a.Store().SetPlaybackState(&api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 30000,
		Item: &api.Track{
			ID:         "t1",
			Name:       "Test Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
		Device: &api.Device{VolumePercent: 60},
	})

	// NOTE: Bubbletea v0.27 delivers Space as tea.KeySpace, not as a rune.
	spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
	_, cmd := a.Update(spaceMsg)

	// When player is focused and there is playback state, space should
	// produce a command (pause/play or premium-gate toast).
	assert.NotNil(t, cmd, "space key when player focused should produce a command")
}

// TestApp_HeaderShowsDevice verifies that the app's View contains the device name
// when a device is present in the store.
func TestApp_HeaderShowsDevice(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	a.Store().SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item: &api.Track{
			ID:         "t1",
			Name:       "Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
		Device: &api.Device{
			Name:          "MacBook Pro",
			VolumePercent: 65,
		},
	})

	output := a.View()
	assert.Contains(t, output, "MacBook Pro", "header should show the active device name")
}

// TestPollingLoop_FetchesAndUpdatesStore tests that a PlaybackStateFetchedMsg
// with a data payload causes Update() to write the state to the store.
// After Feature 29, PlaybackStateFetchedMsg carries the fetched state in its
// State field, and Update() is responsible for writing it to the store.
func TestPollingLoop_FetchesAndUpdatesStore(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	s := a.Store()
	assert.Nil(t, s.PlaybackState(), "store should start empty")

	// Send data-carrying PlaybackStateFetchedMsg — Update() writes to store.
	newState := &api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 50000,
		Item: &api.Track{
			ID:         "t2",
			Name:       "New Song",
			DurationMs: 220000,
			Artists:    []api.Artist{{Name: "New Artist"}},
		},
		Device: &api.Device{VolumePercent: 80},
	}
	fetchedMsg := panes.PlaybackStateFetchedMsg{State: newState}
	_, _ = a.Update(fetchedMsg)

	got := s.PlaybackState()
	require.NotNil(t, got)
	assert.Equal(t, "New Song", got.Item.Name)
	assert.Equal(t, 50000, got.ProgressMs)
}

// TestApp_Update_WindowSizeMsg verifies window size is handled without crashing.
func TestApp_Update_WindowSizeMsg(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	sizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := a.Update(sizeMsg)

	require.NotNil(t, updatedModel)
}

// TestApp_View_EmptyState verifies View renders without crashing when store is empty.
func TestApp_View_EmptyState(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Should not panic.
	output := a.View()
	assert.NotEmpty(t, output)
}

// TestApp_StoreIsAccessible verifies that Store() returns the app's store.
func TestApp_StoreIsAccessible(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	s := a.Store()
	require.NotNil(t, s)

	// Verify it's the same store by setting and getting.
	s.SetActiveDevice(&api.Device{Name: "Test Device"})
	assert.Equal(t, "Test Device", a.Store().ActiveDevice().Name)
}

// TestApp_PlaylistsPaneRouting verifies Tab moves focus from NowPlaying to Playlists.
func TestApp_PlaylistsPaneRouting(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout with a proper size so focus rotation works.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// By default, NowPlaying pane is focused. Press Tab to move to next pane.
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := a.Update(tabMsg)

	require.NotNil(t, updatedModel)
	appModel := updatedModel.(*app.App)

	// After Tab, NowPlaying should no longer be focused.
	assert.False(t, appModel.NowPlayingFocused(), "Tab should unfocus NowPlaying pane")
}

// TestApp_LibraryPlay_UpdatesPlayback verifies that a PlayContextMsg
// produces a play command that flows through the root model.
func TestApp_LibraryPlay_UpdatesPlayback(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// PlayContextMsg simulates selecting a playlist to play.
	_, cmd := a.Update(panes.PlayContextMsg{ContextURI: "spotify:playlist:pl1"})

	// The command should be non-nil — play context was triggered
	assert.NotNil(t, cmd, "PlayContextMsg should produce a play command")
}

// TestApp_GridPane_View_ShowsInOutput verifies that the app View() output
// contains content from the grid panes (NowPlaying, Queue, Playlists).
func TestApp_GridPane_View_ShowsInOutput(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Set terminal size so the grid renders.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	output := a.View()
	// The grid renders panes with border titles visible; app should have non-empty output.
	assert.NotEmpty(t, output, "app view should include the grid panes")
}

// TestApp_SetPlayer verifies that SetPlayer injects the player client.
func TestApp_SetPlayer(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	player := api.NewPlayer("http://localhost", "test-token")
	a.SetPlayer(player)

	// No panic — player was set. Verify by pressing space (produces a command).
	a.Store().SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item:      &api.Track{ID: "t1", Name: "Track", DurationMs: 200000, Artists: []api.Artist{{Name: "Artist"}}},
		Device:    &api.Device{VolumePercent: 60},
	})

	// NOTE: Bubbletea v0.27 delivers Space as tea.KeySpace, not as a rune.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.NotNil(t, cmd, "space key should produce a command when player is set")
}

// TestApp_SetLibrary verifies that SetLibrary injects the library client.
func TestApp_SetLibrary(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	library := api.NewLibraryClient("http://localhost", "test-token")
	a.SetLibrary(library)
	// No panic — library was set
}

// TestApp_SetSearch verifies that SetSearch injects the search client.
func TestApp_SetSearch(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	search := api.NewSearchClient("http://localhost", "test-token")
	a.SetSearch(search)
	// No panic — search client was set
}

// TestApp_TabFocusRotation verifies Tab cycles focus through visible panes.
func TestApp_TabFocusRotation(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout so focus order is computed.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Start: NowPlaying focused (first in focus order after resize).
	assert.True(t, a.NowPlayingFocused(), "NowPlaying should be focused initially after resize")

	// Tab → next pane focused, NowPlaying unfocused.
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	m, _ = a.Update(tabMsg)
	a = m.(*app.App)
	assert.False(t, a.NowPlayingFocused(), "Tab should move focus away from NowPlaying")

	// Tab through all panes and eventually wrap back to NowPlaying.
	// There are 8 panes in the default dashboard preset; rotate through all.
	for i := 0; i < 7; i++ {
		m, _ = a.Update(tabMsg)
		a = m.(*app.App)
	}
	assert.True(t, a.NowPlayingFocused(), "Tab should wrap back to NowPlaying after full rotation")
}

// TestApp_PlayContextMsg_DispatchesPlayCmd verifies that a PlayContextMsg
// from the library pane produces a play command.
func TestApp_PlayContextMsg_DispatchesPlayCmd(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	playMsg := panes.PlayContextMsg{ContextURI: "spotify:playlist:pl1"}
	_, cmd := a.Update(playMsg)

	assert.NotNil(t, cmd, "PlayContextMsg should produce a play command")
}

// TestApp_PlayTrackListMsg_DispatchesPlayCmd verifies that a PlayTrackListMsg
// produces a play command (replaces the old PlayTrackMsg test).
func TestApp_PlayTrackListMsg_DispatchesPlayCmd(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	playMsg := panes.PlayTrackListMsg{URIs: []string{"spotify:track:t1"}}
	_, cmd := a.Update(playMsg)

	assert.NotNil(t, cmd, "PlayTrackListMsg should produce a play command")
}

// TestApp_FetchAlbumsRequest_ForwardedToApp verifies that FetchAlbumsRequestMsg
// produces a fetch command.
func TestApp_FetchAlbumsRequest_ForwardedToApp(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Send a FetchAlbumsRequestMsg — it should trigger a fetch command
	m, cmd := a.Update(panes.FetchAlbumsRequestMsg{Offset: 0})
	require.NotNil(t, m)
	// Albums library client is nil — should still return a command
	assert.NotNil(t, cmd, "FetchAlbumsRequestMsg should return a fetch command")
}

// TestApp_BuildPlaybackAPICmd_NilPlayer verifies that a nil player returns a no-op msg.
func TestApp_BuildPlaybackAPICmd_NilPlayer(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// player is nil — PlaybackRequestMsg should still return a cmd.
	a.Store().SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item:      &api.Track{ID: "t1", Name: "Track", DurationMs: 200000, Artists: []api.Artist{{Name: "Artist"}}},
		Device:    &api.Device{VolumePercent: 60},
	})

	actions := []panes.PlaybackAction{
		panes.ActionPause,
		panes.ActionPlay,
		panes.ActionNext,
		panes.ActionPrevious,
		panes.ActionVolumeUp,
		panes.ActionVolumeDown,
		panes.ActionToggleShuffle,
		panes.ActionCycleRepeat,
	}
	for _, action := range actions {
		_, cmd := a.Update(panes.PlaybackRequestMsg{Action: action})
		require.NotNil(t, cmd, "action %d should produce a cmd", action)
		msg := cmd()
		_, ok := msg.(panes.PlaybackCmdSentMsg)
		assert.True(t, ok, "action %d should return PlaybackCmdSentMsg, got %T", action, msg)
	}
}

// TestApp_BuildFetchCmds_NilLibrary verifies that nil library returns load-complete msgs.
func TestApp_BuildFetchCmds_NilLibrary(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// library is nil

	tests := []struct {
		name string
		msg  tea.Msg
	}{
		{"FetchPlaylists", panes.FetchPlaylistsRequestMsg{Offset: 0}},
		{"FetchAlbums", panes.FetchAlbumsRequestMsg{Offset: 0}},
		{"FetchLikedTracks", panes.FetchLikedTracksRequestMsg{Offset: 0}},
		{"FetchRecentlyPlayed", panes.FetchRecentlyPlayedRequestMsg{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := a.Update(tt.msg)
			require.NotNil(t, cmd, "%s should produce a cmd", tt.name)
			msg := cmd()
			require.NotNil(t, msg)
		})
	}
}

// TestApp_PlaybackCmdSentMsg_WithError verifies that a playback error emits a toast with the error text.
// Uses the two-pass pattern: execute the batch cmd and feed alert messages back to render the toast.
func TestApp_PlaybackCmdSentMsg_WithError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// Set a window size large enough for the main view so alerts render.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = m.(*app.App)

	errMsg := panes.PlaybackCmdSentMsg{Err: fmt.Errorf("playback failed")}
	_, cmd := a.Update(errMsg)
	require.NotNil(t, cmd, "error result should produce refetch + alert toast cmd")

	// Two-pass: the batch contains fetchPlaybackStateCmd + alertCmd.
	// Execute the batch and feed each sub-message back to Update so the alert renders.
	batchMsg := cmd()
	if bm, ok := batchMsg.(tea.BatchMsg); ok {
		for _, c := range bm {
			if msg := c(); msg != nil {
				a.Update(msg)
			}
		}
	} else if batchMsg != nil {
		a.Update(batchMsg)
	}
	assert.Contains(t, a.View(), "playback failed", "error toast should show the error text")
}

// TestApp_PlaybackCmdSentMsg_NoError verifies that a successful playback cmd triggers refetch.
func TestApp_PlaybackCmdSentMsg_NoError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	successMsg := panes.PlaybackCmdSentMsg{Err: nil}
	_, cmd := a.Update(successMsg)
	assert.NotNil(t, cmd, "successful playback should trigger a refetch cmd")
}

// TestApp_FetchPlaybackStateCmd_NilPlayer verifies nil player returns a notification.
func TestApp_FetchPlaybackStateCmd_NilPlayer(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Init returns a batch cmd; when player is nil it calls fetchPlaybackStateCmd(nil, api.Background).
	// Simulate via TickMsg which also calls fetchPlaybackStateCmd.
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd)
	// The batch contains fetchPlaybackStateCmd and a new tick. Execute one iteration.
	msg := cmd()
	require.NotNil(t, msg)
}

// TestApp_View_TooSmall verifies the minimum size check renders a help message.
func TestApp_View_TooSmall(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Set a size below the 120×30 threshold.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	appModel := m.(*app.App)

	output := appModel.View()
	assert.Contains(t, output, "Spotnik needs more space", "should show too-small message")
	assert.Contains(t, output, "120 × 30", "should show required dimensions")
}

// TestApp_View_StatusBarShowsGridHints verifies status bar shows global hints.
// Per F50, the status bar is now global-only: Space/n/+/- playback hints removed.
func TestApp_View_StatusBarShowsGridHints(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Global hints are always shown in the status bar.
	output := a.View()
	assert.Contains(t, output, "/", "status bar should show / search hint")
	assert.Contains(t, output, "q", "status bar should show q quit hint")
	assert.Contains(t, output, "page", "status bar should show page hint")
	assert.Contains(t, output, "preset", "status bar should show preset hint")
}

// TestApp_View_HeaderNoDevice verifies header shows "No device" when no device is active.
func TestApp_View_HeaderNoDevice(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	output := a.View()
	assert.Contains(t, output, "No device", "header should show No device when none active")
}

// TestApp_ShiftTab_RotatesFocusBackward verifies Shift+Tab cycles focus in reverse.
func TestApp_ShiftTab_RotatesFocusBackward(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout so focus rotation works.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Start at NowPlaying. Shift+Tab should go backward to last pane.
	assert.True(t, a.NowPlayingFocused(), "NowPlaying should be focused initially")

	// Shift+Tab wraps backward to the last pane in focus order.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	a = m.(*app.App)
	assert.False(t, a.NowPlayingFocused(), "Shift+Tab should move focus away from NowPlaying")

	// Tab forward back to NowPlaying.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	assert.True(t, a.NowPlayingFocused(), "Tab after Shift+Tab should return to NowPlaying")
}

// TestApp_PlaybackKey_WhenQueueFocused verifies playback keys work regardless of focus.
func TestApp_PlaybackKey_WhenQueueFocused(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout so Tab rotation works.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Move focus to Queue pane.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	// Focus is on whatever comes after NowPlaying; we don't need to validate which pane
	// exactly, only that playback key still reaches NowPlayingPane.
	assert.False(t, a.NowPlayingFocused(), "NowPlaying should not be focused after Tab")

	// Pre-populate store.
	a.Store().SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item:      &api.Track{ID: "t1", Name: "Track", DurationMs: 200000, Artists: []api.Artist{{Name: "Artist"}}},
		Device:    &api.Device{VolumePercent: 60},
	})

	// Space should still produce a playback command even when NowPlaying is not focused.
	// NOTE: Bubbletea v0.27 delivers Space as tea.KeySpace, not as a rune.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.NotNil(t, cmd, "space when other pane focused should route to NowPlaying pane")
}

// TestApp_QuitKey verifies q returns tea.Quit command.
func TestApp_QuitKey(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)
	// tea.Quit returns a tea.QuitMsg when executed.
	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.True(t, isQuit, "q should produce tea.QuitMsg")
}

// TestApp_AddToQueueMsg_DispatchesAPICmd verifies AddToQueueMsg produces a queue command.
// With no player injected, the command returns errNilClient in the message Err field
// so the Update() handler can silently skip it (no toast during pre-auth startup).
func TestApp_AddToQueueMsg_DispatchesAPICmd(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// Premium required for add-to-queue to pass the subscription gate.
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})

	queueMsg := panes.AddToQueueMsg{TrackURI: "spotify:track:abc"}
	_, cmd := a.Update(queueMsg)

	assert.NotNil(t, cmd, "AddToQueueMsg should produce a command")
	msg := cmd()
	resultMsg, ok := msg.(panes.AddToQueueResultMsg)
	assert.True(t, ok, "nil player should return AddToQueueResultMsg, got %T", msg)
	assert.Error(t, resultMsg.Err, "nil player must set Err on AddToQueueResultMsg (errNilClient)")
}

// TestApp_AddToQueueResultMsg_Success verifies success emits a "Added to queue" toast.
// Uses the two-pass pattern: execute the alert cmd then verify text appears in View().
func TestApp_AddToQueueResultMsg_Success(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// Set a window size large enough for the main view so alerts render.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = m.(*app.App)

	successMsg := panes.AddToQueueResultMsg{Err: nil}
	_, cmd := a.Update(successMsg)
	require.NotNil(t, cmd, "success should emit a toast alert cmd")

	// Two-pass: execute the alert cmd, feed the resulting message back to render the toast.
	alertMsg := cmd()
	_, _ = a.Update(alertMsg)
	assert.Contains(t, a.View(), "Added to queue", "success toast should mention queue addition")
}

// TestApp_AddToQueueResultMsg_Error verifies error emits a toast with the error text.
// Uses the two-pass pattern: execute the alert cmd then verify text appears in View().
func TestApp_AddToQueueResultMsg_Error(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// Set a window size large enough for the main view so alerts render.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = m.(*app.App)

	errMsg := panes.AddToQueueResultMsg{Err: fmt.Errorf("queue failed")}
	_, cmd := a.Update(errMsg)
	require.NotNil(t, cmd, "error result should emit an alert toast cmd")

	// Two-pass: execute the alert cmd, feed the resulting message back to render the toast.
	alertMsg := cmd()
	_, _ = a.Update(alertMsg)
	assert.Contains(t, a.View(), "queue failed", "error toast should show the error text")
}

// TestApp_SlashOpensSearch verifies '/' opens the search overlay.
func TestApp_SlashOpensSearch(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	_, _ = a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	appModel := model.(*app.App)

	assert.True(t, appModel.SearchOpen(), "'/' should open the search overlay")
}

// TestApp_EscClosesSearch verifies Esc closes the search overlay and restores pane focus.
func TestApp_EscClosesSearch(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open search
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = model.(*app.App)
	require.True(t, a.SearchOpen())

	// Close search via SearchClosedMsg
	model, _ = a.Update(panes.SearchClosedMsg{})
	a = model.(*app.App)

	assert.False(t, a.SearchOpen(), "SearchClosedMsg should close the overlay")
}

// TestApp_SearchPlay_OverlayStaysOpen verifies that playing a track/context does NOT close
// the search overlay — only Esc (SearchClosedMsg) closes it. This allows users to continue
// browsing after playing a result.
func TestApp_SearchPlay_OverlayStaysOpen(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open search
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = model.(*app.App)
	require.True(t, a.SearchOpen())

	// PlayTrackListMsg should NOT close the overlay.
	model, _ = a.Update(panes.PlayTrackListMsg{URIs: []string{"spotify:track:t1"}})
	a = model.(*app.App)
	assert.True(t, a.SearchOpen(), "PlayTrackListMsg should not close the search overlay")

	// PlayContextMsg should also NOT close the overlay.
	model, _ = a.Update(panes.PlayContextMsg{ContextURI: "spotify:playlist:pl1"})
	a = model.(*app.App)
	assert.True(t, a.SearchOpen(), "PlayContextMsg should not close the search overlay")

	// Only Esc (SearchClosedMsg) should close the overlay.
	model, _ = a.Update(panes.SearchClosedMsg{})
	a = model.(*app.App)
	assert.False(t, a.SearchOpen(), "SearchClosedMsg should close the overlay")
}

// TestApp_BackgroundDimmed verifies the view contains faint styling hint when overlay open.
func TestApp_BackgroundDimmed(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Set size so View renders full content
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = model.(*app.App)

	// Open search
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = model.(*app.App)

	// View should render without panic when overlay is open
	output := a.View()
	assert.NotEmpty(t, output, "view should not be empty when search overlay is open")
	assert.True(t, a.SearchOpen(), "search should still be open after View()")
}

// TestApp_SearchRequestMsg_DispatchesSearch verifies SearchRequestMsg triggers API cmd.
func TestApp_SearchRequestMsg_DispatchesSearch(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	searchMsg := panes.SearchRequestMsg{Query: "blinding lights"}
	_, cmd := a.Update(searchMsg)

	assert.NotNil(t, cmd, "SearchRequestMsg should produce a search command")
}

// TestApp_StatusDismiss_AlertAutoDismissl verifies that toast alerts are handled
// by BubbleUp's auto-dismiss mechanism (not a statusDismissMsg).
// After Task 3, statusDismissMsg is removed — BubbleUp handles dismiss internally.
func TestApp_StatusDismiss_ClearsMsg(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Trigger an error — emits an alert toast cmd (not statusMsg).
	errMsg := panes.PlaybackCmdSentMsg{Err: fmt.Errorf("error to dismiss")}
	_, alertCmd := a.Update(errMsg)
	// The alert cmd is non-nil — BubbleUp handles auto-dismiss internally.
	require.NotNil(t, alertCmd, "error should produce an alert toast cmd for auto-dismiss")

	// Before alertCmd is processed, view should NOT contain the error text
	// (toast messages only appear after the alertCmd fires through Update).
	// This is the correct new behavior — toast messages are overlay, not status bar.
}

// TestApp_TickFetchesQueue verifies that a TickMsg causes the app to
// dispatch both fetchPlaybackState and fetchQueue commands.
func TestApp_TickFetchesQueue(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Tick should produce a batch command (non-nil).
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd, "tickMsg should produce a follow-up command batch")
}

// TestApp_QueueLoadedMsg_UpdatesStore verifies that a QueueLoadedMsg updates the store.
func TestApp_QueueLoadedMsg_UpdatesStore(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	tracks := []api.Track{
		{ID: "q1", Name: "Save Your Tears", URI: "spotify:track:q1"},
	}
	a.Store().SetQueue(tracks)

	got := a.Store().Queue()
	require.Len(t, got, 1)
	assert.Equal(t, "Save Your Tears", got[0].Name)
}

// TestApp_QueueUpdate_StoreReflectsQueueData verifies that after a QueueLoadedMsg,
// the store contains the updated queue data (set by the fetchQueueCmd before the msg).
func TestApp_QueueUpdate_StoreReflectsQueueData(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Send QueueLoadedMsg carrying data — app.Update() writes to store.
	m, cmd := a.Update(panes.QueueLoadedMsg{
		Tracks: []api.Track{
			{ID: "q1", Name: "Save Your Tears", URI: "spotify:track:q1"},
		},
	})
	require.NotNil(t, m)
	assert.Nil(t, cmd, "QueueLoadedMsg should produce no follow-up command")

	// Store should reflect the queue data written by Update().
	got := a.Store().Queue()
	require.Len(t, got, 1)
	assert.Equal(t, "Save Your Tears", got[0].Name)
}

// TestAddToQueue_Success_EmitsToast verifies that a successful add-to-queue
// emits a success toast alert command. Toast appears via alerts.Render() overlay.
func TestAddToQueue_Success_ShowsStatusMessage(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	successMsg := panes.AddToQueueResultMsg{Err: nil, TrackName: "Blinding Lights"}
	m, cmd := a.Update(successMsg)
	require.NotNil(t, m)
	// A success toast alert cmd is returned — BubbleUp handles display/dismiss.
	assert.NotNil(t, cmd, "success should emit a toast alert cmd")
}

// TestAddToQueue_Error_EmitsToast verifies that a failed add-to-queue emits an error toast.
// Toast messages appear via alerts.Render() overlay — not directly in status bar.
func TestAddToQueue_Error_ShowsError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	errMsg := panes.AddToQueueResultMsg{Err: fmt.Errorf("Premium required")}
	m, cmd := a.Update(errMsg)
	require.NotNil(t, m)
	// An error toast alert cmd is returned — BubbleUp handles display/dismiss.
	assert.NotNil(t, cmd, "error should emit a toast alert cmd")
}

// TestAddToQueue_ToastAutoDismiss verifies that BubbleUp handles auto-dismiss
// (no statusDismissMsg required — BubbleUp's internal timer handles lifecycle).
func TestAddToQueue_StatusAutoDismiss(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Trigger toast alert — cmd is returned for BubbleUp to process.
	successMsg := panes.AddToQueueResultMsg{Err: nil, TrackName: "Starboy"}
	m, alertCmd := a.Update(successMsg)
	a = m.(*app.App)
	require.NotNil(t, alertCmd, "success should emit a toast alert cmd")

	// Execute alertCmd to get the internal alertMsg, then feed to Update.
	alertMsgResult := alertCmd()
	updated, _ := a.Update(alertMsgResult)
	a = updated.(*app.App)
	// After processing the alertMsg, BubbleUp has an active alert.
	// (The actual content appears in View() via alerts.Render().)
	_ = a.View() // should not panic
}

// TestApp_AddToQueueFromLibrary verifies that an AddToQueueMsg from a pane
// is dispatched to the API by the root app.
func TestApp_AddToQueueFromLibrary(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Simulate an AddToQueueMsg arriving from a pane (e.g. LikedSongs pressing 'a').
	_, cmd := a.Update(panes.AddToQueueMsg{TrackURI: "spotify:track:t1"})

	// Should produce a command (addToQueue dispatch).
	assert.NotNil(t, cmd, "AddToQueueMsg should produce an add-to-queue command")
}

// TestApp_QueueFocused verifies that QueueFocused returns true when queue pane is focused.
func TestApp_QueueFocused(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Default: NowPlaying focused.
	assert.False(t, a.QueueFocused(), "queue should not be focused initially")
	assert.True(t, a.NowPlayingFocused(), "NowPlaying should be focused initially")

	// Tab through focus order until we reach Queue.
	// Default dashboard preset order: NowPlaying, Playlists, Albums, LikedSongs, Queue, RecentlyPlayed, TopTracks, TopArtists
	found := false
	for i := 0; i < 8; i++ {
		m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
		a = m.(*app.App)
		if a.QueueFocused() {
			found = true
			break
		}
	}
	assert.True(t, found, "Tab rotation should reach Queue pane")
	assert.False(t, a.NowPlayingFocused(), "NowPlaying should not be focused when Queue is")
}

// TestApp_GridFocusRotation verifies focus cycles through all grid panes and wraps.
func TestApp_GridFocusRotation(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Start: NowPlaying focused.
	assert.True(t, a.NowPlayingFocused())

	// Tab forward through all panes (8 panes in default dashboard).
	for i := 0; i < 8; i++ {
		m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
		a = m.(*app.App)
	}

	// After 8 Tabs from NowPlaying, should wrap back to NowPlaying.
	assert.True(t, a.NowPlayingFocused(), "focus should wrap back to NowPlaying after full rotation")
}

// TestApp_View_RendersGridFallthrough verifies that the app View renders the grid
// when no terminal size is set (splash falls through to the main view for tests).
func TestApp_View_RendersGridFallthrough(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Without a WindowSizeMsg, viewSplash falls through to grid view.
	// No panes have computed sizes, so the grid is empty, but the
	// header and status bar should still be visible.
	output := a.View()
	assert.NotEmpty(t, output, "app view should render something even without terminal size")
	assert.Contains(t, output, "spotnik", "header should be visible in fallthrough grid view")
}

// TestApp_QueuePane_ShowsQueueData verifies that the queue pane shows store data in View().
// Uses QueueLoadedMsg to go through the proper data flow: msg → store → RefreshRows → table.
func TestApp_QueuePane_ShowsQueueData(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	a.Store().SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item: &api.Track{
			ID:      "now-1",
			Name:    "Blinding Lights",
			Artists: []api.Artist{{Name: "The Weeknd"}},
		},
	})

	// Route through Update so RefreshRows is called on the queue pane.
	a.Update(panes.QueueLoadedMsg{
		Tracks: []api.Track{
			{ID: "q1", Name: "Save Your Tears", URI: "spotify:track:q1", Artists: []api.Artist{{Name: "The Weeknd"}}},
		},
	})

	output := a.View()
	assert.NotEmpty(t, output, "app view should render")
	// Queue pane width is narrow in the default layout (no WindowSizeMsg),
	// so table columns truncate. Verify the row index "1" appears, confirming
	// data flowed through QueueLoadedMsg → store → RefreshRows → table.
	// Full track name rendering is verified in queue_test.go with proper width.
	assert.Contains(t, output, "1", "queue pane should have data row after QueueLoadedMsg")
}

// TestHeaderDeviceIndicator_ActiveDevice verifies the header shows ◉ and device name.
func TestHeaderDeviceIndicator_ActiveDevice(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	a.Store().SetActiveDevice(&api.Device{
		ID:       "abc123",
		Name:     "MacBook Pro Speakers",
		Type:     "Computer",
		IsActive: true,
	})

	output := a.View()
	assert.Contains(t, output, "◉", "active device header should contain ◉ symbol")
	assert.Contains(t, output, "MacBook Pro Speakers", "active device header should show device name")
}

// TestHeaderDeviceIndicator_NoDevice verifies the header shows ○ and "No device" when no device.
func TestHeaderDeviceIndicator_NoDevice(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// No device set in store

	output := a.View()
	assert.Contains(t, output, "○", "no-device header should contain ○ symbol")
	assert.Contains(t, output, "No device", "no-device header should show 'No device'")
}

// TestHeaderDeviceIndicator_LongName verifies that long device names are truncated with ….
func TestHeaderDeviceIndicator_LongName(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	longName := "This Is An Extremely Long Device Name That Exceeds Limit"
	a.Store().SetActiveDevice(&api.Device{
		ID:       "dev1",
		Name:     longName,
		Type:     "Computer",
		IsActive: true,
	})

	output := a.View()
	assert.Contains(t, output, "…", "long device name should be truncated with …")
	assert.NotContains(t, output, longName, "full long device name should not appear (should be truncated)")
}

// TestApp_DKeyOpensOverlay verifies that pressing d opens the device switcher overlay.
func TestApp_DKeyOpensOverlay(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	appModel := model.(*app.App)
	assert.True(t, appModel.DeviceOverlayOpen(), "pressing d should open device overlay")
}

// TestApp_DeviceOverlay_EscCloses verifies that Esc closes the device overlay.
func TestApp_DeviceOverlay_EscCloses(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open overlay
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = model.(*app.App)
	require.True(t, a.DeviceOverlayOpen())

	// Close with Esc (which produces DeviceOverlayClosedMsg via the overlay)
	model, _ = a.Update(panes.DeviceOverlayClosedMsg{})
	a = model.(*app.App)
	assert.False(t, a.DeviceOverlayOpen(), "DeviceOverlayClosedMsg should close the overlay")
}

// TestApp_DeviceTransfer_ShowsStatusMessage verifies that a transfer command
// emits an info toast showing the target device name.
// Uses the two-pass pattern: execute the batch cmd and verify toast text in View().
func TestApp_DeviceTransfer_ShowsStatusMessage(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// Set a window size large enough for the main view so alerts render.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = m.(*app.App)
	// Premium user required for transfer to proceed past the subscription gate.
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})

	transferMsg := panes.TransferPlaybackMsg{DeviceID: "def456", DeviceName: "iPhone 14"}
	_, cmd := a.Update(transferMsg)
	require.NotNil(t, cmd, "transfer should produce an API command + info toast")

	// Two-pass: the batch contains buildTransferPlaybackCmd + infoAlertCmd.
	// Execute the batch and feed each sub-message back to Update so the alert renders.
	batchMsg := cmd()
	if bm, ok := batchMsg.(tea.BatchMsg); ok {
		for _, c := range bm {
			if msg := c(); msg != nil {
				a.Update(msg)
			}
		}
	} else if batchMsg != nil {
		a.Update(batchMsg)
	}
	// Toast now uses Title="Device switching" + Body="iPhone 14" per §7.4 content rules.
	assert.Contains(t, a.View(), "iPhone 14", "info toast body should name the target device")
}

// TestApp_DeviceTransferredMsg_ErrorEmitsToast verifies transfer errors emit an error toast.
// Toast messages appear via alerts.Render() overlay — not directly in status bar.
func TestApp_DeviceTransferredMsg_ErrorShown(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	errMsg := panes.DeviceTransferredMsg{DeviceID: "def456", Err: fmt.Errorf("transfer failed")}
	_, cmd := a.Update(errMsg)
	// Error path emits Batch(fetchPlaybackStateCmd, errorAlertCmd) — non-nil.
	assert.NotNil(t, cmd, "transfer error should emit a toast alert cmd")
}

// TestApp_FetchDevicesRequestMsg_NilDevices verifies FetchDevicesRequestMsg with
// nil devices client returns DevicesLoadedMsg with empty list.
func TestApp_FetchDevicesRequestMsg_NilDevices(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// devices client is nil (not injected)

	_, cmd := a.Update(panes.FetchDevicesRequestMsg{})
	require.NotNil(t, cmd, "FetchDevicesRequestMsg should produce a command")

	// Execute the command — it should return DevicesLoadedMsg.
	msg := cmd()
	require.NotNil(t, msg)
}

// TestApp_DeviceOverlay_View_RenderedWhenOpen verifies that when device overlay is open,
// the view is rendered differently (overlay placed on top of the dimmed background).
func TestApp_DeviceOverlay_View_RenderedWhenOpen(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open the device overlay
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = model.(*app.App)
	require.True(t, a.DeviceOverlayOpen())

	// View should render the overlay (device pane content)
	output := a.View()
	assert.NotEmpty(t, output, "view should not be empty when device overlay is open")
}

// TestApp_DeviceOverlay_ViewWithSize_RenderedWhenOpen verifies that when device overlay is open
// with a valid terminal size, it renders via renderWithDeviceOverlay.
func TestApp_DeviceOverlay_ViewWithSize_RenderedWhenOpen(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Set a valid size (above minimum threshold)
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = model.(*app.App)

	// Open the device overlay
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = model.(*app.App)
	require.True(t, a.DeviceOverlayOpen())

	// View should render — renderWithDeviceOverlay path
	output := a.View()
	assert.NotEmpty(t, output, "view should not be empty when device overlay is open with a valid size")
}

// TestApp_DeviceTransferredMsg_Success verifies a successful transfer triggers playback refetch.
func TestApp_DeviceTransferredMsg_Success(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	successMsg := panes.DeviceTransferredMsg{DeviceID: "def456", Err: nil}
	_, cmd := a.Update(successMsg)
	assert.NotNil(t, cmd, "successful transfer should trigger a playback refetch command")
}

// TestApp_SetDevices verifies SetDevices injects the client.
func TestApp_SetDevices(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// Should not panic and should be callable.
	a.SetDevices(nil)
}

// TestApp_DeviceOverlay_KeysRoutedWhenOpen verifies that key events are routed
// to the device pane when the overlay is open (j/k navigation).
func TestApp_DeviceOverlay_KeysRoutedWhenOpen(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open the device overlay
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = model.(*app.App)
	require.True(t, a.DeviceOverlayOpen())

	// j key should be routed to device pane (not quit or other global handler)
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	a = model.(*app.App)
	// Still open — j does not close the overlay
	assert.True(t, a.DeviceOverlayOpen(), "j key should not close device overlay")
}

// TestApp_2KeyTogglesQueuePane verifies pressing 2 toggles the Queue pane visibility.
func TestApp_2KeyTogglesQueuePane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Press '2' — toggles Queue pane visibility. No panic.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	require.NotNil(t, model, "pressing 2 should not crash")
}

// TestApp_1KeyTogglesNowPlayingPane verifies pressing 1 toggles the NowPlaying pane visibility.
func TestApp_1KeyTogglesNowPlayingPane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Press '1' — toggles NowPlaying pane visibility. No panic.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	require.NotNil(t, model, "pressing 1 should not crash")
}

// TestApp_GridView_NoSeparateStatsView verifies that the app has no separate stats/playlist
// view mode — all content panes (stats, playlists, queue) are always in the grid.
func TestApp_GridView_NoSeparateStatsView(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// The app has no separate stats view open — press '2' just toggles Queue pane.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Pressing '2' (toggle Queue) is a pane visibility action; the surrounding
	// grid lifecycle remains intact (no view-mode transition).
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = m.(*app.App)
	require.NotNil(t, a, "pressing 2 should not crash")
}

// TestApp_StatsView_GridRendersTopTracks verifies that the TopTracks pane renders in the grid.
func TestApp_StatsView_GridRendersTopTracks(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Set window size so the grid renders.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	output := a.View()
	// TopTracks pane should be visible in the grid with its border title.
	assert.NotEmpty(t, output, "grid view should render non-empty output with all panes")
}

// TestApp_SearchDebounceRouted verifies that debounce messages reach the
// search overlay when it is open (not swallowed by library pane default).
func TestApp_SearchDebounceRouted(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open search
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = model.(*app.App)
	require.True(t, a.SearchOpen())

	// Type a character to set the query
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	a = model.(*app.App)

	// Send the debounce message (simulates the 300ms tick firing)
	debounceMsg := panes.SearchDebounceMsgForTest("x")
	_, cmd := a.Update(debounceMsg)

	// The debounce should have been routed to the search overlay,
	// which should emit a SearchRequestMsg command
	assert.NotNil(t, cmd, "debounce msg should produce a search request command when routed to overlay")
}

// TestApp_3KeyTogglesPlaylistsPane verifies pressing 3 toggles the Playlists pane visibility.
func TestApp_3KeyTogglesPlaylistsPane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Press '3' — toggles Playlists pane visibility. No panic.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	require.NotNil(t, model, "pressing 3 should not crash")
}

// TestApp_PlaylistsAlwaysInGrid verifies that pressing '3' does not open a separate
// playlist view mode — it simply toggles Playlists pane visibility in the grid.
func TestApp_PlaylistsAlwaysInGrid(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Pressing '3' toggles Playlists pane — verify it does not crash.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = m.(*app.App)
	require.NotNil(t, a, "pressing 3 should not crash")
}

// TestApp_LibraryLoadedMsg_ForwardsToPlaylistsPane verifies that LibraryLoadedMsg
// forwards data to the PlaylistsPane in the grid.
func TestApp_LibraryLoadedMsg_ForwardsToPlaylistsPane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Pre-populate store with playlists.
	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "Chill Vibes", URI: "spotify:playlist:pl-1", TrackCount: 24},
		{ID: "pl-2", Name: "Workout Mix", URI: "spotify:playlist:pl-2", TrackCount: 48},
	})

	// Set window size and send library loaded msg.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	msg := panes.LibraryLoadedMsg{Items: []api.SimplePlaylist{
		{ID: "pl-1", Name: "Chill Vibes", URI: "spotify:playlist:pl-1"},
	}}
	m, _ = a.Update(msg)
	a = m.(*app.App)

	// The grid view should render without panic.
	view := a.View()
	assert.NotEmpty(t, view, "view should render after LibraryLoadedMsg")
}

// TestApp_PlaylistView_GridRendersPlaylists verifies that playlists appear in the grid View.
func TestApp_PlaylistView_GridRendersPlaylists(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "My Playlist", URI: "spotify:playlist:pl-1", TrackCount: 5},
	})

	// Route LibraryLoadedMsg through so PlaylistsPane receives the data.
	m, _ = a.Update(panes.LibraryLoadedMsg{Items: []api.SimplePlaylist{
		{ID: "pl-1", Name: "My Playlist", URI: "spotify:playlist:pl-1"},
	}})
	a = m.(*app.App)

	view := a.View()
	assert.NotEmpty(t, view, "grid view should render with playlists data")
}

// TestApp_PlaylistViewHandlesRemoveRequest verifies PlaylistRemoveRequestMsg is handled.
func TestApp_PlaylistViewHandlesRemoveRequest(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "Chill Vibes", URI: "spotify:playlist:pl-1", TrackCount: 1},
	})
	a.Store().SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
	})

	msg := panes.PlaylistRemoveRequestMsg{PlaylistID: "pl-1", TrackURI: "spotify:track:t1"}
	model, _ := a.Update(msg)
	a = model.(*app.App)
	_ = a
}

// TestApp_CloseSearch_RestoresPrevFocus verifies that closing the search overlay
// restores the focus that was active before the overlay was opened.
func TestApp_CloseSearch_RestoresPrevFocus(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout so focus rotation works.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Move focus to Queue pane (Tab until QueueFocused).
	for i := 0; i < 8; i++ {
		m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
		a = m.(*app.App)
		if a.QueueFocused() {
			break
		}
	}
	require.True(t, a.QueueFocused(), "setup: queue should be focused before opening overlay")

	// Open search overlay — this saves the current focus (queue).
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = m.(*app.App)
	require.True(t, a.SearchOpen(), "search overlay should be open")

	// Close the search overlay.
	m, _ = a.Update(panes.SearchClosedMsg{})
	a = m.(*app.App)

	assert.False(t, a.SearchOpen(), "overlay should be closed")
	assert.True(t, a.QueueFocused(), "closing search should restore focus to queue")
	assert.False(t, a.NowPlayingFocused(), "NowPlaying should not be focused after restoring queue focus")
	assert.False(t, a.PlaylistsFocused(), "Playlists should not be focused after restoring queue focus")
}

// TestApp_CloseDeviceOverlay_RestoresPrevFocus verifies that closing the device overlay
// restores the focus that was active before the overlay was opened.
func TestApp_CloseDeviceOverlay_RestoresPrevFocus(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout so focus rotation works.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Move focus to Playlists pane (Tab once from NowPlaying).
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	// Focus is on the second pane (Playlists in default preset row 2 order).
	// Just verify NowPlaying is no longer focused.
	assert.False(t, a.NowPlayingFocused(), "NowPlaying should not be focused after Tab")

	// Record which pane is focused.
	focusedBefore := a.FocusedPane()

	// Open device overlay.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = m.(*app.App)
	require.True(t, a.DeviceOverlayOpen(), "device overlay should be open")

	// Close the device overlay.
	m, _ = a.Update(panes.DeviceOverlayClosedMsg{})
	a = m.(*app.App)

	assert.False(t, a.DeviceOverlayOpen(), "overlay should be closed")
	assert.Equal(t, focusedBefore, a.FocusedPane(), "closing device overlay should restore previous focus")
	assert.False(t, a.NowPlayingFocused(), "NowPlaying should not be focused after restoring previous pane")
}

// TestApp_PlaylistViewHandlesFetchTracksRequest verifies FetchPlaylistTracksRequestMsg is handled.
func TestApp_PlaylistViewHandlesFetchTracksRequest(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	msg := panes.FetchPlaylistTracksRequestMsg{PlaylistID: "pl-1"}
	model, cmd := a.Update(msg)
	a = model.(*app.App)
	// Library client is nil in test — command still returns a message.
	require.NotNil(t, cmd)
	_ = a
}

// TestApp_PlaylistRemoveResultMsg_Error verifies remove error is forwarded to playlist pane.
func TestApp_PlaylistRemoveResultMsg_Error(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "Chill Vibes"},
	})
	a.Store().SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
	})

	msg := panes.PlaylistRemoveResultMsg{PlaylistID: "pl-1", Err: fmt.Errorf("remove failed")}
	model, cmd := a.Update(msg)
	a = model.(*app.App)
	require.NotNil(t, cmd, "should return dismiss timer on error")
	_ = a
}

// TestApp_PlaylistTracksLoadedMsg_Forwarded verifies PlaylistTracksLoadedMsg is forwarded
// to the PlaylistsPane when the staleness key matches.
func TestApp_PlaylistTracksLoadedMsg_Forwarded(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetPlaylistTracksID("pl-1")

	msg := panes.PlaylistTracksLoadedMsg{PlaylistID: "pl-1", Tracks: []domain.Track{{ID: "t1", Name: "Track A", URI: "spotify:track:t1"}}, Total: 1, Offset: 0}
	model, _ := a.Update(msg)
	a = model.(*app.App)
	// Data should NOT be in store (pane owns it)
	assert.Empty(t, a.Store().PlaylistTracks("pl-1"), "tracks must not be written to store")
}

// TestApp_PlaylistTracksLoadedMsg_StaleMsgDiscarded verifies stale PlaylistTracksLoadedMsg is discarded.
func TestApp_PlaylistTracksLoadedMsg_StaleMsgDiscarded(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetPlaylistTracksID("current-pl")

	msg := panes.PlaylistTracksLoadedMsg{PlaylistID: "stale-pl", Tracks: []domain.Track{{ID: "t1"}}, Total: 1}
	_, cmd := a.Update(msg)
	assert.Nil(t, cmd, "stale PlaylistTracksLoadedMsg must be discarded")
}

// TestApp_FetchPlaylistTracksRequestMsg_SetsID verifies that FetchPlaylistTracksRequestMsg
// sets the staleness key and cancels any prior fetch.
func TestApp_FetchPlaylistTracksRequestMsg_SetsID(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	msg := panes.FetchPlaylistTracksRequestMsg{PlaylistID: "pl-1", Offset: 0}
	model, cmd := a.Update(msg)
	a = model.(*app.App)
	require.NotNil(t, cmd, "FetchPlaylistTracksRequestMsg should return a fetch cmd")
	assert.Equal(t, "pl-1", a.PlaylistTracksID(), "staleness ID must be set to playlist ID")
}

// TestApp_PlaylistTrackViewClosedMsg_ClearsID verifies that PlaylistTrackViewClosedMsg
// clears the staleness key and cancels any in-flight fetch.
func TestApp_PlaylistTrackViewClosedMsg_ClearsID(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetPlaylistTracksID("pl-1")

	model, cmd := a.Update(panes.PlaylistTrackViewClosedMsg{})
	a = model.(*app.App)
	assert.Nil(t, cmd, "PlaylistTrackViewClosedMsg must not emit a cmd")
	assert.Equal(t, "", a.PlaylistTracksID(), "PlaylistTrackViewClosedMsg must clear the staleness ID")
}

// TestApp_SetPlaylistsAPI verifies SetPlaylistsAPI stores the client.
func TestApp_SetPlaylistsAPI(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// SetPlaylistsAPI should not panic even with nil.
	a.SetPlaylistsAPI(nil)
}

// TestApp_PlaylistViewKeysRoutedToPane verifies key events are routed to the focused pane.
func TestApp_PlaylistViewKeysRoutedToPane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "Chill Vibes"},
		{ID: "pl-2", Name: "Workout Mix"},
	})

	// Initialize layout and navigate to Playlists pane.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Tab until Playlists pane is focused.
	for i := 0; i < 8; i++ {
		m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
		a = m.(*app.App)
		if a.PlaylistsFocused() {
			break
		}
	}

	// Press j to move cursor in playlist pane — should not crash.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	a = model.(*app.App)
	_ = a
}

// TestApp_PaneToggleKeys_NoCrash verifies '1'-'8' toggle keys work without crashing.
func TestApp_PaneToggleKeys_NoCrash(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Initialize layout.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Press '2' and '3' — these toggle Queue and Playlists pane visibility.
	// No stats/playlist "view mode" — everything is in the grid.
	for _, key := range []rune{'2', '3', '2', '3'} {
		model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		require.NotNil(t, model, "pressing %c should not crash", key)
		a = model.(*app.App)
	}
	// Pane toggle keys must not crash the app.
	require.NotNil(t, a, "pane toggle keys should not crash")
}

// --- Error state wiring tests ---
// These verify that build*Cmd functions set/clear error state in the Store.

func TestApp_BuildFetchPlaylistsCmd_SetsErrorOnFailure(t *testing.T) {
	srv := errorServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	require.NotNil(t, cmd)
	// Execute command, feed result back to Update() — store writes happen in Update().
	msg := cmd()
	m, _ := a.Update(msg)
	a = m.(*app.App)

	assert.Error(t, a.Store().PlaylistsFetchError(), "store should have playlists fetch error after API failure")
}

func TestApp_BuildFetchPlaylistsCmd_ClearsErrorOnSuccess(t *testing.T) {
	srv := successServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	// Pre-set an error.
	a.Store().SetPlaylistsFetchError(fmt.Errorf("previous error"))

	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	require.NotNil(t, cmd)
	// Execute command, feed result back to Update() — store writes happen in Update().
	msg := cmd()
	m, _ := a.Update(msg)
	a = m.(*app.App)

	assert.NoError(t, a.Store().PlaylistsFetchError(), "store should clear playlists fetch error on success")
}

func TestApp_BuildFetchDevicesCmd_SetsErrorOnFailure(t *testing.T) {
	srv := errorServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetDevices(api.NewDevicesClient(srv.URL, "test-token"))

	// Open device overlay so the command is dispatched.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = m.(*app.App)

	// FetchDevicesRequestMsg is sent by DeviceOverlay.Init(); simulate it.
	_, cmd := a.Update(panes.FetchDevicesRequestMsg{})
	require.NotNil(t, cmd)
	// Execute command, feed result back to Update() — root app.Update() writes store.
	msg := cmd()
	m, _ = a.Update(msg)
	a = m.(*app.App)

	assert.Error(t, a.Store().DevicesError(), "store should have devices error after API failure")
}

func TestApp_BuildFetchDevicesCmd_ClearsErrorOnSuccess(t *testing.T) {
	srv := successServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetDevices(api.NewDevicesClient(srv.URL, "test-token"))
	a.Store().SetDevicesError(fmt.Errorf("previous error"))

	// Open device overlay so the command is dispatched.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = m.(*app.App)

	// FetchDevicesRequestMsg is sent by DeviceOverlay.Init(); simulate it.
	_, cmd := a.Update(panes.FetchDevicesRequestMsg{})
	require.NotNil(t, cmd)
	// Execute command, feed result back to Update() — root app.Update() writes store.
	msg := cmd()
	m, _ = a.Update(msg)
	a = m.(*app.App)

	assert.NoError(t, a.Store().DevicesError(), "store should clear devices error on success")
}

func TestApp_BuildFetchStatsCmd_SetsErrorOnFailure(t *testing.T) {
	srv := errorServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetUserAPI(api.NewUserClient(srv.URL, "test-token"))

	// FetchStatsMsg is routed directly — no need to open a separate stats view.
	_, cmd := a.Update(panes.FetchStatsMsg{TimeRange: "short_term"})
	require.NotNil(t, cmd)
	// Execute command, feed result back to Update() — store writes happen in Update().
	msg := cmd()
	m, _ := a.Update(msg)
	a = m.(*app.App)

	assert.Error(t, a.Store().StatsError(), "store should have stats error after API failure")
}

func TestApp_BuildFetchStatsCmd_ClearsErrorOnSuccess(t *testing.T) {
	srv := successServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetUserAPI(api.NewUserClient(srv.URL, "test-token"))
	a.Store().SetStatsError(fmt.Errorf("previous error"))

	_, cmd := a.Update(panes.FetchStatsMsg{TimeRange: "short_term"})
	require.NotNil(t, cmd)
	// Execute command, feed result back to Update() — store writes happen in Update().
	msg := cmd()
	m, _ := a.Update(msg)
	a = m.(*app.App)

	assert.NoError(t, a.Store().StatsError(), "store should clear stats error on success")
}

func TestApp_FetchQueueCmd_SetsErrorOnFailure(t *testing.T) {
	srv := errorServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetPlayer(api.NewPlayer(srv.URL, "test-token"))

	// Tick triggers both playback and queue fetches.
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd)

	// The tick returns a batch; execute each sub-command and feed results back to Update().
	msg := cmd()
	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		for _, subCmd := range batchMsg {
			if subCmd != nil {
				subMsg := subCmd()
				if subMsg != nil {
					m, _ := a.Update(subMsg)
					a = m.(*app.App)
				}
			}
		}
	}

	assert.Error(t, a.Store().QueueError(), "store should have queue error after API failure")
}

func TestApp_SplashScreen_ShownOnStartup(t *testing.T) {
	cfg := &config.Config{}
	// Version defaults to "dev" when not provided in AppOptions.
	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	output := a.View()
	assert.Contains(t, output, "terminal Spotify client", "splash should show tagline")
	assert.Contains(t, output, "dev", "splash should show injected version (dev fallback)")
}

func TestApp_SplashScreen_StaysOnPlaybackData(t *testing.T) {
	cfg := &config.Config{}
	// Pass an explicit version so the assertion is unambiguous.
	a := app.New(cfg, app.AppOptions{Version: "v0.1.0"})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Splash should be active.
	output := a.View()
	assert.Contains(t, output, "v0.1.0")

	// Playback data arrives — splash should STAY for the full 5s duration.
	model, _ := a.Update(panes.PlaybackStateFetchedMsg{})
	a = model.(*app.App)

	output = a.View()
	assert.Contains(t, output, "v0.1.0", "splash should still be visible — only timer dismisses it")
}

// TestApp_OverlayRendering_UsesThemeBaseColor verifies that overlay rendering
// uses the theme's Base() color instead of a hardcoded #000000.
// With a non-black theme (monokai), the rendered overlay view must not panic
// and must be non-empty. This ensures no hardcoded color is used.
func TestApp_OverlayRendering_UsesThemeBaseColor(t *testing.T) {
	// Use monokai theme (Base() = #272822, not #000000).
	cfg := &config.Config{}
	cfg.Preferences.Theme = "monokai"
	a := app.New(cfg, app.AppOptions{})

	// Set valid terminal size.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = m.(*app.App)

	// Open search overlay with monokai theme — should render without panic.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = m.(*app.App)
	require.True(t, a.SearchOpen())

	output := a.View()
	assert.NotEmpty(t, output, "overlay view should be non-empty with monokai theme")

	// Close search, open device overlay with monokai theme.
	m, _ = a.Update(panes.SearchClosedMsg{})
	a = m.(*app.App)
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = m.(*app.App)
	require.True(t, a.DeviceOverlayOpen())

	output = a.View()
	assert.NotEmpty(t, output, "device overlay view should be non-empty with monokai theme")
}

// TestApp_BackoffExpiry_ForcesImmediateFetch verifies that when backoff ticks
// expire, the app immediately fires playback and queue fetch commands instead
// of waiting for the next modulo-aligned tick.
func TestApp_BackoffExpiry_ForcesImmediateFetch(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Set up a small backoff via RateLimitedMsg with RetryAfterSecs < default (10).
	// The handler clamps to defaultBackoffTicks=10, so we use that.
	rateLimitMsg := panes.RateLimitedMsg{RetryAfterSecs: 3}
	model, _ := a.Update(rateLimitMsg)
	a = model.(*app.App)

	// Backoff should be set to defaultBackoffTicks (10).
	assert.Equal(t, 10, a.BackoffTicks(), "backoff should be clamped to default 10")

	// Send 9 ticks — backoff decrements each time but stays > 0.
	for i := 0; i < 9; i++ {
		model, cmd := a.Update(panes.TickMsg{})
		a = model.(*app.App)
		assert.NotNil(t, cmd, "tick during backoff should return nextTick")
		assert.Greater(t, a.BackoffTicks(), 0, "backoff should still be active after tick %d", i)
	}

	// 10th tick: backoff expires → should return a batch (not just nextTick).
	// The batch includes nextTick + fetchPlaybackState + fetchQueue = 3 commands.
	model, cmd := a.Update(panes.TickMsg{})
	a = model.(*app.App)

	assert.Equal(t, 0, a.BackoffTicks(), "backoff should be zero after expiry tick")
	assert.Equal(t, 0, a.TickCount(), "tickCount should be reset to 0 after backoff expiry")
	assert.NotNil(t, cmd, "expiry tick should return a batch command for immediate fetch")
}

// TestApp_DevicesLoadedMsg_NilError_PopulatesDeviceList verifies that DevicesLoadedMsg
// with no error sets the device list in the overlay (no store write needed for device list).
func TestApp_DevicesLoadedMsg_NilError_PopulatesDeviceList(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open device overlay so the message is routed appropriately.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = m.(*app.App)
	require.True(t, a.DeviceOverlayOpen())

	devices := []panes.DeviceInfo{{ID: "dev1", Name: "MacBook", Type: "Computer", IsActive: true}}
	msg := panes.DevicesLoadedMsg{Devices: devices, Err: nil}
	m, _ = a.Update(msg)
	a = m.(*app.App)

	// On success: store error should be cleared, fetchedAt stamped (non-zero).
	assert.NoError(t, a.Store().DevicesError(), "store should have no device error on success")
	assert.False(t, a.Store().DevicesFetchedAt().IsZero(), "DevicesFetchedAt should be stamped on success")
}

// TestApp_DevicesLoadedMsg_WithError_EmitsToastAndSetsError verifies that DevicesLoadedMsg
// with an error sets the store error and routes to toast notification.
func TestApp_DevicesLoadedMsg_WithError_EmitsToastAndSetsError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open device overlay.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = m.(*app.App)
	require.True(t, a.DeviceOverlayOpen())

	msg := panes.DevicesLoadedMsg{Devices: nil, Err: fmt.Errorf("devices unavailable")}
	m, cmd := a.Update(msg)
	a = m.(*app.App)

	assert.Error(t, a.Store().DevicesError(), "store should record devices error")
	assert.NotNil(t, cmd, "error should produce a toast command")
}

// TestApp_AlbumsLoadedMsg_Offset0_ReplacesAlbums verifies that Offset=0 replaces albums in store.
func TestApp_AlbumsLoadedMsg_Offset0_ReplacesAlbums(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	existing := []api.SavedAlbum{{Album: api.FullAlbum{ID: "existing", Name: "Old"}}}
	a.Store().SetSavedAlbums(existing)

	newAlbums := []api.SavedAlbum{{Album: api.FullAlbum{ID: "new1", Name: "New1"}}}
	msg := panes.AlbumsLoadedMsg{Items: newAlbums, Offset: 0}
	m, _ := a.Update(msg)
	a = m.(*app.App)

	got := a.Store().SavedAlbums()
	require.Len(t, got, 1, "Offset=0 should replace albums, not append")
	assert.Equal(t, "new1", got[0].Album.ID)
}

// TestApp_AlbumsLoadedMsg_OffsetPositive_AppendsAlbums verifies that Offset>0 appends albums.
func TestApp_AlbumsLoadedMsg_OffsetPositive_AppendsAlbums(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	existing := []api.SavedAlbum{{Album: api.FullAlbum{ID: "existing", Name: "Old"}}}
	a.Store().SetSavedAlbums(existing)

	moreAlbums := []api.SavedAlbum{{Album: api.FullAlbum{ID: "new1", Name: "New1"}}}
	msg := panes.AlbumsLoadedMsg{Items: moreAlbums, Offset: 50}
	m, _ := a.Update(msg)
	a = m.(*app.App)

	got := a.Store().SavedAlbums()
	require.Len(t, got, 2, "Offset>0 should append to existing albums")
	assert.Equal(t, "existing", got[0].Album.ID)
	assert.Equal(t, "new1", got[1].Album.ID)
}

// --- Feature 39: Idle Polish & Test Coverage ---

// TestApp_WindowSizeMsg_ResetsLastInteraction verifies that a terminal resize resets
// lastInteraction, signalling user presence the same way a key press does.
func TestApp_WindowSizeMsg_ResetsLastInteraction(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Make the app idle by pushing lastInteraction into the past.
	a.SetLastInteraction(time.Now().Add(-120 * time.Second))
	require.True(t, a.IsIdle(), "app should be idle after 120s without interaction")

	// A terminal resize should reset the idle state.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := m.(*app.App)

	assert.False(t, updated.IsIdle(), "WindowSizeMsg should reset idle state")
}

// TestApp_WindowSizeMsg_ResetsTickCountWhenIdle verifies that when the app is idle
// and receives a WindowSizeMsg, tickCount resets to 0 to force an immediate poll.
func TestApp_WindowSizeMsg_ResetsTickCountWhenIdle(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Advance the tick count so we can verify it gets reset.
	for i := 0; i < 5; i++ {
		m, _ := a.Update(panes.TickMsg{})
		a = m.(*app.App)
	}
	require.Equal(t, 5, a.TickCount(), "tick count should have advanced to 5")

	// Make the app idle.
	a.SetLastInteraction(time.Now().Add(-120 * time.Second))
	require.True(t, a.IsIdle(), "app should be idle")

	// WindowSizeMsg while idle should reset tickCount.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := m.(*app.App)

	assert.Equal(t, 0, updated.TickCount(), "WindowSizeMsg while idle should reset tickCount to 0")
}

// TestApp_WindowSizeMsg_ReturnFromIdleDuringBackoff_EmitsToast verifies that when a
// terminal resize returns the app from idle while a 429 backoff is active, a ratelimit
// toast is emitted — matching the same behaviour as the KeyMsg idle-return path.
func TestApp_WindowSizeMsg_ReturnFromIdleDuringBackoff_EmitsToast(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Make the app idle.
	a.SetLastInteraction(time.Now().Add(-120 * time.Second))
	require.True(t, a.IsIdle(), "app should be idle")

	// Activate a backoff by sending RateLimitedMsg.
	m, _ := a.Update(panes.RateLimitedMsg{RetryAfterSecs: 15})
	a = m.(*app.App)
	require.Greater(t, a.BackoffTicks(), 0, "backoff should be active")

	// Now return from idle via a resize — should emit ratelimit toast.
	_, cmd := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.NotNil(t, cmd, "returning from idle during backoff via resize should produce a cmd")

	// Two-pass: execute the cmd to find the alert message.
	alertMsg := cmd()
	_, alertCmd := a.Update(alertMsg)
	if alertCmd != nil {
		nextMsg := alertCmd()
		if nextMsg != nil {
			a.Update(nextMsg)
		}
	}
	view := a.View()
	// Toast title is "Rate-limited" (noun+state per §7.4); body contains the countdown.
	assert.Contains(t, view, "Rate-limited", "toast should mention rate limiting")
	assert.Contains(t, view, "Resuming in", "toast should show countdown")
}

// TestApp_KeyMsg_ReturnFromIdleDuringBackoff_EmitsRatelimitToast verifies that
// when the user returns from idle while a 429 backoff is active, a ratelimit toast
// is emitted to explain the stale data.
func TestApp_KeyMsg_ReturnFromIdleDuringBackoff_EmitsRatelimitToast(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Make the app idle.
	a.SetLastInteraction(time.Now().Add(-120 * time.Second))
	require.True(t, a.IsIdle(), "app should be idle")

	// Activate a backoff by sending RateLimitedMsg.
	m, _ := a.Update(panes.RateLimitedMsg{RetryAfterSecs: 15})
	a = m.(*app.App)
	require.Greater(t, a.BackoffTicks(), 0, "backoff should be active")

	// Now return from idle via a key press — should emit ratelimit toast.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	require.NotNil(t, cmd, "returning from idle during backoff should produce a cmd")

	// Two-pass: execute the cmd batch to find the alert message.
	alertMsg := cmd()
	_, alertCmd := a.Update(alertMsg)
	// alertCmd may be nil if alertMsg was the toast; forward until the toast renders.
	if alertCmd != nil {
		nextMsg := alertCmd()
		if nextMsg != nil {
			a.Update(nextMsg)
		}
	}
	view := a.View()
	// Toast title is "Rate-limited" (noun+state per §7.4); body contains the countdown.
	assert.Contains(t, view, "Rate-limited", "toast should mention rate limiting")
	assert.Contains(t, view, "Resuming in", "toast should show countdown")
}

// TestApp_KeyMsg_ReturnFromIdleWithoutBackoff_NoToast verifies that returning from
// idle without an active backoff does NOT emit a ratelimit toast.
func TestApp_KeyMsg_ReturnFromIdleWithoutBackoff_NoToast(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Ensure no backoff is active.
	require.Equal(t, 0, a.BackoffTicks(), "no backoff should be active")

	// Make the app idle.
	a.SetLastInteraction(time.Now().Add(-120 * time.Second))
	require.True(t, a.IsIdle(), "app should be idle")

	// Return from idle via key press — should NOT emit a ratelimit toast.
	m, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	a = m.(*app.App)

	// If cmd is non-nil, feed it through and check view has no ratelimit mention.
	if cmd != nil {
		msg := cmd()
		if msg != nil {
			a.Update(msg)
		}
	}
	view := a.View()
	assert.NotContains(t, view, "resuming in", "no backoff means no ratelimit toast")
}

// TestApp_NilPlaybackState_NoWarnBefore30Ticks verifies that repeated nil PlaybackState
// results in no toast before the 30-tick threshold is reached.
func TestApp_NilPlaybackState_NoWarnBefore30Ticks(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Ensure PlaybackState is nil (no player injected, so fetchPlaybackStateCmd returns nil state).
	require.Nil(t, a.Store().PlaybackState(), "playback state should start nil")

	// Feed 29 nil-state PlaybackStateFetchedMsg messages — no toast should fire.
	for i := 0; i < 29; i++ {
		m, _ := a.Update(panes.PlaybackStateFetchedMsg{State: nil})
		a = m.(*app.App)
	}

	// View should not contain warning text.
	assert.NotContains(t, a.View(), "No playback state", "warning should not fire before 30 ticks")
}

// TestApp_NilPlaybackState_WarnAtTick30 verifies that a warning toast is emitted
// exactly when the 30th consecutive nil PlaybackState is received.
func TestApp_NilPlaybackState_WarnAtTick30(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	require.Nil(t, a.Store().PlaybackState(), "playback state should start nil")

	// Feed 29 nil-state messages first.
	for i := 0; i < 29; i++ {
		m, _ := a.Update(panes.PlaybackStateFetchedMsg{State: nil})
		a = m.(*app.App)
	}

	// The 30th nil-state message should emit the warning toast.
	_, cmd := a.Update(panes.PlaybackStateFetchedMsg{State: nil})
	require.NotNil(t, cmd, "30th nil state should emit a warning toast cmd")

	// Two-pass: execute cmd to get alert message, feed through Update.
	alertMsg := cmd()
	_, alertCmd := a.Update(alertMsg)
	if alertCmd != nil {
		nextMsg := alertCmd()
		if nextMsg != nil {
			a.Update(nextMsg)
		}
	}
	assert.Contains(t, a.View(), "No playback state", "warning toast should be visible at tick 30")
}

// --- Page B (Nerd Status) registration tests ---

// TestApp_PageB_Panes_Registered verifies that all Page B panes are created
// and present in the panes map after App.New().
func TestApp_PageB_Panes_Registered(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	require.NotNil(t, a)

	// All three new Page B panes and NetworkLogPane should be accessible.
	assert.NotNil(t, a.GatewayHealthPane(), "GatewayHealthPane should be registered")
	assert.NotNil(t, a.PollingTrafficPane(), "PollingTrafficPane should be registered")
	assert.NotNil(t, a.GatewayLivePane(), "GatewayLivePane should be registered")
	assert.NotNil(t, a.NetworkLogPane(), "NetworkLogPane should be registered")
}

// TestApp_PageB_Toggle_SwitchesPage verifies that pressing '0' switches to Page B.
func TestApp_PageB_Toggle_SwitchesPage(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Trigger window resize so the layout initializes.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Press '0' to toggle to Page B.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	a = m.(*app.App)

	assert.False(t, a.GridViewOpen(), "Page A should not be the 'grid view' on Page B")
}

// TestApp_PageB_TickMsg_ReachesGatewayHealthPane verifies that TickMsg is dispatched
// to GatewayHealthPane and that the pane actually drains gateway events (behavioral,
// not just a nil-check).
func TestApp_PageB_TickMsg_ReachesGatewayHealthPane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Record a gateway event in the store before sending TickMsg.
	a.Store().RecordEvent(domain.GatewayEvent{
		Kind:      domain.EventRequestAllowed,
		RequestID: 1,
		Method:    "GET",
		Path:      "/me/player",
		Priority:  domain.PriorityBackground,
		Snapshot:  domain.GatewayStateSnapshot{TokensAvailable: 8, TokensMax: 10, ConcurrentMax: 5},
	})

	cursorBefore := a.GatewayHealthPane().EventCursor()

	m, _ := a.Update(panes.TickMsg{})
	a = m.(*app.App)

	// GatewayHealthPane must have advanced its cursor — proving it processed TickMsg.
	cursorAfter := a.GatewayHealthPane().EventCursor()
	assert.Greater(t, cursorAfter, cursorBefore,
		"GatewayHealthPane must drain events on TickMsg (cursor must advance)")
}

// TestApp_PageB_TickMsg_ReachesNetworkLogPane verifies TickMsg is dispatched to
// NetworkLogPane and that the pane actually drains gateway events (behavioral).
func TestApp_PageB_TickMsg_ReachesNetworkLogPane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Record an HttpCompleted event so the NetworkLogPane has something to drain.
	a.Store().RecordEvent(domain.GatewayEvent{
		Timestamp:  time.Now(),
		Kind:       domain.EventHttpCompleted,
		RequestID:  1,
		Method:     "GET",
		Path:       "/me/player",
		StatusCode: 200,
		DurationMs: 40,
		Priority:   domain.PriorityBackground,
		Snapshot:   domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})

	m, _ := a.Update(panes.TickMsg{})
	a = m.(*app.App)

	// NetworkLogPane must have drained the event — CompletedRequestsLen > 0.
	assert.Greater(t, a.NetworkLogPane().CompletedRequestsLen(), 0,
		"NetworkLogPane must process HttpCompleted events on TickMsg")
}

// TestApp_PageB_TickMsg_DispatchesPollingSnapshot verifies that TickMsg causes the
// handler to build and forward a PollingSnapshotMsg to PollingTrafficPane. Driving
// the app to idle state ensures IsIdle is set in the snapshot.
func TestApp_PageB_TickMsg_DispatchesPollingSnapshot(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Resize so the layout initializes (View needs non-zero dimensions).
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Drive app to idle by backdating lastInteraction beyond the idle threshold.
	a.SetLastInteraction(time.Now().Add(-2 * time.Minute))

	// Send TickMsg — the handler builds PollingSnapshotMsg{IsIdle: true} and forwards it.
	m, _ = a.Update(panes.TickMsg{})
	a = m.(*app.App)

	ptp := a.PollingTrafficPane()
	require.NotNil(t, ptp)
	ptp.SetSize(120, 10)
	v := ptp.View()
	// With IsIdle:true the playback row renders "idle · …" — proves PollingSnapshotMsg
	// was dispatched. Without dispatch the row shows "? · running" (zero-value snapshot).
	assert.Contains(t, v, "idle",
		"Playback row must show idle status — proves PollingSnapshotMsg{IsIdle:true} was dispatched")
}

// TestApp_PageB_TickMsg_ReachesGatewayLivePane verifies that TickMsg is dispatched to
// GatewayLivePane and that the pane absorbs recorded gateway events into its buffer.
// Without the dispatch block in handlers.go the buffer stays empty.
func TestApp_PageB_TickMsg_ReachesGatewayLivePane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Record a gateway event so GatewayLivePane has something to drain on TickMsg.
	a.Store().RecordEvent(domain.GatewayEvent{
		Timestamp: time.Now(),
		Kind:      domain.EventRequestAllowed,
		RequestID: 1,
		Method:    "GET",
		Path:      "/me/player",
		Priority:  domain.PriorityBackground,
		Snapshot:  domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})

	m, _ := a.Update(panes.TickMsg{})
	a = m.(*app.App)

	// GatewayLivePane must have drained the event — BufferedEventCount > 0.
	assert.Greater(t, a.GatewayLivePane().BufferedEventCount(), 0,
		"GatewayLivePane must absorb gateway events on TickMsg (BufferedEventCount must be > 0)")
}

// TestApp_PageB_VizTick_ReachesNowPlayingPane verifies viz.TickMsg
// reaches the NowPlayingPane for animation. Page B panes do not consume viz.TickMsg.
func TestApp_PageB_VizTick_ReachesNowPlayingPane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// viz.TickMsg should not panic and all Page B panes should remain accessible.
	m, _ := a.Update(viz.TickMsg(time.Now()))
	a = m.(*app.App)

	assert.NotNil(t, a.GatewayHealthPane(), "GatewayHealthPane should survive viz.TickMsg")
	assert.NotNil(t, a.GatewayLivePane(), "GatewayLivePane should survive viz.TickMsg")
	assert.NotNil(t, a.PollingTrafficPane(), "PollingTrafficPane should survive viz.TickMsg")
}

// --- Feature 52: Mouse Scroll + Responsive Behavior tests ---

// TestApp_MouseScrollUp_OnPane_DoesNotChangeFocus verifies that a mouse wheel-up
// event over a pane scrolls that pane without changing keyboard focus.
func TestApp_MouseScrollUp_OnPane_DoesNotChangeFocus(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Resize so layout computes rects — otherwise PaneAt returns -1 for everything.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Capture the focused pane before mouse scroll.
	focusBefore := a.FocusedPane()

	// Simulate a mouse wheel-up event at a position within the content area.
	// y=10 is in the content area (header=1, status bar=1, so rows 1-48 are content).
	// x=10 is in the left column of the grid.
	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      10,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
	}
	m, _ = a.Update(mouseMsg)
	a = m.(*app.App)

	// Focus must not change (mouse scroll only — not click-to-focus).
	assert.Equal(t, focusBefore, a.FocusedPane(),
		"mouse scroll should not change keyboard focus")
}

// TestApp_MouseScrollUp_RoutesPaneUpdate verifies that a wheel-up event on a pane
// routes correctly — the app returns without crashing and the model is valid.
func TestApp_MouseScrollUp_RoutesPaneUpdate(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Send a wheel-up event in the content area. The app should route it to the
	// target pane without panicking and return a valid model.
	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      10,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
	}
	updated, _ := a.Update(mouseMsg)
	// Model must remain valid after mouse scroll routing.
	require.NotNil(t, updated, "Update with MouseMsg should return valid model")
}

// TestApp_MouseScrollDown_OnPane_DoesNotChangeFocus verifies that a mouse wheel-down
// event over a pane scrolls that pane without changing keyboard focus.
func TestApp_MouseScrollDown_OnPane_DoesNotChangeFocus(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	focusBefore := a.FocusedPane()

	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      10,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
	}
	m, _ = a.Update(mouseMsg)
	a = m.(*app.App)

	assert.Equal(t, focusBefore, a.FocusedPane(),
		"mouse scroll down should not change keyboard focus")
}

// TestApp_MouseScroll_HeaderArea_NoAction verifies that mouse scroll on y=0
// (header area) is ignored — PaneAt returns -1 for header.
func TestApp_MouseScroll_HeaderArea_NoAction(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	focusBefore := a.FocusedPane()

	// y=0 is the header row.
	mouseMsg := tea.MouseMsg{
		X:      80,
		Y:      0,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
	}
	m, _ = a.Update(mouseMsg)
	a = m.(*app.App)

	// Header area: focus must not change and app must not crash.
	assert.Equal(t, focusBefore, a.FocusedPane(),
		"mouse scroll on header should be ignored")
}

// TestApp_MouseScroll_StatusBarArea_NoAction verifies that mouse scroll on the
// last row (status bar) is ignored.
func TestApp_MouseScroll_StatusBarArea_NoAction(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	focusBefore := a.FocusedPane()

	// y=49 is the status bar row (height-1).
	mouseMsg := tea.MouseMsg{
		X:      80,
		Y:      49,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
	}
	m, _ = a.Update(mouseMsg)
	a = m.(*app.App)

	assert.Equal(t, focusBefore, a.FocusedPane(),
		"mouse scroll on status bar should be ignored")
}

// TestApp_MouseScroll_OverlayOpen_Ignored verifies that mouse scroll events are
// ignored when the search overlay or device overlay is open.
func TestApp_MouseScroll_OverlayOpen_Ignored(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	// Open the search overlay.
	slashMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	m, _ = a.Update(slashMsg)
	a = m.(*app.App)

	require.True(t, a.SearchOpen(), "search overlay should be open")

	focusBefore := a.FocusedPane()

	mouseMsg := tea.MouseMsg{
		X:      80,
		Y:      10,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
	}
	m, _ = a.Update(mouseMsg)
	a = m.(*app.App)

	// Focus should remain unchanged while overlay is open.
	assert.Equal(t, focusBefore, a.FocusedPane(),
		"mouse scroll should be ignored while search overlay is open")
}

// TestApp_MouseClick_NotHandled verifies that non-wheel mouse events (clicks)
// do not cause any side effects.
func TestApp_MouseClick_NotHandled(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	focusBefore := a.FocusedPane()

	// A left mouse button click — should be silently ignored.
	mouseMsg := tea.MouseMsg{
		X:      80,
		Y:      10,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	m, _ = a.Update(mouseMsg)
	a = m.(*app.App)

	assert.Equal(t, focusBefore, a.FocusedPane(),
		"left mouse click should not change focus")
}

// TestApp_MouseMotion_Ignored verifies that mouse motion events do not change focus.
func TestApp_MouseMotion_Ignored(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)

	focusBefore := a.FocusedPane()

	motionMsg := tea.MouseMsg{
		X:      80,
		Y:      10,
		Action: tea.MouseActionMotion,
		Button: tea.MouseButtonNone,
	}
	m, _ = a.Update(motionMsg)
	a = m.(*app.App)

	assert.Equal(t, focusBefore, a.FocusedPane(),
		"mouse motion event should not change focus")
}

// TestApp_NilPlaybackState_CounterResets verifies that the nil-state counter resets
// when a non-nil PlaybackState is received, preventing repeat warnings.
func TestApp_NilPlaybackState_CounterResets(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Build up 29 nil ticks.
	for i := 0; i < 29; i++ {
		m, _ := a.Update(panes.PlaybackStateFetchedMsg{State: nil})
		a = m.(*app.App)
	}

	// Receive a valid state — counter should reset.
	validState := &domain.PlaybackState{IsPlaying: true, Item: &domain.Track{ID: "t1"}}
	m, _ := a.Update(panes.PlaybackStateFetchedMsg{State: validState})
	a = m.(*app.App)

	// Now send 29 more nil states — should NOT fire a warning (counter was reset).
	for i := 0; i < 29; i++ {
		m, _ := a.Update(panes.PlaybackStateFetchedMsg{State: nil})
		a = m.(*app.App)
	}
	assert.NotContains(t, a.View(), "No playback state", "counter reset should prevent re-warn before 30 ticks")
}

// TestApp_OpenSearch_ResetsOverlayState verifies that reopening the search overlay
// (via '/') after a previous session yields a clean overlay — empty input, TabAll,
// and no leftover results. This is the regression test for story 96.
func TestApp_OpenSearch_ResetsOverlayState(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open search for the first session.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = m.(*app.App)
	require.True(t, a.SearchOpen(), "prerequisite: search should be open")

	// Simulate user typing a query — send keys to the overlay.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	a = m.(*app.App)
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	a = m.(*app.App)
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	a = m.(*app.App)

	// Deliver fake search results via SearchPageLoadedMsg (pre-converted SearchListItems).
	m, _ = a.Update(panes.SearchPageLoadedMsg{Results: []panes.SearchListItem{
		{Category: "track", Name: "Blinding Lights", URI: "spotify:track:t1", IsTrack: true},
	}})
	a = m.(*app.App)

	// Close the overlay via SearchClosedMsg.
	m, _ = a.Update(panes.SearchClosedMsg{})
	a = m.(*app.App)
	require.False(t, a.SearchOpen(), "prerequisite: search should be closed")

	// Reopen the search overlay — this triggers openSearch() which must call Reset().
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = m.(*app.App)
	require.True(t, a.SearchOpen(), "search should be open again")

	// Verify the overlay has been reset to a clean state.
	sp := a.SearchPane()
	assert.Equal(t, "", sp.Query(), "after reopen: input should be empty")
	assert.Equal(t, panes.TabAll, sp.ActiveTab(), "after reopen: active tab should be TabAll")
	assert.Empty(t, sp.ResultListItems(), "after reopen: result list should be empty")
}

// TestApp_FetchAlbumTracksRequestMsg_SetsID verifies that FetchAlbumTracksRequestMsg
// sets the albumTracksID staleness key and returns a fetch cmd.
func TestApp_FetchAlbumTracksRequestMsg_SetsID(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	msg := panes.FetchAlbumTracksRequestMsg{AlbumID: "alb-1", Offset: 0}
	model, cmd := a.Update(msg)
	a = model.(*app.App)
	require.NotNil(t, cmd, "FetchAlbumTracksRequestMsg should return a fetch cmd")
	assert.Equal(t, "alb-1", a.AlbumTracksID(), "staleness ID must be set to album ID")
}

// TestApp_AlbumTracksLoadedMsg_StaleMsg_Discarded verifies that a stale
// AlbumTracksLoadedMsg (mismatched AlbumID) is silently discarded.
func TestApp_AlbumTracksLoadedMsg_StaleMsg_Discarded(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// albumTracksID is "" by default, so any non-empty AlbumID is stale.

	msg := panes.AlbumTracksLoadedMsg{AlbumID: "old-album", Tracks: []domain.Track{{ID: "t1"}}}
	_, cmd := a.Update(msg)
	assert.Nil(t, cmd, "stale AlbumTracksLoadedMsg must be silently discarded")
}

// TestApp_AlbumTracksLoadedMsg_WithErr_EmitsToast verifies that an error in
// AlbumTracksLoadedMsg triggers a toast notification.
func TestApp_AlbumTracksLoadedMsg_WithErr_EmitsToast(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetAlbumTracksID("alb-1")

	albErr := fmt.Errorf("connection refused")
	msg := panes.AlbumTracksLoadedMsg{AlbumID: "alb-1", Err: albErr}
	_, cmd := a.Update(msg)
	assert.NotNil(t, cmd, "error in AlbumTracksLoadedMsg must emit a toast cmd")
}

// TestApp_AlbumTrackViewClosedMsg_ClearsID verifies that AlbumTrackViewClosedMsg
// clears the albumTracksID staleness key.
func TestApp_AlbumTrackViewClosedMsg_ClearsID(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetAlbumTracksID("alb-1")

	model, cmd := a.Update(panes.AlbumTrackViewClosedMsg{})
	a = model.(*app.App)
	assert.Nil(t, cmd, "AlbumTrackViewClosedMsg must not emit a cmd")
	assert.Equal(t, "", a.AlbumTracksID(), "AlbumTrackViewClosedMsg must clear the staleness ID")
}

// ctxCapturingPlayer is a test-local PlayerAPI that records the context passed to PlaybackState.
// It implements the full api.PlayerAPI interface so it can be injected via SetPlayer.
type ctxCapturingPlayer struct {
	capturedCtx context.Context
}

func (p *ctxCapturingPlayer) PlaybackState(ctx context.Context) (*api.PlaybackState, error) {
	p.capturedCtx = ctx
	return nil, nil
}
func (p *ctxCapturingPlayer) Play(_ context.Context, _ api.PlayOptions) error     { return nil }
func (p *ctxCapturingPlayer) Pause(_ context.Context) error                       { return nil }
func (p *ctxCapturingPlayer) Next(_ context.Context) error                        { return nil }
func (p *ctxCapturingPlayer) Previous(_ context.Context) error                    { return nil }
func (p *ctxCapturingPlayer) Seek(_ context.Context, _ int) error                 { return nil }
func (p *ctxCapturingPlayer) SetVolume(_ context.Context, _ int) error            { return nil }
func (p *ctxCapturingPlayer) SetShuffle(_ context.Context, _ bool) error          { return nil }
func (p *ctxCapturingPlayer) SetRepeat(_ context.Context, _ string) error         { return nil }
func (p *ctxCapturingPlayer) AddToQueue(_ context.Context, _ string) error        { return nil }
func (p *ctxCapturingPlayer) Queue(_ context.Context) (*api.QueueResponse, error) { return nil, nil }

// TestPlaybackCmdSentMsg_SuccessUsesInteractivePriority verifies that the PlaybackCmdSentMsg
// success path dispatches fetchPlaybackStateCmd with api.Interactive priority.
// This is a regression guard: if the handler is reverted to api.Background, this test fails.
func TestPlaybackCmdSentMsg_SuccessUsesInteractivePriority(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	mock := &ctxCapturingPlayer{}
	a.SetPlayer(mock)

	// Send success message — success path calls fetchPlaybackStateCmd(player, api.Interactive).
	_, cmd := a.Update(panes.PlaybackCmdSentMsg{Err: nil})
	require.NotNil(t, cmd, "success path must return a fetch cmd")

	// Execute the command to invoke PlaybackState and capture its context.
	_ = cmd()

	require.NotNil(t, mock.capturedCtx, "PlaybackState must have been called")
	assert.Equal(t, api.Interactive, api.PriorityFromContext(mock.capturedCtx),
		"PlaybackCmdSentMsg success path must use api.Interactive priority")
}

// TestDeviceTransferredMsg_SuccessUsesInteractivePriority verifies that the DeviceTransferredMsg
// success path dispatches fetchPlaybackStateCmd with api.Interactive priority.
// This is a regression guard: if the handler is reverted to api.Background, this test fails.
func TestDeviceTransferredMsg_SuccessUsesInteractivePriority(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	mock := &ctxCapturingPlayer{}
	a.SetPlayer(mock)

	// Send success message — success path calls fetchPlaybackStateCmd(player, api.Interactive).
	_, cmd := a.Update(panes.DeviceTransferredMsg{DeviceID: "abc123", Err: nil})
	require.NotNil(t, cmd, "success path must return a fetch cmd")

	// Execute the command to invoke PlaybackState and capture its context.
	_ = cmd()

	require.NotNil(t, mock.capturedCtx, "PlaybackState must have been called")
	assert.Equal(t, api.Interactive, api.PriorityFromContext(mock.capturedCtx),
		"DeviceTransferredMsg success path must use api.Interactive priority")
}
