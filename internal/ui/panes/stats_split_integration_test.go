// Package panes — stats_split_integration_test.go contains integration tests
// that exercise the 3 stats panes (RecentlyPlayedPane, TopTracksPane,
// TopArtistsPane) together: resize, time-range independence, edge cases.
package panes

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newIntegrationStore returns a store pre-populated with data for all three panes.
func newIntegrationStore() *state.Store {
	st := state.New()
	now := time.Now()

	// Recently played
	st.SetRecentlyPlayed([]domain.PlayHistory{
		{
			Track:    domain.Track{ID: "r1", Name: "Recent One", URI: "spotify:track:r1", Artists: []domain.Artist{{Name: "Artist R"}}},
			PlayedAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
		{
			Track:    domain.Track{ID: "r2", Name: "Recent Two", URI: "spotify:track:r2", Artists: []domain.Artist{{Name: "Artist S"}}},
			PlayedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		},
	})

	// Top tracks
	tracks := []domain.Track{
		{ID: "tt1", Name: "Integration Track", URI: "spotify:track:it1", Artists: []domain.Artist{{Name: "Test Band"}}},
	}
	st.SetTopTracks("short_term", tracks)
	st.SetTopArtists("short_term", []domain.FullArtist{})
	st.StampStatsFetchedAt("short_term")

	// Top artists
	artists := []domain.FullArtist{
		{ID: "a1", Name: "Integration Artist", URI: "spotify:artist:ia1", Genres: []string{"electronic"}},
	}
	st.SetTopArtists("short_term", artists)
	st.SetTopTracks("short_term", tracks)
	st.StampStatsFetchedAt("short_term")

	return st
}

func TestStatsSplit_AllPanesResizeCorrectly(t *testing.T) {
	st := newIntegrationStore()
	th := theme.Load("black")

	rp := NewRecentlyPlayedPane(st, th, false)
	tp := NewTopTracksPane(st, th, false)
	ap := NewTopArtistsPane(st, th, false)

	// Initially no size set — should not panic
	_ = rp.View()
	_ = tp.View()
	_ = ap.View()

	// Set size
	rp.SetSize(100, 15)
	tp.SetSize(100, 15)
	ap.SetSize(100, 15)

	view := rp.View()
	assert.NotEmpty(t, view)
	view = tp.View()
	assert.NotEmpty(t, view)
	view = ap.View()
	assert.NotEmpty(t, view)

	// Resize to a different size — should not panic
	rp.SetSize(200, 30)
	tp.SetSize(200, 30)
	ap.SetSize(200, 30)

	view = rp.View()
	assert.NotEmpty(t, view)
}

func TestStatsSplit_TopTracksAndTopArtistsCycleIndependently(t *testing.T) {
	st := newIntegrationStore()
	th := theme.Load("black")

	// Populate medium_term only for artists
	medArtists := []domain.FullArtist{
		{ID: "a2", Name: "Medium Artist", URI: "spotify:artist:a2", Genres: []string{"jazz"}},
	}
	st.SetTopArtists("medium_term", medArtists)
	st.SetTopTracks("medium_term", []domain.Track{})
	st.StampStatsFetchedAt("medium_term")

	tp := NewTopTracksPane(st, th, true)
	tp.SetSize(120, 20)

	ap := NewTopArtistsPane(st, th, true)
	ap.SetSize(120, 20)

	// Cycle artists to medium_term
	ap.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint
	assert.Equal(t, "medium_term", ap.TimeRange())

	// Tracks pane should still be on short_term
	assert.Equal(t, "short_term", tp.TimeRange())

	// Verify artists pane shows medium data
	view := ap.View()
	assert.Contains(t, view, "Medium Artist")

	// Verify tracks pane still shows short_term data
	view = tp.View()
	assert.Contains(t, view, "Integration Track")
}

func TestStatsSplit_RecentlyPlayedLoadThenScroll(t *testing.T) {
	st := newIntegrationStore()
	th := theme.Load("black")

	pane := NewRecentlyPlayedPane(st, th, true)
	pane.SetSize(120, 20)

	// Verify both items show
	view := pane.View()
	assert.Contains(t, view, "Recent One")
	assert.Contains(t, view, "Recent Two")

	// Press j to move cursor down
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) //nolint

	// Enter on second item
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	playMsg, ok := msg.(PlayTrackMsg)
	require.True(t, ok)
	assert.Equal(t, "spotify:track:r2", playMsg.TrackURI)
}

func TestStatsSplit_TopTracksFilterThenCycleRange(t *testing.T) {
	st := newIntegrationStore()
	th := theme.Load("black")

	// Add medium_term data
	st.SetTopTracks("medium_term", []domain.Track{
		{ID: "m1", Name: "Medium Track", URI: "spotify:track:m1", Artists: []domain.Artist{{Name: "Medium Band"}}},
	})
	st.SetTopArtists("medium_term", []domain.FullArtist{})
	st.StampStatsFetchedAt("medium_term")

	pane := NewTopTracksPane(st, th, true)
	pane.SetSize(120, 20)

	// Activate filter and type
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint
	for _, r := range "integration" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint
	}

	view := pane.View()
	assert.Contains(t, view, "Integration Track")

	// Close filter with Esc
	pane.Update(tea.KeyMsg{Type: tea.KeyEsc}) //nolint
	assert.False(t, pane.filter.IsActive())

	// Cycle to medium_term
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint
	assert.Equal(t, "medium_term", pane.TimeRange())

	view = pane.View()
	assert.Contains(t, view, "Medium Track")
}

func TestStatsSplit_ArtistVeryLongName(t *testing.T) {
	st := state.New()
	longName := strings.Repeat("A", 100)
	st.SetTopArtists("short_term", []domain.FullArtist{
		{ID: "a1", Name: longName, URI: "spotify:artist:a1", Genres: []string{"pop"}},
	})
	st.SetTopTracks("short_term", []domain.Track{})
	st.StampStatsFetchedAt("short_term")

	th := theme.Load("black")
	pane := NewTopArtistsPane(st, th, false)
	pane.SetSize(60, 15)

	// Should not panic with truncation
	view := pane.View()
	assert.NotEmpty(t, view)
}

func TestStatsSplit_RecentlyPlayedInvalidTimestamp(t *testing.T) {
	st := state.New()
	st.SetRecentlyPlayed([]domain.PlayHistory{
		{
			Track:    domain.Track{ID: "r1", Name: "Bad Time Track", URI: "spotify:track:r1", Artists: []domain.Artist{{Name: "Artist"}}},
			PlayedAt: "not-a-valid-timestamp",
		},
	})

	th := theme.Load("black")
	pane := NewRecentlyPlayedPane(st, th, false)
	pane.SetSize(120, 20)

	// Should not panic with invalid timestamp — shows empty string for played column
	view := pane.View()
	assert.Contains(t, view, "Bad Time Track")
}

func TestStatsSplit_AllPanesEmptyEdgeCase(t *testing.T) {
	st := state.New()
	th := theme.Load("black")

	rp := NewRecentlyPlayedPane(st, th, false)
	tp := NewTopTracksPane(st, th, false)
	ap := NewTopArtistsPane(st, th, false)

	rp.SetSize(120, 20)
	tp.SetSize(120, 20)
	ap.SetSize(120, 20)

	// All should render without panic with empty store
	assert.NotEmpty(t, rp.View())
	assert.NotEmpty(t, tp.View())
	assert.NotEmpty(t, ap.View())
}
