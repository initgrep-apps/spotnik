package uikit_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSectionLabel_TwoLines verifies that Render returns exactly two plain-text
// lines: the padded label on line 1 and a horizontal rule on line 2.
func TestSectionLabel_TwoLines(t *testing.T) {
	th := theme.Load("black")
	sl := uikit.SectionLabel{
		Label:       "GATEWAY",
		Width:       20,
		AccentColor: th.Accent(),
		Theme:       th,
	}
	out := sl.Render()
	lines := uikit.Capture(out)
	require.Len(t, lines, 2, "SectionLabel must render exactly two lines")
}

// TestSectionLabel_FirstLineContainsLabel verifies that the first plain-text
// line contains the label surrounded by spaces: " GATEWAY ".
func TestSectionLabel_FirstLineContainsLabel(t *testing.T) {
	th := theme.Load("black")
	sl := uikit.SectionLabel{
		Label:       "GATEWAY",
		Width:       20,
		AccentColor: th.Accent(),
		Theme:       th,
	}
	out := sl.Render()
	lines := uikit.Capture(out)
	require.GreaterOrEqual(t, len(lines), 1)
	assert.True(t,
		strings.Contains(lines[0], " GATEWAY "),
		"first line must contain ' GATEWAY ', got: %q", lines[0])
}

// TestSectionLabel_SecondLineIsHRule verifies that the second plain-text line
// consists entirely of the unicode horizontal rule char ─ in unicode mode.
func TestSectionLabel_SecondLineIsHRule(t *testing.T) {
	th := theme.Load("black")
	sl := uikit.SectionLabel{
		Label:       "APP",
		Width:       15,
		AccentColor: th.Accent(),
		Theme:       th,
	}
	out := sl.Render()
	lines := uikit.Capture(out)
	require.Len(t, lines, 2)
	rule := lines[1]
	// Rule must not be empty.
	assert.NotEmpty(t, rule, "second line (rule) must not be empty")
	// Every character in the rule must be ─.
	for _, r := range rule {
		assert.Equal(t, '─', r,
			"rule line must contain only ─ chars, found %q in %q", string(r), rule)
	}
}

// TestSectionLabel_RuleWidthMatchesWidth verifies that the rule line is exactly
// Width rune-columns wide.
func TestSectionLabel_RuleWidthMatchesWidth(t *testing.T) {
	th := theme.Load("black")
	const w = 30
	sl := uikit.SectionLabel{
		Label:       "GATEWAY LOG",
		Width:       w,
		AccentColor: th.Accent(),
		Theme:       th,
	}
	out := sl.Render()
	lines := uikit.Capture(out)
	require.Len(t, lines, 2)
	ruleWidth := lipgloss.Width(lines[1])
	assert.Equal(t, w, ruleWidth, "rule must be exactly Width=%d chars wide, got %d", w, ruleWidth)
}

// TestSectionLabel_ASCII_Rule verifies that in ascii mode the rule line uses
// '-' instead of '─'.
func TestSectionLabel_ASCII_Rule(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	sl := uikit.SectionLabel{
		Label:       "SPOTIFY",
		Width:       12,
		AccentColor: th.Accent(),
		Theme:       th,
	}
	out := sl.Render()
	lines := uikit.Capture(out)
	require.Len(t, lines, 2)
	rule := lines[1]
	assert.NotEmpty(t, rule)
	// In ASCII mode every char in the rule must be '-'.
	for _, r := range rule {
		assert.Equal(t, '-', r,
			"ASCII rule must contain only '-' chars, found %q in %q", string(r), rule)
	}
}

// TestSectionLabel_AccentColorOnLabel verifies that the raw output contains ANSI
// colour escapes (accent colour) applied to the label text.
func TestSectionLabel_AccentColorOnLabel(t *testing.T) {
	th := theme.Load("black")
	sl := uikit.SectionLabel{
		Label:       "SPOTIFY",
		Width:       20,
		AccentColor: th.Accent(),
		Theme:       th,
	}
	raw := sl.Render()
	// The raw output must contain at least one ANSI colour escape.
	assert.Contains(t, raw, "\x1b[",
		"output must contain ANSI escapes for accent colour")
	// The first plain-captured line must contain the label.
	plain := uikit.Capture(raw)
	assert.True(t, strings.Contains(plain[0], "SPOTIFY"),
		"plain first line must contain the label")
}

// TestSectionLabel_ZeroWidth verifies that a zero Width does not panic.
func TestSectionLabel_ZeroWidth(t *testing.T) {
	th := theme.Load("black")
	sl := uikit.SectionLabel{
		Label:       "TEST",
		Width:       0,
		AccentColor: th.Accent(),
		Theme:       th,
	}
	out := sl.Render()
	_ = out // must not panic
}

// TestSectionLabel_EmptyLabel verifies that an empty label still renders two
// lines without panicking.
func TestSectionLabel_EmptyLabel(t *testing.T) {
	th := theme.Load("black")
	sl := uikit.SectionLabel{
		Label:       "",
		Width:       10,
		AccentColor: th.Accent(),
		Theme:       th,
	}
	out := sl.Render()
	lines := uikit.Capture(out)
	assert.Len(t, lines, 2, "empty label must still yield two lines")
}
