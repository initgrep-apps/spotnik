package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppNew_ReceivesTheme(t *testing.T) {
	cfg := &config.Config{}
	cfg.UI.Theme = "monokai"

	a := app.New(cfg)
	require.NotNil(t, a)
	assert.Equal(t, "monokai", a.Theme().ID())
}

func TestAppNew_DefaultThemeFallback(t *testing.T) {
	cfg := &config.Config{}
	cfg.UI.Theme = "invalid-theme-id"

	a := app.New(cfg)
	require.NotNil(t, a)
	// Unknown IDs fall back to DefaultThemeID without crashing.
	assert.Equal(t, theme.DefaultThemeID, a.Theme().ID())
}

func TestAppNew_EmptyThemeUsesDefault(t *testing.T) {
	cfg := &config.Config{}
	// cfg.UI.Theme is zero value (empty string)

	a := app.New(cfg)
	require.NotNil(t, a)
	assert.Equal(t, theme.DefaultThemeID, a.Theme().ID())
}

// TestApp_Init_ReturnsBatch verifies Init returns a non-nil batch command.
func TestApp_Init_ReturnsBatch(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg)

	cmd := a.Init()
	assert.NotNil(t, cmd, "Init should return a non-nil batch command")
}

// TestApp_Update_TickMsg_DispatchesFetch verifies that a TickMsg causes the app
// to produce a fetchPlaybackState command (returns a PlaybackStateFetchedMsg).
func TestApp_Update_TickMsg_DispatchesFetch(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg)

	tickMsg := panes.TickMsg{}
	updatedModel, cmd := a.Update(tickMsg)

	assert.NotNil(t, updatedModel)
	assert.NotNil(t, cmd, "tickMsg should produce a follow-up command")
}

// TestApp_PlayerPaneRouting verifies key events are routed to the player pane
// when it is focused.
func TestApp_PlayerPaneRouting(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg)

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
	a := app.New(cfg)

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
// updates the store with new track data.
func TestPollingLoop_FetchesAndUpdatesStore(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg)

	s := a.Store()
	assert.Nil(t, s.PlaybackState(), "store should start empty")

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
	a := app.New(cfg)

	sizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := a.Update(sizeMsg)

	require.NotNil(t, updatedModel)
}

// TestApp_View_EmptyState verifies View renders without crashing when store is empty.
func TestApp_View_EmptyState(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg)

	// Should not panic.
	output := a.View()
	assert.NotEmpty(t, output)
}

// TestApp_StoreIsAccessible verifies that Store() returns the app's store.
func TestApp_StoreIsAccessible(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg)

	s := a.Store()
	require.NotNil(t, s)

	// Verify it's the same store by setting and getting.
	s.SetActiveDevice(&api.Device{Name: "Test Device"})
	assert.Equal(t, "Test Device", a.Store().ActiveDevice().Name)
}
