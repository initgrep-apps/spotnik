package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlyph_AllRolesHaveBothForms(t *testing.T) {
	for _, role := range uikit.AllGlyphRoles() {
		u := uikit.GlyphFor(role, uikit.GlyphUnicode)
		a := uikit.GlyphFor(role, uikit.GlyphASCII)
		assert.NotEmpty(t, u, "role %q missing unicode", role)
		assert.NotEmpty(t, a, "role %q missing ascii fallback", role)
	}
}

func TestGlyph_UnicodeFormsAreAllSingleColumnExceptPlaybackMulti(t *testing.T) {
	// Banned: double-width glyphs in single-glyph roles that must align in tables.
	mustBeSingleCol := []uikit.GlyphRole{
		uikit.GlyphSuccess, uikit.GlyphError, uikit.GlyphWarning,
		uikit.GlyphInfo, uikit.GlyphRateLimit,
		uikit.GlyphActive, uikit.GlyphInactive, uikit.GlyphAvailable,
		uikit.GlyphLocked,
	}
	for _, role := range mustBeSingleCol {
		g := uikit.GlyphFor(role, uikit.GlyphUnicode)
		assert.Equal(t, 1, uikit.GlyphWidth(g),
			"role %q glyph %q must be 1-col wide", role, g)
	}
}

func TestGlyph_WarningIsCirclePlusInsideTriangle(t *testing.T) {
	// Confirms Section 5.2 of spec: U+25EC (◬), not U+26A0 (old warning sign).
	assert.Equal(t, "◬", uikit.GlyphFor(uikit.GlyphWarning, uikit.GlyphUnicode))
	assert.Equal(t, "!", uikit.GlyphFor(uikit.GlyphWarning, uikit.GlyphASCII))
}

func TestGlyph_ActionPrefixIsBanned(t *testing.T) {
	// Confirms Section 5.4: `ᐅ` (U+1405) is removed.
	for _, role := range uikit.AllGlyphRoles() {
		u := uikit.GlyphFor(role, uikit.GlyphUnicode)
		assert.False(t, strings.Contains(u, "ᐅ"),
			"role %q must not use banned glyph ᐅ", role)
	}
}

func TestGlyph_ASCIIModeHasNoBMPNonASCII(t *testing.T) {
	for _, role := range uikit.AllGlyphRoles() {
		a := uikit.GlyphFor(role, uikit.GlyphASCII)
		for _, r := range a {
			assert.Less(t, int(r), 128,
				"role %q ascii form %q contains non-ASCII rune %U",
				role, a, r)
		}
	}
}

func TestGlyph_CornerSharpAndDoubleAreBanned(t *testing.T) {
	// Confirms Section 5.1: only rounded corners ╭╮╰╯.
	u := uikit.GlyphFor(uikit.GlyphCornerTL, uikit.GlyphUnicode)
	require.Equal(t, "╭", u)
	u = uikit.GlyphFor(uikit.GlyphCornerTR, uikit.GlyphUnicode)
	require.Equal(t, "╮", u)
}

func TestGlyph_UnknownRoleReturnsEmpty(t *testing.T) {
	// GlyphFor returns "" for roles not in the catalogue, to surface wiring bugs.
	unknown := uikit.GlyphRole("does.not.exist")
	assert.Empty(t, uikit.GlyphFor(unknown, uikit.GlyphUnicode))
	assert.Empty(t, uikit.GlyphFor(unknown, uikit.GlyphASCII))
}

// TestGlyphFor_KeyboardChords verifies §4.9 keyboard-chord catalogue rows.
func TestGlyphFor_KeyboardChords(t *testing.T) {
	tests := []struct {
		role    uikit.GlyphRole
		unicode string
		ascii   string
	}{
		{uikit.GlyphEnter, "⏎", "Enter"},
		{uikit.GlyphEscape, "⎋", "Esc"},
		{uikit.GlyphTab, "⇥", "Tab"},
		{uikit.GlyphBackspace, "⌫", "BS"},
		{uikit.GlyphSpace, "␣", "Space"},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.unicode, uikit.GlyphFor(tt.role, uikit.GlyphUnicode),
				"unicode form for %q", tt.role)
			assert.Equal(t, tt.ascii, uikit.GlyphFor(tt.role, uikit.GlyphASCII),
				"ascii form for %q", tt.role)
		})
	}
}

// TestGlyphFor_Superscripts verifies §4.10 superscript catalogue rows.
func TestGlyphFor_Superscripts(t *testing.T) {
	tests := []struct {
		role    uikit.GlyphRole
		unicode string
		ascii   string
	}{
		{uikit.GlyphSuperscript0, "⁰", "0"},
		{uikit.GlyphSuperscript1, "¹", "1"},
		{uikit.GlyphSuperscript2, "²", "2"},
		{uikit.GlyphSuperscript3, "³", "3"},
		{uikit.GlyphSuperscript4, "⁴", "4"},
		{uikit.GlyphSuperscript5, "⁵", "5"},
		{uikit.GlyphSuperscript6, "⁶", "6"},
		{uikit.GlyphSuperscript7, "⁷", "7"},
		{uikit.GlyphSuperscript8, "⁸", "8"},
		{uikit.GlyphSuperscript9, "⁹", "9"},
		{uikit.GlyphSuperscriptPlus, "⁺", "+"},
		{uikit.GlyphSuperscriptMinus, "⁻", "-"},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.unicode, uikit.GlyphFor(tt.role, uikit.GlyphUnicode),
				"unicode form for %q", tt.role)
			assert.Equal(t, tt.ascii, uikit.GlyphFor(tt.role, uikit.GlyphASCII),
				"ascii form for %q", tt.role)
		})
	}
}

// TestGlyphFor_NewDomainRoles verifies the catalogue rows for GlyphPlaylist,
// GlyphSeparator, and the four device-type glyphs.
func TestGlyphFor_NewDomainRoles(t *testing.T) {
	tests := []struct {
		role    uikit.GlyphRole
		unicode string
		ascii   string
	}{
		{uikit.GlyphPlaylist, "▤", "[=]"},
		{uikit.GlyphSeparator, "·", "|"},
		{uikit.GlyphDeviceComputer, "⊡", "[c]"},
		{uikit.GlyphDevicePhone, "⊞", "[p]"},
		{uikit.GlyphDeviceSpeaker, "⊟", "[s]"},
		{uikit.GlyphDeviceTV, "⊠", "[tv]"},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.unicode, uikit.GlyphFor(tt.role, uikit.GlyphUnicode),
				"unicode form for %q", tt.role)
			assert.Equal(t, tt.ascii, uikit.GlyphFor(tt.role, uikit.GlyphASCII),
				"ascii form for %q", tt.role)
		})
	}
}

func TestGlyph_BannedGlyphsAbsentEverywhere(t *testing.T) {
	// U+26A0 is the old warning sign — banned in favour of U+25EC (◬).
	// Use the Go Unicode escape \u26A0 so the source file itself does not
	// contain the banned rune; `grep -rn "U+26A0" internal/` must return no matches.
	banned := []string{"\u26A0", "┌", "┐", "└", "┘", "╔", "╗", "╚", "╝", "ᐅ", "✅", "❌", "❗"}
	for _, role := range uikit.AllGlyphRoles() {
		u := uikit.GlyphFor(role, uikit.GlyphUnicode)
		for _, b := range banned {
			assert.False(t, strings.Contains(u, b),
				"role %q unicode form %q contains banned glyph %q", role, u, b)
		}
	}
}
