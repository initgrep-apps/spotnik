package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── EpisodeDetailsOverlay flow ─────────────────────────────────────────────────

// newPodcastEpApp creates an App with a playing episode on the store, ready
// for EpisodeDetailsOverlay and podcast-preset tests.
func newPodcastEpApp(t *testing.T) *app.App {
	t.Helper()
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// Set window size so the grid renders and overlays work.
	a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})

	// Set a playing episode on the store.
	s := state.New()
	ep := &domain.Episode{
		ID:          "ep-int-1",
		Name:        "AI Revolution — The Deep Dive",
		DurationMs:  3600000,
		ReleaseDate: "2026-06-01",
		Description: "A comprehensive look at how artificial intelligence is reshaping industries, from healthcare to finance. Our panel of experts discusses breakthroughs, ethical dilemmas, and what the future holds.",
		Show: &domain.Show{
			ID:        "show-int-1",
			Name:      "Tech Frontiers",
			Publisher: "TechMedia Inc.",
		},
	}
	ps := &domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "episode",
		Episode:              ep,
	}
	s.SetPlaybackState(ps)
	a.Update(panes.PlaybackStateFetchedMsg{State: ps})

	return a
}

// TestPodcastFlow_EpisodeDetailsOverlay verifies the full episode-details overlay
// lifecycle through the App: open with 'i', check View content, close with Esc.
func TestPodcastFlow_EpisodeDetailsOverlay(t *testing.T) {
	a := newPodcastEpApp(t)

	// Dismiss splash screen so the grid renders.
	a.Update(app.SplashDismissMsgForTest{})

	// Step 1: Send 'i' — overlay should open.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	require.True(t, a.EpisodeDetailsOpen(), "'i' should open EpisodeDetailsOverlay")

	// Step 2: Assert overlay content appears in View().
	view := a.View()
	assert.Contains(t, view, "Episode Details", "view should contain overlay title")
	assert.Contains(t, view, "AI Revolution", "view should contain episode name")
	assert.Contains(t, view, "Tech Frontiers", "view should contain show name")

	// Step 3: Send Esc — overlay closes.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "Esc in overlay should produce a cmd")
	escMsg := cmd()
	_, isClosed := escMsg.(panes.EpisodeDetailsClosedMsg)
	assert.True(t, isClosed, "Esc should produce EpisodeDetailsClosedMsg")

	// Feed it back to close.
	a.Update(escMsg)
	assert.False(t, a.EpisodeDetailsOpen(), "overlay should be closed after Esc")
}

// TestPodcastFlow_EpisodeDetailsOverlay_NoOpForTrack verifies 'i' is a no-op
// when a track (not episode) is playing.
func TestPodcastFlow_EpisodeDetailsOverlay_NoOpForTrack(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})

	s := state.New()
	ps := &domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "track",
		Item: &domain.Track{
			ID:   "track-1",
			Name: "Some Song",
		},
	}
	s.SetPlaybackState(ps)
	a.Update(panes.PlaybackStateFetchedMsg{State: ps})

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	assert.False(t, a.EpisodeDetailsOpen(), "'i' should not open overlay when track is playing")
}

// ── Pane-level: FollowedShows drill-down flow ──────────────────────────────────

// TestPodcastFlow_FollowedShowsDrillDown verifies the full drill-down lifecycle
// for FollowedShowsPane: show list → Enter → episode sub-view → Esc → back to list.
func TestPodcastFlow_FollowedShowsDrillDown(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetFollowedShows([]domain.SavedShow{
		{Show: domain.Show{ID: "s1", Name: "Tech Weekly", Publisher: "Tech Media", TotalEpisodes: 42, MediaType: "audio"}},
		{Show: domain.Show{ID: "s2", Name: "The Daily", Publisher: "NYT", TotalEpisodes: 120, MediaType: "mixed"}},
		{Show: domain.Show{ID: "s3", Name: "Science Friday", Publisher: "NPR", TotalEpisodes: 300, MediaType: "video"}},
	})

	pane := panes.NewFollowedShowsPane(s, th, true)
	pane.SetSize(78, 10)

	// Step 1: View should show "Followed Shows" title (not episode sub-view).
	assert.Contains(t, pane.Title(), "Followed Shows")
	assert.NotContains(t, pane.Title(), "Tech Weekly")

	// Step 2: Send Enter on show 0 → drill into episode sub-view.
	_, _ = pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Contains(t, pane.Title(), "Tech Weekly", "title should reflect selected show")
	assert.Contains(t, pane.Title(), "eps", "title should include episode count")

	// Step 3: Send Esc → return to show list.
	_, _ = pane.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, "Followed Shows", pane.Title(), "title should revert to show list")

	// Step 4: Send Enter on show 1 (cursor = show 1) → different episodes.
	// Move cursor down first.
	pane.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, _ = pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Contains(t, pane.Title(), "The Daily", "title should reflect second selected show")
}

// ── Pane-level: Enter produces PlayEpisodeMsg ──────────────────────────────────

// TestPodcastFlow_EnterPlaysEpisode verifies that pressing Enter on an episode
// in both SavedEpisodesPane and FollowedShowsPane (episode sub-view) produces
// a PlayEpisodeMsg command.
func TestPodcastFlow_EnterPlaysEpisode(t *testing.T) {
	t.Run("SavedEpisodes", func(t *testing.T) {
		s := state.New()
		th := theme.Load("black")
		s.SetSavedEpisodes([]domain.SavedEpisode{
			{Episode: domain.Episode{
				ID: "e1", Name: "AI Revolution", URI: "spotify:episode:e1", DurationMs: 1800000,
				IsPlayable: true,
				Show: &domain.Show{ID: "s1", Name: "Tech Weekly"},
			}},
			{Episode: domain.Episode{
				ID: "e2", Name: "Deep Sea Mysteries", URI: "spotify:episode:e2", DurationMs: 3600000,
				IsPlayable: true,
				Show: &domain.Show{ID: "s2", Name: "Nature Podcast"},
			}},
		})

		pane := panes.NewSavedEpisodesPane(s, th, true)
		pane.SetSize(78, 10)

		// Press Enter on the first episode.
		_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, cmd, "Enter on episode should produce a command")

		msg := cmd()
		playMsg, ok := msg.(panes.PlayEpisodeMsg)
		require.True(t, ok, "Enter should produce PlayEpisodeMsg, got %T", msg)
		assert.Equal(t, "spotify:episode:e1", playMsg.EpisodeURI)
		assert.Equal(t, "spotify:show:s1", playMsg.PlaylistURI)
	})

	t.Run("FollowedShowsEpisodeView", func(t *testing.T) {
		s := state.New()
		th := theme.Load("black")
		s.SetFollowedShows([]domain.SavedShow{
			{Show: domain.Show{ID: "s1", Name: "Tech Weekly", Publisher: "Tech Media", TotalEpisodes: 50, MediaType: "audio"}},
		})

		pane := panes.NewFollowedShowsPane(s, th, true)
		pane.SetSize(78, 10)

		// Drill into episode sub-view.
		pane.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// Load episodes for the show.
		pane.Update(panes.ShowEpisodesLoadedMsg{
			ShowID: "s1",
			Items: []domain.Episode{
				{ID: "e1", Name: "AI Revolution", URI: "spotify:episode:e1", DurationMs: 1800000, IsPlayable: true},
				{ID: "e2", Name: "Quantum Computing", URI: "spotify:episode:e2", DurationMs: 2400000, IsPlayable: true},
			},
			Total:   2,
			HasNext: false,
			Offset:  0,
		})

		// Press Enter on the first episode.
		_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, cmd, "Enter on episode should produce a command")

		msg := cmd()
		playMsg, ok := msg.(panes.PlayEpisodeMsg)
		require.True(t, ok, "Enter should produce PlayEpisodeMsg, got %T", msg)
		assert.Equal(t, "spotify:episode:e1", playMsg.EpisodeURI)
		assert.Equal(t, "spotify:show:s1", playMsg.PlaylistURI)
	})
}

// TestPodcastFlow_EnterOnUnplayableEpisode_NoOp verifies that pressing Enter
// on an unplayable episode is a silent no-op (does not produce a command).
func TestPodcastFlow_EnterOnUnplayableEpisode_NoOp(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetSavedEpisodes([]domain.SavedEpisode{
		{Episode: domain.Episode{
			ID: "e1", Name: "Locked Episode", URI: "spotify:episode:e1", DurationMs: 1800000,
			IsPlayable: false,
			Show: &domain.Show{ID: "s1", Name: "Tech Weekly"},
		}},
	})

	pane := panes.NewSavedEpisodesPane(s, th, true)
	pane.SetSize(78, 10)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "Enter on unplayable episode should not produce a command")
}
