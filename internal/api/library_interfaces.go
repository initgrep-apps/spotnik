package api

import "context"

// LibraryAPI defines all Spotify library read and like/unlike operations.
// Concrete implementation: *LibraryClient.
type LibraryAPI interface {
	Playlists(ctx context.Context, limit, offset int) ([]SimplePlaylist, error)
	// GetPlaylist fetches a playlist's metadata and first page of items via GET /playlists/{id}.
	// Use this for the initial drill-down (offset=0); use PlaylistTracks for subsequent pages.
	GetPlaylist(ctx context.Context, playlistID string) ([]Track, int, bool, error)
	// PlaylistTracks fetches a page of playlist items via GET /playlists/{id}/items. Returns tracks,
	// total count, and whether a next page exists. Use for pagination (offset > 0).
	PlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]Track, int, bool, error)
	// AlbumTracks fetches a page of tracks for the given album.
	// Returns the tracks slice, a hasNext bool (true if more pages exist), and any error.
	AlbumTracks(ctx context.Context, albumID string, limit, offset int) ([]Track, bool, error)
	SavedAlbums(ctx context.Context, limit, offset int) ([]SavedAlbum, error)
	LikedTracks(ctx context.Context, limit, offset int) ([]SavedTrack, error)
	RecentlyPlayed(ctx context.Context, limit int) ([]PlayHistory, error)
	LikeTrack(ctx context.Context, trackID string) error
	UnlikeTrack(ctx context.Context, trackID string) error
}

// Compile-time assertion: *LibraryClient must implement LibraryAPI.
var _ LibraryAPI = (*LibraryClient)(nil)
