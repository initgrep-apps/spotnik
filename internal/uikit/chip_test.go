package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

// TestChip_Unicode_ActiveDevice verifies that a unicode chip with GlyphActive renders
// the unicode glyph ◉ followed by the label on a StatusBarBg background.
func TestChip_Unicode_ActiveDevice(t *testing.T) {
	th := theme.Load("black")
	c := uikit.Chip{Glyph: uikit.GlyphActive, Label: "iPhone", Intent: uikit.RoleInfo, Theme: th}
	out := c.Render()
	plain := uikit.Capture(out)[0]
	assert.True(t, strings.Contains(plain, "◉ iPhone"),
		"active chip renders ◉ + label, got: %q", plain)
}

// TestChip_ASCII_Premium verifies that in ASCII mode, GlyphPremium renders as *P
// and the label follows.
func TestChip_ASCII_Premium(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	c := uikit.Chip{Glyph: uikit.GlyphPremium, Label: "Irshad", Intent: uikit.RoleSuccess, Theme: th}
	assert.Contains(t, uikit.Capture(c.Render())[0], "*P Irshad")
}

// TestChip_NoDevice_MutedGlyph verifies that GlyphAvailable + RoleMuted renders ○.
func TestChip_NoDevice_MutedGlyph(t *testing.T) {
	th := theme.Load("black")
	c := uikit.Chip{Glyph: uikit.GlyphAvailable, Label: "No device", Intent: uikit.RoleMuted, Theme: th}
	plain := uikit.Capture(c.Render())[0]
	assert.Contains(t, plain, "○ No device",
		"no-device chip renders ○ + label, got: %q", plain)
}

// TestChip_FreeProfile_FreeTierGlyph verifies that GlyphAvailable renders ○ for free profile.
func TestChip_FreeProfile_FreeTierGlyph(t *testing.T) {
	th := theme.Load("black")
	c := uikit.Chip{Glyph: uikit.GlyphAvailable, Label: "FreeUser", Intent: uikit.RoleMuted, Theme: th}
	plain := uikit.Capture(c.Render())[0]
	assert.Contains(t, plain, "○ FreeUser")
}

// TestChip_LeadingAndTrailingSpaces verifies the chip wraps content in padding spaces
// so that " <glyph> <label> " pattern holds.
func TestChip_LeadingAndTrailingSpaces(t *testing.T) {
	th := theme.Load("black")
	c := uikit.Chip{Glyph: uikit.GlyphActive, Label: "Dev", Intent: uikit.RoleInfo, Theme: th}
	plain := uikit.Capture(c.Render())[0]
	// Leading space before glyph, trailing space after label.
	assert.True(t, strings.HasPrefix(plain, " "), "chip must start with a leading space")
	assert.True(t, strings.HasSuffix(plain, " "), "chip must end with a trailing space")
}
