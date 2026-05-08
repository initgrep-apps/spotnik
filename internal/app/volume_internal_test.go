package app

// volume_internal_test.go — White-box tests for buildSetVolumeCmd error paths.
// Must be package app (not app_test) because unauthorizedMsg is unexported.

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/api/apitest"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newVolumeInternalTestApp creates a premium App with the given mock player,
// mirroring the helper in volume_test.go for use within the internal package.
func newVolumeInternalTestApp(mock *apitest.MockPlayer) *App {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := New(cfg, AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})
	if mock != nil {
		a.SetPlayer(mock)
	}
	return a
}

// TestBuildSetVolumeCmd_401_EmitsVolumeAppliedMsgWithUnauthorized verifies that an
// UnauthorizedError (401) from the player causes buildSetVolumeCmd to return
// VolumeAppliedMsg with the unauthorized error wrapped in Err.
func TestBuildSetVolumeCmd_401_EmitsVolumeAppliedMsgWithUnauthorized(t *testing.T) {
	mock := &apitest.MockPlayer{
		SetVolumeErr: &api.UnauthorizedError{},
	}
	a := newVolumeInternalTestApp(mock)

	intent := panes.VolumeIntentMsg{TargetVol: 60}
	_, cmd := a.Update(intent)
	require.NotNil(t, cmd)

	result := cmd()
	sent, ok := result.(panes.VolumeAppliedMsg)
	require.True(t, ok, "401 from SetVolume must produce VolumeAppliedMsg, got %T", result)
	assert.Error(t, sent.Err)
	var unauth *api.UnauthorizedError
	assert.True(t, errors.As(sent.Err, &unauth), "Err must be an UnauthorizedError")
}
