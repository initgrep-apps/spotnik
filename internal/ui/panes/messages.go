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
type PlaybackCmdSentMsg struct {
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
	// ActionVolumeUp raises volume by 5%.
	ActionVolumeUp
	// ActionVolumeDown lowers volume by 5%.
	ActionVolumeDown
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

// LikeTrackRequestMsg is emitted by the library pane when the user presses 'l'
// on a track. The root app model dispatches the like/unlike API call.
type LikeTrackRequestMsg struct {
	TrackID string
	// Unlike is true when the track should be unliked (currently liked).
	Unlike bool
}

// PlayContextMsg is sent when the user selects a playlist or album to play.
// The root app model receives this and dispatches a play command to the API.
type PlayContextMsg struct {
	ContextURI string
}

// PlayTrackMsg is sent when the user selects a specific track to play.
// The root app model receives this and dispatches a play command to the API.
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

// LikeToggleResultMsg carries the result of a like/unlike operation.
// TrackID identifies which track was affected. Err is non-nil on failure.
type LikeToggleResultMsg struct {
	TrackID string
	Err     error
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
// load tracks for a specific playlist from the API.
type FetchPlaylistTracksRequestMsg struct {
	PlaylistID string
}

// PlaylistTracksLoadedMsg is returned by the playlist tracks fetch command.
// PlaylistID identifies which playlist's tracks were fetched.
// Tracks carries the fetched tracks on success; Err is non-nil on failure.
// Update() writes Tracks to the store; PlaylistsPane reads from store.
type PlaylistTracksLoadedMsg struct {
	PlaylistID string
	Tracks     []domain.Track
	Err        error
}

// PlaylistCreateRequestMsg is emitted by PlaylistsPane when the user submits
// a new playlist name. The root app creates the playlist via the API.
type PlaylistCreateRequestMsg struct {
	Name        string
	Description string
}

// PlaylistRenameRequestMsg is emitted by PlaylistsPane when the user submits
// a rename. The root app updates the playlist via the API.
type PlaylistRenameRequestMsg struct {
	PlaylistID string
	NewName    string
}

// PlaylistCreatedMsg is returned after a create playlist API call completes.
// Err is non-nil if the call failed.
type PlaylistCreatedMsg struct {
	PlaylistID string
	Name       string
	Err        error
}

// PlaylistRenamedMsg is returned after an update playlist API call completes.
// Err is non-nil if the call failed.
type PlaylistRenamedMsg struct {
	PlaylistID string
	NewName    string
	Err        error
}

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

// PlaylistReorderRequestMsg is emitted by PlaylistsPane when the user reorders
// a track via Shift+Up/Down. The root app handles the API call.
type PlaylistReorderRequestMsg struct {
	PlaylistID   string
	RangeStart   int
	InsertBefore int
	RangeLength  int
}

// PlaylistReorderResultMsg is returned after a reorder-tracks API call completes.
// Err is non-nil if the call failed; the pane reverts the optimistic update on error.
type PlaylistReorderResultMsg struct {
	Err error
}

// SearchClearedMsg is emitted by SearchOverlay when the user presses Ctrl+U.
// The root app model handles this by clearing search results and query in the store.
// Panes must never write to the store directly — they emit messages instead.
type SearchClearedMsg struct{}

// SearchTabChangedMsg is emitted by SearchOverlay when the user cycles the active
// category tab (Tab / Shift+Tab). The root app model re-fires the search with the
// new type filter so results refresh immediately for the chosen category.
type SearchTabChangedMsg struct {
	// Types is the list of Spotify API type strings for the new tab (e.g. ["track"]).
	Types []string
	// Query is the current search query string at the time of the tab change.
	Query string
}

// SearchResultData is the UI-facing representation of one page of search results.
// It carries only the fields the overlay needs for rendering, pre-converted
// from domain.SearchResult in commands.go so that search.go never imports api/.
// Total fields report the API-level count for each type, enabling HasMore checks.
type SearchResultData struct {
	Tracks         []SearchTrackItem
	TracksTotal    int
	Artists        []SearchArtistItem
	ArtistsTotal   int
	Albums         []SearchAlbumItem
	AlbumsTotal    int
	Playlists      []SearchPlaylistItem
	PlaylistsTotal int
}

// SearchTrackItem holds the display fields for a single track search result.
type SearchTrackItem struct {
	// URI is the Spotify track URI used for playback and queue commands.
	URI string
	// Name is the track display name.
	Name string
	// Artist is the pre-formatted first artist name.
	Artist string
}

// SearchArtistItem holds the display fields for a single artist search result.
type SearchArtistItem struct {
	// URI is the Spotify artist URI used for playback context.
	URI string
	// Name is the artist display name.
	Name string
}

// SearchAlbumItem holds the display fields for a single album search result.
type SearchAlbumItem struct {
	// URI is the Spotify album URI used for playback context.
	URI string
	// Name is the album display name.
	Name string
	// Artist is the pre-formatted first artist name.
	Artist string
}

// SearchPlaylistItem holds the display fields for a single playlist search result.
type SearchPlaylistItem struct {
	// URI is the Spotify playlist URI used for playback context.
	URI string
	// Name is the playlist display name.
	Name string
	// Owner is the pre-formatted playlist owner display name.
	Owner string
}

// SearchPrefetchMsg is emitted by the search overlay when the user scrolls past
// searchPrefetchThreshold of the loaded results. The root app model responds by
// fetching the next batch of pages starting at NextOffset.
type SearchPrefetchMsg struct {
	// Query is the current search query at the time of the scroll event,
	// used by the root app to discard stale prefetch requests.
	Query string
	// Types is the list of Spotify API type strings for the active tab.
	Types []string
	// NextOffset is the starting offset for the next batch of pages.
	NextOffset int
}

// SearchPageLoadedMsg is sent by the root app model after a search page fetch completes.
// Query is included so stale results (for a superseded query) can be discarded.
// Offset is the page offset that was fetched. Results carries the pre-converted UI data.
// Err is non-nil if the search failed.
type SearchPageLoadedMsg struct {
	// Query is the query string that triggered this search, for staleness detection.
	Query string
	// Type was the search type filter hint ("all", "track", etc.).
	// As of story 83, buildSearchPageCmd no longer populates this field (always "").
	// The store's per-type Append methods (AppendSearchTracks, etc.) derive the type
	// from the Results payload directly. This field is retained for struct compatibility
	// and may be removed once story 84 confirms it is not needed by the search overlay.
	Type string
	// Offset is the page offset that was fetched.
	Offset int
	// Results carries the pre-converted UI data on success.
	Results *SearchResultData
	// Err is non-nil if the fetch failed.
	Err error
}
