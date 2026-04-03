package domain

import "encoding/json"

// SearchArtist represents an artist returned in a search result.
// NOTE: UnmarshalJSON is required because the Spotify API nests follower count
// under "followers.total" rather than exposing it as a flat field.
type SearchArtist struct {
	// ID is the Spotify artist ID.
	ID string `json:"id"`

	// Name is the display name of the artist.
	Name string `json:"name"`

	// URI is the Spotify URI of the artist.
	URI string `json:"uri"`

	// Genres is the list of musical genres associated with this artist.
	Genres []string `json:"genres"`

	// Followers is the total follower count for this artist.
	// Populated from the nested "followers.total" field in the Spotify response.
	Followers int `json:"-"`

	// Popularity is the artist's popularity score (0–100).
	Popularity int `json:"popularity"`
}

// UnmarshalJSON implements custom unmarshaling to extract the nested followers.total
// into the flat Followers field.
func (a *SearchArtist) UnmarshalJSON(data []byte) error {
	raw := &struct {
		ID         string   `json:"id"`
		Name       string   `json:"name"`
		URI        string   `json:"uri"`
		Genres     []string `json:"genres"`
		Popularity int      `json:"popularity"`
		Followers  struct {
			Total int `json:"total"`
		} `json:"followers"`
	}{}
	if err := json.Unmarshal(data, raw); err != nil {
		return err
	}
	a.ID = raw.ID
	a.Name = raw.Name
	a.URI = raw.URI
	a.Genres = raw.Genres
	a.Popularity = raw.Popularity
	a.Followers = raw.Followers.Total
	return nil
}

// SearchAlbum represents an album returned in a search result.
type SearchAlbum struct {
	// ID is the Spotify album ID.
	ID string `json:"id"`

	// Name is the display name of the album.
	Name string `json:"name"`

	// URI is the Spotify URI of the album.
	URI string `json:"uri"`

	// AlbumType is the type of album: "album", "single", or "compilation".
	AlbumType string `json:"album_type"`

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

	// Description is the playlist description text.
	Description string `json:"-"`
}

// UnmarshalJSON implements custom unmarshaling to extract the nested tracks.total
// into the flat TrackCount field, mirroring the pattern used by SimplePlaylist.
func (p *SearchPlaylist) UnmarshalJSON(data []byte) error {
	raw := &struct {
		ID          string              `json:"id"`
		Name        string              `json:"name"`
		URI         string              `json:"uri"`
		Owner       SimplePlaylistOwner `json:"owner"`
		Description string              `json:"description"`
		Tracks      struct {
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
	p.Description = raw.Description
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
