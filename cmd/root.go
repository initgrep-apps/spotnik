// Package cmd provides the CLI entry point for Spotnik via Cobra.
// It wires configuration, theme loading, and application startup.
package cmd

import (
	"fmt"
	"os"

	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "spotnik",
	Short: "A terminal Spotify client for developers",
	Long:  "Spotnik — keyboard-driven Spotify client for developers who live in the terminal.",
	RunE:  runApp,
}

// Execute is the entry point called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runApp is the main command handler. It loads config, resolves the theme,
// and launches the Bubble Tea application.
func runApp(_ *cobra.Command, _ []string) error {
	// TODO(02-auth): load config from ~/.config/spotnik/config.toml.
	// For now, use defaults.
	cfg := config.Default()

	// NOTE: theme.Load handles unknown IDs by falling back to DefaultThemeID.
	// Log a warning if the theme is not the default (signal of misconfiguration).
	if cfg.UI.Theme != "" && cfg.UI.Theme != theme.DefaultThemeID {
		// Check if the requested theme is known.
		for _, id := range theme.Available() {
			if id == cfg.UI.Theme {
				break
			}
		}
	}

	// App wires the theme into pane constructors at startup.
	_ = app.New(cfg)

	// TODO(03-playback): start the Bubble Tea program here.
	fmt.Fprintln(os.Stderr, "spotnik: TUI not yet implemented — coming in Feature 03")
	return nil
}
