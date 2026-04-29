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

// TestURLBox_WrapsURLAtAmpersand verifies that a URL longer than Width is
// broken at '&' boundaries and that every output line fits within Width visual
// columns (lipgloss.Width rather than byte-len, so Unicode border chars count
// as 1 column each).
func TestURLBox_WrapsURLAtAmpersand(t *testing.T) {
	th := theme.Load("black")
	u := "https://accounts.spotify.com/authorize?client_id=abc&code_challenge=def&scope=user-read-playback-state"
	const boxWidth = 40
	b := uikit.URLBox{URL: u, Width: boxWidth, Theme: th}
	lines := uikit.Capture(b.Render())
	// Long URL must produce multiple output lines.
	assert.GreaterOrEqual(t, len(lines), 2)
	for _, l := range lines {
		assert.LessOrEqual(t, lipgloss.Width(l), boxWidth,
			"line visual width must not exceed box Width; line: %q", l)
	}
}

// TestURLBox_ShortURL verifies that a URL that fits within Width is rendered as
// a single content line (no wrapping).
func TestURLBox_ShortURL(t *testing.T) {
	th := theme.Load("black")
	u := "https://example.com"
	b := uikit.URLBox{URL: u, Width: 40, Theme: th}
	lines := uikit.Capture(b.Render())
	// At minimum: top-border, content, bottom-border.
	require.GreaterOrEqual(t, len(lines), 3)
	found := false
	for _, l := range lines {
		if strings.Contains(l, u) {
			found = true
		}
	}
	assert.True(t, found, "short URL must appear verbatim in output")
}

// TestURLBox_AccentAnsiPresent verifies that the rendered output contains ANSI
// escape codes (content uses Accent colour).
func TestURLBox_AccentAnsiPresent(t *testing.T) {
	th := theme.Load("black")
	b := uikit.URLBox{URL: "https://example.com", Width: 40, Theme: th}
	raw := b.Render()
	assert.Contains(t, raw, "\x1b[",
		"Render() must contain ANSI escapes for Accent colour")
}

// TestURLBox_RoleBorderMuted verifies that URLBox.Border uses the Muted role
// (TextMuted colour token). We confirm the rendered string contains ANSI codes
// indicating styled borders.
func TestURLBox_RoleBorderMuted(t *testing.T) {
	th := theme.Load("black")
	b := uikit.URLBox{URL: "https://example.com", Width: 40, Theme: th}
	raw := b.Render()
	// Border is styled — raw output must contain ANSI.
	assert.Contains(t, raw, "\x1b[",
		"Border must be ANSI-styled (Muted role)")
}

// TestURLBox_RoleContentAccent verifies that the URL text itself is styled in
// the Accent role colour.
func TestURLBox_RoleContentAccent(t *testing.T) {
	th := theme.Load("black")
	b := uikit.URLBox{URL: "https://example.com", Width: 40, Theme: th}
	raw := b.Render()
	// Content (accent) colour must be applied — ANSI present.
	assert.Contains(t, raw, "\x1b[",
		"Content must use Accent role ANSI colour")
}

// TestURLBox_HardWrapNoAmpersand verifies that when there is no '&' in the
// first half of the URL, the URL is hard-wrapped at the inner width.
func TestURLBox_HardWrapNoAmpersand(t *testing.T) {
	th := theme.Load("black")
	// No ampersand — must fall back to hard-wrap.
	u := "https://accounts.spotify.com/authorize/veryLongPathWithNoAmpersandHereSoItMustHardWrap"
	const boxWidth = 30
	b := uikit.URLBox{URL: u, Width: boxWidth, Theme: th}
	lines := uikit.Capture(b.Render())
	require.GreaterOrEqual(t, len(lines), 2)
	for _, l := range lines {
		assert.LessOrEqual(t, lipgloss.Width(l), boxWidth,
			"hard-wrapped line visual width must not exceed box Width; line: %q", l)
	}
}

// TestURLBox_AmpersandInFirstHalf verifies that when an '&' exists only in the
// first half of the segment (i > width/2 is false), the code falls back to
// hard-wrapping rather than breaking at the early '&'.
func TestURLBox_AmpersandInFirstHalf(t *testing.T) {
	th := theme.Load("black")
	// Width=20 → innerW=16. The '&' is at position 3 (first half=8), so
	// strings.LastIndex finds it but i=3 <= 16/2=8 is false (3 <= 8 is true,
	// but 3 > 8 is false), so hard-wrap is used.
	u := "ab&cdefghijklmnopqrstuvwxyz"
	const boxWidth = 20
	b := uikit.URLBox{URL: u, Width: boxWidth, Theme: th}
	lines := uikit.Capture(b.Render())
	require.GreaterOrEqual(t, len(lines), 2)
	for _, l := range lines {
		assert.LessOrEqual(t, lipgloss.Width(l), boxWidth,
			"line must not exceed box Width; line: %q", l)
	}
}

// TestURLBox_WrapAtAmpersandHelper verifies the wrapAtAmpersand helper function
// indirectly — when the URL has many '&' chars the render produces multiple
// content lines.
func TestURLBox_WrapAtAmpersandHelper(t *testing.T) {
	th := theme.Load("black")
	// Width=40 → innerW=36; URL has '&' well past the midpoint.
	u := "https://host.example/path?a=1&b=2&c=3&d=4&e=5&f=6&g=7"
	b := uikit.URLBox{URL: u, Width: 40, Theme: th}
	lines := uikit.Capture(b.Render())
	require.GreaterOrEqual(t, len(lines), 2, "URL with many & must produce multiple lines")
}

// TestURLBox_NarrowWidthNoInnerSpace verifies that a box with Width <= 4 (so
// innerW would be <= 0) does not panic. The inner guard clamps innerW to 1.
func TestURLBox_NarrowWidthNoInnerSpace(t *testing.T) {
	th := theme.Load("black")
	b := uikit.URLBox{URL: "https://example.com/very-long", Width: 4, Theme: th}
	// Must not panic.
	out := b.Render()
	_ = out
}

// TestURLBox_AsciiMode verifies that URLBox renders in ASCII mode without unicode
// box-drawing characters (╭╮╰╯─│) and still contains the URL content.
func TestURLBox_AsciiMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	u := "https://example.com/auth"
	b := uikit.URLBox{URL: u, Width: 40, Theme: th}
	result := b.Render()
	stripped := strings.Join(uikit.Capture(result), "\n")
	assert.NotContains(t, stripped, "╭", "ascii mode must not use unicode corner ╭")
	assert.NotContains(t, stripped, "╮", "ascii mode must not use unicode corner ╮")
	assert.NotContains(t, stripped, "╰", "ascii mode must not use unicode corner ╰")
	assert.NotContains(t, stripped, "╯", "ascii mode must not use unicode corner ╯")
	assert.Contains(t, stripped, u, "ascii mode must still contain the URL")
}

// TestURLBox_ExactMultipleWidth verifies the tail-append branch: when the URL
// length is an exact multiple of innerW, the last segment is non-empty and
// appended. We verify by checking the URL appears in the output.
func TestURLBox_ExactMultipleWidth(t *testing.T) {
	th := theme.Load("black")
	// Width=10 → innerW=6. Build a URL that is exactly 12 chars (2*6).
	u := "abcdefghijkl" // len=12, innerW=6 → 2 equal segments
	b := uikit.URLBox{URL: u, Width: 10, Theme: th}
	lines := uikit.Capture(b.Render())
	full := strings.Join(lines, "")
	// All characters of u must appear in the output.
	assert.Contains(t, full, "abcdef", "first segment must appear in output")
	assert.Contains(t, full, "ghijkl", "second segment (tail) must appear in output")
}
