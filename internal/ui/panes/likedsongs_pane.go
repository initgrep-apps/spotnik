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

// Compile-time check: LikedSongsPane implements layout.FilterQueryPane.
var _ layout.FilterQueryPane = &LikedSongsPane{}

// LikedSongsPane is the Bubble Tea model for the Liked Songs pane (toggle key 5).
// It renders a dense bubble-table of the user's liked tracks with columns for index,
// track name, artist, and duration. It supports in-pane filtering by track name and
// artist.
type LikedSongsPane struct {
	*TableBasedPane
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
		Columns:    columns,
		Theme:      th,
		ShowHeader: true,
	})

	l := &LikedSongsPane{
		TableBasedPane: NewTableBasedPane(store, th, focused, t, components.NewFilter(th)),
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

// Init satisfies tea.Model. LikedSongsPane has no startup command.
func (l *LikedSongsPane) Init() tea.Cmd { return nil }

// SetFocused updates the keyboard focus state and propagates it to the table.
func (l *LikedSongsPane) SetFocused(focused bool) {
	l.TableBasedPane.SetFocused(focused)
	l.Table().SetFocused(focused && !l.Filter().IsActive())
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (l *LikedSongsPane) SetSize(width, height int) {
	l.TableBasedPane.SetSize(width, height)
	l.Filter().SetWidth(width)
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

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return l, nil
	}

	// Delegate filter-key routing to the base.
	if consumed, cmd := l.HandleFilterKey(keyMsg, l.refreshRows, l.resizeTable); consumed {
		return l, cmd
	}

	if keyMsg.Type == tea.KeyEnter {
		tracks := l.filteredTracks()
		idx := l.Table().SelectedIndex()
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
	}

	// Forward navigation to the table.
	cmd := l.Table().Update(keyMsg)
	return l, cmd
}

// View renders the liked songs pane content. Pure — reads state, returns string.
func (l *LikedSongsPane) View() string {
	var parts []string

	if l.Filter().IsActive() {
		parts = append(parts, l.Filter().View(l.width))
	}

	parts = append(parts, l.Table().View())
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
	l.Table().SetRows(rows)
}

// filteredTracks returns the liked tracks filtered by the current filter query.
// Filter matches on track name OR artist name.
func (l *LikedSongsPane) filteredTracks() []domain.SavedTrack {
	all := l.store.LikedTracks()
	if l.Filter().Query() == "" {
		return all
	}
	result := make([]domain.SavedTrack, 0, len(all))
	for _, st := range all {
		artistName := ""
		if len(st.Track.Artists) > 0 {
			artistName = st.Track.Artists[0].Name
		}
		if l.Filter().MatchesAny(st.Track.Name, artistName) {
			result = append(result, st)
		}
	}
	return result
}

// SetTheme updates the theme reference and rebuilds the table with new column colors.
// Called when the user switches themes at runtime.
// NOTE: rebuilding Filter resets the active state and query — existing behaviour.
func (l *LikedSongsPane) SetTheme(th theme.Theme) {
	l.theme = th
	cols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
	}
	newTable, newFilter := components.RebuildTableTheme(th, cols, l.Table().Rows(), l.focused)
	l.SwapTableAndFilter(newTable, newFilter)
	l.resizeTable()
	l.refreshRows()
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (l *LikedSongsPane) resizeTable() {
	tableHeight := l.height
	if l.Filter().IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	l.Table().SetSize(l.width, tableHeight)
}
