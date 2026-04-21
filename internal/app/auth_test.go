package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderAuthPanel_ContainsTitle(t *testing.T) {
	th := theme.Load("black")
	view := renderAuthPanel(th, 120, 40, "https://example.com/auth", "Waiting for authorization...")
	assert.Contains(t, view, "Authentication Required")
}

func TestRenderAuthPanel_ContainsURL(t *testing.T) {
	th := theme.Load("black")
	view := renderAuthPanel(th, 120, 40, "https://short.url", "Waiting...")
	assert.Contains(t, view, "https://short.url")
}

func TestRenderAuthPanel_TruncatesLongURL(t *testing.T) {
	th := theme.Load("black")
	longURL := "https://accounts.spotify.com/authorize?client_id=abc123&response_type=code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback"
	view := renderAuthPanel(th, 120, 40, longURL, "Waiting...")
	assert.Contains(t, view, "...")
	assert.NotContains(t, view, longURL, "full long URL should be truncated")
}

func TestRenderAuthPanel_NoSize(t *testing.T) {
	th := theme.Load("black")
	view := renderAuthPanel(th, 0, 0, "https://example.com", "Status text")
	assert.Contains(t, view, "Authentication Required")
	assert.Contains(t, view, "Status text")
}

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
