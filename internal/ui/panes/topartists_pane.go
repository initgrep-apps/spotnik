// Package panes — TopArtistsPane displays the user's top artists in a dense
// bubble-table with in-pane filtering and time range cycling via the g key.
// Enter on a row emits PlayContextMsg to play the selected artist.
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
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// Compile-time check: TopArtistsPane implements layout.Pane.
var _ layout.Pane = &TopArtistsPane{}

// Compile-time check: TopArtistsPane implements layout.FilterQueryPane.
var _ layout.FilterQueryPane = &TopArtistsPane{}

// topArtistsTimeRanges is the cycle order for the g key.
var topArtistsTimeRanges = []string{"short_term", "medium_term", "long_term"}

// topArtistsRangeLabels maps API values to human-readable display labels.
var topArtistsRangeLabels = map[string]string{
	"short_term":  "4wk",
	"medium_term": "6mo",
	"long_term":   "all",
}

// TopArtistsPane is the Bubble Tea model for the Top Artists pane (toggle key 8).
// It renders a dense bubble-table of the user's top artists with columns for index,
// name, popularity (star-graded), and follower count. It supports in-pane filtering
// by artist name and per-pane time range cycling via the g key.
// Enter on a selected row emits PlayContextMsg to start playback of that artist.
type TopArtistsPane struct {
	*TableBasedPane

	// timeRange is the currently active Spotify time range for top artists.
	timeRange string
}

// NewTopArtistsPane creates a TopArtistsPane with the given store, theme, and focus state.
// Default time range is short_term (4 weeks).
func NewTopArtistsPane(store state.StateReader, th theme.Theme, focused bool) *TopArtistsPane {
	// Column widths per DESIGN.md §9: # 5% | Name 55% | Pop 20% | Flw 20%
	// Flex factors: 1 : 11 : 4 : 4 ≈ 5% / 55% / 20% / 20%
	columns := []components.ColumnDef{
		{Key: "name", Header: "Artist", FlexFactor: 11, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "pop", Header: "Pop", FlexFactor: 4, Color: th.ColumnSecondary(), Priority: 3},
		{Key: "flw", Header: "Flw", FlexFactor: 4, Color: th.ColumnTertiary(), Priority: 3},
	}

	t := components.NewTable(components.TableConfig{
		Columns:    columns,
		Theme:      th,
		ShowHeader: true,
	})

	a := &TopArtistsPane{
		TableBasedPane: NewTableBasedPane(store, th, focused, t, components.NewFilter(th)),
		timeRange:      "short_term",
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
// Shows filter (f) and time range cycle (g) with the current range as label.
// The close-notch is retired — the border renderer uses FilterQuery instead.
func (a *TopArtistsPane) Actions() []layout.Action {
	rangeLabel := topArtistsRangeLabels[a.timeRange]
	return []layout.Action{
		a.BaseFilterAction(),
		{Key: "g", Label: rangeLabel},
	}
}

// Init satisfies tea.Model. TopArtistsPane has no startup command.
func (a *TopArtistsPane) Init() tea.Cmd { return nil }

// SetFocused updates the keyboard focus state and propagates it to the table.
func (a *TopArtistsPane) SetFocused(focused bool) {
	a.TableBasedPane.SetFocused(focused)
	a.Table().SetFocused(focused && !a.Filter().IsActive())
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (a *TopArtistsPane) SetSize(width, height int) {
	a.TableBasedPane.SetSize(width, height)
	a.Filter().SetWidth(width)
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

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}

	// Delegate filter-key routing to the base.
	if consumed, cmd := a.HandleFilterKey(keyMsg, a.refreshRows, a.resizeTable); consumed {
		return a, cmd
	}

	switch {
	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "g":
		return a.cycleTimeRange()

	case keyMsg.Type == tea.KeyEnter:
		artists := a.filteredArtists()
		idx := a.Table().SelectedIndex()
		if idx >= 0 && idx < len(artists) {
			uri := artists[idx].URI
			return a, func() tea.Msg { return PlayContextMsg{ContextURI: uri} }
		}
		return a, nil
	}

	// Forward navigation to the table.
	cmd := a.Table().Update(keyMsg)
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

	// Check if data for this range is already cached and fresh.
	if !a.store.StatsStale(nextRange) {
		return a, nil
	}

	// Cache miss — emit a request for the root app to fetch from the API.
	timeRange := nextRange
	return a, func() tea.Msg { return FetchStatsMsg{TimeRange: timeRange} }
}

// View renders the top artists pane content. Pure — reads state, returns string.
func (a *TopArtistsPane) View() string {
	if len(a.store.TopArtists(a.timeRange)) == 0 && !a.Filter().IsActive() {
		return uikit.EmptyState{
			Text:   "No top artists",
			Hint:   "Listen to more music to populate this list",
			Width:  a.width,
			Height: a.height,
			Theme:  a.theme,
		}.Render()
	}

	var parts []string
	if a.Filter().IsActive() {
		parts = append(parts, a.Filter().View(a.width))
	}
	parts = append(parts, a.Table().View())
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
		rows[i] = map[string]string{
			"name": artist.Name,
			"pop":  artistPopStars(artist.Popularity),
			"flw":  formatArtistFollowers(artist.Followers.Total),
		}
	}
	a.Table().SetRows(rows)
}

// filteredArtists returns the top artists for the current time range, filtered by query.
func (a *TopArtistsPane) filteredArtists() []domain.FullArtist {
	all := a.store.TopArtists(a.timeRange)
	if a.Filter().Query() == "" {
		return all
	}
	result := make([]domain.FullArtist, 0, len(all))
	for _, artist := range all {
		if a.Filter().MatchesAny(artist.Name) {
			result = append(result, artist)
		}
	}
	return result
}

// SetTheme updates the theme reference and rebuilds the table with new column colors.
// Called when the user switches themes at runtime.
// NOTE: rebuilding Filter resets the active state and query — existing behaviour.
func (a *TopArtistsPane) SetTheme(th theme.Theme) {
	a.theme = th
	cols := []components.ColumnDef{
		{Key: "name", Header: "Artist", FlexFactor: 11, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "pop", Header: "Pop", FlexFactor: 4, Color: th.ColumnSecondary(), Priority: 3},
		{Key: "flw", Header: "Flw", FlexFactor: 4, Color: th.ColumnTertiary(), Priority: 3},
	}
	newTable, newFilter := components.RebuildTableTheme(th, cols, a.Table().Rows(), a.focused)
	a.SwapTableAndFilter(newTable, newFilter)
	a.resizeTable()
	a.refreshRows()
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (a *TopArtistsPane) resizeTable() {
	tableHeight := a.height
	if a.Filter().IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	a.Table().SetSize(a.width, tableHeight)
}

// artistPopStars converts a Spotify popularity score (0–100) to a 5-star visual grade
// using GlyphPinned (★/*) and GlyphUnpinned (☆/-) — single-char in both glyph modes.
// Thresholds are tuned for Spotify's distribution where most artists score 50+:
//
//	< 30   → ☆☆☆☆☆  (niche / unknown)
//	30–49  → ★☆☆☆☆
//	50–64  → ★★☆☆☆
//	65–79  → ★★★☆☆
//	80–89  → ★★★★☆
//	90–100 → ★★★★★  (superstar)
func artistPopStars(p int) string {
	var filled int
	switch {
	case p >= 90:
		filled = 5
	case p >= 80:
		filled = 4
	case p >= 65:
		filled = 3
	case p >= 50:
		filled = 2
	case p >= 30:
		filled = 1
	default:
		filled = 0
	}
	m := uikit.ActiveMode()
	on := uikit.GlyphFor(uikit.GlyphPinned, m)
	off := uikit.GlyphFor(uikit.GlyphUnpinned, m)
	return strings.Repeat(on, filled) + strings.Repeat(off, 5-filled)
}

// formatArtistFollowers formats a follower count as a compact human-readable string.
// Returns "—" when n is 0 (not returned by this Spotify endpoint for some artists).
//
//	35 000 000 → "35M"   |   1 200 000 → "1.2M"
//	450 000    → "450K"  |   1 500 → "1.5K"   |   999 → "999"
func formatArtistFollowers(n int) string {
	if n == 0 {
		return "—"
	}
	switch {
	case n >= 10_000_000:
		return fmt.Sprintf("%dM", n/1_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 10_000:
		return fmt.Sprintf("%dK", n/1_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
