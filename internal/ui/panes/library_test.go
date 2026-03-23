package panes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- LibraryTree Tests (Task 3.3) ---

func TestLibraryTree_MoveDown(t *testing.T) {
	tree := NewLibraryTree()

	tree.MoveDown()

	assert.Equal(t, 1, tree.CursorPos())
}

func TestLibraryTree_MoveUp(t *testing.T) {
	tree := NewLibraryTree()
	tree.MoveDown()
	tree.MoveDown()

	tree.MoveUp()

	assert.Equal(t, 1, tree.CursorPos())
}

func TestLibraryTree_MoveDown_AtBottom(t *testing.T) {
	tree := NewLibraryTree()
	// Move down far past the end
	for i := 0; i < 20; i++ {
		tree.MoveDown()
	}

	pos := tree.CursorPos()
	tree.MoveDown()
	assert.Equal(t, pos, tree.CursorPos(), "cursor should not move past bottom")
}

func TestLibraryTree_MoveUp_AtTop(t *testing.T) {
	tree := NewLibraryTree()

	tree.MoveUp() // already at top

	assert.Equal(t, 0, tree.CursorPos(), "cursor should not move above top")
}

func TestLibraryTree_ToggleSection_Expands(t *testing.T) {
	tree := NewLibraryTree()
	// cursor is at index 0 = Playlists section header

	tree.ToggleSection()

	assert.True(t, tree.Sections()[0].Expanded, "section should be expanded after toggle")
}

func TestLibraryTree_ToggleSection_Collapses(t *testing.T) {
	tree := NewLibraryTree()
	tree.ToggleSection() // expand

	tree.ToggleSection() // collapse

	assert.False(t, tree.Sections()[0].Expanded, "section should be collapsed after second toggle")
}

func TestLibraryTree_SelectedItem(t *testing.T) {
	tree := NewLibraryTree()
	// Expand Playlists section which has items
	tree.Sections()[0].Items = []LibraryItem{
		{Playlist: &api.SimplePlaylist{ID: "pl1", Name: "Chill Vibes"}},
		{Playlist: &api.SimplePlaylist{ID: "pl2", Name: "Workout Mix"}},
	}
	tree.Sections()[0].Expanded = true

	tree.MoveDown() // move to first item under Playlists

	item := tree.SelectedItem()
	require.NotNil(t, item)
	assert.NotNil(t, item.Playlist)
	assert.Equal(t, "pl1", item.Playlist.ID)
}

// --- LibraryPane Tests (Task 3.4) ---

func TestLibraryPane_Init_FetchesPlaylistsAndRecent(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	cmd := pane.Init()

	assert.NotNil(t, cmd, "Init should return a batch command to fetch playlists and recently played")
}

func TestLibraryPane_View_ShowsSections(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	view := pane.View()

	assert.Contains(t, view, "LIBRARY", "view should contain LIBRARY header")
	assert.Contains(t, view, "Playlists", "view should contain Playlists section")
	assert.Contains(t, view, "Albums", "view should contain Albums section")
}

func TestLibraryPane_View_PlayingIndicator(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Set currently playing track
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item: &api.Track{
			ID:   "track-xyz789",
			Name: "Blinding Lights",
			URI:  "spotify:track:track-xyz789",
		},
	})

	// Add recently played with the same track
	s.SetRecentlyPlayed([]api.PlayHistory{
		{
			Track:    api.Track{ID: "track-xyz789", Name: "Blinding Lights", URI: "spotify:track:track-xyz789"},
			PlayedAt: "2024-03-01T22:15:00Z",
		},
	})

	pane := NewLibraryPane(s, th, true)
	// Expand recently played section to show the items
	pane.tree.SetSectionExpanded(SectionRecentlyPlayed, true)

	view := pane.View()

	assert.Contains(t, view, "▶", "view should contain playing indicator for current track")
}

func TestLibraryPane_Update_Enter_OnPlaylist(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Pre-populate playlists in store
	s.SetPlaylists([]api.SimplePlaylist{
		{ID: "pl1", Name: "Chill Vibes", URI: "spotify:playlist:pl1"},
	})

	pane := NewLibraryPane(s, th, true)
	// Expand the playlists section
	expandMsg := expandSectionMsg{section: SectionPlaylists}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	// Move down to first playlist item
	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pane = m.(*LibraryPane)

	// Press Enter on the playlist
	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = m

	require.NotNil(t, cmd, "Enter on playlist should produce a play command")

	// Execute the command and check the message type
	msg := cmd()
	playMsg, ok := msg.(PlayContextMsg)
	require.True(t, ok, "expected PlayContextMsg, got %T", msg)
	assert.Equal(t, "spotify:playlist:pl1", playMsg.ContextURI)
}

func TestLibraryPane_Update_Enter_OnSection(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	// Cursor starts at Playlists section header — Enter should expand
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pane = m.(*LibraryPane)

	sections := pane.tree.Sections()
	playlistSection := findSectionByType(sections, SectionPlaylists)
	require.NotNil(t, playlistSection)
	assert.True(t, playlistSection.Expanded, "Enter on section header should expand the section")
}

func TestLibraryPane_Update_A_AddsToQueue(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Set recently played so we have a track
	s.SetRecentlyPlayed([]api.PlayHistory{
		{Track: api.Track{ID: "track-xyz789", Name: "Blinding Lights", URI: "spotify:track:track-xyz789"}},
	})

	pane := NewLibraryPane(s, th, true)
	// Expand recently played section to show tracks
	pane.tree.SetSectionExpanded(SectionRecentlyPlayed, true)

	// Navigate to the track
	var foundTrack bool
	for i := 0; i < 15; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.PlayHistory != nil {
			foundTrack = true
			break
		}
	}
	require.True(t, foundTrack, "should be able to navigate to a recently played track")

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	_ = m

	assert.NotNil(t, cmd, "'a' on a track should produce an add-to-queue command")
}

func TestLibraryPane_Update_L_ToggleLike(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Set recently played so we have a track to like
	s.SetRecentlyPlayed([]api.PlayHistory{
		{Track: api.Track{ID: "track-xyz789", Name: "Blinding Lights", URI: "spotify:track:track-xyz789"}},
	})

	pane := NewLibraryPane(s, th, true)
	// Expand recently played section
	pane.tree.SetSectionExpanded(SectionRecentlyPlayed, true)

	// Navigate to the track
	var foundTrack bool
	for i := 0; i < 15; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.PlayHistory != nil {
			foundTrack = true
			break
		}
	}
	require.True(t, foundTrack, "should be able to navigate to a recently played track")

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	_ = m

	assert.NotNil(t, cmd, "'l' on a track should produce a like/unlike command")
}

func TestLibraryPane_View_EmptySection(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	// Mark albums section as loading
	pane.tree.SetSectionLoading(SectionAlbums, true)

	view := pane.View()

	assert.Contains(t, view, "Albums", "view should still show Albums section header when loading")
}

// --- Lazy loading Tests (Task 3.5) ---

func TestLibraryPane_ExpandSection_FetchesIfNotCached(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	// Albums are not yet cached — expanding should trigger a fetch
	expandMsg := expandSectionMsg{section: SectionAlbums}
	m, cmd := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	albumSection := findSectionByType(pane.tree.Sections(), SectionAlbums)
	require.NotNil(t, albumSection)
	assert.True(t, albumSection.Expanded, "section should be expanded")
	assert.NotNil(t, cmd, "expanding uncached section should trigger fetch command")
}

func TestLibraryPane_ExpandSection_SkipsFetchIfCached(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Pre-populate albums (already cached)
	s.SetSavedAlbums([]api.SavedAlbum{
		{AddedAt: "2024-01-15T10:30:00Z", Album: api.FullAlbum{ID: "album-1", Name: "After Hours", URI: "spotify:album:album-1"}},
	})

	pane := NewLibraryPane(s, th, true)
	expandMsg := expandSectionMsg{section: SectionAlbums}
	m, cmd := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	albumSection := findSectionByType(pane.tree.Sections(), SectionAlbums)
	require.NotNil(t, albumSection)
	assert.True(t, albumSection.Expanded, "section should be expanded")

	// cmd should be nil when data is already cached
	if cmd != nil {
		msg := cmd()
		_, isFetch := msg.(savedAlbumsLoadedMsg)
		assert.False(t, isFetch, "should not re-fetch albums that are already cached")
	}
}

func TestLibraryPane_ScrollNearBottom_LoadsMore(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Populate 10 playlists with total=50 (pagination)
	playlists := make([]api.SimplePlaylist, 10)
	for i := range playlists {
		playlists[i] = api.SimplePlaylist{
			ID:   "pl" + string(rune('a'+i)),
			Name: "Playlist " + string(rune('A'+i)),
			URI:  "spotify:playlist:pl" + string(rune('a'+i)),
		}
	}
	s.SetPlaylists(playlists)
	s.SetPlaylistsTotal(50) // 50 total, 10 loaded = more to load

	pane := NewLibraryPane(s, th, true)
	// Expand playlists section
	expandMsg := expandSectionMsg{section: SectionPlaylists}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	// Move near the bottom (within 5 items of end)
	var loadMoreCmd tea.Cmd
	for i := 0; i < 8; i++ {
		m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		if cmd != nil {
			loadMoreCmd = cmd
		}
	}

	assert.NotNil(t, loadMoreCmd, "scrolling near bottom should trigger load-more command")
}

// --- Additional coverage tests ---

func TestLibraryPane_SetLibrary(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	client := api.NewLibraryClient("http://localhost", "token")
	pane.SetLibrary(client)

	// No panic — library was set successfully
}

func TestLibraryPane_SetSize(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	pane.SetSize(80, 24)
	// No panic — size set successfully
}

func TestLibraryPane_SetFocused(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	pane.SetFocused(true)
	// Verify the pane responds to keys when focused
	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	assert.NotNil(t, m)
	_ = cmd
}

func TestLibraryPane_Update_IgnoresKeysWhenNotFocused(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false) // not focused

	// Should not move cursor when not focused
	initialPos := pane.tree.CursorPos()
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pane = m.(*LibraryPane)
	assert.Equal(t, initialPos, pane.tree.CursorPos(), "unfocused pane should not respond to key events")
}

func TestLibraryPane_Update_ArrowKeys(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	// Arrow down
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyDown})
	pane = m.(*LibraryPane)
	assert.Equal(t, 1, pane.tree.CursorPos())

	// Arrow up
	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyUp})
	pane = m.(*LibraryPane)
	assert.Equal(t, 0, pane.tree.CursorPos())
}

func TestLibraryPane_Update_PgUpPgDown(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	// PgDown
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	pane = m.(*LibraryPane)
	assert.True(t, pane.tree.CursorPos() > 0, "PgDown should move cursor down")

	// PgUp
	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	pane = m.(*LibraryPane)
	assert.Equal(t, 0, pane.tree.CursorPos())
}

func TestLibraryPane_Update_GJumpsToTop(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	// Move down first
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pane = m.(*LibraryPane)
	assert.Equal(t, 1, pane.tree.CursorPos())

	// g jumps to top
	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	pane = m.(*LibraryPane)
	assert.Equal(t, 0, pane.tree.CursorPos())
}

func TestLibraryPane_Update_GCapitalJumpsToBottom(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	pane = m.(*LibraryPane)

	// Should be at the last section
	totalSections := len(pane.tree.Sections())
	assert.Equal(t, totalSections-1, pane.tree.CursorPos(), "G should jump to last row")
}

func TestLibraryPane_Update_Backspace_CollapseSection(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	// Expand playlists section
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pane = m.(*LibraryPane)

	// Backspace should collapse
	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	pane = m.(*LibraryPane)

	playlistSection := findSectionByType(pane.tree.Sections(), SectionPlaylists)
	require.NotNil(t, playlistSection)
	assert.False(t, playlistSection.Expanded, "Backspace should collapse the section")
}

func TestLibraryPane_Update_Enter_OnLikedSong(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Pre-populate liked tracks
	s.SetLikedTracks([]api.SavedTrack{
		{AddedAt: "2024-01-01", Track: api.Track{ID: "t1", Name: "Song", URI: "spotify:track:t1"}},
	})

	pane := NewLibraryPane(s, th, true)
	// Expand liked songs
	expandMsg := expandSectionMsg{section: SectionLikedSongs}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	// Navigate to liked song
	for i := 0; i < 15; i++ {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.LikedTrack != nil {
			break
		}
	}

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = m
	require.NotNil(t, cmd)

	msg := cmd()
	playMsg, ok := msg.(PlayTrackMsg)
	require.True(t, ok, "Enter on liked song should produce PlayTrackMsg, got %T", msg)
	assert.Equal(t, "spotify:track:t1", playMsg.TrackURI)
}

func TestLibraryPane_Update_Enter_OnAlbum(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Pre-populate albums
	s.SetSavedAlbums([]api.SavedAlbum{
		{Album: api.FullAlbum{ID: "album-1", Name: "After Hours", URI: "spotify:album:album-1"}},
	})

	pane := NewLibraryPane(s, th, true)
	// Expand albums
	expandMsg := expandSectionMsg{section: SectionAlbums}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	// Navigate to album
	for i := 0; i < 15; i++ {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.Album != nil {
			break
		}
	}

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = m
	require.NotNil(t, cmd)

	msg := cmd()
	playMsg, ok := msg.(PlayContextMsg)
	require.True(t, ok, "Enter on album should produce PlayContextMsg, got %T", msg)
	assert.Equal(t, "spotify:album:album-1", playMsg.ContextURI)
}

func TestLibraryPane_Update_Enter_OnRecentlyPlayedTrack(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	s.SetRecentlyPlayed([]api.PlayHistory{
		{Track: api.Track{ID: "recent-1", Name: "Recent Song", URI: "spotify:track:recent-1"}},
	})

	pane := NewLibraryPane(s, th, true)
	pane.tree.SetSectionExpanded(SectionRecentlyPlayed, true)

	// Navigate to recently played track
	for i := 0; i < 15; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.PlayHistory != nil {
			break
		}
	}

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = m
	require.NotNil(t, cmd)

	msg := cmd()
	playMsg, ok := msg.(PlayTrackMsg)
	require.True(t, ok, "Enter on recently played should produce PlayTrackMsg, got %T", msg)
	assert.Equal(t, "spotify:track:recent-1", playMsg.TrackURI)
}

func TestLibraryPane_Update_LibraryLoadedMsg(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	loadedMsg := libraryLoadedMsg{
		playlists: []api.SimplePlaylist{{ID: "pl1", Name: "Chill Vibes"}},
		total:     5,
	}
	m, _ := pane.Update(loadedMsg)
	_ = m

	assert.Len(t, s.Playlists(), 1)
	assert.Equal(t, 5, s.PlaylistsTotal())
}

func TestLibraryPane_Update_SavedAlbumsLoadedMsg(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	loadedMsg := savedAlbumsLoadedMsg{
		albums: []api.SavedAlbum{
			{Album: api.FullAlbum{ID: "album-1", Name: "After Hours"}},
		},
	}
	m, _ := pane.Update(loadedMsg)
	pane = m.(*LibraryPane)
	_ = pane

	assert.Len(t, s.SavedAlbums(), 1)
	assert.True(t, s.AlbumsLoaded())
}

func TestLibraryPane_Update_LikedTracksLoadedMsg(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	loadedMsg := likedTracksLoadedMsg{
		tracks: []api.SavedTrack{
			{Track: api.Track{ID: "t1", Name: "Song"}},
		},
		total: 287,
	}
	m, _ := pane.Update(loadedMsg)
	_ = m

	assert.Len(t, s.LikedTracks(), 1)
	assert.Equal(t, 287, s.LikedTotal())
}

func TestLibraryPane_Update_RecentlyPlayedLoadedMsg(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	loadedMsg := recentlyPlayedLoadedMsg{
		items: []api.PlayHistory{
			{Track: api.Track{ID: "t1", Name: "Song"}, PlayedAt: "2024-03-01T22:15:00Z"},
		},
	}
	m, _ := pane.Update(loadedMsg)
	_ = m

	assert.Len(t, s.RecentlyPlayed(), 1)
}

func TestLibraryPane_Update_ExpandSection_LikedSongs(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	// Liked songs not loaded — expanding should trigger fetch
	expandMsg := expandSectionMsg{section: SectionLikedSongs}
	m, cmd := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	section := findSectionByType(pane.tree.Sections(), SectionLikedSongs)
	require.NotNil(t, section)
	assert.True(t, section.Expanded)
	assert.NotNil(t, cmd, "expanding unloaded liked songs should trigger fetch")
}

func TestLibraryTree_LibraryExpandMsg(t *testing.T) {
	msg := LibraryExpandMsg(SectionAlbums)
	expandMsg, ok := msg.(expandSectionMsg)
	require.True(t, ok, "LibraryExpandMsg should return expandSectionMsg")
	assert.Equal(t, SectionAlbums, expandMsg.section)
}

func TestLibraryItem_DisplayNames(t *testing.T) {
	tests := []struct {
		name string
		item LibraryItem
		want string
	}{
		{
			name: "playlist",
			item: LibraryItem{Playlist: &api.SimplePlaylist{Name: "Chill Vibes"}},
			want: "Chill Vibes",
		},
		{
			name: "album",
			item: LibraryItem{Album: &api.SavedAlbum{Album: api.FullAlbum{Name: "After Hours"}}},
			want: "After Hours",
		},
		{
			name: "liked track",
			item: LibraryItem{LikedTrack: &api.SavedTrack{Track: api.Track{Name: "Blinding Lights"}}},
			want: "Blinding Lights",
		},
		{
			name: "play history",
			item: LibraryItem{PlayHistory: &api.PlayHistory{Track: api.Track{Name: "Levitating"}}},
			want: "Levitating",
		},
		{
			name: "empty",
			item: LibraryItem{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.item.displayName())
		})
	}
}

func TestLibraryItem_TrackURI(t *testing.T) {
	tests := []struct {
		name string
		item LibraryItem
		want string
	}{
		{
			name: "liked track",
			item: LibraryItem{LikedTrack: &api.SavedTrack{Track: api.Track{URI: "spotify:track:t1"}}},
			want: "spotify:track:t1",
		},
		{
			name: "play history",
			item: LibraryItem{PlayHistory: &api.PlayHistory{Track: api.Track{URI: "spotify:track:t2"}}},
			want: "spotify:track:t2",
		},
		{
			name: "playlist (no track URI)",
			item: LibraryItem{Playlist: &api.SimplePlaylist{URI: "spotify:playlist:p1"}},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.item.trackURI())
		})
	}
}

func TestLibraryItem_PlayableContextURI(t *testing.T) {
	tests := []struct {
		name string
		item LibraryItem
		want string
	}{
		{
			name: "playlist",
			item: LibraryItem{Playlist: &api.SimplePlaylist{URI: "spotify:playlist:p1"}},
			want: "spotify:playlist:p1",
		},
		{
			name: "album",
			item: LibraryItem{Album: &api.SavedAlbum{Album: api.FullAlbum{URI: "spotify:album:a1"}}},
			want: "spotify:album:a1",
		},
		{
			name: "track (no context URI)",
			item: LibraryItem{LikedTrack: &api.SavedTrack{Track: api.Track{URI: "spotify:track:t1"}}},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.item.playableContextURI())
		})
	}
}

func TestLibraryPane_Update_A_NoTrackSelected(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	// Cursor is on section header (no track) — 'a' should return nil cmd
	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	_ = m
	assert.Nil(t, cmd, "'a' on section header should return nil cmd")
}

func TestLibraryPane_Update_L_NoTrackSelected(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	// Cursor is on section header (no track) — 'l' should return nil cmd
	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	_ = m
	assert.Nil(t, cmd, "'l' on section header should return nil cmd")
}

func TestLibraryPane_Update_L_WithLibraryClient(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	s.SetRecentlyPlayed([]api.PlayHistory{
		{Track: api.Track{ID: "t1", Name: "Song", URI: "spotify:track:t1"}},
	})

	pane := NewLibraryPane(s, th, true)
	pane.tree.SetSectionExpanded(SectionRecentlyPlayed, true)

	// Navigate to track
	for i := 0; i < 15; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.PlayHistory != nil {
			break
		}
	}

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	_ = m
	require.NotNil(t, cmd)

	// Execute the command (library is nil, should return likeToggleResultMsg)
	result := cmd()
	likeResult, ok := result.(likeToggleResultMsg)
	require.True(t, ok, "expected likeToggleResultMsg, got %T", result)
	assert.Equal(t, "t1", likeResult.trackID)
}

// --- Helper functions ---

// findSectionByType finds a section by type in the sections slice.
func findSectionByType(sections []Section, sectionType SectionType) *Section {
	for i := range sections {
		if sections[i].Type == sectionType {
			return &sections[i]
		}
	}
	return nil
}
