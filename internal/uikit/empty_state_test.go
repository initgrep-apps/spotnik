package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmptyState_CentersText verifies that Render returns exactly Height lines
// and that the text appears neither on the first nor on the last line
// (i.e. it is vertically centered).
func TestEmptyState_CentersText(t *testing.T) {
	th := theme.Load("black")
	es := uikit.EmptyState{
		Text: "Empty queue", Hint: "Press / to search",
		Width: 40, Height: 6, Theme: th,
	}
	lines := uikit.Capture(es.Render())
	assert.Equal(t, 6, len(lines))
	// Find the line containing the text; it should be roughly centered.
	var textLine int
	for i, l := range lines {
		if containsSubstr(l, "Empty queue") {
			textLine = i
			break
		}
	}
	assert.Greater(t, textLine, 0, "text not on first line")
	assert.Less(t, textLine, 5, "text not on last line")
}

// TestEmptyState_ExactHeight verifies that Render returns exactly Height lines.
func TestEmptyState_ExactHeight(t *testing.T) {
	th := theme.Load("black")
	tests := []struct {
		name   string
		height int
	}{
		{"height 1", 1},
		{"height 3", 3},
		{"height 10", 10},
		{"height 20", 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := uikit.EmptyState{
				Text: "No items", Width: 30, Height: tt.height, Theme: th,
			}
			lines := uikit.Capture(es.Render())
			require.Equal(t, tt.height, len(lines), "must return exactly %d lines", tt.height)
		})
	}
}

// TestEmptyState_HintBelowText verifies that when a hint is present it renders
// on the line immediately after the text line.
func TestEmptyState_HintBelowText(t *testing.T) {
	th := theme.Load("black")
	es := uikit.EmptyState{
		Text: "Empty queue", Hint: "Press / to search",
		Width: 60, Height: 8, Theme: th,
	}
	lines := uikit.Capture(es.Render())
	require.Len(t, lines, 8)
	textIdx := -1
	hintIdx := -1
	for i, l := range lines {
		if containsSubstr(l, "Empty queue") {
			textIdx = i
		}
		if containsSubstr(l, "Press / to search") {
			hintIdx = i
		}
	}
	require.GreaterOrEqual(t, textIdx, 0, "text line not found")
	require.GreaterOrEqual(t, hintIdx, 0, "hint line not found")
	assert.Equal(t, textIdx+1, hintIdx, "hint must appear on the line after text")
}

// TestEmptyState_NoHint verifies that when Hint is empty the text still renders
// and the output has exactly Height lines.
func TestEmptyState_NoHint(t *testing.T) {
	th := theme.Load("black")
	es := uikit.EmptyState{
		Text: "No items", Width: 40, Height: 5, Theme: th,
	}
	lines := uikit.Capture(es.Render())
	require.Len(t, lines, 5)
	found := false
	for _, l := range lines {
		if containsSubstr(l, "No items") {
			found = true
			break
		}
	}
	assert.True(t, found, "text must appear somewhere in the output")
}

// TestEmptyState_TextHorizontallyCentered verifies that the text line has
// padding on both sides (i.e. is horizontally centered within Width).
func TestEmptyState_TextHorizontallyCentered(t *testing.T) {
	th := theme.Load("black")
	text := "Empty"
	es := uikit.EmptyState{
		Text: text, Width: 40, Height: 5, Theme: th,
	}
	lines := uikit.Capture(es.Render())
	for _, l := range lines {
		if containsSubstr(l, text) {
			// The text must not be flush left (there should be leading spaces).
			assert.True(t, strings.HasPrefix(l, " "),
				"text line must have leading padding, got: %q", l)
			return
		}
	}
	t.Fatal("text line not found in output")
}

// TestEmptyState_MutedAnsiPresent verifies that the rendered output contains
// ANSI escape codes (from the Muted colour applied to Text/Hint).
func TestEmptyState_MutedAnsiPresent(t *testing.T) {
	th := theme.Load("black")
	es := uikit.EmptyState{
		Text: "No items", Hint: "hint", Width: 40, Height: 5, Theme: th,
	}
	raw := es.Render()
	assert.Contains(t, raw, "\x1b[",
		"Render() must contain ANSI escapes for the Muted colour")
}

// TestEmptyState_RoleText verifies Text uses RoleMuted colour token.
func TestEmptyState_RoleText(t *testing.T) {
	th := theme.Load("black")
	es := uikit.EmptyState{
		Text: "No items", Width: 40, Height: 5, Theme: th,
	}
	// The rendered output should use TextMuted colour — just verify non-empty ANSI.
	raw := es.Render()
	assert.Contains(t, raw, "\x1b[", "Text must have Muted role ANSI applied")
}

// TestEmptyState_ZeroHeight verifies that a zero Height does not panic and
// returns an empty string (0 lines joined by no newlines).
func TestEmptyState_ZeroHeight(t *testing.T) {
	th := theme.Load("black")
	es := uikit.EmptyState{
		Text: "No items", Width: 40, Height: 0, Theme: th,
	}
	// Must not panic.
	out := es.Render()
	assert.Equal(t, "", out, "zero Height must return empty string")
}

// TestEmptyState_TallBodyClampsPad verifies that when the body (text+hint)
// is taller than Height, the output is still exactly Height lines and does not
// panic. This exercises the break guard in the body-line loop.
func TestEmptyState_TallBodyClampsPad(t *testing.T) {
	th := theme.Load("black")
	// Height=1, body=2 lines (text + hint) → body overflows Height by 1.
	// The body-line loop must break after appending the first line.
	es := uikit.EmptyState{
		Text: "Line one", Hint: "Line two",
		Width: 30, Height: 1, Theme: th,
	}
	lines := uikit.Capture(es.Render())
	require.Len(t, lines, 1, "must return exactly 1 line even when body overflows Height")
}

// TestEmptyState_CenterLineNoOverflow verifies that centerLine does not add
// padding when the rendered text is at least as wide as the column width
// (cur >= w branch).
func TestEmptyState_CenterLineNoOverflow(t *testing.T) {
	th := theme.Load("black")
	// Use a very narrow Width (3) with a longer text so cur >= w.
	es := uikit.EmptyState{
		Text: "Long text here", Width: 3, Height: 3, Theme: th,
	}
	lines := uikit.Capture(es.Render())
	require.Len(t, lines, 3)
	// The text line should not have extra padding added (it's already wider).
	found := false
	for _, l := range lines {
		if containsSubstr(l, "Long text here") {
			found = true
			// Must not have leading spaces added by centerLine.
			assert.False(t, strings.HasPrefix(l, " "),
				"overflowing text must not have leading padding added: %q", l)
			break
		}
	}
	assert.True(t, found, "text must appear somewhere in output")
}

// TestEmptyState_AsciiMode verifies that EmptyState renders correctly in ASCII mode:
// the output has exactly Height lines and contains the text and hint (EmptyState has
// no unicode glyphs to swap, so this is primarily a regression guard).
func TestEmptyState_AsciiMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	es := uikit.EmptyState{
		Text: "No items", Hint: "Press / to search",
		Width: 40, Height: 6, Theme: th,
	}
	lines := uikit.Capture(es.Render())
	assert.Equal(t, 6, len(lines), "ascii mode must return exactly Height lines")
	full := strings.Join(lines, "\n")
	assert.Contains(t, full, "No items", "ascii mode must contain text")
	assert.Contains(t, full, "Press / to search", "ascii mode must contain hint")
}

// containsSubstr is a simple substring check used in tests.
func containsSubstr(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
