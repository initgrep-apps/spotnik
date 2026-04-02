// Package app — command factories extracted from app.go.
// All functions in this file are methods on *App or package-level helpers that
// create tea.Cmd values. No routing or state-mutation logic lives here — this
// file is a pure command factory for the Spotify API calls.
//
// Elm Architecture contract: build*Cmd and fetch*Cmd functions MUST NOT write
// to the Store. All Store mutations happen in Update() when the returned Msg
// is processed. Commands return data in Msg payloads; Update() decides what to
// store.
package app

import (
	"context"
	"errors"
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
)

// errNilClient is returned when a command is built but the required API client is nil.
// This typically means authentication has not completed yet.
// Update() handlers that receive this error skip silently (no toast) — it is an
// expected startup condition, not a user-facing failure.
var errNilClient = fmt.Errorf("API client not initialized")

// buildPlaybackAPICmd dispatches the Spotify API call for the given playback action.
// Store values are snapshotted here in Update() context (which is safe) before the
// closure is returned. The closure must never read from the Store — only use snapshots.
// Reading Store inside the closure would be a data race because closures execute
// asynchronously while Update() may be writing.
func (a *App) buildPlaybackAPICmd(action panes.PlaybackAction) tea.Cmd {
	if a.player == nil {
		return func() tea.Msg { return panes.PlaybackCmdSentMsg{Err: errNilClient} }
	}
	player := a.player
	volStep := a.volumeStep

	// Snapshot store values in Update() context (thread-safe).
	// The closure uses these captured values instead of calling store.PlaybackState() later.
	ps := a.store.PlaybackState()
	currentVolume := 65 // default when no device info is available
	isShuffled := false
	repeatMode := "off"
	if ps != nil {
		if ps.Device != nil {
			currentVolume = ps.Device.VolumePercent
		}
		isShuffled = ps.ShuffleState
		repeatMode = ps.RepeatState
	}

	return func() tea.Msg {
		// Playback controls are user-triggered — bypass token bucket.
		ctx := api.WithPriority(context.Background(), api.Interactive)
		var err error

		switch action {
		case panes.ActionPause:
			err = player.Pause(ctx)
		case panes.ActionPlay:
			err = player.Play(ctx, domain.PlayOptions{})
		case panes.ActionNext:
			err = player.Next(ctx)
		case panes.ActionPrevious:
			err = player.Previous(ctx)
		case panes.ActionVolumeUp:
			newVol := currentVolume + volStep
			if newVol > 100 {
				newVol = 100
			}
			err = player.SetVolume(ctx, newVol)
		case panes.ActionVolumeDown:
			newVol := currentVolume - volStep
			if newVol < 0 {
				newVol = 0
			}
			err = player.SetVolume(ctx, newVol)
		case panes.ActionToggleShuffle:
			err = player.SetShuffle(ctx, !isShuffled)
		case panes.ActionCycleRepeat:
			err = player.SetRepeat(ctx, nextRepeatMode(repeatMode))
		}

		if err != nil {
			if secs := parse429RetryAfter(err); secs > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: secs}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
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
			return panes.PlaybackCmdSentMsg{Err: errNilClient}
		}
		err := player.Play(api.WithPriority(context.Background(), api.Interactive), domain.PlayOptions{ContextURI: contextURI})
		if err != nil {
			if secs := parse429RetryAfter(err); secs > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: secs}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
		}
		return panes.PlaybackCmdSentMsg{Err: err}
	}
}

// buildPlayTrackCmd dispatches a play command for a specific track URI.
func (a *App) buildPlayTrackCmd(trackURI string) tea.Cmd {
	player := a.player
	return func() tea.Msg {
		if player == nil {
			return panes.PlaybackCmdSentMsg{Err: errNilClient}
		}
		err := player.Play(api.WithPriority(context.Background(), api.Interactive), domain.PlayOptions{URIs: []string{trackURI}})
		if err != nil {
			if secs := parse429RetryAfter(err); secs > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: secs}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
		}
		return panes.PlaybackCmdSentMsg{Err: err}
	}
}

// buildFetchPlaylistsCmd creates a command that fetches playlists and returns them
// in the LibraryLoadedMsg payload. No Store writes occur in the command — Update()
// handles pagination and writes to the store.
func (a *App) buildFetchPlaylistsCmd(offset int) tea.Cmd {
	library := a.library
	return func() tea.Msg {
		if library == nil {
			return panes.LibraryLoadedMsg{Offset: offset, Err: errNilClient}
		}
		playlists, err := library.Playlists(context.Background(), 50, offset)
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			return panes.LibraryLoadedMsg{Offset: offset, Err: err}
		}
		return panes.LibraryLoadedMsg{Items: playlists, Offset: offset}
	}
}

// buildFetchAlbumsCmd creates a command that fetches saved albums and returns them
// in AlbumsLoadedMsg. No Store writes occur — Update() writes to the store.
func (a *App) buildFetchAlbumsCmd(offset int) tea.Cmd {
	library := a.library
	return func() tea.Msg {
		if library == nil {
			return panes.AlbumsLoadedMsg{Offset: offset, Err: errNilClient}
		}
		albums, err := library.SavedAlbums(context.Background(), 50, offset)
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			return panes.AlbumsLoadedMsg{Offset: offset, Err: err}
		}
		return panes.AlbumsLoadedMsg{Items: albums, Offset: offset}
	}
}

// buildFetchLikedTracksCmd creates a command that fetches liked tracks and returns them
// in LikedTracksLoadedMsg. No Store writes occur — Update() writes to the store.
func (a *App) buildFetchLikedTracksCmd(offset int) tea.Cmd {
	library := a.library
	return func() tea.Msg {
		if library == nil {
			return panes.LikedTracksLoadedMsg{Offset: offset, Err: errNilClient}
		}
		tracks, err := library.LikedTracks(context.Background(), 50, offset)
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			return panes.LikedTracksLoadedMsg{Offset: offset, Err: err}
		}
		return panes.LikedTracksLoadedMsg{Items: tracks, Offset: offset}
	}
}

// buildFetchRecentlyPlayedCmd creates a command that fetches recently played tracks
// and returns them in RecentlyPlayedLoadedMsg. No Store writes occur — Update() writes.
func (a *App) buildFetchRecentlyPlayedCmd() tea.Cmd {
	library := a.library
	return func() tea.Msg {
		if library == nil {
			return panes.RecentlyPlayedLoadedMsg{Err: errNilClient}
		}
		items, err := library.RecentlyPlayed(context.Background(), 20)
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			return panes.RecentlyPlayedLoadedMsg{Err: err}
		}
		return panes.RecentlyPlayedLoadedMsg{Items: items}
	}
}

// buildAddToQueueCmd creates a command that adds a track to the user's queue.
// trackName is threaded through to AddToQueueResultMsg for status bar display.
func (a *App) buildAddToQueueCmd(trackURI, trackName string) tea.Cmd {
	player := a.player
	return func() tea.Msg {
		if player == nil {
			return panes.AddToQueueResultMsg{Err: errNilClient, TrackName: trackName}
		}
		// Add to queue is user-triggered — bypass token bucket.
		err := player.AddToQueue(api.WithPriority(context.Background(), api.Interactive), trackURI)
		if err != nil {
			if secs := parse429RetryAfter(err); secs > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: secs}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
		}
		return panes.AddToQueueResultMsg{Err: err, TrackName: trackName}
	}
}

// buildSearchCmd creates a command that calls the Spotify search API and delivers
// pre-converted results via SearchResultsMsg so search.go never imports api/.
// NOTE: store.SetSearchQuery and store.SetSearchLoading are called by Update()
// before this command is dispatched — not inside the closure.
func (a *App) buildSearchCmd(query string) tea.Cmd {
	search := a.search
	return func() tea.Msg {
		if search == nil {
			return panes.SearchResultsMsg{Err: errNilClient}
		}
		// Search is user-triggered (debounce fires after keypress) — bypass token bucket.
		results, err := search.Search(
			api.WithPriority(context.Background(), api.Interactive),
			query,
			[]string{"track", "artist", "album", "playlist"},
			5,
		)
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			return panes.SearchResultsMsg{Err: err}
		}
		return panes.SearchResultsMsg{Results: convertSearchResult(results)}
	}
}

// convertSearchResult converts *api.SearchResult (= *domain.SearchResult) to *panes.SearchResultData,
// extracting only the fields the UI needs. This is the sole place where search
// types cross the app/ui boundary.
func convertSearchResult(r *api.SearchResult) *panes.SearchResultData {
	if r == nil {
		return nil
	}

	data := &panes.SearchResultData{}

	for _, t := range r.Tracks.Items {
		item := panes.SearchTrackItem{
			URI:  t.URI,
			Name: t.Name,
		}
		if len(t.Artists) > 0 {
			item.Artist = t.Artists[0].Name
		}
		data.Tracks = append(data.Tracks, item)
	}

	for _, a := range r.Artists.Items {
		data.Artists = append(data.Artists, panes.SearchArtistItem{
			URI:  a.URI,
			Name: a.Name,
		})
	}

	for _, a := range r.Albums.Items {
		item := panes.SearchAlbumItem{
			URI:  a.URI,
			Name: a.Name,
		}
		if len(a.Artists) > 0 {
			item.Artist = a.Artists[0].Name
		}
		data.Albums = append(data.Albums, item)
	}

	for _, p := range r.Playlists.Items {
		data.Playlists = append(data.Playlists, panes.SearchPlaylistItem{
			URI:   p.URI,
			Name:  p.Name,
			Owner: p.Owner.DisplayName,
		})
	}

	return data
}

// buildFetchDevicesCmd creates a command that fetches the available Spotify Connect devices
// and delivers them back via DevicesLoadedMsg. Store mutations (error state, fetchedAt stamp)
// are handled by app.Update() when it receives DevicesLoadedMsg — not inside this command.
func (a *App) buildFetchDevicesCmd() tea.Cmd {
	devices := a.devices
	return func() tea.Msg {
		if devices == nil {
			return panes.DevicesLoadedMsg{Err: errNilClient}
		}
		ctx := api.WithPriority(context.Background(), api.Interactive)
		devList, err := devices.Devices(ctx)
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			// Non-retryable, non-auth error: deliver error via message.
			return panes.DevicesLoadedMsg{Err: err}
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
		return panes.DevicesLoadedMsg{Devices: infos}
	}
}

// buildTransferPlaybackCmd creates a command that calls the TransferPlayback API
// and returns a DeviceTransferredMsg with any error.
func (a *App) buildTransferPlaybackCmd(deviceID string) tea.Cmd {
	devices := a.devices
	return func() tea.Msg {
		if devices == nil {
			return panes.DeviceTransferredMsg{Err: errNilClient, DeviceID: deviceID}
		}
		// Transfer playback is user-triggered — bypass token bucket.
		err := devices.TransferPlayback(api.WithPriority(context.Background(), api.Interactive), deviceID, true)
		return panes.DeviceTransferredMsg{DeviceID: deviceID, Err: err}
	}
}

// buildToggleLikeCmd creates a command that likes or unlikes a track.
func (a *App) buildToggleLikeCmd(trackID string, unlike bool) tea.Cmd {
	library := a.library
	return func() tea.Msg {
		if library == nil {
			return panes.LikeToggleResultMsg{Err: errNilClient, TrackID: trackID}
		}
		// Like/unlike is user-triggered — bypass token bucket.
		ctx := api.WithPriority(context.Background(), api.Interactive)
		var err error
		if unlike {
			err = library.UnlikeTrack(ctx, trackID)
		} else {
			err = library.LikeTrack(ctx, trackID)
		}
		return panes.LikeToggleResultMsg{TrackID: trackID, Err: err}
	}
}

// buildFetchStatsCmd creates a command that fetches top tracks and top artists
// concurrently for the given time range and returns them in a StatsLoadedMsg payload.
// No Store writes occur — Update() writes data to the store when it receives the Msg.
func (a *App) buildFetchStatsCmd(timeRange string) tea.Cmd {
	userAPI := a.userAPI
	return func() tea.Msg {
		if userAPI == nil {
			// Include TimeRange so the StatsLoadedMsg handler can clear the
			// in-flight sentinel. Without it, the conditional guard
			// "if m.TimeRange != """ skips the clear, leaving the sentinel
			// stuck true and blocking all future fetches for this range.
			return panes.StatsLoadedMsg{TimeRange: timeRange, Err: errNilClient}
		}
		ctx := context.Background()

		// Fetch top tracks and top artists concurrently using sync.WaitGroup.
		// NOTE: errgroup is not used because CLAUDE.md restricts dependencies
		// to stdlib only. sync.WaitGroup provides the same fan-out pattern.
		var wg sync.WaitGroup
		var tracks []domain.Track
		var artists []domain.FullArtist
		var tracksErr, artistsErr error

		wg.Add(2)
		go func() {
			defer wg.Done()
			tracks, tracksErr = userAPI.TopTracks(ctx, timeRange, 25)
		}()
		go func() {
			defer wg.Done()
			artists, artistsErr = userAPI.TopArtists(ctx, timeRange, 25)
		}()
		wg.Wait()

		// Rate limit errors take priority — return immediately so backoff kicks in.
		if tracksErr != nil {
			if retryAfter := parse429RetryAfter(tracksErr); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(tracksErr) {
				return unauthorizedMsg{}
			}
			return panes.StatsLoadedMsg{TimeRange: timeRange, Err: tracksErr}
		}
		if artistsErr != nil {
			if retryAfter := parse429RetryAfter(artistsErr); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(artistsErr) {
				return unauthorizedMsg{}
			}
			return panes.StatsLoadedMsg{TimeRange: timeRange, Err: artistsErr}
		}

		return panes.StatsLoadedMsg{
			TimeRange:  timeRange,
			TopTracks:  tracks,
			TopArtists: artists,
		}
	}
}

// fetchQueueCmd creates a command that fetches the current play queue and returns
// the tracks in the QueueLoadedMsg payload. No Store writes occur — Update() writes.
func fetchQueueCmd(player api.PlayerAPI) tea.Cmd {
	return func() tea.Msg {
		if player == nil {
			return panes.QueueLoadedMsg{Err: errNilClient}
		}
		qr, err := player.Queue(context.Background())
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			return panes.QueueLoadedMsg{Err: err}
		}
		if qr != nil {
			return panes.QueueLoadedMsg{Tracks: qr.Queue}
		}
		return panes.QueueLoadedMsg{}
	}
}

// fetchPlaybackStateCmd creates a command that fetches the current playback state
// and returns it in the PlaybackStateFetchedMsg payload. No Store writes occur —
// Update() writes State to the store when the Msg is received.
func fetchPlaybackStateCmd(player api.PlayerAPI) tea.Cmd {
	return func() tea.Msg {
		if player == nil {
			return panes.PlaybackStateFetchedMsg{Err: errNilClient}
		}
		ps, err := player.PlaybackState(context.Background())
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			return panes.PlaybackStateFetchedMsg{Err: err}
		}
		return panes.PlaybackStateFetchedMsg{State: ps}
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

// isUnauthorizedError returns true if err is a *api.UnauthorizedError (401).
func isUnauthorizedError(err error) bool {
	var unauthErr *api.UnauthorizedError
	return errors.As(err, &unauthErr)
}

// buildRefreshTokenCmd creates a command that attempts to refresh the access token.
// If the token store is nil or has no refresh token, it immediately returns
// a tokenRefreshedMsg with an error. Otherwise it calls api.Refresh and
// returns the new access token on success.
func buildRefreshTokenCmd(store keychain.TokenStore, clientID, tokenBaseURL string) tea.Cmd {
	return func() tea.Msg {
		if store == nil {
			return tokenRefreshedMsg{err: errors.New("no token store configured")}
		}
		refreshToken, err := store.Get(keychain.KeyRefreshToken)
		if err != nil || refreshToken == "" {
			return tokenRefreshedMsg{err: errors.New("no refresh token available")}
		}
		if err := api.Refresh(context.Background(), tokenBaseURL, refreshToken, clientID, store); err != nil {
			return tokenRefreshedMsg{err: err}
		}
		newToken, err := store.Get(keychain.KeyAccessToken)
		if err != nil {
			return tokenRefreshedMsg{err: err}
		}
		return tokenRefreshedMsg{newToken: newToken}
	}
}

// buildFetchPlaylistTracksCmd creates a command that fetches tracks for a playlist
// and returns them in PlaylistTracksLoadedMsg. No Store writes occur — Update() writes.
func (a *App) buildFetchPlaylistTracksCmd(playlistID string) tea.Cmd {
	library := a.library
	return func() tea.Msg {
		if library == nil {
			return panes.PlaylistTracksLoadedMsg{Err: errNilClient, PlaylistID: playlistID}
		}
		tracks, err := library.PlaylistTracks(context.Background(), playlistID, 100, 0)
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			return panes.PlaylistTracksLoadedMsg{PlaylistID: playlistID, Err: err}
		}
		return panes.PlaylistTracksLoadedMsg{PlaylistID: playlistID, Tracks: tracks}
	}
}

// buildCreatePlaylistCmd creates a command that calls CreatePlaylist on the playlists API
// and returns a PlaylistCreatedMsg with the result.
func (a *App) buildCreatePlaylistCmd(name, description string) tea.Cmd {
	playlistsAPI := a.playlistsAPI
	return func() tea.Msg {
		if playlistsAPI == nil {
			return panes.PlaylistCreatedMsg{Err: errNilClient, Name: name}
		}
		playlist, err := playlistsAPI.CreatePlaylist(api.WithPriority(context.Background(), api.Interactive), name, description, false)
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
			return panes.PlaylistRenamedMsg{Err: errNilClient, PlaylistID: playlistID, NewName: newName}
		}
		err := playlistsAPI.UpdatePlaylist(api.WithPriority(context.Background(), api.Interactive), playlistID, newName, "")
		return panes.PlaylistRenamedMsg{PlaylistID: playlistID, NewName: newName, Err: err}
	}
}

// buildRemovePlaylistTrackCmd creates a command that calls RemoveTracksFromPlaylist
// and returns a PlaylistRemoveResultMsg.
func (a *App) buildRemovePlaylistTrackCmd(playlistID, trackURI string) tea.Cmd {
	playlistsAPI := a.playlistsAPI
	return func() tea.Msg {
		if playlistsAPI == nil {
			return panes.PlaylistRemoveResultMsg{Err: errNilClient, PlaylistID: playlistID, TrackURI: trackURI}
		}
		err := playlistsAPI.RemoveTracksFromPlaylist(api.WithPriority(context.Background(), api.Interactive), playlistID, []string{trackURI})
		return panes.PlaylistRemoveResultMsg{PlaylistID: playlistID, TrackURI: trackURI, Err: err}
	}
}

// buildReorderPlaylistTracksCmd creates a command that calls ReorderPlaylistTracks
// and returns a PlaylistReorderResultMsg.
func (a *App) buildReorderPlaylistTracksCmd(playlistID string, rangeStart, insertBefore, rangeLength int) tea.Cmd {
	playlistsAPI := a.playlistsAPI
	return func() tea.Msg {
		if playlistsAPI == nil {
			return panes.PlaylistReorderResultMsg{Err: errNilClient}
		}
		err := playlistsAPI.ReorderPlaylistTracks(api.WithPriority(context.Background(), api.Interactive), playlistID, rangeStart, insertBefore, rangeLength)
		return panes.PlaylistReorderResultMsg{Err: err}
	}
}
