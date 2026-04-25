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
// n and r actions were removed in story 120 (dead pane action removal).
func TestPlaylistsPane_Actions_ListView(t *testing.T) {
	pane := newTestPlaylistsPane(true)
	actions := pane.Actions()
	keys := make([]string, len(actions))
	for i, a := range actions {
		keys[i] = a.Key
	}
	assert.Contains(t, keys, "f", "should have filter action")
	assert.NotContains(t, keys, "n", "n (new playlist stub) must be absent")
	assert.NotContains(t, keys, "r", "r (rename stub) must be absent")
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

// TestPlaylistsPane_Actions_TrackView returns only Esc back action in track view (story 106).
func TestPlaylistsPane_Actions_TrackView(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	// Manually set inTrackView=true to test actions
	pane.inTrackView = true
	actions := pane.Actions()
	require.Len(t, actions, 1, "track view should have exactly one action")
	assert.Equal(t, "Esc", actions[0].Key, "track view should show Esc action")
	assert.Equal(t, "back", actions[0].Label, "track view Esc action should be labeled 'back'")
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

// TestPlaylistsPane_Enter_EmitsTracksFetchRequest verifies Enter on a playlist
// eventually leads to FetchPlaylistTracksRequestMsg after the debounce resolves.
// The immediate cmd is a debounce tick; FetchPlaylistTracksRequestMsg fires after.
func TestPlaylistsPane_Enter_EmitsTracksFetchRequest(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	// Cursor is at row 0 (first playlist: pl1)
	_, debounceCmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, debounceCmd, "Enter should return a debounce cmd")

	// Execute debounce cmd → get playlistDebounceMsg
	debounceMsg := debounceCmd()
	dm, ok := debounceMsg.(playlistDebounceMsg)
	require.True(t, ok, "debounce cmd should produce playlistDebounceMsg, got %T", debounceMsg)

	// Feed playlistDebounceMsg back → should emit FetchPlaylistTracksRequestMsg
	_, fetchCmd := pane.Update(dm)
	require.NotNil(t, fetchCmd, "debounce resolution should emit FetchPlaylistTracksRequestMsg cmd")

	fetchMsg := fetchCmd()
	req, ok := fetchMsg.(FetchPlaylistTracksRequestMsg)
	require.True(t, ok, "debounce resolution should produce FetchPlaylistTracksRequestMsg, got %T", fetchMsg)
	assert.Equal(t, "pl1", req.PlaylistID)
	assert.Equal(t, 0, req.Offset)
}

// TestPlaylistsPane_Esc_ReturnsToListView verifies Esc exits track sub-view
// and emits PlaylistTrackViewClosedMsg.
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
	require.NotNil(t, cmd, "Esc should emit PlaylistTrackViewClosedMsg")
	msg := cmd()
	_, isClosed := msg.(PlaylistTrackViewClosedMsg)
	assert.True(t, isClosed, "Esc should produce PlaylistTrackViewClosedMsg, got %T", msg)
}

// TestPlaylistsPane_N_IsNoOp verifies 'n' is a no-op after stub removal (story 120).
func TestPlaylistsPane_N_IsNoOp(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.Nil(t, cmd, "'n' handler was removed; should return nil cmd")
}

// TestPlaylistsPane_R_IsNoOp verifies 'r' is a no-op after stub removal (story 120).
// Note: 'r' is also a global playback key (cycle repeat) so it would never reach
// the pane in practice anyway.
func TestPlaylistsPane_R_IsNoOp(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	assert.Nil(t, cmd, "'r' handler was removed; should return nil cmd")
}

// TestPlaylistsPane_X_IsNoOpInStory106 verifies 'x' in track view is a no-op
// in story 106 (management operations are out of scope and remain non-functional).
func TestPlaylistsPane_X_IsNoOpInStory106(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	// Switch to track sub-view with loaded tracks
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedName = "LoFi"
	pane.loadedTracks = []domain.Track{
		{ID: "t1", Name: "Snowman", URI: "spotify:track:t1", Artists: []domain.Artist{{Name: "Sia"}}},
	}
	pane.refreshTrackRows()

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Nil(t, cmd, "'x' is a no-op in story 106 (management operations out of scope)")
}

// TestPlaylistsPane_ShiftUp_IsNoOp verifies Shift+Up in track view is a no-op
// after removal of the unimplemented reorder handler (story 120).
// Most terminals don't deliver xterm shift-arrow sequences reliably anyway.
func TestPlaylistsPane_ShiftUp_IsNoOp(t *testing.T) {
	s := state.New()
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi", URI: "spotify:playlist:pl1", TrackCount: 2},
	})
	tracks := []domain.Track{
		{ID: "t1", Name: "Snowman", URI: "spotify:track:t1"},
		{ID: "t2", Name: "Coffee", URI: "spotify:track:t2"},
	}
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedName = "LoFi"
	pane.loadedTracks = tracks
	pane.refreshTrackRows()
	pane.trackTable.SetFocused(true)

	// Move cursor to row 1, then press Shift+Up — should be a no-op now.
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) //nolint:errcheck
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	assert.Nil(t, cmd, "Shift+Up handler was removed; should return nil cmd")
}

// TestPlaylistsPane_ShiftDown_IsNoOp verifies Shift+Down in track view is a no-op
// after removal of the unimplemented reorder handler (story 120).
func TestPlaylistsPane_ShiftDown_IsNoOp(t *testing.T) {
	s := state.New()
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "pl1", Name: "LoFi", URI: "spotify:playlist:pl1", TrackCount: 2},
	})
	tracks := []domain.Track{
		{ID: "t1", Name: "Snowman", URI: "spotify:track:t1"},
		{ID: "t2", Name: "Coffee", URI: "spotify:track:t2"},
	}
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, true)
	pane.SetSize(80, 20)

	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedName = "LoFi"
	pane.loadedTracks = tracks
	pane.refreshTrackRows()

	// Cursor at 0, press Shift+Down — should be a no-op now.
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	assert.Nil(t, cmd, "Shift+Down handler was removed; should return nil cmd")
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

// TestPlaylistsPane_PlaylistTracksLoadedMsg_ShowsTracks verifies track sub-view refresh
// with pane-owned data (story 106 architecture: no store write).
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

	// Deliver PlaylistTracksLoadedMsg — pane owns the data (not written to store)
	tracks := []domain.Track{
		{ID: "t1", Name: "Snowman", URI: "spotify:track:t1", Artists: []domain.Artist{{Name: "Sia"}}},
	}
	pane.Update(PlaylistTracksLoadedMsg{PlaylistID: "pl1", Tracks: tracks, Total: 1, Offset: 0}) //nolint:errcheck

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

// ── Story 106: Playlist full functionality ────────────────────────────────────

// TestPlaylistsPane_Enter_EmitsDebounceNotDirectRequest verifies that Enter emits
// a debounce cmd (not FetchPlaylistTracksRequestMsg directly).
func TestPlaylistsPane_Enter_EmitsDebounceNotDirectRequest(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should return a command")

	// The command should be a debounce tick, not FetchPlaylistTracksRequestMsg directly.
	// Execute it — it returns a playlistDebounceMsg (internal) not a FetchPlaylistTracksRequestMsg.
	msg := cmd()
	_, isFetchRequest := msg.(FetchPlaylistTracksRequestMsg)
	assert.False(t, isFetchRequest, "Enter should not directly emit FetchPlaylistTracksRequestMsg — it goes through a debounce")
}

// TestPlaylistsPane_DebounceResolution_EmitsFetchRequest verifies that after the
// debounce tick fires with matching intent, FetchPlaylistTracksRequestMsg is emitted.
func TestPlaylistsPane_DebounceResolution_EmitsFetchRequest(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	// Press Enter → get debounce cmd
	_, debounceCmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, debounceCmd)

	// Fire the debounce tick
	debounceMsg := debounceCmd()
	dm, ok := debounceMsg.(playlistDebounceMsg)
	require.True(t, ok, "debounce cmd should produce playlistDebounceMsg, got %T", debounceMsg)

	// Feed debounce msg to pane → should emit FetchPlaylistTracksRequestMsg
	_, fetchCmd := pane.Update(dm)
	require.NotNil(t, fetchCmd, "debounce resolution should emit FetchPlaylistTracksRequestMsg cmd")

	fetchMsg := fetchCmd()
	req, ok := fetchMsg.(FetchPlaylistTracksRequestMsg)
	require.True(t, ok, "debounce resolution should produce FetchPlaylistTracksRequestMsg, got %T", fetchMsg)
	assert.Equal(t, "pl1", req.PlaylistID)
	assert.Equal(t, 0, req.Offset, "initial fetch must use offset=0")
}

// TestPlaylistsPane_StaleDebounce_Discarded verifies that a stale debounce tick is
// discarded when the user has switched to a different playlist.
func TestPlaylistsPane_StaleDebounce_Discarded(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	// Press Enter on pl1 → get stale debounce snapshot for pl1
	_, debounceCmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, debounceCmd)

	// Before tick fires, simulate user escaping and re-entering a different playlist
	// by manually overriding the playlistIntent to match pl2.
	pane.playlistIntent = playlistDebounceIntent{playlistID: "pl2"}

	// Now fire the original tick (for pl1) — it should be discarded
	debounceMsg := debounceCmd()
	dm, ok := debounceMsg.(playlistDebounceMsg)
	require.True(t, ok)

	_, fetchCmd := pane.Update(dm)
	assert.Nil(t, fetchCmd, "stale debounce tick (pl1) should be discarded when intent is pl2")
}

// TestPlaylistsPane_DebounceGuard_TracksFetchingTrue verifies that a debounce tick
// is discarded when a fetch is already in-flight.
func TestPlaylistsPane_DebounceGuard_TracksFetchingTrue(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	_, debounceCmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, debounceCmd)

	// Simulate a fetch already in-flight
	pane.tracksFetching = true

	debounceMsg := debounceCmd()
	dm, ok := debounceMsg.(playlistDebounceMsg)
	require.True(t, ok)

	_, fetchCmd := pane.Update(dm)
	assert.Nil(t, fetchCmd, "debounce tick must be discarded when tracksFetching=true")
}

// TestPlaylistsPane_Enter_SetsSelectedURI verifies that selectedURI is set on Enter.
func TestPlaylistsPane_Enter_SetsSelectedURI(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	_, _ = pane.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, "spotify:playlist:pl1", pane.selectedURI, "selectedURI must be set to playlist URI on Enter")
}

// TestPlaylistsPane_Enter_ResetsSubViewState verifies that entering a playlist resets
// loadedTracks, trackOffset, trackTotal, hasMoreTracks, tracksFetching.
func TestPlaylistsPane_Enter_ResetsSubViewState(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)

	// Pre-populate state from a prior playlist
	pane.loadedTracks = []domain.Track{{ID: "old"}}
	pane.trackOffset = 5
	pane.trackTotal = 10
	pane.hasMoreTracks = true
	pane.tracksFetching = true

	_, _ = pane.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Nil(t, pane.loadedTracks, "loadedTracks must be reset on Enter")
	assert.Equal(t, 0, pane.trackOffset, "trackOffset must be reset on Enter")
	assert.Equal(t, 0, pane.trackTotal, "trackTotal must be reset on Enter")
	assert.False(t, pane.hasMoreTracks, "hasMoreTracks must be reset on Enter")
	assert.False(t, pane.tracksFetching, "tracksFetching must be reset on Enter")
}

// TestPlaylistsPane_TracksLoadedMsg_InitialPage verifies Offset=0 replaces loadedTracks.
func TestPlaylistsPane_TracksLoadedMsg_InitialPage(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.resizeTable()
	pane.trackTable.SetFocused(true)

	tracks := []domain.Track{
		{ID: "t1", Name: "Track One", URI: "spotify:track:t1", Artists: []domain.Artist{{Name: "A"}}},
		{ID: "t2", Name: "Track Two", URI: "spotify:track:t2", Artists: []domain.Artist{{Name: "B"}}},
	}
	msg := PlaylistTracksLoadedMsg{PlaylistID: "pl1", Tracks: tracks, Total: 5, HasNext: true, Offset: 0}
	pane.Update(msg) //nolint:errcheck

	assert.Equal(t, 2, len(pane.loadedTracks), "initial page must replace loadedTracks")
	assert.Equal(t, 2, pane.trackOffset)
	assert.Equal(t, 5, pane.trackTotal)
	assert.True(t, pane.hasMoreTracks)
	assert.False(t, pane.tracksFetching)
}

// TestPlaylistsPane_TracksLoadedMsg_NextPage verifies Offset>0 appends to loadedTracks.
func TestPlaylistsPane_TracksLoadedMsg_NextPage(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.loadedTracks = []domain.Track{
		{ID: "t1", Name: "Track One", URI: "spotify:track:t1", Artists: []domain.Artist{{Name: "A"}}},
	}
	pane.trackOffset = 1

	tracks := []domain.Track{
		{ID: "t2", Name: "Track Two", URI: "spotify:track:t2", Artists: []domain.Artist{{Name: "B"}}},
	}
	msg := PlaylistTracksLoadedMsg{PlaylistID: "pl1", Tracks: tracks, Total: 2, HasNext: false, Offset: 1}
	pane.Update(msg) //nolint:errcheck

	assert.Equal(t, 2, len(pane.loadedTracks), "next page must be appended to loadedTracks")
	assert.Equal(t, 2, pane.trackOffset)
	assert.False(t, pane.hasMoreTracks, "HasNext=false must set hasMoreTracks=false")
}

// TestPlaylistsPane_TracksLoadedMsg_WrongPlaylistID verifies stale msg is ignored.
func TestPlaylistsPane_TracksLoadedMsg_WrongPlaylistID(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	pane.inTrackView = true
	pane.selectedID = "pl1"

	msg := PlaylistTracksLoadedMsg{PlaylistID: "other-pl", Tracks: []domain.Track{{ID: "t1"}}, Total: 1, Offset: 0}
	pane.Update(msg) //nolint:errcheck

	assert.Nil(t, pane.loadedTracks, "wrong PlaylistID must be ignored (pane stays at nil loadedTracks)")
}

// TestPlaylistsPane_TracksLoadedMsg_ErrorPath_ClearsTracksFetching verifies that
// a PlaylistTracksLoadedMsg with a non-nil Err clears tracksFetching so the pane
// is not permanently stuck in a loading state.
func TestPlaylistsPane_TracksLoadedMsg_ErrorPath_ClearsTracksFetching(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.tracksFetching = true // simulate an in-flight fetch

	msg := PlaylistTracksLoadedMsg{PlaylistID: "pl1", Err: fmt.Errorf("network error")}
	pane.Update(msg) //nolint:errcheck

	assert.False(t, pane.tracksFetching, "error response must clear tracksFetching so the pane is not stuck")
}

// TestPlaylistsPane_Enter_TrackView_EmitsPlayContextMsg verifies Enter on a track emits
// PlayContextMsg with correct ContextURI and OffsetURI.
func TestPlaylistsPane_Enter_TrackView_EmitsPlayContextMsg(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedURI = "spotify:playlist:pl1"
	pane.loadedTracks = []domain.Track{
		{ID: "t1", Name: "Track One", URI: "spotify:track:t1", Artists: []domain.Artist{{Name: "A"}}},
		{ID: "t2", Name: "Track Two", URI: "spotify:track:t2", Artists: []domain.Artist{{Name: "B"}}},
	}
	pane.refreshTrackRows()
	pane.trackTable.SetFocused(true)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter on track should return a command")

	msg := cmd()
	pcMsg, ok := msg.(PlayContextMsg)
	require.True(t, ok, "Enter on track should emit PlayContextMsg, got %T", msg)
	assert.Equal(t, "spotify:playlist:pl1", pcMsg.ContextURI)
	assert.Equal(t, "spotify:track:t1", pcMsg.OffsetURI)
}

// TestPlaylistsPane_Enter_TrackView_EmptyLoadedTracks_NoOp verifies no crash when
// Enter is pressed in track view with no tracks loaded.
func TestPlaylistsPane_Enter_TrackView_EmptyLoadedTracks_NoOp(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedURI = "spotify:playlist:pl1"
	pane.loadedTracks = nil

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "Enter with empty loadedTracks must be a no-op")
}

// TestPlaylistsPane_Esc_TrackView_EmitsClosedMsg verifies Esc emits PlaylistTrackViewClosedMsg.
func TestPlaylistsPane_Esc_TrackView_EmitsClosedMsg(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedName = "LoFi"

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "Esc should emit PlaylistTrackViewClosedMsg")

	msg := cmd()
	_, ok := msg.(PlaylistTrackViewClosedMsg)
	assert.True(t, ok, "Esc should produce PlaylistTrackViewClosedMsg, got %T", msg)
}

// TestPlaylistsPane_CheckPrefetch_FiresWhenNear verifies prefetch fires when cursor
// is within 10 rows of the end.
func TestPlaylistsPane_CheckPrefetch_FiresWhenNear(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 30)
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.selectedURI = "spotify:playlist:pl1"
	pane.hasMoreTracks = true
	pane.tracksFetching = false
	pane.trackOffset = 15

	// 15 loaded tracks, cursor at index 5 (which is >= 15-10=5) → prefetch fires
	tracks := make([]domain.Track, 15)
	for i := range tracks {
		tracks[i] = domain.Track{
			ID: fmt.Sprintf("t%d", i), Name: fmt.Sprintf("Track %d", i),
			URI: fmt.Sprintf("spotify:track:t%d", i), Artists: []domain.Artist{{Name: "A"}},
		}
	}
	pane.loadedTracks = tracks
	pane.refreshTrackRows()
	pane.trackTable.SetFocused(true)

	// Navigate to index 5 (within 10 of end)
	for range 5 {
		pane.trackTable.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) //nolint:errcheck
	}

	// Now a navigation key should trigger prefetch
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	// cmd could be a batch; check it's non-nil since prefetch should fire
	assert.NotNil(t, cmd, "navigating near end of loaded tracks should trigger prefetch cmd")
	assert.True(t, pane.tracksFetching, "tracksFetching should be true after prefetch fires")
}

// TestPlaylistsPane_CheckPrefetch_DoesNotFireWhenFetching verifies prefetch is
// blocked when tracksFetching=true.
func TestPlaylistsPane_CheckPrefetch_DoesNotFireWhenFetching(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 30)
	pane.inTrackView = true
	pane.selectedID = "pl1"
	pane.hasMoreTracks = true
	pane.tracksFetching = true // already fetching
	pane.trackOffset = 15

	tracks := make([]domain.Track, 15)
	for i := range tracks {
		tracks[i] = domain.Track{
			ID: fmt.Sprintf("t%d", i), Name: fmt.Sprintf("Track %d", i),
			URI: fmt.Sprintf("spotify:track:t%d", i), Artists: []domain.Artist{{Name: "A"}},
		}
	}
	pane.loadedTracks = tracks
	pane.refreshTrackRows()
	pane.trackTable.SetFocused(true)

	cmd := pane.checkPrefetch()
	assert.Nil(t, cmd, "checkPrefetch must return nil when tracksFetching=true")
}

// TestPlaylistsPane_RefreshTrackRows_ReadsLoadedTracks verifies refreshTrackRows
// reads from pane.loadedTracks not from the store.
func TestPlaylistsPane_RefreshTrackRows_ReadsLoadedTracks(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	pane.inTrackView = true

	tracks := []domain.Track{
		{ID: "t1", Name: "Test Track", URI: "spotify:track:t1", Artists: []domain.Artist{{Name: "Artist"}}, DurationMs: 180000},
	}
	pane.loadedTracks = tracks
	pane.refreshTrackRows()

	output := pane.View()
	assert.Contains(t, output, "Test Track", "refreshTrackRows must read from loadedTracks")
}

// TestPlaylistsPane_Title_InTrackView_ShowsTrackTotal verifies that Title() shows
// p.trackTotal, not store data.
func TestPlaylistsPane_Title_InTrackView_ShowsTrackTotal(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(true)
	pane.SetSize(80, 20)
	pane.inTrackView = true
	pane.selectedName = "LoFi"
	pane.trackTotal = 42

	title := pane.Title()
	assert.Equal(t, "Playlists ── LoFi (42 tracks)", title)
}

// TestPlaylistsPane_SetFocused_TrackView_PaneFocused verifies that SetFocused(true)
// propagates focused=true to the pane when in track view.
func TestPlaylistsPane_SetFocused_TrackView_PaneFocused(t *testing.T) {
	pane := newTestPlaylistsPaneWithData(false)
	pane.SetSize(80, 20)
	pane.inTrackView = true

	pane.SetFocused(true)
	assert.True(t, pane.IsFocused(), "pane must be focused after SetFocused(true)")

	pane.SetFocused(false)
	assert.False(t, pane.IsFocused(), "pane must be unfocused after SetFocused(false)")
}

// ── Story 71 Task 4: column color tokens ─────────────────────────────────────

// TestPlaylistsPane_UsesColumnColors verifies that PlaylistsPane column definitions
// (both list view and track sub-view) use the new column color tokens.
func TestPlaylistsPane_UsesColumnColors(t *testing.T) {
	th := theme.Load("black")
	p := NewPlaylistsPane(state.New(), th, false)

	// List view: # → ColumnIndex, access (blank header) → ColumnSecondary, Name → ColumnPrimary, Tracks → ColumnTertiary
	listCols := p.table.Columns()
	require.Len(t, listCols, 4, "PlaylistsPane list table should have 4 columns")
	assert.Equal(t, th.ColumnIndex(), listCols[0].Color, "# column should use ColumnIndex()")
	assert.Equal(t, th.ColumnSecondary(), listCols[1].Color, "access column should use ColumnSecondary()")
	assert.Equal(t, th.ColumnPrimary(), listCols[2].Color, "Name column should use ColumnPrimary()")
	assert.Equal(t, th.ColumnTertiary(), listCols[3].Color, "Tracks column should use ColumnTertiary()")

	// Track sub-view: # → ColumnIndex, Track → ColumnPrimary, Artist → ColumnSecondary, Duration → ColumnTertiary
	trackCols := p.trackTable.Columns()
	require.Len(t, trackCols, 4, "PlaylistsPane track table should have 4 columns")
	assert.Equal(t, th.ColumnIndex(), trackCols[0].Color, "# column should use ColumnIndex()")
	assert.Equal(t, th.ColumnPrimary(), trackCols[1].Color, "Track column should use ColumnPrimary()")
	assert.Equal(t, th.ColumnSecondary(), trackCols[2].Color, "Artist column should use ColumnSecondary()")
	assert.Equal(t, th.ColumnTertiary(), trackCols[3].Color, "Duration column should use ColumnTertiary()")
}

// ── Story 120: dead pane action removal ──────────────────────────────────────

// TestPlaylistsPane_Actions_ListView_NoNOrR verifies that Actions() in list view
// iterates and contains no n or r entries, and still has f.
func TestPlaylistsPane_Actions_ListView_NoNOrR(t *testing.T) {
	pane := newTestPlaylistsPane(true)
	actions := pane.Actions()
	for _, a := range actions {
		assert.NotEqual(t, "n", a.Key, "Actions() must not include 'n'")
		assert.NotEqual(t, "r", a.Key, "Actions() must not include 'r'")
	}
	keys := make([]string, len(actions))
	for i, a := range actions {
		keys[i] = a.Key
	}
	assert.Contains(t, keys, "f", "Actions() must still include 'f' (filter)")
}

// ── Story 158: LockedRow for Spotify-owned playlists ─────────────────────────

// TestPlaylistsPane_SpotifyOwnedRow_HasLockedGlyph verifies that a playlist with
// Owner.ID == "spotify" renders with the locked glyph (◌ in unicode mode) in the
// dedicated access column, and that the name column contains only the plain name.
func TestPlaylistsPane_SpotifyOwnedRow_HasLockedGlyph(t *testing.T) {
	s := state.New()
	s.SetPlaylists([]domain.SimplePlaylist{
		{
			ID:         "sp1",
			Name:       "Today's Top Hits",
			URI:        "spotify:playlist:sp1",
			TrackCount: 50,
			Owner:      domain.SimplePlaylistOwner{ID: "spotify"},
		},
		{
			ID:         "pl2",
			Name:       "My Playlist",
			URI:        "spotify:playlist:pl2",
			TrackCount: 10,
			Owner:      domain.SimplePlaylistOwner{ID: "user123"},
		},
	})
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, false)
	pane.SetSize(80, 20)

	// In unicode mode, ◌ is the locked glyph.
	lockedGlyph := uikit.GlyphFor(uikit.GlyphLocked, uikit.GlyphUnicode)

	rows := pane.table.Rows()
	require.Len(t, rows, 2, "expected 2 rows")

	spotifyRow := rows[0]
	assert.True(t, strings.HasPrefix(spotifyRow["access"], lockedGlyph),
		"Spotify-owned playlist access column must start with locked glyph %q, got %q",
		lockedGlyph, spotifyRow["access"])
	assert.Equal(t, "Today's Top Hits", spotifyRow["name"],
		"name column must contain only the plain playlist name, no glyph prefix")

	// The user-owned playlist must NOT have the locked glyph in access.
	userRow := rows[1]
	assert.False(t, strings.HasPrefix(userRow["access"], lockedGlyph),
		"user-owned playlist access must not be locked glyph, got %q", userRow["access"])
	assert.Equal(t, "My Playlist", userRow["name"])
}

// TestPlaylistsPane_AccessColumn_GlyphsPerOwnerType verifies that the dedicated
// access column shows the correct glyph for each ownership type, and that the
// name column is clean (no glyph prefix, no "~ " prefix).
func TestPlaylistsPane_AccessColumn_GlyphsPerOwnerType(t *testing.T) {
	const userID = "user123"
	s := state.New()
	s.SetUserProfile(domain.UserProfile{ID: userID})
	s.SetPlaylists([]domain.SimplePlaylist{
		{
			ID:         "owned",
			Name:       "My Playlist",
			URI:        "spotify:playlist:owned",
			TrackCount: 10,
			Owner:      domain.SimplePlaylistOwner{ID: userID},
		},
		{
			ID:         "followed",
			Name:       "Someone Else's Playlist",
			URI:        "spotify:playlist:followed",
			TrackCount: 20,
			Owner:      domain.SimplePlaylistOwner{ID: "other_user"},
		},
		{
			ID:         "spotify_curated",
			Name:       "Today's Top Hits",
			URI:        "spotify:playlist:spotify_curated",
			TrackCount: 50,
			Owner:      domain.SimplePlaylistOwner{ID: "spotify"},
		},
	})
	th := theme.Load("black")
	pane := NewPlaylistsPane(s, th, false)
	pane.SetSize(80, 20)

	activeGlyph := uikit.GlyphFor(uikit.GlyphActive, uikit.GlyphUnicode)   // ◉
	availGlyph := uikit.GlyphFor(uikit.GlyphAvailable, uikit.GlyphUnicode) // ○
	lockedGlyph := uikit.GlyphFor(uikit.GlyphLocked, uikit.GlyphUnicode)   // ◌

	rows := pane.table.Rows()
	require.Len(t, rows, 3, "expected 3 rows")

	// Row 0: user-owned → ◉
	assert.Equal(t, activeGlyph, rows[0]["access"],
		"user-owned row access must be GlyphActive (%q)", activeGlyph)
	assert.Equal(t, "My Playlist", rows[0]["name"], "user-owned name must be plain")

	// Row 1: followed (non-owned, non-Spotify) → ○
	assert.Equal(t, availGlyph, rows[1]["access"],
		"followed row access must be GlyphAvailable (%q)", availGlyph)
	assert.Equal(t, "Someone Else's Playlist", rows[1]["name"], "followed name must be plain")

	// Row 2: Spotify-curated → ◌
	assert.Equal(t, lockedGlyph, rows[2]["access"],
		"Spotify-curated row access must be GlyphLocked (%q)", lockedGlyph)
	assert.Equal(t, "Today's Top Hits", rows[2]["name"], "Spotify-curated name must be plain")

	// Verify no name cell starts with a glyph character.
	glyphs := []string{activeGlyph, availGlyph, lockedGlyph, "~ "}
	for i, row := range rows {
		for _, g := range glyphs {
			assert.False(t, strings.HasPrefix(row["name"], g),
				"row %d name must not start with glyph/prefix %q, got %q", i, g, row["name"])
		}
	}
}

// TestPlaylistsPane_AccessColumn_ASCIIFallbacks verifies that ASCII mode glyphs
// are used when uikit is in ASCII mode.
func TestPlaylistsPane_AccessColumn_ASCIIFallbacks(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	const userID = "user123"
	s := state.New()
	s.SetUserProfile(domain.UserProfile{ID: userID})
	s.SetPlaylists([]domain.SimplePlaylist{
		{ID: "owned", Name: "My Playlist", TrackCount: 5, Owner: domain.SimplePlaylistOwner{ID: userID}},
		{ID: "followed", Name: "Other Playlist", TrackCount: 5, Owner: domain.SimplePlaylistOwner{ID: "other"}},
		{ID: "sp", Name: "Top Hits", TrackCount: 5, Owner: domain.SimplePlaylistOwner{ID: "spotify"}},
	})
	pane := NewPlaylistsPane(s, theme.Load("black"), false)
	pane.SetSize(80, 20)

	rows := pane.table.Rows()
	require.Len(t, rows, 3)
	assert.Equal(t, uikit.GlyphFor(uikit.GlyphActive, uikit.GlyphASCII), rows[0]["access"], "user-owned ASCII glyph")
	assert.Equal(t, uikit.GlyphFor(uikit.GlyphAvailable, uikit.GlyphASCII), rows[1]["access"], "followed ASCII glyph")
	assert.Equal(t, uikit.GlyphFor(uikit.GlyphLocked, uikit.GlyphASCII), rows[2]["access"], "Spotify-curated ASCII glyph")
}
