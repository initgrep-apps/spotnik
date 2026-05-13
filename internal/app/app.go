// Package app contains the root Bubble Tea model that wires together all
// panes, the central store, and the active theme. It is the single entry
// point for the TUI application.
package app

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/prefs"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"go.dalton.dog/bubbleup"
)

// viewMode identifies which top-level view is currently active.
type viewMode int

const (
	viewSplash     viewMode = iota // Splash screen shown on startup
	viewOnboarding                 // Registration + OAuth flow (covers first-launch and returning users)
	viewGrid                       // Grid layout: 10 panes across 2 pages, managed by LayoutManager
)

// Onboarding sub-step constants track which screen within viewOnboarding is active.
const (
	stepRegister = iota // Step 1: client ID input + instructions
	stepOAuth           // Step 2: browser wait + full URL display
	stepError           // Step 2 error: retry options
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
	// toasts wraps alerts to provide the typed Toast API. It holds a pointer to the
	// alerts field so it automatically reflects theme-switch re-assignments.
	toasts      *uikit.ToastManager
	errorMapper *uikit.ErrorMapper // translates API errors to user-friendly Toast values
	gateway     *api.Gateway       // centralized HTTP gateway shared across all API clients
	player      api.PlayerAPI
	library     api.LibraryAPI
	search      api.SearchAPI
	devices     api.DevicesAPI
	userAPI     api.UserAPI

	// layout manages the grid, focus, preset, and page system.
	layout *layout.Manager

	// statusKeyMap holds the keybindings for the bottom status bar.
	// activePage is set per render call via a copy so renderStatusBar() stays pure.
	statusKeyMap appKeyMap

	// panes holds all 8 Page A panes (Page B panes added in Feature 51).
	panes map[layout.PaneID]layout.Pane

	// searchPane and devicePane are overlay panes — they float above the grid.
	searchPane *panes.SearchOverlay
	devicePane *panes.DeviceOverlay

	// profilePane is the floating user profile overlay. Populated at startup.
	profilePane *panes.ProfileOverlay

	// themeOverlay is the floating theme switcher overlay. Populated when open.
	themeOverlay *panes.ThemeOverlay

	// playlistsAPI is the Spotify playlists mutation client.
	playlistsAPI api.PlaylistsAPI

	// Playlist track sub-view: interactive fetch state (mirrors searchCancel/searchQuery).
	// playlistTracksCancel cancels the active playlist tracks fetch; always safe to call.
	// playlistTracksID is the staleness key: ID of the playlist currently being fetched.
	playlistTracksCancel context.CancelFunc
	playlistTracksID     string

	// Album track sub-view: interactive fetch state (mirrors playlistTracksCancel pattern).
	// albumTracksCancel cancels the active album tracks fetch; always safe to call.
	// albumTracksID is the staleness key: ID of the album currently being fetched.
	albumTracksCancel context.CancelFunc
	albumTracksID     string

	// currentView tracks which top-level view is displayed.
	currentView viewMode

	width  int
	height int

	// searchOpen is true while the search overlay is visible.
	searchOpen bool

	// searchQuery is the staleness key for the current in-flight search.
	// Reset to "" when closeSearch() is called.
	searchQuery string

	// searchPage is the staleness key for the current in-flight page.
	// Reset to 0 when closeSearch() is called.
	searchPage int

	// searchLoading is true while a search HTTP call is in-flight.
	// Reset to false on close, error, or successful result delivery.
	searchLoading bool

	// searchCancel cancels the in-flight search HTTP request context.
	// Initialized to func(){} in New() so it is always safe to call.
	searchCancel context.CancelFunc

	// searchCtx is the context for the current in-flight search request.
	// Exposed via SearchCancelCtx() for tests. Nil when no search is in-flight.
	searchCtx context.Context

	// deviceOverlayOpen is true while the device switcher overlay is visible.
	deviceOverlayOpen bool

	// profileOverlayOpen is true while the user profile overlay is visible.
	profileOverlayOpen bool

	// showThemeSwitcher is true while the theme switcher overlay is visible.
	showThemeSwitcher bool

	// helpOpen is true while the help keybinding overlay is visible.
	helpOpen bool
	// helpOverlay is the floating help overlay. Populated when open.
	helpOverlay *panes.HelpOverlay

	// onboardingPermissionsOverlay is the floating permissions notice shown on
	// Step 2 of onboarding when the user presses 'v'. Nil when closed.
	onboardingPermissionsOverlay *panes.OnboardingPermissionsOverlay

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

	// needsAuth is true when the user is not authenticated and must go through the auth flow.
	needsAuth bool

	// needsRegister is true on first launch when no client_id is present in config.
	// The TUI shows viewOnboarding (stepRegister) after the splash screen.
	needsRegister bool

	// onboardingStep tracks the active sub-step within viewOnboarding.
	onboardingStep int

	// onboardingField is the FormField primitive for the client ID entry field.
	// It wraps bubbles/textinput with an intrinsic 32-char hex validator and an
	// error slot that is rendered beneath the input.
	onboardingField *uikit.FormField

	// onboardingError is the error message shown on the onboarding error screen (stepError).
	onboardingError string

	// onboardingPort is the port the OAuth callback server is already listening on.
	onboardingPort int

	// onboardingCodeCh receives the OAuth authorization code from the callback server.
	onboardingCodeCh <-chan api.CallbackResult

	// onboardingClose closes the callback server. Always safe to call (defaulted to no-op).
	onboardingClose func()

	// onboardingAuthURL is the full Spotify OAuth authorization URL, shown on stepOAuth.
	onboardingAuthURL string

	// onboardingSpinner is the TUI spinner shown while waiting for the OAuth callback.
	// Using *uikit.Spinner so Done/Fail/Cancel terminal states are available.
	onboardingSpinner *uikit.Spinner

	// clientID is the Spotify OAuth client ID, needed for the TUI auth flow.
	clientID string

	// tokenStore is the keychain token store, needed for the TUI auth flow.
	tokenStore keychain.TokenStore

	// tokenBaseURL overrides the Spotify token endpoint for tests.
	// Empty string means use the real production Spotify endpoint.
	tokenBaseURL string

	// version is the build-time injected version string (e.g. "v0.1.0").
	// Set from AppOptions.Version; displayed on the splash screen.
	version string

	// consecutivePlaybackErrors counts successive PlaybackStateFetchedMsg deliveries
	// where Err is non-nil. A toast is emitted when this reaches 5, then the counter
	// continues to increment (so exactly the 5th triggers the toast, not subsequent ones).
	// The counter resets to 0 on any successful fetch.
	consecutivePlaybackErrors int

	// Per-pane polling health — each pane tracks backoff and first-load status
	// independently from the global 429 backoff (a.backoffTicks).
	playlistsPoll    pollState
	albumsPoll       pollState
	likedSongsPoll   pollState
	recentPlayedPoll pollState
	statsPoll        pollState
	devicesPoll      pollState

	// nilPlaybackStateTicks counts successive PlaybackStateFetchedMsg deliveries where
	// State is nil and Err is nil. After 30 consecutive nil states a warning toast fires
	// once to surface a possible stuck/disconnected state. The counter resets to 0 when
	// a non-nil State is received.
	nilPlaybackStateTicks int

	// prefs is the preference store that coalesces in-memory changes and writes
	// them to disk via FlushCmd. All runtime preference changes go through here.
	prefs *prefs.PreferenceStore

	// prefsDirtyGen is incremented on every schedulePrefsFlush call. Used by the
	// debounce timer: a tick whose generation is less than prefsDirtyGen is stale
	// and skipped.
	prefsDirtyGen int
}

// throttleExpiredMsg is sent when the 429 backoff period has elapsed.
// It clears the throttle observability state in the store.
type throttleExpiredMsg struct{}

// splashDismissMsg is sent after 3 seconds to close the splash screen.
type splashDismissMsg struct{}

// unauthorizedMsg is sent by any build*Cmd or fetch*Cmd when the Spotify API returns 401.
// The app handles it by attempting a token refresh.
type unauthorizedMsg struct{}

// userProfileLoadedMsg is sent when the initial GET /v1/me fetch completes.
// profile carries the full authenticated user profile; err is non-nil on failure.
type userProfileLoadedMsg struct {
	profile domain.UserProfile
	err     error
}

// tokenRefreshedMsg is sent when a token refresh attempt completes.
// newToken is non-empty on success; err is non-nil on failure.
type tokenRefreshedMsg struct {
	newToken string
	err      error
}

// AppOptions carries optional startup configuration into the app.
// Zero value means the user is already authenticated and no auth flow is needed.
type AppOptions struct {
	// NeedsRegister is true when no client_id is present in config.
	// The TUI will show the onboarding flow (stepRegister) on first launch.
	NeedsRegister bool
	NeedsAuth     bool
	ClientID      string
	TokenStore    keychain.TokenStore
	// TokenBaseURL overrides the Spotify token endpoint for tests.
	// Leave empty for production (uses the real Spotify endpoint).
	TokenBaseURL string
	// Version is the build-time injected version string (e.g. "v0.1.0").
	// Falls back to "dev" when not provided.
	Version string
	// CallbackPort is the port the OAuth callback server is listening on.
	// Non-zero when the server was started early (needsRegister || needsAuth).
	CallbackPort int
	// CallbackCodeCh receives the OAuth authorization code from the callback server.
	// Non-nil when the server was started early.
	CallbackCodeCh <-chan api.CallbackResult
	// CallbackClose closes the callback server. Non-nil when started early.
	CallbackClose func()
}

// New creates a new App, loading the theme from cfg.Preferences.Theme.
func New(cfg *config.Config, opts AppOptions) *App {
	t := theme.Load(cfg.Preferences.Theme)
	s := state.New()
	gw := api.NewGateway()

	// Create all 8 Page A panes.
	nowPlayingPane := panes.NewNowPlayingPane(s, t, true)
	queuePane := panes.NewQueuePane(s, t, false)
	playlistsPane := panes.NewPlaylistsPane(s, t, false)
	albumsPane := panes.NewAlbumsPane(s, t, false)
	likedSongsPane := panes.NewLikedSongsPane(s, t, false)
	recentlyPlayedPane := panes.NewRecentlyPlayedPane(s, t, false)
	topTracksPane := panes.NewTopTracksPane(s, t, false)
	topArtistsPane := panes.NewTopArtistsPane(s, t, false)

	// Create Page B panes. All three read gateway events from the store's event
	// journal, preserving the ui/ → state/ dependency direction (no gateway reference).
	gatewayHealthPane := panes.NewGatewayHealthPane(s, t)
	pollingTrafficPane := panes.NewPollingTrafficPane(s, t)
	gatewayLivePane := panes.NewGatewayLivePane(s, t)
	networkLogPane := panes.NewNetworkLogPane(s, t)

	panesMap := map[layout.PaneID]layout.Pane{
		layout.PaneNowPlaying:     nowPlayingPane,
		layout.PaneQueue:          queuePane,
		layout.PanePlaylists:      playlistsPane,
		layout.PaneAlbums:         albumsPane,
		layout.PaneLikedSongs:     likedSongsPane,
		layout.PaneRecentlyPlayed: recentlyPlayedPane,
		layout.PaneTopTracks:      topTracksPane,
		layout.PaneTopArtists:     topArtistsPane,
		layout.PaneGatewayHealth:  gatewayHealthPane,
		layout.PanePollingTraffic: pollingTrafficPane,
		layout.PaneGatewayLive:    gatewayLivePane,
		layout.PaneNetworkLog:     networkLogPane,
	}

	searchPane := panes.NewSearchOverlay(t)
	devicePane := panes.NewDeviceOverlay(s, t)
	profilePane := panes.NewProfileOverlay(s, t)

	mgr := layout.NewManager()

	ver := opts.Version
	if ver == "" {
		ver = "dev"
	}

	// Initialise the FormField for the onboarding client ID registration step.
	// The intrinsic validator enforces the 32-char hex Spotify Client ID shape.
	ff := uikit.NewFormField(uikit.FormFieldConfig{
		Label:       "Client ID",
		Placeholder: "your-client-id-here",
		Validate:    config.ValidateClientID,
		Theme:       t,
	})
	ff.Focus()

	// Initialise the spinner shown while waiting for OAuth callback on stepOAuth.
	sp := uikit.NewSpinner("Waiting for authorization...  (times out in 5 minutes)", t)

	// Default onboardingClose to a no-op so it is always safe to call,
	// even when no callback server was started (e.g. in tests).
	callbackClose := opts.CallbackClose
	if callbackClose == nil {
		callbackClose = func() {}
	}

	alertModel := *components.NewNotifications(t)
	a := &App{
		theme:             t,
		store:             s,
		alerts:            alertModel,
		gateway:           gw,
		layout:            mgr,
		panes:             panesMap,
		searchPane:        searchPane,
		devicePane:        devicePane,
		profilePane:       profilePane,
		statusKeyMap:      newAppKeyMap(),
		currentView:       viewSplash,
		needsAuth:         opts.NeedsAuth,
		needsRegister:     opts.NeedsRegister,
		onboardingField:   ff,
		onboardingSpinner: sp,
		onboardingPort:    opts.CallbackPort,
		onboardingCodeCh:  opts.CallbackCodeCh,
		onboardingClose:   callbackClose,
		clientID:          opts.ClientID,
		tokenStore:        opts.TokenStore,
		tokenBaseURL:      opts.TokenBaseURL,
		version:           ver,
		lastInteraction:   time.Now(),
		idleThreshold:     idleThresholdSecs * time.Second,
		prefs:             prefs.New(config.DefaultConfigPath()),
		// searchCancel must never be nil; initialize to a no-op so it is always safe to call.
		searchCancel: func() {},
		// playlistTracksCancel must never be nil; initialize to a no-op.
		playlistTracksCancel: func() {},
		// albumTracksCancel must never be nil; initialize to a no-op.
		albumTracksCancel: func() {},
	}

	// Wire the typed Toast API. ToastManager holds &a.alerts so theme-switch
	// re-assignments to a.alerts are automatically reflected.
	a.toasts = uikit.NewToastManager(&a.alerts)
	// Wire the centralized error mapper — translates API errors to user-facing Toasts.
	a.errorMapper = &uikit.ErrorMapper{}

	// Apply saved layout preset (Page A only).
	// SetPreset is a no-op for out-of-range indices; log a warning when it doesn't take.
	if cfg.Preferences.Preset > 0 {
		a.layout.SetPreset(cfg.Preferences.Preset)
		if a.layout.ActivePresetIndex() != cfg.Preferences.Preset {
			fmt.Fprintf(os.Stderr, "spotnik: warning: saved preset %d is out of range, using default\n", cfg.Preferences.Preset)
		}
		a.propagateSizes()
		a.syncFocus()
	}

	// Apply saved visualizer pattern.
	// SetVisualizerPattern delegates to SetPattern which wraps out-of-range values.
	if cfg.Preferences.Visualizer > 0 {
		if np := a.nowPlayingPane(); np != nil {
			np.SetVisualizerPattern(cfg.Preferences.Visualizer)
		}
	}

	return a
}

// Theme returns the active theme instance.
func (a *App) Theme() theme.Theme {
	return a.theme
}

// Store returns the central state store.
func (a *App) Store() *state.Store {
	return a.store
}

// Layout returns the layout manager, exposed for testing.
func (a *App) Layout() *layout.Manager {
	return a.layout
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

// SetUserAPI injects the Spotify user identity and statistics API client into the app.
func (a *App) SetUserAPI(userAPI api.UserAPI) {
	a.userAPI = userAPI
}

// SetPlaylistsAPI injects the Spotify playlists mutation client into the app.
func (a *App) SetPlaylistsAPI(p api.PlaylistsAPI) {
	a.playlistsAPI = p
}

// GridViewOpen returns true while the grid view is the active top-level view.
func (a *App) GridViewOpen() bool {
	return a.currentView == viewGrid
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

// FocusedPane returns the PaneID of the pane that currently has keyboard focus.
// Returns layout.PaneID(-1) if no pane is focused.
func (a *App) FocusedPane() layout.PaneID {
	return a.layout.FocusedPane()
}

// NowPlayingFocused returns true if the NowPlaying pane currently has keyboard focus.
func (a *App) NowPlayingFocused() bool {
	return a.layout.FocusedPane() == layout.PaneNowPlaying
}

// QueueFocused returns true if the Queue pane currently has keyboard focus.
func (a *App) QueueFocused() bool {
	return a.layout.FocusedPane() == layout.PaneQueue
}

// PlaylistsFocused returns true if the Playlists pane currently has keyboard focus.
func (a *App) PlaylistsFocused() bool {
	return a.layout.FocusedPane() == layout.PanePlaylists
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

// pollState tracks per-pane polling health.
// Isolated from the global 429 backoff — a failed library fetch does not
// affect playback polling intervals.
type pollState struct {
	backoffTicks int // ticks remaining before next retry after an error
	// errorCount is incremented on each consecutive error and drives the
	// exponential backoff calculation in calcBackoffTicks. Reset to 0 on success.
	// Written by error handlers in Story 200; defined here so the type is complete.
	errorCount int  //nolint:unused
	hasData    bool // true after first successful load; switches interval regime
}

// libraryIntervals defines the polling cadence (seconds) for a library data type.
type libraryIntervals struct {
	playing, paused, idle int
}

var (
	recentPlayedIntervals = libraryIntervals{playing: 30, paused: 60, idle: 120}
	likedSongsIntervals   = libraryIntervals{playing: 60, paused: 120, idle: 300}
	playlistsIntervals    = libraryIntervals{playing: 60, paused: 120, idle: 300}
	albumsIntervals       = libraryIntervals{playing: 120, paused: 300, idle: 600}
	statsIntervals        = libraryIntervals{playing: 3600, paused: 3600, idle: 3600}
)

// calcBackoffTicks computes per-pane exponential backoff: min(5 * 2^(errorCount-1), 60).
func calcBackoffTicks(errorCount int) int {
	if ticks := 5 * (1 << uint(errorCount-1)); ticks < 60 {
		return ticks
	}
	return 60
}

// libraryInterval returns the polling interval in seconds for the given pane.
// Returns 5 if the pane has never loaded data (retry mode).
// Music playing → Playing interval regardless of user activity.
// Idle only applies when paused.
func (a *App) libraryInterval(p *pollState, iv libraryIntervals) int {
	if !p.hasData {
		return 5
	}
	state := a.store.PlaybackState()
	if state != nil && state.IsPlaying {
		return iv.playing
	}
	if a.isIdle() {
		return iv.idle
	}
	return iv.paused
}

// SearchOpen returns true while the search overlay is visible.
func (a *App) SearchOpen() bool {
	return a.searchOpen
}

// SearchLoading returns true while a search HTTP call is in-flight.
// Exported for tests.
func (a *App) SearchLoading() bool {
	return a.searchLoading
}

// SearchQuery returns the current search staleness key query.
// Exported for tests.
func (a *App) SearchQuery() string {
	return a.searchQuery
}

// SearchPage returns the current search staleness key page.
// Exported for tests.
func (a *App) SearchPage() int {
	return a.searchPage
}

// CallSearchCancel calls the current searchCancel function.
// Exported for tests to verify cancellation without exposing context.CancelFunc directly.
func (a *App) CallSearchCancel() {
	a.searchCancel()
}

// SearchCancelCtx returns the context associated with the current in-flight search.
// Returns nil when no search is in-flight (searchCancel is the no-op sentinel).
// Exported for tests to observe context cancellation.
func (a *App) SearchCancelCtx() context.Context {
	return a.searchCtx
}

// PlaylistTracksID returns the current playlist tracks staleness key (playlist ID).
// Exported for tests.
func (a *App) PlaylistTracksID() string {
	return a.playlistTracksID
}

// SetPlaylistTracksID sets the playlist tracks staleness key for testing.
// Exported for tests only.
func (a *App) SetPlaylistTracksID(id string) {
	a.playlistTracksID = id
}

// AlbumTracksID returns the current album tracks staleness key (album ID).
// Exported for tests.
func (a *App) AlbumTracksID() string {
	return a.albumTracksID
}

// SetAlbumTracksID sets the album tracks staleness key for testing.
// Exported for tests only.
func (a *App) SetAlbumTracksID(id string) {
	a.albumTracksID = id
}

// SetSearchSession sets the search staleness keys and loading flag for testing.
// This bypasses the normal SearchRequestMsg pathway so tests can set up state directly.
// Exported for tests only.
func (a *App) SetSearchSession(query string, page int, loading bool) {
	a.searchQuery = query
	a.searchPage = page
	a.searchLoading = loading
}

// SearchPane returns the search overlay pane.
// Exported for tests that need to inspect overlay state after openSearch().
func (a *App) SearchPane() *panes.SearchOverlay {
	return a.searchPane
}

// DeviceOverlayOpen returns true while the device switcher overlay is visible.
func (a *App) DeviceOverlayOpen() bool {
	return a.deviceOverlayOpen
}

// ProfileOverlayOpen returns true while the user profile overlay is visible.
func (a *App) ProfileOverlayOpen() bool {
	return a.profileOverlayOpen
}

// ThemeSwitcherOpen returns true while the theme switcher overlay is visible.
func (a *App) ThemeSwitcherOpen() bool {
	return a.showThemeSwitcher
}

// HelpOpen returns true while the help keybinding overlay is visible.
func (a *App) HelpOpen() bool {
	return a.helpOpen
}

// allPanes returns all pane values from the panes map.
// Used by the ThemeSwitchMsg handler to propagate theme changes to every pane.
func (a *App) allPanes() []layout.Pane {
	paneList := make([]layout.Pane, 0, len(a.panes))
	for _, p := range a.panes {
		paneList = append(paneList, p)
	}
	return paneList
}

// propagateSizes calls SetSize on all visible panes using their computed Rects from
// the LayoutManager. Hidden panes do not receive a SetSize call.
func (a *App) propagateSizes() {
	for id, pane := range a.panes {
		if a.layout.IsPaneVisible(id) {
			rect := a.layout.PaneRect(id)
			pane.SetSize(rect.ContentWidth(), rect.ContentHeight())
		}
	}
}

// syncFocus calls SetFocused(true/false) on all panes based on layout.FocusedPane().
func (a *App) syncFocus() {
	focused := a.layout.FocusedPane()
	for id, pane := range a.panes {
		pane.SetFocused(id == focused)
	}
}

// nowPlayingPane returns the NowPlayingPane from the panes map (convenience accessor).
func (a *App) nowPlayingPane() *panes.NowPlayingPane {
	p, ok := a.panes[layout.PaneNowPlaying]
	if !ok {
		return nil
	}
	if np, ok := p.(*panes.NowPlayingPane); ok {
		return np
	}
	return nil
}

// queuePane returns the QueuePane from the panes map (convenience accessor).
func (a *App) queuePane() *panes.QueuePane {
	p, ok := a.panes[layout.PaneQueue]
	if !ok {
		return nil
	}
	if qp, ok := p.(*panes.QueuePane); ok {
		return qp
	}
	return nil
}

// playlistsPane returns the PlaylistsPane from the panes map (convenience accessor).
func (a *App) playlistsPane() *panes.PlaylistsPane {
	p, ok := a.panes[layout.PanePlaylists]
	if !ok {
		return nil
	}
	if pp, ok := p.(*panes.PlaylistsPane); ok {
		return pp
	}
	return nil
}

// forwardToPane forwards a message to the pane at the given key, updates the
// pane slot, and returns any command. Returns nil if no pane is registered at key.
// Used for interactive sub-views (playlist tracks, album tracks) where the pane
// owns the data rather than the global store.
func (a *App) forwardToPane(key layout.PaneID, msg tea.Msg) tea.Cmd {
	p, ok := a.panes[key]
	if !ok || p == nil {
		return nil
	}
	updated, cmd := p.Update(msg)
	if lp, ok := updated.(layout.Pane); ok {
		a.panes[key] = lp
	} else {
		// Programming invariant: every pane in a.panes must implement layout.Pane.
		// If this fires, the pane returned a concrete type from Update that dropped the interface.
		fmt.Fprintf(os.Stderr, "spotnik: forwardToPane: pane at key %d returned %T from Update, which does not implement layout.Pane\n", key, updated)
	}
	return cmd
}

// albumsPane returns the AlbumsPane from the panes map (convenience accessor).
func (a *App) albumsPane() *panes.AlbumsPane {
	p, ok := a.panes[layout.PaneAlbums]
	if !ok {
		return nil
	}
	if ap, ok := p.(*panes.AlbumsPane); ok {
		return ap
	}
	return nil
}

// recentlyPlayedPane returns the RecentlyPlayedPane from the panes map.
func (a *App) recentlyPlayedPane() *panes.RecentlyPlayedPane {
	p, ok := a.panes[layout.PaneRecentlyPlayed]
	if !ok {
		return nil
	}
	if rp, ok := p.(*panes.RecentlyPlayedPane); ok {
		return rp
	}
	return nil
}

// topTracksPane returns the TopTracksPane from the panes map.
func (a *App) topTracksPane() *panes.TopTracksPane {
	p, ok := a.panes[layout.PaneTopTracks]
	if !ok {
		return nil
	}
	if tp, ok := p.(*panes.TopTracksPane); ok {
		return tp
	}
	return nil
}

// topArtistsPane returns the TopArtistsPane from the panes map.
func (a *App) topArtistsPane() *panes.TopArtistsPane {
	p, ok := a.panes[layout.PaneTopArtists]
	if !ok {
		return nil
	}
	if tp, ok := p.(*panes.TopArtistsPane); ok {
		return tp
	}
	return nil
}

// NowPlayingPane returns the NowPlayingPane from the panes map (exported for testing).
func (a *App) NowPlayingPane() *panes.NowPlayingPane {
	return a.nowPlayingPane()
}

// ActivePresetIndex returns the active preset index from the layout manager (exported for testing).
func (a *App) ActivePresetIndex() int {
	return a.layout.ActivePresetIndex()
}

// GatewayHealthPane returns the GatewayHealthPane from the panes map (exported for testing).
func (a *App) GatewayHealthPane() *panes.GatewayHealthPane {
	p, ok := a.panes[layout.PaneGatewayHealth]
	if !ok {
		return nil
	}
	if ghp, ok := p.(*panes.GatewayHealthPane); ok {
		return ghp
	}
	return nil
}

// PollingTrafficPane returns the PollingTrafficPane from the panes map (exported for testing).
func (a *App) PollingTrafficPane() *panes.PollingTrafficPane {
	p, ok := a.panes[layout.PanePollingTraffic]
	if !ok {
		return nil
	}
	if ptp, ok := p.(*panes.PollingTrafficPane); ok {
		return ptp
	}
	return nil
}

// GatewayLivePane returns the GatewayLivePane from the panes map (exported for testing).
func (a *App) GatewayLivePane() *panes.GatewayLivePane {
	p, ok := a.panes[layout.PaneGatewayLive]
	if !ok {
		return nil
	}
	if glp, ok := p.(*panes.GatewayLivePane); ok {
		return glp
	}
	return nil
}

// NetworkLogPane returns the NetworkLogPane from the panes map (exported for testing).
func (a *App) NetworkLogPane() *panes.NetworkLogPane {
	p, ok := a.panes[layout.PaneNetworkLog]
	if !ok {
		return nil
	}
	if nlp, ok := p.(*panes.NetworkLogPane); ok {
		return nlp
	}
	return nil
}

// OnboardingStep returns the current onboarding sub-step (exported for testing).
func (a *App) OnboardingStep() int {
	return a.onboardingStep
}

// OnboardingError returns the onboarding error message (exported for testing).
func (a *App) OnboardingError() string {
	return a.onboardingError
}

// OnboardingAuthURL returns the full OAuth authorization URL for the onboarding flow
// (exported for testing).
func (a *App) OnboardingAuthURL() string {
	return a.onboardingAuthURL
}

// NeedsRegister returns true when the app was started without a client ID in config
// (exported for testing).
func (a *App) NeedsRegister() bool {
	return a.needsRegister
}

// Init starts the splash timer. If the user is already authenticated,
// it also starts data fetching and the polling loop. If not, those are
// deferred until auth succeeds.
func (a *App) Init() tea.Cmd {
	splashTimer := tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
		return splashDismissMsg{}
	})

	// Batch alerts.Init() into the returned commands. BubbleUp currently returns
	// nil by design — it starts its internal tick only when an alert fires, not at
	// startup. Batching it here ensures a future BubbleUp upgrade that returns a
	// setup command is picked up automatically without code changes.
	alertsInitCmd := a.alerts.Init()

	if a.needsRegister || a.needsAuth {
		// Unauthenticated: only show splash, defer everything else until auth succeeds.
		return tea.Batch(splashTimer, alertsInitCmd)
	}

	// Authenticated: start data fetching and pane init alongside splash.
	var paneCmds []tea.Cmd
	for _, pane := range a.panes {
		if cmd := pane.Init(); cmd != nil {
			paneCmds = append(paneCmds, cmd)
		}
	}

	initCmds := append(paneCmds,
		fetchPlaybackStateCmd(a.player, api.Background),
		tea.Tick(time.Second, func(_ time.Time) tea.Msg {
			return panes.TickMsg{}
		}),
		splashTimer,
		alertsInitCmd,
		a.buildFetchCurrentUserCmd(), // fetch user ID for playlist ownership checks
	)
	return tea.Batch(initCmds...)
}

// openSearch opens the search overlay. Reset() is called before Init() so
// each search session starts with a clean slate — no stale query, prefix,
// tab, or result list from the previous session.
// searchCancel is reset to a no-op here (the prior cancel was already called
// by the previous closeSearch call, so there is no in-flight HTTP to abort).
func (a *App) openSearch() (*App, tea.Cmd) {
	// Reset the cancel func to a fresh no-op; prior cancel was already called.
	a.searchCancel = func() {}
	a.searchCtx = nil
	a.searchPane.Reset()
	a.searchOpen = true
	cmd := a.searchPane.Init()
	return a, cmd
}

// closeSearch closes the search overlay and aborts any in-flight search HTTP call.
func (a *App) closeSearch() (*App, tea.Cmd) {
	// Cancel any in-flight HTTP call immediately.
	a.searchCancel()
	// Reset the cancel func to a no-op so subsequent calls are safe.
	a.searchCancel = func() {}
	a.searchCtx = nil
	// Clear all search session state.
	a.searchQuery = ""
	a.searchPage = 0
	a.searchLoading = false
	a.searchOpen = false
	return a, nil
}

// openDeviceOverlay opens the device switcher overlay and fetches the device list.
func (a *App) openDeviceOverlay() (*App, tea.Cmd) {
	a.deviceOverlayOpen = true
	cmd := a.devicePane.Init()
	return a, cmd
}

// closeDeviceOverlay closes the device switcher overlay.
func (a *App) closeDeviceOverlay() (*App, tea.Cmd) {
	a.deviceOverlayOpen = false
	return a, nil
}

// openThemeSwitcher opens the theme switcher overlay.
func (a *App) openThemeSwitcher() (*App, tea.Cmd) {
	a.showThemeSwitcher = true
	a.themeOverlay = panes.NewThemeOverlay(
		theme.AllThemes(),
		a.theme.ID(),
		a.theme,
	)
	return a, nil
}

// closeThemeSwitcher closes the theme switcher overlay without changing the theme.
func (a *App) closeThemeSwitcher() (*App, tea.Cmd) {
	a.showThemeSwitcher = false
	a.themeOverlay = nil
	return a, nil
}

// openHelp opens the help keybinding overlay.
func (a *App) openHelp() (*App, tea.Cmd) {
	a.helpOpen = true
	a.helpOverlay = panes.NewHelpOverlay(a.theme)
	a.helpOverlay.SetSize(a.width, a.height)
	return a, nil
}

// closeHelp closes the help keybinding overlay.
func (a *App) closeHelp() (*App, tea.Cmd) {
	a.helpOpen = false
	a.helpOverlay = nil
	return a, nil
}

// openOnboardingPermissions opens the Step 2 permissions overlay.
func (a *App) openOnboardingPermissions() (*App, tea.Cmd) {
	a.onboardingPermissionsOverlay = panes.NewOnboardingPermissionsOverlay(a.theme)
	a.onboardingPermissionsOverlay.SetSize(a.width, a.height)
	return a, nil
}

// closeOnboardingPermissions closes the Step 2 permissions overlay.
func (a *App) closeOnboardingPermissions() (*App, tea.Cmd) {
	a.onboardingPermissionsOverlay = nil
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
	// BubbleUp.AlertModel.Update always returns AlertModel (value type). If this
	// assertion fails, it indicates a BubbleUp library bug — alert state freezes
	// but the app continues working without crashing.
	if am, ok := updatedAlerts.(bubbleup.AlertModel); ok {
		a.alerts = am
	}

	return model, tea.Batch(mainCmd, alertCmd)
}
