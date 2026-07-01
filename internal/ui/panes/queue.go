// Package panes — QueuePane displays the current play queue in the right pane.
// It renders a dense table with columns Type, Title, Artist, Duration
// and supports in-pane filtering via the 'f' key. The queue may contain both
// tracks and episodes from Spotify's mixed-content queue response.
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
	*TableBasedPane
}

// NewQueuePane creates a new QueuePane with the given store, theme, and focus state.
func NewQueuePane(store state.StateReader, th theme.Theme, focused bool) *QueuePane {
	columns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "type", Header: "", FlexFactor: 1, Color: th.ColumnSecondary(), Priority: 1},
		{Key: "title", Header: "Title", FlexFactor: 7, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "artist", Header: "Artist", FlexFactor: 4, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "duration", Header: "Dur", FlexFactor: 2, Color: th.ColumnTertiary(), Priority: 3},
	}

	t := components.NewTable(components.TableConfig{
		Columns:    columns,
		Theme:      th,
		ShowHeader: true,
	})

	q := &QueuePane{
		TableBasedPane: NewTableBasedPane(store, th, focused, t, components.NewFilter(th)),
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
// Shows the filter hint always, plus a 'l like' hint when focused and the
// selected row is a track (episodes are not likable via /me/tracks).
// When unfocused, only the base filter action is shown — 'l like' is a
// focus-only action (story 270).
func (q *QueuePane) Actions() []layout.Action {
	if !q.IsFocused() {
		return []layout.Action{q.BaseFilterAction()}
	}
	actions := []layout.Action{q.BaseFilterAction()}
	queue := q.filteredQueue()
	idx := q.Table().SelectedIndex()
	if idx >= 0 && idx < len(queue) && queue[idx].Type == domain.QueueItemTypeTrack {
		actions = append(actions, layout.Action{Key: "l", Label: "like"})
	}
	return actions
}

// Init satisfies tea.Model. The queue pane has no startup command of its own —
// the root app's tick loop drives queue refreshes.
func (q *QueuePane) Init() tea.Cmd {
	return nil
}

// SetFocused sets the keyboard focus state and propagates it to the table.
func (q *QueuePane) SetFocused(focused bool) {
	q.TableBasedPane.SetFocused(focused)
	q.Table().SetFocused(focused && !q.Filter().IsActive())
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (q *QueuePane) SetSize(width, height int) {
	q.TableBasedPane.SetSize(width, height)
	q.Filter().SetWidth(width)
	q.resizeTable()
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

	// Delegate filter-key routing to the base.
	if consumed, cmd := q.HandleFilterKey(keyMsg, q.refreshRows, q.resizeTable); consumed {
		return q, cmd
	}

	// Enter: play the selected queue item (track or episode).
	if keyMsg.Type == tea.KeyEnter {
		queue := q.filteredQueue()
		idx := q.Table().SelectedIndex()
		if idx >= 0 && idx < len(queue) {
			item := queue[idx]
			switch item.Type {
			case domain.QueueItemTypeTrack:
				return q, func() tea.Msg {
					return PlayTrackMsg{TrackURI: item.Track.URI}
				}
			case domain.QueueItemTypeEpisode:
				return q, func() tea.Msg {
					return PlayEpisodeMsg{EpisodeURI: item.Episode.URI, PlaylistURI: ""}
				}
			}
		}
		return q, nil
	}

	// 'l' likes/unlikes the selected track. Only applies to track items —
	// episodes are not likable via the /me/tracks endpoint. Emits
	// ToggleLikeRequestMsg for the root app to handle the premium gate,
	// optimistic update, and API dispatch.
	if keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "l" {
		queue := q.filteredQueue()
		idx := q.Table().SelectedIndex()
		if idx >= 0 && idx < len(queue) {
			item := queue[idx]
			if item.Type == domain.QueueItemTypeTrack {
				track := *item.Track
				return q, func() tea.Msg {
					return ToggleLikeRequestMsg{
						Track:          track,
						CurrentlyLiked: q.store.IsTrackLiked(track.ID),
					}
				}
			}
		}
		return q, nil
	}

	// Forward j/k and other navigation to the table.
	cmd := q.Table().Update(msg)
	return q, cmd
}

// View renders the queue pane content. It is pure — reads store and returns a string.
func (q *QueuePane) View() string {
	// Show EmptyState only when the queue is truly empty (no filter active).
	// When the filter is active but yields no results, the filter bar is shown so the
	// user can see and edit their query — the empty table is the correct feedback.
	if len(q.store.Queue()) == 0 && !q.Filter().IsActive() {
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
	if q.Filter().IsActive() {
		parts = append(parts, q.Filter().View(q.width))
	}

	parts = append(parts, q.Table().View())
	return strings.Join(parts, "\n")
}

// RefreshRows re-reads the store and applies filtered rows to the table.
// The app calls this after updating the store (e.g. QueueLoadedMsg handler).
func (q *QueuePane) RefreshRows() { q.refreshRows() }

// refreshRows re-reads the store and applies filtered rows to the table.
func (q *QueuePane) refreshRows() {
	queue := q.filteredQueue()
	// Track names render as-is — no heart prefix (reverted in story 269).
	rows := make([]map[string]string, len(queue))
	for i, item := range queue {
		row := map[string]string{}

		switch item.Type {
		case domain.QueueItemTypeEpisode:
			row["index"] = fmt.Sprintf("%d", i+1)
			row["type"] = uikit.GlyphFor(uikit.GlyphEpisode, uikit.GlyphUnicode)
			row["title"] = item.Episode.Name
			showName := ""
			if item.Episode.Show != nil {
				showName = item.Episode.Show.Name
			}
			row["artist"] = showName
			row["duration"] = formatDurationMsH(item.Episode.DurationMs)
		default:
			row["index"] = fmt.Sprintf("%d", i+1)
			row["type"] = uikit.GlyphFor(uikit.GlyphMusicNote, uikit.GlyphUnicode)
			row["title"] = item.Track.Name
			artistName := ""
			if len(item.Track.Artists) > 0 {
				artistName = item.Track.Artists[0].Name
			}
			row["artist"] = artistName
			row["duration"] = formatDurationMsH(item.Track.DurationMs)
		}

		rows[i] = row
	}
	q.Table().SetRows(rows)
}

// filteredQueue returns the queue items filtered by the current filter query.
func (q *QueuePane) filteredQueue() []domain.QueueItem {
	all := q.store.Queue()
	if q.Filter().Query() == "" {
		return all
	}
	result := make([]domain.QueueItem, 0, len(all))
	for _, item := range all {
		var name, secondary string
		switch item.Type {
		case domain.QueueItemTypeEpisode:
			name = item.Episode.Name
			if item.Episode.Show != nil {
				secondary = item.Episode.Show.Name
			}
		default:
			name = item.Track.Name
			if len(item.Track.Artists) > 0 {
				secondary = item.Track.Artists[0].Name
			}
		}
		if q.Filter().MatchesAny(name, secondary) {
			result = append(result, item)
		}
	}
	return result
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (q *QueuePane) resizeTable() {
	tableHeight := q.height
	if q.Filter().IsActive() {
		tableHeight -= 1 // one line for the filter bar
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	q.Table().SetSize(q.width, tableHeight)
}

// SetTheme updates the theme reference and rebuilds the table with new column colors.
// Called when the user switches themes at runtime.
// NOTE: rebuilding Filter resets the active state and query — this is the existing
// behaviour; preserving filter state across theme switches is out of scope.
func (q *QueuePane) SetTheme(th theme.Theme) {
	q.theme = th
	cols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "type", Header: "", FlexFactor: 1, Color: th.ColumnSecondary(), Priority: 1},
		{Key: "title", Header: "Title", FlexFactor: 7, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "artist", Header: "Artist", FlexFactor: 4, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "duration", Header: "Dur", FlexFactor: 2, Color: th.ColumnTertiary(), Priority: 3},
	}
	newTable, newFilter := components.RebuildTableTheme(th, cols, q.Table().Rows(), q.focused)
	q.SwapTableAndFilter(newTable, newFilter)
	q.resizeTable()
	q.refreshRows()
}

// Cursor returns the currently selected row index (0-based) in the table.
// NOTE: kept for backward compatibility with tests.
func (q *QueuePane) Cursor() int {
	return q.Table().SelectedIndex()
}
