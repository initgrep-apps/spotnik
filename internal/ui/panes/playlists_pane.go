// Package panes — PlaylistsPane displays the user's playlists in a dense table
// and merges PlaylistManager functionality: create, rename, delete, reorder tracks,
// and a track sub-view accessible by pressing Enter on a playlist.
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

// Compile-time check: PlaylistsPane implements layout.Pane.
var _ layout.Pane = &PlaylistsPane{}

// PlaylistsPane is the Bubble Tea model for the Playlists pane (toggle key 3).
// It renders a dense bubble-table of the user's playlists, supporting in-pane
// filtering and a track sub-view for the selected playlist. Playlist management
// operations (create, rename, delete, reorder) emit request messages — the pane
// never calls the API directly.
type PlaylistsPane struct {
	store   *state.Store
	theme   theme.Theme
	focused bool

	width  int
	height int

	// table renders the playlist list.
	table *components.Table
	// filter provides in-pane text filtering for playlist names.
	filter *components.Filter

	// Track sub-view state
	inTrackView  bool
	selectedID   string // Spotify playlist ID of the selected playlist
	selectedName string // display name of the selected playlist
	// trackTable renders the track list when inTrackView is true.
	trackTable *components.Table
}

// NewPlaylistsPane creates a PlaylistsPane with the given store, theme, and focus state.
func NewPlaylistsPane(store *state.Store, th theme.Theme, focused bool) *PlaylistsPane {
	// Playlist list columns: # 5% | Name 70% | Tracks 25%
	// Using flex factors: 1 : 14 : 5 ≈ 5% / 70% / 25%
	listColumns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.TextMuted()},
		{Key: "name", Header: "Name", FlexFactor: 14, Color: th.TextPrimary()},
		{Key: "tracks", Header: "Tracks", FlexFactor: 5, Color: th.TextMuted()},
	}
	t := components.NewTable(components.TableConfig{
		Columns:      listColumns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	// Track sub-view columns: # 5% | Track 45% | Artist 35% | Duration 15%
	// Flex factors: 1 : 9 : 7 : 3 ≈ 5% / 45% / 35% / 15%
	trackColumns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.TextMuted()},
		{Key: "track", Header: "Track", FlexFactor: 9, Color: th.TextPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.TextSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.TextMuted()},
	}
	tt := components.NewTable(components.TableConfig{
		Columns:      trackColumns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	p := &PlaylistsPane{
		store:      store,
		theme:      th,
		focused:    focused,
		table:      t,
		filter:     components.NewFilter(th),
		trackTable: tt,
	}
	t.SetFocused(focused)
	p.refreshPlaylistRows()
	return p
}

// ID returns PanePlaylists — the identifier for the playlists grid slot.
func (p *PlaylistsPane) ID() layout.PaneID { return layout.PanePlaylists }

// Title returns the pane title. In track sub-view, it shows the playlist name.
func (p *PlaylistsPane) Title() string {
	if p.inTrackView {
		tracks := p.store.PlaylistTracks(p.selectedID)
		trackCount := len(tracks)
		return fmt.Sprintf("Playlists ── %s (%d tracks)", p.selectedName, trackCount)
	}
	return "Playlists"
}

// ToggleKey returns 3 — the number key for btop-style pane toggling.
func (p *PlaylistsPane) ToggleKey() int { return 3 }

// Actions returns the pane-specific shortcut hints displayed in the border.
func (p *PlaylistsPane) Actions() []layout.Action {
	if p.filter.IsActive() {
		return []layout.Action{{Key: "Esc", Label: "close"}}
	}
	if p.inTrackView {
		return []layout.Action{
			{Key: "Esc", Label: "back"},
			{Key: "Shift+↕", Label: "reorder"},
		}
	}
	return []layout.Action{
		{Key: "f", Label: "filter"},
		{Key: "n", Label: "new"},
		{Key: "r", Label: "rename"},
		{Key: "x", Label: "delete"},
	}
}

// Init satisfies tea.Model. PlaylistsPane has no startup command.
func (p *PlaylistsPane) Init() tea.Cmd { return nil }

// IsFocused returns true when the pane has keyboard focus.
func (p *PlaylistsPane) IsFocused() bool { return p.focused }

// SetFocused updates the keyboard focus state.
func (p *PlaylistsPane) SetFocused(focused bool) {
	p.focused = focused
	if p.inTrackView {
		p.trackTable.SetFocused(focused && !p.filter.IsActive())
	} else {
		p.table.SetFocused(focused && !p.filter.IsActive())
	}
}

// SetSize updates the render dimensions and propagates them to the table and filter.
func (p *PlaylistsPane) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.filter.SetWidth(width)
	p.resizeTable()
}

// Update handles key events and data-loaded messages.
func (p *PlaylistsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle data-loaded messages regardless of focus.
	switch m := msg.(type) {
	case LibraryLoadedMsg:
		p.refreshPlaylistRows()
		return p, nil
	case PlaylistCreatedMsg:
		if m.Err == nil {
			p.refreshPlaylistRows()
		}
		return p, nil
	case PlaylistRenamedMsg:
		if m.Err == nil {
			p.refreshPlaylistRows()
		}
		return p, nil
	case PlaylistTracksLoadedMsg:
		// Only process if this matches the currently selected playlist.
		if m.PlaylistID == p.selectedID {
			p.refreshTrackRows()
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
			// Filter just closed — refocus the active table.
			if p.inTrackView {
				p.trackTable.SetFocused(true)
			} else {
				p.table.SetFocused(true)
			}
			p.resizeTable()
		}
		// Refresh the active table (playlist list or track list).
		p.RefreshRows()
		return p, cmd
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	// In track sub-view: handle Esc, Shift+Up/Down, x, and table navigation.
	if p.inTrackView {
		return p.handleTrackViewKey(keyMsg)
	}

	// In playlist list: handle keys.
	return p.handleListViewKey(keyMsg)
}

// handleListViewKey handles key events in the playlist list view.
func (p *PlaylistsPane) handleListViewKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Type == tea.KeyRunes && string(key.Runes) == "f":
		p.filter.Toggle()
		p.table.SetFocused(false)
		p.resizeTable()
		return p, nil

	case key.Type == tea.KeyEnter:
		playlist := p.filteredPlaylist()
		idx := p.table.SelectedIndex()
		if idx >= 0 && idx < len(playlist) {
			pl := playlist[idx]
			p.selectedID = pl.ID
			p.selectedName = pl.Name
			// Switch to track sub-view before emitting request.
			p.inTrackView = true
			p.table.SetFocused(false)
			p.trackTable.SetFocused(true)
			p.resizeTable()
			p.refreshTrackRows()
			return p, func() tea.Msg {
				return FetchPlaylistTracksRequestMsg{PlaylistID: pl.ID}
			}
		}
		return p, nil

	case key.Type == tea.KeyRunes && string(key.Runes) == "n":
		return p, func() tea.Msg {
			return PlaylistCreateRequestMsg{Name: "New Playlist"}
		}

	case key.Type == tea.KeyRunes && string(key.Runes) == "r":
		playlist := p.filteredPlaylist()
		idx := p.table.SelectedIndex()
		if idx >= 0 && idx < len(playlist) {
			pl := playlist[idx]
			return p, func() tea.Msg {
				return PlaylistRenameRequestMsg{PlaylistID: pl.ID, NewName: pl.Name}
			}
		}
		return p, nil

	case key.Type == tea.KeyRunes && string(key.Runes) == "x":
		playlist := p.filteredPlaylist()
		idx := p.table.SelectedIndex()
		if idx >= 0 && idx < len(playlist) {
			pl := playlist[idx]
			return p, func() tea.Msg {
				// In list view, 'x' maps to remove the playlist itself.
				// Per spec, use PlaylistRemoveRequestMsg — the playlist URI is its context.
				return PlaylistRemoveRequestMsg{PlaylistID: pl.ID, TrackURI: pl.URI}
			}
		}
		return p, nil
	}

	// Forward navigation to the table.
	cmd := p.table.Update(key)
	return p, cmd
}

// handleTrackViewKey handles key events in the track sub-view.
func (p *PlaylistsPane) handleTrackViewKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Type == tea.KeyEsc:
		// Return to playlist list.
		p.inTrackView = false
		p.trackTable.SetFocused(false)
		p.table.SetFocused(true)
		p.resizeTable()
		return p, nil

	case key.Type == tea.KeyRunes && string(key.Runes) == "x":
		tracks := p.store.PlaylistTracks(p.selectedID)
		idx := p.trackTable.SelectedIndex()
		if idx >= 0 && idx < len(tracks) {
			track := tracks[idx]
			playlistID := p.selectedID
			return p, func() tea.Msg {
				return PlaylistRemoveRequestMsg{PlaylistID: playlistID, TrackURI: track.URI}
			}
		}
		return p, nil

	case key.Type == tea.KeyShiftUp:
		tracks := p.store.PlaylistTracks(p.selectedID)
		idx := p.trackTable.SelectedIndex()
		if idx > 0 && idx < len(tracks) {
			playlistID := p.selectedID
			from := idx
			to := idx - 1
			return p, func() tea.Msg {
				return PlaylistReorderRequestMsg{
					PlaylistID:   playlistID,
					RangeStart:   from,
					InsertBefore: to,
					RangeLength:  1,
				}
			}
		}
		return p, nil

	case key.Type == tea.KeyShiftDown:
		tracks := p.store.PlaylistTracks(p.selectedID)
		idx := p.trackTable.SelectedIndex()
		if idx >= 0 && idx < len(tracks)-1 {
			playlistID := p.selectedID
			from := idx
			to := idx + 2
			return p, func() tea.Msg {
				return PlaylistReorderRequestMsg{
					PlaylistID:   playlistID,
					RangeStart:   from,
					InsertBefore: to,
					RangeLength:  1,
				}
			}
		}
		return p, nil
	}

	// Forward navigation to the track table.
	cmd := p.trackTable.Update(key)
	return p, cmd
}

// View renders the pane content. Pure — reads state, returns string.
func (p *PlaylistsPane) View() string {
	var parts []string

	if p.filter.IsActive() {
		parts = append(parts, p.filter.View(p.width))
	}

	if p.inTrackView {
		parts = append(parts, p.trackTable.View())
	} else {
		parts = append(parts, p.table.View())
	}

	return strings.Join(parts, "\n")
}

// RefreshRows re-reads the store and applies updated rows to the active table.
// The app calls this after updating the store.
func (p *PlaylistsPane) RefreshRows() {
	if p.inTrackView {
		p.refreshTrackRows()
	} else {
		p.refreshPlaylistRows()
	}
}

// refreshPlaylistRows re-reads the store and applies filtered playlist rows.
func (p *PlaylistsPane) refreshPlaylistRows() {
	playlists := p.filteredPlaylist()
	rows := make([]map[string]string, len(playlists))
	for i, pl := range playlists {
		rows[i] = map[string]string{
			"index":  fmt.Sprintf("%d", i+1),
			"name":   pl.Name,
			"tracks": fmt.Sprintf("%d", pl.TrackCount),
		}
	}
	p.table.SetRows(rows)
}

// refreshTrackRows re-reads the store and applies track rows for the selected playlist.
func (p *PlaylistsPane) refreshTrackRows() {
	tracks := p.store.PlaylistTracks(p.selectedID)
	rows := make([]map[string]string, len(tracks))
	for i, track := range tracks {
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
	p.trackTable.SetRows(rows)
}

// filteredPlaylist returns the playlists filtered by the current filter query.
func (p *PlaylistsPane) filteredPlaylist() []domain.SimplePlaylist {
	all := p.store.Playlists()
	if p.filter.Query() == "" {
		return all
	}
	result := make([]domain.SimplePlaylist, 0, len(all))
	for _, pl := range all {
		if p.filter.Matches(pl.Name) {
			result = append(result, pl)
		}
	}
	return result
}

// resizeTable updates the table size, accounting for the filter bar when active.
func (p *PlaylistsPane) resizeTable() {
	tableHeight := p.height
	if p.filter.IsActive() {
		tableHeight-- // one line for the filter bar
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	if p.inTrackView {
		p.trackTable.SetSize(p.width, tableHeight)
	} else {
		p.table.SetSize(p.width, tableHeight)
	}
}
