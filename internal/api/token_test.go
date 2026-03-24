package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStaticTokenProvider_ReturnsToken verifies that StaticTokenProvider always
// returns the exact token it was constructed with.
func TestStaticTokenProvider_ReturnsToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{name: "normal token", token: "my-access-token"},
		{name: "empty token", token: ""},
		{name: "long token", token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.very-long-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &StaticTokenProvider{Token: tt.token}
			got, err := p.AccessToken(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.token, got)
		})
	}
}

// TestStaticTokenProvider_IgnoresContext verifies that a cancelled context does
// not affect StaticTokenProvider — it always succeeds immediately.
func TestStaticTokenProvider_IgnoresContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	p := &StaticTokenProvider{Token: "token-abc"}
	got, err := p.AccessToken(ctx)
	require.NoError(t, err)
	assert.Equal(t, "token-abc", got)
}

// TestNewBaseClientWithProvider_UsesProvider verifies that NewBaseClientWithProvider
// stores the given provider and newRequest calls it per request.
func TestNewBaseClientWithProvider_UsesProvider(t *testing.T) {
	p := &StaticTokenProvider{Token: "provider-token"}
	bc := NewBaseClientWithProvider("https://api.example.com", p)

	req, err := bc.newRequest(context.Background(), "GET", "/v1/me", nil)
	require.NoError(t, err)
	assert.Equal(t, "Bearer provider-token", req.Header.Get("Authorization"))
}
