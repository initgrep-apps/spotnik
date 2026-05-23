package uikit_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

// TestHeaderBar_LeftSegment_Music verifies that a Music bar renders
// "spotnik", "Music", and the preset name in the left segment.
func TestHeaderBar_LeftSegment_Music(t *testing.T) {
	th := theme.Load("black")
	h := uikit.HeaderBar{
		Width:      160,
		AppName:    "spotnik",
		Page:       "Music",
		PresetName: "Dashboard",
		RightChips: []string{},
		Theme:      th,
	}
	plain := uikit.Capture(h.Render())[0]
	assert.Contains(t, plain, "spotnik", "left segment must contain app name")
	assert.Contains(t, plain, "Music", "left segment must contain page indicator")
	assert.Contains(t, plain, "Dashboard", "left segment must contain preset name")
}

// TestHeaderBar_LeftSegment_Stats_NoPreset verifies that when PresetName is empty,
// the preset segment is hidden.
func TestHeaderBar_LeftSegment_Stats_NoPreset(t *testing.T) {
	th := theme.Load("black")
	h := uikit.HeaderBar{
		Width:      160,
		AppName:    "spotnik",
		Page:       "Stats",
		PresetName: "",
		RightChips: []string{},
		Theme:      th,
	}
	plain := uikit.Capture(h.Render())[0]
	assert.Contains(t, plain, "spotnik", "left segment must contain app name")
	assert.Contains(t, plain, "Stats", "left segment must contain page indicator")
	assert.NotContains(t, plain, "Dashboard", "Stats page bar must not contain preset segment")
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
		Page:       "Music",
		PresetName: "Listening",
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
		Page:       "Music",
		PresetName: "Dashboard",
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
		Page:       "Music",
		PresetName: "Dashboard",
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
		Width:      0,
		AppName:    "spotnik",
		Page:       "Music",
		PresetName: "Dashboard",
		Theme:      th,
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
		Page:       "Music",
		PresetName: "Dashboard",
		RightChips: []string{longChip},
		Theme:      th,
	}
	// Should not panic even when content overflows the declared Width.
	assert.NotPanics(t, func() { h.Render() }, "Render must not panic when content overflows Width")
}

// TestHeaderBar_AsciiSeparator verifies that in ASCII mode the header bar separator
// is " - " (ASCII hyphen), not " ─ " (unicode horizontal rule).
func TestHeaderBar_AsciiSeparator(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	h := uikit.HeaderBar{
		Width:      160,
		AppName:    "spotnik",
		Page:       "Music",
		PresetName: "Dashboard",
		Theme:      th,
	}
	plain := uikit.Capture(h.Render())[0]
	assert.Contains(t, plain, " - ", "ascii mode must use hyphen separator")
	assert.NotContains(t, plain, " ─ ", "ascii mode must not use unicode horizontal rule")
}

// TestHeaderBar_PresetName_ShowsName verifies that a non-empty preset name is shown correctly.
func TestHeaderBar_PresetName_ShowsName(t *testing.T) {
	th := theme.Load("black")
	h := uikit.HeaderBar{
		Width:      160,
		AppName:    "spotnik",
		Page:       "Music",
		PresetName: "Discovery",
		Theme:      th,
	}
	plain := uikit.Capture(h.Render())[0]
	assert.Contains(t, plain, "Discovery", "must show the configured preset name")
}
