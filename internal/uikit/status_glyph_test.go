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
