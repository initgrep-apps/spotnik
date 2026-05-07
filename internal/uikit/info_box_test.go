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
// width wraps and is not truncated.
func TestInfoBox_WrapsLongBody(t *testing.T) {
	th := theme.Load("black")
	body := strings.Repeat("the quick brown fox ", 4) // ~80 chars
	box := uikit.InfoBox{
		Title: "Title",
		Body:  body,
		Width: 40,
		Theme: th,
	}.Render()

	require.GreaterOrEqual(t, strings.Count(box, "\n"), 2)
	assert.Contains(t, box, "fox")
}

// TestInfoBox_NarrowWidthGuard verifies that very small widths do not panic.
func TestInfoBox_NarrowWidthGuard(t *testing.T) {
	th := theme.Load("black")
	require.NotPanics(t, func() {
		_ = uikit.InfoBox{Title: "T", Body: "Hi", Width: 4, Theme: th}.Render()
	})
}
