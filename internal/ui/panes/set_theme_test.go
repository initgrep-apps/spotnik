package panes

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueuePane_SetTheme_UpdatesColors verifies that calling SetTheme with a
// different theme causes the next View() call to use the new theme colors.
func TestQueuePane_SetTheme_UpdatesColors(t *testing.T) {
	s := state.New()
	th1 := theme.Load("black")
	pane := NewQueuePane(s, th1, false)
	pane.SetSize(80, 20)

	th2 := theme.Load("dracula")
	require.NotNil(t, th2)

	// Switching theme must not panic.
	pane.SetTheme(th2)

	// After SetTheme, the pane's theme field should be updated.
	assert.Equal(t, th2, pane.theme)

	// View() must succeed without panicking.
	view := pane.View()
	assert.NotEmpty(t, view)
}

// TestAlbumsPane_SetTheme_UpdatesColors verifies SetTheme on AlbumsPane.
func TestAlbumsPane_SetTheme_UpdatesColors(t *testing.T) {
	s := state.New()
	th1 := theme.Load("black")
	pane := NewAlbumsPane(s, th1, false)
	pane.SetSize(80, 20)

	th2 := theme.Load("dracula")
	pane.SetTheme(th2)

	assert.Equal(t, th2, pane.theme)
	assert.NotEmpty(t, pane.View())
}

// TestLikedSongsPane_SetTheme_UpdatesColors verifies SetTheme on LikedSongsPane.
func TestLikedSongsPane_SetTheme_UpdatesColors(t *testing.T) {
	s := state.New()
	th1 := theme.Load("black")
	pane := NewLikedSongsPane(s, th1, false)
	pane.SetSize(80, 20)

	th2 := theme.Load("dracula")
	pane.SetTheme(th2)

	assert.Equal(t, th2, pane.theme)
	assert.NotEmpty(t, pane.View())
}

// TestTopTracksPane_SetTheme_UpdatesColors verifies SetTheme on TopTracksPane.
func TestTopTracksPane_SetTheme_UpdatesColors(t *testing.T) {
	s := state.New()
	th1 := theme.Load("black")
	pane := NewTopTracksPane(s, th1, false)
	pane.SetSize(80, 20)

	th2 := theme.Load("dracula")
	pane.SetTheme(th2)

	assert.Equal(t, th2, pane.theme)
	assert.NotEmpty(t, pane.View())
}

// TestTopArtistsPane_SetTheme_UpdatesColors verifies SetTheme on TopArtistsPane.
func TestTopArtistsPane_SetTheme_UpdatesColors(t *testing.T) {
	s := state.New()
	th1 := theme.Load("black")
	pane := NewTopArtistsPane(s, th1, false)
	pane.SetSize(80, 20)

	th2 := theme.Load("dracula")
	pane.SetTheme(th2)

	assert.Equal(t, th2, pane.theme)
	assert.NotEmpty(t, pane.View())
}

// TestRecentlyPlayedPane_SetTheme_UpdatesColors verifies SetTheme on RecentlyPlayedPane.
func TestRecentlyPlayedPane_SetTheme_UpdatesColors(t *testing.T) {
	s := state.New()
	th1 := theme.Load("black")
	pane := NewRecentlyPlayedPane(s, th1, false)
	pane.SetSize(80, 20)

	th2 := theme.Load("dracula")
	pane.SetTheme(th2)

	assert.Equal(t, th2, pane.theme)
	assert.NotEmpty(t, pane.View())
}

// TestPlaylistsPane_SetTheme_UpdatesColors verifies SetTheme on PlaylistsPane.
func TestPlaylistsPane_SetTheme_UpdatesColors(t *testing.T) {
	s := state.New()
	th1 := theme.Load("black")
	pane := NewPlaylistsPane(s, th1, false)
	pane.SetSize(80, 20)

	th2 := theme.Load("dracula")
	pane.SetTheme(th2)

	assert.Equal(t, th2, pane.theme)
	assert.NotEmpty(t, pane.View())
}

// TestNetworkLogPane_SetTheme_UpdatesColors verifies SetTheme on NetworkLogPane.
func TestNetworkLogPane_SetTheme_UpdatesColors(t *testing.T) {
	s := state.New()
	th1 := theme.Load("black")
	pane := NewNetworkLogPane(s, th1)
	pane.SetSize(80, 20)

	th2 := theme.Load("dracula")
	pane.SetTheme(th2)

	assert.Equal(t, th2, pane.theme)
	assert.NotEmpty(t, pane.View())
}

// TestNowPlayingPane_SetTheme verifies SetTheme on NowPlayingPane.
func TestNowPlayingPane_SetTheme(t *testing.T) {
	s := state.New()
	th1 := theme.Load("black")
	pane := NewNowPlayingPane(s, th1, false)
	pane.SetSize(80, 20)

	th2 := theme.Load("dracula")
	pane.SetTheme(th2)

	assert.Equal(t, th2, pane.theme)
	assert.NotEmpty(t, pane.View())
}

// TestRequestFlowPane_SetTheme verifies SetTheme on RequestFlowPane.
func TestRequestFlowPane_SetTheme(t *testing.T) {
	s := state.New()
	th1 := theme.Load("black")
	pane := NewRequestFlowPane(s, th1)
	pane.SetSize(80, 20)

	th2 := theme.Load("dracula")
	pane.SetTheme(th2)

	assert.Equal(t, th2, pane.theme)
	assert.NotEmpty(t, pane.View())
}

// TestQueuePane_SetTheme_PreservesRows verifies that switching theme does not
// clear the table rows — RebuildTableTheme must copy existing rows to the new table.
func TestQueuePane_SetTheme_PreservesRows(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.Track{
		{ID: "t1", Name: "Blinding Lights", Artists: []domain.Artist{{Name: "The Weeknd"}}},
		{ID: "t2", Name: "Starboy", Artists: []domain.Artist{{Name: "The Weeknd"}}},
	})
	pane := NewQueuePane(s, theme.Load("black"), true)
	pane.SetSize(80, 20)

	view := pane.View()
	assert.Contains(t, view, "Blinding Lights", "rows must be present before theme switch")

	pane.SetTheme(theme.Load("dracula"))
	pane.SetSize(80, 20)

	view = pane.View()
	assert.Contains(t, view, "Blinding Lights", "rows must survive theme switch")
	assert.Contains(t, view, "Starboy", "all rows must survive theme switch")
}

// TestSearchOverlay_SetTheme verifies that SetTheme on SearchOverlay
// updates the theme reference without panicking.
func TestSearchOverlay_SetTheme(t *testing.T) {
	th1 := theme.Load("black")
	overlay := NewSearchOverlay(th1)

	th2 := theme.Load("dracula")
	overlay.SetTheme(th2)

	assert.Equal(t, th2, overlay.theme)
}

// TestDeviceOverlay_SetTheme verifies that SetTheme on DeviceOverlay
// updates the theme reference without panicking.
func TestDeviceOverlay_SetTheme(t *testing.T) {
	s := state.New()
	th1 := theme.Load("black")
	overlay := NewDeviceOverlay(s, th1)

	th2 := theme.Load("dracula")
	overlay.SetTheme(th2)

	assert.Equal(t, th2, overlay.theme)
}
