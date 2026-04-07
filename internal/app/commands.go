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

// SearchPageSize is the number of results fetched per page.
// Matches Spotify's recommended default; named for test clarity.
const SearchPageSize = 10

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

// buildPlayContextCmd dispatches a play command for a playlist, album, or collection
// context URI. When offsetURI is non-empty, playback starts at that track URI within
// the context (e.g. liked songs starting at a selected track).
func (a *App) buildPlayContextCmd(contextURI, offsetURI string) tea.Cmd {
	player := a.player
	return func() tea.Msg {
		if player == nil {
			return panes.PlaybackCmdSentMsg{Err: errNilClient}
		}
		opts := domain.PlayOptions{ContextURI: contextURI}
		if offsetURI != "" {
			opts.Offset = &domain.PlayOffset{URI: offsetURI}
		}
		err := player.Play(api.WithPriority(context.Background(), api.Interactive), opts)
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

// buildPlayTrackListCmd dispatches a play command for an ordered list of track URIs.
// Used by panes without a Spotify collection context (Top Tracks, Recently Played,
// Search). Spotify plays URIs[0] and queues the rest.
func (a *App) buildPlayTrackListCmd(uris []string) tea.Cmd {
	if len(uris) == 0 {
		return nil
	}
	player := a.player
	return func() tea.Msg {
		if player == nil {
			return panes.PlaybackCmdSentMsg{Err: errNilClient}
		}
		err := player.Play(api.WithPriority(context.Background(), api.Interactive),
			domain.PlayOptions{URIs: uris})
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

// buildSearchPageCmd fetches a single page of search results.
// ctx is cancelled by App when a new search starts or the overlay closes.
// Returns nil if ctx is already cancelled — Bubble Tea drops nil messages
// silently, preventing stale SearchPageLoadedMsg from entering the Update loop.
// page is 1-based; offset is computed internally as (page-1)*SearchPageSize.
func buildSearchPageCmd(ctx context.Context, client api.SearchAPI, query string, types []string, page int) tea.Cmd {
	return func() tea.Msg {
		if ctx.Err() != nil {
			return nil
		}
		if client == nil {
			return panes.SearchPageLoadedMsg{Query: query, Page: page, Err: errNilClient}
		}
		offset := (page - 1) * SearchPageSize
		// Spotify rejects any request with offset >= 1000; return nil so Bubble Tea
		// drops the message silently rather than surfacing an API error to the user.
		if offset >= 1000 {
			return nil
		}
		// Search is user-triggered (debounce fires after keypress) — bypass token bucket.
		result, err := client.Search(
			api.WithPriority(ctx, api.Interactive),
			query,
			types,
			SearchPageSize,
			offset,
		)
		if ctx.Err() != nil {
			// Request completed but context was cancelled — caller has moved on.
			return nil
		}
		if err != nil {
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			return panes.SearchPageLoadedMsg{Query: query, Page: page, Err: err}
		}
		items, total := convertSearchResult(result)
		return panes.SearchPageLoadedMsg{
			Query:   query,
			Page:    page,
			Results: items,
			Total:   total,
		}
	}
}

// BuildSearchPageCmd is an exported wrapper around buildSearchPageCmd for testing.
// Exported for tests only — production code must use buildSearchPageCmd.
func BuildSearchPageCmd(ctx context.Context, client api.SearchAPI, query string, types []string, page int) tea.Cmd {
	return buildSearchPageCmd(ctx, client, query, types, page)
}

// convertSearchResult converts a Spotify search API response into a flat list
// of SearchListItems and a total result count.
//
// For the All tab the total is the maximum across all returned types — the
// deepest result set determines how many pages exist. For single-type tabs
// only one field is non-zero so the max equals that type's total.
func convertSearchResult(r *api.SearchResult) ([]panes.SearchListItem, int) {
	if r == nil {
		return nil, 0
	}
	var items []panes.SearchListItem
	items = append(items, panes.TracksToSearchListItems(r.Tracks.Items)...)
	items = append(items, panes.ArtistsToSearchListItems(r.Artists.Items)...)
	items = append(items, panes.AlbumsToSearchListItems(r.Albums.Items)...)
	items = append(items, panes.PlaylistsToSearchListItems(r.Playlists.Items)...)

	total := max(r.Tracks.Total, r.Artists.Total, r.Albums.Total, r.Playlists.Total)
	return items, total
}

// ConvertSearchResult is an exported wrapper around convertSearchResult for testing.
// Exported for tests only.
func ConvertSearchResult(r *api.SearchResult) ([]panes.SearchListItem, int) {
	return convertSearchResult(r)
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

// buildFetchPlaylistTracksCmd creates a command that fetches one page of playlist
// tracks using Interactive priority. The context is cancellable — app.go cancels
// it when the user switches to a different playlist or presses Esc.
//
// Always uses GET /playlists/{id}/items regardless of offset. GET /playlists/{id}
// only embeds items for playlists owned by the authenticated user; non-owned
// (followed) playlists omit the items container entirely from that response.
// Using /items consistently works for all playlists.
//
// No Store writes — data is returned in PlaylistTracksLoadedMsg for the pane.
func (a *App) buildFetchPlaylistTracksCmd(ctx context.Context, playlistID string, offset int) tea.Cmd {
	library := a.library
	return func() tea.Msg {
		// Check for cancellation before making the HTTP call.
		if ctx.Err() != nil {
			return nil
		}
		if library == nil {
			return panes.PlaylistTracksLoadedMsg{Err: errNilClient, PlaylistID: playlistID, Offset: offset}
		}

		// GET /playlists/{id}/items works for all playlists (owned and non-owned).
		tracks, total, hasNext, err := library.PlaylistTracks(
			api.WithPriority(ctx, api.Interactive),
			playlistID, 100, offset,
		)

		if err != nil {
			// Check cancellation again — context.Canceled is expected on playlist switch.
			if ctx.Err() != nil {
				return nil // silently discard; not an error worth toasting
			}
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			var forbiddenErr *api.ForbiddenError
			if errors.As(err, &forbiddenErr) {
				return panes.PlaylistTracksLoadedMsg{PlaylistID: playlistID, Offset: offset, Err: forbiddenErr}
			}
			return panes.PlaylistTracksLoadedMsg{PlaylistID: playlistID, Offset: offset, Err: err}
		}
		return panes.PlaylistTracksLoadedMsg{
			PlaylistID: playlistID,
			Tracks:     tracks,
			Total:      total,
			HasNext:    hasNext,
			Offset:     offset,
		}
	}
}

// buildFetchCurrentUserCmd fetches the authenticated user's Spotify profile via
// GET /v1/me. The returned userProfileLoadedMsg carries the user's Spotify ID,
// which the routing layer stores so the playlist pane can distinguish owned from
// followed playlists.
func (a *App) buildFetchCurrentUserCmd() tea.Cmd {
	userAPI := a.userAPI
	return func() tea.Msg {
		if userAPI == nil {
			return userProfileLoadedMsg{err: errNilClient}
		}
		profile, err := userAPI.Profile(context.Background())
		if err != nil {
			if secs := parse429RetryAfter(err); secs > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: secs}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			return userProfileLoadedMsg{err: err}
		}
		return userProfileLoadedMsg{userID: profile.ID}
	}
}

// buildFetchAlbumTracksCmd fetches a page of tracks for the given album ID.
// Offset 0 = first page (replace); Offset > 0 = subsequent page (append).
// The context is passed in from the caller to support cancellation when the user
// switches albums or presses Esc. api.Interactive priority bypasses the token bucket.
func (a *App) buildFetchAlbumTracksCmd(ctx context.Context, albumID string, offset int) tea.Cmd {
	library := a.library
	return func() tea.Msg {
		// Check for cancellation before making the HTTP call.
		if ctx.Err() != nil {
			return nil
		}
		if library == nil {
			return panes.AlbumTracksLoadedMsg{Err: errNilClient, AlbumID: albumID}
		}
		tracks, hasNext, err := library.AlbumTracks(
			api.WithPriority(ctx, api.Interactive),
			albumID, 50, offset,
		)
		if err != nil {
			// Check cancellation again — context.Canceled is expected on album switch.
			if ctx.Err() != nil {
				return nil // silently discard; not an error worth toasting
			}
			if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
				return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
			}
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			var forbiddenErr *api.ForbiddenError
			if errors.As(err, &forbiddenErr) {
				return panes.AlbumTracksLoadedMsg{AlbumID: albumID, Offset: offset, Err: forbiddenErr}
			}
			return panes.AlbumTracksLoadedMsg{AlbumID: albumID, Offset: offset, Err: err}
		}
		return panes.AlbumTracksLoadedMsg{
			AlbumID: albumID,
			Offset:  offset,
			Tracks:  tracks,
			HasNext: hasNext,
		}
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
