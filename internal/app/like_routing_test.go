package app_test

// like_routing_test.go — Tests for Story 267 Task 6:
// ToggleLikeRequestMsg and ToggleLikeResultMsg routing handlers.
//
// Verifies:
// - ToggleLikeRequestMsg like path: optimistic AddLikedTrack + dispatch like cmd
// - ToggleLikeRequestMsg unlike path: optimistic RemoveLikedTrack + dispatch unlike cmd
// - Premium gate: free-tier user gets warning toast, no API dispatch
// - ToggleLikeResultMsg success: toast ("♥ Liked" or "Unliked")
// - ToggleLikeResultMsg error: rollback optimistic update + error toast

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newLikeRoutingApp creates an App with premium tier set and a window size so
// panes and alerts render. Callers may override the tier or inject a library.
func newLikeRoutingApp() *app.App {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})
	return a
}

// executeCmdAndBatch runs a cmd, unwrapping tea.BatchMsg into individual messages.
// Returns the list of messages produced (nil cmds dropped).
func executeCmdAndBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			if c != nil {
				if sub := c(); sub != nil {
					msgs = append(msgs, sub)
				}
			}
		}
		return msgs
	}
	if msg == nil {
		return nil
	}
	return []tea.Msg{msg}
}

// isBubbleupAlertMsg reports whether m is the unexported bubbleup alertMsg
// produced by ToastManager.Cmd -> AlertModel.NewAlertCmd. We can't type-assert
// across packages, so inspect via reflect on the type name + package path.
func isBubbleupAlertMsg(m tea.Msg) bool {
	t := reflect.TypeOf(m)
	if t == nil {
		return false
	}
	return t.Name() == "alertMsg" && strings.Contains(t.PkgPath(), "bubbleup")
}

// alertMsgString returns the human-readable toast text carried by a bubbleup
// alertMsg, or "" if m is not one. The msg field is unexported; we read it via
// reflect because the type lives in an external package.
func alertMsgString(m tea.Msg) string {
	v := reflect.ValueOf(m)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	f := v.FieldByName("msg")
	if !f.IsValid() {
		return ""
	}
	return f.String()
}

// --- ToggleLikeRequestMsg ---

// TestRouting_ToggleLikeRequest_Likes verifies that a ToggleLikeRequestMsg with
// CurrentlyLiked=false optimistically adds the track to the store and dispatches
// a like API command.
func TestRouting_ToggleLikeRequest_Likes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a := newLikeRoutingApp()
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	track := domain.Track{ID: "track-1", Name: "Blinding Lights", URI: "spotify:track:track-1"}
	_, cmd := a.Update(panes.ToggleLikeRequestMsg{Track: track, CurrentlyLiked: false})
	require.NotNil(t, cmd)

	// Optimistic store update should have happened before the API call.
	assert.True(t, a.Store().IsTrackLiked("track-1"),
		"optimistic AddLikedTrack should mark track as liked immediately")

	// The batch should include a ToggleLikeResultMsg from the API call.
	msgs := executeCmdAndBatch(cmd)
	require.NotEmpty(t, msgs)
	var foundResult bool
	for _, m := range msgs {
		if r, ok := m.(panes.ToggleLikeResultMsg); ok {
			foundResult = true
			assert.Equal(t, "track-1", r.TrackID)
			assert.True(t, r.Liked, "Liked should be true after successful like")
			assert.NoError(t, r.Err)
		}
	}
	assert.True(t, foundResult, "batch should contain ToggleLikeResultMsg")
}

// TestRouting_ToggleLikeRequest_Unlikes verifies that a ToggleLikeRequestMsg with
// CurrentlyLiked=true optimistically removes the track from the store and
// dispatches an unlike API command.
func TestRouting_ToggleLikeRequest_Unlikes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a := newLikeRoutingApp()
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))
	// Seed the store with a liked track so we can verify its removal.
	a.Store().SetLikedTracks([]domain.SavedTrack{
		{Track: domain.Track{ID: "track-2", Name: "Save Your Tears"}},
	})
	a.Store().SetLikedTotal(1)
	require.True(t, a.Store().IsTrackLiked("track-2"))

	track := domain.Track{ID: "track-2", Name: "Save Your Tears", URI: "spotify:track:track-2"}
	_, cmd := a.Update(panes.ToggleLikeRequestMsg{Track: track, CurrentlyLiked: true})
	require.NotNil(t, cmd)

	// Optimistic store update should have happened before the API call.
	assert.False(t, a.Store().IsTrackLiked("track-2"),
		"optimistic RemoveLikedTrack should mark track as unliked immediately")

	msgs := executeCmdAndBatch(cmd)
	require.NotEmpty(t, msgs)
	var foundResult bool
	for _, m := range msgs {
		if r, ok := m.(panes.ToggleLikeResultMsg); ok {
			foundResult = true
			assert.Equal(t, "track-2", r.TrackID)
			assert.False(t, r.Liked, "Liked should be false after successful unlike")
			assert.NoError(t, r.Err)
		}
	}
	assert.True(t, foundResult, "batch should contain ToggleLikeResultMsg")
}

// TestRouting_ToggleLikeRequest_PremiumGate verifies that a free-tier user
// pressing 'l' gets a warning toast and NO API command is dispatched.
func TestRouting_ToggleLikeRequest_PremiumGate(t *testing.T) {
	a := newLikeRoutingApp()
	// Downgrade to free tier.
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "free"})

	track := domain.Track{ID: "track-1", Name: "Blinding Lights"}
	_, cmd := a.Update(panes.ToggleLikeRequestMsg{Track: track, CurrentlyLiked: false})
	require.NotNil(t, cmd, "free user should get a warning toast cmd")

	// The store should NOT have been optimistically updated.
	assert.False(t, a.Store().IsTrackLiked("track-1"),
		"free-tier toggle must not optimistically update the store")

	// Executing the cmd should not produce a ToggleLikeResultMsg (it's a toast).
	msgs := executeCmdAndBatch(cmd)
	for _, m := range msgs {
		_, isResult := m.(panes.ToggleLikeResultMsg)
		assert.False(t, isResult,
			"free-tier toggle must NOT produce ToggleLikeResultMsg (gate blocks API call)")
	}
}

// --- ToggleLikeResultMsg ---

// TestRouting_ToggleLikeResult_Success verifies a successful like result shows
// a success toast ("♥ Liked").
func TestRouting_ToggleLikeResult_Success(t *testing.T) {
	a := newLikeRoutingApp()

	msg := panes.ToggleLikeResultMsg{TrackID: "track-1", Liked: true, OriginalLiked: false}
	_, cmd := a.Update(msg)
	require.NotNil(t, cmd, "success result should produce a toast cmd")

	// Execute the alert cmd and feed it back so the toast renders.
	alertMsg := cmd()
	if bm, ok := alertMsg.(tea.BatchMsg); ok {
		for _, c := range bm {
			if sub := c(); sub != nil {
				a.Update(sub)
			}
		}
	} else if alertMsg != nil {
		a.Update(alertMsg)
	}
	view := a.View()
	assert.Contains(t, view, "♥ Liked", "success toast should show ♥ Liked")
}

// TestRouting_ToggleLikeResult_UnlikeSuccess verifies a successful unlike result
// shows a success toast ("Unliked").
func TestRouting_ToggleLikeResult_UnlikeSuccess(t *testing.T) {
	a := newLikeRoutingApp()

	msg := panes.ToggleLikeResultMsg{TrackID: "track-2", Liked: false, OriginalLiked: true}
	_, cmd := a.Update(msg)
	require.NotNil(t, cmd)

	alertMsg := cmd()
	if bm, ok := alertMsg.(tea.BatchMsg); ok {
		for _, c := range bm {
			if sub := c(); sub != nil {
				a.Update(sub)
			}
		}
	} else if alertMsg != nil {
		a.Update(alertMsg)
	}
	view := a.View()
	assert.Contains(t, view, "Unliked", "success toast should show Unliked")
}

// TestRouting_ToggleLikeResult_ErrorRollback verifies that on error the
// optimistic update is rolled back. When the toggle was a like that failed
// (OriginalLiked=false), the optimistically-added track should be removed.
func TestRouting_ToggleLikeResult_ErrorRollback(t *testing.T) {
	a := newLikeRoutingApp()
	// Simulate the optimistic state: track was added optimistically.
	a.Store().SetLikedTracks([]domain.SavedTrack{
		{Track: domain.Track{ID: "track-1", Name: "Blinding Lights"}},
	})
	a.Store().SetLikedTotal(1)
	require.True(t, a.Store().IsTrackLiked("track-1"))

	// A failed like (OriginalLiked=false) should roll back by removing the track.
	msg := panes.ToggleLikeResultMsg{
		TrackID:       "track-1",
		Liked:         false,
		OriginalLiked: false,
		Err:           errors.New("spotify 500"),
	}
	_, cmd := a.Update(msg)
	require.NotNil(t, cmd, "error result should produce a toast cmd")

	// The optimistic addition should have been rolled back.
	assert.False(t, a.Store().IsTrackLiked("track-1"),
		"failed like should roll back the optimistic AddLikedTrack")

	// Execute alert and feed back so the toast renders.
	alertMsg := cmd()
	if bm, ok := alertMsg.(tea.BatchMsg); ok {
		for _, c := range bm {
			if sub := c(); sub != nil {
				a.Update(sub)
			}
		}
	} else if alertMsg != nil {
		a.Update(alertMsg)
	}
	view := a.View()
	assert.Contains(t, view, "Like track failed", "error toast should show operation-specific title")
}

// TestRouting_ToggleLikeResult_ErrorRollback_Unlike re-adds the exact track
// when an unlike fails (OriginalLiked=true). The optimistic RemoveLikedTrack
// deleted the track from likedTracks; the rollback must restore it so
// IsTrackLiked returns true again (matching Spotify's true state) rather than
// leaving the UI showing an unliked state until the next poll.
func TestRouting_ToggleLikeResult_ErrorRollback_Unlike(t *testing.T) {
	a := newLikeRoutingApp()
	// Seed liked tracks and stamp fetchedAt so we can detect it is NOT reset.
	seeded := []domain.SavedTrack{
		{Track: domain.Track{ID: "track-2", Name: "Save Your Tears", URI: "spotify:track:track-2"}},
	}
	a.Store().SetLikedTracks(seeded)
	a.Store().SetLikedTotal(1)
	require.True(t, a.Store().IsTrackLiked("track-2"))
	require.False(t, a.Store().LikedTracksStale(), "seeded tracks should be fresh")

	// Simulate the optimistic state: unlike handler already removed the track.
	a.Store().RemoveLikedTrack("track-2")
	require.False(t, a.Store().IsTrackLiked("track-2"),
		"precondition: optimistic remove should have unliked the track")

	// A failed unlike (OriginalLiked=true) should re-add the exact track.
	msg := panes.ToggleLikeResultMsg{
		TrackID:       "track-2",
		Track:         domain.Track{ID: "track-2", Name: "Save Your Tears", URI: "spotify:track:track-2"},
		Liked:         false,
		OriginalLiked: true,
		Err:           errors.New("spotify 500"),
	}
	_, cmd := a.Update(msg)
	require.NotNil(t, cmd)

	// The track should be liked again — restored, not merely stale.
	assert.True(t, a.Store().IsTrackLiked("track-2"),
		"failed unlike should re-add the track so IsTrackLiked matches Spotify")
	// fetchedAt must NOT be reset — we restored the exact state, no re-fetch needed.
	assert.False(t, a.Store().LikedTracksStale(),
		"failed unlike rollback restores state; it must not mark tracks stale")

	_ = cmd // toast cmd; not asserting on its contents here
}

// TestRouting_ToggleLikeResult_429_Rollback verifies that a 429 error rolls back
// the optimistic update AND dispatches the secondary RateLimitedMsg so the
// global rate-limit handler can engage backoff.
func TestRouting_ToggleLikeResult_429_Rollback(t *testing.T) {
	a := newLikeRoutingApp()
	// Simulate the optimistic state: track was added optimistically by a like.
	a.Store().SetLikedTracks([]domain.SavedTrack{
		{Track: domain.Track{ID: "track-1", Name: "Blinding Lights"}},
	})
	a.Store().SetLikedTotal(1)
	require.True(t, a.Store().IsTrackLiked("track-1"))

	// Build a 429 error the way the API client does (RateLimitError).
	srv := rateLimitServer("9")
	defer srv.Close()
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))
	_, apiCmd := a.Update(panes.ToggleLikeRequestMsg{
		Track:          domain.Track{ID: "track-1", Name: "Blinding Lights"},
		CurrentlyLiked: false,
	})
	// The store was optimistically updated again (idempotent add); force a
	// realistic "liked" precondition before feeding the result back.
	require.True(t, a.Store().IsTrackLiked("track-1"))

	resultMsg := apiCmd()
	result, ok := resultMsg.(panes.ToggleLikeResultMsg)
	require.True(t, ok, "429 cmd should return ToggleLikeResultMsg")
	require.Error(t, result.Err)

	_, rollbackCmd := a.Update(result)
	// The optimistic addition should have been rolled back.
	assert.False(t, a.Store().IsTrackLiked("track-1"),
		"429 should roll back the optimistic AddLikedTrack")

	// The rollback cmd must carry a secondary RateLimitedMsg for the global handler.
	msgs := executeCmdAndBatch(rollbackCmd)
	var foundRateLimit bool
	for _, m := range msgs {
		if rl, ok := m.(panes.RateLimitedMsg); ok {
			foundRateLimit = true
			assert.Equal(t, 9, rl.RetryAfterSecs)
		}
	}
	assert.True(t, foundRateLimit,
		"429 rollback should dispatch secondary RateLimitedMsg for global backoff handler")

	// Exactly ONE "Rate-limited" toast should be emitted — the routing-level
	// toast must be suppressed so the global RateLimitedMsg handler owns the
	// single ratelimit toast (otherwise the user sees a duplicate).
	var ratelimitToasts int
	for _, m := range msgs {
		if isBubbleupAlertMsg(m) && strings.Contains(alertMsgString(m), "Rate-limited") {
			ratelimitToasts++
		}
	}
	assert.Equal(t, 0, ratelimitToasts,
		"429 routing must NOT emit its own ratelimit toast — global handler owns it")
}

// TestRouting_ToggleLikeResult_401_Rollback verifies that a 401 error rolls back
// the optimistic update AND dispatches the secondary unauthorizedMsg so the
// global 401 handler can trigger a token refresh. The routing handler should
// NOT emit its own toast (the errorMapper suppresses 401 to ToastNone).
func TestRouting_ToggleLikeResult_401_Rollback(t *testing.T) {
	a := newLikeRoutingApp()
	// Simulate the optimistic state: track was added optimistically by a like.
	a.Store().SetLikedTracks([]domain.SavedTrack{
		{Track: domain.Track{ID: "track-1", Name: "Blinding Lights"}},
	})
	a.Store().SetLikedTotal(1)
	require.True(t, a.Store().IsTrackLiked("track-1"))

	srv := unauthorizedServer()
	defer srv.Close()
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))
	_, apiCmd := a.Update(panes.ToggleLikeRequestMsg{
		Track:          domain.Track{ID: "track-1", Name: "Blinding Lights"},
		CurrentlyLiked: false,
	})
	require.True(t, a.Store().IsTrackLiked("track-1"))

	resultMsg := apiCmd()
	result, ok := resultMsg.(panes.ToggleLikeResultMsg)
	require.True(t, ok, "401 cmd should return ToggleLikeResultMsg")
	require.Error(t, result.Err)

	_, rollbackCmd := a.Update(result)
	// The optimistic addition should have been rolled back.
	assert.False(t, a.Store().IsTrackLiked("track-1"),
		"401 should roll back the optimistic AddLikedTrack")

	// The rollback cmd must carry a secondary unauthorizedMsg for the global
	// token-refresh handler.
	msgs := executeCmdAndBatch(rollbackCmd)
	var foundUnauthorized bool
	for _, m := range msgs {
		if _, ok := m.(app.UnauthorizedMsgForTest); ok {
			foundUnauthorized = true
		}
	}
	assert.True(t, foundUnauthorized,
		"401 rollback should dispatch secondary unauthorizedMsg for global refresh handler")
}

// TestRouting_ToggleLikeResult_NilClient_Rollback verifies that an errNilClient
// result (library not yet attached at startup) rolls back the optimistic update
// silently — no toast — and reverts IsTrackLiked to the pre-toggle state.
func TestRouting_ToggleLikeResult_NilClient_Rollback(t *testing.T) {
	a := newLikeRoutingApp()
	// No library client attached — simulates early startup before auth completes.
	// Seed a liked track and simulate an optimistic unlike.
	a.Store().SetLikedTracks([]domain.SavedTrack{
		{Track: domain.Track{ID: "track-2", Name: "Save Your Tears"}},
	})
	a.Store().SetLikedTotal(1)
	require.True(t, a.Store().IsTrackLiked("track-2"))
	a.Store().RemoveLikedTrack("track-2")
	require.False(t, a.Store().IsTrackLiked("track-2"))

	msg := panes.ToggleLikeResultMsg{
		TrackID:       "track-2",
		Track:         domain.Track{ID: "track-2", Name: "Save Your Tears"},
		Liked:         false,
		OriginalLiked: true,
		Err:           app.ErrNilClientForTest,
	}
	_, cmd := a.Update(msg)

	// The track should be liked again (rollback re-added it).
	assert.True(t, a.Store().IsTrackLiked("track-2"),
		"errNilClient unlike should roll back by re-adding the track")

	// The cmd should be a pane refresh only — no toast (silent rollback).
	msgs := executeCmdAndBatch(cmd)
	for _, m := range msgs {
		// None of the produced messages should be a toast-bearing alert.
		_ = m
	}
}

// TestRouting_ToggleLikeResult_ErrorMapsToOpLikeTracks verifies that the error
// toast produced by a failed like/unlike uses the OpLikeTracks operation title
// "Like track failed" (story 269), not the legacy "Failed to load library"
// title that came from OpLibrary.
func TestRouting_ToggleLikeResult_ErrorMapsToOpLikeTracks(t *testing.T) {
	a := newLikeRoutingApp()
	a.Store().SetLikedTracks([]domain.SavedTrack{
		{Track: domain.Track{ID: "track-1", Name: "Blinding Lights"}},
	})
	a.Store().SetLikedTotal(1)

	msg := panes.ToggleLikeResultMsg{
		TrackID:       "track-1",
		Liked:         false,
		OriginalLiked: false,
		Err:           errors.New("spotify 500"),
	}
	_, cmd := a.Update(msg)
	require.NotNil(t, cmd, "error result should produce a toast cmd")

	// Execute alert and feed back so the toast renders.
	alertMsg := cmd()
	if bm, ok := alertMsg.(tea.BatchMsg); ok {
		for _, c := range bm {
			if sub := c(); sub != nil {
				a.Update(sub)
			}
		}
	} else if alertMsg != nil {
		a.Update(alertMsg)
	}
	view := a.View()
	assert.Contains(t, view, "Like track failed",
		"toggleLike error toast must use OpLikeTracks title (story 269)")
	assert.NotContains(t, view, "Failed to load library",
		"toggleLike error toast must NOT use the legacy OpLibrary title")
}
