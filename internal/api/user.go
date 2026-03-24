package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// FullArtist represents a Spotify artist with full details, as returned by
// the top-artists and search endpoints. It extends the basic Artist struct
// with genre, popularity, and external URL fields.
type FullArtist struct {
	// ID is the Spotify artist ID.
	ID string `json:"id"`

	// Name is the display name of the artist.
	Name string `json:"name"`

	// URI is the Spotify URI of the artist (e.g. "spotify:artist:...").
	URI string `json:"uri"`

	// Genres is the list of musical genres associated with this artist.
	Genres []string `json:"genres"`

	// Popularity is the artist's popularity score (0–100).
	Popularity int `json:"popularity"`

	// ExternalURLs maps service name to external URL (e.g. "spotify" → URL).
	ExternalURLs map[string]string `json:"external_urls"`
}

// UserClient provides Spotify API calls for user-specific data: top tracks,
// top artists, and recently played history. It embeds BaseClient for shared
// HTTP functionality.
type UserClient struct {
	BaseClient
}

// NewUserClient constructs a UserClient using the given base URL and access token.
// Pass "" for baseURL to use the production Spotify API.
func NewUserClient(baseURL, accessToken string) *UserClient {
	return &UserClient{BaseClient: NewBaseClient(baseURL, accessToken)}
}

// SetHTTPClient overrides the default HTTP client used for API calls.
func (u *UserClient) SetHTTPClient(c *http.Client) {
	u.setHTTPClient(c)
}

// GetTopTracks fetches the user's top tracks via GET /me/top/tracks.
// timeRange must be "short_term", "medium_term", or "long_term".
// Returns a slice of Track. Errors are wrapped with context.
func (u *UserClient) GetTopTracks(ctx context.Context, timeRange string, limit int) ([]Track, error) {
	req, err := u.newRequest(ctx, http.MethodGet, "/v1/me/top/tracks", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get top tracks request: %w", err)
	}

	q := req.URL.Query()
	q.Set("time_range", timeRange)
	q.Set("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	var response struct {
		Items []Track `json:"items"`
	}
	if err := u.doJSON(req, &response); err != nil {
		return nil, fmt.Errorf("getting top tracks: %w", err)
	}
	if response.Items == nil {
		return []Track{}, nil
	}
	return response.Items, nil
}

// GetTopArtists fetches the user's top artists via GET /me/top/artists.
// timeRange must be "short_term", "medium_term", or "long_term".
// Returns a slice of FullArtist. Errors are wrapped with context.
func (u *UserClient) GetTopArtists(ctx context.Context, timeRange string, limit int) ([]FullArtist, error) {
	req, err := u.newRequest(ctx, http.MethodGet, "/v1/me/top/artists", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get top artists request: %w", err)
	}

	q := req.URL.Query()
	q.Set("time_range", timeRange)
	q.Set("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	var response struct {
		Items []FullArtist `json:"items"`
	}
	if err := u.doJSON(req, &response); err != nil {
		return nil, fmt.Errorf("getting top artists: %w", err)
	}
	if response.Items == nil {
		return []FullArtist{}, nil
	}
	return response.Items, nil
}

// GetRecentlyPlayed fetches the user's recently played tracks via
// GET /me/player/recently-played. Returns a slice of PlayHistory items.
// Errors are wrapped with context.
func (u *UserClient) GetRecentlyPlayed(ctx context.Context, limit int) ([]PlayHistory, error) {
	req, err := u.newRequest(ctx, http.MethodGet, "/v1/me/player/recently-played", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get recently played request: %w", err)
	}

	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	var response struct {
		Items []PlayHistory `json:"items"`
	}
	if err := u.doJSON(req, &response); err != nil {
		return nil, fmt.Errorf("getting recently played: %w", err)
	}
	if response.Items == nil {
		return []PlayHistory{}, nil
	}
	return response.Items, nil
}
