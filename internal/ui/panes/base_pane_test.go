package panes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBasePane_DefaultBehaviour documents the zero-value state of BasePane:
// not focused, no active filter. This prevents regressions in the shared
// default behaviour that all 8 table-based panes embed.
func TestBasePane_DefaultBehaviour(t *testing.T) {
	var b BasePane

	assert.False(t, b.IsFocused(), "new BasePane should not be focused")
	assert.False(t, b.HasActiveFilter(), "default HasActiveFilter should return false")
}

// TestBasePane_SetFocused verifies that SetFocused toggles the focused field
// and IsFocused reflects the change correctly.
func TestBasePane_SetFocused(t *testing.T) {
	var b BasePane

	b.SetFocused(true)
	assert.True(t, b.IsFocused())

	b.SetFocused(false)
	assert.False(t, b.IsFocused())
}

// TestBasePane_SetSize verifies that SetSize stores width and height in the
// embedded struct fields used by pane View() implementations.
func TestBasePane_SetSize(t *testing.T) {
	var b BasePane
	b.SetSize(80, 24)

	assert.Equal(t, 80, b.width)
	assert.Equal(t, 24, b.height)
}
