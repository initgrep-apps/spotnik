package api

import (
	"context"

	"github.com/initgrep-apps/spotnik/internal/domain"
)

// PodcastAPI defines all podcast-related Spotify API operations.
// Concrete implementation: *PodcastClient.
type PodcastAPI interface {
	Show(ctx context.Context, showID string) (*domain.Show, error)
	ShowEpisodes(ctx context.Context, showID string, limit, offset int) ([]domain.Episode, int, bool, error)
	FollowedShows(ctx context.Context, limit, offset int) ([]domain.SavedShow, error)
	Episode(ctx context.Context, episodeID string) (*domain.Episode, error)
	SavedEpisodes(ctx context.Context, limit, offset int) ([]domain.SavedEpisode, error)
}

// Compile-time assertion: *PodcastClient must implement PodcastAPI.
var _ PodcastAPI = (*PodcastClient)(nil)

// savedShowsResponse is the response wrapper for GET /v1/me/shows.
type savedShowsResponse struct {
	Items []domain.SavedShow `json:"items"`
	Total int                `json:"total"`
}

// savedEpisodesResponse is the response wrapper for GET /v1/me/episodes.
type savedEpisodesResponse struct {
	Items []domain.SavedEpisode `json:"items"`
	Total int                   `json:"total"`
}

// showWithEpisodesResponse wraps a Show with inline episode data for the
// GET /v1/shows/{id} response.
type showWithEpisodesResponse struct {
	domain.Show
	Episodes struct {
		Items []domain.Episode `json:"items"`
		Total int              `json:"total"`
	} `json:"episodes"`
}

// showEpisodesResponse wraps episode list for GET /v1/shows/{id}/episodes.
type showEpisodesResponse struct {
	Items []domain.Episode `json:"items"`
	Total int              `json:"total"`
}
