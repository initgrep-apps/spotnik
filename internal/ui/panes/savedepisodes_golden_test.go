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

// TestSavedEpisodesPane_View_Episodes verifies golden snapshot of SavedEpisodesPane
// with 5 saved episodes at normal dimensions (80×24).
func TestSavedEpisodesPane_View_Episodes(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetSavedEpisodes([]domain.SavedEpisode{
		{Episode: domain.Episode{
			ID: "e1", Name: "AI Revolution", URI: "spotify:episode:e1", DurationMs: 1800000,
			IsPlayable: true, ResumePoint: domain.ResumePoint{ResumePositionMs: 300000, FullyPlayed: false},
			Show: &domain.Show{ID: "s1", Name: "Tech Weekly"},
		}},
		{Episode: domain.Episode{
			ID: "e2", Name: "Deep Sea Mysteries", URI: "spotify:episode:e2", DurationMs: 3600000,
			IsPlayable: true, ResumePoint: domain.ResumePoint{FullyPlayed: true},
			Show: &domain.Show{ID: "s2", Name: "Nature Podcast"},
		}},
		{Episode: domain.Episode{
			ID: "e3", Name: "The History of Jazz", URI: "spotify:episode:e3", DurationMs: 2400000,
			IsPlayable: false,
			Show:       &domain.Show{ID: "s3", Name: "Music Hour"},
		}},
		{Episode: domain.Episode{
			ID: "e4", Name: "Space Exploration", URI: "spotify:episode:e4", DurationMs: 1500000,
			IsPlayable: true,
			Show:       &domain.Show{ID: "s4", Name: "Science Weekly"},
		}},
		{Episode: domain.Episode{
			ID: "e5", Name: "Economic Forecast 2026", URI: "spotify:episode:e5", DurationMs: 2700000,
			IsPlayable: true,
			Show:       &domain.Show{ID: "s5", Name: "Market Watch"},
		}},
	})

	pane := panes.NewSavedEpisodesPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestSavedEpisodesPane_View_EmptyState verifies golden snapshot of SavedEpisodesPane
// with no saved episodes (shows empty state) at normal dimensions (80×24).
func TestSavedEpisodesPane_View_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewSavedEpisodesPane(s, th, false)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestSavedEpisodesPane_View_Narrow verifies golden snapshot of SavedEpisodesPane
// at narrow width (40×24).
func TestSavedEpisodesPane_View_Narrow(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetSavedEpisodes([]domain.SavedEpisode{
		{Episode: domain.Episode{
			ID: "e1", Name: "AI Revolution", URI: "spotify:episode:e1", DurationMs: 1800000, IsPlayable: true,
			Show: &domain.Show{ID: "s1", Name: "Tech Weekly"},
		}},
		{Episode: domain.Episode{
			ID: "e2", Name: "Deep Sea Mysteries", URI: "spotify:episode:e2", DurationMs: 3600000, IsPlayable: true,
			Show: &domain.Show{ID: "s2", Name: "Nature Podcast"},
		}},
	})

	pane := panes.NewSavedEpisodesPane(s, th, true)
	pane.SetSize(38, 10)

	tm := goldentest.NewPaneTest(t, pane, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestSavedEpisodesPane_View_FilterActive verifies golden snapshot of SavedEpisodesPane
// with an active filter filtering episodes by name at normal dimensions (80×24).
func TestSavedEpisodesPane_View_FilterActive(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetSavedEpisodes([]domain.SavedEpisode{
		{Episode: domain.Episode{
			ID: "e1", Name: "AI Revolution", URI: "spotify:episode:e1", DurationMs: 1800000, IsPlayable: true,
			Show: &domain.Show{ID: "s1", Name: "Tech Weekly"},
		}},
		{Episode: domain.Episode{
			ID: "e2", Name: "Deep Sea Mysteries", URI: "spotify:episode:e2", DurationMs: 3600000, IsPlayable: true,
			Show: &domain.Show{ID: "s2", Name: "Nature Podcast"},
		}},
		{Episode: domain.Episode{
			ID: "e3", Name: "AI Ethics", URI: "spotify:episode:e3", DurationMs: 2000000, IsPlayable: true,
			Show: &domain.Show{ID: "s3", Name: "Tech Ethics"},
		}},
		{Episode: domain.Episode{
			ID: "e4", Name: "Space Exploration", URI: "spotify:episode:e4", DurationMs: 1500000, IsPlayable: true,
			Show: &domain.Show{ID: "s4", Name: "Science Weekly"},
		}},
	})

	pane := panes.NewSavedEpisodesPane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f' key, then type "AI" to filter by name.
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	for _, r := range "AI" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
