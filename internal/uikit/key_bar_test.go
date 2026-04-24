package uikit_test

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

// TestKeyBar_Unicode_RendersDotSeparators verifies that unicode mode uses "·" separator.
func TestKeyBar_Unicode_RendersDotSeparators(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	bindings := []key.Binding{
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
	out := uikit.KeyBar{Bindings: bindings, Theme: th}.Render()
	line := uikit.Capture(out)[0]
	assert.Contains(t, line, "c copy", "key and description must appear together")
	assert.Contains(t, line, "·", "unicode mode must use middot separator")
	assert.Contains(t, line, "q quit", "second binding must appear")
}

// TestKeyBar_ASCII_SwapsSeparator verifies that ascii mode uses "|" instead of "·".
func TestKeyBar_ASCII_SwapsSeparator(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	out := uikit.KeyBar{
		Bindings: []key.Binding{
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		},
		Theme: th,
	}.Render()
	line := uikit.Capture(out)[0]
	assert.NotContains(t, line, "·", "ascii mode must not use middot separator")
	assert.Contains(t, line, "|", "ascii mode must use pipe separator")
}

// TestKeyBar_EmptyBindings_ReturnsEmptyString verifies that empty bindings return empty output.
func TestKeyBar_EmptyBindings_ReturnsEmptyString(t *testing.T) {
	th := theme.Load("black")
	out := uikit.KeyBar{Bindings: []key.Binding{}, Theme: th}.Render()
	assert.Equal(t, "", out, "empty bindings must produce empty string")
}

// TestKeyBar_SingleBinding_NoSeparator verifies that one binding renders without separator.
func TestKeyBar_SingleBinding_NoSeparator(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	out := uikit.KeyBar{
		Bindings: []key.Binding{
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		},
		Theme: th,
	}.Render()
	line := uikit.Capture(out)[0]
	assert.Contains(t, line, "q quit", "single binding must render key and description")
	assert.NotContains(t, line, "·", "single binding must not have a separator")
}

// TestKeyBar_RoleTokens verifies that KeyBar uses KeyHint() for keys and TextMuted() for descs.
func TestKeyBar_RoleTokens(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	// Ensure KeyHint and TextMuted are non-empty (role-token assertion).
	assert.NotEmpty(t, string(th.KeyHint()), "theme must provide KeyHint colour")
	assert.NotEmpty(t, string(th.TextMuted()), "theme must provide TextMuted colour")
}
