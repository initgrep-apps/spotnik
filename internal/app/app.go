// Package app contains the root Bubble Tea model that wires together all
// panes, the central store, and the active theme. It is the single entry
// point for the TUI application.
package app

import (
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// App is the root application model. It owns the active theme and will own
// all pane models once they are implemented in later features.
type App struct {
	// theme is loaded once at startup from config and injected into panes.
	// Panes never call theme.Load() themselves.
	theme theme.Theme
}

// New creates a new App, loading the theme from cfg.UI.Theme.
// An unknown or empty theme ID falls back to theme.DefaultThemeID without crashing.
func New(cfg *config.Config) *App {
	// NOTE: theme.Load() handles unknown IDs by falling back to the default.
	// Panes receive the theme at construction — they never call theme.Load() themselves.
	t := theme.Load(cfg.UI.Theme)
	return &App{
		theme: t,
	}
}

// Theme returns the active theme instance.
// This is used by pane constructors to receive the theme at startup.
func (a *App) Theme() theme.Theme {
	return a.theme
}
