// Package cmd internal tests for palette resolution.
// These tests are in the cmd package (not cmd_test) because resolveCLIPaletteWith
// is unexported.
package cmd

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/cliout"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestResolveCLIPalette_fixedMode_usesFixed(t *testing.T) {
	prev := cliout.CurrentForTest()
	t.Cleanup(func() { cliout.Use(prev) })

	cfg := config.Default()
	cfg.CLI.Palette = "fixed"
	resolveCLIPaletteWith(cfg, false, false, func() bool { return false })
	assert.Equal(t, cliout.Fixed, cliout.CurrentForTest())
}

func TestResolveCLIPalette_themeMode_usesThemeTokens(t *testing.T) {
	prev := cliout.CurrentForTest()
	t.Cleanup(func() { cliout.Use(prev) })

	cfg := config.Default()
	cfg.Preferences.Theme = "black"
	cfg.CLI.Palette = "theme"
	resolveCLIPaletteWith(cfg, false, false, func() bool { return false })
	th := theme.Load("black")
	got := cliout.CurrentForTest()
	assert.Equal(t, th.Accent(), got.Accent)
	assert.Equal(t, th.TextMuted(), got.Muted)
}

func TestResolveCLIPalette_autoMode_nonTTY_usesFixed(t *testing.T) {
	prev := cliout.CurrentForTest()
	t.Cleanup(func() { cliout.Use(prev) })

	cfg := config.Default()
	cfg.CLI.Palette = "auto"
	resolveCLIPaletteWith(cfg, false, false, func() bool { return true })
	assert.Equal(t, cliout.Fixed, cliout.CurrentForTest())
}

func TestResolveCLIPalette_autoMode_ttyDark_usesTheme(t *testing.T) {
	prev := cliout.CurrentForTest()
	t.Cleanup(func() { cliout.Use(prev) })

	cfg := config.Default()
	cfg.Preferences.Theme = "black"
	cfg.CLI.Palette = "auto"
	resolveCLIPaletteWith(cfg, true, false, func() bool { return true })
	th := theme.Load("black")
	got := cliout.CurrentForTest()
	assert.Equal(t, th.Accent(), got.Accent, "auto+TTY+dark must use theme accent token")
	assert.Equal(t, th.TextMuted(), got.Muted, "auto+TTY+dark must use theme muted token")
}

func TestResolveCLIPalette_noColor_forcesFixed(t *testing.T) {
	prev := cliout.CurrentForTest()
	t.Cleanup(func() { cliout.Use(prev) })

	cfg := config.Default()
	cfg.Preferences.Theme = "black"
	cfg.CLI.Palette = "theme" // would normally use theme, but NO_COLOR overrides
	resolveCLIPaletteWith(cfg, false, true, func() bool { return true })
	assert.Equal(t, cliout.Fixed, cliout.CurrentForTest())
}
