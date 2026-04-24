package uikit_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPanel_TitleInBorder verifies that the Panel title appears on the top
// border line (in-border rendering, not as a separate header).
func TestPanel_TitleInBorder(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	p := uikit.Panel{
		Width: 60, Height: 10,
		Title:  "Onboarding",
		Intent: uikit.PanelIntentDefault,
		Theme:  th,
	}
	lines := uikit.Capture(p.Render(""))
	require.Len(t, lines, 10, "height must match Panel.Height")

	// Title must appear on the top border line (lines[0]), not as a separate row.
	assert.Contains(t, lines[0], "Onboarding", "title must appear in top border line")
	// Rounded corners in unicode mode.
	assert.True(t, strings.HasPrefix(lines[0], "╭"), "top border must start with rounded corner ╭")
	assert.True(t, strings.HasSuffix(lines[0], "╮"), "top border must end with rounded corner ╮")
}

// TestPanel_ErrorIntent_UsesErrorBorder verifies that the error intent produces
// a border using the theme's Error() colour token (not Accent()).
func TestPanel_ErrorIntent_UsesErrorBorder(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	pErr := uikit.Panel{
		Width: 50, Height: 8,
		Title:  "Error",
		Intent: uikit.PanelIntentError,
		Theme:  th,
	}
	pDef := uikit.Panel{
		Width: 50, Height: 8,
		Title:  "Error",
		Intent: uikit.PanelIntentDefault,
		Theme:  th,
	}

	outErr := pErr.Render("")
	outDef := pDef.Render("")

	// The two renders must differ — error intent applies a different colour.
	// We check by comparing raw strings: ANSI codes differ between Error() and Accent().
	assert.NotEqual(t, outErr, outDef,
		"error intent and default intent must produce different output (different border colour)")

	// Both must have the title in the top border line.
	linesErr := uikit.Capture(outErr)
	require.Len(t, linesErr, 8, "height must match Panel.Height")
	assert.Contains(t, linesErr[0], "Error", "title must appear in top border line")
}

// TestPanel_ASCIIMode verifies ascii corners are used in GlyphASCII mode.
func TestPanel_ASCIIMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	p := uikit.Panel{
		Width: 40, Height: 5,
		Title:  "Setup",
		Intent: uikit.PanelIntentDefault,
		Theme:  th,
	}
	lines := uikit.Capture(p.Render(""))
	require.Len(t, lines, 5, "height must match Panel.Height")
	for _, l := range lines {
		assert.NotContains(t, l, "╭", "no unicode corners in ascii mode")
		assert.NotContains(t, l, "╮", "no unicode corners in ascii mode")
	}
}

// TestPanel_WidthAndHeightMatch verifies dimensional correctness.
func TestPanel_WidthAndHeightMatch(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	p := uikit.Panel{
		Width: 50, Height: 6,
		Title:  "Auth",
		Intent: uikit.PanelIntentDefault,
		Theme:  th,
	}
	lines := uikit.Capture(p.Render("line1\nline2\nline3\nline4"))
	assert.Len(t, lines, 6, "height must match Panel.Height")
	for i, l := range lines {
		assert.Equal(t, 50, lipgloss.Width(l),
			"line %d must have width 50, got: %q", i, l)
	}
}
