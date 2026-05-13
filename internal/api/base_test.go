package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestBaseClient creates a BaseClient pointed at the given base URL with a fixed token.
func newTestBaseClient(baseURL, token string) BaseClient {
	return NewBaseClient(strings.TrimRight(baseURL, "/"), token)
}

func TestBaseClient_NewRequest_SetsAuthHeader(t *testing.T) {
	bc := newTestBaseClient("https://api.example.com", "my-token")

	req, err := bc.newRequest(context.Background(), http.MethodGet, "/v1/me/player", nil)

	require.NoError(t, err)
	assert.Equal(t, "Bearer my-token", req.Header.Get("Authorization"))
	assert.Equal(t, "https://api.example.com/v1/me/player", req.URL.String())
}

func TestBaseClient_NewRequest_WithBody(t *testing.T) {
	bc := newTestBaseClient("https://api.example.com", "tok")

	body := strings.NewReader(`{"key":"value"}`)
	req, err := bc.newRequest(context.Background(), http.MethodPost, "/v1/me/player/play", body)

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "Bearer tok", req.Header.Get("Authorization"))

	got, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"key":"value"}`, string(got))
}

func TestBaseClient_DoJSON_Success(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(payload{Name: "test"})
	}))
	defer srv.Close()

	bc := newTestBaseClient(srv.URL, "test-token")

	req, err := bc.newRequest(context.Background(), http.MethodGet, "/v1/something", nil)
	require.NoError(t, err)

	var out payload
	err = bc.doJSON(req, &out)
	require.NoError(t, err)
	assert.Equal(t, "test", out.Name)
}

func TestBaseClient_DoJSON_Returns401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	bc := newTestBaseClient(srv.URL, "bad-token")

	req, err := bc.newRequest(context.Background(), http.MethodGet, "/v1/me", nil)
	require.NoError(t, err)

	var out struct{}
	err = bc.doJSON(req, &out)
	require.Error(t, err)

	var authErr *UnauthorizedError
	assert.ErrorAs(t, err, &authErr, "expected *UnauthorizedError for 401")
}

func TestBaseClient_DoJSON_Returns403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Spotify Premium required"))
	}))
	defer srv.Close()

	bc := newTestBaseClient(srv.URL, "token")

	req, err := bc.newRequest(context.Background(), http.MethodGet, "/v1/me", nil)
	require.NoError(t, err)

	var out struct{}
	err = bc.doJSON(req, &out)
	require.Error(t, err)

	var forbErr *ForbiddenError
	assert.ErrorAs(t, err, &forbErr, "expected *ForbiddenError for 403")
}

func TestBaseClient_DoJSON_Returns403_JSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"status":403,"message":"Premium required to use this endpoint."}}`))
	}))
	defer srv.Close()

	bc := newTestBaseClient(srv.URL, "token")

	req, err := bc.newRequest(context.Background(), http.MethodGet, "/v1/me", nil)
	require.NoError(t, err)

	var out struct{}
	err = bc.doJSON(req, &out)
	require.Error(t, err)

	var forbErr *ForbiddenError
	require.ErrorAs(t, err, &forbErr, "expected *ForbiddenError for 403")
	assert.Equal(t, "Premium required to use this endpoint.", forbErr.Message, "must extract message from JSON envelope")
}

func TestBaseClient_DoJSON_Returns403_JSONBody_EmptyMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"status":403}}`))
	}))
	defer srv.Close()

	bc := newTestBaseClient(srv.URL, "token")

	req, err := bc.newRequest(context.Background(), http.MethodGet, "/v1/me", nil)
	require.NoError(t, err)

	var out struct{}
	err = bc.doJSON(req, &out)
	require.Error(t, err)

	var forbErr *ForbiddenError
	require.ErrorAs(t, err, &forbErr)
	assert.Equal(t, "Spotify Premium required", forbErr.Message, "empty JSON message must yield default")
}

func TestBaseClient_DoJSON_Returns429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	bc := newTestBaseClient(srv.URL, "token")

	req, err := bc.newRequest(context.Background(), http.MethodGet, "/v1/me", nil)
	require.NoError(t, err)

	var out struct{}
	err = bc.doJSON(req, &out)
	require.Error(t, err)

	var rlErr *RateLimitError
	assert.ErrorAs(t, err, &rlErr, "expected *RateLimitError for 429")
	assert.Equal(t, 30, rlErr.RetryAfter)
}

func TestBaseClient_DoNoContent_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	bc := newTestBaseClient(srv.URL, "token")

	req, err := bc.newRequest(context.Background(), http.MethodPut, "/v1/me/player/pause", nil)
	require.NoError(t, err)

	err = bc.doNoContent(req)
	require.NoError(t, err)
}

func TestBaseClient_DoNoContent_Returns429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	bc := newTestBaseClient(srv.URL, "token")

	req, err := bc.newRequest(context.Background(), http.MethodPut, "/v1/me/player/pause", nil)
	require.NoError(t, err)

	err = bc.doNoContent(req)
	require.Error(t, err)

	var rlErr *RateLimitError
	assert.ErrorAs(t, err, &rlErr, "expected *RateLimitError for 429")
	assert.Equal(t, 10, rlErr.RetryAfter)
}

func TestNewBaseClient_DefaultsToSpotifyURL(t *testing.T) {
	bc := NewBaseClient("", "my-token")
	assert.Equal(t, spotifyAPIBaseURL, bc.baseURL)
}

func TestNewBaseClient_UsesProvidedURL(t *testing.T) {
	bc := NewBaseClient("https://mock.example.com", "my-token")
	assert.Equal(t, "https://mock.example.com", bc.baseURL)
}

func TestNewBaseClient_HasTimeout(t *testing.T) {
	bc := NewBaseClient("", "my-token")
	require.NotNil(t, bc.http, "http.Client must be initialised")
	assert.Equal(t, 30*time.Second, bc.HTTPTimeout(), "HTTP client timeout must be 30s")
}

// --- Gateway integration into BaseClient ---

func TestBaseClient_WithGateway_RoutesDoJSON(t *testing.T) {
	// When a Gateway is attached, doJSON should route through the gateway
	// (i.e. the request still reaches the server and returns the correct result).
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"test"}`))
	}))
	defer srv.Close()

	bc := NewBaseClient(srv.URL, "test-token")
	gw := NewGateway()
	bc.SetGateway(gw)

	req, err := bc.newRequest(context.Background(), http.MethodGet, "/v1/test", nil)
	require.NoError(t, err)

	var out struct {
		Name string `json:"name"`
	}
	err = bc.doJSON(req, &out)
	require.NoError(t, err)
	assert.Equal(t, "test", out.Name)
	assert.Equal(t, 1, callCount)
}

func TestBaseClient_WithGateway_RoutesDoNoContent(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	bc := NewBaseClient(srv.URL, "test-token")
	gw := NewGateway()
	bc.SetGateway(gw)

	req, err := bc.newRequest(context.Background(), http.MethodPut, "/v1/test", nil)
	require.NoError(t, err)
	err = bc.doNoContent(req)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestBaseClient_WithoutGateway_WorksAsBeforeGateway(t *testing.T) {
	// Backward compat: without a gateway the client works exactly as before.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"compat"}`))
	}))
	defer srv.Close()

	bc := NewBaseClient(srv.URL, "test-token")
	// No SetGateway call.

	req, err := bc.newRequest(context.Background(), http.MethodGet, "/v1/compat", nil)
	require.NoError(t, err)

	var out struct {
		Name string `json:"name"`
	}
	err = bc.doJSON(req, &out)
	require.NoError(t, err)
	assert.Equal(t, "compat", out.Name)
}
