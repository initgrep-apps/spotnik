// Package panes — RecentlyPlayedPane displays recently played tracks in a dense
// bubble-table with in-pane filtering and a "Played" column showing relative time.
// Selecting a track emits PlayTrackListMsg with URIs from the selected index onward
// so the queue fills with remaining recent tracks.
// Implements layout.Pane (toggle key 6).
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
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// Compile-time check: RecentlyPlayedPane implements layout.Pane.
var _ layout.Pane = &RecentlyPlayedPane{}

// Compile-time check: RecentlyPlayedPane implements layout.FilterQueryPane.
var _ layout.FilterQueryPane = &RecentlyPlayedPane{}

// RecentlyPlayedPane is the Bubble Tea model for the Recently Played pane (toggle key 6).
// It renders a dense bubble-table of recently played tracks with columns for index,
// track name, artist, and relative play time. It supports in-pane filtering by track
// name and artist name.
type RecentlyPlayedPane struct {
	*TableBasedPane
}

// NewRecentlyPlayedPane creates a RecentlyPlayedPane with the given store, theme, and focus state.
func NewRecentlyPlayedPane(store state.StateReader, th theme.Theme, focused bool) *RecentlyPlayedPane {
	// Column widths per DESIGN.md §9: # 5% | Track 55% | Artist 27% | Played 14%
	// Flex factors: 1 : 12 : 6 : 3 ≈ 5% / 55% / 27% / 14%
	columns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "track", Header: "Track", FlexFactor: 12, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "played", Header: "Played", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
	}

	t := components.NewTable(components.TableConfig{
		Columns:    columns,
		Theme:      th,
		ShowHeader: true,
	})

	r := &RecentlyPlayedPane{
		TableBasedPane: NewTableBasedPane(store, th, focused, t, components.NewFilter(th)),
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
// Shows the filter hint always, plus a 'l like' hint when focused and tracks
// are present. When unfocused, only the base filter action is shown — 'l like'
// is a focus-only action (story 270).
func (r *RecentlyPlayedPane) Actions() []layout.Action {
	if !r.IsFocused() {
		return []layout.Action{r.BaseFilterAction()}
	}
	actions := []layout.Action{r.BaseFilterAction()}
	if len(r.filteredItems()) > 0 {
		actions = append(actions, layout.Action{Key: "l", Label: "like"})
	}
	return actions
}

// Init satisfies tea.Model. RecentlyPlayedPane has no startup command.
func (r *RecentlyPlayedPane) Init() tea.Cmd { return nil }

// SetFocused updates the keyboard focus state and propagates it to the table.
func (r *RecentlyPlayedPane) SetFocused(focused bool) {
	r.TableBasedPane.SetFocused(focused)
	r.Table().SetFocused(focused && !r.Filter().IsActive())
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (r *RecentlyPlayedPane) SetSize(width, height int) {
	r.TableBasedPane.SetSize(width, height)
	r.Filter().SetWidth(width)
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

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return r, nil
	}

	// Delegate filter-key routing to the base.
	if consumed, cmd := r.HandleFilterKey(keyMsg, r.refreshRows, r.resizeTable); consumed {
		return r, cmd
	}

	if keyMsg.Type == tea.KeyEnter {
		items := r.filteredItems()
		idx := r.Table().SelectedIndex()
		if idx >= 0 && idx < len(items) {
			uris := make([]string, 0, len(items)-idx)
			for _, item := range items[idx:] {
				uris = append(uris, item.Track.URI)
			}
			return r, func() tea.Msg { return PlayTrackListMsg{URIs: uris} }
		}
		return r, nil
	}

	// 'l' likes/unlikes the selected track. Reads liked status from the store
	// (O(1) lookup) and emits ToggleLikeRequestMsg for the root app to handle
	// the premium gate, optimistic update, and API dispatch.
	if keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "l" {
		items := r.filteredItems()
		idx := r.Table().SelectedIndex()
		if idx >= 0 && idx < len(items) {
			track := items[idx].Track
			return r, func() tea.Msg {
				return ToggleLikeRequestMsg{
					Track:          track,
					CurrentlyLiked: r.store.IsTrackLiked(track.ID),
				}
			}
		}
		return r, nil
	}

	// Forward navigation to the table.
	cmd := r.Table().Update(keyMsg)
	return r, cmd
}

// View renders the recently played pane content. Pure — reads state, returns string.
func (r *RecentlyPlayedPane) View() string {
	var parts []string
	if r.Filter().IsActive() {
		parts = append(parts, r.Filter().View(r.width))
	}
	if len(r.store.RecentlyPlayed()) == 0 && !r.Filter().IsActive() {
		return uikit.EmptyState{
			Text:   "No recently played tracks",
			Hint:   "Listen to something to populate this list",
			Width:  r.width,
			Height: r.height,
			Theme:  r.theme,
		}.Render()
	} else {
		parts = append(parts, r.Table().View())
	}
	return strings.Join(parts, "\n")
}

// RefreshRows re-reads the store and applies updated rows to the table.
// The app calls this after updating the store.
func (r *RecentlyPlayedPane) RefreshRows() { r.refreshRows() }

// refreshRows re-reads the store and applies filtered rows.
func (r *RecentlyPlayedPane) refreshRows() {
	items := r.filteredItems()
	// Track names render as-is — no heart prefix (reverted in story 269).
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
	r.Table().SetRows(rows)
}

// filteredItems returns the recently played items filtered by the current filter query.
// Filter matches on track name OR artist name.
func (r *RecentlyPlayedPane) filteredItems() []domain.PlayHistory {
	all := r.store.RecentlyPlayed()
	if r.Filter().Query() == "" {
		return all
	}
	result := make([]domain.PlayHistory, 0, len(all))
	for _, item := range all {
		artistName := ""
		if len(item.Track.Artists) > 0 {
			artistName = item.Track.Artists[0].Name
		}
		if r.Filter().MatchesAny(item.Track.Name, artistName) {
			result = append(result, item)
		}
	}
	return result
}

// SetTheme updates the theme reference and rebuilds the table with new column colors.
// Called when the user switches themes at runtime.
// NOTE: rebuilding Filter resets the active state and query — existing behaviour.
func (r *RecentlyPlayedPane) SetTheme(th theme.Theme) {
	r.theme = th
	cols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "track", Header: "Track", FlexFactor: 12, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "played", Header: "Played", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
	}
	newTable, newFilter := components.RebuildTableTheme(th, cols, r.Table().Rows(), r.focused)
	r.SwapTableAndFilter(newTable, newFilter)
	r.resizeTable()
	r.refreshRows()
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (r *RecentlyPlayedPane) resizeTable() {
	tableHeight := r.height
	if r.Filter().IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	r.Table().SetSize(r.width, tableHeight)
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
