// Package panes — TopTracksPane displays the user's top tracks in a dense
// bubble-table with in-pane filtering and time range cycling via the g key.
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
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// Compile-time check: TopTracksPane implements layout.Pane.
var _ layout.Pane = &TopTracksPane{}

// Compile-time check: TopTracksPane implements layout.FilterQueryPane.
var _ layout.FilterQueryPane = &TopTracksPane{}

// topTracksTimeRanges is the cycle order for the g key.
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
// time range cycling via the g key (short_term → medium_term → long_term → short_term).
type TopTracksPane struct {
	*TableBasedPane

	// timeRange is the currently active Spotify time range for top tracks.
	timeRange string
}

// NewTopTracksPane creates a TopTracksPane with the given store, theme, and focus state.
// Default time range is short_term (4 weeks).
func NewTopTracksPane(store state.StateReader, th theme.Theme, focused bool) *TopTracksPane {
	// Column widths per DESIGN.md §9: # 5% | Track 55% | Artist 27% | Dur 14%
	// Flex factors: 1 : 12 : 6 : 3 ≈ 5% / 55% / 27% / 14%
	columns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "track", Header: "Track", FlexFactor: 12, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "dur", Header: "Dur", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
	}

	t := components.NewTable(components.TableConfig{
		Columns:    columns,
		Theme:      th,
		ShowHeader: true,
	})

	p := &TopTracksPane{
		TableBasedPane: NewTableBasedPane(store, th, focused, t, components.NewFilter(th)),
		timeRange:      "short_term",
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
// Shows filter (f) and time range cycle (g) with the current range as label.
// The 'l like' hint is shown only when focused and tracks are present.
// When unfocused, only filter and range are shown — 'l like' is a focus-only
// action (story 270).
func (p *TopTracksPane) Actions() []layout.Action {
	rangeLabel := topTracksRangeLabels[p.timeRange]
	actions := []layout.Action{
		p.BaseFilterAction(),
		{Key: "g", Label: rangeLabel},
	}
	if !p.IsFocused() {
		return actions
	}
	// Show the 'l' like hint when tracks are present (story 269).
	if len(p.filteredTracks()) > 0 {
		actions = append(actions, layout.Action{Key: "l", Label: "like"})
	}
	return actions
}

// Init satisfies tea.Model. TopTracksPane has no startup command.
func (p *TopTracksPane) Init() tea.Cmd { return nil }

// SetFocused updates the keyboard focus state and propagates it to the table.
func (p *TopTracksPane) SetFocused(focused bool) {
	p.TableBasedPane.SetFocused(focused)
	p.Table().SetFocused(focused && !p.Filter().IsActive())
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (p *TopTracksPane) SetSize(width, height int) {
	p.TableBasedPane.SetSize(width, height)
	p.Filter().SetWidth(width)
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

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	// Delegate filter-key routing to the base.
	if consumed, cmd := p.HandleFilterKey(keyMsg, p.refreshRows, p.resizeTable); consumed {
		return p, cmd
	}

	switch {
	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "g":
		return p.cycleTimeRange()

	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "l":
		// Like/unlike the selected track. Reads liked status from the store
		// (O(1) lookup) and emits ToggleLikeRequestMsg for the root app to
		// handle the premium gate, optimistic update, and API dispatch.
		tracks := p.filteredTracks()
		idx := p.Table().SelectedIndex()
		if idx >= 0 && idx < len(tracks) {
			track := tracks[idx]
			return p, func() tea.Msg {
				return ToggleLikeRequestMsg{
					Track:          track,
					CurrentlyLiked: p.store.IsTrackLiked(track.ID),
				}
			}
		}
		return p, nil

	case keyMsg.Type == tea.KeyEnter:
		tracks := p.filteredTracks()
		idx := p.Table().SelectedIndex()
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
	cmd := p.Table().Update(keyMsg)
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
	if len(p.store.TopTracks(p.timeRange)) == 0 && !p.Filter().IsActive() {
		return uikit.EmptyState{
			Text:   "No top tracks",
			Hint:   "Listen to more music to populate this list",
			Width:  p.width,
			Height: p.height,
			Theme:  p.theme,
		}.Render()
	}

	var parts []string
	if p.Filter().IsActive() {
		parts = append(parts, p.Filter().View(p.width))
	}
	parts = append(parts, p.Table().View())
	return strings.Join(parts, "\n")
}

// RefreshRows re-reads the store and applies updated rows to the table.
// The app calls this after updating the store.
func (p *TopTracksPane) RefreshRows() { p.refreshRows() }

// refreshRows re-reads the store and applies filtered track rows.
func (p *TopTracksPane) refreshRows() {
	tracks := p.filteredTracks()
	// Track names render as-is — no heart prefix (reverted in story 269).
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
			"dur":    trackDuration(track.DurationMs),
		}
	}
	p.Table().SetRows(rows)
}

// filteredTracks returns the top tracks for the current time range, filtered by query.
// Filter matches on track name OR artist name.
func (p *TopTracksPane) filteredTracks() []domain.Track {
	all := p.store.TopTracks(p.timeRange)
	if p.Filter().Query() == "" {
		return all
	}
	result := make([]domain.Track, 0, len(all))
	for _, track := range all {
		artistName := ""
		if len(track.Artists) > 0 {
			artistName = track.Artists[0].Name
		}
		if p.Filter().MatchesAny(track.Name, artistName) {
			result = append(result, track)
		}
	}
	return result
}

// SetTheme updates the theme reference and rebuilds the table with new column colors.
// Called when the user switches themes at runtime.
// NOTE: rebuilding Filter resets the active state and query — existing behaviour.
func (p *TopTracksPane) SetTheme(th theme.Theme) {
	p.theme = th
	cols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "track", Header: "Track", FlexFactor: 12, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "dur", Header: "Dur", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
	}
	newTable, newFilter := components.RebuildTableTheme(th, cols, p.Table().Rows(), p.focused)
	p.SwapTableAndFilter(newTable, newFilter)
	p.resizeTable()
	p.refreshRows()
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (p *TopTracksPane) resizeTable() {
	tableHeight := p.height
	if p.Filter().IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	p.Table().SetSize(p.width, tableHeight)
}

// trackDuration converts milliseconds to "m:ss" (e.g. 252000 → "4:12").
func trackDuration(ms int) string {
	secs := ms / 1000
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}
