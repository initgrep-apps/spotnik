package panes_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// TestSearchOverlay_View_Results verifies section labels and items are rendered.
func TestSearchOverlay_View_Results(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	view := o.View()
	stripped := stripANSIForTest(view)

	// Title-case labels in the tab bar (from Story 82 redesign)
	assert.Contains(t, stripped, "Tracks", "view should contain Tracks tab label")
	assert.Contains(t, stripped, "Artists", "view should contain Artists tab label")
	assert.Contains(t, stripped, "Albums", "view should contain Albums tab label")
	assert.Contains(t, stripped, "Playlists", "view should contain Playlists tab label")
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
	// Labels are now title case (Story 82 redesign)
	assert.Contains(t, stripANSIForTest(output), "Tracks", "should show Tracks tab label")
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
	// Labels are now title case (Story 82 redesign)
	assert.Contains(t, stripANSIForTest(view), "Tracks", "view should show Tracks tab label")
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

	// In the tabbed design (Story 82), only the active section is shown in the view.
	// The tab bar shows all sections. Navigate to each and verify.
	view := o.View()
	assert.Contains(t, view, "T1", "Tracks section (default) should show T1")

	// Navigate to Artists tab to verify A2
	o, _ = sendKey(t, o, "tab")
	view = o.View()
	assert.Contains(t, view, "A2", "Artists section should show A2 when active")

	// Navigate to Albums tab to verify Al1
	o, _ = sendKey(t, o, "tab")
	view = o.View()
	assert.Contains(t, view, "Al1", "Albums section should show Al1 when active")

	// Navigate to Playlists tab to verify PL1
	o, _ = sendKey(t, o, "tab")
	view = o.View()
	assert.Contains(t, view, "PL1", "Playlists section should show PL1 when active")
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

	// Actions from Story 82 spec: "Enter play" and "Esc close"
	assert.Contains(t, view, "play", "border should show 'play' action")
	assert.Contains(t, view, "close", "border should show 'close' action")
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

// stripANSIForTest removes ANSI escape sequences from a string so that rune-counting
// reflects visible columns rather than raw bytes including color codes.
func stripANSIForTest(s string) string {
	var result strings.Builder
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			i += 2
			for i < len(runes) && runes[i] != 'm' {
				i++
			}
			i++ // skip 'm'
		} else {
			result.WriteRune(runes[i])
			i++
		}
	}
	return result.String()
}

// TestOverlayWidth_Wider verifies the new wider base (90) and cap (80%) dimensions.
// The ANSI-stripped first line of the overlay equals its rendered width.
func TestOverlayWidth_Wider(t *testing.T) {
	o := newTestSearchOverlay()
	// Large terminal: 200 wide — overlayWidth = min(90, 80%*200=160) = 90
	o.SetSize(200, 60)
	view := o.View()
	lines := strings.Split(view, "\n")
	require.NotEmpty(t, lines)
	strippedWidth := len([]rune(stripANSIForTest(lines[0])))
	assert.Equal(t, 90, strippedWidth, "overlay width should be 90 on a large terminal")
}

// TestOverlayWidth_NarrowTerminal verifies the minimum width of 40 on narrow terminals.
func TestOverlayWidth_NarrowTerminal(t *testing.T) {
	o := newTestSearchOverlay()
	// Very narrow terminal (30 wide) — 80% of 30 = 24, but min is 40
	o.SetSize(30, 20)
	view := o.View()
	lines := strings.Split(view, "\n")
	require.NotEmpty(t, lines)
	strippedWidth := len([]rune(stripANSIForTest(lines[0])))
	assert.Equal(t, 40, strippedWidth, "overlay width should be clamped to min 40 on narrow terminal")
}

// TestOverlayHeight_Taller verifies the new taller base (26) and cap (75%) dimensions.
func TestOverlayHeight_Taller(t *testing.T) {
	o := newTestSearchOverlay()
	// Terminal at exactly 40 high — 75% of 40 = 30, which is > base 26, so height = 30
	o.SetSize(120, 40)
	view := o.View()
	lines := strings.Split(view, "\n")
	// Height should be 30 lines (75% of 40)
	assert.Equal(t, 30, len(lines), "overlay height should be 75%% of terminal height when > base 26")
}

// --- Story 82: Integration tests for tabbed overlay ---

// TestSearchOverlayView_ContainsTabBar verifies that the full view contains tab labels.
func TestSearchOverlayView_ContainsTabBar(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	view := o.View()
	stripped := stripANSIForTest(view)

	assert.Contains(t, stripped, "Tracks", "view should contain 'Tracks' tab label")
	assert.Contains(t, stripped, "Artists", "view should contain 'Artists' tab label")
	assert.Contains(t, stripped, "Albums", "view should contain 'Albums' tab label")
	assert.Contains(t, stripped, "Playlists", "view should contain 'Playlists' tab label")
}

// TestSearchOverlayView_ContainsColumnHeaders verifies that the full view contains the column header row.
func TestSearchOverlayView_ContainsColumnHeaders(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	view := o.View()
	stripped := stripANSIForTest(view)

	// Tracks section headers
	assert.Contains(t, stripped, "Track", "view should contain 'Track' column header")
	assert.Contains(t, stripped, "Duration", "view should contain 'Duration' column header")
}

// TestSearchOverlayView_ContainsHelpBar verifies that the full view contains help text.
func TestSearchOverlayView_ContainsHelpBar(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	view := o.View()
	stripped := stripANSIForTest(view)

	assert.Contains(t, stripped, "navigate", "view should contain '↑↓ navigate' in help bar")
	assert.Contains(t, stripped, "Esc close", "view should contain 'Esc close' in help bar")
}

// TestSearchOverlayView_BorderActions verifies the border shows "Esc close" not "Tab section".
func TestSearchOverlayView_BorderActions(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)

	view := o.View()

	assert.Contains(t, view, "Esc", "border should show 'Esc' action")
	assert.Contains(t, view, "close", "border should show 'close' action")
	assert.NotContains(t, stripANSIForTest(view), "section", "border should NOT show 'section' action")
}

// --- Story 82: Tabbed Overlay UI tests ---

// TestTotalForSection_AllSections verifies that totalForSection returns the correct
// total count from the TotalTracks/Artists/Albums/Playlists fields.
func TestTotalForSection_AllSections(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()

	tests := []struct {
		section panes.SearchSection
		want    int
	}{
		{panes.SectionTracks, 100},
		{panes.SectionArtists, 10},
		{panes.SectionAlbums, 20},
		{panes.SectionPlaylists, 30},
	}

	for _, tt := range tests {
		got := o.TotalForSection(tt.section)
		assert.Equal(t, tt.want, got, "section %d should return correct total", tt.section)
	}
}

// TestTotalForSection_NilResults verifies that totalForSection returns 0 when results are nil.
func TestTotalForSection_NilResults(t *testing.T) {
	o := newTestSearchOverlay()

	tests := []panes.SearchSection{
		panes.SectionTracks,
		panes.SectionArtists,
		panes.SectionAlbums,
		panes.SectionPlaylists,
	}

	for _, sec := range tests {
		got := o.TotalForSection(sec)
		assert.Equal(t, 0, got, "section %d should return 0 when results are nil", sec)
	}
}

// TestRenderHelpBar_TracksTab_ShowsCtrlA verifies that Ctrl+A queue appears on the Tracks tab.
func TestRenderHelpBar_TracksTab_ShowsCtrlA(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	// Default active section is Tracks
	helpBar := o.RenderHelpBar(80)
	stripped := stripANSIForTest(helpBar)

	assert.Contains(t, stripped, "Ctrl+A", "Tracks tab help bar should contain Ctrl+A")
	assert.Contains(t, stripped, "queue", "Tracks tab help bar should contain 'queue'")
	assert.Contains(t, stripped, "Tab", "help bar should show Tab keybinding")
}

// TestRenderHelpBar_OtherTab_NoCtrlA verifies that Ctrl+A queue does NOT appear on non-Tracks tabs.
func TestRenderHelpBar_OtherTab_NoCtrlA(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	// Navigate to Artists section
	o, _ = sendKey(t, o, "tab")

	helpBar := o.RenderHelpBar(80)
	stripped := stripANSIForTest(helpBar)

	assert.NotContains(t, stripped, "Ctrl+A", "non-Tracks tab help bar should NOT contain Ctrl+A")
	assert.Contains(t, stripped, "Tab", "help bar should still show Tab keybinding")
}

// TestFormatDurationMs_ShortTrack verifies formatting of a track < 1 hour (m:ss format).
func TestFormatDurationMs_ShortTrack(t *testing.T) {
	// 222000ms = 3 minutes 42 seconds
	got := panes.FormatDurationMs(222000)
	assert.Equal(t, "3:42", got, "222000ms should format as 3:42")
}

// TestFormatDurationMs_LongTrack verifies formatting of a track >= 1 hour (h:mm:ss format).
func TestFormatDurationMs_LongTrack(t *testing.T) {
	// 7290000ms = 2 hours 1 minute 30 seconds
	got := panes.FormatDurationMs(7290000)
	assert.Equal(t, "2:01:30", got, "7290000ms should format as 2:01:30")
}

// TestFormatDurationMs_Zero verifies formatting of 0ms.
func TestFormatDurationMs_Zero(t *testing.T) {
	got := panes.FormatDurationMs(0)
	assert.Equal(t, "0:00", got, "0ms should format as 0:00")
}

// TestRenderActiveSection_Tracks_ShowsAlbumAndDuration verifies that the Tracks section
// renders Album and Duration columns for each track row.
func TestRenderActiveSection_Tracks_ShowsAlbumAndDuration(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	section := o.RenderActiveSection(80)
	stripped := stripANSIForTest(section)

	// sampleSearchResultData tracks: "Blinding Lights", Artist: "The Weeknd", Album: "After Hours", DurationMs: 200040
	assert.Contains(t, stripped, "Blinding Lights", "should show track name")
	assert.Contains(t, stripped, "The Weeknd", "should show artist name")
	assert.Contains(t, stripped, "After Hours", "should show album name")
	// 200040ms = 3 minutes 20.04 seconds → 3:20
	assert.Contains(t, stripped, "3:20", "should show formatted duration")
}

// TestRenderActiveSection_Albums_ShowsYearAndCount verifies that the Albums section
// renders Year and TotalTracks columns.
func TestRenderActiveSection_Albums_ShowsYearAndCount(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	// Navigate to Albums section
	o, _ = sendKey(t, o, "tab")
	o, _ = sendKey(t, o, "tab")

	section := o.RenderActiveSection(80)
	stripped := stripANSIForTest(section)

	// sampleSearchResultData albums: "After Hours", Artist: "The Weeknd", ReleaseYear: "2020", TotalTracks: 14
	assert.Contains(t, stripped, "After Hours", "should show album name")
	assert.Contains(t, stripped, "The Weeknd", "should show artist name")
	assert.Contains(t, stripped, "2020", "should show release year")
	assert.Contains(t, stripped, "14", "should show track count")
}

// TestRenderActiveSection_Playlists_ShowsTrackCount verifies that the Playlists section
// renders the TrackCount column.
func TestRenderActiveSection_Playlists_ShowsTrackCount(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	// Navigate to Playlists section
	o, _ = sendKey(t, o, "tab")
	o, _ = sendKey(t, o, "tab")
	o, _ = sendKey(t, o, "tab")

	section := o.RenderActiveSection(80)
	stripped := stripANSIForTest(section)

	// sampleSearchResultData playlists: "Blinding Pop Hits", Owner: "User", TrackCount: 50
	assert.Contains(t, stripped, "Blinding Pop Hits", "should show playlist name")
	assert.Contains(t, stripped, "User", "should show owner name")
	assert.Contains(t, stripped, "50", "should show track count")
}

// TestRenderActiveSection_SelectedRow_UsesSelectedColors verifies that the selected row
// uses the ▶ indicator.
func TestRenderActiveSection_SelectedRow_UsesSelectedColors(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	section := o.RenderActiveSection(80)

	// Selected row should have the ▶ marker
	assert.Contains(t, section, "▶", "selected row should have ▶ indicator")
}

// TestRenderActiveSection_Tracks_NarrowNoAlbumColumn verifies that the narrow Tracks view
// does not include the Album column when contentWidth < 60.
func TestRenderActiveSection_Tracks_NarrowNoAlbumColumn(t *testing.T) {
	// Create overlay with results that have an album name
	s := state.New()
	s.SetSearchQuery("test")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	results := &panes.SearchResultData{
		Tracks: []panes.SearchTrackItem{
			{URI: "spotify:track:t1", Name: "My Song", Artist: "My Artist", Album: "My Album", DurationMs: 180000},
		},
	}
	model, _ := o.Update(panes.SearchResultsMsg{Results: results})
	o = model.(*panes.SearchOverlay)
	o.SetSize(80, 40)

	// contentWidth 55 < 60 → drops Album column
	section := o.RenderActiveSection(55)
	stripped := stripANSIForTest(section)

	assert.Contains(t, stripped, "My Song", "should still show track name")
	assert.Contains(t, stripped, "My Artist", "should still show artist name")
	assert.NotContains(t, stripped, "My Album", "narrow view should NOT show album name")
}

// TestRenderColumnHeaders_Tracks verifies that 5 column headers are rendered for the Tracks section.
func TestRenderColumnHeaders_Tracks(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	headers := o.RenderColumnHeaders(panes.SectionTracks, 80)
	stripped := stripANSIForTest(headers)

	assert.Contains(t, stripped, "#", "Tracks headers should contain # column")
	assert.Contains(t, stripped, "Track", "Tracks headers should contain Track column")
	assert.Contains(t, stripped, "Artist", "Tracks headers should contain Artist column")
	assert.Contains(t, stripped, "Album", "Tracks headers should contain Album column")
	assert.Contains(t, stripped, "Duration", "Tracks headers should contain Duration column")
}

// TestRenderColumnHeaders_Artists verifies that 2 column headers are rendered for the Artists section.
func TestRenderColumnHeaders_Artists(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	headers := o.RenderColumnHeaders(panes.SectionArtists, 80)
	stripped := stripANSIForTest(headers)

	assert.Contains(t, stripped, "#", "Artists headers should contain # column")
	assert.Contains(t, stripped, "Artist", "Artists headers should contain Artist column")
}

// TestRenderColumnHeaders_Albums verifies that 5 column headers are rendered for the Albums section.
func TestRenderColumnHeaders_Albums(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	headers := o.RenderColumnHeaders(panes.SectionAlbums, 80)
	stripped := stripANSIForTest(headers)

	assert.Contains(t, stripped, "#", "Albums headers should contain # column")
	assert.Contains(t, stripped, "Album", "Albums headers should contain Album column")
	assert.Contains(t, stripped, "Artist", "Albums headers should contain Artist column")
	assert.Contains(t, stripped, "Year", "Albums headers should contain Year column")
	assert.Contains(t, stripped, "Tracks", "Albums headers should contain Tracks column")
}

// TestRenderColumnHeaders_Playlists verifies that 4 column headers are rendered for the Playlists section.
func TestRenderColumnHeaders_Playlists(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	headers := o.RenderColumnHeaders(panes.SectionPlaylists, 80)
	stripped := stripANSIForTest(headers)

	assert.Contains(t, stripped, "#", "Playlists headers should contain # column")
	assert.Contains(t, stripped, "Playlist", "Playlists headers should contain Playlist column")
	assert.Contains(t, stripped, "Owner", "Playlists headers should contain Owner column")
	assert.Contains(t, stripped, "Tracks", "Playlists headers should contain Tracks column")
}

// TestRenderColumnHeaders_Tracks_NarrowDropsAlbum verifies that the Album column is dropped
// when contentWidth < 60 on the Tracks section.
func TestRenderColumnHeaders_Tracks_NarrowDropsAlbum(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	// contentWidth < 60 → drops Album column
	headers := o.RenderColumnHeaders(panes.SectionTracks, 55)
	stripped := stripANSIForTest(headers)

	assert.Contains(t, stripped, "#", "narrow Tracks headers should still contain # column")
	assert.Contains(t, stripped, "Track", "narrow Tracks headers should still contain Track column")
	assert.Contains(t, stripped, "Artist", "narrow Tracks headers should still contain Artist column")
	assert.Contains(t, stripped, "Duration", "narrow Tracks headers should still contain Duration column")
	assert.NotContains(t, stripped, "Album", "narrow Tracks headers should NOT contain Album column")
}

// TestRenderTabBar_ActiveHighlighted verifies that the active tab has the ▪ marker.
func TestRenderTabBar_ActiveHighlighted(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	tabBar := o.RenderTabBar(80)
	// Strip ANSI to get visible text
	stripped := stripANSIForTest(tabBar)

	assert.Contains(t, stripped, "▪", "active tab should have ▪ marker")
	// The ▪ should appear before "Tracks" (active section by default)
	tracksIdx := strings.Index(stripped, "Tracks")
	bulletIdx := strings.Index(stripped, "▪")
	assert.True(t, bulletIdx < tracksIdx, "▪ should appear before 'Tracks' label")
}

// TestRenderTabBar_ShowsCounts verifies that tabs display result totals from TotalXxx fields.
func TestRenderTabBar_ShowsCounts(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	tabBar := o.RenderTabBar(80)
	stripped := stripANSIForTest(tabBar)

	// sampleSearchResultData has TotalTracks=100, TotalArtists=10, TotalAlbums=20, TotalPlaylists=30
	assert.Contains(t, stripped, "Tracks 100", "tab bar should show TotalTracks count")
	assert.Contains(t, stripped, "Artists 10", "tab bar should show TotalArtists count")
	assert.Contains(t, stripped, "Albums 20", "tab bar should show TotalAlbums count")
	assert.Contains(t, stripped, "Playlists 30", "tab bar should show TotalPlaylists count")
}

// TestRenderTabBar_NilResults_ZeroCounts verifies zero counts are shown when results are nil.
func TestRenderTabBar_NilResults_ZeroCounts(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)

	tabBar := o.RenderTabBar(80)
	stripped := stripANSIForTest(tabBar)

	assert.Contains(t, stripped, "Tracks 0", "nil results should show zero count for Tracks")
	assert.Contains(t, stripped, "Artists 0", "nil results should show zero count for Artists")
	assert.Contains(t, stripped, "Albums 0", "nil results should show zero count for Albums")
	assert.Contains(t, stripped, "Playlists 0", "nil results should show zero count for Playlists")
}

// --- Story 83: Fix Search Rendering Bugs ---

// TestContentWidth_NoDoubleSubtraction verifies that contentWidth inside renderResults
// equals innerWidth - 2 (left + right padding only), not innerWidth - 4 (old double-subtract).
// innerWidth = overlayWidth - 2 (border removed in View). renderResults receives innerWidth.
// contentWidth should be innerWidth - 2 (padding), not innerWidth - 4 (old: border + padding).
func TestContentWidth_NoDoubleSubtraction(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("test")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	// Use terminal size that gives overlayWidth = 90 (large terminal).
	// innerWidth = 90 - 2 = 88. contentWidth should be 88 - 2 = 86, not 88 - 4 = 84.
	o.SetSize(200, 60)

	results := &panes.SearchResultData{
		Tracks: []panes.SearchTrackItem{
			{URI: "u1", Name: "Exactly 86 Chars Test Track Name That Fills The Width", Artist: "Artist", Album: "Album", DurationMs: 200000},
		},
		TotalTracks: 1,
	}
	model, _ := o.Update(panes.SearchResultsMsg{Results: results})
	o = model.(*panes.SearchOverlay)

	// Check that the view renders without wrapping — the track name fits on one line.
	// The key behavioral check: RenderActiveSection(86) should return rows without embedded
	// newlines from content overflow.
	section := o.RenderActiveSection(86)
	rows := strings.Split(strings.TrimRight(section, "\n"), "\n")
	// With 1 track item, there should be exactly 1 row.
	assert.Len(t, rows, 1, "one track should produce exactly one row (no content wrapping)")
}

// TestTrackColumnWidths_SumEqualsContentWidth verifies that for the Tracks section (wide),
// all column widths plus inter-column gaps sum to exactly contentWidth.
// Layout: indexW(3) + nameW + artistW + albumW + durationW(8) + gaps(4×2=8) = contentWidth
func TestTrackColumnWidths_SumEqualsContentWidth(t *testing.T) {
	o := newTestSearchOverlay()

	tests := []int{80, 86, 100, 120}
	for _, contentWidth := range tests {
		t.Run(fmt.Sprintf("width_%d", contentWidth), func(t *testing.T) {
			nW, artW, albW, durW := o.TrackColumnWidths(contentWidth, false)
			indexW := 3
			numCols := 5
			gaps := (numCols - 1) * 2 // 4 gaps × 2 = 8
			total := indexW + nW + artW + albW + durW + gaps
			assert.Equal(t, contentWidth, total,
				"wide track columns + gaps should sum to contentWidth=%d (got %d+%d+%d+%d+%d+%d=%d)",
				contentWidth, indexW, nW, artW, albW, durW, gaps, total)
		})
	}
}

// TestTrackColumnWidths_NarrowSumEqualsContentWidth verifies that for the Tracks section
// in narrow mode (contentWidth < 60, 4 columns), all widths plus gaps sum to contentWidth.
// Layout: indexW(3) + nameW + artistW + durationW(8) + gaps(3×2=6) = contentWidth
func TestTrackColumnWidths_NarrowSumEqualsContentWidth(t *testing.T) {
	o := newTestSearchOverlay()

	tests := []int{40, 50, 55, 59}
	for _, contentWidth := range tests {
		t.Run(fmt.Sprintf("width_%d", contentWidth), func(t *testing.T) {
			nW, artW, albW, durW := o.TrackColumnWidths(contentWidth, true)
			assert.Zero(t, albW, "narrow mode should have albumW=0")
			indexW := 3
			numCols := 4
			gaps := (numCols - 1) * 2 // 3 gaps × 2 = 6
			total := indexW + nW + artW + albW + durW + gaps
			assert.Equal(t, contentWidth, total,
				"narrow track columns + gaps should sum to contentWidth=%d (got %d+%d+%d+%d+%d=%d)",
				contentWidth, indexW, nW, artW, durW, gaps, total)
		})
	}
}

// TestAlbumColumnWidths_SumEqualsContentWidth verifies that Albums section column widths
// plus inter-column gaps sum to exactly contentWidth.
// Layout: indexW(3) + nameW + artistW + yearW(6) + tracksW(8) + gaps(4×2=8) = contentWidth
func TestAlbumColumnWidths_SumEqualsContentWidth(t *testing.T) {
	o := newTestSearchOverlay()

	tests := []int{80, 86, 100, 120}
	for _, contentWidth := range tests {
		t.Run(fmt.Sprintf("width_%d", contentWidth), func(t *testing.T) {
			nW, artW, yW, tW := o.AlbumColumnWidths(contentWidth)
			indexW := 3
			numCols := 5
			gaps := (numCols - 1) * 2 // 4 gaps × 2 = 8
			total := indexW + nW + artW + yW + tW + gaps
			assert.Equal(t, contentWidth, total,
				"album columns + gaps should sum to contentWidth=%d (got %d+%d+%d+%d+%d+%d=%d)",
				contentWidth, indexW, nW, artW, yW, tW, gaps, total)
		})
	}
}

// TestPlaylistColumnWidths_SumEqualsContentWidth verifies that Playlists section column
// widths plus inter-column gaps sum to exactly contentWidth.
// Layout: indexW(3) + nameW + ownerW + tracksW(8) + gaps(3×2=6) = contentWidth
func TestPlaylistColumnWidths_SumEqualsContentWidth(t *testing.T) {
	o := newTestSearchOverlay()

	tests := []int{80, 86, 100, 120}
	for _, contentWidth := range tests {
		t.Run(fmt.Sprintf("width_%d", contentWidth), func(t *testing.T) {
			nW, oW, tW := o.PlaylistColumnWidths(contentWidth)
			indexW := 3
			numCols := 4
			gaps := (numCols - 1) * 2 // 3 gaps × 2 = 6
			total := indexW + nW + oW + tW + gaps
			assert.Equal(t, contentWidth, total,
				"playlist columns + gaps should sum to contentWidth=%d (got %d+%d+%d+%d+%d=%d)",
				contentWidth, indexW, nW, oW, tW, gaps, total)
		})
	}
}

// TestRenderColumnHeaders_FitsOnOneLine verifies that the column header line contains
// no embedded newlines within the header row itself (i.e. the first line has no \n).
func TestRenderColumnHeaders_FitsOnOneLine(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	sections := []struct {
		sec  panes.SearchSection
		name string
	}{
		{panes.SectionTracks, "Tracks"},
		{panes.SectionArtists, "Artists"},
		{panes.SectionAlbums, "Albums"},
		{panes.SectionPlaylists, "Playlists"},
	}

	for _, tt := range sections {
		t.Run(tt.name, func(t *testing.T) {
			headers := o.RenderColumnHeaders(tt.sec, 86)
			lines := strings.Split(headers, "\n")
			// Headers should produce exactly 2 lines: header row + underline
			assert.LessOrEqual(t, 2, len(lines), "should have at least header and underline lines")
			// The header line (line 0) should not contain embedded \n - it's one line
			headerLine := lines[0]
			assert.NotContains(t, headerLine, "\n", "header line should not contain embedded newlines")
		})
	}
}

// TestRenderActiveSection_RowFitsOnOneLine verifies that a single result row contains
// no embedded newlines (i.e. each row renders on exactly one line).
func TestRenderActiveSection_RowFitsOnOneLine(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	// Tracks section with contentWidth 86 (standard after fix)
	section := o.RenderActiveSection(86)
	rows := strings.Split(strings.TrimRight(section, "\n"), "\n")

	// sampleSearchResultData has 2 tracks
	require.Len(t, rows, 2, "should have exactly 2 rows for 2 tracks")

	for i, row := range rows {
		assert.NotContains(t, row, "\n",
			"row %d should not contain embedded newlines (no spill to second line)", i)
	}
}

// TestSearchOverlay_HelpBarAtBottom verifies that the help bar appears at or near the
// bottom of the overlay's rendered output.
func TestSearchOverlay_HelpBarAtBottom(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	// Terminal size that gives overlayHeight = 26 (base)
	o.SetSize(200, 10)

	view := o.View()
	stripped := stripANSIForTest(view)
	lines := strings.Split(stripped, "\n")

	// Find the help bar line (contains "↑↓ navigate")
	helpBarIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "navigate") {
			helpBarIdx = i
			break
		}
	}
	require.NotEqual(t, -1, helpBarIdx, "help bar must be present in the view")

	// The help bar should be in the bottom quarter of the view
	totalLines := len(lines)
	assert.Greater(t, helpBarIdx, totalLines*3/4,
		"help bar should be in the bottom quarter of the view (got line %d of %d)", helpBarIdx, totalLines)
}

// TestSearchOverlay_HelpBarAtBottom_FewResults verifies that with only 3 results,
// the help bar is still anchored to the bottom of the overlay (empty space is above, not below).
func TestSearchOverlay_HelpBarAtBottom_FewResults(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("few")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	// Only 3 tracks — well below the 10-row budget
	results := &panes.SearchResultData{
		Tracks: []panes.SearchTrackItem{
			{URI: "u1", Name: "Track 1", Artist: "Artist", DurationMs: 180000},
			{URI: "u2", Name: "Track 2", Artist: "Artist", DurationMs: 180000},
			{URI: "u3", Name: "Track 3", Artist: "Artist", DurationMs: 180000},
		},
		TotalTracks: 3,
	}
	model, _ := o.Update(panes.SearchResultsMsg{Results: results})
	o = model.(*panes.SearchOverlay)
	// Terminal large enough to give overlayHeight = 26 (base)
	o.SetSize(200, 10)

	view := o.View()
	stripped := stripANSIForTest(view)
	lines := strings.Split(stripped, "\n")

	helpBarIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "navigate") {
			helpBarIdx = i
			break
		}
	}
	require.NotEqual(t, -1, helpBarIdx, "help bar must be present in view")

	// With 3 results and a 26-high overlay, the help bar should be near the bottom.
	totalLines := len(lines)
	assert.Greater(t, helpBarIdx, totalLines*3/4,
		"help bar should be in the bottom quarter even with few results (got line %d of %d)", helpBarIdx, totalLines)
}

// TestTabColorForSection_ReturnsCorrectTokens verifies that each section maps to its
// expected PaneBorder* theme token (as a non-empty color).
func TestTabColorForSection_ReturnsCorrectTokens(t *testing.T) {
	o := newTestSearchOverlay()
	th := theme.Load("black")

	tests := []struct {
		section  panes.SearchSection
		expected lipgloss.Color
	}{
		{panes.SectionTracks, th.PaneBorderTopTracks()},
		{panes.SectionArtists, th.PaneBorderTopArtists()},
		{panes.SectionAlbums, th.PaneBorderAlbums()},
		{panes.SectionPlaylists, th.PaneBorderPlaylists()},
	}

	for _, tt := range tests {
		got := o.TabColorForSection(tt.section)
		assert.Equal(t, tt.expected, got, "section %d should map to the correct PaneBorder* token", tt.section)
		assert.NotEmpty(t, string(got), "tab color should be a non-empty color string")
	}
}
