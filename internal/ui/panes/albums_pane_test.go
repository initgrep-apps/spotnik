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

// Compile-time check: AlbumsPane implements layout.Pane.
var _ layout.Pane = &AlbumsPane{}

// newTestAlbumsPane creates an AlbumsPane with a fresh store and black theme.
func newTestAlbumsPane(focused bool) *AlbumsPane {
	s := state.New()
	th := theme.Load("black")
	return NewAlbumsPane(s, th, focused)
}

// newTestAlbumsPaneWithData creates an AlbumsPane pre-loaded with albums.
func newTestAlbumsPaneWithData(focused bool) *AlbumsPane {
	s := state.New()
	s.SetSavedAlbums([]domain.SavedAlbum{
		{
			Album: domain.FullAlbum{
				ID:          "al1",
				Name:        "After Hours",
				URI:         "spotify:album:al1",
				ReleaseDate: "2020-03-20",
				Artists:     []domain.Artist{{Name: "The Weeknd"}},
			},
		},
		{
			Album: domain.FullAlbum{
				ID:          "al2",
				Name:        "OK Computer",
				URI:         "spotify:album:al2",
				ReleaseDate: "1997-05-21",
				Artists:     []domain.Artist{{Name: "Radiohead"}},
			},
		},
		{
			Album: domain.FullAlbum{
				ID:          "al3",
				Name:        "In Rainbows",
				URI:         "spotify:album:al3",
				ReleaseDate: "2007-10-10",
				Artists:     []domain.Artist{{Name: "Radiohead"}},
			},
		},
	})
	th := theme.Load("black")
	pane := NewAlbumsPane(s, th, focused)
	return pane
}

// TestAlbumsPane_ImplementsLayoutPane verifies the interface is satisfied.
func TestAlbumsPane_ImplementsLayoutPane(t *testing.T) {
	pane := newTestAlbumsPane(false)
	assert.NotNil(t, pane)
}

// TestAlbumsPane_ID verifies the pane ID.
func TestAlbumsPane_ID(t *testing.T) {
	pane := newTestAlbumsPane(false)
	assert.Equal(t, layout.PaneAlbums, pane.ID())
}

// TestAlbumsPane_Title returns "Albums".
func TestAlbumsPane_Title(t *testing.T) {
	pane := newTestAlbumsPane(false)
	assert.Equal(t, "Albums", pane.Title())
}

// TestAlbumsPane_ToggleKey returns 4.
func TestAlbumsPane_ToggleKey(t *testing.T) {
	pane := newTestAlbumsPane(false)
	assert.Equal(t, 4, pane.ToggleKey())
}

// TestAlbumsPane_Actions_Default returns filter action by default.
func TestAlbumsPane_Actions_Default(t *testing.T) {
	pane := newTestAlbumsPane(true)
	actions := pane.Actions()
	keys := make([]string, len(actions))
	for i, a := range actions {
		keys[i] = a.Key
	}
	assert.Contains(t, keys, "f", "should have filter action")
}

// TestAlbumsPane_Actions_FilterActive returns close action when filter is active.
func TestAlbumsPane_Actions_FilterActive(t *testing.T) {
	pane := newTestAlbumsPane(true)
	pane.SetSize(80, 20)
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "Esc", actions[0].Key)
}

// TestAlbumsPane_View_Empty verifies clean render on empty data.
func TestAlbumsPane_View_Empty(t *testing.T) {
	pane := newTestAlbumsPane(true)
	pane.SetSize(80, 20)
	output := pane.View()
	assert.NotEmpty(t, output, "should return non-empty string for empty albums")
}

// TestAlbumsPane_View_ShowsAlbums verifies album names appear.
func TestAlbumsPane_View_ShowsAlbums(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)
	output := pane.View()
	assert.Contains(t, output, "After Hours", "first album should appear")
	assert.Contains(t, output, "Radiohead", "artist should appear")
}

// TestAlbumsPane_View_ShowsYear verifies release year appears in the table.
func TestAlbumsPane_View_ShowsYear(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)
	output := pane.View()
	assert.Contains(t, output, "2020", "release year should appear")
	assert.Contains(t, output, "1997", "release year should appear")
}

// TestAlbumsPane_Enter_OpensTrackSubView verifies Enter on an album opens the track sub-view
// instead of immediately playing (story 107 drill-down behaviour).
func TestAlbumsPane_Enter_OpensTrackSubView(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)

	model, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*AlbumsPane)

	// Enter now triggers a debounce rather than directly playing.
	require.NotNil(t, cmd, "Enter should return a debounce cmd")
	assert.True(t, updated.inTrackView, "Enter must open the track sub-view")
	assert.Equal(t, "al1", updated.selectedID, "selectedID must be set from the selected album")
}

// TestAlbumsPane_Filter_ByAlbumName verifies filter narrows results by album name.
func TestAlbumsPane_Filter_ByAlbumName(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)

	// Activate filter and type "after"
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "after" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	output := pane.View()
	assert.Contains(t, output, "After Hours", "filter should show matching album")
}

// TestAlbumsPane_Filter_ByArtistName verifies filter matches artist name.
func TestAlbumsPane_Filter_ByArtistName(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)

	// Activate filter and type "weeknd"
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "weeknd" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	output := pane.View()
	assert.Contains(t, output, "After Hours", "filter by artist should show matching album")
}

// TestAlbumsPane_IsFocused verifies SetFocused/IsFocused.
func TestAlbumsPane_IsFocused(t *testing.T) {
	pane := newTestAlbumsPane(false)
	assert.False(t, pane.IsFocused())
	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
}

// TestAlbumsPane_IgnoresInputWhenUnfocused verifies pane ignores input when not focused.
func TestAlbumsPane_IgnoresInputWhenUnfocused(t *testing.T) {
	pane := newTestAlbumsPaneWithData(false)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "unfocused pane should not emit commands")
}

// TestAlbumsPane_AlbumsLoadedMsg_RefreshesTable verifies data-load integration.
func TestAlbumsPane_AlbumsLoadedMsg_RefreshesTable(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewAlbumsPane(s, th, true)
	pane.SetSize(80, 20)

	s.SetSavedAlbums([]domain.SavedAlbum{
		{
			Album: domain.FullAlbum{
				ID:      "al1",
				Name:    "Discovery",
				URI:     "spotify:album:al1",
				Artists: []domain.Artist{{Name: "Daft Punk"}},
			},
		},
	})
	pane.Update(AlbumsLoadedMsg{}) //nolint:errcheck

	output := pane.View()
	assert.Contains(t, output, "Discovery", "pane should show newly loaded album after AlbumsLoadedMsg")
}

// TestAlbumsPane_LargeAlbumList verifies no panic with many albums.
func TestAlbumsPane_LargeAlbumList(t *testing.T) {
	s := state.New()
	albums := make([]domain.SavedAlbum, 100)
	for i := range albums {
		albums[i] = domain.SavedAlbum{
			Album: domain.FullAlbum{
				ID:          fmt.Sprintf("al%d", i),
				Name:        fmt.Sprintf("Album %d", i+1),
				URI:         fmt.Sprintf("spotify:album:al%d", i),
				ReleaseDate: "2020-01-01",
				Artists:     []domain.Artist{{Name: "Artist"}},
			},
		}
	}
	s.SetSavedAlbums(albums)
	th := theme.Load("black")
	pane := NewAlbumsPane(s, th, true)
	pane.SetSize(80, 20)

	output := pane.View()
	assert.NotEmpty(t, output, "large album list should render without panic")
}

// TestAlbumsPane_RefreshRows_UpdatesTable verifies the exported RefreshRows method.
func TestAlbumsPane_RefreshRows_UpdatesTable(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewAlbumsPane(s, th, true)
	pane.SetSize(80, 20)

	s.SetSavedAlbums([]domain.SavedAlbum{
		{
			Album: domain.FullAlbum{
				ID:      "al1",
				Name:    "NewAlbum",
				URI:     "spotify:album:al1",
				Artists: []domain.Artist{{Name: "Artist"}},
			},
		},
	})
	pane.RefreshRows()

	output := pane.View()
	assert.Contains(t, output, "NewAlbum", "RefreshRows should update the view")
}

// TestAlbumsPane_Navigation_JK verifies j/k move cursor.
func TestAlbumsPane_Navigation_JK(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)

	initialCursor := pane.Cursor()
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) //nolint:errcheck
	assert.Greater(t, pane.Cursor(), initialCursor, "cursor should move down after j")
}

// ── Story 71 Task 2: column color tokens ─────────────────────────────────────

// TestAlbumsPane_UsesColumnColors verifies that AlbumsPane column definitions
// use the new ColumnIndex/ColumnPrimary/ColumnSecondary/ColumnTertiary tokens.
func TestAlbumsPane_UsesColumnColors(t *testing.T) {
	th := theme.Load("black")
	a := NewAlbumsPane(state.New(), th, false)
	cols := a.table.Columns()
	require.Len(t, cols, 4, "AlbumsPane should have 4 columns")

	assert.Equal(t, th.ColumnIndex(), cols[0].Color, "# column should use ColumnIndex()")
	assert.Equal(t, th.ColumnPrimary(), cols[1].Color, "Name column should use ColumnPrimary()")
	assert.Equal(t, th.ColumnSecondary(), cols[2].Color, "Artist column should use ColumnSecondary()")
	assert.Equal(t, th.ColumnTertiary(), cols[3].Color, "Year column should use ColumnTertiary()")
}

// ── Story 107: Album drill-down + track play ─────────────────────────────────

// TestAlbumsPane_EnterOnAlbum_SetsInTrackView verifies that pressing Enter on an
// album sets inTrackView=true and emits a debounce tick cmd.
func TestAlbumsPane_EnterOnAlbum_SetsInTrackView(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	model, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*AlbumsPane)

	assert.True(t, updated.inTrackView, "Enter on album must set inTrackView=true")
	assert.NotNil(t, cmd, "Enter on album must emit a debounce tick cmd")
	assert.Equal(t, "al1", updated.selectedID, "selectedID must be set to album ID")
}

// TestAlbumsPane_AlbumDebounceMsg_MatchingIntent_EmitsFetchRequest verifies that an
// albumDebounceMsg with matching intent emits FetchAlbumTracksRequestMsg.
func TestAlbumsPane_AlbumDebounceMsg_MatchingIntent_EmitsFetchRequest(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	// Open track view to set albumIntent.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)

	// Simulate debounce tick with matching intent.
	dMsg := albumDebounceMsg{intent: a.albumIntent}
	model2, cmd := a.Update(dMsg)
	a = model2.(*AlbumsPane)

	require.NotNil(t, cmd, "matching albumDebounceMsg must emit a cmd")
	assert.True(t, a.tracksFetching, "tracksFetching must be true after debounce fires")

	// Execute the cmd to get the FetchAlbumTracksRequestMsg.
	msg := cmd()
	fetchMsg, ok := msg.(FetchAlbumTracksRequestMsg)
	require.True(t, ok, "debounce cmd must return FetchAlbumTracksRequestMsg, got %T", msg)
	assert.Equal(t, "al1", fetchMsg.AlbumID)
	assert.Equal(t, 0, fetchMsg.Offset)
}

// TestAlbumsPane_AlbumDebounceMsg_StaleIntent_Discards verifies that an albumDebounceMsg
// with a stale intent (different albumID) is silently discarded.
func TestAlbumsPane_AlbumDebounceMsg_StaleIntent_Discards(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	// Open track view, setting albumIntent to the first album.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)

	// Send a debounce tick for a different album — stale.
	staleMsg := albumDebounceMsg{intent: albumDebounceIntent{albumID: "old-album", offset: 0}}
	_, cmd := a.Update(staleMsg)

	assert.Nil(t, cmd, "stale albumDebounceMsg must be discarded")
}

// TestAlbumsPane_AlbumTracksLoadedMsg_Offset0_ReplacesLoadedTracks verifies that
// AlbumTracksLoadedMsg with Offset=0 replaces loadedTracks.
func TestAlbumsPane_AlbumTracksLoadedMsg_Offset0_ReplacesLoadedTracks(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	// Open track view.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)

	tracks := []domain.Track{
		{ID: "t1", URI: "spotify:track:t1", Name: "So What", DurationMs: 200000,
			Artists: []domain.Artist{{ID: "a1", Name: "Miles Davis"}}},
		{ID: "t2", URI: "spotify:track:t2", Name: "Freddie", DurationMs: 300000,
			Artists: []domain.Artist{{ID: "a1", Name: "Miles Davis"}}},
	}
	msg := AlbumTracksLoadedMsg{AlbumID: "al1", Offset: 0, Tracks: tracks, HasNext: false}
	model2, _ := a.Update(msg)
	updated := model2.(*AlbumsPane)

	require.Len(t, updated.loadedTracks, 2, "Offset=0 must replace loadedTracks")
	assert.Equal(t, "t1", updated.loadedTracks[0].ID)
	assert.False(t, updated.hasMoreTracks, "HasNext=false must set hasMoreTracks=false")
}

// TestAlbumsPane_AlbumTracksLoadedMsg_Offset50_AppendsToLoadedTracks verifies that
// AlbumTracksLoadedMsg with Offset>0 appends to existing loadedTracks.
func TestAlbumsPane_AlbumTracksLoadedMsg_Offset50_AppendsToLoadedTracks(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	// Seed initial 50 tracks.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)

	firstPage := make([]domain.Track, 50)
	for i := range firstPage {
		firstPage[i] = domain.Track{ID: fmt.Sprintf("t%d", i+1), URI: fmt.Sprintf("spotify:track:t%d", i+1), Name: fmt.Sprintf("Track %d", i+1)}
	}
	model2, _ := a.Update(AlbumTracksLoadedMsg{AlbumID: "al1", Offset: 0, Tracks: firstPage, HasNext: true})
	a = model2.(*AlbumsPane)

	// Append a second page.
	secondPage := []domain.Track{
		{ID: "t51", URI: "spotify:track:t51", Name: "Track 51"},
	}
	model3, _ := a.Update(AlbumTracksLoadedMsg{AlbumID: "al1", Offset: 50, Tracks: secondPage, HasNext: false})
	updated := model3.(*AlbumsPane)

	assert.Len(t, updated.loadedTracks, 51, "Offset>0 must append to loadedTracks")
	assert.Equal(t, 51, updated.trackOffset)
	assert.False(t, updated.hasMoreTracks)
}

// TestAlbumsPane_AlbumTracksLoadedMsg_WrongAlbumID_Discards verifies that
// AlbumTracksLoadedMsg with mismatched AlbumID is silently discarded.
func TestAlbumsPane_AlbumTracksLoadedMsg_WrongAlbumID_Discards(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	// Open track view for first album.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)

	// Send AlbumTracksLoadedMsg for a different album.
	msg := AlbumTracksLoadedMsg{AlbumID: "wrong-album", Offset: 0, Tracks: []domain.Track{{ID: "t1"}}}
	model2, _ := a.Update(msg)
	updated := model2.(*AlbumsPane)

	assert.Nil(t, updated.loadedTracks, "stale AlbumTracksLoadedMsg must not update loadedTracks")
}

// TestAlbumsPane_AlbumTracksLoadedMsg_HasNextFalse_StopsPrefetch verifies that
// HasNext=false prevents future prefetch attempts.
func TestAlbumsPane_AlbumTracksLoadedMsg_HasNextFalse_StopsPrefetch(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)

	msg := AlbumTracksLoadedMsg{AlbumID: "al1", Offset: 0, Tracks: []domain.Track{{ID: "t1"}}, HasNext: false}
	model2, _ := a.Update(msg)
	updated := model2.(*AlbumsPane)

	assert.False(t, updated.hasMoreTracks, "HasNext=false must set hasMoreTracks=false")
	assert.False(t, updated.tracksFetching, "tracksFetching must be cleared")
}

// TestAlbumsPane_AlbumTracksLoadedMsg_ErrorPath_ClearsFetchingState verifies that when
// AlbumTracksLoadedMsg carries an error, tracksFetching is cleared and loadedTracks is unchanged.
func TestAlbumsPane_AlbumTracksLoadedMsg_ErrorPath_ClearsFetchingState(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	// Open track view so selectedID matches.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)

	// Manually set tracksFetching=true to simulate an in-flight request.
	a.tracksFetching = true

	// Send an AlbumTracksLoadedMsg with an error.
	someErr := fmt.Errorf("failed to fetch album tracks")
	model2, _ := a.Update(AlbumTracksLoadedMsg{AlbumID: "al1", Err: someErr})
	updated := model2.(*AlbumsPane)

	assert.False(t, updated.tracksFetching, "tracksFetching must be cleared on error")
	assert.Nil(t, updated.loadedTracks, "loadedTracks must remain unchanged on error")
}

// TestAlbumsPane_Esc_EmitsAlbumTrackViewClosedMsg verifies that Esc in track sub-view
// emits AlbumTrackViewClosedMsg and clears loadedTracks.
func TestAlbumsPane_Esc_EmitsAlbumTrackViewClosedMsg(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	// Open track view.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)
	a.loadedTracks = []domain.Track{{ID: "t1", URI: "spotify:track:t1"}}
	a.inTrackView = true

	// Press Esc.
	model2, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model2.(*AlbumsPane)

	assert.False(t, updated.inTrackView, "inTrackView must be false after Esc")
	assert.Nil(t, updated.loadedTracks, "loadedTracks must be cleared on Esc")
	// albumIntent must be cleared so any in-flight debounce tick is discarded.
	assert.Equal(t, albumDebounceIntent{}, updated.albumIntent, "albumIntent must be zeroed on Esc")
	require.NotNil(t, cmd, "Esc must emit a cmd")

	msg := cmd()
	_, ok := msg.(AlbumTrackViewClosedMsg)
	assert.True(t, ok, "Esc must emit AlbumTrackViewClosedMsg, got %T", msg)
}

// TestAlbumsPane_Esc_DiscardsPendingDebounceTick verifies that pressing Esc before the
// 150ms debounce tick fires causes the tick to be discarded (because albumIntent is cleared).
func TestAlbumsPane_Esc_DiscardsPendingDebounceTick(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	// Open track view for album al1 — starts 150ms debounce.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)
	require.True(t, a.inTrackView)

	// Press Esc before the debounce fires — clears albumIntent.
	model2, _ := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = model2.(*AlbumsPane)
	require.False(t, a.inTrackView)

	// Now simulate the debounce tick arriving (intent for al1, which is now stale).
	staleDebounce := albumDebounceMsg{intent: albumDebounceIntent{albumID: "al1", offset: 0}}
	_, cmd := a.Update(staleDebounce)

	assert.Nil(t, cmd, "debounce tick after Esc must be discarded because albumIntent is cleared")
}

// TestAlbumsPane_EnterOnTrack_EmitsPlayContextMsg verifies that pressing Enter on a
// track in the sub-view emits PlayContextMsg with the album URI and track URI.
func TestAlbumsPane_EnterOnTrack_EmitsPlayContextMsg(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	// Open track view.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)

	// Simulate tracks loaded.
	tracks := []domain.Track{
		{ID: "t1", URI: "spotify:track:t1", Name: "So What"},
		{ID: "t2", URI: "spotify:track:t2", Name: "Freddie"},
	}
	model2, _ := a.Update(AlbumTracksLoadedMsg{AlbumID: "al1", Offset: 0, Tracks: tracks})
	a = model2.(*AlbumsPane)

	// Press Enter on the first track.
	model3, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = model3.(*AlbumsPane)

	require.NotNil(t, cmd, "Enter on track must emit a cmd")
	msg := cmd()
	playMsg, ok := msg.(PlayContextMsg)
	require.True(t, ok, "Enter on track must emit PlayContextMsg, got %T", msg)
	assert.NotEmpty(t, playMsg.ContextURI, "ContextURI must be the album URI")
	assert.Equal(t, "spotify:track:t1", playMsg.OffsetURI, "OffsetURI must be the selected track URI")
}

// TestAlbumsPane_EnterOnEmptyTrackList_DoesNothing verifies that pressing Enter in
// track sub-view when no tracks are loaded does nothing.
func TestAlbumsPane_EnterOnEmptyTrackList_DoesNothing(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 20)

	// Open track view but do not load tracks.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)
	a.loadedTracks = []domain.Track{} // empty but not nil

	// Press Enter — idx will be -1 (empty table).
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "Enter on empty track list must not emit a cmd")
}

// TestAlbumsPane_CheckPrefetch_CursorNearEnd_EmitsRequest verifies that checkPrefetch
// returns a cmd when cursor is within 5 rows of the end and hasMoreTracks=true,
// and returns nil when the cursor is far from the end.
func TestAlbumsPane_CheckPrefetch_CursorNearEnd_EmitsRequest(t *testing.T) {
	a := newTestAlbumsPaneWithData(true)
	a.SetSize(100, 30)

	// Open track view.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*AlbumsPane)

	// Simulate 50 loaded tracks with hasMoreTracks=true.
	tracks := make([]domain.Track, 50)
	for i := range tracks {
		tracks[i] = domain.Track{ID: fmt.Sprintf("t%d", i+1), URI: fmt.Sprintf("spotify:track:t%d", i+1), Name: fmt.Sprintf("Track %d", i+1)}
	}
	model2, _ := a.Update(AlbumTracksLoadedMsg{AlbumID: "al1", Offset: 0, Tracks: tracks, HasNext: true})
	a = model2.(*AlbumsPane)

	// Negative case: cursor at 0 (far from end) → no prefetch.
	prefetchCmd := a.checkPrefetch()
	// cursor is 0, len=50, threshold=5, so 0 >= 50-5=45 is false → no prefetch.
	assert.Nil(t, prefetchCmd, "cursor at 0 of 50 should not trigger prefetch")

	// Positive case: move cursor to the last row (index 49, which is >= 50-5=45) → emits request.
	// Send enough down-arrow presses to reach the end of the track table.
	for i := 0; i < 49; i++ {
		m, _ := a.Update(tea.KeyMsg{Type: tea.KeyDown})
		a = m.(*AlbumsPane)
	}
	// Reset tracksFetching in case it was set by intermediate down presses.
	a.tracksFetching = false

	prefetchCmd = a.checkPrefetch()
	require.NotNil(t, prefetchCmd, "cursor near end with hasMoreTracks=true should emit a prefetch cmd")

	// Execute the cmd to confirm it's a FetchAlbumTracksRequestMsg.
	msg := prefetchCmd()
	fetchReq, ok := msg.(FetchAlbumTracksRequestMsg)
	require.True(t, ok, "prefetch cmd should emit FetchAlbumTracksRequestMsg, got %T", msg)
	assert.Equal(t, "al1", fetchReq.AlbumID)
	assert.Equal(t, 50, fetchReq.Offset, "offset should equal the number of already loaded tracks")
}

// ── Story 173: Esc scroll-reset ───────────────────────────────────────────────

// TableCurrentPage returns the current page of the albums pane's main list table.
// White-box accessor for testing Esc scroll-reset (story 173).
func (a *AlbumsPane) TableCurrentPage() int { return a.table.CurrentPage() }

// TestAlbumsPane_Esc_ResetsScrollInMainListView verifies that pressing Esc in the
// main album list view (not in the track sub-view, no active filter) resets the
// table scroll position to page 1.
func TestAlbumsPane_Esc_ResetsScrollInMainListView(t *testing.T) {
	s := state.New()
	albums := make([]domain.SavedAlbum, 20)
	for i := range albums {
		albums[i] = domain.SavedAlbum{
			Album: domain.FullAlbum{
				ID:          fmt.Sprintf("al%d", i),
				Name:        fmt.Sprintf("Album %d", i+1),
				URI:         fmt.Sprintf("spotify:album:al%d", i),
				ReleaseDate: "2020-01-01",
				Artists:     []domain.Artist{{Name: "Artist"}},
			},
		}
	}
	s.SetSavedAlbums(albums)
	th := theme.Load("black")
	pane := NewAlbumsPane(s, th, true)
	// height=11 → pageSize=5 with ShowHeader=true (pageSize = height - 6).
	pane.SetSize(80, 11)

	// Scroll 8 rows down to advance past page 1.
	for i := 0; i < 8; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyDown})
		pane = m.(*AlbumsPane)
	}
	require.Greater(t, pane.TableCurrentPage(), 1, "should have scrolled past page 1")

	// Press Esc in the main list view with no active filter — should reset to page 1.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEscape})
	pane = m.(*AlbumsPane)
	assert.Equal(t, 1, pane.TableCurrentPage(), "Esc should reset table to page 1")
}

// TestAlbumsPane_ActiveFilterQuery_ReturnsCommittedQuery verifies that
// ActiveFilterQuery() reflects the committed query after f → type → Enter.
func TestAlbumsPane_ActiveFilterQuery_ReturnsCommittedQuery(t *testing.T) {
	s := state.New()
	s.SetSavedAlbums([]domain.SavedAlbum{{Album: domain.FullAlbum{
		ID: "al1", Name: "Rock Album", URI: "spotify:album:al1",
		ReleaseDate: "2020-01-01", Artists: []domain.Artist{{Name: "Artist"}},
	}}})
	pane := NewAlbumsPane(s, theme.Load("black"), true)
	pane.SetSize(80, 20)

	assert.Equal(t, "", pane.ActiveFilterQuery(), "empty before filter applied")

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	for _, r := range "rock" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	pane.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, "rock", pane.ActiveFilterQuery())
}

// TestAlbumsPane_Esc_ClearsCommittedFilter verifies that Esc in the main list view
// clears a committed filter query before falling back to scroll-reset.
func TestAlbumsPane_Esc_ClearsCommittedFilter(t *testing.T) {
	s := state.New()
	s.SetSavedAlbums([]domain.SavedAlbum{
		{Album: domain.FullAlbum{ID: "al1", Name: "Rock Album", URI: "spotify:album:al1", ReleaseDate: "2020-01-01", Artists: []domain.Artist{{Name: "Artist"}}}},
		{Album: domain.FullAlbum{ID: "al2", Name: "Jazz Album", URI: "spotify:album:al2", ReleaseDate: "2021-01-01", Artists: []domain.Artist{{Name: "Artist"}}}},
	})
	pane := NewAlbumsPane(s, theme.Load("black"), true)
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
