package api

import "context"

// PlaylistsAPI defines all Spotify playlist mutation operations.
// Concrete implementation: *PlaylistsClient.
type PlaylistsAPI interface {
	CreatePlaylist(ctx context.Context, name, description string, public bool) (*SimplePlaylist, error)
	UpdatePlaylist(ctx context.Context, id, name, description string) error
	AddTracksToPlaylist(ctx context.Context, playlistID string, uris []string) error
	RemoveTracksFromPlaylist(ctx context.Context, playlistID string, uris []string) error
	ReorderPlaylistTracks(ctx context.Context, id string, rangeStart, insertBefore, rangeLength int) error
}

// Compile-time assertion: *PlaylistsClient must implement PlaylistsAPI.
var _ PlaylistsAPI = (*PlaylistsClient)(nil)
