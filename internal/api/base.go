package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// BaseClient provides shared HTTP functionality for all API clients.
// All six Spotify API clients embed BaseClient to avoid duplicating
// newRequest/doJSON/doNoContent across each client file.
type BaseClient struct {
	baseURL string
	token   TokenProvider
	http    *http.Client
	// gateway is accessed via atomic load/store to be safe for concurrent
	// calls to SetGateway and doJSON (e.g. during token refresh).
	gateway atomic.Pointer[Gateway]
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
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// setHTTPClient overrides the default HTTP client used for API calls.
func (b *BaseClient) setHTTPClient(c *http.Client) {
	b.http = c
}

// HTTPTimeout returns the configured request timeout for the internal HTTP client.
// Exported for testing only — production code should not depend on this value.
func (b *BaseClient) HTTPTimeout() time.Duration {
	if b.http == nil {
		return 0
	}
	return b.http.Timeout
}

// SetGateway attaches a Gateway to the BaseClient. When set, all requests
// are routed through the gateway for rate limiting, concurrency capping, and dedup.
// The priority is read from the request context via PriorityFromContext.
// Safe to call concurrently with doJSON/doNoContent.
func (b *BaseClient) SetGateway(gw *Gateway) {
	b.gateway.Store(gw)
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
// When a Gateway is attached, the request is routed through it for rate limiting and dedup.
func (b *BaseClient) doJSON(req *http.Request, out interface{}) error {
	var resp *http.Response
	var err error

	if gw := b.gateway.Load(); gw != nil {
		priority := PriorityFromContext(req.Context())
		key := RequestKey{Method: req.Method, Path: req.URL.Path, Priority: priority}
		// priority and key.Priority are always identical — both come from PriorityFromContext.
		// Do() gates Phase 2/4 on the priority argument; the key is used as the inflight map key.
		resp, err = gw.Do(req.Context(), priority, key, func() (*http.Response, error) {
			return b.http.Do(req)
		})
	} else {
		resp, err = b.http.Do(req)
	}

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

// doJSONOptional executes req and routes through the gateway when attached.
// It returns (nil, nil) for 204 No Content responses, which differs from doJSON
// which treats any non-2xx as an error. Used by endpoints like PlaybackState and
// Queue that legitimately return 204 when nothing is active.
func (b *BaseClient) doJSONOptional(req *http.Request, out interface{}) (bool, error) {
	var resp *http.Response
	var err error

	if gw := b.gateway.Load(); gw != nil {
		priority := PriorityFromContext(req.Context())
		key := RequestKey{Method: req.Method, Path: req.URL.Path, Priority: priority}
		// priority and key.Priority are always identical — both come from PriorityFromContext.
		// Do() gates Phase 2/4 on the priority argument; the key is used as the inflight map key.
		resp, err = gw.Do(req.Context(), priority, key, func() (*http.Response, error) {
			return b.http.Do(req)
		})
	} else {
		resp, err = b.http.Do(req)
	}

	if err != nil {
		return false, fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNoContent {
		return false, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("reading response body: %w", err)
	}

	if err := checkResponseStatus(resp, body); err != nil {
		return false, err
	}

	if err := json.Unmarshal(body, out); err != nil {
		return false, fmt.Errorf("parsing response: %w", err)
	}

	return true, nil
}

// doNoContent executes req and returns nil if the response is 2xx.
// Returns typed errors for 401, 403, and 429 responses; a generic error otherwise.
// When a Gateway is attached, the request is routed through it for rate limiting and dedup.
func (b *BaseClient) doNoContent(req *http.Request) error {
	var resp *http.Response
	var err error

	if gw := b.gateway.Load(); gw != nil {
		priority := PriorityFromContext(req.Context())
		key := RequestKey{Method: req.Method, Path: req.URL.Path, Priority: priority}
		// priority and key.Priority are always identical — both come from PriorityFromContext.
		// Do() gates Phase 2/4 on the priority argument; the key is used as the inflight map key.
		resp, err = gw.Do(req.Context(), priority, key, func() (*http.Response, error) {
			return b.http.Do(req)
		})
	} else {
		resp, err = b.http.Do(req)
	}

	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("reading response body: %w", readErr)
	}
	return checkResponseStatus(resp, body)
}
