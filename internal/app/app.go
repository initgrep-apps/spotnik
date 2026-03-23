// Package app contains the root Bubble Tea model that wires together all
// panes, the central store, and the active theme. It is the single entry
// point for the TUI application.
package app

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// App is the root application model. It owns the active theme, the central
// store, the player API client, and all pane models.
type App struct {
	// theme is loaded once at startup from config and injected into panes.
	theme theme.Theme

	// store is the single source of truth for all application state.
	store *state.Store

	// player is the Spotify API client for playback control.
	// It is nil when no access token is available.
	player *api.Player

	// playerPane is the center pane (currently playing + controls).
	playerPane *panes.PlayerPane

	// width and height are set by WindowSizeMsg.
	width  int
	height int
}

// New creates a new App, loading the theme from cfg.UI.Theme.
// An unknown or empty theme ID falls back to theme.DefaultThemeID without crashing.
func New(cfg *config.Config) *App {
	// NOTE: theme.Load() handles unknown IDs by falling back to the default.
	t := theme.Load(cfg.UI.Theme)
	s := state.New()

	playerPane := panes.NewPlayerPane(s, t, true) // player pane is focused by default

	return &App{
		theme:      t,
		store:      s,
		playerPane: playerPane,
	}
}

// Theme returns the active theme instance.
// This is used by pane constructors to receive the theme at startup.
func (a *App) Theme() theme.Theme {
	return a.theme
}

// Store returns the central state store.
// Used by the root CLI command to set tokens and by tests to pre-populate state.
func (a *App) Store() *state.Store {
	return a.store
}

// SetPlayer injects the Spotify API player client into the app.
// Called by the root CLI after authentication succeeds.
func (a *App) SetPlayer(player *api.Player) {
	a.player = player
	a.playerPane.SetPlayer(player)
}

// Init starts the polling loop and fetches initial playback state.
// Returns a batch: first fetch + tick every 1s.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		panes.FetchPlaybackStateCmd(a.player, a.store),
		tea.Tick(time.Second, func(_ time.Time) tea.Msg {
			return panes.TickMsg{}
		}),
	)
}

// Update handles all messages routed through the root model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		// Give the player pane the full width; reserve 2 rows for header/status.
		a.playerPane.SetSize(m.Width, m.Height-2)
		return a, nil

	case tea.KeyMsg:
		// Route key events to the active pane.
		updatedPane, cmd := a.playerPane.Update(m)
		if pp, ok := updatedPane.(*panes.PlayerPane); ok {
			a.playerPane = pp
		}
		return a, cmd

	case panes.TickMsg:
		// On each tick: update local progress + reschedule + fetch playback state.
		updatedPane, _ := a.playerPane.Update(m)
		if pp, ok := updatedPane.(*panes.PlayerPane); ok {
			a.playerPane = pp
		}
		return a, tea.Batch(
			panes.FetchPlaybackStateCmd(a.player, a.store),
			tea.Tick(time.Second, func(_ time.Time) tea.Msg {
				return panes.TickMsg{}
			}),
		)

	case panes.PlaybackStateFetchedMsg:
		// Update the player pane with the new state.
		updatedPane, cmd := a.playerPane.Update(m)
		if pp, ok := updatedPane.(*panes.PlayerPane); ok {
			a.playerPane = pp
		}
		return a, cmd

	case panes.PlaybackCmdSentMsg:
		// After a playback command, trigger an immediate state refresh.
		if m.Err == nil {
			return a, panes.FetchPlaybackStateCmd(a.player, a.store)
		}
		return a, nil
	}

	return a, nil
}

// View renders the full terminal UI.
func (a *App) View() string {
	header := a.renderHeader()
	player := a.playerPane.View()
	statusBar := a.renderStatusBar()

	return strings.Join([]string{header, player, statusBar}, "\n")
}

// renderHeader renders the top bar with the app name and active device.
func (a *App) renderHeader() string {
	headerBg := lipgloss.NewStyle().
		Background(a.theme.SurfaceAlt()).
		Foreground(a.theme.TextPrimary()).
		Bold(true)

	deviceStyle := lipgloss.NewStyle().
		Foreground(a.theme.TextSecondary())

	appName := headerBg.Render(" spotnik ")
	deviceName := panes.DeviceName(a.store)

	if deviceName != "" {
		return appName + deviceStyle.Render(deviceName)
	}
	return appName
}

// renderStatusBar renders the bottom status bar with context-sensitive key hints.
func (a *App) renderStatusBar() string {
	style := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.StatusBarFg())

	keyStyle := lipgloss.NewStyle().
		Foreground(a.theme.KeyHint()).
		Bold(true)

	hints := []string{
		keyStyle.Render("Space") + " play/pause",
		keyStyle.Render("n/p") + " skip",
		keyStyle.Render("+/-") + " volume",
		keyStyle.Render("s") + " shuffle",
		keyStyle.Render("r") + " repeat",
		keyStyle.Render("q") + " quit",
	}

	return style.Render(strings.Join(hints, "  "))
}
