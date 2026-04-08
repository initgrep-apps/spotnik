package panes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHelpOverlay() *HelpOverlay {
	return NewHelpOverlay(theme.Load("black"))
}

func TestHelpOverlay_View_HasBorder(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	require.NotEmpty(t, view)
	assert.Contains(t, view, "╭")
	assert.Contains(t, view, "╰")
}

func TestHelpOverlay_View_HasTitle(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	assert.Contains(t, o.View(), "Help")
}

func TestHelpOverlay_View_ContainsSectionHeaders(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	for _, h := range []string{"Global", "Navigation", "Playback", "Pane Actions"} {
		assert.Contains(t, view, h, "section header %q should appear", h)
	}
}

func TestHelpOverlay_View_ContainsGlobalKeys(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	for _, k := range []string{"/", "d", "t", "?", "q", "0", "1-8", "p"} {
		assert.Contains(t, view, k, "global key %q should appear", k)
	}
}

func TestHelpOverlay_View_ContainsPlaybackKeys(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	for _, k := range []string{"Space", "n", "s", "r", "v"} {
		assert.Contains(t, view, k, "playback key %q should appear", k)
	}
}

func TestHelpOverlay_View_ContainsPaneActionKeys(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	for _, k := range []string{"Enter", "f", "A", "i", "x"} {
		assert.Contains(t, view, k, "pane action key %q should appear", k)
	}
}

func TestHelpOverlay_Update_EscEmitsClosedMsg(t *testing.T) {
	o := newTestHelpOverlay()
	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	_, ok := cmd().(HelpOverlayClosedMsg)
	assert.True(t, ok)
}

func TestHelpOverlay_Update_OtherKeysConsumed(t *testing.T) {
	o := newTestHelpOverlay()
	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'j'}},
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyEnter},
	} {
		_, cmd := o.Update(k)
		assert.Nil(t, cmd, "key %q should be consumed with nil cmd", k.String())
	}
}

func TestHelpOverlay_Update_NonKeyMsgIgnored(t *testing.T) {
	o := newTestHelpOverlay()
	_, cmd := o.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	assert.Nil(t, cmd)
}

func TestHelpOverlay_SetTheme(t *testing.T) {
	o := newTestHelpOverlay()
	assert.NotPanics(t, func() { o.SetTheme(theme.Load("monokai")) })
	assert.Equal(t, "monokai", o.theme.ID())
}

func TestHelpOverlay_SetSize(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(100, 30)
	assert.Equal(t, 100, o.width)
	assert.Equal(t, 30, o.height)
}

func TestHelpOverlay_View_NarrowTerminal(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(60, 30) // narrower than fixed 78-col width
	assert.NotPanics(t, func() { _ = o.View() })
}
