package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestRenderAuthView_ShowsAuthRequired(t *testing.T) {
	th := theme.Load("black")
	output := renderAuthPanel(th, 80, 24, "", "Waiting for authorization...")
	assert.Contains(t, output, "Authentication Required")
	assert.Contains(t, output, "Waiting for authorization")
}

func TestRenderAuthView_ShowsURL(t *testing.T) {
	th := theme.Load("black")
	url := "https://accounts.spotify.com/authorize?client_id=abc123"
	output := renderAuthPanel(th, 80, 24, url, "Opening browser...")
	assert.Contains(t, output, "accounts.spotify.com")
}

func TestRenderAuthView_TruncatesLongURL(t *testing.T) {
	th := theme.Load("black")
	longURL := "https://accounts.spotify.com/authorize?client_id=abc123&response_type=code&redirect_uri=http%3A%2F%2Flocalhost%3A12345%2Fcallback&scope=user-read-playback-state"
	output := renderAuthPanel(th, 80, 24, longURL, "")
	assert.NotEmpty(t, output)
}

func TestRenderAuthView_AuthError(t *testing.T) {
	th := theme.Load("black")
	output := renderAuthPanel(th, 80, 24, "", "Auth failed: token exchange error")
	assert.Contains(t, output, "Auth failed")
}
