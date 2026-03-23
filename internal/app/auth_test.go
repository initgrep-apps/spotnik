package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
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
