package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Controls renders the transport controls row: ⏮ ⏸/▶ ⏭ 🔀 🔁/🔂
// Active icons (shuffle on, repeat on) use PlayingIndicator() color.
// Inactive icons use TextSecondary() color.
type Controls struct {
	isPlaying  bool
	shuffleOn  bool
	repeatMode string

	activeStyle   lipgloss.Style
	inactiveStyle lipgloss.Style
}

// NewControls creates a Controls renderer with the given state and theme.
// repeatMode must be one of "off", "context", or "track".
func NewControls(t theme.Theme, isPlaying, shuffleOn bool, repeatMode string) Controls {
	return Controls{
		isPlaying:     isPlaying,
		shuffleOn:     shuffleOn,
		repeatMode:    repeatMode,
		activeStyle:   lipgloss.NewStyle().Foreground(t.PlayingIndicator()),
		inactiveStyle: lipgloss.NewStyle().Foreground(t.TextSecondary()),
	}
}

// Render returns the controls row as a string.
func (c Controls) Render() string {
	prev := c.inactiveStyle.Render("⏮")

	var playPause string
	if c.isPlaying {
		playPause = c.activeStyle.Render("⏸")
	} else {
		playPause = c.inactiveStyle.Render("▶")
	}

	next := c.inactiveStyle.Render("⏭")

	var shuffle string
	if c.shuffleOn {
		shuffle = c.activeStyle.Render("🔀")
	} else {
		shuffle = c.inactiveStyle.Render("🔀")
	}

	var repeat string
	switch c.repeatMode {
	case "track":
		repeat = c.activeStyle.Render("🔂")
	case "context":
		repeat = c.activeStyle.Render("🔁")
	default:
		// off
		repeat = c.inactiveStyle.Render("🔁")
	}

	return prev + "   " + playPause + "   " + next + "      " + shuffle + "   " + repeat
}
