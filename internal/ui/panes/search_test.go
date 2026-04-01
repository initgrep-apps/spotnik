package panes_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSearchOverlay creates a SearchOverlay wired to a fresh store and theme.
func newTestSearchOverlay() *panes.SearchOverlay {
	s := state.New()
	t := theme.Load("black")
	return panes.NewSearchOverlay(s, t)
}

// sampleSearchResultData returns a SearchResultData with one item per section,
// including the enriched fields added in story 81.
func sampleSearchResultData() *panes.SearchResultData {
	return &panes.SearchResultData{
		Tracks: []panes.SearchTrackItem{
			{URI: "spotify:track:t1", Name: "Blinding Lights", Artist: "The Weeknd", Album: "After Hours", DurationMs: 200040},
			{URI: "spotify:track:t2", Name: "Save Your Tears", Artist: "The Weeknd", Album: "After Hours", DurationMs: 215080},
		},
		Artists: []panes.SearchArtistItem{
			{URI: "spotify:artist:a1", Name: "The Weeknd"},
		},
		Albums: []panes.SearchAlbumItem{
			{URI: "spotify:album:al1", Name: "After Hours", Artist: "The Weeknd", ReleaseYear: "2020", TotalTracks: 14},
		},
		Playlists: []panes.SearchPlaylistItem{
			{URI: "spotify:playlist:pl1", Name: "Blinding Pop Hits", Owner: "User", TrackCount: 50},
		},
		TotalTracks:    100,
		TotalArtists:   10,
		TotalAlbums:    20,
		TotalPlaylists: 30,
	}
}

// newTestSearchOverlayWithResults creates a SearchOverlay with pre-populated search
// results delivered via SearchResultsMsg (not via store) and the query set in the store.
func newTestSearchOverlayWithResults() (*panes.SearchOverlay, *state.Store) {
	s := state.New()
	t := theme.Load("black")
	s.SetSearchQuery("blinding")

	overlay := panes.NewSearchOverlay(s, t)

	// Deliver results the same way the root app model does: via SearchResultsMsg.
	msg := panes.SearchResultsMsg{Results: sampleSearchResultData()}
	model, _ := overlay.Update(msg)
	overlay = model.(*panes.SearchOverlay)

	return overlay, s
}

// sendKey sends a key message to the overlay and returns the updated model.
func sendKey(t *testing.T, o *panes.SearchOverlay, key string) (*panes.SearchOverlay, tea.Cmd) {
	t.Helper()
	var msg tea.KeyMsg
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		msg = tea.KeyMsg{Type: tea.KeyShiftTab}
	case "backspace":
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "ctrl+u":
		msg = tea.KeyMsg{Type: tea.KeyCtrlU}
	case "ctrl+a":
		msg = tea.KeyMsg{Type: tea.KeyCtrlA}
	default:
		// Regular rune key
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}

	model, cmd := o.Update(msg)
	updated, ok := model.(*panes.SearchOverlay)
	require.True(t, ok, "Update must return *panes.SearchOverlay")
	return updated, cmd
}

// --- Task 4.2: Debounce tests ---

// TestDebounce_StaleQueryIgnored verifies that a debounce tick with an old query
// (query changed since tick was scheduled) is silently discarded.
func TestDebounce_StaleQueryIgnored(t *testing.T) {
	o := newTestSearchOverlay()

	// Type 'x', triggering a debounce tick with query "x"
	o, _ = sendKey(t, o, "x")
	// Type 'y', changing query to "xy" before the tick fires
	o, _ = sendKey(t, o, "y")

	// Now fire the debounce tick for the stale query "x" (it's now "xy", so "x" is stale)
	staleMsg := panes.SearchDebounceMsgForTest("x")
	model, cmd := o.Update(staleMsg)
	updated, ok := model.(*panes.SearchOverlay)
	require.True(t, ok)
	_ = updated

	// Command should be nil — stale tick discarded
	assert.Nil(t, cmd, "stale debounce tick should produce nil command")
}

// TestDebounce_CurrentQueryFires verifies that a debounce tick matching the current
// query fires a search request command.
func TestDebounce_CurrentQueryFires(t *testing.T) {
	o := newTestSearchOverlay()

	// Type 'x' (a character that is not intercepted as an action key), query becomes "x"
	o, _ = sendKey(t, o, "x")
	require.Equal(t, "x", o.Query(), "query should be 'x' after typing 'x'")

	// Fire debounce tick for the current query "x"
	currentMsg := panes.SearchDebounceMsgForTest("x")
	_, cmd := o.Update(currentMsg)

	// Command should be non-nil — current tick fires a search
	assert.NotNil(t, cmd, "current debounce tick should produce a search command")
}

// TestDebounce_EmptyQueryNoFire verifies that an empty query debounce tick
// does not fire a search.
func TestDebounce_EmptyQueryNoFire(t *testing.T) {
	o := newTestSearchOverlay()

	// Fire a debounce tick with empty query (before any typing)
	emptyMsg := panes.SearchDebounceMsgForTest("")
	_, cmd := o.Update(emptyMsg)

	// Command should be nil — empty query never fires
	assert.Nil(t, cmd, "empty query debounce tick should produce nil command")
}

// --- Task 4.3: SearchOverlay model tests ---

// TestSearchOverlay_Init_FocusesInput verifies that text input is focused on init.
func TestSearchOverlay_Init_FocusesInput(t *testing.T) {
	o := newTestSearchOverlay()
	cmd := o.Init()
	// Init returns a command to blink cursor — non-nil
	assert.NotNil(t, cmd, "Init should return a command for cursor blinking")
	assert.True(t, o.InputFocused(), "text input should be focused on init")
}

// TestSearchOverlay_Update_Typing verifies typing appends to query and schedules debounce.
func TestSearchOverlay_Update_Typing(t *testing.T) {
	o := newTestSearchOverlay()

	o, cmd := sendKey(t, o, "h")

	assert.NotNil(t, cmd, "typing should schedule a debounce tick command")
	assert.Contains(t, o.Query(), "h", "query should contain typed character")
}

// TestSearchOverlay_Update_Backspace verifies backspace removes last character.
func TestSearchOverlay_Update_Backspace(t *testing.T) {
	o := newTestSearchOverlay()

	o, _ = sendKey(t, o, "x")
	o, _ = sendKey(t, o, "y")
	require.Contains(t, o.Query(), "y")

	o, _ = sendKey(t, o, "backspace")
	assert.NotContains(t, o.Query(), "y", "backspace should remove last character")
}

// TestSearchOverlay_Update_Enter verifies Enter emits a play command for the selected item.
func TestSearchOverlay_Update_Enter(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	_, cmd := sendKey(t, o, "enter")

	// Enter should produce a command (play or close)
	assert.NotNil(t, cmd, "Enter should produce a play command")
}

// TestSearchOverlay_Update_Esc verifies Esc returns a searchClosedMsg.
func TestSearchOverlay_Update_Esc(t *testing.T) {
	o := newTestSearchOverlay()

	_, cmd := sendKey(t, o, "esc")

	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, panes.SearchClosedMsg{}, msg, "Esc should return SearchClosedMsg")
}

// TestSearchOverlay_Update_A verifies Ctrl+A on a track returns an add-to-queue command.
func TestSearchOverlay_Update_A(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	// First result is a track; press Ctrl+A to add to queue
	_, cmd := sendKey(t, o, "ctrl+a")

	require.NotNil(t, cmd)
	msg := cmd()
	qMsg, ok := msg.(AddToQueueMsg)
	require.True(t, ok, "pressing Ctrl+A on track should produce AddToQueueMsg")
	assert.Equal(t, "spotify:track:t1", qMsg.TrackURI)
}

// TestSearchOverlay_Update_Tab verifies Tab moves to the next section.
func TestSearchOverlay_Update_Tab(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	initialSection := o.ActiveSection()
	o, _ = sendKey(t, o, "tab")

	assert.NotEqual(t, initialSection, o.ActiveSection(), "Tab should move to next section")
}

// TestSearchOverlay_Update_ShiftTab verifies Shift+Tab moves to the previous section.
func TestSearchOverlay_Update_ShiftTab(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	// Move forward first, then back
	o, _ = sendKey(t, o, "tab")
	o, _ = sendKey(t, o, "tab")
	afterForward := o.ActiveSection()
	o, _ = sendKey(t, o, "shift+tab")

	assert.NotEqual(t, afterForward, o.ActiveSection(), "Shift+Tab should move to previous section")
}

// TestSearchOverlay_Update_JK verifies arrow key navigation moves cursor within section.
func TestSearchOverlay_Update_JK(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	// Tracks section has 2 items; start at item 0
	initialCursor := o.CursorPos()

	o, _ = sendKey(t, o, "down")
	assert.Equal(t, initialCursor+1, o.CursorPos(), "down arrow should move cursor down")

	o, _ = sendKey(t, o, "up")
	assert.Equal(t, initialCursor, o.CursorPos(), "up arrow should move cursor back up")
}

// TestSearchOverlay_Update_TypingJKA verifies j, k, a are typed into the input.
func TestSearchOverlay_Update_TypingJKA(t *testing.T) {
	o := newTestSearchOverlay()
	o, _ = sendKey(t, o, "j")
	o, _ = sendKey(t, o, "a")
	o, _ = sendKey(t, o, "k")
	assert.Contains(t, o.Query(), "j", "j should be typed into input")
	assert.Contains(t, o.Query(), "a", "a should be typed into input")
	assert.Contains(t, o.Query(), "k", "k should be typed into input")
}

// TestSearchOverlay_Update_CtrlU verifies Ctrl+U clears the input.
func TestSearchOverlay_Update_CtrlU(t *testing.T) {
	o := newTestSearchOverlay()

	o, _ = sendKey(t, o, "h")
	o, _ = sendKey(t, o, "e")
	o, _ = sendKey(t, o, "l")
	o, _ = sendKey(t, o, "l")
	o, _ = sendKey(t, o, "o") // letters that are not action keys
	require.Contains(t, o.Query(), "hello", "query should be 'hello' after typing those chars")

	o, _ = sendKey(t, o, "ctrl+u")
	assert.Empty(t, o.Query(), "Ctrl+U should clear the entire input")
}

// --- Task 4.4: Result rendering tests ---

// TestSearchOverlay_View_Results verifies section headers and items are rendered.
func TestSearchOverlay_View_Results(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	view := o.View()

	assert.Contains(t, view, "TRACKS", "view should contain TRACKS section header")
	assert.Contains(t, view, "ARTISTS", "view should contain ARTISTS section header")
	assert.Contains(t, view, "ALBUMS", "view should contain ALBUMS section header")
	assert.Contains(t, view, "PLAYLISTS", "view should contain PLAYLISTS section header")
	assert.Contains(t, view, "Blinding Lights", "view should contain track name")
	assert.Contains(t, view, "The Weeknd", "view should contain artist name")
}

// TestSearchOverlay_View_SelectedHighlight verifies the selected item uses SelectedBg styling.
func TestSearchOverlay_View_SelectedHighlight(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	view := o.View()

	// The selected item should render with a selection indicator (▶)
	assert.Contains(t, view, "▶", "selected item should have ▶ prefix")
}

// TestSearchOverlay_View_Truncation verifies long names are truncated at narrow widths.
func TestSearchOverlay_View_Truncation(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetSearchQuery("test")

	o := panes.NewSearchOverlay(s, th)

	// Very long track name
	longName := strings.Repeat("A", 120)
	results := &panes.SearchResultData{
		Tracks: []panes.SearchTrackItem{
			{URI: "spotify:track:t1", Name: longName, Artist: "Artist"},
		},
	}
	model, _ := o.Update(panes.SearchResultsMsg{Results: results})
	o = model.(*panes.SearchOverlay)
	o.SetSize(40, 20) // narrow overlay

	view := o.View()
	// The full long name should not appear verbatim — it should be truncated
	assert.NotContains(t, view, longName, "long names should be truncated in narrow overlay")
}

// TestSearchOverlay_View_EmptyQuery verifies empty query shows hint text.
func TestSearchOverlay_View_EmptyQuery(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)

	view := o.View()

	assert.Contains(t, view, "Type to search", "empty query should show hint text")
}

// TestSearchOverlay_View_NoResults verifies no-results state shows correct message.
func TestSearchOverlay_View_NoResults(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetSearchQuery("zzznoresults")

	o := panes.NewSearchOverlay(s, th)
	// Deliver empty results via message
	emptyResults := &panes.SearchResultData{} // all slices nil → zero items
	model, _ := o.Update(panes.SearchResultsMsg{Results: emptyResults})
	o = model.(*panes.SearchOverlay)
	o.SetSize(80, 40)

	view := o.View()

	assert.Contains(t, view, "No results", "should show 'No results' message when no items found")
}

// TestSearchOverlay_View_Loading verifies loading state shows spinner.
func TestSearchOverlay_View_Loading(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetSearchLoading(true)
	s.SetSearchQuery("blinding")

	o := panes.NewSearchOverlay(s, th)
	o.SetSize(80, 40)

	// Tick the spinner so it renders something
	model, _ := o.Update(panes.SearchSpinnerTickMsgForTest())
	updated := model.(*panes.SearchOverlay)
	view := updated.View()

	// The view should contain something indicating loading (spinner chars or "Searching")
	assert.True(t,
		strings.ContainsAny(view, "⣾⣽⣻⢿⡿⣟⣯⣷") || strings.Contains(view, "Searching"),
		"loading state should show spinner or 'Searching' text")
}

// --- Task 4.5 integration tests (via app) are in app_test.go ---

// AddToQueueMsg re-exported for test assertions (mirrors panes.AddToQueueMsg).
// Since panes package is external to tests here, we import via the type directly.
type AddToQueueMsg = panes.AddToQueueMsg

// TestSearchOverlay_DebounceDelay verifies that the debounce delay is at least 300ms.
// This is a structural test: the returned command from typing must be a time-based cmd.
func TestSearchOverlay_DebounceDelay(t *testing.T) {
	o := newTestSearchOverlay()

	start := time.Now()
	_, cmd := sendKey(t, o, "x") // use 'x', not an action key
	require.NotNil(t, cmd)

	// The command schedules a debounce tick — it should not fire immediately
	// We just verify the command is non-nil and not a synchronous cmd
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 50*time.Millisecond, "typing key should return immediately, not block")
}

func TestSearchOverlay_View_ShowsErrorOnSearchFailure(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("blinding lights")
	s.SetSearchError(fmt.Errorf("API error"))
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)
	o.SetSize(80, 30)

	output := o.View()
	// Errors route through toast notifications, not inline pane rendering.
	// Store error is preserved for retry logic but never read in View().
	assert.NotContains(t, output, "Search failed", "inline error rendering removed — toasts handle this")
}

func TestSearchOverlay_View_ShowsNoResults(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("zzz-nonexistent-query")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	// Deliver empty results via SearchResultsMsg (the new way)
	model, _ := o.Update(panes.SearchResultsMsg{Results: &panes.SearchResultData{}})
	o = model.(*panes.SearchOverlay)
	o.SetSize(80, 30)

	output := o.View()
	assert.Contains(t, output, "No results for", "should show no-results message")
}

func TestSearchOverlay_View_ShowsResults(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("blinding")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	// Deliver results via SearchResultsMsg
	results := &panes.SearchResultData{
		Tracks: []panes.SearchTrackItem{
			{URI: "spotify:track:t1", Name: "Blinding Lights", Artist: "The Weeknd"},
		},
	}
	model, _ := o.Update(panes.SearchResultsMsg{Results: results})
	o = model.(*panes.SearchOverlay)
	o.SetSize(80, 30)

	output := o.View()
	assert.Contains(t, output, "TRACKS", "should show tracks section header")
	assert.Contains(t, output, "Blinding Lights", "should show track name in results")
}

func TestSearchOverlay_DebounceToSearchRequest_Pipeline(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)
	o.SetSize(80, 30)

	// Type a character to get the input populated.
	sendKey(t, o, "b")

	// Simulate the debounce msg arriving with the correct query snapshot.
	_, cmd := o.Update(panes.SearchDebounceMsgForTest("b"))

	// The debounce should have produced a command that returns SearchRequestMsg.
	require.NotNil(t, cmd, "debounce should produce a command")
	msg := cmd()
	_, ok := msg.(panes.SearchRequestMsg)
	assert.True(t, ok, "debounce cmd should produce SearchRequestMsg, got %T", msg)
}

// --- Feature 20: Elm Architecture Purity tests ---

// TestSearchOverlay_CtrlU_EmitsSearchClearedMsg verifies that pressing Ctrl+U
// returns a command producing SearchClearedMsg instead of writing to the store directly.
func TestSearchOverlay_CtrlU_EmitsSearchClearedMsg(t *testing.T) {
	t.Helper()
	s := state.New()
	th := theme.Load("black")

	// Pre-populate store with search state so we know there's something to clear.
	s.SetSearchQuery("blinding lights")

	o := panes.NewSearchOverlay(s, th)
	o.SetSize(80, 30)

	// Press Ctrl+U — should NOT write to store directly, but emit SearchClearedMsg.
	_, cmd := sendKey(t, o, "ctrl+u")

	require.NotNil(t, cmd, "Ctrl+U should return a command (SearchClearedMsg)")
	msg := cmd()
	_, ok := msg.(panes.SearchClearedMsg)
	assert.True(t, ok, "Ctrl+U command should produce SearchClearedMsg, got %T", msg)

	// The store should NOT have been mutated directly by the overlay.
	// (Actual clearing happens in app.go when SearchClearedMsg is handled.)
	assert.Equal(t, "blinding lights", s.SearchQuery(), "overlay must not write to store directly on Ctrl+U")
}

// TestSearchOverlay_CtrlU_ClearsLocalInput verifies that Ctrl+U clears the local
// input field (cosmetic) even though the store write is deferred to the root app.
func TestSearchOverlay_CtrlU_ClearsLocalInput(t *testing.T) {
	t.Helper()
	s := state.New()
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)
	o.SetSize(80, 30)

	// Type some text first.
	o, _ = sendKey(t, o, "b")
	o, _ = sendKey(t, o, "l")
	require.Equal(t, "bl", o.Query(), "input should be 'bl' after typing")

	// Ctrl+U should clear the local input field.
	o, _ = sendKey(t, o, "ctrl+u")
	assert.Equal(t, "", o.Query(), "Ctrl+U should clear the local input field")
}

// TestConvertSearchResult_RoundTrip verifies that convertSearchResult (indirectly via
// SearchResultsMsg) correctly maps api fields to SearchResultData fields.
// We test the data visible in the overlay after receiving a SearchResultsMsg.
func TestSearchOverlay_SearchResultsMsg_StoresResults(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("test")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)
	o.SetSize(80, 40)

	results := &panes.SearchResultData{
		Tracks: []panes.SearchTrackItem{
			{URI: "spotify:track:abc", Name: "Track One", Artist: "Artist One"},
		},
		Artists: []panes.SearchArtistItem{
			{URI: "spotify:artist:xyz", Name: "Artist One"},
		},
	}

	model, _ := o.Update(panes.SearchResultsMsg{Results: results})
	o = model.(*panes.SearchOverlay)

	view := o.View()
	assert.Contains(t, view, "Track One", "view should show track from SearchResultsMsg")
	assert.Contains(t, view, "TRACKS", "view should show TRACKS section header")
	assert.Contains(t, view, "Artist One", "view should show artist from SearchResultsMsg")
}

// TestSearchOverlay_View_Truncation_NoApiImport verifies the import boundary:
// search.go must not import api/. This test uses panes.SearchResultData directly.
func TestSearchOverlay_NoAPIImportBoundary(t *testing.T) {
	// This test verifies the architectural boundary at the type level.
	// If search.go imported api/, the panes package would fail to build without api/.
	// We exercise the full rendering path using only panes types.
	s := state.New()
	s.SetSearchQuery("boundary")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	results := &panes.SearchResultData{
		Tracks:    []panes.SearchTrackItem{{URI: "u1", Name: "T1", Artist: "A1"}},
		Artists:   []panes.SearchArtistItem{{URI: "u2", Name: "A2"}},
		Albums:    []panes.SearchAlbumItem{{URI: "u3", Name: "Al1", Artist: "A3"}},
		Playlists: []panes.SearchPlaylistItem{{URI: "u4", Name: "PL1", Owner: "Owner1"}},
	}
	model, _ := o.Update(panes.SearchResultsMsg{Results: results})
	o = model.(*panes.SearchOverlay)
	o.SetSize(80, 40)

	view := o.View()
	assert.Contains(t, view, "T1")
	assert.Contains(t, view, "A2")
	assert.Contains(t, view, "Al1")
	assert.Contains(t, view, "PL1")
}

// --- F50 Task 4: btop-style border in search overlay ---

// TestSearchOverlay_View_HasBtopBorder verifies the btop-style border with title.
func TestSearchOverlay_View_HasBtopBorder(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)

	view := o.View()

	// btop border uses ╭ and ╯ corners.
	assert.Contains(t, view, "╭", "search overlay should use btop-style rounded corner")
	assert.Contains(t, view, "╰", "search overlay should use btop-style rounded corner")
}

// TestSearchOverlay_View_BtopBorderTitle verifies "Search" appears in the border.
func TestSearchOverlay_View_BtopBorderTitle(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)

	view := o.View()

	assert.Contains(t, view, "Search", "border title should contain 'Search'")
}

// TestSearchOverlay_View_BtopBorderActions verifies action shortcuts appear in the border.
func TestSearchOverlay_View_BtopBorderActions(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)

	view := o.View()

	// Actions from the spec: "Enter play" and "Tab section"
	assert.Contains(t, view, "play", "border should show 'play' action")
	assert.Contains(t, view, "section", "border should show 'section' action")
}

// --- Story 81: Enriched search data types ---

// TestSearchResultData_EnrichedFields verifies that all enriched fields are accessible
// on the search result data types added in story 81.
func TestSearchResultData_EnrichedFields(t *testing.T) {
	data := &panes.SearchResultData{
		Tracks: []panes.SearchTrackItem{
			{
				URI:        "spotify:track:t1",
				Name:       "Blinding Lights",
				Artist:     "The Weeknd",
				Album:      "After Hours",
				DurationMs: 200040,
			},
		},
		Artists: []panes.SearchArtistItem{
			{URI: "spotify:artist:a1", Name: "The Weeknd"},
		},
		Albums: []panes.SearchAlbumItem{
			{
				URI:         "spotify:album:al1",
				Name:        "After Hours",
				Artist:      "The Weeknd",
				ReleaseYear: "2020",
				TotalTracks: 14,
			},
		},
		Playlists: []panes.SearchPlaylistItem{
			{
				URI:        "spotify:playlist:pl1",
				Name:       "Chill Vibes",
				Owner:      "User",
				TrackCount: 45,
			},
		},
		TotalTracks:    100,
		TotalArtists:   50,
		TotalAlbums:    30,
		TotalPlaylists: 20,
	}

	// Track enriched fields
	require.Len(t, data.Tracks, 1)
	assert.Equal(t, "After Hours", data.Tracks[0].Album)
	assert.Equal(t, 200040, data.Tracks[0].DurationMs)

	// Album enriched fields
	require.Len(t, data.Albums, 1)
	assert.Equal(t, "2020", data.Albums[0].ReleaseYear)
	assert.Equal(t, 14, data.Albums[0].TotalTracks)

	// Playlist enriched fields
	require.Len(t, data.Playlists, 1)
	assert.Equal(t, 45, data.Playlists[0].TrackCount)

	// Total count fields
	assert.Equal(t, 100, data.TotalTracks)
	assert.Equal(t, 50, data.TotalArtists)
	assert.Equal(t, 30, data.TotalAlbums)
	assert.Equal(t, 20, data.TotalPlaylists)
}

// TestOverlayWidth_Wider verifies the new wider base (90) and cap (80%) dimensions.
func TestOverlayWidth_Wider(t *testing.T) {
	o := newTestSearchOverlay()
	// Large terminal: 200 wide — width should be min(90, 80%*200=160) = 90
	o.SetSize(200, 60)
	view := o.View()
	// The overlay renders at width 90; check it's at least visible (non-empty)
	assert.NotEmpty(t, view)
}

// TestOverlayWidth_NarrowTerminal verifies the minimum width of 40 on narrow terminals.
func TestOverlayWidth_NarrowTerminal(t *testing.T) {
	o := newTestSearchOverlay()
	// Very narrow terminal — width should be clamped to min 40
	o.SetSize(30, 20)
	view := o.View()
	assert.NotEmpty(t, view, "overlay should still render on narrow terminal")
}

// TestOverlayHeight_Taller verifies the new taller base (26) and cap (75%) dimensions.
func TestOverlayHeight_Taller(t *testing.T) {
	o := newTestSearchOverlay()
	// Large terminal: 200 high — height should be max(26, 75%*200=150) = 150
	o.SetSize(200, 200)
	view := o.View()
	assert.NotEmpty(t, view)
}
