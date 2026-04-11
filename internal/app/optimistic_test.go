package app_test

// optimistic_test.go — Tests for optimistic store updates on key press.
//
// Verifies that Update(PlaybackRequestMsg) immediately mutates the store
// so the UI reflects the new state on the next render frame — before the
// API round-trip completes.

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyOptimisticUpdate verifies that each PlaybackAction mutates the store
// immediately when passed via PlaybackRequestMsg.
func TestApplyOptimisticUpdate(t *testing.T) {
	tests := []struct {
		name    string
		initial domain.PlaybackState
		action  panes.PlaybackAction
		check   func(t *testing.T, got *domain.PlaybackState)
	}{
		{
			name: "volume up",
			initial: domain.PlaybackState{
				Device: &domain.Device{VolumePercent: 65},
			},
			action: panes.ActionVolumeUp,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 66, got.Device.VolumePercent)
			},
		},
		{
			name: "volume up at max",
			initial: domain.PlaybackState{
				Device: &domain.Device{VolumePercent: 100},
			},
			action: panes.ActionVolumeUp,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 100, got.Device.VolumePercent, "clamped at 100")
			},
		},
		{
			name: "volume down",
			initial: domain.PlaybackState{
				Device: &domain.Device{VolumePercent: 65},
			},
			action: panes.ActionVolumeDown,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 64, got.Device.VolumePercent)
			},
		},
		{
			name: "volume down at floor",
			initial: domain.PlaybackState{
				Device: &domain.Device{VolumePercent: 0},
			},
			action: panes.ActionVolumeDown,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 0, got.Device.VolumePercent, "clamped at 0")
			},
		},
		{
			name: "pause",
			initial: domain.PlaybackState{
				IsPlaying: true,
				Device:    &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionPause,
			check: func(t *testing.T, got *domain.PlaybackState) {
				assert.False(t, got.IsPlaying)
			},
		},
		{
			name: "play",
			initial: domain.PlaybackState{
				IsPlaying: false,
				Device:    &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionPlay,
			check: func(t *testing.T, got *domain.PlaybackState) {
				assert.True(t, got.IsPlaying)
			},
		},
		{
			name: "shuffle on",
			initial: domain.PlaybackState{
				ShuffleState: false,
				Device:       &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionToggleShuffle,
			check: func(t *testing.T, got *domain.PlaybackState) {
				assert.True(t, got.ShuffleState)
			},
		},
		{
			name: "shuffle off",
			initial: domain.PlaybackState{
				ShuffleState: true,
				Device:       &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionToggleShuffle,
			check: func(t *testing.T, got *domain.PlaybackState) {
				assert.False(t, got.ShuffleState)
			},
		},
		{
			name: "repeat cycle off→context",
			initial: domain.PlaybackState{
				RepeatState: "off",
				Device:      &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionCycleRepeat,
			check: func(t *testing.T, got *domain.PlaybackState) {
				assert.Equal(t, "context", got.RepeatState)
			},
		},
		{
			name: "nil device no panic",
			initial: domain.PlaybackState{
				Device: nil,
			},
			action: panes.ActionVolumeUp,
			check: func(t *testing.T, got *domain.PlaybackState) {
				// Must not panic; device stays nil.
				assert.Nil(t, got.Device)
			},
		},
		{
			name: "next no-op",
			initial: domain.PlaybackState{
				Device: &domain.Device{VolumePercent: 65},
			},
			action: panes.ActionNext,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 65, got.Device.VolumePercent, "volume unchanged for ActionNext")
			},
		},
		{
			name: "previous no-op",
			initial: domain.PlaybackState{
				Device: &domain.Device{VolumePercent: 65},
			},
			action: panes.ActionPrevious,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 65, got.Device.VolumePercent, "volume unchanged for ActionPrevious")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestApp()
			initial := tt.initial
			a.Store().SetPlaybackState(&initial)

			a.Update(panes.PlaybackRequestMsg{Action: tt.action})

			got := a.Store().PlaybackState()
			require.NotNil(t, got, "playback state must not be nil after Update")
			tt.check(t, got)
		})
	}
}

// TestApplyOptimisticUpdate_NilPlaybackState_DoesNotPanic verifies that when the
// store has no playback state, applyOptimisticUpdate returns without panic and the
// store remains nil.
func TestApplyOptimisticUpdate_NilPlaybackState_DoesNotPanic(t *testing.T) {
	a := newTestApp()
	// Store starts nil by default — no SetPlaybackState call.

	assert.NotPanics(t, func() {
		a.Update(panes.PlaybackRequestMsg{Action: panes.ActionVolumeUp})
	})

	assert.Nil(t, a.Store().PlaybackState(), "store must remain nil when initial state was nil")
}
