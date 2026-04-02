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
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
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

// searchSection enumerates the four result sections in display order.
// TODO(search-redesign): replace with bubbles/list delegate in Story 84.
type searchSection int

const (
	sectionTracks    searchSection = iota // 0
	sectionArtists                        // 1
	sectionAlbums                         // 2
	sectionPlaylists                      // 3
	numSections      = 4
)

// maxResultsPerSection is the number of results shown per section in the overlay.
const maxResultsPerSection = 5

// searchPrefetchThreshold is the fraction of loaded items at which the next
// prefetch batch is triggered. Kept in sync with app.SearchPrefetchThreshold (0.6).
// Defined here to avoid a circular dependency between panes and app packages.
const searchPrefetchThreshold = 0.6

// searchDebounceMsg carries a query snapshot fired 300ms after a keypress.
// In Update, it is only acted on if msg.query still matches the current input.
type searchDebounceMsg struct{ query string }

// SearchClosedMsg is emitted when the user presses Esc, signalling the root
// app model to close the overlay and restore the previous pane focus.
type SearchClosedMsg struct{}

// SearchRequestMsg is emitted when the debounce fires and the query is non-empty.
// The root app model receives it and dispatches the actual Spotify API call.
type SearchRequestMsg struct {
	Query string
}

// searchSpinnerTickMsg is used by the bubbles/spinner to advance its frame.
type searchSpinnerTickMsg spinner.TickMsg

// searchKeyMap defines all keybindings shown in the help bar (Panel 3).
// It implements the bubbles/help KeyMap interface.
type searchKeyMap struct {
	Play    key.Binding
	Queue   key.Binding
	TabNext key.Binding
	TabPrev key.Binding
	Close   key.Binding
	Clear   key.Binding
}

// ShortHelp returns 5 bindings for the compact help bar view.
func (k searchKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Play, k.Queue, k.TabNext, k.TabPrev, k.Close}
}

// FullHelp returns all 6 bindings grouped in a single column.
func (k searchKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Play, k.Queue, k.TabNext, k.TabPrev, k.Close, k.Clear}}
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
	}
}

// NOTE: SearchPageLoadedMsg is defined in messages.go alongside all other shared
// message types. Search result data types (SearchResultData, SearchTrackItem, etc.)
// are also in messages.go.

// SearchOverlay is the floating search UI model. It is layered above the
// three-pane view while open — it does not replace any pane.
//
// The overlay renders as three separate bordered panels stacked vertically:
//   - Panel 1 (Search): text input
//   - Panel 2 (Results): tab bar + separator + scrollable results list
//   - Panel 3 (Keys): bubbles/help keybinding bar
type SearchOverlay struct {
	store   *state.Store
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

	// results holds the most recent search results delivered via SearchPageLoadedMsg.
	// This avoids reading domain.SearchResult from the store, keeping the ui/api boundary clean.
	results *SearchResultData

	// activeTab is which category tab (All/Songs/Artists/Albums/Playlists) is selected.
	activeTab SearchTab

	// activeSection is which section (Tracks/Artists/Albums/Playlists) has focus.
	// Kept for backward-compatibility with cursor-based selection until Story 84.
	activeSection searchSection

	// cursorPos is the cursor within the active section (0-based).
	cursorPos int
}

// NewSearchOverlay constructs a SearchOverlay wired to the given store and theme.
// The text input is focused by default.
func NewSearchOverlay(store *state.Store, t theme.Theme) *SearchOverlay {
	ti := textinput.New()
	ti.Placeholder = "Type to search tracks, artists, albums..."
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(t.TextMuted())

	h := help.New()
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
		store:         store,
		theme:         t,
		input:         ti,
		spinner:       sp,
		help:          h,
		keyMap:        km,
		resultList:    rl,
		activeTab:     TabAll,
		activeSection: sectionTracks,
		cursorPos:     0,
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

// ActiveSection returns the currently active result section index.
// Exposed for tests.
func (o *SearchOverlay) ActiveSection() searchSection {
	return o.activeSection
}

// ActiveTab returns the currently active category tab.
// Exported for tests.
func (o *SearchOverlay) ActiveTab() SearchTab {
	return o.activeTab
}

// OverlayWidth returns the computed overlay width (80% of terminal width, min 40).
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

// SetSize updates the overlay dimensions (forwarded from root app on resize).
// Dimensions are propagated to the list.Model inner area.
func (o *SearchOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height

	w := o.overlayWidth()
	totalH := o.overlayHeight()

	searchBarH := 3
	helpH := 3
	// 1 margin between search bar and results only.
	resultsH := totalH - searchBarH - helpH - 1
	if resultsH < 5 {
		resultsH = 5
	}

	// Inner list dimensions: subtract results border (2) + tab bar (1) + separator (1).
	listW := w - 2
	if listW < 1 {
		listW = 1
	}
	listH := resultsH - 4
	if listH < 1 {
		listH = 1
	}
	o.resultList.SetSize(listW, listH)

	// Update help model width so it can truncate bindings appropriately.
	o.help.Width = w - 4 // inside help panel border
}

// Init starts the cursor blink loop.
func (o *SearchOverlay) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, o.spinner.Tick)
}

// Update handles all messages for the search overlay.
func (o *SearchOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case searchSpinnerTickMsg:
		var cmd tea.Cmd
		o.spinner, cmd = o.spinner.Update(spinner.TickMsg(m))
		return o, cmd

	case searchDebounceMsg:
		return o.handleDebounce(m)

	case SearchPageLoadedMsg:
		if m.Err != nil {
			// Error response — preserve existing results so the screen is not
			// blanked. Toast notifications (handled by app.go) give user feedback.
			return o, nil
		}
		// Save results locally so we never read api types from the store.
		o.results = m.Results
		o.cursorPos = 0
		o.activeSection = sectionTracks
		// Rebuild the list from the store (which has been updated by app.go).
		o.rebuildListItems()
		return o, nil

	case tea.KeyMsg:
		return o.handleKey(m)
	}

	// Forward key events to text input for cursor blinking.
	var cmd tea.Cmd
	o.input, cmd = o.input.Update(msg)
	return o, cmd
}

// handleDebounce processes a searchDebounceMsg. Only fires a search request if
// the snapshot query still matches the current input value (discards stale ticks).
func (o *SearchOverlay) handleDebounce(m searchDebounceMsg) (tea.Model, tea.Cmd) {
	if m.query != o.input.Value() {
		// Query changed since this tick was scheduled — discard.
		return o, nil
	}
	if m.query == "" {
		// Never fire on empty query.
		return o, nil
	}
	// Fire a search request — root app model handles it.
	return o, func() tea.Msg {
		return SearchRequestMsg{Query: m.query}
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
		return o.cycleTabForward()

	case tea.KeyShiftTab:
		return o.cycleTabBackward()

	case tea.KeyUp:
		var cmd tea.Cmd
		o.resultList, cmd = o.resultList.Update(m)
		prefetchCmd := o.checkPrefetch()
		return o, tea.Batch(cmd, prefetchCmd)

	case tea.KeyDown:
		var cmd tea.Cmd
		o.resultList, cmd = o.resultList.Update(m)
		prefetchCmd := o.checkPrefetch()
		return o, tea.Batch(cmd, prefetchCmd)

	case tea.KeyCtrlA:
		return o.handleAddToQueue()

	case tea.KeyCtrlU:
		// Clear local input and cursor — visual reset happens immediately.
		// Store writes are deferred: emit SearchClearedMsg for the root app to handle.
		o.input.SetValue("")
		o.cursorPos = 0
		return o, func() tea.Msg { return SearchClearedMsg{} }

	case tea.KeyBackspace:
		// Let textinput handle backspace normally.
		var cmd tea.Cmd
		o.input, cmd = o.input.Update(m)
		// Schedule debounce for updated value.
		q := o.input.Value()
		debounceCmd := debounceSearch(q)
		return o, tea.Batch(cmd, debounceCmd)

	default:
		// Regular typing — update input, schedule debounce.
		var cmd tea.Cmd
		o.input, cmd = o.input.Update(m)
		q := o.input.Value()
		debounceCmd := debounceSearch(q)
		return o, tea.Batch(cmd, debounceCmd)
	}
}

// handleEnter plays the currently selected result.
// It reads the selected item from the list.Model delegate first; if the list is
// empty (no results loaded yet) it falls back to the legacy selectedURI helper.
func (o *SearchOverlay) handleEnter() (tea.Model, tea.Cmd) {
	// Try list-based selection first.
	if selected := o.resultList.SelectedItem(); selected != nil {
		si, ok := selected.(SearchListItem)
		if ok && si.URI != "" {
			var playCmd tea.Cmd
			if si.IsTrack {
				uri := si.URI
				playCmd = func() tea.Msg { return PlayTrackMsg{TrackURI: uri} }
			} else {
				uri := si.URI
				playCmd = func() tea.Msg { return PlayContextMsg{ContextURI: uri} }
			}
			closeCmd := func() tea.Msg { return SearchClosedMsg{} }
			return o, tea.Batch(playCmd, closeCmd)
		}
	}

	// Fallback: legacy selectedURI (used when list is empty but o.results is set).
	uri, isTrack := o.selectedURI()
	if uri == "" {
		return o, nil
	}
	var playCmd tea.Cmd
	if isTrack {
		playCmd = func() tea.Msg { return PlayTrackMsg{TrackURI: uri} }
	} else {
		playCmd = func() tea.Msg { return PlayContextMsg{ContextURI: uri} }
	}
	closeCmd := func() tea.Msg { return SearchClosedMsg{} }
	return o, tea.Batch(playCmd, closeCmd)
}

// handleAddToQueue adds the currently selected track to the queue.
// Uses list.SelectedItem() to get the selection; falls back to legacy logic
// when the list is empty.
func (o *SearchOverlay) handleAddToQueue() (tea.Model, tea.Cmd) {
	// List-based selection first.
	if selected := o.resultList.SelectedItem(); selected != nil {
		si, ok := selected.(SearchListItem)
		if ok && si.IsTrack && si.URI != "" {
			uri := si.URI
			return o, func() tea.Msg { return AddToQueueMsg{TrackURI: uri} }
		}
		// Selected item is not a track — Ctrl+A is a no-op.
		return o, nil
	}

	// Fallback: legacy logic when list has no items yet.
	if o.results == nil || o.activeSection != sectionTracks {
		return o, nil
	}
	items := clampedTrackItems(o.results)
	if o.cursorPos >= len(items) {
		return o, nil
	}
	trackURI := items[o.cursorPos].URI
	return o, func() tea.Msg { return AddToQueueMsg{TrackURI: trackURI} }
}

// cycleTabForward advances the active tab, wrapping from the last tab back to TabAll.
// It rebuilds the list items for the new tab and emits SearchTabChangedMsg so the
// root app re-fires the search with the new type filter.
func (o *SearchOverlay) cycleTabForward() (tea.Model, tea.Cmd) {
	o.activeTab = SearchTab((int(o.activeTab) + 1) % NumTabs)
	o.activeSection = sectionTracks
	o.cursorPos = 0
	o.rebuildListItems()
	query := o.input.Value()
	types := TabToAPITypes(o.activeTab)
	return o, func() tea.Msg {
		return SearchTabChangedMsg{Types: types, Query: query}
	}
}

// cycleTabBackward retreats the active tab, wrapping from TabAll back to the last tab.
// It rebuilds the list items for the new tab and emits SearchTabChangedMsg so the
// root app re-fires the search with the new type filter.
func (o *SearchOverlay) cycleTabBackward() (tea.Model, tea.Cmd) {
	o.activeTab = SearchTab((int(o.activeTab) + NumTabs - 1) % NumTabs)
	o.activeSection = sectionTracks
	o.cursorPos = 0
	o.rebuildListItems()
	query := o.input.Value()
	types := TabToAPITypes(o.activeTab)
	return o, func() tea.Msg {
		return SearchTabChangedMsg{Types: types, Query: query}
	}
}

// selectedURI returns the URI for the currently selected result item,
// and whether it is a track (vs. a context URI).
func (o *SearchOverlay) selectedURI() (uri string, isTrack bool) {
	if o.results == nil {
		return "", false
	}

	switch o.activeSection {
	case sectionTracks:
		items := clampedTrackItems(o.results)
		if o.cursorPos < len(items) {
			return items[o.cursorPos].URI, true
		}
	case sectionArtists:
		items := clampedArtistItems(o.results)
		if o.cursorPos < len(items) {
			return items[o.cursorPos].URI, false
		}
	case sectionAlbums:
		items := clampedAlbumItems(o.results)
		if o.cursorPos < len(items) {
			return items[o.cursorPos].URI, false
		}
	case sectionPlaylists:
		items := clampedPlaylistItems(o.results)
		if o.cursorPos < len(items) {
			return items[o.cursorPos].URI, false
		}
	}
	return "", false
}

// View renders the search overlay as three separate bordered panels stacked vertically:
//   - Panel 1 (Search): text input
//   - Panel 2 (Results): tab bar + separator + results area
//   - Panel 3 (Keys): help keybinding bar
//
// A 1-line margin separates Panel 1 from Panel 2. Panel 2 and Panel 3 are flush.
func (o *SearchOverlay) View() string {
	w := o.overlayWidth()
	totalH := o.overlayHeight()

	// Panel 1: Search bar (fixed height 3).
	searchBarH := 3
	searchPanel := o.renderSearchPanel(w, searchBarH)

	// Panel 3: Help bar (fixed height 3).
	helpH := 3
	helpPanel := o.renderHelpPanel(w, helpH)

	// Panel 2: Results (fills remaining space; subtract 1 for margin between search and results).
	resultsH := totalH - searchBarH - helpH - 1
	if resultsH < 5 {
		resultsH = 5
	}
	resultsPanel := o.renderResultsPanel(w, resultsH)

	// Compose: Panel1 + 1 empty line (margin) + Panel2 + Panel3 (flush, no margin).
	return lipgloss.JoinVertical(lipgloss.Left,
		searchPanel,
		"", // 1-line margin between search and results
		resultsPanel,
		helpPanel,
	)
}

// renderSearchPanel builds Panel 1: the search input box.
// Height is 3 lines (border top + input + border bottom).
func (o *SearchOverlay) renderSearchPanel(w, h int) string {
	innerWidth := w - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	// Inner content: just the text input view.
	inner := lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Height(h - 2).MaxHeight(h - 2).
		Render(o.input.View())

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

// renderResultsPanel builds Panel 2: the tab bar, separator, and results area.
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

	// Results area fills the remaining lines.
	resultsAreaH := innerHeight - 2 // subtract tab bar + separator
	if resultsAreaH < 1 {
		resultsAreaH = 1
	}
	resultsArea := o.renderResults(innerWidth)
	resultsArea = lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Height(resultsAreaH).MaxHeight(resultsAreaH).
		Render(resultsArea)

	// Combine inner content lines.
	inner := strings.Join([]string{tabBar, separator, resultsArea}, "\n")
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
		AccentColor: o.theme.SectionHeader(),
		Focused:     false, // dimmer than the search bar
		Theme:       o.theme,
	}
	return layout.RenderPaneBorder(inner, cfg)
}

// renderTabBar renders the tab selector row inside Panel 2.
// The active tab is shown with brackets and highlight styling; inactive tabs use TextMuted.
func (o *SearchOverlay) renderTabBar(innerWidth int) string {
	var parts []string
	for i := 0; i < NumTabs; i++ {
		tab := SearchTab(i)
		label := TabLabels[tab]
		if tab == o.activeTab {
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
	// Pad/truncate to exactly innerWidth.
	return lipgloss.NewStyle().Width(innerWidth).MaxWidth(innerWidth).Render(tabLine)
}

// renderHelpPanel builds Panel 3: the keybinding help bar.
// Height is always 3 lines (border top + help content + border bottom).
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
		Title:       "Keys",
		Actions:     []layout.Action{},
		AccentColor: o.theme.TextMuted(),
		Focused:     false,
		Theme:       o.theme,
	}
	return layout.RenderPaneBorder(inner, cfg)
}

// renderResults builds the results area content inside Panel 2.
// When results are loaded, delegates to resultList.View() for scrollable rendering.
// Falls back to hint/loading/no-results text as needed.
func (o *SearchOverlay) renderResults(_ int) string {
	query := o.store.SearchQuery()
	loading := o.store.SearchLoading()

	if loading && len(o.resultList.Items()) == 0 {
		// Only show spinner when there are no items yet (first load).
		return fmt.Sprintf("%s Searching...\n", o.spinner.View())
	}

	// NOTE: Search errors are now routed through toast notifications (app.go).
	// store.SearchError() is preserved for retry logic but no longer read in View().

	if query == "" && len(o.resultList.Items()) == 0 {
		return lipgloss.NewStyle().
			Foreground(o.theme.TextMuted()).
			Render("Type to search tracks, artists, albums...")
	}

	if len(o.resultList.Items()) == 0 && o.results != nil {
		return lipgloss.NewStyle().
			Foreground(o.theme.TextMuted()).
			Render(fmt.Sprintf("No results for '%s'", query))
	}

	if len(o.resultList.Items()) == 0 && query == "" {
		return lipgloss.NewStyle().
			Foreground(o.theme.TextMuted()).
			Render("Type to search tracks, artists, albums...")
	}

	// Render the list component — it handles scrolling and selection highlighting.
	return o.resultList.View()
}

// rebuildListItems repopulates resultList from the store based on the active tab.
// If the store's per-type TypePages are empty (e.g. in overlay-standalone tests),
// it falls back to the locally cached o.results (SearchResultData). This dual-source
// approach allows the overlay to work correctly both in production (store populated
// by app.go) and in isolation tests (results delivered via SearchPageLoadedMsg only).
func (o *SearchOverlay) rebuildListItems() {
	// Use store data when available (non-empty TypePages).
	storeTracks := o.store.SearchTracks().Items
	storeArtists := o.store.SearchArtists().Items
	storeAlbums := o.store.SearchAlbums().Items
	storePlaylists := o.store.SearchPlaylists().Items

	storeHasData := len(storeTracks)+len(storeArtists)+len(storeAlbums)+len(storePlaylists) > 0

	if storeHasData {
		o.rebuildFromStore(storeTracks, storeArtists, storeAlbums, storePlaylists)
		return
	}

	// Fall back to locally cached SearchResultData (overlay-standalone / test path).
	if o.results != nil {
		o.rebuildFromResults()
	}
}

// rebuildFromStore rebuilds the list from pre-fetched store slices.
func (o *SearchOverlay) rebuildFromStore(
	tracks []domain.Track,
	artists []domain.SearchArtist,
	albums []domain.SearchAlbum,
	playlists []domain.SearchPlaylist,
) {
	var items []list.Item

	switch o.activeTab {
	case TabSongs:
		items = tracksToListItems(tracks)
	case TabArtists:
		items = artistsToListItems(artists)
	case TabAlbums:
		items = albumsToListItems(albums)
	case TabPlaylists:
		items = playlistsToListItems(playlists)
	default: // TabAll
		items = append(items, tracksToListItems(tracks)...)
		items = append(items, artistsToListItems(artists)...)
		items = append(items, albumsToListItems(albums)...)
		items = append(items, playlistsToListItems(playlists)...)
	}

	o.resultList.SetItems(items)
}

// rebuildFromResults rebuilds the list from the locally cached SearchResultData.
// Used when the store's TypePages are empty (overlay-standalone / test scenarios).
func (o *SearchOverlay) rebuildFromResults() {
	if o.results == nil {
		return
	}

	var items []list.Item

	switch o.activeTab {
	case TabSongs:
		items = searchTrackItemsToListItems(o.results.Tracks)
	case TabArtists:
		items = searchArtistItemsToListItems(o.results.Artists)
	case TabAlbums:
		items = searchAlbumItemsToListItems(o.results.Albums)
	case TabPlaylists:
		items = searchPlaylistItemsToListItems(o.results.Playlists)
	default: // TabAll
		items = append(items, searchTrackItemsToListItems(o.results.Tracks)...)
		items = append(items, searchArtistItemsToListItems(o.results.Artists)...)
		items = append(items, searchAlbumItemsToListItems(o.results.Albums)...)
		items = append(items, searchPlaylistItemsToListItems(o.results.Playlists)...)
	}

	o.resultList.SetItems(items)
}

// checkPrefetch returns a SearchPrefetchMsg command when the list cursor has
// scrolled past searchPrefetchThreshold of the loaded items. Returns nil if
// below threshold, no items, or no more pages are available.
func (o *SearchOverlay) checkPrefetch() tea.Cmd {
	total := len(o.resultList.Items())
	if total == 0 {
		return nil
	}

	cursor := o.resultList.Index()
	threshold := int(float64(total) * searchPrefetchThreshold)

	if cursor < threshold {
		return nil
	}

	nextOffset := o.nextOffsetForTab()
	if nextOffset < 0 {
		return nil
	}

	types := TabToAPITypes(o.activeTab)
	query := o.store.SearchQuery()
	offset := nextOffset
	return func() tea.Msg {
		return SearchPrefetchMsg{
			Query:      query,
			Types:      types,
			NextOffset: offset,
		}
	}
}

// nextOffsetForTab returns the next offset to fetch for the active tab.
// Returns -1 when no more pages are available (offset >= total).
// For tabAll, uses tracks as the representative type.
func (o *SearchOverlay) nextOffsetForTab() int {
	switch o.activeTab {
	case TabSongs:
		p := o.store.SearchTracks()
		if p.Offset >= p.Total && p.Total > 0 {
			return -1
		}
		return p.Offset
	case TabArtists:
		p := o.store.SearchArtists()
		if p.Offset >= p.Total && p.Total > 0 {
			return -1
		}
		return p.Offset
	case TabAlbums:
		p := o.store.SearchAlbums()
		if p.Offset >= p.Total && p.Total > 0 {
			return -1
		}
		return p.Offset
	case TabPlaylists:
		p := o.store.SearchPlaylists()
		if p.Offset >= p.Total && p.Total > 0 {
			return -1
		}
		return p.Offset
	default: // TabAll — use tracks as representative type
		p := o.store.SearchTracks()
		if p.Offset >= p.Total && p.Total > 0 {
			return -1
		}
		return p.Offset
	}
}

// overlayWidth returns the effective overlay width: 80% of terminal width, minimum 40.
func (o *SearchOverlay) overlayWidth() int {
	w := o.width * 80 / 100
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

// debounceSearch returns a tea.Cmd that fires a searchDebounceMsg after 300ms.
// The query snapshot is captured in the closure so stale ticks can be detected.
func debounceSearch(query string) tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(_ time.Time) tea.Msg {
		return searchDebounceMsg{query: query}
	})
}

// --- Clamping helpers (max 5 per section) ---

func clampedTrackItems(r *SearchResultData) []SearchTrackItem {
	items := r.Tracks
	if len(items) > maxResultsPerSection {
		items = items[:maxResultsPerSection]
	}
	return items
}

func clampedArtistItems(r *SearchResultData) []SearchArtistItem {
	items := r.Artists
	if len(items) > maxResultsPerSection {
		items = items[:maxResultsPerSection]
	}
	return items
}

func clampedAlbumItems(r *SearchResultData) []SearchAlbumItem {
	items := r.Albums
	if len(items) > maxResultsPerSection {
		items = items[:maxResultsPerSection]
	}
	return items
}

func clampedPlaylistItems(r *SearchResultData) []SearchPlaylistItem {
	items := r.Playlists
	if len(items) > maxResultsPerSection {
		items = items[:maxResultsPerSection]
	}
	return items
}

// SetTheme updates the theme reference for runtime theme switching.
func (o *SearchOverlay) SetTheme(th theme.Theme) {
	o.theme = th
}

// --- Test helpers (exported only for test packages) ---

// SearchDebounceMsgForTest creates a searchDebounceMsg for use in tests.
// This allows the test package (panes_test) to inject debounce messages
// without exposing the unexported type in the production API.
func SearchDebounceMsgForTest(query string) tea.Msg {
	return searchDebounceMsg{query: query}
}

// SearchSpinnerTickMsgForTest creates a searchSpinnerTickMsg for advancing the spinner in tests.
func SearchSpinnerTickMsgForTest() tea.Msg {
	return searchSpinnerTickMsg{}
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
	o.activeTab = tab
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

// CallCheckPrefetch calls checkPrefetch on the overlay — exported for tests.
func CallCheckPrefetch(o *SearchOverlay) tea.Cmd {
	return o.checkPrefetch()
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
