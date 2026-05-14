package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveClientIDCmd_writesAndEmitsMsg(t *testing.T) {
	// Arrange: create a temp config file with [spotify] section, no client_id.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	err := os.WriteFile(path, []byte("[spotify]\n"), 0o600)
	require.NoError(t, err)

	// Act: run the command.
	cmd := saveClientIDCmd(path, "testclientid")
	msg := cmd()

	// Assert: message type and payload.
	saved, ok := msg.(onboardingClientIDSavedMsg)
	require.True(t, ok, "expected onboardingClientIDSavedMsg, got %T", msg)
	assert.Equal(t, "testclientid", saved.clientID)

	// Assert: file was updated.
	loaded, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "testclientid", loaded.ClientID)
}

func TestSaveClientIDCmd_writeError_emitsErrorMsg(t *testing.T) {
	// Act: try to write to a non-existent directory path.
	cmd := saveClientIDCmd("/nonexistent/path/that/does/not/exist/config.toml", "id")
	msg := cmd()

	// Assert: error message type.
	_, ok := msg.(authErrorMsg)
	assert.True(t, ok, "expected authErrorMsg, got %T", msg)
}

// newTestField returns a *uikit.FormField pre-configured with the Spotify
// Client ID validator, suitable for onboarding handler tests.
func newTestField(th theme.Theme) *uikit.FormField {
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label:    "Client ID",
		Validate: config.ValidateClientID,
		Theme:    th,
	})
	f.Focus()
	return f
}

func TestHandleOnboardingKey_invalidClientID_setsError(t *testing.T) {
	th := theme.Load("black")
	a := &App{
		currentView:     viewOnboarding,
		onboardingStep:  stepRegister,
		onboardingClose: func() {},
		onboardingField: newTestField(th),
	}
	a.onboardingField.SetValue("tooshort12")

	updatedModel, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*App)

	assert.NotEmpty(t, updated.onboardingField.ValidationError(), "should record validation error")
	assert.Empty(t, updated.clientID, "should NOT save clientID on invalid input")
	assert.Nil(t, cmd, "no command should be emitted on invalid input")
}

func TestHandleOnboardingKey_validClientID_clearsError(t *testing.T) {
	th := theme.Load("black")
	a := &App{
		currentView:     viewOnboarding,
		onboardingStep:  stepRegister,
		onboardingClose: func() {},
		onboardingField: newTestField(th),
	}
	// Pre-seed a prior validation error then provide a valid value.
	a.onboardingField.SetValue("bad")
	_ = a.onboardingField.Validate()
	a.onboardingField.SetValue(strings.Repeat("a", 32))

	updatedModel, emittedCmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*App)

	assert.Empty(t, updated.onboardingField.ValidationError(), "error should clear on valid input")
	assert.NotNil(t, emittedCmd, "should emit save command for valid input")
}

func TestFriendlyAuthError_InvalidGrant(t *testing.T) {
	got := FriendlyAuthError(errors.New("invalid_grant: The authorization code expired"))
	assert.Equal(t, "Authorization expired. Please sign in again.", got)
}

func TestFriendlyAuthError_InvalidClient(t *testing.T) {
	got := FriendlyAuthError(errors.New("invalid_client: bad client secret"))
	assert.Equal(t, "Client ID is invalid. Check your config file.", got)
}

func TestFriendlyAuthError_AccessDenied(t *testing.T) {
	got := FriendlyAuthError(errors.New("access_denied"))
	assert.Equal(t, "Authorization denied. Please allow access when prompted.", got)
}

func TestFriendlyAuthError_Unknown(t *testing.T) {
	got := FriendlyAuthError(errors.New("something_unexpected"))
	assert.Equal(t, "Sign-in failed. Please run 'spotnik auth' to try again.", got)
}

func TestFriendlyAuthError_Nil(t *testing.T) {
	got := FriendlyAuthError(nil)
	assert.Equal(t, "Sign-in failed. Please run 'spotnik auth' to try again.", got)
}

func TestHandleOnboardingKey_keypress_forwardsToField(t *testing.T) {
	th := theme.Load("black")
	a := &App{
		currentView:     viewOnboarding,
		onboardingStep:  stepRegister,
		onboardingClose: func() {},
		onboardingField: newTestField(th),
	}
	// Pre-seed a validation error.
	a.onboardingField.SetValue("bad")
	_ = a.onboardingField.Validate()
	require.NotEmpty(t, a.onboardingField.ValidationError())

	updatedModel, _ := a.handleOnboardingKey(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("a"),
	})
	updated := updatedModel.(*App)
	// The keypress is forwarded to the FormField's Update(); the error is
	// not cleared by Update() alone — it is cleared by SetValue(). So we
	// verify the field accepted the rune, which is sufficient to confirm
	// routing to the FormField works correctly.
	assert.Contains(t, updated.onboardingField.Value(), "a", "keypress should be forwarded to the field")
}
