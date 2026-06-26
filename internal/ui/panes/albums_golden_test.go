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

// TestAlbumsPane_View_AlbumList verifies golden snapshot of AlbumsPane
// with loaded albums at normal width (80×24).
func TestAlbumsPane_View_AlbumList(t *testing.T) {
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
		{
			Album: domain.FullAlbum{
				ID:          "al2",
				Name:        "OK Computer",
				URI:         "spotify:album:al2",
				ReleaseDate: "1997-05-21",
				Artists:     []domain.Artist{{Name: "Radiohead"}},
			},
		},
		{
			Album: domain.FullAlbum{
				ID:          "al3",
				Name:        "In Rainbows",
				URI:         "spotify:album:al3",
				ReleaseDate: "2007-10-10",
				Artists:     []domain.Artist{{Name: "Radiohead"}},
			},
		},
	})

	pane := panes.NewAlbumsPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestAlbumsPane_View_EmptyState verifies golden snapshot when no albums exist.
func TestAlbumsPane_View_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	pane := panes.NewAlbumsPane(s, th, false)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestAlbumsPane_View_TrackSubView verifies golden snapshot after pressing Enter
// on an album to show track sub-view.
func TestAlbumsPane_View_TrackSubView(t *testing.T) {
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
	pane.SetSize(78, 10)

	// Press Enter on album 0 to enter track sub-view
	model, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pane = model.(*panes.AlbumsPane)

	// Wait for debounce, then load tracks into sub-view
	pane.Update(panes.AlbumTracksLoadedMsg{
		AlbumID: "al1",
		Tracks: []domain.Track{
			{Name: "Alone Again", URI: "spotify:track:ta1", DurationMs: 242000,
				Artists: []domain.Artist{{Name: "The Weeknd"}}},
			{Name: "Too Late", URI: "spotify:track:ta2", DurationMs: 239000,
				Artists: []domain.Artist{{Name: "The Weeknd"}}},
			{Name: "Hardest To Love", URI: "spotify:track:ta3", DurationMs: 211000,
				Artists: []domain.Artist{{Name: "The Weeknd"}}},
		},
		HasNext: false,
	})

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestAlbumsPane_View_TrackSubView_FilterActive verifies golden snapshot of
// album track sub-view with filter activated.
func TestAlbumsPane_View_TrackSubView_FilterActive(t *testing.T) {
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
	pane.SetSize(78, 10)

	// Press Enter on album 0
	model, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pane = model.(*panes.AlbumsPane)

	// Wait for debounce, then load tracks
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

	// Activate filter in track sub-view with 'f' key
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	for _, r := range "Too" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestAlbumsPane_View_Narrow verifies golden snapshot at narrow width (40×24).
func TestAlbumsPane_View_Narrow(t *testing.T) {
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
		{
			Album: domain.FullAlbum{
				ID:          "al2",
				Name:        "OK Computer",
				URI:         "spotify:album:al2",
				ReleaseDate: "1997-05-21",
				Artists:     []domain.Artist{{Name: "Radiohead"}},
			},
		},
	})

	pane := panes.NewAlbumsPane(s, th, true)
	pane.SetSize(38, 10)

	tm := goldentest.NewPaneTest(t, pane, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestAlbumsPane_View_FilterActive verifies golden snapshot with filter active
// and filtering albums by name.
func TestAlbumsPane_View_FilterActive(t *testing.T) {
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
		{
			Album: domain.FullAlbum{
				ID:          "al2",
				Name:        "OK Computer",
				URI:         "spotify:album:al2",
				ReleaseDate: "1997-05-21",
				Artists:     []domain.Artist{{Name: "Radiohead"}},
			},
		},
		{
			Album: domain.FullAlbum{
				ID:          "al3",
				Name:        "In Rainbows",
				URI:         "spotify:album:al3",
				ReleaseDate: "2007-10-10",
				Artists:     []domain.Artist{{Name: "Radiohead"}},
			},
		},
	})

	pane := panes.NewAlbumsPane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f', then type "Radiohead" to filter by artist
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	for _, r := range "Radiohead" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
