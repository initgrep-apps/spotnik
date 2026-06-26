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

// Compile-time check: PlaylistsPane implements layout.FilterablePane.
var _ layout.FilterablePane = &PlaylistsPane{}

// Compile-time check: PlaylistsPane implements layout.FilterQueryPane.
var _ layout.FilterQueryPane = &PlaylistsPane{}

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
	*TableBasedPane

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

	// removing guards against duplicate 'x' presses during an in-flight removal.
	removing bool
}

// NewPlaylistsPane creates a PlaylistsPane with the given store, theme, and focus state.
func NewPlaylistsPane(store state.StateReader, th theme.Theme, focused bool) *PlaylistsPane {
	// Playlist list columns: access (blank header) 5% | Name 65% | Tracks 25%
	// Using flex factors: 1 : 13 : 5 ≈ 5% / 65% / 25%
	listColumns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "access", Header: "", FlexFactor: 1, Color: th.ColumnSecondary(), Priority: 1},
		{Key: "name", Header: "Name", FlexFactor: 13, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "tracks", Header: "Tracks", FlexFactor: 5, Color: th.ColumnTertiary(), Priority: 3},
	}
	t := components.NewTable(components.TableConfig{
		Columns:    listColumns,
		Theme:      th,
		ShowHeader: true,
	})

	// Track sub-view columns: Track 50% | Artist 30% | Duration 15%
	// Flex factors: 10 : 6 : 3 ≈ 50% / 30% / 15%
	trackColumns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "track", Header: "Track", FlexFactor: 10, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "duration", Header: "Dur", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
	}
	tt := components.NewTable(components.TableConfig{
		Columns:    trackColumns,
		Theme:      th,
		ShowHeader: true,
	})

	f := components.NewFilter(th)
	p := &PlaylistsPane{
		TableBasedPane: NewTableBasedPane(store, th, focused, t, f),
		trackTable:     tt,
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
		hrule := uikit.GlyphFor(uikit.GlyphHRule, uikit.ActiveMode())
		return fmt.Sprintf("Playlists %s%s %s (%d tracks)", hrule, hrule, p.selectedName, p.trackTotal)
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
	return []layout.Action{p.BaseFilterAction()}
}

// Init satisfies tea.Model. PlaylistsPane has no startup command.
func (p *PlaylistsPane) Init() tea.Cmd { return nil }

// SetFocused updates the keyboard focus state, routing focus to the correct table
// based on whether the track sub-view is active.
func (p *PlaylistsPane) SetFocused(focused bool) {
	p.TableBasedPane.SetFocused(focused)
	if p.inTrackView {
		p.trackTable.SetFocused(focused)
		p.Table().SetFocused(false)
	} else {
		p.Table().SetFocused(focused && !p.Filter().IsActive())
		p.trackTable.SetFocused(false)
	}
}

// SetSize updates the render dimensions and propagates them to both tables and filter.
func (p *PlaylistsPane) SetSize(width, height int) {
	p.TableBasedPane.SetSize(width, height)
	p.Filter().SetWidth(width)
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
		// User ID is now available in the store — refresh rows so the access glyph
		// reflects ownership correctly without waiting for a library reload.
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

	case PlaylistRemoveResultMsg:
		p.removing = false
		if m.Err != nil {
			// Error case: toast is emitted by app.go. Pane just clears the sentinel.
			return p, nil
		}
		// Success: filter the removed track from loadedTracks.
		newTracks := make([]domain.Track, 0, len(p.loadedTracks))
		for _, t := range p.loadedTracks {
			if t.URI != m.TrackURI {
				newTracks = append(newTracks, t)
			}
		}
		p.loadedTracks = newTracks
		p.trackTotal--
		if p.trackTotal < 0 {
			p.trackTotal = 0
		}
		p.trackOffset = len(p.loadedTracks)
		p.refreshTrackRows()
		return p, nil
	}

	if !p.focused {
		return p, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	// In track sub-view: handle Enter (play), Esc (back), and table navigation.
	// Filter is not active in track view.
	if p.inTrackView {
		return p.handleTrackViewKey(keyMsg)
	}

	// In playlist list: handle keys (including filter routing).
	return p.handleListViewKey(keyMsg)
}

// handleListViewKey handles key events in the playlist list view.
func (p *PlaylistsPane) handleListViewKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Delegate filter keys (f, Esc) to the shared handler first.
	if consumed, cmd := p.HandleFilterKey(key, p.refreshPlaylistRows, p.resizeTable); consumed {
		return p, cmd
	}

	switch key.Type {
	case tea.KeyEnter:
		playlist := p.filteredPlaylist()
		idx := p.Table().SelectedIndex()
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
			p.Table().SetFocused(false)
			p.trackTable.SetFocused(true)
			p.resizeTable()
			p.refreshTrackRows() // shows 0 rows initially

			// Schedule 150ms debounce for initial fetch
			return p, p.schedulePlaylistDebounce()
		}
		return p, nil
	}

	// Forward navigation to the table.
	cmd := p.Table().Update(key)
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
		p.Table().SetFocused(true)
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
		// Guard against duplicate presses during in-flight removal.
		if p.removing {
			return p, nil
		}
		// Remove selected track from the playlist.
		if idx := p.trackTable.SelectedIndex(); idx >= 0 && idx < len(p.loadedTracks) {
			p.removing = true
			track := p.loadedTracks[idx]
			return p, func() tea.Msg {
				return PlaylistRemoveRequestMsg{
					PlaylistID: p.selectedID,
					TrackURI:   track.URI,
				}
			}
		}
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
	if !p.inTrackView && len(p.store.Playlists()) == 0 && !p.Filter().IsActive() {
		return uikit.EmptyState{
			Text:   "No playlists",
			Hint:   "Create playlists in Spotify or search with /",
			Width:  p.width,
			Height: p.height,
			Theme:  p.theme,
		}.Render()
	}

	var parts []string

	if p.Filter().IsActive() {
		parts = append(parts, p.Filter().View(p.width))
	}

	if p.inTrackView {
		parts = append(parts, p.trackTable.View())
	} else {
		parts = append(parts, p.Table().View())
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
// Each row has a dedicated "access" column that shows an ownership glyph:
//   - User-owned: ◉ (GlyphActive) — full access
//   - Followed (non-owned, non-Spotify): ○ (GlyphAvailable) — read-only
//   - Spotify-curated: ◌ (GlyphLocked) — read-only, Spotify managed
//
// The "name" column contains only the plain playlist name — no prefixes, no ANSI.
func (p *PlaylistsPane) refreshPlaylistRows() {
	playlists := p.filteredPlaylist()
	rows := make([]map[string]string, len(playlists))
	mode := uikit.ActiveMode()
	for i, pl := range playlists {
		var accessGlyph string
		switch {
		case pl.Owner.ID == "spotify":
			accessGlyph = uikit.GlyphFor(uikit.GlyphLocked, mode)
		case !p.isOwnedByCurrentUser(pl):
			accessGlyph = uikit.GlyphFor(uikit.GlyphAvailable, mode)
		default:
			accessGlyph = uikit.GlyphFor(uikit.GlyphActive, mode)
		}
		rows[i] = map[string]string{
			"index":  fmt.Sprintf("%d", i+1),
			"access": accessGlyph,
			"name":   pl.Name,
			"tracks": fmt.Sprintf("%d", pl.TrackCount),
		}
	}
	p.Table().SetRows(rows)
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
	if p.Filter().Query() == "" {
		return all
	}
	result := make([]domain.SimplePlaylist, 0, len(all))
	for _, pl := range all {
		if p.Filter().Matches(pl.Name) {
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
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "access", Header: "", FlexFactor: 1, Color: th.ColumnSecondary(), Priority: 1},
		{Key: "name", Header: "Name", FlexFactor: 13, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "tracks", Header: "Tracks", FlexFactor: 5, Color: th.ColumnTertiary(), Priority: 3},
	}
	newTable, newFilter := components.RebuildTableTheme(th, listCols, p.Table().Rows(), p.focused && !p.inTrackView)
	p.SwapTableAndFilter(newTable, newFilter)

	// Rebuild track table with new column colors.
	trackCols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "track", Header: "Track", FlexFactor: 10, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "artist", Header: "Artist", FlexFactor: 6, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "duration", Header: "Dur", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
	}
	p.trackTable = components.NewTable(components.TableConfig{
		Columns:    trackCols,
		Theme:      th,
		ShowHeader: true,
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
	if p.Filter().IsActive() {
		tableHeight-- // one line for the filter bar
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	if p.inTrackView {
		p.trackTable.SetSize(p.width, tableHeight)
	} else {
		p.Table().SetSize(p.width, tableHeight)
	}
}
