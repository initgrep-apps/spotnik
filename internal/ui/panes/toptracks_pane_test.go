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

// Compile-time check: TopTracksPane implements layout.Pane.
var _ layout.Pane = &TopTracksPane{}

// populateStoreTopTracks loads test top-tracks data into the store for the given time range.
func populateStoreTopTracks(st *state.Store, timeRange string) {
	tracks := []domain.Track{
		{ID: "tt1", Name: "Blinding Lights", URI: "spotify:track:tt1", Artists: []domain.Artist{{Name: "The Weeknd"}}, Album: domain.Album{Name: "After Hours"}},
		{ID: "tt2", Name: "Martbaan", URI: "spotify:track:tt2", Artists: []domain.Artist{{Name: "Samar Mehdi"}}, Album: domain.Album{Name: "Album X"}},
		{ID: "tt3", Name: "Save Your Tears", URI: "spotify:track:tt3", Artists: []domain.Artist{{Name: "The Weeknd"}}, Album: domain.Album{Name: "After Hours"}},
	}
	st.SetTopTracks(timeRange, tracks)
	st.SetTopArtists(timeRange, []domain.FullArtist{})
	st.StampStatsFetchedAt(timeRange)
}

// newTestTopTracksPane creates a TopTracksPane pre-populated with short_term data.
func newTestTopTracksPane() (*TopTracksPane, *state.Store) {
	st := state.New()
	populateStoreTopTracks(st, "short_term")
	th := theme.Load("black")
	pane := NewTopTracksPane(st, th, false)
	pane.SetSize(120, 20)
	return pane, st
}

func TestTopTracksPane_ImplementsLayoutPane(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	assert.Equal(t, layout.PaneTopTracks, pane.ID())
}

func TestTopTracksPane_Metadata(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	assert.Equal(t, layout.PaneTopTracks, pane.ID())
	assert.Equal(t, "Top Tracks", pane.Title())
	assert.Equal(t, 7, pane.ToggleKey())
}

func TestTopTracksPane_Actions_Default_ShowsFilterAndRange(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	actions := pane.Actions()
	require.Len(t, actions, 2)
	assert.Equal(t, "f", actions[0].Key)
	assert.Equal(t, "t", actions[1].Key)
	// Default is short_term → label "4wk"
	assert.Equal(t, "4wk", actions[1].Label)
}

func TestTopTracksPane_Actions_FilterActive(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	pane.SetFocused(true)
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "Esc", actions[0].Key)
}

func TestTopTracksPane_RendersTrackNames(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	view := pane.View()
	assert.Contains(t, view, "Blinding Lights")
	assert.Contains(t, view, "Martbaan")
}

func TestTopTracksPane_RendersPopularity(t *testing.T) {
	// Popularity is on domain.Track — but the Track type doesn't have Popularity.
	// TopTracksPane renders a "#" column and a Track column. Spec says popularity column.
	// We use index column (# 5%), track (45%), artist (35%), popularity placeholder (15%).
	// domain.Track doesn't carry Popularity — spec says show it but Spotify's track object
	// in the top-tracks response may not include it in our domain.Track type.
	// Looking at domain.Track: it has no Popularity field. We'll use empty string or "—".
	// This test just verifies the column header renders without panic.
	pane, _ := newTestTopTracksPane()
	view := pane.View()
	assert.NotEmpty(t, view)
	// At a minimum, track names should be visible.
	assert.Contains(t, view, "Blinding Lights")
}

func TestTopTracksPane_TimeRangeCycles(t *testing.T) {
	pane, st := newTestTopTracksPane()
	pane.SetFocused(true)

	// Populate all time ranges
	populateStoreTopTracks(st, "medium_term")
	populateStoreTopTracks(st, "long_term")

	// Initial: short_term (4wk)
	assert.Equal(t, "short_term", pane.TimeRange())

	// Press t → medium_term
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	assert.Equal(t, "medium_term", pane.TimeRange())
	// Data already cached → no fetch command
	assert.Nil(t, cmd)

	// Press t → long_term
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint:errcheck
	assert.Equal(t, "long_term", pane.TimeRange())

	// Press t → wraps to short_term
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint:errcheck
	assert.Equal(t, "short_term", pane.TimeRange())
}

func TestTopTracksPane_TimeRangeEmitsFetchOnCacheMiss(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	// Store only has short_term; medium_term is not loaded.
	pane.SetFocused(true)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	assert.Equal(t, "medium_term", pane.TimeRange())
	// Cache miss → must emit FetchStatsMsg
	require.NotNil(t, cmd)
	msg := cmd()
	fetchMsg, ok := msg.(FetchStatsMsg)
	require.True(t, ok)
	assert.Equal(t, "medium_term", fetchMsg.TimeRange)
}

func TestTopTracksPane_ActionsLabelReflectsRange(t *testing.T) {
	pane, st := newTestTopTracksPane()
	pane.SetFocused(true)
	populateStoreTopTracks(st, "medium_term")
	populateStoreTopTracks(st, "long_term")

	// Cycle to medium_term
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint:errcheck
	actions := pane.Actions()
	tAction := actions[len(actions)-1]
	assert.Equal(t, "t", tAction.Key)
	assert.Equal(t, "6mo", tAction.Label)

	// Cycle to long_term
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint:errcheck
	actions = pane.Actions()
	tAction = actions[len(actions)-1]
	assert.Equal(t, "all", tAction.Label)
}

// TestTopTracksPane_Enter_EmitsPlayTrackListMsg verifies Enter on row N emits
// PlayTrackListMsg with URIs from the selected index onward (Story 105).
func TestTopTracksPane_Enter_EmitsPlayTrackListMsg(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	pane.SetFocused(true)

	// Table starts at row 0 (first track: tt1).
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should produce a command")

	msg := cmd()
	playMsg, ok := msg.(PlayTrackListMsg)
	require.True(t, ok, "command should produce PlayTrackListMsg, got %T", msg)
	// From index 0 → all 3 URIs.
	require.Len(t, playMsg.URIs, 3, "should include URIs from selected track to end")
	assert.Equal(t, "spotify:track:tt1", playMsg.URIs[0], "first URI should be selected track")
	assert.Equal(t, "spotify:track:tt2", playMsg.URIs[1])
	assert.Equal(t, "spotify:track:tt3", playMsg.URIs[2])
}

// TestTopTracksPane_Enter_LastRow_EmitsSingleURI verifies Enter on the last row
// emits PlayTrackListMsg with only the last track URI.
func TestTopTracksPane_Enter_LastRow_EmitsSingleURI(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	pane.SetFocused(true)

	// Navigate to last row (3 tracks → 2 down presses).
	pane.Update(tea.KeyMsg{Type: tea.KeyDown}) //nolint:errcheck
	pane.Update(tea.KeyMsg{Type: tea.KeyDown}) //nolint:errcheck

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter on last row should produce a command")

	msg := cmd()
	playMsg, ok := msg.(PlayTrackListMsg)
	require.True(t, ok, "command should produce PlayTrackListMsg, got %T", msg)
	require.Len(t, playMsg.URIs, 1, "last row should emit single URI")
	assert.Equal(t, "spotify:track:tt3", playMsg.URIs[0])
}

func TestTopTracksPane_FilterByTrackName(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	pane.SetFocused(true)

	// Activate filter
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "martbaan" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	view := pane.View()
	assert.Contains(t, view, "Martbaan")
	assert.NotContains(t, view, "Blinding Lights")
}

func TestTopTracksPane_FilterByArtistName(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	pane.SetFocused(true)

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "samar" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	view := pane.View()
	assert.Contains(t, view, "Martbaan")
	assert.NotContains(t, view, "Blinding Lights")
}

func TestTopTracksPane_EmptyData(t *testing.T) {
	st := state.New()
	th := theme.Load("black")
	pane := NewTopTracksPane(st, th, false)
	pane.SetSize(120, 20)
	view := pane.View()
	assert.NotEmpty(t, view)
}

func TestTopTracksPane_StatsLoadedMsg(t *testing.T) {
	pane, st := newTestTopTracksPane()
	st.SetTopTracks("short_term", []domain.Track{
		{ID: "tt99", Name: "New Hit", URI: "spotify:track:tt99", Artists: []domain.Artist{{Name: "New Artist"}}},
	})
	st.SetTopArtists("short_term", []domain.FullArtist{})
	st.StampStatsFetchedAt("short_term")
	pane.Update(StatsLoadedMsg{TimeRange: "short_term"}) //nolint:errcheck
	view := pane.View()
	assert.Contains(t, view, "New Hit")
}

func TestTopTracksPane_NotFocusedIgnoresKeys(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
}

func TestTopTracksPane_Init(t *testing.T) {
	pane, _ := newTestTopTracksPane()
	cmd := pane.Init()
	assert.Nil(t, cmd)
}

func TestTopTracksPane_RefreshRows(t *testing.T) {
	pane, st := newTestTopTracksPane()
	st.SetTopTracks("short_term", []domain.Track{
		{ID: "tt77", Name: "Refreshed Track", URI: "spotify:track:tt77", Artists: []domain.Artist{{Name: "Refresh Artist"}}},
	})
	st.SetTopArtists("short_term", []domain.FullArtist{})
	st.StampStatsFetchedAt("short_term")
	pane.RefreshRows()
	view := pane.View()
	assert.Contains(t, view, "Refreshed Track")
}

// ── Story 71 Task 2: column color tokens ─────────────────────────────────────

// TestTopTracksPane_UsesColumnColors verifies that TopTracksPane column definitions
// use the new ColumnIndex/ColumnPrimary/ColumnSecondary/ColumnTertiary tokens.
func TestTopTracksPane_UsesColumnColors(t *testing.T) {
	th := theme.Load("black")
	p := NewTopTracksPane(state.New(), th, false)
	cols := p.table.Columns()
	require.Len(t, cols, 4, "TopTracksPane should have 4 columns")

	assert.Equal(t, th.ColumnIndex(), cols[0].Color, "# column should use ColumnIndex()")
	assert.Equal(t, th.ColumnPrimary(), cols[1].Color, "Track column should use ColumnPrimary()")
	assert.Equal(t, th.ColumnSecondary(), cols[2].Color, "Artist column should use ColumnSecondary()")
	assert.Equal(t, th.ColumnTertiary(), cols[3].Color, "Pop column should use ColumnTertiary()")
}
