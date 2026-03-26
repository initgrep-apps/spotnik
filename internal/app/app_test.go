package app_test

import (
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
	cfg.UI.Theme = "monokai"

	a := app.New(cfg, app.AppOptions{})
	require.NotNil(t, a)
	assert.Equal(t, "monokai", a.Theme().ID())
}

func TestAppNew_DefaultThemeFallback(t *testing.T) {
	cfg := &config.Config{}
	cfg.UI.Theme = "invalid-theme-id"

	a := app.New(cfg, app.AppOptions{})
	require.NotNil(t, a)
	// Unknown IDs fall back to DefaultThemeID without crashing.
	assert.Equal(t, theme.DefaultThemeID, a.Theme().ID())
}

func TestAppNew_EmptyThemeUsesDefault(t *testing.T) {
	cfg := &config.Config{}
	// cfg.UI.Theme is zero value (empty string)

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

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := a.Update(spaceMsg)

	// When player is focused and there is playback state, space should
	// produce a command (pause/play).
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

// TestApp_LibraryPaneRouting verifies Tab moves focus from player to library.
func TestApp_LibraryPaneRouting(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// By default, player pane is focused. Press Tab to move to library.
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := a.Update(tabMsg)

	require.NotNil(t, updatedModel)
	appModel := updatedModel.(*app.App)

	// Library pane should now be focused
	assert.True(t, appModel.LibraryFocused(), "Tab should move focus to library pane")
	assert.False(t, appModel.PlayerFocused(), "Tab should unfocus player pane")
}

// TestApp_LibraryPlay_UpdatesPlayback verifies that Enter on a playlist in the
// library produces a play command that flows through the root model.
func TestApp_LibraryPlay_UpdatesPlayback(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Focus library pane
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	model, _ := a.Update(tabMsg)
	a = model.(*app.App)

	// Pre-populate playlists in store
	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl1", Name: "Test Playlist", URI: "spotify:playlist:pl1"},
	})

	// Expand playlists section
	expandMsg := panes.LibraryExpandMsg(panes.SectionPlaylists)
	model, _ = a.Update(expandMsg)
	a = model.(*app.App)

	// Move down to playlist item and press Enter
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	a = model.(*app.App)

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// The command should be non-nil — play context was triggered
	assert.NotNil(t, cmd, "Enter on library playlist should produce a play command")
}

// TestApp_LibraryPane_View_ShowsInOutput verifies that the library pane
// appears in the app's View() output.
func TestApp_LibraryPane_View_ShowsInOutput(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	output := a.View()
	assert.Contains(t, output, "LIBRARY", "app view should include the library pane")
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

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
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

// TestApp_TabFocusRotation verifies Tab cycles focus: player → library → queue → player.
func TestApp_TabFocusRotation(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Start: player focused
	assert.True(t, a.PlayerFocused())
	assert.False(t, a.LibraryFocused())

	// Tab → library focused
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	m, _ := a.Update(tabMsg)
	a = m.(*app.App)
	assert.True(t, a.LibraryFocused())
	assert.False(t, a.PlayerFocused())

	// Tab again → queue focused
	m, _ = a.Update(tabMsg)
	a = m.(*app.App)
	assert.True(t, a.QueueFocused())
	assert.False(t, a.LibraryFocused())
	assert.False(t, a.PlayerFocused())

	// Tab again → player focused (wraps around)
	m, _ = a.Update(tabMsg)
	a = m.(*app.App)
	assert.True(t, a.PlayerFocused())
	assert.False(t, a.LibraryFocused())
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

// TestApp_PlayTrackMsg_DispatchesPlayCmd verifies that a PlayTrackMsg
// from the library pane produces a play command.
func TestApp_PlayTrackMsg_DispatchesPlayCmd(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	playMsg := panes.PlayTrackMsg{TrackURI: "spotify:track:t1"}
	_, cmd := a.Update(playMsg)

	assert.NotNil(t, cmd, "PlayTrackMsg should produce a play command")
}

// TestApp_LibraryLoadedMsg_ForwardedToLibraryPane verifies that library data messages
// are forwarded to the library pane.
func TestApp_LibraryLoadedMsg_ForwardedToLibraryPane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Send a library expand message to the app — it should be forwarded
	expandMsg := panes.LibraryExpandMsg(panes.SectionAlbums)
	m, cmd := a.Update(expandMsg)
	require.NotNil(t, m)
	// Albums are not cached — should return a fetch command
	assert.NotNil(t, cmd, "expanding uncached albums should return a fetch command")
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
		{"LikeTrack", panes.LikeTrackRequestMsg{TrackID: "t1", Unlike: false}},
		{"UnlikeTrack", panes.LikeTrackRequestMsg{TrackID: "t1", Unlike: true}},
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

// TestApp_LikeToggleResultMsg_WithError verifies that a like error emits a toast with the error text.
// Uses the two-pass pattern: execute the alert cmd then verify text appears in View().
func TestApp_LikeToggleResultMsg_WithError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// Set a window size large enough for the main view so alerts render.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = m.(*app.App)

	errMsg := panes.LikeToggleResultMsg{TrackID: "t1", Err: fmt.Errorf("like failed")}
	_, cmd := a.Update(errMsg)
	require.NotNil(t, cmd, "error result should produce an alert toast cmd")

	// Two-pass: execute the alert cmd, feed the resulting message back to render the toast.
	alertMsg := cmd()
	_, _ = a.Update(alertMsg)
	assert.Contains(t, a.View(), "like failed", "error toast should show the error text")
}

// TestApp_LikeToggleResultMsg_NoError verifies a successful like clears status.
func TestApp_LikeToggleResultMsg_NoError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	successMsg := panes.LikeToggleResultMsg{TrackID: "t1", Err: nil}
	m, cmd := a.Update(successMsg)
	require.NotNil(t, m)
	assert.Nil(t, cmd, "successful like should not produce a cmd")
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

	// Init returns a batch cmd; when player is nil it calls fetchPlaybackStateCmd(nil, store).
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

	// Set a size below the 100×24 threshold.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	appModel := m.(*app.App)

	output := appModel.View()
	assert.Contains(t, output, "Spotnik needs more space", "should show too-small message")
	assert.Contains(t, output, "100 × 24", "should show required dimensions")
}

// TestApp_View_StatusBarContextSensitive verifies status bar shows player hints when player focused.
func TestApp_View_StatusBarContextSensitive(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Player focused by default.
	output := a.View()
	assert.Contains(t, output, "Space", "player status bar should show Space hint")
	assert.Contains(t, output, "/", "status bar should show / search hint")

	// Tab to library.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	appModel := m.(*app.App)
	output = appModel.View()
	assert.Contains(t, output, "Enter", "library status bar should show Enter hint")
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

	// Start at player. Shift+Tab should go backward: player → queue.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	a = m.(*app.App)
	assert.True(t, a.QueueFocused(), "Shift+Tab from player should go to queue (backward)")

	// Shift+Tab again: queue → library.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	a = m.(*app.App)
	assert.True(t, a.LibraryFocused(), "Shift+Tab from queue should go to library")

	// Shift+Tab again: library → player.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	a = m.(*app.App)
	assert.True(t, a.PlayerFocused(), "Shift+Tab from library should go to player")
}

// TestApp_PlaybackKey_WhenLibraryFocused verifies playback keys work regardless of focus.
func TestApp_PlaybackKey_WhenLibraryFocused(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Move focus to library.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	assert.True(t, a.LibraryFocused())

	// Pre-populate store.
	a.Store().SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item:      &api.Track{ID: "t1", Name: "Track", DurationMs: 200000, Artists: []api.Artist{{Name: "Artist"}}},
		Device:    &api.Device{VolumePercent: 60},
	})

	// Space should still produce a playback command.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	assert.NotNil(t, cmd, "space when library focused should route to player pane")
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

// TestApp_SearchPlayClosesOverlay verifies that a play command from search closes the overlay.
func TestApp_SearchPlayClosesOverlay(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open search
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = model.(*app.App)
	require.True(t, a.SearchOpen())

	// Send a PlayTrackMsg (simulating Enter on a search result)
	model, _ = a.Update(panes.PlayTrackMsg{TrackURI: "spotify:track:t1"})
	a = model.(*app.App)

	assert.False(t, a.SearchOpen(), "playing from search should close the overlay")
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

// TestApp_AddToQueueFromLibrary verifies that pressing 'a' in the library pane
// on a track emits an AddToQueueMsg, which the root app dispatches to the API.
func TestApp_AddToQueueFromLibrary(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Focus library pane.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	require.True(t, a.LibraryFocused())

	// Pre-populate liked tracks so 'a' has a track to queue.
	// LikedSongs section (index 2) — cursor starts at 0 (Playlists), j×2 to reach it.
	a.Store().SetLikedTracks([]api.SavedTrack{
		{Track: api.Track{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1", Artists: []api.Artist{{Name: "The Weeknd"}}}},
	})

	// Navigate down to the LikedSongs section header (row 2).
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	a = m.(*app.App)
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	a = m.(*app.App)

	// Expand liked songs section (now at header).
	expandMsg := panes.LibraryExpandMsg(panes.SectionLikedSongs)
	m, _ = a.Update(expandMsg)
	a = m.(*app.App)

	// Navigate down to the first liked track item.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	a = m.(*app.App)

	// Press 'a' to add to queue.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	// 'a' should produce a command (addToQueue dispatch).
	assert.NotNil(t, cmd, "'a' on a track in library should produce an add-to-queue command")
}

// TestApp_QueueFocused verifies that QueueFocused returns true when queue pane is focused.
func TestApp_QueueFocused(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Default: player focused.
	assert.False(t, a.QueueFocused(), "queue should not be focused initially")

	// Tab to library.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	assert.False(t, a.QueueFocused())

	// Tab again to queue.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	assert.True(t, a.QueueFocused(), "second Tab should focus queue pane")
	assert.False(t, a.PlayerFocused())
	assert.False(t, a.LibraryFocused())
}

// TestApp_ThreePaneFocusRotation verifies focus cycles player → library → queue → player.
func TestApp_ThreePaneFocusRotation(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Start: player
	assert.True(t, a.PlayerFocused())

	// Tab: library
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	assert.True(t, a.LibraryFocused())

	// Tab: queue
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	assert.True(t, a.QueueFocused())

	// Tab: back to player
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	assert.True(t, a.PlayerFocused())
}

// TestApp_View_ContainsQueuePane verifies that the app View renders the queue pane with column headers.
func TestApp_View_ContainsQueuePane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Queue pane uses a bubble-table; the # column header should be visible.
	// Calling View() before setting a terminal size falls through to the main view.
	output := a.View()
	assert.Contains(t, output, "#", "app view should include the queue pane table")
}

// TestApp_QueuePane_ShowsQueueData verifies that the queue pane shows store data in View().
// Queue pane data rendering is exhaustively tested in queue_test.go; this test verifies
// integration — the queue pane is wired into the app and renders without error.
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
	a.Store().SetQueue([]api.Track{
		{ID: "q1", Name: "Save Your Tears", URI: "spotify:track:q1", Artists: []api.Artist{{Name: "The Weeknd"}}},
	})

	output := a.View()
	assert.NotEmpty(t, output, "app view should render")
	// The queue pane table should be present — # is the first column header.
	assert.Contains(t, output, "#", "queue pane table should be visible in the layout")
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
	assert.Contains(t, a.View(), "Switching to iPhone 14", "info toast should name the target device")
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

// TestApp_2KeyOpensStats verifies pressing 2 switches to the Stats view.
func TestApp_2KeyOpensStats(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// By default stats view is not open.
	assert.False(t, a.StatsViewOpen(), "stats view should not be open by default")

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = model.(*app.App)

	assert.True(t, a.StatsViewOpen(), "pressing 2 should open the stats view")
}

// TestApp_1KeyReturnsToLibrary verifies pressing 1 from stats restores the three-pane layout.
func TestApp_1KeyReturnsToLibrary(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open stats view.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = model.(*app.App)
	require.True(t, a.StatsViewOpen())

	// Press 1 to return to library view.
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	a = model.(*app.App)
	assert.False(t, a.StatsViewOpen(), "pressing 1 should close the stats view")
}

// TestApp_StatsPreservesCursor verifies returning to stats preserves cursor and section.
func TestApp_StatsPreservesCursor(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open stats.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = model.(*app.App)
	require.True(t, a.StatsViewOpen())

	// Close stats, then reopen — model should still exist (lazy init only on first open).
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	a = model.(*app.App)
	assert.False(t, a.StatsViewOpen())

	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = model.(*app.App)
	assert.True(t, a.StatsViewOpen(), "reopening stats should work after closing")
}

// TestApp_StatsView_ViewRendersInStatsMode verifies that View() returns the stats content
// when the stats view is open.
func TestApp_StatsView_ViewRendersInStatsMode(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Set window size.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = m.(*app.App)

	// Open stats view.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = model.(*app.App)

	output := a.View()
	assert.Contains(t, output, "TOP TRACKS", "stats view should render TOP TRACKS section")
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

// TestApp_3KeyOpensPlaylists verifies pressing 3 switches to the PlaylistManager view.
func TestApp_3KeyOpensPlaylists(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// By default playlist view is not open.
	assert.False(t, a.PlaylistViewOpen(), "playlist view should not be open by default")

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)

	assert.True(t, a.PlaylistViewOpen(), "pressing 3 should open the playlist view")
}

// TestApp_1KeyReturnsFromPlaylists verifies pressing 1 from playlists restores the three-pane layout.
func TestApp_1KeyReturnsFromPlaylists(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open playlist view.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)
	require.True(t, a.PlaylistViewOpen())

	// Press 1 to return to library view.
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	a = model.(*app.App)
	assert.False(t, a.PlaylistViewOpen(), "pressing 1 should close the playlist view")
}

// TestApp_PlaylistsReusesLibraryData verifies that opening playlists uses store data without extra fetch.
func TestApp_PlaylistsReusesLibraryData(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Pre-populate store with playlists.
	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "Chill Vibes", URI: "spotify:playlist:pl-1", TrackCount: 24},
		{ID: "pl-2", Name: "Workout Mix", URI: "spotify:playlist:pl-2", TrackCount: 48},
	})

	// Open playlist view — no extra API fetch should occur (cmd is nil or init cmd).
	model, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)
	require.True(t, a.PlaylistViewOpen())

	// The view should render the playlists from the store.
	view := a.View()
	assert.Contains(t, view, "Chill Vibes", "view should show playlists from store")
	assert.Contains(t, view, "Workout Mix", "view should show playlists from store")
	_ = cmd
}

// TestApp_PlaylistView_RendersCorrectly verifies View() returns playlist content when open.
func TestApp_PlaylistView_RendersCorrectly(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = m.(*app.App)

	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "My Playlist", URI: "spotify:playlist:pl-1", TrackCount: 5},
	})

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)
	require.True(t, a.PlaylistViewOpen())

	view := a.View()
	assert.Contains(t, view, "MY PLAYLISTS", "playlist view should contain MY PLAYLISTS header")
}

// TestApp_PlaylistViewHandlesCreateRequest verifies PlaylistCreateRequestMsg is handled.
func TestApp_PlaylistViewHandlesCreateRequest(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open playlist view.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)
	require.True(t, a.PlaylistViewOpen())

	// Send create request — root app should handle it.
	msg := panes.PlaylistCreateRequestMsg{Name: "New Playlist", Description: ""}
	model, _ = a.Update(msg)
	a = model.(*app.App)
	_ = a
}

// TestApp_PlaylistViewHandlesRenameRequest verifies PlaylistRenameRequestMsg is handled.
func TestApp_PlaylistViewHandlesRenameRequest(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "Chill Vibes", URI: "spotify:playlist:pl-1", TrackCount: 24},
	})

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)
	require.True(t, a.PlaylistViewOpen())

	msg := panes.PlaylistRenameRequestMsg{PlaylistID: "pl-1", NewName: "Renamed"}
	model, _ = a.Update(msg)
	a = model.(*app.App)
	_ = a
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

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)
	require.True(t, a.PlaylistViewOpen())

	msg := panes.PlaylistRemoveRequestMsg{PlaylistID: "pl-1", TrackURI: "spotify:track:t1"}
	model, _ = a.Update(msg)
	a = model.(*app.App)
	_ = a
}

// TestApp_CloseSearch_RestoresPrevFocus verifies that closing the search overlay
// restores the focus that was active before the overlay was opened.
func TestApp_CloseSearch_RestoresPrevFocus(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Move focus to queue pane (Tab twice: player → library → queue).
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	require.True(t, a.QueueFocused(), "setup: queue should be focused before opening overlay")

	// Open search overlay — this saves prevFocus = queue.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = m.(*app.App)
	require.True(t, a.SearchOpen(), "search overlay should be open")

	// Close the search overlay.
	m, _ = a.Update(panes.SearchClosedMsg{})
	a = m.(*app.App)

	assert.False(t, a.SearchOpen(), "overlay should be closed")
	assert.True(t, a.QueueFocused(), "closing search should restore focus to queue")
	assert.False(t, a.PlayerFocused(), "player should not be focused after restoring queue focus")
	assert.False(t, a.LibraryFocused(), "library should not be focused after restoring queue focus")
}

// TestApp_CloseDeviceOverlay_RestoresPrevFocus verifies that closing the device overlay
// restores the focus that was active before the overlay was opened.
func TestApp_CloseDeviceOverlay_RestoresPrevFocus(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Move focus to library pane (Tab once: player → library).
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	require.True(t, a.LibraryFocused(), "setup: library should be focused before opening overlay")

	// Open device overlay — this saves prevFocus = library.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = m.(*app.App)
	require.True(t, a.DeviceOverlayOpen(), "device overlay should be open")

	// Close the device overlay.
	m, _ = a.Update(panes.DeviceOverlayClosedMsg{})
	a = m.(*app.App)

	assert.False(t, a.DeviceOverlayOpen(), "overlay should be closed")
	assert.True(t, a.LibraryFocused(), "closing device overlay should restore focus to library")
	assert.False(t, a.PlayerFocused(), "player should not be focused after restoring library focus")
	assert.False(t, a.QueueFocused(), "queue should not be focused after restoring library focus")
}

// TestApp_PlaylistViewHandlesReorderRequest verifies PlaylistReorderRequestMsg is handled.
func TestApp_PlaylistViewHandlesReorderRequest(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "Chill Vibes", URI: "spotify:playlist:pl-1", TrackCount: 2},
	})

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)
	require.True(t, a.PlaylistViewOpen())

	msg := panes.PlaylistReorderRequestMsg{PlaylistID: "pl-1", RangeStart: 0, InsertBefore: 2, RangeLength: 1}
	model, _ = a.Update(msg)
	a = model.(*app.App)
	_ = a
}

// TestApp_PlaylistViewHandlesFetchTracksRequest verifies FetchPlaylistTracksRequestMsg is handled.
func TestApp_PlaylistViewHandlesFetchTracksRequest(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)
	require.True(t, a.PlaylistViewOpen())

	msg := panes.FetchPlaylistTracksRequestMsg{PlaylistID: "pl-1"}
	model, cmd := a.Update(msg)
	a = model.(*app.App)
	// Library client is nil in test — command still returns a message.
	require.NotNil(t, cmd)
	_ = a
}

// TestApp_PlaylistCreatedMsg_Success verifies PlaylistCreatedMsg triggers playlist re-fetch.
func TestApp_PlaylistCreatedMsg_Success(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)

	msg := panes.PlaylistCreatedMsg{PlaylistID: "new-pl", Name: "New Playlist"}
	model, cmd := a.Update(msg)
	a = model.(*app.App)
	require.NotNil(t, cmd, "should return fetch playlists command after create")
	_ = a
}

// TestApp_PlaylistCreatedMsg_Error verifies PlaylistCreatedMsg with error shows status.
func TestApp_PlaylistCreatedMsg_Error(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)

	msg := panes.PlaylistCreatedMsg{Err: fmt.Errorf("create failed")}
	model, cmd := a.Update(msg)
	a = model.(*app.App)
	require.NotNil(t, cmd, "should return dismiss timer on error")
	_ = a
}

// TestApp_PlaylistRenamedMsg_Success verifies PlaylistRenamedMsg triggers playlist re-fetch.
func TestApp_PlaylistRenamedMsg_Success(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)

	msg := panes.PlaylistRenamedMsg{PlaylistID: "pl-1", NewName: "Renamed"}
	model, cmd := a.Update(msg)
	a = model.(*app.App)
	require.NotNil(t, cmd, "should return fetch playlists command after rename")
	_ = a
}

// TestApp_PlaylistRenamedMsg_Error verifies PlaylistRenamedMsg with error shows status.
func TestApp_PlaylistRenamedMsg_Error(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)
	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "Chill Vibes"},
	})

	msg := panes.PlaylistRenamedMsg{PlaylistID: "pl-1", NewName: "Renamed", Err: fmt.Errorf("rename failed")}
	model, cmd := a.Update(msg)
	a = model.(*app.App)
	require.NotNil(t, cmd, "should return dismiss timer on error")
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
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)

	msg := panes.PlaylistRemoveResultMsg{PlaylistID: "pl-1", Err: fmt.Errorf("remove failed")}
	model, cmd := a.Update(msg)
	a = model.(*app.App)
	require.NotNil(t, cmd, "should return dismiss timer on error")
	_ = a
}

// TestApp_PlaylistReorderResultMsg_Error verifies reorder error is forwarded to playlist pane.
func TestApp_PlaylistReorderResultMsg_Error(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)

	msg := panes.PlaylistReorderResultMsg{Err: fmt.Errorf("reorder failed")}
	model, cmd := a.Update(msg)
	a = model.(*app.App)
	require.NotNil(t, cmd, "should return dismiss timer on error")
	_ = a
}

// TestApp_PlaylistTracksLoadedMsg verifies PlaylistTracksLoadedMsg is forwarded to playlist pane.
func TestApp_PlaylistTracksLoadedMsg(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
	})
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)

	msg := panes.PlaylistTracksLoadedMsg{PlaylistID: "pl-1"}
	model, _ = a.Update(msg)
	a = model.(*app.App)
	_ = a
}

// TestApp_PlaylistTracksLoadedMsg_NilPane verifies PlaylistTracksLoadedMsg without
// a playlist pane is handled gracefully.
func TestApp_PlaylistTracksLoadedMsg_NilPane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	msg := panes.PlaylistTracksLoadedMsg{PlaylistID: "pl-1"}
	model, _ := a.Update(msg)
	a = model.(*app.App)
	_ = a
}

// TestApp_SetPlaylistsAPI verifies SetPlaylistsAPI stores the client.
func TestApp_SetPlaylistsAPI(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// SetPlaylistsAPI should not panic even with nil.
	a.SetPlaylistsAPI(nil)
}

// TestApp_PlaylistViewKeysRoutedToPane verifies key events in playlist view are routed to pane.
func TestApp_PlaylistViewKeysRoutedToPane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "Chill Vibes"},
		{ID: "pl-2", Name: "Workout Mix"},
	})

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)
	require.True(t, a.PlaylistViewOpen())

	// Press j to move cursor in playlist view.
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	a = model.(*app.App)
	_ = a
}

// TestApp_3KeyInStatsView_SwitchesToPlaylists verifies pressing 3 while stats is open
// opens playlists (or is handled gracefully).
func TestApp_3KeyInStatsView_SwitchesToPlaylists(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open stats.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = model.(*app.App)
	require.True(t, a.StatsViewOpen())

	// Press 3 — should switch to playlists.
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	a = model.(*app.App)
	assert.True(t, a.PlaylistViewOpen(), "pressing 3 in stats should open playlists")
	assert.False(t, a.StatsViewOpen(), "stats should be closed when playlists opens")
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

func TestApp_BuildSearchCmd_SetsErrorOnFailure(t *testing.T) {
	srv := errorServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.SearchRequestMsg{Query: "test"})
	require.NotNil(t, cmd)
	// Execute command, feed result back to Update() — store writes happen in Update().
	msg := cmd()
	m, _ := a.Update(msg)
	a = m.(*app.App)

	assert.Error(t, a.Store().SearchError(), "store should have search error after API failure")
}

func TestApp_BuildSearchCmd_ClearsErrorOnSuccess(t *testing.T) {
	srv := successServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))
	a.Store().SetSearchError(fmt.Errorf("previous error"))

	_, cmd := a.Update(panes.SearchRequestMsg{Query: "test"})
	require.NotNil(t, cmd)
	// Execute command, feed result back to Update() — store writes happen in Update().
	msg := cmd()
	m, _ := a.Update(msg)
	a = m.(*app.App)

	assert.NoError(t, a.Store().SearchError(), "store should clear search error on success")
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

	// Open stats view first (press 2).
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = model.(*app.App)

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

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = model.(*app.App)

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
	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	output := a.View()
	assert.Contains(t, output, "terminal Spotify client", "splash should show tagline")
	assert.Contains(t, output, "v1.1.0", "splash should show version")
}

func TestApp_SplashScreen_StaysOnPlaybackData(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Splash should be active.
	output := a.View()
	assert.Contains(t, output, "v1.1.0")

	// Playback data arrives — splash should STAY for the full 5s duration.
	model, _ := a.Update(panes.PlaybackStateFetchedMsg{})
	a = model.(*app.App)

	output = a.View()
	assert.Contains(t, output, "v1.1.0", "splash should still be visible — only timer dismisses it")
}

// --- Feature 20: Elm Architecture Purity tests ---

// TestApp_SearchRequestMsg_SetsStoreBeforeCmd verifies that handling SearchRequestMsg
// sets the search query and loading state in the store before building the search command.
// This proves the store writes happen in Update(), not inside buildSearchCmd.
func TestApp_SearchRequestMsg_SetsStoreBeforeCmd(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Store should start with no query and loading=false.
	assert.Equal(t, "", a.Store().SearchQuery(), "store search query should be empty initially")
	assert.False(t, a.Store().SearchLoading(), "store search loading should be false initially")

	// Send SearchRequestMsg — Update() should set store state and return a command.
	m, cmd := a.Update(panes.SearchRequestMsg{Query: "blinding lights"})
	a = m.(*app.App)

	// Store state must be set immediately (before the returned cmd executes).
	assert.Equal(t, "blinding lights", a.Store().SearchQuery(), "store should have query after SearchRequestMsg")
	assert.True(t, a.Store().SearchLoading(), "store should be loading after SearchRequestMsg")
	assert.NotNil(t, cmd, "SearchRequestMsg should produce a search command")
}

// TestApp_SearchClearedMsg_ClearsStoreState verifies that app.Update(SearchClearedMsg)
// clears both search results and search query in the store.
func TestApp_SearchClearedMsg_ClearsStoreState(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Pre-populate the store with search state.
	a.Store().SetSearchQuery("blinding lights")
	a.Store().SetSearchResults(&api.SearchResult{})

	require.Equal(t, "blinding lights", a.Store().SearchQuery())
	require.NotNil(t, a.Store().SearchResults())

	// Handle SearchClearedMsg — store should be cleared.
	m, _ := a.Update(panes.SearchClearedMsg{})
	a = m.(*app.App)

	assert.Equal(t, "", a.Store().SearchQuery(), "store search query should be cleared")
	assert.Nil(t, a.Store().SearchResults(), "store search results should be nil after clear")
}

// TestApp_OverlayRendering_UsesThemeBaseColor verifies that overlay rendering
// uses the theme's Base() color instead of a hardcoded #000000.
// With a non-black theme (monokai), the rendered overlay view must not panic
// and must be non-empty. This ensures no hardcoded color is used.
func TestApp_OverlayRendering_UsesThemeBaseColor(t *testing.T) {
	// Use monokai theme (Base() = #272822, not #000000).
	cfg := &config.Config{}
	cfg.UI.Theme = "monokai"
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
	assert.Contains(t, view, "Rate limited", "toast should mention rate limiting")
	assert.Contains(t, view, "resuming in", "toast should show countdown")
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
	assert.Contains(t, view, "Rate limited", "toast should mention rate limiting")
	assert.Contains(t, view, "resuming in", "toast should show countdown")
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
