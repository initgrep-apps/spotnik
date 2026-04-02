package panes

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// searchSectionLabels are the header labels rendered above each section.
var searchSectionLabels = [numSections]string{
	sectionTracks:    "TRACKS",
	sectionArtists:   "ARTISTS",
	sectionAlbums:    "ALBUMS",
	sectionPlaylists: "PLAYLISTS",
}

// maxResultsPerSection is the number of results shown per section in the overlay.
const maxResultsPerSection = 5

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

	// Initialize a blank list.Model. Items will be populated in Story 84.
	// All built-in chrome is disabled since we render our own tab bar, separator,
	// and help bar outside the list.
	delegate := list.NewDefaultDelegate()
	rl := list.New(nil, delegate, 0, 0)
	rl.SetShowTitle(false)
	rl.SetShowFilter(false)
	rl.SetShowStatusBar(false)
	rl.SetShowHelp(false)

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

// CursorPos returns the current cursor position within the active section.
// Exposed for tests.
func (o *SearchOverlay) CursorPos() int {
	return o.cursorPos
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
		return o.moveCursorUp()

	case tea.KeyDown:
		return o.moveCursorDown()

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
func (o *SearchOverlay) handleEnter() (tea.Model, tea.Cmd) {
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
// Only valid when the active section is Tracks.
func (o *SearchOverlay) handleAddToQueue() (tea.Model, tea.Cmd) {
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
// It emits SearchTabChangedMsg so the root app re-fires the search with the new type filter.
func (o *SearchOverlay) cycleTabForward() (tea.Model, tea.Cmd) {
	o.activeTab = SearchTab((int(o.activeTab) + 1) % NumTabs)
	o.activeSection = sectionTracks
	o.cursorPos = 0
	query := o.input.Value()
	types := TabToAPITypes(o.activeTab)
	return o, func() tea.Msg {
		return SearchTabChangedMsg{Types: types, Query: query}
	}
}

// cycleTabBackward retreats the active tab, wrapping from TabAll back to the last tab.
// It emits SearchTabChangedMsg so the root app re-fires the search with the new type filter.
func (o *SearchOverlay) cycleTabBackward() (tea.Model, tea.Cmd) {
	o.activeTab = SearchTab((int(o.activeTab) + NumTabs - 1) % NumTabs)
	o.activeSection = sectionTracks
	o.cursorPos = 0
	query := o.input.Value()
	types := TabToAPITypes(o.activeTab)
	return o, func() tea.Msg {
		return SearchTabChangedMsg{Types: types, Query: query}
	}
}

// moveCursorDown moves cursor down within the active section.
func (o *SearchOverlay) moveCursorDown() (tea.Model, tea.Cmd) {
	max := o.maxCursorForActiveSection() - 1
	if max < 0 {
		return o, nil
	}
	if o.cursorPos < max {
		o.cursorPos++
	}
	return o, nil
}

// moveCursorUp moves cursor up within the active section.
func (o *SearchOverlay) moveCursorUp() (tea.Model, tea.Cmd) {
	if o.cursorPos > 0 {
		o.cursorPos--
	}
	return o, nil
}

// maxCursorForActiveSection returns the count of items in the current section.
func (o *SearchOverlay) maxCursorForActiveSection() int {
	if o.results == nil {
		return 0
	}
	switch o.activeSection {
	case sectionTracks:
		return len(clampedTrackItems(o.results))
	case sectionArtists:
		return len(clampedArtistItems(o.results))
	case sectionAlbums:
		return len(clampedAlbumItems(o.results))
	case sectionPlaylists:
		return len(clampedPlaylistItems(o.results))
	}
	return 0
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
// This reuses the existing section-based rendering until Story 84 wires list.Model.
func (o *SearchOverlay) renderResults(overlayWidth int) string {
	query := o.store.SearchQuery()
	loading := o.store.SearchLoading()

	if loading {
		return fmt.Sprintf("%s Searching...\n", o.spinner.View())
	}

	// NOTE: Search errors are now routed through toast notifications (app.go).
	// store.SearchError() is preserved for retry logic but no longer read in View().

	if query == "" || o.results == nil {
		return lipgloss.NewStyle().
			Foreground(o.theme.TextMuted()).
			Render("Type to search tracks, artists, albums...")
	}

	// Check if all sections are empty
	totalItems := len(clampedTrackItems(o.results)) +
		len(clampedArtistItems(o.results)) +
		len(clampedAlbumItems(o.results)) +
		len(clampedPlaylistItems(o.results))

	if totalItems == 0 {
		return lipgloss.NewStyle().
			Foreground(o.theme.TextMuted()).
			Render(fmt.Sprintf("No results for '%s'", query))
	}

	contentWidth := overlayWidth - 4 // subtract border + padding
	if contentWidth < 10 {
		contentWidth = 10
	}

	var sb strings.Builder
	sb.WriteString(o.renderSection(sectionTracks, clampedTrackItemsAsRows(o.results, contentWidth), contentWidth))
	sb.WriteString(o.renderSection(sectionArtists, clampedArtistItemsAsRows(o.results, contentWidth), contentWidth))
	sb.WriteString(o.renderSection(sectionAlbums, clampedAlbumItemsAsRows(o.results, contentWidth), contentWidth))
	sb.WriteString(o.renderSection(sectionPlaylists, clampedPlaylistItemsAsRows(o.results, contentWidth), contentWidth))
	return sb.String()
}

// renderSection renders one section with its header and items.
func (o *SearchOverlay) renderSection(sec searchSection, rows []string, contentWidth int) string {
	if len(rows) == 0 {
		return ""
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(o.theme.SectionHeader()).
		Bold(true)

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(headerStyle.Render(fmt.Sprintf("● %s", searchSectionLabels[sec])))
	sb.WriteString("\n")

	for i, row := range rows {
		isSelected := o.activeSection == sec && o.cursorPos == i

		if isSelected {
			lineStyle := lipgloss.NewStyle().
				Background(o.theme.SelectedBg()).
				Foreground(o.theme.SelectedFg()).
				Width(contentWidth)
			sb.WriteString(lineStyle.Render("▶ " + row))
		} else {
			lineStyle := lipgloss.NewStyle().
				Foreground(o.theme.TextPrimary()).
				Width(contentWidth)
			sb.WriteString(lineStyle.Render("  " + row))
		}
		sb.WriteString("\n")
	}

	return sb.String()
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

// --- Row string builders for each section ---

func clampedTrackItemsAsRows(r *SearchResultData, width int) []string {
	items := clampedTrackItems(r)
	rows := make([]string, len(items))
	for i, t := range items {
		row := fmt.Sprintf("%-*s  %s", width/2, truncate(t.Name, width/2), truncate(t.Artist, width/2-4))
		rows[i] = truncate(row, width-2)
	}
	return rows
}

func clampedArtistItemsAsRows(r *SearchResultData, width int) []string {
	items := clampedArtistItems(r)
	rows := make([]string, len(items))
	for i, a := range items {
		rows[i] = truncate(a.Name, width-2)
	}
	return rows
}

func clampedAlbumItemsAsRows(r *SearchResultData, width int) []string {
	items := clampedAlbumItems(r)
	rows := make([]string, len(items))
	for i, a := range items {
		row := fmt.Sprintf("%-*s  %s", width/2, truncate(a.Name, width/2), truncate(a.Artist, width/2-4))
		rows[i] = truncate(row, width-2)
	}
	return rows
}

func clampedPlaylistItemsAsRows(r *SearchResultData, width int) []string {
	items := clampedPlaylistItems(r)
	rows := make([]string, len(items))
	for i, p := range items {
		row := fmt.Sprintf("%-*s  %s", width/2, truncate(p.Name, width/2), truncate(p.Owner, width/2-4))
		rows[i] = truncate(row, width-2)
	}
	return rows
}

// truncate shortens s to at most maxRunes runes, appending "…" if truncated.
func truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	count := utf8.RuneCountInString(s)
	if count <= maxRunes {
		return s
	}
	// Leave room for the ellipsis
	if maxRunes <= 1 {
		return "…"
	}
	runes := []rune(s)
	return string(runes[:maxRunes-1]) + "…"
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
