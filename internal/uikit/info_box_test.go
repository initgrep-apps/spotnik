package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInfoBox_RendersTitleAndBody verifies the box contains both the title
// and the body inside a rounded border.
func TestInfoBox_RendersTitleAndBody(t *testing.T) {
	th := theme.Load("black")
	box := uikit.InfoBox{
		Title: "About these permissions",
		Body:  "All Spotify access stays on this device.",
		Width: 60,
		Theme: th,
	}.Render()

	require.NotEmpty(t, box)
	assert.Contains(t, box, "About these permissions", "title must appear in box")
	assert.Contains(t, box, "All Spotify access stays on this device", "body must appear inside")
	assert.Contains(t, box, "╭", "must use rounded border in unicode mode")
}

// TestInfoBox_WrapsLongBody verifies that body text longer than the inner
// width wraps to additional lines (not truncated, and not collapsed onto the
// same physical row). Compares line counts of a short vs long body at the
// same Width to prove the long body actually expands the rendered height.
func TestInfoBox_WrapsLongBody(t *testing.T) {
	th := theme.Load("black")
	const width = 40

	short := uikit.InfoBox{
		Title: "Title",
		Body:  "fox",
		Width: width,
		Theme: th,
	}.Render()
	long := uikit.InfoBox{
		Title: "Title",
		Body:  strings.Repeat("the quick brown fox ", 4), // ~80 chars
		Width: width,
		Theme: th,
	}.Render()

	shortLines := strings.Count(short, "\n")
	longLines := strings.Count(long, "\n")
	assert.Greater(t, longLines, shortLines,
		"long body must wrap onto more rows than a short body at the same Width (short=%d long=%d)",
		shortLines, longLines)
	assert.Contains(t, long, "fox", "wrapped body must still contain its content")
}

// TestInfoBox_NarrowWidthGuard verifies that very small widths do not panic.
func TestInfoBox_NarrowWidthGuard(t *testing.T) {
	th := theme.Load("black")
	require.NotPanics(t, func() {
		_ = uikit.InfoBox{Title: "T", Body: "Hi", Width: 4, Theme: th}.Render()
	})
}
