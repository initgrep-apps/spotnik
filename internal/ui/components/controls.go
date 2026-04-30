package components

import (
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// Controls renders the transport controls row: shuffle  play/pause  queue  repeat.
// It is a thin compatibility wrapper around uikit.PlaybackControls that translates
// the legacy string repeat-mode arg ("off" / "context" / "track") to the typed
// uikit.RepeatMode enum. Callers use the same NewControls signature as before.
type Controls struct {
	inner uikit.PlaybackControls
}

// NewControls creates a Controls renderer with the given state and theme.
// repeatMode must be one of "off", "context", or "track"; any other value is
// treated as "off" (RepeatOff).
func NewControls(t theme.Theme, isPlaying, shuffleOn bool, repeatMode string) Controls {
	var rm uikit.RepeatMode
	switch repeatMode {
	case "track":
		rm = uikit.RepeatOne
	case "context":
		rm = uikit.RepeatAll
	default:
		rm = uikit.RepeatOff
	}
	return Controls{
		inner: uikit.PlaybackControls{
			Playing:    isPlaying,
			Shuffle:    shuffleOn,
			RepeatMode: rm,
			Theme:      t,
		},
	}
}

// Render returns the controls row as a string.
func (c Controls) Render() string { return c.inner.Render() }
