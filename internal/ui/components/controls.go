package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Controls renders the transport controls row: ⇄ ▷/⏸ ≡ ↻
// Three visual states per icon:
//   - active:   PlayingIndicator color (on and available)
//   - inactive: TextSecondary color   (off but available)
//   - disabled: TextMuted color       (unavailable per device/context/subscription)
type Controls struct {
	isPlaying      bool
	shuffleOn      bool
	repeatMode     string
	disallows      domain.PlaybackActions
	supportsVolume bool

	activeStyle   lipgloss.Style
	inactiveStyle lipgloss.Style
	disabledStyle lipgloss.Style
}

// NewControls creates a Controls renderer with the given state, capability context, and theme.
// repeatMode must be one of "off", "context", or "track".
// disallows reflects the current Spotify actions.disallows object from PlaybackState.
// supportsVolume is device.SupportsVolume from the active device.
func NewControls(t theme.Theme, isPlaying, shuffleOn bool, repeatMode string, disallows domain.PlaybackActions, supportsVolume bool) Controls {
	return Controls{
		isPlaying:      isPlaying,
		shuffleOn:      shuffleOn,
		repeatMode:     repeatMode,
		disallows:      disallows,
		supportsVolume: supportsVolume,
		activeStyle:    lipgloss.NewStyle().Foreground(t.PlayingIndicator()),
		inactiveStyle:  lipgloss.NewStyle().Foreground(t.TextSecondary()),
		disabledStyle:  lipgloss.NewStyle().Foreground(t.TextMuted()),
	}
}

// Render returns the controls row as a string.
func (c Controls) Render() string {
	// Shuffle: active (on), inactive (off), disabled (Spotify disallows toggling).
	var shuffle string
	switch {
	case c.disallows.TogglingShuffle:
		shuffle = c.disabledStyle.Render("⇄")
	case c.shuffleOn:
		shuffle = c.activeStyle.Render("⇄")
	default:
		shuffle = c.inactiveStyle.Render("⇄")
	}

	// Play/Pause: disabled when the current state's action is disallowed.
	var playPause string
	switch {
	case c.isPlaying && c.disallows.Pausing:
		playPause = c.disabledStyle.Render("⏸")
	case !c.isPlaying && c.disallows.Resuming:
		playPause = c.disabledStyle.Render("▷")
	case c.isPlaying:
		playPause = c.activeStyle.Render("⏸")
	default:
		playPause = c.inactiveStyle.Render("▷")
	}

	queue := c.inactiveStyle.Render("≡")

	// Repeat: disabled only when BOTH modes are disallowed.
	// ↻¹ uses superscript one (U+00B9) for visual balance with the arrow glyph.
	repeatDisabled := c.disallows.TogglingRepeatContext && c.disallows.TogglingRepeatTrack
	var repeat string
	switch {
	case repeatDisabled:
		if c.repeatMode == "track" {
			repeat = c.disabledStyle.Render("↻¹")
		} else {
			repeat = c.disabledStyle.Render("↻")
		}
	case c.repeatMode == "track":
		repeat = c.activeStyle.Render("↻¹")
	case c.repeatMode == "context":
		repeat = c.activeStyle.Render("↻")
	default:
		repeat = c.inactiveStyle.Render("↻")
	}

	return shuffle + "  " + playPause + "  " + queue + "  " + repeat
}
