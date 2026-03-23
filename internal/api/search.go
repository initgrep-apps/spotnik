package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

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
	if err := unmarshalJSON(data, raw); err != nil {
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

// SearchClient provides the Spotify search API call.
// It shares the same base URL pattern as Player and LibraryClient —
// pass "" for baseURL to use the production Spotify API.
type SearchClient struct {
	baseURL     string
	accessToken string
	client      *http.Client
}

// NewSearchClient constructs a SearchClient using the given base URL and access token.
// Pass "" for baseURL to use the production Spotify API.
func NewSearchClient(baseURL, accessToken string) *SearchClient {
	base := baseURL
	if base == "" {
		base = spotifyAPIBaseURL
	}
	return &SearchClient{
		baseURL:     strings.TrimRight(base, "/"),
		accessToken: accessToken,
		client:      &http.Client{},
	}
}

// Search calls GET /v1/search with the given query, types, and per-type limit.
// Always includes market=from_token per Spotify API recommendations.
// types should contain one or more of: "track", "artist", "album", "playlist".
// Returns a fully populated SearchResult on success.
func (s *SearchClient) Search(ctx context.Context, query string, types []string, limit int) (*SearchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/v1/search", nil)
	if err != nil {
		return nil, fmt.Errorf("creating search request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.accessToken)

	q := req.URL.Query()
	q.Set("q", query)
	q.Set("type", strings.Join(types, ","))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("market", "from_token")
	req.URL.RawQuery = q.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending search request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("429 rate limited: retry after %s seconds", retryAfter)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s %s: unexpected status %d: %s", req.Method, req.URL.Path, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading search response body: %w", err)
	}

	var result SearchResult
	if err := unmarshalJSON(body, &result); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}

	return &result, nil
}
