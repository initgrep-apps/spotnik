package panes

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check: LikedSongsPane implements layout.Pane.
var _ layout.Pane = &LikedSongsPane{}

// newTestLikedSongsPane creates a LikedSongsPane with a fresh store and black theme.
func newTestLikedSongsPane(focused bool) *LikedSongsPane {
	s := state.New()
	th := theme.Load("black")
	return NewLikedSongsPane(s, th, focused)
}

// newTestLikedSongsPaneWithData creates a LikedSongsPane pre-loaded with liked tracks.
func newTestLikedSongsPaneWithData(focused bool) *LikedSongsPane {
	s := state.New()
	s.SetLikedTracks([]domain.SavedTrack{
		{
			Track: domain.Track{
				ID:         "t1",
				Name:       "Blinding Lights",
				URI:        "spotify:track:t1",
				DurationMs: 202000, // 3:22
				Artists:    []domain.Artist{{Name: "The Weeknd"}},
			},
		},
		{
			Track: domain.Track{
				ID:         "t2",
				Name:       "Save Your Tears",
				URI:        "spotify:track:t2",
				DurationMs: 215000, // 3:35
				Artists:    []domain.Artist{{Name: "The Weeknd"}},
			},
		},
		{
			Track: domain.Track{
				ID:         "t3",
				Name:       "Levitating",
				URI:        "spotify:track:t3",
				DurationMs: 203000, // 3:23
				Artists:    []domain.Artist{{Name: "Dua Lipa"}},
			},
		},
	})
	th := theme.Load("black")
	return NewLikedSongsPane(s, th, focused)
}

// TestLikedSongsPane_ImplementsLayoutPane verifies the interface is satisfied.
func TestLikedSongsPane_ImplementsLayoutPane(t *testing.T) {
	pane := newTestLikedSongsPane(false)
	assert.NotNil(t, pane)
}

// TestLikedSongsPane_ID verifies the pane ID.
func TestLikedSongsPane_ID(t *testing.T) {
	pane := newTestLikedSongsPane(false)
	assert.Equal(t, layout.PaneLikedSongs, pane.ID())
}

// TestLikedSongsPane_Title returns "Liked Songs".
func TestLikedSongsPane_Title(t *testing.T) {
	pane := newTestLikedSongsPane(false)
	assert.Equal(t, "Liked Songs", pane.Title())
}

// TestLikedSongsPane_ToggleKey returns 5.
func TestLikedSongsPane_ToggleKey(t *testing.T) {
	pane := newTestLikedSongsPane(false)
	assert.Equal(t, 5, pane.ToggleKey())
}

// TestLikedSongsPane_Actions_Default returns only filter action.
// 'i' (like/unlike) was removed in story 120: the feature returned 403 and was pulled.
func TestLikedSongsPane_Actions_Default(t *testing.T) {
	pane := newTestLikedSongsPane(true)
	actions := pane.Actions()
	keys := make([]string, len(actions))
	for i, a := range actions {
		keys[i] = a.Key
	}
	assert.Contains(t, keys, "f", "should have filter action")
	assert.NotContains(t, keys, "i", "i (like/unlike) must be absent")
}

// TestLikedSongsPane_Actions_FilterActive returns close action when filter is active.
func TestLikedSongsPane_Actions_FilterActive(t *testing.T) {
	pane := newTestLikedSongsPane(true)
	pane.SetSize(80, 20)
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "Esc", actions[0].Key)
}

// TestLikedSongsPane_View_Empty verifies clean render on empty data.
func TestLikedSongsPane_View_Empty(t *testing.T) {
	pane := newTestLikedSongsPane(true)
	pane.SetSize(80, 20)
	output := pane.View()
	assert.NotEmpty(t, output, "should return non-empty string for empty liked songs")
}

// TestLikedSongsPane_View_ShowsTracks verifies liked song tracks appear.
func TestLikedSongsPane_View_ShowsTracks(t *testing.T) {
	pane := newTestLikedSongsPaneWithData(true)
	pane.SetSize(80, 20)
	output := pane.View()
	assert.Contains(t, output, "Blinding Lights", "first track should appear")
	assert.Contains(t, output, "Levitating", "third track should appear")
}

// TestLikedSongsPane_View_ShowsDuration verifies duration is formatted as M:SS.
func TestLikedSongsPane_View_ShowsDuration(t *testing.T) {
	pane := newTestLikedSongsPaneWithData(true)
	pane.SetSize(80, 20)
	output := pane.View()
	assert.Contains(t, output, "3:22", "duration should be formatted as M:SS")
}

// TestLikedSongsPane_Enter_EmitsPlayContextMsg verifies Enter on row N emits
// PlayContextMsg with the liked songs collection context and the selected track's URI
// as OffsetURI (Story 105: context-aware playback).
func TestLikedSongsPane_Enter_EmitsPlayContextMsg(t *testing.T) {
	pane := newTestLikedSongsPaneWithData(true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should return a command")

	msg := cmd()
	playMsg, ok := msg.(PlayContextMsg)
	require.True(t, ok, "command should produce PlayContextMsg, got %T", msg)
	assert.Equal(t, "spotify:collection:tracks", playMsg.ContextURI,
		"ContextURI should be the liked songs collection")
	assert.Equal(t, "spotify:track:t1", playMsg.OffsetURI,
		"OffsetURI should be the selected track URI")
}

// TestLikedSongsPane_Enter_EmptyList_EmitsNoCommand verifies Enter on an empty list
// emits no command (idx == -1 bounds guard preserved).
func TestLikedSongsPane_Enter_EmptyList_EmitsNoCommand(t *testing.T) {
	pane := newTestLikedSongsPane(true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "Enter on empty list should not emit a command")
}

// TestLikedSongsPane_I_IsNoOp verifies 'i' is a no-op after handler removal (story 120).
// The like/unlike feature always returned 403; it has been removed entirely.
func TestLikedSongsPane_I_IsNoOp(t *testing.T) {
	pane := newTestLikedSongsPaneWithData(true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	assert.Nil(t, cmd, "'i' handler was removed; should return nil cmd")
}

// TestLikedSongsPane_Filter_ByTrackName verifies filter narrows results by track name.
func TestLikedSongsPane_Filter_ByTrackName(t *testing.T) {
	pane := newTestLikedSongsPaneWithData(true)
	pane.SetSize(80, 20)

	// Activate filter and type "blinding"
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "blinding" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	output := pane.View()
	assert.Contains(t, output, "Blinding Lights", "filter should show matching track")
}

// TestLikedSongsPane_Filter_ByArtistName verifies filter matches artist name.
func TestLikedSongsPane_Filter_ByArtistName(t *testing.T) {
	pane := newTestLikedSongsPaneWithData(true)
	pane.SetSize(80, 20)

	// Activate filter and type "dua"
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "dua" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	output := pane.View()
	assert.Contains(t, output, "Levitating", "filter by artist should show matching track")
}

// TestLikedSongsPane_IsFocused verifies SetFocused/IsFocused.
func TestLikedSongsPane_IsFocused(t *testing.T) {
	pane := newTestLikedSongsPane(false)
	assert.False(t, pane.IsFocused())
	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
}

// TestLikedSongsPane_IgnoresInputWhenUnfocused verifies pane ignores input when not focused.
func TestLikedSongsPane_IgnoresInputWhenUnfocused(t *testing.T) {
	pane := newTestLikedSongsPaneWithData(false)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "unfocused pane should not emit commands")
}

// TestLikedSongsPane_LikedTracksLoadedMsg_RefreshesTable verifies data-load integration.
func TestLikedSongsPane_LikedTracksLoadedMsg_RefreshesTable(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLikedSongsPane(s, th, true)
	pane.SetSize(80, 20)

	s.SetLikedTracks([]domain.SavedTrack{
		{
			Track: domain.Track{
				ID:      "t1",
				Name:    "New Track",
				URI:     "spotify:track:t1",
				Artists: []domain.Artist{{Name: "Artist"}},
			},
		},
	})
	pane.Update(LikedTracksLoadedMsg{}) //nolint:errcheck

	output := pane.View()
	assert.Contains(t, output, "New Track", "pane should show newly loaded track after LikedTracksLoadedMsg")
}

// TestLikedSongsPane_LargeTracklist verifies no panic with many tracks.
func TestLikedSongsPane_LargeTracklist(t *testing.T) {
	s := state.New()
	tracks := make([]domain.SavedTrack, 200)
	for i := range tracks {
		tracks[i] = domain.SavedTrack{
			Track: domain.Track{
				ID:      fmt.Sprintf("t%d", i),
				Name:    fmt.Sprintf("Track %d", i+1),
				URI:     fmt.Sprintf("spotify:track:t%d", i),
				Artists: []domain.Artist{{Name: "Artist"}},
			},
		}
	}
	s.SetLikedTracks(tracks)
	th := theme.Load("black")
	pane := NewLikedSongsPane(s, th, true)
	pane.SetSize(80, 20)

	output := pane.View()
	assert.NotEmpty(t, output, "large track list should render without panic")
}

// TestLikedSongsPane_RefreshRows_UpdatesTable verifies the exported RefreshRows method.
func TestLikedSongsPane_RefreshRows_UpdatesTable(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLikedSongsPane(s, th, true)
	pane.SetSize(80, 20)

	s.SetLikedTracks([]domain.SavedTrack{
		{
			Track: domain.Track{
				ID:      "t1",
				Name:    "RefreshedTrack",
				URI:     "spotify:track:t1",
				Artists: []domain.Artist{{Name: "Artist"}},
			},
		},
	})
	pane.RefreshRows()

	output := pane.View()
	assert.Contains(t, output, "RefreshedTrack", "RefreshRows should update the view")
}

// TestLikedSongsPane_I_EmptyList does not panic on empty list.
// Handler was removed in story 120; 'i' is always a no-op regardless.
func TestLikedSongsPane_I_EmptyList(t *testing.T) {
	pane := newTestLikedSongsPane(true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	assert.Nil(t, cmd, "'i' should return nil cmd (handler removed)")
}

// ── Story 120: dead pane action removal ──────────────────────────────────────

// TestLikedSongsPane_Actions_NoLikeEntry verifies 'i' is not in Actions()
// after dead action removal (story 120).
func TestLikedSongsPane_Actions_NoLikeEntry(t *testing.T) {
	pane := newTestLikedSongsPane(false)
	for _, a := range pane.Actions() {
		assert.NotEqual(t, "i", a.Key, "Actions() must not include 'i'")
	}
}

// ── Story 71 Task 2: column color tokens ─────────────────────────────────────

// TestLikedSongsPane_UsesColumnColors verifies that LikedSongsPane column definitions
// use the new ColumnIndex/ColumnPrimary/ColumnSecondary/ColumnTertiary tokens.
func TestLikedSongsPane_UsesColumnColors(t *testing.T) {
	th := theme.Load("black")
	l := NewLikedSongsPane(state.New(), th, false)
	cols := l.table.Columns()
	require.Len(t, cols, 4, "LikedSongsPane should have 4 columns")

	assert.Equal(t, th.ColumnIndex(), cols[0].Color, "# column should use ColumnIndex()")
	assert.Equal(t, th.ColumnPrimary(), cols[1].Color, "Track column should use ColumnPrimary()")
	assert.Equal(t, th.ColumnSecondary(), cols[2].Color, "Artist column should use ColumnSecondary()")
	assert.Equal(t, th.ColumnTertiary(), cols[3].Color, "Duration column should use ColumnTertiary()")
}

// ── Story 173: Esc scroll-reset ───────────────────────────────────────────────

// TableCurrentPage returns the current page of the liked songs pane's inner table.
// White-box accessor for testing Esc scroll-reset (story 173).
func (l *LikedSongsPane) TableCurrentPage() int { return l.table.CurrentPage() }

// TestLikedSongsPane_Esc_ResetsScrollToPage1 verifies that pressing Esc when no
// filter is active resets the table scroll position back to page 1.
func TestLikedSongsPane_Esc_ResetsScrollToPage1(t *testing.T) {
	st := state.New()
	tracks := make([]domain.SavedTrack, 20)
	for i := range tracks {
		tracks[i] = domain.SavedTrack{
			Track: domain.Track{
				ID:      fmt.Sprintf("t%d", i),
				Name:    fmt.Sprintf("Track %d", i+1),
				URI:     fmt.Sprintf("spotify:track:t%d", i),
				Artists: []domain.Artist{{Name: "Artist"}},
			},
		}
	}
	st.SetLikedTracks(tracks)
	th := theme.Load("black")
	pane := NewLikedSongsPane(st, th, true)
	// height=11 → pageSize=5 with ShowHeader=true (pageSize = height - 6).
	pane.SetSize(80, 11)

	// Scroll 8 rows down to advance past page 1.
	for i := 0; i < 8; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyDown})
		pane = m.(*LikedSongsPane)
	}
	require.Greater(t, pane.TableCurrentPage(), 1, "should have scrolled past page 1")

	// Press Esc with no active filter — should reset to page 1.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEscape})
	pane = m.(*LikedSongsPane)
	assert.Equal(t, 1, pane.TableCurrentPage(), "Esc should reset table to page 1")
}
