// Package app contains the root Bubble Tea model that wires together all
// panes, the central store, and the active theme. It is the single entry
// point for the TUI application.
package app

import (
	"context"
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

// focusedPane identifies which pane currently has keyboard focus.
type focusedPane int

const (
	focusPlayer  focusedPane = iota // default: player pane has focus
	focusLibrary                    // library pane has focus
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

	// library is the Spotify API client for library operations.
	// It is nil when no access token is available.
	library *api.LibraryClient

	// playerPane is the center pane (currently playing + controls).
	playerPane *panes.PlayerPane

	// libraryPane is the left pane (library browser).
	libraryPane *panes.LibraryPane

	// focus tracks which pane has keyboard focus.
	focus focusedPane

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

	playerPane := panes.NewPlayerPane(s, t, true)    // player pane is focused by default
	libraryPane := panes.NewLibraryPane(s, t, false) // library pane starts unfocused

	return &App{
		theme:       t,
		store:       s,
		playerPane:  playerPane,
		libraryPane: libraryPane,
		focus:       focusPlayer,
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

// SetLibrary injects the Spotify API library client into the app.
// Called by the root CLI after authentication succeeds.
func (a *App) SetLibrary(library *api.LibraryClient) {
	a.library = library
	a.libraryPane.SetLibrary(library)
}

// LibraryFocused returns true if the library pane currently has keyboard focus.
// Exported for integration tests.
func (a *App) LibraryFocused() bool {
	return a.focus == focusLibrary
}

// PlayerFocused returns true if the player pane currently has keyboard focus.
// Exported for integration tests.
func (a *App) PlayerFocused() bool {
	return a.focus == focusPlayer
}

// Init starts the polling loop and fetches initial playback state.
// Also initializes the library pane (fetches playlists + recently played).
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		panes.FetchPlaybackStateCmd(a.player, a.store),
		a.libraryPane.Init(),
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
		// Distribute width: library gets 30%, player gets 70%; reserve 2 rows for header/status.
		libraryWidth := m.Width * 30 / 100
		playerWidth := m.Width - libraryWidth
		a.libraryPane.SetSize(libraryWidth, m.Height-2)
		a.playerPane.SetSize(playerWidth, m.Height-2)
		return a, nil

	case tea.KeyMsg:
		// Tab rotates focus between library and player.
		if m.Type == tea.KeyTab {
			return a.rotateFocus()
		}
		// Route key events to the focused pane.
		switch a.focus {
		case focusLibrary:
			updated, cmd := a.libraryPane.Update(m)
			if lp, ok := updated.(*panes.LibraryPane); ok {
				a.libraryPane = lp
			}
			return a, cmd
		default:
			updatedPane, cmd := a.playerPane.Update(m)
			if pp, ok := updatedPane.(*panes.PlayerPane); ok {
				a.playerPane = pp
			}
			return a, cmd
		}

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

	case panes.PlayContextMsg:
		// Library pane selected a playlist or album — dispatch play command.
		return a, a.buildPlayContextCmd(m.ContextURI)

	case panes.PlayTrackMsg:
		// Library pane selected a specific track — dispatch play track command.
		return a, a.buildPlayTrackCmd(m.TrackURI)
	}

	// Forward unhandled messages to the library pane (e.g., library loaded msgs).
	updated, cmd := a.libraryPane.Update(msg)
	if lp, ok := updated.(*panes.LibraryPane); ok {
		a.libraryPane = lp
	}
	return a, cmd
}

// rotateFocus cycles keyboard focus between library and player panes.
func (a *App) rotateFocus() (*App, tea.Cmd) {
	switch a.focus {
	case focusPlayer:
		a.focus = focusLibrary
		a.playerPane.SetFocused(false)
		a.libraryPane.SetFocused(true)
	default:
		a.focus = focusPlayer
		a.libraryPane.SetFocused(false)
		a.playerPane.SetFocused(true)
	}
	return a, nil
}

// buildPlayContextCmd dispatches a play command for a playlist or album context URI.
func (a *App) buildPlayContextCmd(contextURI string) tea.Cmd {
	player := a.player
	return func() tea.Msg {
		if player == nil {
			return panes.PlaybackCmdSentMsg{}
		}
		err := player.Play(context.Background(), api.PlayOptions{ContextURI: contextURI})
		return panes.PlaybackCmdSentMsg{Err: err}
	}
}

// buildPlayTrackCmd dispatches a play command for a specific track URI.
func (a *App) buildPlayTrackCmd(trackURI string) tea.Cmd {
	player := a.player
	return func() tea.Msg {
		if player == nil {
			return panes.PlaybackCmdSentMsg{}
		}
		err := player.Play(context.Background(), api.PlayOptions{URIs: []string{trackURI}})
		return panes.PlaybackCmdSentMsg{Err: err}
	}
}

// View renders the full terminal UI as a three-pane layout:
// Library (left) | Player (center) | (Queue pane to be added in feature 06)
func (a *App) View() string {
	header := a.renderHeader()
	library := a.libraryPane.View()
	player := a.playerPane.View()
	statusBar := a.renderStatusBar()

	// Render library and player side-by-side.
	// Use a simple separator — the panes render themselves without borders for now.
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, library, "  ", player)

	return strings.Join([]string{header, mainContent, statusBar}, "\n")
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
