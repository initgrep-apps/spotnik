// Package app contains the root Bubble Tea model that wires together all
// panes, the central store, and the active theme. It is the single entry
// point for the TUI application.
package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	viewAuth                      // Auth panel shown when user needs to authenticate
	viewStats                     // Stats dashboard (press 2 to open, 1 to return)
	viewPlaylists                 // Playlist Manager (press 3 to open, 1 to return)
)

const (
	// playbackFetchInterval is how many ticks (seconds) between playback state polls.
	playbackFetchInterval = 3
	// queueFetchInterval is how many ticks (seconds) between queue polls.
	queueFetchInterval = 9
	// defaultBackoffTicks is how many ticks to pause polling after a 429 rate limit.
	defaultBackoffTicks = 10
)

// App is the root application model. It owns the active theme, the central
// store, the API clients, and all pane models. It is the ONLY layer that
// calls the Spotify API — panes emit request messages and app.go dispatches them.
type App struct {
	theme   theme.Theme
	store   *state.Store
	player  api.PlayerAPI
	library api.LibraryAPI
	search  api.SearchAPI
	devices api.DevicesAPI
	userAPI api.UserAPI

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
	playlistsAPI api.PlaylistsAPI

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

	// tickCount increments every 1s tick. Used to throttle API polling:
	// playback fetched every playbackFetchInterval ticks, queue every queueFetchInterval ticks.
	tickCount int

	// backoffTicks is the number of ticks to skip all API fetches after a 429 rate limit.
	// Decremented each tick; when >0 no polling commands are dispatched.
	backoffTicks int

	// volumeStep is the percentage change per volume up/down keypress.
	volumeStep int

	// needsAuth is true when the user is not authenticated and must go through the auth flow.
	needsAuth bool

	// clientID is the Spotify OAuth client ID, needed for the TUI auth flow.
	clientID string

	// tokenStore is the keychain token store, needed for the TUI auth flow.
	tokenStore keychain.TokenStore

	// authURL is the Spotify authorization URL shown in the auth panel.
	authURL string

	// authStatus is the status message shown in the auth panel.
	authStatus string
}

// statusDismissMsg is sent after 4 seconds to clear a transient status bar message.
type statusDismissMsg struct{}

// splashDismissMsg is sent after 2 seconds to close the splash screen.
type splashDismissMsg struct{}

// AppOptions carries optional startup configuration into the app.
// Zero value means the user is already authenticated and no auth flow is needed.
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
func (a *App) SetPlayer(player api.PlayerAPI) {
	a.player = player
}

// SetLibrary injects the Spotify API library client into the app.
func (a *App) SetLibrary(library api.LibraryAPI) {
	a.library = library
}

// SetSearch injects the Spotify API search client into the app.
func (a *App) SetSearch(search api.SearchAPI) {
	a.search = search
}

// SetDevices injects the Spotify Connect devices API client into the app.
func (a *App) SetDevices(devices api.DevicesAPI) {
	a.devices = devices
}

// SetUserAPI injects the Spotify user stats API client into the app.
func (a *App) SetUserAPI(userAPI api.UserAPI) {
	a.userAPI = userAPI
}

// SetPlaylistsAPI injects the Spotify playlists mutation client into the app.
func (a *App) SetPlaylistsAPI(p api.PlaylistsAPI) {
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

// AuthViewOpen returns true while the auth panel is the active view.
func (a *App) AuthViewOpen() bool {
	return a.currentView == viewAuth
}

// TickCount returns the current tick counter (exported for testing).
func (a *App) TickCount() int {
	return a.tickCount
}

// BackoffTicks returns the remaining backoff ticks (exported for testing).
func (a *App) BackoffTicks() int {
	return a.backoffTicks
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

// Init starts the splash timer. If the user is already authenticated,
// it also starts data fetching and the polling loop. If not, those are
// deferred until auth succeeds.
func (a *App) Init() tea.Cmd {
	splashTimer := tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
		return splashDismissMsg{}
	})

	if a.needsAuth {
		// Unauthenticated: only show splash, defer everything else.
		return splashTimer
	}

	// Authenticated: start data fetching alongside splash.
	return tea.Batch(
		fetchPlaybackStateCmd(a.player, a.store),
		a.libraryPane.Init(),
		tea.Tick(time.Second, func(_ time.Time) tea.Msg {
			return panes.TickMsg{}
		}),
		splashTimer,
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
	a.focus = a.prevFocus
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
	a.focus = a.prevFocus
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
			if a.needsAuth {
				a.currentView = viewAuth
				a.authStatus = "Opening browser for authorization..."
				return a, prepareAuthCmd(a.clientID)
			}
			a.currentView = viewMain
		}
		return a, nil

	case authPreparedMsg:
		a.authURL = m.authURL
		if m.browserErr != nil {
			a.authStatus = "Could not open browser. Please visit the URL above."
		} else {
			a.authStatus = "Waiting for authorization in browser..."
		}
		return a, waitForCallbackCmd(a.clientID, a.tokenStore, m.verifier, m.redirectURI, m.codeCh, m.serverClose)

	case authSuccessMsg:
		a.needsAuth = false
		a.currentView = viewMain
		a.initAPIClients(m.accessToken)
		// Start data fetching and tick loop.
		return a, tea.Batch(
			fetchPlaybackStateCmd(a.player, a.store),
			a.libraryPane.Init(),
			tea.Tick(time.Second, func(_ time.Time) tea.Msg {
				return panes.TickMsg{}
			}),
		)

	case authErrorMsg:
		a.authStatus = fmt.Sprintf("Error: %s — press q to quit", m.err.Error())
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
		return a.handleKeyMsg(m)

	case panes.TickMsg:
		// Always forward to playerPane for progress bar animation.
		updatedPane, _ := a.playerPane.Update(m)
		if pp, ok := updatedPane.(*panes.PlayerPane); ok {
			a.playerPane = pp
		}

		nextTick := tea.Tick(time.Second, func(_ time.Time) tea.Msg {
			return panes.TickMsg{}
		})

		// During backoff, skip all API fetches.
		if a.backoffTicks > 0 {
			a.backoffTicks--
			a.tickCount++
			// When backoff just expired, force an immediate fetch.
			if a.backoffTicks == 0 {
				a.tickCount = 0
				return a, tea.Batch(
					nextTick,
					fetchPlaybackStateCmd(a.player, a.store),
					fetchQueueCmd(a.player, a.store),
				)
			}
			return a, nextTick
		}

		// Throttled polling: playback every 3s, queue every 9s.
		// Check before incrementing so tick 0 fires both fetches immediately.
		var cmds []tea.Cmd
		cmds = append(cmds, nextTick)
		if a.tickCount%playbackFetchInterval == 0 {
			cmds = append(cmds, fetchPlaybackStateCmd(a.player, a.store))
		}
		if a.tickCount%queueFetchInterval == 0 {
			cmds = append(cmds, fetchQueueCmd(a.player, a.store))
		}
		a.tickCount++
		return a, tea.Batch(cmds...)

	case panes.RateLimitedMsg:
		// 429 from Spotify — activate backoff and show status message.
		backoff := m.RetryAfterSecs
		if backoff < defaultBackoffTicks {
			backoff = defaultBackoffTicks
		}
		a.backoffTicks = backoff
		a.statusMsg = fmt.Sprintf("Rate limited — pausing requests for %ds", backoff)
		return a, tea.Tick(4*time.Second, func(_ time.Time) tea.Msg { return statusDismissMsg{} })

	case panes.QueueLoadedMsg:
		// Queue data is already in the store — no pane-specific handling needed here.
		// QueuePane reads directly from store on View().
		return a, nil

	case panes.PlaybackStateFetchedMsg:
		// Data fetched during splash is stored but splash stays visible
		// for the full 5s duration — the splashDismissMsg timer handles transition.
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
	if model, cmd, handled := a.routePlaylistMsg(msg); handled {
		return model, cmd
	}

	// Forward unhandled messages to the library pane (notification msgs, etc.).
	updated, cmd := a.libraryPane.Update(msg)
	if lp, ok := updated.(*panes.LibraryPane); ok {
		a.libraryPane = lp
	}
	return a, cmd
}
