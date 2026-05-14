// Package app — key and message routing helpers extracted from app.go.
// These methods handle the routing logic within Update() so that app.go stays
// focused on the Bubble Tea lifecycle (Init/Update/View structure) rather than
// the full dispatch table.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// toggleKeyMap maps rune keys '1'-'8' to their corresponding Music page PaneID.
// This is used for btop-style pane visibility toggling on the Music page.
var toggleKeyMap = map[rune]layout.PaneID{
	'1': layout.PaneNowPlaying,
	'2': layout.PaneQueue,
	'3': layout.PanePlaylists,
	'4': layout.PaneAlbums,
	'5': layout.PaneLikedSongs,
	'6': layout.PaneRecentlyPlayed,
	'7': layout.PaneTopTracks,
	'8': layout.PaneTopArtists,
}

// statsToggleKeyMap maps rune keys '1'-'5' to Stats page PaneIDs.
// This is used for btop-style pane visibility toggling on the Stats page.
var statsToggleKeyMap = map[rune]layout.PaneID{
	'1': layout.PaneNowPlaying,
	'2': layout.PaneGatewayHealth,
	'3': layout.PanePollingTraffic,
	'4': layout.PaneGatewayLive,
	'5': layout.PaneNetworkLog,
}

// isPlaybackKey returns true for keys that control playback regardless of focus.
// NOTE: Bubbletea v0.27 delivers Space as tea.KeySpace (not a rune), so tea.KeySpace
// is checked here. "n" was removed — → (tea.KeyRight) is the sole next-track binding.
func isPlaybackKey(m tea.KeyMsg) bool {
	if m.Type == tea.KeyRunes {
		switch string(m.Runes) {
		case "+", "-", "s", "r", "v":
			return true
		}
	}
	return m.Type == tea.KeyLeft || m.Type == tea.KeyRight || m.Type == tea.KeySpace
}

// isPremiumOnlyPlaybackKey returns true for playback keys that require Spotify Premium.
// 'v' (visualizer cycle) is excluded — it is a local UI action that makes no API call.
// NOTE: Space is included here because play/pause requires an active Spotify session.
// "n" was removed — → (tea.KeyRight) is the sole next-track binding.
func isPremiumOnlyPlaybackKey(m tea.KeyMsg) bool {
	if m.Type == tea.KeyRunes {
		switch string(m.Runes) {
		case "+", "-", "s", "r":
			return true
		}
	}
	return m.Type == tea.KeyLeft || m.Type == tea.KeyRight || m.Type == tea.KeySpace
}

// handleKeyMsg routes a keyboard event through all overlay and view guards before
// dispatching to the focused pane. Extracted from Update() to keep that function readable.
func (a *App) handleKeyMsg(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When theme switcher is open, route all keys to it.
	if a.showThemeSwitcher && a.themeOverlay != nil {
		updated, cmd := a.themeOverlay.Update(m)
		if to, ok := updated.(*panes.ThemeOverlay); ok {
			a.themeOverlay = to
		}
		return a, cmd
	}

	// When help overlay is open, route all keys to it.
	if a.helpOpen && a.helpOverlay != nil {
		updated, cmd := a.helpOverlay.Update(m)
		if ho, ok := updated.(*panes.HelpOverlay); ok {
			a.helpOverlay = ho
		}
		return a, cmd
	}

	// When device overlay is open, route all keys to the device pane.
	if a.deviceOverlayOpen {
		updated, cmd := a.devicePane.Update(m)
		if dp, ok := updated.(*panes.DeviceOverlay); ok {
			a.devicePane = dp
		}
		return a, cmd
	}

	// When profile overlay is open, route all keys to the profile pane.
	if a.profileOverlayOpen {
		updated, cmd := a.profilePane.Update(m)
		if pp, ok := updated.(*panes.ProfileOverlay); ok {
			a.profilePane = pp
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

	// During onboarding, route all keys through the step-aware handler.
	// This covers both first-launch (stepRegister) and returning users (stepOAuth).
	if a.currentView == viewOnboarding {
		return a.handleOnboardingKey(m)
	}

	// When a pane's filter is active, route all keys directly to it.
	// This prevents global shortcuts (q, /, d, etc.) from firing while typing.
	focusedID := a.layout.FocusedPane()
	if pane, ok := a.panes[focusedID]; ok {
		if fp, ok := pane.(layout.FilterablePane); ok && fp.HasActiveFilter() {
			updated, cmd := pane.Update(m)
			if lp, ok := updated.(layout.Pane); ok {
				a.panes[focusedID] = lp
			}
			return a, cmd
		}
	}

	// Global: q always quits.
	if m.Type == tea.KeyRunes && string(m.Runes) == "q" {
		return a, tea.Quit
	}

	// '/' opens the search overlay from any pane.
	if m.Type == tea.KeyRunes && string(m.Runes) == "/" {
		return a.openSearch()
	}
	// 'd' opens the device switcher overlay from any pane.
	if m.Type == tea.KeyRunes && string(m.Runes) == "d" {
		return a.openDeviceOverlay()
	}
	// 'u' opens the user profile overlay — only if no other overlay is already open.
	if m.Type == tea.KeyRunes && string(m.Runes) == "u" {
		if !a.searchOpen && !a.deviceOverlayOpen && !a.showThemeSwitcher && !a.helpOpen {
			return a.openProfileOverlay()
		}
		return a, nil
	}

	// 't' opens the theme switcher overlay — but only if no other overlay is already open.
	if m.Type == tea.KeyRunes && string(m.Runes) == "t" {
		if !a.searchOpen && !a.deviceOverlayOpen && !a.showThemeSwitcher && !a.profileOverlayOpen {
			return a.openThemeSwitcher()
		}
		return a, nil
	}

	// '?' opens the help overlay — but only if no other overlay is already open.
	if m.Type == tea.KeyRunes && string(m.Runes) == "?" {
		if !a.searchOpen && !a.deviceOverlayOpen && !a.showThemeSwitcher && !a.helpOpen && !a.profileOverlayOpen {
			return a.openHelp()
		}
		return a, nil
	}

	// '0' toggles between Music and Stats.
	if m.Type == tea.KeyRunes && string(m.Runes) == "0" {
		a.layout.TogglePage()
		a.propagateSizes()
		a.syncFocus()
		return a, nil
	}

	// 'p' cycles presets within the current page.
	if m.Type == tea.KeyRunes && string(m.Runes) == "p" {
		a.layout.CyclePreset()
		a.propagateSizes()
		a.syncFocus()
		a.prefs.Set("preset", a.layout.ActivePresetIndex())
		return a, a.schedulePrefsFlush()
	}

	// '1'-'8' (Music) or '1'-'5' (Stats) toggle pane visibility.
	if m.Type == tea.KeyRunes && len(m.Runes) == 1 {
		keyMap := toggleKeyMap
		if a.layout.ActivePage() == layout.PageStats {
			keyMap = statsToggleKeyMap
		}
		if id, ok := keyMap[m.Runes[0]]; ok {
			a.layout.TogglePane(id)
			a.propagateSizes()
			a.syncFocus()
			return a, nil
		}
	}

	// Tab rotates focus forward.
	if m.Type == tea.KeyTab {
		a.layout.RotateFocus(true)
		a.syncFocus()
		return a, nil
	}
	// Shift+Tab rotates focus backward.
	if m.Type == tea.KeyShiftTab {
		a.layout.RotateFocus(false)
		a.syncFocus()
		return a, nil
	}

	// Playback keys always go to the NowPlaying pane regardless of focus.
	// Temporarily enable focus so the pane handles the key even when it isn't focused.
	if isPlaybackKey(m) {
		// Gate: free-tier users are blocked from Premium-only API operations.
		// 'v' (visualizer cycle) is exempt — it is a local UI action, not an API call.
		if isPremiumOnlyPlaybackKey(m) && !a.store.IsPremium() {
			return a, a.toasts.Cmd(uikit.Toast{
				Intent: uikit.ToastWarning,
				Title:  "Spotify Premium required",
			})
		}
		np := a.nowPlayingPane()
		if np == nil {
			return a, nil
		}
		wasFocused := np.IsFocused()
		if !wasFocused {
			np.SetFocused(true)
		}
		updatedPane, cmd := np.Update(m)
		if pp, ok := updatedPane.(*panes.NowPlayingPane); ok {
			a.panes[layout.PaneNowPlaying] = pp
			np = pp
		}
		if !wasFocused {
			np.SetFocused(false)
		}
		return a, cmd
	}

	// Route remaining keys to the focused pane.
	// focusedID was computed above for the filter guard; reuse it here.
	pane, ok := a.panes[focusedID]
	if !ok {
		return a, nil
	}
	updated, cmd := pane.Update(m)
	if lp, ok := updated.(layout.Pane); ok {
		a.panes[focusedID] = lp
	}
	return a, cmd
}

// handleMouseMsg handles tea.MouseMsg events.
// Only mouse wheel scroll events are handled — clicks and motion are ignored.
// Scroll events are converted to j/k key messages and routed to the pane under
// the cursor via PaneAt(), WITHOUT changing keyboard focus (btop behavior).
// Mouse scroll is ignored when any overlay is open.
func (a *App) handleMouseMsg(m tea.MouseMsg) tea.Cmd {
	// Ignore mouse events when any overlay is open.
	// Overlays handle their own input; scroll behind them is unintuitive.
	if a.deviceOverlayOpen || a.searchOpen || a.helpOpen || a.profileOverlayOpen {
		return nil
	}

	// Only handle wheel scroll events on press action.
	if m.Action != tea.MouseActionPress {
		return nil
	}
	if m.Button != tea.MouseButtonWheelUp && m.Button != tea.MouseButtonWheelDown {
		return nil
	}

	// Hit-test: which pane is under the cursor?
	targetID := a.layout.PaneAt(m.X, m.Y)
	if targetID < 0 {
		return nil // header, status bar, or outside any pane
	}
	target, ok := a.panes[targetID]
	if !ok {
		return nil
	}

	// Convert mouse scroll to the equivalent keyboard scroll message.
	// Wheel-up → 'k' (scroll up), Wheel-down → 'j' (scroll down).
	var scrollMsg tea.KeyMsg
	if m.Button == tea.MouseButtonWheelUp {
		scrollMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	} else {
		scrollMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	}

	// Route scroll to the target pane WITHOUT changing keyboard focus.
	updated, cmd := target.Update(scrollMsg)
	if lp, ok := updated.(layout.Pane); ok {
		a.panes[targetID] = lp
	}
	return cmd
}

// routePlaylistMsg handles playlist-specific messages that may arrive regardless of
// which view is currently active. Returns (model, cmd, true) when handled, (nil, nil, false) otherwise.
func (a *App) routePlaylistMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch m := msg.(type) {
	case panes.FetchPlaylistTracksRequestMsg:
		// Cancel any prior in-flight fetch (user switched playlists or re-entered same one).
		a.playlistTracksCancel()
		ctx, cancel := context.WithCancel(context.Background())
		a.playlistTracksCancel = cancel
		a.playlistTracksID = m.PlaylistID
		return a, a.buildFetchPlaylistTracksCmd(ctx, m.PlaylistID, m.Offset), true

	case panes.PlaylistTracksLoadedMsg:
		// Staleness gate: discard if the user has already switched to a different playlist.
		if m.PlaylistID != a.playlistTracksID {
			return a, nil, true
		}
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				// Forward to pane so it clears tracksFetching — even though we won't toast.
				return a, a.forwardToPane(layout.PanePlaylists, m), true
			}
			var forbiddenErr *api.ForbiddenError
			if errors.As(m.Err, &forbiddenErr) {
				toast := a.errorMapper.Map(uikit.OpPlaylistTracks, m.Err)
				return a, tea.Batch(
					a.forwardToPane(layout.PanePlaylists, m),
					a.toasts.Cmd(toast),
				), true
			}
			return a, tea.Batch(
				a.forwardToPane(layout.PanePlaylists, m),
				a.toasts.Cmd(uikit.Toast{
					Intent: uikit.ToastError,
					Title:  "Failed to load playlist tracks",
					Body:   string(uikit.RecoveryPressEnterRetry),
				}),
			), true
		}
		// Forward to pane — pane owns the data, not the store.
		return a, a.forwardToPane(layout.PanePlaylists, m), true

	case panes.PlaylistTrackViewClosedMsg:
		// User pressed Esc — cancel any in-flight fetch.
		a.playlistTracksCancel()
		a.playlistTracksCancel = func() {}
		a.playlistTracksID = ""
		return a, nil, true

	case userProfileLoadedMsg:
		// Forward result to profile overlay when open so it can show/clear error state.
		// errNilClient is a programming error (nil userAPI) — don't surface it to the
		// overlay, which would show a misleading "Check your connection" message.
		overlayErr := m.err
		if errors.Is(m.err, errNilClient) {
			overlayErr = nil
		}
		var overlayCmd tea.Cmd
		if a.profileOverlayOpen && a.profilePane != nil {
			updated, cmd := a.profilePane.Update(panes.UserProfileLoadedMsg{Err: overlayErr})
			if pu, ok := updated.(*panes.ProfileOverlay); ok {
				a.profilePane = pu
			}
			overlayCmd = cmd
		}

		if m.err != nil {
			if errors.Is(m.err, errNilClient) {
				// Programming error — userAPI was nil at startup; log to stderr but no toast.
				fmt.Fprintf(os.Stderr, "spotnik: userProfileLoadedMsg: userAPI is nil — profile fetch skipped\n")
				return a, overlayCmd, true
			}
			var forbiddenErr *api.ForbiddenError
			if errors.As(m.err, &forbiddenErr) {
				toast := a.errorMapper.Map(uikit.OpPlayback, m.err)
				return a, tea.Batch(overlayCmd, a.toasts.Cmd(toast)), true
			}
			// Surface the failure so the user knows ownership detection is degraded.
			return a, tea.Batch(overlayCmd, a.toasts.Cmd(uikit.Toast{
				Intent: uikit.ToastWarning,
				Title:  "Profile load failed",
				Body:   "Playlist ownership markers may be incorrect.",
			})), true
		}
		if m.profile.ID != "" {
			a.store.SetUserProfile(m.profile)
			// Refresh playlist rows so the ~ prefix appears immediately.
			playlistCmd := a.forwardToPane(layout.PanePlaylists, panes.UserProfileReadyMsg{})
			return a, tea.Batch(overlayCmd, playlistCmd), true
		}
		fmt.Fprintf(os.Stderr, "spotnik: userProfileLoadedMsg: profile loaded with empty ID (unexpected)\n")
		return a, tea.Batch(overlayCmd, a.toasts.Cmd(uikit.Toast{
			Intent: uikit.ToastWarning,
			Title:  "Profile load failed",
			Body:   "Playlist ownership markers may be incorrect.",
		})), true

	case panes.PlaylistAccessDeniedMsg:
		return a, a.toasts.Cmd(uikit.Toast{
			Intent: uikit.ToastWarning,
			Title:  "Playlist access denied",
			Body:   "Track access limited to playlists you own or collaborate on.",
		}), true

	case panes.PlaylistRemoveRequestMsg:
		return a, a.buildRemovePlaylistTrackCmd(m.PlaylistID, m.TrackURI), true

	case panes.PlaylistRemoveResultMsg:
		if m.Err != nil && errors.Is(m.Err, errNilClient) {
			return a, nil, true
		}
		if pp := a.playlistsPane(); pp != nil {
			updated, cmd := pp.Update(m)
			if ppu, ok := updated.(*panes.PlaylistsPane); ok {
				a.panes[layout.PanePlaylists] = ppu
			}
			if m.Err != nil {
				if errors.Is(m.Err, errNilClient) {
					return a, nil, true
				}
				toast := a.errorMapper.Map(uikit.OpPlaylists, m.Err)
				if toast.Intent == uikit.ToastNone {
					return a, cmd, true
				}
				return a, tea.Batch(cmd, a.toasts.Cmd(toast)), true
			}
			return a, cmd, true
		}
		return a, nil, true

	}

	return nil, nil, false
}

// routeAlbumMsg handles album track sub-view messages that may arrive regardless of
// which view is currently active. Returns (model, cmd, true) when handled, (nil, nil, false) otherwise.
func (a *App) routeAlbumMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch m := msg.(type) {
	case panes.FetchAlbumTracksRequestMsg:
		// Cancel any prior in-flight fetch (user switched albums or re-entered same one).
		a.albumTracksCancel()
		ctx, cancel := context.WithCancel(context.Background())
		a.albumTracksCancel = cancel
		a.albumTracksID = m.AlbumID
		return a, a.buildFetchAlbumTracksCmd(ctx, m.AlbumID, m.Offset), true

	case panes.AlbumTracksLoadedMsg:
		// Staleness gate: discard if the user has already switched to a different album.
		if m.AlbumID == "" || m.AlbumID != a.albumTracksID {
			return a, nil, true
		}
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				// Forward to pane so it clears tracksFetching — even though we won't toast.
				return a, a.forwardToPane(layout.PaneAlbums, m), true
			}
			var forbiddenErr *api.ForbiddenError
			if errors.As(m.Err, &forbiddenErr) {
				toast := a.errorMapper.Map(uikit.OpAlbums, m.Err)
				return a, tea.Batch(
					a.forwardToPane(layout.PaneAlbums, m),
					a.toasts.Cmd(toast),
				), true
			}
			return a, tea.Batch(
				a.forwardToPane(layout.PaneAlbums, m),
				a.toasts.Cmd(uikit.Toast{
					Intent: uikit.ToastError,
					Title:  "Failed to load album tracks",
					Body:   string(uikit.RecoveryPressEnterRetry),
				}),
			), true
		}
		// Forward to pane — pane owns the data, not the store.
		return a, a.forwardToPane(layout.PaneAlbums, m), true

	case panes.AlbumTrackViewClosedMsg:
		// User pressed Esc — cancel any in-flight fetch.
		a.albumTracksCancel()
		a.albumTracksCancel = func() {}
		a.albumTracksID = ""
		return a, nil, true
	}

	return nil, nil, false
}

// handleOnboardingKey routes key events during viewOnboarding.
// q and Ctrl+C quit from any step; all other keys are step-specific.
func (a *App) handleOnboardingKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	// q or Ctrl+C quits from any onboarding step.
	// Close the callback server before quitting so it doesn't leak.
	if m.Type == tea.KeyCtrlC || (m.Type == tea.KeyRunes && string(m.Runes) == "q") {
		a.onboardingClose()
		return a, tea.Quit
	}

	switch a.onboardingStep {
	case stepRegister:
		// 'c' copies the redirect URI — only when the input field is empty.
		// Once the user starts typing, treat 'c' as ordinary input so they can edit freely.
		if m.Type == tea.KeyRunes && string(m.Runes) == "c" && a.onboardingField.Value() == "" {
			return a, copyToClipboardCmd(fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort))
		}
		// Enter with non-empty input → validate via FormField then save.
		if m.Type == tea.KeyEnter {
			clientID := strings.TrimSpace(a.onboardingField.Value())
			if clientID == "" {
				return a, nil
			}
			if err := a.onboardingField.Validate(); err != nil {
				// Error is now cached in the field; Render() will display it.
				return a, nil
			}
			return a, saveClientIDCmd(config.DefaultConfigPath(), clientID)
		}
		// All other keys → delegate to the FormField (clears stale error on SetValue,
		// but here we let the field process the key event directly).
		var cmd tea.Cmd
		a.onboardingField, cmd = a.onboardingField.Update(m)
		return a, cmd

	case stepOAuth:
		// While the permissions overlay is open, route all keys to it.
		// Esc round-trips through OnboardingPermissionsOverlayClosedMsg.
		if a.onboardingPermissionsOverlay != nil {
			updated, cmd := a.onboardingPermissionsOverlay.Update(m)
			if po, ok := updated.(*panes.OnboardingPermissionsOverlay); ok {
				a.onboardingPermissionsOverlay = po
			}
			return a, cmd
		}
		// 'v' opens the permissions overlay only on stepOAuth.
		if m.Type == tea.KeyRunes && string(m.Runes) == "v" {
			return a.openOnboardingPermissions()
		}
		// c → copy auth URL to clipboard; toast is emitted by the clipboardCopiedMsg handler.
		if m.Type == tea.KeyRunes && string(m.Runes) == "c" {
			return a, copyToClipboardCmd(a.onboardingAuthURL)
		}
		return a, nil

	case stepError:
		if m.Type == tea.KeyRunes {
			switch string(m.Runes) {
			case "r":
				// Emit retry message; the onboardingRetryMsg handler resets to stepRegister.
				return a, func() tea.Msg { return onboardingRetryMsg{} }
			case "l":
				// Re-launch OAuth without resetting the client ID.
				a.onboardingStep = stepOAuth
				return a, tea.Batch(
					prepareOAuthCmd(a.clientID, a.onboardingPort, a.onboardingCodeCh),
					a.onboardingSpinner.Init(),
				)
			}
		}
		return a, nil
	}

	return a, nil
}
