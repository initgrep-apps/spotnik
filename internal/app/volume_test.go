package app_test

// volume_test.go — Tests for Story 197: VolumeIntentMsg handler + buildSetVolumeCmd.

import (
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
// returns a PlaybackCmdSentMsg with no error.
func TestApp_VolumeIntentMsg_CallsSetVolume(t *testing.T) {
	mock := &apitest.MockPlayer{}
	a := newVolumeTestApp(mock)

	intent := panes.VolumeIntentMsg{TargetVol: 72}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd, "VolumeIntentMsg must return a cmd")

	result := cmd()
	sent, ok := result.(panes.PlaybackCmdSentMsg)
	require.True(t, ok, "cmd must return PlaybackCmdSentMsg, got %T", result)
	assert.NoError(t, sent.Err)
	assert.Equal(t, 72, mock.LastSetVolume, "SetVolume called with exact intent target")
}

// TestApp_VolumeIntentMsg_NilPlayer_ReturnsErrNilClient verifies that
// buildSetVolumeCmd returns errNilClient when no player is injected.
func TestApp_VolumeIntentMsg_NilPlayer_ReturnsErrNilClient(t *testing.T) {
	// Use newVolumeTestApp with nil mock so premium is set but no player is injected.
	a := newVolumeTestApp(nil)

	intent := panes.VolumeIntentMsg{TargetVol: 50}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd)

	result := cmd()
	sent, ok := result.(panes.PlaybackCmdSentMsg)
	require.True(t, ok)
	assert.Error(t, sent.Err, "nil player must return an error")
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
	sent, ok := sentMsg.(panes.PlaybackCmdSentMsg)
	require.True(t, ok, "buildSetVolumeCmd must return PlaybackCmdSentMsg, got %T", sentMsg)
	assert.NoError(t, sent.Err)
	assert.True(t, mock.SetVolumeCalled, "SetVolume must have been called")
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

// TestBuildSetVolumeCmd_429_EmitsRateLimitedMsg verifies that a 429 (RateLimitError)
// from the player causes buildSetVolumeCmd to return RateLimitedMsg so the gateway-
// level rate limit is surfaced and fetching sentinels are cleared.
func TestBuildSetVolumeCmd_429_EmitsRateLimitedMsg(t *testing.T) {
	mock := &apitest.MockPlayer{
		SetVolumeErr: &api.RateLimitError{RetryAfter: 5},
	}
	a := newVolumeTestApp(mock)

	intent := panes.VolumeIntentMsg{TargetVol: 60}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd)

	result := cmd()
	rl, ok := result.(panes.RateLimitedMsg)
	require.True(t, ok, "429 from SetVolume must produce RateLimitedMsg, got %T", result)
	assert.Equal(t, 5, rl.RetryAfterSecs, "RetryAfterSecs must match RateLimitError.RetryAfter")
}
