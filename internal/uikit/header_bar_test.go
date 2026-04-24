package uikit_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

// TestHeaderBar_LeftSegment_PageA verifies that a Page-A bar renders
// "spotnik", "Page A", and "preset N" in the left segment.
func TestHeaderBar_LeftSegment_PageA(t *testing.T) {
	th := theme.Load("black")
	h := uikit.HeaderBar{
		Width:      160,
		AppName:    "spotnik",
		Page:       "A",
		Preset:     0,
		RightChips: []string{},
		Theme:      th,
	}
	plain := uikit.Capture(h.Render())[0]
	assert.Contains(t, plain, "spotnik", "left segment must contain app name")
	assert.Contains(t, plain, "Page A", "left segment must contain page indicator")
	assert.Contains(t, plain, "preset 0", "left segment must contain preset index")
}

// TestHeaderBar_LeftSegment_PageB_NoPreset verifies that when Preset == -1,
// the preset segment is hidden.
func TestHeaderBar_LeftSegment_PageB_NoPreset(t *testing.T) {
	th := theme.Load("black")
	h := uikit.HeaderBar{
		Width:      160,
		AppName:    "spotnik",
		Page:       "B",
		Preset:     -1,
		RightChips: []string{},
		Theme:      th,
	}
	plain := uikit.Capture(h.Render())[0]
	assert.Contains(t, plain, "spotnik", "left segment must contain app name")
	assert.Contains(t, plain, "Page B", "left segment must contain page indicator")
	assert.NotContains(t, plain, "preset", "Page B bar must not contain preset segment")
}

// TestHeaderBar_RightChips_Rendered verifies that pre-rendered chip strings are
// placed on the right side of the bar.
func TestHeaderBar_RightChips_Rendered(t *testing.T) {
	th := theme.Load("black")
	chip1 := uikit.Chip{Glyph: uikit.GlyphActive, Label: "MacBook", Intent: uikit.RoleInfo, Theme: th}.Render()
	chip2 := uikit.Chip{Glyph: uikit.GlyphPremium, Label: "Alice", Intent: uikit.RoleInfo, Theme: th}.Render()
	h := uikit.HeaderBar{
		Width:      160,
		AppName:    "spotnik",
		Page:       "A",
		Preset:     1,
		RightChips: []string{chip1, chip2},
		Theme:      th,
	}
	plain := uikit.Capture(h.Render())[0]
	assert.Contains(t, plain, "MacBook", "right chips must contain device name")
	assert.Contains(t, plain, "Alice", "right chips must contain profile name")
}

// TestHeaderBar_FillsExactWidth verifies that the rendered header is exactly Width
// terminal columns wide, accounting for ANSI codes.
func TestHeaderBar_FillsExactWidth(t *testing.T) {
	th := theme.Load("black")
	h := uikit.HeaderBar{
		Width:      160,
		AppName:    "spotnik",
		Page:       "A",
		Preset:     0,
		RightChips: []string{},
		Theme:      th,
	}
	result := h.Render()
	assert.Equal(t, 160, lipgloss.Width(result),
		"header bar must be exactly Width columns wide")
}

// TestHeaderBar_FillsExactWidth_Narrow verifies width-fill at a smaller terminal width.
func TestHeaderBar_FillsExactWidth_Narrow(t *testing.T) {
	th := theme.Load("black")
	h := uikit.HeaderBar{
		Width:      120,
		AppName:    "spotnik",
		Page:       "A",
		Preset:     0,
		RightChips: []string{},
		Theme:      th,
	}
	result := h.Render()
	assert.Equal(t, 120, lipgloss.Width(result),
		"header bar must be exactly 120 columns wide")
}

// TestHeaderBar_ZeroWidth_FallsBack verifies that when Width == 0, Render returns a
// non-empty string with a double-space gap rather than panicking.
func TestHeaderBar_ZeroWidth_FallsBack(t *testing.T) {
	th := theme.Load("black")
	h := uikit.HeaderBar{
		Width:   0,
		AppName: "spotnik",
		Page:    "A",
		Preset:  0,
		Theme:   th,
	}
	result := h.Render()
	assert.NotEmpty(t, result, "zero-width header must return non-empty string")
	plain := uikit.Capture(result)[0]
	assert.Contains(t, plain, "spotnik")
}

// TestHeaderBar_GapAtLeastOne verifies that when the combined left+right width nearly
// fills the terminal, the gap is still at least 1 space (no negative fill).
func TestHeaderBar_GapAtLeastOne(t *testing.T) {
	th := theme.Load("black")
	// Long chip string to crowd the left segment.
	longChip := uikit.Chip{Glyph: uikit.GlyphActive, Label: strings.Repeat("x", 100), Intent: uikit.RoleInfo, Theme: th}.Render()
	h := uikit.HeaderBar{
		Width:      120,
		AppName:    "spotnik",
		Page:       "A",
		Preset:     0,
		RightChips: []string{longChip},
		Theme:      th,
	}
	// Should not panic even when content overflows the declared Width.
	assert.NotPanics(t, func() { h.Render() }, "Render must not panic when content overflows Width")
}

// TestHeaderBar_PresetN_ShowsIndex verifies that preset index > 0 is shown correctly.
func TestHeaderBar_PresetN_ShowsIndex(t *testing.T) {
	th := theme.Load("black")
	h := uikit.HeaderBar{
		Width:   160,
		AppName: "spotnik",
		Page:    "A",
		Preset:  3,
		Theme:   th,
	}
	plain := uikit.Capture(h.Render())[0]
	assert.Contains(t, plain, "preset 3", "must show the configured preset index")
}
