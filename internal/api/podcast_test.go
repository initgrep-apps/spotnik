package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestPodcastClient creates a PodcastClient pointing at the given base URL.
func newTestPodcastClient(baseURL, token string) *PodcastClient {
	return NewPodcastClient(baseURL, token)
}

func TestPodcastClient_Show_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/shows/show-123", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "from_token", r.URL.Query().Get("market"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "show-123",
			"name": "Test Show",
			"publisher": "Test Publisher",
			"description": "A test show",
			"total_episodes": 0,
			"images": [],
			"media_type": "audio",
			"explicit": false,
			"episodes": {"items": [{"id": "ep-1", "name": "Episode 1", "duration_ms": 1800000, "release_date": "2024-01-15", "is_playable": true, "is_externally_hosted": false, "language": "en", "uri": "spotify:episode:ep-1", "resume_point": {"fully_played": false, "resume_position_ms": 0}, "restrictions": {"reason": ""}}], "total": 1}
		}`))
	}))
	defer srv.Close()

	client := newTestPodcastClient(srv.URL, "test-token")
	show, err := client.Show(context.Background(), "show-123")
	require.NoError(t, err)
	require.NotNil(t, show)
	assert.Equal(t, "show-123", show.ID)
	assert.Equal(t, "Test Show", show.Name)
	assert.Equal(t, "Test Publisher", show.Publisher)
	assert.Equal(t, 1, show.TotalEpisodes)
}

func TestPodcastClient_Show_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "server error"}`))
	}))
	defer srv.Close()

	client := newTestPodcastClient(srv.URL, "test-token")
	show, err := client.Show(context.Background(), "show-123")
	require.Error(t, err)
	assert.Nil(t, show)
	assert.Contains(t, err.Error(), "500")
}

func TestPodcastClient_ShowEpisodes_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/shows/show-123/episodes", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "from_token", r.URL.Query().Get("market"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "0", r.URL.Query().Get("offset"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [
				{"id": "ep-1", "name": "Episode 1", "duration_ms": 1800000, "release_date": "2024-01-15", "is_playable": true, "is_externally_hosted": false, "language": "en", "uri": "spotify:episode:ep-1", "resume_point": {"fully_played": false, "resume_position_ms": 0}, "restrictions": {"reason": ""}},
				{"id": "ep-2", "name": "Episode 2", "duration_ms": 1200000, "release_date": "2024-01-22", "is_playable": true, "is_externally_hosted": false, "language": "en", "uri": "spotify:episode:ep-2", "resume_point": {"fully_played": true, "resume_position_ms": 1200000}, "restrictions": {"reason": ""}}
			],
			"total": 15
		}`))
	}))
	defer srv.Close()

	client := newTestPodcastClient(srv.URL, "test-token")
	episodes, total, hasNext, err := client.ShowEpisodes(context.Background(), "show-123", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 15, total)
	assert.True(t, hasNext)
	require.Len(t, episodes, 2)
	assert.Equal(t, "ep-1", episodes[0].ID)
	assert.Equal(t, "Episode 1", episodes[0].Name)
	assert.Equal(t, "ep-2", episodes[1].ID)
}

func TestPodcastClient_ShowEpisodes_LastPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [
				{"id": "ep-3", "name": "Episode 3", "duration_ms": 900000, "release_date": "2024-02-01", "is_playable": true, "is_externally_hosted": false, "language": "en", "uri": "spotify:episode:ep-3", "resume_point": {"fully_played": false, "resume_position_ms": 0}, "restrictions": {"reason": ""}}
			],
			"total": 11
		}`))
	}))
	defer srv.Close()

	client := newTestPodcastClient(srv.URL, "test-token")
	episodes, total, hasNext, err := client.ShowEpisodes(context.Background(), "show-123", 10, 10)
	require.NoError(t, err)
	assert.Equal(t, 11, total)
	assert.False(t, hasNext, "offset 10 + limit 10 = 20 which exceeds total 11, so no next page")
	require.Len(t, episodes, 1)
}

func TestPodcastClient_ShowEpisodes_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestPodcastClient(srv.URL, "test-token")
	episodes, total, hasNext, err := client.ShowEpisodes(context.Background(), "show-123", 10, 0)
	require.Error(t, err)
	assert.Nil(t, episodes)
	assert.Equal(t, 0, total)
	assert.False(t, hasNext)
}

func TestPodcastClient_FollowedShows_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/shows", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "20", r.URL.Query().Get("limit"))
		assert.Equal(t, "0", r.URL.Query().Get("offset"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [
				{
					"added_at": "2024-01-01T00:00:00Z",
					"show": {"id": "show-1", "name": "Saved Show", "publisher": "Publisher", "description": "A saved show", "total_episodes": 10, "images": [], "media_type": "audio", "explicit": false}
				}
			],
			"total": 1
		}`))
	}))
	defer srv.Close()

	client := newTestPodcastClient(srv.URL, "test-token")
	shows, err := client.FollowedShows(context.Background(), 20, 0)
	require.NoError(t, err)
	require.Len(t, shows, 1)
	assert.Equal(t, "2024-01-01T00:00:00Z", shows[0].AddedAt)
	assert.Equal(t, "show-1", shows[0].Show.ID)
	assert.Equal(t, "Saved Show", shows[0].Show.Name)
}

func TestPodcastClient_Episode_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/episodes/ep-123", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "from_token", r.URL.Query().Get("market"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "ep-123",
			"name": "Single Episode",
			"description": "Fetched directly",
			"duration_ms": 900000,
			"release_date": "2024-03-01",
			"explicit": false,
			"is_playable": true,
			"is_externally_hosted": false,
			"audio_preview_url": "",
			"language": "en",
			"uri": "spotify:episode:ep-123",
			"show": {"id": "show-1", "name": "My Show"},
			"resume_point": {"fully_played": true, "resume_position_ms": 900000},
			"restrictions": {"reason": ""}
		}`))
	}))
	defer srv.Close()

	client := newTestPodcastClient(srv.URL, "test-token")
	ep, err := client.Episode(context.Background(), "ep-123")
	require.NoError(t, err)
	require.NotNil(t, ep)
	assert.Equal(t, "ep-123", ep.ID)
	assert.Equal(t, "Single Episode", ep.Name)
	assert.Equal(t, 900000, ep.DurationMs)
	require.NotNil(t, ep.Show)
	assert.Equal(t, "show-1", ep.Show.ID)
	assert.True(t, ep.ResumePoint.FullyPlayed)
}

func TestPodcastClient_SavedEpisodes_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/episodes", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "0", r.URL.Query().Get("offset"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [
				{
					"added_at": "2024-02-15T00:00:00Z",
					"episode": {"id": "ep-saved-1", "name": "Saved Episode", "description": "A saved episode", "duration_ms": 1800000, "release_date": "2024-02-10", "explicit": false, "is_playable": true, "is_externally_hosted": false, "audio_preview_url": "", "language": "en", "uri": "spotify:episode:ep-saved-1", "show": {"id": "show-1", "name": "My Show"}, "resume_point": {"fully_played": false, "resume_position_ms": 300000}, "restrictions": {"reason": ""}}
				}
			],
			"total": 1
		}`))
	}))
	defer srv.Close()

	client := newTestPodcastClient(srv.URL, "test-token")
	saved, err := client.SavedEpisodes(context.Background(), 10, 0)
	require.NoError(t, err)
	require.Len(t, saved, 1)
	assert.Equal(t, "2024-02-15T00:00:00Z", saved[0].AddedAt)
	assert.Equal(t, "ep-saved-1", saved[0].Episode.ID)
	assert.Equal(t, "Saved Episode", saved[0].Episode.Name)
	assert.Equal(t, 300000, saved[0].Episode.ResumePoint.ResumePositionMs)
}

func TestPodcastClient_SavedEpisodes_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := newTestPodcastClient(srv.URL, "test-token")
	saved, err := client.SavedEpisodes(context.Background(), 10, 0)
	require.Error(t, err)
	assert.Nil(t, saved)
	var rateLimitErr *RateLimitError
	assert.ErrorAs(t, err, &rateLimitErr)
	assert.Equal(t, 3, rateLimitErr.RetryAfter)
}

func TestPodcastClient_AuthorizationHeader(t *testing.T) {
	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"show-1","name":"Show","description":"","total_episodes":0,"images":[],"media_type":"audio","explicit":false,"episodes":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	client := newTestPodcastClient(srv.URL, "my-access-token")
	_, _ = client.Show(context.Background(), "show-1")
	assert.Equal(t, "Bearer my-access-token", capturedAuth)
}
