package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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

// GetPlaylist fetches a playlist's first page of items via GET /playlists/{id}.
// NOTE: Spotify only embeds the items container in this response for playlists owned
// by the authenticated user. For non-owned (followed) playlists the response contains
// only metadata and no items. Use PlaylistTracks (GET /playlists/{id}/items) when
// ownership is unknown or the playlist is not owned by the user.
// Local files, null track objects, and podcast episodes are skipped.
// Returns tracks, total track count, whether a next page exists, and any error.
func (l *LibraryClient) GetPlaylist(ctx context.Context, playlistID string) ([]Track, int, bool, error) {
	path := fmt.Sprintf("/v1/playlists/%s", playlistID)
	req, err := l.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, false, fmt.Errorf("creating get playlist request: %w", err)
	}

	// NOTE: Spotify renamed "tracks" → "items" and "track" → "item" in February 2026.
	// The container field and per-item track field both changed.
	var response struct {
		Items struct {
			Items []struct {
				IsLocal bool `json:"is_local"`
				Item    *struct {
					Track           // embeds domain.Track; all fields promoted
					ItemType string `json:"type"` // "track" | "episode" — not on domain.Track
				} `json:"item"`
			} `json:"items"`
			Total int    `json:"total"`
			Next  string `json:"next"` // empty string when null in JSON
		} `json:"items"`
	}
	if err := l.doJSON(req, &response); err != nil {
		return nil, 0, false, fmt.Errorf("getting playlist: %w", err)
	}

	tracks := make([]Track, 0, len(response.Items.Items))
	for _, item := range response.Items.Items {
		// Skip local files, unavailable tracks, and podcast episodes.
		if item.IsLocal || item.Item == nil || item.Item.URI == "" {
			continue
		}
		if item.Item.ItemType != "" && item.Item.ItemType != "track" {
			continue
		}
		tracks = append(tracks, item.Item.Track)
	}
	return tracks, response.Items.Total, response.Items.Next != "", nil
}

// PlaylistTracks fetches one page of playlist items via GET /playlists/{id}/items.
// Use this for pagination (offset > 0) after the initial GetPlaylist call.
// Local files (is_local=true), unavailable tracks (null track object), and podcast
// episodes (type != "track") are skipped. Errors are wrapped with context.
func (l *LibraryClient) PlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]Track, int, bool, error) {
	path := fmt.Sprintf("/v1/playlists/%s/items", playlistID)
	req, err := l.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, false, fmt.Errorf("creating get playlist tracks request: %w", err)
	}

	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()

	// NOTE: Spotify renamed "track" → "item" per-item in February 2026.
	var response struct {
		Items []struct {
			IsLocal bool `json:"is_local"`
			Item    *struct {
				Track           // embeds domain.Track; all fields promoted
				ItemType string `json:"type"` // "track" | "episode" — not on domain.Track
			} `json:"item"`
		} `json:"items"`
		Total int    `json:"total"`
		Next  string `json:"next"` // empty string when null in JSON
	}
	if err := l.doJSON(req, &response); err != nil {
		return nil, 0, false, fmt.Errorf("getting playlist tracks: %w", err)
	}

	tracks := make([]Track, 0, len(response.Items))
	for _, item := range response.Items {
		// Skip local files, unavailable tracks, and podcast episodes.
		if item.IsLocal || item.Item == nil || item.Item.URI == "" {
			continue
		}
		if item.Item.ItemType != "" && item.Item.ItemType != "track" {
			continue
		}
		tracks = append(tracks, item.Item.Track)
	}
	return tracks, response.Total, response.Next != "", nil
}

// AlbumTracks fetches a page of tracks for the given album ID via
// GET /v1/albums/{id}/tracks. Returns the tracks, a hasNext bool (true when the
// API's "next" field is non-empty, indicating more pages), and any error.
// The caller controls pagination via limit and offset.
//
// NOTE: Album tracks are SimplifiedTrackObject — no "album" field in the response.
// The Album field on each returned domain.Track is intentionally empty; the caller
// already knows the album from context.
func (l *LibraryClient) AlbumTracks(ctx context.Context, albumID string, limit, offset int) ([]Track, bool, error) {
	path := fmt.Sprintf("/v1/albums/%s/tracks", albumID)
	req, err := l.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, false, fmt.Errorf("creating get album tracks request: %w", err)
	}
	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()

	var resp struct {
		Items []struct {
			ID         string   `json:"id"`
			URI        string   `json:"uri"`
			Name       string   `json:"name"`
			DurationMs int      `json:"duration_ms"`
			Explicit   bool     `json:"explicit"`
			Artists    []Artist `json:"artists"`
		} `json:"items"`
		Next string `json:"next"` // empty string when null in JSON
	}
	if err := l.doJSON(req, &resp); err != nil {
		return nil, false, fmt.Errorf("fetching album tracks: %w", err)
	}
	tracks := make([]Track, len(resp.Items))
	for i, item := range resp.Items {
		tracks[i] = Track{
			ID:         item.ID,
			URI:        item.URI,
			Name:       item.Name,
			DurationMs: item.DurationMs,
			Explicit:   item.Explicit,
			Artists:    item.Artists,
			// Album field intentionally empty — caller knows the album from context.
		}
	}
	return tracks, resp.Next != "", nil
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

// LikeTrack adds the given track to the user's library via PUT /me/library?uris=spotify:track:<id>.
// No JSON body — the track URI is passed as a query parameter.
// Errors are wrapped with context.
func (l *LibraryClient) LikeTrack(ctx context.Context, trackID string) error {
	uri := "spotify:track:" + trackID
	path := "/v1/me/library?uris=" + url.QueryEscape(uri)
	req, err := l.newRequest(ctx, http.MethodPut, path, nil)
	if err != nil {
		return fmt.Errorf("creating like track request: %w", err)
	}
	return l.doNoContent(req)
}

// UnlikeTrack removes the given track from the user's library via DELETE /me/library?uris=spotify:track:<id>.
// No JSON body — the track URI is passed as a query parameter.
// Errors are wrapped with context.
func (l *LibraryClient) UnlikeTrack(ctx context.Context, trackID string) error {
	uri := "spotify:track:" + trackID
	path := "/v1/me/library?uris=" + url.QueryEscape(uri)
	req, err := l.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("creating unlike track request: %w", err)
	}
	return l.doNoContent(req)
}
