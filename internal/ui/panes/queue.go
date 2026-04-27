// Package panes — QueuePane displays the current play queue in the right pane.
// It renders a dense table with columns #, Track, Artist, Duration and supports
// in-pane filtering via the 'f' key.
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

// Compile-time check: QueuePane implements layout.Pane.
var _ layout.Pane = &QueuePane{}

// Compile-time check: QueuePane implements layout.FilterQueryPane.
var _ layout.FilterQueryPane = &QueuePane{}

// QueuePane is the right-pane Bubble Tea model that shows the upcoming play queue
// as a dense bubble-table with optional in-pane filtering. It reads all data from
// the central Store — it never imports api/ directly.
type QueuePane struct {
	BasePane

	// table is the bubble-table wrapper for dense queue rendering.
	table *components.Table

	// filter provides in-pane filtering by track name and artist.
	filter *components.Filter
}

// NewQueuePane creates a new QueuePane with the given store, theme, and focus state.
func NewQueuePane(store state.StateReader, th theme.Theme, focused bool) *QueuePane {
	columns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
	}

	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	q := &QueuePane{
		BasePane: BasePane{store: store, theme: th, focused: focused},
		table:    t,
		filter:   components.NewFilter(th),
	}
	// Sync table focus state with initial focused parameter.
	t.SetFocused(focused)
	q.refreshRows()
	return q
}

// ID returns the PaneQueue identifier for this pane slot.
func (q *QueuePane) ID() layout.PaneID { return layout.PaneQueue }

// Title returns the display title shown in the pane border.
func (q *QueuePane) Title() string { return "Queue" }

// ToggleKey returns 2 — the number key for btop-style pane toggling.
func (q *QueuePane) ToggleKey() int { return 2 }

// Actions returns the pane-specific shortcut hints displayed in the border.
// Returns close action when filter is active; otherwise filter action.
func (q *QueuePane) Actions() []layout.Action {
	if q.filter.IsActive() {
		return []layout.Action{{Key: "Esc", Label: "close"}}
	}
	return []layout.Action{
		{Key: "f", Label: "filter"},
	}
}

// Init satisfies tea.Model. The queue pane has no startup command of its own —
// the root app's tick loop drives queue refreshes.
func (q *QueuePane) Init() tea.Cmd {
	return nil
}

// HasActiveFilter returns true when the in-pane filter is capturing keystrokes.
func (q *QueuePane) HasActiveFilter() bool {
	return q.filter.IsActive()
}

// ActiveFilterQuery returns the committed filter query for border display.
// Satisfies layout.FilterQueryPane.
func (q *QueuePane) ActiveFilterQuery() string {
	return q.filter.Query()
}

// SetFocused sets the keyboard focus state and propagates it to the table.
func (q *QueuePane) SetFocused(focused bool) {
	q.BasePane.SetFocused(focused)
	q.table.SetFocused(focused && !q.filter.IsActive())
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (q *QueuePane) SetSize(width, height int) {
	q.BasePane.SetSize(width, height)
	q.filter.SetWidth(width)
	q.resizeTable()
}

// SetPlayingIndex marks which row shows the ▶ indicator. Pass -1 to clear.
func (q *QueuePane) SetPlayingIndex(index int) {
	q.table.SetPlayingIndex(index)
}

// Update handles key events when the pane is focused.
func (q *QueuePane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !q.focused {
		return q, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return q, nil
	}

	// When filter is active, forward all key events to the filter.
	if q.filter.IsActive() {
		cmd := q.filter.Update(msg)
		// If the filter just closed (Esc/Enter consumed it), re-focus the table and resize.
		if !q.filter.IsActive() {
			q.table.SetFocused(true)
			q.resizeTable()
		}
		// Refresh rows regardless — query may have changed or filter may have closed.
		q.refreshRows()
		return q, cmd
	}

	// f key: activate filter.
	if keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "f" {
		q.filter.Toggle()
		q.table.SetFocused(false)
		q.resizeTable()
		return q, nil
	}

	// Enter: play the selected track.
	if keyMsg.Type == tea.KeyEnter {
		queue := q.filteredQueue()
		idx := q.table.SelectedIndex()
		if idx >= 0 && idx < len(queue) {
			uri := queue[idx].URI
			return q, func() tea.Msg {
				return PlayTrackMsg{TrackURI: uri}
			}
		}
		return q, nil
	}

	// Esc: first clear a committed filter query; second press resets scroll.
	if keyMsg.Type == tea.KeyEscape {
		if q.filter.Query() != "" {
			q.filter.ClearQuery()
			q.refreshRows()
			return q, nil
		}
		q.table.GotoTop()
		return q, nil
	}

	// Forward j/k and other navigation to the table.
	cmd := q.table.Update(msg)
	return q, cmd
}

// View renders the queue pane content. It is pure — reads store and returns a string.
func (q *QueuePane) View() string {
	// Show EmptyState only when the queue is truly empty (no filter active).
	// When the filter is active but yields no results, the filter bar is shown so the
	// user can see and edit their query — the empty table is the correct feedback.
	if len(q.store.Queue()) == 0 && !q.filter.IsActive() {
		return uikit.EmptyState{
			Text:   "Empty queue",
			Hint:   "Press / to search for tracks to add",
			Width:  q.width,
			Height: q.height,
			Theme:  q.theme,
		}.Render()
	}

	var parts []string

	// If filter is active, prepend the filter bar and shrink the table.
	if q.filter.IsActive() {
		parts = append(parts, q.filter.View(q.width))
	}

	parts = append(parts, q.table.View())
	return strings.Join(parts, "\n")
}

// RefreshRows re-reads the store and applies filtered rows to the table.
// The app calls this after updating the store (e.g. QueueLoadedMsg handler).
func (q *QueuePane) RefreshRows() { q.refreshRows() }

// refreshRows re-reads the store and applies filtered rows to the table.
func (q *QueuePane) refreshRows() {
	queue := q.filteredQueue()
	rows := make([]map[string]string, len(queue))
	for i, track := range queue {
		artistName := ""
		if len(track.Artists) > 0 {
			artistName = track.Artists[0].Name
		}
		rows[i] = map[string]string{
			"index":    fmt.Sprintf("%d", i+1),
			"track":    track.Name,
			"artist":   artistName,
			"duration": formatDurationMs(track.DurationMs),
		}
	}
	q.table.SetRows(rows)
}

// filteredQueue returns the queue tracks filtered by the current filter query.
func (q *QueuePane) filteredQueue() []domain.Track {
	all := q.store.Queue()
	if q.filter.Query() == "" {
		return all
	}
	result := make([]domain.Track, 0, len(all))
	for _, track := range all {
		artistName := ""
		if len(track.Artists) > 0 {
			artistName = track.Artists[0].Name
		}
		if q.filter.MatchesAny(track.Name, artistName) {
			result = append(result, track)
		}
	}
	return result
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (q *QueuePane) resizeTable() {
	tableHeight := q.height
	if q.filter.IsActive() {
		tableHeight -= 1 // one line for the filter bar
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	q.table.SetSize(q.width, tableHeight)
}

// SetTheme updates the theme reference and rebuilds the table with new column colors.
// Called when the user switches themes at runtime.
func (q *QueuePane) SetTheme(th theme.Theme) {
	q.theme = th
	cols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
	}
	q.table, q.filter = components.RebuildTableTheme(th, cols, q.table.Rows(), q.focused)
	q.resizeTable()
	q.refreshRows()
}

// Cursor returns the currently selected row index (0-based) in the table.
// NOTE: kept for backward compatibility with tests.
func (q *QueuePane) Cursor() int {
	return q.table.SelectedIndex()
}
