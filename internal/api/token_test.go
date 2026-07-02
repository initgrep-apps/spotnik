package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/keychain"
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

// --- RefreshableTokenProvider tests ---

func TestRefreshableTokenProvider_ReturnsTokenWhenNotExpiring(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "refresh-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(1*time.Hour).Unix())))

	p := NewRefreshableTokenProvider(store, "client-id", "", http.DefaultClient)
	token, err := p.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "valid-token", token)
}

func TestRefreshableTokenProvider_RefreshesWhenExpiringSoon(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "old-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "refresh-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(1*time.Minute).Unix())))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "fresh-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer srv.Close()

	p := NewRefreshableTokenProvider(store, "client-id", srv.URL, http.DefaultClient)
	token, err := p.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "fresh-token", token)

	stored, err := store.Get(keychain.KeyAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "fresh-token", stored)
}

func TestRefreshableTokenProvider_ReturnsExistingTokenOnRefreshFailure(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "old-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "refresh-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(1*time.Minute).Unix())))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	p := NewRefreshableTokenProvider(store, "client-id", srv.URL, http.DefaultClient)
	token, err := p.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "old-token", token)
}

func TestRefreshableTokenProvider_ErrorWhenNoAccessToken(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()

	p := NewRefreshableTokenProvider(store, "client-id", "", http.DefaultClient)
	_, err := p.AccessToken(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading access token")
}

func TestRefreshableTokenProvider_ErrorWhenEmptyAccessToken(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, ""))

	p := NewRefreshableTokenProvider(store, "client-id", "", http.DefaultClient)
	_, err := p.AccessToken(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no access token")
}

func TestRefreshableTokenProvider_RefreshesWhenNoExpiryKey(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "old-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "refresh-token"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "fresh-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer srv.Close()

	p := NewRefreshableTokenProvider(store, "client-id", srv.URL, http.DefaultClient)
	token, err := p.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "fresh-token", token)
}

func TestRefreshableTokenProvider_RefreshesWhenInvalidExpiryFormat(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "old-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "refresh-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "not-a-number"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "fresh-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer srv.Close()

	p := NewRefreshableTokenProvider(store, "client-id", srv.URL, http.DefaultClient)
	token, err := p.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "fresh-token", token)
}

func TestRefreshableTokenProvider_NoRefreshWhenNoRefreshToken(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "old-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(1*time.Minute).Unix())))

	p := NewRefreshableTokenProvider(store, "client-id", "", http.DefaultClient)
	token, err := p.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "old-token", token)
}

func TestRefreshableTokenProvider_ConcurrentAccess(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "old-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "refresh-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(1*time.Minute).Unix())))

	var callCount int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "fresh-token-" + strconv.Itoa(callCount),
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer srv.Close()

	p := NewRefreshableTokenProvider(store, "client-id", srv.URL, http.DefaultClient)

	var wg sync.WaitGroup
	results := make([]string, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			token, err := p.AccessToken(context.Background())
			require.NoError(t, err)
			results[idx] = token
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 1, callCount, "only one refresh should happen under mutex")

	for i, token := range results {
		assert.NotEmpty(t, token, "goroutine %d got empty token", i)
	}
}

func TestRefreshableTokenProvider_StoresNewExpiry(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "old-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "refresh-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(1*time.Minute).Unix())))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "fresh-token",
			"expires_in":   7200,
			"token_type":   "Bearer",
		})
	}))
	defer srv.Close()

	p := NewRefreshableTokenProvider(store, "client-id", srv.URL, http.DefaultClient)
	_, err := p.AccessToken(context.Background())
	require.NoError(t, err)

	expiryStr, err := store.Get(keychain.KeyTokenExpiry)
	require.NoError(t, err)
	expiryUnix, err := strconv.ParseInt(expiryStr, 10, 64)
	require.NoError(t, err)
	expiry := time.Unix(expiryUnix, 0)

	assert.True(t, expiry.After(time.Now().Add(1*time.Hour)), "expiry should be at least 1 hour in the future")
}

func TestSetTokenProvider_UpdatesProvider(t *testing.T) {
	bc := NewBaseClientWithProvider("https://api.example.com", &StaticTokenProvider{Token: "old"})
	bc.SetTokenProvider(&StaticTokenProvider{Token: "new"})

	req, err := bc.newRequest(context.Background(), "GET", "/v1/me", nil)
	require.NoError(t, err)
	assert.Equal(t, "Bearer new", req.Header.Get("Authorization"))
}
