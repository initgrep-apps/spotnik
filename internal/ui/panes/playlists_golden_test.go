package panes_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestPlaylistsPane_View_ListView verifies golden snapshot of PlaylistsPane
// with loaded playlists at normal width (80×24).
func TestPlaylistsPane_View_ListView(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi Beats", URI: "spotify:playlist:pl1", TrackCount: 42,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
		{ID: "pl2", Name: "Best of Coke Studio", URI: "spotify:playlist:pl2", TrackCount: 28,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
		{ID: "pl3", Name: "Soul Vibes", URI: "spotify:playlist:pl3", TrackCount: 15,
			Owner: domain.SimplePlaylistOwner{ID: "other", DisplayName: "Other User"}},
	})
	s.SetUserProfile(domain.UserProfile{ID: "user123"})

	pane := panes.NewPlaylistsPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestPlaylistsPane_View_EmptyState verifies golden snapshot when no playlists exist.
func TestPlaylistsPane_View_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	pane := panes.NewPlaylistsPane(s, th, false)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestPlaylistsPane_View_TrackSubView verifies golden snapshot after pressing Enter
// on a playlist to show track sub-view.
func TestPlaylistsPane_View_TrackSubView(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi Beats", URI: "spotify:playlist:pl1", TrackCount: 3,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
	})
	s.SetUserProfile(domain.UserProfile{ID: "user123"})

	pane := panes.NewPlaylistsPane(s, th, true)
	pane.SetSize(78, 10)

	// Press Enter on playlist 0 to enter track sub-view
	model, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pane = model.(*panes.PlaylistsPane)

	// Load tracks into pane's sub-view
	pane.Update(panes.PlaylistTracksLoadedMsg{
		PlaylistID: "pl1",
		Tracks: []domain.Track{
			{Name: "Calm Morning", URI: "spotify:track:t1", DurationMs: 210000,
				Artists: []domain.Artist{{Name: "Lofi Girl"}}},
			{Name: "Rainy Evening", URI: "spotify:track:t2", DurationMs: 185000,
				Artists: []domain.Artist{{Name: "Chillhop"}}},
			{Name: "Sunset Drive", URI: "spotify:track:t3", DurationMs: 195000,
				Artists: []domain.Artist{{Name: "Jazzhop"}}},
		},
		Total: 3,
	})

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestPlaylistsPane_View_TrackSubView_FilterActive verifies golden snapshot of
// track sub-view with filter activated.
func TestPlaylistsPane_View_TrackSubView_FilterActive(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi Beats", URI: "spotify:playlist:pl1", TrackCount: 3,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
	})
	s.SetUserProfile(domain.UserProfile{ID: "user123"})

	pane := panes.NewPlaylistsPane(s, th, true)
	pane.SetSize(78, 10)

	// Press Enter on playlist 0
	model, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pane = model.(*panes.PlaylistsPane)

	// Load tracks into pane's sub-view
	pane.Update(panes.PlaylistTracksLoadedMsg{
		PlaylistID: "pl1",
		Tracks: []domain.Track{
			{Name: "Calm Morning", URI: "spotify:track:t1", DurationMs: 210000,
				Artists: []domain.Artist{{Name: "Lofi Girl"}}},
			{Name: "Rainy Evening", URI: "spotify:track:t2", DurationMs: 185000,
				Artists: []domain.Artist{{Name: "Chillhop"}}},
		},
		Total: 2,
	})

	// Activate filter in track sub-view with 'f' key
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	for _, r := range "Rainy" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestPlaylistsPane_View_SpotifyOwnedLocked verifies that a Spotify-owned playlist
// shows the locked glyph (◌) in the access column.
func TestPlaylistsPane_View_SpotifyOwnedLocked(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi Beats", URI: "spotify:playlist:pl1", TrackCount: 42,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
		{ID: "spot1", Name: "Relax & Unwind", URI: "spotify:playlist:spot1", TrackCount: 60,
			Owner: domain.SimplePlaylistOwner{ID: "spotify", DisplayName: "Spotify"}},
		{ID: "pl2", Name: "Best of Coke Studio", URI: "spotify:playlist:pl2", TrackCount: 28,
			Owner: domain.SimplePlaylistOwner{ID: "other", DisplayName: "Other User"}},
	})
	s.SetUserProfile(domain.UserProfile{ID: "user123"})

	pane := panes.NewPlaylistsPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestPlaylistsPane_View_Narrow verifies golden snapshot at narrow width (40×24),
// showing column hiding at reduced dimensions.
func TestPlaylistsPane_View_Narrow(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi Beats", URI: "spotify:playlist:pl1", TrackCount: 42,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
		{ID: "pl2", Name: "Best of Coke Studio", URI: "spotify:playlist:pl2", TrackCount: 28,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
		{ID: "pl3", Name: "Soul Vibes", URI: "spotify:playlist:pl3", TrackCount: 15,
			Owner: domain.SimplePlaylistOwner{ID: "other", DisplayName: "Other User"}},
	})
	s.SetUserProfile(domain.UserProfile{ID: "user123"})

	pane := panes.NewPlaylistsPane(s, th, true)
	pane.SetSize(38, 10)

	tm := goldentest.NewPaneTest(t, pane, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestPlaylistsPane_View_FilterActive verifies golden snapshot with filter active
// and filtering by playlist name.
func TestPlaylistsPane_View_FilterActive(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi Beats", URI: "spotify:playlist:pl1", TrackCount: 42,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
		{ID: "pl2", Name: "Best of Coke Studio", URI: "spotify:playlist:pl2", TrackCount: 28,
			Owner: domain.SimplePlaylistOwner{ID: "user123", DisplayName: "Test User"}},
		{ID: "pl3", Name: "Soul Vibes", URI: "spotify:playlist:pl3", TrackCount: 15,
			Owner: domain.SimplePlaylistOwner{ID: "other", DisplayName: "Other User"}},
	})
	s.SetUserProfile(domain.UserProfile{ID: "user123"})

	pane := panes.NewPlaylistsPane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f', then type "LoFi"
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	for _, r := range "LoFi" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
