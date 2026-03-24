package api

import "context"

// LibraryAPI defines all Spotify library read and like/unlike operations.
// Concrete implementation: *LibraryClient.
type LibraryAPI interface {
	GetPlaylists(ctx context.Context, limit, offset int) ([]SimplePlaylist, error)
	GetPlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]Track, error)
	GetSavedAlbums(ctx context.Context, limit, offset int) ([]SavedAlbum, error)
	GetLikedTracks(ctx context.Context, limit, offset int) ([]SavedTrack, error)
	GetRecentlyPlayed(ctx context.Context, limit int) ([]PlayHistory, error)
	LikeTrack(ctx context.Context, trackID string) error
	UnlikeTrack(ctx context.Context, trackID string) error
}

// Compile-time assertion: *LibraryClient must implement LibraryAPI.
var _ LibraryAPI = (*LibraryClient)(nil)
