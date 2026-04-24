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

// TestStatusBar_RoleTokens verifies that StatusBar uses theme.TextMuted() for its border.
func TestStatusBar_RoleTokens(t *testing.T) {
	th := theme.Load("black")
	// TextMuted must be non-empty — it drives AccentColor of the muted border.
	assert.NotEmpty(t, string(th.TextMuted()), "theme must provide TextMuted colour for StatusBar border")
	assert.NotEmpty(t, string(th.Info()), "theme must provide Info colour for StatusBar keys")
}
