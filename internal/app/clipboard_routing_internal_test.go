package app

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeOSC52 strips the OSC 52 frame from out and returns the decoded payload.
// Fails the test if out is not a well-formed OSC 52 system-clipboard sequence.
func decodeOSC52(t *testing.T, out string) string {
	t.Helper()
	require.True(t, strings.HasPrefix(out, "\x1b]52;c;"),
		"missing OSC 52 prefix; got %q", out)
	require.True(t, strings.HasSuffix(out, "\x07"),
		"missing BEL terminator; got %q", out)
	body := strings.TrimSuffix(strings.TrimPrefix(out, "\x1b]52;c;"), "\x07")
	decoded, err := base64.StdEncoding.DecodeString(body)
	require.NoError(t, err, "OSC 52 payload must be valid base64")
	return string(decoded)
}

func TestHandleKeyMsg_stepOAuth_c_dispatchesCopyCmd(t *testing.T) {
	a := newTestApp(false)
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth
	a.onboardingAuthURL = "https://example.test/authorize"

	_, cmd := a.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd, "stepOAuth 'c' must dispatch copyToClipboardCmd")

	var msg tea.Msg
	out := captureStderr(t, func() { msg = cmd() })
	_, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok, "stepOAuth 'c' cmd must emit clipboardCopiedMsg; got %T", msg)

	// Decode the captured OSC 52 payload — proves the cmd encoded the right URL,
	// not just that some clipboard cmd was dispatched.
	assert.Equal(t, a.onboardingAuthURL, decodeOSC52(t, out))
}

func TestHandleOnboardingKey_stepRegister_c_emptyInput_dispatchesCopyCmd(t *testing.T) {
	a := newTestApp(false)
	a.currentView = viewOnboarding
	a.onboardingStep = stepRegister
	a.onboardingPort = 8888

	_, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd, "stepRegister 'c' on empty input must dispatch copyToClipboardCmd")

	var msg tea.Msg
	out := captureStderr(t, func() { msg = cmd() })
	_, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok, "stepRegister 'c' cmd must emit clipboardCopiedMsg; got %T", msg)

	// Decode the OSC 52 payload — confirms the redirect URI uses the configured
	// callback port. Catches a regression where the wrong field is wired in.
	expected := fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort)
	assert.Equal(t, expected, decodeOSC52(t, out))
}

func TestHandleOnboardingKey_stepRegister_c_typing_passesThroughToInput(t *testing.T) {
	a := newTestApp(false)
	a.currentView = viewOnboarding
	a.onboardingStep = stepRegister
	a.onboardingField.SetValue("ab") // user has started typing

	_, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	// 'c' must reach the FormField and append to the value.
	assert.Equal(t, "abc", a.onboardingField.Value(),
		"c should pass through to the input when typing has begun")

	// FormField may or may not return a cmd (textinput cursor blink etc.).
	// Regardless of whether it does, no clipboard cmd must have been dispatched
	// AND no OSC 52 sequence must have reached stderr.
	out := captureStderr(t, func() {
		if cmd != nil {
			msg := cmd()
			_, isCopy := msg.(clipboardCopiedMsg)
			assert.False(t, isCopy, "no clipboard cmd while typing")
		}
	})
	assert.NotContains(t, out, "\x1b]52;c;",
		"no OSC 52 sequence should be emitted while typing")
}

func TestHandleOnboardingKey_stepOAuth_c_dispatchesCopyCmd(t *testing.T) {
	a := newTestApp(false)
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth
	a.onboardingAuthURL = "https://accounts.spotify.com/authorize?x=1"

	_, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd, "stepOAuth 'c' must dispatch copyToClipboardCmd")

	var msg tea.Msg
	out := captureStderr(t, func() { msg = cmd() })
	_, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok, "stepOAuth 'c' cmd must emit clipboardCopiedMsg; got %T", msg)

	// Decode the OSC 52 payload — confirms the OAuth URL is the one wired in.
	assert.Equal(t, a.onboardingAuthURL, decodeOSC52(t, out))
}

// TestHandleOnboardingKey_stepError_c_noCopyAndNoPanic confirms that 'c' in
// stepError dispatches no command and does not panic.
func TestHandleOnboardingKey_stepError_c_noCopyAndNoPanic(t *testing.T) {
	a := newTestApp(false)
	a.currentView = viewOnboarding
	a.onboardingStep = stepError

	_, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	assert.Nil(t, cmd)
}

// TestHandleOnboardingKey_stepRegister_c_afterDeleteAll_dispatchesCopy confirms that
// 'c' still copies after the user typed and then deleted all input.
func TestHandleOnboardingKey_stepRegister_c_afterDeleteAll_dispatchesCopy(t *testing.T) {
	a := newTestApp(false)
	a.currentView = viewOnboarding
	a.onboardingStep = stepRegister
	a.onboardingPort = 9090
	a.onboardingField.SetValue("abc123")
	a.onboardingField.SetValue("")

	_, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd, "stepRegister 'c' on empty field (after delete-all) must dispatch copy")
	var msg tea.Msg
	captureStderr(t, func() { msg = cmd() })
	copied, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok)
	assert.NoError(t, copied.Err)
}
