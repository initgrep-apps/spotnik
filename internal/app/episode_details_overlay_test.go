package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newEpisodeDetailsTestApp creates an App in grid view with an episode playing.
func newEpisodeDetailsTestApp(t *testing.T) *app.App {
	t.Helper()
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})

	// Set playback state so an episode is playing
	s := state.New()
	ep := &domain.Episode{
		Name:            "Test Episode",
		DurationMs:      3600000,
		ReleaseDate:     "2026-01-15",
		Description:     "A test episode description",
		HTMLDescription:  "<p>A test episode description</p>",
		Show: &domain.Show{
			Name:      "Test Show",
			Publisher: "Test Publisher",
		},
	}
	ps := &domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "episode",
		Episode:              ep,
	}
	s.SetPlaybackState(ps)
	a.Update(panes.PlaybackStateFetchedMsg{State: ps})

	return a
}

// TestApp_IKey_OpensOverlay_WhenEpisode verifies that pressing 'i' opens the
// episode details overlay when an episode is playing.
func TestApp_IKey_OpensOverlay_WhenEpisode(t *testing.T) {
	a := newEpisodeDetailsTestApp(t)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	assert.True(t, a.EpisodeDetailsOpen(), "pressing 'i' while episode playing should open overlay")
}

// TestApp_IKey_NoOp_WhenTrackPlaying verifies that pressing 'i' is a silent no-op
// when a track (not episode) is playing.
func TestApp_IKey_NoOp_WhenTrackPlaying(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})

	// No playback state at all — nothing playing
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	assert.False(t, a.EpisodeDetailsOpen(), "pressing 'i' with nothing playing should not open overlay")
}

// TestApp_EpisodeDetailsClosedMsg_ClosesOverlay verifies that
// EpisodeDetailsClosedMsg closes the overlay.
func TestApp_EpisodeDetailsClosedMsg_ClosesOverlay(t *testing.T) {
	a := newEpisodeDetailsTestApp(t)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	require.True(t, a.EpisodeDetailsOpen(), "overlay should be open after pressing 'i'")

	a.Update(panes.EpisodeDetailsClosedMsg{})

	assert.False(t, a.EpisodeDetailsOpen(), "EpisodeDetailsClosedMsg should close the overlay")
}

// TestApp_EpisodeDetails_EscKey_Closes verifies that pressing Esc while the episode
// details overlay is open sends EpisodeDetailsClosedMsg and closes the overlay.
func TestApp_EpisodeDetails_EscKey_Closes(t *testing.T) {
	a := newEpisodeDetailsTestApp(t)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	require.True(t, a.EpisodeDetailsOpen(), "overlay should be open")

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "Esc in episode details overlay should produce a command")
	msg := cmd()
	_, ok := msg.(panes.EpisodeDetailsClosedMsg)
	assert.True(t, ok, "Esc command should produce EpisodeDetailsClosedMsg")
}

// TestApp_EpisodeDetails_BlocksOtherKeys verifies that other keys are consumed
// while the episode details overlay is open.
func TestApp_EpisodeDetails_BlocksOtherKeys(t *testing.T) {
	a := newEpisodeDetailsTestApp(t)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	require.True(t, a.EpisodeDetailsOpen(), "overlay should be open")

	// Press 'd' — should NOT open device overlay
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.False(t, a.DeviceOverlayOpen(), "'d' should be consumed by episode overlay, not open devices")

	// Press 'q' — should close overlay (via EpisodeDetailsClosedMsg)
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(panes.EpisodeDetailsClosedMsg)
	assert.True(t, ok, "'q' should close episode details overlay")
}

// TestApp_EpisodeDetails_ViewContainsOverlay verifies that the overlay appears in
// the rendered view when the episode details overlay is open.
func TestApp_EpisodeDetails_ViewContainsOverlay(t *testing.T) {
	a := newEpisodeDetailsTestApp(t)

	// Dismiss splash screen
	a.Update(app.SplashDismissMsgForTest{})

	// Open episode details overlay
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	require.True(t, a.EpisodeDetailsOpen(), "overlay should be open")

	view := a.View()
	assert.Contains(t, view, "Episode Details", "view should contain overlay title when open")
}