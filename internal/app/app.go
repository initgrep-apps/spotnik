// Package app contains the root Bubble Tea model that wires together all
// panes, the central store, and the active theme. It is the single entry
// point for the TUI application.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/prefs"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/components/viz"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"go.dalton.dog/bubbleup"
)

// viewMode identifies which top-level view is currently active.
type viewMode int

const (
	viewSplash viewMode = iota // Splash screen shown on startup
	viewAuth                   // Auth panel shown when user needs to authenticate
	viewGrid                   // Grid layout: 10 panes across 2 pages, managed by LayoutManager
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

	// layout manages the grid, focus, preset, and page system.
	layout *layout.Manager

	// statusHelp renders the bottom keybinding bar using bubbles/help (ShowAll=true).
	// statusKeyMap holds the bindings; activePage is set per render call.
	statusHelp   help.Model
	statusKeyMap appKeyMap

	// panes holds all 8 Page A panes (Page B panes added in Feature 51).
	panes map[layout.PaneID]layout.Pane

	// searchPane and devicePane are overlay panes — they float above the grid.
	searchPane *panes.SearchOverlay
	devicePane *panes.DeviceOverlay

	// themeOverlay is the floating theme switcher overlay. Populated when open.
	themeOverlay *panes.ThemeOverlay

	// playlistsAPI is the Spotify playlists mutation client.
	playlistsAPI api.PlaylistsAPI

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

	// showThemeSwitcher is true while the theme switcher overlay is visible.
	showThemeSwitcher bool

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

	// consecutivePlaybackErrors counts successive PlaybackStateFetchedMsg deliveries
	// where Err is non-nil. A toast is emitted when this reaches 5, then the counter
	// continues to increment (so exactly the 5th triggers the toast, not subsequent ones).
	// The counter resets to 0 on any successful fetch.
	consecutivePlaybackErrors int

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

// prefsFlushTickMsg is sent by the debounce timer started in schedulePrefsFlush.
// Only the tick whose generation matches the current prefsDirtyGen triggers a flush;
// stale ticks from superseded changes are ignored.
type prefsFlushTickMsg struct{ generation int }

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

	// Create Page B panes.
	// RequestFlowPane reads gateway events from the store's event journal,
	// preserving the ui/ → state/ dependency direction (no gateway reference).
	requestFlowPane := panes.NewRequestFlowPane(s, t)
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
		layout.PaneRequestFlow:    requestFlowPane,
		layout.PaneNetworkLog:     networkLogPane,
	}

	searchPane := panes.NewSearchOverlay(t)
	devicePane := panes.NewDeviceOverlay(s, t)

	mgr := layout.NewManager()

	a := &App{
		theme:           t,
		store:           s,
		alerts:          *components.NewNotifications(t),
		gateway:         gw,
		layout:          mgr,
		panes:           panesMap,
		searchPane:      searchPane,
		devicePane:      devicePane,
		statusHelp:      newStatusHelp(t),
		statusKeyMap:    newAppKeyMap(),
		currentView:     viewSplash,
		volumeStep:      5,
		needsAuth:       opts.NeedsAuth,
		clientID:        opts.ClientID,
		tokenStore:      opts.TokenStore,
		tokenBaseURL:    opts.TokenBaseURL,
		lastInteraction: time.Now(),
		idleThreshold:   idleThresholdSecs * time.Second,
		prefs:           prefs.New(config.DefaultConfigPath()),
		// searchCancel must never be nil; initialize to a no-op so it is always safe to call.
		searchCancel: func() {},
	}

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

// GridViewOpen returns true while the grid view is the active top-level view.
func (a *App) GridViewOpen() bool {
	return a.currentView == viewGrid
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

// ThemeSwitcherOpen returns true while the theme switcher overlay is visible.
func (a *App) ThemeSwitcherOpen() bool {
	return a.showThemeSwitcher
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

// RequestFlowPane returns the RequestFlowPane from the panes map (exported for testing).
func (a *App) RequestFlowPane() *panes.RequestFlowPane {
	p, ok := a.panes[layout.PaneRequestFlow]
	if !ok {
		return nil
	}
	if rfp, ok := p.(*panes.RequestFlowPane); ok {
		return rfp
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

// Init starts the splash timer. If the user is already authenticated,
// it also starts data fetching and the polling loop. If not, those are
// deferred until auth succeeds.
func (a *App) Init() tea.Cmd {
	splashTimer := tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
		return splashDismissMsg{}
	})

	// Batch alerts.Init() into the returned commands. BubbleUp currently returns
	// nil by design — it starts its internal tick only when an alert fires, not at
	// startup. Batching it here ensures a future BubbleUp upgrade that returns a
	// setup command is picked up automatically without code changes.
	alertsInitCmd := a.alerts.Init()

	if a.needsAuth {
		// Unauthenticated: only show splash, defer everything else.
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
		fetchPlaybackStateCmd(a.player),
		tea.Tick(time.Second, func(_ time.Time) tea.Msg {
			return panes.TickMsg{}
		}),
		splashTimer,
		alertsInitCmd,
	)
	initCmds = append(initCmds, a.initialFetchCmds()...)
	return tea.Batch(initCmds...)
}

// initialFetchCmds returns commands that trigger the initial data load for all
// library and stats panes. Each command emits a Fetch*RequestMsg which flows
// through the existing staleness/dedup guards in handleMsg.
func (a *App) initialFetchCmds() []tea.Cmd {
	return []tea.Cmd{
		func() tea.Msg { return panes.FetchPlaylistsRequestMsg{Offset: 0} },
		func() tea.Msg { return panes.FetchAlbumsRequestMsg{Offset: 0} },
		func() tea.Msg { return panes.FetchLikedTracksRequestMsg{Offset: 0} },
		func() tea.Msg { return panes.FetchRecentlyPlayedRequestMsg{} },
		func() tea.Msg { return panes.FetchStatsMsg{TimeRange: "short_term"} },
	}
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

// schedulePrefsFlush marks preferences dirty and starts a 500ms debounce timer.
// Returns the tea.Cmd for the timer. Call this after every prefs.Set().
// The generation counter ensures only the latest change triggers a disk write.
func (a *App) schedulePrefsFlush() tea.Cmd {
	a.prefsDirtyGen++
	gen := a.prefsDirtyGen
	return tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
		return prefsFlushTickMsg{generation: gen}
	})
}

// SchedulePrefsFlush is the exported test accessor for schedulePrefsFlush.
// It increments the generation counter and returns the debounce tick Cmd.
func (a *App) SchedulePrefsFlush() tea.Cmd {
	return a.schedulePrefsFlush()
}

// Prefs returns the underlying PreferenceStore for inspection in tests.
func (a *App) Prefs() *prefs.PreferenceStore {
	return a.prefs
}

// PrefsDirtyGen returns the current preference dirty generation counter.
// Used in tests to verify that schedulePrefsFlush increments it correctly.
func (a *App) PrefsDirtyGen() int {
	return a.prefsDirtyGen
}

// clearAllFetchingSentinels resets every in-flight fetch sentinel to false.
// Called from RateLimitedMsg and unauthorizedMsg handlers, which short-circuit
// the normal command → loaded-message path, so the loaded-message handler never
// runs to clear the sentinel itself. Leaving sentinels true would permanently
// block the staleness gate for those domains.
func (a *App) clearAllFetchingSentinels() {
	a.store.SetPlaylistsFetching(false)
	a.store.SetAlbumsFetching(false)
	a.store.SetLikedFetching(false)
	a.store.SetRecentFetching(false)
	a.store.SetDevicesFetching(false)
	// Stats sentinels are keyed by time range — clear all known ranges.
	for _, r := range []string{"short_term", "medium_term", "long_term"} {
		a.store.SetStatsFetching(r, false)
	}
	// Clear app-level search loading flag: when a 429 or 401 short-circuits the search command,
	// the SearchPageLoadedMsg handler never fires, so loading must be cleared here.
	a.searchLoading = false
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

// handleMsg contains the core application message routing logic.
// It is called by Update() which also forwards messages to the alerts model.
func (a *App) handleMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Preference flush messages are handled first and are independent of view mode.
	if model, cmd, handled := a.handlePrefsMsg(msg); handled {
		return model, cmd
	}

	switch m := msg.(type) {
	case splashDismissMsg:
		if a.currentView == viewSplash {
			if a.needsAuth {
				a.currentView = viewAuth
				a.authStatus = "Opening browser for authorization..."
				return a, prepareAuthCmd(a.clientID)
			}
			a.currentView = viewGrid
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
		a.currentView = viewGrid
		a.initAPIClients(m.accessToken)
		// Start data fetching and tick loop.
		var paneCmds []tea.Cmd
		for _, pane := range a.panes {
			if cmd := pane.Init(); cmd != nil {
				paneCmds = append(paneCmds, cmd)
			}
		}
		authCmds := append(paneCmds,
			fetchPlaybackStateCmd(a.player),
			tea.Tick(time.Second, func(_ time.Time) tea.Msg {
				return panes.TickMsg{}
			}),
		)
		authCmds = append(authCmds, a.initialFetchCmds()...)
		return a, tea.Batch(authCmds...)

	case authErrorMsg:
		a.authStatus = fmt.Sprintf("Error: %s — press q to quit", m.err.Error())
		return a, nil

	case panes.SearchClosedMsg:
		// Search overlay closed — close overlay.
		return a.closeSearch()

	case panes.SearchRequestMsg:
		// Cancel any in-flight HTTP call before starting a new one.
		a.searchCancel()

		// Use the type filter from the overlay when set (e.g. ":songs" → ["track"]).
		// Fall back to all four types when no prefix filter is active.
		searchTypes := m.Types
		if len(searchTypes) == 0 {
			searchTypes = []string{"track", "artist", "album", "playlist"}
		}

		// Record staleness keys for the incoming request.
		a.searchQuery = m.Query
		a.searchPage = m.Page
		a.searchLoading = true

		// Create a cancellable context for this request.
		ctx, cancel := context.WithCancel(context.Background())
		a.searchCancel = cancel
		a.searchCtx = ctx

		// Tell the overlay we are loading before the HTTP call goes out.
		isFirst := len(a.searchPane.Results()) == 0
		loadingCmd := func() tea.Msg { return panes.SearchLoadingMsg{IsFirstPage: isFirst} }

		fetchCmd := buildSearchPageCmd(ctx, a.search, m.Query, searchTypes, m.Page)

		return a, tea.Batch(loadingCmd, fetchCmd)

	case panes.SearchClearedMsg:
		// Overlay owns cleared state (story 99) — nothing to do at app level.
		return a, nil

	case panes.SearchPageLoadedMsg:
		// Search page fetch returned — route errors through toast, forward success to overlay.
		//
		// NOTE: Staleness check runs FIRST, before the error check. A context-cancelled
		// error from a prior query (e.g. user typed a new query mid-flight) must be
		// silently discarded — emitting a toast for "context canceled" would disrupt the
		// new request's loading state and confuse the user.
		if errors.Is(m.Err, errNilClient) {
			// errNilClient is a programming error, not a network result — always surface it.
			return a, nil
		}
		// Discard stale results AND stale errors — the user moved on to a different query or page.
		if m.Query != a.searchQuery || m.Page != a.searchPage {
			return a, nil
		}
		if m.Err != nil {
			// Clear app-level loading flag so the overlay's spinner can be dismissed.
			a.searchLoading = false
			// Forward the error msg to overlay so it can clear its loading flags
			// (loadingFirstPage / loadingNextPage). The overlay keeps its existing
			// results visible — it only clears spinners.
			updated, _ := a.searchPane.Update(m)
			if sp, ok := updated.(*panes.SearchOverlay); ok {
				a.searchPane = sp
			}
			return a, a.alerts.NewAlertCmd("warning", "Search failed: "+m.Err.Error())
		}
		a.searchLoading = false
		// Forward to the search pane so it can update its local display state.
		updated, cmd := a.searchPane.Update(m)
		if sp, ok := updated.(*panes.SearchOverlay); ok {
			a.searchPane = sp
		}
		return a, cmd

	case tea.MouseMsg:
		return a, a.handleMouseMsg(m)

	case tea.WindowSizeMsg:
		// Terminal resize implies user presence — reset idle state the same way KeyMsg does.
		wasIdle := a.isIdle()
		a.lastInteraction = time.Now()
		var toastCmd tea.Cmd
		if wasIdle {
			// User returned from idle via resize — force immediate poll on the next tick.
			a.tickCount = 0
			if a.backoffTicks > 0 {
				// Active 429 backoff prevents any fetches after idle return.
				// Emit a ratelimit toast so the user knows data is stale and why.
				toastCmd = a.alerts.NewAlertCmd("ratelimit",
					fmt.Sprintf("Rate limited — resuming in %ds", a.backoffTicks))
			}
		}
		a.width = m.Width
		a.height = m.Height
		// Propagate terminal size through LayoutManager, then to all visible panes.
		a.layout.Resize(m.Width, m.Height)
		a.propagateSizes()
		a.syncFocus()
		a.searchPane.SetSize(m.Width, m.Height)
		a.devicePane.SetSize(m.Width, m.Height)
		if a.themeOverlay != nil {
			a.themeOverlay.SetSize(m.Width, m.Height)
		}
		if toastCmd != nil {
			return a, toastCmd
		}
		return a, nil

	case panes.FetchStatsMsg:
		// Stats pane requesting data for a time range.
		// If a fetch is already in-flight, silently skip to prevent TOCTOU duplicates.
		if a.store.StatsFetching(m.TimeRange) {
			return a, nil
		}
		// If data is fresh, return cached data so the view can initialize without
		// waiting for a redundant API round-trip.
		if !a.store.StatsStale(m.TimeRange) {
			cachedTracks := a.store.TopTracks(m.TimeRange)
			cachedArtists := a.store.TopArtists(m.TimeRange)
			timeRange := m.TimeRange
			return a, func() tea.Msg {
				return panes.StatsLoadedMsg{
					TimeRange:  timeRange,
					TopTracks:  cachedTracks,
					TopArtists: cachedArtists,
				}
			}
		}
		a.store.SetStatsFetching(m.TimeRange, true)
		return a, a.buildFetchStatsCmd(m.TimeRange)

	case panes.StatsLoadedMsg:
		// Stats data fetched — clear fetching sentinel, write to store, forward to panes.
		if m.TimeRange != "" {
			a.store.SetStatsFetching(m.TimeRange, false)
		}
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
			a.store.SetStatsError(m.Err)
			return a, a.alerts.NewAlertCmd("error", "Failed to load stats. Press f to retry")
		}
		a.store.ClearStatsError()
		if m.TimeRange != "" {
			a.store.SetTopTracks(m.TimeRange, m.TopTracks)
			a.store.SetTopArtists(m.TimeRange, m.TopArtists)
			// Stamp once after both setters so StatsStale only returns false when
			// both datasets are present. Stamping inside each setter would mark the
			// range fresh after only one write, hiding partial-data state.
			a.store.StampStatsFetchedAt(m.TimeRange)
		}
		// Forward to TopTracks and TopArtists panes.
		var cmds []tea.Cmd
		if tp := a.topTracksPane(); tp != nil {
			updated, cmd := tp.Update(m)
			if tpu, ok := updated.(*panes.TopTracksPane); ok {
				a.panes[layout.PaneTopTracks] = tpu
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if ap := a.topArtistsPane(); ap != nil {
			updated, cmd := ap.Update(m)
			if apu, ok := updated.(*panes.TopArtistsPane); ok {
				a.panes[layout.PaneTopArtists] = apu
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if len(cmds) > 0 {
			return a, tea.Batch(cmds...)
		}
		return a, nil

	case tea.KeyMsg:
		wasIdle := a.isIdle()
		a.lastInteraction = time.Now()
		if wasIdle {
			// User returned from idle — force immediate poll on the next tick.
			a.tickCount = 0
			if a.backoffTicks > 0 {
				// Active 429 backoff prevents any fetches after idle return.
				// Emit a ratelimit toast so the user knows data is stale and why.
				toastCmd := a.alerts.NewAlertCmd("ratelimit",
					fmt.Sprintf("Rate limited — resuming in %ds", a.backoffTicks))
				updated, keyCmd := a.handleKeyMsg(m)
				return updated, tea.Batch(toastCmd, keyCmd)
			}
		}
		return a.handleKeyMsg(m)

	case panes.TickMsg:
		// Forward TickMsg to NowPlaying for progress bar animation.
		if np := a.nowPlayingPane(); np != nil {
			updatedPane, _ := np.Update(m)
			if pp, ok := updatedPane.(*panes.NowPlayingPane); ok {
				a.panes[layout.PaneNowPlaying] = pp
			}
		}

		// Forward TickMsg to Page B panes so they refresh their data.
		if rfp := a.RequestFlowPane(); rfp != nil {
			updated, _ := rfp.Update(m)
			if p, ok := updated.(*panes.RequestFlowPane); ok {
				a.panes[layout.PaneRequestFlow] = p
			}
		}
		if nlp := a.NetworkLogPane(); nlp != nil {
			updated, _ := nlp.Update(m)
			if p, ok := updated.(*panes.NetworkLogPane); ok {
				a.panes[layout.PaneNetworkLog] = p
			}
		}

		// Send current polling snapshot to RequestFlowPane for the status strip.
		// This is done by sending a PollingSnapshotMsg after the TickMsg.
		idle := a.isIdle()
		var idleSecs int
		if idle {
			idleSecs = int(time.Since(a.lastInteraction).Seconds())
		}
		playbackIntervalForSnap, _ := a.pollIntervals()
		pollingSnapshot := panes.PollingSnapshotMsg{
			TickIntervalMs: playbackIntervalForSnap * 1000,
			IsIdle:         idle,
			IdleSecs:       idleSecs,
		}
		if rfp := a.RequestFlowPane(); rfp != nil {
			updated, _ := rfp.Update(pollingSnapshot)
			if p, ok := updated.(*panes.RequestFlowPane); ok {
				a.panes[layout.PaneRequestFlow] = p
			}
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

	case viz.TickMsg:
		// Emit periodic gateway events (token refills, backoff expiry).
		// These run on every 200ms viz tick so the event journal stays current
		// without requiring the gateway hot-path to perform periodic work.
		a.gateway.CheckAndEmitRefill()
		a.gateway.CheckAndEmitBackoffExpiry()
		// Forward viz.TickMsg to NowPlaying pane and Page B RequestFlowPane.
		// Both panes share the 200ms animation tick for visual consistency.
		var visCmds []tea.Cmd
		if np := a.nowPlayingPane(); np != nil {
			updated, cmd := np.Update(m)
			if pp, ok := updated.(*panes.NowPlayingPane); ok {
				a.panes[layout.PaneNowPlaying] = pp
			}
			if cmd != nil {
				visCmds = append(visCmds, cmd)
			}
		}
		if rfp := a.RequestFlowPane(); rfp != nil {
			updated, rfpCmd := rfp.Update(m)
			if p, ok := updated.(*panes.RequestFlowPane); ok {
				a.panes[layout.PaneRequestFlow] = p
			}
			if rfpCmd != nil {
				visCmds = append(visCmds, rfpCmd)
			}
		}
		if len(visCmds) > 0 {
			return a, tea.Batch(visCmds...)
		}
		return a, nil

	case panes.RateLimitedMsg:
		// 429 from Spotify — activate backoff and emit a ratelimit toast.
		backoff := m.RetryAfterSecs
		if backoff < defaultBackoffTicks {
			backoff = defaultBackoffTicks
		}
		a.backoffTicks = backoff
		// Update store throttle observability so UI components can read gateway state.
		a.store.SetThrottle(true, m.RetryAfterSecs, time.Now())
		// Clear all fetching sentinels so the staleness gate can re-dispatch after backoff.
		// When a rate-limited command fires, it returns RateLimitedMsg instead of a domain
		// loaded message, so the loaded-message handler never runs to clear the sentinel.
		// Without this, any domain with a sentinel set at the time of the 429 would be
		// permanently blocked from fetching again.
		a.clearAllFetchingSentinels()
		return a, tea.Batch(
			a.alerts.NewAlertCmd("ratelimit", fmt.Sprintf("Rate limited, retrying in %ds", backoff)),
			tea.Tick(time.Duration(backoff)*time.Second, func(_ time.Time) tea.Msg { return throttleExpiredMsg{} }),
		)

	case unauthorizedMsg:
		// 401 from any Spotify API call — attempt a token refresh.
		// Clear sentinels for the same reason as RateLimitedMsg: the domain loaded-message
		// handler never fires when the command short-circuits to unauthorizedMsg.
		a.clearAllFetchingSentinels()
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
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
			a.store.SetQueueError(m.Err)
			return a, a.alerts.NewAlertCmd("error", "Queue update failed")
		}
		a.store.ClearQueueError()
		a.store.SetQueue(m.Tracks)
		if qp := a.queuePane(); qp != nil {
			qp.RefreshRows()
		}
		return a, nil

	case panes.PlaybackStateFetchedMsg:
		// Write state to store from Msg payload (Elm Architecture: only Update writes store).
		// Only write to store when State is non-nil — nil State means either:
		//   a) player == nil (no API client injected), or
		//   b) a transient error (m.Err != nil).
		// In both cases we leave the existing store state unchanged.
		if m.Err != nil {
			// errNilClient means the API client is not yet initialized (pre-auth startup).
			// Skip silently — must come BEFORE the counter so startup doesn't increment it.
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
			a.consecutivePlaybackErrors++
			// Emit a warning toast only on the exact 5th consecutive failure to avoid
			// flooding the user with toasts at 1-3s polling intervals. The counter
			// continues to increment after 5 so subsequent errors don't re-toast.
			if a.consecutivePlaybackErrors == 5 {
				return a, a.alerts.NewAlertCmd("warning", "Playback updates failing — check connection")
			}
			return a, nil
		}
		a.consecutivePlaybackErrors = 0
		if m.State != nil {
			a.store.SetPlaybackState(m.State)
			// A non-nil state means the connection is healthy — reset the nil-state counter.
			a.nilPlaybackStateTicks = 0
		} else {
			// Nil State + nil Err means 204 (nothing playing) or no API client.
			// Track how long we've been in this state to surface stuck connections.
			a.nilPlaybackStateTicks++
			// Warn once at exactly the 30th tick (~30-90s depending on polling interval).
			// Avoids flooding toasts on startup; only fires once since the counter is not reset here.
			if a.nilPlaybackStateTicks == 30 {
				return a, a.alerts.NewAlertCmd("warning", "No playback state received — check Spotify connection")
			}
		}
		// Data fetched during splash is stored but splash stays visible
		// for the full 5s duration — the splashDismissMsg timer handles transition.
		if np := a.nowPlayingPane(); np != nil {
			updatedPane, cmd := np.Update(m)
			if pp, ok := updatedPane.(*panes.NowPlayingPane); ok {
				a.panes[layout.PaneNowPlaying] = pp
			}
			return a, cmd
		}
		return a, nil

	case panes.PlaybackCmdSentMsg:
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
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
		// Overlay stays open — only Esc (SearchClosedMsg) closes it.
		return a, a.buildPlayContextCmd(m.ContextURI)

	case panes.PlayTrackMsg:
		// Overlay stays open — only Esc (SearchClosedMsg) closes it.
		return a, a.buildPlayTrackCmd(m.TrackURI)

	case panes.AddToQueueMsg:
		return a, a.buildAddToQueueCmd(m.TrackURI, m.TrackName)

	case panes.AddToQueueResultMsg:
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
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
		// Paginated requests (Offset > 0) always proceed to avoid incomplete data.
		if m.Offset == 0 {
			if !a.store.PlaylistsStale() {
				// Data is fresh — send cached playlists so the pane can initialize
				// without waiting for a redundant API round-trip.
				cached := a.store.Playlists()
				return a, func() tea.Msg {
					return panes.LibraryLoadedMsg{Items: cached, Offset: 0}
				}
			}
			if a.store.PlaylistsFetching() {
				// Fetch already in-flight — skip to prevent TOCTOU duplicates.
				return a, nil
			}
			a.store.SetPlaylistsFetching(true)
		}
		return a, a.buildFetchPlaylistsCmd(m.Offset)

	case panes.LibraryLoadedMsg:
		// Write playlist data to store from Msg payload (Elm Architecture: only Update writes store).
		// Clear fetching sentinel so a subsequent stale check can dispatch a fresh fetch.
		a.store.SetPlaylistsFetching(false)
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
			a.store.SetPlaylistsFetchError(m.Err)
			// Forward to PlaylistsPane for error display.
			if pp := a.playlistsPane(); pp != nil {
				updated, _ := pp.Update(m)
				if ppu, ok := updated.(*panes.PlaylistsPane); ok {
					a.panes[layout.PanePlaylists] = ppu
				}
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
		// Forward to PlaylistsPane so it can refresh from store.
		if pp := a.playlistsPane(); pp != nil {
			updated, cmd := pp.Update(m)
			if ppu, ok := updated.(*panes.PlaylistsPane); ok {
				a.panes[layout.PanePlaylists] = ppu
			}
			return a, cmd
		}
		return a, nil

	case panes.FetchAlbumsRequestMsg:
		// Paginated requests (Offset > 0) always proceed to avoid incomplete data.
		if m.Offset == 0 {
			if !a.store.AlbumsStale() {
				// Data is fresh — send cached albums so the pane can initialize.
				cached := a.store.SavedAlbums()
				return a, func() tea.Msg {
					return panes.AlbumsLoadedMsg{Items: cached, Offset: 0}
				}
			}
			if a.store.AlbumsFetching() {
				return a, nil
			}
			a.store.SetAlbumsFetching(true)
		}
		return a, a.buildFetchAlbumsCmd(m.Offset)

	case panes.AlbumsLoadedMsg:
		// Write album data to store from Msg payload.
		// Clear fetching sentinel so a subsequent stale check can dispatch a fresh fetch.
		a.store.SetAlbumsFetching(false)
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
			a.store.SetAlbumsFetchError(m.Err)
			// Forward to AlbumsPane for error display.
			if ap := a.albumsPane(); ap != nil {
				updated, _ := ap.Update(m)
				if apu, ok := updated.(*panes.AlbumsPane); ok {
					a.panes[layout.PaneAlbums] = apu
				}
			}
			return a, a.alerts.NewAlertCmd("error", "Failed to load albums. Press Tab to retry")
		}
		a.store.ClearAlbumsFetchError()
		if m.Offset == 0 {
			a.store.SetSavedAlbums(m.Items)
		} else {
			a.store.SetSavedAlbums(append(a.store.SavedAlbums(), m.Items...))
		}
		// Forward to AlbumsPane.
		if ap := a.albumsPane(); ap != nil {
			updated, cmd := ap.Update(m)
			if apu, ok := updated.(*panes.AlbumsPane); ok {
				a.panes[layout.PaneAlbums] = apu
			}
			return a, cmd
		}
		return a, nil

	case panes.FetchLikedTracksRequestMsg:
		// Paginated requests (Offset > 0) always proceed to avoid incomplete data.
		if m.Offset == 0 {
			if !a.store.LikedTracksStale() {
				// Data is fresh — send cached liked tracks so the pane can initialize.
				cached := a.store.LikedTracks()
				return a, func() tea.Msg {
					return panes.LikedTracksLoadedMsg{Items: cached, Offset: 0}
				}
			}
			if a.store.LikedFetching() {
				return a, nil
			}
			a.store.SetLikedFetching(true)
		}
		return a, a.buildFetchLikedTracksCmd(m.Offset)

	case panes.LikedTracksLoadedMsg:
		// Write liked tracks to store from Msg payload.
		// Clear fetching sentinel so a subsequent stale check can dispatch a fresh fetch.
		a.store.SetLikedFetching(false)
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
			a.store.SetLikedTracksFetchError(m.Err)
			return a, a.alerts.NewAlertCmd("error", "Failed to load liked tracks. Press Tab to retry")
		}
		a.store.ClearLikedTracksFetchError()
		a.store.SetLikedTracks(m.Items)
		a.store.SetLikedTotal(len(m.Items) + m.Offset)
		// LikedSongsPane reads from store on RefreshRows — forward message to trigger refresh.
		if lsp, ok := a.panes[layout.PaneLikedSongs]; ok {
			updated, cmd := lsp.Update(m)
			a.panes[layout.PaneLikedSongs] = updated.(layout.Pane)
			return a, cmd
		}
		return a, nil

	case panes.FetchRecentlyPlayedRequestMsg:
		// NOTE: recently-played has no pagination.
		if !a.store.RecentlyPlayedStale() {
			// Data is fresh — send cached recently-played so the pane can initialize.
			cached := a.store.RecentlyPlayed()
			return a, func() tea.Msg {
				return panes.RecentlyPlayedLoadedMsg{Items: cached}
			}
		}
		if a.store.RecentFetching() {
			// Fetch already in-flight — skip to prevent TOCTOU duplicates.
			return a, nil
		}
		a.store.SetRecentFetching(true)
		return a, a.buildFetchRecentlyPlayedCmd()

	case panes.RecentlyPlayedLoadedMsg:
		// Write recently played to store from Msg payload.
		// Clear fetching sentinel so a subsequent stale check can dispatch a fresh fetch.
		a.store.SetRecentFetching(false)
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
			a.store.SetRecentPlayedFetchError(m.Err)
			return a, a.alerts.NewAlertCmd("error", "Failed to load recently played. Press Tab to retry")
		}
		a.store.ClearRecentPlayedFetchError()
		a.store.SetRecentlyPlayed(m.Items)
		// Forward to RecentlyPlayedPane.
		if rp := a.recentlyPlayedPane(); rp != nil {
			updated, cmd := rp.Update(m)
			if rpu, ok := updated.(*panes.RecentlyPlayedPane); ok {
				a.panes[layout.PaneRecentlyPlayed] = rpu
			}
			return a, cmd
		}
		return a, nil

	case panes.LikeTrackRequestMsg:
		return a, a.buildToggleLikeCmd(m.TrackID, m.Unlike)

	case panes.LikeToggleResultMsg:
		// Like/unlike result — no action needed unless there's an error.
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
			return a, a.alerts.NewAlertCmd("error", m.Err.Error())
		}
		return a, nil

	case panes.DeviceOverlayClosedMsg:
		// Device overlay closed via Esc — close overlay.
		return a.closeDeviceOverlay()

	case panes.ThemeSwitchMsg:
		// User selected a theme in the overlay — load new theme, propagate to all panes.
		newTheme := theme.Load(m.ThemeID)
		a.theme = newTheme
		// Propagate to all grid panes.
		for _, p := range a.allPanes() {
			p.SetTheme(newTheme)
		}
		// Propagate to overlay panes.
		a.searchPane.SetTheme(newTheme)
		a.devicePane.SetTheme(newTheme)
		if a.themeOverlay != nil {
			a.themeOverlay.SetTheme(newTheme)
		}
		// Re-style the status bar help component with the new theme colors.
		a.statusHelp = newStatusHelp(newTheme)
		// Close the overlay.
		a.showThemeSwitcher = false
		a.themeOverlay = nil
		// Recreate alerts so new toasts use the new theme's colors.
		// This must happen before the NewAlertCmd call below so the success
		// toast itself renders with the new theme's success color.
		// NOTE: Any in-flight toast is intentionally dropped — theme switch is
		// user-initiated and the success toast fires immediately after.
		a.alerts = *components.NewNotifications(newTheme)
		// Queue the theme preference and schedule a debounced flush.
		a.prefs.Set("theme", m.ThemeID)
		alertInitCmd := a.alerts.Init()
		return a, tea.Batch(
			alertInitCmd,
			a.alerts.NewAlertCmd("success", "Theme: "+newTheme.Name()),
			a.schedulePrefsFlush(),
		)

	case panes.ThemeOverlayClosedMsg:
		// Theme overlay closed via Esc — close overlay without changing theme.
		return a.closeThemeSwitcher()

	case panes.FetchDevicesRequestMsg:
		// Device overlay is requesting the device list from the API.
		// If a fetch is already in-flight, silently skip to prevent TOCTOU duplicates.
		if a.store.DevicesFetching() {
			return a, nil
		}
		// Short cooldown (5s) prevents rapid-fire API calls while keeping data fresh.
		if !a.store.DevicesStale() {
			cached := a.store.Devices()
			infos := make([]panes.DeviceInfo, 0, len(cached))
			for _, d := range cached {
				infos = append(infos, panes.DeviceInfo{
					ID:       d.ID,
					Name:     d.Name,
					Type:     d.Type,
					IsActive: d.IsActive,
				})
			}
			return a, func() tea.Msg {
				return panes.DevicesLoadedMsg{Devices: infos}
			}
		}
		a.store.SetDevicesFetching(true)
		return a, a.buildFetchDevicesCmd()

	case panes.DevicesLoadedMsg:
		// Device list fetched — write store state here (Elm purity: only root Update writes store).
		// Clear fetching sentinel, then forward to DeviceOverlay for rendering.
		a.store.SetDevicesFetching(false)
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
			a.store.SetDevicesError(m.Err)
			// Forward to overlay so it can update its render state.
			if a.deviceOverlayOpen {
				updated, _ := a.devicePane.Update(m)
				if dp, ok := updated.(*panes.DeviceOverlay); ok {
					a.devicePane = dp
				}
			}
			return a, a.alerts.NewAlertCmd("error", fmt.Sprintf("Failed to load devices: %s", m.Err.Error()))
		}
		a.store.ClearDevicesError()
		a.store.SetDevicesFetchedAt(time.Now())
		// Cache the raw domain device list in the store and stamp fetchedAt.
		// Reverse-convert panes.DeviceInfo → domain.Device (same fields, lossless).
		rawDevices := make([]domain.Device, 0, len(m.Devices))
		for _, info := range m.Devices {
			rawDevices = append(rawDevices, domain.Device{
				ID:       info.ID,
				Name:     info.Name,
				Type:     info.Type,
				IsActive: info.IsActive,
			})
		}
		a.store.SetDevices(rawDevices)
		// Forward to overlay to update its local device list.
		if a.deviceOverlayOpen {
			updated, _ := a.devicePane.Update(m)
			if dp, ok := updated.(*panes.DeviceOverlay); ok {
				a.devicePane = dp
			}
		}
		return a, nil

	case panes.TransferPlaybackMsg:
		// User selected a device; show info toast and dispatch transfer API call.
		a.deviceOverlayOpen = false
		return a, tea.Batch(
			a.buildTransferPlaybackCmd(m.DeviceID),
			a.alerts.NewAlertCmd("info", fmt.Sprintf("Switching to %s...", m.DeviceName)),
		)

	case panes.DeviceTransferredMsg:
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil
			}
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

	return a, nil
}

// handlePrefsMsg routes preference-related messages. Called from handleMsg switch
// for prefsFlushTickMsg, prefs.FlushedMsg, and panes.VisualizerPatternChangedMsg.
//
// prefsFlushTickMsg: debounce timer fired — flush only if generation matches.
// prefs.FlushedMsg: log error to stderr on failure (non-critical, no toast).
// panes.VisualizerPatternChangedMsg: persist new visualizer index via PreferenceStore.
func (a *App) handlePrefsMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch m := msg.(type) {
	case prefsFlushTickMsg:
		// Only flush if no newer preference change has been made (debounce).
		if m.generation == a.prefsDirtyGen {
			return a, a.prefs.FlushCmd(), true
		}
		return a, nil, true

	case prefs.FlushedMsg:
		if m.Err != nil {
			// Non-critical — log to stderr; no user toast (a failed flush is invisible).
			// Re-queue retry: re-queued changes sit in pending map (done by FlushCmd on
			// error), and schedulePrefsFlush arms a new debounce timer to flush them.
			fmt.Fprintf(os.Stderr, "spotnik: prefs flush failed: %v\n", m.Err)
			return a, a.schedulePrefsFlush(), true
		}
		return a, nil, true

	case panes.VisualizerPatternChangedMsg:
		// User cycled the visualizer pattern — persist via PreferenceStore.
		a.prefs.Set("visualizer", m.PatternIndex)
		return a, a.schedulePrefsFlush(), true
	}
	return a, nil, false
}
