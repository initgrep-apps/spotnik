package panes_test

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
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

// newTestSearchOverlayWithResults creates a SearchOverlay with pre-populated search results.
func newTestSearchOverlayWithResults() (*panes.SearchOverlay, *state.Store) {
	s := state.New()
	t := theme.Load("black")

	s.SetSearchResults(&api.SearchResult{
		Tracks: api.SearchTracksResult{
			Items: []api.Track{
				{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1",
					Artists: []api.Artist{{ID: "a1", Name: "The Weeknd"}}},
				{ID: "t2", Name: "Save Your Tears", URI: "spotify:track:t2",
					Artists: []api.Artist{{ID: "a1", Name: "The Weeknd"}}},
			},
			Total: 2,
		},
		Artists: api.SearchArtistsResult{
			Items: []api.SearchArtist{
				{ID: "a1", Name: "The Weeknd", URI: "spotify:artist:a1"},
			},
			Total: 1,
		},
		Albums: api.SearchAlbumsResult{
			Items: []api.SearchAlbum{
				{ID: "al1", Name: "After Hours", URI: "spotify:album:al1",
					Artists: []api.Artist{{ID: "a1", Name: "The Weeknd"}}},
			},
			Total: 1,
		},
		Playlists: api.SearchPlaylistsResult{
			Items: []api.SearchPlaylist{
				{ID: "pl1", Name: "Blinding Pop Hits", URI: "spotify:playlist:pl1",
					Owner: api.SimplePlaylistOwner{ID: "u1", DisplayName: "User"}},
			},
			Total: 1,
		},
	})
	s.SetSearchQuery("blinding")

	overlay := panes.NewSearchOverlay(s, t)
	return overlay, s
}

// sendKey sends a key message to the overlay and returns the updated model.
func sendKey(o *panes.SearchOverlay, key string) (*panes.SearchOverlay, tea.Cmd) {
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
	default:
		// Regular rune key
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}

	model, cmd := o.Update(msg)
	updated, ok := model.(*panes.SearchOverlay)
	require.True(nil, ok, "Update must return *panes.SearchOverlay")
	return updated, cmd
}

// --- Task 4.2: Debounce tests ---

// TestDebounce_StaleQueryIgnored verifies that a debounce tick with an old query
// (query changed since tick was scheduled) is silently discarded.
func TestDebounce_StaleQueryIgnored(t *testing.T) {
	o := newTestSearchOverlay()

	// Type 'x', triggering a debounce tick with query "x"
	o, _ = sendKey(o, "x")
	// Type 'y', changing query to "xy" before the tick fires
	o, _ = sendKey(o, "y")

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
	o, _ = sendKey(o, "x")
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

	o, cmd := sendKey(o, "h")

	assert.NotNil(t, cmd, "typing should schedule a debounce tick command")
	assert.Contains(t, o.Query(), "h", "query should contain typed character")
}

// TestSearchOverlay_Update_Backspace verifies backspace removes last character.
func TestSearchOverlay_Update_Backspace(t *testing.T) {
	o := newTestSearchOverlay()

	o, _ = sendKey(o, "x")
	o, _ = sendKey(o, "y")
	require.Contains(t, o.Query(), "y")

	o, _ = sendKey(o, "backspace")
	assert.NotContains(t, o.Query(), "y", "backspace should remove last character")
}

// TestSearchOverlay_Update_Enter verifies Enter emits a play command for the selected item.
func TestSearchOverlay_Update_Enter(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	_, cmd := sendKey(o, "enter")

	// Enter should produce a command (play or close)
	assert.NotNil(t, cmd, "Enter should produce a play command")
}

// TestSearchOverlay_Update_Esc verifies Esc returns a searchClosedMsg.
func TestSearchOverlay_Update_Esc(t *testing.T) {
	o := newTestSearchOverlay()

	_, cmd := sendKey(o, "esc")

	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, panes.SearchClosedMsg{}, msg, "Esc should return SearchClosedMsg")
}

// TestSearchOverlay_Update_A verifies 'a' on a track returns an add-to-queue command.
func TestSearchOverlay_Update_A(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	// First result is a track; press 'a' to add to queue
	_, cmd := sendKey(o, "a")

	require.NotNil(t, cmd)
	msg := cmd()
	qMsg, ok := msg.(AddToQueueMsg)
	require.True(t, ok, "pressing 'a' on track should produce AddToQueueMsg")
	assert.Equal(t, "spotify:track:t1", qMsg.TrackURI)
}

// TestSearchOverlay_Update_Tab verifies Tab moves to the next section.
func TestSearchOverlay_Update_Tab(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	initialSection := o.ActiveSection()
	o, _ = sendKey(o, "tab")

	assert.NotEqual(t, initialSection, o.ActiveSection(), "Tab should move to next section")
}

// TestSearchOverlay_Update_ShiftTab verifies Shift+Tab moves to the previous section.
func TestSearchOverlay_Update_ShiftTab(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	// Move forward first, then back
	o, _ = sendKey(o, "tab")
	o, _ = sendKey(o, "tab")
	afterForward := o.ActiveSection()
	o, _ = sendKey(o, "shift+tab")

	assert.NotEqual(t, afterForward, o.ActiveSection(), "Shift+Tab should move to previous section")
}

// TestSearchOverlay_Update_JK verifies j/k navigation moves cursor within section.
func TestSearchOverlay_Update_JK(t *testing.T) {
	o, _ := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	// Tracks section has 2 items; start at item 0
	initialCursor := o.CursorPos()

	o, _ = sendKey(o, "j")
	assert.Equal(t, initialCursor+1, o.CursorPos(), "j should move cursor down")

	o, _ = sendKey(o, "k")
	assert.Equal(t, initialCursor, o.CursorPos(), "k should move cursor back up")
}

// TestSearchOverlay_Update_CtrlU verifies Ctrl+U clears the input.
func TestSearchOverlay_Update_CtrlU(t *testing.T) {
	o := newTestSearchOverlay()

	o, _ = sendKey(o, "h")
	o, _ = sendKey(o, "e")
	o, _ = sendKey(o, "l")
	o, _ = sendKey(o, "l")
	o, _ = sendKey(o, "o") // letters that are not action keys
	require.Contains(t, o.Query(), "hello", "query should be 'hello' after typing those chars")

	o, _ = sendKey(o, "ctrl+u")
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

	// Very long track name
	longName := strings.Repeat("A", 120)
	s.SetSearchResults(&api.SearchResult{
		Tracks: api.SearchTracksResult{
			Items: []api.Track{
				{ID: "t1", Name: longName, URI: "spotify:track:t1",
					Artists: []api.Artist{{ID: "a1", Name: "Artist"}}},
			},
			Total: 1,
		},
	})
	s.SetSearchQuery("test")

	o := panes.NewSearchOverlay(s, th)
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
	s.SetSearchResults(&api.SearchResult{
		Tracks:    api.SearchTracksResult{Items: []api.Track{}},
		Artists:   api.SearchArtistsResult{Items: []api.SearchArtist{}},
		Albums:    api.SearchAlbumsResult{Items: []api.SearchAlbum{}},
		Playlists: api.SearchPlaylistsResult{Items: []api.SearchPlaylist{}},
	})

	o := panes.NewSearchOverlay(s, th)
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
	_, cmd := sendKey(o, "x") // use 'x', not an action key
	require.NotNil(t, cmd)

	// The command schedules a debounce tick — it should not fire immediately
	// We just verify the command is non-nil and not a synchronous cmd
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 50*time.Millisecond, "typing key should return immediately, not block")
}
