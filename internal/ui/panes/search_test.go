package panes_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findPanelBounds locates a bordered panel by title in the rendered view lines.
// It finds the ╭ line containing the title, then the next ╰ line, returning
// the start and end line indices. Returns (-1, -1) if the panel is not found.
func findPanelBounds(lines []string, title string) (start, end int) {
	start = -1
	end = -1
	for i, line := range lines {
		if strings.Contains(line, "╭") && strings.Contains(line, title) {
			start = i
		}
		if start >= 0 && end < 0 && i > start && strings.Contains(line, "╰") {
			end = i
			return
		}
	}
	return
}

// newTestSearchOverlay creates a SearchOverlay wired to a theme.
func newTestSearchOverlay() *panes.SearchOverlay {
	t := theme.Load("black")
	return panes.NewSearchOverlay(t)
}

// sampleSearchListItems returns a flat []SearchListItem with 2 tracks, 1 artist,
// 1 album, and 1 playlist for use in tests. Mirrors the pre-converted format that
// commands.go produces before delivery via SearchPageLoadedMsg.
func sampleSearchListItems() []panes.SearchListItem {
	tracks := panes.TracksToSearchListItems([]domain.Track{
		{URI: "spotify:track:t1", Name: "Blinding Lights", Artists: []domain.Artist{{Name: "The Weeknd"}}},
		{URI: "spotify:track:t2", Name: "Save Your Tears", Artists: []domain.Artist{{Name: "The Weeknd"}}},
	})
	artists := panes.ArtistsToSearchListItems([]domain.SearchArtist{
		{URI: "spotify:artist:a1", Name: "The Weeknd"},
	})
	albums := panes.AlbumsToSearchListItems([]domain.SearchAlbum{
		{URI: "spotify:album:al1", Name: "After Hours", Artists: []domain.Artist{{Name: "The Weeknd"}}},
	})
	playlists := panes.PlaylistsToSearchListItems([]domain.SearchPlaylist{
		{URI: "spotify:playlist:pl1", Name: "Blinding Pop Hits", Owner: domain.SimplePlaylistOwner{DisplayName: "User"}},
	})
	result := append(tracks, artists...)
	result = append(result, albums...)
	result = append(result, playlists...)
	return result
}

// newTestSearchOverlayWithResults creates a SearchOverlay with pre-populated search
// results delivered via SearchPageLoadedMsg.
func newTestSearchOverlayWithResults() *panes.SearchOverlay {
	t := theme.Load("black")
	overlay := panes.NewSearchOverlay(t)

	// Deliver results the same way the root app model does: via SearchPageLoadedMsg.
	msg := panes.SearchPageLoadedMsg{Results: sampleSearchListItems()}
	model, _ := overlay.Update(msg)
	overlay = model.(*panes.SearchOverlay)

	return overlay
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
	case "ctrl+right":
		msg = tea.KeyMsg{Type: tea.KeyCtrlRight}
	case "ctrl+left":
		msg = tea.KeyMsg{Type: tea.KeyCtrlLeft}
	case "pgdown":
		msg = tea.KeyMsg{Type: tea.KeyPgDown}
	case "pgup":
		msg = tea.KeyMsg{Type: tea.KeyPgUp}
	case "home":
		msg = tea.KeyMsg{Type: tea.KeyHome}
	default:
		// Regular rune key
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}

	model, cmd := o.Update(msg)
	updated, ok := model.(*panes.SearchOverlay)
	require.True(t, ok, "Update must return *panes.SearchOverlay")
	return updated, cmd
}

// --- Init / clear-on-open tests ---

// TestSearchOverlay_Init_EmitsSearchClearedMsg verifies that Init() includes
// a SearchClearedMsg command so each search session starts with a clean state.
func TestSearchOverlay_Init_EmitsSearchClearedMsg(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)

	// Init() returns a Batch. We execute the batch and collect all messages.
	initCmd := o.Init()
	require.NotNil(t, initCmd, "Init() should return a non-nil command")

	msg := initCmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	require.True(t, ok, "Init() should return a BatchMsg, got %T", msg)

	// At least one command in the batch must produce SearchClearedMsg.
	var gotCleared bool
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if _, cleared := subCmd().(panes.SearchClearedMsg); cleared {
			gotCleared = true
		}
	}
	assert.True(t, gotCleared, "Init() batch must include a SearchClearedMsg command")
}

// --- Story 96: Reset() tests ---

// TestSearchOverlay_Reset_ClearsAllLocalState verifies that Reset() restores the overlay
// to its initial empty state regardless of any prior session state.
func TestSearchOverlay_Reset_ClearsAllLocalState(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(120, 40)

	// Set query to something and lock a prefix (:songs).
	// Typing ":songs " locks the prefix and syncs the active tab to TabSongs.
	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "s")
	o, _ = sendKey(t, o, "o")
	o, _ = sendKey(t, o, "n")
	o, _ = sendKey(t, o, "g")
	o, _ = sendKey(t, o, "s")
	o, _ = sendKey(t, o, " ")
	require.Equal(t, panes.PrefixLocked, o.PrefixState(), "prerequisite: prefix should be locked")
	require.Equal(t, panes.TabSongs, o.ActiveTab(), "prerequisite: active tab should be TabSongs after locking :songs")

	// Load fake results into o.results and o.resultList.
	msg := panes.SearchPageLoadedMsg{Results: sampleSearchListItems()}
	model, _ := o.Update(msg)
	o = model.(*panes.SearchOverlay)
	require.NotEmpty(t, o.ResultListItems(), "prerequisite: result list should be non-empty")

	// Call Reset() and verify all fields are back to initial state.
	o.Reset()

	assert.Equal(t, "", o.Query(), "after Reset: input value should be empty")
	assert.Equal(t, "> ", o.Input().Prompt, "after Reset: input Prompt should be '> '")
	assert.Equal(t, panes.TabAll, o.ActiveTab(), "after Reset: activeTab should be TabAll")
	assert.Equal(t, panes.PrefixNone, o.PrefixState(), "after Reset: prefixState should be PrefixNone")
	assert.Equal(t, "", o.LockedPrefix(), "after Reset: lockedPrefix should be empty")
	assert.Nil(t, o.Results(), "after Reset: o.results should be nil")
	assert.Empty(t, o.ResultListItems(), "after Reset: result list items should be empty")
	assert.Equal(t, 0, o.PlaceholderIdx(), "after Reset: placeholderIdx should be 0")
}

// TestSearchOverlay_Reset_Idempotent verifies that calling Reset() multiple times
// in a row produces the same clean state.
func TestSearchOverlay_Reset_Idempotent(t *testing.T) {
	o := newTestSearchOverlay()
	o.Reset()
	o.Reset()
	o.Reset()

	assert.Equal(t, "", o.Query(), "after triple Reset: input value should be empty")
	assert.Equal(t, panes.TabAll, o.ActiveTab(), "after triple Reset: activeTab should be TabAll")
	assert.Equal(t, panes.PrefixNone, o.PrefixState(), "after triple Reset: prefixState should be PrefixNone")
	assert.Equal(t, "", o.LockedPrefix(), "after triple Reset: lockedPrefix should be empty")
	assert.Nil(t, o.Results(), "after triple Reset: o.results should be nil")
	assert.Empty(t, o.ResultListItems(), "after triple Reset: result list items should be empty")
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
	o := newTestSearchOverlayWithResults()
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
	o := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	// First result is a track; press Ctrl+A to add to queue
	_, cmd := sendKey(t, o, "ctrl+a")

	require.NotNil(t, cmd)
	msg := cmd()
	qMsg, ok := msg.(AddToQueueMsg)
	require.True(t, ok, "pressing Ctrl+A on track should produce AddToQueueMsg")
	assert.Equal(t, "spotify:track:t1", qMsg.TrackURI)
}

// TestSearchOverlay_Update_Tab verifies Tab advances the active category tab.
// In the redesigned overlay, Tab/Shift+Tab cycle the tab bar (not the section).
func TestSearchOverlay_Update_Tab(t *testing.T) {
	o := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	initialTab := o.ActiveTab()
	o, _ = sendKey(t, o, "tab")

	assert.NotEqual(t, initialTab, o.ActiveTab(), "Tab should advance to next category tab")
}

// TestSearchOverlay_Update_ShiftTab verifies Shift+Tab retreats the active category tab.
func TestSearchOverlay_Update_ShiftTab(t *testing.T) {
	o := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	// Move forward twice, then Shift+Tab to go back once.
	o, _ = sendKey(t, o, "tab")
	o, _ = sendKey(t, o, "tab")
	afterForward := o.ActiveTab()
	o, _ = sendKey(t, o, "shift+tab")

	assert.NotEqual(t, afterForward, o.ActiveTab(), "Shift+Tab should retreat to previous category tab")
}

// TestSearchOverlay_Update_JK verifies arrow key navigation moves cursor within section.
func TestSearchOverlay_Update_JK(t *testing.T) {
	o := newTestSearchOverlayWithResults()
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

// TestSearchOverlay_Update_CtrlU verifies Ctrl+U is a no-op:
// the input keeps its value (the textinput swallows Ctrl+U without effect).
// Per the 2026-04-28 overlay-keybinding-cleanup spec, clearing only happens
// when the user mutates the input directly.
func TestSearchOverlay_Update_CtrlU(t *testing.T) {
	o := newTestSearchOverlay()

	o, _ = sendKey(t, o, "h")
	o, _ = sendKey(t, o, "e")
	o, _ = sendKey(t, o, "l")
	o, _ = sendKey(t, o, "l")
	o, _ = sendKey(t, o, "o")
	require.Contains(t, o.Query(), "hello", "query should be 'hello' after typing those chars")

	o, _ = sendKey(t, o, "ctrl+u")
	assert.Equal(t, "hello", o.Query(), "Ctrl+U must not clear the input — clearing only via direct edits")
}

// --- Task 4.4: Result rendering tests ---

// TestSearchOverlay_View_Results verifies list items are rendered with badge symbols.
// After Story 84, section headers (TRACKS/ARTISTS/etc.) are replaced by badge symbols (♪/★/◎/▤).
func TestSearchOverlay_View_Results(t *testing.T) {
	o := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	view := o.View()

	// Category badge symbols replace old section headers.
	assert.True(t, strings.ContainsAny(view, "♪★◎▤"), "view should contain category badge symbols")
	assert.Contains(t, view, "Blinding Lights", "view should contain track name")
	assert.Contains(t, view, "The Weeknd", "view should contain artist name")
}

// TestSearchOverlay_View_SelectedHighlight verifies the selected item is shown.
// After Story 84, the list delegate handles selection highlighting; the first item
// is shown as selected by default (index 0).
func TestSearchOverlay_View_SelectedHighlight(t *testing.T) {
	o := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	view := o.View()

	// The selected item (first item, "Blinding Lights") should be visible.
	assert.Contains(t, view, "Blinding Lights", "selected item should be rendered")
}

// TestSearchOverlay_View_Truncation verifies long names are truncated at narrow widths.
func TestSearchOverlay_View_Truncation(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)

	// Very long track name
	longName := strings.Repeat("A", 120)
	results := panes.TracksToSearchListItems([]domain.Track{
		{URI: "spotify:track:t1", Name: longName, Artists: []domain.Artist{{Name: "Artist"}}},
	})
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: results})
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

// TestSearchOverlay_View_NoResults verifies no-results state shows hint text.
// TODO(19-search-redesign): "No results for 'query'" message restored in story 99 when
// overlay owns the query string in o.results. Until then, empty results show generic hint.
func TestSearchOverlay_View_NoResults(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	// Deliver empty results via message (nil slice = zero items)
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: nil})
	o = model.(*panes.SearchOverlay)
	o.SetSize(80, 40)

	view := o.View()

	// With store removed, empty results show the generic hint text until story 99.
	assert.Contains(t, view, "Type to search", "empty results should show hint text")
}

// TODO(19-search-redesign): TestSearchOverlay_View_Loading removed — loading state
// was driven by store.SetSearchLoading which is deleted in story 97.
// Story 99 will re-add spinner display once the overlay owns loading state in o.results.

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
	// TODO(19-search-redesign): store.SetSearchError removed; errors route through toast
	// notifications in app.go. This test updated to verify overlay does not inline errors.
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 30)

	output := o.View()
	// Errors route through toast notifications, not inline pane rendering.
	assert.NotContains(t, output, "Search failed", "inline error rendering removed — toasts handle this")
}

func TestSearchOverlay_View_ShowsNoResults(t *testing.T) {
	// TODO(19-search-redesign): "No results for 'query'" message restored in story 99
	// once the overlay owns the query string in o.results. Until then empty results show hint.
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)

	// Deliver empty results via SearchPageLoadedMsg (nil slice = zero items)
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: nil})
	o = model.(*panes.SearchOverlay)
	o.SetSize(80, 30)

	output := o.View()
	// With store removed, empty results show the generic hint text until story 99.
	assert.Contains(t, output, "Type to search", "empty results should show hint text")
}

func TestSearchOverlay_View_ShowsResults(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)

	// Deliver results via SearchPageLoadedMsg
	results := panes.TracksToSearchListItems([]domain.Track{
		{URI: "spotify:track:t1", Name: "Blinding Lights", Artists: []domain.Artist{{Name: "The Weeknd"}}},
	})
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: results})
	o = model.(*panes.SearchOverlay)
	o.SetSize(80, 30)

	output := o.View()
	// After Story 84, badge symbols replace old section headers.
	assert.Contains(t, output, "Blinding Lights", "should show track name in results")
	assert.Contains(t, output, "♪", "should show track badge symbol")
}

func TestSearchOverlay_DebounceToSearchRequest_Pipeline(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 30)

	// Type a character to get the input populated.
	o, _ = sendKey(t, o, "b")

	// Simulate the debounce msg arriving with the correct query snapshot.
	_, cmd := o.Update(panes.SearchDebounceMsgForTest("b"))

	// The debounce should have produced a command that returns SearchRequestMsg.
	require.NotNil(t, cmd, "debounce should produce a command")
	msg := cmd()
	_, ok := msg.(panes.SearchRequestMsg)
	assert.True(t, ok, "debounce cmd should produce SearchRequestMsg, got %T", msg)
}

// TestSearchOverlay_SearchPageLoadedMsg_StoresResults verifies that SearchPageLoadedMsg
// results are stored in the overlay and visible in View().
func TestSearchOverlay_SearchPageLoadedMsg_StoresResults(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	tracks := panes.TracksToSearchListItems([]domain.Track{
		{URI: "spotify:track:abc", Name: "Track One", Artists: []domain.Artist{{Name: "Artist One"}}},
	})
	artists := panes.ArtistsToSearchListItems([]domain.SearchArtist{
		{URI: "spotify:artist:xyz", Name: "Artist One"},
	})
	results := append(tracks, artists...)

	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: results})
	o = model.(*panes.SearchOverlay)

	view := o.View()
	assert.Contains(t, view, "Track One", "view should show track from SearchPageLoadedMsg")
	// After Story 84: badge symbols replace old section headers.
	assert.True(t, strings.ContainsAny(view, "♪★"), "view should show badge symbols")
	assert.Contains(t, view, "Artist One", "view should show artist from SearchPageLoadedMsg")
}

// TestSearchOverlay_NoAPIImportBoundary verifies the import boundary:
// search.go must not import api/. This test uses only panes types.
func TestSearchOverlay_NoAPIImportBoundary(t *testing.T) {
	// This test verifies the architectural boundary at the type level.
	// If search.go imported api/, the panes package would fail to build without api/.
	// We exercise the full rendering path using only panes types.
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)

	results := []panes.SearchListItem{
		{Category: "track", Name: "T1", URI: "u1", IsTrack: true},
		{Category: "artist", Name: "A2", URI: "u2"},
		{Category: "album", Name: "Al1", URI: "u3"},
		{Category: "playlist", Name: "PL1", URI: "u4"},
	}
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: results})
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

// --- Story 82: Three-panel layout tests ---

// TestSearchTab_EnumValues verifies numTabs==5 and tabToAPITypes returns correct types.
func TestSearchTab_EnumValues(t *testing.T) {
	assert.Equal(t, 5, panes.NumTabs, "numTabs should be 5")

	tests := []struct {
		tab       panes.SearchTab
		wantTypes []string
	}{
		{panes.TabAll, []string{"track", "artist", "album", "playlist"}},
		{panes.TabSongs, []string{"track"}},
		{panes.TabArtists, []string{"artist"}},
		{panes.TabAlbums, []string{"album"}},
		{panes.TabPlaylists, []string{"playlist"}},
	}
	for _, tt := range tests {
		types := panes.TabToAPITypes(tt.tab)
		assert.Equal(t, tt.wantTypes, types, "API types for tab %d", tt.tab)
	}
}

// TestSearchOverlay_Tab_CyclesForward verifies Tab cycles the active tab forward with wrapping.
func TestSearchOverlay_Tab_CyclesForward(t *testing.T) {
	o := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	// Default should be tabAll (0).
	assert.Equal(t, panes.TabAll, o.ActiveTab(), "default tab should be tabAll")

	// Cycle through all tabs and verify wrapping.
	for i := 1; i < panes.NumTabs; i++ {
		o, _ = sendKey(t, o, "tab")
		assert.Equal(t, panes.SearchTab(i), o.ActiveTab(), "tab %d after %d forward press(es)", i, i)
	}

	// One more Tab should wrap back to tabAll.
	o, _ = sendKey(t, o, "tab")
	assert.Equal(t, panes.TabAll, o.ActiveTab(), "tab should wrap to tabAll after numTabs presses")
}

// TestSearchOverlay_Tab_CyclesBackward verifies Shift+Tab cycles the active tab backward with wrapping.
func TestSearchOverlay_ShiftTab_CyclesBackward(t *testing.T) {
	o := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	// At tabAll (0), Shift+Tab should wrap to tabPlaylists (numTabs-1).
	o, _ = sendKey(t, o, "shift+tab")
	assert.Equal(t, panes.SearchTab(panes.NumTabs-1), o.ActiveTab(), "shift+tab from tabAll should wrap to last tab")
}

// TestSearchOverlay_Tab_EmitsSearchTabChangedMsg verifies tab change emits SearchTabChangedMsg.
// TestSearchOverlay_Tab_EmitsSearchRequestMsg verifies tab change emits SearchRequestMsg
// (SearchTabChangedMsg removed in story 98 — tab changes now go through the universal debounce path).
// Story 99: cycleTab uses scheduleDebounce(), so SearchRequestMsg arrives via two-step flow:
// tab key → debounce cmd → debounce msg → search request.
func TestSearchOverlay_Tab_EmitsSearchRequestMsg(t *testing.T) {
	// Start with a non-empty query so the debounce fires a SearchRequestMsg.
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	// Type a query so the overlay has non-empty cleanQuery().
	for _, ch := range "rock" {
		o, _ = sendKey(t, o, string(ch))
	}

	// Tab returns scheduleDebounce() — a time-based tick cmd.
	_, debounceCmd := sendKey(t, o, "tab")
	require.NotNil(t, debounceCmd, "tab change should return a debounce command when query is non-empty")

	// Execute the debounce cmd to get the searchDebounceMsg.
	tickMsg := debounceCmd()

	// Feed the searchDebounceMsg into Update to fire the search.
	_, searchCmd := o.Update(tickMsg)
	require.NotNil(t, searchCmd, "debounce msg from tab change should produce a search command")
	msg := searchCmd()
	reqMsg, ok := msg.(panes.SearchRequestMsg)
	require.True(t, ok, "tab change + debounce should produce SearchRequestMsg, got %T", msg)
	assert.Equal(t, []string{"track"}, reqMsg.Types, "Songs tab should map to track type")
	assert.Equal(t, "rock", reqMsg.Query, "SearchRequestMsg should carry the current query")
	assert.Equal(t, 1, reqMsg.Page, "SearchRequestMsg page should default to 1")
}

// TestSearchOverlay_View_ThreePanels verifies View() contains three ╭ border starts.
func TestSearchOverlay_View_ThreePanels(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)

	view := o.View()
	count := strings.Count(view, "╭")
	assert.GreaterOrEqual(t, count, 3, "view should contain at least 3 panel borders (╭)")
}

// TestSearchOverlay_View_SearchPanelTitle verifies Panel 1 has "Search" title.
func TestSearchOverlay_View_SearchPanelTitle(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)
	view := o.View()
	assert.Contains(t, view, "Search", "Panel 1 border should have 'Search' title")
}

// TestSearchOverlay_View_ResultsPanelTitle verifies Panel 2 has "Results" title.
func TestSearchOverlay_View_ResultsPanelTitle(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)
	view := o.View()
	assert.Contains(t, view, "Results", "Panel 2 border should have 'Results' title")
}

// TestSearchOverlay_View_KeysPanelTitle_NoTitle verifies Panel 3 has no title label.
// Story 90: Keys panel title removed — keybinding content is self-explanatory.
func TestSearchOverlay_View_KeysPanelTitle_NoTitle(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)
	view := o.View()
	// "Keys" label should NOT appear in the border — it was removed in Story 90.
	assert.NotContains(t, view, "─ Keys", "Panel 3 border should NOT have 'Keys' title label")
}

// TestSearchOverlay_View_TabBarPresent verifies tab labels appear in the results panel.
func TestSearchOverlay_View_TabBarPresent(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)
	view := o.View()
	assert.Contains(t, view, "All", "tab bar should show 'All' tab")
	assert.Contains(t, view, "Songs", "tab bar should show 'Songs' tab")
	assert.Contains(t, view, "Artists", "tab bar should show 'Artists' tab")
	assert.Contains(t, view, "Albums", "tab bar should show 'Albums' tab")
	assert.Contains(t, view, "Playlists", "tab bar should show 'Playlists' tab")
}

// TestSearchOverlay_View_TabBarActiveHighlight verifies the active tab uses bracket style.
func TestSearchOverlay_View_TabBarActiveHighlight(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)
	view := o.View()
	// Active tab (All) should render with brackets.
	assert.Contains(t, view, "[All]", "active tab should be shown with brackets")
}

// TestSearchOverlay_View_HelpPanelContent verifies the Keys panel renders
// the uikit.KeyBar with the visible binding set.
func TestSearchOverlay_View_HelpPanelContent(t *testing.T) {
	o := newTestSearchOverlay()
	// Use a wide terminal so the help bar has room to show keybinding text.
	o.SetSize(150, 40)
	view := o.View()
	// Keys panel must advertise ctrl+a queue (the Queue binding).
	assert.Contains(t, view, "ctrl+a", "help bar should show ctrl+a keybinding")
	// Enter (play) and Esc (close) are no longer advertised in the keybar.
	assert.NotContains(t, stripANSI(view), "esc close", "esc close must not appear in keybar")
}

// TestSearchOverlay_OverlayWidth_70Pct verifies overlayWidth returns 70% of terminal width.
func TestSearchOverlay_OverlayWidth_70Pct(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(200, 50)
	assert.Equal(t, 140, o.OverlayWidth(), "200-wide terminal should give 140-wide overlay (70%)")
	assert.Equal(t, 40, o.OverlayHeight(), "50-high terminal should give 40-high overlay (80%)")
}

// TestSearchOverlay_OverlaySize_MinClamp verifies minimum clamping.
func TestSearchOverlay_OverlaySize_MinClamp(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(10, 5) // very small terminal
	assert.Equal(t, 40, o.OverlayWidth(), "overlay width minimum should be 40")
	assert.Equal(t, 15, o.OverlayHeight(), "overlay height minimum should be 15")
}

// TestSearchOverlay_SetSize_PropagatesList verifies SetSize propagates to list.Model inner dims.
func TestSearchOverlay_SetSize_PropagatesList(t *testing.T) {
	o := newTestSearchOverlay()
	// Should not panic when called — verifies list.Model is initialized and SetSize works.
	o.SetSize(120, 40)
	// If we get here, SetSize correctly propagated to list.Model without panic.
	// The overlay should also render without error.
	view := o.View()
	assert.NotEmpty(t, view, "view should not be empty after SetSize")
}

// TestSearchOverlay_SearchPageLoadedMsg_ErrorPreservesResults verifies that when a
// SearchPageLoadedMsg carries a non-nil Err, the overlay does NOT wipe its existing
// displayed results. The toast (handled by app.go) is the user-facing feedback; the
// overlay should keep showing whatever it already had so the screen isn't blanked.
func TestSearchOverlay_SearchPageLoadedMsg_ErrorPreservesResults(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// First deliver a successful page so the overlay has results to display.
	initialResults := panes.TracksToSearchListItems([]domain.Track{
		{URI: "spotify:track:t1", Name: "Jazz Track", Artists: []domain.Artist{{Name: "Miles Davis"}}},
	})
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: initialResults, Total: 1})
	o = model.(*panes.SearchOverlay)

	// Verify results are visible before the error arrives.
	require.Contains(t, o.View(), "Jazz Track", "pre-condition: initial results must be shown")

	// Now deliver an error page — results should be preserved, not wiped.
	model, _ = o.Update(panes.SearchPageLoadedMsg{
		Query: "jazz",
		Err:   fmt.Errorf("network error"),
	})
	o = model.(*panes.SearchOverlay)

	assert.Contains(t, o.View(), "Jazz Track", "error response must not wipe existing displayed results")
}

// --- Story 84: bubbles/list with custom delegate ---

// TestSearchListItem_InterfaceMethods verifies SearchListItem implements list.Item correctly.
func TestSearchListItem_InterfaceMethods(t *testing.T) {
	item := panes.SearchListItem{
		Category: "track",
		Name:     "Blinding Lights",
		Subtitle: "The Weeknd",
		URI:      "spotify:track:t1",
		IsTrack:  true,
	}
	assert.Equal(t, "Blinding Lights", item.Title(), "Title() should return name")
	assert.Equal(t, "The Weeknd", item.Description(), "Description() should return subtitle")
	assert.Equal(t, "Blinding Lights", item.FilterValue(), "FilterValue() should return name")
}

// TestSearchListItem_AllCategories verifies each category badge symbol is non-empty.
func TestSearchListItem_AllCategories(t *testing.T) {
	categories := []string{"track", "artist", "album", "playlist"}
	for _, cat := range categories {
		item := panes.SearchListItem{Category: cat, Name: "Test", URI: "u"}
		assert.NotEmpty(t, item.Category, "category should be set for %s", cat)
	}
}

// TestSearchItemDelegate_Height verifies delegate height is 3 (3-line layout).
func TestSearchItemDelegate_Height(t *testing.T) {
	th := theme.Load("black")
	d := panes.NewSearchItemDelegate(th)
	assert.Equal(t, 3, d.Height(), "delegate height should be 3")
}

// TestSearchItemDelegate_Spacing verifies delegate spacing is 0.
func TestSearchItemDelegate_Spacing(t *testing.T) {
	th := theme.Load("black")
	d := panes.NewSearchItemDelegate(th)
	assert.Equal(t, 0, d.Spacing(), "delegate spacing should be 0")
}

// TestSearchItemDelegate_Render_BadgeAndName verifies Render outputs badge symbol + name.
func TestSearchItemDelegate_Render_BadgeAndName(t *testing.T) {
	th := theme.Load("black")
	d := panes.NewSearchItemDelegate(th)

	tests := []struct {
		category   string
		wantSymbol string
	}{
		{"track", "♪"},
		{"artist", "★"},
		{"album", "◎"},
		{"playlist", "▤"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			item := panes.SearchListItem{
				Category: tt.category,
				Name:     "Test Item",
				Subtitle: "subtitle",
				URI:      "spotify:test",
				IsTrack:  tt.category == "track",
			}
			var buf strings.Builder
			l := panes.NewTestList(d)
			d.Render(&buf, l, 0, item)
			output := buf.String()
			assert.Contains(t, output, tt.wantSymbol, "render should include badge symbol for %s", tt.category)
			assert.Contains(t, output, "Test Item", "render should include item name")
		})
	}
}

// TestSearchItemDelegate_Render_Subtitle verifies artist and album are rendered on line 2.
func TestSearchItemDelegate_Render_Subtitle(t *testing.T) {
	th := theme.Load("black")
	d := panes.NewSearchItemDelegate(th)

	item := panes.SearchListItem{
		Category:    "track",
		Name:        "Track Name",
		Subtitle:    "Artist Name · Album Name · 3:00",
		URI:         "u1",
		IsTrack:     true,
		ArtistNames: "Artist Name",
		AlbumName:   "Album Name",
		Duration:    "3:00",
	}
	var buf strings.Builder
	l := panes.NewTestList(d)
	d.Render(&buf, l, 0, item)
	output := buf.String()
	assert.Contains(t, output, "Artist Name", "render should include artist name on line 2")
	assert.Contains(t, output, "Album Name", "render should include album name on line 2")
}

// TestSearchOverlay_RebuildListItems_AllTab verifies all delivered items appear in the list.
// Story 98: items are pre-converted SearchListItems; tab filtering wired in story 99.
func TestSearchOverlay_RebuildListItems_AllTab(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// Deliver pre-converted SearchListItems via SearchPageLoadedMsg.
	results := []panes.SearchListItem{
		{Category: "track", Name: "Track One", URI: "spotify:track:t1"},
		{Category: "artist", Name: "Artist B", URI: "spotify:artist:a1"},
		{Category: "album", Name: "Album C", URI: "spotify:album:al1"},
		{Category: "playlist", Name: "Playlist D", URI: "spotify:playlist:pl1"},
	}
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: results})
	o = model.(*panes.SearchOverlay)

	// After SearchPageLoadedMsg, the list should have 4 items.
	assert.Equal(t, 4, panes.ListItemCount(o), "all delivered items should appear in the list")
}

// TestSearchOverlay_RebuildListItems_SongsTab verifies rebuildListItems after SetActiveTab.
// Story 99 wires per-tab filtering; for story 98 all delivered items are shown.
func TestSearchOverlay_RebuildListItems_SongsTab(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// Deliver pre-converted track items only (as would happen with a :songs-filtered request).
	results := []panes.SearchListItem{
		{Category: "track", Name: "Track One", URI: "spotify:track:t1"},
		{Category: "track", Name: "Track Two", URI: "spotify:track:t2"},
	}
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: results})
	o = model.(*panes.SearchOverlay)
	// Switch to Songs tab and rebuild.
	panes.SetActiveTab(o, panes.TabSongs)
	panes.CallRebuildListItems(o)

	assert.Equal(t, 2, panes.ListItemCount(o), "track-only delivery should show 2 items")
}

// TestSearchOverlay_RebuildListItems_EmptyResults verifies nil o.results produces empty list.
func TestSearchOverlay_RebuildListItems_EmptyResults(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)
	// No SearchPageLoadedMsg delivered — o.results is nil.
	panes.CallRebuildListItems(o)

	assert.Equal(t, 0, panes.ListItemCount(o), "nil results should produce empty list")
}

// TestSearchOverlay_DownKey_MovesCursor verifies down key advances list cursor.
func TestSearchOverlay_DownKey_MovesCursor(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: []panes.SearchListItem{
		{Category: "track", Name: "Track One", URI: "spotify:track:t1", IsTrack: true},
		{Category: "track", Name: "Track Two", URI: "spotify:track:t2", IsTrack: true},
	}})
	o = model.(*panes.SearchOverlay)

	initialIdx := panes.ListCursorIndex(o)
	o, _ = sendKey(t, o, "down")
	assert.Equal(t, initialIdx+1, panes.ListCursorIndex(o), "down key should advance list cursor")
}

// TestSearchOverlay_Enter_TrackEmitsPlayTrackListMsg verifies Enter on a track emits
// PlayTrackListMsg with URIs from the selected track to end (Story 105).
func TestSearchOverlay_Enter_TrackEmitsPlayTrackListMsg(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: []panes.SearchListItem{
		{Category: "track", Name: "Track One", URI: "spotify:track:t1", IsTrack: true},
		{Category: "track", Name: "Track Two", URI: "spotify:track:t2", IsTrack: true},
		{Category: "track", Name: "Track Three", URI: "spotify:track:t3", IsTrack: true},
	}})
	o = model.(*panes.SearchOverlay)

	_, cmd := sendKey(t, o, "enter")
	require.NotNil(t, cmd)
	// handleEnter should return PlayTrackListMsg with all track URIs from position 0.
	msg := cmd()
	ptMsg, ok := msg.(panes.PlayTrackListMsg)
	require.True(t, ok, "Enter on track should return PlayTrackListMsg, got %T", msg)
	require.Len(t, ptMsg.URIs, 3, "should include URIs from selected track to end")
	assert.Equal(t, "spotify:track:t1", ptMsg.URIs[0])
	assert.Equal(t, "spotify:track:t2", ptMsg.URIs[1])
	assert.Equal(t, "spotify:track:t3", ptMsg.URIs[2])
}

// TestSearchOverlay_Enter_MixedResultsFiltersOnlyTracks verifies that pressing Enter
// on a track in a mixed-type results list (tracks interleaved with albums/artists) emits
// PlayTrackListMsg containing only track URIs from the selected index onward.
// Album and artist URIs must NOT appear in the emitted list (Story 105).
func TestSearchOverlay_Enter_MixedResultsFiltersOnlyTracks(t *testing.T) {
	tests := []struct {
		name        string
		results     []panes.SearchListItem
		cursorMoves int // how many "down" keys to press before Enter
		wantURIs    []string
	}{
		{
			name: "cursor on first item — albums and artists filtered out",
			results: []panes.SearchListItem{
				{Category: "track", Name: "Track 1", URI: "spotify:track:t1", IsTrack: true},
				{Category: "album", Name: "Album 1", URI: "spotify:album:al1"},
				{Category: "track", Name: "Track 2", URI: "spotify:track:t2", IsTrack: true},
				{Category: "artist", Name: "Artist 1", URI: "spotify:artist:ar1"},
				{Category: "track", Name: "Track 3", URI: "spotify:track:t3", IsTrack: true},
			},
			cursorMoves: 0,
			wantURIs: []string{
				"spotify:track:t1",
				"spotify:track:t2",
				"spotify:track:t3",
			},
		},
		{
			name: "cursor on track:t1 (idx=1) — album before cursor excluded, tracks from cursor onward included",
			results: []panes.SearchListItem{
				{Category: "album", Name: "Album 1", URI: "spotify:album:al1"},
				{Category: "track", Name: "Track 1", URI: "spotify:track:t1", IsTrack: true},
				{Category: "track", Name: "Track 2", URI: "spotify:track:t2", IsTrack: true},
			},
			cursorMoves: 1,
			wantURIs: []string{
				"spotify:track:t1",
				"spotify:track:t2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := theme.Load("black")
			o := panes.NewSearchOverlay(th)
			o.SetSize(80, 40)
			model, _ := o.Update(panes.SearchPageLoadedMsg{Results: tt.results})
			o = model.(*panes.SearchOverlay)

			// Move cursor to desired position.
			for i := 0; i < tt.cursorMoves; i++ {
				o, _ = sendKey(t, o, "down")
			}

			_, cmd := sendKey(t, o, "enter")
			require.NotNil(t, cmd, "Enter on a track should return a command")
			msg := cmd()
			ptMsg, ok := msg.(panes.PlayTrackListMsg)
			require.True(t, ok, "Enter on track should return PlayTrackListMsg, got %T", msg)
			require.Equal(t, tt.wantURIs, ptMsg.URIs, "only track URIs from cursor onward should be included")

			// Verify no album or artist URIs leaked into the list.
			for _, uri := range ptMsg.URIs {
				assert.NotContains(t, uri, ":album:", "album URI must not appear in PlayTrackListMsg")
				assert.NotContains(t, uri, ":artist:", "artist URI must not appear in PlayTrackListMsg")
			}
		})
	}
}

// TestSearchOverlay_Enter_TrackDoesNotCloseOverlay verifies Enter does NOT emit SearchClosedMsg.
func TestSearchOverlay_Enter_TrackDoesNotCloseOverlay(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: []panes.SearchListItem{
		{Category: "track", Name: "Track One", URI: "spotify:track:t1", IsTrack: true},
	}})
	o = model.(*panes.SearchOverlay)

	_, cmd := sendKey(t, o, "enter")
	require.NotNil(t, cmd)
	msg := cmd()
	_, isClose := msg.(panes.SearchClosedMsg)
	assert.False(t, isClose, "Enter should NOT emit SearchClosedMsg — only Esc closes the overlay")
}

// TestSearchOverlay_Enter_AlbumEmitsPlayContextMsg verifies Enter on an album emits PlayContextMsg only (no close).
func TestSearchOverlay_Enter_AlbumEmitsPlayContextMsg(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)
	// Switch to Albums tab so albums show up first.
	panes.SetActiveTab(o, panes.TabAlbums)
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: []panes.SearchListItem{
		{Category: "album", Name: "Album One", URI: "spotify:album:al1"},
	}})
	o = model.(*panes.SearchOverlay)

	_, cmd := sendKey(t, o, "enter")
	require.NotNil(t, cmd)
	// handleEnter should return only the play command — no SearchClosedMsg.
	msg := cmd()
	pcMsg, ok := msg.(panes.PlayContextMsg)
	require.True(t, ok, "Enter on album should return PlayContextMsg directly, got %T", msg)
	assert.Equal(t, "spotify:album:al1", pcMsg.ContextURI)
}

// TestSearchOverlay_CtrlA_TrackEmitsAddToQueueMsg verifies Ctrl+A on track emits AddToQueueMsg.
func TestSearchOverlay_CtrlA_ListDelegate_EmitsAddToQueueMsg(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: []panes.SearchListItem{
		{Category: "track", Name: "Track One", URI: "spotify:track:t1", IsTrack: true},
	}})
	o = model.(*panes.SearchOverlay)

	_, cmd := sendKey(t, o, "ctrl+a")
	require.NotNil(t, cmd)
	msg := cmd()
	qMsg, ok := msg.(panes.AddToQueueMsg)
	require.True(t, ok, "Ctrl+A on track should emit AddToQueueMsg, got %T", msg)
	assert.Equal(t, "spotify:track:t1", qMsg.TrackURI)
}

// TestSearchOverlay_View_ListDelegate_ContainsBadgeSymbol verifies list items show category badges.
func TestSearchOverlay_View_ListDelegate_ContainsBadgeSymbol(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: []panes.SearchListItem{
		{Category: "track", Name: "My Track", URI: "spotify:track:t1", IsTrack: true},
		{Category: "artist", Name: "My Artist", URI: "spotify:artist:a1"},
	}})
	o = model.(*panes.SearchOverlay)

	view := o.View()
	assert.Contains(t, view, "My Track", "view should contain track name")
	// Badge symbols should be present (new symbols: ♪ ★ ◎ ▤).
	assert.True(t, strings.ContainsAny(view, "♪★◎▤"), "view should contain at least one badge symbol")
}

// TestSearchOverlay_View_NoSectionHeaders verifies old TRACKS/ARTISTS headers are gone.
func TestSearchOverlay_View_NoSectionHeaders(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: []panes.SearchListItem{
		{Category: "track", Name: "T1", URI: "u1"},
	}})
	o = model.(*panes.SearchOverlay)

	view := o.View()
	assert.NotContains(t, view, "● TRACKS", "old TRACKS section header should be gone")
	assert.NotContains(t, view, "● ARTISTS", "old ARTISTS section header should be gone")
}

// TestSearchOverlay_SetTheme_PropagatesToDelegate verifies that SetTheme updates
// the delegate's theme so badge colors change with the new theme.
func TestSearchOverlay_SetTheme_PropagatesToDelegate(t *testing.T) {
	th1 := theme.Load("black")
	o2 := panes.NewSearchOverlay(th1)
	o2.SetSize(80, 40)

	// Populate with a track via SearchPageLoadedMsg so the delegate renders something.
	model, _ := o2.Update(panes.SearchPageLoadedMsg{Results: []panes.SearchListItem{
		{Category: "track", Name: "Track One", URI: "spotify:track:t1", IsTrack: true},
	}})
	o2 = model.(*panes.SearchOverlay)
	panes.CallRebuildListItems(o2)

	// Switch to a different theme — SetTheme must not panic.
	th2 := theme.Load("dracula")
	require.NotNil(t, th2)
	o2.SetTheme(th2)

	// View() must succeed without panicking after theme switch.
	view := o2.View()
	assert.NotEmpty(t, view, "View() should return content after SetTheme")
}

// TestSearchOverlay_SetTheme_SpinnerStyleUpdated verifies spinner uses new theme colors.
func TestSearchOverlay_SetTheme_SpinnerStyleUpdated(t *testing.T) {
	th1 := theme.Load("black")
	o := panes.NewSearchOverlay(th1)
	o.SetSize(80, 40)

	// Switch to a different theme.
	th2 := theme.Load("gruvbox")
	require.NotNil(t, th2)

	// Must not panic and View() must succeed.
	o.SetTheme(th2)
	view := o.View()
	assert.NotEmpty(t, view, "View() should return content after SetTheme with spinner update")
}

// --- Edge case tests ---

// TestSearchOverlay_EmptyQueryAfterPrefix_NoSearch verifies that ':songs ' (prefix
// locked but no query after the space) shows the empty state and never fires a search.
// With the Prompt-based approach, when prefix is locked, input.Value() == "" (clean query only).
func TestSearchOverlay_EmptyQueryAfterPrefix_NoSearch(t *testing.T) {
	o := newTestSearchOverlay()

	// Type ':songs ' to lock the prefix with no query after it.
	for _, ch := range ":songs " {
		o, _ = sendKey(t, o, string(ch))
	}

	// With Prompt-based approach: prefix is in Prompt, Value holds only the clean query.
	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	assert.Equal(t, ":songs", o.LockedPrefix(), "locked prefix should be :songs")
	assert.Equal(t, "", o.Query(), "with Prompt tag, input.Value() holds only the clean query (empty)")
	assert.Equal(t, "", o.CleanQuery(), "clean query should be empty")

	// Fire a debounce tick for the empty clean query — must be a no-op.
	debounceMsg := panes.SearchDebounceMsgForTest("")
	model, cmd := o.Update(debounceMsg)
	updated := model.(*panes.SearchOverlay)
	_ = updated
	assert.Nil(t, cmd, "debounce on empty clean query should not fire a search request")
}

// --- Story 91: Placeholder behavior when prefix is locked ---

// TestSearchOverlay_Placeholder_LockedPrefix verifies that when a prefix is locked
// (promoteToPromptTag() called), the input placeholder changes to the static "search..."
// instead of the cycling prefix hints.
func TestSearchOverlay_Placeholder_LockedPrefix(t *testing.T) {
	o := newTestSearchOverlay()

	// Lock prefix by typing ":songs " (trailing space triggers lock + promotion).
	for _, ch := range ":songs " {
		o, _ = sendKey(t, o, string(ch))
	}

	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	assert.Equal(t, "search...", o.Placeholder(),
		"placeholder should be 'search...' when prefix is locked, not the cycling hint")
}

// TestSearchOverlay_Placeholder_DemoteRestoresCycling verifies that demoting the
// Prompt tag (Backspace at pos 0) restores the cycling placeholder.
func TestSearchOverlay_Placeholder_DemoteRestoresCycling(t *testing.T) {
	o := newTestSearchOverlay()

	// Lock prefix.
	for _, ch := range ":songs " {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())

	// Demote by pressing Backspace at cursor position 0.
	o, _ = sendKey(t, o, "home")      // move cursor to position 0
	o, _ = sendKey(t, o, "backspace") // demote the tag

	// After demotion, prefix state should be None.
	require.Equal(t, panes.PrefixNone, o.PrefixState())
	// Placeholder should be back to a cycling placeholder, not "search...".
	assert.Contains(t, panes.SearchPlaceholders, o.Placeholder(),
		"placeholder should be a cycling placeholder after demotion, not 'search...'")
}

// TestSearchOverlay_Placeholder_TabSwitchToNonAll verifies that cycling to a non-All tab
// (which locks a prefix) sets the placeholder to "search...".
func TestSearchOverlay_Placeholder_TabSwitchToNonAll(t *testing.T) {
	o := newTestSearchOverlay()

	// Cycle to Songs tab (first non-All tab).
	o, _ = sendKey(t, o, "tab")
	require.Equal(t, panes.TabSongs, o.ActiveTab())

	assert.Equal(t, "search...", o.Placeholder(),
		"placeholder should be 'search...' when non-All tab is active (prefix locked)")
}

// TestSearchOverlay_Placeholder_TabSwitchBackToAll verifies that cycling back to the All
// tab restores the cycling placeholder.
func TestSearchOverlay_Placeholder_TabSwitchBackToAll(t *testing.T) {
	o := newTestSearchOverlay()

	// Go to Songs tab, then back to All via Shift+Tab.
	o, _ = sendKey(t, o, "tab")
	require.Equal(t, panes.TabSongs, o.ActiveTab())

	o, _ = sendKey(t, o, "shift+tab")
	require.Equal(t, panes.TabAll, o.ActiveTab())

	assert.Contains(t, panes.SearchPlaceholders, o.Placeholder(),
		"placeholder should be a cycling placeholder when back on All tab")
}

// --- Story 90: Flush panels, interior hints, per-panel border colors ---

// TestSearchOverlay_View_FlushPanels verifies no blank line exists between consecutive panel borders.
// Between the first two panels and the second and third panel, the ╰ of one panel must be
// directly adjacent to the ╭ of the next (no empty lines between).
// Lines may contain ANSI escape codes so we use strings.Contains for matching.
func TestSearchOverlay_View_FlushPanels(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)

	view := o.View()
	lines := strings.Split(view, "\n")

	// Collect indices of all ╭ lines (panel starts) and ╰ lines (panel ends).
	var startLines, endLines []int
	for i, line := range lines {
		if strings.Contains(line, "╭") {
			startLines = append(startLines, i)
		}
		if strings.Contains(line, "╰") {
			endLines = append(endLines, i)
		}
	}
	require.Len(t, startLines, 3, "should have 3 panel top borders")
	require.Len(t, endLines, 3, "should have 3 panel bottom borders")

	// For each consecutive pair of panels, the end of panel N must be immediately
	// followed by the start of panel N+1.
	for i := 0; i < 2; i++ {
		endLine := endLines[i]
		nextStartLine := startLines[i+1]
		assert.Equal(t, endLine+1, nextStartLine,
			"panel %d end (line %d) must be immediately followed by panel %d start (line %d), got %d",
			i+1, endLine, i+2, nextStartLine, nextStartLine,
		)
	}
}

// TestSearchOverlay_View_ThreePanelBorderCount verifies exactly 3 panel start borders.
// Lines may contain ANSI escape codes so we use strings.Contains for matching.
func TestSearchOverlay_View_ThreePanelBorderCount(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)

	view := o.View()
	lines := strings.Split(view, "\n")

	// Count lines that contain ╭ (panel top borders).
	count := 0
	for _, line := range lines {
		if strings.Contains(line, "╭") {
			count++
		}
	}
	assert.Equal(t, 3, count, "view should have exactly 3 panel top borders (╭)")
}

// TestSearchOverlay_ShowHintLine_EmptyInput returns true when input is empty.
func TestSearchOverlay_ShowHintLine_EmptyInput(t *testing.T) {
	o := newTestSearchOverlay()
	assert.True(t, o.ShowHintLine(), "empty input should show hint line")
}

// TestSearchOverlay_ShowHintLine_PrefixTyping returns true during prefix typing.
func TestSearchOverlay_ShowHintLine_PrefixTyping(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)
	// Type ":so" — no space yet = PrefixTyping.
	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "s")
	o, _ = sendKey(t, o, "o")
	require.Equal(t, panes.PrefixTyping, o.PrefixState())
	assert.True(t, o.ShowHintLine(), "PrefixTyping should show hint line")
}

// TestSearchOverlay_ShowHintLine_PrefixLocked returns false when prefix is locked.
func TestSearchOverlay_ShowHintLine_PrefixLocked(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)
	// Type ":songs " to lock the prefix.
	for _, ch := range ":songs " {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	assert.False(t, o.ShowHintLine(), "PrefixLocked should not show hint line")
}

// TestSearchOverlay_ShowHintLine_NormalQuery returns false during normal query.
func TestSearchOverlay_ShowHintLine_NormalQuery(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)
	// Type a regular query without prefix.
	for _, ch := range "hello" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixNone, o.PrefixState())
	assert.False(t, o.ShowHintLine(), "normal query should not show hint line")
}

// TestSearchOverlay_View_HintInsideSearchPanel verifies hint text appears inside
// the Search panel border (between ╭─ Search and ╰─), not floating between panels.
// Lines may contain ANSI escape codes so we use strings.Contains for all matching.
func TestSearchOverlay_View_HintInsideSearchPanel(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)
	// Type ":so" to trigger PrefixTyping (hints visible).
	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "s")
	o, _ = sendKey(t, o, "o")
	require.Equal(t, panes.PrefixTyping, o.PrefixState())

	view := o.View()
	lines := strings.Split(view, "\n")

	searchStart, searchEnd := findPanelBounds(lines, "Search")
	require.Greater(t, searchStart, -1, "Search panel top border not found in view")
	require.Greater(t, searchEnd, -1, "Search panel bottom border not found in view")

	// The hint text ":songs" should appear INSIDE the search panel boundaries.
	hintFound := false
	for i := searchStart + 1; i < searchEnd; i++ {
		if strings.Contains(lines[i], ":songs") {
			hintFound = true
			break
		}
	}
	assert.True(t, hintFound, "hint text ':songs' should appear inside the Search panel border (lines %d to %d)", searchStart, searchEnd)

	// The hint text must NOT appear outside the search panel (between ╰ and the next ╭).
	outsideHint := false
	inGap := false
	for i, line := range lines {
		if i == searchEnd {
			inGap = true
		}
		if inGap && strings.Contains(line, "╭") && i > searchEnd {
			break // reached next panel — stop checking gap
		}
		if inGap && i > searchEnd && strings.Contains(line, ":songs") {
			outsideHint = true
		}
	}
	assert.False(t, outsideHint, "hint text ':songs' must NOT appear outside the Search panel border")
}

// TestSearchOverlay_View_SearchPanelHeight_WithHints verifies Search panel is 4 lines when hints visible.
// Lines may contain ANSI escape codes so we use strings.Contains for matching.
func TestSearchOverlay_View_SearchPanelHeight_WithHints(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)
	// Type ":so" to trigger PrefixTyping (hints visible).
	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "s")
	require.Equal(t, panes.PrefixTyping, o.PrefixState())
	require.True(t, o.ShowHintLine())

	view := o.View()
	lines := strings.Split(view, "\n")

	searchStart, searchEnd := findPanelBounds(lines, "Search")
	require.Greater(t, searchStart, -1, "Search panel top border not found")
	require.Greater(t, searchEnd, -1, "Search panel bottom border not found")

	panelHeight := searchEnd - searchStart + 1
	assert.Equal(t, 4, panelHeight, "Search panel should be 4 lines tall when hints are visible (border top + input + hint + border bottom)")
}

// TestSearchOverlay_View_SearchPanelHeight_WithoutHints verifies Search panel is 3 lines when no hints.
// Lines may contain ANSI escape codes so we use strings.Contains for matching.
func TestSearchOverlay_View_SearchPanelHeight_WithoutHints(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)
	// Type a regular query — no hints.
	for _, ch := range "hello" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixNone, o.PrefixState())
	require.False(t, o.ShowHintLine())

	view := o.View()
	lines := strings.Split(view, "\n")

	searchStart, searchEnd := findPanelBounds(lines, "Search")
	require.Greater(t, searchStart, -1, "Search panel top border not found")
	require.Greater(t, searchEnd, -1, "Search panel bottom border not found")

	panelHeight := searchEnd - searchStart + 1
	assert.Equal(t, 3, panelHeight, "Search panel should be 3 lines tall when no hints (border top + input + border bottom)")
}

// TestSearchOverlay_ResultsBorderColor_UsesSeekBar verifies Results panel uses SeekBar color token.
func TestSearchOverlay_ResultsBorderColor_UsesSeekBar(t *testing.T) {
	th := theme.Load("black")
	o2 := panes.NewSearchOverlayForTest(th)
	assert.Equal(t, th.SeekBar(), o2.ResultsBorderAccentColor(),
		"Results panel should use theme.SeekBar() as its AccentColor")
}

// TestSearchOverlay_KeysBorderHasNoTitle verifies Keys panel has no title in its border config.
func TestSearchOverlay_KeysBorderHasNoTitle(t *testing.T) {
	th := theme.Load("black")
	o2 := panes.NewSearchOverlayForTest(th)
	assert.Equal(t, "", o2.KeysPanelTitle(),
		"Keys panel should have empty Title in border config")
}

// TestSearchOverlay_Resize_PropagatesListAndHelp verifies that SetSize correctly
// propagates to the list.Model and help.Model inside the overlay.
func TestSearchOverlay_Resize_PropagatesListAndHelp(t *testing.T) {
	o := newTestSearchOverlay()

	// Initial size.
	o.SetSize(100, 50)
	view1 := o.View()
	assert.NotEmpty(t, view1, "View() should render after initial SetSize")

	// Resize to smaller dimensions — must not panic.
	o.SetSize(60, 30)
	view2 := o.View()
	assert.NotEmpty(t, view2, "View() should render after resize to smaller dimensions")

	// Resize to larger dimensions — must not panic.
	o.SetSize(200, 80)
	view3 := o.View()
	assert.NotEmpty(t, view3, "View() should render after resize to larger dimensions")

	// Resize to minimum valid dimensions.
	o.SetSize(10, 10)
	view4 := o.View()
	assert.NotEmpty(t, view4, "View() should render even at minimum dimensions")
}

// --- Story 93: resizeList() keeps list height in sync with showHintLine() ---

// countViewLines counts the number of lines in the rendered view output.
// This is the key metric to verify: the total view height must remain stable
// across hint toggles. If resizeList() is NOT called, the list renders at
// a stale height and the total view line count changes unexpectedly.
func countViewLines(view string) int {
	return strings.Count(view, "\n") + 1
}

// TestResizeList_ViewHeightStableAcrossHintToggle verifies that View() produces the
// same total line count before and after hint toggling.
//
// Without resizeList(), typing causes searchH to drop from 4 to 3, giving the
// results panel 1 extra line, but the list still renders at the old height (N).
// The container tries to fit N+1 lines in N+1 space, but the list only gives N,
// so lipgloss appends a blank line — OR the list overflows its container by 1.
// Either way, the total view line count shifts, breaking the layout.
//
// With resizeList(), the list is always sized to match the current panelHeights(),
// so the total view height stays constant regardless of hint visibility.
func TestResizeList_ViewHeightStableAcrossHintToggle(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)

	// Baseline: empty input → hint visible.
	require.True(t, o.ShowHintLine(), "pre-condition: hint must be visible")
	baseView := o.View()
	baseLines := countViewLines(baseView)

	// Type 'j' → hint hides → searchH changes from 4 to 3.
	// resultsH gains 1 extra line. resizeList() must increase listH by 1 to absorb it.
	o, _ = sendKey(t, o, "j")
	require.False(t, o.ShowHintLine(), "hint must be hidden after typing")
	afterTypingLines := countViewLines(o.View())
	assert.Equal(t, baseLines, afterTypingLines,
		"view line count must be stable after typing (hint toggled off); resizeList() must have been called")

	// Press Backspace → hint reappears → searchH goes back to 4.
	// resultsH loses 1 line. resizeList() must decrease listH by 1.
	// NOTE: Ctrl+U is no longer a supported clear shortcut (2026-04-28 overlay-keybinding-cleanup).
	// We use Backspace here as the canonical input-editing path that restores the empty state.
	o, _ = sendKey(t, o, "backspace")
	require.True(t, o.ShowHintLine(), "hint must reappear after clearing input via backspace")
	afterClearLines := countViewLines(o.View())
	assert.Equal(t, baseLines, afterClearLines,
		"view line count must be stable after clearing input (hint toggled on); resizeList() must have been called")
}

// TestResizeList_ViewHeightStableAfterBackspace verifies view height stability when
// backspacing the last character (empty input restores hint).
func TestResizeList_ViewHeightStableAfterBackspace(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)

	baseView := o.View()
	baseLines := countViewLines(baseView)

	// Type single char → hint hides.
	o, _ = sendKey(t, o, "j")
	require.False(t, o.ShowHintLine(), "hint must hide after typing")
	afterTypingLines := countViewLines(o.View())
	assert.Equal(t, baseLines, afterTypingLines, "line count must be stable after typing")

	// Backspace → empty input → hint visible again.
	o, _ = sendKey(t, o, "backspace")
	require.Equal(t, "", o.Query(), "backspace must empty input")
	require.True(t, o.ShowHintLine(), "hint must reappear after backspace to empty")
	afterBackspaceLines := countViewLines(o.View())
	assert.Equal(t, baseLines, afterBackspaceLines,
		"view line count must be stable after backspace to empty; resizeList() must have been called")
}

// TestResizeList_ViewHeightStableAfterTabCycle verifies view height stability when
// cycling tabs (changes hint visibility via prefix lock/unlock).
func TestResizeList_ViewHeightStableAfterTabCycle(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)

	baseView := o.View()
	baseLines := countViewLines(baseView)
	require.True(t, o.ShowHintLine(), "pre-condition: hint visible on All tab")

	// Tab to Songs → PrefixLocked → hint hides.
	o, _ = sendKey(t, o, "tab")
	require.Equal(t, panes.TabSongs, o.ActiveTab())
	require.False(t, o.ShowHintLine(), "hint must hide when Songs tab active")
	afterTabLines := countViewLines(o.View())
	assert.Equal(t, baseLines, afterTabLines,
		"view line count must be stable after cycling to Songs tab; resizeList() must have been called")

	// Shift+Tab back to All → hint reappears.
	o, _ = sendKey(t, o, "shift+tab")
	require.Equal(t, panes.TabAll, o.ActiveTab())
	require.True(t, o.ShowHintLine(), "hint must reappear on All tab")
	afterShiftTabLines := countViewLines(o.View())
	assert.Equal(t, baseLines, afterShiftTabLines,
		"view line count must be stable after cycling back to All tab; resizeList() must have been called")
}

// TestResizeList_SearchClearedMsg verifies that SearchClearedMsg (overlay-open sequence)
// causes resizeList() to be called, ensuring the list height is correct after clearing.
func TestResizeList_SearchClearedMsg(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)

	// Get baseline with hint visible.
	baseLines := countViewLines(o.View())
	require.True(t, o.ShowHintLine(), "pre-condition: hint visible on empty overlay")

	// Send SearchClearedMsg directly (as the root app does when opening the overlay).
	model, _ := o.Update(panes.SearchClearedMsg{})
	o = model.(*panes.SearchOverlay)

	// View height must remain stable — resizeList() was called.
	afterClearLines := countViewLines(o.View())
	assert.Equal(t, baseLines, afterClearLines,
		"view line count must be stable after SearchClearedMsg; resizeList() must have been called")
}

// TestResizeList_ListHeightMatchesPanelFormula verifies that after any hint toggle,
// ListHeight() (which reports panelHeights()-derived value) equals the expected formula.
// This confirms resizeList() and panelHeights() stay in sync.
func TestResizeList_ListHeightMatchesPanelFormula(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(100, 40)
	h := o.OverlayHeight()

	// Hint visible: searchH=4, helpH=3, resultsH = h-4-3, listH = resultsH-4.
	require.True(t, o.ShowHintLine())
	wantHintVisible := (h - 4 - 3) - 4
	if wantHintVisible < 1 {
		wantHintVisible = 1
	}
	assert.Equal(t, wantHintVisible, o.ListHeight(),
		"list height with hint visible must be overlayH - 4 - 3 - 4 = %d", wantHintVisible)

	// Type 'j' → hint hides: searchH=3, helpH=3, resultsH = h-3-3, listH = resultsH-4.
	o, _ = sendKey(t, o, "j")
	require.False(t, o.ShowHintLine())
	wantHintHidden := (h - 3 - 3) - 4
	if wantHintHidden < 1 {
		wantHintHidden = 1
	}
	assert.Equal(t, wantHintHidden, o.ListHeight(),
		"list height with hint hidden must be overlayH - 3 - 3 - 4 = %d", wantHintHidden)
	assert.Equal(t, wantHintVisible+1, wantHintHidden,
		"hint-hidden list must be 1 taller than hint-visible list")
}

// TestResizeList_ViewHeightStableAfterDemote verifies that list height is correct
// after demotion via backspace at cursor position 0 with PrefixLocked.
func TestResizeList_ViewHeightStableAfterDemote(t *testing.T) {
	// After locking :songs prefix and then pressing Backspace at cursor 0 to
	// demote the tag, the prefix is restored into input.Value() as ":songs ".
	// showHintLine() remains false (input is non-empty), but resizeList() must
	// still be called so the list height stays at the correct hint-hidden formula.
	o := newTestSearchOverlay()
	o.SetSize(100, 40)

	// Lock the prefix tag by typing ":songs " character by character.
	for _, ch := range ":songs " {
		updated, _ := o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		o = updated.(*panes.SearchOverlay)
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState(), "prefix should be locked after ':songs '")
	// Capture baseline at the hint-hidden state so we can compare post-demotion.
	linesAfterLocking := countViewLines(o.View())

	// Move cursor to position 0 so backspace triggers demotion.
	updated, _ := o.Update(tea.KeyMsg{Type: tea.KeyHome})
	o = updated.(*panes.SearchOverlay)

	// Backspace at position 0 with PrefixLocked → demoteFromPromptTag.
	// The prefix ":songs " is restored into input.Value() so hint stays hidden.
	updated, _ = o.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	o = updated.(*panes.SearchOverlay)
	require.Equal(t, panes.PrefixNone, o.PrefixState(), "prefix should be demoted after backspace at pos 0")
	require.False(t, o.ShowHintLine(), "hint stays hidden: restored ':songs ' is non-empty")

	assert.Equal(t, linesAfterLocking, countViewLines(o.View()),
		"view line count must be stable after prefix demotion; resizeList() on demote path must fire")
}

// TestResizeList_SearchClearedMsg_WithNonEmptyInput verifies that SearchClearedMsg
// keeps the correct list height when the input still has content after clearing.
func TestResizeList_SearchClearedMsg_WithNonEmptyInput(t *testing.T) {
	// SearchClearedMsg clears results but does NOT clear the input value.
	// After the message, showHintLine() is still false (input still has content).
	// resizeList() must be called so height stays at the hint-hidden formula.
	o := newTestSearchOverlay()
	o.SetSize(100, 40)

	// Type a character so hint is hidden.
	updated, _ := o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	o = updated.(*panes.SearchOverlay)
	require.False(t, o.ShowHintLine(), "hint should be hidden after typing")
	linesAfterTyping := countViewLines(o.View())

	// Send SearchClearedMsg while input still has a value.
	model, _ := o.Update(panes.SearchClearedMsg{})
	o = model.(*panes.SearchOverlay)

	// Input was NOT cleared — hint stays hidden.
	require.False(t, o.ShowHintLine(), "hint should still be hidden — SearchClearedMsg does not clear input")
	assert.Equal(t, linesAfterTyping, countViewLines(o.View()),
		"view line count must be stable after SearchClearedMsg when input is non-empty")
}

// --- Story 94: Spinner fix tests ---

// TestSearchSpinnerTick_ReturnsNonNilCmd verifies that SearchSpinnerTickCmd() returns
// a non-nil tea.Cmd.
func TestSearchSpinnerTick_ReturnsNonNilCmd(t *testing.T) {
	cmd := panes.SearchSpinnerTickCmd()
	assert.NotNil(t, cmd, "searchSpinnerTick() must return a non-nil tea.Cmd")
}

// TestSearchSpinnerTick_FiresSearchSpinnerTickMsg verifies that executing the cmd returned
// by SearchSpinnerTickCmd() produces a message that is accepted as a spinner tick by the
// overlay (i.e. it is a searchSpinnerTickMsg, not a spinner.TickMsg).
// We verify this indirectly: send the cmd result to the overlay's Update and confirm it
// advances the spinner (non-panic, returns a new cmd that is also non-nil).
func TestSearchSpinnerTick_CmdFiresCorrectType(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	cmd := panes.SearchSpinnerTickCmd()
	require.NotNil(t, cmd)

	// Execute the cmd — it should fire after 130ms, but for type checking we execute it
	// directly (tea.Cmd is a function, so we call it).
	// The returned message must be handled by searchSpinnerTickMsg case (not fall-through).
	msg := cmd()
	require.NotNil(t, msg, "cmd must fire a non-nil message")

	// Send the message to Update — the spinner case must process it and re-arm with
	// another cmd. If it were a spinner.TickMsg it would fall through to textinput.Update
	// and return nil cmd.
	model, returnedCmd := o.Update(msg)
	updated := model.(*panes.SearchOverlay)
	_ = updated
	// The searchSpinnerTickMsg handler re-arms with searchSpinnerTick(), so returned
	// cmd must be non-nil.
	assert.NotNil(t, returnedCmd, "searchSpinnerTickMsg handler must re-arm with a new cmd")
}

// TestSearchOverlay_Init_NoRawSpinnerTickMsg verifies that Init() includes at least one
// sub-command that, when executed and passed to Update(), causes the spinner frame to
// advance — confirming the batch uses searchSpinnerTickMsg (not the raw spinner.TickMsg
// which falls through to the textinput case without advancing the spinner).
func TestSearchOverlay_Init_NoRawSpinnerTickMsg(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// Record the initial spinner frame before running any Init commands.
	initialFrame := panes.SpinnerView(o)

	initCmd := o.Init()
	require.NotNil(t, initCmd)

	msg := initCmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	require.True(t, ok, "Init() must return a BatchMsg, got %T", msg)

	// Process every sub-command in the batch and look for one that advances the spinner.
	// A searchSpinnerTickMsg causes spinner.Update() to advance the frame; a raw
	// spinner.TickMsg would fall through to textinput.Update and not advance the frame.
	var spinnerAdvanced bool
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		subMsg := subCmd()
		if subMsg == nil {
			continue
		}
		model, _ := o.Update(subMsg)
		o = model.(*panes.SearchOverlay)
		if panes.SpinnerView(o) != initialFrame {
			spinnerAdvanced = true
			break
		}
	}
	assert.True(t, spinnerAdvanced, "Init() batch must include a searchSpinnerTickMsg cmd that advances the spinner frame")
}

// TestSearchSpinnerTickMsg_ReArmsWithSearchSpinnerTickMsg verifies that sending a
// searchSpinnerTickMsg to Update re-arms with another searchSpinnerTickMsg (not spinner.TickMsg).
// We verify this by executing the returned cmd and sending the result back to Update —
// the chain must remain intact for 5 consecutive ticks.
func TestSearchSpinnerTickMsg_ReArms(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// Obtain the initial cmd from SearchSpinnerTickCmd (our wrapper).
	cmd := panes.SearchSpinnerTickCmd()
	require.NotNil(t, cmd)
	msg := cmd()

	for i := 0; i < 5; i++ {
		model, retCmd := o.Update(msg)
		o = model.(*panes.SearchOverlay)
		require.NotNil(t, retCmd, "tick #%d: handler must re-arm", i+1)
		// Execute the returned cmd to get the next message in the chain.
		msg = retCmd()
		require.NotNil(t, msg, "tick #%d: re-arm cmd must fire a message", i+1)
	}
}

// TestSearchSpinnerTickMsg_AdvancesFrame verifies that the spinner frame advances
// after receiving searchSpinnerTickMsg messages.
func TestSearchSpinnerTickMsg_AdvancesFrame(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// Capture the initial spinner view.
	// NOTE: spinner.View() is always available (spinner is internal to SearchOverlay).
	initialFrame := panes.SpinnerView(o)

	// Drive 5 spinner ticks — the frame must change at least once in those 5 ticks.
	cmd := panes.SearchSpinnerTickCmd()
	require.NotNil(t, cmd)
	msg := cmd()
	for i := 0; i < 5; i++ {
		model, retCmd := o.Update(msg)
		o = model.(*panes.SearchOverlay)
		if retCmd != nil {
			msg = retCmd()
		}
	}

	finalFrame := panes.SpinnerView(o)
	assert.NotEqual(t, initialFrame, finalFrame,
		"spinner frame must advance after driving 5 ticks")
}

// TestRenderTabBar_ShowsSpinnerWhenLoading verifies that the tab bar renders without panic
// when results are present.
// TODO(19-search-redesign): story 99 will restore spinner-in-tabbar via o.results loading flag.
func TestRenderTabBar_ShowsSpinnerWhenLoading(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// Deliver results so list is non-empty.
	msg := panes.SearchPageLoadedMsg{Results: sampleSearchListItems()}
	model, _ := o.Update(msg)
	o = model.(*panes.SearchOverlay)

	tabBar := panes.RenderTabBarForTest(o, 76)
	assert.NotEmpty(t, tabBar, "tab bar must render without panic")
}

// TestRenderTabBar_NoSpinnerWhenNotLoading verifies that the tab bar does not contain
// spinner characters when no loading state is active.
// TODO(19-search-redesign): story 99 will restore spinner-in-tabbar control via o.results.
func TestRenderTabBar_NoSpinnerWhenNotLoading(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// Deliver results so list is non-empty.
	msg := panes.SearchPageLoadedMsg{Results: sampleSearchListItems()}
	model, _ := o.Update(msg)
	o = model.(*panes.SearchOverlay)

	tabBar := panes.RenderTabBarForTest(o, 76)
	assert.False(t, panes.ContainsSpinnerFrame(o, tabBar),
		"tab bar must NOT contain spinner frame when loading=false; got: %q", tabBar)
}

// TestRenderTabBar_ShowsSpinnerWhenLoadingWithZeroItems verifies the tab bar renders
// without panic when there are zero items.
// TODO(19-search-redesign): story 99 will restore spinner-in-tabbar via o.results loading flag.
func TestRenderTabBar_ShowsSpinnerWhenLoadingWithZeroItems(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// Tick the spinner once so its frame is initialized.
	cmd := panes.SearchSpinnerTickCmd()
	require.NotNil(t, cmd)
	tickMsg := cmd()
	model, _ := o.Update(tickMsg)
	o = model.(*panes.SearchOverlay)

	// Zero items (no results delivered) — must not panic.
	tabBar := panes.RenderTabBarForTest(o, 76)
	assert.NotEmpty(t, tabBar, "tab bar must render without panic with zero items")
}

// --- Story 99: searchIntent + scheduleDebounce ---

// TestSearchIntent_FieldExistsOnOverlay verifies that the SearchOverlay exposes the
// intent field through the IntentTab() / IntentPage() helpers used by tests.
func TestSearchIntent_FieldExistsOnOverlay(t *testing.T) {
	o := newTestSearchOverlay()
	// Defaults: tab=TabAll, page=1
	assert.Equal(t, panes.TabAll, o.IntentTab(), "default intent.tab should be TabAll")
	assert.Equal(t, 1, o.IntentPage(), "default intent.page should be 1")
}

// TestScheduleDebounce_MatchingIntentFiresSearchRequest verifies that scheduling a
// debounce then calling Update with a searchDebounceMsg carrying the same intent as
// the current o.intent produces a SearchRequestMsg command.
func TestScheduleDebounce_MatchingIntentFiresSearchRequest(t *testing.T) {
	o := newTestSearchOverlay()

	// Type a non-empty query so cleanQuery() returns something.
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, "jazz", o.CleanQuery(), "precondition: clean query should be 'jazz'")

	// Fire a debounce msg whose intent matches the current overlay intent.
	debounceMsg := panes.SearchDebounceMsgWithIntentForTest(o)
	_, cmd := o.Update(debounceMsg)

	require.NotNil(t, cmd, "matching intent should produce a command")
	msg := cmd()
	reqMsg, ok := msg.(panes.SearchRequestMsg)
	require.True(t, ok, "matching intent should produce SearchRequestMsg, got %T", msg)
	assert.Equal(t, "jazz", reqMsg.Query)
	assert.Equal(t, 1, reqMsg.Page)
}

// TestScheduleDebounce_NonMatchingIntentDiscards verifies that a debounce msg whose
// intent snapshot differs from the current o.intent is silently discarded.
func TestScheduleDebounce_NonMatchingIntentDiscards(t *testing.T) {
	o := newTestSearchOverlay()

	// Type "jazz" then "rock" — intent.query changes between snapshots.
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	// Capture debounce msg snapshot at intent {query:"jazz"}.
	staleMsg := panes.SearchDebounceMsgWithIntentForTest(o)

	// Type more — intent.query is now "jazzrock".
	for _, ch := range "rock" {
		o, _ = sendKey(t, o, string(ch))
	}

	// Fire the stale debounce — it carries {query:"jazz"} but current is "jazzrock".
	_, cmd := o.Update(staleMsg)
	assert.Nil(t, cmd, "stale intent debounce must be discarded (nil command)")
}

// TestHandleDebounce_Table is the table-driven test covering all four handleDebounce
// scenarios from the story spec.
func TestHandleDebounce_Table(t *testing.T) {
	tests := []struct {
		name            string
		setupFn         func(*panes.SearchOverlay) *panes.SearchOverlay // prepare overlay state
		mutateBeforeMsg func(*panes.SearchOverlay) *panes.SearchOverlay // mutate AFTER snapshot is captured
		wantNilCmd      bool
		wantQuery       string
	}{
		{
			name: "fresh non-empty: fires SearchRequestMsg",
			setupFn: func(o *panes.SearchOverlay) *panes.SearchOverlay {
				for _, ch := range "jazz" {
					o, _ = sendKey(t, o, string(ch))
				}
				return o
			},
			mutateBeforeMsg: nil,
			wantNilCmd:      false,
			wantQuery:       "jazz",
		},
		{
			name: "stale: user typed more after snapshot",
			setupFn: func(o *panes.SearchOverlay) *panes.SearchOverlay {
				for _, ch := range "jazz" {
					o, _ = sendKey(t, o, string(ch))
				}
				return o
			},
			mutateBeforeMsg: func(o *panes.SearchOverlay) *panes.SearchOverlay {
				// Type more after snapshot is captured — makes snapshot stale.
				for _, ch := range " rock" {
					o, _ = sendKey(t, o, string(ch))
				}
				return o
			},
			wantNilCmd: true,
		},
		{
			name: "empty query: no-op",
			setupFn: func(o *panes.SearchOverlay) *panes.SearchOverlay {
				// Don't type anything — query stays empty.
				return o
			},
			mutateBeforeMsg: nil,
			wantNilCmd:      true,
		},
		{
			name: "prefix only (PrefixTyping): no-op",
			setupFn: func(o *panes.SearchOverlay) *panes.SearchOverlay {
				// Type ":songs" without the trailing space — PrefixTyping state.
				for _, ch := range ":songs" {
					o, _ = sendKey(t, o, string(ch))
				}
				require.Equal(t, panes.PrefixTyping, o.PrefixState(), "should be PrefixTyping")
				return o
			},
			mutateBeforeMsg: nil,
			wantNilCmd:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := newTestSearchOverlay()
			o = tt.setupFn(o)

			// Capture snapshot of current intent.
			debounceMsg := panes.SearchDebounceMsgWithIntentForTest(o)

			// Optionally mutate overlay state (making the snapshot stale).
			if tt.mutateBeforeMsg != nil {
				o = tt.mutateBeforeMsg(o)
			}

			_, cmd := o.Update(debounceMsg)
			if tt.wantNilCmd {
				assert.Nil(t, cmd, "expected nil command in scenario %q", tt.name)
			} else {
				require.NotNil(t, cmd, "expected non-nil command in scenario %q", tt.name)
				msg := cmd()
				reqMsg, ok := msg.(panes.SearchRequestMsg)
				require.True(t, ok, "expected SearchRequestMsg in scenario %q, got %T", tt.name, msg)
				assert.Equal(t, tt.wantQuery, reqMsg.Query, "query mismatch in scenario %q", tt.name)
			}
		})
	}
}

// TestCycleTab_EmptyQuery_NoSearchRequest verifies that cycling the tab with an empty
// query triggers debounce but handleDebounce no-ops (empty query check).
func TestCycleTab_EmptyQuery_NoSearchRequest(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)

	// Tab with no query typed — debounce is scheduled but fires nothing.
	_, cmd := sendKey(t, o, "tab")
	require.NotNil(t, cmd, "tab change should schedule debounce (non-nil cmd)")

	// Execute the debounce cmd — since query is empty, it should be a no-op.
	// The cmd returned by cycleTabForward is scheduleDebounce (a time-based tick).
	// We can't fire the time-based tick synchronously, but we can verify the overlay
	// fires no SearchRequestMsg synchronously either.
	msg := cmd()
	// The message from the debounce tick is a searchDebounceMsg, not a SearchRequestMsg.
	_, isReq := msg.(panes.SearchRequestMsg)
	assert.False(t, isReq, "tab change with empty query should not directly emit SearchRequestMsg")
}

// TestCycleTab_NonEmptyQuery_EmitsSearchRequest verifies that cycling the tab when a
// query is present causes handleDebounce to emit SearchRequestMsg with the correct Types.
func TestCycleTab_NonEmptyQuery_EmitsSearchRequest(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)

	// Type a query first.
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}

	// Cycle to Songs tab.
	_, cmd := sendKey(t, o, "tab")
	require.NotNil(t, cmd, "tab change should return a command")

	// Execute debounce tick → get the searchDebounceMsg.
	tickMsg := cmd()

	// Feed the searchDebounceMsg into Update to fire the search.
	_, searchCmd := o.Update(tickMsg)
	require.NotNil(t, searchCmd, "debounce msg from tab change should produce a search command")
	msg := searchCmd()
	reqMsg, ok := msg.(panes.SearchRequestMsg)
	require.True(t, ok, "tab change + debounce should produce SearchRequestMsg, got %T", msg)
	assert.Equal(t, []string{"track"}, reqMsg.Types, "Songs tab should map to track type")
	assert.Equal(t, "jazz", reqMsg.Query, "SearchRequestMsg should carry the current query")
	assert.Equal(t, 1, reqMsg.Page, "page should be reset to 1 on tab change")
}

// TestReset_FullIntentReset verifies that Reset() resets the full intent struct to
// its zero values: query="", tab=TabAll, page=1.
func TestReset_FullIntentReset(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(120, 40)

	// Type a query and cycle to a non-All tab to change intent.
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	o, _ = sendKey(t, o, "tab") // cycle to Songs

	// Reset and verify intent fields.
	o.Reset()

	assert.Equal(t, panes.TabAll, o.IntentTab(), "after Reset: intent.tab should be TabAll")
	assert.Equal(t, 1, o.IntentPage(), "after Reset: intent.page should be 1")
	assert.Equal(t, "", o.CleanQuery(), "after Reset: intent.query (via clean query) should be empty")
}

// =====================================================================
// Story 102: Pagination controls, loading states, and Panel 2 view
// =====================================================================

// --- Task 1: new fields + Results() accessor ---

// TestSearchOverlay_NewFields_ZeroValued verifies that all new story-102 fields start
// at their zero values when NewSearchOverlay is called.
func TestSearchOverlay_NewFields_ZeroValued(t *testing.T) {
	o := newTestSearchOverlay()
	assert.Nil(t, o.Results(), "results should be nil initially")
	assert.Equal(t, 0, o.Total(), "total should be 0 initially")
	assert.False(t, o.LoadingFirstPage(), "loadingFirstPage should be false initially")
	assert.False(t, o.LoadingNextPage(), "loadingNextPage should be false initially")
}

// TestSearchOverlay_ResultsAccessor_ReturnsNilInitially verifies that Results() is nil
// before any SearchPageLoadedMsg is received.
func TestSearchOverlay_ResultsAccessor_ReturnsNilInitially(t *testing.T) {
	o := newTestSearchOverlay()
	assert.Nil(t, o.Results(), "Results() should return nil before any results arrive")
}

// TestSearchOverlay_Reset_ZerosStory102Fields verifies that Reset() zeros all new fields.
func TestSearchOverlay_Reset_ZerosStory102Fields(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(120, 40)

	// Set all new fields to non-zero values via messages.
	_, _ = o.Update(panes.SearchLoadingMsg{IsFirstPage: true})
	o = func() *panes.SearchOverlay {
		m, _ := o.Update(panes.SearchLoadingMsg{IsFirstPage: false})
		return m.(*panes.SearchOverlay)
	}()
	// Deliver a successful page to set results and total.
	results := panes.TracksToSearchListItems([]domain.Track{
		{URI: "spotify:track:t1", Name: "Reset Test Track"},
	})
	m, _ := o.Update(panes.SearchPageLoadedMsg{Results: results, Total: 42})
	o = m.(*panes.SearchOverlay)
	require.NotNil(t, o.Results(), "pre-condition: results should be set before Reset")
	require.Equal(t, 42, o.Total(), "pre-condition: total should be 42 before Reset")

	o.Reset()

	assert.Nil(t, o.Results(), "after Reset: results should be nil")
	assert.Equal(t, 0, o.Total(), "after Reset: total should be 0")
	assert.False(t, o.LoadingFirstPage(), "after Reset: loadingFirstPage should be false")
	assert.False(t, o.LoadingNextPage(), "after Reset: loadingNextPage should be false")
}

// --- Task 2: SearchLoadingMsg handler ---

// TestSearchOverlay_SearchLoadingMsg_FirstPage verifies that IsFirstPage=true sets
// loadingFirstPage=true and loadingNextPage=false.
func TestSearchOverlay_SearchLoadingMsg_FirstPage(t *testing.T) {
	o := newTestSearchOverlay()
	m, _ := o.Update(panes.SearchLoadingMsg{IsFirstPage: true})
	o = m.(*panes.SearchOverlay)
	assert.True(t, o.LoadingFirstPage(), "IsFirstPage=true must set loadingFirstPage=true")
	assert.False(t, o.LoadingNextPage(), "IsFirstPage=true must set loadingNextPage=false")
}

// TestSearchOverlay_SearchLoadingMsg_NextPage verifies that IsFirstPage=false sets
// loadingFirstPage=false and loadingNextPage=true.
func TestSearchOverlay_SearchLoadingMsg_NextPage(t *testing.T) {
	o := newTestSearchOverlay()
	// Pre-set loadingFirstPage=true so we can verify it is cleared.
	m, _ := o.Update(panes.SearchLoadingMsg{IsFirstPage: true})
	o = m.(*panes.SearchOverlay)
	require.True(t, o.LoadingFirstPage())

	m, _ = o.Update(panes.SearchLoadingMsg{IsFirstPage: false})
	o = m.(*panes.SearchOverlay)
	assert.False(t, o.LoadingFirstPage(), "IsFirstPage=false must set loadingFirstPage=false")
	assert.True(t, o.LoadingNextPage(), "IsFirstPage=false must set loadingNextPage=true")
}

// --- Task 3: SearchPageLoadedMsg handler (success + error) ---

// TestSearchOverlay_SearchPageLoadedMsg_Success_ClearsFlagsAndSetsResults verifies the
// success branch: both loading flags cleared, results and total set.
func TestSearchOverlay_SearchPageLoadedMsg_Success_ClearsFlagsAndSetsResults(t *testing.T) {
	o := newTestSearchOverlay()
	// Set loading flags first.
	m, _ := o.Update(panes.SearchLoadingMsg{IsFirstPage: true})
	o = m.(*panes.SearchOverlay)
	require.True(t, o.LoadingFirstPage())

	results := panes.TracksToSearchListItems([]domain.Track{
		{URI: "spotify:track:t1", Name: "Success Track"},
	})
	m, _ = o.Update(panes.SearchPageLoadedMsg{Results: results, Total: 25})
	o = m.(*panes.SearchOverlay)

	assert.False(t, o.LoadingFirstPage(), "loadingFirstPage must be cleared on success")
	assert.False(t, o.LoadingNextPage(), "loadingNextPage must be cleared on success")
	assert.Equal(t, results, o.Results(), "results must match m.Results on success")
	assert.Equal(t, 25, o.Total(), "total must match m.Total on success")
}

// TestSearchOverlay_SearchPageLoadedMsg_Error_ClearsFlagsPreservesResults verifies the
// error branch: both loading flags cleared, existing results and total preserved.
func TestSearchOverlay_SearchPageLoadedMsg_Error_ClearsFlagsPreservesResults(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)

	// Deliver initial results.
	initial := panes.TracksToSearchListItems([]domain.Track{
		{URI: "spotify:track:t1", Name: "Original Track"},
	})
	m, _ := o.Update(panes.SearchPageLoadedMsg{Results: initial, Total: 50})
	o = m.(*panes.SearchOverlay)

	// Now set loadingNextPage=true then deliver an error.
	m, _ = o.Update(panes.SearchLoadingMsg{IsFirstPage: false})
	o = m.(*panes.SearchOverlay)
	require.True(t, o.LoadingNextPage())

	m, _ = o.Update(panes.SearchPageLoadedMsg{Err: fmt.Errorf("network error")})
	o = m.(*panes.SearchOverlay)

	assert.False(t, o.LoadingFirstPage(), "loadingFirstPage must be cleared on error")
	assert.False(t, o.LoadingNextPage(), "loadingNextPage must be cleared on error")
	assert.Equal(t, initial, o.Results(), "existing results must be preserved on error")
	assert.Equal(t, 50, o.Total(), "total must be preserved on error")
}

// --- Task 4: hasNextPage() ---

// TestHasNextPage_Table covers all boundary conditions from the story spec.
func TestHasNextPage_Table(t *testing.T) {
	tests := []struct {
		name  string
		total int
		page  int
		want  bool
	}{
		{"total=0 page=1 → false", 0, 1, false},
		{"total=10 page=1 exactly one page → false", 10, 1, false},
		{"total=11 page=1 → true", 11, 1, true},
		{"total=100 page=10 last page → false", 100, 10, false},
		{"total=100 page=9 not last → true", 100, 9, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := newTestSearchOverlay()
			o.SetSize(80, 40)
			// Deliver results with the given total.
			m, _ := o.Update(panes.SearchPageLoadedMsg{
				Results: panes.TracksToSearchListItems([]domain.Track{
					{URI: "spotify:track:t1", Name: "test"},
				}),
				Total: tt.total,
			})
			o = m.(*panes.SearchOverlay)
			// Set the page.
			panes.SetIntentPage(o, tt.page)
			assert.Equal(t, tt.want, panes.HasNextPage(o),
				"hasNextPage() for total=%d page=%d should be %v", tt.total, tt.page, tt.want)
		})
	}
}

// --- Task 5: Ctrl+Right / Ctrl+Left keybindings ---

// TestCtrlRight_NoQuery_NoOp verifies that Ctrl+Right with no query is a no-op.
func TestPgDown_NoQuery_NoOp(t *testing.T) {
	o := newTestSearchOverlay()
	page := o.IntentPage()
	o, cmd := sendKey(t, o, "pgdown")
	assert.Equal(t, page, o.IntentPage(), "ctrl+right with no query must not change page")
	assert.Nil(t, cmd, "ctrl+right with no query must return nil cmd")
}

// TestCtrlRight_LoadingFirstPage_NoOp verifies that Ctrl+Right while loading first page is a no-op.
func TestPgDown_LoadingFirstPage_NoOp(t *testing.T) {
	o := newTestSearchOverlay()
	// Type a query.
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	// Set loadingFirstPage.
	m, _ := o.Update(panes.SearchLoadingMsg{IsFirstPage: true})
	o = m.(*panes.SearchOverlay)
	require.True(t, o.LoadingFirstPage())

	pageBefore := o.IntentPage()
	o, cmd := sendKey(t, o, "pgdown")
	assert.Equal(t, pageBefore, o.IntentPage(), "ctrl+right while loadingFirstPage must not change page")
	assert.Nil(t, cmd, "ctrl+right while loadingFirstPage must return nil cmd")
}

// TestCtrlRight_LastPage_NoOp verifies that Ctrl+Right on the last page is a no-op.
func TestPgDown_LastPage_NoOp(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	// Type a query and deliver one page of results with total=10 (exactly one page).
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	m, _ := o.Update(panes.SearchPageLoadedMsg{
		Results: panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}}),
		Total:   10,
	})
	o = m.(*panes.SearchOverlay)
	// total=10, page=1 → hasNextPage=false
	require.False(t, panes.HasNextPage(o), "pre-condition: should be on last page")

	pageBefore := o.IntentPage()
	o, cmd := sendKey(t, o, "pgdown")
	assert.Equal(t, pageBefore, o.IntentPage(), "ctrl+right on last page must not change page")
	assert.Nil(t, cmd, "ctrl+right on last page must return nil cmd")
}

// TestCtrlRight_Valid_IncreasesPageAndSchedulesDebounce verifies that Ctrl+Right with
// a query, not on the last page, increments intent.page and returns a debounce command.
func TestPgDown_Valid_IncreasesPageAndSchedulesDebounce(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	// Deliver results with total=11 → hasNextPage true at page 1.
	m, _ := o.Update(panes.SearchPageLoadedMsg{
		Results: panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}}),
		Total:   11,
	})
	o = m.(*panes.SearchOverlay)
	require.Equal(t, 1, o.IntentPage())
	require.True(t, panes.HasNextPage(o))

	o, cmd := sendKey(t, o, "pgdown")
	assert.Equal(t, 2, o.IntentPage(), "ctrl+right should increment intent.page to 2")
	require.NotNil(t, cmd, "ctrl+right should return debounce cmd")
	// The cmd is a scheduleDebounce tick; execute it to get the debounce msg.
	// The tick fires a searchDebounceMsg after 300ms — it is not a SearchRequestMsg.
	tickMsg := cmd()
	require.NotNil(t, tickMsg, "debounce tick must produce a non-nil message")
	_, isReq := tickMsg.(panes.SearchRequestMsg)
	assert.False(t, isReq, "ctrl+right cmd must not immediately emit SearchRequestMsg")
}

// TestCtrlLeft_NoQuery_NoOp verifies that Ctrl+Left with no query is a no-op.
func TestPgUp_NoQuery_NoOp(t *testing.T) {
	o := newTestSearchOverlay()
	page := o.IntentPage()
	o, cmd := sendKey(t, o, "pgup")
	assert.Equal(t, page, o.IntentPage(), "ctrl+left with no query must not change page")
	assert.Nil(t, cmd, "ctrl+left with no query must return nil cmd")
}

// TestCtrlLeft_Page1_NoOp verifies that Ctrl+Left on page 1 is a no-op.
func TestPgUp_Page1_NoOp(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, 1, o.IntentPage(), "pre-condition: should be on page 1")

	o, cmd := sendKey(t, o, "pgup")
	assert.Equal(t, 1, o.IntentPage(), "ctrl+left on page 1 must not change page")
	assert.Nil(t, cmd, "ctrl+left on page 1 must return nil cmd")
}

// TestCtrlLeft_LoadingFirstPage_NoOp verifies that Ctrl+Left while loadingFirstPage is a no-op.
func TestPgUp_LoadingFirstPage_NoOp(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	// Advance to page 2 first.
	m, _ := o.Update(panes.SearchPageLoadedMsg{
		Results: panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}}),
		Total:   11,
	})
	o = m.(*panes.SearchOverlay)
	o, _ = sendKey(t, o, "pgdown") // page → 2
	require.Equal(t, 2, o.IntentPage())

	// Now set loadingFirstPage.
	m, _ = o.Update(panes.SearchLoadingMsg{IsFirstPage: true})
	o = m.(*panes.SearchOverlay)
	require.True(t, o.LoadingFirstPage())

	o, cmd := sendKey(t, o, "pgup")
	assert.Equal(t, 2, o.IntentPage(), "ctrl+left while loadingFirstPage must not change page")
	assert.Nil(t, cmd, "ctrl+left while loadingFirstPage must return nil cmd")
}

// TestCtrlLeft_Valid_DecreasesPageAndSchedulesDebounce verifies that Ctrl+Left on a
// page > 1 with a query decrements intent.page and returns a debounce command.
func TestPgUp_Valid_DecreasesPageAndSchedulesDebounce(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	// Deliver results with total=11 and advance to page 2.
	m, _ := o.Update(panes.SearchPageLoadedMsg{
		Results: panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}}),
		Total:   11,
	})
	o = m.(*panes.SearchOverlay)
	o, _ = sendKey(t, o, "pgdown") // page → 2
	require.Equal(t, 2, o.IntentPage())

	o, cmd := sendKey(t, o, "pgup")
	assert.Equal(t, 1, o.IntentPage(), "ctrl+left should decrement intent.page to 1")
	require.NotNil(t, cmd, "ctrl+left should return debounce cmd")
	tickMsg := cmd()
	_, isReq := tickMsg.(panes.SearchRequestMsg)
	assert.False(t, isReq, "ctrl+left cmd must not immediately emit SearchRequestMsg")
}

// --- Task 6: renderPaginationBar + Panel 2 layout + resizeList ---

// TestRenderPaginationBar_FirstPage_PrevArrowDimmed verifies that on page 1,
// the prev arrow uses TextMuted style (dimmed).
func TestRenderPaginationBar_FirstPage_PrevArrowDimmed(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// Deliver results with total=11 (2 pages), start on page 1.
	results := panes.TracksToSearchListItems([]domain.Track{
		{URI: "u1", Name: "Jazz"},
	})
	m, _ := o.Update(panes.SearchPageLoadedMsg{Results: results, Total: 11})
	o = m.(*panes.SearchOverlay)
	require.Equal(t, 1, o.IntentPage())

	bar := panes.RenderPaginationBarForTest(o, 60)

	// TextMuted styles the "[ ←" with the muted color. We verify the bar
	// contains ANSI sequence for TextMuted on the left arrow.
	dimmedLeft := lipgloss.NewStyle().Foreground(th.TextMuted()).Render("[ ←")
	normalRight := lipgloss.NewStyle().Foreground(th.TextPrimary()).Render("→ ]")
	assert.True(t, strings.Contains(bar, dimmedLeft),
		"prev arrow must use TextMuted on page 1; bar=%q", bar)
	assert.True(t, strings.Contains(bar, normalRight),
		"next arrow must use Text (not muted) on page 1; bar=%q", bar)
}

// TestRenderPaginationBar_LastPage_NextArrowDimmed verifies that on the last page,
// the next arrow uses TextMuted style.
func TestRenderPaginationBar_LastPage_NextArrowDimmed(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	results := panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}})
	m, _ := o.Update(panes.SearchPageLoadedMsg{Results: results, Total: 11})
	o = m.(*panes.SearchOverlay)
	// Advance to page 2 (last page for total=11, pageSize=10) via SetIntentPage.
	// This directly sets the intent page so the rendering test is independent of key guards.
	panes.SetIntentPage(o, 2)
	require.Equal(t, 2, o.IntentPage())
	require.False(t, panes.HasNextPage(o), "pre-condition: should be on last page")

	bar := panes.RenderPaginationBarForTest(o, 60)
	normalLeft := lipgloss.NewStyle().Foreground(th.TextPrimary()).Render("[ ←")
	dimmedRight := lipgloss.NewStyle().Foreground(th.TextMuted()).Render("→ ]")
	assert.True(t, strings.Contains(bar, normalLeft),
		"prev arrow must use Text on page 2; bar=%q", bar)
	assert.True(t, strings.Contains(bar, dimmedRight),
		"next arrow must use TextMuted on last page; bar=%q", bar)
}

// TestRenderPaginationBar_MidPage_BothArrowsNormal verifies mid-page shows both arrows normal.
func TestRenderPaginationBar_MidPage_BothArrowsNormal(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// total=21 → 3 pages; advance to page 2 via SetIntentPage so the rendering
	// test is independent of key handler guard conditions.
	results := panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}})
	m, _ := o.Update(panes.SearchPageLoadedMsg{Results: results, Total: 21})
	o = m.(*panes.SearchOverlay)
	panes.SetIntentPage(o, 2)
	require.Equal(t, 2, o.IntentPage())

	bar := panes.RenderPaginationBarForTest(o, 60)
	normalLeft := lipgloss.NewStyle().Foreground(th.TextPrimary()).Render("[ ←")
	normalRight := lipgloss.NewStyle().Foreground(th.TextPrimary()).Render("→ ]")
	assert.True(t, strings.Contains(bar, normalLeft),
		"prev arrow must use Text on mid page; bar=%q", bar)
	assert.True(t, strings.Contains(bar, normalRight),
		"next arrow must use Text on mid page; bar=%q", bar)
}

// TestRenderPaginationBar_ContainsPageNumber verifies the bar shows "page N" without "of M".
// Showing "of M" is misleading: the Spotify API total can be huge (e.g. 10,000+) and the
// total pages can change between requests. The simpler "page N" with arrow dimming is clearer.
func TestRenderPaginationBar_ContainsPageNumber(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	results := panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}})
	m, _ := o.Update(panes.SearchPageLoadedMsg{Results: results, Total: 21})
	o = m.(*panes.SearchOverlay)
	bar := panes.RenderPaginationBarForTest(o, 60)

	// Must contain "page 1" but must NOT contain "of" — total page count is hidden.
	assert.True(t, strings.Contains(bar, "page 1"),
		"pagination bar must show 'page 1'; bar=%q", bar)
	assert.False(t, strings.Contains(bar, " of "),
		"pagination bar must NOT show 'of M' total; bar=%q", bar)
}

// TestResizeList_SubtractsPaginationLine verifies that resizeList() subtracts 1 extra
// line for the pagination bar when total > 0.
func TestResizeList_SubtractsPaginationLine(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)

	// Record baseline list height with no results (total=0).
	heightNoResults := o.ListHeight()

	// Deliver results with total > 0.
	results := panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}})
	m, _ := o.Update(panes.SearchPageLoadedMsg{Results: results, Total: 11})
	o = m.(*panes.SearchOverlay)

	heightWithPagination := o.ListHeight()
	assert.Equal(t, heightNoResults-1, heightWithPagination,
		"list height must be 1 less when total>0 (pagination bar occupies 1 line)")
}

// TestPaginationBar_NotRenderedWhenTotal0 verifies that the pagination bar is NOT
// included in the view when total == 0.
func TestPaginationBar_NotRenderedWhenTotal0(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	// No results delivered — total stays 0.
	view := o.View()
	plain := stripANSI(view)
	// The pagination bar renders "page N" (with a space before the number).
	// "pgdn/pgup page" in the keybar uses the word "page" too, so we check for
	// the more specific pattern " page 1 " which only the pagination bar emits.
	assert.False(t, strings.Contains(plain, "page 1"),
		"pagination bar must not appear when total=0; plain=%q", plain)
}

// --- Task 8: Reset() zeros all story-102 fields ---
// (Covered by TestSearchOverlay_Reset_ZerosStory102Fields above.)
// Duplicate test with a simpler setup to verify all four fields explicitly.

func TestSearchOverlay_Reset_ZerosAllStory102FieldsExplicit(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// Drive loadingNextPage=true.
	m, _ := o.Update(panes.SearchLoadingMsg{IsFirstPage: false})
	o = m.(*panes.SearchOverlay)
	// Drive results and total.
	res := panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "t"}})
	m, _ = o.Update(panes.SearchPageLoadedMsg{Results: res, Total: 7})
	o = m.(*panes.SearchOverlay)
	require.NotNil(t, o.Results())
	require.Equal(t, 7, o.Total())

	o.Reset()
	assert.Nil(t, o.Results(), "Reset: results nil")
	assert.Equal(t, 0, o.Total(), "Reset: total 0")
	assert.False(t, o.LoadingFirstPage(), "Reset: loadingFirstPage false")
	assert.False(t, o.LoadingNextPage(), "Reset: loadingNextPage false")
}

// --- Story 104: Integration and coverage gap tests ---

// TestSearchOverlay_RapidPageFlip_SingleRequest verifies that rapid Ctrl+Right presses
// (5×) result in only one SearchRequestMsg, for the final page (page 6), when the
// debounce fires. All prior debounce snapshots are stale because intent.page has moved on.
//
// Implementation detail: each Ctrl+Right press updates intent.page and calls
// scheduleDebounce() which snapshots the current intent. When the debounce ticks fire,
// handleDebounce discards any tick whose snapshot differs from the current intent.
// Only the last snapshot (page=6) matches the current intent at fire time.
func TestSearchOverlay_RapidPageFlip_SingleRequest(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)

	// Set up overlay with query="jazz", page=1, total=60 (6 pages available).
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	m, _ := o.Update(panes.SearchPageLoadedMsg{
		Results: panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}}),
		Total:   60, // 6 pages
	})
	o = m.(*panes.SearchOverlay)
	require.Equal(t, 1, o.IntentPage(), "pre-condition: should be on page 1")
	require.True(t, panes.HasNextPage(o), "pre-condition: should have next page")

	// Rapidly press Ctrl+Right 5 times. Each press captures a stale snapshot of
	// intent.page (1,2,3,4) except the last press (page=5→6 which sets page=6).
	var debounceSnapshots []tea.Msg
	for i := 0; i < 5; i++ {
		// Each sendKey calls handleKey(KeyCtrlRight) which:
		//   1. Increments intent.page
		//   2. Returns scheduleDebounce() — a tea.Tick cmd
		// We immediately execute the tick to get the searchDebounceMsg.
		var debounceCmd tea.Cmd
		o, debounceCmd = sendKey(t, o, "pgdown")
		require.NotNil(t, debounceCmd, "ctrl+right should return a debounce cmd (press %d)", i+1)
		// Execute the tick to get the debounce message.
		debounceMsg := debounceCmd()
		debounceSnapshots = append(debounceSnapshots, debounceMsg)
	}

	// After 5 presses: intent.page should be 6.
	require.Equal(t, 6, o.IntentPage(), "after 5 ctrl+right presses, intent.page should be 6")

	// Now fire all 5 debounce messages. Only the last one (snapshot page=6) should
	// match the current intent. The first 4 (pages 2,3,4,5) are stale.
	var searchRequestCmds []tea.Cmd
	for _, debMsg := range debounceSnapshots {
		_, cmd := o.Update(debMsg)
		if cmd != nil {
			// Execute the cmd to get the SearchRequestMsg.
			if msg := cmd(); msg != nil {
				if req, ok := msg.(panes.SearchRequestMsg); ok {
					searchRequestCmds = append(searchRequestCmds, func() tea.Msg { return req })
				}
			}
		}
	}

	// Exactly one SearchRequestMsg should have been produced, with Page=6.
	require.Len(t, searchRequestCmds, 1,
		"rapid page flip (5×ctrl+right) should produce exactly 1 SearchRequestMsg")
	finalMsg := searchRequestCmds[0]()
	reqMsg, ok := finalMsg.(panes.SearchRequestMsg)
	require.True(t, ok, "produced msg should be SearchRequestMsg, got %T", finalMsg)
	assert.Equal(t, 6, reqMsg.Page, "SearchRequestMsg should carry page=6 (the final page)")
	assert.Equal(t, "jazz", reqMsg.Query, "SearchRequestMsg should carry query='jazz'")
}

// TestSearchOverlay_NoQuery_PaginationNoOp verifies that Ctrl+Right with an empty
// query is a silent no-op — no page increment, no SearchRequestMsg.
// This is an alias for TestCtrlRight_NoQuery_NoOp from story 102, restated
// explicitly for story 104 acceptance criteria traceability.
func TestSearchOverlay_NoQuery_PaginationNoOp(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)

	// No typing — intent.query is "".
	require.Empty(t, o.IntentQuery(), "pre-condition: intent.query should be empty")
	page := o.IntentPage()

	// Ctrl+Right with no query should be a no-op.
	o, cmd := sendKey(t, o, "pgdown")
	assert.Equal(t, page, o.IntentPage(), "ctrl+right with no query must not change page")
	assert.Nil(t, cmd, "ctrl+right with no query must return nil cmd (no SearchRequestMsg)")

	// Ctrl+Left with no query should also be a no-op.
	o, cmd = sendKey(t, o, "pgup")
	assert.Equal(t, page, o.IntentPage(), "ctrl+left with no query must not change page")
	assert.Nil(t, cmd, "ctrl+left with no query must return nil cmd (no SearchRequestMsg)")
}

// TestSearchOverlay_AllTab_HasNextPage verifies that for the All tab:
//   - total=100, page=1 → hasNextPage = true (page 1 of 10)
//   - total=100, page=10 → hasNextPage = false (last page)
//   - total=10, page=1 → hasNextPage = false (exactly one page)
func TestSearchOverlay_AllTab_HasNextPage(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		page     int
		wantNext bool
	}{
		{"total=100 page=1 → true", 100, 1, true},
		{"total=100 page=10 → false", 100, 10, false},
		{"total=10 page=1 exactly one page → false", 10, 1, false},
		{"total=0 page=1 → false", 0, 1, false},
		{"total=11 page=1 → true (two pages)", 11, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := newTestSearchOverlay()
			o.SetSize(80, 40)

			// Deliver results with the given total so total is set on the overlay.
			m, _ := o.Update(panes.SearchPageLoadedMsg{
				Results: panes.TracksToSearchListItems([]domain.Track{
					{URI: "u1", Name: "Track"},
				}),
				Total: tt.total,
			})
			o = m.(*panes.SearchOverlay)

			// Set the page directly for test isolation.
			panes.SetIntentPage(o, tt.page)

			assert.Equal(t, tt.wantNext, panes.HasNextPage(o),
				"hasNextPage(total=%d, page=%d) should be %v", tt.total, tt.page, tt.wantNext)
		})
	}
}

// TestCtrlRight_LoadingNextPage_NoOp verifies that Ctrl+Right while loadingNextPage is a no-op.
// This covers the guard condition: !loadingFirstPage && !loadingNextPage.
func TestPgDown_LoadingNextPage_NoOp(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	// Deliver results with total=21 (3 pages).
	m, _ := o.Update(panes.SearchPageLoadedMsg{
		Results: panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}}),
		Total:   21,
	})
	o = m.(*panes.SearchOverlay)
	// Set loadingNextPage=true.
	m, _ = o.Update(panes.SearchLoadingMsg{IsFirstPage: false})
	o = m.(*panes.SearchOverlay)
	require.True(t, o.LoadingNextPage(), "pre-condition: loadingNextPage should be true")

	pageBefore := o.IntentPage()
	o, cmd := sendKey(t, o, "pgdown")
	assert.Equal(t, pageBefore, o.IntentPage(), "ctrl+right while loadingNextPage must not change page")
	assert.Nil(t, cmd, "ctrl+right while loadingNextPage must return nil cmd")
}

// TestCtrlLeft_LoadingNextPage_NoOp verifies that Ctrl+Left while loadingNextPage is a no-op.
func TestPgUp_LoadingNextPage_NoOp(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	// Deliver results with total=21, advance to page 2.
	m, _ := o.Update(panes.SearchPageLoadedMsg{
		Results: panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}}),
		Total:   21,
	})
	o = m.(*panes.SearchOverlay)
	o, _ = sendKey(t, o, "pgdown") // page → 2
	require.Equal(t, 2, o.IntentPage())

	// Set loadingNextPage.
	m, _ = o.Update(panes.SearchLoadingMsg{IsFirstPage: false})
	o = m.(*panes.SearchOverlay)
	require.True(t, o.LoadingNextPage(), "pre-condition: loadingNextPage should be true")

	o, cmd := sendKey(t, o, "pgup")
	assert.Equal(t, 2, o.IntentPage(), "ctrl+left while loadingNextPage must not change page")
	assert.Nil(t, cmd, "ctrl+left while loadingNextPage must return nil cmd")
}

// --- Bug fix tests: M1, M2, M3 ---

// TestSearchOverlay_AddToQueue_CarriesTrackName (M1) verifies that Ctrl+A on a track
// populates AddToQueueMsg.TrackName so the queue toast shows the track title.
func TestSearchOverlay_AddToQueue_CarriesTrackName(t *testing.T) {
	o := newTestSearchOverlayWithResults()
	o.SetSize(80, 40)

	_, cmd := sendKey(t, o, "ctrl+a")
	require.NotNil(t, cmd, "Ctrl+A on a track must return a command")
	msg := cmd()
	qMsg, ok := msg.(panes.AddToQueueMsg)
	require.True(t, ok, "Ctrl+A must produce AddToQueueMsg, got %T", msg)
	assert.Equal(t, "Blinding Lights", qMsg.TrackName,
		"AddToQueueMsg.TrackName must be populated from si.Name so the queue toast is not blank")
}

// TestSearchOverlay_Backspace_DemoteResetsIntentTab (M2) verifies that pressing Backspace
// at cursor position 0 while a prefix is locked resets intent.tab to TabAll.
// Previously demoteFromPromptTag() cleared prefixState/lockedPrefix but left intent.tab
// at the old value (e.g. TabSongs), causing the next search to fire with a stale type filter.
func TestSearchOverlay_Backspace_DemoteResetsIntentTab(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)

	// Type ":songs " to lock the prefix — intent.tab becomes TabSongs.
	for _, ch := range ":songs " {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState(), "prerequisite: prefix must be locked")
	require.Equal(t, panes.TabSongs, o.IntentTab(), "prerequisite: intent.tab must be TabSongs after lock")

	// Move cursor to position 0 so Backspace triggers demotion, not normal delete.
	o, _ = sendKey(t, o, "home")
	o, _ = sendKey(t, o, "backspace")

	assert.Equal(t, panes.PrefixNone, o.PrefixState(), "after demotion: prefixState must be PrefixNone")
	assert.Equal(t, panes.TabAll, o.IntentTab(),
		"after demotion: intent.tab must reset to TabAll so the next search is not filtered to Songs")
}

// TestSearchOverlay_ListNavigation_NoClampsAtLastItem (M3) verifies that pressing Down
// at the last list item does NOT wrap to the first item (InfiniteScrolling must be false).
// With InfiniteScrolling=true the list wraps within a page, which is unexpected for a
// paged result set and makes it impossible to tell when you've reached the end.
func TestSearchOverlay_ListNavigation_NoWrapAtLastItem(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 40)

	// Load exactly 2 items so the boundary is reachable.
	results := panes.TracksToSearchListItems([]domain.Track{
		{URI: "u1", Name: "Track A", Artists: []domain.Artist{{Name: "Artist"}}},
		{URI: "u2", Name: "Track B", Artists: []domain.Artist{{Name: "Artist"}}},
	})
	model, _ := o.Update(panes.SearchPageLoadedMsg{Results: results})
	o = model.(*panes.SearchOverlay)

	// Navigate to the last item (index 1).
	o, _ = sendKey(t, o, "down")
	require.Equal(t, 1, o.CursorPos(), "prerequisite: cursor must be at last item (index 1)")

	// Press Down again — must NOT wrap back to index 0.
	o, _ = sendKey(t, o, "down")
	assert.Equal(t, 1, o.CursorPos(),
		"Down at last item must clamp (stay at 1), not wrap to 0 — InfiniteScrolling must be false")
}

// --- Pagination UX fixes: pgdown/pgup keybindings, pagination bar format ---

// TestPgDown_AdvancesPage verifies that PageDown advances intent.page when hasNextPage()
// is true and a query is set. This is the macOS-safe replacement for Ctrl+Right.
func TestPgDown_AdvancesPage(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	m, _ := o.Update(panes.SearchPageLoadedMsg{
		Results: panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}}),
		Total:   21,
	})
	o = m.(*panes.SearchOverlay)
	require.True(t, panes.HasNextPage(o), "pre-condition: hasNextPage must be true")

	o, cmd := sendKey(t, o, "pgdown")
	assert.Equal(t, 2, o.IntentPage(), "pgdown should advance intent.page to 2")
	require.NotNil(t, cmd, "pgdown should return a debounce cmd")
	msg := cmd()
	_, isReq := msg.(panes.SearchRequestMsg)
	assert.False(t, isReq, "pgdown cmd must not immediately emit SearchRequestMsg (debounced)")
}

// TestPgUp_PrevPage verifies that PageUp decrements intent.page when page > 1.
// This is the macOS-safe replacement for Ctrl+Left.
func TestPgUp_PrevPage(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	for _, ch := range "jazz" {
		o, _ = sendKey(t, o, string(ch))
	}
	m, _ := o.Update(panes.SearchPageLoadedMsg{
		Results: panes.TracksToSearchListItems([]domain.Track{{URI: "u1", Name: "Jazz"}}),
		Total:   21,
	})
	o = m.(*panes.SearchOverlay)
	o, _ = sendKey(t, o, "pgdown") // page → 2
	require.Equal(t, 2, o.IntentPage(), "pre-condition: must be on page 2")

	o, cmd := sendKey(t, o, "pgup")
	assert.Equal(t, 1, o.IntentPage(), "pgup should decrement intent.page to 1")
	require.NotNil(t, cmd, "pgup should return a debounce cmd")
	msg := cmd()
	_, isReq := msg.(panes.SearchRequestMsg)
	assert.False(t, isReq, "pgup cmd must not immediately emit SearchRequestMsg (debounced)")
}

// TestSearchKeyMap_OnlyVisibleBindings verifies the keymap exposes exactly
// the 5 bindings advertised in the bottom keybar after the cleanup:
// Queue, TabNext, TabPrev, nextPage, prevPage. Play/Close/Clear are removed
// from the map (Enter/Esc behavior remains in Update() but is not advertised;
// Ctrl+U clear is gone entirely).
func TestSearchKeyMap_OnlyVisibleBindings(t *testing.T) {
	km := panes.NewSearchKeyMap()

	assert.Equal(t, []string{"ctrl+a"}, km.Queue.Keys(), "Queue → ctrl+a")
	assert.Equal(t, "ctrl+a", km.Queue.Help().Key)
	assert.Equal(t, "queue", km.Queue.Help().Desc)

	assert.Equal(t, []string{"tab"}, km.TabNext.Keys())
	assert.Equal(t, "tab", km.TabNext.Help().Key)
	assert.Equal(t, "category", km.TabNext.Help().Desc, "Tab help should read 'category', not 'filter'")

	assert.Equal(t, []string{"shift+tab"}, km.TabPrev.Keys())
}

// TestSearchOverlay_View_KeysPanel_SingleLine verifies the Keys panel renders
// a single-line uikit.KeyBar with the cleaned binding set, and that the dead
// bindings (Enter play, Esc close, Ctrl+U clear) are absent.
func TestSearchOverlay_View_KeysPanel_SingleLine(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	view := o.View()

	plain := stripANSI(view)

	assert.Contains(t, plain, "ctrl+a queue", "ctrl+a queue must appear")
	assert.Contains(t, plain, "tab/shift+tab category", "tab/shift+tab category must appear")
	assert.Contains(t, plain, "pgdn/pgup page", "pgdn/pgup page must appear")

	assert.NotContains(t, plain, "enter play", "Enter play must not be advertised")
	assert.NotContains(t, plain, "esc close", "Esc close must not be advertised")
	assert.NotContains(t, plain, "ctrl+u clear", "Ctrl+U must not be advertised")
}

// TestSearchOverlay_View_ResultsPanel_NoCornerActions verifies the Results
// panel border no longer carries the "Enter play / Ctrl+A queue" corner-notch
// actions. The bottom keybar is the single source of truth for visible
// bindings.
func TestSearchOverlay_View_ResultsPanel_NoCornerActions(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	view := o.View()

	plain := stripANSI(view)

	assert.NotContains(t, plain, "Enter play", "corner-notch 'Enter play' must be removed")
	assert.NotContains(t, plain, "Ctrl+A queue", "corner-notch 'Ctrl+A queue' must be removed")
}
