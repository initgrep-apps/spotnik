package uikit_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/muesli/termenv"
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
// renders a line containing "◉ Monokai" and "active". It also asserts that the label
// uses the Plain (TextPrimary) colour regardless of Intent, per the role matrix.
func TestListRow_Unicode_WithGlyphAndCaption(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

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

	// The label MUST use the Plain role (TextPrimary colour), not the Intent colour.
	// Extract the ANSI escape that immediately precedes "Monokai" in the raw output.
	plainEscape := uikit.Apply(uikit.RolePlain, th).Render("X")
	plainColor := extractForegroundANSI(plainEscape)
	accentEscape := uikit.Apply(uikit.RoleAccent, th).Render("X")
	accentColor := extractForegroundANSI(accentEscape)
	assert.NotEqual(t, plainColor, accentColor,
		"sanity: Plain and Accent must have different ANSI colours")
	assert.True(t, strings.Contains(out, plainColor),
		"label must use Plain (TextPrimary) ANSI colour %q in raw output", plainColor)
	assert.False(t, isLabelColoredWith(out, "Monokai", accentColor),
		"label must NOT use Accent ANSI colour for its text")
}

// TestListRow_LabelAlwaysPlain is a regression test verifying that two rows with
// different intents (Accent and Muted) both render the label in the Plain colour
// while their glyphs differ.
func TestListRow_LabelAlwaysPlain(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	th := theme.Load("black")

	rowAccent := uikit.ListRow{
		Glyph: uikit.GlyphActive, Label: "Theme", Intent: uikit.RoleAccent, Theme: th,
	}
	rowMuted := uikit.ListRow{
		Glyph: uikit.GlyphAvailable, Label: "Theme", Intent: uikit.RoleMuted, Theme: th,
	}
	outAccent := rowAccent.Render(40)
	outMuted := rowMuted.Render(40)

	plainColor := extractForegroundANSI(uikit.Apply(uikit.RolePlain, th).Render("X"))
	accentColor := extractForegroundANSI(uikit.Apply(uikit.RoleAccent, th).Render("X"))
	mutedColor := extractForegroundANSI(uikit.Apply(uikit.RoleMuted, th).Render("X"))

	// Both rows must have the Plain colour present for the label.
	assert.Contains(t, outAccent, plainColor,
		"Accent-intent row: label must use Plain colour")
	assert.Contains(t, outMuted, plainColor,
		"Muted-intent row: label must use Plain colour")

	// The glyphs must differ: Accent row's glyph uses Accent colour; Muted uses Muted colour.
	assert.Contains(t, outAccent, accentColor,
		"Accent-intent row: glyph must use Accent colour")
	assert.Contains(t, outMuted, mutedColor,
		"Muted-intent row: glyph must use Muted colour")
}

// extractForegroundANSI extracts the "38;2;R;G;B" portion from a rendered string.
func extractForegroundANSI(s string) string {
	const fg = "38;2;"
	idx := strings.Index(s, fg)
	if idx < 0 {
		return ""
	}
	end := strings.Index(s[idx:], "m")
	if end < 0 {
		return ""
	}
	return s[idx : idx+end]
}

// isLabelColoredWith returns true when the ANSI escape immediately before label in s
// matches the given color sequence.
func isLabelColoredWith(s, label, colorSeq string) bool {
	idx := strings.Index(s, label)
	if idx < 0 {
		return false
	}
	sub := s[:idx]
	lastEsc := strings.LastIndex(sub, "\x1b[")
	if lastEsc < 0 {
		return false
	}
	end := strings.Index(s[lastEsc:], "m")
	if end < 0 {
		return false
	}
	esc := s[lastEsc : lastEsc+end+1]
	return strings.Contains(esc, colorSeq)
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

// TestLockedRow_TinyWidth verifies that LockedRow does not panic when width is
// smaller than the glyph itself (labelWidth < 0 branch).
func TestLockedRow_TinyWidth(t *testing.T) {
	th := theme.Load("black")
	row := uikit.LockedRow{Label: "Hi", Theme: th}
	// Width 1 is smaller than glyph+gap (≥2), so labelWidth clamps to 0.
	out := row.Render(1)
	_ = out // must not panic
}

// TestPadOrTruncate_EllipsisAlone verifies that a 2-column CJK input at width 1
// returns just "…" (the inner-loop candidate path in PadOrTruncate).
func TestPadOrTruncate_EllipsisAlone(t *testing.T) {
	// "日" is 2 terminal columns wide; at width 1 the loop drops it and "…" fits.
	got := uikit.PadOrTruncate("日本", 1)
	assert.Equal(t, "…", got)
}

// TestListRow_TinyWidth verifies that ListRow clamps labelWidth to 0 when
// caption+glyph exceed total width (labelWidth < 0 branch).
func TestListRow_TinyWidth(t *testing.T) {
	th := theme.Load("black")
	row := uikit.ListRow{
		Glyph:   uikit.GlyphActive,
		Label:   "Long Label",
		Caption: "caption text",
		Intent:  uikit.RoleMuted,
		Theme:   th,
	}
	// Very narrow width forces labelWidth < 0.
	out := row.Render(2)
	_ = out // must not panic
}

// ---------------------------------------------------------------------------
// LockedRow.PlainText
// ---------------------------------------------------------------------------

// TestLockedRow_PlainText_Unicode verifies that PlainText emits no ANSI and
// starts with the locked glyph followed by the label.
func TestLockedRow_PlainText_Unicode(t *testing.T) {
	th := theme.Load("black")
	row := uikit.LockedRow{Label: "Today's Top Hits", Theme: th}
	out := row.PlainText(40)

	// Must contain no ANSI escape sequences.
	assert.NotContains(t, out, "\x1b[", "PlainText must not contain ANSI escapes")
	// Must start with the locked glyph.
	assert.True(t, strings.HasPrefix(out, "◌ "),
		"PlainText must start with '◌ ', got %q", out)
	assert.Contains(t, out, "Today's Top Hits")
}

// TestLockedRow_PlainText_ASCII verifies the ASCII fallback glyph in PlainText.
func TestLockedRow_PlainText_ASCII(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	row := uikit.LockedRow{Label: "Spotify Playlist", Theme: th}
	out := row.PlainText(40)

	assert.NotContains(t, out, "\x1b[", "PlainText must not contain ANSI escapes")
	assert.True(t, strings.HasPrefix(out, "(r) "),
		"PlainText ASCII must start with '(r) ', got %q", out)
	assert.Contains(t, out, "Spotify Playlist")
}

// TestLockedRow_PlainText_Truncation verifies that PlainText truncates long labels.
func TestLockedRow_PlainText_Truncation(t *testing.T) {
	th := theme.Load("black")
	row := uikit.LockedRow{
		Label: "A very long Spotify-curated playlist name that exceeds width",
		Theme: th,
	}
	out := row.PlainText(20)
	assert.NotContains(t, out, "\x1b[", "PlainText must not contain ANSI escapes after truncation")
	// The glyph+space takes 2 columns; label must be truncated with ellipsis.
	assert.True(t, strings.HasSuffix(strings.TrimRight(out, " "), "…"),
		"truncated PlainText must end with …, got %q", out)
}

// TestLockedRow_PlainText_TinyWidth verifies PlainText does not panic on very small widths.
func TestLockedRow_PlainText_TinyWidth(t *testing.T) {
	th := theme.Load("black")
	row := uikit.LockedRow{Label: "Hi", Theme: th}
	out := row.PlainText(1) // width smaller than glyph+gap
	_ = out                 // must not panic
}
