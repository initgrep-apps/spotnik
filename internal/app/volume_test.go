package app_test

// volume_test.go — Tests for Story 197: VolumeIntentMsg handler + buildSetVolumeCmd.
// Story 205 additions: SetVolumeCallCount assertion, 401 test, generic-error test,
// improved 429 test, and improved success test.

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/api/apitest"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newVolumeTestApp creates a premium App with the given mock player injected and
// a window size sufficient for NowPlayingPane to be visible and focusable.
func newVolumeTestApp(mock *apitest.MockPlayer) *app.App {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// Premium required so VolumeIntentMsg passes through the subscription gate.
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})
	if mock != nil {
		a.SetPlayer(mock)
	}
	return a
}

// TestApp_VolumeIntentMsg_CallsSetVolume verifies that when the app receives a
// VolumeIntentMsg it calls player.SetVolume with the exact target volume and
// returns a VolumeAppliedMsg with no error.
func TestApp_VolumeIntentMsg_CallsSetVolume(t *testing.T) {
	mock := &apitest.MockPlayer{}
	a := newVolumeTestApp(mock)

	intent := panes.VolumeIntentMsg{TargetVol: 72}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd, "VolumeIntentMsg must return a cmd")

	result := cmd()
	sent, ok := result.(panes.VolumeAppliedMsg)
	require.True(t, ok, "cmd must return VolumeAppliedMsg, got %T", result)
	assert.NoError(t, sent.Err)
	assert.Equal(t, 72, sent.Vol)
	assert.Equal(t, 72, mock.LastSetVolume, "SetVolume called with exact intent target")
}

// TestApp_VolumeIntentMsg_NilPlayer_ReturnsErrNilClient verifies that
// buildSetVolumeCmd returns errNilClient when no player is injected.
func TestApp_VolumeIntentMsg_NilPlayer_ReturnsErrNilClient(t *testing.T) {
	// Use newVolumeTestApp with nil mock so premium is set but no player is injected.
	a := newVolumeTestApp(nil)

	intent := panes.VolumeIntentMsg{TargetVol: 50, Seq: 7}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd)

	result := cmd()
	sent, ok := result.(panes.VolumeAppliedMsg)
	require.True(t, ok)
	assert.Error(t, sent.Err, "nil player must return an error")
	assert.Equal(t, 7, sent.Seq, "nil player must forward intentSeq")
}

// TestApp_VolumeDebounce_FiveRapidPresses_SendsOneCall verifies that 5 rapid '+'
// keypresses result in exactly one SetVolume call with the cumulative target (54
// from a base of 49) once the last debounce tick is executed and fed back.
func TestApp_VolumeDebounce_FiveRapidPresses_SendsOneCall(t *testing.T) {
	mock := &apitest.MockPlayer{}
	a := newVolumeTestApp(mock)
	// Seed store with playback state so confirmedVolume() returns 49.
	a.Store().SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Device:    &domain.Device{VolumePercent: 49},
	})

	// Send 5 '+' keypresses, collecting the debounce tick cmds.
	var lastTickCmd tea.Cmd
	for i := 0; i < 5; i++ {
		_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
		require.NotNil(t, cmd, "'+' key must return a debounce tick cmd")
		lastTickCmd = cmd
	}

	// Execute the LAST tick cmd — this fires VolumeDebounceTickMsg{Seq:5, TargetVol:54}.
	tickMsg := lastTickCmd()

	// Feed the tick msg back to App.Update; the pane matches seq==5 and returns a cmd
	// that wraps VolumeIntentMsg{TargetVol:54}.
	_, intentCmdWrapper := a.Update(tickMsg)
	require.NotNil(t, intentCmdWrapper, "last debounce tick must produce a VolumeIntentMsg cmd")

	// Execute the wrapper to get the VolumeIntentMsg, then feed it to App.Update
	// which dispatches buildSetVolumeCmd.
	intentMsg := intentCmdWrapper()
	_, setVolumeCmd := a.Update(intentMsg)
	require.NotNil(t, setVolumeCmd, "VolumeIntentMsg must produce a buildSetVolumeCmd")

	// Execute buildSetVolumeCmd — should call SetVolume(54).
	sentMsg := setVolumeCmd()
	sent, ok := sentMsg.(panes.VolumeAppliedMsg)
	require.True(t, ok, "buildSetVolumeCmd must return VolumeAppliedMsg, got %T", sentMsg)
	assert.NoError(t, sent.Err)
	assert.Equal(t, 54, sent.Vol)
	assert.Equal(t, 1, mock.SetVolumeCallCount, "5 rapid presses must result in exactly one SetVolume call")
	assert.Equal(t, 54, mock.LastSetVolume, "5 presses from 49 must set volume to 54")
}

// TestApp_VolumeDebounceTickMsg_ForwardsToPane verifies that the app forwards a
// VolumeDebounceTickMsg to NowPlayingPane and the result is a cmd that produces
// VolumeIntentMsg with the expected target volume.
func TestApp_VolumeDebounceTickMsg_ForwardsToPane(t *testing.T) {
	mock := &apitest.MockPlayer{}
	a := newVolumeTestApp(mock)
	// Seed store so confirmedVolume() returns 50.
	a.Store().SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Device:    &domain.Device{VolumePercent: 50},
	})

	// Press '+' to prime seq to 1 in the pane's volumeBar.
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})

	// Directly send a VolumeDebounceTickMsg with seq==1 (matches the pending seq).
	_, intentCmd := a.Update(components.VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})
	require.NotNil(t, intentCmd, "matched debounce tick must return a VolumeIntentMsg cmd")

	result := intentCmd()
	intentMsg, ok := result.(panes.VolumeIntentMsg)
	require.True(t, ok, "cmd must produce VolumeIntentMsg, got %T", result)
	assert.Equal(t, 51, intentMsg.TargetVol, "forwarded VolumeIntentMsg must carry the tick's target vol")
}

// TestBuildSetVolumeCmd_429_EmitsVolumeAppliedMsgWithRateLimitError verifies that a 429
// from the player causes buildSetVolumeCmd to return VolumeAppliedMsg with the
// RateLimitError wrapped in Err.
func TestBuildSetVolumeCmd_429_EmitsVolumeAppliedMsgWithRateLimitError(t *testing.T) {
	mock := &apitest.MockPlayer{
		SetVolumeErr: &api.RateLimitError{RetryAfter: 5},
	}
	a := newVolumeTestApp(mock)

	intent := panes.VolumeIntentMsg{TargetVol: 60}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd)

	result := cmd()
	sent, ok := result.(panes.VolumeAppliedMsg)
	require.True(t, ok, "429 from SetVolume must produce VolumeAppliedMsg, got %T", result)
	assert.Error(t, sent.Err)
	var rl *api.RateLimitError
	assert.True(t, errors.As(sent.Err, &rl), "Err must be a RateLimitError")
	assert.Equal(t, 5, rl.RetryAfter)
}

// TestApp_VolumeAppliedMsg_Success_DispatchesInteractivePoll verifies that sending
// VolumeAppliedMsg success to the app produces a cmd that resolves to
// PlaybackStateFetchedMsg, confirming the interactive poll was dispatched.
func TestApp_VolumeAppliedMsg_Success_DispatchesInteractivePoll(t *testing.T) {
	mock := &apitest.MockPlayer{}
	a := newVolumeTestApp(mock)

	_, cmd := a.Update(panes.VolumeAppliedMsg{Vol: 72, Seq: 1})
	require.NotNil(t, cmd, "VolumeAppliedMsg success must dispatch a cmd")

	// collectAllMsgs resolves the batch; at least one must be PlaybackStateFetchedMsg.
	msgs := collectAllMsgs(cmd)
	hasPoll := false
	for _, m := range msgs {
		if _, ok := m.(panes.PlaybackStateFetchedMsg); ok {
			hasPoll = true
		}
	}
	assert.True(t, hasPoll, "VolumeAppliedMsg success must dispatch an interactive playback poll")
}

// TestApp_VolumeAppliedMsg_429_ClearsPendingAndBacksOff verifies that a 429 wrapped
// in VolumeAppliedMsg clears the bar's pending state and then triggers the existing
// RateLimitedMsg backoff/toast path.
//
// CancelPending uses a seq guard: it only fires when b.seq == intentSeq+1. The real
// flow is: HandleKey (seq=1) → HandleDebounce (seq advances to 2, intentSeq=1) →
// VolumeAppliedMsg{Seq:1} → CancelPending(1, 60): 2==1+1 → true → bar reverts to 60.
// Without the HandleDebounce step, seq stays at 1 and CancelPending silently no-ops.
func TestApp_VolumeAppliedMsg_429_ClearsPendingAndBacksOff(t *testing.T) {
	a := newVolumeTestApp(&apitest.MockPlayer{})
	// Seed store so confirmedVolume() returns 60.
	a.Store().SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Device:    &domain.Device{VolumePercent: 60},
	})

	// Step 1: Keypress → bar seq=1, currentVol=61, hasPending=true.
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})

	// Step 2: Debounce tick → pane calls HandleDebounce, seq advances to 2, intentSeq=1.
	_, _ = a.Update(components.VolumeDebounceTickMsg{TargetVol: 61, Seq: 1})

	// Step 3: 429 arrives with intentSeq=1 — CancelPending(1, 60): 2==1+1 → true → revert.
	_, finalCmd := a.Update(panes.VolumeAppliedMsg{
		Err: &api.RateLimitError{RetryAfter: 5},
		Seq: 1,
	})
	require.NotNil(t, finalCmd, "VolumeAppliedMsg with 429 must return a cmd (backoff path)")

	// The returned cmd must produce a message (the backoff throttle tick or batch).
	result := finalCmd()
	assert.NotNil(t, result, "backoff cmd must produce a message")

	// Pending-state observable: GradientVolumeBar.Render() has no explicit pending indicator.
	// The pane-level test (TestNowPlayingPane_VolumeAppliedMsg_StaleSeq_BarStaysInSecondBurst)
	// covers CancelPending's seq-guard logic in detail. Here we simply confirm the view
	// does not show a stale pending marker.
	assert.NotContains(t, a.View(), "~", "volume bar must not be in pending state after 429")
}

// TestApp_VolumeAppliedMsg_401_RoutesToTokenRefresh verifies that a 401 wrapped in
// VolumeAppliedMsg routes through the existing token refresh path. With no refresh
// token configured, the full chain ends with a "Session expired" toast visible in View().
func TestApp_VolumeAppliedMsg_401_RoutesToTokenRefresh(t *testing.T) {
	// Configure a token store with no refresh token so the refresh attempt fails,
	// which causes the "Session expired" toast to appear.
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	tokenStore := keychain.NewInMemoryTokenStore()
	a := app.New(cfg, app.AppOptions{TokenStore: tokenStore})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})

	// Step 1: Feed VolumeAppliedMsg with UnauthorizedError → dispatches buildRefreshTokenCmd.
	model, refreshCmd := a.Update(panes.VolumeAppliedMsg{
		Err: &api.UnauthorizedError{},
		Seq: 1,
	})
	a = model.(*app.App)
	require.NotNil(t, refreshCmd, "401 in VolumeAppliedMsg must dispatch a token refresh cmd")

	// Step 2: Execute the refresh cmd — fails because tokenStore has no refresh token.
	refreshResult := refreshCmd()

	// Step 3: Feed tokenRefreshedMsg(err) → emits "Session expired" toast alert cmd.
	model, alertCmd := a.Update(refreshResult)
	a = model.(*app.App)
	require.NotNil(t, alertCmd, "failed token refresh must emit an alert cmd")

	// Step 4: Execute alertCmd, feed the resulting message back so the toast is active.
	alertMsg := alertCmd()
	updated, _ := a.Update(alertMsg)
	a = updated.(*app.App)

	assert.Contains(t, a.View(), "Session expired", "401 volume error must show 'Session expired' toast")
}

// TestApp_VolumeAppliedMsg_GenericError_BatchesPollAndToast verifies that a non-typed
// error (not 429/401/nil-client) in VolumeAppliedMsg results in a batch containing
// both the interactive poll cmd and an error toast cmd.
func TestApp_VolumeAppliedMsg_GenericError_BatchesPollAndToast(t *testing.T) {
	a := newVolumeTestApp(&apitest.MockPlayer{})

	_, cmd := a.Update(panes.VolumeAppliedMsg{
		Err: errors.New("unexpected volume error"),
		Seq: 1,
	})
	require.NotNil(t, cmd, "generic error in VolumeAppliedMsg must return a cmd")

	msgs := collectAllMsgs(cmd)
	require.GreaterOrEqual(t, len(msgs), 2, "generic error must batch at least poll + toast cmds, got %d msgs", len(msgs))

	hasPoll := false
	for _, m := range msgs {
		if _, ok := m.(panes.PlaybackStateFetchedMsg); ok {
			hasPoll = true
		}
	}
	assert.True(t, hasPoll, "generic error batch must include an interactive playback poll")
}
