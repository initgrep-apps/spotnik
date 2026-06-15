package viz

import "github.com/charmbracelet/lipgloss"

// Renderer produces a Frame from column heights, dimensions, and per-row colors.
// Implementations may use different character sets (braille, block, etc.).
type Renderer interface {
	// RenderFrame converts colHeights (one value per column, range [0, height*4] for
	// braille or [0, height] for block) into a Frame of height StyledLines.
	// colors must have at least height elements; colors[i] applies to row i.
	// Returns an empty Frame when width or height is zero.
	RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame

	// MaxHeight returns the maximum HeightFunc value for a given display height.
	// For braille this is height*4 (dot rows per display row).
	// For block this is height (one unit per display row).
	MaxHeight(displayHeight int) int
}

// FrameAwareRenderer is an optional interface for Renderers that need
// to know the frame index to compute per-cell density.
// When a Renderer implements this, the Engine calls RenderFrameAt
// instead of RenderFrame, and the HeightFunc result is unused for
// this pattern.
type FrameAwareRenderer interface {
	Renderer
	RenderFrameAt(width, height, frameIdx int, colors []lipgloss.Color) Frame
}
