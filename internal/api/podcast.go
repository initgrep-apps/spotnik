package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/initgrep-apps/spotnik/internal/domain"
)

// PodcastClient implements PodcastAPI for all Spotify podcast operations.
// It embeds BaseClient for shared HTTP functionality.
type PodcastClient struct {
	BaseClient
}

// NewPodcastClient constructs a PodcastClient using the given base URL and access token.
// Pass "" for baseURL to use the production Spotify API.
func NewPodcastClient(baseURL, accessToken string) *PodcastClient {
	return &PodcastClient{BaseClient: NewBaseClient(baseURL, accessToken)}
}

// Show fetches a single podcast show by ID, including its episodes.
// Returns the show with TotalEpisodes populated from the inline episodes count.
func (p *PodcastClient) Show(ctx context.Context, showID string) (*domain.Show, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/shows/"+showID, nil)
	if err != nil {
		return nil, fmt.Errorf("creating get show request: %w", err)
	}
	q := req.URL.Query()
	q.Set("market", "from_token")
	req.URL.RawQuery = q.Encode()

	var resp showWithEpisodesResponse
	if err := p.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("getting show: %w", err)
	}
	resp.TotalEpisodes = resp.Episodes.Total
	return &resp.Show, nil
}

// ShowEpisodes fetches episodes for a given show with pagination.
// Returns the episodes list, total count, and whether more pages exist.
func (p *PodcastClient) ShowEpisodes(ctx context.Context, showID string, limit, offset int) ([]domain.Episode, int, bool, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/shows/"+showID+"/episodes", nil)
	if err != nil {
		return nil, 0, false, fmt.Errorf("creating get show episodes request: %w", err)
	}
	q := req.URL.Query()
	q.Set("market", "from_token")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()

	var resp showEpisodesResponse
	if err := p.doJSON(req, &resp); err != nil {
		return nil, 0, false, fmt.Errorf("getting show episodes: %w", err)
	}
	hasNext := offset+limit < resp.Total
	return resp.Items, resp.Total, hasNext, nil
}

// FollowedShows fetches the current user's saved/followed shows with pagination.
func (p *PodcastClient) FollowedShows(ctx context.Context, limit, offset int) ([]domain.SavedShow, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/me/shows", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get followed shows request: %w", err)
	}
	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()

	var resp savedShowsResponse
	if err := p.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("getting followed shows: %w", err)
	}
	return resp.Items, nil
}

// Episode fetches a single podcast episode by ID.
func (p *PodcastClient) Episode(ctx context.Context, episodeID string) (*domain.Episode, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/episodes/"+episodeID, nil)
	if err != nil {
		return nil, fmt.Errorf("creating get episode request: %w", err)
	}
	q := req.URL.Query()
	q.Set("market", "from_token")
	req.URL.RawQuery = q.Encode()

	var episode domain.Episode
	if err := p.doJSON(req, &episode); err != nil {
		return nil, fmt.Errorf("getting episode: %w", err)
	}
	return &episode, nil
}

// SavedEpisodes fetches the current user's saved episodes with pagination.
func (p *PodcastClient) SavedEpisodes(ctx context.Context, limit, offset int) ([]domain.SavedEpisode, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/me/episodes", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get saved episodes request: %w", err)
	}
	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()

	var resp savedEpisodesResponse
	if err := p.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("getting saved episodes: %w", err)
	}
	return resp.Items, nil
}
