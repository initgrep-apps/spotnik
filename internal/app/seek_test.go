package app_test

// seek_test.go — Tests for Story 224: SeekIntentMsg handler + buildSeekCmd.

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/api/apitest"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSeekTestApp creates a premium App with the given mock player and a window
// size sufficient for NowPlayingPane to be visible and focusable.
func newSeekTestApp(mock *apitest.MockPlayer) *app.App {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// Premium required so SeekIntentMsg passes the subscription gate.
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})
	if mock != nil {
		a.SetPlayer(mock)
	}
	return a
}

// TestApp_SeekIntentMsg_CallsSeek verifies that SeekIntentMsg calls player.Seek
// with the exact target position and returns SeekAppliedMsg with no error.
func TestApp_SeekIntentMsg_CallsSeek(t *testing.T) {
	mock := &apitest.MockPlayer{}
	a := newSeekTestApp(mock)

	intent := panes.SeekIntentMsg{TargetMs: 30000}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd, "SeekIntentMsg must return a cmd")

	result := cmd()
	sent, ok := result.(panes.SeekAppliedMsg)
	require.True(t, ok, "cmd must return SeekAppliedMsg, got %T", result)
	assert.NoError(t, sent.Err)
	assert.Equal(t, 30000, sent.PosMs)
	assert.Equal(t, 30000, mock.LastSeekMs, "Seek called with exact intent target")
}

// TestApp_SeekIntentMsg_NilPlayer_ReturnsErrNilClient verifies that
// buildSeekCmd returns errNilClient when no player is injected.
func TestApp_SeekIntentMsg_NilPlayer_ReturnsErrNilClient(t *testing.T) {
	// Use newSeekTestApp with nil mock so premium is set but no player is injected.
	a := newSeekTestApp(nil)

	intent := panes.SeekIntentMsg{TargetMs: 30000, Seq: 3}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd)

	result := cmd()
	sent, ok := result.(panes.SeekAppliedMsg)
	require.True(t, ok)
	assert.Error(t, sent.Err, "nil player must return an error")
	assert.Equal(t, 3, sent.Seq, "nil player must forward intentSeq")
}

// TestApp_SeekIntentMsg_NonPremium_Blocked verifies that free-tier users
// see a "Spotify Premium required" toast when attempting to seek.
func TestApp_SeekIntentMsg_NonPremium_Blocked(t *testing.T) {
	mock := &apitest.MockPlayer{}
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// Free-tier user.
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "free"})
	a.SetPlayer(mock)

	intent := panes.SeekIntentMsg{TargetMs: 30000}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd, "non-premium seek must return a toast cmd")

	// The cmd should produce a toast, not a SeekAppliedMsg.
	// Verify no Seek call was made.
	assert.Equal(t, 0, mock.SeekCallCount, "non-premium must not call Seek API")
}

// TestApp_SeekDebounceTickMsg_ForwardsToPane verifies that the app forwards a
// SeekDebounceTickMsg to NowPlayingPane and the result is a cmd that produces
// SeekIntentMsg with the expected target position.
func TestApp_SeekDebounceTickMsg_ForwardsToPane(t *testing.T) {
	mock := &apitest.MockPlayer{}
	a := newSeekTestApp(mock)
	// Seed store so confirmedProgress() returns 25000 and confirmedDuration() returns 180000.
	a.Store().SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item:     &domain.Track{DurationMs: 180000},
		ProgressMs: 25000,
	})

	// Press right arrow to prime seq to 1 in the pane's seekBar.
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyRight})

	// Directly send a SeekDebounceTickMsg with seq==1 (matches the pending seq).
	_, intentCmd := a.Update(components.SeekDebounceTickMsg{TargetMs: 30000, Seq: 1})
	require.NotNil(t, intentCmd, "matched debounce tick must return a SeekIntentMsg cmd")

	result := intentCmd()
	intentMsg, ok := result.(panes.SeekIntentMsg)
	require.True(t, ok, "cmd must produce SeekIntentMsg, got %T", result)
	assert.Equal(t, 30000, intentMsg.TargetMs, "forwarded SeekIntentMsg must carry the tick's target ms")
}

// TestApp_SeekAppliedMsg_Success_DispatchesInteractivePoll verifies that sending
// SeekAppliedMsg success to the app produces a cmd that resolves to
// PlaybackStateFetchedMsg, confirming the interactive poll was dispatched.
func TestApp_SeekAppliedMsg_Success_DispatchesInteractivePoll(t *testing.T) {
	mock := &apitest.MockPlayer{}
	a := newSeekTestApp(mock)

	_, cmd := a.Update(panes.SeekAppliedMsg{PosMs: 30000, Seq: 1})
	require.NotNil(t, cmd, "SeekAppliedMsg success must dispatch a cmd")

	// collectAllMsgs resolves the batch; at least one must be PlaybackStateFetchedMsg.
	msgs := collectAllMsgs(cmd)
	hasPoll := false
	for _, m := range msgs {
		if _, ok := m.(panes.PlaybackStateFetchedMsg); ok {
			hasPoll = true
		}
	}
	assert.True(t, hasPoll, "SeekAppliedMsg success must dispatch an interactive playback poll")
}

// TestBuildSeekCmd_429_EmitsSeekAppliedMsgWithRateLimitError verifies that a 429
// from the player causes buildSeekCmd to return SeekAppliedMsg with the
// RateLimitError wrapped in Err.
func TestBuildSeekCmd_429_EmitsSeekAppliedMsgWithRateLimitError(t *testing.T) {
	mock := &apitest.MockPlayer{
		SeekErr: &api.RateLimitError{RetryAfter: 10},
	}
	a := newSeekTestApp(mock)

	intent := panes.SeekIntentMsg{TargetMs: 30000}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd)

	result := cmd()
	sent, ok := result.(panes.SeekAppliedMsg)
	require.True(t, ok, "429 from Seek must produce SeekAppliedMsg, got %T", result)
	assert.Error(t, sent.Err)
	var rl *api.RateLimitError
	assert.True(t, errors.As(sent.Err, &rl), "Err must be a RateLimitError")
	assert.Equal(t, 10, rl.RetryAfter)
}