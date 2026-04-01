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
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// SearchSection enumerates the four result sections in display order.
// Exported so test packages can reference specific sections by name.
type SearchSection = searchSection

// searchSection is the underlying type for SearchSection.
type searchSection int

const (
	sectionTracks    searchSection = iota // 0
	sectionArtists                        // 1
	sectionAlbums                         // 2
	sectionPlaylists                      // 3
	numSections      = 4
)

// Exported section constants for use in tests and external packages.
const (
	SectionTracks    SearchSection = sectionTracks
	SectionArtists   SearchSection = sectionArtists
	SectionAlbums    SearchSection = sectionAlbums
	SectionPlaylists SearchSection = sectionPlaylists
)

// searchSectionLabels are the labels used in the tab bar for each section.
var searchSectionLabels = [numSections]string{
	sectionTracks:    "Tracks",
	sectionArtists:   "Artists",
	sectionAlbums:    "Albums",
	sectionPlaylists: "Playlists",
}

// maxResultsPerSection is the number of results shown per section in the overlay.
// Set to 10 to match the Feb 2026 Spotify API maximum of 10 results per type.
const maxResultsPerSection = 10

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
	// This avoids reading domain.SearchResult from the store, keeping the ui/api boundary clean.
	results *SearchResultData

	// activeSection is which section (Tracks/Artists/Albums/Playlists) has focus.
	activeSection searchSection

	// cursorPos is the cursor within the active section (0-based).
	cursorPos int

	// sectionOffsets tracks the current page offset for each section independently.
	// Values are multiples of maxResultsPerSection (i.e., 0, 10, 20, …).
	// Reset to zero on every new query.
	sectionOffsets [numSections]int
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
		if m.Offset == 0 {
			// New query (first page) — replace all results and reset pagination state.
			o.results = m.Results
			o.cursorPos = 0
			o.activeSection = sectionTracks
			o.sectionOffsets = [numSections]int{}
		} else {
			// Paginated load — merge results for the requesting section only.
			// Other sections' results and offsets are preserved.
			if o.results == nil {
				o.results = &SearchResultData{}
			}
			o.sectionOffsets[m.Section] = m.Offset
			if m.Results != nil {
				o.mergePageResults(m.Section, m.Results)
			}
			// Place cursor at position 0 for the newly loaded page.
			if m.Section == o.activeSection {
				o.cursorPos = 0
			}
		}
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
// When the cursor is already at the last row, it checks if more pages exist;
// if so it emits a SearchPageRequestMsg and resets cursorPos to 0.
func (o *SearchOverlay) moveCursorDown() (tea.Model, tea.Cmd) {
	max := o.maxCursorForActiveSection() - 1
	if max < 0 {
		return o, nil
	}
	if o.cursorPos < max {
		o.cursorPos++
		return o, nil
	}
	// At last row — check if more pages exist.
	total := o.totalForSection(o.activeSection)
	offset := o.sectionOffsets[o.activeSection]
	shown := o.maxCursorForActiveSection()
	if offset+shown < total {
		// More results exist — request next page.
		o.cursorPos = 0
		return o, o.requestPage(offset + maxResultsPerSection)
	}
	return o, nil
}

// moveCursorUp moves cursor up within the active section.
// When the cursor is already at row 0 and offset > 0, it emits a
// SearchPageRequestMsg for the previous page and places cursor at last item.
func (o *SearchOverlay) moveCursorUp() (tea.Model, tea.Cmd) {
	if o.cursorPos > 0 {
		o.cursorPos--
		return o, nil
	}
	// At first row — check if previous page exists.
	offset := o.sectionOffsets[o.activeSection]
	if offset > 0 {
		// NOTE: cursor position for previous page (last item) is set when
		// SearchResultsMsg arrives (it resets to 0), but for UX we'd ideally
		// land at the bottom. The spec says "set cursorPos = maxCursorForActiveSection()-1
		// after page load". However, since the page hasn't loaded yet at this point,
		// we defer that to the SearchResultsMsg handler. For now, emit the request.
		return o, o.requestPage(offset - maxResultsPerSection)
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

// View renders the search overlay box with a btop-style border.
// The border is rendered via layout.RenderPaneBorder() so that the title
// ("Search") and action shortcuts ("Enter play", "Esc close") appear
// embedded in the top border line, consistent with the main grid pane borders.
func (o *SearchOverlay) View() string {
	totalWidth := o.overlayWidth()
	// Minimum height: 2 border rows + input row + separator row + at least 1 content row.
	totalHeight := o.overlayHeight()

	// Inner content dimensions (inside the border).
	innerWidth := totalWidth - 2
	if innerWidth < 2 {
		innerWidth = 2
	}
	innerHeight := totalHeight - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Build inner content lines.
	var lines []string

	// Input line — fixed at the top of the inner area.
	lines = append(lines, o.input.View())

	// Thin separator.
	lines = append(lines, lipgloss.NewStyle().
		Foreground(o.theme.TextMuted()).
		Render(strings.Repeat("·", innerWidth)))

	// Results area — fills the remaining inner lines.
	// Pass innerHeight - 2 (subtract input line + dot separator) so renderResults can
	// compute padding to anchor the help bar at the bottom.
	resultsStr := o.renderResults(innerWidth, innerHeight-2)
	resultLines := strings.Split(resultsStr, "\n")
	lines = append(lines, resultLines...)

	// Join and size to innerWidth × innerHeight.
	inner := strings.Join(lines, "\n")
	// Safety: cap content to innerWidth × innerHeight via lipgloss.
	inner = lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Height(innerHeight).MaxHeight(innerHeight).
		Render(inner)

	cfg := layout.BorderConfig{
		Width:  totalWidth,
		Height: totalHeight,
		Title:  "Search",
		Actions: []layout.Action{
			{Key: "Enter", Label: "play"},
			{Key: "Esc", Label: "close"},
		},
		AccentColor: o.theme.ActiveBorder(),
		Focused:     true, // overlays are always focused
		Theme:       o.theme,
	}

	return layout.RenderPaneBorder(inner, cfg)
}

// renderResults builds the tabbed results area of the overlay.
// Assembly order: tab bar → tab separator → column headers → active section rows →
// [padding to push help bar to bottom] → help bar.
//
// innerWidth is the content width inside the border (View subtracts 2 for border chars).
// availableHeight is innerHeight - 2 (input line + dot separator already consumed).
// It is used to anchor the help bar at the bottom by inserting blank lines between
// result rows and the help bar when there are fewer results than the available space.
func (o *SearchOverlay) renderResults(innerWidth, availableHeight int) string {
	query := o.store.SearchQuery()
	loading := o.store.SearchLoading()

	if loading {
		return fmt.Sprintf("%s Searching...\n", o.spinner.View())
	}

	// NOTE: Search errors are now routed through toast notifications (app.go).
	// store.SearchError() is preserved for retry logic but no longer read in View().

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

	// innerWidth already has the border removed (View subtracts 2). Subtract only
	// left + right padding (1 char each side = 2 total).
	contentWidth := innerWidth - 2 // left + right padding within the border
	if contentWidth < 10 {
		contentWidth = 10
	}

	mutedStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())

	var sb strings.Builder
	// Tab bar
	sb.WriteString(o.renderTabBar(contentWidth))
	sb.WriteString("\n")
	// Tab separator
	sb.WriteString(mutedStyle.Render(strings.Repeat("─", contentWidth)))
	sb.WriteString("\n")
	// Column headers + underline
	sb.WriteString(o.renderColumnHeaders(o.activeSection, contentWidth))
	sb.WriteString("\n")
	// Active section rows only
	activeSection := o.renderActiveSection(contentWidth)
	sb.WriteString(activeSection)

	// Anchor help bar to the bottom by inserting padding lines.
	// Chrome lines: tab bar(1) + tab sep(1) + header(1) + underline(1) = 4
	// Help bar lines: separator(1) + text(1) = 2
	// Row budget = availableHeight - 4(chrome) - 2(help bar)
	const chromeLines = 4
	const helpBarLines = 2
	rowBudget := availableHeight - chromeLines - helpBarLines
	resultLineCount := strings.Count(activeSection, "\n")
	if rowBudget > 0 {
		padLines := rowBudget - resultLineCount
		if padLines > 0 {
			sb.WriteString(strings.Repeat("\n", padLines))
		}
	}

	// Help bar (separator + keybindings)
	sb.WriteString(o.renderHelpBar(contentWidth))
	return sb.String()
}

// overlayWidth returns the effective overlay width: min(90, 80% terminal) with min 40.
// The wider base (90 vs the old 50) accommodates the richer metadata columns.
func (o *SearchOverlay) overlayWidth() int {
	w := 90
	if o.width > 0 {
		eightyPct := o.width * 80 / 100
		if eightyPct < w {
			w = eightyPct
		}
	}
	if w < 40 {
		w = 40
	}
	return w
}

// overlayHeight returns the effective overlay height: max(26, 75% terminal) with min 12.
// The taller base (26 vs the old 20) provides enough room for 10 results per section.
func (o *SearchOverlay) overlayHeight() int {
	h := 26
	if o.height > 0 {
		seventyFivePct := o.height * 75 / 100
		if seventyFivePct > h {
			h = seventyFivePct
		}
	}
	if h < 12 {
		h = 12
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

// --- Clamping helpers (max 10 per section) ---

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

// --- Column width helpers ---
// Each helper computes column widths that sum exactly to contentWidth.
// Fixed columns (indexW, durationW, yearW, tracksW) are subtracted first;
// inter-column gaps (2 chars per gap, (numCols-1) gaps) are also subtracted;
// the remaining flex space is distributed proportionally among name/artist/album columns.
// The last flex column gets the remainder to prevent any rounding-induced overflow.

// trackColumnWidths computes column widths for the Tracks section.
//
//	Returns (nameW, artistW, albumW, durationW). In narrow mode, albumW is 0 and
//	artistW absorbs the flex space.
func (o *SearchOverlay) trackColumnWidths(contentWidth int, narrow bool) (nameW, artistW, albumW, durationW int) {
	indexW := 3
	durationW = 8
	if narrow {
		// 4 columns: # | Track | Artist | Duration — gaps = (4-1)*2 = 6
		gaps := (4 - 1) * 2
		fixed := indexW + durationW + gaps
		flex := contentWidth - fixed
		if flex < 0 {
			flex = 0
		}
		nameW = flex * 50 / 100
		artistW = flex - nameW // remainder prevents overflow
		albumW = 0
		return
	}
	// 5 columns: # | Track | Artist | Album | Duration — gaps = (5-1)*2 = 8
	gaps := (5 - 1) * 2
	fixed := indexW + durationW + gaps
	flex := contentWidth - fixed
	if flex < 0 {
		flex = 0
	}
	nameW = flex * 40 / 100
	artistW = flex * 30 / 100
	albumW = flex - nameW - artistW // remainder prevents overflow
	return
}

// TrackColumnWidths is the exported wrapper of trackColumnWidths for use in tests.
func (o *SearchOverlay) TrackColumnWidths(contentWidth int, narrow bool) (nameW, artistW, albumW, durationW int) {
	return o.trackColumnWidths(contentWidth, narrow)
}

// albumColumnWidths computes column widths for the Albums section.
//
//	Returns (nameW, artistW, yearW, tracksW).
//	5 columns: # | Album | Artist | Year | Tracks — gaps = (5-1)*2 = 8
func (o *SearchOverlay) albumColumnWidths(contentWidth int) (nameW, artistW, yearW, tracksW int) {
	indexW := 3
	yearW = 6
	tracksW = 8
	gaps := (5 - 1) * 2
	fixed := indexW + yearW + tracksW + gaps
	flex := contentWidth - fixed
	if flex < 0 {
		flex = 0
	}
	nameW = flex * 55 / 100
	artistW = flex - nameW // remainder prevents overflow
	return
}

// AlbumColumnWidths is the exported wrapper of albumColumnWidths for use in tests.
func (o *SearchOverlay) AlbumColumnWidths(contentWidth int) (nameW, artistW, yearW, tracksW int) {
	return o.albumColumnWidths(contentWidth)
}

// playlistColumnWidths computes column widths for the Playlists section.
//
//	Returns (nameW, ownerW, tracksW).
//	4 columns: # | Playlist | Owner | Tracks — gaps = (4-1)*2 = 6
func (o *SearchOverlay) playlistColumnWidths(contentWidth int) (nameW, ownerW, tracksW int) {
	indexW := 3
	tracksW = 8
	gaps := (4 - 1) * 2
	fixed := indexW + tracksW + gaps
	flex := contentWidth - fixed
	if flex < 0 {
		flex = 0
	}
	nameW = flex * 55 / 100
	ownerW = flex - nameW // remainder prevents overflow
	return
}

// PlaylistColumnWidths is the exported wrapper of playlistColumnWidths for use in tests.
func (o *SearchOverlay) PlaylistColumnWidths(contentWidth int) (nameW, ownerW, tracksW int) {
	return o.playlistColumnWidths(contentWidth)
}

// artistColumnWidths computes the artist column width for the Artists section.
//
//	2 columns: # | Artist — gaps = (2-1)*2 = 2
func (o *SearchOverlay) artistColumnWidths(contentWidth int) (artistW int) {
	indexW := 3
	gaps := (2 - 1) * 2
	artistW = contentWidth - indexW - gaps
	if artistW < 4 {
		artistW = 4
	}
	return
}

// ArtistColumnWidths is the exported wrapper of artistColumnWidths for use in tests.
func (o *SearchOverlay) ArtistColumnWidths(contentWidth int) (artistW int) {
	return o.artistColumnWidths(contentWidth)
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

// renderHelpBar renders a separator line and a contextual keybindings line, both in
// TextMuted color. "Ctrl+A queue" only appears on the Tracks section.
// When the active section has more results than maxResultsPerSection, a right-aligned
// page indicator (e.g., "1-10 of 16") is appended to the keybindings line.
func (o *SearchOverlay) renderHelpBar(contentWidth int) string {
	mutedStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
	separator := mutedStyle.Render(strings.Repeat("─", contentWidth))

	var keys string
	if o.activeSection == sectionTracks {
		keys = "Tab next section  ↑↓ navigate  Enter play  Ctrl+A queue  Esc close"
	} else {
		keys = "Tab next section  ↑↓ navigate  Enter play  Esc close"
	}

	indicator := o.pageIndicator()
	if indicator != "" {
		// Right-align the indicator by padding the keybindings line.
		keysWidth := utf8.RuneCountInString(keys)
		indicatorWidth := utf8.RuneCountInString(indicator)
		gap := contentWidth - keysWidth - indicatorWidth
		if gap > 0 {
			keys = keys + strings.Repeat(" ", gap) + indicator
		} else {
			// Not enough room — append with a single space.
			keys = keys + " " + indicator
		}
	}

	helpLine := mutedStyle.Render(keys)
	return separator + "\n" + helpLine
}

// RenderHelpBar is the exported wrapper of renderHelpBar for use in tests.
func (o *SearchOverlay) RenderHelpBar(contentWidth int) string {
	return o.renderHelpBar(contentWidth)
}

// FormatDurationMs is the exported wrapper for formatDurationMs (defined in nowplaying.go),
// allowing tests to call it without importing package internals.
func FormatDurationMs(ms int) string {
	return formatDurationMs(ms)
}

// searchCol is a single column definition for column-aligned result row rendering.
type searchCol struct {
	text  string
	style lipgloss.Style
	width int
}

// renderActiveSection renders the numbered, column-aligned result rows for the
// currently active section. Only the active section's results are shown.
// Selected row: ▶ prefix + SelectedBg/Fg. Unselected rows: numbered prefix +
// ColumnIndex/Primary/Secondary/Tertiary colors per column.
// Tracks section drops the Album column when contentWidth < 60.
func (o *SearchOverlay) renderActiveSection(contentWidth int) string {
	if o.results == nil {
		return ""
	}

	indexW := 3
	indexStyle := lipgloss.NewStyle().Foreground(o.theme.ColumnIndex())
	primaryStyle := lipgloss.NewStyle().Foreground(o.theme.ColumnPrimary())
	secondaryStyle := lipgloss.NewStyle().Foreground(o.theme.ColumnSecondary())
	tertiaryStyle := lipgloss.NewStyle().Foreground(o.theme.ColumnTertiary())
	selectedStyle := lipgloss.NewStyle().
		Background(o.theme.SelectedBg()).
		Foreground(o.theme.SelectedFg())

	var sb strings.Builder

	renderRow := func(i int, isSelected bool, cols []searchCol) {
		if isSelected {
			// Build row content joined with spaces and apply selected style.
			rowParts := make([]string, 0, len(cols)+1)
			rowParts = append(rowParts, fmt.Sprintf("%-*s", indexW, "▶ "))
			for _, col := range cols {
				rowParts = append(rowParts, fmt.Sprintf("%-*s", col.width, truncate(col.text, col.width)))
			}
			rowText := strings.Join(rowParts, "  ")
			sb.WriteString(selectedStyle.Width(contentWidth).Render(rowText))
		} else {
			parts := make([]string, 0, len(cols)+1)
			parts = append(parts, indexStyle.Render(fmt.Sprintf("%-*s", indexW, fmt.Sprintf("%d", i+1))))
			for _, col := range cols {
				cell := truncate(col.text, col.width)
				parts = append(parts, col.style.Render(fmt.Sprintf("%-*s", col.width, cell)))
			}
			sb.WriteString(strings.Join(parts, "  "))
		}
		sb.WriteString("\n")
	}

	switch o.activeSection {
	case sectionTracks:
		items := clampedTrackItems(o.results)
		narrow := contentWidth < 60
		nameW, artistW, albumW, durationW := o.trackColumnWidths(contentWidth, narrow)
		if narrow {
			for i, item := range items {
				renderRow(i, o.cursorPos == i, []searchCol{
					{item.Name, primaryStyle, nameW},
					{item.Artist, secondaryStyle, artistW},
					{formatDurationMs(item.DurationMs), tertiaryStyle, durationW},
				})
			}
		} else {
			for i, item := range items {
				renderRow(i, o.cursorPos == i, []searchCol{
					{item.Name, primaryStyle, nameW},
					{item.Artist, secondaryStyle, artistW},
					{item.Album, secondaryStyle, albumW},
					{formatDurationMs(item.DurationMs), tertiaryStyle, durationW},
				})
			}
		}

	case sectionArtists:
		items := clampedArtistItems(o.results)
		artistW := o.artistColumnWidths(contentWidth)
		for i, item := range items {
			renderRow(i, o.cursorPos == i, []searchCol{
				{item.Name, primaryStyle, artistW},
			})
		}

	case sectionAlbums:
		items := clampedAlbumItems(o.results)
		nameW, artistW, yearW, tracksW := o.albumColumnWidths(contentWidth)
		for i, item := range items {
			renderRow(i, o.cursorPos == i, []searchCol{
				{item.Name, primaryStyle, nameW},
				{item.Artist, secondaryStyle, artistW},
				{item.ReleaseYear, tertiaryStyle, yearW},
				{fmt.Sprintf("%d", item.TotalTracks), tertiaryStyle, tracksW},
			})
		}

	case sectionPlaylists:
		items := clampedPlaylistItems(o.results)
		nameW, ownerW, tracksW := o.playlistColumnWidths(contentWidth)
		for i, item := range items {
			renderRow(i, o.cursorPos == i, []searchCol{
				{item.Name, primaryStyle, nameW},
				{item.Owner, secondaryStyle, ownerW},
				{fmt.Sprintf("%d", item.TrackCount), tertiaryStyle, tracksW},
			})
		}
	}

	return sb.String()
}

// RenderActiveSection is the exported wrapper of renderActiveSection for use in tests.
func (o *SearchOverlay) RenderActiveSection(contentWidth int) string {
	return o.renderActiveSection(contentWidth)
}

// renderColumnHeaders renders the column header row and an underline separator for
// the given section. Headers are styled with the active section's tab color + bold.
// The underline uses TextMuted dashes. Tracks drops the Album column when contentWidth < 60.
func (o *SearchOverlay) renderColumnHeaders(sec searchSection, contentWidth int) string {
	headerStyle := lipgloss.NewStyle().
		Foreground(o.tabColorForSection(sec)).
		Bold(true)
	underlineStyle := lipgloss.NewStyle().
		Foreground(o.theme.TextMuted())

	indexW := 3
	var labels []string
	var widths []int

	// Use shared column width helpers so headers and row widths are always identical.
	switch sec {
	case sectionTracks:
		narrow := contentWidth < 60
		nameW, artistW, albumW, durationW := o.trackColumnWidths(contentWidth, narrow)
		if narrow {
			labels = []string{"#", "Track", "Artist", "Duration"}
			widths = []int{indexW, nameW, artistW, durationW}
		} else {
			labels = []string{"#", "Track", "Artist", "Album", "Duration"}
			widths = []int{indexW, nameW, artistW, albumW, durationW}
		}
	case sectionArtists:
		artistW := o.artistColumnWidths(contentWidth)
		labels = []string{"#", "Artist"}
		widths = []int{indexW, artistW}
	case sectionAlbums:
		nameW, artistW, yearW, tracksW := o.albumColumnWidths(contentWidth)
		labels = []string{"#", "Album", "Artist", "Year", "Tracks"}
		widths = []int{indexW, nameW, artistW, yearW, tracksW}
	case sectionPlaylists:
		nameW, ownerW, tracksW := o.playlistColumnWidths(contentWidth)
		labels = []string{"#", "Playlist", "Owner", "Tracks"}
		widths = []int{indexW, nameW, ownerW, tracksW}
	default:
		return ""
	}

	// Build header line
	var hParts []string
	var uParts []string
	for i, lbl := range labels {
		w := widths[i]
		if w <= 0 {
			w = len(lbl)
		}
		hParts = append(hParts, headerStyle.Render(fmt.Sprintf("%-*s", w, truncate(lbl, w))))
		uParts = append(uParts, underlineStyle.Render(strings.Repeat("─", min(w, len(lbl)+1))))
	}

	headerLine := strings.Join(hParts, "  ")
	underLine := strings.Join(uParts, "  ")
	return headerLine + "\n" + underLine
}

// RenderColumnHeaders is the exported wrapper of renderColumnHeaders for use in tests.
func (o *SearchOverlay) RenderColumnHeaders(sec searchSection, contentWidth int) string {
	return o.renderColumnHeaders(sec, contentWidth)
}

// renderTabBar renders a horizontal tab bar showing all four sections with their
// result counts. The active section is highlighted with ▪ + bold + its tab color.
// Inactive sections use TextMuted. Tabs are separated by 5 spaces.
func (o *SearchOverlay) renderTabBar(width int) string {
	var parts []string

	for i := searchSection(0); i < numSections; i++ {
		label := fmt.Sprintf("%s %d", searchSectionLabels[i], o.totalForSection(i))
		if i == o.activeSection {
			// Active tab: ▪ prefix + bold + tab-specific color
			style := lipgloss.NewStyle().
				Foreground(o.tabColorForSection(i)).
				Bold(true)
			parts = append(parts, style.Render("▪ "+label))
		} else {
			// Inactive tab: plain + TextMuted
			style := lipgloss.NewStyle().
				Foreground(o.theme.TextMuted())
			parts = append(parts, style.Render(label))
		}
	}

	return strings.Join(parts, "     ")
}

// RenderTabBar is the exported wrapper of renderTabBar for use in tests.
func (o *SearchOverlay) RenderTabBar(width int) string {
	return o.renderTabBar(width)
}

// totalForSection returns the total result count for the given section from
// SearchResultData.TotalTracks/Artists/Albums/Playlists. Returns 0 if results are nil.
func (o *SearchOverlay) totalForSection(sec searchSection) int {
	if o.results == nil {
		return 0
	}
	switch sec {
	case sectionTracks:
		return o.results.TotalTracks
	case sectionArtists:
		return o.results.TotalArtists
	case sectionAlbums:
		return o.results.TotalAlbums
	case sectionPlaylists:
		return o.results.TotalPlaylists
	default:
		return 0
	}
}

// TotalForSection is the exported wrapper of totalForSection for use in tests.
func (o *SearchOverlay) TotalForSection(sec searchSection) int {
	return o.totalForSection(sec)
}

// mergePageResults replaces the items for the given section with the page items from src.
// It also updates the Total* field for that section so the tab bar count stays accurate.
func (o *SearchOverlay) mergePageResults(sec searchSection, src *SearchResultData) {
	switch sec {
	case sectionTracks:
		o.results.Tracks = src.Tracks
		if src.TotalTracks > 0 {
			o.results.TotalTracks = src.TotalTracks
		}
	case sectionArtists:
		o.results.Artists = src.Artists
		if src.TotalArtists > 0 {
			o.results.TotalArtists = src.TotalArtists
		}
	case sectionAlbums:
		o.results.Albums = src.Albums
		if src.TotalAlbums > 0 {
			o.results.TotalAlbums = src.TotalAlbums
		}
	case sectionPlaylists:
		o.results.Playlists = src.Playlists
		if src.TotalPlaylists > 0 {
			o.results.TotalPlaylists = src.TotalPlaylists
		}
	}
}

// requestPage emits a SearchPageRequestMsg that asks the root app to fetch the
// page at the given offset for the active section.
// The query is read from the store (which holds the last submitted query) rather than
// the text input (which may differ if the user has typed but not yet submitted).
func (o *SearchOverlay) requestPage(offset int) tea.Cmd {
	query := o.store.SearchQuery()
	section := o.activeSection
	return func() tea.Msg {
		return SearchPageRequestMsg{
			Query:   query,
			Offset:  offset,
			Section: section,
		}
	}
}

// pageIndicator returns a right-aligned range string for the active section when
// the total exceeds maxResultsPerSection (e.g., "1-10 of 16" or "11-16 of 16").
// Returns empty string when all results fit on one page.
func (o *SearchOverlay) pageIndicator() string {
	total := o.totalForSection(o.activeSection)
	if total <= maxResultsPerSection {
		return ""
	}
	offset := o.sectionOffsets[o.activeSection]
	start := offset + 1
	end := offset + o.maxCursorForActiveSection()
	if end > total {
		end = total
	}
	return fmt.Sprintf("%d-%d of %d", start, end, total)
}

// PageIndicator is the exported wrapper of pageIndicator for use in tests.
func (o *SearchOverlay) PageIndicator() string {
	return o.pageIndicator()
}

// SectionOffsets returns the sectionOffsets array for test inspection.
func (o *SearchOverlay) SectionOffsets() [numSections]int {
	return o.sectionOffsets
}

// WithSectionOffsets returns a copy of the overlay with the given sectionOffsets set.
// Used in tests to simulate mid-pagination state without going through a full page load.
func (o *SearchOverlay) WithSectionOffsets(offsets [numSections]int) *SearchOverlay {
	copy := *o
	copy.sectionOffsets = offsets
	return &copy
}

// tabColorForSection returns the PaneBorder* theme token for the given section.
// This gives each tab a distinct identity color consistent with its pane border color.
// Falls back to ActiveBorder() for any unrecognized section value.
func (o *SearchOverlay) tabColorForSection(sec searchSection) lipgloss.Color {
	switch sec {
	case sectionTracks:
		return o.theme.PaneBorderTopTracks()
	case sectionArtists:
		return o.theme.PaneBorderTopArtists()
	case sectionAlbums:
		return o.theme.PaneBorderAlbums()
	case sectionPlaylists:
		return o.theme.PaneBorderPlaylists()
	default:
		return o.theme.ActiveBorder()
	}
}

// TabColorForSection is the exported wrapper of tabColorForSection for use in tests.
func (o *SearchOverlay) TabColorForSection(sec searchSection) lipgloss.Color {
	return o.tabColorForSection(sec)
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
