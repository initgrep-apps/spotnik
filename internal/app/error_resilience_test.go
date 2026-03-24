package app_test

// error_resilience_test.go — Tests for Feature 27: Error Resilience
//
// Task 2: All build*Cmd functions emit RateLimitedMsg on 429
// Task 3: AddToQueueResultMsg handler checks ForbiddenError

import (
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

// rateLimitServer returns an httptest.Server that always responds with 429
// and a Retry-After header set to retryAfter seconds.
func rateLimitServer(retryAfter string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
}

// forbiddenServer returns an httptest.Server that always responds with 403.
func forbiddenServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Spotify Premium required"))
	}))
}

// --- Task 2: 429 backoff extension ---

// TestBuildFetchPlaylistsCmd_429_EmitsRateLimitedMsg verifies that a 429 from
// the playlists endpoint causes buildFetchPlaylistsCmd to return RateLimitedMsg.
func TestBuildFetchPlaylistsCmd_429_EmitsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("15")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	require.NotNil(t, cmd)

	msg := cmd()
	rateLimitMsg, ok := msg.(panes.RateLimitedMsg)
	assert.True(t, ok, "429 from playlists endpoint should emit RateLimitedMsg, got %T", msg)
	assert.Equal(t, 15, rateLimitMsg.RetryAfterSecs, "RetryAfterSecs should match Retry-After header")
}

// TestBuildFetchAlbumsCmd_429_EmitsRateLimitedMsg verifies that a 429 from
// the albums endpoint causes buildFetchAlbumsCmd to return RateLimitedMsg.
func TestBuildFetchAlbumsCmd_429_EmitsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("20")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchAlbumsRequestMsg{Offset: 0})
	require.NotNil(t, cmd)

	msg := cmd()
	rateLimitMsg, ok := msg.(panes.RateLimitedMsg)
	assert.True(t, ok, "429 from albums endpoint should emit RateLimitedMsg, got %T", msg)
	assert.Equal(t, 20, rateLimitMsg.RetryAfterSecs)
}

// TestBuildFetchLikedTracksCmd_429_EmitsRateLimitedMsg verifies that a 429 from
// the liked tracks endpoint causes buildFetchLikedTracksCmd to return RateLimitedMsg.
func TestBuildFetchLikedTracksCmd_429_EmitsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("10")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchLikedTracksRequestMsg{Offset: 0})
	require.NotNil(t, cmd)

	msg := cmd()
	rateLimitMsg, ok := msg.(panes.RateLimitedMsg)
	assert.True(t, ok, "429 from liked tracks endpoint should emit RateLimitedMsg, got %T", msg)
	assert.Equal(t, 10, rateLimitMsg.RetryAfterSecs)
}

// TestBuildFetchRecentlyPlayedCmd_429_EmitsRateLimitedMsg verifies that a 429 from
// the recently played endpoint causes buildFetchRecentlyPlayedCmd to return RateLimitedMsg.
func TestBuildFetchRecentlyPlayedCmd_429_EmitsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("8")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchRecentlyPlayedRequestMsg{})
	require.NotNil(t, cmd)

	msg := cmd()
	rateLimitMsg, ok := msg.(panes.RateLimitedMsg)
	assert.True(t, ok, "429 from recently played endpoint should emit RateLimitedMsg, got %T", msg)
	assert.Equal(t, 8, rateLimitMsg.RetryAfterSecs)
}

// TestBuildSearchCmd_429_EmitsRateLimitedMsg verifies that a 429 from
// the search endpoint causes buildSearchCmd to return RateLimitedMsg.
func TestBuildSearchCmd_429_EmitsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("12")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.SearchRequestMsg{Query: "beatles"})
	require.NotNil(t, cmd)

	msg := cmd()
	rateLimitMsg, ok := msg.(panes.RateLimitedMsg)
	assert.True(t, ok, "429 from search endpoint should emit RateLimitedMsg, got %T", msg)
	assert.Equal(t, 12, rateLimitMsg.RetryAfterSecs)
}

// TestBuildFetchDevicesCmd_429_EmitsRateLimitedMsg verifies that a 429 from
// the devices endpoint causes buildFetchDevicesCmd to return RateLimitedMsg.
func TestBuildFetchDevicesCmd_429_EmitsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("5")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetDevices(api.NewDevicesClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchDevicesRequestMsg{})
	require.NotNil(t, cmd)

	msg := cmd()
	rateLimitMsg, ok := msg.(panes.RateLimitedMsg)
	assert.True(t, ok, "429 from devices endpoint should emit RateLimitedMsg, got %T", msg)
	assert.Equal(t, 5, rateLimitMsg.RetryAfterSecs)
}

// TestBuildFetchStatsCmd_429_EmitsRateLimitedMsg verifies that a 429 from
// the stats endpoint causes buildFetchStatsCmd to return RateLimitedMsg.
func TestBuildFetchStatsCmd_429_EmitsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("30")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetUserAPI(api.NewUserClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchStatsMsg{TimeRange: "short_term"})
	require.NotNil(t, cmd)

	msg := cmd()
	rateLimitMsg, ok := msg.(panes.RateLimitedMsg)
	assert.True(t, ok, "429 from stats endpoint should emit RateLimitedMsg, got %T", msg)
	assert.Equal(t, 30, rateLimitMsg.RetryAfterSecs)
}

// TestBuildFetchPlaylistTracksCmd_429_EmitsRateLimitedMsg verifies that a 429 from
// the playlist tracks endpoint causes buildFetchPlaylistTracksCmd to return RateLimitedMsg.
func TestBuildFetchPlaylistTracksCmd_429_EmitsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("7")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchPlaylistTracksRequestMsg{PlaylistID: "pl123"})
	require.NotNil(t, cmd)

	msg := cmd()
	rateLimitMsg, ok := msg.(panes.RateLimitedMsg)
	assert.True(t, ok, "429 from playlist tracks endpoint should emit RateLimitedMsg, got %T", msg)
	assert.Equal(t, 7, rateLimitMsg.RetryAfterSecs)
}

// TestApp_RateLimitedMsg_ActivatesBackoff verifies that a RateLimitedMsg returned
// from any command is handled by Update and activates the backoff mechanism.
// This is the integration test showing the full loop works.
func TestApp_RateLimitedMsg_ActivatesBackoff(t *testing.T) {
	srv := rateLimitServer("25")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	// Execute the command that returns RateLimitedMsg.
	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	require.NotNil(t, cmd)
	msg := cmd()

	// Feed the RateLimitedMsg back to Update — it should activate backoff.
	model, _ := a.Update(msg)
	a = model.(*app.App)

	// Backoff must be active — the default minimum is 10 ticks.
	assert.Greater(t, a.BackoffTicks(), 0, "RateLimitedMsg from any API should activate backoff")
}

// --- Task 3: 403 handling for AddToQueueResultMsg ---

// TestApp_AddToQueueResultMsg_ForbiddenError_ShowsPremiumMessage verifies that
// a 403 ForbiddenError from AddToQueue shows the ForbiddenError.Message directly,
// not the full "forbidden: ..." formatted string from ForbiddenError.Error().
func TestApp_AddToQueueResultMsg_ForbiddenError_ShowsPremiumMessage(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Simulate the message returned by buildAddToQueueCmd when the API 403s.
	forbiddenErr := &api.ForbiddenError{Message: "Spotify Premium required"}
	resultMsg := panes.AddToQueueResultMsg{Err: forbiddenErr, TrackName: "Song"}

	model, cmd := a.Update(resultMsg)
	require.NotNil(t, model)
	assert.NotNil(t, cmd, "error result should schedule dismiss timer")

	appModel := model.(*app.App)
	output := appModel.View()
	// Should show "Spotify Premium required" from ForbiddenError.Message,
	// NOT the raw "forbidden: Spotify Premium required" from ForbiddenError.Error().
	assert.Contains(t, output, "Spotify Premium required", "status bar should show ForbiddenError.Message on 403")
	assert.NotContains(t, output, "forbidden:", "status bar should NOT use the raw ForbiddenError.Error() prefix")
}

// TestApp_AddToQueueResultMsg_ForbiddenError_WithLiveServer verifies the end-to-end
// flow: a 403 from the actual HTTP call reaches the handler and shows a Premium message.
func TestApp_AddToQueueResultMsg_ForbiddenError_WithLiveServer(t *testing.T) {
	srv := forbiddenServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetPlayer(api.NewPlayer(srv.URL, "test-token"))

	// Send AddToQueueMsg — this builds a command that will hit the 403 server.
	_, cmd := a.Update(panes.AddToQueueMsg{TrackURI: "spotify:track:abc", TrackName: "Test Song"})
	require.NotNil(t, cmd)

	// Execute the command to get the result message.
	resultMsg := cmd()
	addToQueueResult, ok := resultMsg.(panes.AddToQueueResultMsg)
	require.True(t, ok, "expected AddToQueueResultMsg, got %T", resultMsg)
	require.Error(t, addToQueueResult.Err)

	// Feed the result back to Update — should show Premium message.
	model, _ := a.Update(resultMsg)
	appModel := model.(*app.App)
	output := appModel.View()
	assert.Contains(t, output, "Premium", "403 AddToQueue should show Premium message in status bar")
}
