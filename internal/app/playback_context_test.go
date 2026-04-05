package app_test

// playback_context_test.go — Tests for Story 105: context-aware playback commands.
// Covers buildPlayContextCmd (with/without offset) and buildPlayTrackListCmd.

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApp_PlayContextMsg_WithOffsetURI verifies that PlayContextMsg with a non-empty
// OffsetURI produces a play command (forwarded to buildPlayContextCmd).
func TestApp_PlayContextMsg_WithOffsetURI_ProducesCmd(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	msg := panes.PlayContextMsg{
		ContextURI: "spotify:collection:tracks",
		OffsetURI:  "spotify:track:abc123",
	}
	_, cmd := a.Update(msg)
	assert.NotNil(t, cmd, "PlayContextMsg with OffsetURI should produce a play command")
}

// TestApp_PlayContextMsg_WithoutOffsetURI_ProducesCmd verifies that PlayContextMsg
// without OffsetURI still produces a command (no regression on existing behaviour).
func TestApp_PlayContextMsg_WithoutOffsetURI_ProducesCmd(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	msg := panes.PlayContextMsg{ContextURI: "spotify:playlist:pl1"}
	_, cmd := a.Update(msg)
	assert.NotNil(t, cmd, "PlayContextMsg without OffsetURI should produce a play command")
}

// TestApp_PlayContextMsg_NilPlayer_ReturnsErrNilClient verifies that PlayContextMsg
// with a nil player returns PlaybackCmdSentMsg with errNilClient.
func TestApp_PlayContextMsg_NilPlayer_ReturnsErrNilClient(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{}) // no player set

	msg := panes.PlayContextMsg{
		ContextURI: "spotify:collection:tracks",
		OffsetURI:  "spotify:track:abc123",
	}
	_, cmd := a.Update(msg)
	require.NotNil(t, cmd)

	result := cmd()
	sentMsg, ok := result.(panes.PlaybackCmdSentMsg)
	require.True(t, ok, "expected PlaybackCmdSentMsg, got %T", result)
	assert.Error(t, sentMsg.Err, "nil player should produce an error")
}

// TestApp_PlayTrackListMsg_ProducesCmd verifies that PlayTrackListMsg produces a
// play command.
func TestApp_PlayTrackListMsg_ProducesCmd(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	msg := panes.PlayTrackListMsg{
		URIs: []string{"spotify:track:t1", "spotify:track:t2", "spotify:track:t3"},
	}
	_, cmd := a.Update(msg)
	assert.NotNil(t, cmd, "PlayTrackListMsg should produce a play command")
}

// TestApp_PlayTrackListMsg_NilPlayer_ReturnsErrNilClient verifies that PlayTrackListMsg
// with a nil player returns PlaybackCmdSentMsg with an error.
func TestApp_PlayTrackListMsg_NilPlayer_ReturnsErrNilClient(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{}) // no player set

	msg := panes.PlayTrackListMsg{
		URIs: []string{"spotify:track:t1"},
	}
	_, cmd := a.Update(msg)
	require.NotNil(t, cmd)

	result := cmd()
	sentMsg, ok := result.(panes.PlaybackCmdSentMsg)
	require.True(t, ok, "expected PlaybackCmdSentMsg, got %T", result)
	assert.Error(t, sentMsg.Err, "nil player should produce an error")
}
