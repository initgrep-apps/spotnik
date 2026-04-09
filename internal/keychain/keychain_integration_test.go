//go:build integration

package keychain_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewKeychainTokenStore_ReturnsNonNil verifies the constructor works.
func TestNewKeychainTokenStore_ReturnsNonNil(t *testing.T) {
	store := keychain.NewKeychainTokenStore()
	require.NotNil(t, store)
}

// TestKeychainTokenStore_SetAndGet verifies the real OS keychain store can
// store and retrieve a value. This test requires OS keychain access.
func TestKeychainTokenStore_SetAndGet(t *testing.T) {
	store := keychain.NewKeychainTokenStore()

	// Use test-specific keys to avoid interfering with real credentials.
	const testKey = "spotnik:test_access_token"

	// Cleanup on exit.
	t.Cleanup(func() {
		_ = store.Delete()
	})

	// Set via the store's generic Set interface.
	err := store.Set(testKey, "test-value")
	if err != nil {
		t.Skipf("OS keychain not available: %v", err)
	}

	val, err := store.Get(testKey)
	require.NoError(t, err)
	assert.Equal(t, "test-value", val)
}

// TestKeychainTokenStore_Delete verifies the real OS keychain store deletes all token keys.
func TestKeychainTokenStore_Delete(t *testing.T) {
	store := keychain.NewKeychainTokenStore()

	// Set all three keys first.
	err := store.Set(keychain.KeyAccessToken, "test-access")
	if err != nil {
		t.Skipf("OS keychain not available: %v", err)
	}
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "test-refresh"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "1234567890"))

	// Delete should clear all three.
	err = store.Delete()
	require.NoError(t, err)
}

// TestKeychainTokenStore_GetMissing verifies Get on a non-existent key returns an error.
func TestKeychainTokenStore_GetMissing(t *testing.T) {
	store := keychain.NewKeychainTokenStore()

	// Clean up in case key exists from a previous test run.
	_ = store.Delete()

	_, err := store.Get(keychain.KeyAccessToken)
	if err == nil {
		// Key unexpectedly existed — skip this test.
		t.Skip("access token key unexpectedly exists in keychain")
	}
	require.Error(t, err)
}

// TestKeychainTokenStore_IsExpiringSoon verifies IsExpiringSoon on the real keychain.
func TestKeychainTokenStore_IsExpiringSoon(t *testing.T) {
	store := keychain.NewKeychainTokenStore()

	// Set up a valid token expiry in the keychain.
	expiry := time.Now().Add(10 * time.Minute)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())))
	t.Cleanup(func() { _ = store.Delete() })

	soon, err := store.IsExpiringSoon()
	require.NoError(t, err)
	assert.False(t, soon, "token expiring in 10 minutes should not be expiring soon")
}

// TestKeychainTokenStore_GetExpiry verifies GetExpiry on the real keychain.
func TestKeychainTokenStore_GetExpiry(t *testing.T) {
	store := keychain.NewKeychainTokenStore()

	expected := time.Unix(1735689600, 0)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "1735689600"))
	t.Cleanup(func() { _ = store.Delete() })

	got, err := store.GetExpiry()
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

// TestKeychainTokenStore_Set verifies Set on the real keychain.
func TestKeychainTokenStore_Set(t *testing.T) {
	store := keychain.NewKeychainTokenStore()
	t.Cleanup(func() { _ = store.Delete() })

	require.NoError(t, store.Set(keychain.KeyAccessToken, "test-access"))

	val, err := store.Get(keychain.KeyAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "test-access", val)
}
