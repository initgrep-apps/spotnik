// Package panes — TopArtistsPane displays the user's top artists in a dense
// bubble-table with in-pane filtering and time range cycling via the t key.
// The genre column shows the first genre from each artist's genre list.
// Implements layout.Pane (toggle key 8).
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

// Compile-time check: TopArtistsPane implements layout.Pane.
var _ layout.Pane = &TopArtistsPane{}

// topArtistsTimeRanges is the cycle order for the t key.
var topArtistsTimeRanges = []string{"short_term", "medium_term", "long_term"}

// topArtistsRangeLabels maps API values to human-readable display labels.
var topArtistsRangeLabels = map[string]string{
	"short_term":  "4wk",
	"medium_term": "6mo",
	"long_term":   "all",
}

// TopArtistsPane is the Bubble Tea model for the Top Artists pane (toggle key 8).
// It renders a dense bubble-table of the user's top artists with columns for index,
// name, and genre (first genre from each artist's genre list). It supports in-pane
// filtering by artist name and genre, and per-pane time range cycling via the t key.
type TopArtistsPane struct {
	store   *state.Store
	theme   theme.Theme
	focused bool

	width  int
	height int

	// timeRange is the currently active Spotify time range for top artists.
	timeRange string

	// table renders the top artists list.
	table *components.Table
	// filter provides in-pane text filtering by artist name and genre.
	filter *components.Filter
}

// NewTopArtistsPane creates a TopArtistsPane with the given store, theme, and focus state.
// Default time range is short_term (4 weeks).
func NewTopArtistsPane(store *state.Store, th theme.Theme, focused bool) *TopArtistsPane {
	// Column widths per DESIGN.md §9: # 5% | Name 70% | Genre 25%
	// Flex factors: 1 : 14 : 5 ≈ 5% / 70% / 25%
	columns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.TextMuted()},
		{Key: "name", Header: "Artist", FlexFactor: 14, Color: th.TextPrimary()},
		{Key: "genre", Header: "Genre", FlexFactor: 5, Color: th.TextSecondary()},
	}

	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	a := &TopArtistsPane{
		store:     store,
		theme:     th,
		focused:   focused,
		timeRange: "short_term",
		table:     t,
		filter:    components.NewFilter(th),
	}
	t.SetFocused(focused)
	a.refreshRows()
	return a
}

// ID returns PaneTopArtists — the identifier for the top artists grid slot.
func (a *TopArtistsPane) ID() layout.PaneID { return layout.PaneTopArtists }

// Title returns "Top Artists".
func (a *TopArtistsPane) Title() string { return "Top Artists" }

// ToggleKey returns 8 — the number key for btop-style pane toggling.
func (a *TopArtistsPane) ToggleKey() int { return 8 }

// TimeRange returns the currently active time range string (exported for testing).
func (a *TopArtistsPane) TimeRange() string { return a.timeRange }

// Actions returns the pane-specific shortcut hints displayed in the border.
// When the filter is active, only shows the Esc/close hint.
// Otherwise shows filter (f) and time range cycle (t) with the current range as label.
func (a *TopArtistsPane) Actions() []layout.Action {
	if a.filter.IsActive() {
		return []layout.Action{{Key: "Esc", Label: "close"}}
	}
	rangeLabel := topArtistsRangeLabels[a.timeRange]
	return []layout.Action{
		{Key: "f", Label: "filter"},
		{Key: "t", Label: rangeLabel},
	}
}

// Init satisfies tea.Model. TopArtistsPane has no startup command.
func (a *TopArtistsPane) Init() tea.Cmd { return nil }

// IsFocused returns true when the pane has keyboard focus.
func (a *TopArtistsPane) IsFocused() bool { return a.focused }

// SetFocused updates the keyboard focus state.
func (a *TopArtistsPane) SetFocused(focused bool) {
	a.focused = focused
	a.table.SetFocused(focused && !a.filter.IsActive())
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (a *TopArtistsPane) SetSize(width, height int) {
	a.width = width
	a.height = height
	a.filter.SetWidth(width)
	a.resizeTable()
}

// Update handles key events and data-loaded messages.
func (a *TopArtistsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle data-loaded messages regardless of focus.
	switch m := msg.(type) {
	case StatsLoadedMsg:
		if m.TimeRange == a.timeRange {
			a.refreshRows()
		}
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

	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "t":
		return a.cycleTimeRange()
	}

	// Forward navigation to the table.
	// NOTE: Enter has no action for artists — artists aren't directly playable
	// (PlayContextMsg requires explicit artist play support).
	cmd := a.table.Update(keyMsg)
	return a, cmd
}

// cycleTimeRange advances to the next time range, checking the store cache first.
// On a cache hit, renders immediately with no fetch. On a miss, emits FetchStatsMsg.
func (a *TopArtistsPane) cycleTimeRange() (tea.Model, tea.Cmd) {
	currentIdx := 0
	for i, r := range topArtistsTimeRanges {
		if r == a.timeRange {
			currentIdx = i
			break
		}
	}
	nextRange := topArtistsTimeRanges[(currentIdx+1)%len(topArtistsTimeRanges)]
	a.timeRange = nextRange
	a.refreshRows()

	// Check if data for this range is already cached.
	if a.store.TopArtists(nextRange) != nil {
		return a, nil
	}

	// Cache miss — emit a request for the root app to fetch from the API.
	timeRange := nextRange
	return a, func() tea.Msg { return FetchStatsMsg{TimeRange: timeRange} }
}

// View renders the top artists pane content. Pure — reads state, returns string.
func (a *TopArtistsPane) View() string {
	var parts []string
	if a.filter.IsActive() {
		parts = append(parts, a.filter.View(a.width))
	}
	parts = append(parts, a.table.View())
	return strings.Join(parts, "\n")
}

// RefreshRows re-reads the store and applies updated rows to the table.
// The app calls this after updating the store.
func (a *TopArtistsPane) RefreshRows() { a.refreshRows() }

// refreshRows re-reads the store and applies filtered artist rows.
func (a *TopArtistsPane) refreshRows() {
	artists := a.filteredArtists()
	rows := make([]map[string]string, len(artists))
	for i, artist := range artists {
		genre := "—"
		if len(artist.Genres) > 0 {
			genre = artist.Genres[0]
		}
		rows[i] = map[string]string{
			"index": fmt.Sprintf("%d", i+1),
			"name":  artist.Name,
			"genre": genre,
		}
	}
	a.table.SetRows(rows)
}

// filteredArtists returns the top artists for the current time range, filtered by query.
// Filter matches on artist name OR first genre.
func (a *TopArtistsPane) filteredArtists() []domain.FullArtist {
	all := a.store.TopArtists(a.timeRange)
	if a.filter.Query() == "" {
		return all
	}
	result := make([]domain.FullArtist, 0, len(all))
	for _, artist := range all {
		genre := ""
		if len(artist.Genres) > 0 {
			genre = artist.Genres[0]
		}
		if a.filter.MatchesAny(artist.Name, genre) {
			result = append(result, artist)
		}
	}
	return result
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (a *TopArtistsPane) resizeTable() {
	tableHeight := a.height
	if a.filter.IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	a.table.SetSize(a.width, tableHeight)
}
