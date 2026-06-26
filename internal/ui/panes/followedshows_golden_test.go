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

// TestFollowedShowsPane_View_Shows verifies golden snapshot of FollowedShowsPane
// with 3 followed shows loaded at normal dimensions (80×24).
func TestFollowedShowsPane_View_Shows(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetFollowedShows([]domain.SavedShow{
		{Show: domain.Show{ID: "s1", Name: "Tech Weekly", Publisher: "Tech Media", TotalEpisodes: 42, MediaType: "audio"}},
		{Show: domain.Show{ID: "s2", Name: "The Daily", Publisher: "NYT", TotalEpisodes: 120, MediaType: "mixed"}},
		{Show: domain.Show{ID: "s3", Name: "Science Friday", Publisher: "NPR", TotalEpisodes: 300, MediaType: "video"}},
	})

	pane := panes.NewFollowedShowsPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestFollowedShowsPane_View_EmptyState verifies golden snapshot of FollowedShowsPane
// with no followed shows (shows empty state) at normal dimensions (80×24).
func TestFollowedShowsPane_View_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewFollowedShowsPane(s, th, false)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestFollowedShowsPane_View_EpisodeSubView verifies golden snapshot of FollowedShowsPane
// after pressing Enter on a show and receiving episode data at normal dimensions (80×24).
func TestFollowedShowsPane_View_EpisodeSubView(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetFollowedShows([]domain.SavedShow{
		{Show: domain.Show{ID: "s1", Name: "Tech Weekly", Publisher: "Tech Media", TotalEpisodes: 50, MediaType: "audio"}},
		{Show: domain.Show{ID: "s2", Name: "The Daily", Publisher: "NYT", TotalEpisodes: 120, MediaType: "mixed"}},
	})

	pane := panes.NewFollowedShowsPane(s, th, true)
	pane.SetSize(78, 10)

	// Press Enter on the first show to drill into episode sub-view.
	pane.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Simulate episode data loaded for the selected show.
	pane.Update(panes.ShowEpisodesLoadedMsg{
		ShowID: "s1",
		Items: []domain.Episode{
			{ID: "e1", Name: "AI Revolution", URI: "spotify:episode:e1", DurationMs: 1800000, ReleaseDate: "2026-01-15", IsPlayable: true},
			{ID: "e2", Name: "Quantum Computing Basics", URI: "spotify:episode:e2", DurationMs: 2400000, ReleaseDate: "2026-01-10", IsPlayable: true},
			{ID: "e3", Name: "Future of Work", URI: "spotify:episode:e3", DurationMs: 1500000, ReleaseDate: "2026-01-08", IsPlayable: false},
		},
		Total:   3,
		HasNext: false,
		Offset:  0,
	})

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestFollowedShowsPane_View_Narrow verifies golden snapshot of FollowedShowsPane
// at narrow width (40×24).
func TestFollowedShowsPane_View_Narrow(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetFollowedShows([]domain.SavedShow{
		{Show: domain.Show{ID: "s1", Name: "Tech Weekly", Publisher: "Tech Media", TotalEpisodes: 42, MediaType: "audio"}},
		{Show: domain.Show{ID: "s2", Name: "The Daily", Publisher: "NYT", TotalEpisodes: 120, MediaType: "mixed"}},
	})

	pane := panes.NewFollowedShowsPane(s, th, true)
	pane.SetSize(38, 10)

	tm := goldentest.NewPaneTest(t, pane, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestFollowedShowsPane_View_FilterActive verifies golden snapshot of FollowedShowsPane
// with an active filter filtering shows by name at normal dimensions (80×24).
func TestFollowedShowsPane_View_FilterActive(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetFollowedShows([]domain.SavedShow{
		{Show: domain.Show{ID: "s1", Name: "Tech Weekly", Publisher: "Tech Media", TotalEpisodes: 42, MediaType: "audio"}},
		{Show: domain.Show{ID: "s2", Name: "The Daily", Publisher: "NYT", TotalEpisodes: 120, MediaType: "mixed"}},
		{Show: domain.Show{ID: "s3", Name: "Tech Today", Publisher: "Tech Co", TotalEpisodes: 85, MediaType: "audio"}},
		{Show: domain.Show{ID: "s4", Name: "Science Friday", Publisher: "NPR", TotalEpisodes: 300, MediaType: "video"}},
	})

	pane := panes.NewFollowedShowsPane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f' key, then type "Tech" to filter by name.
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	for _, r := range "Tech" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
