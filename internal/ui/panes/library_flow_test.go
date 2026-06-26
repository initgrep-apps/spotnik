package panes_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlaylistsDrillDown_EnterThenEsc verifies that pressing Enter on a playlist
// opens the track sub-view, and pressing Esc returns to the list view. Also verifies
// title changes correctly between views.
func TestPlaylistsDrillDown_EnterThenEsc(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi Beats", URI: "spotify:playlist:pl1", TrackCount: 3,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
	})
	s.SetUserProfile(domain.UserProfile{ID: "user123"})

	pane := panes.NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	// Verify initial state: list view
	assert.Equal(t, "Playlists", pane.Title())

	// 1. Press Enter on playlist 0 → should enter track sub-view
	model, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pane = model.(*panes.PlaylistsPane)
	assert.NotNil(t, pane, "pane should still exist after Enter")

	// Title should now show playlist name in track view
	// (pane enters track view immediately, title changes before tracks load)
	_ = cmd // debounce cmd for track fetch — we don't need to process it

	// Load tracks to populate the sub-view
	pane.Update(panes.PlaylistTracksLoadedMsg{
		PlaylistID: "pl1",
		Tracks: []domain.Track{
			{Name: "Calm Morning", URI: "spotify:track:t1", DurationMs: 210000,
				Artists: []domain.Artist{{Name: "Lofi Girl"}}},
			{Name: "Rainy Evening", URI: "spotify:track:t2", DurationMs: 185000,
				Artists: []domain.Artist{{Name: "Chillhop"}}},
		},
		Total:   2,
		HasNext: false,
	})

	// Verify view output now contains track data
	view := pane.View()
	assert.Contains(t, view, "Calm Morning", "track sub-view should show track names")
	assert.Contains(t, view, "Rainy Evening", "track sub-view should show all tracks")

	// 2. Press Esc → should return to list view
	model, cmd = pane.Update(tea.KeyMsg{Type: tea.KeyEsc})
	pane = model.(*panes.PlaylistsPane)
	assert.NotNil(t, cmd, "Esc in track view should emit PlaylistTrackViewClosedMsg")

	// Verify view output no longer shows tracks
	view = pane.View()
	assert.NotContains(t, view, "Calm Morning", "list view should not show track names")
	assert.Contains(t, view, "LoFi Beats", "list view should show playlist names")

	// Verify title returns to "Playlists"
	assert.Equal(t, "Playlists", pane.Title())
}

// TestAlbumsDrillDown_EnterThenEsc verifies that pressing Enter on an album
// opens the track sub-view, and pressing Esc returns to the list view.
func TestAlbumsDrillDown_EnterThenEsc(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetSavedAlbums([]domain.SavedAlbum{
		{
			Album: domain.FullAlbum{
				ID:          "al1",
				Name:        "After Hours",
				URI:         "spotify:album:al1",
				ReleaseDate: "2020-03-20",
				Artists:     []domain.Artist{{Name: "The Weeknd"}},
			},
		},
	})

	pane := panes.NewAlbumsPane(s, th, true)
	pane.SetSize(80, 20)

	// Verify initial state: list view
	assert.Equal(t, "Albums", pane.Title())

	// 1. Press Enter on album 0 → should enter track sub-view
	model, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pane = model.(*panes.AlbumsPane)
	assert.NotNil(t, pane, "pane should still exist after Enter")
	_ = cmd // debounce cmd

	// Load tracks to populate the sub-view
	pane.Update(panes.AlbumTracksLoadedMsg{
		AlbumID: "al1",
		Tracks: []domain.Track{
			{Name: "Alone Again", URI: "spotify:track:ta1", DurationMs: 242000,
				Artists: []domain.Artist{{Name: "The Weeknd"}}},
			{Name: "Too Late", URI: "spotify:track:ta2", DurationMs: 239000,
				Artists: []domain.Artist{{Name: "The Weeknd"}}},
		},
		HasNext: false,
	})

	view := pane.View()
	assert.Contains(t, view, "Alone Again", "track sub-view should show track names")

	// 2. Press Esc → should return to list view
	model, cmd = pane.Update(tea.KeyMsg{Type: tea.KeyEsc})
	pane = model.(*panes.AlbumsPane)
	assert.NotNil(t, cmd, "Esc in track view should emit AlbumTrackViewClosedMsg")

	view = pane.View()
	assert.NotContains(t, view, "Alone Again", "list view should not show track names")
	assert.Contains(t, view, "After Hours", "list view should show album names")
	assert.Equal(t, "Albums", pane.Title())
}

// TestPlaylistsPane_TrackView_DeleteTrack verifies that pressing 'x' in the
// track sub-view emits a PlaylistRemoveRequestMsg with the correct PlaylistID
// and TrackURI for the selected track.
func TestPlaylistsPane_TrackView_DeleteTrack(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi Beats", URI: "spotify:playlist:pl1", TrackCount: 3,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
	})
	s.SetUserProfile(domain.UserProfile{ID: "user123"})

	pane := panes.NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	// Enter track sub-view
	model, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pane = model.(*panes.PlaylistsPane)

	// Load tracks
	pane.Update(panes.PlaylistTracksLoadedMsg{
		PlaylistID: "pl1",
		Tracks: []domain.Track{
			{Name: "Track 1", URI: "spotify:track:t1", DurationMs: 200000,
				Artists: []domain.Artist{{Name: "Artist 1"}}},
			{Name: "Track 2", URI: "spotify:track:t2", DurationMs: 200000,
				Artists: []domain.Artist{{Name: "Artist 2"}}},
		},
		Total:   2,
		HasNext: false,
	})

	// Navigate cursor to track 1 (index 0 → index 1 using 'j')
	pane.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Send 'x' → assert PlaylistRemoveRequestMsg cmd produced
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	require.NotNil(t, cmd, "'x' in track view should produce a command")

	// Execute the command to get the message
	msg := cmd()
	removeMsg, ok := msg.(panes.PlaylistRemoveRequestMsg)
	require.True(t, ok, "expected PlaylistRemoveRequestMsg, got %T", msg)
	assert.Equal(t, "pl1", removeMsg.PlaylistID)
	assert.Equal(t, "spotify:track:t2", removeMsg.TrackURI)
}

// TestPlaylistsPane_EnterOnSpotifyOwned_EmitsAccessDenied verifies that pressing
// Enter on a Spotify-owned playlist emits PlaylistAccessDeniedMsg and does NOT
// open the track sub-view.
func TestPlaylistsPane_EnterOnSpotifyOwned_EmitsAccessDenied(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "My Playlist", URI: "spotify:playlist:pl1", TrackCount: 10,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
		{ID: "spot1", Name: "Relax & Unwind", URI: "spotify:playlist:spot1", TrackCount: 60,
			Owner: domain.SimplePlaylistOwner{ID: "spotify", DisplayName: "Spotify"}},
	})
	s.SetUserProfile(domain.UserProfile{ID: "user123"})

	pane := panes.NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	// Navigate to Spotify-owned playlist (index 1)
	pane.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Press Enter on Spotify-owned playlist
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter on Spotify-owned playlist should produce a command")

	// Execute the command to get the message
	msg := cmd()
	_, ok := msg.(panes.PlaylistAccessDeniedMsg)
	require.True(t, ok, "expected PlaylistAccessDeniedMsg, got %T", msg)

	// Verify sub-view did NOT open (title should still be "Playlists")
	assert.Equal(t, "Playlists", pane.Title(), "title should remain 'Playlists' for list view")
}
