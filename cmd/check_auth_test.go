package cmd

import (
	"fmt"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckAuthState_ValidToken(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "valid-refresh"))
	expiry := time.Now().Add(1 * time.Hour)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())))

	cfg := &config.Config{ClientID: "test-id"}
	authenticated, token := checkAuthState(cfg, store)

	assert.True(t, authenticated)
	assert.Equal(t, "valid-token", token)
}

func TestCheckAuthState_NoToken(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	cfg := &config.Config{ClientID: "test-id"}
	authenticated, token := checkAuthState(cfg, store)

	assert.False(t, authenticated)
	assert.Empty(t, token)
}

func TestCheckAuthState_EmptyToken(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, ""))

	cfg := &config.Config{ClientID: "test-id"}
	authenticated, token := checkAuthState(cfg, store)

	assert.False(t, authenticated)
	assert.Empty(t, token)
}

func TestCheckAuthState_MissingExpiry(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	// Token present but no expiry — IsExpiringSoon will error.
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-token"))

	cfg := &config.Config{ClientID: "test-id"}
	authenticated, token := checkAuthState(cfg, store)

	assert.False(t, authenticated)
	assert.Empty(t, token)
}

func TestCheckAuthState_ExpiringSoon_NoRefreshToken(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-token"))
	// Set expiry to 2 minutes from now (within the 5-minute threshold).
	expiry := time.Now().Add(2 * time.Minute)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())))
	// No refresh token — should return unauthenticated.

	cfg := &config.Config{ClientID: "test-id"}
	authenticated, token := checkAuthState(cfg, store)

	assert.False(t, authenticated)
	assert.Empty(t, token)
}
