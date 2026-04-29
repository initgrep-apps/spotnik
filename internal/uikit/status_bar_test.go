package uikit_test

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testKeyMap is a minimal help.KeyMap implementation for StatusBar tests.
type testKeyMap struct {
	bindings []key.Binding
}

func (m testKeyMap) ShortHelp() []key.Binding { return m.bindings }
func (m testKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.bindings}
}

// TestStatusBar_ThreeLines verifies that StatusBar.Render() produces exactly 3 lines
// (top border line + content line + bottom border line).
func TestStatusBar_ThreeLines(t *testing.T) {
	th := theme.Load("black")
	km := testKeyMap{bindings: []key.Binding{
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}}
	sb := uikit.StatusBar{Width: 160, Bindings: km, Theme: th}
	lines := uikit.Capture(sb.Render())
	require.Len(t, lines, 3, "StatusBar must render exactly 3 lines")
}

// TestStatusBar_ContainsBindingHints verifies that StatusBar renders binding key and desc.
func TestStatusBar_ContainsBindingHints(t *testing.T) {
	th := theme.Load("black")
	km := testKeyMap{bindings: []key.Binding{
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}}
	sb := uikit.StatusBar{Width: 160, Bindings: km, Theme: th}
	result := sb.Render()
	assert.Contains(t, result, "search", "status bar must contain binding description")
	assert.Contains(t, result, "quit", "status bar must contain quit description")
}

// TestStatusBar_MinWidthFallback verifies that Width < 160 is bumped to 160.
func TestStatusBar_MinWidthFallback(t *testing.T) {
	th := theme.Load("black")
	km := testKeyMap{bindings: []key.Binding{
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}}
	// Width 0 should still produce a valid 3-line output at effective width 160.
	sb := uikit.StatusBar{Width: 0, Bindings: km, Theme: th}
	lines := uikit.Capture(sb.Render())
	require.Len(t, lines, 3, "zero-width StatusBar must still render 3 lines via min-width fallback")

	// Check effective width is at least 160 columns.
	rendered := sb.Render()
	assert.GreaterOrEqual(t, lipgloss.Width(rendered), 160,
		"rendered width must be >= 160 when Width=0")
}

// TestStatusBar_ExplicitWidth verifies that a Width >= 160 is honoured.
func TestStatusBar_ExplicitWidth(t *testing.T) {
	th := theme.Load("black")
	km := testKeyMap{bindings: []key.Binding{
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}}
	sb := uikit.StatusBar{Width: 200, Bindings: km, Theme: th}
	lines := uikit.Capture(sb.Render())
	require.Len(t, lines, 3, "StatusBar at width 200 must render 3 lines")
}

// TestStatusBar_PageAwareBindings verifies that different KeyMaps produce different output.
func TestStatusBar_PageAwareBindings(t *testing.T) {
	th := theme.Load("black")

	kmA := testKeyMap{bindings: []key.Binding{
		key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "preset")),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}}
	kmB := testKeyMap{bindings: []key.Binding{
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}}

	resultA := uikit.StatusBar{Width: 160, Bindings: kmA, Theme: th}.Render()
	resultB := uikit.StatusBar{Width: 160, Bindings: kmB, Theme: th}.Render()

	assert.Contains(t, resultA, "preset", "Page A must show preset binding")
	assert.NotContains(t, resultB, "preset", "Page B must not show preset binding")
}

// TestStatusBar_AsciiBorder verifies that in ASCII mode the StatusBar border uses
// ASCII corner and rule characters (+, -, |) rather than unicode box-drawing chars.
func TestStatusBar_AsciiBorder(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	km := testKeyMap{bindings: []key.Binding{
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}}
	sb := uikit.StatusBar{Width: 160, Bindings: km, Theme: th}
	result := sb.Render()
	assert.NotContains(t, result, "╭", "ascii mode must not use unicode corner ╭")
	assert.NotContains(t, result, "╮", "ascii mode must not use unicode corner ╮")
	assert.NotContains(t, result, "╰", "ascii mode must not use unicode corner ╰")
	assert.NotContains(t, result, "╯", "ascii mode must not use unicode corner ╯")
	assert.Contains(t, result, "+", "ascii mode must use + for corners")
}

// TestStatusBar_RoleTokens verifies that StatusBar uses theme.KeyHint() for key labels
// and theme.TextMuted() for its border accent — NOT theme.Info(). Two themes where
// KeyHint and Info diverge are used to prove the correct token is applied.
func TestStatusBar_RoleTokens(t *testing.T) {
	km := testKeyMap{bindings: []key.Binding{
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}}

	tests := []struct {
		theme string
		// ANSI TrueColor escape substrings: "38;2;R;G;B"
		keyHintANSI string // must appear   in output (theme.KeyHint colour)
		infoANSI    string // must NOT appear in output (theme.Info colour — wrong token)
	}{
		{
			// dracula: KeyHint=#BD93F9 (189,147,249) vs Info=#8BE9FD (139,233,253)
			theme:       "dracula",
			keyHintANSI: "38;2;189;147;249",
			infoANSI:    "38;2;139;233;253",
		},
		{
			// gruvbox: KeyHint=#fe8019 (254,128,25) vs Info=#83A598 (131,165,152)
			theme:       "gruvbox",
			keyHintANSI: "38;2;254;128;25",
			infoANSI:    "38;2;131;165;152",
		},
	}

	for _, tt := range tests {
		t.Run(tt.theme, func(t *testing.T) {
			th := theme.Load(tt.theme)
			// Sanity: ensure KeyHint and Info actually differ for this theme.
			require.NotEqual(t, th.KeyHint(), th.Info(),
				"test precondition: KeyHint and Info must differ for %s", tt.theme)

			sb := uikit.StatusBar{Width: 160, Bindings: km, Theme: th}
			result := sb.Render()

			assert.Contains(t, result, tt.keyHintANSI,
				"StatusBar must use theme.KeyHint() colour for key labels in %s", tt.theme)
			assert.NotContains(t, result, tt.infoANSI,
				"StatusBar must NOT use theme.Info() colour for key labels in %s — use KeyHint() instead", tt.theme)
		})
	}
}
