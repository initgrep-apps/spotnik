// Package panes — PlaylistsPane displays the user's playlists in a dense table
// and supports a track sub-view accessible by pressing Enter on a playlist.
// Playlist management operations (create, rename, delete, reorder) emit request
// messages — the pane never calls the API directly.
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

// Compile-time check: PlaylistsPane implements layout.Pane.
var _ layout.Pane = &PlaylistsPane{}

// playlistDebounceIntent is a snapshot of the user's desired playlist at the
// moment of pressing Enter. The debounce tick carries this snapshot; if the
// current intent has changed by the time the tick fires, the tick is discarded.
type playlistDebounceIntent struct {
	playlistID string
}

// playlistDebounceMsg is the internal 150ms tick fired by schedulePlaylistDebounce.
// It is never forwarded to app.go — handled entirely within the pane.
type playlistDebounceMsg struct {
	intent playlistDebounceIntent
}

// PlaylistsPane is the Bubble Tea model for the Playlists pane (toggle key 3).
// It renders a dense bubble-table of the user's playlists, supporting in-pane
// filtering and a track sub-view for the selected playlist.
//
// Architecture: playlist tracks are interactive (user-session) data, not polled
// background data. Tracks are stored in pane fields (loadedTracks), not in the
// global store. This mirrors the search pane pattern.
type PlaylistsPane struct {
	BasePane

	// table renders the playlist list.
	table *components.Table
	// filter provides in-pane text filtering for playlist names.
	filter *components.Filter
	// trackTable renders the track list when inTrackView is true.
	trackTable *components.Table

	// Sub-view identity (what playlist is open)
	inTrackView  bool
	selectedID   string // Spotify playlist ID of the selected playlist
	selectedName string // display name of the selected playlist
	selectedURI  string // Spotify playlist URI — needed for PlayContextMsg

	// Sub-view data (pane-owned, NOT in global store)
	loadedTracks []domain.Track // all tracks fetched so far for this playlist
	trackTotal   int            // total tracks in playlist (from API response)

	// Pagination state (pane-owned)
	trackOffset    int  // count of tracks fetched so far (= len(loadedTracks))
	hasMoreTracks  bool // last API response had next != ""
	tracksFetching bool // a request is in-flight; blocks duplicate prefetch

	// Debounce (protects rapid playlist switching)
	playlistIntent playlistDebounceIntent // current desired playlist
}

// NewPlaylistsPane creates a PlaylistsPane with the given store, theme, and focus state.
func NewPlaylistsPane(store state.StateReader, th theme.Theme, focused bool) *PlaylistsPane {
	// Playlist list columns: # 5% | Name 70% | Tracks 25%
	// Using flex factors: 1 : 14 : 5 ≈ 5% / 70% / 25%
	listColumns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Name", FlexFactor: 14, Color: th.ColumnPrimary()},
		{Key: "tracks", Header: "Tracks", FlexFactor: 5, Color: th.ColumnTertiary()},
	}
	t := components.NewTable(components.TableConfig{
		Columns:      listColumns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	// Track sub-view columns: # 5% | Track 50% | Artist 30% | Duration 15%
	// Flex factors: 1 : 10 : 6 : 3 ≈ 5% / 50% / 30% / 15%
	trackColumns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "track", Header: "Track", FlexFactor: 10, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
	}
	tt := components.NewTable(components.TableConfig{
		Columns:      trackColumns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	p := &PlaylistsPane{
		BasePane:   BasePane{store: store, theme: th, focused: focused},
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

// Title returns the pane title. In track sub-view, it shows the playlist name and
// track count sourced from the API response (p.trackTotal), not from the store.
func (p *PlaylistsPane) Title() string {
	if p.inTrackView {
		return fmt.Sprintf("Playlists ── %s (%d tracks)", p.selectedName, p.trackTotal)
	}
	return "Playlists"
}

// ToggleKey returns 3 — the number key for btop-style pane toggling.
func (p *PlaylistsPane) ToggleKey() int { return 3 }

// Actions returns the pane-specific shortcut hints displayed in the border.
func (p *PlaylistsPane) Actions() []layout.Action {
	if p.inTrackView {
		return []layout.Action{{Key: "Esc", Label: "back"}}
	}
	if p.filter.IsActive() {
		return []layout.Action{{Key: "Esc", Label: "close"}}
	}
	return []layout.Action{
		{Key: "f", Label: "filter"},
	}
}

// Init satisfies tea.Model. PlaylistsPane has no startup command.
func (p *PlaylistsPane) Init() tea.Cmd { return nil }

// HasActiveFilter returns true when the in-pane filter is capturing keystrokes.
func (p *PlaylistsPane) HasActiveFilter() bool { return p.filter.IsActive() }

// SetFocused updates the keyboard focus state, routing focus to the correct table
// based on whether the track sub-view is active.
func (p *PlaylistsPane) SetFocused(focused bool) {
	p.BasePane.SetFocused(focused)
	if p.inTrackView {
		p.trackTable.SetFocused(focused)
		p.table.SetFocused(false)
	} else {
		p.table.SetFocused(focused && !p.filter.IsActive())
		p.trackTable.SetFocused(false)
	}
}

// SetSize updates the render dimensions and propagates them to both tables and filter.
func (p *PlaylistsPane) SetSize(width, height int) {
	p.BasePane.SetSize(width, height)
	p.filter.SetWidth(width)
	p.trackTable.SetSize(width, height)
	p.resizeTable()
}

// Update handles key events and data-loaded messages.
// Data messages (playlistDebounceMsg, PlaylistTracksLoadedMsg) are processed regardless
// of focus because they carry async results from commands.
func (p *PlaylistsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {

	case playlistDebounceMsg:
		return p.handlePlaylistDebounce(m)

	case LibraryLoadedMsg:
		p.refreshPlaylistRows()
		return p, nil

	case UserProfileReadyMsg:
		// User ID is now available in the store — refresh rows so the ~ prefix
		// appears on followed playlists without waiting for a library reload.
		p.refreshPlaylistRows()
		return p, nil

	case PlaylistTracksLoadedMsg:
		// Guard: only process if this matches the currently selected playlist.
		// Discards responses that arrive after the user switched playlists.
		if m.PlaylistID != p.selectedID {
			return p, nil
		}
		p.tracksFetching = false
		if m.Err != nil {
			// Error is toasted by app.go. Pane just clears loading state.
			return p, nil
		}
		if m.Offset == 0 {
			// Initial page: replace
			p.loadedTracks = m.Tracks
		} else {
			// Subsequent page: append
			p.loadedTracks = append(p.loadedTracks, m.Tracks...)
		}
		p.trackOffset = len(p.loadedTracks)
		p.trackTotal = m.Total
		p.hasMoreTracks = m.HasNext
		p.refreshTrackRows()
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

	// In track sub-view: handle Enter (play), Esc (back), and table navigation.
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

			// Spotify API restricts GET /playlists/{id}/items to playlists owned or
			// collaborated on by the user. Block drill-down for followed playlists.
			if !p.isOwnedByCurrentUser(pl) {
				return p, func() tea.Msg { return PlaylistAccessDeniedMsg{} }
			}

			// Update identity
			p.selectedID = pl.ID
			p.selectedName = pl.Name
			p.selectedURI = pl.URI // needed for PlayContextMsg

			// Reset sub-view data (new playlist, old data invalid)
			p.loadedTracks = nil
			p.trackOffset = 0
			p.trackTotal = 0
			p.hasMoreTracks = false
			p.tracksFetching = false // cleared here; set true in debounce handler

			// Update debounce intent
			p.playlistIntent = playlistDebounceIntent{playlistID: pl.ID}

			// Switch to track sub-view immediately (shows empty table while loading)
			p.inTrackView = true
			p.table.SetFocused(false)
			p.trackTable.SetFocused(true)
			p.resizeTable()
			p.refreshTrackRows() // shows 0 rows initially

			// Schedule 150ms debounce for initial fetch
			return p, p.schedulePlaylistDebounce()
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
		// Return to playlist list and emit closed message for app.go to cancel
		// any in-flight fetch.
		p.inTrackView = false
		p.trackTable.SetFocused(false)
		p.table.SetFocused(true)
		p.resizeTable()
		return p, func() tea.Msg { return PlaylistTrackViewClosedMsg{} }

	case key.Type == tea.KeyEnter:
		// Play selected track with the playlist as context.
		if idx := p.trackTable.SelectedIndex(); idx >= 0 && idx < len(p.loadedTracks) {
			track := p.loadedTracks[idx]
			playlistURI := p.selectedURI
			return p, func() tea.Msg {
				return PlayContextMsg{
					ContextURI: playlistURI,
					OffsetURI:  track.URI,
				}
			}
		}
		return p, nil

	case key.Type == tea.KeyRunes && string(key.Runes) == "x":
		// NOTE: 'x' (remove track) is out of scope for story 106 — remains non-functional.
		return p, nil
	}

	// Forward j/k and other navigation to the track table.
	cmd := p.trackTable.Update(key)
	// After navigation, check if we should prefetch the next page.
	prefetchCmd := p.checkPrefetch()
	return p, tea.Batch(cmd, prefetchCmd)
}

// schedulePlaylistDebounce snapshots the current playlist intent and returns
// a 150ms tick. Stale ticks are discarded in handlePlaylistDebounce.
// Used only for the initial fetch (offset 0) triggered by Enter in list view.
func (p *PlaylistsPane) schedulePlaylistDebounce() tea.Cmd {
	snapshot := p.playlistIntent
	return tea.Tick(150*time.Millisecond, func(_ time.Time) tea.Msg {
		return playlistDebounceMsg{intent: snapshot}
	})
}

// handlePlaylistDebounce fires when a 150ms debounce tick arrives.
// It discards stale ticks (user switched to a different playlist) and
// blocks duplicate requests (tracksFetching is already true).
func (p *PlaylistsPane) handlePlaylistDebounce(m playlistDebounceMsg) (tea.Model, tea.Cmd) {
	// Stale: user switched to a different playlist before tick fired.
	if m.intent.playlistID != p.playlistIntent.playlistID {
		return p, nil
	}
	// Already fetching: a request is in-flight for this playlist.
	// This happens when user Enter → Esc → Enter on same playlist quickly.
	if p.tracksFetching {
		return p, nil
	}
	// Fire the initial fetch.
	p.tracksFetching = true
	id := p.playlistIntent.playlistID
	return p, func() tea.Msg {
		return FetchPlaylistTracksRequestMsg{
			PlaylistID: id,
			Offset:     0,
		}
	}
}

// checkPrefetch fires a next-page request when the cursor is within 10 rows
// of the last loaded track and more pages are available.
// Pagination requests bypass the debounce — they fire immediately.
func (p *PlaylistsPane) checkPrefetch() tea.Cmd {
	if !p.hasMoreTracks || p.tracksFetching {
		return nil
	}
	if len(p.loadedTracks) == 0 {
		return nil
	}
	cursor := p.trackTable.SelectedIndex()
	if cursor < len(p.loadedTracks)-10 {
		return nil
	}
	p.tracksFetching = true
	id := p.selectedID
	offset := p.trackOffset
	return func() tea.Msg {
		return FetchPlaylistTracksRequestMsg{PlaylistID: id, Offset: offset}
	}
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

// RefreshRows re-reads the store (for playlist list) or pane state (for track list).
// The app calls this after updating the store for playlist list changes.
func (p *PlaylistsPane) RefreshRows() {
	if p.inTrackView {
		p.refreshTrackRows()
	} else {
		p.refreshPlaylistRows()
	}
}

// refreshPlaylistRows re-reads the store and applies filtered playlist rows.
// Spotify-curated playlists (Owner.ID == "spotify") render via LockedRow to
// signal they are read-only and cannot be drill-downed into. Other non-owned
// (followed) playlists are prefixed with "~ " to signal restricted access.
func (p *PlaylistsPane) refreshPlaylistRows() {
	playlists := p.filteredPlaylist()
	rows := make([]map[string]string, len(playlists))
	for i, pl := range playlists {
		var name string
		if pl.Owner.ID == "spotify" {
			// Plain glyph+label so the table's column renderer can apply its own
			// colour. Embedding ANSI from LockedRow.Render conflicts with bubble-table's
			// per-column foreground pass (applyRows sets Foreground over the cell string).
			name = uikit.GlyphFor(uikit.GlyphLocked, uikit.ActiveMode()) + " " + pl.Name
		} else if !p.isOwnedByCurrentUser(pl) {
			name = "~ " + pl.Name
		} else {
			name = pl.Name
		}
		rows[i] = map[string]string{
			"index":  fmt.Sprintf("%d", i+1),
			"name":   name,
			"tracks": fmt.Sprintf("%d", pl.TrackCount),
		}
	}
	p.table.SetRows(rows)
}

// isOwnedByCurrentUser returns true if the playlist is owned by the current user.
// Returns true when the user ID is not yet known (benefit of the doubt — let the API decide).
func (p *PlaylistsPane) isOwnedByCurrentUser(pl domain.SimplePlaylist) bool {
	userID := p.store.UserID()
	if userID == "" {
		return true // user ID not yet loaded; don't block prematurely
	}
	return pl.Owner.ID == userID
}

// refreshTrackRows rebuilds track rows from p.loadedTracks (pane-owned data).
// It no longer reads from the global store.
func (p *PlaylistsPane) refreshTrackRows() {
	rows := make([]map[string]string, len(p.loadedTracks))
	for i, track := range p.loadedTracks {
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

// SetTheme updates the theme reference and rebuilds both tables with new column colors.
// Called when the user switches themes at runtime.
func (p *PlaylistsPane) SetTheme(th theme.Theme) {
	p.theme = th
	listCols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Name", FlexFactor: 14, Color: th.ColumnPrimary()},
		{Key: "tracks", Header: "Tracks", FlexFactor: 5, Color: th.ColumnTertiary()},
	}
	p.table, p.filter = components.RebuildTableTheme(th, listCols, p.table.Rows(), p.focused && !p.inTrackView)

	// Rebuild track table with new column colors.
	trackCols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "track", Header: "Track", FlexFactor: 10, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
	}
	p.trackTable = components.NewTable(components.TableConfig{
		Columns:      trackCols,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})
	p.trackTable.SetSize(p.width, p.height)
	if p.inTrackView {
		p.trackTable.SetFocused(p.focused)
		p.refreshTrackRows()
	}

	p.resizeTable()
	p.refreshPlaylistRows()
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
