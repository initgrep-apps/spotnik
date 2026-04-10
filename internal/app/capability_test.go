package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
)

func TestCheckCapability(t *testing.T) {
	tests := []struct {
		name        string
		ps          *domain.PlaybackState
		action      panes.PlaybackAction
		wantAllowed bool
		wantReason  string
	}{
		{
			name:        "nil playback state — always allowed (no state yet)",
			ps:          nil,
			action:      panes.ActionVolumeUp,
			wantAllowed: true,
		},
		{
			name: "device is_restricted — blocks all actions",
			ps: &domain.PlaybackState{
				Device: &domain.Device{IsRestricted: true, SupportsVolume: true},
			},
			action:      panes.ActionPlay,
			wantAllowed: false,
			wantReason:  "Device not controllable via API",
		},
		{
			name: "volume up — device supports volume — allowed",
			ps: &domain.PlaybackState{
				Device: &domain.Device{IsRestricted: false, SupportsVolume: true},
			},
			action:      panes.ActionVolumeUp,
			wantAllowed: true,
		},
		{
			name: "volume down — device does not support volume — blocked",
			ps: &domain.PlaybackState{
				Device: &domain.Device{IsRestricted: false, SupportsVolume: false},
			},
			action:      panes.ActionVolumeDown,
			wantAllowed: false,
			wantReason:  "Volume not available on this device",
		},
		{
			name: "skip next — disallowed by actions",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{SkippingNext: true}},
			},
			action:      panes.ActionNext,
			wantAllowed: false,
			wantReason:  "Skip not available in this context",
		},
		{
			name: "skip previous — disallowed by actions",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{SkippingPrev: true}},
			},
			action:      panes.ActionPrevious,
			wantAllowed: false,
			wantReason:  "Skip not available in this context",
		},
		{
			name: "toggle shuffle — disallowed",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{TogglingShuffle: true}},
			},
			action:      panes.ActionToggleShuffle,
			wantAllowed: false,
			wantReason:  "Shuffle not available in this context",
		},
		{
			name: "cycle repeat — both repeat modes disallowed",
			ps: &domain.PlaybackState{
				Device: &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{
					TogglingRepeatContext: true,
					TogglingRepeatTrack:   true,
				}},
			},
			action:      panes.ActionCycleRepeat,
			wantAllowed: false,
			wantReason:  "Repeat not available in this context",
		},
		{
			name: "cycle repeat — only repeat-context disallowed (track still available) — allowed",
			ps: &domain.PlaybackState{
				Device: &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{
					TogglingRepeatContext: true,
					TogglingRepeatTrack:   false,
				}},
			},
			action:      panes.ActionCycleRepeat,
			wantAllowed: true,
		},
		{
			name: "play — resuming disallowed",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{Resuming: true}},
			},
			action:      panes.ActionPlay,
			wantAllowed: false,
			wantReason:  "Playback not available in this context",
		},
		{
			name: "pause — pausing disallowed",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{Disallows: domain.PlaybackActions{Pausing: true}},
			},
			action:      panes.ActionPause,
			wantAllowed: false,
			wantReason:  "Playback not available in this context",
		},
		{
			name: "all disallows false — action allowed",
			ps: &domain.PlaybackState{
				Device:  &domain.Device{SupportsVolume: true},
				Actions: domain.PlaybackActionsWrapper{},
			},
			action:      panes.ActionNext,
			wantAllowed: true,
		},
		{
			name: "nil device — volume check skipped (safe default)",
			ps: &domain.PlaybackState{
				Device: nil,
			},
			action:      panes.ActionVolumeUp,
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := checkCapability(tt.ps, tt.action)
			assert.Equal(t, tt.wantAllowed, got, "allowed mismatch")
			if !tt.wantAllowed {
				assert.Equal(t, tt.wantReason, reason, "reason mismatch")
			}
		})
	}
}
