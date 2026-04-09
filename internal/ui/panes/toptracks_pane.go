// Package panes — TopTracksPane displays the user's top tracks in a dense
// bubble-table with in-pane filtering and time range cycling via the t key.
// Selecting a track emits PlayTrackListMsg with URIs from the selected index
// onward so the queue fills with remaining top tracks.
// Implements layout.Pane (toggle key 7).
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

// Compile-time check: TopTracksPane implements layout.Pane.
var _ layout.Pane = &TopTracksPane{}

// topTracksTimeRanges is the cycle order for the t key.
var topTracksTimeRanges = []string{"short_term", "medium_term", "long_term"}

// topTracksRangeLabels maps API values to human-readable display labels.
var topTracksRangeLabels = map[string]string{
	"short_term":  "4wk",
	"medium_term": "6mo",
	"long_term":   "all",
}

// TopTracksPane is the Bubble Tea model for the Top Tracks pane (toggle key 7).
// It renders a dense bubble-table of the user's top tracks with columns for index,
// track name, artist, and popularity. It supports in-pane filtering and per-pane
// time range cycling via the t key (short_term → medium_term → long_term → short_term).
type TopTracksPane struct {
	BasePane

	// timeRange is the currently active Spotify time range for top tracks.
	timeRange string

	// table renders the top tracks list.
	table *components.Table
	// filter provides in-pane text filtering by track name and artist.
	filter *components.Filter
}

// NewTopTracksPane creates a TopTracksPane with the given store, theme, and focus state.
// Default time range is short_term (4 weeks).
func NewTopTracksPane(store state.StateReader, th theme.Theme, focused bool) *TopTracksPane {
	// Column widths per DESIGN.md §9: # 5% | Track 45% | Artist 35% | Pop 15%
	// Flex factors: 1 : 9 : 7 : 3 ≈ 5% / 45% / 35% / 15%
	columns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
		{Key: "pop", Header: "Pop", FlexFactor: 3, Color: th.ColumnTertiary()},
	}

	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	p := &TopTracksPane{
		BasePane:  BasePane{store: store, theme: th, focused: focused},
		timeRange: "short_term",
		table:     t,
		filter:    components.NewFilter(th),
	}
	t.SetFocused(focused)
	p.refreshRows()
	return p
}

// ID returns PaneTopTracks — the identifier for the top tracks grid slot.
func (p *TopTracksPane) ID() layout.PaneID { return layout.PaneTopTracks }

// Title returns "Top Tracks".
func (p *TopTracksPane) Title() string { return "Top Tracks" }

// ToggleKey returns 7 — the number key for btop-style pane toggling.
func (p *TopTracksPane) ToggleKey() int { return 7 }

// TimeRange returns the currently active time range string (exported for testing).
func (p *TopTracksPane) TimeRange() string { return p.timeRange }

// Actions returns the pane-specific shortcut hints displayed in the border.
// When the filter is active, only shows the Esc/close hint.
// Otherwise shows filter (f) and time range cycle (t) with the current range as label.
func (p *TopTracksPane) Actions() []layout.Action {
	if p.filter.IsActive() {
		return []layout.Action{{Key: "Esc", Label: "close"}}
	}
	rangeLabel := topTracksRangeLabels[p.timeRange]
	return []layout.Action{
		{Key: "f", Label: "filter"},
		{Key: "t", Label: rangeLabel},
	}
}

// Init satisfies tea.Model. TopTracksPane has no startup command.
func (p *TopTracksPane) Init() tea.Cmd { return nil }

// HasActiveFilter returns true when the in-pane filter is capturing keystrokes.
func (p *TopTracksPane) HasActiveFilter() bool { return p.filter.IsActive() }

// SetFocused updates the keyboard focus state and propagates it to the table.
func (p *TopTracksPane) SetFocused(focused bool) {
	p.BasePane.SetFocused(focused)
	p.table.SetFocused(focused && !p.filter.IsActive())
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (p *TopTracksPane) SetSize(width, height int) {
	p.BasePane.SetSize(width, height)
	p.filter.SetWidth(width)
	p.resizeTable()
}

// Update handles key events and data-loaded messages.
func (p *TopTracksPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle data-loaded messages regardless of focus.
	switch m := msg.(type) {
	case StatsLoadedMsg:
		if m.TimeRange == p.timeRange {
			p.refreshRows()
		}
		return p, nil
	}

	if !p.focused {
		return p, nil
	}

	// When filter is active, forward all key events to the filter.
	if p.filter.IsActive() {
		cmd := p.filter.Update(msg)
		if !p.filter.IsActive() {
			p.table.SetFocused(true)
			p.resizeTable()
		}
		p.refreshRows()
		return p, cmd
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	switch {
	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "f":
		p.filter.Toggle()
		p.table.SetFocused(false)
		p.resizeTable()
		return p, nil

	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "t":
		return p.cycleTimeRange()

	case keyMsg.Type == tea.KeyEnter:
		tracks := p.filteredTracks()
		idx := p.table.SelectedIndex()
		if idx >= 0 && idx < len(tracks) {
			uris := make([]string, 0, len(tracks)-idx)
			for _, tr := range tracks[idx:] {
				uris = append(uris, tr.URI)
			}
			return p, func() tea.Msg { return PlayTrackListMsg{URIs: uris} }
		}
		return p, nil
	}

	// Forward navigation to the table.
	cmd := p.table.Update(keyMsg)
	return p, cmd
}

// cycleTimeRange advances to the next time range, checking the store cache first.
// On a cache hit, renders immediately with no fetch. On a miss, emits FetchStatsMsg.
func (p *TopTracksPane) cycleTimeRange() (tea.Model, tea.Cmd) {
	currentIdx := 0
	for i, r := range topTracksTimeRanges {
		if r == p.timeRange {
			currentIdx = i
			break
		}
	}
	nextRange := topTracksTimeRanges[(currentIdx+1)%len(topTracksTimeRanges)]
	p.timeRange = nextRange
	p.refreshRows()

	// Check if data for this range is already cached and fresh.
	// Use StatsStale instead of nil check — an empty slice from a prior fetch
	// is a valid cache hit, but StatsStale correctly uses fetchedAt timestamps.
	if !p.store.StatsStale(nextRange) {
		return p, nil
	}

	// Cache miss — emit a request for the root app to fetch from the API.
	timeRange := nextRange
	return p, func() tea.Msg { return FetchStatsMsg{TimeRange: timeRange} }
}

// View renders the top tracks pane content. Pure — reads state, returns string.
func (p *TopTracksPane) View() string {
	var parts []string
	if p.filter.IsActive() {
		parts = append(parts, p.filter.View(p.width))
	}
	parts = append(parts, p.table.View())
	return strings.Join(parts, "\n")
}

// RefreshRows re-reads the store and applies updated rows to the table.
// The app calls this after updating the store.
func (p *TopTracksPane) RefreshRows() { p.refreshRows() }

// refreshRows re-reads the store and applies filtered track rows.
func (p *TopTracksPane) refreshRows() {
	tracks := p.filteredTracks()
	rows := make([]map[string]string, len(tracks))
	for i, track := range tracks {
		artistName := ""
		if len(track.Artists) > 0 {
			artistName = track.Artists[0].Name
		}
		rows[i] = map[string]string{
			"index":  fmt.Sprintf("%d", i+1),
			"track":  track.Name,
			"artist": artistName,
			"pop":    "—",
		}
	}
	p.table.SetRows(rows)
}

// filteredTracks returns the top tracks for the current time range, filtered by query.
// Filter matches on track name OR artist name.
func (p *TopTracksPane) filteredTracks() []domain.Track {
	all := p.store.TopTracks(p.timeRange)
	if p.filter.Query() == "" {
		return all
	}
	result := make([]domain.Track, 0, len(all))
	for _, track := range all {
		artistName := ""
		if len(track.Artists) > 0 {
			artistName = track.Artists[0].Name
		}
		if p.filter.MatchesAny(track.Name, artistName) {
			result = append(result, track)
		}
	}
	return result
}

// SetTheme updates the theme reference and rebuilds the table with new column colors.
// Called when the user switches themes at runtime.
func (p *TopTracksPane) SetTheme(th theme.Theme) {
	p.theme = th
	cols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
		{Key: "pop", Header: "Pop", FlexFactor: 3, Color: th.ColumnTertiary()},
	}
	p.table, p.filter = components.RebuildTableTheme(th, cols, p.table.Rows(), p.focused && !p.filter.IsActive())
	p.resizeTable()
	p.refreshRows()
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (p *TopTracksPane) resizeTable() {
	tableHeight := p.height
	if p.filter.IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	p.table.SetSize(p.width, tableHeight)
}
