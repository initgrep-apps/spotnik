package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// volumeBarWidth is the fixed number of characters in the volume bar fill area.
const volumeBarWidth = 14

// VolumeBar renders a volume indicator with "VOL ████████░░░░░░ 65%" format.
// The bar width is fixed at 14 characters, and volume is clamped to [0, 100].
type VolumeBar struct {
	fillStyle  lipgloss.Style
	emptyStyle lipgloss.Style
}

// NewVolumeBar creates a VolumeBar using colors from the given theme.
func NewVolumeBar(t theme.Theme) VolumeBar {
	return VolumeBar{
		fillStyle:  lipgloss.NewStyle().Foreground(t.VolumeBar()),
		emptyStyle: lipgloss.NewStyle().Foreground(t.Surface()),
	}
}

// Render returns the volume bar string for the given volume level.
// Volume is clamped to [0, 100].
func (vb VolumeBar) Render(volume int) string {
	if volume > 100 {
		volume = 100
	}
	if volume < 0 {
		volume = 0
	}

	filled := int(float64(volume) / 100.0 * float64(volumeBarWidth))
	empty := volumeBarWidth - filled

	bar := vb.fillStyle.Render(strings.Repeat("█", filled)) +
		vb.emptyStyle.Render(strings.Repeat("░", empty))

	return fmt.Sprintf("VOL  %s  %d%%", bar, volume)
}
