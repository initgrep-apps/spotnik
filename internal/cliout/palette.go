package cliout

import (
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/muesli/termenv"
)

// Palette holds the six role colours used by all message types.
type Palette struct {
	Accent  lipgloss.TerminalColor
	Success lipgloss.TerminalColor
	Error   lipgloss.TerminalColor
	Warning lipgloss.TerminalColor
	Muted   lipgloss.TerminalColor
	Plain   lipgloss.TerminalColor
}

// Fixed is the built-in palette — safe on any terminal, matches story 145 hex values.
var Fixed = Palette{
	Accent:  lipgloss.AdaptiveColor{Dark: "#1DB954", Light: "#1A8C41"},
	Success: lipgloss.AdaptiveColor{Dark: "#1DB954", Light: "#1A8C41"},
	Error:   lipgloss.AdaptiveColor{Dark: "#FF5555", Light: "#CC0000"},
	Warning: lipgloss.AdaptiveColor{Dark: "#F1C40F", Light: "#B8860B"},
	Muted:   lipgloss.AdaptiveColor{Dark: "#6C7083", Light: "#888888"},
	Plain:   lipgloss.AdaptiveColor{Dark: "", Light: ""}, // terminal default fg
}

// PaletteMode controls resolution strategy. Mirrors config.toml [cli] palette.
type PaletteMode int

const (
	// ModeAuto uses theme tokens on dark TTY, falls back to Fixed otherwise.
	ModeAuto PaletteMode = iota
	// ModeFixed always uses the built-in Fixed palette.
	ModeFixed
	// ModeTheme always uses the active TUI theme tokens.
	ModeTheme
)

// themePalette maps a theme.Theme onto a Palette using its semantic colour tokens.
func themePalette(t theme.Theme) Palette {
	return Palette{
		Accent:  t.Accent(),
		Success: t.Success(),
		Error:   t.Error(),
		Warning: t.Warning(),
		Muted:   t.TextMuted(),
		Plain:   t.TextPrimary(),
	}
}

// resolve picks the palette to render with given a mode, TTY check, NO_COLOR
// env flag, and (optionally) the active TUI theme. A nil theme means use Fixed.
func resolve(mode PaletteMode, tty bool, noColor bool, t theme.Theme) Palette {
	if noColor {
		return Fixed
	}
	switch mode {
	case ModeFixed:
		return Fixed
	case ModeTheme:
		if t == nil {
			return Fixed
		}
		return themePalette(t)
	case ModeAuto:
		if tty && termenv.HasDarkBackground() && t != nil {
			return themePalette(t)
		}
		return Fixed
	default:
		return Fixed
	}
}

// Global palette state — callers set once at CLI startup via Use().
var (
	activeMu      sync.RWMutex
	activePalette = Fixed
)

// Resolve picks a Palette given mode, TTY status, NO_COLOR flag, and an optional
// TUI theme. Exported so cmd/ can compose resolution without reimplementing it.
// A nil theme falls back to Fixed.
func Resolve(mode PaletteMode, tty bool, noColor bool, t theme.Theme) Palette {
	return resolve(mode, tty, noColor, t)
}

// Use replaces the active palette. Called by cmd/root.go once config is loaded.
// Safe to call more than once; safe to skip (default is Fixed).
func Use(p Palette) {
	activeMu.Lock()
	defer activeMu.Unlock()
	activePalette = p
}

// current returns the active palette under RLock.
func current() Palette {
	activeMu.RLock()
	defer activeMu.RUnlock()
	return activePalette
}
