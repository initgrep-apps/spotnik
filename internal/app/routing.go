// Package app — key and message routing helpers extracted from app.go.
// These methods handle the routing logic within Update() so that app.go stays
// focused on the Bubble Tea lifecycle (Init/Update/View structure) rather than
// the full dispatch table.
package app

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
)

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

// handleKeyMsg routes a keyboard event through all overlay and view guards before
// dispatching to the focused pane. Extracted from Update() to keep that function readable.
func (a *App) handleKeyMsg(m tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if pp, ok := updatedPane.(*panes.NowPlayingPane); ok {
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
		if pp, ok := updatedPane.(*panes.NowPlayingPane); ok {
			a.playerPane = pp
		}
		return a, cmd
	}
}

// routePlaylistMsg handles playlist-specific messages that may arrive regardless of
// which view is currently active. Returns (model, cmd, true) when handled, (nil, nil, false) otherwise.
func (a *App) routePlaylistMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch m := msg.(type) {
	case panes.FetchPlaylistTracksRequestMsg:
		return a, a.buildFetchPlaylistTracksCmd(m.PlaylistID), true

	case panes.PlaylistTracksLoadedMsg:
		// Write playlist tracks to store from Msg payload (Elm Architecture: only Update writes store).
		if m.Err != nil {
			if errors.Is(m.Err, errNilClient) {
				return a, nil, true
			}
			a.store.SetPlaylistsError(m.Err)
			if a.playlistPane != nil {
				updated, _ := a.playlistPane.Update(m)
				if pm, ok := updated.(*panes.PlaylistManager); ok {
					a.playlistPane = pm
				}
			}
			return a, a.alerts.NewAlertCmd("error", "Failed to load playlist tracks. Press Enter to retry"), true
		}
		a.store.ClearPlaylistsError()
		a.store.SetPlaylistTracks(m.PlaylistID, m.Tracks)
		// Forward to playlist pane so it can refresh from store.
		if a.playlistPane != nil {
			updated, cmd := a.playlistPane.Update(m)
			if pm, ok := updated.(*panes.PlaylistManager); ok {
				a.playlistPane = pm
			}
			return a, cmd, true
		}
		return a, nil, true

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
			if a.playlistPane != nil {
				updated, _ := a.playlistPane.Update(m)
				if pm, ok := updated.(*panes.PlaylistManager); ok {
					a.playlistPane = pm
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
		if a.playlistPane != nil {
			updated, cmd := a.playlistPane.Update(m)
			if pm, ok := updated.(*panes.PlaylistManager); ok {
				a.playlistPane = pm
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
		if a.playlistPane != nil {
			updated, cmd := a.playlistPane.Update(m)
			if pm, ok := updated.(*panes.PlaylistManager); ok {
				a.playlistPane = pm
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
