// Package panes — RecentlyPlayedPane displays recently played tracks in a dense
// bubble-table with in-pane filtering and a "Played" column showing relative time.
// Selecting a track emits PlayTrackMsg. Implements layout.Pane (toggle key 6).
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

// Compile-time check: RecentlyPlayedPane implements layout.Pane.
var _ layout.Pane = &RecentlyPlayedPane{}

// RecentlyPlayedPane is the Bubble Tea model for the Recently Played pane (toggle key 6).
// It renders a dense bubble-table of recently played tracks with columns for index,
// track name, artist, and relative play time. It supports in-pane filtering by track
// name and artist name.
type RecentlyPlayedPane struct {
	store   *state.Store
	theme   theme.Theme
	focused bool

	width  int
	height int

	// table renders the recently played track list.
	table *components.Table
	// filter provides in-pane text filtering by track name and artist.
	filter *components.Filter
}

// NewRecentlyPlayedPane creates a RecentlyPlayedPane with the given store, theme, and focus state.
func NewRecentlyPlayedPane(store *state.Store, th theme.Theme, focused bool) *RecentlyPlayedPane {
	// Column widths per DESIGN.md §9: # 5% | Track 45% | Artist 35% | Played 15%
	// Flex factors: 1 : 9 : 7 : 3 ≈ 5% / 45% / 35% / 15%
	columns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
		{Key: "played", Header: "Played", FlexFactor: 3, Color: th.ColumnTertiary()},
	}

	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	r := &RecentlyPlayedPane{
		store:   store,
		theme:   th,
		focused: focused,
		table:   t,
		filter:  components.NewFilter(th),
	}
	t.SetFocused(focused)
	r.refreshRows()
	return r
}

// ID returns PaneRecentlyPlayed — the identifier for the recently played grid slot.
func (r *RecentlyPlayedPane) ID() layout.PaneID { return layout.PaneRecentlyPlayed }

// Title returns "Recently Played".
func (r *RecentlyPlayedPane) Title() string { return "Recently Played" }

// ToggleKey returns 6 — the number key for btop-style pane toggling.
func (r *RecentlyPlayedPane) ToggleKey() int { return 6 }

// Actions returns the pane-specific shortcut hints displayed in the border.
func (r *RecentlyPlayedPane) Actions() []layout.Action {
	if r.filter.IsActive() {
		return []layout.Action{{Key: "Esc", Label: "close"}}
	}
	return []layout.Action{{Key: "f", Label: "filter"}}
}

// Init satisfies tea.Model. RecentlyPlayedPane has no startup command.
func (r *RecentlyPlayedPane) Init() tea.Cmd { return nil }

// IsFocused returns true when the pane has keyboard focus.
func (r *RecentlyPlayedPane) IsFocused() bool { return r.focused }

// HasActiveFilter returns true when the in-pane filter is capturing keystrokes.
func (r *RecentlyPlayedPane) HasActiveFilter() bool { return r.filter.IsActive() }

// SetFocused updates the keyboard focus state.
func (r *RecentlyPlayedPane) SetFocused(focused bool) {
	r.focused = focused
	r.table.SetFocused(focused && !r.filter.IsActive())
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (r *RecentlyPlayedPane) SetSize(width, height int) {
	r.width = width
	r.height = height
	r.filter.SetWidth(width)
	r.resizeTable()
}

// Update handles key events and data-loaded messages.
func (r *RecentlyPlayedPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle data-loaded messages regardless of focus.
	switch msg.(type) {
	case RecentlyPlayedLoadedMsg:
		r.refreshRows()
		return r, nil
	}

	if !r.focused {
		return r, nil
	}

	// When filter is active, forward all key events to the filter.
	if r.filter.IsActive() {
		cmd := r.filter.Update(msg)
		if !r.filter.IsActive() {
			r.table.SetFocused(true)
			r.resizeTable()
		}
		r.refreshRows()
		return r, cmd
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return r, nil
	}

	switch {
	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "f":
		r.filter.Toggle()
		r.table.SetFocused(false)
		r.resizeTable()
		return r, nil

	case keyMsg.Type == tea.KeyEnter:
		items := r.filteredItems()
		idx := r.table.SelectedIndex()
		if idx >= 0 && idx < len(items) {
			uri := items[idx].Track.URI
			return r, func() tea.Msg {
				return PlayTrackMsg{TrackURI: uri}
			}
		}
		return r, nil
	}

	// Forward navigation to the table.
	cmd := r.table.Update(keyMsg)
	return r, cmd
}

// View renders the recently played pane content. Pure — reads state, returns string.
func (r *RecentlyPlayedPane) View() string {
	var parts []string
	if r.filter.IsActive() {
		parts = append(parts, r.filter.View(r.width))
	}
	if len(r.store.RecentlyPlayed()) == 0 && !r.filter.IsActive() {
		parts = append(parts, "  No recently played tracks")
	} else {
		parts = append(parts, r.table.View())
	}
	return strings.Join(parts, "\n")
}

// RefreshRows re-reads the store and applies updated rows to the table.
// The app calls this after updating the store.
func (r *RecentlyPlayedPane) RefreshRows() { r.refreshRows() }

// refreshRows re-reads the store and applies filtered rows.
func (r *RecentlyPlayedPane) refreshRows() {
	items := r.filteredItems()
	rows := make([]map[string]string, len(items))
	for i, item := range items {
		artistName := ""
		if len(item.Track.Artists) > 0 {
			artistName = item.Track.Artists[0].Name
		}
		playedAt := formatPlayedAtFromHistory(item.PlayedAt)
		rows[i] = map[string]string{
			"index":  fmt.Sprintf("%d", i+1),
			"track":  item.Track.Name,
			"artist": artistName,
			"played": playedAt,
		}
	}
	r.table.SetRows(rows)
}

// filteredItems returns the recently played items filtered by the current filter query.
// Filter matches on track name OR artist name.
func (r *RecentlyPlayedPane) filteredItems() []domain.PlayHistory {
	all := r.store.RecentlyPlayed()
	if r.filter.Query() == "" {
		return all
	}
	result := make([]domain.PlayHistory, 0, len(all))
	for _, item := range all {
		artistName := ""
		if len(item.Track.Artists) > 0 {
			artistName = item.Track.Artists[0].Name
		}
		if r.filter.MatchesAny(item.Track.Name, artistName) {
			result = append(result, item)
		}
	}
	return result
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (r *RecentlyPlayedPane) resizeTable() {
	tableHeight := r.height
	if r.filter.IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	r.table.SetSize(r.width, tableHeight)
}

// formatPlayedAtFromHistory parses an ISO 8601 played_at timestamp and returns
// a relative time string using the shared utility.
func formatPlayedAtFromHistory(playedAt string) string {
	t, err := time.Parse(time.RFC3339, playedAt)
	if err != nil {
		return ""
	}
	return components.FormatRelativeTime(t)
}
