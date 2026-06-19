package panes

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check: RecentlyPlayedPane implements layout.Pane.
var _ layout.Pane = &RecentlyPlayedPane{}

// newTestRecentlyPlayedPane creates a pane wired to a store pre-populated with test data.
func newTestRecentlyPlayedPane() (*RecentlyPlayedPane, *state.Store) {
	st := state.New()
	now := time.Now()
	st.SetRecentlyPlayed([]domain.PlayHistory{
		{
			Track:    domain.Track{ID: "t1", Name: "Track One", URI: "spotify:track:t1", Artists: []domain.Artist{{Name: "Artist A"}}},
			PlayedAt: now.Add(-2 * time.Minute).Format(time.RFC3339),
		},
		{
			Track:    domain.Track{ID: "t2", Name: "Track Two", URI: "spotify:track:t2", Artists: []domain.Artist{{Name: "Artist B"}}},
			PlayedAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
		},
		{
			Track:    domain.Track{ID: "t3", Name: "Another Song", URI: "spotify:track:t3", Artists: []domain.Artist{{Name: "Band C"}}},
			PlayedAt: now.Add(-3 * 24 * time.Hour).Format(time.RFC3339),
		},
	})
	th := theme.Load("black")
	pane := NewRecentlyPlayedPane(st, th, false)
	pane.SetSize(120, 20)
	return pane, st
}

// TestRecentlyPlayedPane_ImplementsLayoutPane verifies the compile-time check.
func TestRecentlyPlayedPane_ImplementsLayoutPane(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	assert.Equal(t, layout.PaneRecentlyPlayed, pane.ID())
}

func TestRecentlyPlayedPane_Metadata(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	assert.Equal(t, layout.PaneRecentlyPlayed, pane.ID())
	assert.Equal(t, "Recently Played", pane.Title())
	assert.Equal(t, 6, pane.ToggleKey())
}

func TestRecentlyPlayedPane_Actions_Default(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "f", actions[0].Key)
}

func TestRecentlyPlayedPane_Actions_FilterActive(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	pane.SetFocused(true)
	pane.SetSize(120, 20)
	// Toggle filter on — Actions() must still return {f, filter}, not {Esc, close}.
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "f", actions[0].Key)
	assert.Equal(t, "filter", actions[0].Label)
}

func TestRecentlyPlayedPane_RendersTrackNames(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	pane.SetFocused(true)
	view := pane.View()
	assert.Contains(t, view, "Track One")
	assert.Contains(t, view, "Track Two")
	assert.Contains(t, view, "Another Song")
}

func TestRecentlyPlayedPane_RendersRelativeTime(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	view := pane.View()
	// Track One played 2 min ago
	assert.Contains(t, view, "min ago")
	// Track Two played 1 hr ago
	assert.Contains(t, view, "hr ago")
	// Track Three played 3 days ago
	assert.Contains(t, view, "days ago")
}

// TestRecentlyPlayedPane_Enter_EmitsPlayTrackListMsg verifies Enter on row N emits
// PlayTrackListMsg with URIs from the selected index onward (Story 105).
func TestRecentlyPlayedPane_Enter_EmitsPlayTrackListMsg(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	pane.SetFocused(true)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should return a command")

	msg := cmd()
	playMsg, ok := msg.(PlayTrackListMsg)
	require.True(t, ok, "command should produce PlayTrackListMsg, got %T", msg)
	// Row 0 selected (t1) → 3 URIs: t1, t2, t3
	require.Len(t, playMsg.URIs, 3, "should include URIs from selected track to end")
	assert.Equal(t, "spotify:track:t1", playMsg.URIs[0])
	assert.Equal(t, "spotify:track:t2", playMsg.URIs[1])
	assert.Equal(t, "spotify:track:t3", playMsg.URIs[2])
}

func TestRecentlyPlayedPane_FilterByTrackName(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	pane.SetFocused(true)

	// Activate filter
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck

	// Type "another"
	for _, r := range "another" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	view := pane.View()
	assert.Contains(t, view, "Another Song")
	assert.NotContains(t, strings.ToLower(view), "track one")
}

func TestRecentlyPlayedPane_FilterByArtistName(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	pane.SetFocused(true)

	// Activate filter
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck

	// Type "Artist A"
	for _, r := range "Artist A" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	view := pane.View()
	assert.Contains(t, view, "Track One")
	assert.NotContains(t, view, "Track Two")
}

func TestRecentlyPlayedPane_EmptyData(t *testing.T) {
	st := state.New()
	th := theme.Load("black")
	pane := NewRecentlyPlayedPane(st, th, false)
	pane.SetSize(120, 20)
	// Should not panic and should show empty state message.
	view := pane.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "No recently played tracks")
}

func TestRecentlyPlayedPane_RefreshRows(t *testing.T) {
	pane, st := newTestRecentlyPlayedPane()
	now := time.Now()
	// Update store directly
	st.SetRecentlyPlayed([]domain.PlayHistory{
		{
			Track:    domain.Track{ID: "t99", Name: "New Track", URI: "spotify:track:t99", Artists: []domain.Artist{{Name: "New Artist"}}},
			PlayedAt: now.Add(-10 * time.Minute).Format(time.RFC3339),
		},
	})
	pane.RefreshRows()
	view := pane.View()
	assert.Contains(t, view, "New Track")
}

func TestRecentlyPlayedPane_RecentlyPlayedLoadedMsg(t *testing.T) {
	pane, st := newTestRecentlyPlayedPane()
	now := time.Now()
	// Simulate a RecentlyPlayedLoadedMsg (app writes store then sends msg)
	st.SetRecentlyPlayed([]domain.PlayHistory{
		{
			Track:    domain.Track{ID: "t55", Name: "Loaded Track", URI: "spotify:track:t55", Artists: []domain.Artist{{Name: "Loaded Artist"}}},
			PlayedAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
	})
	pane.Update(RecentlyPlayedLoadedMsg{}) //nolint:errcheck
	view := pane.View()
	assert.Contains(t, view, "Loaded Track")
}

func TestRecentlyPlayedPane_NotFocusedIgnoresKeys(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	// pane is not focused — Enter should not emit a command
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
}

func TestRecentlyPlayedPane_SetFocused(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	assert.False(t, pane.IsFocused())
	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
	pane.SetFocused(false)
	assert.False(t, pane.IsFocused())
}

func TestRecentlyPlayedPane_Init(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	cmd := pane.Init()
	assert.Nil(t, cmd)
}

// ── Story 71 Task 2: column color tokens ─────────────────────────────────────

// TestRecentlyPlayedPane_UsesColumnColors verifies that RecentlyPlayedPane column
// definitions use the new ColumnIndex/ColumnPrimary/ColumnSecondary/ColumnTertiary tokens.
func TestRecentlyPlayedPane_UsesColumnColors(t *testing.T) {
	th := theme.Load("black")
	p := NewRecentlyPlayedPane(state.New(), th, false)
	p.SetSize(80, 20)
	cols := p.table.Columns()
	require.Len(t, cols, 4, "RecentlyPlayedPane should have 4 columns")

	assert.Equal(t, th.ColumnIndex(), cols[0].Color, "Index column should use ColumnIndex()")
	assert.Equal(t, th.ColumnPrimary(), cols[1].Color, "Track column should use ColumnPrimary()")
	assert.Equal(t, th.ColumnSecondary(), cols[2].Color, "Artist column should use ColumnSecondary()")
	assert.Equal(t, th.ColumnTertiary(), cols[3].Color, "Played column should use ColumnTertiary()")
}

// ── Story 173: Esc scroll-reset ───────────────────────────────────────────────

// TableCurrentPage returns the current page of the recently played pane's inner table.
// White-box accessor for testing Esc scroll-reset (story 173).
func (p *RecentlyPlayedPane) TableCurrentPage() int { return p.table.CurrentPage() }

// TestRecentlyPlayedPane_Esc_ResetsScrollToPage1 verifies that pressing Esc when no
// filter is active resets the table scroll position back to page 1.
func TestRecentlyPlayedPane_Esc_ResetsScrollToPage1(t *testing.T) {
	st := state.New()
	now := time.Now()
	histories := make([]domain.PlayHistory, 20)
	for i := range histories {
		histories[i] = domain.PlayHistory{
			Track:    domain.Track{ID: fmt.Sprintf("t%d", i), Name: fmt.Sprintf("Track %d", i+1), URI: fmt.Sprintf("spotify:track:t%d", i), Artists: []domain.Artist{{Name: "Artist"}}},
			PlayedAt: now.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
		}
	}
	st.SetRecentlyPlayed(histories)
	th := theme.Load("black")
	pane := NewRecentlyPlayedPane(st, th, true)
	// height=11 → pageSize=5 with ShowHeader=true (pageSize = height - 4).
	pane.SetSize(80, 11)

	// Scroll 8 rows down to advance past page 1.
	for i := 0; i < 8; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyDown})
		pane = m.(*RecentlyPlayedPane)
	}
	require.Greater(t, pane.TableCurrentPage(), 1, "should have scrolled past page 1")

	// Press Esc with no active filter — should reset to page 1.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEscape})
	pane = m.(*RecentlyPlayedPane)
	assert.Equal(t, 1, pane.TableCurrentPage(), "Esc should reset table to page 1")
}

// ── Story 174: Filter_EscCloses ───────────────────────────────────────────────

// TestRecentlyPlayedPane_Filter_EscCloses verifies that Esc while the filter is active
// closes the filter and does NOT reset scroll position.
func TestRecentlyPlayedPane_Filter_EscCloses(t *testing.T) {
	st := state.New()
	now := time.Now()
	histories := make([]domain.PlayHistory, 20)
	for i := range histories {
		histories[i] = domain.PlayHistory{
			Track:    domain.Track{ID: fmt.Sprintf("t%d", i), Name: fmt.Sprintf("Track %d", i+1), URI: fmt.Sprintf("spotify:track:t%d", i), Artists: []domain.Artist{{Name: "Artist"}}},
			PlayedAt: now.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
		}
	}
	st.SetRecentlyPlayed(histories)
	th := theme.Load("black")
	pane := NewRecentlyPlayedPane(st, th, true)
	pane.SetSize(80, 11) // pageSize=5

	// Scroll to page 2 before activating the filter.
	for i := 0; i < 8; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyDown})
		pane = m.(*RecentlyPlayedPane)
	}
	pageBeforeFilter := pane.TableCurrentPage()
	require.Greater(t, pageBeforeFilter, 1, "pre-condition: should be past page 1")

	// Activate filter.
	updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pp := updated.(*RecentlyPlayedPane)
	require.True(t, pp.filter.IsActive(), "filter should be active after pressing f")

	// Press Esc — filter should close without resetting scroll.
	updated2, _ := pp.Update(tea.KeyMsg{Type: tea.KeyEscape})
	pp2 := updated2.(*RecentlyPlayedPane)
	assert.False(t, pp2.filter.IsActive(), "Esc should close the filter")
	assert.Equal(t, pageBeforeFilter, pp2.TableCurrentPage(), "Esc should NOT reset scroll when closing the filter")
	assert.Contains(t, pp2.View(), "Track", "full list should be visible after filter close")
}

// TestRecentlyPlayedPane_ActiveFilterQuery_ReturnsCommittedQuery verifies that
// ActiveFilterQuery() reflects the committed query after f → type → Enter.
func TestRecentlyPlayedPane_ActiveFilterQuery_ReturnsCommittedQuery(t *testing.T) {
	st := state.New()
	st.SetRecentlyPlayed([]domain.PlayHistory{{Track: domain.Track{Name: "Rock Track", URI: "uri:1"}}})
	pane := NewRecentlyPlayedPane(st, theme.Load("black"), true)
	pane.SetSize(80, 20)

	assert.Equal(t, "", pane.ActiveFilterQuery(), "empty before filter applied")

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	for _, r := range "rock" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	pane.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, "rock", pane.ActiveFilterQuery())
}

// TestRecentlyPlayedPane_Esc_ClearsCommittedFilter verifies that Esc clears a committed
// filter query before falling back to scroll-reset.
func TestRecentlyPlayedPane_Esc_ClearsCommittedFilter(t *testing.T) {
	st := state.New()
	st.SetRecentlyPlayed([]domain.PlayHistory{
		{Track: domain.Track{Name: "Rock Track", URI: "uri:1"}},
		{Track: domain.Track{Name: "Jazz Track", URI: "uri:2"}},
	})
	pane := NewRecentlyPlayedPane(st, theme.Load("black"), true)
	pane.SetSize(80, 20)

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	for _, r := range "rock" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Equal(t, "rock", pane.ActiveFilterQuery(), "filter must be committed")

	pane.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, "", pane.ActiveFilterQuery(), "Esc must clear committed filter")
}
