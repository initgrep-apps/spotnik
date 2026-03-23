// Package app contains the root Bubble Tea model that wires together all
// panes, the central store, and the active theme. It is the single entry
// point for the TUI application.
package app

import (
	"context"
	"fmt"
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
// store, the API clients, and all pane models. It is the ONLY layer that
// calls the Spotify API — panes emit request messages and app.go dispatches them.
type App struct {
	theme   theme.Theme
	store   *state.Store
	player  *api.Player
	library *api.LibraryClient

	playerPane  *panes.PlayerPane
	libraryPane *panes.LibraryPane

	focus  focusedPane
	width  int
	height int

	// statusMsg is a transient error/info message shown in the status bar for 4 seconds.
	statusMsg string
}

// statusDismissMsg is sent after 4 seconds to clear a transient status bar message.
type statusDismissMsg struct{}

// New creates a new App, loading the theme from cfg.UI.Theme.
func New(cfg *config.Config) *App {
	t := theme.Load(cfg.UI.Theme)
	s := state.New()

	playerPane := panes.NewPlayerPane(s, t, true)
	libraryPane := panes.NewLibraryPane(s, t, false)

	return &App{
		theme:       t,
		store:       s,
		playerPane:  playerPane,
		libraryPane: libraryPane,
		focus:       focusPlayer,
	}
}

// Theme returns the active theme instance.
func (a *App) Theme() theme.Theme {
	return a.theme
}

// Store returns the central state store.
func (a *App) Store() *state.Store {
	return a.store
}

// SetPlayer injects the Spotify API player client into the app.
func (a *App) SetPlayer(player *api.Player) {
	a.player = player
}

// SetLibrary injects the Spotify API library client into the app.
func (a *App) SetLibrary(library *api.LibraryClient) {
	a.library = library
}

// LibraryFocused returns true if the library pane currently has keyboard focus.
func (a *App) LibraryFocused() bool {
	return a.focus == focusLibrary
}

// PlayerFocused returns true if the player pane currently has keyboard focus.
func (a *App) PlayerFocused() bool {
	return a.focus == focusPlayer
}

// Init starts the polling loop and fetches initial playback state.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		fetchPlaybackStateCmd(a.player, a.store),
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
		// DESIGN.md: Library 22%, Player 50%, Queue 28%.
		// Queue is not yet implemented (Feature 06), so Player gets 78%.
		libraryWidth := m.Width * 22 / 100
		playerWidth := m.Width - libraryWidth
		// Subtract 2 per border (left+right) from content width.
		a.libraryPane.SetSize(libraryWidth-2, m.Height-4)
		a.playerPane.SetSize(playerWidth-2, m.Height-4)
		return a, nil

	case tea.KeyMsg:
		// Global: q always quits.
		if m.Type == tea.KeyRunes && string(m.Runes) == "q" {
			return a, tea.Quit
		}
		// Tab rotates focus forward.
		if m.Type == tea.KeyTab {
			return a.rotateFocus(1)
		}
		// Shift+Tab rotates focus backward.
		if m.Type == tea.KeyShiftTab {
			return a.rotateFocus(-1)
		}
		// Playback keys always go to the player pane regardless of focus.
		// Temporarily enable focus so the pane handles the key even when
		// the library pane is active.
		if isPlaybackKey(m) {
			wasUnfocused := !a.playerPane.IsFocused()
			if wasUnfocused {
				a.playerPane.SetFocused(true)
			}
			updatedPane, cmd := a.playerPane.Update(m)
			if pp, ok := updatedPane.(*panes.PlayerPane); ok {
				a.playerPane = pp
			}
			if wasUnfocused {
				a.playerPane.SetFocused(false)
			}
			return a, cmd
		}
		// Route remaining keys to the focused pane.
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
		updatedPane, _ := a.playerPane.Update(m)
		if pp, ok := updatedPane.(*panes.PlayerPane); ok {
			a.playerPane = pp
		}
		return a, tea.Batch(
			fetchPlaybackStateCmd(a.player, a.store),
			tea.Tick(time.Second, func(_ time.Time) tea.Msg {
				return panes.TickMsg{}
			}),
		)

	case panes.PlaybackStateFetchedMsg:
		updatedPane, cmd := a.playerPane.Update(m)
		if pp, ok := updatedPane.(*panes.PlayerPane); ok {
			a.playerPane = pp
		}
		return a, cmd

	case panes.PlaybackCmdSentMsg:
		if m.Err != nil {
			a.statusMsg = fmt.Sprintf("✗ %s", m.Err.Error())
			return a, tea.Batch(
				fetchPlaybackStateCmd(a.player, a.store),
				tea.Tick(4*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} }),
			)
		}
		return a, fetchPlaybackStateCmd(a.player, a.store)

	case panes.PlaybackRequestMsg:
		return a, a.buildPlaybackAPICmd(m.Action)

	case panes.PlayContextMsg:
		return a, a.buildPlayContextCmd(m.ContextURI)

	case panes.PlayTrackMsg:
		return a, a.buildPlayTrackCmd(m.TrackURI)

	case panes.FetchPlaylistsRequestMsg:
		return a, a.buildFetchPlaylistsCmd(m.Offset)

	case panes.FetchAlbumsRequestMsg:
		return a, a.buildFetchAlbumsCmd(m.Offset)

	case panes.FetchLikedTracksRequestMsg:
		return a, a.buildFetchLikedTracksCmd(m.Offset)

	case panes.FetchRecentlyPlayedRequestMsg:
		return a, a.buildFetchRecentlyPlayedCmd()

	case panes.LikeTrackRequestMsg:
		return a, a.buildToggleLikeCmd(m.TrackID, m.Unlike)

	case panes.LikeToggleResultMsg:
		// Like/unlike result — no action needed unless there's an error.
		if m.Err != nil {
			a.statusMsg = fmt.Sprintf("✗ %s", m.Err.Error())
			return a, tea.Tick(4*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} })
		}
		return a, nil

	case statusDismissMsg:
		a.statusMsg = ""
		return a, nil
	}

	// Forward unhandled messages to the library pane (notification msgs, etc.).
	updated, cmd := a.libraryPane.Update(msg)
	if lp, ok := updated.(*panes.LibraryPane); ok {
		a.libraryPane = lp
	}
	return a, cmd
}

// isPlaybackKey returns true for keys that control playback regardless of focus.
func isPlaybackKey(m tea.KeyMsg) bool {
	if m.Type == tea.KeyRunes {
		switch string(m.Runes) {
		case " ", "n", "p", "+", "-", "s", "r":
			return true
		}
	}
	return m.Type == tea.KeyLeft || m.Type == tea.KeyRight
}

// rotateFocus cycles keyboard focus between library and player panes.
// direction: 1 = forward, -1 = backward.
func (a *App) rotateFocus(direction int) (*App, tea.Cmd) {
	// Only two panes for now; direction doesn't matter.
	_ = direction
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

// buildPlaybackAPICmd dispatches the Spotify API call for the given playback action.
func (a *App) buildPlaybackAPICmd(action panes.PlaybackAction) tea.Cmd {
	player := a.player
	store := a.store

	return func() tea.Msg {
		if player == nil {
			return panes.PlaybackCmdSentMsg{}
		}
		ctx := context.Background()
		var err error

		switch action {
		case panes.ActionPause:
			err = player.Pause(ctx)
		case panes.ActionPlay:
			err = player.Play(ctx, api.PlayOptions{})
		case panes.ActionNext:
			err = player.Next(ctx)
		case panes.ActionPrevious:
			err = player.Previous(ctx)
		case panes.ActionVolumeUp:
			ps := store.PlaybackState()
			vol := 65
			if ps != nil && ps.Device != nil {
				vol = ps.Device.VolumePercent
			}
			err = player.SetVolume(ctx, vol+5)
		case panes.ActionVolumeDown:
			ps := store.PlaybackState()
			vol := 65
			if ps != nil && ps.Device != nil {
				vol = ps.Device.VolumePercent
			}
			err = player.SetVolume(ctx, vol-5)
		case panes.ActionToggleShuffle:
			ps := store.PlaybackState()
			shuffle := false
			if ps != nil {
				shuffle = !ps.ShuffleState
			}
			err = player.SetShuffle(ctx, shuffle)
		case panes.ActionCycleRepeat:
			ps := store.PlaybackState()
			mode := "off"
			if ps != nil {
				mode = nextRepeatMode(ps.RepeatState)
			}
			err = player.SetRepeat(ctx, mode)
		}

		return panes.PlaybackCmdSentMsg{Err: err}
	}
}

// nextRepeatMode returns the next repeat mode in the cycle off→context→track→off.
func nextRepeatMode(current string) string {
	switch current {
	case "off":
		return "context"
	case "context":
		return "track"
	default:
		return "off"
	}
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

// buildFetchPlaylistsCmd creates a command that fetches playlists and writes to store.
func (a *App) buildFetchPlaylistsCmd(offset int) tea.Cmd {
	library := a.library
	store := a.store
	return func() tea.Msg {
		if library == nil {
			return panes.LibraryLoadedMsg{}
		}
		playlists, err := library.GetPlaylists(context.Background(), 50, offset)
		if err != nil {
			return panes.LibraryLoadedMsg{}
		}
		if offset == 0 {
			store.SetPlaylists(playlists)
		} else {
			store.SetPlaylists(append(store.Playlists(), playlists...))
		}
		store.SetPlaylistsTotal(len(store.Playlists()))
		return panes.LibraryLoadedMsg{}
	}
}

// buildFetchAlbumsCmd creates a command that fetches saved albums and writes to store.
func (a *App) buildFetchAlbumsCmd(offset int) tea.Cmd {
	library := a.library
	store := a.store
	return func() tea.Msg {
		if library == nil {
			return panes.AlbumsLoadedMsg{}
		}
		albums, err := library.GetSavedAlbums(context.Background(), 50, offset)
		if err != nil {
			return panes.AlbumsLoadedMsg{}
		}
		store.SetSavedAlbums(albums)
		return panes.AlbumsLoadedMsg{}
	}
}

// buildFetchLikedTracksCmd creates a command that fetches liked tracks and writes to store.
func (a *App) buildFetchLikedTracksCmd(offset int) tea.Cmd {
	library := a.library
	store := a.store
	return func() tea.Msg {
		if library == nil {
			return panes.LikedTracksLoadedMsg{}
		}
		tracks, err := library.GetLikedTracks(context.Background(), 50, offset)
		if err != nil {
			return panes.LikedTracksLoadedMsg{}
		}
		store.SetLikedTracks(tracks)
		store.SetLikedTotal(len(tracks) + offset)
		return panes.LikedTracksLoadedMsg{}
	}
}

// buildFetchRecentlyPlayedCmd creates a command that fetches recently played and writes to store.
func (a *App) buildFetchRecentlyPlayedCmd() tea.Cmd {
	library := a.library
	store := a.store
	return func() tea.Msg {
		if library == nil {
			return panes.RecentlyPlayedLoadedMsg{}
		}
		items, err := library.GetRecentlyPlayed(context.Background(), 20)
		if err != nil {
			return panes.RecentlyPlayedLoadedMsg{}
		}
		store.SetRecentlyPlayed(items)
		return panes.RecentlyPlayedLoadedMsg{}
	}
}

// buildToggleLikeCmd creates a command that likes or unlikes a track.
func (a *App) buildToggleLikeCmd(trackID string, unlike bool) tea.Cmd {
	library := a.library
	return func() tea.Msg {
		if library == nil {
			return panes.LikeToggleResultMsg{TrackID: trackID}
		}
		ctx := context.Background()
		var err error
		if unlike {
			err = library.UnlikeTrack(ctx, trackID)
		} else {
			err = library.LikeTrack(ctx, trackID)
		}
		return panes.LikeToggleResultMsg{TrackID: trackID, Err: err}
	}
}

// fetchPlaybackStateCmd creates a command that fetches the current playback state,
// writes directly to the store, and notifies panes via PlaybackStateFetchedMsg.
func fetchPlaybackStateCmd(player *api.Player, store *state.Store) tea.Cmd {
	return func() tea.Msg {
		if player == nil {
			return panes.PlaybackStateFetchedMsg{}
		}
		ps, err := player.GetPlaybackState(context.Background())
		if err != nil {
			return panes.PlaybackStateFetchedMsg{}
		}
		store.SetPlaybackState(ps)
		return panes.PlaybackStateFetchedMsg{}
	}
}

// View renders the full terminal UI.
func (a *App) View() string {
	// DESIGN.md: minimum terminal size check.
	if a.width > 0 && a.height > 0 && (a.width < 100 || a.height < 24) {
		return a.renderTooSmall()
	}

	header := a.renderHeader()
	statusBar := a.renderStatusBar()

	libraryView := a.renderPaneWithBorder(a.libraryPane.View(), a.focus == focusLibrary)
	playerView := a.renderPaneWithBorder(a.playerPane.View(), a.focus == focusPlayer)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, libraryView, playerView)

	return strings.Join([]string{header, mainContent, statusBar}, "\n")
}

// renderTooSmall renders the minimum size message per DESIGN.md.
func (a *App) renderTooSmall() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.ActiveBorder()).
		Padding(1, 2)

	msg := fmt.Sprintf(
		"Spotnik needs more space\n\nCurrent:  %d × %d\nRequired: 100 × 24\n\nPlease resize your terminal and retry.",
		a.width, a.height,
	)
	return style.Render(msg)
}

// renderPaneWithBorder wraps a pane's view with a rounded border per DESIGN.md.
func (a *App) renderPaneWithBorder(content string, focused bool) string {
	borderColor := a.theme.InactiveBorder()
	if focused {
		borderColor = a.theme.ActiveBorder()
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Render(content)
}

// renderHeader renders the top bar: app name left-aligned, device indicator right-aligned.
func (a *App) renderHeader() string {
	appNameStyle := lipgloss.NewStyle().
		Background(a.theme.SurfaceAlt()).
		Foreground(a.theme.TextPrimary()).
		Bold(true)

	device := a.store.ActiveDevice()
	var deviceStr string
	if device != nil {
		deviceStr = lipgloss.NewStyle().
			Foreground(a.theme.DeviceActive()).
			Render(fmt.Sprintf("◉ %s", device.Name))
	} else {
		deviceStr = lipgloss.NewStyle().
			Foreground(a.theme.TextMuted()).
			Render("○ No device")
	}

	appName := appNameStyle.Render(" spotnik ")

	if a.width > 0 {
		gap := a.width - lipgloss.Width(appName) - lipgloss.Width(deviceStr)
		if gap < 1 {
			gap = 1
		}
		return appName + strings.Repeat(" ", gap) + deviceStr
	}
	return appName + "  " + deviceStr
}

// renderStatusBar renders the bottom status bar with context-sensitive key hints.
func (a *App) renderStatusBar() string {
	if a.statusMsg != "" {
		return lipgloss.NewStyle().
			Background(a.theme.StatusBarBg()).
			Foreground(a.theme.Error()).
			Render("  " + a.statusMsg)
	}

	style := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.StatusBarFg())

	keyStyle := lipgloss.NewStyle().
		Foreground(a.theme.KeyHint()).
		Bold(true)

	var hints []string
	switch a.focus {
	case focusLibrary:
		hints = []string{
			keyStyle.Render("Enter") + " play",
			keyStyle.Render("a") + " queue",
			keyStyle.Render("l") + " like",
			keyStyle.Render("Tab") + " pane",
			keyStyle.Render("q") + " quit",
		}
	default:
		hints = []string{
			keyStyle.Render("Space") + " play",
			keyStyle.Render("n/p") + " skip",
			keyStyle.Render("+/-") + " vol",
			keyStyle.Render("s") + " shuffle",
			keyStyle.Render("r") + " repeat",
			keyStyle.Render("Tab") + " pane",
			keyStyle.Render("q") + " quit",
		}
	}

	return style.Render("  " + strings.Join(hints, "  "))
}
