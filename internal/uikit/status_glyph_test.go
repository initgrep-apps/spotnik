package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

// TestStatusGlyph_WarningRendersCircleTriangle verifies that StatusGlyph with
// RoleWarning emits ◬ (U+25EC) and not U+26A0, per spec §5.2.
func TestStatusGlyph_WarningRendersCircleTriangle(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load(theme.DefaultThemeID)
	sg := uikit.StatusGlyph{
		Role:  uikit.RoleWarning,
		Text:  "Premium required",
		Theme: th,
	}
	out := sg.Render()
	assert.Contains(t, out, "◬", "warning StatusGlyph must contain ◬ (U+25EC)")
	assert.Contains(t, out, "Premium required", "warning StatusGlyph must contain text")
	assert.NotContains(t, out, "\u26A0", "warning StatusGlyph must NOT contain U+26A0")
}

// TestStatusGlyph_ASCII_Warning verifies that in ASCII mode RoleWarning emits "! <text>".
func TestStatusGlyph_ASCII_Warning(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load(theme.DefaultThemeID)
	sg := uikit.StatusGlyph{
		Role:  uikit.RoleWarning,
		Text:  "X",
		Theme: th,
	}
	out := sg.Render()
	stripped := stripANSI(out)
	assert.True(t, strings.HasPrefix(stripped, "! "),
		"ascii warning StatusGlyph must start with '! ', got: %q", stripped)
}

// TestStatusGlyph_AllRolesRender verifies all four roles produce non-empty output
// without panicking.
func TestStatusGlyph_AllRolesRender(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load(theme.DefaultThemeID)
	roles := []struct {
		role  uikit.Role
		glyph string
	}{
		{uikit.RoleSuccess, "✓"},
		{uikit.RoleError, "✗"},
		{uikit.RoleWarning, "◬"},
		{uikit.RoleInfo, "→"},
	}
	for _, tt := range roles {
		t.Run(string(tt.role), func(t *testing.T) {
			sg := uikit.StatusGlyph{Role: tt.role, Text: "msg", Theme: th}
			out := sg.Render()
			assert.NotEmpty(t, out, "Render() must not be empty for role %q", tt.role)
			assert.Contains(t, out, tt.glyph,
				"role %q must produce glyph %q", tt.role, tt.glyph)
			assert.Contains(t, out, "msg",
				"role %q must include text", tt.role)
		})
	}
}

// TestStatusGlyph_RenderFormat verifies output is "<glyph> <text>" (glyph space text).
func TestStatusGlyph_RenderFormat(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load(theme.DefaultThemeID)
	sg := uikit.StatusGlyph{Role: uikit.RoleSuccess, Text: "Done", Theme: th}
	out := stripANSI(sg.Render())
	// After stripping ANSI, the output should be exactly "✓ Done".
	assert.Equal(t, "✓ Done", out,
		"stripped Render() output should be '<glyph> <text>'")
}

// TestStatusGlyph_EmptyText renders without panicking when Text is empty.
func TestStatusGlyph_EmptyText(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load(theme.DefaultThemeID)
	sg := uikit.StatusGlyph{Role: uikit.RoleInfo, Text: "", Theme: th}
	assert.NotPanics(t, func() { _ = sg.Render() }, "empty text must not panic")
}

// TestStatusGlyph_UnknownRoleFallsBackToInfo verifies that an unrecognised Role
// falls back to the info glyph (→) rather than panicking or returning empty.
func TestStatusGlyph_UnknownRoleFallsBackToInfo(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load(theme.DefaultThemeID)
	sg := uikit.StatusGlyph{
		Role:  uikit.Role("unknown-role"),
		Text:  "fallback",
		Theme: th,
	}
	out := sg.Render()
	stripped := stripANSI(out)
	assert.True(t, strings.HasPrefix(stripped, "→"),
		"unknown role must fall back to info glyph →, got: %q", stripped)
	assert.Contains(t, out, "fallback", "unknown role must still include text")
}

// TestStatusGlyph_GapAddsExtraSpaces verifies that Gap > 0 inserts extra spaces
// between the glyph and the text, preserving alignment with adjacent two-space
// padded lines (e.g. "✓  text").
func TestStatusGlyph_GapAddsExtraSpaces(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load(theme.DefaultThemeID)

	tests := []struct {
		name      string
		gap       int
		wantSep   string // expected separator between glyph and text
		wantStart string // expected ANSI-stripped prefix
	}{
		{
			name:      "Gap=0 produces single space",
			gap:       0,
			wantSep:   " ",
			wantStart: "✓ Done",
		},
		{
			name:      "Gap=1 produces two spaces",
			gap:       1,
			wantSep:   "  ",
			wantStart: "✓  Done",
		},
		{
			name:      "Gap=2 produces three spaces",
			gap:       2,
			wantSep:   "   ",
			wantStart: "✓   Done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sg := uikit.StatusGlyph{
				Role:  uikit.RoleSuccess,
				Text:  "Done",
				Theme: th,
				Gap:   tt.gap,
			}
			out := stripANSI(sg.Render())
			assert.Equal(t, tt.wantStart, out,
				"ANSI-stripped output must be %q for Gap=%d", tt.wantStart, tt.gap)
			// Verify separator length independently by counting spaces between
			// the glyph character and the first letter of text.
			glyphEnd := strings.Index(out, "✓") + len("✓")
			textStart := strings.Index(out, "Done")
			sep := out[glyphEnd:textStart]
			assert.Equal(t, tt.wantSep, sep,
				"separator between glyph and text must be %q for Gap=%d", tt.wantSep, tt.gap)
		})
	}
}
