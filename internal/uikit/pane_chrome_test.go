package uikit_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaneChrome_UnicodeSnapshot_ActionsMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	pc := uikit.PaneChrome{
		Width: 60, Height: 4,
		Title: "Playlists", ToggleKey: 3,
		Actions:     []layout.Action{{Key: "f", Label: "filter"}, {Key: "n", Label: "new"}},
		AccentColor: th.PaneBorderPlaylists(),
		Focused:     true, Theme: th,
	}
	out := pc.Render("  (content)")
	lines := uikit.Capture(out)
	require.Len(t, lines, 4)

	assert.True(t, strings.HasPrefix(lines[0], "╭─ ³Playlists"),
		"title immediately after '─ ', no trailing space")
	assert.Contains(t, lines[0], "╮ f filter ╭",
		"first action notch")
	assert.Contains(t, lines[0], "╮ n new ╭",
		"second action notch")
	assert.True(t, strings.HasSuffix(lines[0], "╭╮"),
		"last action ╭ immediately followed by top-right corner ╮")
}

func TestPaneChrome_ASCIISnapshot_ActionsMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	pc := uikit.PaneChrome{
		Width: 60, Height: 4,
		Title: "Playlists", ToggleKey: 3,
		Actions:     []layout.Action{{Key: "f", Label: "filter"}},
		AccentColor: th.PaneBorderPlaylists(),
		Focused:     true, Theme: th,
	}
	lines := uikit.Capture(pc.Render(""))
	assert.True(t, strings.HasPrefix(lines[0], "+- 3 Playlists"))
	assert.Contains(t, lines[0], "+ f filter +")
	assert.True(t, strings.HasSuffix(lines[0], "++"))
	// No unicode corners anywhere.
	for _, l := range lines {
		assert.NotContains(t, l, "╭")
		assert.NotContains(t, l, "╮")
	}
}

func TestPaneChrome_FilterMode_NoArrow(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	pc := uikit.PaneChrome{
		Width: 60, Height: 3,
		Title: "Queue", ToggleKey: 2,
		FilterQuery: "rock",
		AccentColor: th.PaneBorderQueue(),
		Focused:     true, Theme: th,
	}
	lines := uikit.Capture(pc.Render(""))
	assert.NotContains(t, lines[0], "ᐅ")
	assert.Contains(t, lines[0], "f(rock)")
	assert.NotContains(t, lines[0], `filtering:`)
	assert.NotContains(t, lines[0], "╮ Esc close ╭")
}

func TestPaneChrome_UnfocusedTitleNotBold(t *testing.T) {
	// Structural assertion: when Focused=false, title is rendered without bold.
	// We don't have a direct "bold" check on the plain string, so assert
	// that the raw bytes don't contain the ANSI bold sequence (ESC[1m).
	th := theme.Load("black")
	pc := uikit.PaneChrome{
		Width: 40, Height: 3, Title: "Test",
		AccentColor: th.PaneBorderPlaylists(), Focused: false, Theme: th,
	}
	raw := pc.Render("")
	assert.NotContains(t, raw, "\x1b[1m", "unfocused title must not be bold")
}

func TestPaneChrome_WidthAndHeightMatch(t *testing.T) {
	th := theme.Load("black")
	pc := uikit.PaneChrome{
		Width: 50, Height: 5, Title: "X",
		AccentColor: th.PaneBorderPlaylists(), Focused: true, Theme: th,
	}
	lines := uikit.Capture(pc.Render("line1\nline2\nline3"))
	assert.Len(t, lines, 5, "height matches")
	for _, l := range lines {
		assert.Equal(t, 50, lipgloss.Width(l), "width matches")
	}
}
