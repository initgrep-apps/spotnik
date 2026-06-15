// Package panes contains Bubble Tea pane models for the Spotnik TUI.
// This file defines the message types shared between panes and the root app model.
//
// Design: panes never import api/. Instead they emit request messages that the
// root app model handles by dispatching actual API commands.
// Messages that carry data use domain/ types so that ui/ never imports api/.
package panes

import "github.com/initgrep-apps/spotnik/internal/domain"

// TickMsg is sent every second by the polling tick loop.
// It drives both progress interpolation and playback state refresh.
type TickMsg struct{}

// RateLimitedMsg is returned by fetch commands when the Spotify API responds with 429.
// The root app uses RetryAfterSecs to set the backoff timer.
type RateLimitedMsg struct {
	RetryAfterSecs int
}

// PlaybackStateFetchedMsg is returned by the playback polling command.
// State carries the fetched playback state on success (may be nil if 204 — nothing playing).
// Err is non-nil if a non-rate-limit, non-401 error occurred.
// Update() writes State to the store; panes read from store directly.
type PlaybackStateFetchedMsg struct {
	State *domain.PlaybackState
	Err   error
}

// PlaybackCmdSentMsg is returned after any playback control command completes.
// Err is non-nil if the API returned an error.
// Source identifies the originating command (e.g. "playback", "volume") so the
// handler can select the correct Operation for error mapping.
type PlaybackCmdSentMsg struct {
	Err    error
	Source string
}

// VolumeIntentMsg is emitted by NowPlayingPane after the volume debounce
// resolves. TargetVol is the exact percentage to set. Seq is the debounce
// sequence number; it is threaded through to VolumeAppliedMsg so the bar
// can guard ConfirmFromAPI / CancelPending against concurrent bursts.
type VolumeIntentMsg struct {
	TargetVol int
	Seq       int // intentSeq returned by HandleDebounce
}

// VolumeAppliedMsg is returned by buildSetVolumeCmd after the Spotify volume
// API call completes (success or failure). It replaces PlaybackCmdSentMsg for

// SeekIntentMsg is emitted by NowPlayingPane after the seek debounce resolves.
// TargetMs is the exact position in milliseconds to seek to. Seq is the debounce
// sequence number; it is threaded through to SeekAppliedMsg so the bar can guard
// ConfirmFromAPI / CancelPending against concurrent bursts.
type SeekIntentMsg struct {
	TargetMs int
	Seq      int // tickSeq returned by HandleDebounce
}

// SeekAppliedMsg is returned by buildSeekCmd after the Spotify seek API call
// completes (success or failure). On success: PosMs holds the confirmed position,
// Err is nil. On error: PosMs is 0, Err holds the underlying error.
type SeekAppliedMsg struct {
	PosMs int
	Seq   int
	Err   error
}

// the volume-specific path so the bar's pending state is managed correctly.
//
// On success: Vol holds the confirmed volume, Err is nil.
// On error:   Vol is 0, Err holds the underlying error (may be *api.RateLimitError,
//
//	*api.UnauthorizedError, or a generic error).
type VolumeAppliedMsg struct {
	Vol int
	Seq int
	Err error
}

// PlaybackAction identifies what kind of playback command to send.
// Used in PlaybackRequestMsg so the root app model can dispatch the right call.
type PlaybackAction int

const (
	// ActionPause pauses playback.
	ActionPause PlaybackAction = iota
	// ActionPlay resumes playback.
	ActionPlay
	// ActionNext skips to the next track.
	ActionNext
	// ActionPrevious goes back to the previous track.
	ActionPrevious
	// ActionToggleShuffle toggles shuffle mode.
	ActionToggleShuffle
	// ActionCycleRepeat cycles through repeat modes (off → context → track → off).
	ActionCycleRepeat
)

// PlaybackRequestMsg is emitted by the player pane when the user presses a
// playback control key. The root app model receives it and dispatches the
// appropriate Spotify API command.
type PlaybackRequestMsg struct {
	Action PlaybackAction
}

// FetchPlaylistsRequestMsg is emitted by the library pane when it needs to
// load (or paginate) playlists from the API.
type FetchPlaylistsRequestMsg struct {
	Offset int
}

// FetchAlbumsRequestMsg is emitted by the library pane when it needs to
// load saved albums from the API.
type FetchAlbumsRequestMsg struct {
	Offset int
}

// FetchLikedTracksRequestMsg is emitted by the library pane when it needs to
// load liked tracks from the API.
type FetchLikedTracksRequestMsg struct {
	Offset int
}

// FetchRecentlyPlayedRequestMsg is emitted by the library pane when it needs to
// load recently played tracks from the API.
type FetchRecentlyPlayedRequestMsg struct{}

// PlayContextMsg is sent when the user selects a playlist, album, or collection
// to play. OffsetURI is optional — when set, playback starts at that track URI
// within the context rather than from the beginning.
type PlayContextMsg struct {
	ContextURI string
	// OffsetURI is optional: start at this track URI within the context.
	OffsetURI string
}

// PlayTrackListMsg is sent when the user plays a track from a pane that has no
// Spotify collection context (Top Tracks, Recently Played, Search results).
// URIs is the ordered list of track URIs starting from the selected track —
// Spotify will play URIs[0] and queue the rest.
type PlayTrackListMsg struct {
	URIs []string
}

// PlayTrackMsg is sent by QueuePane when the user selects a queued track to play.
// This is a skip-to operation — the queue pane selects by URI, not by context.
// Other panes use PlayContextMsg or PlayTrackListMsg instead.
type PlayTrackMsg struct {
	TrackURI string
}

// AddToQueueMsg is sent when the user presses 'a' on a track.
// The root app model receives this and dispatches an add-to-queue API call.
type AddToQueueMsg struct {
	TrackURI  string
	TrackName string
}

// LibraryLoadedMsg is sent by the root app model after playlists have been fetched.
// Items carries the raw page of playlists; Offset indicates whether to replace (0)
// or append (>0) to existing playlists. Err is non-nil on failure.
// Update() handles pagination and writes to the store.
type LibraryLoadedMsg struct {
	Items  []domain.SimplePlaylist
	Offset int
	Err    error
}

// AlbumsLoadedMsg is sent after saved albums have been fetched.
// Items carries the albums; Offset indicates whether to replace (0) or append (>0)
// to existing albums. Err is non-nil on failure. Update() writes Items to the store.
type AlbumsLoadedMsg struct {
	Items  []domain.SavedAlbum
	Offset int
	Err    error
}

// LikedTracksLoadedMsg is sent after liked tracks have been fetched.
// Items carries the tracks; Offset indicates the page offset for total calculation.
// Err is non-nil on failure. Update() writes Items to the store.
type LikedTracksLoadedMsg struct {
	Items  []domain.SavedTrack
	Offset int
	Err    error
}

// RecentlyPlayedLoadedMsg is sent after recently played tracks have been fetched.
// Items carries the play history; Err is non-nil on failure.
// Update() writes Items to the store.
type RecentlyPlayedLoadedMsg struct {
	Items []domain.PlayHistory
	Err   error
}

// AddToQueueResultMsg carries the result of an add-to-queue operation.
// Err is non-nil on failure. TrackName is the name of the queued track for
// status bar display.
type AddToQueueResultMsg struct {
	Err       error
	TrackName string
}

// QueueLoadedMsg is returned by the queue fetch command.
// Tracks carries the fetched queue on success; Err is non-nil on failure.
// Update() writes Tracks to the store; QueuePane reads from store directly.
type QueueLoadedMsg struct {
	Tracks []domain.Track
	Err    error
}

// FetchStatsMsg is a request message emitted by stats panes (TopTracksPane, TopArtistsPane)
// to ask the root app to fetch stats data for the given time range from the API.
type FetchStatsMsg struct {
	// TimeRange is "short_term", "medium_term", or "long_term".
	TimeRange string
}

// StatsLoadedMsg is returned by the stats fetch command.
// TimeRange identifies which range was fetched.
// TopTracks and TopArtists carry the fetched data on success.
// Err is non-nil on failure. Update() writes data to the store; pane reads from store.
type StatsLoadedMsg struct {
	// TimeRange is the time range that was fetched ("short_term", "medium_term", "long_term").
	TimeRange string
	// TopTracks contains the fetched top tracks for the time range.
	TopTracks []domain.Track
	// TopArtists contains the fetched top artists for the time range.
	TopArtists []domain.FullArtist
	// Err is non-nil if the fetch failed.
	Err error
}

// DeviceInfo is the UI-facing representation of a Spotify device.
// It mirrors the fields needed for rendering without importing api/.
type DeviceInfo struct {
	ID       string
	Name     string
	Type     string
	IsActive bool
}

// DevicesLoadedMsg is returned by the fetch-devices command after the device list
// has been fetched from the Spotify API. All other data-carrying messages are exported;
// this type follows the same convention. The root app.Update() handles store mutations.
type DevicesLoadedMsg struct {
	// Devices is the list of available Spotify Connect devices on success.
	Devices []DeviceInfo
	// Err is non-nil if the fetch failed.
	Err error
}

// DeviceOverlayClosedMsg is emitted by DeviceOverlay when the user presses Esc,
// signalling the root app model to close the overlay and restore the previous focus.
type DeviceOverlayClosedMsg struct{}

// ProfileOverlayClosedMsg is emitted by ProfileOverlay when the user presses Esc,
// signalling the root app model to close the profile overlay.
type ProfileOverlayClosedMsg struct{}

// FetchCurrentUserRequestMsg is emitted by ProfileOverlay.Init() when the store
// has no user profile loaded, triggering a fetch from the app layer.
type FetchCurrentUserRequestMsg struct{}

// UserProfileLoadedMsg is forwarded to ProfileOverlay after a fetch triggered by
// FetchCurrentUserRequestMsg completes. Err is nil on success; the overlay reads
// the profile from the store.
type UserProfileLoadedMsg struct {
	Err error
}

// ProfileConfirmToastMsg is emitted when the user arms a logout or forget action
// (first keypress). The app converts it into a warning alert via the notifications system.
type ProfileConfirmToastMsg struct {
	Text string // e.g. "Press l again to confirm logout"
}

// ProfileLogoutMsg is emitted when the user confirms logout from the profile overlay.
// The app clears tokens and quits.
type ProfileLogoutMsg struct{}

// ProfileForgetMsg is emitted when the user confirms forget from the profile overlay.
// The app clears tokens and client_id from config, then quits.
type ProfileForgetMsg struct{}

// TransferPlaybackMsg is emitted by DeviceOverlay when the user presses Enter on
// a non-active device. The root app model receives it, calls TransferPlayback, and
// shows the status bar message.
type TransferPlaybackMsg struct {
	DeviceID   string
	DeviceName string
}

// DeviceTransferredMsg is returned after a TransferPlayback API call completes.
// Err is non-nil if the transfer failed.
type DeviceTransferredMsg struct {
	DeviceID string
	Err      error
}

// FetchPlaylistTracksRequestMsg is emitted by PlaylistsPane when it needs to
// load (or page) playlist tracks. Offset 0 = initial load; Offset > 0 = next page.
type FetchPlaylistTracksRequestMsg struct {
	PlaylistID string
	Offset     int // 0 for first page, len(loadedTracks) for subsequent pages
}

// PlaylistTracksLoadedMsg is returned by the playlist tracks fetch command.
// PlaylistID identifies which playlist's tracks were fetched.
// Tracks contains the page of tracks. Total is the playlist's total track count.
// HasNext is true when more pages are available (API next != "").
// Offset mirrors the request offset so the pane can detect stale responses.
type PlaylistTracksLoadedMsg struct {
	PlaylistID string
	Tracks     []domain.Track
	Total      int  // total tracks in playlist (from API response)
	HasNext    bool // true when next page exists
	Offset     int  // mirrors request offset for stale-response detection
	Err        error
}

// PlaylistTrackViewClosedMsg is emitted by PlaylistsPane when the user presses
// Esc to return to the playlist list. App.go uses it to cancel any in-flight
// playlist track fetch and clear the staleness key.
type PlaylistTrackViewClosedMsg struct{}

// PlaylistAccessDeniedMsg is emitted by PlaylistsPane when the user presses Enter
// on a playlist the app has determined is not owned or collaborated on by the
// current user. The Spotify API's GET /playlists/{id}/items endpoint returns 403
// for such playlists. The app routing layer converts this into a warning toast
// rather than making a request that will always fail.
type PlaylistAccessDeniedMsg struct{}

// UserProfileReadyMsg is sent by the root app after the authenticated user's
// Spotify ID has been stored in the Store. PlaylistsPane handles this by
// refreshing its row display so that the "~ " prefix appears on followed
// playlists without waiting for the next library reload.
type UserProfileReadyMsg struct{}

// PlaylistRemoveRequestMsg is emitted by PlaylistsPane when the user confirms
// removing a track from a playlist. The root app handles the API call.
type PlaylistRemoveRequestMsg struct {
	PlaylistID string
	TrackURI   string
}

// PlaylistRemoveResultMsg is returned after a remove-track API call completes.
// Err is non-nil if the call failed.
type PlaylistRemoveResultMsg struct {
	PlaylistID string
	TrackURI   string
	Err        error
}

// FetchAlbumTracksRequestMsg is emitted by AlbumsPane when the user opens an album's
// track sub-view. Offset > 0 is used for lazy pagination (triggered by cursor proximity).
type FetchAlbumTracksRequestMsg struct {
	// AlbumID is the Spotify album ID whose tracks are being requested.
	AlbumID string
	// Offset is the 0-based index to start fetching from. 0 = first page.
	Offset int
}

// AlbumTracksLoadedMsg is returned by buildFetchAlbumTracksCmd after the API call.
// AlbumsPane owns the tracks — they are NOT written to the Store.
type AlbumTracksLoadedMsg struct {
	// AlbumID identifies which album's tracks arrived (used for staleness check).
	AlbumID string
	// Offset is the page offset this response corresponds to.
	Offset int
	// Tracks is the loaded slice; nil on error.
	Tracks []domain.Track
	// HasNext is true when the API response had a non-empty "next" URL — more pages exist.
	HasNext bool
	// Err is non-nil if the API call failed.
	Err error
}

// AlbumTrackViewClosedMsg is emitted by AlbumsPane when the user presses Esc to close
// the track sub-view. app.go cancels the in-flight context and clears the staleness key.
type AlbumTrackViewClosedMsg struct{}

// SearchClearedMsg is emitted by SearchOverlay when the user presses Ctrl+U.
// Story 99 will wire the root app to clear overlay-local search state in response.
// Panes must never write to the store directly — they emit messages instead.
type SearchClearedMsg struct{}

// SearchLoadingMsg is sent by app.go to the search overlay immediately before
// dispatching a new HTTP request. IsFirstPage=true means results are nil (spinner
// only); IsFirstPage=false means previous results are still visible (spinner line
// above list + list). Story 100 dispatches this message.
type SearchLoadingMsg struct {
	// IsFirstPage is true when this is the first page of a new query (results cleared).
	IsFirstPage bool
}

// SearchResultSelectedMsg is emitted by the search overlay when the user presses
// Enter on a show or episode result. The root app model uses IsShow/IsEpisode
// to route to the podcasts page or play the episode directly.
type SearchResultSelectedMsg struct {
	URI       string
	IsShow    bool
	IsEpisode bool
}

// SearchPageLoadedMsg is sent by the root app model after a single page of search
// results has loaded. Query and Page are staleness keys — app.go discards this message
// if either does not match the current search session. Results carries the pre-converted
// list items for this page. Total is the flat count across all types/pages.
// Err is non-nil if the search failed.
type SearchPageLoadedMsg struct {
	// Query is the query string that triggered this search, for staleness detection.
	Query string
	// Page is the 1-based page number for this result, used as a staleness key.
	Page int
	// Results carries the pre-converted list items for this page (max SearchPageSize=10).
	Results []SearchListItem
	// Total is the total result count across all types/pages, for the pagination bar.
	Total int
	// Err is non-nil if the fetch failed.
	Err error
}

// Podcast messages

// FetchFollowedShowsRequestMsg is emitted by the FollowedShowsPane when it needs to
// load followed shows from the API.
type FetchFollowedShowsRequestMsg struct{}

// FetchSavedEpisodesRequestMsg is emitted by the SavedEpisodesPane when it needs to
// load saved episodes from the API.
type FetchSavedEpisodesRequestMsg struct{}

// FetchShowEpisodesRequestMsg is emitted by the FollowedShowsPane or the app layer
// when episodes for a specific show need to be loaded from the API.
type FetchShowEpisodesRequestMsg struct {
	ShowID string
}

// FollowedShowsLoadedMsg is sent by the root app model after followed shows have
// been fetched. Items carries the fetched shows; Err is non-nil on failure.
// Update() writes Items to the store.
type FollowedShowsLoadedMsg struct {
	Items []domain.SavedShow
	Err   error
}

// SavedEpisodesLoadedMsg is sent by the root app model after saved episodes have
// been fetched. Items carries the fetched episodes; Err is non-nil on failure.
// Update() writes Items to the store.
type SavedEpisodesLoadedMsg struct {
	Items []domain.SavedEpisode
	Err   error
}

// ShowEpisodesLoadedMsg is sent by the root app model after episodes for a show
// have been fetched. ShowID identifies which show's episodes arrived.
// Items carries the fetched episodes; Total is the total episode count.
// HasNext is true when more pages are available. Err is non-nil on failure.
// Update() writes Items to the store.
type ShowEpisodesLoadedMsg struct {
	ShowID  string
	Items   []domain.Episode
	Total   int
	HasNext bool
	Err     error
}

// SelectedShowChangedMsg is emitted by the FollowedShowsPane when the user selects
// a different show. The root app model uses ShowID to update the store and
// trigger an episode fetch if needed.
type SelectedShowChangedMsg struct {
	ShowID string
}

// PlayEpisodeMsg is emitted by the SavedEpisodesPane or the app layer when the user
// presses Enter on an episode. EpisodeURI is the URI of the episode to play. PlaylistURI
// is the show URI for context, empty for saved episodes.
type PlayEpisodeMsg struct {
	EpisodeURI  string
	PlaylistURI string
}

// EpisodeDetailsOpenMsg is emitted when the user presses 'i' while an episode
// is playing. The root app handles this by opening the EpisodeDetailsOverlay.
type EpisodeDetailsOpenMsg struct{}

// EpisodeDetailsClosedMsg is emitted when the user presses Esc or 'q' in the
// EpisodeDetailsOverlay. The root app handles this by closing the overlay.
type EpisodeDetailsClosedMsg struct{}

// PollingSnapshotMsg carries app-level polling state to the PollingTrafficPane.
// The app sends this on each TickMsg so the pane can display polling diagnostics.
type PollingSnapshotMsg struct {
	// TickIntervalMs is the current playback polling interval in milliseconds.
	TickIntervalMs int
	// IsIdle is true when the user has not pressed a key for idleThresholdSecs.
	IsIdle bool
	// IdleSecs is how long the user has been idle (0 when not idle).
	IdleSecs int
}
