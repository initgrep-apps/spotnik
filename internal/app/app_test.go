package app_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppNew_ReceivesTheme(t *testing.T) {
	cfg := &config.Config{}
	cfg.UI.Theme = "monokai"

	a := app.New(cfg)
	require.NotNil(t, a)
	assert.Equal(t, "monokai", a.Theme().ID())
}

func TestAppNew_DefaultThemeFallback(t *testing.T) {
	cfg := &config.Config{}
	cfg.UI.Theme = "invalid-theme-id"

	a := app.New(cfg)
	require.NotNil(t, a)
	// Unknown IDs fall back to DefaultThemeID without crashing.
	assert.Equal(t, theme.DefaultThemeID, a.Theme().ID())
}

func TestAppNew_EmptyThemeUsesDefault(t *testing.T) {
	cfg := &config.Config{}
	// cfg.UI.Theme is zero value (empty string)

	a := app.New(cfg)
	require.NotNil(t, a)
	assert.Equal(t, theme.DefaultThemeID, a.Theme().ID())
}
