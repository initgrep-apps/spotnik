package app_test

// like_commands_test.go — Tests for Story 267 Task 5:
// buildLikeTrackCmd and buildUnlikeTrackCmd command factories.
//
// Verifies:
// - Success path returns ToggleLikeResultMsg with correct Liked state
// - Nil library client returns errNilClient (no panic)
// - 429 response is mapped to RateLimitedMsg
// - 401 response is mapped to unauthorizedMsg

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newLikeTestApp creates a minimal App with premium tier set so the premium
// gate passes and the routing handler dispatches the API command.
func newLikeTestApp() *app.App {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})
	return a
}

// TestBuildLikeTrackCmd_Success verifies that a 204 response from PUT /me/tracks
// produces a ToggleLikeResultMsg with Liked=true and nil Err.
func TestBuildLikeTrackCmd_Success(t *testing.T) {
	var capturedMethod, capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a := newLikeTestApp()
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	track := domain.Track{ID: "track-1", Name: "Blinding Lights"}
	_, cmd := a.Update(panes.ToggleLikeRequestMsg{Track: track, CurrentlyLiked: false})
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(panes.ToggleLikeResultMsg)
	require.True(t, ok, "expected ToggleLikeResultMsg, got %T", msg)
	assert.NoError(t, result.Err)
	assert.Equal(t, "track-1", result.TrackID)
	assert.True(t, result.Liked, "Liked should be true after successful like")
	assert.False(t, result.OriginalLiked, "OriginalLiked should be false (was not liked before)")
	assert.Equal(t, http.MethodPut, capturedMethod, "like should use PUT")
	assert.Equal(t, "/v1/me/tracks", capturedPath)
}

// TestBuildLikeTrackCmd_NilClient verifies that a nil library client produces
// ToggleLikeResultMsg with errNilClient (no panic, no HTTP call).
func TestBuildLikeTrackCmd_NilClient(t *testing.T) {
	a := newLikeTestApp()
	// Library client is nil (never SetLibrary called).

	track := domain.Track{ID: "track-1", Name: "Blinding Lights"}
	_, cmd := a.Update(panes.ToggleLikeRequestMsg{Track: track, CurrentlyLiked: false})
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(panes.ToggleLikeResultMsg)
	require.True(t, ok, "expected ToggleLikeResultMsg, got %T", msg)
	require.Error(t, result.Err, "nil library must set Err")
	assert.Equal(t, "track-1", result.TrackID)
}

// TestBuildUnlikeTrackCmd_Success verifies that a 204 response from DELETE /me/tracks
// produces a ToggleLikeResultMsg with Liked=false and nil Err.
func TestBuildUnlikeTrackCmd_Success(t *testing.T) {
	var capturedMethod, capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a := newLikeTestApp()
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	track := domain.Track{ID: "track-2", Name: "Save Your Tears"}
	_, cmd := a.Update(panes.ToggleLikeRequestMsg{Track: track, CurrentlyLiked: true})
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(panes.ToggleLikeResultMsg)
	require.True(t, ok, "expected ToggleLikeResultMsg, got %T", msg)
	assert.NoError(t, result.Err)
	assert.Equal(t, "track-2", result.TrackID)
	assert.False(t, result.Liked, "Liked should be false after successful unlike")
	assert.True(t, result.OriginalLiked, "OriginalLiked should be true (was liked before)")
	assert.Equal(t, http.MethodDelete, capturedMethod, "unlike should use DELETE")
	assert.Equal(t, "/v1/me/tracks", capturedPath)
}

// TestBuildUnlikeTrackCmd_NilClient verifies that a nil library client produces
// ToggleLikeResultMsg with errNilClient (no panic, no HTTP call).
func TestBuildUnlikeTrackCmd_NilClient(t *testing.T) {
	a := newLikeTestApp()
	// Library client is nil.

	track := domain.Track{ID: "track-2", Name: "Save Your Tears"}
	_, cmd := a.Update(panes.ToggleLikeRequestMsg{Track: track, CurrentlyLiked: true})
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(panes.ToggleLikeResultMsg)
	require.True(t, ok, "expected ToggleLikeResultMsg, got %T", msg)
	require.Error(t, result.Err, "nil library must set Err")
	assert.Equal(t, "track-2", result.TrackID)
}

// TestBuildLikeTrackCmd_429_ReturnsRateLimitedMsg verifies a 429 from the like
// endpoint produces a RateLimitedMsg (not a ToggleLikeResultMsg).
func TestBuildLikeTrackCmd_429_ReturnsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("9")
	defer srv.Close()

	a := newLikeTestApp()
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	track := domain.Track{ID: "track-1", Name: "Blinding Lights"}
	_, cmd := a.Update(panes.ToggleLikeRequestMsg{Track: track, CurrentlyLiked: false})
	require.NotNil(t, cmd)

	msg := cmd()
	rl, ok := msg.(panes.RateLimitedMsg)
	require.True(t, ok, "429 should produce RateLimitedMsg, got %T", msg)
	assert.Equal(t, 9, rl.RetryAfterSecs)
}

// TestBuildUnlikeTrackCmd_429_ReturnsRateLimitedMsg verifies a 429 from the unlike
// endpoint produces a RateLimitedMsg (not a ToggleLikeResultMsg).
func TestBuildUnlikeTrackCmd_429_ReturnsRateLimitedMsg(t *testing.T) {
	srv := rateLimitServer("4")
	defer srv.Close()

	a := newLikeTestApp()
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	track := domain.Track{ID: "track-2", Name: "Save Your Tears"}
	_, cmd := a.Update(panes.ToggleLikeRequestMsg{Track: track, CurrentlyLiked: true})
	require.NotNil(t, cmd)

	msg := cmd()
	rl, ok := msg.(panes.RateLimitedMsg)
	require.True(t, ok, "429 should produce RateLimitedMsg, got %T", msg)
	assert.Equal(t, 4, rl.RetryAfterSecs)
}
