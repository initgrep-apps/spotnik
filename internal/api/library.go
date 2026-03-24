package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// LibraryClient provides all Spotify library API calls: playlists, saved albums,
// liked tracks, recently played, and track like/unlike.
// It embeds BaseClient for shared HTTP functionality.
type LibraryClient struct {
	BaseClient
}

// NewLibraryClient constructs a LibraryClient using the given base URL and access token.
// Pass "" for baseURL to use the production Spotify API.
func NewLibraryClient(baseURL, accessToken string) *LibraryClient {
	return &LibraryClient{BaseClient: NewBaseClient(baseURL, accessToken)}
}

// SetHTTPClient overrides the default HTTP client used for API calls.
func (l *LibraryClient) SetHTTPClient(c *http.Client) {
	l.setHTTPClient(c)
}

// Playlists fetches the user's saved playlists via GET /me/playlists.
// Returns a slice of SimplePlaylist. Errors are wrapped with context.
func (l *LibraryClient) Playlists(ctx context.Context, limit, offset int) ([]SimplePlaylist, error) {
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

// PlaylistTracks fetches tracks for a specific playlist via GET /playlists/{id}/tracks.
// Returns a slice of Track. Errors are wrapped with context.
func (l *LibraryClient) PlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]Track, error) {
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

// SavedAlbums fetches the user's saved albums via GET /me/albums.
// Returns a slice of SavedAlbum. Errors are wrapped with context.
func (l *LibraryClient) SavedAlbums(ctx context.Context, limit, offset int) ([]SavedAlbum, error) {
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

// LikedTracks fetches the user's liked tracks via GET /me/tracks.
// Returns a slice of SavedTrack. Errors are wrapped with context.
func (l *LibraryClient) LikedTracks(ctx context.Context, limit, offset int) ([]SavedTrack, error) {
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

// RecentlyPlayed fetches recently played tracks via GET /me/player/recently-played.
// Returns a slice of PlayHistory items. Errors are wrapped with context.
func (l *LibraryClient) RecentlyPlayed(ctx context.Context, limit int) ([]PlayHistory, error) {
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
