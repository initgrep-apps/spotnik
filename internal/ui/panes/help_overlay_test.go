package panes

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/muesli/termenv"
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
	for _, k := range []string{"/", "d", "t", "?", "q", "0", "1-8", "1-5", "p"} {
		assert.Contains(t, view, k, "global key %q should appear", k)
	}
	// Podcasts page removed — "1-4" no longer listed separately.
	assert.NotContains(t, view, "1-4", "global section must not contain 1-4 (Podcasts page removed)")
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
	// Enter, f, g remain after story 120 dead action removal (now title-case labels).
	for _, label := range []string{"Select / Play", "Filter", "Cycle time range"} {
		assert.Contains(t, view, label, "pane action label %q should appear", label)
	}
	// Removed dead action labels must not appear.
	for _, label := range []string{"add to queue", "like / unlike", "remove track", "reorder (playlists)"} {
		assert.NotContains(t, view, label, "removed pane action label %q must not appear", label)
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

// TestHelpOverlay_Labels_TitleCase asserts that every binding label in the help
// content starts with an uppercase letter. A label like "search" fails; "Search" passes.
func TestHelpOverlay_Labels_TitleCase(t *testing.T) {
	for _, col := range buildHelpContent("") {
		for _, sec := range col {
			for _, b := range sec.bindings {
				if len(b.label) == 0 {
					continue
				}
				first := rune(b.label[0])
				assert.True(t, first >= 'A' && first <= 'Z',
					"label %q must start with an uppercase letter", b.label)
			}
		}
	}
}

// TestHelpOverlay_Navigation_NoJK asserts that the Navigation section contains
// no binding whose key is "j / k" or "j/k".
func TestHelpOverlay_Navigation_NoJK(t *testing.T) {
	for _, col := range buildHelpContent("") {
		for _, sec := range col {
			if sec.title != "Navigation" {
				continue
			}
			for _, b := range sec.bindings {
				assert.NotEqual(t, "j / k", b.key,
					"Navigation must not list j / k (implicit scroll)")
				assert.NotEqual(t, "j/k", b.key,
					"Navigation must not list j/k (implicit scroll)")
			}
		}
	}
}

// TestHelpOverlay_ContainsIDetails asserts that the help overlay includes the
// 'i' keybinding for episode details in the Playback section when
// currentlyPlayingType is "episode".
func TestHelpOverlay_ContainsIDetails(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetCurrentlyPlayingType("episode")
	o.SetSize(120, 40)
	view := o.View()
	found := false
	for _, col := range buildHelpContent("episode") {
		for _, sec := range col {
			if sec.title == "Playback" {
				for _, b := range sec.bindings {
					if b.key == "i" {
						found = true
					}
				}
			}
		}
	}
	assert.True(t, found, "Playback section must contain 'i' keybinding for episode details")
	// Also check the rendered view contains the text.
	assert.Contains(t, view, "Episode details")
}

// TestHelpOverlay_NoIDetailsWhenTrack asserts that the 'i' keybinding is
// omitted when currentlyPlayingType is not "episode".
func TestHelpOverlay_NoIDetailsWhenTrack(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetCurrentlyPlayingType("track")
	o.SetSize(120, 40)
	view := o.View()
	for _, col := range buildHelpContent("track") {
		for _, sec := range col {
			if sec.title == "Playback" {
				for _, b := range sec.bindings {
					assert.NotEqual(t, "i", b.key, "Playback section must not contain 'i' when playing track")
				}
			}
		}
	}
	assert.NotContains(t, view, "Episode details", "rendered view must not contain 'Episode details' when playing track")
}

// TestHelpOverlay_ZeroKey_PlayerStats asserts the '0' key label is
// "Cycle Player / Stats" reflecting the 2-page model.
func TestHelpOverlay_ZeroKey_PlayerStats(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	assert.Contains(t, view, "Cycle Player / Stats", "0 key must show 'Cycle Player / Stats'")
}

// TestHelpOverlay_NoPodcastPageKeys asserts that the help overlay no longer
// references Podcasts page toggle keys.
func TestHelpOverlay_NoPodcastPageKeys(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	assert.NotContains(t, view, "1-4", "help overlay must not contain 1-4 (Podcasts page removed)")
	assert.NotContains(t, view, "Podcasts", "help overlay must not reference Podcasts page")
}

// TestHelpOverlay_AsciiBorder verifies that the help overlay border renders
// ASCII-safe characters when the uikit glyph mode is ASCII. Corner characters
// (╭╮╰╯), horizontal rules (─), and the inner column divider (│) must not
// appear. The divider is rendered via uikit.GlyphFor(uikit.GlyphVRule, mode)
// and therefore must emit "|" in ASCII mode.
func TestHelpOverlay_AsciiBorder(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	out := stripANSI(o.View())
	// Borders and the inner divider must all be ASCII in ASCII mode.
	if strings.ContainsAny(out, "╭╮╰╯─") {
		t.Errorf("ascii overlay border must not contain unicode corner/rule glyphs, got: %q", out)
	}
	// The inner column divider routes through GlyphFor(GlyphVRule) and must not
	// emit the unicode │ in ASCII mode.
	assert.NotContains(t, out, "│", "unicode vertical rule │ must not appear in ASCII mode")
	assert.Contains(t, out, "+", "ASCII mode should render '+' corners")
	// The ASCII vertical rule "|" must appear as the column divider.
	assert.Contains(t, out, "|", "ASCII vertical rule '|' must appear as column divider in ASCII mode")
}
