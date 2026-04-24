package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverlayChrome_Unicode_DefaultBorderAccent(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	oc := uikit.OverlayChrome{
		Width: 40, Height: 10,
		Title:   "Search",
		Actions: []uikit.Action{{Key: "Enter", Label: "play"}, {Key: "Tab", Label: "filter"}},
		Theme:   th,
	}
	lines := uikit.Capture(oc.Render("  body"))
	require.Equal(t, 10, len(lines), "height must match")
	assert.True(t, strings.HasPrefix(lines[0], "╭─ Search"),
		"top-left corner + hrule + title, got: %q", lines[0])
	assert.Contains(t, lines[0], "╮ Enter play ╭",
		"first action notch")
}

func TestOverlayChrome_ASCII(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	oc := uikit.OverlayChrome{
		Width: 30, Height: 5,
		Title: "X",
		Theme: th,
	}
	lines := uikit.Capture(oc.Render(""))
	require.Len(t, lines, 5, "height must match")
	// No unicode corners in ascii mode.
	for _, l := range lines {
		assert.NotContains(t, l, "╭", "unexpected unicode corner in ascii mode")
		assert.NotContains(t, l, "╮", "unexpected unicode corner in ascii mode")
	}
}

func TestOverlayChrome_WidthAndHeightMatch(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	oc := uikit.OverlayChrome{
		Width: 50, Height: 6,
		Title: "Devices",
		Theme: th,
	}
	lines := uikit.Capture(oc.Render("line1\nline2\nline3\nline4"))
	assert.Len(t, lines, 6, "height matches")
	for i, l := range lines {
		// lipgloss.Width strips ANSI — check raw visible width.
		w := visibleWidth(l)
		assert.Equal(t, 50, w, "line %d width must be 50, got %d: %q", i, w, l)
	}
}

// visibleWidth measures terminal column width of a line without ANSI sequences.
// Thin wrapper using the same approach as Capture (strip ANSI then rune-count).
func visibleWidth(s string) int {
	stripped := uikit.Capture(s)
	if len(stripped) == 0 {
		return 0
	}
	return len([]rune(stripped[0]))
}
