package panes_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTopTracksTimeRangeCycle_UpdatesView verifies time-range cycling on
// TopTracksPane: 'g' key cycles short_term → medium_term → long_term →
// short_term and updates the View() output.
func TestTopTracksTimeRangeCycle_UpdatesView(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Populate all three time ranges with distinct data
	s.SetTopTracks("short_term", []domain.Track{
		{ID: "s1", Name: "Short Track", URI: "spotify:track:s1", DurationMs: 200000, Artists: []domain.Artist{{Name: "Short Artist"}}},
	})
	s.SetTopArtists("short_term", []domain.FullArtist{})
	s.StampStatsFetchedAt("short_term")

	s.SetTopTracks("medium_term", []domain.Track{
		{ID: "m1", Name: "Medium Track", URI: "spotify:track:m1", DurationMs: 210000, Artists: []domain.Artist{{Name: "Medium Artist"}}},
	})
	s.SetTopArtists("medium_term", []domain.FullArtist{})
	s.StampStatsFetchedAt("medium_term")

	s.SetTopTracks("long_term", []domain.Track{
		{ID: "l1", Name: "Long Track", URI: "spotify:track:l1", DurationMs: 220000, Artists: []domain.Artist{{Name: "Long Artist"}}},
	})
	s.SetTopArtists("long_term", []domain.FullArtist{})
	s.StampStatsFetchedAt("long_term")

	pane := panes.NewTopTracksPane(s, th, true)
	pane.SetSize(80, 15)

	// Start at short_term
	view := pane.View()
	assert.Contains(t, view, "Short Track", "should show short_term data initially")

	// 1. Send 'g' → medium_term
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck
	view = pane.View()
	assert.Contains(t, view, "Medium Track", "should show medium_term data after first 'g'")
	assert.NotContains(t, view, "Short Track")

	// 2. Send 'g' → long_term
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck
	view = pane.View()
	assert.Contains(t, view, "Long Track", "should show long_term data after second 'g'")
	assert.NotContains(t, view, "Medium Track")

	// 3. Send 'g' → wraps back to short_term
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck
	view = pane.View()
	assert.Contains(t, view, "Short Track", "should wrap back to short_term data after third 'g'")
	assert.NotContains(t, view, "Long Track")
}

// TestStatsEnterPlaysContextOrTrackList verifies that Enter on TopTracksPane
// emits PlayTrackListMsg, and Enter on TopArtistsPane emits PlayContextMsg.
func TestStatsEnterPlaysContextOrTrackList(t *testing.T) {
	// --- TopTracks Enter → PlayTrackListMsg ---
	{
		s := state.New()
		th := theme.Load("black")
		s.SetTopTracks("short_term", []domain.Track{
			{ID: "t1", Name: "Track One", URI: "spotify:track:t1", DurationMs: 200000, Artists: []domain.Artist{{Name: "Artist A"}}},
			{ID: "t2", Name: "Track Two", URI: "spotify:track:t2", DurationMs: 210000, Artists: []domain.Artist{{Name: "Artist B"}}},
			{ID: "t3", Name: "Track Three", URI: "spotify:track:t3", DurationMs: 220000, Artists: []domain.Artist{{Name: "Artist C"}}},
		})
		s.SetTopArtists("short_term", []domain.FullArtist{})
		s.StampStatsFetchedAt("short_term")

		pane := panes.NewTopTracksPane(s, th, true)
		pane.SetSize(80, 15)

		// Move down to select the second track
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) //nolint:errcheck

		_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, cmd, "Enter on TopTracks should produce a command")
		msg := cmd()
		playMsg, ok := msg.(panes.PlayTrackListMsg)
		require.True(t, ok, "expected PlayTrackListMsg, got %T", msg)
		// Should include URIs from selected index (1) onward: t2, t3
		require.Len(t, playMsg.URIs, 2)
		assert.Equal(t, "spotify:track:t2", playMsg.URIs[0])
		assert.Equal(t, "spotify:track:t3", playMsg.URIs[1])
	}

	// --- TopArtists Enter → PlayContextMsg ---
	{
		s := state.New()
		th := theme.Load("black")
		s.SetTopArtists("short_term", []domain.FullArtist{
			{ID: "a1", Name: "Artist One", URI: "spotify:artist:a1", Popularity: 80, Followers: domain.ArtistFollowers{Total: 1000000}},
			{ID: "a2", Name: "Artist Two", URI: "spotify:artist:a2", Popularity: 70, Followers: domain.ArtistFollowers{Total: 500000}},
		})
		s.SetTopTracks("short_term", []domain.Track{})
		s.StampStatsFetchedAt("short_term")

		pane := panes.NewTopArtistsPane(s, th, true)
		pane.SetSize(80, 15)

		_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, cmd, "Enter on TopArtists should produce a command")
		msg := cmd()
		ctxMsg, ok := msg.(panes.PlayContextMsg)
		require.True(t, ok, "expected PlayContextMsg, got %T", msg)
		assert.Equal(t, "spotify:artist:a1", ctxMsg.ContextURI)
	}
}
