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

// Compile-time check: PlaylistsPane implements layout.Pane.
var _ layout.Pane = &PlaylistsPane{}

// newTestPlaylistsPane creates a PlaylistsPane with a fresh store and black theme.
func newTestPlaylistsPane(focused bool) *PlaylistsPane {
	s := state.New()
	th := theme.Load("black")
	return NewPlaylistsPane(s, th, focused)
}

// newTestPlaylistsPaneWithData creates a PlaylistsPane pre-loaded with playlists.
func newTestPlaylistsPaneWithData(focused bool) *PlaylistsPane {
	s := state.New()
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi", URI: "spotify:playlist:pl1", TrackCount: 42},
		{ID: "pl2", Name: "Best of Coke Studio", URI: "spotify:playlist:pl2", TrackCount: 28},
		{ID: "pl3", Name: "Soul", URI: "spotify:playlist:pl3", TrackCount: 15},
	})
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, focused)
	return pane
}

// TestPlaylistsPane_ImplementsLayoutPane verifies the interface is satisfied.
func TestPlaylistsPane_ImplementsLayoutPane(t *testing.T) {
	pane := newTestPlaylistsPane(false)
	assert.NotNil(t, pane)
}

// TestPlaylistsPane_ID verifies the pane ID.
func TestPlaylistsPane_ID(t *testing.T) {
	pane := newTestPlaylistsPane(false)
	assert.Equal(t, layout.PanePlaylists, pane.ID())
}

// TestPlaylistsPane_Title returns "Playlists" in list view.
func TestPlaylistsPane_Title(t *testing.T) {
	pane := newTestPlaylistsPane(false)
	assert.Equal(t, "Playlists", pane.Title())
}

// TestPlaylistsPane_ToggleKey returns 3.
func TestPlaylistsPane_ToggleKey(t *testing.T) {
	pane := newTestPlaylistsPane(false)
	assert.Equal(t, 3, pane.ToggleKey())
}

// TestPlaylistsPane_Actions_ListView returns standard actions when not in track view.
func TestPlaylistsPane_Actions_ListView(t *testing.T) {
	pane := newTestPlaylistsPane(true)
	actions := pane.Actions()
	keys := make([]string, len(actions))
	for i, a := range actions {
		keys[i] = a.Key
	}
	assert.Contains(t, keys, "f", "should have filter action")
	assert.Contains(t, keys, "n", "should have new action")
	assert.Contains(t, keys, "r", "should have rename action")
}

// TestPlaylistsPane_Actions_FilterActive returns close action when filter is active.
func TestPlaylistsPane_Actions_FilterActive(t *testing.T) {
	pane := newTestPlaylistsPane(true)
	pane.SetSize(80, 20)
	// Activate filter
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "Esc", actions[0].Key)
}

// TestPlaylistsPane_Actions_TrackView returns back and reorder actions in track view.
func TestPlaylistsPane_Actions_TrackView(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	// Open track sub-view by pressing Enter
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should return a command for FetchPlaylistTracksRequestMsg")
	// Manually set inTrackView=true to test actions
	pane.inTrackView = true
	actions := pane.Actions()
	keys := make([]string, len(actions))
	for i, a := range actions {
		keys[i] = a.Key
	}
	assert.Contains(t, keys, "Esc", "track view should have Esc action")
	assert.Contains(t, keys, "Shift+↕", "track view should have reorder action")
}

// TestPlaylistsPane_View_EmptyPlaylists verifies clean render on empty data.
func TestPlaylistsPane_View_EmptyPlaylists(t *testing.T) {
	pane := newTestPlaylistsPane(true)
	pane.SetSize(80, 20)
	output := pane.View()
	assert.NotEmpty(t, output, "should return non-empty string for empty playlists")
}

// TestPlaylistsPane_View_ShowsPlaylists verifies playlist names appear in list.
func TestPlaylistsPane_View_ShowsPlaylists(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	output := pane.View()
	assert.Contains(t, output, "LoFi", "first playlist should appear")
	assert.Contains(t, output, "Soul", "third playlist should appear")
}

// TestPlaylistsPane_Enter_EmitsTracksFetchRequest verifies Enter on a playlist emits FetchPlaylistTracksRequestMsg.
func TestPlaylistsPane_Enter_EmitsTracksFetchRequest(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	// Cursor is at row 0 (first playlist: pl1)
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should return a command")

	msg := cmd()
	req, ok := msg.(FetchPlaylistTracksRequestMsg)
	require.True(t, ok, "command should produce FetchPlaylistTracksRequestMsg")
	assert.Equal(t, "pl1", req.PlaylistID)
}

// TestPlaylistsPane_Esc_ReturnsToListView verifies Esc exits track sub-view.
func TestPlaylistsPane_Esc_ReturnsToListView(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	// Manually put pane in track view
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedName = "LoFi"

	updated, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEsc})
	pp, ok := updated.(*PlaylistsPane)
	require.True(t, ok)
	assert.False(t, pp.inTrackView, "Esc should close track sub-view")
	assert.Nil(t, cmd, "Esc back to list should not produce a command")
}

// TestPlaylistsPane_N_EmitsCreateRequest verifies 'n' emits PlaylistCreateRequestMsg.
func TestPlaylistsPane_N_EmitsCreateRequest(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	require.NotNil(t, cmd, "'n' should return a command")

	msg := cmd()
	_, ok := msg.(PlaylistCreateRequestMsg)
	assert.True(t, ok, "command should produce PlaylistCreateRequestMsg")
}

// TestPlaylistsPane_R_EmitsRenameRequest verifies 'r' emits PlaylistRenameRequestMsg.
func TestPlaylistsPane_R_EmitsRenameRequest(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	require.NotNil(t, cmd, "'r' should return a command")

	msg := cmd()
	req, ok := msg.(PlaylistRenameRequestMsg)
	assert.True(t, ok, "command should produce PlaylistRenameRequestMsg")
	assert.Equal(t, "pl1", req.PlaylistID)
}

// TestPlaylistsPane_X_EmitsRemoveRequest verifies 'x' in track view emits PlaylistRemoveRequestMsg.
func TestPlaylistsPane_X_EmitsRemoveRequest(t *testing.T) {
	s := state.New()
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi", URI: "spotify:playlist:pl1", TrackCount: 1},
	})
	tracks := []domain.Track{
		{ID: "t1", Name: "Snowman", URI: "spotify:track:t1", Artists: []domain.Artist{{Name: "Sia"}}},
	}
	s.SetPlaylistTracks("pl1", tracks)
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	// Switch to track sub-view
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedName = "LoFi"
	pane.refreshTrackRows()

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	require.NotNil(t, cmd, "'x' should return a command in track view")

	msg := cmd()
	req, ok := msg.(PlaylistRemoveRequestMsg)
	assert.True(t, ok, "command should produce PlaylistRemoveRequestMsg")
	assert.Equal(t, "pl1", req.PlaylistID)
}

// TestPlaylistsPane_ShiftUp_EmitsReorderRequest verifies Shift+Up reorders tracks.
func TestPlaylistsPane_ShiftUp_EmitsReorderRequest(t *testing.T) {
	s := state.New()
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi", URI: "spotify:playlist:pl1", TrackCount: 2},
	})
	tracks := []domain.Track{
		{ID: "t1", Name: "Snowman", URI: "spotify:track:t1"},
		{ID: "t2", Name: "Coffee", URI: "spotify:track:t2"},
	}
	s.SetPlaylistTracks("pl1", tracks)
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	// Switch to track sub-view
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedName = "LoFi"
	pane.refreshTrackRows()
	pane.trackTable.SetFocused(true)

	// Move cursor to row 1, then press Shift+Up
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) //nolint:errcheck
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	require.NotNil(t, cmd, "Shift+Up should return a command")

	msg := cmd()
	req, ok := msg.(PlaylistReorderRequestMsg)
	assert.True(t, ok, "command should produce PlaylistReorderRequestMsg")
	assert.Equal(t, "pl1", req.PlaylistID)
	assert.Equal(t, 1, req.RangeStart)
	assert.Equal(t, 0, req.InsertBefore)
}

// TestPlaylistsPane_ShiftDown_EmitsReorderRequest verifies Shift+Down reorders tracks.
func TestPlaylistsPane_ShiftDown_EmitsReorderRequest(t *testing.T) {
	s := state.New()
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi", URI: "spotify:playlist:pl1", TrackCount: 2},
	})
	tracks := []domain.Track{
		{ID: "t1", Name: "Snowman", URI: "spotify:track:t1"},
		{ID: "t2", Name: "Coffee", URI: "spotify:track:t2"},
	}
	s.SetPlaylistTracks("pl1", tracks)
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedName = "LoFi"
	pane.refreshTrackRows()

	// Cursor at 0, press Shift+Down
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	require.NotNil(t, cmd, "Shift+Down should return a command")

	msg := cmd()
	req, ok := msg.(PlaylistReorderRequestMsg)
	assert.True(t, ok, "command should produce PlaylistReorderRequestMsg")
	assert.Equal(t, "pl1", req.PlaylistID)
	assert.Equal(t, 0, req.RangeStart)
	assert.Equal(t, 2, req.InsertBefore)
}

// TestPlaylistsPane_Filter_FiltersPlaylists verifies filter narrows the list.
func TestPlaylistsPane_Filter_FiltersPlaylists(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	// Activate filter and type "lofi"
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "lofi" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	output := pane.View()
	assert.Contains(t, output, "LoFi", "filter should show matching playlist")
}

// TestPlaylistsPane_TitleInTrackView verifies dynamic title in track sub-view.
func TestPlaylistsPane_TitleInTrackView(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedName = "LoFi"

	title := pane.Title()
	assert.Contains(t, title, "LoFi", "title should contain playlist name in track view")
	assert.Contains(t, title, "Playlists", "title should still contain Playlists prefix")
}

// TestPlaylistsPane_IsFocused verifies SetFocused/IsFocused.
func TestPlaylistsPane_IsFocused(t *testing.T) {
	pane := newTestPlaylistsPane(false)
	assert.False(t, pane.IsFocused())
	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
}

// TestPlaylistsPane_IgnoresInputWhenUnfocused verifies pane ignores input when not focused.
func TestPlaylistsPane_IgnoresInputWhenUnfocused(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(false)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "unfocused pane should not emit commands")
}

// TestPlaylistsPane_LibraryLoadedMsg_RefreshesTable verifies data-load integration.
func TestPlaylistsPane_LibraryLoadedMsg_RefreshesTable(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	// Store already has playlists set by app.Update before sending LibraryLoadedMsg
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "NewPlaylist", URI: "spotify:playlist:pl1", TrackCount: 5},
	})
	// Send LibraryLoadedMsg — pane should refresh from store
	pane.Update(LibraryLoadedMsg{}) //nolint:errcheck

	output := pane.View()
	assert.Contains(t, output, "NewPlaylist", "pane should show newly loaded playlist after LibraryLoadedMsg")
}

// TestPlaylistsPane_PlaylistCreatedMsg_RefreshesList verifies playlist creation refresh.
func TestPlaylistsPane_PlaylistCreatedMsg_RefreshesList(t *testing.T) {
	s := state.New()
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi", URI: "spotify:playlist:pl1", TrackCount: 10},
	})
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	// Add new playlist to store (simulating what app.Update does), then send msg
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi", URI: "spotify:playlist:pl1", TrackCount: 10},
		{ID: "pl2", Name: "NewList", URI: "spotify:playlist:pl2", TrackCount: 0},
	})
	pane.Update(PlaylistCreatedMsg{PlaylistID: "pl2", Name: "NewList"}) //nolint:errcheck

	output := pane.View()
	assert.Contains(t, output, "NewList", "pane should show new playlist after PlaylistCreatedMsg")
}

// TestPlaylistsPane_PlaylistTracksLoadedMsg_ShowsTracks verifies track sub-view refresh.
func TestPlaylistsPane_PlaylistTracksLoadedMsg_ShowsTracks(t *testing.T) {
	s := state.New()
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi", URI: "spotify:playlist:pl1", TrackCount: 2},
	})
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedName = "LoFi"
	pane.resizeTable() // ensure track table is sized correctly
	pane.trackTable.SetFocused(true)

	// Simulate store update by app.Update, then send msg
	tracks := []domain.Track{
		{ID: "t1", Name: "Snowman", URI: "spotify:track:t1", Artists: []domain.Artist{{Name: "Sia"}}},
	}
	s.SetPlaylistTracks("pl1", tracks)
	pane.Update(PlaylistTracksLoadedMsg{PlaylistID: "pl1", Tracks: tracks}) //nolint:errcheck

	output := pane.View()
	assert.Contains(t, output, "Snowman", "track sub-view should show loaded track")
}

// TestPlaylistsPane_LargePlaylistList verifies no panic with many playlists.
func TestPlaylistsPane_LargePlaylistList(t *testing.T) {
	s := state.New()
	playlists := make([]domain.SimplePlaylist, 100)
	for i := range playlists {
		playlists[i] = domain.SimplePlaylist{
			ID:         fmt.Sprintf("pl%d", i),
			Name:       fmt.Sprintf("Playlist %d", i+1),
			URI:        fmt.Sprintf("spotify:playlist:pl%d", i),
			TrackCount: i * 2,
		}
	}
	s.SetPlaylists(playlists)
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	// Should not panic
	output := pane.View()
	assert.NotEmpty(t, output, "large playlist list should render without panic")
}

// TestPlaylistsPane_RefreshRows_UpdatesTable verifies the exported RefreshRows method.
func TestPlaylistsPane_RefreshRows_UpdatesTable(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "RefreshedPlaylist", URI: "spotify:playlist:pl1", TrackCount: 5},
	})
	pane.RefreshRows()

	output := pane.View()
	assert.Contains(t, output, "RefreshedPlaylist", "RefreshRows should update the view")
}

// ── Story 71 Task 4: column color tokens ─────────────────────────────────────

// TestPlaylistsPane_UsesColumnColors verifies that PlaylistsPane column definitions
// (both list view and track sub-view) use the new column color tokens.
func TestPlaylistsPane_UsesColumnColors(t *testing.T) {
	th := theme.Load("black")
	p := NewPlaylistsPane(state.New(), th, false)

	// List view: # → ColumnIndex, Name → ColumnPrimary, Tracks → ColumnTertiary
	listCols := p.table.Columns()
	require.Len(t, listCols, 3, "PlaylistsPane list table should have 3 columns")
	assert.Equal(t, th.ColumnIndex(), listCols[0].Color, "# column should use ColumnIndex()")
	assert.Equal(t, th.ColumnPrimary(), listCols[1].Color, "Name column should use ColumnPrimary()")
	assert.Equal(t, th.ColumnTertiary(), listCols[2].Color, "Tracks column should use ColumnTertiary()")

	// Track sub-view: # → ColumnIndex, Track → ColumnPrimary, Artist → ColumnSecondary, Duration → ColumnTertiary
	trackCols := p.trackTable.Columns()
	require.Len(t, trackCols, 4, "PlaylistsPane track table should have 4 columns")
	assert.Equal(t, th.ColumnIndex(), trackCols[0].Color, "# column should use ColumnIndex()")
	assert.Equal(t, th.ColumnPrimary(), trackCols[1].Color, "Track column should use ColumnPrimary()")
	assert.Equal(t, th.ColumnSecondary(), trackCols[2].Color, "Artist column should use ColumnSecondary()")
	assert.Equal(t, th.ColumnTertiary(), trackCols[3].Color, "Duration column should use ColumnTertiary()")
}
