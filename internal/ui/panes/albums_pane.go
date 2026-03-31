// Package panes — AlbumsPane displays the user's saved albums in a dense table
// with in-pane filtering. Selecting an album emits a PlayContextMsg to play it.
package panes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Compile-time check: AlbumsPane implements layout.Pane.
var _ layout.Pane = &AlbumsPane{}

// AlbumsPane is the Bubble Tea model for the Albums pane (toggle key 4).
// It renders a dense bubble-table of the user's saved albums with columns
// for index, name, artist, and year. It supports in-pane filtering by album
// name and artist and emits PlayContextMsg when the user presses Enter.
type AlbumsPane struct {
	store   *state.Store
	theme   theme.Theme
	focused bool

	width  int
	height int

	// table renders the album list.
	table *components.Table
	// filter provides in-pane text filtering by album name and artist.
	filter *components.Filter
}

// NewAlbumsPane creates an AlbumsPane with the given store, theme, and focus state.
func NewAlbumsPane(store *state.Store, th theme.Theme, focused bool) *AlbumsPane {
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

	a := &AlbumsPane{
		store:   store,
		theme:   th,
		focused: focused,
		table:   t,
		filter:  components.NewFilter(th),
	}
	t.SetFocused(focused)
	a.refreshRows()
	return a
}

// ID returns PaneAlbums — the identifier for the albums grid slot.
func (a *AlbumsPane) ID() layout.PaneID { return layout.PaneAlbums }

// Title returns "Albums".
func (a *AlbumsPane) Title() string { return "Albums" }

// ToggleKey returns 4 — the number key for btop-style pane toggling.
func (a *AlbumsPane) ToggleKey() int { return 4 }

// Actions returns the pane-specific shortcut hints displayed in the border.
func (a *AlbumsPane) Actions() []layout.Action {
	if a.filter.IsActive() {
		return []layout.Action{{Key: "Esc", Label: "close"}}
	}
	return []layout.Action{{Key: "f", Label: "filter"}}
}

// Init satisfies tea.Model. AlbumsPane has no startup command.
func (a *AlbumsPane) Init() tea.Cmd { return nil }

// IsFocused returns true when the pane has keyboard focus.
func (a *AlbumsPane) IsFocused() bool { return a.focused }

// HasActiveFilter returns true when the in-pane filter is capturing keystrokes.
func (a *AlbumsPane) HasActiveFilter() bool { return a.filter.IsActive() }

// SetFocused updates the keyboard focus state.
func (a *AlbumsPane) SetFocused(focused bool) {
	a.focused = focused
	a.table.SetFocused(focused && !a.filter.IsActive())
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (a *AlbumsPane) SetSize(width, height int) {
	a.width = width
	a.height = height
	a.filter.SetWidth(width)
	a.resizeTable()
}

// Update handles key events and data-loaded messages.
func (a *AlbumsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle data-loaded messages regardless of focus.
	switch msg.(type) {
	case AlbumsLoadedMsg:
		a.refreshRows()
		return a, nil
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

	switch {
	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "f":
		a.filter.Toggle()
		a.table.SetFocused(false)
		a.resizeTable()
		return a, nil

	case keyMsg.Type == tea.KeyEnter:
		albums := a.filteredAlbums()
		idx := a.table.SelectedIndex()
		if idx >= 0 && idx < len(albums) {
			uri := albums[idx].Album.URI
			return a, func() tea.Msg {
				return PlayContextMsg{ContextURI: uri}
			}
		}
		return a, nil
	}

	// Forward navigation to the table.
	cmd := a.table.Update(keyMsg)
	return a, cmd
}

// View renders the albums pane content. Pure — reads state, returns string.
func (a *AlbumsPane) View() string {
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

// SetTheme updates the theme reference and rebuilds the table with new column colors.
// Called when the user switches themes at runtime.
func (a *AlbumsPane) SetTheme(th theme.Theme) {
	a.theme = th
	a.filter = components.NewFilter(th)
	columns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Name", FlexFactor: 10, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary()},
		{Key: "year", Header: "Year", FlexFactor: 3, Color: th.ColumnTertiary()},
	}
	a.table = components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})
	a.table.SetFocused(a.focused)
	a.table.SetSize(a.width, a.height)
	a.refreshRows()
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
