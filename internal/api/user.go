package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
// top artists, and recently played history. It uses the same base URL pattern
// as Player and LibraryClient — pass "" for the production Spotify API.
type UserClient struct {
	baseURL     string
	accessToken string
	client      *http.Client
}

// NewUserClient constructs a UserClient using the given base URL and access token.
// Pass "" for baseURL to use the production Spotify API.
func NewUserClient(baseURL, accessToken string) *UserClient {
	base := baseURL
	if base == "" {
		base = spotifyAPIBaseURL
	}
	return &UserClient{
		baseURL:     strings.TrimRight(base, "/"),
		accessToken: accessToken,
		client:      &http.Client{},
	}
}

// SetHTTPClient overrides the default HTTP client used for API calls.
func (u *UserClient) SetHTTPClient(c *http.Client) {
	u.client = c
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

// newRequest builds an HTTP request with the Authorization header set.
func (u *UserClient) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := u.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+u.accessToken)
	return req, nil
}

// doJSON executes req, checks for a 2xx status, then decodes the JSON body into out.
func (u *UserClient) doJSON(req *http.Request, out interface{}) error {
	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		return fmt.Errorf("429 rate limited: retry after %s seconds", retryAfter)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: unexpected status %d: %s", req.Method, req.URL.Path, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if err := unmarshalJSON(body, out); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	return nil
}
