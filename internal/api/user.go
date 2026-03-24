package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

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

// TopTracks fetches the user's top tracks via GET /me/top/tracks.
// timeRange must be "short_term", "medium_term", or "long_term".
// Returns a slice of Track. Errors are wrapped with context.
func (u *UserClient) TopTracks(ctx context.Context, timeRange string, limit int) ([]Track, error) {
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

// TopArtists fetches the user's top artists via GET /me/top/artists.
// timeRange must be "short_term", "medium_term", or "long_term".
// Returns a slice of FullArtist. Errors are wrapped with context.
func (u *UserClient) TopArtists(ctx context.Context, timeRange string, limit int) ([]FullArtist, error) {
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

// RecentlyPlayed fetches the user's recently played tracks via
// GET /me/player/recently-played. Returns a slice of PlayHistory items.
// Errors are wrapped with context.
func (u *UserClient) RecentlyPlayed(ctx context.Context, limit int) ([]PlayHistory, error) {
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
