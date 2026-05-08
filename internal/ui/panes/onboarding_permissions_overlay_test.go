package panes_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOnboardingPermissionsOverlay_ContainsExpectedSections asserts the overlay
// renders the four content blocks the design specifies.
func TestOnboardingPermissionsOverlay_ContainsExpectedSections(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewOnboardingPermissionsOverlay(th)
	o.SetSize(120, 40)

	view := o.View()

	assert.Contains(t, view, "Permissions Spotnik requests", "title must appear in border")
	assert.Contains(t, view, "Read access", "must contain Read access section heading")
	assert.Contains(t, view, "Write access", "must contain Write access section heading")
	assert.Contains(t, view, "spotify.com/account/apps", "must point users to Spotify for full details and revoke")
	assert.Contains(t, view, "Esc", "key bar must mention Esc")
}

// TestOnboardingPermissionsOverlay_EscEmitsCloseMsg asserts pressing Esc
// produces a close message.
func TestOnboardingPermissionsOverlay_EscEmitsCloseMsg(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewOnboardingPermissionsOverlay(th)

	updated, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.Same(t, o, updated.(*panes.OnboardingPermissionsOverlay))
	require.NotNil(t, cmd, "Esc must produce a close cmd")

	msg := cmd()
	_, ok := msg.(panes.OnboardingPermissionsOverlayClosedMsg)
	assert.True(t, ok, "Esc must emit OnboardingPermissionsOverlayClosedMsg, got %T", msg)
}

// TestOnboardingPermissionsOverlay_NonEscKeysAreConsumed asserts that any other
// key produces a nil cmd (modal — does not propagate).
func TestOnboardingPermissionsOverlay_NonEscKeysAreConsumed(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewOnboardingPermissionsOverlay(th)

	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyRunes, Runes: []rune{'v'}},
		{Type: tea.KeyEnter},
		{Type: tea.KeyTab},
	} {
		updated, cmd := o.Update(k)
		require.Same(t, o, updated.(*panes.OnboardingPermissionsOverlay))
		assert.Nil(t, cmd, "non-Esc key %v must be consumed (cmd nil)", k)
	}
}

// TestOnboardingPermissionsOverlay_NonKeyMessagesAreIgnored asserts that
// unrelated messages do not produce a cmd.
func TestOnboardingPermissionsOverlay_NonKeyMessagesAreIgnored(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewOnboardingPermissionsOverlay(th)

	updated, cmd := o.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	require.Same(t, o, updated.(*panes.OnboardingPermissionsOverlay))
	assert.Nil(t, cmd)
}

// TestOnboardingPermissionsOverlay_SetThemeUpdatesRender asserts that calling
// SetTheme is non-fatal and the title remains after switching.
func TestOnboardingPermissionsOverlay_SetThemeUpdatesRender(t *testing.T) {
	a := theme.Load("black")
	b := theme.Load("light")
	o := panes.NewOnboardingPermissionsOverlay(a)
	o.SetSize(120, 40)
	before := o.View()
	o.SetTheme(b)
	after := o.View()

	require.NotEmpty(t, before)
	require.NotEmpty(t, after)
	assert.Contains(t, after, "Permissions Spotnik requests")
	assert.NotEqual(t, before, after, "switching theme must change the rendered output")
}
