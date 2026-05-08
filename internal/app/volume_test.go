package app_test

// volume_test.go — Tests for Story 197: VolumeIntentMsg handler + buildSetVolumeCmd.

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/api/apitest"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newVolumeTestApp(mock *apitest.MockPlayer) *app.App {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := app.New(cfg, app.AppOptions{})
	a.SetPlayer(mock)
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
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := app.New(cfg, app.AppOptions{}) // no player set

	intent := panes.VolumeIntentMsg{TargetVol: 50}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd)

	result := cmd()
	sent, ok := result.(panes.PlaybackCmdSentMsg)
	require.True(t, ok)
	assert.Error(t, sent.Err, "nil player must return an error")
}
