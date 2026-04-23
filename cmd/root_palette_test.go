// Package cmd internal tests for palette resolution.
// These tests are in the cmd package (not cmd_test) because resolveCLIPalette
// is unexported.
package cmd

import (
	"bytes"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/cliout"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestResolveCLIPalette_fixedMode_usesFixed(t *testing.T) {
	cfg := config.Default()
	cfg.CLI.Palette = "fixed"
	var buf bytes.Buffer // not a TTY — buf is *bytes.Buffer, not *os.File
	resolveCLIPalette(cfg, &buf)
	assert.Equal(t, cliout.Fixed, cliout.CurrentForTest())
}

func TestResolveCLIPalette_themeMode_usesThemeTokens(t *testing.T) {
	cfg := config.Default()
	cfg.Preferences.Theme = "black"
	cfg.CLI.Palette = "theme"
	var buf bytes.Buffer
	resolveCLIPalette(cfg, &buf)
	th := theme.Load("black")
	assert.Equal(t, th.Accent(), cliout.CurrentForTest().Accent)
}

func TestResolveCLIPalette_autoMode_nonTTY_usesFixed(t *testing.T) {
	cfg := config.Default()
	cfg.CLI.Palette = "auto"
	var buf bytes.Buffer // non-TTY
	resolveCLIPalette(cfg, &buf)
	assert.Equal(t, cliout.Fixed, cliout.CurrentForTest())
}
