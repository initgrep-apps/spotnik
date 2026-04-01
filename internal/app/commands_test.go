package app_test

// commands_test.go — Tests for search command enrichment (story 81).
//
// These tests verify that convertSearchResult populates all enriched fields
// and that buildSearchCmd uses limit=10.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertSearchResult_EnrichedFields verifies that buildSearchCmd/convertSearchResult
// populates all enriched fields: Album, DurationMs, ReleaseYear, TotalTracks, TrackCount,
// and all Total* counts.
func TestConvertSearchResult_EnrichedFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"tracks": map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id":          "t1",
						"name":        "Blinding Lights",
						"uri":         "spotify:track:t1",
						"duration_ms": 200040,
						"artists":     []map[string]interface{}{{"name": "The Weeknd"}},
						"album":       map[string]interface{}{"name": "After Hours"},
					},
				},
				"total": 100,
			},
			"artists": map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": "a1", "name": "The Weeknd", "uri": "spotify:artist:a1"},
				},
				"total": 50,
			},
			"albums": map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id":           "al1",
						"name":         "After Hours",
						"uri":          "spotify:album:al1",
						"total_tracks": 14,
						"release_date": "2020-03-20",
						"artists":      []map[string]interface{}{{"name": "The Weeknd"}},
					},
				},
				"total": 30,
			},
			"playlists": map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id":   "pl1",
						"name": "Chill Vibes",
						"uri":  "spotify:playlist:pl1",
						"owner": map[string]interface{}{
							"id":           "user1",
							"display_name": "User",
						},
						"tracks": map[string]interface{}{"total": 45},
					},
				},
				"total": 20,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.SearchRequestMsg{Query: "test"})
	require.NotNil(t, cmd, "SearchRequestMsg should return a command")

	msg := cmd()
	searchMsg, ok := msg.(panes.SearchResultsMsg)
	require.True(t, ok, "command should return SearchResultsMsg, got %T", msg)
	require.NotNil(t, searchMsg.Results, "results should be non-nil")

	results := searchMsg.Results

	// Track enriched fields
	require.Len(t, results.Tracks, 1)
	assert.Equal(t, "After Hours", results.Tracks[0].Album, "track Album should be populated")
	assert.Equal(t, 200040, results.Tracks[0].DurationMs, "track DurationMs should be populated")

	// Artist fields (no enrichment — just verify still works)
	require.Len(t, results.Artists, 1)
	assert.Equal(t, "The Weeknd", results.Artists[0].Name)

	// Album enriched fields
	require.Len(t, results.Albums, 1)
	assert.Equal(t, "2020", results.Albums[0].ReleaseYear, "album ReleaseYear should be the first 4 chars")
	assert.Equal(t, 14, results.Albums[0].TotalTracks, "album TotalTracks should be populated")

	// Playlist enriched field
	require.Len(t, results.Playlists, 1)
	assert.Equal(t, 45, results.Playlists[0].TrackCount, "playlist TrackCount should be populated")

	// Total counts
	assert.Equal(t, 100, results.TotalTracks, "TotalTracks should be populated from API")
	assert.Equal(t, 50, results.TotalArtists, "TotalArtists should be populated from API")
	assert.Equal(t, 30, results.TotalAlbums, "TotalAlbums should be populated from API")
	assert.Equal(t, 20, results.TotalPlaylists, "TotalPlaylists should be populated from API")
}

// TestConvertSearchResult_ShortReleaseDate verifies that a short ReleaseDate (< 4 chars)
// does not cause a panic — ReleaseYear should be empty or the date itself.
func TestConvertSearchResult_ShortReleaseDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"tracks":  map[string]interface{}{"items": []interface{}{}, "total": 0},
			"artists": map[string]interface{}{"items": []interface{}{}, "total": 0},
			"albums": map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id":           "al1",
						"name":         "Short Date Album",
						"uri":          "spotify:album:al1",
						"total_tracks": 5,
						// Intentionally short: less than 4 chars — should not panic
						"release_date": "20",
						"artists":      []map[string]interface{}{{"name": "Artist"}},
					},
				},
				"total": 1,
			},
			"playlists": map[string]interface{}{"items": []interface{}{}, "total": 0},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.SearchRequestMsg{Query: "short"})
	require.NotNil(t, cmd)

	// This should NOT panic.
	msg := cmd()
	searchMsg, ok := msg.(panes.SearchResultsMsg)
	require.True(t, ok, "should return SearchResultsMsg")
	require.NotNil(t, searchMsg.Results)

	// ReleaseYear should be empty (guard: len < 4 → empty string)
	require.Len(t, searchMsg.Results.Albums, 1)
	assert.Empty(t, searchMsg.Results.Albums[0].ReleaseYear, "short ReleaseDate should yield empty ReleaseYear")
}

// TestSearchPlaylist_UnmarshalJSON_TracksTotal verifies that SearchPlaylist.UnmarshalJSON
// correctly extracts tracks.total from a realistic Spotify search response fixture.
func TestSearchPlaylist_UnmarshalJSON_TracksTotal(t *testing.T) {
	// Real Spotify search API response shape for a playlist item (truncated to essentials).
	jsonData := []byte(`{
		"id": "pl1",
		"name": "Chill Vibes",
		"uri": "spotify:playlist:pl1",
		"owner": {"id": "user1", "display_name": "User"},
		"tracks": {"href": "https://api.spotify.com/...", "total": 45}
	}`)

	var pl api.SearchPlaylist
	require.NoError(t, json.Unmarshal(jsonData, &pl), "should unmarshal without error")
	assert.Equal(t, 45, pl.TrackCount, "TrackCount should be extracted from nested tracks.total")
}

// TestConvertSearchResult_PlaylistTrackCount verifies the end-to-end path from API
// response to UI type preserves the playlist TrackCount.
func TestConvertSearchResult_PlaylistTrackCount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"tracks":  map[string]interface{}{"items": []interface{}{}, "total": 0},
			"artists": map[string]interface{}{"items": []interface{}{}, "total": 0},
			"albums":  map[string]interface{}{"items": []interface{}{}, "total": 0},
			"playlists": map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id":    "pl1",
						"name":  "My Playlist",
						"uri":   "spotify:playlist:pl1",
						"owner": map[string]interface{}{"id": "u1", "display_name": "Owner1"},
						// Spotify search API returns tracks as a simplified object with href + total.
						"tracks": map[string]interface{}{"href": "https://api.spotify.com/v1/playlists/pl1/tracks", "total": 99},
					},
				},
				"total": 1,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.SearchRequestMsg{Query: "test"})
	require.NotNil(t, cmd)
	msg := cmd()
	searchMsg, ok := msg.(panes.SearchResultsMsg)
	require.True(t, ok)
	require.NotNil(t, searchMsg.Results)
	require.Len(t, searchMsg.Results.Playlists, 1)
	assert.Equal(t, 99, searchMsg.Results.Playlists[0].TrackCount,
		"playlist TrackCount should be 99 from the nested tracks.total field in the API response")
}

// TestBuildSearchCmd_Limit10 verifies that buildSearchCmd passes limit=10 to the search API.
func TestBuildSearchCmd_Limit10(t *testing.T) {
	var capturedLimit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the limit query parameter
		capturedLimit = r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tracks":{"items":[],"total":0},
			"artists":{"items":[],"total":0},
			"albums":{"items":[],"total":0},
			"playlists":{"items":[],"total":0}
		}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.SearchRequestMsg{Query: "test"})
	require.NotNil(t, cmd)
	cmd() // execute to trigger HTTP call

	assert.Equal(t, "10", capturedLimit, "buildSearchCmd should pass limit=10 to the search API")
}

// TestBuildSearchCmd_WithOffset verifies that buildSearchCmd passes the given offset
// to the search API so paginated requests fetch the correct result page.
func TestBuildSearchCmd_WithOffset(t *testing.T) {
	var capturedOffset string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedOffset = r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tracks":{"items":[],"total":0},
			"artists":{"items":[],"total":0},
			"albums":{"items":[],"total":0},
			"playlists":{"items":[],"total":0}
		}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Directly call buildSearchCmd with offset=10 to simulate a page-2 request.
	cmd := a.BuildSearchCmd("test", 10)
	require.NotNil(t, cmd)
	cmd() // execute to trigger HTTP call

	assert.Equal(t, "10", capturedOffset, "buildSearchCmd should pass the given offset to the search API")
}

// TestApp_SearchPageRequestMsg_DispatchesSearch verifies that the root app dispatches
// a search command carrying the correct offset when it receives a SearchPageRequestMsg.
func TestApp_SearchPageRequestMsg_DispatchesSearch(t *testing.T) {
	var capturedOffset string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedOffset = r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tracks":{"items":[],"total":0},
			"artists":{"items":[],"total":0},
			"albums":{"items":[],"total":0},
			"playlists":{"items":[],"total":0}
		}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Emit a page request for offset=10 (page 2).
	_, cmd := a.Update(panes.SearchPageRequestMsg{
		Query:   "test",
		Offset:  10,
		Section: panes.SectionTracks,
	})
	require.NotNil(t, cmd, "SearchPageRequestMsg should produce a search command")

	msg := cmd() // execute to trigger HTTP call
	searchMsg, ok := msg.(panes.SearchResultsMsg)
	require.True(t, ok, "search command should return SearchResultsMsg, got %T", msg)
	assert.Equal(t, 10, searchMsg.Offset, "SearchResultsMsg should carry the requested offset")
	assert.Equal(t, panes.SectionTracks, searchMsg.Section, "SearchResultsMsg should carry the requesting section")
	assert.Equal(t, "10", capturedOffset, "HTTP request should include offset=10")
}

// TestApp_SearchPageCmd_FetchesSingleType verifies that buildSearchPageCmd sends only
// the type string for the requested section, not all four types.
// This matches the spec requirement that "each type is paginated independently."
func TestApp_SearchPageCmd_FetchesSingleType(t *testing.T) {
	tests := []struct {
		name     string
		section  panes.SearchSection
		wantType string
	}{
		{"tracks section", panes.SectionTracks, "track"},
		{"artists section", panes.SectionArtists, "artist"},
		{"albums section", panes.SectionAlbums, "album"},
		{"playlists section", panes.SectionPlaylists, "playlist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedType string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedType = r.URL.Query().Get("type")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"tracks":{"items":[],"total":0},
					"artists":{"items":[],"total":0},
					"albums":{"items":[],"total":0},
					"playlists":{"items":[],"total":0}
				}`))
			}))
			defer srv.Close()

			cfg := &config.Config{}
			a := app.New(cfg, app.AppOptions{})
			a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

			_, cmd := a.Update(panes.SearchPageRequestMsg{
				Query:   "test",
				Offset:  10,
				Section: tt.section,
			})
			require.NotNil(t, cmd)
			cmd() // trigger HTTP call

			assert.Equal(t, tt.wantType, capturedType,
				"buildSearchPageCmd should fetch only the type for section %v", tt.section)
		})
	}
}
