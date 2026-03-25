// Package app contains the root Bubble Tea model that wires together all
// panes, the central store, and the active theme. It is the single entry
// point for the TUI application.
package app

import (
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"go.dalton.dog/bubbleup"
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
	// defaultBackoffTicks is how many ticks to pause polling after a 429 rate limit.
	defaultBackoffTicks = 10

	// idleThresholdSecs is the number of seconds without a KeyMsg before the app
	// is considered idle. Polling intervals increase when idle.
	idleThresholdSecs = 60

	// Polling intervals (in ticks = seconds) for the 4-state matrix.
	// Active + playing: full speed.
	activePlayingPlaybackInterval = 3
	activePlayingQueueInterval    = 9
	// Active + paused OR idle + playing: reduced speed.
	reducedPlaybackInterval = 10
	reducedQueueInterval    = 30
	// Idle + paused: slowest speed.
	idlePlaybackInterval = 30
	idleQueueInterval    = 60
)

// App is the root application model. It owns the active theme, the central
// store, the API clients, and all pane models. It is the ONLY layer that
// calls the Spotify API — panes emit request messages and app.go dispatches them.
type App struct {
	theme  theme.Theme
	store  *state.Store
	alerts bubbleup.AlertModel // BubbleUp toast notification model
	// NOTE: alerts.Render(content) must be called in View() — never alerts.View().
	// BubbleUp's View() returns empty string by design; Render() overlays alerts.
	gateway *api.Gateway // centralized HTTP gateway shared across all API clients
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

	// tickCount increments every 1s tick. Used to throttle API polling:
	// intervals are computed dynamically by pollIntervals() based on idle state and playback.
	tickCount int

	// lastInteraction is the last time a tea.KeyMsg was received.
	// Used to determine idle state for polling backoff.
	lastInteraction time.Time

	// idleThreshold is how long without input before the app is considered idle.
	idleThreshold time.Duration

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

	// tokenBaseURL overrides the Spotify token endpoint for tests.
	// Empty string means use the real production Spotify endpoint.
	tokenBaseURL string

	// authURL is the Spotify authorization URL shown in the auth panel.
	authURL string

	// authStatus is the status message shown in the auth panel.
	authStatus string
}

// throttleExpiredMsg is sent when the 429 backoff period has elapsed.
// It clears the throttle observability state in the store.
type throttleExpiredMsg struct{}

// splashDismissMsg is sent after 2 seconds to close the splash screen.
type splashDismissMsg struct{}

// unauthorizedMsg is sent by any build*Cmd or fetch*Cmd when the Spotify API returns 401.
// The app handles it by attempting a token refresh.
type unauthorizedMsg struct{}

// tokenRefreshedMsg is sent when a token refresh attempt completes.
// newToken is non-empty on success; err is non-nil on failure.
type tokenRefreshedMsg struct {
	newToken string
	err      error
}

// AppOptions carries optional startup configuration into the app.
// Zero value means the user is already authenticated and no auth flow is needed.
type AppOptions struct {
	NeedsAuth  bool
	ClientID   string
	TokenStore keychain.TokenStore
	// TokenBaseURL overrides the Spotify token endpoint for tests.
	// Leave empty for production (uses the real Spotify endpoint).
	TokenBaseURL string
}

// New creates a new App, loading the theme from cfg.UI.Theme.
func New(cfg *config.Config, opts AppOptions) *App {
	t := theme.Load(cfg.UI.Theme)
	s := state.New()
	gw := api.NewGateway()

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
		theme:           t,
		store:           s,
		alerts:          *components.NewNotifications(t),
		gateway:         gw,
		playerPane:      playerPane,
		libraryPane:     libraryPane,
		queuePane:       queuePane,
		searchPane:      searchPane,
		devicePane:      devicePane,
		focus:           focusPlayer,
		currentView:     viewSplash,
		volumeStep:      volStep,
		needsAuth:       opts.NeedsAuth,
		clientID:        opts.ClientID,
		tokenStore:      opts.TokenStore,
		tokenBaseURL:    opts.TokenBaseURL,
		lastInteraction: time.Now(),
		idleThreshold:   idleThresholdSecs * time.Second,
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

// IsIdle returns true if no user input has been received within idleThreshold.
// Exported for testing.
func (a *App) IsIdle() bool {
	return a.isIdle()
}

// SetLastInteraction sets the lastInteraction timestamp (exported for testing).
func (a *App) SetLastInteraction(t time.Time) {
	a.lastInteraction = t
}

// PollIntervals returns the current playback and queue polling intervals based
// on user activity and playback state. Exported for testing.
func (a *App) PollIntervals() (playbackInterval, queueInterval int) {
	return a.pollIntervals()
}

// isIdle returns true if no user input has been received within idleThreshold.
func (a *App) isIdle() bool {
	return time.Since(a.lastInteraction) > a.idleThreshold
}

// pollIntervals returns the current playback and queue polling intervals
// based on user activity and playback state.
//
// Four-state matrix:
//
//	Active + Playing  →  3s / 9s   (full speed)
//	Active + Paused   → 10s / 30s  (reduced)
//	Idle   + Playing  → 10s / 30s  (reduced)
//	Idle   + Paused   → 30s / 60s  (slowest)
func (a *App) pollIntervals() (playbackInterval, queueInterval int) {
	idle := a.isIdle()
	playing := false
	if ps := a.store.PlaybackState(); ps != nil {
		playing = ps.IsPlaying
	}

	switch {
	case !idle && playing:
		return activePlayingPlaybackInterval, activePlayingQueueInterval
	case !idle && !playing:
		return reducedPlaybackInterval, reducedQueueInterval
	case idle && playing:
		return reducedPlaybackInterval, reducedQueueInterval
	default: // idle && !playing
		return idlePlaybackInterval, idleQueueInterval
	}
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

	// NOTE: a.alerts.Init() returns nil by design — BubbleUp starts its
	// internal tick only when an alert is triggered, not at startup.
	_ = a.alerts.Init()

	if a.needsAuth {
		// Unauthenticated: only show splash, defer everything else.
		return splashTimer
	}

	// Authenticated: start data fetching alongside splash.
	return tea.Batch(
		fetchPlaybackStateCmd(a.player),
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
// It delegates to handleMsg for application logic, then forwards the message to
// the BubbleUp alerts model so alert lifecycle timers (auto-dismiss) keep running.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, mainCmd := a.handleMsg(msg)

	// Forward message to BubbleUp alerts for lifecycle management.
	// This ensures the auto-dismiss timer keeps ticking while any alert is active,
	// regardless of which other message is being processed.
	updatedAlerts, alertCmd := a.alerts.Update(msg)
	if am, ok := updatedAlerts.(bubbleup.AlertModel); ok {
		a.alerts = am
	}

	return model, tea.Batch(mainCmd, alertCmd)
}

// handleMsg contains the core application message routing logic.
// It is called by Update() which also forwards messages to the alerts model.
func (a *App) handleMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			fetchPlaybackStateCmd(a.player),
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
		// Debounce fired — set store state here (in Update) before dispatching.
		// Store writes belong in Update, not inside command builders.
		a.store.SetSearchQuery(m.Query)
		a.store.SetSearchLoading(true)
		return a, a.buildSearchCmd(m.Query)

	case panes.SearchClearedMsg:
		// SearchOverlay emitted this when the user pressed Ctrl+U.
		// Clear search state in store — store writes belong in Update, not in panes.
		// NOTE: store.SetSearchResults(nil) is kept for symmetry with SetSearchQuery;
		// store.SearchResults() is not used in production rendering (overlay uses o.results).
		a.store.SetSearchResults(nil)
		a.store.SetSearchQuery("")
		return a, nil

	case panes.SearchResultsMsg:
		// Search command returned — write error state to store, then deliver results to overlay.
		// NOTE: SearchResultsMsg.Results is a UI-adapted *panes.SearchResultData, not the raw
		// *api.SearchResult stored in store.SearchResults(). The overlay stores results in its
		// own model field (o.results) from the Msg payload; store.SearchResults() is not used
		// in production rendering and can be ignored here.
		a.store.SetSearchLoading(false)
		if m.Err != nil {
			a.store.SetSearchError(m.Err)
			// Route search error through toast; search overlay shows loading→empty (not error).
			updated, _ := a.searchPane.Update(m)
			if sp, ok := updated.(*panes.SearchOverlay); ok {
				a.searchPane = sp
			}
			return a, a.alerts.NewAlertCmd("error", fmt.Sprintf("Search failed: %s", m.Err.Error()))
		}
		a.store.ClearSearchError()
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
		// Skip fetch if the store already has fresh stats for this range (within StatsTTL).
		if !a.store.StatsStale(m.TimeRange) {
			return a, nil
		}
		return a, a.buildFetchStatsCmd(m.TimeRange)

	case panes.StatsLoadedMsg:
		// Stats data fetched — write to store, then forward to stats pane.
		if m.Err != nil {
			a.store.SetStatsError(m.Err)
			if a.statsPane != nil {
				updated, _ := a.statsPane.Update(m)
				if sv, ok := updated.(*panes.StatsView); ok {
					a.statsPane = sv
				}
			}
			return a, a.alerts.NewAlertCmd("error", "Failed to load stats. Press f to retry")
		}
		a.store.ClearStatsError()
		if m.TimeRange != "" {
			a.store.SetTopTracks(m.TimeRange, m.TopTracks)
			a.store.SetTopArtists(m.TimeRange, m.TopArtists)
		}
		if a.statsPane != nil {
			updated, cmd := a.statsPane.Update(m)
			if sv, ok := updated.(*panes.StatsView); ok {
				a.statsPane = sv
			}
			return a, cmd
		}
		return a, nil

	case tea.KeyMsg:
		wasIdle := a.isIdle()
		a.lastInteraction = time.Now()
		if wasIdle {
			// User returned from idle — force immediate poll on the next tick.
			a.tickCount = 0
		}
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
					fetchPlaybackStateCmd(a.player),
					fetchQueueCmd(a.player),
				)
			}
			return a, nextTick
		}

		// Adaptive polling: intervals computed dynamically based on idle state and playback.
		// Check before incrementing so tick 0 fires both fetches immediately.
		playbackInterval, queueInterval := a.pollIntervals()
		var cmds []tea.Cmd
		cmds = append(cmds, nextTick)
		if a.tickCount%playbackInterval == 0 {
			cmds = append(cmds, fetchPlaybackStateCmd(a.player))
		}
		if a.tickCount%queueInterval == 0 {
			cmds = append(cmds, fetchQueueCmd(a.player))
		}
		a.tickCount++
		return a, tea.Batch(cmds...)

	case panes.RateLimitedMsg:
		// 429 from Spotify — activate backoff and emit a ratelimit toast.
		backoff := m.RetryAfterSecs
		if backoff < defaultBackoffTicks {
			backoff = defaultBackoffTicks
		}
		a.backoffTicks = backoff
		// Update store throttle observability so UI components can read gateway state.
		a.store.SetThrottle(true, m.RetryAfterSecs, time.Now())
		return a, tea.Batch(
			a.alerts.NewAlertCmd("ratelimit", fmt.Sprintf("Rate limited, retrying in %ds", backoff)),
			tea.Tick(time.Duration(backoff)*time.Second, func(_ time.Time) tea.Msg { return throttleExpiredMsg{} }),
		)

	case unauthorizedMsg:
		// 401 from any Spotify API call — attempt a token refresh.
		// If tokenStore is nil or has no refresh token, skip to show expired message.
		return a, buildRefreshTokenCmd(a.tokenStore, a.clientID, a.tokenBaseURL)

	case tokenRefreshedMsg:
		if m.err != nil {
			// Refresh failed — user must re-authenticate manually.
			return a, a.alerts.NewAlertCmd("error", "Session expired. Run: spotnik auth")
		}
		// Refresh succeeded — re-initialize all API clients with the new token.
		a.initAPIClients(m.newToken)
		return a, nil

	case panes.QueueLoadedMsg:
		// Write queue data to store from Msg payload (Elm Architecture: only Update writes store).
		if m.Err != nil {
			a.store.SetQueueError(m.Err)
			return a, a.alerts.NewAlertCmd("error", "Queue update failed")
		}
		a.store.ClearQueueError()
		a.store.SetQueue(m.Tracks)
		// QueuePane reads directly from store on View().
		return a, nil

	case panes.PlaybackStateFetchedMsg:
		// Write state to store from Msg payload (Elm Architecture: only Update writes store).
		// Only write to store when State is non-nil — nil State means either:
		//   a) player == nil (no API client injected), or
		//   b) a transient error (m.Err != nil).
		// In both cases we leave the existing store state unchanged.
		if m.State != nil {
			a.store.SetPlaybackState(m.State)
		}
		// Data fetched during splash is stored but splash stays visible
		// for the full 5s duration — the splashDismissMsg timer handles transition.
		updatedPane, cmd := a.playerPane.Update(m)
		if pp, ok := updatedPane.(*panes.PlayerPane); ok {
			a.playerPane = pp
		}
		return a, cmd

	case panes.PlaybackCmdSentMsg:
		if m.Err != nil {
			var forbiddenErr *api.ForbiddenError
			if errors.As(m.Err, &forbiddenErr) {
				return a, tea.Batch(
					fetchPlaybackStateCmd(a.player),
					a.alerts.NewAlertCmd("warning", "Playback control not available on this device"),
				)
			}
			return a, tea.Batch(
				fetchPlaybackStateCmd(a.player),
				a.alerts.NewAlertCmd("error", m.Err.Error()),
			)
		}
		return a, fetchPlaybackStateCmd(a.player)

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
			var forbiddenErr *api.ForbiddenError
			if errors.As(m.Err, &forbiddenErr) {
				return a, a.alerts.NewAlertCmd("error", forbiddenErr.Message)
			}
			return a, a.alerts.NewAlertCmd("error", m.Err.Error())
		}
		if m.TrackName != "" {
			return a, a.alerts.NewAlertCmd("success", fmt.Sprintf("Added to queue: %s", m.TrackName))
		}
		return a, a.alerts.NewAlertCmd("success", "Added to queue")

	case panes.FetchPlaylistsRequestMsg:
		// Skip fetch if playlists are fresh and this is an initial (non-paginated) load.
		// Paginated requests (offset > 0) always proceed to avoid incomplete data.
		if m.Offset == 0 && !a.store.PlaylistsStale() {
			return a, nil
		}
		return a, a.buildFetchPlaylistsCmd(m.Offset)

	case panes.LibraryLoadedMsg:
		// Write playlist data to store from Msg payload (Elm Architecture: only Update writes store).
		if m.Err != nil {
			a.store.SetPlaylistsFetchError(m.Err)
			updated, _ := a.libraryPane.Update(m)
			if lp, ok := updated.(*panes.LibraryPane); ok {
				a.libraryPane = lp
			}
			return a, a.alerts.NewAlertCmd("error", "Failed to load playlists. Press Tab to retry")
		}
		a.store.ClearPlaylistsFetchError()
		if m.Offset == 0 {
			a.store.SetPlaylists(m.Items)
		} else {
			a.store.SetPlaylists(append(a.store.Playlists(), m.Items...))
		}
		a.store.SetPlaylistsTotal(len(a.store.Playlists()))
		// Forward to library pane so it can refresh from store.
		updated, cmd := a.libraryPane.Update(m)
		if lp, ok := updated.(*panes.LibraryPane); ok {
			a.libraryPane = lp
		}
		return a, cmd

	case panes.FetchAlbumsRequestMsg:
		// Skip fetch if albums are fresh and this is an initial (non-paginated) load.
		// Paginated requests (offset > 0) always proceed to avoid incomplete data.
		if m.Offset == 0 && !a.store.AlbumsStale() {
			return a, nil
		}
		return a, a.buildFetchAlbumsCmd(m.Offset)

	case panes.AlbumsLoadedMsg:
		// Write album data to store from Msg payload.
		if m.Err != nil {
			a.store.SetAlbumsFetchError(m.Err)
			updated, _ := a.libraryPane.Update(m)
			if lp, ok := updated.(*panes.LibraryPane); ok {
				a.libraryPane = lp
			}
			return a, a.alerts.NewAlertCmd("error", "Failed to load albums. Press Tab to retry")
		}
		a.store.ClearAlbumsFetchError()
		a.store.SetSavedAlbums(m.Items)
		// Forward to library pane.
		updated, cmd := a.libraryPane.Update(m)
		if lp, ok := updated.(*panes.LibraryPane); ok {
			a.libraryPane = lp
		}
		return a, cmd

	case panes.FetchLikedTracksRequestMsg:
		// Skip fetch if liked tracks are fresh and this is an initial (non-paginated) load.
		// Paginated requests (offset > 0) always proceed to avoid incomplete data.
		if m.Offset == 0 && !a.store.LikedTracksStale() {
			return a, nil
		}
		return a, a.buildFetchLikedTracksCmd(m.Offset)

	case panes.LikedTracksLoadedMsg:
		// Write liked tracks to store from Msg payload.
		if m.Err != nil {
			a.store.SetLikedTracksFetchError(m.Err)
			updated, _ := a.libraryPane.Update(m)
			if lp, ok := updated.(*panes.LibraryPane); ok {
				a.libraryPane = lp
			}
			return a, a.alerts.NewAlertCmd("error", "Failed to load liked tracks. Press Tab to retry")
		}
		a.store.ClearLikedTracksFetchError()
		a.store.SetLikedTracks(m.Items)
		a.store.SetLikedTotal(len(m.Items) + m.Offset)
		// Forward to library pane.
		updated, cmd := a.libraryPane.Update(m)
		if lp, ok := updated.(*panes.LibraryPane); ok {
			a.libraryPane = lp
		}
		return a, cmd

	case panes.FetchRecentlyPlayedRequestMsg:
		// Skip fetch if recently played data is fresh (within RecentlyPlayedTTL).
		if !a.store.RecentlyPlayedStale() {
			return a, nil
		}
		return a, a.buildFetchRecentlyPlayedCmd()

	case panes.RecentlyPlayedLoadedMsg:
		// Write recently played to store from Msg payload.
		if m.Err != nil {
			a.store.SetRecentPlayedFetchError(m.Err)
			updated, _ := a.libraryPane.Update(m)
			if lp, ok := updated.(*panes.LibraryPane); ok {
				a.libraryPane = lp
			}
			return a, a.alerts.NewAlertCmd("error", "Failed to load recently played. Press Tab to retry")
		}
		a.store.ClearRecentPlayedFetchError()
		a.store.SetRecentlyPlayed(m.Items)
		// Forward to library pane.
		updated, cmd := a.libraryPane.Update(m)
		if lp, ok := updated.(*panes.LibraryPane); ok {
			a.libraryPane = lp
		}
		return a, cmd

	case panes.LikeTrackRequestMsg:
		return a, a.buildToggleLikeCmd(m.TrackID, m.Unlike)

	case panes.LikeToggleResultMsg:
		// Like/unlike result — no action needed unless there's an error.
		if m.Err != nil {
			return a, a.alerts.NewAlertCmd("error", m.Err.Error())
		}
		return a, nil

	case panes.DeviceOverlayClosedMsg:
		// Device overlay closed via Esc — restore previous focus.
		return a.closeDeviceOverlay()

	case panes.FetchDevicesRequestMsg:
		// Device overlay is requesting the device list from the API.
		// Skip fetch if the device list is still within DevicesTTL (30s).
		if !a.store.DevicesStale() {
			return a, nil
		}
		return a, a.buildFetchDevicesCmd()

	case panes.TransferPlaybackMsg:
		// User selected a device; show info toast and dispatch transfer API call.
		a.deviceOverlayOpen = false
		return a, tea.Batch(
			a.buildTransferPlaybackCmd(m.DeviceID),
			a.alerts.NewAlertCmd("info", fmt.Sprintf("Switching to %s...", m.DeviceName)),
		)

	case panes.DevicesLoadErrorMsg:
		// Device fetch failed — show error toast so the user knows why the overlay is empty.
		return a, a.alerts.NewAlertCmd("error", fmt.Sprintf("Failed to load devices: %s", m.Err.Error()))

	case panes.DeviceTransferredMsg:
		if m.Err != nil {
			return a, tea.Batch(
				fetchPlaybackStateCmd(a.player),
				a.alerts.NewAlertCmd("error", m.Err.Error()),
			)
		}
		// Transfer succeeded — next poll will update the header.
		return a, fetchPlaybackStateCmd(a.player)

	case throttleExpiredMsg:
		// Clear throttle state in the store once the backoff period expires.
		a.store.SetThrottle(false, 0, time.Time{})
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
