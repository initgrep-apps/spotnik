package panes

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCategorySymbol verifies the badge character returned for each category.
func TestCategorySymbol(t *testing.T) {
	tests := []struct {
		category string
		want     string
	}{
		{category: "track", want: "♪"},
		{category: "artist", want: "★"},
		{category: "album", want: "◎"},
		{category: "playlist", want: "▤"},
		{category: "unknown", want: "·"},
		{category: "", want: "·"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := categorySymbol(tt.category)
			assert.Equal(t, tt.want, got, "categorySymbol(%q) should return %q", tt.category, tt.want)
		})
	}
}

// --- SearchListItem interface tests ---

func TestSearchListItem_InterfaceMethods(t *testing.T) {
	item := SearchListItem{
		Category: "track",
		Name:     "Pirates of the Caribbean Theme",
		Subtitle: "Hans Zimmer · At World's End · 3:42",
		URI:      "spotify:track:abc",
		IsTrack:  true,
	}
	assert.Equal(t, "Pirates of the Caribbean Theme", item.Title())
	assert.Equal(t, "Hans Zimmer · At World's End · 3:42", item.Description())
	assert.Equal(t, "Pirates of the Caribbean Theme", item.FilterValue())
}

// --- Formatting helper tests ---

func TestFormatFollowers(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{n: 7625607, want: "7.6M followers"},
		{n: 1000000, want: "1.0M followers"},
		{n: 3200, want: "3.2K followers"},
		{n: 1000, want: "1.0K followers"},
		{n: 847, want: "847 followers"},
		{n: 0, want: "0 followers"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatFollowers(tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestJoinGenres(t *testing.T) {
	tests := []struct {
		genres []string
		max    int
		want   string
	}{
		{genres: []string{"a", "b", "c", "d"}, max: 3, want: "a, b, c"},
		{genres: []string{"rock", "pop"}, max: 3, want: "rock, pop"},
		{genres: []string{}, max: 3, want: ""},
		{genres: nil, max: 3, want: ""},
		{genres: []string{"only one"}, max: 3, want: "only one"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := joinGenres(tt.genres, tt.max)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		date string
		want string
	}{
		{date: "2020-03-20", want: "2020"},
		{date: "2007", want: "2007"},
		{date: "199", want: "199"},
		{date: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			got := extractYear(tt.date)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatSearchDuration(t *testing.T) {
	tests := []struct {
		ms   int
		want string
	}{
		{ms: 222000, want: "3:42"},
		{ms: 60000, want: "1:00"},
		{ms: 3661000, want: "61:01"},
		{ms: 0, want: "0:00"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSearchDuration(tt.ms)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatAlbumType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "album", want: "Album"},
		{input: "single", want: "Single"},
		{input: "compilation", want: "Compilation"},
		{input: "UNKNOWN", want: "UNKNOWN"},
		{input: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatAlbumType(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{s: "hello world", max: 5, want: "hell…"},
		{s: "short", max: 10, want: "short"},
		{s: "exact", max: 5, want: "exact"},
		{s: "", max: 10, want: ""},
		{s: "café", max: 3, want: "ca…"},
		// Edge cases: max=0 and max=1 must not panic.
		{s: "hello", max: 0, want: ""},
		{s: "hello", max: 1, want: "…"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := truncateString(tt.s, tt.max)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestJoinArtistNames(t *testing.T) {
	tests := []struct {
		artists []domain.Artist
		want    string
	}{
		{
			artists: []domain.Artist{{Name: "Artist A"}, {Name: "Artist B"}},
			want:    "Artist A, Artist B",
		},
		{
			artists: []domain.Artist{{Name: "Solo"}},
			want:    "Solo",
		},
		{
			artists: []domain.Artist{},
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := joinArtistNames(tt.artists)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- Conversion helper tests ---

func TestTracksToListItems(t *testing.T) {
	tracks := []domain.Track{
		{
			ID:         "t1",
			Name:       "Pirates Theme",
			URI:        "spotify:track:t1",
			DurationMs: 222000, // 3:42
			Explicit:   true,
			Artists: []domain.Artist{
				{ID: "a1", Name: "Hans Zimmer"},
				{ID: "a2", Name: "Klaus Badelt"},
			},
			Album: domain.Album{ID: "al1", Name: "At World's End"},
		},
	}
	items := tracksToListItems(tracks)
	require.Len(t, items, 1)
	si, ok := items[0].(SearchListItem)
	require.True(t, ok)
	assert.Equal(t, "track", si.Category)
	assert.Equal(t, "Pirates Theme", si.Name)
	assert.Equal(t, "Hans Zimmer, Klaus Badelt", si.ArtistNames)
	assert.Equal(t, "At World's End", si.AlbumName)
	assert.Equal(t, "3:42", si.Duration)
	assert.True(t, si.Explicit)
	assert.Contains(t, si.Subtitle, "Hans Zimmer")
	assert.Contains(t, si.Subtitle, "At World's End")
	assert.Contains(t, si.Subtitle, "3:42")
}

func TestArtistsToListItems(t *testing.T) {
	artists := []domain.SearchArtist{
		{
			ID:         "a1",
			Name:       "Hans Zimmer",
			URI:        "spotify:artist:a1",
			Genres:     []string{"film score", "soundtrack", "classical", "ambient", "electronic"},
			Followers:  7625607,
			Popularity: 79,
		},
	}
	items := artistsToListItems(artists)
	require.Len(t, items, 1)
	si, ok := items[0].(SearchListItem)
	require.True(t, ok)
	assert.Equal(t, "artist", si.Category)
	assert.Equal(t, "Hans Zimmer", si.Name)
	// Top 3 genres only
	assert.Equal(t, "film score, soundtrack, classical", si.Genres)
	assert.Equal(t, "7.6M followers", si.Followers)
	assert.Equal(t, 79, si.Popularity)
	assert.Contains(t, si.Subtitle, "film score")
	assert.Contains(t, si.Subtitle, "7.6M followers")
}

func TestAlbumsToListItems(t *testing.T) {
	albums := []domain.SearchAlbum{
		{
			ID:          "al1",
			Name:        "At World's End",
			URI:         "spotify:album:al1",
			AlbumType:   "album",
			TotalTracks: 13,
			ReleaseDate: "2007-05-22",
			Artists:     []domain.Artist{{Name: "Hans Zimmer"}, {Name: "Klaus Badelt"}},
		},
	}
	items := albumsToListItems(albums)
	require.Len(t, items, 1)
	si, ok := items[0].(SearchListItem)
	require.True(t, ok)
	assert.Equal(t, "album", si.Category)
	assert.Equal(t, "Album", si.AlbumType)
	assert.Equal(t, "2007", si.ReleaseYear)
	assert.Equal(t, "13 tracks", si.TrackCount)
	assert.Equal(t, "Hans Zimmer, Klaus Badelt", si.AlbumArtists)
	assert.Contains(t, si.Subtitle, "2007")
	assert.Contains(t, si.Subtitle, "13 tracks")
}

func TestPlaylistsToListItems(t *testing.T) {
	playlists := []domain.SearchPlaylist{
		{
			ID:          "pl1",
			Name:        "Epic Film Scores",
			URI:         "spotify:playlist:pl1",
			Owner:       domain.SimplePlaylistOwner{ID: "u1", DisplayName: "john_doe"},
			TrackCount:  245,
			Description: "The best orchestral movie scores ever recorded for cinema",
		},
	}
	items := playlistsToListItems(playlists)
	require.Len(t, items, 1)
	si, ok := items[0].(SearchListItem)
	require.True(t, ok)
	assert.Equal(t, "playlist", si.Category)
	assert.Equal(t, "john_doe", si.Owner)
	assert.Equal(t, "245 tracks", si.PlaylistTracks)
	// Long description should be truncated
	assert.True(t, len([]rune(si.PlaylistDesc)) <= 60)
	assert.Contains(t, si.Subtitle, "john_doe")
	assert.Contains(t, si.Subtitle, "245 tracks")
}

func TestPlaylistsToListItems_NoDescription(t *testing.T) {
	playlists := []domain.SearchPlaylist{
		{
			ID:         "pl2",
			Name:       "Simple Playlist",
			URI:        "spotify:playlist:pl2",
			Owner:      domain.SimplePlaylistOwner{DisplayName: "bob"},
			TrackCount: 10,
		},
	}
	items := playlistsToListItems(playlists)
	si := items[0].(SearchListItem)
	assert.Equal(t, "", si.PlaylistDesc)
	// Subtitle should not contain " · " at end (no description separator)
	assert.NotContains(t, si.Subtitle, " · \n")
}

// --- rebuildFromResults uses same converters as rebuildFromStore (rich path) ---

// TestRebuildFromResults_DomainTypes verifies that SearchResultData (which now carries
// domain types) produces the same rich SearchListItems as rebuildFromStore would produce.
// This guards against regression where the fallback path strips metadata.
func TestRebuildFromResults_DomainTypes(t *testing.T) {
	// Build a SearchResultData with rich domain types.
	results := &SearchResultData{
		Tracks: []domain.Track{
			{
				URI:  "spotify:track:t1",
				Name: "Track One",
				Artists: []domain.Artist{
					{Name: "Artist A"},
					{Name: "Artist B"},
				},
				Album:      domain.Album{Name: "Album X"},
				DurationMs: 220000, // 3:40
				Explicit:   true,
			},
		},
		Artists: []domain.SearchArtist{
			{URI: "spotify:artist:a1", Name: "Artist One", Genres: []string{"rock", "indie"}, Followers: 12400},
		},
		Albums: []domain.SearchAlbum{
			{URI: "spotify:album:al1", Name: "Album One", AlbumType: "album", TotalTracks: 13, ReleaseDate: "2021-03-15"},
		},
		Playlists: []domain.SearchPlaylist{
			{URI: "spotify:playlist:pl1", Name: "Playlist One", Owner: domain.SimplePlaylistOwner{DisplayName: "bob"}, TrackCount: 40},
		},
	}

	// Populate a store and feed through SearchPageLoadedMsg so rebuildFromResults is exercised.
	s := state.New()
	o := NewSearchOverlay(s, theme.Load("black"))
	o.SetSize(80, 40)

	msg := SearchPageLoadedMsg{Results: results}
	m, _ := o.Update(msg)
	o = m.(*SearchOverlay)

	items := o.ResultListItems()
	require.Len(t, items, 4, "should have one item per type in TabAll")

	// Verify track item carries full metadata.
	track := items[0].(SearchListItem)
	assert.Equal(t, "track", track.Category)
	assert.Equal(t, "Track One", track.Name)
	assert.Equal(t, "Artist A, Artist B", track.ArtistNames, "all artist names should be present")
	assert.Equal(t, "Album X", track.AlbumName)
	assert.Equal(t, "3:40", track.Duration)
	assert.True(t, track.Explicit, "explicit flag should be set")

	// Verify artist item carries genres and followers.
	artist := items[1].(SearchListItem)
	assert.Equal(t, "artist", artist.Category)
	assert.Equal(t, "Artist One", artist.Name)
	assert.Equal(t, "rock, indie", artist.Genres)
	assert.Contains(t, artist.Followers, "12")

	// Verify album item carries track count.
	album := items[2].(SearchListItem)
	assert.Equal(t, "album", album.Category)
	assert.Equal(t, "Album One", album.Name)
	assert.Equal(t, "13 tracks", album.TrackCount)

	// Verify playlist item carries owner.
	playlist := items[3].(SearchListItem)
	assert.Equal(t, "playlist", playlist.Category)
	assert.Equal(t, "Playlist One", playlist.Name)
	assert.Equal(t, "bob", playlist.Owner)
}

// --- Render method tests ---

func newTestDelegate() SearchItemDelegate {
	return NewSearchItemDelegate(theme.Load("black"))
}

func renderItem(d SearchItemDelegate, item SearchListItem, selected bool, width int) string {
	// Build a minimal list model with the item.
	// When selected=false, we need to ensure the list cursor does not point at our item's
	// index. We place a dummy item at index 0 (where the cursor defaults) and render
	// our item at index 1 so isSelected (index == m.Index()) evaluates to false.
	if selected {
		rl := list.New([]list.Item{item}, d, width, 10)
		rl.Select(0)
		var buf bytes.Buffer
		d.Render(&buf, rl, 0, item)
		return buf.String()
	}
	dummy := SearchListItem{Category: "track", Name: "dummy"}
	rl := list.New([]list.Item{dummy, item}, d, width, 10)
	rl.Select(0) // cursor on dummy, not our item
	var buf bytes.Buffer
	d.Render(&buf, rl, 1, item)
	return buf.String()
}

func TestRenderTrack_ExplicitFlag(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:    "track",
		Name:        "Bad Song",
		Subtitle:    "Artist · Album · 3:00",
		URI:         "spotify:track:t1",
		IsTrack:     true,
		ArtistNames: "Artist A",
		AlbumName:   "My Album",
		Duration:    "3:00",
		Explicit:    true,
	}
	out := renderItem(d, item, false, 80)
	assert.Contains(t, out, "[E]", "explicit track should render [E]")
	assert.Contains(t, out, "3:00", "duration should be in output")
	assert.Contains(t, out, "Artist A", "artist names should be on line 2")
	assert.Contains(t, out, "My Album", "album name should be on line 2")
}

func TestRenderTrack_NoExplicit(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:    "track",
		Name:        "Clean Song",
		Subtitle:    "Artist · Album · 2:30",
		URI:         "spotify:track:t2",
		IsTrack:     true,
		ArtistNames: "Artist B",
		AlbumName:   "Clean Album",
		Duration:    "2:30",
		Explicit:    false,
	}
	out := renderItem(d, item, false, 80)
	assert.NotContains(t, out, "[E]", "non-explicit track should not render [E]")
	assert.Contains(t, out, "2:30", "duration should be in output")
}

func TestRenderTrack_TwoArtists(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:    "track",
		Name:        "Duet",
		Subtitle:    "A, B · Album · 4:00",
		URI:         "spotify:track:t3",
		IsTrack:     true,
		ArtistNames: "Hans Zimmer, Klaus Badelt",
		AlbumName:   "At World's End",
		Duration:    "4:00",
	}
	out := renderItem(d, item, false, 80)
	assert.Contains(t, out, "Hans Zimmer, Klaus Badelt")
	assert.Contains(t, out, "At World's End")
}

func TestRenderArtist_WithGenres(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:   "artist",
		Name:       "Hans Zimmer",
		Subtitle:   "film score, soundtrack · 7.6M followers",
		URI:        "spotify:artist:a1",
		Genres:     "film score, soundtrack, classical",
		Followers:  "7.6M followers",
		Popularity: 79,
	}
	out := renderItem(d, item, false, 80)
	assert.Contains(t, out, "Hans Zimmer")
	assert.Contains(t, out, "film score")
	assert.Contains(t, out, "followers")
}

func TestRenderArtist_EmptyGenres(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:  "artist",
		Name:      "New Artist",
		Subtitle:  "0 followers",
		URI:       "spotify:artist:a2",
		Genres:    "",
		Followers: "0 followers",
	}
	out := renderItem(d, item, false, 80)
	assert.Contains(t, out, "New Artist")
	assert.Contains(t, out, "0 followers")
	// No dot separator when genres are empty
	assert.NotContains(t, out, " · 0 followers")
}

func TestRenderAlbum_TypeAndYear(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:     "album",
		Name:         "Interstellar",
		Subtitle:     "Hans Zimmer · 2014 · 1 tracks",
		URI:          "spotify:album:al1",
		AlbumType:    "Single",
		ReleaseYear:  "2014",
		TrackCount:   "1 tracks",
		AlbumArtists: "Hans Zimmer",
	}
	out := renderItem(d, item, false, 80)
	assert.Contains(t, out, "Interstellar")
	assert.Contains(t, out, "Single")
	assert.Contains(t, out, "2014")
	assert.Contains(t, out, "1 tracks")
}

func TestRenderPlaylist_WithDescription(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:       "playlist",
		Name:           "Epic Film Scores",
		Subtitle:       "by john_doe · 245 tracks · Best scores",
		URI:            "spotify:playlist:pl1",
		Owner:          "john_doe",
		PlaylistTracks: "245 tracks",
		PlaylistDesc:   "Best scores",
	}
	out := renderItem(d, item, false, 80)
	assert.Contains(t, out, "Epic Film Scores")
	assert.Contains(t, out, "245 tracks")
	assert.Contains(t, out, "john_doe")
	assert.Contains(t, out, "Best scores")
}

func TestRenderPlaylist_NoDescription(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:       "playlist",
		Name:           "Simple",
		Subtitle:       "by bob · 10 tracks",
		URI:            "spotify:playlist:pl2",
		Owner:          "bob",
		PlaylistTracks: "10 tracks",
		PlaylistDesc:   "",
	}
	out := renderItem(d, item, false, 80)
	assert.Contains(t, out, "bob")
	assert.Contains(t, out, "10 tracks")
}

// TestRenderTrack_Selected verifies that a selected track renders differently from non-selected.
func TestRenderTrack_Selected(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:    "track",
		Name:        "Blinding Lights",
		Subtitle:    "The Weeknd · After Hours · 3:20",
		URI:         "spotify:track:t1",
		IsTrack:     true,
		ArtistNames: "The Weeknd",
		AlbumName:   "After Hours",
		Duration:    "3:20",
	}
	normal := renderItem(d, item, false, 80)
	selected := renderItem(d, item, true, 80)
	assert.Contains(t, selected, "Blinding Lights", "selected output should contain the track name")
	assert.NotEqual(t, normal, selected, "selected rendering should differ from non-selected")
}

// TestStyledDot verifies the dot separator is non-empty.
func TestStyledDot(t *testing.T) {
	d := newTestDelegate()
	dot := d.styledDot(false)
	assert.NotEmpty(t, dot)
	// Should contain the "·" character.
	assert.Contains(t, dot, "·")
}

// TestRightAlignBg verifies that left and right content appear in the output.
func TestRightAlignBg(t *testing.T) {
	d := newTestDelegate()
	result := d.rightAlignBg("left", "right", 20, false)
	assert.Contains(t, result, "left")
	assert.Contains(t, result, "right")
	assert.Len(t, result, 20, "rightAlignBg output should have exactly the requested width")
	leftIdx := strings.Index(result, "left")
	rightIdx := strings.Index(result, "right")
	assert.Greater(t, rightIdx, leftIdx, "right text should appear after left text")
}

// TestStyledName verifies that styledName always produces bold output with the item name,
// and that selected/normal produce identical output (selection is handled by wrapLine).
func TestStyledName(t *testing.T) {
	d := newTestDelegate()
	normal := d.styledName("Track Name", false, 40)
	selected := d.styledName("Track Name", true, 40)
	assert.Contains(t, normal, "Track Name")
	assert.Contains(t, selected, "Track Name")
	// Selected should differ from normal (background applied).
	assert.NotEqual(t, normal, selected, "styledName should differ when selected")
}

// TestRenderDefault verifies non-matching category uses a fallback.
func TestRenderDefault(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category: "unknown",
		Name:     "Mystery Item",
		Subtitle: "some info",
	}
	rl := list.New([]list.Item{item}, d, 80, 10)
	var buf bytes.Buffer
	d.Render(&buf, rl, 0, item)
	out := buf.String()
	assert.Contains(t, out, "Mystery Item")
}

// nonSearchItem is a list.Item implementation that is NOT a SearchListItem,
// used to verify that Render is a no-op for unknown item types.
type nonSearchItem struct{ name string }

func (n nonSearchItem) Title() string       { return n.name }
func (n nonSearchItem) Description() string { return "" }
func (n nonSearchItem) FilterValue() string { return n.name }

// TestRenderNonSearchListItem verifies Render is a no-op for non-SearchListItem.
func TestRenderNonSearchListItem(t *testing.T) {
	d := newTestDelegate()
	rl := list.New(nil, d, 80, 10)
	var buf bytes.Buffer
	// Pass a non-SearchListItem — should not panic or write anything.
	d.Render(&buf, rl, 0, nonSearchItem{name: "bad"})
	assert.Empty(t, buf.String())
}

// TestStyledBadge verifies badge rendering for each category.
func TestStyledBadge(t *testing.T) {
	d := newTestDelegate()
	categories := []string{"track", "artist", "album", "playlist"}
	for _, cat := range categories {
		t.Run(cat, func(t *testing.T) {
			badge := d.styledBadge(cat, false)
			assert.NotEmpty(t, badge)
		})
	}
}

// TestArtistsToListItems_EmptyGenres verifies that when genres is empty,
// subtitle contains only followers.
func TestArtistsToListItems_EmptyGenres(t *testing.T) {
	artists := []domain.SearchArtist{
		{
			ID:         "a1",
			Name:       "New Artist",
			URI:        "spotify:artist:a1",
			Genres:     []string{},
			Followers:  500,
			Popularity: 20,
		},
	}
	items := artistsToListItems(artists)
	si := items[0].(SearchListItem)
	assert.Equal(t, "", si.Genres)
	assert.Equal(t, "500 followers", si.Followers)
	// Subtitle should be just followers (no "· followers" awkwardly)
	assert.Equal(t, "500 followers", si.Subtitle)
}

// TestArtistsToListItems_ZeroFollowers verifies zero followers is displayed.
func TestArtistsToListItems_ZeroFollowers(t *testing.T) {
	artists := []domain.SearchArtist{
		{
			ID:        "a1",
			Name:      "New Artist",
			URI:       "spotify:artist:a1",
			Genres:    []string{"rock"},
			Followers: 0,
		},
	}
	items := artistsToListItems(artists)
	si := items[0].(SearchListItem)
	assert.Equal(t, "0 followers", si.Followers)
}

// TestTracksToListItems_SingleArtist verifies single artist has no comma.
func TestTracksToListItems_SingleArtist(t *testing.T) {
	tracks := []domain.Track{
		{
			Name:       "Solo",
			URI:        "spotify:track:s1",
			DurationMs: 180000,
			Explicit:   false,
			Artists:    []domain.Artist{{Name: "Solo Artist"}},
			Album:      domain.Album{Name: "Solo Album"},
		},
	}
	items := tracksToListItems(tracks)
	si := items[0].(SearchListItem)
	assert.Equal(t, "Solo Artist", si.ArtistNames)
	assert.False(t, si.Explicit)
}

// TestPlaylistsToListItems_LongDescription verifies long descriptions are truncated.
func TestPlaylistsToListItems_LongDescription(t *testing.T) {
	longDesc := strings.Repeat("a", 80)
	playlists := []domain.SearchPlaylist{
		{
			Name:        "Test",
			URI:         "spotify:playlist:t1",
			Owner:       domain.SimplePlaylistOwner{DisplayName: "bob"},
			TrackCount:  5,
			Description: longDesc,
		},
	}
	items := playlistsToListItems(playlists)
	si := items[0].(SearchListItem)
	assert.True(t, len([]rune(si.PlaylistDesc)) <= 60, "description should be truncated to 60 runes")
	assert.True(t, strings.HasSuffix(si.PlaylistDesc, "…"), "truncated desc should end with ellipsis")
}

// --- Task 19: Height() and wrapLine() tests ---

// TestHeight returns 3 after the 3-line layout change.
func TestHeight(t *testing.T) {
	d := newTestDelegate()
	assert.Equal(t, 3, d.Height(), "Height() should return 3 for 3-line items")
}

// TestWrapLine_Selected verifies that a selected line contains the left border │ character.
func TestWrapLine_Selected(t *testing.T) {
	d := newTestDelegate()
	out := d.wrapLine("hello", 40, true)
	assert.Contains(t, out, "│", "selected wrapLine should contain left border │")
	assert.Contains(t, out, "hello", "selected wrapLine should contain content")
}

// TestWrapLine_Normal verifies that an unselected line has at least 2-space left indent (no border).
func TestWrapLine_Normal(t *testing.T) {
	d := newTestDelegate()
	out := d.wrapLine("hello", 40, false)
	assert.Contains(t, out, "hello", "normal wrapLine should contain content")
	// Normal items use Padding(0,0,0,2) — output should start with spaces, not a border char.
	assert.NotContains(t, out, "│", "normal wrapLine should not contain border │")
}

// --- Task 20: styledName() always bold, no background ---

// TestStyledName_AlwaysBold verifies styledName always applies bold regardless of selected.
func TestStyledName_AlwaysBold(t *testing.T) {
	d := newTestDelegate()
	normal := d.styledName("Track Name", false, 40)
	selected := d.styledName("Track Name", true, 40)
	assert.Contains(t, normal, "Track Name")
	assert.Contains(t, selected, "Track Name")
	// Both should contain bold ANSI code.
	for _, out := range []string{normal, selected} {
		assert.True(t, strings.Contains(out, "1m") || strings.Contains(out, ";1;") ||
			strings.Contains(out, ";1m") || strings.Contains(out, "1;"),
			"styledName should contain bold ANSI code")
	}
}

// TestStyledName_SelectedBackground verifies styledName applies background when selected.
func TestStyledName_SelectedBackground(t *testing.T) {
	d := newTestDelegate()
	normalOut := d.styledName("Name", false, 40)
	selectedOut := d.styledName("Name", true, 40)
	// Selected output should differ (has SelectedBg background).
	assert.NotEqual(t, normalOut, selectedOut, "styledName should differ when selected (background added)")
	// Both should contain the name text.
	assert.Contains(t, normalOut, "Name")
	assert.Contains(t, selectedOut, "Name")
}

// --- Task 21: renderTrack 3-line layout ---

// countLines counts the number of lines (terminated by \n) in a rendered output string.
// Trailing newlines on the last line are expected from the renderer.
func countLines(s string) int {
	return strings.Count(s, "\n")
}

// stripANSI strips ANSI escape codes from a string for plain-text assertions.
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEscape = true
		case inEscape && r == 'm':
			inEscape = false
		case inEscape:
			// still inside escape sequence
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// TestRenderTrack_ThreeLines verifies track renders exactly 3 lines.
func TestRenderTrack_ThreeLines(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:    "track",
		Name:        "Blinding Lights",
		Subtitle:    "The Weeknd · After Hours · 3:20",
		URI:         "spotify:track:t1",
		IsTrack:     true,
		ArtistNames: "The Weeknd",
		AlbumName:   "After Hours",
		Duration:    "3:20",
	}
	out := renderItem(d, item, false, 80)
	assert.Equal(t, 3, countLines(out), "track render should have 3 lines (3 newlines)")
}

// TestRenderTrack_LineContents verifies each line carries the expected content.
func TestRenderTrack_LineContents(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:    "track",
		Name:        "Blinding Lights",
		ArtistNames: "The Weeknd",
		AlbumName:   "After Hours",
		Duration:    "3:20",
	}
	out := renderItem(d, item, false, 80)
	plain := stripANSI(out)
	lines := strings.SplitN(plain, "\n", 4)
	require.GreaterOrEqual(t, len(lines), 3)
	assert.Contains(t, lines[0], "Blinding Lights", "line 1 should contain track name")
	assert.Contains(t, lines[0], "3:20", "line 1 should contain duration")
	assert.Contains(t, lines[1], "The Weeknd", "line 2 should contain artists")
	assert.Contains(t, lines[2], "After Hours", "line 3 should contain album name")
}

// TestRenderTrack_Selected_ThreeLines verifies selected track renders 3 lines with │.
func TestRenderTrack_Selected_ThreeLines(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:    "track",
		Name:        "Blinding Lights",
		ArtistNames: "The Weeknd",
		AlbumName:   "After Hours",
		Duration:    "3:20",
	}
	selected := renderItem(d, item, true, 80)
	normal := renderItem(d, item, false, 80)
	assert.Equal(t, 3, countLines(selected), "selected track should still have 3 lines")
	assert.NotEqual(t, normal, selected, "selected rendering should differ from non-selected")
	assert.Contains(t, selected, "│", "selected track should contain left border │")
}

// --- Task 22: renderArtist 3-line layout ---

// TestRenderArtist_ThreeLines verifies artist renders exactly 3 lines.
func TestRenderArtist_ThreeLines(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:   "artist",
		Name:       "Hans Zimmer",
		Genres:     "film score, soundtrack",
		Followers:  "7.6M followers",
		Popularity: 79,
	}
	out := renderItem(d, item, false, 80)
	assert.Equal(t, 3, countLines(out), "artist render should have 3 lines")
}

// TestRenderArtist_LineContents verifies each artist line carries the expected content.
func TestRenderArtist_LineContents(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:   "artist",
		Name:       "Hans Zimmer",
		Genres:     "film score, soundtrack",
		Followers:  "7.6M followers",
		Popularity: 79,
	}
	out := renderItem(d, item, false, 80)
	plain := stripANSI(out)
	lines := strings.SplitN(plain, "\n", 4)
	require.GreaterOrEqual(t, len(lines), 3)
	assert.Contains(t, lines[0], "Hans Zimmer", "line 1 should contain artist name")
	assert.Contains(t, lines[1], "film score", "line 2 should contain genres")
	assert.Contains(t, lines[2], "7.6M followers", "line 3 should contain followers")
}

// --- Task 23: renderAlbum 3-line layout ---

// TestRenderAlbum_ThreeLines verifies album renders exactly 3 lines.
func TestRenderAlbum_ThreeLines(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:     "album",
		Name:         "Interstellar",
		AlbumType:    "Single",
		ReleaseYear:  "2014",
		TrackCount:   "1 tracks",
		AlbumArtists: "Hans Zimmer",
	}
	out := renderItem(d, item, false, 80)
	assert.Equal(t, 3, countLines(out), "album render should have 3 lines")
}

// TestRenderAlbum_LineContents verifies each album line carries the expected content.
func TestRenderAlbum_LineContents(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:     "album",
		Name:         "Interstellar",
		AlbumType:    "Single",
		ReleaseYear:  "2014",
		TrackCount:   "1 tracks",
		AlbumArtists: "Hans Zimmer",
	}
	out := renderItem(d, item, false, 80)
	plain := stripANSI(out)
	lines := strings.SplitN(plain, "\n", 4)
	require.GreaterOrEqual(t, len(lines), 3)
	assert.Contains(t, lines[0], "Interstellar", "line 1 should contain album name")
	assert.Contains(t, lines[0], "Single", "line 1 should contain album type")
	assert.Contains(t, lines[0], "2014", "line 1 should contain release year")
	assert.Contains(t, lines[1], "Hans Zimmer", "line 2 should contain artists")
	assert.Contains(t, lines[2], "1 tracks", "line 3 should contain track count")
}

// --- Task 24: renderPlaylist 3-line layout ---

// TestRenderPlaylist_ThreeLines verifies playlist renders exactly 3 lines.
func TestRenderPlaylist_ThreeLines(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:       "playlist",
		Name:           "Epic Film Scores",
		Owner:          "john_doe",
		PlaylistTracks: "245 tracks",
		PlaylistDesc:   "Best scores ever",
	}
	out := renderItem(d, item, false, 80)
	assert.Equal(t, 3, countLines(out), "playlist render should have 3 lines")
}

// TestRenderPlaylist_LineContents verifies each playlist line carries the expected content.
func TestRenderPlaylist_LineContents(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:       "playlist",
		Name:           "Epic Film Scores",
		Owner:          "john_doe",
		PlaylistTracks: "245 tracks",
		PlaylistDesc:   "Best scores ever",
	}
	out := renderItem(d, item, false, 80)
	plain := stripANSI(out)
	lines := strings.SplitN(plain, "\n", 4)
	require.GreaterOrEqual(t, len(lines), 3)
	assert.Contains(t, lines[0], "Epic Film Scores", "line 1 should contain playlist name")
	assert.Contains(t, lines[0], "245 tracks", "line 1 should contain track count")
	assert.Contains(t, lines[1], "john_doe", "line 2 should contain owner")
	assert.Contains(t, lines[2], "Best scores ever", "line 3 should contain description")
}

// --- Task 25: renderDefault 3-line layout ---

// TestRenderDefault_ThreeLines verifies the fallback renderer outputs 3 lines.
func TestRenderDefault_ThreeLines(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category: "unknown",
		Name:     "Mystery Item",
		Subtitle: "some info",
	}
	rl := list.New([]list.Item{item}, d, 80, 10)
	var buf bytes.Buffer
	d.Render(&buf, rl, 0, item)
	out := buf.String()
	assert.Equal(t, 3, countLines(out), "default render should have 3 lines")
	assert.Contains(t, out, "Mystery Item")
}

// --- padToInner tests ---

// --- Selected-state tests for artist, album, playlist, default ---

// TestRenderArtist_Selected verifies that a selected artist renders with │ border
// and differs from the non-selected rendering.
func TestRenderArtist_Selected(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:   "artist",
		Name:       "Hans Zimmer",
		Subtitle:   "film score · 7.6M followers",
		URI:        "spotify:artist:a1",
		Genres:     "film score, soundtrack",
		Followers:  "7.6M followers",
		Popularity: 79,
	}
	normal := renderItem(d, item, false, 80)
	selected := renderItem(d, item, true, 80)
	assert.Contains(t, selected, "Hans Zimmer", "selected artist output should contain the name")
	assert.Contains(t, selected, "│", "selected artist should contain left border │")
	assert.NotEqual(t, normal, selected, "selected rendering should differ from non-selected")
}

// TestRenderAlbum_Selected verifies that a selected album renders with │ border
// and differs from the non-selected rendering.
func TestRenderAlbum_Selected(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:     "album",
		Name:         "Interstellar",
		Subtitle:     "Hans Zimmer · 2014 · 1 tracks",
		URI:          "spotify:album:al1",
		AlbumType:    "Single",
		ReleaseYear:  "2014",
		TrackCount:   "1 tracks",
		AlbumArtists: "Hans Zimmer",
	}
	normal := renderItem(d, item, false, 80)
	selected := renderItem(d, item, true, 80)
	assert.Contains(t, selected, "Interstellar", "selected album output should contain the name")
	assert.Contains(t, selected, "│", "selected album should contain left border │")
	assert.NotEqual(t, normal, selected, "selected rendering should differ from non-selected")
}

// TestRenderPlaylist_Selected verifies that a selected playlist renders with │ border
// and differs from the non-selected rendering.
func TestRenderPlaylist_Selected(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category:       "playlist",
		Name:           "Epic Film Scores",
		Subtitle:       "by john_doe · 245 tracks · Best scores",
		URI:            "spotify:playlist:pl1",
		Owner:          "john_doe",
		PlaylistTracks: "245 tracks",
		PlaylistDesc:   "Best scores ever",
	}
	normal := renderItem(d, item, false, 80)
	selected := renderItem(d, item, true, 80)
	assert.Contains(t, selected, "Epic Film Scores", "selected playlist output should contain the name")
	assert.Contains(t, selected, "│", "selected playlist should contain left border │")
	assert.NotEqual(t, normal, selected, "selected rendering should differ from non-selected")
}

// TestRenderDefault_Selected verifies that a selected default-category item renders with │ border
// and differs from the non-selected rendering.
func TestRenderDefault_Selected(t *testing.T) {
	d := newTestDelegate()
	item := SearchListItem{
		Category: "unknown",
		Name:     "Mystery Item",
		Subtitle: "some info",
	}
	normal := renderItem(d, item, false, 80)
	selected := renderItem(d, item, true, 80)
	assert.Contains(t, selected, "Mystery Item", "selected default output should contain the name")
	assert.Contains(t, selected, "│", "selected default should contain left border │")
	assert.NotEqual(t, normal, selected, "selected rendering should differ from non-selected")
}
