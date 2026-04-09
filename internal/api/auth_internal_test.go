// Tests in package api (not api_test) can access package-private functions.
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostTokenRequest_UsesInjectedClient verifies that postTokenRequest reaches
// the injected *http.Client rather than falling through to http.DefaultClient.
// If the test server is not called, http.DefaultClient is still being used.
func TestPostTokenRequest_UsesInjectedClient(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"a","refresh_token":"r","expires_in":3600}`))
	}))
	defer srv.Close()

	pair, err := postTokenRequest(
		context.Background(),
		&http.Client{},
		srv.URL+"/token",
		url.Values{"grant_type": {"client_credentials"}},
	)

	require.NoError(t, err)
	assert.True(t, called, "test server was not reached — http.DefaultClient still used")
	assert.Equal(t, "a", pair.AccessToken)
}
