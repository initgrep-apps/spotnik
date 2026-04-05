package api

import "context"

// LibraryAPI defines all Spotify library read and like/unlike operations.
// Concrete implementation: *LibraryClient.
type LibraryAPI interface {
	Playlists(ctx context.Context, limit, offset int) ([]SimplePlaylist, error)
	// PlaylistTracks fetches a page of playlist tracks. Returns tracks, total count,
	// and whether a next page exists.
	PlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]Track, int, bool, error)
	SavedAlbums(ctx context.Context, limit, offset int) ([]SavedAlbum, error)
	LikedTracks(ctx context.Context, limit, offset int) ([]SavedTrack, error)
	RecentlyPlayed(ctx context.Context, limit int) ([]PlayHistory, error)
	LikeTrack(ctx context.Context, trackID string) error
	UnlikeTrack(ctx context.Context, trackID string) error
}

// Compile-time assertion: *LibraryClient must implement LibraryAPI.
var _ LibraryAPI = (*LibraryClient)(nil)
