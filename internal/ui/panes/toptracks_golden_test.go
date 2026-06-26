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

// makeTopTracks returns a store with 10 fake top tracks for short_term.
func makeTopTracks() *state.Store {
	s := state.New()
	tracks := []domain.Track{
		{ID: "tt1", Name: "Blinding Lights", URI: "spotify:track:tt1", DurationMs: 202000, Artists: []domain.Artist{{Name: "The Weeknd"}}},
		{ID: "tt2", Name: "Save Your Tears", URI: "spotify:track:tt2", DurationMs: 215000, Artists: []domain.Artist{{Name: "The Weeknd"}}},
		{ID: "tt3", Name: "Levitating", URI: "spotify:track:tt3", DurationMs: 203000, Artists: []domain.Artist{{Name: "Dua Lipa"}}},
		{ID: "tt4", Name: "Good 4 U", URI: "spotify:track:tt4", DurationMs: 178000, Artists: []domain.Artist{{Name: "Olivia Rodrigo"}}},
		{ID: "tt5", Name: "Montero", URI: "spotify:track:tt5", DurationMs: 143000, Artists: []domain.Artist{{Name: "Lil Nas X"}}},
		{ID: "tt6", Name: "Stay", URI: "spotify:track:tt6", DurationMs: 142000, Artists: []domain.Artist{{Name: "The Kid LAROI"}, {Name: "Justin Bieber"}}},
		{ID: "tt7", Name: "Bad Habits", URI: "spotify:track:tt7", DurationMs: 231000, Artists: []domain.Artist{{Name: "Ed Sheeran"}}},
		{ID: "tt8", Name: "Peaches", URI: "spotify:track:tt8", DurationMs: 199000, Artists: []domain.Artist{{Name: "Justin Bieber"}}},
		{ID: "tt9", Name: "Kiss Me More", URI: "spotify:track:tt9", DurationMs: 216000, Artists: []domain.Artist{{Name: "Doja Cat"}, {Name: "SZA"}}},
		{ID: "tt10", Name: "Industry Baby", URI: "spotify:track:tt10", DurationMs: 222000, Artists: []domain.Artist{{Name: "Lil Nas X"}, {Name: "Jack Harlow"}}},
	}
	s.SetTopTracks("short_term", tracks)
	s.SetTopArtists("short_term", []domain.FullArtist{})
	s.StampStatsFetchedAt("short_term")
	return s
}

// TestTopTracksPane_View_Tracks verifies golden snapshot of TopTracksPane
// with 10 tracks loaded at normal width (80×24), short_term range.
func TestTopTracksPane_View_Tracks(t *testing.T) {
	s := makeTopTracks()
	th := theme.Load("black")

	pane := panes.NewTopTracksPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestTopTracksPane_View_EmptyState verifies golden snapshot when no data exists.
func TestTopTracksPane_View_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	pane := panes.NewTopTracksPane(s, th, false)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestTopTracksPane_View_MediumTerm verifies golden snapshot with medium_term
// time range label ("past 6 months").
func TestTopTracksPane_View_MediumTerm(t *testing.T) {
	s := makeTopTracks()
	// Add medium_term data
	s.SetTopTracks("medium_term", []domain.Track{
		{ID: "m1", Name: "Medium Track One", URI: "spotify:track:m1", DurationMs: 200000, Artists: []domain.Artist{{Name: "Mid Artist"}}},
		{ID: "m2", Name: "Medium Track Two", URI: "spotify:track:m2", DurationMs: 210000, Artists: []domain.Artist{{Name: "Mid Artist"}}},
	})
	s.SetTopArtists("medium_term", []domain.FullArtist{})
	s.StampStatsFetchedAt("medium_term")

	th := theme.Load("black")
	pane := panes.NewTopTracksPane(s, th, true)
	pane.SetSize(78, 10)

	// Cycle to medium_term via 'g' key
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestTopTracksPane_View_Narrow verifies golden snapshot at narrow width (40×24).
func TestTopTracksPane_View_Narrow(t *testing.T) {
	s := makeTopTracks()
	th := theme.Load("black")

	pane := panes.NewTopTracksPane(s, th, true)
	pane.SetSize(38, 10)

	tm := goldentest.NewPaneTest(t, pane, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestTopTracksPane_View_FilterActive verifies golden snapshot with filter active
// and filtering tracks by artist name.
func TestTopTracksPane_View_FilterActive(t *testing.T) {
	s := makeTopTracks()
	th := theme.Load("black")

	pane := panes.NewTopTracksPane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f', then type "Weeknd" to filter by artist
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "Weeknd" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestTopTracksPane_View_FilterActive_NoMatches verifies golden snapshot with
// filter active but no matching results.
func TestTopTracksPane_View_FilterActive_NoMatches(t *testing.T) {
	s := makeTopTracks()
	th := theme.Load("black")

	pane := panes.NewTopTracksPane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f', then type query that matches nothing
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "zzzznomatch" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
