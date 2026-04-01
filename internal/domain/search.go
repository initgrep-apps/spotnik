package domain

import "encoding/json"

// SearchArtist represents an artist returned in a search result.
type SearchArtist struct {
	// ID is the Spotify artist ID.
	ID string `json:"id"`

	// Name is the display name of the artist.
	Name string `json:"name"`

	// URI is the Spotify URI of the artist.
	URI string `json:"uri"`
}

// SearchAlbum represents an album returned in a search result.
type SearchAlbum struct {
	// ID is the Spotify album ID.
	ID string `json:"id"`

	// Name is the display name of the album.
	Name string `json:"name"`

	// URI is the Spotify URI of the album.
	URI string `json:"uri"`

	// TotalTracks is the total number of tracks in the album.
	TotalTracks int `json:"total_tracks"`

	// ReleaseDate is the release date string (e.g. "2020-03-20").
	ReleaseDate string `json:"release_date"`

	// Artists is the list of artists for this album.
	Artists []Artist `json:"artists"`
}

// SearchPlaylist represents a playlist returned in a search result.
// NOTE: SearchPlaylist uses the same nested tracks.total pattern as SimplePlaylist
// but is a separate type to avoid coupling search results to library types.
type SearchPlaylist struct {
	// ID is the Spotify playlist ID.
	ID string `json:"id"`

	// Name is the display name of the playlist.
	Name string `json:"name"`

	// URI is the Spotify URI of the playlist.
	URI string `json:"uri"`

	// Owner is the playlist owner.
	Owner SimplePlaylistOwner `json:"owner"`

	// TrackCount is the total number of tracks in the playlist.
	// Populated from the nested "tracks.total" field in the Spotify response.
	TrackCount int `json:"-"`
}

// UnmarshalJSON implements custom unmarshaling to extract the nested tracks.total
// into the flat TrackCount field, mirroring the pattern used by SimplePlaylist.
func (p *SearchPlaylist) UnmarshalJSON(data []byte) error {
	raw := &struct {
		ID     string              `json:"id"`
		Name   string              `json:"name"`
		URI    string              `json:"uri"`
		Owner  SimplePlaylistOwner `json:"owner"`
		Tracks struct {
			Total int `json:"total"`
		} `json:"tracks"`
	}{}
	if err := json.Unmarshal(data, raw); err != nil {
		return err
	}
	p.ID = raw.ID
	p.Name = raw.Name
	p.URI = raw.URI
	p.Owner = raw.Owner
	p.TrackCount = raw.Tracks.Total
	return nil
}

// SearchTracksResult holds the tracks section of a search response.
type SearchTracksResult struct {
	// Items is the list of matching tracks.
	Items []Track `json:"items"`

	// Total is the total number of matching tracks.
	Total int `json:"total"`
}

// SearchArtistsResult holds the artists section of a search response.
type SearchArtistsResult struct {
	// Items is the list of matching artists.
	Items []SearchArtist `json:"items"`

	// Total is the total number of matching artists.
	Total int `json:"total"`
}

// SearchAlbumsResult holds the albums section of a search response.
type SearchAlbumsResult struct {
	// Items is the list of matching albums.
	Items []SearchAlbum `json:"items"`

	// Total is the total number of matching albums.
	Total int `json:"total"`
}

// SearchPlaylistsResult holds the playlists section of a search response.
type SearchPlaylistsResult struct {
	// Items is the list of matching playlists.
	Items []SearchPlaylist `json:"items"`

	// Total is the total number of matching playlists.
	Total int `json:"total"`
}

// SearchResult contains results for all four search categories returned by the Spotify search API.
type SearchResult struct {
	// Tracks contains the matching tracks.
	Tracks SearchTracksResult `json:"tracks"`

	// Artists contains the matching artists.
	Artists SearchArtistsResult `json:"artists"`

	// Albums contains the matching albums.
	Albums SearchAlbumsResult `json:"albums"`

	// Playlists contains the matching playlists.
	Playlists SearchPlaylistsResult `json:"playlists"`
}

// SearchTrackItem holds the display fields for a single track search result.
// Defined here so both state/ and ui/panes/ can reference it without an import cycle.
type SearchTrackItem struct {
	// URI is the Spotify track URI used for playback and queue commands.
	URI string
	// Name is the track display name.
	Name string
	// Artist is the pre-formatted first artist name.
	Artist string
	// Album is the album name for this track.
	Album string
	// DurationMs is the track duration in milliseconds.
	DurationMs int
}

// SearchArtistItem holds the display fields for a single artist search result.
// Defined here so both state/ and ui/panes/ can reference it without an import cycle.
type SearchArtistItem struct {
	// URI is the Spotify artist URI used for playback context.
	URI string
	// Name is the artist display name.
	Name string
}

// SearchAlbumItem holds the display fields for a single album search result.
// Defined here so both state/ and ui/panes/ can reference it without an import cycle.
type SearchAlbumItem struct {
	// URI is the Spotify album URI used for playback context.
	URI string
	// Name is the album display name.
	Name string
	// Artist is the pre-formatted first artist name.
	Artist string
	// ReleaseYear is the 4-character year extracted from the release date string.
	ReleaseYear string
	// TotalTracks is the total number of tracks in the album.
	TotalTracks int
}

// SearchPlaylistItem holds the display fields for a single playlist search result.
// Defined here so both state/ and ui/panes/ can reference it without an import cycle.
type SearchPlaylistItem struct {
	// URI is the Spotify playlist URI used for playback context.
	URI string
	// Name is the playlist display name.
	Name string
	// Owner is the pre-formatted playlist owner display name.
	Owner string
	// TrackCount is the total number of tracks in the playlist.
	TrackCount int
}
