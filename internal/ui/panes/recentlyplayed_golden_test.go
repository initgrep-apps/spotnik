package panes_test

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// makeRecentlyPlayed returns a store with 5 recently played tracks.
func makeRecentlyPlayed() *state.Store {
	s := state.New()
	now := time.Now()
	s.SetRecentlyPlayed([]domain.PlayHistory{
		{
			Track:    domain.Track{ID: "rp1", Name: "Blinding Lights", URI: "spotify:track:rp1", DurationMs: 202000, Artists: []domain.Artist{{Name: "The Weeknd"}}},
			PlayedAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
		{
			Track:    domain.Track{ID: "rp2", Name: "Save Your Tears", URI: "spotify:track:rp2", DurationMs: 215000, Artists: []domain.Artist{{Name: "The Weeknd"}}},
			PlayedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		},
		{
			Track:    domain.Track{ID: "rp3", Name: "Levitating", URI: "spotify:track:rp3", DurationMs: 203000, Artists: []domain.Artist{{Name: "Dua Lipa"}}},
			PlayedAt: now.Add(-1 * 24 * time.Hour).Format(time.RFC3339),
		},
		{
			Track:    domain.Track{ID: "rp4", Name: "Good 4 U", URI: "spotify:track:rp4", DurationMs: 178000, Artists: []domain.Artist{{Name: "Olivia Rodrigo"}}},
			PlayedAt: now.Add(-3 * 24 * time.Hour).Format(time.RFC3339),
		},
		{
			Track:    domain.Track{ID: "rp5", Name: "Montero", URI: "spotify:track:rp5", DurationMs: 143000, Artists: []domain.Artist{{Name: "Lil Nas X"}}},
			PlayedAt: now.Add(-2 * 7 * 24 * time.Hour).Format(time.RFC3339),
		},
	})
	return s
}

// TestRecentlyPlayedPane_View_Tracks verifies golden snapshot of RecentlyPlayedPane
// with 5 recently played tracks with timestamps at normal width (80×24).
func TestRecentlyPlayedPane_View_Tracks(t *testing.T) {
	s := makeRecentlyPlayed()
	th := theme.Load("black")

	pane := panes.NewRecentlyPlayedPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestRecentlyPlayedPane_View_EmptyState verifies golden snapshot when no data exists.
func TestRecentlyPlayedPane_View_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	pane := panes.NewRecentlyPlayedPane(s, th, false)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestRecentlyPlayedPane_View_Narrow verifies golden snapshot at narrow width (40×24).
func TestRecentlyPlayedPane_View_Narrow(t *testing.T) {
	s := makeRecentlyPlayed()
	th := theme.Load("black")

	pane := panes.NewRecentlyPlayedPane(s, th, true)
	pane.SetSize(38, 10)

	tm := goldentest.NewPaneTest(t, pane, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestRecentlyPlayedPane_View_FilterActive verifies golden snapshot with filter active
// and filtering by track name or artist.
func TestRecentlyPlayedPane_View_FilterActive(t *testing.T) {
	s := makeRecentlyPlayed()
	th := theme.Load("black")

	pane := panes.NewRecentlyPlayedPane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f', then type "Weeknd" to filter by artist
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "Weeknd" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestRecentlyPlayedPane_View_FilterActive_NoMatches verifies golden snapshot with
// filter active but no matching results.
func TestRecentlyPlayedPane_View_FilterActive_NoMatches(t *testing.T) {
	s := makeRecentlyPlayed()
	th := theme.Load("black")

	pane := panes.NewRecentlyPlayedPane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f', then type query that matches nothing
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "zzzznomatch" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
