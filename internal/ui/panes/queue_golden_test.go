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

// TestQueuePane_View_WithTracks_Normal verifies the golden snapshot of QueuePane
// with two track items at normal width (80×24).
func TestQueuePane_View_WithTracks_Normal(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetQueue([]domain.QueueItem{
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{
			Name: "Blinding Lights", URI: "spotify:track:1",
			Artists:    []domain.Artist{{Name: "The Weeknd"}},
			DurationMs: 200000,
		}},
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{
			Name: "Shape of You", URI: "spotify:track:2",
			Artists:    []domain.Artist{{Name: "Ed Sheeran"}},
			DurationMs: 240000,
		}},
	})

	pane := panes.NewQueuePane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestQueuePane_View_Empty verifies the golden snapshot of QueuePane
// with an empty queue (shows empty state).
func TestQueuePane_View_Empty(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewQueuePane(s, th, false)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestQueuePane_View_WithEpisodes_Narrow verifies the golden snapshot of QueuePane
// with an episode item at narrow width (40×24).
func TestQueuePane_View_WithEpisodes_Narrow(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetQueue([]domain.QueueItem{
		{Type: domain.QueueItemTypeEpisode, Episode: &domain.Episode{
			Name:       "The Future of AI",
			URI:        "spotify:episode:1",
			DurationMs: 3600000,
			Show:       &domain.Show{Name: "Tech Weekly"},
		}},
	})

	pane := panes.NewQueuePane(s, th, true)
	pane.SetSize(38, 10)

	tm := goldentest.NewPaneTest(t, pane, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestQueuePane_View_MixedContent verifies the golden snapshot of QueuePane
// with mixed tracks and episodes (80×24).
func TestQueuePane_View_MixedContent(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetQueue([]domain.QueueItem{
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{
			Name: "Blinding Lights", URI: "spotify:track:1",
			Artists:    []domain.Artist{{Name: "The Weeknd"}},
			DurationMs: 200000,
		}},
		{Type: domain.QueueItemTypeEpisode, Episode: &domain.Episode{
			Name:       "The Future of AI",
			URI:        "spotify:episode:1",
			DurationMs: 3600000,
			Show:       &domain.Show{Name: "Tech Weekly"},
		}},
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{
			Name: "Shape of You", URI: "spotify:track:2",
			Artists:    []domain.Artist{{Name: "Ed Sheeran"}},
			DurationMs: 240000,
		}},
	})

	pane := panes.NewQueuePane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestQueuePane_View_FilterActive verifies the golden snapshot of QueuePane
// with an active filter that has matching results (80×24).
func TestQueuePane_View_FilterActive(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetQueue([]domain.QueueItem{
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{
			Name: "Blinding Lights", URI: "spotify:track:1",
			Artists:    []domain.Artist{{Name: "The Weeknd"}},
			DurationMs: 200000,
		}},
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{
			Name: "Shape of You", URI: "spotify:track:2",
			Artists:    []domain.Artist{{Name: "Ed Sheeran"}},
			DurationMs: 240000,
		}},
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{
			Name: "Starboy", URI: "spotify:track:3",
			Artists:    []domain.Artist{{Name: "The Weeknd"}},
			DurationMs: 230000,
		}},
	})

	pane := panes.NewQueuePane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f' key, then type "The Weeknd" to filter by artist
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	for _, r := range "The Weeknd" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestQueuePane_View_FilterActive_NoMatches verifies the golden snapshot of QueuePane
// with an active filter that has no matching results (80×24).
func TestQueuePane_View_FilterActive_NoMatches(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetQueue([]domain.QueueItem{
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{
			Name: "Blinding Lights", URI: "spotify:track:1",
			Artists:    []domain.Artist{{Name: "The Weeknd"}},
			DurationMs: 200000,
		}},
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{
			Name: "Shape of You", URI: "spotify:track:2",
			Artists:    []domain.Artist{{Name: "Ed Sheeran"}},
			DurationMs: 240000,
		}},
	})

	pane := panes.NewQueuePane(s, th, true)
	pane.SetSize(78, 10)

	// Activate filter with 'f' key, then type a query that won't match anything
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	for _, r := range "zzzznomatch" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
