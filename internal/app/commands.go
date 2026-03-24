// Package app — command factories extracted from app.go.
// All functions in this file are methods on *App or package-level helpers that
// create tea.Cmd values. No routing or state-mutation logic lives here — this
// file is a pure command factory for the Spotify API calls.
package app

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
)

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
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
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
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
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
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
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
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
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
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
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
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			store.SetDevicesError(err)
		} else {
			store.ClearDevicesError()
		}
		// Convert api.Device to panes.DeviceInfo to respect ui/ -> api/ boundary.
		var infos []panes.DeviceInfo
		for _, d := range devList {
			infos = append(infos, panes.DeviceInfo{
				ID:       d.ID,
				Name:     d.Name,
				Type:     d.Type,
				IsActive: d.IsActive,
			})
		}
		return panes.NewDevicesLoadedMsg(infos, err)
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
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			store.SetStatsError(err)
			return panes.StatsLoadedMsg{TimeRange: timeRange}
		}
		artists, err := userAPI.GetTopArtists(ctx, timeRange, 25)
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
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
func fetchQueueCmd(player api.PlayerAPI, store *state.Store) tea.Cmd {
	return func() tea.Msg {
		if player == nil {
			return panes.QueueLoadedMsg{}
		}
		qr, err := player.GetQueue(context.Background())
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
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
func fetchPlaybackStateCmd(player api.PlayerAPI, store *state.Store) tea.Cmd {
	return func() tea.Msg {
		if player == nil {
			return panes.PlaybackStateFetchedMsg{}
		}
		ps, err := player.GetPlaybackState(context.Background())
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			return panes.PlaybackStateFetchedMsg{}
		}
		store.SetPlaybackState(ps)
		return panes.PlaybackStateFetchedMsg{}
	}
}

// parse429RetryAfter checks if err is a RateLimitError and extracts RetryAfter.
// Returns 0 if the error is not a rate limit error.
func parse429RetryAfter(err error) int {
	var rateLimitErr *api.RateLimitError
	if errors.As(err, &rateLimitErr) {
		if rateLimitErr.RetryAfter <= 0 {
			return defaultBackoffTicks
		}
		return rateLimitErr.RetryAfter
	}
	return 0
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
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
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
