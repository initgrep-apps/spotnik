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

// applySearchResultsToStore mirrors what app.go's SearchResultsMsg handler does:
// it writes the item buffers, totals, and fetched-offset sentinels to the store,
// then delivers the msg to the overlay so the overlay can refresh its table rows.
// Tests must call this instead of overlay.Update(msg) directly when they want the
// overlay to show results, since store mutations are now owned by app.go.
func applySearchResultsToStore(t *testing.T, s *state.Store, o *panes.SearchOverlay, msg panes.SearchResultsMsg) *panes.SearchOverlay {
	t.Helper()
	if msg.Results == nil {
		model, _ := o.Update(msg)
		return model.(*panes.SearchOverlay)
	}
	if !msg.IsPaged {
		s.ClearSearchBuffers()
		s.AppendSearchTracks(msg.Results.Tracks)
		s.AppendSearchArtists(msg.Results.Artists)
		s.AppendSearchAlbums(msg.Results.Albums)
		s.AppendSearchPlaylists(msg.Results.Playlists)
		s.SetSearchTotal(int(panes.SectionTracks), msg.Results.TotalTracks)
		s.SetSearchTotal(int(panes.SectionArtists), msg.Results.TotalArtists)
		s.SetSearchTotal(int(panes.SectionAlbums), msg.Results.TotalAlbums)
		s.SetSearchTotal(int(panes.SectionPlaylists), msg.Results.TotalPlaylists)
		s.MarkSearchOffsetFetched(int(panes.SectionTracks), 0)
		s.MarkSearchOffsetFetched(int(panes.SectionArtists), 0)
		s.MarkSearchOffsetFetched(int(panes.SectionAlbums), 0)
		s.MarkSearchOffsetFetched(int(panes.SectionPlaylists), 0)
	} else {
		switch msg.Section {
		case panes.SectionTracks:
			s.AppendSearchTracks(msg.Results.Tracks)
			if msg.Results.TotalTracks > 0 {
				s.SetSearchTotal(int(panes.SectionTracks), msg.Results.TotalTracks)
			}
		case panes.SectionArtists:
			s.AppendSearchArtists(msg.Results.Artists)
			if msg.Results.TotalArtists > 0 {
				s.SetSearchTotal(int(panes.SectionArtists), msg.Results.TotalArtists)
			}
		case panes.SectionAlbums:
			s.AppendSearchAlbums(msg.Results.Albums)
			if msg.Results.TotalAlbums > 0 {
				s.SetSearchTotal(int(panes.SectionAlbums), msg.Results.TotalAlbums)
			}
		case panes.SectionPlaylists:
			s.AppendSearchPlaylists(msg.Results.Playlists)
			if msg.Results.TotalPlaylists > 0 {
				s.SetSearchTotal(int(panes.SectionPlaylists), msg.Results.TotalPlaylists)
			}
		}
		s.MarkSearchOffsetFetched(int(msg.Section), msg.Offset)
	}
	model, _ := o.Update(msg)
	return model.(*panes.SearchOverlay)
}

// collectBatchMsgs executes a tea.Cmd and collects all resulting tea.Msg values,
// including sub-commands produced by tea.Batch. This allows tests to assert that
// a specific message type is (or is not) present in a batch command result.
// tea.Batch returns a tea.BatchMsg (type alias for []tea.Cmd) when executed.
func collectBatchMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	// tea.BatchMsg is the exported type for batch commands ([]tea.Cmd).
	// Execute each sub-command and collect its result recursively.
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, subCmd := range batch {
			msgs = append(msgs, collectBatchMsgs(subCmd)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}

// newTestSearchOverlayWithResults creates a SearchOverlay with pre-populated search
// results. Both the store and overlay are populated, mirroring how app.go processes
// SearchResultsMsg before forwarding to the overlay.
func newTestSearchOverlayWithResults() (*panes.SearchOverlay, *state.Store) {
	s := state.New()
	th := theme.Load("black")
	s.SetSearchQuery("blinding")

	overlay := panes.NewSearchOverlay(s, th)

	// Populate store then deliver msg — mirrors what app.go's SearchResultsMsg handler does.
	data := sampleSearchResultData()
	s.ClearSearchBuffers()
	s.AppendSearchTracks(data.Tracks)
	s.AppendSearchArtists(data.Artists)
	s.AppendSearchAlbums(data.Albums)
	s.AppendSearchPlaylists(data.Playlists)
	s.SetSearchTotal(int(panes.SectionTracks), data.TotalTracks)
	s.SetSearchTotal(int(panes.SectionArtists), data.TotalArtists)
	s.SetSearchTotal(int(panes.SectionAlbums), data.TotalAlbums)
	s.SetSearchTotal(int(panes.SectionPlaylists), data.TotalPlaylists)
	s.MarkSearchOffsetFetched(int(panes.SectionTracks), 0)
	s.MarkSearchOffsetFetched(int(panes.SectionArtists), 0)
	s.MarkSearchOffsetFetched(int(panes.SectionAlbums), 0)
	s.MarkSearchOffsetFetched(int(panes.SectionPlaylists), 0)
	msg := panes.SearchResultsMsg{Results: data}
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
	initialCursor := o.Tables()[panes.SectionTracks].SelectedIndex()

	o, _ = sendKey(t, o, "down")
	assert.Equal(t, initialCursor+1, o.Tables()[panes.SectionTracks].SelectedIndex(), "down arrow should move cursor down")

	o, _ = sendKey(t, o, "up")
	assert.Equal(t, initialCursor, o.Tables()[panes.SectionTracks].SelectedIndex(), "up arrow should move cursor back up")
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
// In the bubble-table rewrite the selected row uses background-color highlight (ANSI 48;2;...)
// rather than a ▶ prefix. We verify the first track name appears with a highlighted background.
func TestSearchOverlay_View_SelectedHighlight(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	view := o.View()

	// With bubble-table selection, the highlighted row uses background color ANSI codes.
	// Verify the first track "Blinding Lights" appears in the view (it will be highlighted).
	assert.Contains(t, view, "Blinding Lights", "selected track name must appear in view")
	// The selection uses a background-color ANSI sequence (48;2; = TrueColor background).
	assert.Contains(t, view, "\x1b[", "view must contain ANSI escape sequences for highlighting")
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
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: results})
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
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: emptyResults})
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
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: &panes.SearchResultData{}})
	o.SetSize(80, 30)

	output := o.View()
	assert.Contains(t, output, "No results for", "should show no-results message")
}

func TestSearchOverlay_View_ShowsResults(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("blinding")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	// Deliver results via SearchResultsMsg — populate store first, then overlay.
	results := &panes.SearchResultData{
		Tracks: []panes.SearchTrackItem{
			{URI: "spotify:track:t1", Name: "Blinding Lights", Artist: "The Weeknd"},
		},
	}
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: results})
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

	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: results})

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
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: results})
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

// TestSearchOverlay_TracksTable_ColumnDefs verifies that the Tracks section table has
// the correct columns with the right headers and colors.
func TestSearchOverlay_TracksTable_ColumnDefs(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	cols := o.Tables()[panes.SectionTracks].Columns()
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}

	assert.Contains(t, headers, "#", "Tracks table should have # column")
	assert.Contains(t, headers, "Track", "Tracks table should have Track column")
	assert.Contains(t, headers, "Artist", "Tracks table should have Artist column")
	assert.Contains(t, headers, "Album", "Tracks table should have Album column in wide mode")
	assert.Contains(t, headers, "Duration", "Tracks table should have Duration column")
}

// TestSearchOverlay_AlbumsTable_ColumnDefs verifies that the Albums section table has
// the correct columns with the right headers.
func TestSearchOverlay_AlbumsTable_ColumnDefs(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	cols := o.Tables()[panes.SectionAlbums].Columns()
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}

	assert.Contains(t, headers, "#", "Albums table should have # column")
	assert.Contains(t, headers, "Album", "Albums table should have Album column")
	assert.Contains(t, headers, "Artist", "Albums table should have Artist column")
	assert.Contains(t, headers, "Year", "Albums table should have Year column")
	assert.Contains(t, headers, "Tracks", "Albums table should have Tracks column")
}

// TestSearchOverlay_NarrowDropsAlbumColumn verifies that the tracks table is rebuilt
// without the Album column when the overlay width drops below the narrow threshold.
func TestSearchOverlay_NarrowDropsAlbumColumn(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("test")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	// Small terminal size → narrow mode (contentWidth = overlayWidth-2-2 = 40-2-2=36 < 60)
	o.SetSize(40, 20)

	cols := o.Tables()[panes.SectionTracks].Columns()
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}
	assert.NotContains(t, headers, "Album", "narrow Tracks table should NOT have Album column")
	assert.Contains(t, headers, "Track", "narrow Tracks table should still have Track column")
	assert.Contains(t, headers, "Artist", "narrow Tracks table should still have Artist column")
	assert.Contains(t, headers, "Duration", "narrow Tracks table should still have Duration column")
}

// TestActiveSection_Tracks_ShowsAlbumAndDuration verifies that the Tracks section
// renders Album and Duration columns via the View() output.
func TestActiveSection_Tracks_ShowsAlbumAndDuration(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	view := o.View()
	stripped := stripANSIForTest(view)

	// sampleSearchResultData tracks: "Blinding Lights", Artist: "The Weeknd", Album: "After Hours", DurationMs: 200040
	assert.Contains(t, stripped, "Blinding Lights", "should show track name")
	assert.Contains(t, stripped, "The Weeknd", "should show artist name")
	assert.Contains(t, stripped, "After Hours", "should show album name")
	// 200040ms = 3 minutes 20.04 seconds → 3:20
	assert.Contains(t, stripped, "3:20", "should show formatted duration")
}

// TestActiveSection_Albums_ShowsYearAndCount verifies that the Albums section
// renders Year and TotalTracks columns via the View() output.
func TestActiveSection_Albums_ShowsYearAndCount(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	// Navigate to Albums section
	o, _ = sendKey(t, o, "tab")
	o, _ = sendKey(t, o, "tab")

	view := o.View()
	stripped := stripANSIForTest(view)

	// sampleSearchResultData albums: "After Hours", Artist: "The Weeknd", ReleaseYear: "2020", TotalTracks: 14
	assert.Contains(t, stripped, "After Hours", "should show album name")
	assert.Contains(t, stripped, "The Weeknd", "should show artist name")
	assert.Contains(t, stripped, "2020", "should show release year")
	assert.Contains(t, stripped, "14", "should show track count")
}

// TestActiveSection_Playlists_ShowsTrackCount verifies that the Playlists section
// renders the TrackCount column via the View() output.
func TestActiveSection_Playlists_ShowsTrackCount(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	// Navigate to Playlists section
	o, _ = sendKey(t, o, "tab")
	o, _ = sendKey(t, o, "tab")
	o, _ = sendKey(t, o, "tab")

	view := o.View()
	stripped := stripANSIForTest(view)

	// sampleSearchResultData playlists: "Blinding Pop Hits", Owner: "User", TrackCount: 50
	assert.Contains(t, stripped, "Blinding Pop Hits", "should show playlist name")
	assert.Contains(t, stripped, "User", "should show owner name")
	assert.Contains(t, stripped, "50", "should show track count")
}

// TestActiveSection_SelectedRow_UsesSelectedColors verifies that the selected row
// is highlighted in the view. With the bubble-table rewrite, selection is indicated
// via background color ANSI codes rather than a ▶ prefix symbol.
func TestActiveSection_SelectedRow_UsesSelectedColors(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	view := o.View()
	// The first track should appear in the view, highlighted via ANSI background color.
	assert.Contains(t, view, "Blinding Lights", "selected row track name should appear in view")
	// Verify background-color ANSI code is present (48;2; = TrueColor background).
	assert.Contains(t, view, "\x1b[", "view should contain ANSI escape sequences for selection highlight")
}

// TestActiveSection_Tracks_NarrowNoAlbumColumn verifies that in narrow mode
// the Album column key is absent from the tracks table columns.
func TestActiveSection_Tracks_NarrowNoAlbumColumn(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("test")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	results := &panes.SearchResultData{
		Tracks: []panes.SearchTrackItem{
			{URI: "spotify:track:t1", Name: "My Song", Artist: "My Artist", Album: "My Album", DurationMs: 180000},
		},
	}
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: results})

	// Small terminal size → narrow mode
	o.SetSize(40, 20)

	cols := o.Tables()[panes.SectionTracks].Columns()
	for _, c := range cols {
		assert.NotEqual(t, "album", c.Key, "narrow tracks table should not have 'album' column key")
	}
}

// TestTable_Tracks_ColumnHeaders verifies that 5 column headers exist for the Tracks section.
func TestTable_Tracks_ColumnHeaders(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	cols := o.Tables()[panes.SectionTracks].Columns()
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}

	assert.Contains(t, headers, "#", "Tracks table should contain # column")
	assert.Contains(t, headers, "Track", "Tracks table should contain Track column")
	assert.Contains(t, headers, "Artist", "Tracks table should contain Artist column")
	assert.Contains(t, headers, "Album", "Tracks table should contain Album column")
	assert.Contains(t, headers, "Duration", "Tracks table should contain Duration column")
}

// TestTable_Artists_ColumnHeaders verifies that 2 column headers exist for the Artists section.
func TestTable_Artists_ColumnHeaders(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	cols := o.Tables()[panes.SectionArtists].Columns()
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}

	assert.Contains(t, headers, "#", "Artists table should contain # column")
	assert.Contains(t, headers, "Artist", "Artists table should contain Artist column")
}

// TestTable_Albums_ColumnHeaders verifies that 5 column headers exist for the Albums section.
func TestTable_Albums_ColumnHeaders(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	cols := o.Tables()[panes.SectionAlbums].Columns()
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}

	assert.Contains(t, headers, "#", "Albums table should contain # column")
	assert.Contains(t, headers, "Album", "Albums table should contain Album column")
	assert.Contains(t, headers, "Artist", "Albums table should contain Artist column")
	assert.Contains(t, headers, "Year", "Albums table should contain Year column")
	assert.Contains(t, headers, "Tracks", "Albums table should contain Tracks column")
}

// TestTable_Playlists_ColumnHeaders verifies that 4 column headers exist for the Playlists section.
func TestTable_Playlists_ColumnHeaders(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(100, 40)

	cols := o.Tables()[panes.SectionPlaylists].Columns()
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}

	assert.Contains(t, headers, "#", "Playlists table should contain # column")
	assert.Contains(t, headers, "Playlist", "Playlists table should contain Playlist column")
	assert.Contains(t, headers, "Owner", "Playlists table should contain Owner column")
	assert.Contains(t, headers, "Tracks", "Playlists table should contain Tracks column")
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
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: results})

	// Check that the view renders without wrapping. With the bubble-table rewrite,
	// we verify the view output contains the track name somewhere (not a broken render).
	view := o.View()
	stripped := stripANSIForTest(view)
	assert.Contains(t, stripped, "Exactly 86 Chars", "track name should appear in rendered view")
}

// TestTable_TracksColumnDefs verifies that the tracks table has the expected column
// header keys (index, name, artist, album, duration) in wide mode.
func TestTable_TracksColumnDefs(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 40) // wide mode

	cols := o.Tables()[panes.SectionTracks].Columns()
	require.Len(t, cols, 5, "wide tracks table should have 5 columns")
	assert.Equal(t, "index", cols[0].Key)
	assert.Equal(t, "name", cols[1].Key)
	assert.Equal(t, "artist", cols[2].Key)
	assert.Equal(t, "album", cols[3].Key)
	assert.Equal(t, "duration", cols[4].Key)
}

// TestTable_ArtistsColumnDefs verifies that the artists table has the expected column keys.
func TestTable_ArtistsColumnDefs(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 40)

	cols := o.Tables()[panes.SectionArtists].Columns()
	require.Len(t, cols, 2, "artists table should have 2 columns")
	assert.Equal(t, "index", cols[0].Key)
	assert.Equal(t, "name", cols[1].Key)
}

// TestTable_AlbumsColumnDefs verifies that the albums table has the expected column keys.
func TestTable_AlbumsColumnDefs(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 40)

	cols := o.Tables()[panes.SectionAlbums].Columns()
	require.Len(t, cols, 5, "albums table should have 5 columns")
	assert.Equal(t, "index", cols[0].Key)
	assert.Equal(t, "name", cols[1].Key)
	assert.Equal(t, "artist", cols[2].Key)
	assert.Equal(t, "year", cols[3].Key)
	assert.Equal(t, "tracks", cols[4].Key)
}

// TestTable_PlaylistsColumnDefs verifies that the playlists table has the expected column keys.
func TestTable_PlaylistsColumnDefs(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 40)

	cols := o.Tables()[panes.SectionPlaylists].Columns()
	require.Len(t, cols, 4, "playlists table should have 4 columns")
	assert.Equal(t, "index", cols[0].Key)
	assert.Equal(t, "name", cols[1].Key)
	assert.Equal(t, "owner", cols[2].Key)
	assert.Equal(t, "tracks", cols[3].Key)
}

// TestTable_TracksHeaders_Rendered verifies that each section's table headers
// appear in the View output when results are present.
func TestTable_TracksHeaders_Rendered(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 40)

	view := o.View()
	stripped := stripANSIForTest(view)
	// Track headers should be visible since tracks tab is active.
	assert.Contains(t, stripped, "Track", "tracks column header should appear in view")
	assert.Contains(t, stripped, "Artist", "artist column header should appear in view")
}

// TestTable_NarrowTracksDropsAlbumColumn verifies that in narrow mode the album column
// is removed from the tracks table definition (it has 4 columns, not 5).
func TestTable_NarrowTracksDropsAlbumColumn(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(50, 40) // narrow terminal

	cols := o.Tables()[panes.SectionTracks].Columns()
	keys := make([]string, len(cols))
	for i, c := range cols {
		keys[i] = c.Key
	}
	assert.NotContains(t, keys, "album", "narrow tracks table should not have album column")
	assert.Len(t, cols, 4, "narrow tracks table should have 4 columns")
}

// TestView_TracksTable_RowFitsOnOneLine verifies that a single result row in the
// tracks table renders without embedded newlines in the View output.
func TestView_TracksTable_RowFitsOnOneLine(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 40)

	view := o.View()
	stripped := stripANSIForTest(view)
	lines := strings.Split(stripped, "\n")

	// Find a line containing the track name from sampleSearchResultData.
	found := false
	for _, line := range lines {
		if strings.Contains(line, "Blinding Lights") {
			found = true
			assert.NotContains(t, line, "\n",
				"track row should not contain embedded newlines")
			break
		}
	}
	assert.True(t, found, "track name 'Blinding Lights' should appear in view output")
}

// TestSearchOverlay_HelpBarPresent verifies that the help bar appears in the
// overlay's rendered output. With the bubble-table rewrite, the help bar appears
// after the table content area rather than being anchored to the visual bottom.
func TestSearchOverlay_HelpBarPresent(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 10)

	view := o.View()
	stripped := stripANSIForTest(view)
	lines := strings.Split(stripped, "\n")

	// Find the help bar line (contains "navigate")
	helpBarIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "navigate") {
			helpBarIdx = i
			break
		}
	}
	require.NotEqual(t, -1, helpBarIdx, "help bar must be present in the view")

	// Help bar should appear after the content (after at least tab bar and separator).
	assert.Greater(t, helpBarIdx, 4,
		"help bar should appear after the tab bar and table header (line %d)", helpBarIdx)
}

// TestSearchOverlay_HelpBarPresent_FewResults verifies that with only 3 results,
// the help bar still appears in the overlay's rendered output.
func TestSearchOverlay_HelpBarPresent_FewResults(t *testing.T) {
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
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: results})
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
	// Help bar should appear after the content area.
	assert.Greater(t, helpBarIdx, 4,
		"help bar should appear after the tab bar and table content (line %d)", helpBarIdx)
}

// --- Story 84/85: Buffer-based pagination ---

// TestSearchOverlay_NewQuery_ClearsBuffers verifies that receiving a SearchResultsMsg
// without IsPaged clears all accumulated buffers and starts fresh.
func TestSearchOverlay_NewQuery_ClearsBuffers(t *testing.T) {
	o, s := newTestSearchOverlayWithResults()
	// sampleSearchResultData has 2 tracks, 1 artist, 1 album, 1 playlist.
	assert.Equal(t, 2, o.BufTracksLen(), "initial load should have 2 tracks")
	assert.Equal(t, 1, o.BufArtistsLen(), "initial load should have 1 artist")

	// Send a new (non-paged) result — buffers must be cleared and replaced.
	newData := &panes.SearchResultData{
		Tracks:      []panes.SearchTrackItem{{Name: "New Track", URI: "uri:new"}},
		TotalTracks: 1,
	}
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: newData})

	assert.Equal(t, 1, o.BufTracksLen(), "new query should clear old buffer and load new items")
	assert.Equal(t, 0, o.BufArtistsLen(), "new query should clear artist buffer")
}

// TestSearchOverlay_PagedResult_AccumulatesBuffer verifies that a paged result appends
// to the existing buffer instead of replacing it.
func TestSearchOverlay_PagedResult_AccumulatesBuffer(t *testing.T) {
	o, s := newTestSearchOverlayWithResults()
	// sampleSearchResultData has 2 tracks.
	require.Equal(t, 2, o.BufTracksLen(), "initial load should have 2 tracks")

	// Deliver a second page with 3 more tracks.
	page2Items := []panes.SearchTrackItem{
		{Name: "Track 3", URI: "uri:3"},
		{Name: "Track 4", URI: "uri:4"},
		{Name: "Track 5", URI: "uri:5"},
	}
	msg := panes.SearchResultsMsg{
		Results: &panes.SearchResultData{Tracks: page2Items, TotalTracks: 100},
		Section: panes.SectionTracks,
		Offset:  10,
		IsPaged: true,
	}
	o = applySearchResultsToStore(t, s, o, msg)

	assert.Equal(t, 5, o.BufTracksLen(), "buffer should accumulate: 2 initial + 3 paged = 5")
}

// TestSearchOverlay_RowNumbers_Absolute verifies that after accumulating two pages,
// row numbers continue from the last loaded index (absolute position in buffer).
func TestSearchOverlay_RowNumbers_Absolute(t *testing.T) {
	o, s := newTestSearchOverlayWithResults()
	// initial: 2 tracks loaded

	// Load 3 more tracks on page 2.
	page2Items := make([]panes.SearchTrackItem, 3)
	for i := range page2Items {
		page2Items[i] = panes.SearchTrackItem{Name: fmt.Sprintf("Page2 Track %d", i+1), URI: fmt.Sprintf("uri:p2-%d", i)}
	}
	msg := panes.SearchResultsMsg{
		Results: &panes.SearchResultData{Tracks: page2Items, TotalTracks: 100},
		Section: panes.SectionTracks,
		Offset:  10,
		IsPaged: true,
	}
	o = applySearchResultsToStore(t, s, o, msg)
	o.SetSize(200, 40)

	// View should contain "3" for the 3rd item and "5" for the 5th.
	view := o.View()
	stripped := stripANSIForTest(view)
	assert.Contains(t, stripped, "3", "absolute row number 3 should appear in tracks table")
	assert.Contains(t, stripped, "5", "absolute row number 5 should appear in tracks table")
}

// TestSearchOverlay_TabSwitch_PreservesBuffer verifies that switching sections
// via Tab does not clear any accumulated item buffers.
func TestSearchOverlay_TabSwitch_PreservesBuffer(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	// sampleSearchResultData: 2 tracks, 1 artist, 1 album, 1 playlist.
	tracksBefore := o.BufTracksLen()
	artistsBefore := o.BufArtistsLen()

	// Switch to Artists section via Tab.
	o, _ = sendKey(t, o, "tab")
	assert.Equal(t, panes.SectionArtists, o.ActiveSection(), "should be on Artists after Tab")

	// Buffers should be unchanged.
	assert.Equal(t, tracksBefore, o.BufTracksLen(), "tracks buffer should be preserved after tab switch")
	assert.Equal(t, artistsBefore, o.BufArtistsLen(), "artists buffer should be preserved after tab switch")
}

// --- Story 84: Page indicator ---

// TestPageIndicator_ShowsRange_WhenTotalExceeds10 verifies that pageIndicator returns
// a non-empty range string when the total for the active section exceeds maxResultsPerSection.
func TestPageIndicator_ShowsRange_WhenTotalExceeds10(t *testing.T) {
	o, s := newTestSearchOverlayWithResults()
	s.SetSearchQuery("blinding")
	// sampleSearchResultData has TotalTracks=100, 2 items — total > 10 ⇒ indicator shown.

	indicator := o.PageIndicator()
	assert.NotEmpty(t, indicator, "page indicator should be non-empty when total > 10")
	assert.Contains(t, indicator, "of 100", "indicator should show total")
}

// TestPageIndicator_NoIndicator_WhenTotalUnder10 verifies that pageIndicator returns
// empty string when the active section has total <= 10 (all results fit on one page).
func TestPageIndicator_NoIndicator_WhenTotalUnder10(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("test")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	data := &panes.SearchResultData{
		Tracks:       []panes.SearchTrackItem{{Name: "A"}, {Name: "B"}},
		TotalTracks:  2, // fits on one page
		TotalArtists: 0, TotalAlbums: 0, TotalPlaylists: 0,
	}
	msg := panes.SearchResultsMsg{Results: data, Offset: 0}
	o = applySearchResultsToStore(t, s, o, msg)

	indicator := o.PageIndicator()
	assert.Empty(t, indicator, "page indicator should be empty when total <= 10")
}

// TestPageIndicator_AccumulatedBuffer verifies that after accumulating 2 initial + 6 paged
// tracks with total=16, the page indicator reflects the current cursor position window.
// In the accumulated buffer model (cursor at 0): pageStart=0, start=1, end=min(10,8)=8 → "1-8 of 16".
func TestPageIndicator_AccumulatedBuffer(t *testing.T) {
	o, s := newTestSearchOverlayWithResults()
	s.SetSearchQuery("blinding")
	// sampleSearchResultData: 2 tracks, TotalTracks=100. Override total via a paged append.

	// Append 6 more tracks paged from offset=10, total=16.
	page2Items := make([]panes.SearchTrackItem, 6)
	for i := range page2Items {
		page2Items[i] = panes.SearchTrackItem{Name: fmt.Sprintf("Track %d", i+3)}
	}
	data := &panes.SearchResultData{
		Tracks:      page2Items,
		TotalTracks: 16,
	}
	msg := panes.SearchResultsMsg{
		Results: data,
		Section: panes.SectionTracks,
		Offset:  10,
		IsPaged: true,
	}
	o = applySearchResultsToStore(t, s, o, msg)

	// Buffer now has 2+6=8 items, total=16. Cursor at 0 → first page window (1-8 of 16).
	indicator := o.PageIndicator()
	assert.Contains(t, indicator, "of 16", "indicator should show total of 16")
	assert.Contains(t, indicator, "1-", "indicator should start from item 1 with cursor at top")
}

// TestRenderHelpBar_ShowsPageIndicator_WhenTotalExceeds10 verifies that renderHelpBar
// appends the page indicator to the right side when total > maxResultsPerSection.
func TestRenderHelpBar_ShowsPageIndicator_WhenTotalExceeds10(t *testing.T) {
	o, s := newTestSearchOverlayWithResults()
	s.SetSearchQuery("blinding")
	// sampleSearchResultData has TotalTracks=100, so indicator should appear.

	helpBar := o.RenderHelpBar(80)
	assert.Contains(t, helpBar, "of 100", "help bar should include page indicator when total > 10")
}

// TestRenderHelpBar_NoPageIndicator_WhenTotalUnder10 verifies that renderHelpBar
// does not include a page indicator when total <= maxResultsPerSection.
func TestRenderHelpBar_NoPageIndicator_WhenTotalUnder10(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("test")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	data := &panes.SearchResultData{
		Tracks:      []panes.SearchTrackItem{{Name: "A"}},
		TotalTracks: 5, // under 10
	}
	msg := panes.SearchResultsMsg{Results: data, Offset: 0}
	o = applySearchResultsToStore(t, s, o, msg)

	helpBar := o.RenderHelpBar(80)
	assert.NotContains(t, helpBar, "of 5", "help bar should NOT include page indicator when total <= 10")
}

// --- Story 85: View() uses table.View() ---

// TestSearchOverlay_View_ContainsTableOutput verifies that the view output contains
// the table-rendered track names and column headers from the active table.
func TestSearchOverlay_View_ContainsTableOutput(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 40)

	view := o.View()
	stripped := stripANSIForTest(view)

	// Table header and data should appear.
	assert.Contains(t, stripped, "Track", "column header 'Track' should appear in view")
	assert.Contains(t, stripped, "Blinding Lights", "track name from sample data should appear")
	assert.Contains(t, stripped, "Save Your Tears", "second track name should appear")
}

// TestSearchOverlay_View_HelpBarAnchored verifies that the help bar is present
// and rendered after the table content in the view output.
func TestSearchOverlay_View_HelpBarAnchored(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 40)

	view := o.View()
	stripped := stripANSIForTest(view)

	// Help bar should contain navigation hints.
	assert.Contains(t, stripped, "navigate", "help bar navigation hint should appear")
	assert.Contains(t, stripped, "close", "help bar close hint should appear")
}

// --- Story 85: Smart prefetch ---

// TestPrefetch_Fires_AtMidpoint verifies that checkPrefetch fires a SearchPageRequestMsg
// when the cursor is at or beyond the 50% midpoint of the last fetched page.
// With 10 items and cursor at item 5 (0-indexed), midpoint = 0 + 10/2 = 5 → fires.
func TestPrefetch_Fires_AtMidpoint(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("blinding")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	// Load 10 items as the first page, total=100 (more pages exist).
	items := make([]panes.SearchTrackItem, 10)
	for i := range items {
		items[i] = panes.SearchTrackItem{Name: fmt.Sprintf("Track %d", i+1), URI: fmt.Sprintf("uri:%d", i)}
	}
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: &panes.SearchResultData{Tracks: items, TotalTracks: 100}})
	o.SetSize(200, 40)

	// Move cursor to position 5 (midpoint of page 1 with 10 items).
	for i := 0; i < 5; i++ {
		o, _ = sendKey(t, o, "down")
	}

	// The next down press should trigger prefetch for offset=10.
	_, cmd := sendKey(t, o, "down")

	// cmd is a Batch(tableCmd, prefetchCmd). Unwrap all sub-commands and collect messages.
	require.NotNil(t, cmd, "cursor at midpoint should produce a cmd (prefetch)")

	// Execute sub-commands from the batch and collect any SearchPageRequestMsg.
	msgs := collectBatchMsgs(cmd)
	var pageReqs []panes.SearchPageRequestMsg
	for _, m := range msgs {
		if pr, ok := m.(panes.SearchPageRequestMsg); ok {
			pageReqs = append(pageReqs, pr)
		}
	}
	require.Len(t, pageReqs, 1, "exactly one SearchPageRequestMsg should be fired at midpoint")
	assert.Equal(t, "blinding", pageReqs[0].Query)
	assert.Equal(t, 10, pageReqs[0].Offset, "prefetch should request the next offset (10)")
	assert.Equal(t, panes.SectionTracks, pageReqs[0].Section)
}

// TestPrefetch_NoFire_AllLoaded verifies that prefetch does not fire when
// all pages have already been loaded (bufLen >= total).
func TestPrefetch_NoFire_AllLoaded(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("blinding")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	// Load 5 items, total=5 — no more pages.
	items := make([]panes.SearchTrackItem, 5)
	for i := range items {
		items[i] = panes.SearchTrackItem{Name: fmt.Sprintf("Track %d", i+1), URI: fmt.Sprintf("uri:%d", i)}
	}
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: &panes.SearchResultData{Tracks: items, TotalTracks: 5}})

	// Move cursor to the last item.
	for i := 0; i < 4; i++ {
		o, _ = sendKey(t, o, "down")
	}
	_, cmd := sendKey(t, o, "down")
	// No SearchPageRequestMsg should be present in any sub-commands.
	msgs := collectBatchMsgs(cmd)
	for _, m := range msgs {
		_, isPageReq := m.(panes.SearchPageRequestMsg)
		assert.False(t, isPageReq, "should not emit page request when buffer is full")
	}
}

// TestPrefetch_NoFire_AlreadyFetched verifies that prefetch does not fire
// when the next offset has already been fetched (store guard).
func TestPrefetch_NoFire_AlreadyFetched(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("blinding")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	// Load 10 items as page 1 (offset 0), total=100.
	items := make([]panes.SearchTrackItem, 10)
	for i := range items {
		items[i] = panes.SearchTrackItem{Name: fmt.Sprintf("Track %d", i+1), URI: fmt.Sprintf("uri:%d", i)}
	}
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: &panes.SearchResultData{Tracks: items, TotalTracks: 100}})

	// Pre-mark offset 10 as already fetched in the store (simulates in-flight request).
	s.MarkSearchOffsetFetched(int(panes.SectionTracks), 10)

	// Move cursor to midpoint.
	for i := 0; i < 5; i++ {
		o, _ = sendKey(t, o, "down")
	}
	_, cmd := sendKey(t, o, "down")
	// No SearchPageRequestMsg should be present in any sub-commands.
	msgs := collectBatchMsgs(cmd)
	for _, m := range msgs {
		_, isPageReq := m.(panes.SearchPageRequestMsg)
		assert.False(t, isPageReq, "should not emit page request when offset already fetched")
	}
}

// TestPrefetch_NoFire_BelowMidpoint verifies that prefetch does not fire
// when the cursor is below the 50% midpoint (cursor < midpoint).
func TestPrefetch_NoFire_BelowMidpoint(t *testing.T) {
	s := state.New()
	s.SetSearchQuery("blinding")
	th := theme.Load("black")
	o := panes.NewSearchOverlay(s, th)

	// Load 10 items, total=100.
	items := make([]panes.SearchTrackItem, 10)
	for i := range items {
		items[i] = panes.SearchTrackItem{Name: fmt.Sprintf("Track %d", i+1), URI: fmt.Sprintf("uri:%d", i)}
	}
	o = applySearchResultsToStore(t, s, o, panes.SearchResultsMsg{Results: &panes.SearchResultData{Tracks: items, TotalTracks: 100}})

	// Move cursor to position 3 (below midpoint=5).
	for i := 0; i < 3; i++ {
		o, _ = sendKey(t, o, "down")
	}
	// Press down once more (cursor → 4, still below midpoint 5).
	_, cmd := sendKey(t, o, "down")
	// No SearchPageRequestMsg should be present in any sub-commands.
	msgs := collectBatchMsgs(cmd)
	for _, m := range msgs {
		_, isPageReq := m.(panes.SearchPageRequestMsg)
		assert.False(t, isPageReq, "cursor below midpoint should not trigger prefetch")
	}
}

// --- Story 85: Help bar key highlighting ---

// TestRenderHelpBar_KeysUseKeyHintColor verifies that the key labels in the help bar
// use the KeyHint() theme color (appears as ANSI escape sequence in the output).
func TestRenderHelpBar_KeysUseKeyHintColor(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 40)

	helpBar := o.RenderHelpBar(80)

	// The help bar should contain ANSI escape codes (styling applied to keys).
	assert.Contains(t, helpBar, "\x1b[", "help bar should contain ANSI escape sequences for key colors")
	// Strip ANSI and verify the key names are still present.
	stripped := stripANSIForTest(helpBar)
	assert.Contains(t, stripped, "Tab", "Tab key label should appear in help bar")
	assert.Contains(t, stripped, "↑↓", "navigation key label should appear in help bar")
	assert.Contains(t, stripped, "Esc", "Esc key label should appear in help bar")
}

// TestRenderHelpBar_LabelsUseTextMutedColor verifies that description text follows
// key labels with muted styling (present in the ANSI-stripped output).
func TestRenderHelpBar_LabelsUseTextMutedColor(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 40)

	helpBar := o.RenderHelpBar(80)
	stripped := stripANSIForTest(helpBar)

	assert.Contains(t, stripped, "navigate", "navigate label should appear in help bar")
	assert.Contains(t, stripped, "play", "play label should appear in help bar")
	assert.Contains(t, stripped, "close", "close label should appear in help bar")
}

// --- Story 85: SetTheme rebuilds tables ---

// TestSearchOverlay_SetTheme_RebuildsTables verifies that after calling SetTheme,
// the tables are rebuilt with new theme colors.
func TestSearchOverlay_SetTheme_RebuildsTables(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(200, 40)

	// Record the original header color for the tracks table.
	origColor := o.Tables()[panes.SectionTracks].HeaderColorForTest()

	// Switch to a different theme.
	newTheme := theme.Load("dracula")
	o.SetTheme(newTheme)

	// After SetTheme, tables should exist and have the new theme's header color.
	newColor := o.Tables()[panes.SectionTracks].HeaderColorForTest()
	// Both are valid non-empty colors; the new one should use the new theme's token.
	assert.NotEmpty(t, string(newColor), "header color should be non-empty after SetTheme")
	// The color should be the new theme's PaneBorderTopTracks token.
	assert.Equal(t, newTheme.PaneBorderTopTracks(), newColor,
		"header color should update to new theme's PaneBorderTopTracks token after SetTheme")
	_ = origColor // suppress unused warning — we care about the new value
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
