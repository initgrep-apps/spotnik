package components

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

// infoboxTestTheme returns the black theme for InfoBox tests.
// NOTE: kept separate from table_test.go's testTheme() which lives in
// package components_test; this file is in package components (internal).
func infoboxTestTheme() theme.Theme {
	return theme.Load("black")
}

// newTestInfoBox returns an InfoBox with the black theme for testing.
func newTestInfoBox(w, h int) *InfoBox {
	ib := NewInfoBox(infoboxTestTheme())
	ib.SetSize(w, h)
	return ib
}

// countLeadingBlankLines returns the number of blank interior lines (lines
// containing only a border character and spaces) before the first non-blank
// content line inside the box.
func countLeadingBlankLinesInsideBox(rendered string) int {
	lines := strings.Split(rendered, "\n")
	// Skip top border (first line) and bottom border (last line).
	// Each interior line has the form "│ ... │".
	interior := lines[1 : len(lines)-1]

	count := 0
	for _, line := range interior {
		// Strip border characters and spaces to get content.
		inner := strings.TrimPrefix(line, "│")
		inner = strings.TrimSuffix(inner, "│")
		inner = strings.TrimSpace(inner)
		if inner == "" {
			count++
		} else {
			break
		}
	}
	return count
}

// ---------------------------------------------------------------------------
// Test: rounded borders and title in output
// ---------------------------------------------------------------------------

func TestInfoBox_Render_RoundedBordersAndTitle(t *testing.T) {
	ib := newTestInfoBox(28, 10)
	out := ib.Render("Track Info", []string{"Song Name", "Artist"}, false)

	assert.Contains(t, out, "╭", "top-left rounded corner must be present")
	assert.Contains(t, out, "╰", "bottom-left rounded corner must be present")
	assert.Contains(t, out, "╮", "top-right rounded corner must be present")
	assert.Contains(t, out, "╯", "bottom-right rounded corner must be present")
	assert.Contains(t, out, "Track Info", "title must appear in the border")
}

// ---------------------------------------------------------------------------
// Test: vertical centering
// ---------------------------------------------------------------------------

func TestInfoBox_Render_ContentVerticallycentered(t *testing.T) {
	// width=28, height=10 → 8 inner rows
	// 5 content lines → topPad = (8 - 5) / 2 = 1
	ib := newTestInfoBox(28, 10)
	lines := []string{"Song Name", "Artist", "", "|< || >|", "VOL 65%"}
	out := ib.Render("Track Info", lines, false)

	leading := countLeadingBlankLinesInsideBox(out)
	assert.Equal(t, 1, leading, "leading blank lines inside box should equal topPad = (8-5)/2 = 1")
}

func TestInfoBox_Render_CenteringEvenSpread(t *testing.T) {
	// width=30, height=12 → 10 inner rows
	// 4 content lines → topPad = (10 - 4) / 2 = 3
	ib := newTestInfoBox(30, 12)
	lines := []string{"Alpha", "Beta", "Gamma", "Delta"}
	out := ib.Render("Box", lines, false)

	leading := countLeadingBlankLinesInsideBox(out)
	assert.Equal(t, 3, leading, "topPad should be (10-4)/2 = 3")
}

// ---------------------------------------------------------------------------
// Test: content taller than box is truncated from bottom
// ---------------------------------------------------------------------------

func TestInfoBox_Render_TruncateTallContent(t *testing.T) {
	// width=30, height=6 → 4 inner rows
	// 8 content lines → only first 4 should be shown
	ib := newTestInfoBox(30, 6)
	lines := []string{"Line1", "Line2", "Line3", "Line4", "Line5", "Line6", "Line7", "Line8"}
	out := ib.Render("Box", lines, false)

	// First 4 lines must appear.
	for _, expected := range lines[:4] {
		assert.Contains(t, out, expected, "line %q should appear when content fits", expected)
	}
	// Lines 5-8 must NOT appear (truncated from bottom).
	for _, notExpected := range lines[4:] {
		assert.NotContains(t, out, notExpected, "line %q should be truncated", notExpected)
	}
}

// ---------------------------------------------------------------------------
// Test: each content line truncated to inner width
// ---------------------------------------------------------------------------

func TestInfoBox_Render_LongLinesTruncated(t *testing.T) {
	// Inner width = width - 2 = 8
	ib := newTestInfoBox(10, 8)
	// 20 chars — should be truncated to fit inner width of 8.
	lines := []string{"ABCDEFGHIJKLMNOPQRST"}
	out := ib.Render("T", lines, false)

	// Full string must NOT appear.
	assert.NotContains(t, out, "ABCDEFGHIJKLMNOPQRST", "long line must be truncated")
	// A truncation indicator (ellipsis) must appear.
	assert.Contains(t, out, "…", "truncated line must end with ellipsis")
}

// ---------------------------------------------------------------------------
// Test: focused vs unfocused renders different border chars (structure check)
// ---------------------------------------------------------------------------

func TestInfoBox_Render_FocusedAndUnfocusedBothRenderBorders(t *testing.T) {
	ib := newTestInfoBox(28, 10)
	lines := []string{"Hello"}

	outFocused := ib.Render("Box", lines, true)
	outUnfocused := ib.Render("Box", lines, false)

	// Both must render rounded borders — the difference is ANSI color codes which
	// are environment-dependent; just verify structure is present in both cases.
	assert.Contains(t, outFocused, "╭")
	assert.Contains(t, outFocused, "╰")
	assert.Contains(t, outUnfocused, "╭")
	assert.Contains(t, outUnfocused, "╰")
	assert.Contains(t, outFocused, "Box")
	assert.Contains(t, outUnfocused, "Box")
}

// ---------------------------------------------------------------------------
// Test: empty content lines render blank interior
// ---------------------------------------------------------------------------

func TestInfoBox_Render_EmptyLines(t *testing.T) {
	ib := newTestInfoBox(20, 6)
	out := ib.Render("Empty", nil, false)

	assert.Contains(t, out, "╭")
	assert.Contains(t, out, "╰")
	assert.Contains(t, out, "Empty")
}

// ---------------------------------------------------------------------------
// Test: minimum dimensions don't panic
// ---------------------------------------------------------------------------

func TestInfoBox_Render_MinimumDimensions(t *testing.T) {
	ib := newTestInfoBox(4, 4)
	assert.NotPanics(t, func() {
		_ = ib.Render("X", []string{"very long string that should be truncated hard"}, false)
	})
}
