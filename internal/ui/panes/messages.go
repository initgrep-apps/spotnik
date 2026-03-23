// Package panes contains Bubble Tea pane models for the Spotnik TUI.
// This file defines the message types shared between panes and the root app model.
//
// Design: panes never import api/. Instead they emit request messages that the
// root app model handles by dispatching actual API commands.
package panes

// TickMsg is sent every second by the polling tick loop.
// It drives both progress interpolation and playback state refresh.
type TickMsg struct{}

// PlaybackStateFetchedMsg notifies the player pane that the store has been
// updated with a fresh playback state. The pane reads from the store directly.
// NOTE: no api payload — store is the single source of truth.
type PlaybackStateFetchedMsg struct{}

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

// LibraryLoadedMsg is sent by the root app model after playlists have been loaded
// into the store. The library pane reads from store directly on receipt.
type LibraryLoadedMsg struct{}

// AlbumsLoadedMsg is sent by the root app model after saved albums have been loaded
// into the store.
type AlbumsLoadedMsg struct{}

// LikedTracksLoadedMsg is sent by the root app model after liked tracks have been
// loaded into the store.
type LikedTracksLoadedMsg struct{}

// RecentlyPlayedLoadedMsg is sent by the root app model after recently played tracks
// have been loaded into the store.
type RecentlyPlayedLoadedMsg struct{}

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

// QueueLoadedMsg is sent by the root app model after the queue has been fetched
// from the Spotify API and written to the store.
// The QueuePane reads from store directly on receipt.
type QueueLoadedMsg struct{}

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

// FetchPlaylistTracksRequestMsg is emitted by PlaylistManager when it needs to
// load tracks for a specific playlist from the API.
type FetchPlaylistTracksRequestMsg struct {
	PlaylistID string
}

// PlaylistTracksLoadedMsg is sent by the root app after playlist tracks have been
// fetched and written to the store. The PlaylistManager reads from store on receipt.
type PlaylistTracksLoadedMsg struct {
	PlaylistID string
}

// PlaylistCreateRequestMsg is emitted by PlaylistManager when the user submits
// a new playlist name. The root app creates the playlist via the API.
type PlaylistCreateRequestMsg struct {
	Name        string
	Description string
}

// PlaylistRenameRequestMsg is emitted by PlaylistManager when the user submits
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

// PlaylistRemoveRequestMsg is emitted by PlaylistManager when the user confirms
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

// PlaylistReorderRequestMsg is emitted by PlaylistManager when the user reorders
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
