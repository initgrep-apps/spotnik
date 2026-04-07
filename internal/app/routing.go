// Package app — key and message routing helpers extracted from app.go.
// These methods handle the routing logic within Update() so that app.go stays
// focused on the Bubble Tea lifecycle (Init/Update/View structure) rather than
// the full dispatch table.
package app

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
)

// toggleKeyMap maps rune keys '1'-'8' to their corresponding PaneID.
// This is used for btop-style pane visibility toggling.
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

// isPlaybackKey returns true for keys that control playback regardless of focus.
func isPlaybackKey(m tea.KeyMsg) bool {
	if m.Type == tea.KeyRunes {
		switch string(m.Runes) {
		case " ", "n", "+", "-", "s", "r", "v":
			return true
		}
	}
	return m.Type == tea.KeyLeft || m.Type == tea.KeyRight
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

	// During auth, only allow quit keys — ignore everything else.
	if a.currentView == viewAuth {
		if m.Type == tea.KeyCtrlC || (m.Type == tea.KeyRunes && string(m.Runes) == "q") || m.Type == tea.KeyEsc {
			return a, tea.Quit
		}
		return a, nil
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

	// 't' opens the theme switcher overlay — but only if no other overlay is already open.
	if m.Type == tea.KeyRunes && string(m.Runes) == "t" {
		if !a.searchOpen && !a.deviceOverlayOpen && !a.showThemeSwitcher {
			return a.openThemeSwitcher()
		}
		return a, nil
	}

	// '0' toggles between Page A and Page B.
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

	// '1'-'8' toggle pane visibility (Page A only).
	if m.Type == tea.KeyRunes && len(m.Runes) == 1 {
		if id, ok := toggleKeyMap[m.Runes[0]]; ok {
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
// Mouse scroll is ignored when an overlay (search or device) is open.
func (a *App) handleMouseMsg(m tea.MouseMsg) tea.Cmd {
	// Ignore mouse events when any overlay is open.
	// Overlays handle their own input; scroll behind them is unintuitive.
	if a.deviceOverlayOpen || a.searchOpen {
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
				return a, tea.Batch(
					a.forwardToPane(layout.PanePlaylists, m),
					a.alerts.NewAlertCmd("warning", "Spotify Premium required or playlist access denied"),
				), true
			}
			return a, tea.Batch(
				a.forwardToPane(layout.PanePlaylists, m),
				a.alerts.NewAlertCmd("error", "Failed to load playlist tracks. Press Enter to retry"),
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
		if m.err != nil {
			if errors.Is(m.err, errNilClient) {
				// Programming error — userAPI was nil at startup; no toast.
				return a, nil, true
			}
			// Surface the failure so the user knows ownership detection is degraded.
			return a, a.alerts.NewAlertCmd("warning",
				"Could not load your Spotify profile. Playlist ownership markers may be incorrect."), true
		}
		if m.userID != "" {
			a.store.SetUserID(m.userID)
			// Refresh playlist rows so the ~ prefix appears immediately.
			return a, a.forwardToPane(layout.PanePlaylists, panes.UserProfileReadyMsg{}), true
		}
		return a, nil, true

	case panes.PlaylistAccessDeniedMsg:
		return a, a.alerts.NewAlertCmd("warning", "Track access limited to playlists you own or collaborate on"), true

	case panes.PlaylistCreateRequestMsg:
		return a, a.buildCreatePlaylistCmd(m.Name, m.Description), true

	case panes.PlaylistCreatedMsg:
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil, true
			}
			return a, a.alerts.NewAlertCmd("error", m.Err.Error()), true
		}
		// Re-fetch playlists so the new one appears.
		return a, a.buildFetchPlaylistsCmd(0), true

	case panes.PlaylistRenameRequestMsg:
		return a, a.buildRenamePlaylistCmd(m.PlaylistID, m.NewName), true

	case panes.PlaylistRenamedMsg:
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil, true
			}
			if pp := a.playlistsPane(); pp != nil {
				updated, _ := pp.Update(m)
				if ppu, ok := updated.(*panes.PlaylistsPane); ok {
					a.panes[layout.PanePlaylists] = ppu
				}
			}
			return a, a.alerts.NewAlertCmd("error", m.Err.Error()), true
		}
		// Re-fetch playlists to reflect rename.
		return a, a.buildFetchPlaylistsCmd(0), true

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
				return a, tea.Batch(cmd, a.alerts.NewAlertCmd("error", m.Err.Error())), true
			}
			return a, cmd, true
		}
		return a, nil, true

	case panes.PlaylistReorderRequestMsg:
		return a, a.buildReorderPlaylistTracksCmd(m.PlaylistID, m.RangeStart, m.InsertBefore, m.RangeLength), true

	case panes.PlaylistReorderResultMsg:
		if m.Err != nil && errors.Is(m.Err, errNilClient) {
			return a, nil, true
		}
		if pp := a.playlistsPane(); pp != nil {
			updated, cmd := pp.Update(m)
			if ppu, ok := updated.(*panes.PlaylistsPane); ok {
				a.panes[layout.PanePlaylists] = ppu
			}
			if m.Err != nil {
				return a, tea.Batch(cmd, a.alerts.NewAlertCmd("error", m.Err.Error())), true
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
				return a, tea.Batch(
					a.forwardToPane(layout.PaneAlbums, m),
					a.alerts.NewAlertCmd("warning", "Spotify Premium required"),
				), true
			}
			return a, tea.Batch(
				a.forwardToPane(layout.PaneAlbums, m),
				a.alerts.NewAlertCmd("error", "Failed to load album tracks. Press Enter to retry"),
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
