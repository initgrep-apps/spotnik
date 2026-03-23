package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestPlaylists creates a PlaylistsClient pointing at the given base URL.
func newTestPlaylists(baseURL, token string) *PlaylistsClient {
	return NewPlaylistsClient(baseURL, token)
}

// TestCreatePlaylist_Success verifies CreatePlaylist returns the created playlist.
func TestCreatePlaylist_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/playlists", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "My New Playlist", body["name"])
		assert.Equal(t, "A cool playlist", body["description"])
		assert.Equal(t, false, body["public"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": "new-playlist-id",
			"name": "My New Playlist",
			"uri": "spotify:playlist:new-playlist-id",
			"tracks": {"total": 0},
			"owner": {"id": "user-1", "display_name": "Test User"}
		}`))
	}))
	defer srv.Close()

	client := newTestPlaylists(srv.URL, "test-token")
	playlist, err := client.CreatePlaylist(context.Background(), "My New Playlist", "A cool playlist", false)

	require.NoError(t, err)
	require.NotNil(t, playlist)
	assert.Equal(t, "new-playlist-id", playlist.ID)
	assert.Equal(t, "My New Playlist", playlist.Name)
}

// TestCreatePlaylist_ServerError verifies CreatePlaylist returns a descriptive error on failure.
func TestCreatePlaylist_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "server error"}`))
	}))
	defer srv.Close()

	client := newTestPlaylists(srv.URL, "test-token")
	playlist, err := client.CreatePlaylist(context.Background(), "Fail Playlist", "", false)

	require.Error(t, err)
	assert.Nil(t, playlist)
	assert.Contains(t, err.Error(), "creating playlist")
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
	assert.Equal(t, "/v1/playlists/playlist-123/tracks", capturedPath)
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
	assert.Equal(t, "/v1/playlists/playlist-123/tracks", capturedPath)
	tracks, ok := capturedBody["tracks"].([]interface{})
	require.True(t, ok)
	require.Len(t, tracks, 1)
	track, ok := tracks[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "spotify:track:abc", track["uri"])
}

// TestReorderPlaylistTracks_Success verifies ReorderPlaylistTracks sends the correct PUT body with range params.
func TestReorderPlaylistTracks_Success(t *testing.T) {
	var capturedPath string
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"snapshot_id": "snap-ghi789"}`))
	}))
	defer srv.Close()

	client := newTestPlaylists(srv.URL, "test-token")
	err := client.ReorderPlaylistTracks(context.Background(), "playlist-123", 2, 0, 1)

	require.NoError(t, err)
	assert.Equal(t, "/v1/playlists/playlist-123/tracks", capturedPath)
	assert.Equal(t, float64(2), capturedBody["range_start"])
	assert.Equal(t, float64(0), capturedBody["insert_before"])
	assert.Equal(t, float64(1), capturedBody["range_length"])
}

// TestReorderPlaylistTracks_Error verifies ReorderPlaylistTracks returns an error with context.
func TestReorderPlaylistTracks_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": "forbidden"}`))
	}))
	defer srv.Close()

	client := newTestPlaylists(srv.URL, "test-token")
	err := client.ReorderPlaylistTracks(context.Background(), "playlist-123", 0, 1, 1)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reordering playlist tracks")
}
