package api

import "context"

// SearchAPI defines the Spotify search operation.
// Concrete implementation: *SearchClient.
type SearchAPI interface {
	Search(ctx context.Context, query string, types []string, limit, offset int) (*SearchResult, error)
}

// Compile-time assertion: *SearchClient must implement SearchAPI.
var _ SearchAPI = (*SearchClient)(nil)
