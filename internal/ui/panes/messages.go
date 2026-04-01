// Package panes contains Bubble Tea pane models for the Spotnik TUI.
// This file defines the message types shared between panes and the root app model.
//
// Design: panes never import api/. Instead they emit request messages that the
// root app model handles by dispatching actual API commands.
// Messages that carry data use domain/ types so that ui/ never imports api/.
package panes

import (
	"github.com/initgrep-apps/spotnik/internal/domain"
)

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

// SearchResultData is the UI-facing representation of search results.
// It carries only the fields the overlay needs for rendering, pre-converted
// from domain.SearchResult in commands.go so that search.go never imports api/.
type SearchResultData struct {
	Tracks    []domain.SearchTrackItem
	Artists   []domain.SearchArtistItem
	Albums    []domain.SearchAlbumItem
	Playlists []domain.SearchPlaylistItem

	// TotalTracks is the total number of matching tracks across all pages.
	TotalTracks int
	// TotalArtists is the total number of matching artists across all pages.
	TotalArtists int
	// TotalAlbums is the total number of matching albums across all pages.
	TotalAlbums int
	// TotalPlaylists is the total number of matching playlists across all pages.
	TotalPlaylists int
}

// SearchTrackItem is the UI-facing track item type, aliased from domain for zero-cost reuse.
// Defined here for backward compat — panes code can use panes.SearchTrackItem directly.
type SearchTrackItem = domain.SearchTrackItem

// SearchArtistItem is the UI-facing artist item type, aliased from domain.
type SearchArtistItem = domain.SearchArtistItem

// SearchAlbumItem is the UI-facing album item type, aliased from domain.
type SearchAlbumItem = domain.SearchAlbumItem

// SearchPlaylistItem is the UI-facing playlist item type, aliased from domain.
type SearchPlaylistItem = domain.SearchPlaylistItem

// SearchResultsMsg is sent by the root app model after a search completes.
// Results carries the pre-converted UI data; Err is non-nil if the search failed.
type SearchResultsMsg struct {
	Results *SearchResultData
	Err     error
	// Section identifies which section the results belong to when loading a paginated page.
	// Zero value (SectionTracks = 0) is used for initial full-result loads.
	Section SearchSection
	// Offset is the pagination offset that was requested; used by SearchOverlay to
	// update sectionOffsets[Section] after the page loads.
	Offset int
	// IsPaged distinguishes a paginated load from a new-query load. When true the
	// overlay merges only the requesting section's results; when false it replaces
	// all results and resets pagination state. This allows offset=0 to be used
	// legitimately as the first page of a paginated back-navigation.
	IsPaged bool
}

// SearchPageRequestMsg is emitted when the user navigates past the current page boundary.
// The root app handles it by calling buildSearchCmd with the given offset.
// Section identifies which section triggered the page request so the overlay can
// store results for only that section.
type SearchPageRequestMsg struct {
	Query   string
	Offset  int
	Section SearchSection
}
