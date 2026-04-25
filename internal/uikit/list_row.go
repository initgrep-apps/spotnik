package uikit

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// JoinSpace joins non-empty parts with a single space. Empty parts are skipped
// so that optional fields (e.g. absent glyphs or captions) do not produce
// double-spaces.
func JoinSpace(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}

// Pad returns a string of n ASCII spaces.
func Pad(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

// PadOrTruncate returns s padded with spaces to exactly width columns, or
// truncated with a trailing "…" if s exceeds width. Width is measured in
// terminal columns via lipgloss.Width.
func PadOrTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w == width {
		return s
	}
	if w < width {
		return s + Pad(width-w)
	}
	// Truncate: remove characters from the right until we fit, then append "…".
	runes := []rune(s)
	for len(runes) > 0 {
		candidate := string(runes) + "…"
		if lipgloss.Width(candidate) <= width {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}
	return "…"
}

// ListRow renders a single-line list item with an optional leading glyph, a
// label, and an optional trailing caption. The glyph colour is determined by
// the Intent role; the label colour uses Plain (TextPrimary); the caption uses Muted.
// The total output fits within width terminal columns.
type ListRow struct {
	// Glyph is optional. Zero value ("") means no glyph is rendered.
	Glyph GlyphRole
	// Label is the primary text shown in the row.
	Label string
	// Caption is optional secondary text shown at the right side of the row.
	Caption string
	// Intent controls the colour of the glyph only. The label always renders Plain.
	Intent Role
	// Theme provides colour tokens.
	Theme theme.Theme
}

// Render returns the styled, width-constrained string for this row.
func (r ListRow) Render(width int) string {
	mode := ActiveMode()
	th := r.Theme

	// Render glyph (empty string if no glyph role set).
	glyphStr := ""
	if r.Glyph != "" {
		raw := GlyphFor(r.Glyph, mode)
		glyphStr = Apply(r.Intent, th).Render(raw)
	}

	// Caption part — rendered in Muted.
	captionStr := ""
	if r.Caption != "" {
		captionStr = Apply(RoleMuted, th).Render(r.Caption)
	}

	// Reserve space for caption (plain width, no ANSI).
	captionWidth := lipgloss.Width(captionStr)
	glyphWidth := lipgloss.Width(glyphStr)

	// Space between glyph and label (only when glyph present).
	gapAfterGlyph := 0
	if glyphStr != "" {
		gapAfterGlyph = 1
	}

	// Space between label and caption (only when both present).
	gapBeforeCaption := 0
	if captionStr != "" {
		gapBeforeCaption = 1
	}

	// Available width for the label.
	labelWidth := width - glyphWidth - gapAfterGlyph - gapBeforeCaption - captionWidth
	if labelWidth < 0 {
		labelWidth = 0
	}

	labelStr := Apply(RolePlain, th).Render(PadOrTruncate(r.Label, labelWidth))

	// Assemble with plain spaces between segments.
	parts := []string{}
	if glyphStr != "" {
		parts = append(parts, glyphStr)
	}
	parts = append(parts, labelStr)
	if captionStr != "" {
		parts = append(parts, captionStr)
	}
	return strings.Join(parts, " ")
}

// LockedRow renders a read-only list item using the ◌ glyph in Muted colour.
// The entire row — glyph and label — is coloured with the Muted role to signal
// that the item cannot be interacted with.
type LockedRow struct {
	// Label is the display name for the locked item.
	Label string
	// Theme provides colour tokens.
	Theme theme.Theme
}

// Render returns the styled, width-constrained string for this locked row.
func (r LockedRow) Render(width int) string {
	mode := ActiveMode()
	th := r.Theme
	muted := Apply(RoleMuted, th)

	glyphStr := muted.Render(GlyphFor(GlyphLocked, mode))
	glyphWidth := lipgloss.Width(glyphStr)

	// 1-space gap between glyph and label.
	labelWidth := width - glyphWidth - 1
	if labelWidth < 0 {
		labelWidth = 0
	}

	labelStr := muted.Render(PadOrTruncate(r.Label, labelWidth))
	return glyphStr + " " + labelStr
}

// PlainText returns a plain-text (no ANSI) representation of the locked row:
// "<glyph> <label>" truncated to width terminal columns. It is intended for
// contexts where the surrounding renderer (e.g. bubble-table) applies its own
// colouring — embedding ANSI from Render would conflict with the per-column or
// per-cell foreground pass performed by the table renderer.
func (r LockedRow) PlainText(width int) string {
	mode := ActiveMode()
	glyph := GlyphFor(GlyphLocked, mode)
	glyphWidth := lipgloss.Width(glyph)

	// 1-space gap between glyph and label.
	labelWidth := width - glyphWidth - 1
	if labelWidth < 0 {
		labelWidth = 0
	}

	return glyph + " " + PadOrTruncate(r.Label, labelWidth)
}
