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

// makeTopArtists returns a store with 10 fake top artists for short_term.
func makeTopArtists() *state.Store {
	s := state.New()
	artists := []domain.FullArtist{
		{ID: "a1", Name: "The Weeknd", URI: "spotify:artist:a1", Popularity: 95, Followers: domain.ArtistFollowers{Total: 75000000}, Genres: []string{"pop", "r&b"}},
		{ID: "a2", Name: "Dua Lipa", URI: "spotify:artist:a2", Popularity: 90, Followers: domain.ArtistFollowers{Total: 42000000}, Genres: []string{"pop"}},
		{ID: "a3", Name: "Ed Sheeran", URI: "spotify:artist:a3", Popularity: 88, Followers: domain.ArtistFollowers{Total: 110000000}, Genres: []string{"pop"}},
		{ID: "a4", Name: "Olivia Rodrigo", URI: "spotify:artist:a4", Popularity: 85, Followers: domain.ArtistFollowers{Total: 35000000}, Genres: []string{"pop"}},
		{ID: "a5", Name: "Lil Nas X", URI: "spotify:artist:a5", Popularity: 72, Followers: domain.ArtistFollowers{Total: 18000000}, Genres: []string{"hip hop", "pop"}},
		{ID: "a6", Name: "Justin Bieber", URI: "spotify:artist:a6", Popularity: 82, Followers: domain.ArtistFollowers{Total: 90000000}, Genres: []string{"pop"}},
		{ID: "a7", Name: "Doja Cat", URI: "spotify:artist:a7", Popularity: 78, Followers: domain.ArtistFollowers{Total: 25000000}, Genres: []string{"pop", "hip hop"}},
		{ID: "a8", Name: "Drake", URI: "spotify:artist:a8", Popularity: 93, Followers: domain.ArtistFollowers{Total: 85000000}, Genres: []string{"hip hop", "r&b"}},
		{ID: "a9", Name: "Taylor Swift", URI: "spotify:artist:a9", Popularity: 97, Followers: domain.ArtistFollowers{Total: 120000000}, Genres: []string{"pop"}},
		{ID: "a10", Name: "Billie Eilish", URI: "spotify:artist:a10", Popularity: 92, Followers: domain.ArtistFollowers{Total: 95000000}, Genres: []string{"pop"}},
	}
	s.SetTopArtists("short_term", artists)
	s.SetTopTracks("short_term", []domain.Track{})
	s.StampStatsFetchedAt("short_term")
	return s
}

// TestTopArtistsPane_View_Artists verifies golden snapshot of TopArtistsPane
// with 10 artists loaded at normal width (80×24), short_term range.
func TestTopArtistsPane_View_Artists(t *testing.T) {
	s := makeTopArtists()
	th := theme.Load("black")

	pane := panes.NewTopArtistsPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestTopArtistsPane_View_EmptyState verifies golden snapshot when no data exists.
func TestTopArtistsPane_View_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	pane := panes.NewTopArtistsPane(s, th, false)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestTopArtistsPane_View_LongTerm verifies golden snapshot with long_term
// time range ("all time" label).
func TestTopArtistsPane_View_LongTerm(t *testing.T) {
	s := makeTopArtists()
	// Add long_term data
	s.SetTopArtists("long_term", []domain.FullArtist{
		{ID: "l1", Name: "Classic Band", URI: "spotify:artist:l1", Popularity: 60, Followers: domain.ArtistFollowers{Total: 5000000}, Genres: []string{"rock"}},
		{ID: "l2", Name: "Timeless Artist", URI: "spotify:artist:l2", Popularity: 45, Followers: domain.ArtistFollowers{Total: 1200000}, Genres: []string{"folk"}},
	})
	s.SetTopTracks("long_term", []domain.Track{})
	s.StampStatsFetchedAt("long_term")

	th := theme.Load("black")
	pane := panes.NewTopArtistsPane(s, th, true)
	pane.SetSize(78, 10)

	// Cycle twice: short → medium → long
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestTopArtistsPane_View_Narrow verifies golden snapshot at narrow width (40×24).
func TestTopArtistsPane_View_Narrow(t *testing.T) {
	s := makeTopArtists()
	th := theme.Load("black")

	pane := panes.NewTopArtistsPane(s, th, true)
	pane.SetSize(38, 10)

	tm := goldentest.NewPaneTest(t, pane, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestTopArtistsPane_View_FilterActive verifies golden snapshot with filter active
// and filtering artists by name.
func TestTopArtistsPane_View_FilterActive(t *testing.T) {
	s := makeTopArtists()
	th := theme.Load("black")

	pane := panes.NewTopArtistsPane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f', then type "Weeknd" to filter by artist name
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "Weeknd" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestTopArtistsPane_View_FilterActive_NoMatches verifies golden snapshot with
// filter active but no matching results.
func TestTopArtistsPane_View_FilterActive_NoMatches(t *testing.T) {
	s := makeTopArtists()
	th := theme.Load("black")

	pane := panes.NewTopArtistsPane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f', then type query that matches nothing
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "zzzznomatch" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
