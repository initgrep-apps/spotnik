package components

import (
	"strings"

	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// InfoBox renders a bordered sub-pane with a title in the top border and
// vertically-centered content lines inside. Border glyphs are resolved via
// uikit.PaneChrome so the output honours ui.glyphs = "ascii" / "unicode".
type InfoBox struct {
	th     theme.Theme
	width  int
	height int
}

// NewInfoBox creates an InfoBox using the given theme.
func NewInfoBox(th theme.Theme) *InfoBox {
	return &InfoBox{th: th}
}

// SetSize updates the total width and height of the box (including the border
// rows and columns themselves).
func (b *InfoBox) SetSize(w, h int) {
	b.width = w
	b.height = h
}

// Render returns the InfoBox as a multi-line string.
//
// title is rendered in the top border: ╭─ Title ─────────────╮ (unicode) or
// +- Title -------------+ (ascii), depending on uikit.ActiveMode().
// lines contains the content to vertically-center inside the box.
// Border colour encodes focus state: focused → ActiveBorder(), unfocused →
// InactiveBorder(). PaneChrome is always passed Focused=true so it does not
// additionally apply a Faint dim over the chosen colour.
//
// Content behaviour:
//   - Each line is truncated (with "…") to the inner width (width-2).
//   - If len(lines) exceeds the inner height (height-2) the excess is
//     truncated from the bottom — the top lines (track name, artist) are
//     always shown first.
//   - Remaining vertical space is distributed as topPad above and bottom
//     padding below to centre the block.
func (b *InfoBox) Render(title string, lines []string, focused bool) string {
	w := b.width
	h := b.height

	// Enforce a sane minimum so we never index out-of-range.
	if w < 4 {
		w = 4
	}
	if h < 2 {
		h = 2
	}

	innerW := w - 2 // subtract left and right border columns
	innerH := h - 2 // subtract top and bottom border rows

	// -----------------------------------------------------------------------
	// Build vertically-centered interior content (no border chars).
	// -----------------------------------------------------------------------

	// Truncate content lines to available inner height.
	content := lines
	if len(content) > innerH {
		content = content[:innerH]
	}

	// Vertical centering: spread remaining rows above and below.
	remaining := innerH - len(content)
	topPad := 0
	if remaining > 0 {
		topPad = remaining / 2
	}
	bottomPad := remaining - topPad

	var sb strings.Builder

	// Top padding rows (blank lines).
	for i := 0; i < topPad; i++ {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(strings.Repeat(" ", innerW))
	}

	// Content rows — truncated/padded to innerW.
	for i, line := range content {
		if topPad > 0 || i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(layout.TruncateOrPad(line, innerW))
	}

	// Bottom padding rows (blank lines).
	for i := 0; i < bottomPad; i++ {
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat(" ", innerW))
	}

	interior := sb.String()

	// -----------------------------------------------------------------------
	// Delegate border rendering to uikit.PaneChrome so glyphs are
	// resolved via the active glyph mode (ascii / unicode).
	// Border colour encodes focus state directly; Focused is always true so
	// PaneChrome does not additionally Faint the already-chosen colour.
	// -----------------------------------------------------------------------
	border := b.th.ActiveBorder()
	if !focused {
		border = b.th.InactiveBorder()
	}
	chrome := uikit.PaneChrome{
		Width:       w,
		Height:      h,
		Title:       title,
		AccentColor: border,
		Focused:     true, // colour encodes focus; don't ALSO faint over it
		Theme:       b.th,
	}
	return chrome.Render(interior)
}
