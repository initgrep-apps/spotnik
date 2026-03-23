package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// PlaylistsClient provides Spotify playlist mutation API calls:
// create, rename, add/remove tracks, and reorder tracks.
// It uses the same base URL pattern as LibraryClient — pass "" for production Spotify.
type PlaylistsClient struct {
	baseURL     string
	accessToken string
	client      *http.Client
}

// NewPlaylistsClient constructs a PlaylistsClient using the given base URL and access token.
// Pass "" for baseURL to use the production Spotify API.
func NewPlaylistsClient(baseURL, accessToken string) *PlaylistsClient {
	base := baseURL
	if base == "" {
		base = spotifyAPIBaseURL
	}
	return &PlaylistsClient{
		baseURL:     strings.TrimRight(base, "/"),
		accessToken: accessToken,
		client:      &http.Client{},
	}
}

// CreatePlaylist creates a new playlist for the current user via POST /me/playlists.
// Returns the created SimplePlaylist on success. Errors are wrapped with context.
func (p *PlaylistsClient) CreatePlaylist(ctx context.Context, name, description string, public bool) (*SimplePlaylist, error) {
	reqBody := map[string]interface{}{
		"name":        name,
		"description": description,
		"public":      public,
	}
	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling create playlist body: %w", err)
	}

	req, err := p.newRequest(ctx, http.MethodPost, "/v1/me/playlists", bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("creating create playlist request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var playlist SimplePlaylist
	if err := p.doJSON(req, &playlist); err != nil {
		return nil, fmt.Errorf("creating playlist: %w", err)
	}
	return &playlist, nil
}

// UpdatePlaylist renames a playlist and updates its description via PUT /playlists/{id}.
// Errors are wrapped with context.
func (p *PlaylistsClient) UpdatePlaylist(ctx context.Context, id, name, description string) error {
	reqBody := map[string]interface{}{
		"name":        name,
		"description": description,
	}
	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling update playlist body: %w", err)
	}

	req, err := p.newRequest(ctx, http.MethodPut, fmt.Sprintf("/v1/playlists/%s", id), bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("creating update playlist request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return p.doNoContent(req)
}

// AddTracksToPlaylist adds one or more tracks to a playlist via POST /playlists/{id}/items.
// uris should be Spotify track URIs (e.g. "spotify:track:..."). Errors are wrapped with context.
func (p *PlaylistsClient) AddTracksToPlaylist(ctx context.Context, playlistID string, uris []string) error {
	reqBody := map[string]interface{}{
		"uris": uris,
	}
	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling add tracks body: %w", err)
	}

	path := fmt.Sprintf("/v1/playlists/%s/tracks", playlistID)
	req, err := p.newRequest(ctx, http.MethodPost, path, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("creating add tracks request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// The Spotify API returns 201 with a snapshot_id — we use doJSON to consume the body.
	var result struct {
		SnapshotID string `json:"snapshot_id"`
	}
	if err := p.doJSON(req, &result); err != nil {
		return fmt.Errorf("adding tracks to playlist: %w", err)
	}
	return nil
}

// RemoveTracksFromPlaylist removes one or more tracks from a playlist
// via DELETE /playlists/{id}/items. uris should be Spotify track URIs.
// Errors are wrapped with context.
func (p *PlaylistsClient) RemoveTracksFromPlaylist(ctx context.Context, playlistID string, uris []string) error {
	type trackItem struct {
		URI string `json:"uri"`
	}
	tracks := make([]trackItem, len(uris))
	for i, u := range uris {
		tracks[i] = trackItem{URI: u}
	}
	reqBody := map[string]interface{}{
		"tracks": tracks,
	}
	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling remove tracks body: %w", err)
	}

	path := fmt.Sprintf("/v1/playlists/%s/tracks", playlistID)
	req, err := p.newRequest(ctx, http.MethodDelete, path, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("creating remove tracks request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// The Spotify API returns 200 with a snapshot_id.
	var result struct {
		SnapshotID string `json:"snapshot_id"`
	}
	if err := p.doJSON(req, &result); err != nil {
		return fmt.Errorf("removing tracks from playlist: %w", err)
	}
	return nil
}

// ReorderPlaylistTracks moves a range of tracks in a playlist
// via PUT /playlists/{id}/tracks with range_start, insert_before, and range_length.
// Errors are wrapped with context.
func (p *PlaylistsClient) ReorderPlaylistTracks(ctx context.Context, id string, rangeStart, insertBefore, rangeLength int) error {
	reqBody := map[string]interface{}{
		"range_start":   rangeStart,
		"insert_before": insertBefore,
		"range_length":  rangeLength,
	}
	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling reorder tracks body: %w", err)
	}

	path := fmt.Sprintf("/v1/playlists/%s/tracks", id)
	req, err := p.newRequest(ctx, http.MethodPut, path, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("creating reorder tracks request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var result struct {
		SnapshotID string `json:"snapshot_id"`
	}
	if err := p.doJSON(req, &result); err != nil {
		return fmt.Errorf("reordering playlist tracks: %w", err)
	}
	return nil
}

// newRequest builds an HTTP request with the Authorization header set.
func (p *PlaylistsClient) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	u := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	return req, nil
}

// doJSON executes req, checks for a 2xx status, then decodes the JSON body into out.
func (p *PlaylistsClient) doJSON(req *http.Request, out interface{}) error {
	resp, err := p.client.Do(req)
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
func (p *PlaylistsClient) doNoContent(req *http.Request) error {
	resp, err := p.client.Do(req)
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
