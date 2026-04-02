package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// SearchClient provides the Spotify search API call.
// It embeds BaseClient for shared HTTP functionality.
type SearchClient struct {
	BaseClient
}

// NewSearchClient constructs a SearchClient using the given base URL and access token.
// Pass "" for baseURL to use the production Spotify API.
func NewSearchClient(baseURL, accessToken string) *SearchClient {
	return &SearchClient{BaseClient: NewBaseClient(baseURL, accessToken)}
}

// SetHTTPClient overrides the default HTTP client used for API calls.
func (s *SearchClient) SetHTTPClient(c *http.Client) {
	s.setHTTPClient(c)
}

// Search calls GET /v1/search with the given query, types, per-type limit, and page offset.
// Always includes market=from_token per Spotify API recommendations.
// types should contain one or more of: "track", "artist", "album", "playlist".
// offset shifts the result window for pagination (0-based); pass 0 for the first page.
// Returns a fully populated SearchResult on success.
func (s *SearchClient) Search(ctx context.Context, query string, types []string, limit, offset int) (*SearchResult, error) {
	req, err := s.newRequest(ctx, http.MethodGet, "/v1/search", nil)
	if err != nil {
		return nil, fmt.Errorf("creating search request: %w", err)
	}

	q := req.URL.Query()
	q.Set("q", query)
	q.Set("type", strings.Join(types, ","))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	q.Set("market", "from_token")
	req.URL.RawQuery = q.Encode()

	var result SearchResult
	if err := s.doJSON(req, &result); err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return &result, nil
}
