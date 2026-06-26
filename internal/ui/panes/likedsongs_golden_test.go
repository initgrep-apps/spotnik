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

// TestLikedSongsPane_View_Tracks verifies golden snapshot of LikedSongsPane
// with loaded tracks at normal width (80×24).
func TestLikedSongsPane_View_Tracks(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetLikedTracks([]domain.SavedTrack{
		{
			Track: domain.Track{
				ID:         "t1",
				Name:       "Blinding Lights",
				URI:        "spotify:track:t1",
				DurationMs: 202000,
				Artists:    []domain.Artist{{Name: "The Weeknd"}},
			},
		},
		{
			Track: domain.Track{
				ID:         "t2",
				Name:       "Save Your Tears",
				URI:        "spotify:track:t2",
				DurationMs: 215000,
				Artists:    []domain.Artist{{Name: "The Weeknd"}},
			},
		},
		{
			Track: domain.Track{
				ID:         "t3",
				Name:       "Levitating",
				URI:        "spotify:track:t3",
				DurationMs: 203000,
				Artists:    []domain.Artist{{Name: "Dua Lipa"}},
			},
		},
	})

	pane := panes.NewLikedSongsPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestLikedSongsPane_View_EmptyState verifies golden snapshot when no liked songs exist.
func TestLikedSongsPane_View_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	pane := panes.NewLikedSongsPane(s, th, false)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestLikedSongsPane_View_Narrow verifies golden snapshot at narrow width (40×24).
func TestLikedSongsPane_View_Narrow(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetLikedTracks([]domain.SavedTrack{
		{
			Track: domain.Track{
				ID:         "t1",
				Name:       "Blinding Lights",
				URI:        "spotify:track:t1",
				DurationMs: 202000,
				Artists:    []domain.Artist{{Name: "The Weeknd"}},
			},
		},
		{
			Track: domain.Track{
				ID:         "t2",
				Name:       "Save Your Tears",
				URI:        "spotify:track:t2",
				DurationMs: 215000,
				Artists:    []domain.Artist{{Name: "The Weeknd"}},
			},
		},
	})

	pane := panes.NewLikedSongsPane(s, th, true)
	pane.SetSize(38, 10)

	tm := goldentest.NewPaneTest(t, pane, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestLikedSongsPane_View_FilterActive verifies golden snapshot with filter active
// and filtering by track name or artist.
func TestLikedSongsPane_View_FilterActive(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetLikedTracks([]domain.SavedTrack{
		{
			Track: domain.Track{
				ID:         "t1",
				Name:       "Blinding Lights",
				URI:        "spotify:track:t1",
				DurationMs: 202000,
				Artists:    []domain.Artist{{Name: "The Weeknd"}},
			},
		},
		{
			Track: domain.Track{
				ID:         "t2",
				Name:       "Save Your Tears",
				URI:        "spotify:track:t2",
				DurationMs: 215000,
				Artists:    []domain.Artist{{Name: "The Weeknd"}},
			},
		},
		{
			Track: domain.Track{
				ID:         "t3",
				Name:       "Levitating",
				URI:        "spotify:track:t3",
				DurationMs: 203000,
				Artists:    []domain.Artist{{Name: "Dua Lipa"}},
			},
		},
	})

	pane := panes.NewLikedSongsPane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f', then type "Weeknd" to filter by artist
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	for _, r := range "Weeknd" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
