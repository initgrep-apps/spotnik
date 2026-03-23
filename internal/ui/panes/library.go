package panes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// libraryItemKind distinguishes what kind of item a LibraryItem represents.
type libraryItemKind int

const (
	kindPlaylist libraryItemKind = iota
	kindAlbum
	kindLikedTrack
	kindPlayHistory
)

// LibraryItem represents a single row in the library tree.
// It holds only primitive strings needed for display and interaction.
// NOTE: previously held api.* pointers; changed to primitives to eliminate api/ import.
type LibraryItem struct {
	kind libraryItemKind

	// DisplayName is the human-readable name shown in the tree.
	DisplayName string

	// ContextURI is the Spotify context URI for playlists and albums.
	ContextURI string

	// TrackURI is the Spotify track URI for liked songs and recently played.
	TrackURI string

	// TrackID is the Spotify track ID used for like/unlike operations.
	TrackID string
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
type LibraryTree struct {
	sections  []Section
	cursorPos int
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

// visibleRows returns the total number of visible rows.
func (t *LibraryTree) visibleRows() int {
	count := 0
	for _, s := range t.sections {
		count++
		if s.Expanded {
			count += len(s.Items)
		}
	}
	return count
}

// MoveDown moves the cursor one row down, clamped at the last row.
func (t *LibraryTree) MoveDown() {
	maxRow := t.visibleRows() - 1
	if t.cursorPos < maxRow {
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
func (t *LibraryTree) ToggleSection() {
	row := 0
	for i := range t.sections {
		if row == t.cursorPos {
			t.sections[i].Expanded = !t.sections[i].Expanded
			return
		}
		row++
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
			return nil
		}
		row++
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

// CurrentSectionType returns the SectionType of the section the cursor is currently in.
func (t *LibraryTree) CurrentSectionType() SectionType {
	row := 0
	for _, s := range t.sections {
		if row == t.cursorPos {
			return s.Type
		}
		row++
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
			return -1
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
// Copies only the primitive fields needed for display/interaction.
func (t *LibraryTree) UpdateFromStore(s *state.Store) {
	for i := range t.sections {
		switch t.sections[i].Type {
		case SectionPlaylists:
			playlists := s.Playlists()
			items := make([]LibraryItem, len(playlists))
			for j, pl := range playlists {
				items[j] = LibraryItem{
					kind:        kindPlaylist,
					DisplayName: pl.Name,
					ContextURI:  pl.URI,
				}
			}
			t.sections[i].Items = items
			t.sections[i].Total = s.PlaylistsTotal()

		case SectionAlbums:
			albums := s.SavedAlbums()
			items := make([]LibraryItem, len(albums))
			for j, al := range albums {
				items[j] = LibraryItem{
					kind:        kindAlbum,
					DisplayName: al.Album.Name,
					ContextURI:  al.Album.URI,
				}
			}
			t.sections[i].Items = items

		case SectionLikedSongs:
			liked := s.LikedTracks()
			items := make([]LibraryItem, len(liked))
			for j, lt := range liked {
				items[j] = LibraryItem{
					kind:        kindLikedTrack,
					DisplayName: lt.Track.Name,
					TrackURI:    lt.Track.URI,
					TrackID:     lt.Track.ID,
				}
			}
			t.sections[i].Items = items
			t.sections[i].Total = s.LikedTotal()

		case SectionRecentlyPlayed:
			recent := s.RecentlyPlayed()
			items := make([]LibraryItem, len(recent))
			for j, ph := range recent {
				items[j] = LibraryItem{
					kind:        kindPlayHistory,
					DisplayName: ph.Track.Name,
					TrackURI:    ph.Track.URI,
					TrackID:     ph.Track.ID,
				}
			}
			t.sections[i].Items = items
		}
	}
}

// NOTE: Notification message types (LibraryLoadedMsg, AlbumsLoadedMsg,
// LikedTracksLoadedMsg, RecentlyPlayedLoadedMsg, LikeToggleResultMsg)
// are defined in messages.go as exported types because they are sent by
// app.go commands after writing data to the store.

// expandSectionMsg triggers expanding a section (used internally and in tests).
type expandSectionMsg struct {
	section SectionType
}

// LibraryExpandMsg creates an expandSectionMsg for use outside the panes package.
func LibraryExpandMsg(section SectionType) tea.Msg {
	return expandSectionMsg{section: section}
}

// --- LibraryPane ---

// LibraryPane is the left-pane Bubble Tea model for the library browser.
// It renders the collapsible section tree and dispatches request messages for
// playback, likes, queue additions, and data fetching.
// It reads all data from the Store; it never imports or calls the API directly.
type LibraryPane struct {
	store *state.Store
	theme theme.Theme

	tree    *LibraryTree
	focused bool
	width   int
	height  int
}

// NewLibraryPane creates a LibraryPane with the given store and theme.
func NewLibraryPane(s *state.Store, t theme.Theme, focused bool) *LibraryPane {
	return &LibraryPane{
		store:   s,
		theme:   t,
		focused: focused,
		tree:    NewLibraryTree(),
	}
}

// SetSize updates the pane's dimensions.
func (p *LibraryPane) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetFocused updates the focused state of the pane.
func (p *LibraryPane) SetFocused(focused bool) {
	p.focused = focused
}

// Init returns a batch command that requests playlists and recently played on startup.
func (p *LibraryPane) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return FetchPlaylistsRequestMsg{Offset: 0} },
		func() tea.Msg { return FetchRecentlyPlayedRequestMsg{} },
	)
}

// Update handles all messages for the LibraryPane.
func (p *LibraryPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	p.tree.UpdateFromStore(p.store)

	switch m := msg.(type) {
	case expandSectionMsg:
		return p.handleExpandSection(m.section)

	case LibraryLoadedMsg:
		p.tree.UpdateFromStore(p.store)
		// Auto-expand Playlists section when data arrives so items are
		// visible within 2 seconds of app start (per original spec).
		if len(p.store.Playlists()) > 0 {
			p.tree.SetSectionExpanded(SectionPlaylists, true)
		}
		return p, nil

	case AlbumsLoadedMsg:
		p.tree.UpdateFromStore(p.store)
		p.tree.SetSectionLoading(SectionAlbums, false)
		return p, nil

	case LikedTracksLoadedMsg:
		p.tree.UpdateFromStore(p.store)
		p.tree.SetSectionLoading(SectionLikedSongs, false)
		return p, nil

	case RecentlyPlayedLoadedMsg:
		p.tree.UpdateFromStore(p.store)
		// Auto-expand Recently Played section when data arrives.
		if len(p.store.RecentlyPlayed()) > 0 {
			p.tree.SetSectionExpanded(SectionRecentlyPlayed, true)
		}
		return p, nil

	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		return p.handleKey(m)
	}

	return p, nil
}

// View renders the library pane.
func (p *LibraryPane) View() string {
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
		strings.Repeat("─", paneMax(p.width-4, 20)),
		"",
	}

	var playingURI string
	if ps := p.store.PlaybackState(); ps != nil && ps.Item != nil {
		playingURI = ps.Item.URI
	}

	row := 0
	for _, section := range p.tree.Sections() {
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

		if section.Expanded {
			for _, item := range section.Items {
				name := item.DisplayName
				uri := item.TrackURI

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

// handleKey dispatches key events.
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
		for p.tree.cursorPos > 0 {
			p.tree.MoveUp()
		}
		return p, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "G":
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

// handleEnter processes Enter key.
func (p *LibraryPane) handleEnter() (*LibraryPane, tea.Cmd) {
	if p.tree.IsOnSectionHeader() {
		return p.handleExpandSection(p.tree.CurrentSectionType())
	}

	item := p.tree.SelectedItem()
	if item == nil {
		return p, nil
	}

	if item.ContextURI != "" {
		contextURI := item.ContextURI
		return p, func() tea.Msg { return PlayContextMsg{ContextURI: contextURI} }
	}

	if item.TrackURI != "" {
		trackURI := item.TrackURI
		return p, func() tea.Msg { return PlayTrackMsg{TrackURI: trackURI} }
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
			return p, func() tea.Msg { return FetchAlbumsRequestMsg{Offset: 0} }
		}

	case SectionLikedSongs:
		if !p.store.LikedLoaded() {
			p.tree.SetSectionLoading(SectionLikedSongs, true)
			return p, func() tea.Msg { return FetchLikedTracksRequestMsg{Offset: 0} }
		}
	}

	return p, nil
}

// handleAddToQueue handles the 'a' key.
func (p *LibraryPane) handleAddToQueue() (*LibraryPane, tea.Cmd) {
	item := p.tree.SelectedItem()
	if item == nil {
		return p, nil
	}
	uri := item.TrackURI
	if uri == "" {
		return p, nil
	}
	name := item.DisplayName
	return p, func() tea.Msg { return AddToQueueMsg{TrackURI: uri, TrackName: name} }
}

// handleToggleLike handles the 'l' key.
// Emits LikeTrackRequestMsg; the root app model dispatches the API call.
func (p *LibraryPane) handleToggleLike() (*LibraryPane, tea.Cmd) {
	item := p.tree.SelectedItem()
	if item == nil {
		return p, nil
	}
	id := item.TrackID
	if id == "" {
		return p, nil
	}
	unlike := item.kind == kindLikedTrack
	capturedID := id
	capturedUnlike := unlike
	return p, func() tea.Msg {
		return LikeTrackRequestMsg{TrackID: capturedID, Unlike: capturedUnlike}
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
					offset := loaded
					return func() tea.Msg { return FetchPlaylistsRequestMsg{Offset: offset} }
				}
			case SectionLikedSongs:
				total := p.store.LikedTotal()
				loaded := len(p.store.LikedTracks())
				if loaded < total {
					offset := loaded
					return func() tea.Msg { return FetchLikedTracksRequestMsg{Offset: offset} }
				}
			}
		}
	}
	return nil
}
