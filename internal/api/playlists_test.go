package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestPlaylists creates a PlaylistsClient pointing at the given base URL.
func newTestPlaylists(baseURL, token string) *PlaylistsClient {
	return NewPlaylistsClient(baseURL, token)
}

// TestCreatePlaylist covers success and server-error variants for CreatePlaylist.
func TestCreatePlaylist(t *testing.T) {
	successBody := string(testhelpers.LoadFixture(t, "create_playlist_response.json"))
	tests := []struct {
		name       string
		status     int
		body       string
		wantErr    bool
		wantErrMsg string
		checkResp  func(t *testing.T, pl *SimplePlaylist)
	}{
		{
			name:   "success",
			status: http.StatusCreated,
			body:   successBody,
			checkResp: func(t *testing.T, pl *SimplePlaylist) {
				require.NotNil(t, pl)
				assert.Equal(t, "new-playlist-id", pl.ID)
				assert.Equal(t, "My New Playlist", pl.Name)
			},
		},
		{
			name:       "server error",
			status:     http.StatusInternalServerError,
			body:       `{"error": "server error"}`,
			wantErr:    true,
			wantErrMsg: "creating playlist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/me/playlists", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				if tt.status == http.StatusCreated {
					var body map[string]interface{}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					assert.Equal(t, "My New Playlist", body["name"])
					assert.Equal(t, "A cool playlist", body["description"])
					assert.Equal(t, false, body["public"])
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := newTestPlaylists(srv.URL, "test-token")
			playlist, err := client.CreatePlaylist(context.Background(), "My New Playlist", "A cool playlist", false)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, playlist)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				require.NoError(t, err)
				if tt.checkResp != nil {
					tt.checkResp(t, playlist)
				}
			}
		})
	}
}

// TestUpdatePlaylist_Success verifies UpdatePlaylist sends the correct PUT body.
func TestUpdatePlaylist_Success(t *testing.T) {
	var capturedPath string
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestPlaylists(srv.URL, "test-token")
	err := client.UpdatePlaylist(context.Background(), "playlist-123", "Renamed Playlist", "New description")

	require.NoError(t, err)
	assert.Equal(t, "/v1/playlists/playlist-123", capturedPath)
	assert.Equal(t, "Renamed Playlist", capturedBody["name"])
	assert.Equal(t, "New description", capturedBody["description"])
}

// TestAddTracksToPlaylist_Success verifies AddTracksToPlaylist sends the correct POST body with URIs.
func TestAddTracksToPlaylist_Success(t *testing.T) {
	var capturedPath string
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"snapshot_id": "snap-abc123"}`))
	}))
	defer srv.Close()

	client := newTestPlaylists(srv.URL, "test-token")
	err := client.AddTracksToPlaylist(context.Background(), "playlist-123", []string{"spotify:track:abc", "spotify:track:def"})

	require.NoError(t, err)
	assert.Equal(t, "/v1/playlists/playlist-123/items", capturedPath)
	uris, ok := capturedBody["uris"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, "spotify:track:abc", uris[0])
	assert.Equal(t, "spotify:track:def", uris[1])
}

// TestRemoveTracksFromPlaylist_Success verifies RemoveTracksFromPlaylist sends the correct DELETE body.
func TestRemoveTracksFromPlaylist_Success(t *testing.T) {
	var capturedPath string
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"snapshot_id": "snap-def456"}`))
	}))
	defer srv.Close()

	client := newTestPlaylists(srv.URL, "test-token")
	err := client.RemoveTracksFromPlaylist(context.Background(), "playlist-123", []string{"spotify:track:abc"})

	require.NoError(t, err)
	assert.Equal(t, "/v1/playlists/playlist-123/items", capturedPath)
	items, ok := capturedBody["items"].([]interface{})
	require.True(t, ok, "DELETE body must use 'items' field per Spotify Feb 2026 spec, not 'tracks'")
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "spotify:track:abc", item["uri"])
}

// TestReorderPlaylistTracks covers success and error variants for ReorderPlaylistTracks.
func TestReorderPlaylistTracks(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		body         string
		rangeStart   int
		insertBefore int
		rangeLength  int
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name:         "success",
			status:       http.StatusOK,
			body:         `{"snapshot_id": "snap-ghi789"}`,
			rangeStart:   2,
			insertBefore: 0,
			rangeLength:  1,
		},
		{
			name:       "server error",
			status:     http.StatusForbidden,
			body:       `{"error": "forbidden"}`,
			wantErr:    true,
			wantErrMsg: "reordering playlist tracks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			var capturedBody map[string]interface{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				_ = json.NewDecoder(r.Body).Decode(&capturedBody)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := newTestPlaylists(srv.URL, "test-token")
			err := client.ReorderPlaylistTracks(context.Background(), "playlist-123", tt.rangeStart, tt.insertBefore, tt.rangeLength)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "/v1/playlists/playlist-123/items", capturedPath)
				assert.Equal(t, float64(tt.rangeStart), capturedBody["range_start"])
				assert.Equal(t, float64(tt.insertBefore), capturedBody["insert_before"])
				assert.Equal(t, float64(tt.rangeLength), capturedBody["range_length"])
			}
		})
	}
}
