package panes

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// searchSection enumerates the four result sections in display order.
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

// NOTE: SearchResultsMsg is defined in messages.go alongside all other shared
// message types. Search result data types (SearchResultData, SearchTrackItem, etc.)
// are also in messages.go.

// SearchOverlay is the floating search UI model. It is layered above the
// three-pane view while open — it does not replace any pane.
type SearchOverlay struct {
	store   *state.Store
	theme   theme.Theme
	input   textinput.Model
	spinner spinner.Model
	width   int
	height  int

	// results holds the most recent search results delivered via SearchResultsMsg.
	// This avoids reading api.SearchResult from the store, keeping the ui/api boundary clean.
	results *SearchResultData

	// activeSection is which section (Tracks/Artists/Albums/Playlists) has focus.
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

	return &SearchOverlay{
		store:         store,
		theme:         t,
		input:         ti,
		spinner:       sp,
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

// CursorPos returns the current cursor position within the active section.
// Exposed for tests.
func (o *SearchOverlay) CursorPos() int {
	return o.cursorPos
}

// SetSize updates the overlay dimensions (forwarded from root app on resize).
func (o *SearchOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height
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

	case SearchResultsMsg:
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
		return o.moveSectionForward()

	case tea.KeyShiftTab:
		return o.moveSectionBackward()

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

// moveSectionForward advances the active section, wrapping from last to first.
func (o *SearchOverlay) moveSectionForward() (tea.Model, tea.Cmd) {
	o.activeSection = (o.activeSection + 1) % numSections
	o.cursorPos = 0
	return o, nil
}

// moveSectionBackward retreats the active section, wrapping from first to last.
func (o *SearchOverlay) moveSectionBackward() (tea.Model, tea.Cmd) {
	o.activeSection = (o.activeSection + numSections - 1) % numSections
	o.cursorPos = 0
	return o, nil
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

// View renders the search overlay box.
func (o *SearchOverlay) View() string {
	overlayWidth := o.overlayWidth()

	var sb strings.Builder

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Foreground(o.theme.TextPrimary()).
		Bold(true)
	sb.WriteString(titleStyle.Render("Search"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("·", overlayWidth-2))
	sb.WriteString("\n")

	// Input line
	sb.WriteString(o.input.View())
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("·", overlayWidth-2))
	sb.WriteString("\n")

	// Results area
	sb.WriteString(o.renderResults(overlayWidth))

	// Wrap in a border
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(o.theme.ActiveBorder()).
		Padding(0, 1).
		Width(overlayWidth)

	return borderStyle.Render(sb.String())
}

// renderResults builds the results area of the overlay.
func (o *SearchOverlay) renderResults(overlayWidth int) string {
	query := o.store.SearchQuery()
	loading := o.store.SearchLoading()

	if loading {
		return fmt.Sprintf("%s Searching...\n", o.spinner.View())
	}

	// Show search error state if API call failed.
	if err := o.store.SearchError(); err != nil && query != "" {
		return lipgloss.NewStyle().
			Foreground(o.theme.Error()).
			Render(fmt.Sprintf("Search failed: %s", err.Error()))
	}

	if query == "" {
		return lipgloss.NewStyle().
			Foreground(o.theme.TextMuted()).
			Render("Type to search tracks, artists, albums...")
	}

	if o.results == nil {
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

// overlayWidth returns the effective overlay width clamped to 50 chars or 60%
// of terminal width per DESIGN.md spec.
func (o *SearchOverlay) overlayWidth() int {
	w := 50
	if o.width > 0 {
		sixtyPct := o.width * 60 / 100
		if sixtyPct < w {
			w = sixtyPct
		}
	}
	if w < 20 {
		w = 20
	}
	return w
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
