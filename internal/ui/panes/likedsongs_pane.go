// Package panes — LikedSongsPane displays the user's liked tracks in a dense table
// with in-pane filtering and like/unlike toggle. Selecting a track emits PlayContextMsg
// with the liked songs collection context so the queue fills with subsequent liked songs.
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

// Compile-time check: LikedSongsPane implements layout.Pane.
var _ layout.Pane = &LikedSongsPane{}

// LikedSongsPane is the Bubble Tea model for the Liked Songs pane (toggle key 5).
// It renders a dense bubble-table of the user's liked tracks with columns for index,
// track name, artist, and duration. It supports in-pane filtering by track name and
// artist, and the 'i' key toggles like/unlike for the selected track.
type LikedSongsPane struct {
	BasePane

	// table renders the liked track list.
	table *components.Table
	// filter provides in-pane text filtering by track name and artist.
	filter *components.Filter
}

// NewLikedSongsPane creates a LikedSongsPane with the given store, theme, and focus state.
func NewLikedSongsPane(store state.StateReader, th theme.Theme, focused bool) *LikedSongsPane {
	// Liked songs columns: # 5% | Track 45% | Artist 35% | Duration 15%
	// Flex factors: 1 : 9 : 7 : 3 ≈ 5% / 45% / 35% / 15%
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

	l := &LikedSongsPane{
		BasePane: BasePane{store: store, theme: th, focused: focused},
		table:    t,
		filter:   components.NewFilter(th),
	}
	t.SetFocused(focused)
	l.refreshRows()
	return l
}

// ID returns PaneLikedSongs — the identifier for the liked songs grid slot.
func (l *LikedSongsPane) ID() layout.PaneID { return layout.PaneLikedSongs }

// Title returns "Liked Songs".
func (l *LikedSongsPane) Title() string { return "Liked Songs" }

// ToggleKey returns 5 — the number key for btop-style pane toggling.
func (l *LikedSongsPane) ToggleKey() int { return 5 }

// Actions returns the pane-specific shortcut hints displayed in the border.
func (l *LikedSongsPane) Actions() []layout.Action {
	if l.filter.IsActive() {
		return []layout.Action{{Key: "Esc", Label: "close"}}
	}
	return []layout.Action{
		{Key: "f", Label: "filter"},
		{Key: "i", Label: "like"},
	}
}

// Init satisfies tea.Model. LikedSongsPane has no startup command.
func (l *LikedSongsPane) Init() tea.Cmd { return nil }

// HasActiveFilter returns true when the in-pane filter is capturing keystrokes.
func (l *LikedSongsPane) HasActiveFilter() bool { return l.filter.IsActive() }

// SetFocused updates the keyboard focus state and propagates it to the table.
func (l *LikedSongsPane) SetFocused(focused bool) {
	l.BasePane.SetFocused(focused)
	l.table.SetFocused(focused && !l.filter.IsActive())
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (l *LikedSongsPane) SetSize(width, height int) {
	l.BasePane.SetSize(width, height)
	l.filter.SetWidth(width)
	l.resizeTable()
}

// Update handles key events and data-loaded messages.
func (l *LikedSongsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle data-loaded messages regardless of focus.
	switch msg.(type) {
	case LikedTracksLoadedMsg:
		l.refreshRows()
		return l, nil
	}

	if !l.focused {
		return l, nil
	}

	// When filter is active, forward all key events to the filter.
	if l.filter.IsActive() {
		cmd := l.filter.Update(msg)
		if !l.filter.IsActive() {
			l.table.SetFocused(true)
			l.resizeTable()
		}
		l.refreshRows()
		return l, cmd
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return l, nil
	}

	switch {
	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "f":
		l.filter.Toggle()
		l.table.SetFocused(false)
		l.resizeTable()
		return l, nil

	case keyMsg.Type == tea.KeyEnter:
		tracks := l.filteredTracks()
		idx := l.table.SelectedIndex()
		if idx >= 0 && idx < len(tracks) {
			uri := tracks[idx].Track.URI
			return l, func() tea.Msg {
				return PlayContextMsg{
					ContextURI: "spotify:collection:tracks",
					OffsetURI:  uri,
				}
			}
		}
		return l, nil

	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "i":
		// Toggle like/unlike. Tracks shown here are already liked, so 'i' means unlike.
		tracks := l.filteredTracks()
		idx := l.table.SelectedIndex()
		if idx >= 0 && idx < len(tracks) {
			trackID := tracks[idx].Track.ID
			return l, func() tea.Msg {
				return LikeTrackRequestMsg{TrackID: trackID, Unlike: true}
			}
		}
		return l, nil
	}

	// Forward navigation to the table.
	cmd := l.table.Update(keyMsg)
	return l, cmd
}

// View renders the liked songs pane content. Pure — reads state, returns string.
func (l *LikedSongsPane) View() string {
	var parts []string

	if l.filter.IsActive() {
		parts = append(parts, l.filter.View(l.width))
	}

	parts = append(parts, l.table.View())
	return strings.Join(parts, "\n")
}

// RefreshRows re-reads the store and applies updated rows to the table.
// The app calls this after updating the store.
func (l *LikedSongsPane) RefreshRows() { l.refreshRows() }

// refreshRows re-reads the store and applies filtered track rows.
func (l *LikedSongsPane) refreshRows() {
	tracks := l.filteredTracks()
	rows := make([]map[string]string, len(tracks))
	for i, st := range tracks {
		artistName := ""
		if len(st.Track.Artists) > 0 {
			artistName = st.Track.Artists[0].Name
		}
		rows[i] = map[string]string{
			"index":    fmt.Sprintf("%d", i+1),
			"track":    st.Track.Name,
			"artist":   artistName,
			"duration": formatDurationMs(st.Track.DurationMs),
		}
	}
	l.table.SetRows(rows)
}

// filteredTracks returns the liked tracks filtered by the current filter query.
// Filter matches on track name OR artist name.
func (l *LikedSongsPane) filteredTracks() []domain.SavedTrack {
	all := l.store.LikedTracks()
	if l.filter.Query() == "" {
		return all
	}
	result := make([]domain.SavedTrack, 0, len(all))
	for _, st := range all {
		artistName := ""
		if len(st.Track.Artists) > 0 {
			artistName = st.Track.Artists[0].Name
		}
		if l.filter.MatchesAny(st.Track.Name, artistName) {
			result = append(result, st)
		}
	}
	return result
}

// SetTheme updates the theme reference and rebuilds the table with new column colors.
// Called when the user switches themes at runtime.
func (l *LikedSongsPane) SetTheme(th theme.Theme) {
	l.theme = th
	cols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
	}
	l.table, l.filter = components.RebuildTableTheme(th, cols, l.table.Rows(), l.focused && !l.filter.IsActive())
	l.resizeTable()
	l.refreshRows()
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (l *LikedSongsPane) resizeTable() {
	tableHeight := l.height
	if l.filter.IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	l.table.SetSize(l.width, tableHeight)
}
