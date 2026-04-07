package api

import "context"

// UserAPI defines all Spotify user statistics operations.
// Concrete implementation: *UserClient.
type UserAPI interface {
	// Profile fetches the authenticated user's Spotify profile (GET /v1/me).
	Profile(ctx context.Context) (UserProfile, error)
	TopTracks(ctx context.Context, timeRange string, limit int) ([]Track, error)
	TopArtists(ctx context.Context, timeRange string, limit int) ([]FullArtist, error)
	RecentlyPlayed(ctx context.Context, limit int) ([]PlayHistory, error)
}

// Compile-time assertion: *UserClient must implement UserAPI.
var _ UserAPI = (*UserClient)(nil)
