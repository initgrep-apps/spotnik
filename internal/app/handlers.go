// central message dispatch for the root Bubble Tea model — called by
// Update() for every non-key, non-mouse message; routes data-carrying Msg payloads to
// Store writes and returns follow-up commands.
package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/components/viz"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

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
			switch {
			case a.needsRegister:
				a.currentView = viewOnboarding
				a.onboardingStep = stepRegister
				a.onboardingInput.Focus()
			case a.needsAuth:
				a.currentView = viewAuth
				a.authStatus = "Opening browser for authorization..."
				return a, prepareOAuthCmd(a.clientID, a.onboardingPort, a.onboardingCodeCh)
			default:
				a.currentView = viewGrid
			}
		}
		return a, nil

	case authPreparedMsg:
		a.onboardingAuthURL = m.authURL
		a.authURL = m.authURL
		if m.browserErr != nil {
			a.authStatus = "Browser didn't open. Visit the URL above manually."
		} else {
			a.authStatus = "Waiting for authorization..."
		}
		return a, waitForCallbackCmd(a.clientID, a.tokenStore, m.verifier, m.redirectURI, m.codeCh)

	case authSuccessMsg:
		// Close the callback server — OAuth completed successfully, no retries needed.
		a.onboardingClose()
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
			fetchPlaybackStateCmd(a.player, api.Background),
			tea.Tick(time.Second, func(_ time.Time) tea.Msg {
				return panes.TickMsg{}
			}),
		)
		authCmds = append(authCmds, a.initialFetchCmds()...)
		return a, tea.Batch(authCmds...)

	case authErrorMsg:
		if a.currentView == viewOnboarding {
			a.onboardingStep = stepError
			a.onboardingError = m.err.Error()
			return a, nil
		}
		a.authStatus = fmt.Sprintf("Error: %s — press q to quit", m.err.Error())
		return a, nil

	case onboardingClientIDSavedMsg:
		a.clientID = m.clientID
		a.onboardingStep = stepOAuth
		a.authStatus = "Opening browser for authorization..."
		return a, tea.Batch(
			prepareOAuthCmd(a.clientID, a.onboardingPort, a.onboardingCodeCh),
			a.onboardingSpinner.Tick,
		)

	case onboardingRetryMsg:
		a.onboardingStep = stepRegister
		a.onboardingError = ""
		a.onboardingInput.Reset()
		a.onboardingInput.Focus()
		return a, nil

	case spinner.TickMsg:
		if a.currentView == viewOnboarding && a.onboardingStep == stepOAuth {
			var cmd tea.Cmd
			a.onboardingSpinner, cmd = a.onboardingSpinner.Update(m)
			return a, cmd
		}
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
			// errNilClient is an expected pre-auth startup condition — drop silently.
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
		a.profilePane.SetSize(40, 12) // fixed size — profile card is not resizable
		if a.themeOverlay != nil {
			a.themeOverlay.SetSize(m.Width, m.Height)
		}
		if a.helpOverlay != nil {
			a.helpOverlay.SetSize(m.Width, m.Height)
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
					fetchPlaybackStateCmd(a.player, api.Background),
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
			cmds = append(cmds, fetchPlaybackStateCmd(a.player, api.Background))
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
			var rateLimitErr *api.RateLimitError
			if errors.As(m.Err, &rateLimitErr) {
				// Defense-in-depth: buildPlaybackAPICmd may return a RateLimitError if the
				// gateway rejects the request during active backoff (F27-S126). Emit a
				// distinct "Rate limited" toast rather than the raw error string.
				// NOTE: no fetchPlaybackStateCmd — the request never reached Spotify,
				// so there is no state change to reconcile. Dispatching a Background
				// fetch here would itself be rejected (backoff still active), producing
				// a second toast and noise in the request-flow pane.
				return a, a.alerts.NewAlertCmd("warning",
					fmt.Sprintf("Rate limited — wait %ds before retrying", rateLimitErr.RetryAfter))
			}
			var forbiddenErr *api.ForbiddenError
			if errors.As(m.Err, &forbiddenErr) {
				return a, tea.Batch(
					fetchPlaybackStateCmd(a.player, api.Background),
					a.alerts.NewAlertCmd("warning", "Spotify Premium required"),
				)
			}
			return a, tea.Batch(
				fetchPlaybackStateCmd(a.player, api.Background),
				a.alerts.NewAlertCmd("error", m.Err.Error()),
			)
		}
		// User command succeeded — use Interactive priority so the reconcile GET
		// fires a fresh HTTP call and does not join a pre-command Background poll.
		return a, fetchPlaybackStateCmd(a.player, api.Interactive)

	case panes.PlaybackRequestMsg:
		return a, a.buildPlaybackAPICmd(m.Action)

	case panes.PlayContextMsg:
		// Overlay stays open — only Esc (SearchClosedMsg) closes it.
		return a, a.buildPlayContextCmd(m.ContextURI, m.OffsetURI)

	case panes.PlayTrackListMsg:
		// Overlay stays open — only Esc (SearchClosedMsg) closes it.
		return a, a.buildPlayTrackListCmd(m.URIs)

	case panes.PlayTrackMsg:
		// QueuePane skip-to: play a single track URI directly.
		// Single-URI list is functionally equivalent to the old buildPlayTrackCmd.
		return a, a.buildPlayTrackListCmd([]string{m.TrackURI})

	case panes.AddToQueueMsg:
		// Gate: free-tier users cannot add to queue — block before any API call.
		if !a.store.IsPremium() {
			return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required")
		}
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

	case panes.DeviceOverlayClosedMsg:
		// Device overlay closed via Esc — close overlay.
		return a.closeDeviceOverlay()

	case panes.ProfileOverlayClosedMsg:
		// Profile overlay closed via Esc — clear flag.
		a.profileOverlayOpen = false
		return a, nil

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
		a.profilePane.SetTheme(newTheme)
		if a.themeOverlay != nil {
			a.themeOverlay.SetTheme(newTheme)
		}
		if a.helpOverlay != nil {
			a.helpOverlay.SetTheme(newTheme)
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

	case panes.HelpOverlayClosedMsg:
		// Help overlay closed via Esc — close overlay without any state change.
		return a.closeHelp()

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
		// User selected a device; close overlay first.
		a.deviceOverlayOpen = false
		// Gate: free-tier users cannot transfer playback — block before any API call.
		if !a.store.IsPremium() {
			return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required")
		}
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
				fetchPlaybackStateCmd(a.player, api.Background),
				a.alerts.NewAlertCmd("error", m.Err.Error()),
			)
		}
		// Transfer succeeded — use Interactive priority so the reconcile GET fires
		// a fresh HTTP call and does not join a pre-transfer Background poll.
		return a, fetchPlaybackStateCmd(a.player, api.Interactive)

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

	// Album track sub-view routing — handled regardless of current view.
	if model, cmd, handled := a.routeAlbumMsg(msg); handled {
		return model, cmd
	}

	// Forward any remaining unhandled messages to the playlist and album panes.
	// This ensures pane-internal debounce ticks (playlistDebounceMsg, albumDebounceMsg)
	// reach their panes — those types are unexported so the switch above cannot match them.
	// Both panes' Update methods safely return (p, nil) for messages they don't recognise.
	return a, tea.Batch(
		a.forwardToPane(layout.PanePlaylists, msg),
		a.forwardToPane(layout.PaneAlbums, msg),
	)
}
