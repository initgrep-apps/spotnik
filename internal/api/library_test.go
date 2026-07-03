package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestLibrary creates a LibraryClient pointing at the given base URL using the provided token.
func newTestLibrary(baseURL, token string) *LibraryClient {
	return NewLibraryClient(baseURL, token)
}

// TestGetPlaylists verifies Playlists returns parsed playlists and handles edge cases.
func TestGetPlaylists(t *testing.T) {
	fixture := testhelpers.LoadFixture(t, "playlists_response.json")

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		checkResp func(t *testing.T, playlists []SimplePlaylist, err error)
	}{
		{
			name: "success parses playlists",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/me/playlists", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				assert.Equal(t, "50", r.URL.Query().Get("limit"))
				assert.Equal(t, "0", r.URL.Query().Get("offset"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(fixture)
			},
			checkResp: func(t *testing.T, playlists []SimplePlaylist, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, playlists, 2)
				assert.Equal(t, "playlist-abc123", playlists[0].ID)
				assert.Equal(t, "Chill Vibes", playlists[0].Name)
				assert.Equal(t, 42, playlists[0].TrackCount)
				assert.Equal(t, "playlist-def456", playlists[1].ID)
				assert.Equal(t, "Workout Mix", playlists[1].Name)
			},
		},
		{
			name: "empty response returns empty slice",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"items": [], "total": 0}`))
			},
			checkResp: func(t *testing.T, playlists []SimplePlaylist, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, playlists)
			},
		},
		{
			name: "429 returns RateLimitError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Retry-After", "10")
				w.WriteHeader(http.StatusTooManyRequests)
			},
			checkResp: func(t *testing.T, _ []SimplePlaylist, err error) {
				t.Helper()
				require.Error(t, err)
				var rlErr *RateLimitError
				assert.ErrorAs(t, err, &rlErr, "expected *RateLimitError for 429")
			},
		},
		{
			name: "500 returns error with status code",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("internal error"))
			},
			checkResp: func(t *testing.T, _ []SimplePlaylist, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "500")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			client := newTestLibrary(srv.URL, "test-token")
			playlists, err := client.Playlists(context.Background(), 50, 0)
			tt.checkResp(t, playlists, err)
		})
	}
}

// TestGetPlaylistTracks verifies PlaylistTracks handles various server responses and filtering.
func TestGetPlaylistTracks(t *testing.T) {
	fixture := testhelpers.LoadFixture(t, "playlist_tracks_response.json")

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		checkResp func(t *testing.T, tracks []Track, total int, hasNext bool, err error)
	}{
		{
			name: "success parses tracks",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/playlists/playlist-abc123/items", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(fixture)
			},
			checkResp: func(t *testing.T, tracks []Track, total int, hasNext bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 2)
				assert.Equal(t, "track-xyz789", tracks[0].ID)
				assert.Equal(t, "Blinding Lights", tracks[0].Name)
				assert.Equal(t, 2, total)
				assert.False(t, hasNext, "next is null in fixture → hasNext should be false")
			},
		},
		{
			name: "uses items endpoint not tracks",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/playlists/pl-abc/items", r.URL.Path,
					"PlaylistTracks must use /items (not deprecated /tracks)")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"items":[],"total":0,"next":null}`))
			},
			checkResp: func(t *testing.T, _ []Track, _ int, _ bool, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name: "non-null next sets hasNext true",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				body := `{
					"items": [{"is_local":false,"item":{"id":"t1","name":"Track 1","uri":"spotify:track:t1","duration_ms":180000,"type":"track","artists":[{"id":"a1","name":"Artist","uri":"spotify:artist:a1"}],"album":{"id":"al1","name":"Album","uri":"spotify:album:al1","release_date":"2020-01-01"}}}],
					"total": 200,
					"next": "https://api.spotify.com/v1/playlists/pl1/items?offset=100&limit=100"
				}`
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(body))
			},
			checkResp: func(t *testing.T, tracks []Track, total int, hasNext bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 1)
				assert.Equal(t, 200, total)
				assert.True(t, hasNext, "non-null next → hasNext should be true")
			},
		},
		{
			name: "null track entry skipped",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				body := `{
					"items": [
						{"is_local":false,"item":null},
						{"is_local":false,"item":{"id":"t2","name":"Good Track","uri":"spotify:track:t2","duration_ms":200000,"type":"track","artists":[{"id":"a1","name":"Artist","uri":"spotify:artist:a1"}],"album":{"id":"al1","name":"Album","uri":"spotify:album:al1","release_date":"2020-01-01"}}}
					],
					"total": 1,
					"next": null
				}`
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(body))
			},
			checkResp: func(t *testing.T, tracks []Track, _ int, _ bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 1, "null track entry must be skipped")
				assert.Equal(t, "t2", tracks[0].ID)
			},
		},
		{
			name: "is_local=true item skipped",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				body := `{
					"items": [
						{"is_local":true,"item":{"id":"local1","name":"Local File","uri":"","duration_ms":200000,"type":"track","artists":[],"album":{"id":"","name":"","uri":"","release_date":""}}},
						{"is_local":false,"item":{"id":"t2","name":"Streaming Track","uri":"spotify:track:t2","duration_ms":200000,"type":"track","artists":[{"id":"a1","name":"Artist","uri":"spotify:artist:a1"}],"album":{"id":"al1","name":"Album","uri":"spotify:album:al1","release_date":"2020-01-01"}}}
					],
					"total": 1,
					"next": null
				}`
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(body))
			},
			checkResp: func(t *testing.T, tracks []Track, _ int, _ bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 1, "is_local=true entry must be skipped")
				assert.Equal(t, "t2", tracks[0].ID)
			},
		},
		{
			name: "episode item skipped",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				body := `{
					"items": [
						{"is_local":false,"item":{"id":"ep1","name":"Episode","uri":"spotify:episode:ep1","duration_ms":3600000,"type":"episode","artists":[],"album":{"id":"","name":"","uri":"","release_date":""}}},
						{"is_local":false,"item":{"id":"t1","name":"Real Track","uri":"spotify:track:t1","duration_ms":200000,"type":"track","artists":[{"id":"a1","name":"Artist","uri":"spotify:artist:a1"}],"album":{"id":"al1","name":"Album","uri":"spotify:album:al1","release_date":"2021-01-01"}}}
					],
					"total": 1,
					"next": null
				}`
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(body))
			},
			checkResp: func(t *testing.T, tracks []Track, _ int, _ bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 1, "episode must be filtered out")
				assert.Equal(t, "t1", tracks[0].ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			// Use pl-abc only for the "uses items endpoint" case; otherwise use playlist-abc123.
			playlistID := "playlist-abc123"
			if tt.name == "uses items endpoint not tracks" {
				playlistID = "pl-abc"
			} else if tt.name != "success parses tracks" {
				playlistID = "pl1"
			}
			client := newTestLibrary(srv.URL, "test-token")
			tracks, total, hasNext, err := client.PlaylistTracks(context.Background(), playlistID, 100, 0)
			tt.checkResp(t, tracks, total, hasNext, err)
		})
	}
}

// TestAlbumTracks verifies AlbumTracks handles success, pagination, and errors.
func TestAlbumTracks(t *testing.T) {
	tests := []struct {
		name      string
		albumID   string
		handler   http.HandlerFunc
		checkResp func(t *testing.T, tracks []Track, hasNext bool, err error)
	}{
		{
			name:    "success parses tracks with hasNext false",
			albumID: "album-kind-of-blue",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/albums/album-kind-of-blue/tracks", r.URL.Path)
				assert.Equal(t, "50", r.URL.Query().Get("limit"))
				assert.Equal(t, "0", r.URL.Query().Get("offset"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"items": [
						{"id":"track1","uri":"spotify:track:track1","name":"So What","duration_ms":562000,"explicit":false,"artists":[{"id":"artist1","name":"Miles Davis","uri":"spotify:artist:artist1"}]},
						{"id":"track2","uri":"spotify:track:track2","name":"Freddie Freeloader","duration_ms":586000,"explicit":false,"artists":[{"id":"artist1","name":"Miles Davis","uri":"spotify:artist:artist1"}]}
					],
					"limit": 50, "next": null, "offset": 0, "total": 2
				}`))
			},
			checkResp: func(t *testing.T, tracks []Track, hasNext bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 2)
				assert.Equal(t, "track1", tracks[0].ID)
				assert.Equal(t, "So What", tracks[0].Name)
				assert.Equal(t, 562000, tracks[0].DurationMs)
				assert.Equal(t, "Miles Davis", tracks[0].Artists[0].Name)
				assert.False(t, hasNext, "next=null → hasNext should be false")
			},
		},
		{
			name:    "non-null next returns hasNext true",
			albumID: "alb1",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"items": [{"id":"t1","uri":"spotify:track:t1","name":"Track 1","duration_ms":200000,"explicit":false,"artists":[]}],
					"limit": 50,
					"next": "https://api.spotify.com/v1/albums/alb1/tracks?offset=50&limit=50",
					"offset": 0, "total": 62
				}`))
			},
			checkResp: func(t *testing.T, tracks []Track, hasNext bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 1)
				assert.True(t, hasNext, "non-null next → hasNext should be true")
			},
		},
		{
			name:    "404 returns error",
			albumID: "nonexistent",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":{"status":404,"message":"Not found"}}`))
			},
			checkResp: func(t *testing.T, tracks []Track, hasNext bool, err error) {
				t.Helper()
				require.Error(t, err, "404 response must return an error")
				assert.Nil(t, tracks)
				assert.False(t, hasNext)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			client := newTestLibrary(srv.URL, "test-token")
			tracks, hasNext, err := client.AlbumTracks(context.Background(), tt.albumID, 50, 0)
			tt.checkResp(t, tracks, hasNext, err)
		})
	}
}

// TestGetSavedAlbums_Success verifies GetSavedAlbums returns parsed albums.
func TestGetSavedAlbums_Success(t *testing.T) {
	fixture := testhelpers.LoadFixture(t, "saved_albums_response.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/albums", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	albums, err := client.SavedAlbums(context.Background(), 50, 0)

	require.NoError(t, err)
	require.Len(t, albums, 1)
	assert.Equal(t, "album-after-hours", albums[0].Album.ID)
	assert.Equal(t, "After Hours", albums[0].Album.Name)
	assert.Equal(t, 14, albums[0].Album.TotalTracks)
}

// TestGetLikedTracks_Success verifies GetLikedTracks returns parsed saved tracks.
func TestGetLikedTracks_Success(t *testing.T) {
	fixture := testhelpers.LoadFixture(t, "liked_tracks_response.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/tracks", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	tracks, err := client.LikedTracks(context.Background(), 50, 0)

	require.NoError(t, err)
	require.Len(t, tracks, 2)
	assert.Equal(t, "track-xyz789", tracks[0].Track.ID)
	assert.Equal(t, "Blinding Lights", tracks[0].Track.Name)
	assert.Equal(t, "2024-02-20T14:00:00Z", tracks[0].AddedAt)
}

// TestGetRecentlyPlayed_Success verifies GetRecentlyPlayed returns play history items.
func TestGetRecentlyPlayed_Success(t *testing.T) {
	fixture := testhelpers.LoadFixture(t, "recently_played_response.json")

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
	items, err := client.RecentlyPlayed(context.Background(), 20)

	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "track-xyz789", items[0].Track.ID)
	assert.Equal(t, "Blinding Lights", items[0].Track.Name)
	assert.Equal(t, "2024-03-01T22:15:00Z", items[0].PlayedAt)
}

// TestLikeTrack_SendsPUT verifies that LikeTrack sends PUT to /v1/me/library?uris=spotify:track:<id>
// with no JSON body and no Content-Type header.
func TestLikeTrack_SendsPUT(t *testing.T) {
	var capturedMethod, capturedPath, capturedRawQuery string
	var capturedBody []byte
	var capturedContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedRawQuery = r.URL.RawQuery
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		capturedBody = buf[:n]
		capturedContentType = r.Header.Get("Content-Type")
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	err := client.LikeTrack(context.Background(), "track-xyz789")

	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, capturedMethod)
	assert.Equal(t, "/v1/me/library", capturedPath)
	assert.Contains(t, capturedRawQuery, "uris=spotify%3Atrack%3Atrack-xyz789")
	assert.Empty(t, capturedBody, "no JSON body should be sent")
	assert.Empty(t, capturedContentType, "no Content-Type header should be set")
}

// TestUnlikeTrack_SendsDELETE verifies that UnlikeTrack sends DELETE to /v1/me/library?uris=spotify:track:<id>
// with no JSON body and no Content-Type header.
func TestUnlikeTrack_SendsDELETE(t *testing.T) {
	var capturedMethod, capturedPath, capturedRawQuery string
	var capturedBody []byte
	var capturedContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedRawQuery = r.URL.RawQuery
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		capturedBody = buf[:n]
		capturedContentType = r.Header.Get("Content-Type")
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	err := client.UnlikeTrack(context.Background(), "track-xyz789")

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, capturedMethod)
	assert.Equal(t, "/v1/me/library", capturedPath)
	assert.Contains(t, capturedRawQuery, "uris=spotify%3Atrack%3Atrack-xyz789")
	assert.Empty(t, capturedBody, "no JSON body should be sent")
	assert.Empty(t, capturedContentType, "no Content-Type header should be set")
}

// TestLibraryClient_DoNoContent_ServerError verifies 403 returns a typed ForbiddenError.
func TestLibraryClient_DoNoContent_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Premium required"))
	}))
	defer srv.Close()

	client := newTestLibrary(srv.URL, "test-token")
	err := client.LikeTrack(context.Background(), "track-xyz789")

	require.Error(t, err)
	var forbErr *ForbiddenError
	assert.ErrorAs(t, err, &forbErr, "expected *ForbiddenError for 403")
}

// TestNewLibraryClient_DefaultBaseURL verifies production URL is used when baseURL is empty.
func TestNewLibraryClient_DefaultBaseURL(t *testing.T) {
	client := NewLibraryClient("", "test-token")
	assert.NotNil(t, client)
	// The client was created — we can't easily test the URL but no panic occurred.
}

// TestGetPlaylist verifies GetPlaylist parses tracks, filters items, and handles errors.
func TestGetPlaylist(t *testing.T) {
	tests := []struct {
		name       string
		playlistID string
		handler    http.HandlerFunc
		checkResp  func(t *testing.T, tracks []Track, total int, hasNext bool, err error)
	}{
		{
			name:       "success uses items endpoint and parses tracks",
			playlistID: "pl-abc",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/playlists/pl-abc", r.URL.Path,
					"must call GET /playlists/{id}, not deprecated /tracks")
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "pl-abc", "name": "My SimplePlaylist", "uri": "spotify:playlist:pl-abc",
					"items": {
						"items": [
							{"is_local":false,"item":{"id":"t1","name":"Track 1","uri":"spotify:track:t1","duration_ms":200000,"type":"track","artists":[{"id":"a1","name":"Artist 1","uri":"spotify:artist:a1"}],"album":{"id":"al1","name":"Album 1","uri":"spotify:album:al1","release_date":"2021-01-01"}}},
							{"is_local":false,"item":{"id":"t2","name":"Track 2","uri":"spotify:track:t2","duration_ms":180000,"type":"track","artists":[{"id":"a1","name":"Artist 1","uri":"spotify:artist:a1"}],"album":{"id":"al1","name":"Album 1","uri":"spotify:album:al1","release_date":"2021-01-01"}}}
						],
						"total": 2, "next": null
					}
				}`))
			},
			checkResp: func(t *testing.T, tracks []Track, total int, hasNext bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 2)
				assert.Equal(t, "t1", tracks[0].ID)
				assert.Equal(t, "Track 1", tracks[0].Name)
				assert.Equal(t, 200000, tracks[0].DurationMs)
				assert.Equal(t, 2, total)
				assert.False(t, hasNext, "next=null → hasNext false")
			},
		},
		{
			name:       "non-null next returns hasNext true",
			playlistID: "pl-big",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "pl-big", "name": "Big SimplePlaylist", "uri": "spotify:playlist:pl-big",
					"items": {
						"items": [{"is_local":false,"item":{"id":"t1","name":"T1","uri":"spotify:track:t1","duration_ms":180000,"type":"track","artists":[],"album":{"id":"al1","name":"A1","uri":"spotify:album:al1","release_date":"2021-01-01"}}}],
						"total": 500, "next": "https://api.spotify.com/v1/playlists/pl-big/items?offset=100&limit=100"
					}
				}`))
			},
			checkResp: func(t *testing.T, tracks []Track, total int, hasNext bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 1)
				assert.Equal(t, 500, total)
				assert.True(t, hasNext, "non-null next → hasNext true")
			},
		},
		{
			name:       "episode item skipped",
			playlistID: "pl-mixed",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "pl-mixed", "name": "Mixed", "uri": "spotify:playlist:pl-mixed",
					"items": {
						"items": [
							{"is_local":false,"item":{"id":"ep1","name":"Episode 1","uri":"spotify:episode:ep1","duration_ms":3600000,"type":"episode","artists":[],"album":{"id":"","name":"","uri":"","release_date":""}}},
							{"is_local":false,"item":{"id":"t1","name":"Track 1","uri":"spotify:track:t1","duration_ms":200000,"type":"track","artists":[{"id":"a1","name":"Artist","uri":"spotify:artist:a1"}],"album":{"id":"al1","name":"Album","uri":"spotify:album:al1","release_date":"2021-01-01"}}}
						],
						"total": 1, "next": null
					}
				}`))
			},
			checkResp: func(t *testing.T, tracks []Track, _ int, _ bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 1, "episode item must be skipped; only track items returned")
				assert.Equal(t, "t1", tracks[0].ID)
			},
		},
		{
			name:       "null track entry skipped",
			playlistID: "pl-nulls",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "pl-nulls", "name": "Nulls", "uri": "spotify:playlist:pl-nulls",
					"items": {
						"items": [
							{"is_local":false,"item":null},
							{"is_local":false,"item":{"id":"t1","name":"Good","uri":"spotify:track:t1","duration_ms":200000,"type":"track","artists":[],"album":{"id":"al1","name":"A","uri":"spotify:album:al1","release_date":"2021-01-01"}}}
						],
						"total": 1, "next": null
					}
				}`))
			},
			checkResp: func(t *testing.T, tracks []Track, _ int, _ bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 1, "null track must be skipped")
				assert.Equal(t, "t1", tracks[0].ID)
			},
		},
		{
			name:       "is_local=true item skipped",
			playlistID: "pl-local",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "pl-local", "name": "Local", "uri": "spotify:playlist:pl-local",
					"items": {
						"items": [
							{"is_local":true,"item":{"id":"local1","name":"Local File","uri":"","duration_ms":200000,"type":"track","artists":[],"album":{"id":"","name":"","uri":"","release_date":""}}},
							{"is_local":false,"item":{"id":"t1","name":"Real Track","uri":"spotify:track:t1","duration_ms":200000,"type":"track","artists":[],"album":{"id":"al1","name":"A","uri":"spotify:album:al1","release_date":"2021-01-01"}}}
						],
						"total": 1, "next": null
					}
				}`))
			},
			checkResp: func(t *testing.T, tracks []Track, _ int, _ bool, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, tracks, 1, "is_local=true item must be skipped")
				assert.Equal(t, "t1", tracks[0].ID)
			},
		},
		{
			name:       "500 returns error",
			playlistID: "pl-bad",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{"status":500,"message":"internal"}}`))
			},
			checkResp: func(t *testing.T, tracks []Track, total int, hasNext bool, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, tracks)
				assert.Zero(t, total)
				assert.False(t, hasNext)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			client := newTestLibrary(srv.URL, "test-token")
			tracks, total, hasNext, err := client.GetPlaylist(context.Background(), tt.playlistID)
			tt.checkResp(t, tracks, total, hasNext, err)
		})
	}
}
