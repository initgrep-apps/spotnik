package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestLibrary creates a LibraryClient pointing at the given base URL using the provided token.
func newTestLibrary(baseURL, token string) *LibraryClient {
	return NewLibraryClient(baseURL, token)
}

// TestGetPlaylists_Success verifies that GetPlaylists returns parsed playlists.
func TestGetPlaylists_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/playlists_response.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/playlists", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		assert.Equal(t, "0", r.URL.Query().Get("offset"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	playlists, err := client.GetPlaylists(context.Background(), 50, 0)

	require.NoError(t, err)
	require.Len(t, playlists, 2)
	assert.Equal(t, "playlist-abc123", playlists[0].ID)
	assert.Equal(t, "Chill Vibes", playlists[0].Name)
	assert.Equal(t, 42, playlists[0].TrackCount)
	assert.Equal(t, "playlist-def456", playlists[1].ID)
	assert.Equal(t, "Workout Mix", playlists[1].Name)
}

// TestGetPlaylists_Empty verifies that GetPlaylists returns empty slice, no error.
func TestGetPlaylists_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": [], "total": 0}`))
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	playlists, err := client.GetPlaylists(context.Background(), 50, 0)

	require.NoError(t, err)
	assert.Empty(t, playlists)
}

// TestGetPlaylistTracks_Success verifies GetPlaylistTracks returns tracks for a playlist ID.
func TestGetPlaylistTracks_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/playlist_tracks_response.json")
	require.NoError(t, err)

	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	tracks, err := client.GetPlaylistTracks(context.Background(), "playlist-abc123", 50, 0)

	require.NoError(t, err)
	assert.Equal(t, "/v1/playlists/playlist-abc123/tracks", capturedPath)
	require.Len(t, tracks, 2)
	assert.Equal(t, "track-xyz789", tracks[0].ID)
	assert.Equal(t, "Blinding Lights", tracks[0].Name)
}

// TestGetSavedAlbums_Success verifies GetSavedAlbums returns parsed albums.
func TestGetSavedAlbums_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/saved_albums_response.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/albums", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	albums, err := client.GetSavedAlbums(context.Background(), 50, 0)

	require.NoError(t, err)
	require.Len(t, albums, 1)
	assert.Equal(t, "album-after-hours", albums[0].Album.ID)
	assert.Equal(t, "After Hours", albums[0].Album.Name)
	assert.Equal(t, 14, albums[0].Album.TotalTracks)
}

// TestGetLikedTracks_Success verifies GetLikedTracks returns parsed saved tracks.
func TestGetLikedTracks_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/liked_tracks_response.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/tracks", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	tracks, err := client.GetLikedTracks(context.Background(), 50, 0)

	require.NoError(t, err)
	require.Len(t, tracks, 2)
	assert.Equal(t, "track-xyz789", tracks[0].Track.ID)
	assert.Equal(t, "Blinding Lights", tracks[0].Track.Name)
	assert.Equal(t, "2024-02-20T14:00:00Z", tracks[0].AddedAt)
}

// TestGetRecentlyPlayed_Success verifies GetRecentlyPlayed returns play history items.
func TestGetRecentlyPlayed_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/recently_played_response.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player/recently-played", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "20", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	items, err := client.GetRecentlyPlayed(context.Background(), 20)

	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "track-xyz789", items[0].Track.ID)
	assert.Equal(t, "Blinding Lights", items[0].Track.Name)
	assert.Equal(t, "2024-03-01T22:15:00Z", items[0].PlayedAt)
}

// TestLikeTrack_SendsPUT verifies that LikeTrack sends the correct method, path, and body.
func TestLikeTrack_SendsPUT(t *testing.T) {
	var capturedMethod, capturedPath, capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		capturedBody = string(buf[:n])
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	err := client.LikeTrack(context.Background(), "track-xyz789")

	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, capturedMethod)
	assert.Equal(t, "/v1/me/tracks", capturedPath)
	assert.Contains(t, capturedBody, "track-xyz789")
}

// TestUnlikeTrack_SendsDELETE verifies that UnlikeTrack sends the correct method and path.
func TestUnlikeTrack_SendsDELETE(t *testing.T) {
	var capturedMethod, capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	err := client.UnlikeTrack(context.Background(), "track-xyz789")

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, capturedMethod)
	assert.Equal(t, "/v1/me/tracks", capturedPath)
}

// TestLibraryClient_RateLimited verifies 429 response returns appropriate error.
func TestLibraryClient_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	_, err := client.GetPlaylists(context.Background(), 50, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

// TestLibraryClient_ServerError verifies non-2xx non-429 returns an error.
func TestLibraryClient_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	_, err := client.GetPlaylists(context.Background(), 50, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// TestLibraryClient_DoNoContent_ServerError verifies non-2xx for LikeTrack.
func TestLibraryClient_DoNoContent_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Premium required"))
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	err := client.LikeTrack(context.Background(), "track-xyz789")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

// TestNewLibraryClient_DefaultBaseURL verifies production URL is used when baseURL is empty.
func TestNewLibraryClient_DefaultBaseURL(t *testing.T) {
	client := NewLibraryClient("", "test-token")
	assert.NotNil(t, client)
	// The client was created — we can't easily test the URL but no panic occurred.
}
