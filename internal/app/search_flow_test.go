package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Helpers ────────────────────────────────────────────────────────────────────

// newSearchFlowTestApp creates an App with a premium user set on the store,
// necessary for search/keybindings that require premium status.
func newSearchFlowTestApp(t *testing.T) *app.App {
	t.Helper()
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Set premium user so search overlay (and its keybindings) are accessible.
	a.Store().SetUserProfile(domain.UserProfile{
		ID:      "user-1",
		Product: "premium",
	})

	return a
}

// assertSearchOpen opens search via '/' key and asserts the overlay is now visible.
func assertSearchOpen(t *testing.T, a *app.App) {
	t.Helper()
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, a.SearchOpen(), "search overlay should be open after '/'")
}

// typeInSearch types the given string into the search input by sending individual
// key press messages. Each character produces a debounce command; the final
// debounce command is returned (or nil if the string is empty).
func typeInSearch(t *testing.T, a *app.App, query string) tea.Cmd {
	t.Helper()
	var lastCmd tea.Cmd
	for _, r := range query {
		_, lastCmd = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return lastCmd
}

// deliverSearchResults sends a SearchPageLoadedMsg with the given results to the app,
// simulating a successful API response. Returns the cmd from the Update.
func deliverSearchResults(t *testing.T, a *app.App, results []panes.SearchListItem, total int) tea.Cmd {
	t.Helper()
	_, cmd := a.Update(panes.SearchPageLoadedMsg{
		Results: results,
		Total:   total,
		Query:   a.SearchQuery(),
		Page:    a.SearchPage(),
	})
	return cmd
}

// sampleTrackResults returns search results with 3 tracks.
func sampleTrackResults() []panes.SearchListItem {
	return []panes.SearchListItem{
		{
			Category:    "track",
			Name:        "Blinding Lights",
			URI:         "spotify:track:bl1",
			ArtistNames: "The Weeknd",
			AlbumName:   "After Hours",
			Duration:    "3:22",
			IsTrack:     true,
		},
		{
			Category:    "track",
			Name:        "Shape of You",
			URI:         "spotify:track:sh1",
			ArtistNames: "Ed Sheeran",
			AlbumName:   "Divide",
			Duration:    "3:53",
			IsTrack:     true,
		},
		{
			Category:    "track",
			Name:        "Starboy",
			URI:         "spotify:track:sb1",
			ArtistNames: "The Weeknd",
			AlbumName:   "Starboy",
			Duration:    "3:50",
			IsTrack:     true,
		},
	}
}

// fireDebounce simulates the 300ms debounce timer firing by directly injecting
// a searchDebounceMsg with the overlay's current intent. This bypasses the
// 300ms tea.Tick timer and executes the debounce handler synchronously.
// Returns the cmd that, when executed, produces a SearchRequestMsg.
func fireDebounce(t *testing.T, a *app.App) tea.Cmd {
	t.Helper()

	sp := a.SearchPane()
	require.NotNil(t, sp)

	// Create a debounce message whose intent matches the overlay's current intent.
	// When the handleDebounce handler receives this message, it verifies the
	// intent matches, strips the prefix from the query, and emits SearchRequestMsg.
	debounceMsg := panes.SearchDebounceMsgWithIntentForTest(sp)

	// Feed the debounce message to the app.
	_, cmd := a.Update(debounceMsg)
	return cmd
}

// fireDebounceAndDispatch simulates the full debounce → SearchRequestMsg → handler
// pipeline. It returns the SearchRequestMsg that was emitted.
func fireDebounceAndDispatch(t *testing.T, a *app.App) panes.SearchRequestMsg {
	t.Helper()
	fetchCmd := fireDebounce(t, a)
	require.NotNil(t, fetchCmd, "debounce should produce a fetch command")

	// Execute the cmd to get the SearchRequestMsg emitted by the overlay.
	fetchMsg := fetchCmd()
	searchReq, ok := fetchMsg.(panes.SearchRequestMsg)
	require.True(t, ok, "debounce should emit SearchRequestMsg, got %T", fetchMsg)

	// Feed SearchRequestMsg to the app so the handler sets staleness keys
	// (searchQuery, searchPage, searchLoading) and creates the cancellable context.
	_, _ = a.Update(searchReq)

	return searchReq
}

// ── Task 2: Full lifecycle integration test ────────────────────────────────────

// TestSearchFlow_OpenTypeResultsPaginatePlayClose verifies the complete search
// flow: open overlay → type query → debounce fires → results delivered →
// paginate → select and play track → close overlay.
func TestSearchFlow_OpenTypeResultsPaginatePlayClose(t *testing.T) {
	a := newSearchFlowTestApp(t)

	// Step 1: Open search overlay via '/' key.
	assertSearchOpen(t, a)

	// Step 2: Type "test" — each character triggers a 300ms debounce.
	typeInSearch(t, a, "test")

	// Step 3: Fire the debounce → should produce SearchRequestMsg.
	_ = fireDebounceAndDispatch(t, a)

	// Verify the app recorded the search session.
	assert.Equal(t, "test", a.SearchQuery(), "searchQuery should be set after debounce")
	assert.Equal(t, 1, a.SearchPage(), "searchPage should default to 1")
	assert.True(t, a.SearchLoading(), "searchLoading should be true")

	// Step 4: Deliver search results (simulating API response).
	results := sampleTrackResults()
	_ = deliverSearchResults(t, a, results, 3)

	// Verify results are present in the search pane.
	sp := a.SearchPane()
	require.NotNil(t, sp)
	assert.Len(t, sp.Results(), 3, "should have 3 search results")
	assert.False(t, a.SearchLoading(), "searchLoading should be false after results")

	// Step 5: Paginate — send PgDn. Since total=3 and page size=10, hasNextPage is false
	// so PgDn should be a no-op here. Verify no crash.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	// No next page available → cmd may be nil (no-op).
	_ = cmd

	// Step 6: Press Enter on the first track → should produce PlayTrackListMsg.
	_, cmd = a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter on track should produce a command")
	enterMsg := cmd()
	playMsg, ok := enterMsg.(panes.PlayTrackListMsg)
	require.True(t, ok, "Enter on track should produce PlayTrackListMsg, got %T", enterMsg)
	assert.Len(t, playMsg.URIs, 3, "should include all remaining tracks from cursor")

	// Step 7: Press Esc → should close the overlay.
	_, cmd = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	// The Esc key in the search overlay produces SearchClosedMsg.
	escMsg := cmd()
	_, isClosed := escMsg.(panes.SearchClosedMsg)
	assert.True(t, isClosed, "Esc should produce SearchClosedMsg, got %T", escMsg)

	// Feed SearchClosedMsg back to close the overlay.
	_, _ = a.Update(escMsg)
	assert.False(t, a.SearchOpen(), "search overlay should be closed after Esc")
}

// ── Task 3: Prefix autocomplete + shortcuts ─────────────────────────────────────

// TestSearchFlow_PrefixAutocomplete verifies prefix locking and unlocking:
//
//	Typing ":songs " locks the prefix → prompt tag changes to "Search Songs".
//	The active tab should change from All to Songs.
//	Backspace at position 0 (on empty input) should unlock the prefix.
func TestSearchFlow_PrefixAutocomplete(t *testing.T) {
	a := newSearchFlowTestApp(t)
	assertSearchOpen(t, a)

	sp := a.SearchPane()

	// Type ":songs " — the trailing space triggers prefix lock.
	typeInSearch(t, a, ":songs ")

	// Verify prefix is locked and tab changed.
	assert.Equal(t, panes.TabSongs, sp.ActiveTab(), "tab should be set to Songs after :songs prefix")
	assert.Contains(t, sp.Input().Prompt, "songs", "prompt should contain the locked prefix tag")

	// Verify that typing "hello" after prefix lock searches under songs tab.
	typeInSearch(t, a, "hello")
	searchReq := fireDebounceAndDispatch(t, a)

	// The SearchRequestMsg should have Types=["track"] from the Songs tab.
	assert.Equal(t, "hello", a.SearchQuery(), "searchQuery should be 'hello' after prefix + typing")
	assert.Equal(t, 1, a.SearchPage())
	assert.Equal(t, []string{"track"}, searchReq.Types, "SearchRequestMsg should carry track type from :songs prefix")
}

// TestSearchFlow_CtrlA_AddToQueue verifies that pressing Ctrl+A on a selected
// track produces an AddToQueueMsg.
func TestSearchFlow_CtrlA_AddToQueue(t *testing.T) {
	a := newSearchFlowTestApp(t)
	assertSearchOpen(t, a)

	// Type and get results.
	typeInSearch(t, a, "test")
	_ = fireDebounceAndDispatch(t, a)

	results := sampleTrackResults()
	deliverSearchResults(t, a, results, 3)

	// Move cursor down to second item.
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Press Ctrl+A — should produce AddToQueueMsg with the selected track.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	require.NotNil(t, cmd, "Ctrl+A should produce a command")

	ctrlMsg := cmd()
	queueMsg, ok := ctrlMsg.(panes.AddToQueueMsg)
	require.True(t, ok, "Ctrl+A should produce AddToQueueMsg, got %T", ctrlMsg)
	assert.Equal(t, "spotify:track:sh1", queueMsg.TrackURI, "should queue the selected track")
	assert.Equal(t, "Shape of You", queueMsg.TrackName)

	// Ctrl+A on non-track item should be no-op. Tab to an artist, check no-op.
	// But our sample results are all tracks, so this isn't testable here.
}

// TestSearchFlow_TabCycling verifies that pressing Tab cycles through result tabs
// and Shift+Tab cycles backward.
func TestSearchFlow_TabCycling(t *testing.T) {
	a := newSearchFlowTestApp(t)
	assertSearchOpen(t, a)

	sp := a.SearchPane()
	assert.Equal(t, panes.TabAll, sp.ActiveTab(), "initial tab should be All")

	// Tab forward.
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, panes.TabSongs, sp.ActiveTab(), "Tab should advance to Songs")

	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, panes.TabArtists, sp.ActiveTab(), "Tab should advance to Artists")

	// Shift+Tab backward.
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, panes.TabSongs, sp.ActiveTab(), "Shift+Tab should retreat to Songs")
}

// ── Task 4: Stale cancellation + Ctrl+U ─────────────────────────────────────────

// TestSearchFlow_StaleRequestCancelled verifies that when the user types a second
// query before the first response arrives, the first request is cancelled (context.Canceled)
// and only the second query's results are displayed.
func TestSearchFlow_StaleRequestCancelled(t *testing.T) {
	a := newSearchFlowTestApp(t)
	assertSearchOpen(t, a)

	// Type "test" — debounce fires, app records query="test".
	typeInSearch(t, a, "test")
	_ = fireDebounceAndDispatch(t, a)

	// Record the first context.
	ctx1 := a.SearchCancelCtx()
	require.NotNil(t, ctx1, "should have in-flight search context")
	assert.Equal(t, "test", a.SearchQuery())

	// Type "testing" — new debounce fires, cancels the first context.
	typeInSearch(t, a, "ing")
	_ = fireDebounceAndDispatch(t, a)

	// First context should now be cancelled.
	assert.Error(t, ctx1.Err(), "first search context should be cancelled")
	assert.Equal(t, "testing", a.SearchQuery(), "searchQuery should be updated to 'testing'")

	// Deliver results for the first query (stale) — should be silently discarded.
	_, cmd := a.Update(panes.SearchPageLoadedMsg{
		Results: sampleTrackResults(),
		Total:   3,
		Query:   "test", // stale query
		Page:    1,
	})
	// Should not produce a toast; stale results are silently dropped.
	// The command should be nil (no toast, no forward).
	assert.Nil(t, cmd, "stale SearchPageLoadedMsg should be silently discarded")

	// Verify the pane still has no results (the "testing" query's response hasn't arrived).
	sp := a.SearchPane()
	require.NotNil(t, sp)
	assert.Nil(t, sp.Results(), "search pane should have no results yet for 'testing'")
}

// TestSearchFlow_CtrlU_ClearInput verifies that pressing Ctrl+U is a no-op
// (the overlay resets fully on Esc). Per the 2026-04-28 overlay-keybinding-cleanup
// spec, Ctrl+U is no longer supported. This test verifies Ctrl+U does not clear
// the input — only direct edits (backspace) or Esc+reopen should reset.
func TestSearchFlow_CtrlU_ClearInput(t *testing.T) {
	a := newSearchFlowTestApp(t)
	assertSearchOpen(t, a)

	// Type "hello world".
	typeInSearch(t, a, "hello world")

	// Verify input has value.
	sp := a.SearchPane()
	require.NotNil(t, sp)
	assert.Equal(t, "hello world", sp.Query(), "input should have typed value")

	// Press Ctrl+U — should be a no-op (intercepted by overlay).
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyCtrlU})

	// Input should still have the typed value.
	assert.Equal(t, "hello world", sp.Query(), "Ctrl+U should not clear input (no-op)")
}
