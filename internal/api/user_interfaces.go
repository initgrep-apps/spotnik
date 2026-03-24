package api

import "context"

// UserAPI defines all Spotify user statistics operations.
// Concrete implementation: *UserClient.
type UserAPI interface {
	GetTopTracks(ctx context.Context, timeRange string, limit int) ([]Track, error)
	GetTopArtists(ctx context.Context, timeRange string, limit int) ([]FullArtist, error)
	GetRecentlyPlayed(ctx context.Context, limit int) ([]PlayHistory, error)
}

// Compile-time assertion: *UserClient must implement UserAPI.
var _ UserAPI = (*UserClient)(nil)
