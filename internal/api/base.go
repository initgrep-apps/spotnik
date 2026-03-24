package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// BaseClient provides shared HTTP functionality for all API clients.
// All six Spotify API clients embed BaseClient to avoid duplicating
// newRequest/doJSON/doNoContent across each client file.
type BaseClient struct {
	baseURL string
	token   TokenProvider
	http    *http.Client
}

// NewBaseClient creates a BaseClient with sensible defaults.
// The access token string is wrapped in a StaticTokenProvider so all existing
// client constructors remain unchanged.
// Pass "" for baseURL to use the production Spotify API.
func NewBaseClient(baseURL, accessToken string) BaseClient {
	return NewBaseClientWithProvider(baseURL, &StaticTokenProvider{Token: accessToken})
}

// NewBaseClientWithProvider creates a BaseClient that calls tp for every request
// to obtain a fresh access token. Use this when you need per-request token
// resolution (e.g. a RefreshableTokenProvider).
func NewBaseClientWithProvider(baseURL string, tp TokenProvider) BaseClient {
	if tp == nil {
		panic("api: TokenProvider must not be nil")
	}
	base := baseURL
	if base == "" {
		base = spotifyAPIBaseURL
	}
	return BaseClient{
		baseURL: strings.TrimRight(base, "/"),
		token:   tp,
		http:    &http.Client{},
	}
}

// setHTTPClient overrides the default HTTP client used for API calls.
// Used to inject a LoggingTransport for network call recording.
func (b *BaseClient) setHTTPClient(c *http.Client) {
	b.http = c
}

// newRequest builds an authenticated HTTP request with the Authorization header set.
// It calls b.token.AccessToken(ctx) per-request so that a RefreshableTokenProvider
// can silently update the token without restarting the client.
func (b *BaseClient) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	token, err := b.token.AccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolving access token: %w", err)
	}

	u := b.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

// doJSON executes req, checks for a known error status, then decodes the JSON body into out.
// Returns typed errors for 401, 403, and 429 responses.
func (b *BaseClient) doJSON(req *http.Request, out interface{}) error {
	resp, err := b.http.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if err := checkResponseStatus(resp, body); err != nil {
		return err
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	return nil
}

// doNoContent executes req and returns nil if the response is 2xx.
// Returns typed errors for 401, 403, and 429 responses; a generic error otherwise.
func (b *BaseClient) doNoContent(req *http.Request) error {
	resp, err := b.http.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	return checkResponseStatus(resp, body)
}
