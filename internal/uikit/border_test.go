package uikit_test

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

func TestRoundedBorder_Unicode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	got := uikit.RoundedBorder()
	want := lipgloss.RoundedBorder()

	assert.Equal(t, want.Top, got.Top)
	assert.Equal(t, want.Bottom, got.Bottom)
	assert.Equal(t, want.Left, got.Left)
	assert.Equal(t, want.Right, got.Right)
	assert.Equal(t, want.TopLeft, got.TopLeft)
	assert.Equal(t, want.TopRight, got.TopRight)
	assert.Equal(t, want.BottomLeft, got.BottomLeft)
	assert.Equal(t, want.BottomRight, got.BottomRight)
}

func TestRoundedBorder_ASCII(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	got := uikit.RoundedBorder()

	assert.Equal(t, "-", got.Top)
	assert.Equal(t, "-", got.Bottom)
	assert.Equal(t, "|", got.Left)
	assert.Equal(t, "|", got.Right)
	assert.Equal(t, "+", got.TopLeft)
	assert.Equal(t, "+", got.TopRight)
	assert.Equal(t, "+", got.BottomLeft)
	assert.Equal(t, "+", got.BottomRight)

	// Negative: must not contain unicode rounded glyphs
	for _, g := range []string{got.Top, got.Bottom, got.Left, got.Right,
		got.TopLeft, got.TopRight, got.BottomLeft, got.BottomRight} {
		assert.NotContains(t, g, "╭")
		assert.NotContains(t, g, "╮")
		assert.NotContains(t, g, "╰")
		assert.NotContains(t, g, "╯")
		assert.NotContains(t, g, "─")
		assert.NotContains(t, g, "│")
	}
}
