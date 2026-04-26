package panes

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check: TopArtistsPane implements layout.Pane.
var _ layout.Pane = &TopArtistsPane{}

// populateStoreTopArtists loads test top-artists data into the store for the given time range.
func populateStoreTopArtists(st *state.Store, timeRange string) {
	artists := []domain.FullArtist{
		{ID: "a1", Name: "The Weeknd", URI: "spotify:artist:a1", Genres: []string{"pop", "r&b"}, Popularity: 95, Followers: domain.ArtistFollowers{Total: 35000000}},
		{ID: "a2", Name: "Drake", URI: "spotify:artist:a2", Genres: []string{"hip-hop", "rap"}, Popularity: 82, Followers: domain.ArtistFollowers{Total: 22000000}},
		{ID: "a3", Name: "Dua Lipa", URI: "spotify:artist:a3", Genres: []string{"dance pop", "pop"}, Popularity: 88, Followers: domain.ArtistFollowers{Total: 12500000}},
		{ID: "a4", Name: "Niche Artist", URI: "spotify:artist:a4", Genres: []string{}, Popularity: 15, Followers: domain.ArtistFollowers{Total: 800}},
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
	assert.Equal(t, "g", actions[1].Key)
	assert.Equal(t, "4wk", actions[1].Label)
}

func TestTopArtistsPane_Actions_FilterActive(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	pane.SetFocused(true)
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
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

func TestTopArtistsPane_RendersPopularityDots(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	view := pane.View()
	m := uikit.ActiveMode()
	on := uikit.GlyphFor(uikit.GlyphPinned, m)
	off := uikit.GlyphFor(uikit.GlyphUnpinned, m)
	// The Weeknd pop=95 → 5 filled stars
	assert.Contains(t, view, strings.Repeat(on, 5))
	// Niche Artist pop=15 → 5 empty stars
	assert.Contains(t, view, strings.Repeat(off, 5))
}

func TestTopArtistsPane_RendersFollowers(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	view := pane.View()
	// The Weeknd 35000000 → "35M"
	assert.Contains(t, view, "35M")
	// Niche Artist 800 → "800"
	assert.Contains(t, view, "800")
}

func TestTopArtistsPane_ZeroFollowersShowsDash(t *testing.T) {
	st := state.New()
	th := theme.Load("black")
	st.SetTopArtists("short_term", []domain.FullArtist{
		{ID: "a1", Name: "Unknown Artist", URI: "spotify:artist:a1", Popularity: 10, Followers: domain.ArtistFollowers{Total: 0}},
	})
	st.SetTopTracks("short_term", []domain.Track{})
	st.StampStatsFetchedAt("short_term")
	pane := NewTopArtistsPane(st, th, false)
	pane.SetSize(120, 20)
	view := pane.View()
	assert.Contains(t, view, "—")
}

func TestTopArtistsPane_TimeRangeCycles(t *testing.T) {
	pane, st := newTestTopArtistsPane()
	pane.SetFocused(true)

	populateStoreTopArtists(st, "medium_term")
	populateStoreTopArtists(st, "long_term")

	assert.Equal(t, "short_term", pane.TimeRange())

	// short → medium
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck
	assert.Equal(t, "medium_term", pane.TimeRange())

	// medium → long
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck
	assert.Equal(t, "long_term", pane.TimeRange())

	// long → short (wraps)
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck
	assert.Equal(t, "short_term", pane.TimeRange())
}

func TestTopArtistsPane_TimeRangeEmitsFetchOnCacheMiss(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	pane.SetFocused(true)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
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

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck
	actions := pane.Actions()
	assert.Equal(t, "6mo", actions[len(actions)-1].Label)

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck
	actions = pane.Actions()
	assert.Equal(t, "all", actions[len(actions)-1].Label)
}

func TestTopArtistsPane_FilterByArtistName(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	pane.SetFocused(true)

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "weeknd" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	view := pane.View()
	assert.Contains(t, view, "The Weeknd")
	assert.NotContains(t, view, "Drake")
}

func TestTopArtistsPane_FilterByArtistNameExcludes(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	pane.SetFocused(true)

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "drake" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
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
		{ID: "a99", Name: "New Artist", URI: "spotify:artist:a99", Genres: []string{"jazz"}, Popularity: 70, Followers: domain.ArtistFollowers{Total: 500000}},
	})
	st.SetTopTracks("short_term", []domain.Track{})
	st.StampStatsFetchedAt("short_term")
	pane.Update(StatsLoadedMsg{TimeRange: "short_term"}) //nolint:errcheck
	view := pane.View()
	assert.Contains(t, view, "New Artist")
}

func TestTopArtistsPane_StatsLoadedMsgWrongRange(t *testing.T) {
	pane, st := newTestTopArtistsPane()
	// Load new data for medium_term but pane is on short_term
	st.SetTopArtists("medium_term", []domain.FullArtist{
		{ID: "a88", Name: "Medium Artist", URI: "spotify:artist:a88", Genres: []string{"indie"}, Popularity: 60, Followers: domain.ArtistFollowers{Total: 250000}},
	})
	st.SetTopTracks("medium_term", []domain.Track{})
	st.StampStatsFetchedAt("medium_term")
	pane.Update(StatsLoadedMsg{TimeRange: "medium_term"}) //nolint:errcheck
	view := pane.View()
	// Should still show short_term data (The Weeknd) not medium_term data
	assert.Contains(t, view, "The Weeknd")
	assert.NotContains(t, view, "Medium Artist")
}

func TestTopArtistsPane_NotFocusedIgnoresKeys(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	// g key on unfocused pane should not change time range
	initialRange := pane.TimeRange()
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}) //nolint:errcheck
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
		{ID: "a77", Name: "Refreshed Artist", URI: "spotify:artist:a77", Genres: []string{"ambient"}, Popularity: 55, Followers: domain.ArtistFollowers{Total: 120000}},
	})
	st.SetTopTracks("short_term", []domain.Track{})
	st.StampStatsFetchedAt("short_term")
	pane.RefreshRows()
	view := pane.View()
	assert.Contains(t, view, "Refreshed Artist")
}

// ── Story 71 Task 3: column color tokens ─────────────────────────────────────

// TestTopArtistsPane_UsesColumnColors verifies that TopArtistsPane column definitions
// use the new ColumnIndex/ColumnPrimary/ColumnSecondary tokens.
func TestTopArtistsPane_UsesColumnColors(t *testing.T) {
	th := theme.Load("black")
	p := NewTopArtistsPane(state.New(), th, false)
	cols := p.table.Columns()
	require.Len(t, cols, 4, "TopArtistsPane should have 4 columns")

	assert.Equal(t, th.ColumnIndex(), cols[0].Color, "# column should use ColumnIndex()")
	assert.Equal(t, th.ColumnPrimary(), cols[1].Color, "Artist column should use ColumnPrimary()")
	assert.Equal(t, th.ColumnSecondary(), cols[2].Color, "Pop column should use ColumnSecondary()")
	assert.Equal(t, th.ColumnTertiary(), cols[3].Color, "Flw column should use ColumnTertiary()")
}

// ── Story 119: t→g rebind and Enter to play artist ──────────────────────────

// TestTopArtistsPane_TKey_DoesNotCycle verifies that pressing t no longer cycles
// the time range — it passes through to global routing (theme switcher).
func TestTopArtistsPane_TKey_DoesNotCycle(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	pane.SetFocused(true)

	initial := pane.TimeRange()
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}) //nolint:errcheck
	assert.Equal(t, initial, pane.TimeRange())
}

// TestTopArtistsPane_Enter_EmitsPlayContextMsg verifies Enter on a row emits
// PlayContextMsg with the selected artist's URI as ContextURI.
func TestTopArtistsPane_Enter_EmitsPlayContextMsg(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	pane.SetFocused(true)

	// Table starts at row 0 (first artist: a1, The Weeknd).
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should produce a command")

	msg := cmd()
	playMsg, ok := msg.(PlayContextMsg)
	require.True(t, ok, "command should produce PlayContextMsg, got %T", msg)
	assert.Equal(t, "spotify:artist:a1", playMsg.ContextURI)
}

// TestTopArtistsPane_Enter_NoData_NoOp verifies Enter on an empty list returns nil cmd.
func TestTopArtistsPane_Enter_NoData_NoOp(t *testing.T) {
	st := state.New()
	th := theme.Load("black")
	pane := NewTopArtistsPane(st, th, true)
	pane.SetSize(120, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "Enter on empty list should not emit a command")
}

// ── Story 173: Esc scroll-reset ───────────────────────────────────────────────

// TableCurrentPage returns the current page of the top artists pane's inner table.
// White-box accessor for testing Esc scroll-reset (story 173).
func (a *TopArtistsPane) TableCurrentPage() int { return a.table.CurrentPage() }

// TestTopArtistsPane_Esc_ResetsScrollToPage1 verifies that pressing Esc when no
// filter is active resets the table scroll position back to page 1.
func TestTopArtistsPane_Esc_ResetsScrollToPage1(t *testing.T) {
	st := state.New()
	artists := make([]domain.FullArtist, 20)
	for i := range artists {
		artists[i] = domain.FullArtist{
			ID:   fmt.Sprintf("a%d", i),
			Name: fmt.Sprintf("Artist %d", i+1),
			URI:  fmt.Sprintf("spotify:artist:a%d", i),
		}
	}
	st.SetTopArtists("short_term", artists)
	st.StampStatsFetchedAt("short_term")
	th := theme.Load("black")
	pane := NewTopArtistsPane(st, th, true)
	// height=11 → pageSize=5 with ShowHeader=true (pageSize = height - 6).
	pane.SetSize(80, 11)

	// Scroll 8 rows down to advance past page 1.
	for i := 0; i < 8; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyDown})
		pane = m.(*TopArtistsPane)
	}
	require.Greater(t, pane.TableCurrentPage(), 1, "should have scrolled past page 1")

	// Press Esc with no active filter — should reset to page 1.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEscape})
	pane = m.(*TopArtistsPane)
	assert.Equal(t, 1, pane.TableCurrentPage(), "Esc should reset table to page 1")
}

// ── Story 174: Filter_EscCloses ───────────────────────────────────────────────

// TestTopArtistsPane_Filter_EscCloses verifies that Esc while the filter is active
// closes the filter and restores the full list — it does NOT reset scroll position.
func TestTopArtistsPane_Filter_EscCloses(t *testing.T) {
	pane, _ := newTestTopArtistsPane()
	pane.SetFocused(true)

	// Activate filter and type a query that narrows to one row.
	updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pp := updated.(*TopArtistsPane)
	require.True(t, pp.filter.IsActive(), "filter should be active after pressing f")

	for _, r := range "dua" {
		u, _ := pp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		pp = u.(*TopArtistsPane)
	}

	// Press Esc — filter should close.
	updated2, _ := pp.Update(tea.KeyMsg{Type: tea.KeyEscape})
	pp2 := updated2.(*TopArtistsPane)
	assert.False(t, pp2.filter.IsActive(), "Esc should close the filter")
	// Full list should be restored.
	output := pp2.View()
	assert.Contains(t, output, "The Weeknd", "full list should be visible after filter close")
}
