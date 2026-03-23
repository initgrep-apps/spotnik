package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// LibraryClient provides all Spotify library API calls: playlists, saved albums,
// liked tracks, recently played, and track like/unlike.
// It shares the same base URL pattern as Player — pass "" for production Spotify.
type LibraryClient struct {
	baseURL     string
	accessToken string
	client      *http.Client
}

// NewLibraryClient constructs a LibraryClient using the given base URL and access token.
// Pass "" for baseURL to use the production Spotify API.
func NewLibraryClient(baseURL, accessToken string) *LibraryClient {
	base := baseURL
	if base == "" {
		base = spotifyAPIBaseURL
	}
	return &LibraryClient{
		baseURL:     strings.TrimRight(base, "/"),
		accessToken: accessToken,
		client:      &http.Client{},
	}
}

// SetHTTPClient overrides the default HTTP client used for API calls.
func (l *LibraryClient) SetHTTPClient(c *http.Client) {
	l.client = c
}

// GetPlaylists fetches the user's saved playlists via GET /me/playlists.
// Returns a slice of SimplePlaylist. Errors are wrapped with context.
func (l *LibraryClient) GetPlaylists(ctx context.Context, limit, offset int) ([]SimplePlaylist, error) {
	req, err := l.newRequest(ctx, http.MethodGet, "/v1/me/playlists", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get playlists request: %w", err)
	}

	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()

	var response struct {
		Items []SimplePlaylist `json:"items"`
	}
	if err := l.doJSON(req, &response); err != nil {
		return nil, fmt.Errorf("getting playlists: %w", err)
	}
	if response.Items == nil {
		return []SimplePlaylist{}, nil
	}
	return response.Items, nil
}

// GetPlaylistTracks fetches tracks for a specific playlist via GET /playlists/{id}/tracks.
// Returns a slice of Track. Errors are wrapped with context.
func (l *LibraryClient) GetPlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]Track, error) {
	path := fmt.Sprintf("/v1/playlists/%s/tracks", playlistID)
	req, err := l.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating get playlist tracks request: %w", err)
	}

	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()

	var response struct {
		Items []struct {
			Track Track `json:"track"`
		} `json:"items"`
	}
	if err := l.doJSON(req, &response); err != nil {
		return nil, fmt.Errorf("getting playlist tracks: %w", err)
	}

	tracks := make([]Track, 0, len(response.Items))
	for _, item := range response.Items {
		tracks = append(tracks, item.Track)
	}
	return tracks, nil
}

// GetSavedAlbums fetches the user's saved albums via GET /me/albums.
// Returns a slice of SavedAlbum. Errors are wrapped with context.
func (l *LibraryClient) GetSavedAlbums(ctx context.Context, limit, offset int) ([]SavedAlbum, error) {
	req, err := l.newRequest(ctx, http.MethodGet, "/v1/me/albums", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get saved albums request: %w", err)
	}

	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()

	var response struct {
		Items []SavedAlbum `json:"items"`
	}
	if err := l.doJSON(req, &response); err != nil {
		return nil, fmt.Errorf("getting saved albums: %w", err)
	}
	if response.Items == nil {
		return []SavedAlbum{}, nil
	}
	return response.Items, nil
}

// GetLikedTracks fetches the user's liked tracks via GET /me/tracks.
// Returns a slice of SavedTrack. Errors are wrapped with context.
func (l *LibraryClient) GetLikedTracks(ctx context.Context, limit, offset int) ([]SavedTrack, error) {
	req, err := l.newRequest(ctx, http.MethodGet, "/v1/me/tracks", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get liked tracks request: %w", err)
	}

	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()

	var response struct {
		Items []SavedTrack `json:"items"`
	}
	if err := l.doJSON(req, &response); err != nil {
		return nil, fmt.Errorf("getting liked tracks: %w", err)
	}
	if response.Items == nil {
		return []SavedTrack{}, nil
	}
	return response.Items, nil
}

// GetRecentlyPlayed fetches recently played tracks via GET /me/player/recently-played.
// Returns a slice of PlayHistory items. Errors are wrapped with context.
func (l *LibraryClient) GetRecentlyPlayed(ctx context.Context, limit int) ([]PlayHistory, error) {
	req, err := l.newRequest(ctx, http.MethodGet, "/v1/me/player/recently-played", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get recently played request: %w", err)
	}

	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	var response struct {
		Items []PlayHistory `json:"items"`
	}
	if err := l.doJSON(req, &response); err != nil {
		return nil, fmt.Errorf("getting recently played: %w", err)
	}
	if response.Items == nil {
		return []PlayHistory{}, nil
	}
	return response.Items, nil
}

// LikeTrack adds the given track to the user's liked songs via PUT /me/tracks.
// Errors are wrapped with context.
func (l *LibraryClient) LikeTrack(ctx context.Context, trackID string) error {
	body, err := json.Marshal(map[string][]string{"ids": {trackID}})
	if err != nil {
		return fmt.Errorf("marshaling like track body: %w", err)
	}

	req, err := l.newRequest(ctx, http.MethodPut, "/v1/me/tracks", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating like track request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return l.doNoContent(req)
}

// UnlikeTrack removes the given track from the user's liked songs via DELETE /me/tracks.
// Errors are wrapped with context.
func (l *LibraryClient) UnlikeTrack(ctx context.Context, trackID string) error {
	body, err := json.Marshal(map[string][]string{"ids": {trackID}})
	if err != nil {
		return fmt.Errorf("marshaling unlike track body: %w", err)
	}

	req, err := l.newRequest(ctx, http.MethodDelete, "/v1/me/tracks", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating unlike track request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return l.doNoContent(req)
}

// newRequest builds an HTTP request with the Authorization header set.
func (l *LibraryClient) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	u := l.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+l.accessToken)
	return req, nil
}

// doJSON executes req, checks for a 2xx status, then decodes the JSON body into out.
func (l *LibraryClient) doJSON(req *http.Request, out interface{}) error {
	resp, err := l.client.Do(req)
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

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	return nil
}

// doNoContent executes req and returns nil if the response is 2xx.
// Returns an error for non-2xx responses.
func (l *LibraryClient) doNoContent(req *http.Request) error {
	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: unexpected status %d: %s", req.Method, req.URL.Path, resp.StatusCode, string(body))
	}

	return nil
}
