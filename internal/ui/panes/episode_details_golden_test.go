package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestEpisodeDetailsOverlay_View_EpisodeInfo verifies golden snapshot of EpisodeDetailsOverlay
// with episode name, show name, description, and duration at normal dimensions (80×24).
func TestEpisodeDetailsOverlay_View_EpisodeInfo(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:          "ep-1",
			Name:        "The Future of AI in Healthcare",
			URI:         "spotify:episode:ep-1",
			DurationMs:  3600000,
			ReleaseDate: "2026-06-15",
			Description: "In this episode, we explore how artificial intelligence is transforming healthcare delivery, from diagnostic algorithms to robotic surgery assistants. Our guest experts discuss the latest breakthroughs and ethical considerations.",
			Show: &domain.Show{
				ID:        "show-1",
				Name:      "Tech Frontiers",
				Publisher: "TechMedia Inc.",
			},
		},
	})

	pane := panes.NewEpisodeDetailsOverlay(s, th)
	pane.SetSize(80, 24)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestEpisodeDetailsOverlay_View_Narrow verifies golden snapshot of EpisodeDetailsOverlay
// at narrow width (40×24).
func TestEpisodeDetailsOverlay_View_Narrow(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:          "ep-1",
			Name:        "Shortcast: Daily Tech Brief",
			URI:         "spotify:episode:ep-1",
			DurationMs:  600000,
			ReleaseDate: "2026-06-26",
			Description: "Today's top stories in tech, delivered in 10 minutes. We cover the latest AI announcements, startup funding rounds, and developer tool releases.",
			Show: &domain.Show{
				ID:        "show-1",
				Name:      "Daily Brief",
				Publisher: "NewsNet",
			},
		},
	})

	pane := panes.NewEpisodeDetailsOverlay(s, th)
	pane.SetSize(40, 24)

	tm := goldentest.NewPaneTest(t, pane, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
