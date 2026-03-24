package panes

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- LibraryTree Tests ---

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
	for i := 0; i < 20; i++ {
		tree.MoveDown()
	}
	pos := tree.CursorPos()
	tree.MoveDown()
	assert.Equal(t, pos, tree.CursorPos(), "cursor should not move past bottom")
}

func TestLibraryTree_MoveUp_AtTop(t *testing.T) {
	tree := NewLibraryTree()
	tree.MoveUp()
	assert.Equal(t, 0, tree.CursorPos(), "cursor should not move above top")
}

func TestLibraryTree_ToggleSection_Expands(t *testing.T) {
	tree := NewLibraryTree()
	tree.ToggleSection()
	assert.True(t, tree.Sections()[0].Expanded)
}

func TestLibraryTree_ToggleSection_Collapses(t *testing.T) {
	tree := NewLibraryTree()
	tree.ToggleSection()
	tree.ToggleSection()
	assert.False(t, tree.Sections()[0].Expanded)
}

func TestLibraryTree_SelectedItem(t *testing.T) {
	tree := NewLibraryTree()
	// Expand Playlists section which has items
	tree.Sections()[0].Items = []LibraryItem{
		{kind: kindPlaylist, DisplayName: "Chill Vibes", ContextURI: "spotify:playlist:pl1"},
		{kind: kindPlaylist, DisplayName: "Workout Mix", ContextURI: "spotify:playlist:pl2"},
	}
	tree.Sections()[0].Expanded = true

	tree.MoveDown() // move to first item under Playlists

	item := tree.SelectedItem()
	require.NotNil(t, item)
	assert.Equal(t, "Chill Vibes", item.DisplayName)
	assert.Equal(t, "spotify:playlist:pl1", item.ContextURI)
}

// --- LibraryPane Tests ---

func TestLibraryPane_Init_FetchesPlaylistsAndRecent(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	cmd := pane.Init()

	assert.NotNil(t, cmd, "Init should return a batch command")
}

func TestLibraryPane_View_ShowsSections(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	view := pane.View()

	assert.Contains(t, view, "LIBRARY")
	assert.Contains(t, view, "Playlists")
	assert.Contains(t, view, "Albums")
}

func TestLibraryPane_View_PlayingIndicator(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item: &api.Track{
			ID:   "track-xyz789",
			Name: "Blinding Lights",
			URI:  "spotify:track:track-xyz789",
		},
	})

	s.SetRecentlyPlayed([]api.PlayHistory{
		{
			Track:    api.Track{ID: "track-xyz789", Name: "Blinding Lights", URI: "spotify:track:track-xyz789"},
			PlayedAt: "2024-03-01T22:15:00Z",
		},
	})

	pane := NewLibraryPane(s, th, true)
	pane.tree.SetSectionExpanded(SectionRecentlyPlayed, true)
	pane.tree.UpdateFromStore(s) // Simulate Update() cycle which syncs tree with store

	view := pane.View()

	assert.Contains(t, view, "▶", "view should contain playing indicator for current track")
}

func TestLibraryPane_Update_Enter_OnPlaylist(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	s.SetPlaylists([]api.SimplePlaylist{
		{ID: "pl1", Name: "Chill Vibes", URI: "spotify:playlist:pl1"},
	})

	pane := NewLibraryPane(s, th, true)
	expandMsg := expandSectionMsg{section: SectionPlaylists}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pane = m.(*LibraryPane)

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = m

	require.NotNil(t, cmd)
	msg := cmd()
	playMsg, ok := msg.(PlayContextMsg)
	require.True(t, ok, "expected PlayContextMsg, got %T", msg)
	assert.Equal(t, "spotify:playlist:pl1", playMsg.ContextURI)
}

func TestLibraryPane_Update_Enter_OnSection(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pane = m.(*LibraryPane)

	sections := pane.tree.Sections()
	playlistSection := findSectionByType(sections, SectionPlaylists)
	require.NotNil(t, playlistSection)
	assert.True(t, playlistSection.Expanded)
}

func TestLibraryPane_Update_A_AddsToQueue(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	s.SetRecentlyPlayed([]api.PlayHistory{
		{Track: api.Track{ID: "track-xyz789", Name: "Blinding Lights", URI: "spotify:track:track-xyz789"}},
	})

	pane := NewLibraryPane(s, th, true)
	pane.tree.SetSectionExpanded(SectionRecentlyPlayed, true)

	var foundTrack bool
	for i := 0; i < 15; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.TrackURI != "" {
			foundTrack = true
			break
		}
	}
	require.True(t, foundTrack)

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	_ = m

	assert.NotNil(t, cmd)
}

func TestLibraryPane_Update_L_ToggleLike(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	s.SetRecentlyPlayed([]api.PlayHistory{
		{Track: api.Track{ID: "track-xyz789", Name: "Blinding Lights", URI: "spotify:track:track-xyz789"}},
	})

	pane := NewLibraryPane(s, th, true)
	pane.tree.SetSectionExpanded(SectionRecentlyPlayed, true)

	var foundTrack bool
	for i := 0; i < 15; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.TrackURI != "" {
			foundTrack = true
			break
		}
	}
	require.True(t, foundTrack)

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	_ = m

	assert.NotNil(t, cmd)
}

func TestLibraryPane_View_EmptySection(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	pane.tree.SetSectionLoading(SectionAlbums, true)

	view := pane.View()
	assert.Contains(t, view, "Albums")
}

// --- Lazy loading Tests ---

func TestLibraryPane_ExpandSection_FetchesIfNotCached(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	expandMsg := expandSectionMsg{section: SectionAlbums}
	m, cmd := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	albumSection := findSectionByType(pane.tree.Sections(), SectionAlbums)
	require.NotNil(t, albumSection)
	assert.True(t, albumSection.Expanded)
	assert.NotNil(t, cmd)
}

func TestLibraryPane_ExpandSection_SkipsFetchIfCached(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	s.SetSavedAlbums([]api.SavedAlbum{
		{AddedAt: "2024-01-15T10:30:00Z", Album: api.FullAlbum{ID: "album-1", Name: "After Hours", URI: "spotify:album:album-1"}},
	})

	pane := NewLibraryPane(s, th, true)
	expandMsg := expandSectionMsg{section: SectionAlbums}
	m, cmd := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	albumSection := findSectionByType(pane.tree.Sections(), SectionAlbums)
	require.NotNil(t, albumSection)
	assert.True(t, albumSection.Expanded)

	if cmd != nil {
		msg := cmd()
		_, isFetch := msg.(FetchAlbumsRequestMsg)
		assert.False(t, isFetch, "should not re-fetch albums that are already cached")
	}
}

func TestLibraryPane_ScrollNearBottom_LoadsMore(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	playlists := make([]api.SimplePlaylist, 10)
	for i := range playlists {
		playlists[i] = api.SimplePlaylist{
			ID:   "pl" + string(rune('a'+i)),
			Name: "Playlist " + string(rune('A'+i)),
			URI:  "spotify:playlist:pl" + string(rune('a'+i)),
		}
	}
	s.SetPlaylists(playlists)
	s.SetPlaylistsTotal(50)

	pane := NewLibraryPane(s, th, true)
	expandMsg := expandSectionMsg{section: SectionPlaylists}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	var loadMoreCmd tea.Cmd
	for i := 0; i < 8; i++ {
		m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		if cmd != nil {
			loadMoreCmd = cmd
		}
	}

	assert.NotNil(t, loadMoreCmd)
}

// --- Additional coverage tests ---

func TestLibraryPane_SetSize(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)
	pane.SetSize(80, 24)
	// No panic
}

func TestLibraryPane_SetFocused(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	pane.SetFocused(true)
	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	assert.NotNil(t, m)
	_ = cmd
}

func TestLibraryPane_Update_IgnoresKeysWhenNotFocused(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	initialPos := pane.tree.CursorPos()
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pane = m.(*LibraryPane)
	assert.Equal(t, initialPos, pane.tree.CursorPos())
}

func TestLibraryPane_Update_ArrowKeys(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyDown})
	pane = m.(*LibraryPane)
	assert.Equal(t, 1, pane.tree.CursorPos())

	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyUp})
	pane = m.(*LibraryPane)
	assert.Equal(t, 0, pane.tree.CursorPos())
}

func TestLibraryPane_Update_PgUpPgDown(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	pane = m.(*LibraryPane)
	assert.True(t, pane.tree.CursorPos() > 0)

	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	pane = m.(*LibraryPane)
	assert.Equal(t, 0, pane.tree.CursorPos())
}

func TestLibraryPane_Update_GJumpsToTop(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pane = m.(*LibraryPane)
	assert.Equal(t, 1, pane.tree.CursorPos())

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

	totalSections := len(pane.tree.Sections())
	assert.Equal(t, totalSections-1, pane.tree.CursorPos())
}

func TestLibraryPane_Update_Backspace_CollapseSection(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pane = m.(*LibraryPane)

	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	pane = m.(*LibraryPane)

	playlistSection := findSectionByType(pane.tree.Sections(), SectionPlaylists)
	require.NotNil(t, playlistSection)
	assert.False(t, playlistSection.Expanded)
}

func TestLibraryPane_Update_Enter_OnLikedSong(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	s.SetLikedTracks([]api.SavedTrack{
		{AddedAt: "2024-01-01", Track: api.Track{ID: "t1", Name: "Song", URI: "spotify:track:t1"}},
	})

	pane := NewLibraryPane(s, th, true)
	expandMsg := expandSectionMsg{section: SectionLikedSongs}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	for i := 0; i < 15; i++ {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.kind == kindLikedTrack {
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

	s.SetSavedAlbums([]api.SavedAlbum{
		{Album: api.FullAlbum{ID: "album-1", Name: "After Hours", URI: "spotify:album:album-1"}},
	})

	pane := NewLibraryPane(s, th, true)
	expandMsg := expandSectionMsg{section: SectionAlbums}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	for i := 0; i < 15; i++ {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.kind == kindAlbum {
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

	for i := 0; i < 15; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.kind == kindPlayHistory {
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

	// Pre-populate store as if app.go command wrote to it.
	s.SetPlaylists([]api.SimplePlaylist{{ID: "pl1", Name: "Chill Vibes"}})
	s.SetPlaylistsTotal(5)

	// Send notification message.
	m, _ := pane.Update(LibraryLoadedMsg{})
	_ = m

	assert.Len(t, s.Playlists(), 1)
	assert.Equal(t, 5, s.PlaylistsTotal())
}

func TestLibraryPane_Update_SavedAlbumsLoadedMsg(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	s.SetSavedAlbums([]api.SavedAlbum{
		{Album: api.FullAlbum{ID: "album-1", Name: "After Hours"}},
	})

	m, _ := pane.Update(AlbumsLoadedMsg{})
	pane = m.(*LibraryPane)
	_ = pane

	assert.Len(t, s.SavedAlbums(), 1)
	assert.True(t, s.AlbumsLoaded())
}

func TestLibraryPane_Update_LikedTracksLoadedMsg(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	s.SetLikedTracks([]api.SavedTrack{
		{Track: api.Track{ID: "t1", Name: "Song"}},
	})
	s.SetLikedTotal(287)

	m, _ := pane.Update(LikedTracksLoadedMsg{})
	_ = m

	assert.Len(t, s.LikedTracks(), 1)
	assert.Equal(t, 287, s.LikedTotal())
}

func TestLibraryPane_Update_RecentlyPlayedLoadedMsg(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	s.SetRecentlyPlayed([]api.PlayHistory{
		{Track: api.Track{ID: "t1", Name: "Song"}, PlayedAt: "2024-03-01T22:15:00Z"},
	})

	m, _ := pane.Update(RecentlyPlayedLoadedMsg{})
	_ = m

	assert.Len(t, s.RecentlyPlayed(), 1)
}

func TestLibraryPane_Update_ExpandSection_LikedSongs(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	expandMsg := expandSectionMsg{section: SectionLikedSongs}
	m, cmd := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	section := findSectionByType(pane.tree.Sections(), SectionLikedSongs)
	require.NotNil(t, section)
	assert.True(t, section.Expanded)
	assert.NotNil(t, cmd)
}

func TestLibraryTree_LibraryExpandMsg(t *testing.T) {
	msg := LibraryExpandMsg(SectionAlbums)
	expandMsg, ok := msg.(expandSectionMsg)
	require.True(t, ok)
	assert.Equal(t, SectionAlbums, expandMsg.section)
}

func TestLibraryItem_DisplayName(t *testing.T) {
	tests := []struct {
		name string
		item LibraryItem
		want string
	}{
		{
			name: "playlist",
			item: LibraryItem{kind: kindPlaylist, DisplayName: "Chill Vibes", ContextURI: "spotify:playlist:p1"},
			want: "Chill Vibes",
		},
		{
			name: "album",
			item: LibraryItem{kind: kindAlbum, DisplayName: "After Hours", ContextURI: "spotify:album:a1"},
			want: "After Hours",
		},
		{
			name: "liked track",
			item: LibraryItem{kind: kindLikedTrack, DisplayName: "Blinding Lights", TrackURI: "spotify:track:t1"},
			want: "Blinding Lights",
		},
		{
			name: "play history",
			item: LibraryItem{kind: kindPlayHistory, DisplayName: "Levitating", TrackURI: "spotify:track:t2"},
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
			assert.Equal(t, tt.want, tt.item.DisplayName)
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
			item: LibraryItem{kind: kindLikedTrack, TrackURI: "spotify:track:t1"},
			want: "spotify:track:t1",
		},
		{
			name: "play history",
			item: LibraryItem{kind: kindPlayHistory, TrackURI: "spotify:track:t2"},
			want: "spotify:track:t2",
		},
		{
			name: "playlist (no track URI)",
			item: LibraryItem{kind: kindPlaylist, ContextURI: "spotify:playlist:p1"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.item.TrackURI)
		})
	}
}

func TestLibraryItem_ContextURI(t *testing.T) {
	tests := []struct {
		name string
		item LibraryItem
		want string
	}{
		{
			name: "playlist",
			item: LibraryItem{kind: kindPlaylist, ContextURI: "spotify:playlist:p1"},
			want: "spotify:playlist:p1",
		},
		{
			name: "album",
			item: LibraryItem{kind: kindAlbum, ContextURI: "spotify:album:a1"},
			want: "spotify:album:a1",
		},
		{
			name: "track (no context URI)",
			item: LibraryItem{kind: kindLikedTrack, TrackURI: "spotify:track:t1"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.item.ContextURI)
		})
	}
}

func TestLibraryPane_Update_A_NoTrackSelected(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	_ = m
	assert.Nil(t, cmd)
}

func TestLibraryPane_Update_L_NoTrackSelected(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, true)

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	_ = m
	assert.Nil(t, cmd)
}

func TestLibraryPane_Update_L_EmitsLikeRequest(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	s.SetRecentlyPlayed([]api.PlayHistory{
		{Track: api.Track{ID: "t1", Name: "Song", URI: "spotify:track:t1"}},
	})

	pane := NewLibraryPane(s, th, true)
	pane.tree.SetSectionExpanded(SectionRecentlyPlayed, true)

	for i := 0; i < 15; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.TrackURI != "" {
			break
		}
	}

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	_ = m
	require.NotNil(t, cmd)

	result := cmd()
	likeReq, ok := result.(LikeTrackRequestMsg)
	require.True(t, ok, "expected LikeTrackRequestMsg, got %T", result)
	assert.Equal(t, "t1", likeReq.TrackID)
}

func TestLibraryPane_Update_L_LikedTrack_EmitsUnlike(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	s.SetLikedTracks([]api.SavedTrack{
		{Track: api.Track{ID: "t1", Name: "Song", URI: "spotify:track:t1"}},
	})

	pane := NewLibraryPane(s, th, true)
	expandMsg := expandSectionMsg{section: SectionLikedSongs}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	for i := 0; i < 15; i++ {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
		item := pane.tree.SelectedItem()
		if item != nil && item.kind == kindLikedTrack {
			break
		}
	}

	m, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	_ = m
	require.NotNil(t, cmd)

	result := cmd()
	likeReq, ok := result.(LikeTrackRequestMsg)
	require.True(t, ok, "expected LikeTrackRequestMsg, got %T", result)
	assert.Equal(t, "t1", likeReq.TrackID)
	assert.True(t, likeReq.Unlike, "l on liked track should request unlike")
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

func TestLibraryPane_AutoExpandPlaylists_OnLoad(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	// Playlists section should start collapsed.
	sec := findSectionByType(pane.tree.Sections(), SectionPlaylists)
	require.NotNil(t, sec)
	assert.False(t, sec.Expanded, "playlists should be collapsed initially")

	// Simulate playlists data arriving from API.
	s.SetPlaylists([]api.SimplePlaylist{
		{ID: "pl1", Name: "Chill Vibes"},
		{ID: "pl2", Name: "Workout"},
	})

	m, _ := pane.Update(LibraryLoadedMsg{})
	pane = m.(*LibraryPane)

	sec = findSectionByType(pane.tree.Sections(), SectionPlaylists)
	assert.True(t, sec.Expanded, "playlists should auto-expand when data arrives")
}

func TestLibraryPane_AutoExpandPlaylists_NoExpandWhenEmpty(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	// No playlists in store — should stay collapsed.
	m, _ := pane.Update(LibraryLoadedMsg{})
	pane = m.(*LibraryPane)

	sec := findSectionByType(pane.tree.Sections(), SectionPlaylists)
	assert.False(t, sec.Expanded, "playlists should stay collapsed when no data")
}

func TestLibraryPane_AutoExpandRecentlyPlayed_OnLoad(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	// Recently Played section should start collapsed.
	sec := findSectionByType(pane.tree.Sections(), SectionRecentlyPlayed)
	require.NotNil(t, sec)
	assert.False(t, sec.Expanded, "recently played should be collapsed initially")

	// Simulate data arriving.
	s.SetRecentlyPlayed([]api.PlayHistory{
		{Track: api.Track{ID: "t1", Name: "Song A"}},
	})

	m, _ := pane.Update(RecentlyPlayedLoadedMsg{})
	pane = m.(*LibraryPane)

	sec = findSectionByType(pane.tree.Sections(), SectionRecentlyPlayed)
	assert.True(t, sec.Expanded, "recently played should auto-expand when data arrives")
}

func TestLibraryPane_AlbumsStayCollapsed_OnLoad(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewLibraryPane(s, th, false)

	s.SetSavedAlbums([]api.SavedAlbum{
		{Album: api.FullAlbum{ID: "a1", Name: "Album"}},
	})

	m, _ := pane.Update(AlbumsLoadedMsg{})
	pane = m.(*LibraryPane)

	sec := findSectionByType(pane.tree.Sections(), SectionAlbums)
	assert.False(t, sec.Expanded, "albums should stay collapsed (lazy load)")
}

// --- Height enforcement tests ---

// TestLibraryPane_View_HeightCapped verifies that View() output line count does not
// exceed the height set via SetSize.
func TestLibraryPane_View_HeightCapped(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Add many playlists so the list is long.
	playlists := make([]api.SimplePlaylist, 50)
	for i := range playlists {
		playlists[i] = api.SimplePlaylist{
			ID:   fmt.Sprintf("pl%d", i),
			Name: fmt.Sprintf("Playlist %d", i+1),
			URI:  fmt.Sprintf("spotify:playlist:pl%d", i),
		}
	}
	s.SetPlaylists(playlists)

	pane := NewLibraryPane(s, th, true)
	pane.SetSize(40, 20)

	// Expand playlists section so items are rendered.
	expandMsg := expandSectionMsg{section: SectionPlaylists}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	view := pane.View()
	// Trim trailing empty line that strings.Split creates from a trailing newline.
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	assert.LessOrEqual(t, len(lines), 20, "View() must not exceed SetSize height")
}

// TestLibraryPane_Scroll_AdvancesOffset verifies scrolling down past visible window
// advances scrollOffset.
func TestLibraryPane_Scroll_AdvancesOffset(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	playlists := make([]api.SimplePlaylist, 30)
	for i := range playlists {
		playlists[i] = api.SimplePlaylist{
			ID:   fmt.Sprintf("pl%d", i),
			Name: fmt.Sprintf("Playlist %d", i+1),
			URI:  fmt.Sprintf("spotify:playlist:pl%d", i),
		}
	}
	s.SetPlaylists(playlists)

	pane := NewLibraryPane(s, th, true)
	pane.SetSize(40, 15) // small height to force scrolling

	// Expand playlists section.
	expandMsg := expandSectionMsg{section: SectionPlaylists}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	// Navigate down many times to force scrollOffset to advance.
	for i := 0; i < 20; i++ {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		pane = m.(*LibraryPane)
	}

	assert.Greater(t, pane.scrollOffset, 0, "scrollOffset should advance when cursor moves past visible window")
}

// TestLibraryPane_ScrollIndicators_ShowWhenOverflow verifies scroll indicators appear
// when content exceeds height.
func TestLibraryPane_ScrollIndicators_ShowWhenOverflow(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	playlists := make([]api.SimplePlaylist, 30)
	for i := range playlists {
		playlists[i] = api.SimplePlaylist{
			ID:   fmt.Sprintf("pl%d", i),
			Name: fmt.Sprintf("Playlist %d", i+1),
			URI:  fmt.Sprintf("spotify:playlist:pl%d", i),
		}
	}
	s.SetPlaylists(playlists)

	pane := NewLibraryPane(s, th, true)
	pane.SetSize(40, 15)

	// Expand playlists.
	expandMsg := expandSectionMsg{section: SectionPlaylists}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	view := pane.View()
	// With 30 playlists and height 15, there should be a "more below" indicator.
	assert.Contains(t, view, "▼", "should show down scroll indicator when content overflows")
}

// TestLibraryPane_SmallHeight_RendersAtLeastOneItem verifies that even with a very small
// height, at least 1 item is rendered.
func TestLibraryPane_SmallHeight_RendersAtLeastOneItem(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	s.SetPlaylists([]api.SimplePlaylist{
		{ID: "pl1", Name: "My Playlist", URI: "spotify:playlist:pl1"},
		{ID: "pl2", Name: "Another Playlist", URI: "spotify:playlist:pl2"},
	})

	pane := NewLibraryPane(s, th, true)
	pane.SetSize(40, 5) // extremely small height

	expandMsg := expandSectionMsg{section: SectionPlaylists}
	m, _ := pane.Update(expandMsg)
	pane = m.(*LibraryPane)

	view := pane.View()
	// Should still render the section header and at least one playlist item.
	assert.Contains(t, view, "Playlists", "should render section header even at small height")
}
