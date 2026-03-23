package panes

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// SectionType identifies a library section.
type SectionType int

const (
	// SectionPlaylists is the user's saved playlists section.
	SectionPlaylists SectionType = iota
	// SectionAlbums is the user's saved albums section.
	SectionAlbums
	// SectionLikedSongs is the user's liked songs section.
	SectionLikedSongs
	// SectionRecentlyPlayed is the recently played tracks section.
	SectionRecentlyPlayed
)

// LibraryItem represents a single row in the library tree.
// Exactly one of the pointer fields is non-nil.
type LibraryItem struct {
	// Playlist is set when this item is a playlist.
	Playlist *api.SimplePlaylist

	// Album is set when this item is a saved album.
	Album *api.SavedAlbum

	// LikedTrack is set when this item is a liked track.
	LikedTrack *api.SavedTrack

	// PlayHistory is set when this item is a recently played track.
	PlayHistory *api.PlayHistory
}

// trackURI returns the Spotify URI for the item if it is a track, or empty string.
func (li *LibraryItem) trackURI() string {
	if li.LikedTrack != nil {
		return li.LikedTrack.Track.URI
	}
	if li.PlayHistory != nil {
		return li.PlayHistory.Track.URI
	}
	return ""
}

// trackID returns the Spotify ID for the item if it is a track, or empty string.
func (li *LibraryItem) trackID() string {
	if li.LikedTrack != nil {
		return li.LikedTrack.Track.ID
	}
	if li.PlayHistory != nil {
		return li.PlayHistory.Track.ID
	}
	return ""
}

// displayName returns the display name for the item.
func (li *LibraryItem) displayName() string {
	if li.Playlist != nil {
		return li.Playlist.Name
	}
	if li.Album != nil {
		return li.Album.Album.Name
	}
	if li.LikedTrack != nil {
		return li.LikedTrack.Track.Name
	}
	if li.PlayHistory != nil {
		return li.PlayHistory.Track.Name
	}
	return ""
}

// playableContextURI returns the Spotify context URI for playlist/album items.
// Returns empty string for track items (use trackURI() instead).
func (li *LibraryItem) playableContextURI() string {
	if li.Playlist != nil {
		return li.Playlist.URI
	}
	if li.Album != nil {
		return li.Album.Album.URI
	}
	return ""
}

// Section holds the state for one collapsible library section.
type Section struct {
	// Type identifies which section this is.
	Type SectionType

	// Name is the display label shown in the header row.
	Name string

	// Items are the rows shown when the section is expanded.
	Items []LibraryItem

	// Expanded indicates whether the section is open.
	Expanded bool

	// Loading indicates that a fetch is in progress.
	Loading bool

	// Total is the server-side total count (for pagination display).
	Total int
}

// LibraryTree manages the collapsible section list and cursor position.
// It is pane-local state and is never stored in the central Store.
type LibraryTree struct {
	sections  []Section
	cursorPos int // absolute visible row index (0 = first row)
}

// NewLibraryTree constructs a LibraryTree with the default section order.
func NewLibraryTree() *LibraryTree {
	return &LibraryTree{
		sections: []Section{
			{Type: SectionPlaylists, Name: "Playlists"},
			{Type: SectionAlbums, Name: "Albums"},
			{Type: SectionLikedSongs, Name: "Liked Songs"},
			{Type: SectionRecentlyPlayed, Name: "Recently Played"},
		},
	}
}

// Sections returns the current sections slice (exported for tests).
func (t *LibraryTree) Sections() []Section {
	return t.sections
}

// CursorPos returns the current absolute cursor row position.
func (t *LibraryTree) CursorPos() int {
	return t.cursorPos
}

// visibleRows returns the total number of visible rows (section headers + expanded items).
func (t *LibraryTree) visibleRows() int {
	count := 0
	for _, s := range t.sections {
		count++ // section header row
		if s.Expanded {
			count += len(s.Items)
		}
	}
	return count
}

// MoveDown moves the cursor one row down, clamped at the last row.
func (t *LibraryTree) MoveDown() {
	max := t.visibleRows() - 1
	if t.cursorPos < max {
		t.cursorPos++
	}
}

// MoveUp moves the cursor one row up, clamped at 0.
func (t *LibraryTree) MoveUp() {
	if t.cursorPos > 0 {
		t.cursorPos--
	}
}

// ToggleSection expands or collapses the section whose header is at the current cursor.
// If the cursor is on an item (not a header), this is a no-op.
func (t *LibraryTree) ToggleSection() {
	row := 0
	for i := range t.sections {
		if row == t.cursorPos {
			t.sections[i].Expanded = !t.sections[i].Expanded
			return
		}
		row++ // header
		if t.sections[i].Expanded {
			row += len(t.sections[i].Items)
		}
	}
}

// SetSectionExpanded sets the expanded state of the section with the given type.
func (t *LibraryTree) SetSectionExpanded(sectionType SectionType, expanded bool) {
	for i := range t.sections {
		if t.sections[i].Type == sectionType {
			t.sections[i].Expanded = expanded
			return
		}
	}
}

// SetSectionLoading sets the loading state of the section with the given type.
func (t *LibraryTree) SetSectionLoading(sectionType SectionType, loading bool) {
	for i := range t.sections {
		if t.sections[i].Type == sectionType {
			t.sections[i].Loading = loading
			return
		}
	}
}

// SelectedItem returns the LibraryItem at the current cursor position,
// or nil if the cursor is on a section header.
func (t *LibraryTree) SelectedItem() *LibraryItem {
	row := 0
	for i := range t.sections {
		if row == t.cursorPos {
			// cursor is on section header
			return nil
		}
		row++ // count the header
		if t.sections[i].Expanded {
			for j := range t.sections[i].Items {
				if row == t.cursorPos {
					return &t.sections[i].Items[j]
				}
				row++
			}
		}
	}
	return nil
}

// CurrentSectionType returns the SectionType of the section the cursor is currently in
// (whether on the header or on an item within it).
func (t *LibraryTree) CurrentSectionType() SectionType {
	row := 0
	for _, s := range t.sections {
		if row == t.cursorPos {
			return s.Type
		}
		row++ // header
		if s.Expanded {
			for range s.Items {
				if row == t.cursorPos {
					return s.Type
				}
				row++
			}
		}
	}
	return SectionPlaylists
}

// IsOnSectionHeader returns true if the cursor is on a section header row.
func (t *LibraryTree) IsOnSectionHeader() bool {
	return t.SelectedItem() == nil
}

// ItemIndexInSection returns the 0-based index of the currently selected item
// within its section, or -1 if on a header.
func (t *LibraryTree) ItemIndexInSection() int {
	row := 0
	for _, s := range t.sections {
		if row == t.cursorPos {
			return -1 // on header
		}
		row++
		if s.Expanded {
			for j := range s.Items {
				if row == t.cursorPos {
					return j
				}
				row++
			}
		}
	}
	return -1
}

// UpdateFromStore refreshes section items from the store.
func (t *LibraryTree) UpdateFromStore(s *state.Store) {
	for i := range t.sections {
		switch t.sections[i].Type {
		case SectionPlaylists:
			playlists := s.Playlists()
			items := make([]LibraryItem, len(playlists))
			for j := range playlists {
				pl := playlists[j]
				items[j] = LibraryItem{Playlist: &pl}
			}
			t.sections[i].Items = items
			t.sections[i].Total = s.PlaylistsTotal()

		case SectionAlbums:
			albums := s.SavedAlbums()
			items := make([]LibraryItem, len(albums))
			for j := range albums {
				al := albums[j]
				items[j] = LibraryItem{Album: &al}
			}
			t.sections[i].Items = items

		case SectionLikedSongs:
			liked := s.LikedTracks()
			items := make([]LibraryItem, len(liked))
			for j := range liked {
				lt := liked[j]
				items[j] = LibraryItem{LikedTrack: &lt}
			}
			t.sections[i].Items = items
			t.sections[i].Total = s.LikedTotal()

		case SectionRecentlyPlayed:
			recent := s.RecentlyPlayed()
			items := make([]LibraryItem, len(recent))
			for j := range recent {
				ph := recent[j]
				items[j] = LibraryItem{PlayHistory: &ph}
			}
			t.sections[i].Items = items
		}
	}
}

// --- Message types ---

// libraryLoadedMsg is sent when playlists have been loaded from the API.
type libraryLoadedMsg struct {
	playlists []api.SimplePlaylist
	total     int
}

// savedAlbumsLoadedMsg is sent when saved albums have been loaded from the API.
type savedAlbumsLoadedMsg struct {
	albums []api.SavedAlbum
}

// likedTracksLoadedMsg is sent when liked tracks have been loaded from the API.
type likedTracksLoadedMsg struct {
	tracks []api.SavedTrack
	total  int
}

// recentlyPlayedLoadedMsg is sent when recently played has been loaded from the API.
type recentlyPlayedLoadedMsg struct {
	items []api.PlayHistory
}

// playContextMsg is sent when the user selects a playlist or album to play.
type playContextMsg struct {
	contextURI string
}

// playTrackMsg is sent when the user selects a specific track to play.
type playTrackMsg struct {
	trackURI string
}

// addToQueueMsg is sent when the user presses 'a' on a track.
type addToQueueMsg struct {
	trackURI string
}

// likeToggleResultMsg carries the result of a like/unlike operation.
type likeToggleResultMsg struct {
	trackID string
	err     error
}

// expandSectionMsg triggers expanding a section (used internally and in tests).
type expandSectionMsg struct {
	section SectionType
}

// --- LibraryPane ---

// LibraryPane is the left-pane Bubble Tea model for the library browser.
// It renders the collapsible section tree and dispatches commands for
// playback, likes, and queue additions.
// It reads all data from the Store; it never stores API data in its own fields.
type LibraryPane struct {
	store   *state.Store
	theme   theme.Theme
	library *api.LibraryClient

	tree    *LibraryTree
	focused bool
	width   int
	height  int
}

// NewLibraryPane creates a LibraryPane with the given store and theme.
// The library client is nil initially — it is set by the root app via SetLibrary.
func NewLibraryPane(s *state.Store, t theme.Theme, focused bool) *LibraryPane {
	return &LibraryPane{
		store:   s,
		theme:   t,
		focused: focused,
		tree:    NewLibraryTree(),
	}
}

// SetLibrary injects the API library client into the pane.
// Called by the root app model after construction.
func (p *LibraryPane) SetLibrary(library *api.LibraryClient) {
	p.library = library
}

// SetSize updates the pane's dimensions (called by the root model on WindowSizeMsg).
func (p *LibraryPane) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetFocused updates the focused state of the pane.
func (p *LibraryPane) SetFocused(focused bool) {
	p.focused = focused
}

// Init returns a batch command that fetches playlists and recently played on startup.
func (p *LibraryPane) Init() tea.Cmd {
	return tea.Batch(
		p.fetchPlaylistsCmd(0),
		p.fetchRecentlyPlayedCmd(),
	)
}

// Update handles all messages for the LibraryPane.
func (p *LibraryPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Keep tree in sync with store on every update so navigation sees current data.
	p.tree.UpdateFromStore(p.store)

	switch m := msg.(type) {
	case expandSectionMsg:
		return p.handleExpandSection(m.section)

	case libraryLoadedMsg:
		p.store.SetPlaylists(m.playlists)
		p.store.SetPlaylistsTotal(m.total)
		p.tree.UpdateFromStore(p.store)
		return p, nil

	case savedAlbumsLoadedMsg:
		p.store.SetSavedAlbums(m.albums)
		p.tree.UpdateFromStore(p.store)
		p.tree.SetSectionLoading(SectionAlbums, false)
		return p, nil

	case likedTracksLoadedMsg:
		p.store.SetLikedTracks(m.tracks)
		p.store.SetLikedTotal(m.total)
		p.tree.UpdateFromStore(p.store)
		p.tree.SetSectionLoading(SectionLikedSongs, false)
		return p, nil

	case recentlyPlayedLoadedMsg:
		p.store.SetRecentlyPlayed(m.items)
		p.tree.UpdateFromStore(p.store)
		return p, nil

	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		return p.handleKey(m)
	}

	return p, nil
}

// View renders the library pane. It reads from the store and never calls the API.
func (p *LibraryPane) View() string {
	// Refresh tree from store before rendering.
	p.tree.UpdateFromStore(p.store)

	headerStyle := lipgloss.NewStyle().Foreground(p.theme.SectionHeader()).Bold(true)
	sectionStyle := lipgloss.NewStyle().Foreground(p.theme.SectionHeader())
	primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	selectedStyle := lipgloss.NewStyle().
		Background(p.theme.SelectedBg()).
		Foreground(p.theme.SelectedFg())
	playingStyle := lipgloss.NewStyle().Foreground(p.theme.PlayingIndicator())

	lines := []string{
		headerStyle.Render("LIBRARY"),
		strings.Repeat("─", maxInt(p.width-4, 20)),
		"",
	}

	// Get current playing track URI for the playing indicator.
	var playingURI string
	if ps := p.store.PlaybackState(); ps != nil && ps.Item != nil {
		playingURI = ps.Item.URI
	}

	row := 0
	for _, section := range p.tree.Sections() {
		// Section header row
		arrow := "▸"
		if section.Expanded {
			arrow = "▾"
		}
		countStr := ""
		if section.Total > 0 {
			countStr = fmt.Sprintf(" (%d)", section.Total)
		} else if len(section.Items) > 0 {
			countStr = fmt.Sprintf(" (%d)", len(section.Items))
		}
		if section.Loading {
			countStr = " ..."
		}

		headerText := fmt.Sprintf("%s %s%s", arrow, section.Name, countStr)

		var headerRendered string
		if row == p.tree.cursorPos {
			headerRendered = selectedStyle.Render(headerText)
		} else {
			headerRendered = sectionStyle.Render(headerText)
		}
		lines = append(lines, headerRendered)
		row++

		// Items (only when expanded)
		if section.Expanded {
			for _, item := range section.Items {
				name := item.displayName()
				uri := item.trackURI()

				var indicator string
				if uri != "" && uri == playingURI {
					indicator = playingStyle.Render("▶") + " "
				} else {
					indicator = "  "
				}

				itemText := fmt.Sprintf("  %s%s", indicator, name)

				var rendered string
				if row == p.tree.cursorPos {
					rendered = selectedStyle.Render(itemText)
				} else if uri != "" && uri == playingURI {
					rendered = primaryStyle.Render(itemText)
				} else {
					rendered = mutedStyle.Render(itemText)
				}
				lines = append(lines, rendered)
				row++
			}
		}
	}

	return strings.Join(lines, "\n")
}

// handleKey dispatches key events to the appropriate action.
func (p *LibraryPane) handleKey(msg tea.KeyMsg) (*LibraryPane, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "j",
		msg.Type == tea.KeyDown:
		p.tree.MoveDown()
		return p, p.loadMoreIfNearBottom()

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "k",
		msg.Type == tea.KeyUp:
		p.tree.MoveUp()
		return p, nil

	case msg.Type == tea.KeyEnter:
		return p.handleEnter()

	case msg.Type == tea.KeyBackspace:
		// Collapse current section and move to its header.
		p.tree.SetSectionExpanded(p.tree.CurrentSectionType(), false)
		return p, nil

	case msg.Type == tea.KeyPgDown:
		for i := 0; i < 10; i++ {
			p.tree.MoveDown()
		}
		return p, nil

	case msg.Type == tea.KeyPgUp:
		for i := 0; i < 10; i++ {
			p.tree.MoveUp()
		}
		return p, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "g":
		// Jump to top
		for p.tree.cursorPos > 0 {
			p.tree.MoveUp()
		}
		return p, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "G":
		// Jump to bottom
		for {
			prev := p.tree.cursorPos
			p.tree.MoveDown()
			if p.tree.cursorPos == prev {
				break
			}
		}
		return p, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "a":
		return p.handleAddToQueue()

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "l":
		return p.handleToggleLike()
	}

	return p, nil
}

// handleEnter processes Enter key: expands/collapses sections or plays items.
func (p *LibraryPane) handleEnter() (*LibraryPane, tea.Cmd) {
	if p.tree.IsOnSectionHeader() {
		sectionType := p.tree.CurrentSectionType()
		return p.handleExpandSection(sectionType)
	}

	item := p.tree.SelectedItem()
	if item == nil {
		return p, nil
	}

	// Playlist or album: play with context URI
	if uri := item.playableContextURI(); uri != "" {
		return p, func() tea.Msg { return playContextMsg{contextURI: uri} }
	}

	// Liked track: play with track URI
	if item.LikedTrack != nil {
		uri := item.LikedTrack.Track.URI
		return p, func() tea.Msg { return playTrackMsg{trackURI: uri} }
	}

	// Recently played track: play with track URI
	if item.PlayHistory != nil {
		uri := item.PlayHistory.Track.URI
		return p, func() tea.Msg { return playTrackMsg{trackURI: uri} }
	}

	return p, nil
}

// handleExpandSection expands a section and triggers a lazy fetch if needed.
func (p *LibraryPane) handleExpandSection(sectionType SectionType) (*LibraryPane, tea.Cmd) {
	p.tree.SetSectionExpanded(sectionType, true)

	switch sectionType {
	case SectionAlbums:
		if !p.store.AlbumsLoaded() {
			p.tree.SetSectionLoading(SectionAlbums, true)
			return p, p.fetchAlbumsCmd(0)
		}

	case SectionLikedSongs:
		if !p.store.LikedLoaded() {
			p.tree.SetSectionLoading(SectionLikedSongs, true)
			return p, p.fetchLikedTracksCmd(0)
		}
	}

	return p, nil
}

// handleAddToQueue handles the 'a' key — adds the selected track to queue.
func (p *LibraryPane) handleAddToQueue() (*LibraryPane, tea.Cmd) {
	item := p.tree.SelectedItem()
	if item == nil {
		return p, nil
	}
	uri := item.trackURI()
	if uri == "" {
		return p, nil
	}
	return p, func() tea.Msg { return addToQueueMsg{trackURI: uri} }
}

// handleToggleLike handles the 'l' key — likes/unlikes the selected track.
func (p *LibraryPane) handleToggleLike() (*LibraryPane, tea.Cmd) {
	item := p.tree.SelectedItem()
	if item == nil {
		return p, nil
	}
	id := item.trackID()
	if id == "" {
		return p, nil
	}
	library := p.library
	return p, func() tea.Msg {
		if library == nil {
			return likeToggleResultMsg{trackID: id}
		}
		err := library.LikeTrack(context.Background(), id)
		return likeToggleResultMsg{trackID: id, err: err}
	}
}

// loadMoreIfNearBottom triggers a fetch when cursor is within 5 items of the section end.
func (p *LibraryPane) loadMoreIfNearBottom() tea.Cmd {
	sectionType := p.tree.CurrentSectionType()
	idx := p.tree.ItemIndexInSection()
	if idx < 0 {
		return nil
	}

	for _, s := range p.tree.sections {
		if s.Type != sectionType || !s.Expanded {
			continue
		}
		itemCount := len(s.Items)
		if idx >= itemCount-5 {
			switch sectionType {
			case SectionPlaylists:
				total := p.store.PlaylistsTotal()
				loaded := len(p.store.Playlists())
				if loaded < total {
					return p.fetchPlaylistsCmd(loaded)
				}
			case SectionLikedSongs:
				total := p.store.LikedTotal()
				loaded := len(p.store.LikedTracks())
				if loaded < total {
					return p.fetchLikedTracksCmd(loaded)
				}
			}
		}
	}
	return nil
}

// --- Fetch commands ---

// fetchPlaylistsCmd creates a command that fetches playlists from the API.
func (p *LibraryPane) fetchPlaylistsCmd(offset int) tea.Cmd {
	library := p.library
	return func() tea.Msg {
		if library == nil {
			return libraryLoadedMsg{}
		}
		playlists, err := library.GetPlaylists(context.Background(), 50, offset)
		if err != nil {
			return libraryLoadedMsg{}
		}
		return libraryLoadedMsg{playlists: playlists, total: len(playlists) + offset}
	}
}

// fetchAlbumsCmd creates a command that fetches saved albums from the API.
func (p *LibraryPane) fetchAlbumsCmd(offset int) tea.Cmd {
	library := p.library
	return func() tea.Msg {
		if library == nil {
			return savedAlbumsLoadedMsg{}
		}
		albums, err := library.GetSavedAlbums(context.Background(), 50, offset)
		if err != nil {
			return savedAlbumsLoadedMsg{}
		}
		return savedAlbumsLoadedMsg{albums: albums}
	}
}

// fetchLikedTracksCmd creates a command that fetches liked tracks from the API.
func (p *LibraryPane) fetchLikedTracksCmd(offset int) tea.Cmd {
	library := p.library
	return func() tea.Msg {
		if library == nil {
			return likedTracksLoadedMsg{}
		}
		tracks, err := library.GetLikedTracks(context.Background(), 50, offset)
		if err != nil {
			return likedTracksLoadedMsg{}
		}
		return likedTracksLoadedMsg{tracks: tracks, total: len(tracks) + offset}
	}
}

// fetchRecentlyPlayedCmd creates a command that fetches recently played tracks.
func (p *LibraryPane) fetchRecentlyPlayedCmd() tea.Cmd {
	library := p.library
	return func() tea.Msg {
		if library == nil {
			return recentlyPlayedLoadedMsg{}
		}
		items, err := library.GetRecentlyPlayed(context.Background(), 20)
		if err != nil {
			return recentlyPlayedLoadedMsg{}
		}
		return recentlyPlayedLoadedMsg{items: items}
	}
}

// maxInt returns the larger of two ints.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
