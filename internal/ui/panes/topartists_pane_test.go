package panes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check: TopArtistsPane implements layout.Pane.
var _ layout.Pane = &TopArtistsPane{}

// populateStoreTopArtists loads test top-artists data into the store for the given time range.
func populateStoreTopArtists(st *state.Store, timeRange string) {
	artists := []domain.FullArtist{
		{ID: "a1", Name: "The Weeknd", URI: "spotify:artist:a1", Genres: []string{"pop", "r&b"}},
		{ID: "a2", Name: "Drake", URI: "spotify:artist:a2", Genres: []string{"hip-hop", "rap"}},
		{ID: "a3", Name: "Dua Lipa", URI: "spotify:artist:a3", Genres: []string{"dance pop", "pop"}},
		{ID: "a4", Name: "Artist No Genre", URI: "spotify:artist:a4", Genres: []string{}},
	}
	st.SetTopArtists(timeRange, artists)
	st.SetTopTracks(timeRange, []domain.Track{})
	st.StampStatsFetchedAt(timeRange)
}

// newTestTopArtistsPane creates a TopArtistsPane pre-populated with short_term data.
func newTestTopArtistsPane() (*TopArtistsPane, *state.Store) {
	st := state.New()
	populateStoreTopArtists(st, "short_term")
	th := theme.Load("black")
	pane := NewTopArtistsPane(st, th, false)
	pane.SetSize(120, 20)
	return pane, st
}

func TestTopArtistsPane_ImplementsLayoutPane(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	assert.Equal(t, layout.PaneTopArtists, pane.ID())
}

func TestTopArtistsPane_Metadata(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	assert.Equal(t, layout.PaneTopArtists, pane.ID())
	assert.Equal(t, "Top Artists", pane.Title())
	assert.Equal(t, 8, pane.ToggleKey())
}

func TestTopArtistsPane_Actions_Default_ShowsFilterAndRange(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	actions := pane.Actions()
	require.Len(t, actions, 2)
	assert.Equal(t, "f", actions[0].Key)
	assert.Equal(t, "t", actions[1].Key)
	assert.Equal(t, "4wk", actions[1].Label)
}

func TestTopArtistsPane_Actions_FilterActive(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	pane.SetFocused(true)
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "Esc", actions[0].Key)
}

func TestTopArtistsPane_RendersArtistNames(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	view := pane.View()
	assert.Contains(t, view, "The Weeknd")
	assert.Contains(t, view, "Drake")
	assert.Contains(t, view, "Dua Lipa")
}

func TestTopArtistsPane_RendersGenreColumn(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	view := pane.View()
	// First genre of The Weeknd is "pop"
	assert.Contains(t, view, "pop")
	// First genre of Drake is "hip-hop"
	assert.Contains(t, view, "hip-hop")
}

func TestTopArtistsPane_ArtistNoGenreShowsDash(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	view := pane.View()
	// Artist with no genres should render "—"
	assert.Contains(t, view, "—")
}

func TestTopArtistsPane_TimeRangeCycles(t *testing.T) {
	pane, st := newTestTopArtistsPane()
	pane.SetFocused(true)

	populateStoreTopArtists(st, "medium_term")
	populateStoreTopArtists(st, "long_term")

	assert.Equal(t, "short_term", pane.TimeRange())

	// short → medium
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint
	assert.Equal(t, "medium_term", pane.TimeRange())

	// medium → long
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint
	assert.Equal(t, "long_term", pane.TimeRange())

	// long → short (wraps)
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint
	assert.Equal(t, "short_term", pane.TimeRange())
}

func TestTopArtistsPane_TimeRangeEmitsFetchOnCacheMiss(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	pane.SetFocused(true)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	assert.Equal(t, "medium_term", pane.TimeRange())
	require.NotNil(t, cmd)
	msg := cmd()
	fetchMsg, ok := msg.(FetchStatsMsg)
	require.True(t, ok)
	assert.Equal(t, "medium_term", fetchMsg.TimeRange)
}

func TestTopArtistsPane_ActionsLabelReflectsRange(t *testing.T) {
	pane, st := newTestTopArtistsPane()
	pane.SetFocused(true)
	populateStoreTopArtists(st, "medium_term")
	populateStoreTopArtists(st, "long_term")

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint
	actions := pane.Actions()
	assert.Equal(t, "6mo", actions[len(actions)-1].Label)

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint
	actions = pane.Actions()
	assert.Equal(t, "all", actions[len(actions)-1].Label)
}

func TestTopArtistsPane_FilterByArtistName(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	pane.SetFocused(true)

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint
	for _, r := range "weeknd" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint
	}

	view := pane.View()
	assert.Contains(t, view, "The Weeknd")
	assert.NotContains(t, view, "Drake")
}

func TestTopArtistsPane_FilterByGenre(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	pane.SetFocused(true)

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint
	for _, r := range "hip-hop" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint
	}

	view := pane.View()
	assert.Contains(t, view, "Drake")
	assert.NotContains(t, view, "The Weeknd")
}

func TestTopArtistsPane_EmptyData(t *testing.T) {
	st := state.New()
	th := theme.Load("black")
	pane := NewTopArtistsPane(st, th, false)
	pane.SetSize(120, 20)
	view := pane.View()
	assert.NotEmpty(t, view)
}

func TestTopArtistsPane_StatsLoadedMsg(t *testing.T) {
	pane, st := newTestTopArtistsPane()
	st.SetTopArtists("short_term", []domain.FullArtist{
		{ID: "a99", Name: "New Artist", URI: "spotify:artist:a99", Genres: []string{"jazz"}},
	})
	st.SetTopTracks("short_term", []domain.Track{})
	st.StampStatsFetchedAt("short_term")
	pane.Update(StatsLoadedMsg{TimeRange: "short_term"}) //nolint
	view := pane.View()
	assert.Contains(t, view, "New Artist")
}

func TestTopArtistsPane_StatsLoadedMsgWrongRange(t *testing.T) {
	pane, st := newTestTopArtistsPane()
	// Load new data for medium_term but pane is on short_term
	st.SetTopArtists("medium_term", []domain.FullArtist{
		{ID: "a88", Name: "Medium Artist", URI: "spotify:artist:a88", Genres: []string{"indie"}},
	})
	st.SetTopTracks("medium_term", []domain.Track{})
	st.StampStatsFetchedAt("medium_term")
	pane.Update(StatsLoadedMsg{TimeRange: "medium_term"}) //nolint
	view := pane.View()
	// Should still show short_term data (The Weeknd) not medium_term data
	assert.Contains(t, view, "The Weeknd")
	assert.NotContains(t, view, "Medium Artist")
}

func TestTopArtistsPane_NotFocusedIgnoresKeys(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	// t key on unfocused pane should not change time range
	initialRange := pane.TimeRange()
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint
	assert.Equal(t, initialRange, pane.TimeRange())
}

func TestTopArtistsPane_Init(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	cmd := pane.Init()
	assert.Nil(t, cmd)
}

func TestTopArtistsPane_SetFocused(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	assert.False(t, pane.IsFocused())
	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
}

func TestTopArtistsPane_RefreshRows(t *testing.T) {
	pane, st := newTestTopArtistsPane()
	st.SetTopArtists("short_term", []domain.FullArtist{
		{ID: "a77", Name: "Refreshed Artist", URI: "spotify:artist:a77", Genres: []string{"ambient"}},
	})
	st.SetTopTracks("short_term", []domain.Track{})
	st.StampStatsFetchedAt("short_term")
	pane.RefreshRows()
	view := pane.View()
	assert.Contains(t, view, "Refreshed Artist")
}
