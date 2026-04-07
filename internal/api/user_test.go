package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUserClient builds a UserClient pointed at the test server URL.
func newUserClient(baseURL string) *api.UserClient {
	return api.NewUserClient(baseURL, "test-token")
}

// TestGetTopTracks_Success verifies that GetTopTracks returns parsed tracks for a time range.
func TestGetTopTracks_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/top_tracks_response.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/top/tracks", r.URL.Path)
		assert.Equal(t, "short_term", r.URL.Query().Get("time_range"))
		assert.Equal(t, "25", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newUserClient(srv.URL)
	tracks, err := client.TopTracks(context.Background(), "short_term", 25)
	require.NoError(t, err)
	require.Len(t, tracks, 2)
	assert.Equal(t, "track-001", tracks[0].ID)
	assert.Equal(t, "Blinding Lights", tracks[0].Name)
	assert.Equal(t, "The Weeknd", tracks[0].Artists[0].Name)
}

// TestGetTopTracks_EmptyResults verifies that an empty items array returns an empty slice, not nil.
func TestGetTopTracks_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer srv.Close()

	client := newUserClient(srv.URL)
	tracks, err := client.TopTracks(context.Background(), "short_term", 25)
	require.NoError(t, err)
	assert.NotNil(t, tracks, "empty result should be empty slice not nil")
	assert.Empty(t, tracks)
}

// TestGetTopArtists_Success verifies that GetTopArtists returns parsed artists for a time range.
func TestGetTopArtists_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/top_artists_response.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/top/artists", r.URL.Path)
		assert.Equal(t, "medium_term", r.URL.Query().Get("time_range"))
		assert.Equal(t, "25", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newUserClient(srv.URL)
	artists, err := client.TopArtists(context.Background(), "medium_term", 25)
	require.NoError(t, err)
	require.Len(t, artists, 2)
	assert.Equal(t, "artist-weeknd", artists[0].ID)
	assert.Equal(t, "The Weeknd", artists[0].Name)
	assert.Contains(t, artists[0].Genres, "pop")
	assert.Equal(t, 95, artists[0].Popularity)
}

// TestGetTopArtists_EmptyResults verifies that an empty items array returns an empty slice, not nil.
func TestGetTopArtists_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer srv.Close()

	client := newUserClient(srv.URL)
	artists, err := client.TopArtists(context.Background(), "short_term", 25)
	require.NoError(t, err)
	assert.NotNil(t, artists, "empty result should be empty slice not nil")
	assert.Empty(t, artists)
}

// TestGetRecentlyPlayed_Success verifies GetRecentlyPlayed returns play history items with timestamps.
func TestGetRecentlyPlayed_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/recently_played_response.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player/recently-played", r.URL.Path)
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newUserClient(srv.URL)
	history, err := client.RecentlyPlayed(context.Background(), 50)
	require.NoError(t, err)
	require.Len(t, history, 2)
	assert.Equal(t, "Blinding Lights", history[0].Track.Name)
	assert.Equal(t, "2024-03-01T22:15:00Z", history[0].PlayedAt)
}

// TestArtist_Unmarshal verifies the FullArtist struct unmarshals all required fields.
func TestArtist_Unmarshal(t *testing.T) {
	data := `{
		"id": "artist-weeknd",
		"name": "The Weeknd",
		"uri": "spotify:artist:artist-weeknd",
		"genres": ["canadian pop", "pop", "r&b"],
		"popularity": 95,
		"external_urls": {
			"spotify": "https://open.spotify.com/artist/artist-weeknd"
		}
	}`

	var artist api.FullArtist
	err := json.Unmarshal([]byte(data), &artist)
	require.NoError(t, err)

	assert.Equal(t, "artist-weeknd", artist.ID)
	assert.Equal(t, "The Weeknd", artist.Name)
	assert.Equal(t, "spotify:artist:artist-weeknd", artist.URI)
	assert.Equal(t, []string{"canadian pop", "pop", "r&b"}, artist.Genres)
	assert.Equal(t, 95, artist.Popularity)
	assert.Equal(t, "https://open.spotify.com/artist/artist-weeknd", artist.ExternalURLs["spotify"])
}

// TestGetTopTracks_ErrorWrapped verifies that API errors are wrapped with context.
func TestGetTopTracks_ErrorWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server error"}`))
	}))
	defer srv.Close()

	client := newUserClient(srv.URL)
	_, err := client.TopTracks(context.Background(), "short_term", 25)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting top tracks")
}

// TestGetTopArtists_ErrorWrapped verifies that API errors are wrapped with context.
func TestGetTopArtists_ErrorWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server error"}`))
	}))
	defer srv.Close()

	client := newUserClient(srv.URL)
	_, err := client.TopArtists(context.Background(), "short_term", 25)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting top artists")
}

// TestUserClient_Profile_Success verifies that Profile returns the user's Spotify ID.
func TestUserClient_Profile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"user123","display_name":"Test User"}`))
	}))
	defer srv.Close()

	client := newUserClient(srv.URL)
	profile, err := client.Profile(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "user123", profile.ID)
}

// TestUserClient_Profile_ErrorWrapped verifies that API errors are wrapped with context.
func TestUserClient_Profile_ErrorWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server error"}`))
	}))
	defer srv.Close()

	client := newUserClient(srv.URL)
	_, err := client.Profile(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetching profile")
}
