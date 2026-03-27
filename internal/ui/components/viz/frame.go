// Package viz provides the visualizer engine for the NowPlaying pane.
// It defines a Renderer interface with braille and block implementations,
// an animation Pattern registry, and an Engine that manages frame
// precomputation and pattern cycling.
package viz

import "github.com/charmbracelet/lipgloss"

// StyledLine is a single display row with its text content and assigned color.
type StyledLine struct {
	Text  string
	Color lipgloss.Color
}

// Frame is a slice of StyledLines representing one animation frame.
// Index 0 is the top row; the last index is the bottom row.
type Frame []StyledLine
