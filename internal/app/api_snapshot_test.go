package app_test

// api_snapshot_test.go — Tests for story 125: buildPlaybackAPICmd must snapshot
// store state BEFORE the optimistic write, not after.
//
// Each test verifies that the API receives the pre-optimistic value when the
// handler calls buildPlaybackAPICmd first, then applyOptimisticUpdate.

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/api/apitest"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlaybackAPICmd_UsesPreOptimisticState verifies that buildPlaybackAPICmd
// captures store state before applyOptimisticUpdate writes the predicted value.
//
// The bug (story 125): when applyOptimisticUpdate ran first, buildPlaybackAPICmd
// snapshotted the already-mutated store and computed the wrong next state:
//   - vol=74 → optimistic writes 75 → API snapshots 75 → sends SetVolume(76) ← WRONG
//   - shuffle=off → optimistic writes on → API snapshots on → sends SetShuffle(false) ← WRONG
//   - repeat=off → optimistic writes context → API snapshots context → sends SetRepeat("track") ← WRONG
//
// The fix: call buildPlaybackAPICmd first, then applyOptimisticUpdate.
func TestPlaybackAPICmd_UsesPreOptimisticState(t *testing.T) {
	tests := []struct {
		name          string
		initial       domain.PlaybackState
		action        panes.PlaybackAction
		checkAPI      func(t *testing.T, mock *apitest.MockPlayer)
		checkStore    func(t *testing.T, got *domain.PlaybackState)
	}{
		{
			name: "volume up at 74 — API receives 75 not 76",
			initial: domain.PlaybackState{
				Device: &domain.Device{VolumePercent: 74},
			},
			action: panes.ActionVolumeUp,
			checkAPI: func(t *testing.T, mock *apitest.MockPlayer) {
				require.True(t, mock.SetVolumeCalled, "SetVolume must be called")
				assert.Equal(t, 75, mock.LastSetVolume,
					"API must receive pre-optimistic volume (74+1=75), not post-optimistic (75+1=76)")
			},
			checkStore: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 75, got.Device.VolumePercent, "store shows optimistic 75")
			},
		},
		{
			name: "volume down at 30 — API receives 29 not 28",
			initial: domain.PlaybackState{
				Device: &domain.Device{VolumePercent: 30},
			},
			action: panes.ActionVolumeDown,
			checkAPI: func(t *testing.T, mock *apitest.MockPlayer) {
				require.True(t, mock.SetVolumeCalled, "SetVolume must be called")
				assert.Equal(t, 29, mock.LastSetVolume,
					"API must receive pre-optimistic volume (30-1=29), not post-optimistic (29-1=28)")
			},
			checkStore: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 29, got.Device.VolumePercent, "store shows optimistic 29")
			},
		},
		{
			name: "shuffle off → on — API receives true not false",
			initial: domain.PlaybackState{
				ShuffleState: false,
				Device:       &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionToggleShuffle,
			checkAPI: func(t *testing.T, mock *apitest.MockPlayer) {
				require.True(t, mock.SetShuffleCalled, "SetShuffle must be called")
				assert.True(t, mock.LastSetShuffle,
					"API must receive true (pre-optimistic shuffle=false → !false=true), not false")
			},
			checkStore: func(t *testing.T, got *domain.PlaybackState) {
				assert.True(t, got.ShuffleState, "store shows optimistic shuffle=true")
			},
		},
		{
			name: "shuffle on → off — API receives false not true",
			initial: domain.PlaybackState{
				ShuffleState: true,
				Device:       &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionToggleShuffle,
			checkAPI: func(t *testing.T, mock *apitest.MockPlayer) {
				require.True(t, mock.SetShuffleCalled, "SetShuffle must be called")
				assert.False(t, mock.LastSetShuffle,
					"API must receive false (pre-optimistic shuffle=true → !true=false), not true")
			},
			checkStore: func(t *testing.T, got *domain.PlaybackState) {
				assert.False(t, got.ShuffleState, "store shows optimistic shuffle=false")
			},
		},
		{
			name: "repeat off → context — API receives 'context' not 'track'",
			initial: domain.PlaybackState{
				RepeatState: "off",
				Device:      &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionCycleRepeat,
			checkAPI: func(t *testing.T, mock *apitest.MockPlayer) {
				require.True(t, mock.SetRepeatCalled, "SetRepeat must be called")
				assert.Equal(t, "context", mock.LastSetRepeat,
					"API must receive 'context' (off→context), not 'track' (context→track)")
			},
			checkStore: func(t *testing.T, got *domain.PlaybackState) {
				assert.Equal(t, "context", got.RepeatState, "store shows optimistic repeat=context")
			},
		},
		{
			name: "repeat context → track — API receives 'track' not 'off'",
			initial: domain.PlaybackState{
				RepeatState: "context",
				Device:      &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionCycleRepeat,
			checkAPI: func(t *testing.T, mock *apitest.MockPlayer) {
				require.True(t, mock.SetRepeatCalled, "SetRepeat must be called")
				assert.Equal(t, "track", mock.LastSetRepeat,
					"API must receive 'track' (context→track), not 'off' (track→off)")
			},
			checkStore: func(t *testing.T, got *domain.PlaybackState) {
				assert.Equal(t, "track", got.RepeatState, "store shows optimistic repeat=track")
			},
		},
		{
			name: "repeat track → off — API receives 'off' not 'context'",
			initial: domain.PlaybackState{
				RepeatState: "track",
				Device:      &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionCycleRepeat,
			checkAPI: func(t *testing.T, mock *apitest.MockPlayer) {
				require.True(t, mock.SetRepeatCalled, "SetRepeat must be called")
				assert.Equal(t, "off", mock.LastSetRepeat,
					"API must receive 'off' (track→off), not 'context' (off→context)")
			},
			checkStore: func(t *testing.T, got *domain.PlaybackState) {
				assert.Equal(t, "off", got.RepeatState, "store shows optimistic repeat=off")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &apitest.MockPlayer{}
			a := app.New(&config.Config{}, app.AppOptions{})
			a.SetPlayer(mock)

			initial := tt.initial
			a.Store().SetPlaybackState(&initial)

			// Update returns cmd built from pre-optimistic store state (after fix).
			_, cmd := a.Update(panes.PlaybackRequestMsg{Action: tt.action})
			require.NotNil(t, cmd, "PlaybackRequestMsg must return a cmd")

			// Execute the cmd synchronously — this triggers the API call.
			cmd()

			// Verify the API received the correct pre-optimistic value.
			tt.checkAPI(t, mock)

			// Verify the store was updated optimistically.
			got := a.Store().PlaybackState()
			require.NotNil(t, got)
			tt.checkStore(t, got)
		})
	}
}

// TestPlaybackAPICmd_PlayPause_Unaffected verifies that play/pause are not
// affected by the call-order fix — they don't read store state.
func TestPlaybackAPICmd_PlayPause_Unaffected(t *testing.T) {
	tests := []struct {
		name    string
		initial domain.PlaybackState
		action  panes.PlaybackAction
		check   func(t *testing.T, mock *apitest.MockPlayer)
	}{
		{
			name: "pause sends Pause not Play",
			initial: domain.PlaybackState{
				IsPlaying: true,
				Device:    &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionPause,
			check: func(t *testing.T, mock *apitest.MockPlayer) {
				assert.True(t, mock.PauseCalled, "Pause must be called")
				assert.False(t, mock.PlayCalled, "Play must NOT be called")
			},
		},
		{
			name: "play sends Play not Pause",
			initial: domain.PlaybackState{
				IsPlaying: false,
				Device:    &domain.Device{VolumePercent: 50},
			},
			action: panes.ActionPlay,
			check: func(t *testing.T, mock *apitest.MockPlayer) {
				assert.True(t, mock.PlayCalled, "Play must be called")
				assert.False(t, mock.PauseCalled, "Pause must NOT be called")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &apitest.MockPlayer{}
			a := app.New(&config.Config{}, app.AppOptions{})
			a.SetPlayer(mock)

			initial := tt.initial
			a.Store().SetPlaybackState(&initial)

			_, cmd := a.Update(panes.PlaybackRequestMsg{Action: tt.action})
			require.NotNil(t, cmd)
			cmd()

			tt.check(t, mock)
		})
	}
}
