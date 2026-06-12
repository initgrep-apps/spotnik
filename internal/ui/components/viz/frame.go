// Package viz provides the visualizer engine for the NowPlaying pane.
// It defines a Renderer interface with braille and block implementations,
// an animation Pattern registry, and an Engine that manages frame
// precomputation and pattern cycling.
package viz

import "github.com/charmbracelet/lipgloss"

// StyledSegment is a contiguous run of text within a line that shares a single
// foreground color. When Segments is populated on a StyledLine, the renderer
// uses per-segment colors instead of the line-level Color field.
type StyledSegment struct {
	Text  string
	Color lipgloss.Color
}

// StyledLine is a single display row with its text content and assigned color.
// For patterns that use per-column color shifts (e.g. Spectrum Sweep), Segments
// is populated with one segment per color boundary; the line-level Color is
// ignored. For all other patterns, Segments is nil and Color is used directly.
type StyledLine struct {
	Text     string
	Color    lipgloss.Color
	Segments []StyledSegment
}

// Frame is a slice of StyledLines representing one animation frame.
// Index 0 is the top row; the last index is the bottom row.
type Frame []StyledLine
