package app

import "github.com/initgrep-apps/spotnik/internal/ui/panes"

// applyOptimisticUpdate writes the predicted post-action state to the store
// immediately when a PlaybackRequestMsg is received — before the API command
// fires. This gives instant UI feedback (the next render frame shows the new
// state) without waiting for the full HTTP round-trip.
//
// Only actions whose outcome is fully predictable from local state are handled.
// ActionNext and ActionPrevious are explicit no-ops: the next track is determined
// by Spotify, not by local state.
//
// When the API response arrives via PlaybackStateFetchedMsg, SetPlaybackState
// overwrites the optimistic value with authoritative data. On error, the
// fetchPlaybackStateCmd fired from the PlaybackCmdSentMsg error handler corrects
// the store automatically within ~200–400ms.
func (a *App) applyOptimisticUpdate(action panes.PlaybackAction) {
	ps := a.store.PlaybackState()
	if ps == nil {
		return
	}

	// Deep-copy: copy the struct value and any pointer fields to avoid aliasing.
	updated := *ps
	if ps.Device != nil {
		dev := *ps.Device
		updated.Device = &dev
	}

	switch action {
	case panes.ActionVolumeUp:
		if updated.Device != nil {
			v := updated.Device.VolumePercent + a.volumeStep
			if v > 100 {
				v = 100
			}
			updated.Device.VolumePercent = v
		}
	case panes.ActionVolumeDown:
		if updated.Device != nil {
			v := updated.Device.VolumePercent - a.volumeStep
			if v < 0 {
				v = 0
			}
			updated.Device.VolumePercent = v
		}
	case panes.ActionPause:
		updated.IsPlaying = false
	case panes.ActionPlay:
		updated.IsPlaying = true
	case panes.ActionToggleShuffle:
		updated.ShuffleState = !updated.ShuffleState
	case panes.ActionCycleRepeat:
		updated.RepeatState = nextRepeatMode(updated.RepeatState)
	case panes.ActionNext, panes.ActionPrevious:
		// no-op: next/previous track is determined by Spotify, not local state.
	}

	a.store.SetPlaybackState(&updated)
}
