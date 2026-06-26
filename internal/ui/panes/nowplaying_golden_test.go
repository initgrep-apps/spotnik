package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// ── Helpers ────────────────────────────────────────────────────────────────────

// newNPTestStore creates a Store pre-loaded with a playing track with default values.
func newNPTestStore() *state.Store {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "track",
		ProgressMs:           30000,
		ShuffleState:         false,
		RepeatState:          "off",
		Item: &domain.Track{
			ID:         "track-1",
			Name:       "Blinding Lights",
			URI:        "spotify:track:track-1",
			DurationMs: 252000,
			Artists:    []domain.Artist{{ID: "a1", Name: "The Weeknd"}},
			Album: domain.Album{
				ID:   "alb1",
				Name: "After Hours",
			},
		},
		Device: &domain.Device{
			ID:            "dev-1",
			Name:          "MacBook Pro",
			VolumePercent: 65,
		},
	})
	return s
}

// newNPEpisodeStore creates a Store pre-loaded with a playing episode.
func newNPEpisodeStore() *state.Store {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "episode",
		ProgressMs:           45000,
		ShuffleState:         false,
		RepeatState:          "off",
		Episode: &domain.Episode{
			ID:          "ep-1",
			Name:        "The Future of AI",
			URI:         "spotify:episode:ep-1",
			DurationMs:  3600000,
			ReleaseDate: "2025-06-15",
			Show: &domain.Show{
				ID:   "show-1",
				Name: "Tech Weekly",
			},
		},
		Device: &domain.Device{
			ID:            "dev-1",
			Name:          "MacBook Pro",
			VolumePercent: 65,
		},
	})
	return s
}

// ── Task 1: Track mode ─────────────────────────────────────────────────────────

// TestNowPlayingPane_View_TrackPlaying verifies golden snapshot of NowPlayingPane
// with a playing track at normal dimensions (80×24).
func TestNowPlayingPane_View_TrackPlaying(t *testing.T) {
	s := newNPTestStore()
	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestNowPlayingPane_View_TrackPaused verifies golden snapshot of NowPlayingPane
// with a paused track at normal dimensions (80×24).
func TestNowPlayingPane_View_TrackPaused(t *testing.T) {
	s := newNPTestStore()
	// Set playing to false.
	ps := s.PlaybackState()
	ps.IsPlaying = false
	s.SetPlaybackState(ps)

	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestNowPlayingPane_View_TrackNoData verifies golden snapshot of NowPlayingPane
// with no playback state (empty state) at normal dimensions (80×24).
func TestNowPlayingPane_View_TrackNoData(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// ── Task 2: Episode mode ───────────────────────────────────────────────────────

// TestNowPlayingPane_View_EpisodePlaying verifies golden snapshot of NowPlayingPane
// with a playing episode at normal dimensions (80×24).
func TestNowPlayingPane_View_EpisodePlaying(t *testing.T) {
	s := newNPEpisodeStore()
	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestNowPlayingPane_View_EpisodePaused verifies golden snapshot of NowPlayingPane
// with a paused episode at normal dimensions (80×24).
func TestNowPlayingPane_View_EpisodePaused(t *testing.T) {
	s := newNPEpisodeStore()
	ps := s.PlaybackState()
	ps.IsPlaying = false
	s.SetPlaybackState(ps)

	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// ── Task 3: Adaptive layout ────────────────────────────────────────────────────

// TestNowPlayingPane_View_CompactStrip verifies golden snapshot of NowPlayingPane
// at compact strip height (< 8), where track info is embedded in the title bar.
func TestNowPlayingPane_View_CompactStrip(t *testing.T) {
	s := newNPTestStore()
	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(78, 6)

	tm := goldentest.NewPaneTest(t, pane, 80, 7)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestNowPlayingPane_View_NarrowFallback verifies golden snapshot of NowPlayingPane
// at narrow width (40 cols), where the InfoBox is dropped and the visualizer fills
// the full content area.
func TestNowPlayingPane_View_NarrowFallback(t *testing.T) {
	s := newNPTestStore()
	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(38, 10)

	tm := goldentest.NewPaneTest(t, pane, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestNowPlayingPane_View_Wide verifies golden snapshot of NowPlayingPane
// at wide width (120 cols), where the InfoBox occupies ~25% and the seek bar is
// on the right side.
func TestNowPlayingPane_View_Wide(t *testing.T) {
	s := newNPTestStore()
	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(118, 10)

	tm := goldentest.NewPaneTest(t, pane, 120, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// ── Task 4: Seek bar + Volume ──────────────────────────────────────────────────

// TestNowPlayingPane_View_SeekBar_AtPosition verifies golden snapshot of NowPlayingPane
// with the seek bar at ~30% progress (75s of 252s track) at normal dimensions.
func TestNowPlayingPane_View_SeekBar_AtPosition(t *testing.T) {
	s := newNPTestStore()
	ps := s.PlaybackState()
	ps.ProgressMs = 75600 // 30% of 252000
	s.SetPlaybackState(ps)

	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestNowPlayingPane_View_VolumeBar verifies golden snapshot of NowPlayingPane
// with volume bar at 65% at normal dimensions.
func TestNowPlayingPane_View_VolumeBar(t *testing.T) {
	s := newNPTestStore()
	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// ── Task 5: Edge case types ────────────────────────────────────────────────────

// TestNowPlayingPane_View_AdType_EmptyState verifies golden snapshot when
// currently_playing_type is "ad" — should render empty state.
func TestNowPlayingPane_View_AdType_EmptyState(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "ad",
		ProgressMs:           15000,
	})
	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestNowPlayingPane_View_UnknownType_EmptyState verifies golden snapshot when
// currently_playing_type is "unknown" — should render empty state.
func TestNowPlayingPane_View_UnknownType_EmptyState(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "unknown",
		ProgressMs:           0,
	})
	th := theme.Load("black")
	pane := panes.NewNowPlayingPane(s, th, true)
	pane.SetSize(78, 10)

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
