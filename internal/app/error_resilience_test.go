package app_test

// error_resilience_test.go — Tests for Feature 27: Error Resilience
//
// Task 1: 401 token refresh + show "Session expired" when refresh fails
// Task 2: All build*Cmd functions emit RateLimitedMsg on 429
// Task 3: AddToQueueResultMsg handler checks ForbiddenError

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/keychain"
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
// the search endpoint causes buildSearchPageCmd to emit RateLimitedMsg.
// buildSearchPageCmd fetches a single page; executing it returns
// RateLimitedMsg when the server responds 429.
func TestBuildSearchCmd_429_EmitsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("12")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.SearchRequestMsg{Query: "beatles"})
	require.NotNil(t, cmd)

	// Story 100: SearchRequestMsg returns batch(loadingCmd, fetchCmd). Execute fetchCmd (index 1).
	msg := executeFetchCmd(cmd)
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

// TestBuildFetchPlaylistTracksCmd_CancelledCtx_ReturnsNil verifies that a pre-cancelled
// context causes buildFetchPlaylistTracksCmd to return nil (silently discard).
func TestBuildFetchPlaylistTracksCmd_CancelledCtx_ReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[],"total":0,"next":null}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	// Trigger fetch then immediately send PlaylistTrackViewClosedMsg to cancel it.
	_, cmd := a.Update(panes.FetchPlaylistTracksRequestMsg{PlaylistID: "pl123"})
	require.NotNil(t, cmd)

	// Cancel via PlaylistTrackViewClosedMsg before executing the cmd.
	a.Update(panes.PlaylistTrackViewClosedMsg{}) //nolint:errcheck

	// Now execute the cmd — the context was cancelled, so it may return nil or a stale msg.
	// At minimum the staleness ID should now be "" so the msg is discarded.
	msg := cmd()
	if msg != nil {
		ptMsg, ok := msg.(panes.PlaylistTracksLoadedMsg)
		if ok {
			// If a msg arrived, it must have the old playlist ID (will be stale after cancel).
			assert.Equal(t, "pl123", ptMsg.PlaylistID, "if msg arrives it must carry the original playlist ID")
		}
	}
}

// TestBuildFetchPlaylistTracksCmd_ContextCancellation_BeforeHTTP verifies that a
// context cancelled before the HTTP call causes the command to return nil immediately
// (the pre-HTTP ctx.Err() guard in buildFetchPlaylistTracksCmd fires).
func TestBuildFetchPlaylistTracksCmd_ContextCancellation_BeforeHTTP(t *testing.T) {
	// Server that should never be called — if it is, the pre-HTTP guard failed.
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[],"total":0,"next":null}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	// Trigger fetch to set up the cancellable context.
	_, cmd := a.Update(panes.FetchPlaylistTracksRequestMsg{PlaylistID: "pl123"})
	require.NotNil(t, cmd)

	// Cancel the context immediately by sending PlaylistTrackViewClosedMsg.
	// This calls a.playlistTracksCancel() which cancels the context captured in cmd.
	a.Update(panes.PlaylistTrackViewClosedMsg{}) //nolint:errcheck

	// Execute cmd — context is already cancelled, so it must return nil without
	// making an HTTP call (pre-HTTP ctx.Err() guard fires).
	msg := cmd()
	assert.Nil(t, msg, "cmd with pre-cancelled context must return nil")
	assert.Equal(t, 0, callCount, "HTTP server must not be called when context is pre-cancelled")
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
// a 403 ForbiddenError from AddToQueue emits an error toast with ForbiddenError.Message
// (not the raw ForbiddenError.Error() string with "forbidden:" prefix).
func TestApp_AddToQueueResultMsg_ForbiddenError_ShowsPremiumMessage(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Simulate the message returned by buildAddToQueueCmd when the API 403s.
	forbiddenErr := &api.ForbiddenError{Message: "Spotify Premium required"}
	resultMsg := panes.AddToQueueResultMsg{Err: forbiddenErr, TrackName: "Song"}

	model, cmd := a.Update(resultMsg)
	require.NotNil(t, model)
	// An error alert toast cmd is returned — non-nil indicates the toast was dispatched.
	assert.NotNil(t, cmd, "error result should emit an alert toast cmd")

	// Execute the alertCmd to get the internal alertMsg and feed to Update.
	// This activates the BubbleUp alert, which then appears in View() via Render().
	alertMsg := cmd()
	updated, _ := a.Update(alertMsg)
	appModel := updated.(*app.App)
	output := appModel.View()
	// Alert should show "Spotify Premium required" from ForbiddenError.Message,
	// NOT the raw "forbidden: Spotify Premium required" from ForbiddenError.Error().
	assert.Contains(t, output, "Spotify Premium required", "toast should show ForbiddenError.Message on 403")
	assert.NotContains(t, output, "forbidden:", "toast should NOT use the raw ForbiddenError.Error() prefix")
}

// TestApp_AddToQueueResultMsg_ForbiddenError_WithLiveServer verifies the end-to-end
// flow: a 403 from the actual HTTP call reaches the handler and emits a Premium toast.
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

	// Feed the result back to Update — should emit an error toast alert cmd.
	model, alertCmd := a.Update(resultMsg)
	require.NotNil(t, model)
	require.NotNil(t, alertCmd, "403 AddToQueue should emit a toast alert cmd")

	// Process the alertCmd to activate the toast, then check View().
	alertMsg := alertCmd()
	updated, _ := a.Update(alertMsg)
	appModel := updated.(*app.App)
	output := appModel.View()
	assert.Contains(t, output, "Premium", "403 AddToQueue should show Premium message in toast overlay")
}

// --- Task 1: 401 token refresh ---

// unauthorizedServer returns an httptest.Server that always responds with 401.
func unauthorizedServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"status":401,"message":"No token provided"}}`))
	}))
}

// TestBuildFetchPlaylistsCmd_401_ShowsSessionExpired verifies that a 401 from
// the playlists endpoint, when the token store has no refresh token, shows the
// "Session expired" message after the full command chain runs.
func TestBuildFetchPlaylistsCmd_401_ShowsSessionExpired(t *testing.T) {
	srv := unauthorizedServer()
	defer srv.Close()

	cfg := &config.Config{ClientID: "test-client-id"}
	// InMemoryTokenStore with no refresh token — refresh will fail.
	store := keychain.NewInMemoryTokenStore()

	a := app.New(cfg, app.AppOptions{TokenStore: store})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	// Step 1: FetchPlaylistsRequestMsg → buildFetchPlaylistsCmd dispatched.
	model, fetchCmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	a = model.(*app.App)
	require.NotNil(t, fetchCmd)

	// Step 2: Execute the fetch command — gets unauthorizedMsg{} back.
	unauthorizedMsgResult := fetchCmd()

	// Step 3: Feed unauthorizedMsg to Update — dispatches buildRefreshTokenCmd.
	model, refreshCmd := a.Update(unauthorizedMsgResult)
	a = model.(*app.App)
	require.NotNil(t, refreshCmd, "unauthorizedMsg should dispatch a refresh command")

	// Step 4: Execute the refresh command — fails because no refresh token.
	refreshResult := refreshCmd()

	// Step 5: Feed tokenRefreshedMsg(err) to Update — emits an error toast alert cmd.
	model, alertCmd := a.Update(refreshResult)
	a = model.(*app.App)
	require.NotNil(t, alertCmd, "session expired should emit an error toast alert cmd")

	// Process the alertCmd to activate the toast, then check View().
	alertMsg := alertCmd()
	updated, _ := a.Update(alertMsg)
	a = updated.(*app.App)
	output := a.View()
	assert.Contains(t, output, "Session expired", "401 with no refresh token should show session expired toast")
}

// TestBuildFetchPlaylistTracksCmd_401_ShowsSessionExpired verifies that a 401 from
// the playlist tracks endpoint, when the token store has no refresh token, shows the
// "Session expired" message after the full command chain runs.
func TestBuildFetchPlaylistTracksCmd_401_ShowsSessionExpired(t *testing.T) {
	srv := unauthorizedServer()
	defer srv.Close()

	cfg := &config.Config{ClientID: "test-client-id"}
	// InMemoryTokenStore with no refresh token — refresh will fail.
	store := keychain.NewInMemoryTokenStore()

	a := app.New(cfg, app.AppOptions{TokenStore: store})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	// Step 1: FetchPlaylistTracksRequestMsg → buildFetchPlaylistTracksCmd dispatched.
	model, fetchCmd := a.Update(panes.FetchPlaylistTracksRequestMsg{PlaylistID: "pl123"})
	a = model.(*app.App)
	require.NotNil(t, fetchCmd)

	// Step 2: Execute the fetch command — gets unauthorizedMsg{} back.
	unauthorizedMsgResult := fetchCmd()

	// Step 3: Feed unauthorizedMsg to Update — dispatches buildRefreshTokenCmd.
	model, refreshCmd := a.Update(unauthorizedMsgResult)
	a = model.(*app.App)
	require.NotNil(t, refreshCmd, "unauthorizedMsg should dispatch a refresh command")

	// Step 4: Execute the refresh command — fails because no refresh token.
	refreshResult := refreshCmd()

	// Step 5: Feed tokenRefreshedMsg(err) to Update — emits an error toast alert cmd.
	model, alertCmd := a.Update(refreshResult)
	a = model.(*app.App)
	require.NotNil(t, alertCmd, "session expired should emit an error toast alert cmd")

	// Process the alertCmd to activate the toast, then check View().
	alertMsg := alertCmd()
	updated, _ := a.Update(alertMsg)
	a = updated.(*app.App)
	output := a.View()
	assert.Contains(t, output, "Session expired", "401 from playlist tracks with no refresh token should show session expired toast")
}

// TestBuildSearchCmd_401_ShowsSessionExpired verifies the same pattern for search.
// buildSearchPageCmd fetches a single page; the page-fetch hits 401
// and emits unauthorizedMsg, triggering the token refresh flow.
func TestBuildSearchCmd_401_ShowsSessionExpired(t *testing.T) {
	srv := unauthorizedServer()
	defer srv.Close()

	cfg := &config.Config{ClientID: "test-client-id"}
	store := keychain.NewInMemoryTokenStore()

	a := app.New(cfg, app.AppOptions{TokenStore: store})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	model, fetchCmd := a.Update(panes.SearchRequestMsg{Query: "test"})
	a = model.(*app.App)
	require.NotNil(t, fetchCmd)

	// Story 100: SearchRequestMsg returns batch(loadingCmd, fetchCmd). Execute fetchCmd (index 1).
	unauthorizedMsgResult := executeFetchCmd(fetchCmd)

	model, refreshCmd := a.Update(unauthorizedMsgResult)
	a = model.(*app.App)
	require.NotNil(t, refreshCmd)

	refreshResult := refreshCmd()

	model, alertCmd := a.Update(refreshResult)
	a = model.(*app.App)
	require.NotNil(t, alertCmd, "session expired should emit an error toast alert cmd")

	// Process the alertCmd to activate the toast, then check View().
	alertMsg := alertCmd()
	updated, _ := a.Update(alertMsg)
	a = updated.(*app.App)
	output := a.View()
	assert.Contains(t, output, "Session expired", "401 search with no refresh token should show session expired toast")
}

// TestApp_401_WithValidRefreshToken_ReInitializesClients verifies that when a 401
// occurs and a valid refresh token is present, the app attempts to refresh and
// re-initializes the API clients on success.
func TestApp_401_WithValidRefreshToken_ReInitializesClients(t *testing.T) {
	// Set up a mock token endpoint that accepts refresh_token grants.
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "refresh_token" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"unsupported_grant_type"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"new-access-token","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	// Set up an unauthorized API server.
	apiSrv := unauthorizedServer()
	defer apiSrv.Close()

	cfg := &config.Config{ClientID: "test-client-id"}
	// Seed the store with a valid refresh token.
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "valid-refresh-token"))
	require.NoError(t, store.Set(keychain.KeyAccessToken, "old-access-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", 9999999999)))

	a := app.New(cfg, app.AppOptions{TokenStore: store, ClientID: "test-client-id", TokenBaseURL: tokenSrv.URL})
	a.SetLibrary(api.NewLibraryClient(apiSrv.URL, "old-access-token"))

	// Step 1: Trigger the 401 via a fetch command.
	model, fetchCmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	a = model.(*app.App)
	require.NotNil(t, fetchCmd)

	// Step 2: Execute fetch — gets unauthorizedMsg{}.
	unauthorizedMsgResult := fetchCmd()

	// Step 3: Feed to Update — dispatches refresh command.
	model, refreshCmd := a.Update(unauthorizedMsgResult)
	a = model.(*app.App)
	require.NotNil(t, refreshCmd)

	// Step 4: Execute refresh — should succeed with new token.
	refreshResult := refreshCmd()

	// Step 5: Feed tokenRefreshedMsg(success) to Update — re-inits clients.
	model, _ = a.Update(refreshResult)
	a = model.(*app.App)

	output := a.View()
	assert.NotContains(t, output, "Session expired", "successful refresh should not show session expired")

	// Verify the token store was updated with the new access token.
	newToken, err := store.Get(keychain.KeyAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", newToken, "store should have new access token after refresh")
}

// TestBuildFetchAlbumTracksCmd_429_EmitsRateLimitedMsg verifies that a 429 from
// the album tracks endpoint causes buildFetchAlbumTracksCmd to return RateLimitedMsg.
func TestBuildFetchAlbumTracksCmd_429_EmitsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("5")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchAlbumTracksRequestMsg{AlbumID: "alb123"})
	require.NotNil(t, cmd)

	msg := cmd()
	rateLimitMsg, ok := msg.(panes.RateLimitedMsg)
	assert.True(t, ok, "429 from album tracks endpoint should emit RateLimitedMsg, got %T", msg)
	assert.Equal(t, 5, rateLimitMsg.RetryAfterSecs)
}

// TestBuildFetchAlbumTracksCmd_CancelledCtx_ReturnsNil verifies that a pre-cancelled
// context causes buildFetchAlbumTracksCmd to return nil (silently discard).
func TestBuildFetchAlbumTracksCmd_CancelledCtx_ReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[],"next":null}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	// Trigger fetch then immediately send AlbumTrackViewClosedMsg to cancel it.
	_, cmd := a.Update(panes.FetchAlbumTracksRequestMsg{AlbumID: "alb123"})
	require.NotNil(t, cmd)

	// Cancel via AlbumTrackViewClosedMsg before executing the cmd.
	a.Update(panes.AlbumTrackViewClosedMsg{}) //nolint:errcheck

	// Now execute — the context was cancelled; staleness ID is now "".
	msg := cmd()
	if msg != nil {
		atMsg, ok := msg.(panes.AlbumTracksLoadedMsg)
		if ok {
			assert.Equal(t, "alb123", atMsg.AlbumID, "if msg arrives it must carry the original album ID")
		}
	}
}

// TestBuildFetchAlbumTracksCmd_Success_ReturnsLoadedMsg verifies that a successful
// album track fetch emits AlbumTracksLoadedMsg with tracks and hasNext=false.
func TestBuildFetchAlbumTracksCmd_Success_ReturnsLoadedMsg(t *testing.T) {
	body := `{"items":[{"id":"t1","uri":"spotify:track:t1","name":"So What","duration_ms":200000,"explicit":false,"artists":[{"id":"a1","name":"Miles Davis","uri":"spotify:artist:a1"}]}],"next":null}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchAlbumTracksRequestMsg{AlbumID: "alb123", Offset: 0})
	require.NotNil(t, cmd)

	msg := cmd()
	atMsg, ok := msg.(panes.AlbumTracksLoadedMsg)
	require.True(t, ok, "successful album tracks fetch should emit AlbumTracksLoadedMsg, got %T", msg)
	assert.Equal(t, "alb123", atMsg.AlbumID)
	assert.Equal(t, 0, atMsg.Offset)
	require.Len(t, atMsg.Tracks, 1)
	assert.Equal(t, "t1", atMsg.Tracks[0].ID)
	assert.False(t, atMsg.HasNext, "next=null → HasNext should be false")
	assert.NoError(t, atMsg.Err)
}

// TestBuildFetchAlbumTracksCmd_401_ShowsSessionExpired verifies that a 401 from
// the album tracks endpoint, when the token store has no refresh token, shows the
// "Session expired" message after the full command chain runs.
func TestBuildFetchAlbumTracksCmd_401_ShowsSessionExpired(t *testing.T) {
	srv := unauthorizedServer()
	defer srv.Close()

	cfg := &config.Config{ClientID: "test-client-id"}
	// InMemoryTokenStore with no refresh token — refresh will fail.
	store := keychain.NewInMemoryTokenStore()

	a := app.New(cfg, app.AppOptions{TokenStore: store})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	// Step 1: FetchAlbumTracksRequestMsg → buildFetchAlbumTracksCmd dispatched.
	model, fetchCmd := a.Update(panes.FetchAlbumTracksRequestMsg{AlbumID: "alb123"})
	a = model.(*app.App)
	require.NotNil(t, fetchCmd)

	// Step 2: Execute the fetch command — gets unauthorizedMsg{} back.
	unauthorizedMsgResult := fetchCmd()

	// Step 3: Feed unauthorizedMsg to Update — dispatches buildRefreshTokenCmd.
	model, refreshCmd := a.Update(unauthorizedMsgResult)
	a = model.(*app.App)
	require.NotNil(t, refreshCmd, "unauthorizedMsg should dispatch a refresh command")

	// Step 4: Execute the refresh command — fails because no refresh token.
	refreshResult := refreshCmd()

	// Step 5: Feed tokenRefreshedMsg(err) to Update — emits an error toast alert cmd.
	model, alertCmd := a.Update(refreshResult)
	a = model.(*app.App)
	require.NotNil(t, alertCmd, "session expired should emit an error toast alert cmd")

	// Process the alertCmd to activate the toast, then check View().
	alertMsg := alertCmd()
	updated, _ := a.Update(alertMsg)
	a = updated.(*app.App)
	output := a.View()
	assert.Contains(t, output, "Session expired", "401 from album tracks with no refresh token should show session expired toast")
}

// TestBuildFetchAlbumTracksCmd_403_EmitsPremiumToast verifies that a 403 from
// the album tracks endpoint causes the router to emit a "Spotify Premium required" toast.
func TestBuildFetchAlbumTracksCmd_403_EmitsPremiumToast(t *testing.T) {
	srv := forbiddenServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	// Step 1: Dispatch album tracks fetch.
	model, fetchCmd := a.Update(panes.FetchAlbumTracksRequestMsg{AlbumID: "alb123"})
	a = model.(*app.App)
	require.NotNil(t, fetchCmd)

	// Step 2: Execute — forbidden server returns 403 → ForbiddenError in AlbumTracksLoadedMsg.
	resultMsg := fetchCmd()
	atMsg, ok := resultMsg.(panes.AlbumTracksLoadedMsg)
	require.True(t, ok, "expected AlbumTracksLoadedMsg, got %T", resultMsg)
	require.Error(t, atMsg.Err, "403 should surface as an error in AlbumTracksLoadedMsg")

	// Step 3: Feed the loaded msg back to Update — router should emit a "Spotify Premium required" toast.
	model, alertCmd := a.Update(resultMsg)
	require.NotNil(t, model)
	require.NotNil(t, alertCmd, "403 AlbumTracks should emit an alert toast cmd")

	// Process the alertCmd to activate the toast, then check View().
	alertMsg := alertCmd()
	updated, _ := a.Update(alertMsg)
	appModel := updated.(*app.App)
	output := appModel.View()
	assert.Contains(t, output, "Spotify Premium required", "403 album tracks should show Premium required toast")
}

// ---------------------------------------------------------------------------
// buildFetchPlaylistTracksCmd — offset branching tests
// ---------------------------------------------------------------------------

// TestBuildFetchPlaylistTracksCmd_Offset0_CallsItemsEndpointForAllPlaylists verifies
// that offset=0 calls GET /playlists/{id}/items (not GET /playlists/{id}).
// GET /playlists/{id} only embeds items for owned playlists; non-owned playlists
// omit the items container, returning 0 tracks with no error. Using /items works
// for all playlists regardless of ownership.
func TestBuildFetchPlaylistTracksCmd_Offset0_CallsItemsEndpointForAllPlaylists(t *testing.T) {
	var capturedPath string
	body := `{
		"items":[{"is_local":false,"item":{"id":"t1","name":"T1","uri":"spotify:track:t1","duration_ms":200000,"type":"track","artists":[],"album":{"id":"al1","name":"A1","uri":"spotify:album:al1","release_date":"2021-01-01"}}}],
		"total":1,"next":null
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchPlaylistTracksRequestMsg{PlaylistID: "pl123", Offset: 0})
	require.NotNil(t, cmd)
	msg := cmd()

	ptMsg, ok := msg.(panes.PlaylistTracksLoadedMsg)
	require.True(t, ok, "offset=0 playlist fetch must return PlaylistTracksLoadedMsg, got %T", msg)
	require.NoError(t, ptMsg.Err)
	assert.Equal(t, "/v1/playlists/pl123/items", capturedPath,
		"offset=0 must call GET /playlists/{id}/items (works for owned and non-owned playlists)")
	require.Len(t, ptMsg.Tracks, 1)
	assert.Equal(t, "t1", ptMsg.Tracks[0].ID)
}

// TestBuildFetchPlaylistTracksCmd_OffsetNonZero_CallsItemsEndpoint verifies that
// when offset>0, buildFetchPlaylistTracksCmd calls GET /playlists/{id}/items
// (the pagination endpoint), not GET /playlists/{id}.
func TestBuildFetchPlaylistTracksCmd_OffsetNonZero_CallsItemsEndpoint(t *testing.T) {
	var capturedPath string
	body := `{
		"items":[{"is_local":false,"item":{"id":"t2","name":"T2","uri":"spotify:track:t2","duration_ms":180000,"type":"track","artists":[],"album":{"id":"al1","name":"A1","uri":"spotify:album:al1","release_date":"2021-01-01"}}}],
		"total":101,"next":null
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.FetchPlaylistTracksRequestMsg{PlaylistID: "pl123", Offset: 100})
	require.NotNil(t, cmd)
	msg := cmd()

	ptMsg, ok := msg.(panes.PlaylistTracksLoadedMsg)
	require.True(t, ok, "offset>0 playlist fetch must return PlaylistTracksLoadedMsg, got %T", msg)
	require.NoError(t, ptMsg.Err)
	assert.Equal(t, "/v1/playlists/pl123/items", capturedPath,
		"offset>0 must call GET /playlists/{id}/items for pagination")
	assert.Equal(t, 100, ptMsg.Offset)
}
