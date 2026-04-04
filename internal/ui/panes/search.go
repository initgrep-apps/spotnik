package panes

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
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

// searchSpinnerTickMsg is used by the bubbles/spinner to advance its frame.
type searchSpinnerTickMsg spinner.TickMsg

// searchKeyMap defines all keybindings shown in the help bar (Panel 3).
// It implements the bubbles/help KeyMap interface.
type searchKeyMap struct {
	Play     key.Binding
	Queue    key.Binding
	TabNext  key.Binding
	TabPrev  key.Binding
	Close    key.Binding
	Clear    key.Binding
	nextPage key.Binding
	prevPage key.Binding
}

// ShortHelp returns 8 bindings for the compact help bar view.
// Clear (ctrl+u) is included so users can discover the clear-search shortcut.
// nextPage and prevPage are included so users can discover pagination keys.
func (k searchKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Play, k.Queue, k.TabNext, k.TabPrev, k.Clear, k.Close, k.nextPage, k.prevPage}
}

// FullHelp returns all 8 bindings grouped in a single column.
func (k searchKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Play, k.Queue, k.TabNext, k.TabPrev, k.Close, k.Clear, k.nextPage, k.prevPage}}
}

// NewSearchKeyMap creates the default keybindings for the search overlay.
// Exported for tests.
func NewSearchKeyMap() searchKeyMap {
	return searchKeyMap{
		Play: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "play"),
		),
		Queue: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "queue"),
		),
		TabNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "filter"),
		),
		TabPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "clear"),
		),
		nextPage: key.NewBinding(
			key.WithKeys("ctrl+right"),
			key.WithHelp("ctrl+→", "next page"),
		),
		prevPage: key.NewBinding(
			key.WithKeys("ctrl+left"),
			key.WithHelp("ctrl+←", "prev page"),
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
//   - Panel 3 (Keys): bubbles/help keybinding bar
type SearchOverlay struct {
	theme   theme.Theme
	input   textinput.Model
	spinner spinner.Model
	help    help.Model
	keyMap  searchKeyMap

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

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(t.TextMuted())

	h := help.New()
	// Override default muted-gray help styles with theme tokens so key names and
	// descriptions match the overlay's visual language and update when the theme changes.
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(t.Info())
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(t.TextMuted())
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(t.TextMuted())
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(t.Info())
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(t.TextMuted())
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(t.TextMuted())
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
	rl.InfiniteScrolling = true

	return &SearchOverlay{
		theme:      t,
		input:      ti,
		spinner:    sp,
		help:       h,
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
	helpH = 3
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

	// Update help model width so it can truncate bindings appropriately.
	w := o.overlayWidth()
	o.help.Width = w - 4 // inside help panel border
}

// resizeList recomputes the list dimensions from the current panelHeights() and
// applies them via resultList.SetSize(). Must be called after any state change that
// could affect showHintLine() (typing, backspace, Ctrl+U, tab cycle, SearchClearedMsg),
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

// Init starts the cursor blink loop, placeholder ticker, and emits SearchClearedMsg
// so each search session begins with a clean state (previous results and query are discarded).
// searchSpinnerTick() is used instead of o.spinner.Tick so the spinner advances via the
// private searchSpinnerTickMsg type, preventing cross-component spinner.TickMsg interference.
func (o *SearchOverlay) Init() tea.Cmd {
	clearCmd := func() tea.Msg { return SearchClearedMsg{} }
	return tea.Batch(textinput.Blink, searchSpinnerTick(), clearCmd, searchPlaceholderTick())
}

// Update handles all messages for the search overlay.
func (o *SearchOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case searchSpinnerTickMsg:
		// Advance the spinner frame. Ignore the cmd returned by spinner.Update because
		// it fires spinner.TickMsg (the raw bubbles type), not searchSpinnerTickMsg.
		// We re-arm manually with searchSpinnerTick() to keep the private type in the loop.
		o.spinner, _ = o.spinner.Update(spinner.TickMsg(m))
		return o, searchSpinnerTick()

	case searchPlaceholderTickMsg:
		// Advance the cycling placeholder only when the input is empty.
		// When the user has typed something the tick is not re-armed, so cycling stops.
		// Re-arming happens in handleKey(KeyCtrlU) when the input is cleared.
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
	}

	// Forward key events to text input for cursor blinking.
	var cmd tea.Cmd
	o.input, cmd = o.input.Update(msg)
	return o, cmd
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

	case tea.KeyCtrlRight:
		// Next page — only when there is a query, not loading first page, and there is a next page.
		if o.intent.query == "" || o.loadingFirstPage || !o.hasNextPage() {
			return o, nil
		}
		o.intent.page++
		return o, o.scheduleDebounce()

	case tea.KeyCtrlLeft:
		// Prev page — only when there is a query, not loading first page, and not on page 1.
		if o.intent.query == "" || o.loadingFirstPage || o.intent.page <= 1 {
			return o, nil
		}
		o.intent.page--
		return o, o.scheduleDebounce()

	case tea.KeyCtrlU:
		// Clear local input — visual reset happens immediately.
		// Reset the Prompt to the default and force-clear prefix state so that
		// a subsequent cleanQuery() call does not index into a stale lockedPrefix.
		// Also reset intent.page and intent.query — clearing the search must not
		// fire a search for the empty string (no scheduleDebounce).
		// Restart the placeholder ticker so the animated cycling resumes.
		// Store writes are deferred: emit SearchClearedMsg for the root app to handle.
		o.input.Prompt = "> "
		o.input.SetValue("")
		o.lockedPrefix = ""
		o.prefixState = PrefixNone
		o.intent.page = 1
		o.intent.query = ""
		// Re-apply list dimensions: clearing the input makes showHintLine() return true
		// (searchH=4), which changes resultsH. resizeList() keeps the list height in sync.
		o.resizeList()
		return o, tea.Batch(
			func() tea.Msg { return SearchClearedMsg{} },
			searchPlaceholderTick(),
		)

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
		uri := si.URI
		return o, func() tea.Msg { return PlayTrackMsg{TrackURI: uri} }
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
	return o, func() tea.Msg { return AddToQueueMsg{TrackURI: uri} }
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

	cfg := layout.BorderConfig{
		Width:       w,
		Height:      h,
		Title:       "Search",
		Actions:     []layout.Action{},
		AccentColor: o.theme.ActiveBorder(),
		Focused:     true, // search bar is always focused — it captures all input
		Theme:       o.theme,
	}
	return layout.RenderPaneBorder(inner, cfg)
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
			Render(o.spinner.View() + " Loading…")
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
			Render(o.spinner.View() + " Searching…")
		resultsArea = lipgloss.NewStyle().
			Width(innerWidth).MaxWidth(innerWidth).
			Height(resultsAreaH).MaxHeight(resultsAreaH).
			Render(centered)
	} else {
		resultsContent := o.renderResults(innerWidth)
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

	cfg := layout.BorderConfig{
		Width:  w,
		Height: h,
		Title:  "Results",
		Actions: []layout.Action{
			{Key: "Enter", Label: "play"},
			{Key: "Ctrl+A", Label: "queue"},
		},
		// SeekBar() (cyan-family) is distinct from Search's ActiveBorder() (bright blue/green)
		// and Keys' TextMuted() (dim), giving the three panels a clear visual hierarchy.
		AccentColor: o.theme.SeekBar(),
		Focused:     false, // dimmer than the search bar
		Theme:       o.theme,
	}
	return layout.RenderPaneBorder(inner, cfg)
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

// renderHelpPanel builds Panel 3: the keybinding help bar.
// Height is always 3 lines (border top + help content + border bottom).
// The panel has no title — the keybinding content is self-explanatory,
// and an empty title lets the dim TextMuted() border recede into the background.
func (o *SearchOverlay) renderHelpPanel(w, h int) string {
	innerWidth := w - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	// Render help bar using bubbles/help.View(keyMap).
	helpContent := o.help.View(o.keyMap)
	inner := lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Height(h - 2).MaxHeight(h - 2).
		Render(helpContent)

	cfg := layout.BorderConfig{
		Width:       w,
		Height:      h,
		Title:       "", // no title — keybinding content is self-explanatory
		Actions:     []layout.Action{},
		AccentColor: o.theme.TextMuted(),
		Focused:     false,
		Theme:       o.theme,
	}
	return layout.RenderPaneBorder(inner, cfg)
}

// renderPaginationBar renders the [ ←  page N of M  → ] line.
// Arrows are dimmed (TextMuted) when navigation in that direction is not possible.
// The bar is centered within the given width w.
func (o *SearchOverlay) renderPaginationBar(w int) string {
	totalPages := (o.total + SearchPageSize - 1) / SearchPageSize
	if totalPages == 0 {
		totalPages = 1
	}
	center := fmt.Sprintf("  page %d of %d  ", o.intent.page, totalPages)

	prevStyle := o.theme.TextPrimary()
	nextStyle := o.theme.TextPrimary()
	if o.intent.page <= 1 {
		prevStyle = o.theme.TextMuted()
	}
	if !o.hasNextPage() {
		nextStyle = o.theme.TextMuted()
	}

	left := lipgloss.NewStyle().Foreground(prevStyle).Render("[ ←")
	right := lipgloss.NewStyle().Foreground(nextStyle).Render("→ ]")
	bar := lipgloss.JoinHorizontal(lipgloss.Center, left, center, right)
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(bar)
}

// renderResults builds the results area content inside Panel 2.
// When results are loaded, delegates to resultList.View() for scrollable rendering.
// Falls back to hint text when no results are present.
// NOTE: Loading states (loadingFirstPage / loadingNextPage) are handled by the
// caller renderResultsPanel, which renders the appropriate spinner before calling here.
func (o *SearchOverlay) renderResults(_ int) string {
	if len(o.resultList.Items()) == 0 {
		return lipgloss.NewStyle().
			Foreground(o.theme.TextMuted()).
			Render("Type to search tracks, artists, albums...")
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

// searchSpinnerTick returns a tea.Cmd that fires searchSpinnerTickMsg after 130ms.
// The 130ms interval matches the spinner.Dot tick rate from bubbles/spinner.
// We wrap the tick as searchSpinnerTickMsg instead of using o.spinner.Tick directly
// because spinner.Tick fires spinner.TickMsg, which our Update handler does not match
// (intentional isolation to prevent cross-component interference).
func searchSpinnerTick() tea.Cmd {
	return tea.Tick(130*time.Millisecond, func(_ time.Time) tea.Msg {
		return searchSpinnerTickMsg{}
	})
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
	// Update spinner foreground so loading indicator uses the new theme.
	o.spinner.Style = lipgloss.NewStyle().Foreground(th.TextMuted())
	// Update placeholder style so the cycling hints use the new Info() color.
	o.input.PlaceholderStyle = lipgloss.NewStyle().Foreground(th.Info())
	// Update completion/ghost text style so suggestions use the new TextMuted() color.
	o.input.CompletionStyle = lipgloss.NewStyle().Foreground(th.TextMuted())
	// Propagate to help styles so key names and descriptions use the new theme colors.
	o.help.Styles.ShortKey = lipgloss.NewStyle().Foreground(th.Info())
	o.help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(th.TextMuted())
	o.help.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(th.TextMuted())
	o.help.Styles.FullKey = lipgloss.NewStyle().Foreground(th.Info())
	o.help.Styles.FullDesc = lipgloss.NewStyle().Foreground(th.TextMuted())
	o.help.Styles.FullSeparator = lipgloss.NewStyle().Foreground(th.TextMuted())
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

// SearchSpinnerTickCmd exposes the private searchSpinnerTick() function for tests.
// Tests that need to drive the spinner should use this to obtain a cmd, execute it,
// and pass the resulting message to overlay.Update().
func SearchSpinnerTickCmd() tea.Cmd {
	return searchSpinnerTick()
}

// SpinnerView returns the current rendered spinner frame string — exported for tests
// that need to detect frame advancement.
func SpinnerView(o *SearchOverlay) string {
	return o.spinner.View()
}

// RenderTabBarForTest calls renderTabBar with the given inner width and returns the result.
// Exported so tests can inspect tab bar content in isolation without calling full View().
func RenderTabBarForTest(o *SearchOverlay, innerWidth int) string {
	return o.renderTabBar(innerWidth)
}

// ContainsSpinnerFrame returns true when s contains the current spinner frame string.
// Used in tests to verify the spinner appears (or does not appear) in rendered output.
func ContainsSpinnerFrame(o *SearchOverlay, s string) bool {
	frame := o.spinner.View()
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

// RenderHelpForTest calls o.help.View(o.keyMap) and returns the raw rendered string.
// Exported for tests that need to inspect help bar ANSI output.
func RenderHelpForTest(o *SearchOverlay) string {
	return o.help.View(o.keyMap)
}

// HelpShortKeyForegroundForTest returns the foreground color set on o.help.Styles.ShortKey.
// Exported for tests that verify SetTheme() propagates help colors correctly.
func HelpShortKeyForegroundForTest(o *SearchOverlay) lipgloss.TerminalColor {
	return o.help.Styles.ShortKey.GetForeground()
}

// HelpShortDescForegroundForTest returns the foreground color set on o.help.Styles.ShortDesc.
// Exported for tests that verify SetTheme() propagates help colors correctly.
func HelpShortDescForegroundForTest(o *SearchOverlay) lipgloss.TerminalColor {
	return o.help.Styles.ShortDesc.GetForeground()
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
