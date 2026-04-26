// Package panes — AlbumsPane displays the user's saved albums in a dense table
// with in-pane filtering. Pressing Enter on an album opens a track sub-view that
// lets the user pick a specific track to play within the album's context.
package panes

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Compile-time check: AlbumsPane implements layout.Pane.
var _ layout.Pane = &AlbumsPane{}

// albumDebounceIntent is a snapshot of the user's desired album at the moment of
// pressing Enter. The debounce tick carries this snapshot; if the current intent
// has changed by the time the tick fires, the tick is discarded.
type albumDebounceIntent struct {
	albumID string
	offset  int
}

// albumDebounceMsg is the internal 150ms tick fired by scheduleAlbumDebounce.
// It is never forwarded to app.go — handled entirely within the pane.
type albumDebounceMsg struct {
	intent albumDebounceIntent
}

// AlbumsPane is the Bubble Tea model for the Albums pane (toggle key 4).
// It renders a dense bubble-table of the user's saved albums with columns
// for index, name, artist, and year. It supports in-pane filtering by album
// name and artist.
//
// When the user presses Enter on an album, the pane transitions to a track
// sub-view. Album tracks are interactive (user-session) data — they are stored
// in pane fields (loadedTracks), not in the global store.
type AlbumsPane struct {
	BasePane

	// table renders the album list.
	table *components.Table
	// filter provides in-pane text filtering by album name and artist.
	filter *components.Filter
	// trackTable renders the track list when inTrackView is true.
	trackTable *components.Table

	// Track sub-view identity (which album is open)
	inTrackView  bool
	selectedID   string // Spotify album ID (e.g. "1weenld61qoidwYuZ1GESA")
	selectedURI  string // Spotify album URI for PlayContextMsg
	selectedName string // Album display name for sub-view title

	// Track sub-view data (pane-owned, NOT in global store)
	loadedTracks []domain.Track // all tracks fetched so far for this album

	// Pagination state (pane-owned)
	trackOffset    int  // count of tracks fetched so far (= len(loadedTracks))
	hasMoreTracks  bool // last API response had next != ""
	tracksFetching bool // a request is in-flight; blocks duplicate prefetch

	// Debounce (protects rapid album switching)
	albumIntent albumDebounceIntent // current desired album
}

// NewAlbumsPane creates an AlbumsPane with the given store, theme, and focus state.
func NewAlbumsPane(store state.StateReader, th theme.Theme, focused bool) *AlbumsPane {
	// Album columns: # 5% | Name 50% | Artist 30% | Year 15%
	// Flex factors: 1 : 10 : 6 : 3 ≈ 5% / 50% / 30% / 15%
	columns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Name", FlexFactor: 10, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary()},
		{Key: "year", Header: "Year", FlexFactor: 3, Color: th.ColumnTertiary()},
	}

	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	// Track sub-view columns: # 5% | Track 50% | Artist 30% | Duration 15%
	// Flex factors: 1 : 10 : 6 : 3 (same proportions as playlist track sub-view)
	trackCols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Track", FlexFactor: 10, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
	}
	tt := components.NewTable(components.TableConfig{
		Columns:      trackCols,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	a := &AlbumsPane{
		BasePane:   BasePane{store: store, theme: th, focused: focused},
		table:      t,
		trackTable: tt,
		filter:     components.NewFilter(th),
	}
	t.SetFocused(focused)
	a.refreshRows()
	return a
}

// ID returns PaneAlbums — the identifier for the albums grid slot.
func (a *AlbumsPane) ID() layout.PaneID { return layout.PaneAlbums }

// Title returns the pane title. In track sub-view it shows the album name and
// number of loaded tracks.
func (a *AlbumsPane) Title() string {
	if a.inTrackView {
		return fmt.Sprintf("Albums ── %s (%d tracks)", a.selectedName, len(a.loadedTracks))
	}
	return "Albums"
}

// ToggleKey returns 4 — the number key for btop-style pane toggling.
func (a *AlbumsPane) ToggleKey() int { return 4 }

// Actions returns the pane-specific shortcut hints displayed in the border.
func (a *AlbumsPane) Actions() []layout.Action {
	if a.inTrackView {
		return []layout.Action{{Key: "Esc", Label: "back"}}
	}
	if a.filter.IsActive() {
		return []layout.Action{{Key: "Esc", Label: "close"}}
	}
	return []layout.Action{{Key: "f", Label: "filter"}}
}

// Init satisfies tea.Model. AlbumsPane has no startup command.
func (a *AlbumsPane) Init() tea.Cmd { return nil }

// HasActiveFilter returns true when the in-pane filter is capturing keystrokes.
func (a *AlbumsPane) HasActiveFilter() bool { return a.filter.IsActive() }

// SetFocused updates the keyboard focus state. Routes focus to the correct table
// based on whether the track sub-view is active.
func (a *AlbumsPane) SetFocused(focused bool) {
	a.BasePane.SetFocused(focused)
	if a.inTrackView {
		a.trackTable.SetFocused(focused)
		a.table.SetFocused(false)
	} else {
		a.table.SetFocused(focused && !a.filter.IsActive())
		a.trackTable.SetFocused(false)
	}
}

// SetSize updates the render dimensions and propagates them to both tables and the filter.
func (a *AlbumsPane) SetSize(width, height int) {
	a.BasePane.SetSize(width, height)
	a.filter.SetWidth(width)
	a.trackTable.SetSize(width, height)
	a.resizeTable()
}

// Update handles key events and data-loaded messages.
func (a *AlbumsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle data-loaded messages regardless of focus.
	switch m := msg.(type) {
	case AlbumsLoadedMsg:
		a.refreshRows()
		return a, nil

	case AlbumTracksLoadedMsg:
		if m.AlbumID != a.selectedID {
			return a, nil // double guard — app.go staleness check should already discard
		}
		a.tracksFetching = false
		if m.Err != nil {
			// Error was already toasted by app.go; just clear fetching state.
			return a, nil
		}
		if m.Offset == 0 {
			a.loadedTracks = m.Tracks
		} else {
			a.loadedTracks = append(a.loadedTracks, m.Tracks...)
		}
		a.trackOffset = len(a.loadedTracks)
		a.hasMoreTracks = m.HasNext
		a.refreshTrackRows()
		return a, nil

	case albumDebounceMsg:
		if m.intent != a.albumIntent {
			return a, nil // stale tick — user switched album or pressed Esc
		}
		a.tracksFetching = true
		intent := m.intent
		return a, func() tea.Msg {
			return FetchAlbumTracksRequestMsg{AlbumID: intent.albumID, Offset: intent.offset}
		}
	}

	if !a.focused {
		return a, nil
	}

	// When filter is active, forward all key events to the filter.
	if a.filter.IsActive() {
		cmd := a.filter.Update(msg)
		if !a.filter.IsActive() {
			a.table.SetFocused(true)
			a.resizeTable()
		}
		a.refreshRows()
		return a, cmd
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}

	// Route key events based on which view is active.
	if a.inTrackView {
		return a.handleTrackViewKey(keyMsg)
	}
	return a.handleListViewKey(keyMsg)
}

// handleListViewKey handles key events in the album list view.
func (a *AlbumsPane) handleListViewKey(keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "f":
		a.filter.Toggle()
		a.table.SetFocused(false)
		a.resizeTable()
		return a, nil

	case keyMsg.Type == tea.KeyEnter:
		albums := a.filteredAlbums()
		idx := a.table.SelectedIndex()
		if idx < 0 || idx >= len(albums) {
			return a, nil
		}
		alb := albums[idx].Album
		a.selectedID = alb.ID
		a.selectedURI = alb.URI
		a.selectedName = alb.Name
		a.loadedTracks = nil
		a.trackOffset = 0
		a.hasMoreTracks = false
		a.tracksFetching = false
		a.inTrackView = true
		a.table.SetFocused(false)
		a.trackTable.SetFocused(true)
		intent := albumDebounceIntent{albumID: alb.ID, offset: 0}
		a.albumIntent = intent
		return a.scheduleAlbumDebounce(intent)

	// Esc with no active filter: reset scroll to page 1.
	case keyMsg.Type == tea.KeyEscape:
		a.table.GotoTop()
		return a, nil
	}

	// Forward navigation to the table.
	cmd := a.table.Update(keyMsg)
	return a, cmd
}

// handleTrackViewKey handles key events in the track sub-view.
func (a *AlbumsPane) handleTrackViewKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		a.inTrackView = false
		a.loadedTracks = nil
		a.trackOffset = 0
		a.hasMoreTracks = false
		a.tracksFetching = false
		// Clear albumIntent so any in-flight debounce tick is discarded as stale.
		a.albumIntent = albumDebounceIntent{}
		a.trackTable.SetFocused(false)
		a.table.SetFocused(true)
		return a, func() tea.Msg { return AlbumTrackViewClosedMsg{} }

	case tea.KeyEnter:
		idx := a.trackTable.SelectedIndex()
		if idx < 0 || idx >= len(a.loadedTracks) {
			return a, nil
		}
		track := a.loadedTracks[idx]
		albumURI := a.selectedURI
		return a, func() tea.Msg {
			return PlayContextMsg{
				ContextURI: albumURI,
				OffsetURI:  track.URI,
			}
		}
	}

	cmd := a.trackTable.Update(key)
	prefetchCmd := a.checkPrefetch()
	return a, tea.Batch(cmd, prefetchCmd)
}

// scheduleAlbumDebounce returns a 150ms debounce command for the given album intent.
// The intent is carried in the tick message so the handler can detect stale ticks.
func (a *AlbumsPane) scheduleAlbumDebounce(intent albumDebounceIntent) (tea.Model, tea.Cmd) {
	return a, tea.Tick(150*time.Millisecond, func(_ time.Time) tea.Msg {
		return albumDebounceMsg{intent: intent}
	})
}

// checkPrefetch fires a lazy pagination request when the cursor is within 5 rows
// of the end of loaded tracks, there are more pages, and no fetch is in-flight.
// Returns nil when no prefetch is needed.
func (a *AlbumsPane) checkPrefetch() tea.Cmd {
	if !a.hasMoreTracks || a.tracksFetching {
		return nil
	}
	cursor := a.trackTable.SelectedIndex()
	if cursor < len(a.loadedTracks)-5 {
		return nil
	}
	a.tracksFetching = true
	offset := a.trackOffset
	albumID := a.selectedID
	return func() tea.Msg {
		return FetchAlbumTracksRequestMsg{AlbumID: albumID, Offset: offset}
	}
}

// refreshTrackRows rebuilds the track table rows from loadedTracks.
func (a *AlbumsPane) refreshTrackRows() {
	rows := make([]map[string]string, len(a.loadedTracks))
	for i, tr := range a.loadedTracks {
		artistName := ""
		if len(tr.Artists) > 0 {
			artistName = tr.Artists[0].Name
		}
		rows[i] = map[string]string{
			"index":    fmt.Sprintf("%d", i+1),
			"name":     tr.Name,
			"artist":   artistName,
			"duration": formatDurationMs(tr.DurationMs),
		}
	}
	a.trackTable.SetRows(rows)
}

// View renders the albums pane content. Pure — reads state, returns string.
func (a *AlbumsPane) View() string {
	if a.inTrackView {
		return a.trackTable.View()
	}
	var parts []string
	if a.filter.IsActive() {
		parts = append(parts, a.filter.View(a.width))
	}
	parts = append(parts, a.table.View())
	return strings.Join(parts, "\n")
}

// RefreshRows re-reads the store and applies updated rows to the table.
// The app calls this after updating the store.
func (a *AlbumsPane) RefreshRows() { a.refreshRows() }

// refreshRows re-reads the store and applies filtered album rows.
func (a *AlbumsPane) refreshRows() {
	albums := a.filteredAlbums()
	rows := make([]map[string]string, len(albums))
	for i, sa := range albums {
		artistName := ""
		if len(sa.Album.Artists) > 0 {
			artistName = sa.Album.Artists[0].Name
		}
		year := extractYear(sa.Album.ReleaseDate)
		rows[i] = map[string]string{
			"index":  fmt.Sprintf("%d", i+1),
			"name":   sa.Album.Name,
			"artist": artistName,
			"year":   year,
		}
	}
	a.table.SetRows(rows)
}

// filteredAlbums returns the albums filtered by the current filter query.
// Filter matches on album name OR artist name.
func (a *AlbumsPane) filteredAlbums() []domain.SavedAlbum {
	all := a.store.SavedAlbums()
	if a.filter.Query() == "" {
		return all
	}
	result := make([]domain.SavedAlbum, 0, len(all))
	for _, sa := range all {
		artistName := ""
		if len(sa.Album.Artists) > 0 {
			artistName = sa.Album.Artists[0].Name
		}
		if a.filter.MatchesAny(sa.Album.Name, artistName) {
			result = append(result, sa)
		}
	}
	return result
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (a *AlbumsPane) resizeTable() {
	tableHeight := a.height
	if a.filter.IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	a.table.SetSize(a.width, tableHeight)
}

// SetTheme updates the theme reference and rebuilds both tables with new column colors.
// Called when the user switches themes at runtime.
func (a *AlbumsPane) SetTheme(th theme.Theme) {
	a.theme = th
	cols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Name", FlexFactor: 10, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary()},
		{Key: "year", Header: "Year", FlexFactor: 3, Color: th.ColumnTertiary()},
	}
	a.table, a.filter = components.RebuildTableTheme(th, cols, a.table.Rows(), a.focused && !a.inTrackView)
	a.resizeTable()
	a.refreshRows()

	// Rebuild track table with new column colors.
	trackCols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Track", FlexFactor: 10, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
	}
	a.trackTable = components.NewTable(components.TableConfig{
		Columns:      trackCols,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})
	a.trackTable.SetSize(a.width, a.height)
	if a.inTrackView {
		a.trackTable.SetFocused(a.focused)
		a.refreshTrackRows()
	}
}

// Cursor returns the currently selected row index (0-based).
func (a *AlbumsPane) Cursor() int {
	return a.table.SelectedIndex()
}

// extractYear extracts the 4-digit year from a Spotify release date string.
// Spotify returns dates as "YYYY", "YYYY-MM", or "YYYY-MM-DD".
func extractYear(releaseDate string) string {
	if len(releaseDate) >= 4 {
		return releaseDate[:4]
	}
	return releaseDate
}
