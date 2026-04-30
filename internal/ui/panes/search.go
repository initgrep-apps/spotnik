package panes

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// SearchTab identifies the active category tab in the search overlay.
type SearchTab int

// Tab constants define the five result category tabs shown in Panel 2.
const (
	TabAll       SearchTab = iota // 0: show all result types
	TabSongs                      // 1: songs/tracks only
	TabArtists                    // 2: artists only
	TabAlbums                     // 3: albums only
	TabPlaylists                  // 4: playlists only
	// NumTabs is the total number of category tabs. Exported for tests.
	NumTabs = 5

	// SearchPageSize is the number of results fetched per search page.
	// Must equal app.SearchPageSize (both reference the same Spotify API limit).
	// Defined here so the panes package can compute hasNextPage() without
	// importing the app package (which would create a circular dependency).
	SearchPageSize = 10
)

// TabLabels holds the display label for each tab.
var TabLabels = [NumTabs]string{"All", "Songs", "Artists", "Albums", "Playlists"}

// TabToAPITypes returns the Spotify API type strings for the given tab.
// Exported for tests.
func TabToAPITypes(tab SearchTab) []string {
	switch tab {
	case TabSongs:
		return []string{"track"}
	case TabArtists:
		return []string{"artist"}
	case TabAlbums:
		return []string{"album"}
	case TabPlaylists:
		return []string{"playlist"}
	default: // TabAll
		return []string{"track", "artist", "album", "playlist"}
	}
}

// searchIntent captures the full desired search state at a point in time.
// All four triggers (type, Tab, Ctrl+Right, Ctrl+Left) write to this struct
// and call scheduleDebounce(). The debounce tick carries a snapshot; if the
// snapshot differs from the current intent at fire time, the tick is stale and discarded.
type searchIntent struct {
	query string
	tab   SearchTab
	page  int
}

// searchDebounceMsg is the internal tick fired by scheduleDebounce.
// It is never routed to app.go — handled entirely within SearchOverlay.Update().
type searchDebounceMsg struct {
	intent searchIntent
}

// searchSpinnerTickMsg wraps bubbles/spinner.TickMsg so the search overlay's
// spinner ticks are never intercepted by the global spinner.TickMsg handler in
// app/handlers.go (which is reserved for the onboarding spinner).
// This is the same isolation pattern established in story 94.
type searchSpinnerTickMsg spinner.TickMsg

// wrapSearchSpinnerTick takes a cmd returned by bubbles/spinner.Update (or Init)
// and wraps any resulting spinner.TickMsg in searchSpinnerTickMsg so it stays
// invisible to the global handler. Must be applied to every spinner command
// in the chain — not just the first one — because bubbles re-arms its own tick
// on every update.
func wrapSearchSpinnerTick(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if t, ok := msg.(spinner.TickMsg); ok {
			return searchSpinnerTickMsg(t)
		}
		return msg
	}
}

// SearchClosedMsg is emitted when the user presses Esc, signalling the root
// app model to close the overlay and restore the previous pane focus.
type SearchClosedMsg struct{}

// SearchRequestMsg is emitted when the debounce fires and the query is non-empty.
// The root app model receives it and dispatches the actual Spotify API call.
// Types carries the Spotify API type filter derived from the locked prefix (e.g. ["track"]
// for ":songs"). When empty the app handler defaults to all four types.
// Page is the 1-based page number to fetch; incremented/decremented by Ctrl+Right/Left.
type SearchRequestMsg struct {
	Query string
	Types []string
	Page  int // 1-based page number; reflects intent.page at debounce fire time
}

// searchKeyMap holds only the bindings advertised in the search overlay's
// bottom keybar. Enter (play) and Esc (close) are handled in Update() but
// not advertised — they are universal overlay conventions. Ctrl+U is no
// longer wired (see the 2026-04-28 overlay-keybinding-cleanup spec).
type searchKeyMap struct {
	Queue    key.Binding
	TabNext  key.Binding
	TabPrev  key.Binding
	nextPage key.Binding
	prevPage key.Binding
}

// NewSearchKeyMap creates the default keybindings for the search overlay.
// Exported for tests. Only includes bindings shown in the bottom keybar.
func NewSearchKeyMap() searchKeyMap {
	return searchKeyMap{
		Queue: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "queue"),
		),
		TabNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "category"),
		),
		TabPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev"),
		),
		nextPage: key.NewBinding(
			// pgdown is the sole pagination key. ctrl+right was removed because macOS
			// intercepts it at the OS level for Spaces/Desktop navigation.
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "next"),
		),
		prevPage: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "prev"),
		),
	}
}

// NOTE: SearchPageLoadedMsg and SearchLoadingMsg are defined in messages.go alongside
// all other shared message types used between the app layer and the overlay.

// SearchOverlay is the floating search UI model. It is layered above the
// three-pane view while open — it does not replace any pane.
//
// The overlay renders as three separate bordered panels stacked vertically:
//   - Panel 1 (Search): text input
//   - Panel 2 (Results): tab bar + separator + scrollable results list
//   - Panel 3 (Keys): uikit.KeyBar single-line keybinding strip
type SearchOverlay struct {
	theme  theme.Theme
	input  textinput.Model
	sp     *uikit.Spinner
	keyMap searchKeyMap

	// resultList is the bubbles/list model used in the results panel.
	// Story 84 will wire actual items into it; here it is initialized and sized.
	resultList list.Model

	width  int
	height int

	// results holds the current page of search result items delivered via SearchPageLoadedMsg.
	// This avoids reading domain.SearchResult from the store, keeping the ui/api boundary clean.
	// nil until the first successful search response arrives.
	results []SearchListItem

	// total is the total result count across all types/pages, as reported by the API.
	// Used by hasNextPage() and renderPaginationBar(). Zero until results arrive.
	total int

	// loadingFirstPage is true while the first page of a new query is in flight.
	// When true, results==nil and we show a centered spinner only.
	loadingFirstPage bool

	// loadingNextPage is true while a subsequent page fetch is in flight.
	// When true, results!=nil and we show a spinner line above the existing list.
	loadingNextPage bool

	// intent is the single source of truth for what the user currently wants to search.
	// All four triggers (type, Tab, Ctrl+Right, Ctrl+Left) write to this struct and call
	// scheduleDebounce(). The debounce tick carries a snapshot; stale ticks are discarded.
	intent searchIntent

	// prefixState tracks which stage of command-prefix entry the user is in.
	// See search_prefix.go for the state machine.
	prefixState prefixState

	// lockedPrefix holds the confirmed prefix once prefixState == PrefixLocked (e.g. ":songs").
	lockedPrefix string

	// placeholderIdx cycles through searchPlaceholders (0..3) on a 2-second tick.
	// The tick stops when the user starts typing and restarts when the input is cleared.
	placeholderIdx int

	// lastSetListH is the most recent height passed to resultList.SetSize().
	// Tracked so tests can verify resizeList() was called with the correct value.
	lastSetListH int
}

// NewSearchOverlay constructs a SearchOverlay wired to the given theme.
// The text input is focused by default.
func NewSearchOverlay(t theme.Theme) *SearchOverlay {
	ti := textinput.New()
	// Start with the first cycling placeholder — the tick will advance it every 2s.
	ti.Placeholder = searchPlaceholders[0]
	// Placeholder uses Info() color so it looks like an actionable suggestion,
	// not a passive muted hint.
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(t.Info())
	// Enable native inline ghost completion for command prefixes.
	// Each suggestion has a trailing space so that acceptance immediately
	// triggers parsePrefix() and the lock + Prompt-tag promotion.
	ti.ShowSuggestions = true
	ti.SetSuggestions([]string{":songs ", ":artists ", ":albums ", ":playlists "})
	// Ghost/completion text appears dim so it doesn't compete with the typed input.
	ti.CompletionStyle = lipgloss.NewStyle().Foreground(t.TextMuted())
	ti.Focus()

	km := NewSearchKeyMap()

	// Initialize the list.Model with the custom SearchItemDelegate.
	// All built-in chrome is disabled since we render our own tab bar, separator,
	// and help bar outside the list.
	delegate := NewSearchItemDelegate(t)
	rl := list.New(nil, delegate, 0, 0)
	rl.SetShowTitle(false)
	rl.SetShowFilter(false)
	rl.SetShowStatusBar(false)
	rl.SetShowHelp(false)
	rl.SetShowPagination(false)
	// InfiniteScrolling must be false: the list should clamp at the last item rather
	// than wrapping to the first. Wrapping within a page makes it impossible for the
	// user to tell they've reached the end; Ctrl+Right/Left handles cross-page navigation.
	rl.InfiniteScrolling = false

	return &SearchOverlay{
		theme: t,
		input: ti,
		// Spinner text is empty so each render site can supply its own context label
		// ("Searching…" on first page load, "Loading…" on next-page fetch).
		// uikit.Spinner.View() returns "frame " (frame + space) when text is empty,
		// so callers append their label directly: sp.View() + "Searching…".
		sp:         uikit.NewSpinner("", t),
		keyMap:     km,
		resultList: rl,
		intent:     searchIntent{query: "", tab: TabAll, page: 1},
	}
}

// InputFocused returns true if the text input currently has focus.
// Exposed for tests.
func (o *SearchOverlay) InputFocused() bool {
	return o.input.Focused()
}

// Query returns the current search query string from the text input.
// Exposed for tests.
func (o *SearchOverlay) Query() string {
	return o.input.Value()
}

// ActiveTab returns the currently active category tab.
// Exported for tests.
func (o *SearchOverlay) ActiveTab() SearchTab {
	return o.intent.tab
}

// OverlayWidth returns the computed overlay width (70% of terminal width, min 40).
// Exported for tests.
func (o *SearchOverlay) OverlayWidth() int {
	return o.overlayWidth()
}

// OverlayHeight returns the computed overlay height (80% of terminal height, min 15).
// Exported for tests.
func (o *SearchOverlay) OverlayHeight() int {
	return o.overlayHeight()
}

// CursorPos returns the current cursor position in the list.
// After Story 84, this reflects the list.Model cursor index.
// Exposed for tests.
func (o *SearchOverlay) CursorPos() int {
	return o.resultList.Index()
}

// ResultListItems returns the current items in the result list.
// Exported for tests that need to inspect list contents directly.
func (o *SearchOverlay) ResultListItems() []list.Item {
	return o.resultList.Items()
}

// Input returns the underlying textinput model.
// Exported for tests that need to inspect Prompt or other input fields after Reset().
func (o *SearchOverlay) Input() textinput.Model {
	return o.input
}

// Results returns the current page of search results.
// Returns nil until the first successful search response arrives.
func (o *SearchOverlay) Results() []SearchListItem { return o.results }

// Total returns the total result count across all pages as reported by the API.
// Exported for tests.
func (o *SearchOverlay) Total() int { return o.total }

// LoadingFirstPage returns true while the first page of a new query is in flight.
// Exported for tests.
func (o *SearchOverlay) LoadingFirstPage() bool { return o.loadingFirstPage }

// LoadingNextPage returns true while a subsequent page fetch is in flight.
// Exported for tests.
func (o *SearchOverlay) LoadingNextPage() bool { return o.loadingNextPage }

// hasNextPage returns true when there are more pages to fetch.
// The condition is: total > 0 AND current page * page size < total.
func (o *SearchOverlay) hasNextPage() bool {
	return o.total > 0 && o.intent.page*SearchPageSize < o.total
}

// Reset restores the overlay to its initial empty state, as if it had just been
// constructed. Called by the root app when the overlay is opened (openSearch) to
// guarantee a fresh start every session, regardless of the previous session's state.
// Reset does not call resizeList() because the terminal size may not be set at the
// moment of Reset (overlay not yet rendered); the first SetSize() call will size the
// list correctly.
func (o *SearchOverlay) Reset() {
	o.input.SetValue("")
	o.input.Prompt = "> "
	o.input.Placeholder = searchPlaceholders[0]
	o.placeholderIdx = 0
	o.intent = searchIntent{query: "", tab: TabAll, page: 1}
	o.prefixState = PrefixNone
	o.lockedPrefix = ""
	o.results = nil
	o.total = 0
	o.loadingFirstPage = false
	o.loadingNextPage = false
	o.resultList.SetItems(nil)
}

// panelHeights returns the computed heights for the three overlay panels:
// searchH (3 or 4 depending on hint line), resultsH (fills remaining), helpH (always 3).
func (o *SearchOverlay) panelHeights() (searchH, resultsH, helpH int) {
	searchH = 3
	if o.showHintLine() {
		searchH = 4
	}
	helpH = 3 // 1 content row (single-line KeyBar) + top/bottom border
	totalH := o.overlayHeight()
	resultsH = totalH - searchH - helpH
	if resultsH < 5 {
		resultsH = 5
	}
	return
}

// SetSize updates the overlay dimensions (forwarded from root app on resize).
// Dimensions are propagated to the list.Model inner area.
func (o *SearchOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height

	o.resizeList()
}

// resizeList recomputes the list dimensions from the current panelHeights() and
// applies them via resultList.SetSize(). Must be called after any state change that
// could affect showHintLine() (typing, backspace, tab cycle, SearchClearedMsg),
// total (which controls whether the pagination bar occupies a line), or loading state
// (which controls whether the spinner line occupies a line above the list).
// Without this call, the list renders at a stale height whenever the hint line toggles,
// causing visual artifacts (duplicate lines, misaligned borders).
func (o *SearchOverlay) resizeList() {
	w := o.overlayWidth()
	_, resultsH, _ := o.panelHeights()

	// Subtract 1 line for the pagination bar when total > 0.
	paginationLine := 0
	if o.total > 0 {
		paginationLine = 1
	}

	// Subtract 1 line for the spinner line when loadingNextPage (spinner above list).
	spinnerLine := 0
	if o.loadingNextPage {
		spinnerLine = 1
	}

	// Inner list dimensions: subtract results border (2) + tab bar (1) + separator (1) + optional lines.
	listW := w - 2
	if listW < 1 {
		listW = 1
	}
	listH := resultsH - 4 - paginationLine - spinnerLine
	if listH < 1 {
		listH = 1
	}
	o.resultList.SetSize(listW, listH)
	o.lastSetListH = listH
}

// Init starts the cursor blink loop, placeholder ticker, spinner tick, and emits
// SearchClearedMsg so each search session begins with a clean state.
// The spinner's init command is wrapped in wrapSearchSpinnerTick so its ticks
// use searchSpinnerTickMsg instead of the raw spinner.TickMsg, preventing
// the global onboarding spinner handler (handlers.go) from silently dropping them.
func (o *SearchOverlay) Init() tea.Cmd {
	clearCmd := func() tea.Msg { return SearchClearedMsg{} }
	return tea.Batch(textinput.Blink, wrapSearchSpinnerTick(o.sp.Init()), clearCmd, searchPlaceholderTick())
}

// Update handles all messages for the search overlay.
func (o *SearchOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case searchPlaceholderTickMsg:
		// Advance the cycling placeholder only when the input is empty.
		// When the user has typed something the tick is not re-armed, so cycling stops.
		// Re-arming resumes naturally when the user clears the input by editing (backspace, etc.).
		if o.input.Value() == "" && o.prefixState == PrefixNone {
			o.placeholderIdx = (o.placeholderIdx + 1) % len(searchPlaceholders)
			o.input.Placeholder = searchPlaceholders[o.placeholderIdx]
			return o, searchPlaceholderTick()
		}
		return o, nil

	case searchDebounceMsg:
		return o.handleDebounce(m)

	case SearchClearedMsg:
		// Root app has cleared the store; clear local overlay state too so the
		// results panel shows the empty-query hint rather than stale items.
		// Also clear total and loading flags so the pagination bar does not linger.
		o.results = nil
		o.total = 0
		o.loadingFirstPage = false
		o.loadingNextPage = false
		o.resultList.SetItems(nil)
		// Re-apply list dimensions: clearing the input makes showHintLine() return
		// true (searchH=4), shrinking resultsH by 1. resizeList() keeps the list
		// height in sync so the panel layout does not overflow.
		o.resizeList()
		return o, nil

	case SearchLoadingMsg:
		// Set the correct loading flag based on whether this is the first page.
		// Clears the other flag so the two states are mutually exclusive.
		if m.IsFirstPage {
			o.loadingFirstPage = true
			o.loadingNextPage = false
		} else {
			o.loadingFirstPage = false
			o.loadingNextPage = true
		}
		// Re-apply list dimensions: loadingNextPage toggles the spinner line, which
		// affects the available height for the results list.
		o.resizeList()
		return o, nil

	case SearchPageLoadedMsg:
		// Always clear loading flags — the spinner must not stay visible after
		// any response (success or error). App.go handles the error toast.
		o.loadingFirstPage = false
		o.loadingNextPage = false
		if m.Err != nil {
			// Keep existing results visible (previous page preserved on page-change error).
			// loading flags were cleared above — recalculate list height so the spinner
			// line is removed from the layout.
			o.resizeList()
			return o, nil
		}
		// Save the pre-converted list items so we never read api types from the store.
		o.results = m.Results
		o.total = m.Total
		// Rebuild the list from locally cached items.
		o.rebuildListItems()
		// Re-apply list dimensions: total may have changed, affecting pagination line.
		o.resizeList()
		return o, nil

	case tea.KeyMsg:
		return o.handleKey(m)

	case searchSpinnerTickMsg:
		// Drive the spinner frame forward. Re-wrap the returned tick command so
		// each subsequent tick also arrives as searchSpinnerTickMsg (not raw
		// spinner.TickMsg), keeping it invisible to the global onboarding handler.
		var spCmd tea.Cmd
		o.sp, spCmd = o.sp.Update(spinner.TickMsg(m))
		return o, wrapSearchSpinnerTick(spCmd)
	}

	// Forward all other messages to the text input (cursor blink, etc.).
	// NOTE: raw spinner.TickMsg is intentionally NOT forwarded here — the search
	// overlay's spinner ticks arrive exclusively as searchSpinnerTickMsg above.
	var inputCmd tea.Cmd
	o.input, inputCmd = o.input.Update(msg)
	return o, inputCmd
}

// handleDebounce is called when a searchDebounceMsg arrives. It discards stale
// ticks (intent has changed since the tick was scheduled) and no-ops on empty
// or prefix-only queries.
func (o *SearchOverlay) handleDebounce(m searchDebounceMsg) (tea.Model, tea.Cmd) {
	// Stale: user has moved on since this tick was scheduled.
	if m.intent != o.intent {
		return o, nil
	}
	// No-op: nothing to search.
	query := o.cleanQuery()
	if query == "" || o.prefixState == PrefixTyping {
		return o, nil
	}
	types := searchTypesForTab(o.intent.tab)
	page := o.intent.page
	return o, func() tea.Msg {
		return SearchRequestMsg{Query: query, Types: types, Page: page}
	}
}

// handleKey processes key events for the overlay.
func (o *SearchOverlay) handleKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEsc:
		return o, func() tea.Msg { return SearchClosedMsg{} }

	case tea.KeyEnter:
		return o.handleEnter()

	case tea.KeyTab:
		if o.prefixState == PrefixTyping {
			// Forward Tab to textinput so it can accept the ghost suggestion
			// (textinput.KeyMap.AcceptSuggestion is bound to Tab by default).
			var cmd tea.Cmd
			o.input, cmd = o.input.Update(m)
			// Re-parse after acceptance: if the suggestion was accepted (e.g. ":songs "),
			// parsePrefix() will detect PrefixLocked and we promote the prefix to the Prompt.
			o.parsePrefix()
			if o.prefixState == PrefixLocked {
				o.promoteToPromptTag()
			}
			return o, cmd
		}
		return o.cycleTabForward()

	case tea.KeyShiftTab:
		return o.cycleTabBackward()

	case tea.KeyUp:
		var cmd tea.Cmd
		o.resultList, cmd = o.resultList.Update(m)
		return o, cmd

	case tea.KeyDown:
		var cmd tea.Cmd
		o.resultList, cmd = o.resultList.Update(m)
		return o, cmd

	case tea.KeyCtrlA:
		return o.handleAddToQueue()

	case tea.KeyPgDown:
		// Next page — only when there is a query, not loading any page, and there is a next page.
		if o.intent.query == "" || o.loadingFirstPage || o.loadingNextPage || !o.hasNextPage() {
			return o, nil
		}
		o.intent.page++
		return o, o.scheduleDebounce()

	case tea.KeyPgUp:
		// Prev page — only when there is a query, not loading any page, and not on page 1.
		if o.intent.query == "" || o.loadingFirstPage || o.loadingNextPage || o.intent.page <= 1 {
			return o, nil
		}
		o.intent.page--
		return o, o.scheduleDebounce()

	case tea.KeyCtrlU:
		// No-op: Ctrl+U is no longer a supported shortcut per the 2026-04-28
		// overlay-keybinding-cleanup spec. We intercept it here to prevent the
		// underlying textinput from treating it as a readline "kill to beginning of line"
		// (which would clear the input). Clearing only happens via direct edits.
		return o, nil

	case tea.KeyBackspace:
		// When a prefix is locked (in Prompt) and the cursor is at position 0,
		// demote the tag back into the input value so the user can edit the prefix.
		if o.prefixState == PrefixLocked && o.input.Position() == 0 {
			o.demoteFromPromptTag()
			// Re-apply list dimensions after demotion: the prefix tag is removed,
			// which may change showHintLine() (PrefixNone, empty input → hint visible).
			o.resizeList()
			return o, nil
		}
		// Otherwise let textinput handle backspace normally.
		var cmd tea.Cmd
		o.input, cmd = o.input.Update(m)
		// Re-parse prefix state after backspace so it can unlock/re-type.
		// NOTE: parsePrefix() guard skips re-parsing if already locked+promoted,
		// so we only need to force-reset when the prefix is NOT locked.
		if o.prefixState != PrefixLocked {
			o.parsePrefix()
		}
		// After a demote→re-lock cycle, the Prompt is still "> " but parsePrefix
		// has set PrefixLocked again. Promote so cleanQuery() works correctly.
		if o.prefixState == PrefixLocked && o.input.Prompt == "> " {
			o.promoteToPromptTag()
		}
		// Re-apply list dimensions: backspace may change showHintLine() (e.g. input
		// cleared → hint reappears → searchH changes from 3 to 4 → resultsH shrinks).
		o.resizeList()
		// Update intent.query to reflect the current input value, then update page to 1.
		o.intent.query = o.input.Value()
		o.intent.page = 1
		if o.prefixState == PrefixTyping {
			// Still editing the prefix — don't fire debounce yet.
			return o, cmd
		}
		debounceCmd := o.scheduleDebounce()
		// Restart placeholder tick when backspace clears input to empty.
		if o.input.Value() == "" && o.prefixState == PrefixNone {
			return o, tea.Batch(cmd, debounceCmd, searchPlaceholderTick())
		}
		return o, tea.Batch(cmd, debounceCmd)

	default:
		// Regular typing — update input, re-parse prefix, schedule debounce.
		var cmd tea.Cmd
		o.input, cmd = o.input.Update(m)
		o.parsePrefix()
		if o.prefixState == PrefixLocked && o.input.Prompt == "> " {
			// Prefix just locked (e.g. user typed the trailing space) and the Prompt
			// tag hasn't been applied yet — promote the prefix to the Prompt field.
			o.promoteToPromptTag()
		}
		// Re-apply list dimensions: typing may change showHintLine() (e.g. first
		// char typed on empty input → hint hides → searchH changes from 4 to 3 →
		// resultsH gains 1 line). resizeList() keeps the list height in sync.
		o.resizeList()
		// Update intent.query to reflect the current input value, then reset page to 1.
		o.intent.query = o.input.Value()
		o.intent.page = 1
		if o.prefixState == PrefixTyping {
			// User is still typing the command prefix — don't fire debounce yet.
			return o, cmd
		}
		debounceCmd := o.scheduleDebounce()
		return o, tea.Batch(cmd, debounceCmd)
	}
}

// handleEnter plays the currently selected result without closing the overlay.
// The overlay stays open so users can continue browsing after playing a track.
// Only Esc emits SearchClosedMsg.
// Reads the selected item from the list.Model delegate; no-op when list is empty.
func (o *SearchOverlay) handleEnter() (tea.Model, tea.Cmd) {
	selected := o.resultList.SelectedItem()
	if selected == nil {
		return o, nil
	}
	si, ok := selected.(SearchListItem)
	if !ok || si.URI == "" {
		return o, nil
	}
	if si.IsTrack {
		// Build the URI list from the selected result onward, collecting only tracks.
		// This gives Spotify the full remaining track context so the queue fills correctly.
		idx := o.resultList.Index()
		uris := make([]string, 0)
		for _, item := range o.results[idx:] {
			if item.IsTrack && item.URI != "" {
				uris = append(uris, item.URI)
			}
		}
		return o, func() tea.Msg { return PlayTrackListMsg{URIs: uris} }
	}
	uri := si.URI
	return o, func() tea.Msg { return PlayContextMsg{ContextURI: uri} }
}

// handleAddToQueue adds the currently selected track to the queue.
// Uses list.SelectedItem() to get the selection; no-op when list is empty
// or when the selected item is not a track.
func (o *SearchOverlay) handleAddToQueue() (tea.Model, tea.Cmd) {
	selected := o.resultList.SelectedItem()
	if selected == nil {
		return o, nil
	}
	si, ok := selected.(SearchListItem)
	if !ok || !si.IsTrack || si.URI == "" {
		// Selected item is not a track — Ctrl+A is a no-op.
		return o, nil
	}
	uri := si.URI
	name := si.Name
	return o, func() tea.Msg { return AddToQueueMsg{TrackURI: uri, TrackName: name} }
}

// cycleTabForward advances the active tab, wrapping from the last tab back to TabAll.
// It updates o.intent.tab and resets o.intent.page to 1, then schedules a debounce.
// handleDebounce fires SearchRequestMsg if the query is non-empty at tick time.
// NOTE: Tab cycling is only reachable when prefixState is not PrefixTyping (Tab routing
// in handleKey sends PrefixTyping → textinput suggestion acceptance instead). When a
// prefix is locked the clean query (prefix stripped) is used so the API never sees raw ":songs kk".
func (o *SearchOverlay) cycleTabForward() (tea.Model, tea.Cmd) {
	o.intent.tab = SearchTab((int(o.intent.tab) + 1) % NumTabs)
	o.intent.page = 1
	o.syncInputToTab()
	o.rebuildListItems()
	// Re-apply list dimensions: tab cycling changes prefixState (and therefore
	// showHintLine()), which changes searchH and hence resultsH.
	o.resizeList()
	return o, o.scheduleDebounce()
}

// cycleTabBackward retreats the active tab, wrapping from TabAll back to the last tab.
// It updates o.intent.tab and resets o.intent.page to 1, then schedules a debounce.
// handleDebounce fires SearchRequestMsg if the query is non-empty at tick time.
func (o *SearchOverlay) cycleTabBackward() (tea.Model, tea.Cmd) {
	o.intent.tab = SearchTab((int(o.intent.tab) + NumTabs - 1) % NumTabs)
	o.intent.page = 1
	o.syncInputToTab()
	o.rebuildListItems()
	// Re-apply list dimensions: tab cycling changes prefixState (and therefore
	// showHintLine()), which changes searchH and hence resultsH.
	o.resizeList()
	return o, o.scheduleDebounce()
}

// View renders the search overlay as three separate bordered panels stacked vertically:
//   - Panel 1 (Search): text input + optional prefix hint line (inside the border)
//   - Panel 2 (Results): tab bar + separator + scrollable results list
//   - Panel 3 (Keys): help keybinding bar (no title)
//
// All three panels sit flush — no margin lines between them. The ╰╯ of one panel
// directly touches the ╭╮ of the next. Hints render inside Panel 1, not between panels.
func (o *SearchOverlay) View() string {
	w := o.overlayWidth()

	searchBarH, resultsH, helpH := o.panelHeights()
	searchPanel := o.renderSearchPanel(w, searchBarH)
	helpPanel := o.renderHelpPanel(w, helpH)
	resultsPanel := o.renderResultsPanel(w, resultsH)

	// Compose: all three panels flush (no margin between them).
	return lipgloss.JoinVertical(lipgloss.Left,
		searchPanel,
		resultsPanel,
		helpPanel,
	)
}

// renderSearchPanel builds Panel 1: the search input box.
// Height is 3 lines (border top + input + border bottom) when no hints are showing,
// or 4 lines (border top + input + hint line + border bottom) when hints are visible.
// The prefix hint line renders INSIDE the panel border, not between panels.
func (o *SearchOverlay) renderSearchPanel(w, h int) string {
	innerWidth := w - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	// Build inner content: text input + optional hint line.
	inputView := o.input.View()
	hintLine := o.renderPrefixHints(innerWidth)

	var inner string
	if hintLine != "" {
		// 4-line panel: input row + hint row inside the border.
		inner = lipgloss.JoinVertical(lipgloss.Left, inputView, hintLine)
	} else {
		inner = inputView
	}
	inner = lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Height(h - 2).MaxHeight(h - 2).
		Render(inner)

	chrome := uikit.OverlayChrome{
		Width:  w,
		Height: h,
		Title:  "Search",
		Theme:  o.theme,
	}
	return chrome.Render(inner)
}

// renderResultsPanel builds Panel 2: the tab bar, separator, optional spinner line,
// scrollable results list, and optional pagination bar.
//
// Panel 2 layout (top to bottom):
//
//	tab bar        (1 line)
//	separator      (1 line)
//	spinner line   (0 or 1 line, loadingNextPage only)
//	list           (fills remaining height)
//	pagination bar (1 line, only when total > 0)
func (o *SearchOverlay) renderResultsPanel(w, h int) string {
	innerWidth := w - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	innerHeight := h - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Tab bar row (1 line).
	tabBar := o.renderTabBar(innerWidth)

	// Separator row (1 line) — thin dashes in TextMuted color.
	separator := lipgloss.NewStyle().
		Foreground(o.theme.TextMuted()).
		Render(strings.Repeat("─", innerWidth))

	// Spinner line (1 line) — only visible while loadingNextPage.
	// When loadingFirstPage, the entire results area shows only the spinner.
	var spinnerLine string
	if o.loadingNextPage {
		spinnerLine = lipgloss.NewStyle().
			Foreground(o.theme.TextMuted()).
			Render(o.sp.View() + "Loading" + uikit.GlyphFor(uikit.GlyphEllipsis, uikit.ActiveMode()))
	}

	// Calculate extra lines consumed by optional elements.
	fixedLines := 2 // tab bar + separator
	if o.loadingNextPage {
		fixedLines++
	}
	paginationLine := 0
	if o.total > 0 {
		paginationLine = 1
	}

	// Results area fills the remaining lines.
	resultsAreaH := innerHeight - fixedLines - paginationLine
	if resultsAreaH < 1 {
		resultsAreaH = 1
	}

	var resultsArea string
	if o.loadingFirstPage {
		// First-page loading: centered spinner only, no list.
		centered := lipgloss.NewStyle().
			Foreground(o.theme.TextMuted()).
			Width(innerWidth).Align(lipgloss.Center).
			Render(o.sp.View() + "Searching" + uikit.GlyphFor(uikit.GlyphEllipsis, uikit.ActiveMode()))
		resultsArea = lipgloss.NewStyle().
			Width(innerWidth).MaxWidth(innerWidth).
			Height(resultsAreaH).MaxHeight(resultsAreaH).
			Render(centered)
	} else {
		resultsContent := o.renderResults(innerWidth, resultsAreaH)
		resultsArea = lipgloss.NewStyle().
			Width(innerWidth).MaxWidth(innerWidth).
			Height(resultsAreaH).MaxHeight(resultsAreaH).
			Render(resultsContent)
	}

	// Assemble inner content lines.
	lines := []string{tabBar, separator}
	if o.loadingNextPage {
		lines = append(lines, spinnerLine)
	}
	lines = append(lines, resultsArea)
	if o.total > 0 {
		lines = append(lines, o.renderPaginationBar(innerWidth))
	}

	inner := strings.Join(lines, "\n")
	inner = lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Height(innerHeight).MaxHeight(innerHeight).
		Render(inner)

	// NOTE: OverlayChrome uses Accent() and Focused=true, consolidating the three panels
	// under a uniform glyph-aware border. The former SeekBar()/TextMuted() accent distinction
	// was a visual-hierarchy hint; OverlayChrome is the canonical overlay primitive.
	chrome := uikit.OverlayChrome{
		Width:  w,
		Height: h,
		Title:  "Results",
		Theme:  o.theme,
	}
	return chrome.Render(inner)
}

// renderTabBar renders the tab selector row inside Panel 2.
// The active tab is shown with brackets and highlight styling; inactive tabs use TextMuted.
// When store.SearchLoading() is true, a spinner frame is appended to the right side so
// re-searches are visible even when existing results remain on screen.
func (o *SearchOverlay) renderTabBar(innerWidth int) string {
	var parts []string
	for i := 0; i < NumTabs; i++ {
		tab := SearchTab(i)
		label := TabLabels[tab]
		if tab == o.intent.tab {
			// Active tab: brackets + selected colors.
			style := lipgloss.NewStyle().
				Foreground(o.theme.SelectedFg()).
				Background(o.theme.SelectedBg())
			parts = append(parts, style.Render("["+label+"]"))
		} else {
			// Inactive tab: muted text.
			style := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
			parts = append(parts, style.Render(label))
		}
	}
	tabLine := strings.Join(parts, "  ")

	// NOTE: spinner display during loading is rendered in renderResultsPanel via the
	// loadingNextPage spinner line and loadingFirstPage full-panel spinner.

	// Pad/truncate to exactly innerWidth.
	return lipgloss.NewStyle().Width(innerWidth).MaxWidth(innerWidth).Render(tabLine)
}

// renderHelpPanel builds Panel 3: the keybinding hint bar.
// Height is fixed at 3 (top border + single content line + bottom border).
// Renders a uikit.KeyBar over the visible binding subset. Title is empty
// because the binding content is self-explanatory, and the dim TextMuted
// border lets the panel recede.
func (o *SearchOverlay) renderHelpPanel(w, h int) string {
	innerWidth := w - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	bar := uikit.KeyBar{
		Bindings: o.hintBindings(),
		Theme:    o.theme,
	}.Render()

	inner := lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Height(h - 2).MaxHeight(h - 2).
		Render(bar)

	chrome := uikit.OverlayChrome{
		Width:  w,
		Height: h,
		Title:  "",
		Theme:  o.theme,
	}
	return chrome.Render(inner)
}

// hintBindings returns the synthetic key.Binding list rendered by the bottom
// keybar. The composite Key strings ("tab/shift+tab", "pgdn/pgup") exist
// purely for display and are NEVER matched against tea.KeyMsg input — real
// key handling lives in handleKey().
func (o *SearchOverlay) hintBindings() []key.Binding {
	return []key.Binding{
		o.keyMap.Queue,
		key.NewBinding(key.WithHelp("tab/shift+tab", "category")),
		key.NewBinding(key.WithHelp("pgdn/pgup", "page")),
	}
}

// renderPaginationBar renders the [ ←  page N of M  → ] line.
// Arrows are dimmed (TextMuted) when navigation in that direction is not possible.
// The bar is centered within the given width w.
func (o *SearchOverlay) renderPaginationBar(w int) string {
	// Show only the current page number — not "of M".
	// The Spotify API total can be very large (e.g. 10,000+ results) and showing
	// "page 1 of 1000" is misleading when we only ever fetch 10 items per page.
	// The → arrow dims when hasNextPage() is false, giving the same directional signal
	// without the confusing denominator.
	center := fmt.Sprintf("  page %d  ", o.intent.page)

	prevStyle := o.theme.TextPrimary()
	nextStyle := o.theme.TextPrimary()
	if o.intent.page <= 1 {
		prevStyle = o.theme.TextMuted()
	}
	if !o.hasNextPage() {
		nextStyle = o.theme.TextMuted()
	}

	m := uikit.ActiveMode()
	left := lipgloss.NewStyle().Foreground(prevStyle).Render("[ " + uikit.GlyphFor(uikit.GlyphArrowLeft, m))
	right := lipgloss.NewStyle().Foreground(nextStyle).Render(uikit.GlyphFor(uikit.GlyphArrowRight, m) + " ]")
	bar := lipgloss.JoinHorizontal(lipgloss.Center, left, center, right)
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(bar)
}

// renderResults builds the results area content inside Panel 2.
// When results are loaded, delegates to resultList.View() for scrollable rendering.
// Falls back to an EmptyState primitive when no results are present.
// NOTE: Loading states (loadingFirstPage / loadingNextPage) are handled by the
// caller renderResultsPanel, which renders the appropriate spinner before calling here.
func (o *SearchOverlay) renderResults(w, h int) string {
	if len(o.resultList.Items()) == 0 {
		return uikit.EmptyState{
			Text:   "Type to search tracks, artists, albums...",
			Width:  w,
			Height: h,
			Theme:  o.theme,
		}.Render()
	}

	// Render the list component — it handles scrolling and selection highlighting.
	return o.resultList.View()
}

// rebuildListItems repopulates resultList from o.results.
// The full results slice is shown directly — tab-filtered views from the prefix
// state machine narrow the API request types upstream via SearchRequestMsg.Types.
func (o *SearchOverlay) rebuildListItems() {
	if o.results != nil {
		o.rebuildFromResults()
	}
}

// rebuildFromResults rebuilds the list from the locally cached SearchListItems.
// Items were pre-converted by commands.go before delivery via SearchPageLoadedMsg,
// so no further domain conversion is needed here.
func (o *SearchOverlay) rebuildFromResults() {
	if o.results == nil {
		return
	}
	// Convert []SearchListItem to []list.Item for the bubbles/list component.
	items := make([]list.Item, len(o.results))
	for i, r := range o.results {
		items[i] = r
	}
	o.resultList.SetItems(items)
}

// overlayWidth returns the effective overlay width: 70% of terminal width, minimum 40.
func (o *SearchOverlay) overlayWidth() int {
	w := o.width * 70 / 100
	if w < 40 {
		w = 40
	}
	return w
}

// overlayHeight returns the effective overlay height: 80% of terminal height, minimum 15.
func (o *SearchOverlay) overlayHeight() int {
	h := o.height * 80 / 100
	if h < 15 {
		h = 15
	}
	return h
}

// scheduleDebounce snapshots the current intent and returns a 300ms tick.
// When the tick fires, handleDebounce compares the snapshot to the current
// intent — if they differ, the tick is discarded (the user has since moved on).
func (o *SearchOverlay) scheduleDebounce() tea.Cmd {
	snapshot := o.intent
	return tea.Tick(300*time.Millisecond, func(_ time.Time) tea.Msg {
		return searchDebounceMsg{intent: snapshot}
	})
}

// searchTypesForTab returns the Spotify API type strings for the given SearchTab.
// Used by handleDebounce to derive the types filter from o.intent.tab.
func searchTypesForTab(tab SearchTab) []string {
	return TabToAPITypes(tab)
}

// SetTheme updates the theme reference for runtime theme switching.
// Propagates to the list delegate (badge/selection colors), spinner style,
// placeholder/completion styles, and the active Prompt tag if one is set.
func (o *SearchOverlay) SetTheme(th theme.Theme) {
	o.theme = th
	// Update the list delegate so badge and selection colors use the new theme.
	o.resultList.SetDelegate(NewSearchItemDelegate(th))
	// Reconstruct spinner so loading indicator uses the new theme's colors.
	// Empty text: each render site appends its own context label.
	o.sp = uikit.NewSpinner("", th)
	// Update placeholder style so the cycling hints use the new Info() color.
	o.input.PlaceholderStyle = lipgloss.NewStyle().Foreground(th.Info())
	// Update completion/ghost text style so suggestions use the new TextMuted() color.
	o.input.CompletionStyle = lipgloss.NewStyle().Foreground(th.TextMuted())
	// Re-render the Prompt tag if a prefix is locked, applying the new theme colors.
	if o.prefixState == PrefixLocked {
		o.promoteToPromptTag()
	}
}

// --- Test helpers (exported only for test packages) ---

// NewSearchOverlayForTest creates a SearchOverlay for use in tests that need to
// inspect configuration without wiring additional state.
func NewSearchOverlayForTest(t theme.Theme) *SearchOverlay {
	return NewSearchOverlay(t)
}

// ResultsBorderAccentColor returns the AccentColor that renderResultsPanel() uses
// in its BorderConfig. Exported for tests verifying per-panel border color assignments.
func (o *SearchOverlay) ResultsBorderAccentColor() lipgloss.Color {
	return o.theme.SeekBar()
}

// KeysPanelTitle returns the Title string used in renderHelpPanel()'s BorderConfig.
// Exported for tests verifying that the Keys panel has no title label.
func (o *SearchOverlay) KeysPanelTitle() string {
	return "" // Keys panel title is always empty (self-explanatory keybinding content)
}

// SearchDebounceMsgForTest creates a searchDebounceMsg for use in tests.
// The intent is constructed with the given query, TabAll, and page 1.
// This allows legacy tests to inject debounce messages without wiring the full
// overlay intent — for testing stale detection based on query alone.
func SearchDebounceMsgForTest(query string) tea.Msg {
	return searchDebounceMsg{intent: searchIntent{query: query, tab: TabAll, page: 1}}
}

// SearchDebounceMsgWithIntentForTest creates a searchDebounceMsg whose intent is a
// snapshot of the overlay's current o.intent. Used by tests that need accurate
// stale-tick detection (e.g. TestScheduleDebounce_MatchingIntentFiresSearchRequest).
func SearchDebounceMsgWithIntentForTest(o *SearchOverlay) tea.Msg {
	return searchDebounceMsg{intent: o.intent}
}

// SearchSpinnerInitCmd returns the overlay's spinner Init() cmd wrapped in
// wrapSearchSpinnerTick so the resulting TickMsg arrives as searchSpinnerTickMsg
// and is handled by Update's explicit case — not intercepted by the global
// onboarding spinner handler in handlers.go.
// Exported for tests that drive the spinner tick chain manually.
func SearchSpinnerInitCmd(o *SearchOverlay) tea.Cmd {
	return wrapSearchSpinnerTick(o.sp.Init())
}

// SpinnerView returns the current rendered spinner frame string — exported for tests
// that need to detect frame advancement.
func SpinnerView(o *SearchOverlay) string {
	return o.sp.View()
}

// RenderTabBarForTest calls renderTabBar with the given inner width and returns the result.
// Exported so tests can inspect tab bar content in isolation without calling full View().
func RenderTabBarForTest(o *SearchOverlay, innerWidth int) string {
	return o.renderTabBar(innerWidth)
}

// ContainsSpinnerFrame returns true when s contains the current spinner view string.
// Used in tests to verify the spinner appears (or does not appear) in rendered output.
func ContainsSpinnerFrame(o *SearchOverlay, s string) bool {
	frame := o.sp.View()
	return frame != "" && strings.Contains(s, frame)
}

// CallRebuildListItems calls rebuildListItems on the overlay — exported for tests.
func CallRebuildListItems(o *SearchOverlay) {
	o.rebuildListItems()
}

// ListItemCount returns the number of items in the overlay's result list — exported for tests.
func ListItemCount(o *SearchOverlay) int {
	return len(o.resultList.Items())
}

// SetActiveTab sets the overlay's active tab — exported for tests.
func SetActiveTab(o *SearchOverlay, tab SearchTab) {
	o.intent.tab = tab
}

// SetListCursor moves the list cursor to the given index — exported for tests.
func SetListCursor(o *SearchOverlay, index int) {
	// Send down key presses to advance to target index.
	items := o.resultList.Items()
	if index >= len(items) {
		return
	}
	// Reset to start by setting a fresh list to get cursor at 0.
	// Then advance by sending down key messages directly.
	for i := 0; i < index; i++ {
		o.resultList, _ = o.resultList.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
}

// ListCursorIndex returns the current list cursor index — exported for tests.
func ListCursorIndex(o *SearchOverlay) int {
	return o.resultList.Index()
}

// NewTestList creates a minimal list.Model for delegate rendering tests.
// The list has one item at index 0 so Render can detect selection state.
func NewTestList(d SearchItemDelegate) list.Model {
	l := list.New([]list.Item{
		SearchListItem{Category: "track", Name: "placeholder", URI: "u"},
	}, d, 40, 10)
	l.SetShowTitle(false)
	l.SetShowFilter(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	return l
}

// SearchPlaceholderTickMsgForTest creates a searchPlaceholderTickMsg for use in tests.
// This allows the test package to inject placeholder tick messages without exposing
// the unexported type in the production API.
func SearchPlaceholderTickMsgForTest() tea.Msg {
	return searchPlaceholderTickMsg{}
}

// IntentTab returns the tab stored in the current search intent — exported for tests.
func (o *SearchOverlay) IntentTab() SearchTab {
	return o.intent.tab
}

// IntentPage returns the page number stored in the current search intent — exported for tests.
func (o *SearchOverlay) IntentPage() int {
	return o.intent.page
}

// PlaceholderIdx returns the current placeholder cycle index — exported for tests.
func (o *SearchOverlay) PlaceholderIdx() int {
	return o.placeholderIdx
}

// Placeholder returns the current textinput placeholder string — exported for tests.
func (o *SearchOverlay) Placeholder() string {
	return o.input.Placeholder
}

// ListHeight returns the last height applied to the result list via resultList.SetSize().
// This is tracked in lastSetListH so tests can directly verify resizeList() was called
// with the correct value after any state change that affects showHintLine().
// When resizeList() has not been called yet (before SetSize), this returns 0.
func (o *SearchOverlay) ListHeight() int {
	return o.lastSetListH
}

// InputShowSuggestions returns whether the textinput has ShowSuggestions enabled — exported for tests.
func (o *SearchOverlay) InputShowSuggestions() bool {
	return o.input.ShowSuggestions
}

// InputAvailableSuggestions returns the available suggestions — exported for tests.
func (o *SearchOverlay) InputAvailableSuggestions() []string {
	return o.input.AvailableSuggestions()
}

// PlaceholderStyleFg returns the foreground color of the input's PlaceholderStyle — exported for tests.
func (o *SearchOverlay) PlaceholderStyleFg() lipgloss.TerminalColor {
	return o.input.PlaceholderStyle.GetForeground()
}

// CompletionStyleFg returns the foreground color of the input's CompletionStyle — exported for tests.
func (o *SearchOverlay) CompletionStyleFg() lipgloss.TerminalColor {
	return o.input.CompletionStyle.GetForeground()
}

// SyncInputToTab is the exported wrapper for syncInputToTab — used in tests.
func (o *SearchOverlay) SyncInputToTab() {
	o.syncInputToTab()
}

// PromoteToPromptTag is the exported wrapper for promoteToPromptTag — used in tests.
func (o *SearchOverlay) PromoteToPromptTag() {
	o.promoteToPromptTag()
}

// DemoteFromPromptTag is the exported wrapper for demoteFromPromptTag — used in tests.
func (o *SearchOverlay) DemoteFromPromptTag() {
	o.demoteFromPromptTag()
}

// HasNextPage exposes the private hasNextPage() method for tests.
func HasNextPage(o *SearchOverlay) bool { return o.hasNextPage() }

// SetIntentPage sets the overlay's intent.page — exported for tests that need to
// simulate a specific page without triggering key handlers.
func SetIntentPage(o *SearchOverlay, page int) { o.intent.page = page }

// IntentQuery returns the query string stored in the current search intent — exported for tests.
func (o *SearchOverlay) IntentQuery() string { return o.intent.query }

// RenderPaginationBarForTest exposes renderPaginationBar for tests.
func RenderPaginationBarForTest(o *SearchOverlay, w int) string {
	return o.renderPaginationBar(w)
}

// SearchDebounceMsgForTestType is a type alias for searchDebounceMsg, exported solely
// so tests can use type assertions against the debounce message type.
// It is only used in type-switch assertions; never construct it directly.
type SearchDebounceMsgForTestType = searchDebounceMsg
