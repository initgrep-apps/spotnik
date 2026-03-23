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
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// focusedPane identifies which pane currently has keyboard focus.
type focusedPane int

const (
	focusPlayer  focusedPane = iota // default: player pane has focus
	focusLibrary                    // library pane has focus
	focusQueue                      // queue pane has focus
)

// viewMode identifies which top-level view is currently active.
type viewMode int

const (
	viewSplash    viewMode = iota // Splash screen shown on startup
	viewMain                      // three-pane Library | Player | Queue layout
	viewAuth                      // Auth panel — shown when user needs to authenticate
	viewStats                     // Stats dashboard (press 2 to open, 1 to return)
	viewPlaylists                 // Playlist Manager (press 3 to open, 1 to return)
)

// App is the root application model. It owns the active theme, the central
// store, the API clients, and all pane models. It is the ONLY layer that
// calls the Spotify API — panes emit request messages and app.go dispatches them.
type App struct {
	theme   theme.Theme
	store   *state.Store
	player  *api.Player
	library *api.LibraryClient
	search  *api.SearchClient
	devices *api.DevicesClient
	userAPI *api.UserClient

	playerPane  *panes.PlayerPane
	libraryPane *panes.LibraryPane
	queuePane   *panes.QueuePane
	searchPane  *panes.SearchOverlay
	devicePane  *panes.DeviceOverlay

	// statsPane is lazy-initialized the first time the user presses 2.
	statsPane *panes.StatsView

	// playlistPane is lazy-initialized the first time the user presses 3.
	playlistPane *panes.PlaylistManager

	// playlistsAPI is the Spotify playlists mutation client.
	playlistsAPI *api.PlaylistsClient

	// currentView tracks which top-level view is displayed.
	currentView viewMode

	focus  focusedPane
	width  int
	height int

	// searchOpen is true while the search overlay is visible.
	searchOpen bool

	// deviceOverlayOpen is true while the device switcher overlay is visible.
	deviceOverlayOpen bool

	// prevFocus captures which pane was focused before search/device overlay opened,
	// so it can be restored when the overlay closes.
	prevFocus focusedPane

	// statusMsg is a transient error/info message shown in the status bar for 3–4 seconds.
	statusMsg string

	// volumeStep is the percentage change per volume up/down keypress.
	volumeStep int

	// Auth state — set at construction, used for splash→auth transition.
	needsAuth  bool
	clientID   string
	tokenStore keychain.TokenStore
	authURL    string // Spotify authorization URL, set when auth flow starts
	authStatus string // Message shown in auth panel
}

// statusDismissMsg is sent after 4 seconds to clear a transient status bar message.
type statusDismissMsg struct{}

// splashDismissMsg is sent after 2 seconds to close the splash screen.
type splashDismissMsg struct{}

// AppOptions holds optional configuration for App construction.
type AppOptions struct {
	NeedsAuth  bool
	ClientID   string
	TokenStore keychain.TokenStore
}

// New creates a new App, loading the theme from cfg.UI.Theme.
func New(cfg *config.Config, opts AppOptions) *App {
	t := theme.Load(cfg.UI.Theme)
	s := state.New()

	playerPane := panes.NewPlayerPane(s, t, true)
	libraryPane := panes.NewLibraryPane(s, t, false)
	queuePane := panes.NewQueuePane(s, t, false)
	searchPane := panes.NewSearchOverlay(s, t)
	devicePane := panes.NewDeviceOverlay(s, t)

	volStep := cfg.UI.VolumeStep
	if volStep <= 0 {
		volStep = 5
	}

	return &App{
		theme:       t,
		store:       s,
		playerPane:  playerPane,
		libraryPane: libraryPane,
		queuePane:   queuePane,
		searchPane:  searchPane,
		devicePane:  devicePane,
		focus:       focusPlayer,
		currentView: viewSplash,
		volumeStep:  volStep,
		needsAuth:   opts.NeedsAuth,
		clientID:    opts.ClientID,
		tokenStore:  opts.TokenStore,
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

// SetSearch injects the Spotify API search client into the app.
func (a *App) SetSearch(search *api.SearchClient) {
	a.search = search
}

// SetDevices injects the Spotify Connect devices API client into the app.
func (a *App) SetDevices(devices *api.DevicesClient) {
	a.devices = devices
}

// SetUserAPI injects the Spotify user stats API client into the app.
func (a *App) SetUserAPI(userAPI *api.UserClient) {
	a.userAPI = userAPI
}

// SetPlaylistsAPI injects the Spotify playlists mutation client into the app.
func (a *App) SetPlaylistsAPI(p *api.PlaylistsClient) {
	a.playlistsAPI = p
}

// StatsViewOpen returns true while the stats view is the active top-level view.
func (a *App) StatsViewOpen() bool {
	return a.currentView == viewStats
}

// PlaylistViewOpen returns true while the Playlist Manager is the active top-level view.
func (a *App) PlaylistViewOpen() bool {
	return a.currentView == viewPlaylists
}

// SearchOpen returns true while the search overlay is visible.
func (a *App) SearchOpen() bool {
	return a.searchOpen
}

// DeviceOverlayOpen returns true while the device switcher overlay is visible.
func (a *App) DeviceOverlayOpen() bool {
	return a.deviceOverlayOpen
}

// LibraryFocused returns true if the library pane currently has keyboard focus.
func (a *App) LibraryFocused() bool {
	return a.focus == focusLibrary
}

// PlayerFocused returns true if the player pane currently has keyboard focus.
func (a *App) PlayerFocused() bool {
	return a.focus == focusPlayer
}

// QueueFocused returns true if the queue pane currently has keyboard focus.
func (a *App) QueueFocused() bool {
	return a.focus == focusQueue
}

// Init starts the polling loop and fetches initial playback state.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		fetchPlaybackStateCmd(a.player, a.store),
		a.libraryPane.Init(),
		tea.Tick(time.Second, func(_ time.Time) tea.Msg {
			return panes.TickMsg{}
		}),
		tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
			return splashDismissMsg{}
		}),
	)
}

// openSearch opens the search overlay and captures the current pane focus.
func (a *App) openSearch() (*App, tea.Cmd) {
	a.prevFocus = a.focus
	a.searchOpen = true
	cmd := a.searchPane.Init()
	return a, cmd
}

// closeSearch closes the search overlay and restores the previous pane focus.
func (a *App) closeSearch() (*App, tea.Cmd) {
	a.searchOpen = false
	return a, nil
}

// openDeviceOverlay opens the device switcher overlay and fetches the device list.
func (a *App) openDeviceOverlay() (*App, tea.Cmd) {
	a.prevFocus = a.focus
	a.deviceOverlayOpen = true
	cmd := a.devicePane.Init()
	return a, cmd
}

// closeDeviceOverlay closes the device switcher overlay and restores previous focus.
func (a *App) closeDeviceOverlay() (*App, tea.Cmd) {
	a.deviceOverlayOpen = false
	return a, nil
}

// openStats switches to the Stats view. The StatsView is lazy-initialized on the
// first call — cursor and section focus are preserved on subsequent calls.
func (a *App) openStats() (*App, tea.Cmd) {
	a.currentView = viewStats
	if a.statsPane == nil {
		// First open: construct and init the stats pane.
		sv := panes.NewStatsView(a.store, a.theme)
		if a.width > 0 {
			sv.SetSize(a.width, a.height-4) // subtract header + status bar rows
		}
		a.statsPane = sv
		return a, sv.Init()
	}
	return a, nil
}

// closeStats returns to the main three-pane layout.
func (a *App) closeStats() (*App, tea.Cmd) {
	a.currentView = viewMain
	return a, nil
}

// openPlaylists switches to the Playlist Manager view. The PlaylistManager is
// lazy-initialized on the first call and Init() is called to trigger a playlist
// fetch if the store is empty. Subsequent calls reuse the existing pane and store data.
func (a *App) openPlaylists() (*App, tea.Cmd) {
	a.currentView = viewPlaylists
	if a.playlistPane == nil {
		pm := panes.NewPlaylistManager(a.store, a.theme)
		if a.width > 0 {
			pm.SetSize(a.width, a.height-4) // subtract header + status bar rows
		}
		a.playlistPane = pm
		return a, pm.Init()
	}
	return a, nil
}

// closePlaylists returns to the main three-pane layout.
func (a *App) closePlaylists() (*App, tea.Cmd) {
	a.currentView = viewMain
	return a, nil
}

// Update handles all messages routed through the root model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case splashDismissMsg:
		if a.currentView == viewSplash {
			a.currentView = viewMain
		}
		return a, nil

	case panes.SearchClosedMsg:
		// Search overlay closed — restore previous focus and close overlay.
		return a.closeSearch()

	case panes.SearchRequestMsg:
		// Debounce fired — dispatch search API call.
		return a, a.buildSearchCmd(m.Query)

	case panes.SearchResultsMsg:
		// Search results are in the store; notify the overlay.
		a.store.SetSearchLoading(false)
		updated, cmd := a.searchPane.Update(m)
		if sp, ok := updated.(*panes.SearchOverlay); ok {
			a.searchPane = sp
		}
		return a, cmd

	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		// DESIGN.md: Library 22%, Player 50%, Queue 28%.
		libraryWidth := m.Width * 22 / 100
		queueWidth := m.Width * 28 / 100
		playerWidth := m.Width - libraryWidth - queueWidth
		// Subtract 2 per border (left+right) from content width.
		a.libraryPane.SetSize(libraryWidth-2, m.Height-4)
		a.playerPane.SetSize(playerWidth-2, m.Height-4)
		a.queuePane.SetSize(queueWidth-2, m.Height-4)
		a.searchPane.SetSize(m.Width, m.Height)
		a.devicePane.SetSize(m.Width, m.Height)
		if a.statsPane != nil {
			a.statsPane.SetSize(m.Width, m.Height-4)
		}
		if a.playlistPane != nil {
			a.playlistPane.SetSize(m.Width, m.Height-4)
		}
		return a, nil

	case panes.FetchStatsMsg:
		// Stats view requesting data for a time range.
		return a, a.buildFetchStatsCmd(m.TimeRange)

	case panes.StatsLoadedMsg:
		// Stats data fetched — forward to stats pane.
		if a.statsPane != nil {
			updated, cmd := a.statsPane.Update(m)
			if sv, ok := updated.(*panes.StatsView); ok {
				a.statsPane = sv
			}
			return a, cmd
		}
		return a, nil

	case tea.KeyMsg:
		// When device overlay is open, route all keys to the device pane.
		if a.deviceOverlayOpen {
			updated, cmd := a.devicePane.Update(m)
			if dp, ok := updated.(*panes.DeviceOverlay); ok {
				a.devicePane = dp
			}
			return a, cmd
		}

		// When search overlay is open, route all keys to the search pane first.
		if a.searchOpen {
			updated, cmd := a.searchPane.Update(m)
			if sp, ok := updated.(*panes.SearchOverlay); ok {
				a.searchPane = sp
			}
			return a, cmd
		}

		// Global: q always quits.
		if m.Type == tea.KeyRunes && string(m.Runes) == "q" {
			return a, tea.Quit
		}
		// '2' opens the Stats view.
		if m.Type == tea.KeyRunes && string(m.Runes) == "2" {
			return a.openStats()
		}
		// '3' opens the Playlist Manager view.
		if m.Type == tea.KeyRunes && string(m.Runes) == "3" {
			return a.openPlaylists()
		}
		// '1' returns to the main three-pane layout from any alternate view.
		if m.Type == tea.KeyRunes && string(m.Runes) == "1" {
			if a.currentView == viewPlaylists {
				return a.closePlaylists()
			}
			return a.closeStats()
		}

		// When stats view is open, route all non-global keys to it.
		if a.currentView == viewStats {
			if a.statsPane != nil {
				updated, cmd := a.statsPane.Update(m)
				if sv, ok := updated.(*panes.StatsView); ok {
					a.statsPane = sv
				}
				return a, cmd
			}
			return a, nil
		}

		// When playlist view is open, route all non-global keys to the playlist pane.
		if a.currentView == viewPlaylists {
			if a.playlistPane != nil {
				updated, cmd := a.playlistPane.Update(m)
				if pm, ok := updated.(*panes.PlaylistManager); ok {
					a.playlistPane = pm
				}
				return a, cmd
			}
			return a, nil
		}

		// '/' opens the search overlay from any pane.
		if m.Type == tea.KeyRunes && string(m.Runes) == "/" {
			return a.openSearch()
		}
		// 'd' opens the device switcher overlay from any pane.
		if m.Type == tea.KeyRunes && string(m.Runes) == "d" {
			return a.openDeviceOverlay()
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
		case focusQueue:
			updated, cmd := a.queuePane.Update(m)
			if qp, ok := updated.(*panes.QueuePane); ok {
				a.queuePane = qp
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
			fetchQueueCmd(a.player, a.store),
			tea.Tick(time.Second, func(_ time.Time) tea.Msg {
				return panes.TickMsg{}
			}),
		)

	case panes.QueueLoadedMsg:
		// Queue data is already in the store — no pane-specific handling needed here.
		// QueuePane reads directly from store on View().
		return a, nil

	case panes.PlaybackStateFetchedMsg:
		// Dismiss splash screen when first playback data arrives.
		if a.currentView == viewSplash {
			a.currentView = viewMain
		}
		updatedPane, cmd := a.playerPane.Update(m)
		if pp, ok := updatedPane.(*panes.PlayerPane); ok {
			a.playerPane = pp
		}
		return a, cmd

	case panes.PlaybackCmdSentMsg:
		if m.Err != nil {
			errMsg := m.Err.Error()
			if strings.Contains(errMsg, "403") {
				a.statusMsg = "Playback control not available on this device"
			} else {
				a.statusMsg = fmt.Sprintf("✗ %s", errMsg)
			}
			return a, tea.Batch(
				fetchPlaybackStateCmd(a.player, a.store),
				tea.Tick(4*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} }),
			)
		}
		return a, fetchPlaybackStateCmd(a.player, a.store)

	case panes.PlaybackRequestMsg:
		return a, a.buildPlaybackAPICmd(m.Action)

	case panes.PlayContextMsg:
		// Close search overlay when playing from search results.
		if a.searchOpen {
			a.searchOpen = false
		}
		return a, a.buildPlayContextCmd(m.ContextURI)

	case panes.PlayTrackMsg:
		// Close search overlay when playing from search results.
		if a.searchOpen {
			a.searchOpen = false
		}
		return a, a.buildPlayTrackCmd(m.TrackURI)

	case panes.AddToQueueMsg:
		return a, a.buildAddToQueueCmd(m.TrackURI, m.TrackName)

	case panes.AddToQueueResultMsg:
		if m.Err != nil {
			a.statusMsg = fmt.Sprintf("✗ %s", m.Err.Error())
			return a, tea.Tick(3*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} })
		}
		if m.TrackName != "" {
			a.statusMsg = fmt.Sprintf("✓ Added to queue: %s", m.TrackName)
		} else {
			a.statusMsg = "✓ Added to queue"
		}
		return a, tea.Tick(3*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} })

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

	case panes.DeviceOverlayClosedMsg:
		// Device overlay closed via Esc — restore previous focus.
		return a.closeDeviceOverlay()

	case panes.FetchDevicesRequestMsg:
		// Device overlay is requesting the device list from the API.
		return a, a.buildFetchDevicesCmd()

	case panes.TransferPlaybackMsg:
		// User selected a device; show status and dispatch transfer API call.
		a.statusMsg = fmt.Sprintf("Switching to %s...", m.DeviceName)
		a.deviceOverlayOpen = false
		return a, tea.Batch(
			a.buildTransferPlaybackCmd(m.DeviceID),
			tea.Tick(4*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} }),
		)

	case panes.DeviceTransferredMsg:
		if m.Err != nil {
			a.statusMsg = fmt.Sprintf("✗ %s", m.Err.Error())
			return a, tea.Batch(
				fetchPlaybackStateCmd(a.player, a.store),
				tea.Tick(4*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} }),
			)
		}
		// Transfer succeeded — next poll will update the header.
		return a, fetchPlaybackStateCmd(a.player, a.store)

	case statusDismissMsg:
		a.statusMsg = ""
		return a, nil
	}

	// When device overlay is open, forward non-key messages (devices loaded, etc.)
	// to the device pane so they can be processed.
	if a.deviceOverlayOpen {
		updated, cmd := a.devicePane.Update(msg)
		if dp, ok := updated.(*panes.DeviceOverlay); ok {
			a.devicePane = dp
		}
		return a, cmd
	}

	// When search overlay is open, forward non-key messages (debounce ticks,
	// spinner ticks) to the search pane so they can be processed.
	if a.searchOpen {
		updated, cmd := a.searchPane.Update(msg)
		if sp, ok := updated.(*panes.SearchOverlay); ok {
			a.searchPane = sp
		}
		return a, cmd
	}

	// Playlist message routing — handled regardless of current view.
	switch m := msg.(type) {
	case panes.FetchPlaylistTracksRequestMsg:
		return a, a.buildFetchPlaylistTracksCmd(m.PlaylistID)

	case panes.PlaylistTracksLoadedMsg:
		// Forward to playlist pane so it can refresh from store.
		if a.playlistPane != nil {
			updated, cmd := a.playlistPane.Update(m)
			if pm, ok := updated.(*panes.PlaylistManager); ok {
				a.playlistPane = pm
			}
			return a, cmd
		}
		return a, nil

	case panes.PlaylistCreateRequestMsg:
		return a, a.buildCreatePlaylistCmd(m.Name, m.Description)

	case panes.PlaylistCreatedMsg:
		if m.Err != nil {
			a.statusMsg = fmt.Sprintf("✗ %s", m.Err.Error())
			return a, tea.Tick(4*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} })
		}
		// Re-fetch playlists so the new one appears.
		return a, a.buildFetchPlaylistsCmd(0)

	case panes.PlaylistRenameRequestMsg:
		return a, a.buildRenamePlaylistCmd(m.PlaylistID, m.NewName)

	case panes.PlaylistRenamedMsg:
		if m.Err != nil {
			a.statusMsg = fmt.Sprintf("✗ %s", m.Err.Error())
			if a.playlistPane != nil {
				updated, _ := a.playlistPane.Update(m)
				if pm, ok := updated.(*panes.PlaylistManager); ok {
					a.playlistPane = pm
				}
			}
			return a, tea.Tick(4*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} })
		}
		// Re-fetch playlists to reflect rename.
		return a, a.buildFetchPlaylistsCmd(0)

	case panes.PlaylistRemoveRequestMsg:
		return a, a.buildRemovePlaylistTrackCmd(m.PlaylistID, m.TrackURI)

	case panes.PlaylistRemoveResultMsg:
		if a.playlistPane != nil {
			updated, cmd := a.playlistPane.Update(m)
			if pm, ok := updated.(*panes.PlaylistManager); ok {
				a.playlistPane = pm
			}
			if m.Err != nil {
				a.statusMsg = fmt.Sprintf("✗ %s", m.Err.Error())
				return a, tea.Batch(cmd, tea.Tick(4*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} }))
			}
			return a, cmd
		}
		return a, nil

	case panes.PlaylistReorderRequestMsg:
		return a, a.buildReorderPlaylistTracksCmd(m.PlaylistID, m.RangeStart, m.InsertBefore, m.RangeLength)

	case panes.PlaylistReorderResultMsg:
		if a.playlistPane != nil {
			updated, cmd := a.playlistPane.Update(m)
			if pm, ok := updated.(*panes.PlaylistManager); ok {
				a.playlistPane = pm
			}
			if m.Err != nil {
				a.statusMsg = fmt.Sprintf("✗ %s", m.Err.Error())
				return a, tea.Batch(cmd, tea.Tick(4*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} }))
			}
			return a, cmd
		}
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

// rotateFocus cycles keyboard focus between the three panes.
// direction: 1 = forward (player → library → queue → player),
//
//	-1 = backward (player → queue → library → player).
func (a *App) rotateFocus(direction int) (*App, tea.Cmd) {
	// Clear all pane focus states.
	a.playerPane.SetFocused(false)
	a.libraryPane.SetFocused(false)
	a.queuePane.SetFocused(false)

	// Advance focus in the requested direction.
	switch a.focus {
	case focusPlayer:
		if direction >= 0 {
			a.focus = focusLibrary
			a.libraryPane.SetFocused(true)
		} else {
			a.focus = focusQueue
			a.queuePane.SetFocused(true)
		}
	case focusLibrary:
		if direction >= 0 {
			a.focus = focusQueue
			a.queuePane.SetFocused(true)
		} else {
			a.focus = focusPlayer
			a.playerPane.SetFocused(true)
		}
	default: // focusQueue
		if direction >= 0 {
			a.focus = focusPlayer
			a.playerPane.SetFocused(true)
		} else {
			a.focus = focusLibrary
			a.libraryPane.SetFocused(true)
		}
	}
	return a, nil
}

// buildPlaybackAPICmd dispatches the Spotify API call for the given playback action.
func (a *App) buildPlaybackAPICmd(action panes.PlaybackAction) tea.Cmd {
	player := a.player
	store := a.store

	volStep := a.volumeStep

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
			newVol := vol + volStep
			if newVol > 100 {
				newVol = 100
			}
			err = player.SetVolume(ctx, newVol)
		case panes.ActionVolumeDown:
			ps := store.PlaybackState()
			vol := 65
			if ps != nil && ps.Device != nil {
				vol = ps.Device.VolumePercent
			}
			newVol := vol - volStep
			if newVol < 0 {
				newVol = 0
			}
			err = player.SetVolume(ctx, newVol)
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
			store.SetPlaylistsFetchError(err)
			return panes.LibraryLoadedMsg{}
		}
		store.ClearPlaylistsFetchError()
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
			store.SetAlbumsFetchError(err)
			return panes.AlbumsLoadedMsg{}
		}
		store.ClearAlbumsFetchError()
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
			store.SetLikedTracksFetchError(err)
			return panes.LikedTracksLoadedMsg{}
		}
		store.ClearLikedTracksFetchError()
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
			store.SetRecentPlayedFetchError(err)
			return panes.RecentlyPlayedLoadedMsg{}
		}
		store.ClearRecentPlayedFetchError()
		store.SetRecentlyPlayed(items)
		return panes.RecentlyPlayedLoadedMsg{}
	}
}

// buildAddToQueueCmd creates a command that adds a track to the user's queue.
// trackName is threaded through to AddToQueueResultMsg for status bar display.
func (a *App) buildAddToQueueCmd(trackURI, trackName string) tea.Cmd {
	player := a.player
	return func() tea.Msg {
		if player == nil {
			return panes.AddToQueueResultMsg{TrackName: trackName}
		}
		err := player.AddToQueue(context.Background(), trackURI)
		return panes.AddToQueueResultMsg{Err: err, TrackName: trackName}
	}
}

// buildSearchCmd creates a command that calls the Spotify search API and writes results to store.
func (a *App) buildSearchCmd(query string) tea.Cmd {
	search := a.search
	store := a.store
	store.SetSearchQuery(query)
	store.SetSearchLoading(true)

	return func() tea.Msg {
		if search == nil {
			store.SetSearchLoading(false)
			return panes.SearchResultsMsg{}
		}
		results, err := search.Search(
			context.Background(),
			query,
			[]string{"track", "artist", "album", "playlist"},
			5,
		)
		if err != nil {
			store.SetSearchLoading(false)
			store.SetSearchError(err)
			return panes.SearchResultsMsg{Err: err}
		}
		store.ClearSearchError()
		store.SetSearchResults(results)
		store.SetSearchLoading(false)
		return panes.SearchResultsMsg{}
	}
}

// buildFetchDevicesCmd creates a command that fetches the available Spotify Connect devices
// and delivers them back to the DeviceOverlay via devicesLoadedMsg.
func (a *App) buildFetchDevicesCmd() tea.Cmd {
	devices := a.devices
	store := a.store
	return func() tea.Msg {
		if devices == nil {
			// Deliver empty list when no client is injected (tests / uninitialized).
			return panes.NewDevicesLoadedMsg(nil, nil)
		}
		devList, err := devices.GetDevices(context.Background())
		if err != nil {
			store.SetDevicesError(err)
		} else {
			store.ClearDevicesError()
		}
		return panes.NewDevicesLoadedMsg(devList, err)
	}
}

// buildTransferPlaybackCmd creates a command that calls the TransferPlayback API
// and returns a DeviceTransferredMsg with any error.
func (a *App) buildTransferPlaybackCmd(deviceID string) tea.Cmd {
	devices := a.devices
	return func() tea.Msg {
		if devices == nil {
			return panes.DeviceTransferredMsg{DeviceID: deviceID}
		}
		err := devices.TransferPlayback(context.Background(), deviceID, true)
		return panes.DeviceTransferredMsg{DeviceID: deviceID, Err: err}
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

// buildFetchStatsCmd creates a command that fetches top tracks and artists for
// the given time range, writes to the store, and returns a StatsLoadedMsg.
func (a *App) buildFetchStatsCmd(timeRange string) tea.Cmd {
	userAPI := a.userAPI
	store := a.store
	return func() tea.Msg {
		if userAPI == nil {
			return panes.StatsLoadedMsg{TimeRange: timeRange}
		}
		ctx := context.Background()
		tracks, err := userAPI.GetTopTracks(ctx, timeRange, 25)
		if err != nil {
			store.SetStatsError(err)
			return panes.StatsLoadedMsg{TimeRange: timeRange}
		}
		artists, err := userAPI.GetTopArtists(ctx, timeRange, 25)
		if err != nil {
			store.SetStatsError(err)
			return panes.StatsLoadedMsg{TimeRange: timeRange}
		}
		store.ClearStatsError()
		store.SetTopTracks(timeRange, tracks)
		store.SetTopArtists(timeRange, artists)
		return panes.StatsLoadedMsg{TimeRange: timeRange}
	}
}

// fetchQueueCmd creates a command that fetches the current play queue,
// writes the queue tracks to the store, and notifies panes via QueueLoadedMsg.
func fetchQueueCmd(player *api.Player, store *state.Store) tea.Cmd {
	return func() tea.Msg {
		if player == nil {
			return panes.QueueLoadedMsg{}
		}
		qr, err := player.GetQueue(context.Background())
		if err != nil {
			store.SetQueueError(err)
			return panes.QueueLoadedMsg{}
		}
		store.ClearQueueError()
		if qr != nil {
			store.SetQueue(qr.Queue)
		}
		return panes.QueueLoadedMsg{}
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

	// Splash screen on startup (only when terminal size is known).
	if a.currentView == viewSplash {
		if a.width > 0 && a.height > 0 {
			return a.renderSplash()
		}
		// No size yet — fall through to main view for tests.
	}

	// Stats view replaces the three-pane layout when active.
	if a.currentView == viewStats && a.statsPane != nil {
		header := a.renderStatsHeader()
		statsContent := a.statsPane.View()
		statusBar := a.renderStatsStatusBar()
		return strings.Join([]string{header, statsContent, statusBar}, "\n")
	}

	// Playlist Manager replaces the three-pane layout when active.
	if a.currentView == viewPlaylists && a.playlistPane != nil {
		header := a.renderPlaylistsHeader()
		playlistContent := a.playlistPane.View()
		statusBar := a.renderPlaylistsStatusBar()
		return strings.Join([]string{header, playlistContent, statusBar}, "\n")
	}

	header := a.renderHeader()
	statusBar := a.renderStatusBar()

	libraryView := a.renderPaneWithBorder(a.libraryPane.View(), a.focus == focusLibrary)
	playerView := a.renderPaneWithBorder(a.playerPane.View(), a.focus == focusPlayer)
	queueView := a.renderPaneWithBorder(a.queuePane.View(), a.focus == focusQueue)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, libraryView, playerView, queueView)
	body := strings.Join([]string{header, mainContent, statusBar}, "\n")

	if a.deviceOverlayOpen {
		return a.renderWithDeviceOverlay(body)
	}

	if a.searchOpen {
		return a.renderWithSearchOverlay(body)
	}

	return body
}

// renderWithDeviceOverlay renders the three-pane view dimmed and places the
// device switcher overlay in the top-right area per the DESIGN.md spec.
func (a *App) renderWithDeviceOverlay(background string) string {
	overlay := a.devicePane.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + overlay
	}

	// Position overlay in the top-right area (below the header/device indicator).
	centered := lipgloss.Place(
		a.width, a.height,
		lipgloss.Right, lipgloss.Top,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)
	return centered
}

// renderWithSearchOverlay renders the three-pane view dimmed and places the
// search overlay centered on top using lipgloss.Place() per the DESIGN.md spec.
func (a *App) renderWithSearchOverlay(background string) string {
	overlay := a.searchPane.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + overlay
	}

	// Center the overlay on a consistent black background so the dimmed
	// three-pane view is replaced with a uniform dark surface behind the modal.
	centered := lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)
	return centered
}

// renderTooSmall renders the minimum size message per DESIGN.md.
// renderSplash renders the startup splash screen with ASCII art branding.
func (a *App) renderSplash() string {
	return renderSplashView(a.theme, a.width, a.height)
}

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

// maxDeviceNameLen is the maximum number of characters for the device name in the header.
const maxDeviceNameLen = 25

// truncateDeviceName truncates a device name to maxDeviceNameLen chars, appending … if needed.
func truncateDeviceName(name string) string {
	runes := []rune(name)
	if len(runes) > maxDeviceNameLen {
		return string(runes[:maxDeviceNameLen-1]) + "…"
	}
	return name
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
		name := truncateDeviceName(device.Name)
		deviceStr = lipgloss.NewStyle().
			Foreground(a.theme.DeviceActive()).
			Render(fmt.Sprintf("◉ %s", name))
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

// renderStatsHeader renders the header bar while the stats view is active.
// It shows "spotnik [STATS]" on the left and the device indicator on the right.
func (a *App) renderStatsHeader() string {
	appNameStyle := lipgloss.NewStyle().
		Background(a.theme.SurfaceAlt()).
		Foreground(a.theme.TextPrimary()).
		Bold(true)

	statsLabelStyle := lipgloss.NewStyle().
		Foreground(a.theme.SectionHeader()).
		Bold(true)

	device := a.store.ActiveDevice()
	var deviceStr string
	if device != nil {
		name := truncateDeviceName(device.Name)
		deviceStr = lipgloss.NewStyle().
			Foreground(a.theme.DeviceActive()).
			Render(fmt.Sprintf("◉ %s", name))
	} else {
		deviceStr = lipgloss.NewStyle().
			Foreground(a.theme.TextMuted()).
			Render("○ No device")
	}

	appName := appNameStyle.Render(" spotnik ") + " " + statsLabelStyle.Render("[STATS]")

	if a.width > 0 {
		gap := a.width - lipgloss.Width(appName) - lipgloss.Width(deviceStr)
		if gap < 1 {
			gap = 1
		}
		return appName + strings.Repeat(" ", gap) + deviceStr
	}
	return appName + "  " + deviceStr
}

// renderStatsStatusBar renders the bottom status bar for the stats view.
func (a *App) renderStatsStatusBar() string {
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

	hints := []string{
		keyStyle.Render("Tab") + " next section",
		keyStyle.Render("j/k") + " move",
		keyStyle.Render("Enter") + " play",
		keyStyle.Render("f") + " cycle range",
		keyStyle.Render("1") + " library",
		keyStyle.Render("q") + " quit",
	}
	return style.Render("  " + strings.Join(hints, "  "))
}

// renderPlaylistsHeader renders the header bar while the Playlist Manager view is active.
// It shows "spotnik [PLAYLISTS]" on the left and the device indicator on the right.
func (a *App) renderPlaylistsHeader() string {
	appNameStyle := lipgloss.NewStyle().
		Background(a.theme.SurfaceAlt()).
		Foreground(a.theme.TextPrimary()).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(a.theme.SectionHeader()).
		Bold(true)

	device := a.store.ActiveDevice()
	var deviceStr string
	if device != nil {
		name := truncateDeviceName(device.Name)
		deviceStr = lipgloss.NewStyle().
			Foreground(a.theme.DeviceActive()).
			Render(fmt.Sprintf("◉ %s", name))
	} else {
		deviceStr = lipgloss.NewStyle().
			Foreground(a.theme.TextMuted()).
			Render("○ No device")
	}

	appName := appNameStyle.Render(" spotnik ") + " " + labelStyle.Render("[PLAYLISTS]")

	if a.width > 0 {
		gap := a.width - lipgloss.Width(appName) - lipgloss.Width(deviceStr)
		if gap < 1 {
			gap = 1
		}
		return appName + strings.Repeat(" ", gap) + deviceStr
	}
	return appName + "  " + deviceStr
}

// renderPlaylistsStatusBar renders the bottom status bar for the Playlist Manager view.
func (a *App) renderPlaylistsStatusBar() string {
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

	hints := []string{
		keyStyle.Render("Enter") + " play",
		keyStyle.Render("r") + " rename",
		keyStyle.Render("n") + " new playlist",
		keyStyle.Render("x") + " remove track",
		keyStyle.Render("Shift+↑↓") + " reorder",
		keyStyle.Render("Tab") + " switch pane",
		keyStyle.Render("1") + " library",
		keyStyle.Render("q") + " quit",
	}
	return style.Render("  " + strings.Join(hints, "  "))
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
			keyStyle.Render("/") + " search",
			keyStyle.Render("Enter") + " play",
			keyStyle.Render("a") + " queue",
			keyStyle.Render("l") + " like",
			keyStyle.Render("d") + " devices",
			keyStyle.Render("2") + " stats",
			keyStyle.Render("3") + " lists",
			keyStyle.Render("Tab") + " pane",
			keyStyle.Render("q") + " quit",
		}
	case focusQueue:
		hints = []string{
			keyStyle.Render("/") + " search",
			keyStyle.Render("j/k") + " navigate",
			keyStyle.Render("Enter") + " play",
			keyStyle.Render("d") + " devices",
			keyStyle.Render("2") + " stats",
			keyStyle.Render("3") + " lists",
			keyStyle.Render("Tab") + " pane",
			keyStyle.Render("q") + " quit",
		}
	default:
		hints = []string{
			keyStyle.Render("/") + " search",
			keyStyle.Render("Space") + " play",
			keyStyle.Render("n/p") + " skip",
			keyStyle.Render("+/-") + " vol",
			keyStyle.Render("s") + " shuffle",
			keyStyle.Render("r") + " repeat",
			keyStyle.Render("d") + " devices",
			keyStyle.Render("2") + " stats",
			keyStyle.Render("3") + " lists",
			keyStyle.Render("Tab") + " pane",
			keyStyle.Render("q") + " quit",
		}
	}

	return style.Render("  " + strings.Join(hints, "  "))
}

// buildFetchPlaylistTracksCmd creates a command that fetches tracks for a playlist
// and writes them to the store, then sends PlaylistTracksLoadedMsg.
func (a *App) buildFetchPlaylistTracksCmd(playlistID string) tea.Cmd {
	library := a.library
	store := a.store
	return func() tea.Msg {
		if library == nil {
			return panes.PlaylistTracksLoadedMsg{PlaylistID: playlistID}
		}
		tracks, err := library.GetPlaylistTracks(context.Background(), playlistID, 100, 0)
		if err != nil {
			store.SetPlaylistsError(err)
			return panes.PlaylistTracksLoadedMsg{PlaylistID: playlistID}
		}
		store.ClearPlaylistsError()
		store.SetPlaylistTracks(playlistID, tracks)
		return panes.PlaylistTracksLoadedMsg{PlaylistID: playlistID}
	}
}

// buildCreatePlaylistCmd creates a command that calls CreatePlaylist on the playlists API
// and returns a PlaylistCreatedMsg with the result.
func (a *App) buildCreatePlaylistCmd(name, description string) tea.Cmd {
	playlistsAPI := a.playlistsAPI
	return func() tea.Msg {
		if playlistsAPI == nil {
			// No API client in tests — return a success with empty playlist.
			return panes.PlaylistCreatedMsg{Name: name}
		}
		playlist, err := playlistsAPI.CreatePlaylist(context.Background(), name, description, false)
		if err != nil {
			return panes.PlaylistCreatedMsg{Name: name, Err: err}
		}
		return panes.PlaylistCreatedMsg{PlaylistID: playlist.ID, Name: playlist.Name}
	}
}

// buildRenamePlaylistCmd creates a command that calls UpdatePlaylist on the playlists API
// and returns a PlaylistRenamedMsg.
func (a *App) buildRenamePlaylistCmd(playlistID, newName string) tea.Cmd {
	playlistsAPI := a.playlistsAPI
	return func() tea.Msg {
		if playlistsAPI == nil {
			return panes.PlaylistRenamedMsg{PlaylistID: playlistID, NewName: newName}
		}
		err := playlistsAPI.UpdatePlaylist(context.Background(), playlistID, newName, "")
		return panes.PlaylistRenamedMsg{PlaylistID: playlistID, NewName: newName, Err: err}
	}
}

// buildRemovePlaylistTrackCmd creates a command that calls RemoveTracksFromPlaylist
// and returns a PlaylistRemoveResultMsg.
func (a *App) buildRemovePlaylistTrackCmd(playlistID, trackURI string) tea.Cmd {
	playlistsAPI := a.playlistsAPI
	return func() tea.Msg {
		if playlistsAPI == nil {
			return panes.PlaylistRemoveResultMsg{PlaylistID: playlistID, TrackURI: trackURI}
		}
		err := playlistsAPI.RemoveTracksFromPlaylist(context.Background(), playlistID, []string{trackURI})
		return panes.PlaylistRemoveResultMsg{PlaylistID: playlistID, TrackURI: trackURI, Err: err}
	}
}

// buildReorderPlaylistTracksCmd creates a command that calls ReorderPlaylistTracks
// and returns a PlaylistReorderResultMsg.
func (a *App) buildReorderPlaylistTracksCmd(playlistID string, rangeStart, insertBefore, rangeLength int) tea.Cmd {
	playlistsAPI := a.playlistsAPI
	return func() tea.Msg {
		if playlistsAPI == nil {
			return panes.PlaylistReorderResultMsg{}
		}
		err := playlistsAPI.ReorderPlaylistTracks(context.Background(), playlistID, rangeStart, insertBefore, rangeLength)
		return panes.PlaylistReorderResultMsg{Err: err}
	}
}
