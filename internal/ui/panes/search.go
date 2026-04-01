package panes

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
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

	// activeSection is which section (Tracks/Artists/Albums/Playlists) has focus.
	activeSection searchSection

	// tables holds one components.Table per section. They manage cursor position,
	// page size, and row rendering. Replaces the former manual cursor/row rendering.
	tables [numSections]*components.Table

	// narrowTracks tracks whether the tracks table was last built in narrow mode.
	// Used to detect width changes that require rebuilding the tracks table.
	narrowTracks bool
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

	o := &SearchOverlay{
		store:         store,
		theme:         t,
		input:         ti,
		spinner:       sp,
		activeSection: sectionTracks,
	}
	o.buildAllTables(false)
	return o
}

// buildAllTables constructs or replaces all 4 components.Table instances.
// narrowTracks controls whether the tracks table is built without the Album column.
func (o *SearchOverlay) buildAllTables(narrowTracks bool) {
	o.narrowTracks = narrowTracks
	o.tables[sectionTracks] = components.NewTable(components.TableConfig{
		Columns:      o.trackColumns(narrowTracks),
		Theme:        o.theme,
		PlayingIndex: -1,
		ShowHeader:   true,
		HeaderColor:  o.tabColorForSection(sectionTracks),
	})
	o.tables[sectionTracks].SetFocused(o.activeSection == sectionTracks)

	o.tables[sectionArtists] = components.NewTable(components.TableConfig{
		Columns:      o.artistColumns(),
		Theme:        o.theme,
		PlayingIndex: -1,
		ShowHeader:   true,
		HeaderColor:  o.tabColorForSection(sectionArtists),
	})
	o.tables[sectionArtists].SetFocused(o.activeSection == sectionArtists)

	o.tables[sectionAlbums] = components.NewTable(components.TableConfig{
		Columns:      o.albumColumns(),
		Theme:        o.theme,
		PlayingIndex: -1,
		ShowHeader:   true,
		HeaderColor:  o.tabColorForSection(sectionAlbums),
	})
	o.tables[sectionAlbums].SetFocused(o.activeSection == sectionAlbums)

	o.tables[sectionPlaylists] = components.NewTable(components.TableConfig{
		Columns:      o.playlistColumns(),
		Theme:        o.theme,
		PlayingIndex: -1,
		ShowHeader:   true,
		HeaderColor:  o.tabColorForSection(sectionPlaylists),
	})
	o.tables[sectionPlaylists].SetFocused(o.activeSection == sectionPlaylists)
}

// trackColumns returns ColumnDef slice for the tracks table.
// When narrow is true, the Album column is omitted.
func (o *SearchOverlay) trackColumns(narrow bool) []components.ColumnDef {
	th := o.theme
	cols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
	}
	if !narrow {
		cols = append(cols, components.ColumnDef{Key: "album", Header: "Album", FlexFactor: 7, Color: th.ColumnSecondary()})
	}
	cols = append(cols, components.ColumnDef{Key: "duration", Header: "Duration", FlexFactor: 2, Color: th.ColumnTertiary()})
	return cols
}

// artistColumns returns ColumnDef slice for the artists table.
func (o *SearchOverlay) artistColumns() []components.ColumnDef {
	th := o.theme
	return []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Artist", FlexFactor: 9, Color: th.ColumnPrimary()},
	}
}

// albumColumns returns ColumnDef slice for the albums table.
func (o *SearchOverlay) albumColumns() []components.ColumnDef {
	th := o.theme
	return []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Album", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
		{Key: "year", Header: "Year", FlexFactor: 2, Color: th.ColumnTertiary()},
		{Key: "tracks", Header: "Tracks", FlexFactor: 2, Color: th.ColumnTertiary()},
	}
}

// playlistColumns returns ColumnDef slice for the playlists table.
func (o *SearchOverlay) playlistColumns() []components.ColumnDef {
	th := o.theme
	return []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Playlist", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "owner", Header: "Owner", FlexFactor: 6, Color: th.ColumnSecondary()},
		{Key: "tracks", Header: "Tracks", FlexFactor: 2, Color: th.ColumnTertiary()},
	}
}

// rebuildTrackTable reconstructs the tracks table — needed when terminal width
// crosses the narrow threshold (< 60 content chars) to add/remove the Album column.
func (o *SearchOverlay) rebuildTrackTable(narrow bool) {
	o.narrowTracks = narrow
	o.tables[sectionTracks] = components.NewTable(components.TableConfig{
		Columns:      o.trackColumns(narrow),
		Theme:        o.theme,
		PlayingIndex: -1,
		ShowHeader:   true,
		HeaderColor:  o.tabColorForSection(sectionTracks),
	})
	o.tables[sectionTracks].SetFocused(o.activeSection == sectionTracks)
	o.refreshTrackRows()
}

// refreshTrackRows converts the store's accumulated track buffer into table rows.
func (o *SearchOverlay) refreshTrackRows() {
	tracks := o.store.SearchTracks()
	rows := make([]map[string]string, len(tracks))
	for i, t := range tracks {
		row := map[string]string{
			"index":    fmt.Sprintf("%d", i+1),
			"name":     t.Name,
			"artist":   t.Artist,
			"duration": formatDurationMs(t.DurationMs),
		}
		if !o.narrowTracks {
			row["album"] = t.Album
		}
		rows[i] = row
	}
	o.tables[sectionTracks].SetRows(rows)
}

// refreshArtistRows converts the store's accumulated artist buffer into table rows.
func (o *SearchOverlay) refreshArtistRows() {
	artists := o.store.SearchArtists()
	rows := make([]map[string]string, len(artists))
	for i, a := range artists {
		rows[i] = map[string]string{
			"index": fmt.Sprintf("%d", i+1),
			"name":  a.Name,
		}
	}
	o.tables[sectionArtists].SetRows(rows)
}

// refreshAlbumRows converts the store's accumulated album buffer into table rows.
func (o *SearchOverlay) refreshAlbumRows() {
	albums := o.store.SearchAlbums()
	rows := make([]map[string]string, len(albums))
	for i, a := range albums {
		rows[i] = map[string]string{
			"index":  fmt.Sprintf("%d", i+1),
			"name":   a.Name,
			"artist": a.Artist,
			"year":   a.ReleaseYear,
			"tracks": fmt.Sprintf("%d", a.TotalTracks),
		}
	}
	o.tables[sectionAlbums].SetRows(rows)
}

// refreshPlaylistRows converts the store's accumulated playlist buffer into table rows.
func (o *SearchOverlay) refreshPlaylistRows() {
	playlists := o.store.SearchPlaylists()
	rows := make([]map[string]string, len(playlists))
	for i, p := range playlists {
		rows[i] = map[string]string{
			"index":  fmt.Sprintf("%d", i+1),
			"name":   p.Name,
			"owner":  p.Owner,
			"tracks": fmt.Sprintf("%d", p.TrackCount),
		}
	}
	o.tables[sectionPlaylists].SetRows(rows)
}

// refreshAllRows refreshes all 4 table row sets from the accumulated buffers.
func (o *SearchOverlay) refreshAllRows() {
	o.refreshTrackRows()
	o.refreshArtistRows()
	o.refreshAlbumRows()
	o.refreshPlaylistRows()
}

// clearBuffers wipes all local table rows and resets the active section.
// Item buffer clearing is handled by the store (via app.go ClearSearchBuffers call).
func (o *SearchOverlay) clearBuffers() {
	// Wipe all table rows so the UI reflects the cleared state immediately.
	for i := range o.tables {
		if o.tables[i] != nil {
			o.tables[i].SetRows(nil)
		}
	}
}

// sectionBufferLen returns the current accumulated buffer length for a section
// by reading from the store.
func (o *SearchOverlay) sectionBufferLen(sec searchSection) int {
	switch sec {
	case sectionTracks:
		return len(o.store.SearchTracks())
	case sectionArtists:
		return len(o.store.SearchArtists())
	case sectionAlbums:
		return len(o.store.SearchAlbums())
	case sectionPlaylists:
		return len(o.store.SearchPlaylists())
	}
	return 0
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

// SetSize updates the overlay dimensions (forwarded from root app on resize).
// Rebuilds the tracks table if the narrow threshold is crossed.
func (o *SearchOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height

	// Compute content width (inner border - 2 padding) to decide narrow mode.
	innerWidth := o.overlayWidth() - 2
	if innerWidth < 2 {
		innerWidth = 2
	}
	contentWidth := innerWidth - 2
	narrow := contentWidth < 60

	if narrow != o.narrowTracks {
		o.rebuildTrackTable(narrow)
	}

	// Set table sizes: table gets height = innerHeight - 4 (input + sep + tabbar + sep).
	innerHeight := o.overlayHeight() - 2
	tableHeight := innerHeight - 4 - 2 // subtract input(1)+sep(1)+tabbar(1)+tabsep(1) and helpbar(2)
	if tableHeight < 4 {
		tableHeight = 4
	}
	tableWidth := innerWidth
	if tableWidth < 10 {
		tableWidth = 10
	}
	for i := range o.tables {
		if o.tables[i] != nil {
			o.tables[i].SetSize(tableWidth, tableHeight)
		}
	}
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
		return o.handleSearchResults(m)

	case tea.KeyMsg:
		return o.handleKey(m)
	}

	// Forward key events to text input for cursor blinking.
	var cmd tea.Cmd
	o.input, cmd = o.input.Update(msg)
	return o, cmd
}

// handleSearchResults processes SearchResultsMsg — either a fresh query result or
// a paginated page append. All store mutations (ClearSearchBuffers, AppendSearch*,
// SetSearchTotal, MarkSearchOffsetFetched) are performed by app.go before this
// method is called. This method only updates local UI state.
// NOTE: m.Err is handled by app.go before reaching the overlay.
func (o *SearchOverlay) handleSearchResults(m SearchResultsMsg) (tea.Model, tea.Cmd) {
	if m.Results == nil {
		return o, nil
	}

	if !m.IsPaged {
		// New query — clear local table rows, rebuild from store, switch to tracks tab.
		o.clearBuffers()
		o.refreshAllRows()
		o.activeSection = sectionTracks
		o.syncTableFocus()
	} else {
		// Paginated load — refresh only the section that received new data.
		switch m.Section {
		case sectionTracks:
			o.refreshTrackRows()
		case sectionArtists:
			o.refreshArtistRows()
		case sectionAlbums:
			o.refreshAlbumRows()
		case sectionPlaylists:
			o.refreshPlaylistRows()
		}
	}
	return o, nil
}

// syncTableFocus ensures only the active section's table is focused.
func (o *SearchOverlay) syncTableFocus() {
	for i := range o.tables {
		if o.tables[i] != nil {
			o.tables[i].SetFocused(searchSection(i) == o.activeSection)
		}
	}
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

	case tea.KeyUp, tea.KeyDown:
		// Forward navigation to the active table, then check for prefetch.
		cmd := o.tables[o.activeSection].Update(m)
		prefetchCmd := o.checkPrefetch()
		return o, tea.Batch(cmd, prefetchCmd)

	case tea.KeyCtrlA:
		return o.handleAddToQueue()

	case tea.KeyCtrlU:
		// Clear local input, buffers, and reset tables — visual reset happens immediately.
		// Store writes are deferred: emit SearchClearedMsg for the root app to handle.
		o.input.SetValue("")
		o.clearBuffers()
		o.refreshAllRows()
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
	tracks := o.store.SearchTracks()
	if o.activeSection != sectionTracks || len(tracks) == 0 {
		return o, nil
	}
	idx := o.tables[sectionTracks].SelectedIndex()
	if idx < 0 || idx >= len(tracks) {
		return o, nil
	}
	trackURI := tracks[idx].URI
	return o, func() tea.Msg { return AddToQueueMsg{TrackURI: trackURI} }
}

// moveSectionForward advances the active section, wrapping from last to first.
func (o *SearchOverlay) moveSectionForward() (tea.Model, tea.Cmd) {
	o.activeSection = (o.activeSection + 1) % numSections
	o.syncTableFocus()
	return o, nil
}

// moveSectionBackward retreats the active section, wrapping from first to last.
func (o *SearchOverlay) moveSectionBackward() (tea.Model, tea.Cmd) {
	o.activeSection = (o.activeSection + numSections - 1) % numSections
	o.syncTableFocus()
	return o, nil
}

// selectedURI returns the URI for the currently selected result item
// and whether it is a track (vs. a context URI).
func (o *SearchOverlay) selectedURI() (uri string, isTrack bool) {
	idx := o.tables[o.activeSection].SelectedIndex()
	switch o.activeSection {
	case sectionTracks:
		tracks := o.store.SearchTracks()
		if idx >= 0 && idx < len(tracks) {
			return tracks[idx].URI, true
		}
	case sectionArtists:
		artists := o.store.SearchArtists()
		if idx >= 0 && idx < len(artists) {
			return artists[idx].URI, false
		}
	case sectionAlbums:
		albums := o.store.SearchAlbums()
		if idx >= 0 && idx < len(albums) {
			return albums[idx].URI, false
		}
	case sectionPlaylists:
		playlists := o.store.SearchPlaylists()
		if idx >= 0 && idx < len(playlists) {
			return playlists[idx].URI, false
		}
	}
	return "", false
}

// checkPrefetch fires a SearchPageRequestMsg when the cursor crosses the 50%
// mark of the last fetched page and more results exist.
func (o *SearchOverlay) checkPrefetch() tea.Cmd {
	sec := o.activeSection
	cursor := o.tables[sec].SelectedIndex()
	bufLen := o.sectionBufferLen(sec)
	total := o.store.SearchSectionTotal(int(sec))

	if bufLen == 0 || bufLen >= total {
		return nil // empty or all pages loaded
	}

	// Prefetch when cursor crosses 50% of the last loaded page.
	lastPageStart := bufLen - min(bufLen, maxResultsPerSection)
	midpoint := lastPageStart + maxResultsPerSection/2

	if cursor >= midpoint {
		nextOffset := bufLen // next unloaded offset
		if o.store.IsSearchOffsetFetched(int(sec), nextOffset) {
			return nil // already fetched or in-flight
		}
		return o.requestPage(nextOffset)
	}
	return nil
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
// Assembly order: tab bar → tab separator → table.View() → [padding] → help bar.
//
// innerWidth is the content width inside the border (View subtracts 2 for border chars).
// availableHeight is innerHeight - 2 (input line + dot separator already consumed).
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

	hasResults := len(o.store.SearchTracks())+len(o.store.SearchArtists())+len(o.store.SearchAlbums())+len(o.store.SearchPlaylists()) > 0

	if !hasResults {
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

	// NOTE: Table SetSize and narrow-mode rebuild are intentionally NOT done here.
	// renderResults is called from View() which must be pure (no state mutations).
	// SetSize() and SetSize-driven rebuilds are handled by the overlay's SetSize() method,
	// which is called by the root app on every WindowSizeMsg before the next View().

	mutedStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())

	var sb strings.Builder
	// Tab bar
	sb.WriteString(o.renderTabBar(contentWidth))
	sb.WriteString("\n")
	// Tab separator
	sb.WriteString(mutedStyle.Render(strings.Repeat("─", contentWidth)))
	sb.WriteString("\n")
	// Active section table output
	sb.WriteString(o.tables[o.activeSection].View())
	sb.WriteString("\n")

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

// renderHelpBar renders a separator line and a contextual keybindings line.
// Key labels use KeyHint() + Bold(true); descriptions use TextMuted().
// "Ctrl+A queue" only appears on the Tracks section.
// When the active section has more results than maxResultsPerSection, a right-aligned
// page indicator (e.g., "1-10 of 39") is appended to the keybindings line.
func (o *SearchOverlay) renderHelpBar(contentWidth int) string {
	mutedStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
	keyStyle := lipgloss.NewStyle().Foreground(o.theme.KeyHint()).Bold(true)
	separator := mutedStyle.Render(strings.Repeat("─", contentWidth))

	type hint struct{ Key, Label string }
	hints := []hint{
		{"Tab", "next section"},
		{"↑↓", "navigate"},
		{"Enter", "play"},
	}
	if o.activeSection == sectionTracks {
		hints = append(hints, hint{"Ctrl+A", "queue"})
	}
	hints = append(hints, hint{"Esc", "close"})

	var parts []string
	for _, h := range hints {
		parts = append(parts, keyStyle.Render(h.Key)+" "+mutedStyle.Render(h.Label))
	}
	keysLine := strings.Join(parts, "  ")

	// Right-align page indicator when present.
	indicator := o.pageIndicator()
	if indicator != "" {
		keysWidth := lipgloss.Width(keysLine)
		indicatorWidth := lipgloss.Width(mutedStyle.Render(indicator))
		gap := contentWidth - keysWidth - indicatorWidth
		if gap > 0 {
			keysLine += strings.Repeat(" ", gap) + mutedStyle.Render(indicator)
		} else {
			keysLine += " " + mutedStyle.Render(indicator)
		}
	}

	return separator + "\n" + keysLine
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

// renderTabBar renders a horizontal tab bar showing all four sections with their
// result counts. The active section is highlighted with ▪ + bold + its tab color.
// Inactive sections use TextMuted. Tabs are separated by 5 spaces.
func (o *SearchOverlay) renderTabBar(width int) string {
	var parts []string

	for i := searchSection(0); i < numSections; i++ {
		label := fmt.Sprintf("%s %d", searchSectionLabels[i], o.store.SearchSectionTotal(int(i)))
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

// totalForSection returns the total result count for the given section from the store.
func (o *SearchOverlay) totalForSection(sec searchSection) int {
	return o.store.SearchSectionTotal(int(sec))
}

// TotalForSection is the exported wrapper of totalForSection for use in tests.
func (o *SearchOverlay) TotalForSection(sec searchSection) int {
	return o.totalForSection(sec)
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
// the total exceeds maxResultsPerSection (e.g., "1-10 of 39").
// Returns empty string when all results fit on one page.
func (o *SearchOverlay) pageIndicator() string {
	sec := o.activeSection
	total := o.store.SearchSectionTotal(int(sec))
	bufLen := o.sectionBufferLen(sec)
	if total <= maxResultsPerSection {
		return ""
	}
	if bufLen == 0 {
		return ""
	}
	// Show range based on the visible page window (bubble-table pagination).
	cursor := o.tables[sec].SelectedIndex()
	pageSize := maxResultsPerSection
	pageStart := (cursor / pageSize) * pageSize
	start := pageStart + 1
	end := min(pageStart+pageSize, bufLen)
	if end > total {
		end = total
	}
	return fmt.Sprintf("%d-%d of %d", start, end, total)
}

// PageIndicator is the exported wrapper of pageIndicator for use in tests.
func (o *SearchOverlay) PageIndicator() string {
	return o.pageIndicator()
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
// Rebuilds all 4 tables with the new theme colors and re-populates rows.
func (o *SearchOverlay) SetTheme(th theme.Theme) {
	o.theme = th
	o.buildAllTables(o.narrowTracks)
	o.refreshAllRows()
	// Update spinner style for the new theme.
	o.spinner.Style = lipgloss.NewStyle().Foreground(th.TextMuted())
}

// Tables returns the array of table pointers for inspection in tests.
func (o *SearchOverlay) Tables() [numSections]*components.Table {
	return o.tables
}

// BufTracksLen returns the number of accumulated track items in the store. Used in tests.
func (o *SearchOverlay) BufTracksLen() int { return len(o.store.SearchTracks()) }

// BufArtistsLen returns the number of accumulated artist items in the store. Used in tests.
func (o *SearchOverlay) BufArtistsLen() int { return len(o.store.SearchArtists()) }

// BufAlbumsLen returns the number of accumulated album items in the store. Used in tests.
func (o *SearchOverlay) BufAlbumsLen() int { return len(o.store.SearchAlbums()) }

// BufPlaylistsLen returns the number of accumulated playlist items in the store. Used in tests.
func (o *SearchOverlay) BufPlaylistsLen() int { return len(o.store.SearchPlaylists()) }

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
