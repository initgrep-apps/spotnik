package app

import (
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
)

// checkCapability is the single pre-flight gate for all playback key actions.
// It returns (true, "") when the action is permitted and (false, reason) when blocked.
//
// Three signal sources, checked in order:
//  1. device.IsRestricted — total lockout; no Web API commands accepted by this device.
//  2. device.SupportsVolume — volume-specific; only for ActionVolumeUp/Down.
//  3. playbackState.Actions.Disallows — runtime signal reflecting subscription tier,
//     content type (ads, DRM, radio), and device capability in one object.
//
// Returns (true, "") when ps is nil — no state yet; let the API respond.
// This function lives in app/ (not state/) to avoid importing ui/panes/ from state/.
func checkCapability(ps *domain.PlaybackState, action panes.PlaybackAction) (bool, string) {
	if ps == nil {
		return true, ""
	}

	// Total device lockout — no Web API commands accepted.
	if ps.Device != nil && ps.Device.IsRestricted {
		return false, "Device not controllable via API"
	}

	d := ps.Actions.Disallows

	switch action {
	case panes.ActionVolumeUp, panes.ActionVolumeDown:
		if ps.Device != nil && !ps.Device.SupportsVolume {
			return false, "Volume not available on this device"
		}
	case panes.ActionNext:
		if d.SkippingNext {
			return false, "Skip not available in this context"
		}
	case panes.ActionPrevious:
		if d.SkippingPrev {
			return false, "Skip not available in this context"
		}
	case panes.ActionToggleShuffle:
		if d.TogglingShuffle {
			return false, "Shuffle not available in this context"
		}
	case panes.ActionCycleRepeat:
		// Only blocked when BOTH repeat modes are disallowed — if either is available,
		// the cycle can still advance to that mode.
		if d.TogglingRepeatContext && d.TogglingRepeatTrack {
			return false, "Repeat not available in this context"
		}
	case panes.ActionPlay:
		if d.Resuming {
			return false, "Playback not available in this context"
		}
	case panes.ActionPause:
		if d.Pausing {
			return false, "Playback not available in this context"
		}
	}

	return true, ""
}
