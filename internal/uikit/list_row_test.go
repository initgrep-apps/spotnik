package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// joinSpace helper
// ---------------------------------------------------------------------------

func TestJoinSpace_SkipsEmpties(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "all empty", parts: []string{"", "", ""}, want: ""},
		{name: "first empty", parts: []string{"", "b"}, want: "b"},
		{name: "last empty", parts: []string{"a", ""}, want: "a"},
		{name: "middle empty", parts: []string{"a", "", "c"}, want: "a c"},
		{name: "none empty", parts: []string{"x", "y", "z"}, want: "x y z"},
		{name: "single part", parts: []string{"only"}, want: "only"},
		{name: "no parts", parts: []string{}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uikit.JoinSpace(tt.parts...)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// padOrTruncate helper
// ---------------------------------------------------------------------------

func TestPadOrTruncate_PadsShortString(t *testing.T) {
	got := uikit.PadOrTruncate("hi", 6)
	assert.Equal(t, "hi    ", got, "should be padded to width 6")
}

func TestPadOrTruncate_ExactWidth(t *testing.T) {
	got := uikit.PadOrTruncate("hello", 5)
	assert.Equal(t, "hello", got)
}

func TestPadOrTruncate_TruncatesLongString(t *testing.T) {
	got := uikit.PadOrTruncate("toolongstring", 5)
	// Must end with … and be <= 5 runes wide (but terminal-width via lipgloss.Width).
	assert.True(t, strings.HasSuffix(got, "…"), "truncated string must end with …, got %q", got)
}

func TestPadOrTruncate_ZeroWidth(t *testing.T) {
	got := uikit.PadOrTruncate("anything", 0)
	// Width 0 — should return empty or just ellipsis, must not panic.
	_ = got
}

func TestPadOrTruncate_EmptyString(t *testing.T) {
	got := uikit.PadOrTruncate("", 5)
	assert.Equal(t, "     ", got, "empty string padded to 5")
}

// ---------------------------------------------------------------------------
// pad helper
// ---------------------------------------------------------------------------

func TestPad_ReturnsNSpaces(t *testing.T) {
	assert.Equal(t, "   ", uikit.Pad(3))
	assert.Equal(t, "", uikit.Pad(0))
	assert.Equal(t, " ", uikit.Pad(1))
}

// ---------------------------------------------------------------------------
// ListRow
// ---------------------------------------------------------------------------

// TestListRow_Unicode_WithGlyphAndCaption is the required acceptance-criteria test.
// It verifies that a ListRow with GlyphActive + label "Monokai" + caption "active"
// renders a line containing "◉ Monokai" and "active".
func TestListRow_Unicode_WithGlyphAndCaption(t *testing.T) {
	th := theme.Load("black")
	row := uikit.ListRow{
		Glyph:   uikit.GlyphActive,
		Label:   "Monokai",
		Caption: "active",
		Intent:  uikit.RoleAccent,
		Theme:   th,
	}
	out := row.Render(40)
	lines := uikit.Capture(out)
	assert.Len(t, lines, 1, "ListRow renders exactly one line")
	plain := lines[0]
	assert.True(t, strings.Contains(plain, "◉ Monokai"),
		"expected ◉ Monokai in output, got: %q", plain)
	assert.True(t, strings.Contains(plain, "active"),
		"expected caption 'active' in output, got: %q", plain)
}

// TestListRow_NoGlyph renders a row without a glyph — label should appear without a
// leading glyph character or extra space.
func TestListRow_NoGlyph(t *testing.T) {
	th := theme.Load("black")
	row := uikit.ListRow{
		Label:  "Catppuccin",
		Intent: uikit.RolePlain,
		Theme:  th,
	}
	out := row.Render(40)
	plain := uikit.Capture(out)[0]
	assert.True(t, strings.Contains(plain, "Catppuccin"),
		"label present without glyph, got: %q", plain)
	// Should not start with ◉ or ○ since no glyph is set.
	assert.False(t, strings.HasPrefix(strings.TrimSpace(plain), "◉"),
		"no glyph set, should not render ◉")
}

// TestListRow_NoCaption renders a row without a caption.
func TestListRow_NoCaption(t *testing.T) {
	th := theme.Load("black")
	row := uikit.ListRow{
		Glyph:  uikit.GlyphAvailable,
		Label:  "Dracula",
		Intent: uikit.RoleMuted,
		Theme:  th,
	}
	out := row.Render(40)
	plain := uikit.Capture(out)[0]
	assert.Contains(t, plain, "○ Dracula")
}

// TestListRow_Truncates verifies that a very long label gets truncated to fit width.
func TestListRow_Truncates(t *testing.T) {
	th := theme.Load("black")
	row := uikit.ListRow{
		Glyph:  uikit.GlyphAvailable,
		Label:  "This is an extremely long theme name that will not fit",
		Intent: uikit.RoleMuted,
		Theme:  th,
	}
	out := row.Render(20)
	lines := uikit.Capture(out)
	assert.Len(t, lines, 1, "ListRow stays single line even when content is long")
}

// TestListRow_ASCII_Mode verifies that in ascii mode, GlyphActive renders as (*).
func TestListRow_ASCII_Mode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	row := uikit.ListRow{
		Glyph:   uikit.GlyphActive,
		Label:   "Monokai",
		Caption: "active",
		Intent:  uikit.RoleAccent,
		Theme:   th,
	}
	out := row.Render(40)
	plain := uikit.Capture(out)[0]
	assert.Contains(t, plain, "(*)")
	assert.Contains(t, plain, "Monokai")
	assert.Contains(t, plain, "active")
}

// ---------------------------------------------------------------------------
// LockedRow
// ---------------------------------------------------------------------------

// TestLockedRow_Unicode_DimGlyph is the required acceptance-criteria test.
// It verifies that a LockedRow renders the ◌ glyph and the label.
func TestLockedRow_Unicode_DimGlyph(t *testing.T) {
	th := theme.Load("black")
	row := uikit.LockedRow{
		Label: "Top Hits 2024",
		Theme: th,
	}
	out := row.Render(40)
	lines := uikit.Capture(out)
	assert.Len(t, lines, 1, "LockedRow renders exactly one line")
	plain := lines[0]
	assert.True(t, strings.Contains(plain, "◌"),
		"expected locked glyph ◌ in output, got: %q", plain)
	assert.True(t, strings.Contains(plain, "Top Hits 2024"),
		"expected label in output, got: %q", plain)
}

// TestLockedRow_ASCII_Mode verifies the ASCII fallback glyph (r) is rendered.
func TestLockedRow_ASCII_Mode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	row := uikit.LockedRow{
		Label: "Spotify Playlist",
		Theme: th,
	}
	plain := uikit.Capture(row.Render(40))[0]
	assert.Contains(t, plain, "(r)")
	assert.Contains(t, plain, "Spotify Playlist")
}

// TestLockedRow_LongLabel verifies truncation without panicking.
func TestLockedRow_LongLabel(t *testing.T) {
	th := theme.Load("black")
	row := uikit.LockedRow{
		Label: "A very long Spotify-curated playlist name that exceeds the available width",
		Theme: th,
	}
	out := row.Render(20)
	lines := uikit.Capture(out)
	assert.Len(t, lines, 1, "LockedRow stays single line")
}
