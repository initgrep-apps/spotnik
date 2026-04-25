package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

// TestPanelSize_Proportional verifies the 70%/65% policy on a wide terminal.
// A 220×50 terminal should yield ≈154×32 (rounded down by integer division).
func TestPanelSize_Proportional(t *testing.T) {
	w, h := uikit.PanelSize(220, 50)
	assert.Equal(t, 154, w, "width should be 70%% of 220")
	assert.Equal(t, 32, h, "height should be 65%% of 50")
}

// TestPanelSize_MinimumClamp verifies that a narrow/short terminal is clamped
// to the minimum dimensions (80×20) rather than returning something smaller.
func TestPanelSize_MinimumClamp(t *testing.T) {
	w, h := uikit.PanelSize(50, 10)
	assert.Equal(t, 80, w, "width must be clamped to minimum 80")
	assert.Equal(t, 20, h, "height must be clamped to minimum 20")
}

// TestPanelSize_Zero verifies that a 0×0 terminal (e.g. during unit tests
// before the first WindowSizeMsg) returns the safe minimum dimensions.
func TestPanelSize_Zero(t *testing.T) {
	w, h := uikit.PanelSize(0, 0)
	assert.Equal(t, 80, w, "width must be at least 80 for a zero terminal")
	assert.Equal(t, 20, h, "height must be at least 20 for a zero terminal")
}
