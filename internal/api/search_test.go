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

// newTestSearch creates a SearchClient pointing at the given base URL using the provided token.
func newTestSearch(baseURL, token string) *SearchClient {
	return NewSearchClient(baseURL, token)
}

// TestSearch_Success verifies that Search returns a fully parsed SearchResult.
func TestSearch_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/search_result.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/search", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newTestSearch(srv.URL, "test-token")
	result, err := client.Search(context.Background(), "blinding lights", []string{"track", "artist", "album", "playlist"}, 5)

	require.NoError(t, err)
	require.NotNil(t, result)
	// Tracks
	require.Len(t, result.Tracks.Items, 1)
	assert.Equal(t, "Blinding Lights", result.Tracks.Items[0].Name)
	// Artists
	require.Len(t, result.Artists.Items, 1)
	assert.Equal(t, "The Weeknd", result.Artists.Items[0].Name)
	// Albums
	require.Len(t, result.Albums.Items, 1)
	assert.Equal(t, "After Hours", result.Albums.Items[0].Name)
	// Playlists
	require.Len(t, result.Playlists.Items, 1)
	assert.Equal(t, "Blinding Pop Hits", result.Playlists.Items[0].Name)
}

// TestSearch_EmptyResults verifies that Search returns empty slices without errors.
func TestSearch_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tracks":    {"items": [], "total": 0},
			"artists":   {"items": [], "total": 0},
			"albums":    {"items": [], "total": 0},
			"playlists": {"items": [], "total": 0}
		}`))
	}))
	defer srv.Close()

	client := newTestSearch(srv.URL, "test-token")
	result, err := client.Search(context.Background(), "zzznoresults", []string{"track"}, 5)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Tracks.Items)
	assert.Empty(t, result.Artists.Items)
	assert.Empty(t, result.Albums.Items)
	assert.Empty(t, result.Playlists.Items)
}

// TestSearch_ServerError verifies that Search returns a descriptive error on non-2xx responses.
func TestSearch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer srv.Close()

	client := newTestSearch(srv.URL, "test-token")
	result, err := client.Search(context.Background(), "blinding lights", []string{"track"}, 5)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "500")
}

// TestSearch_InvalidJSON verifies that Search returns a parse error on invalid JSON.
func TestSearch_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not-valid-json`))
	}))
	defer srv.Close()

	client := newTestSearch(srv.URL, "test-token")
	result, err := client.Search(context.Background(), "test", []string{"track"}, 5)

	require.Error(t, err)
	assert.Nil(t, result)
}

// TestSearch_RequestParams verifies that Search sends the correct query params.
func TestSearch_RequestParams(t *testing.T) {
	var capturedQuery, capturedType, capturedLimit, capturedMarket string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("q")
		capturedType = r.URL.Query().Get("type")
		capturedLimit = r.URL.Query().Get("limit")
		capturedMarket = r.URL.Query().Get("market")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tracks":    {"items": [], "total": 0},
			"artists":   {"items": [], "total": 0},
			"albums":    {"items": [], "total": 0},
			"playlists": {"items": [], "total": 0}
		}`))
	}))
	defer srv.Close()

	client := newTestSearch(srv.URL, "test-token")
	_, err := client.Search(context.Background(), "hello world", []string{"track", "artist", "album", "playlist"}, 5)

	require.NoError(t, err)
	assert.Equal(t, "hello world", capturedQuery)
	assert.Equal(t, "track,artist,album,playlist", capturedType)
	assert.Equal(t, "5", capturedLimit)
	assert.Equal(t, "from_token", capturedMarket)
}
