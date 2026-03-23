// Package components provides reusable Bubble Tea UI components for the Spotnik TUI.
// All colors come from the Theme interface — never raw hex values.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// ProgressBar renders a seek bar with filled/empty characters and time labels.
// Width is passed at construction and adapts to the available pane width.
type ProgressBar struct {
	width      int
	fillStyle  lipgloss.Style
	emptyStyle lipgloss.Style
}

// NewProgressBar creates a ProgressBar using the given width and theme.
// FilledColor comes from theme.SeekBar(), empty from theme.Surface().
func NewProgressBar(width int, t theme.Theme) ProgressBar {
	return ProgressBar{
		width:      width,
		fillStyle:  lipgloss.NewStyle().Foreground(t.SeekBar()),
		emptyStyle: lipgloss.NewStyle().Foreground(t.Surface()),
	}
}

// Render returns a two-line string: the filled/empty bar, then the time label row.
// progressMs and durationMs are in milliseconds. Zero duration renders gracefully.
func (pb ProgressBar) Render(progressMs, durationMs int) string {
	// Reserve space for 4 padding chars (2 each side), min bar width = 1.
	barWidth := pb.width - 4
	if barWidth < 1 {
		barWidth = 1
	}

	// Calculate fill ratio.
	var ratio float64
	if durationMs > 0 {
		ratio = float64(progressMs) / float64(durationMs)
	}
	if ratio > 1.0 {
		ratio = 1.0
	}

	filled := int(ratio * float64(barWidth))
	empty := barWidth - filled

	bar := pb.fillStyle.Render(strings.Repeat("█", filled)) +
		pb.emptyStyle.Render(strings.Repeat("░", empty))

	elapsed := formatDuration(progressMs)
	total := formatDuration(durationMs)

	// Build the time row: elapsed on left, total on right, fill the middle.
	labelWidth := pb.width - len(elapsed) - len(total) - 4
	if labelWidth < 1 {
		labelWidth = 1
	}
	timeRow := elapsed + " " + strings.Repeat("─", labelWidth) + " " + total

	return bar + "\n" + timeRow
}

// formatDuration converts milliseconds to "m:ss" string (e.g. 154000 → "2:34").
func formatDuration(ms int) string {
	totalSeconds := ms / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}
