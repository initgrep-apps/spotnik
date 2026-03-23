package keychain_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInMemoryStore_SetAndGet verifies round-trip store and retrieve of a value.
func TestInMemoryStore_SetAndGet(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	err := store.Set(keychain.KeyAccessToken, "my-access-token")
	require.NoError(t, err)

	val, err := store.Get(keychain.KeyAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "my-access-token", val)
}

// TestInMemoryStore_Delete verifies that Delete removes the value,
// and that subsequent Get returns an error.
func TestInMemoryStore_Delete(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()

	// Populate all three keys.
	require.NoError(t, store.Set(keychain.KeyAccessToken, "access"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "refresh"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "1234567890"))

	// Delete clears all keys.
	err := store.Delete()
	require.NoError(t, err)

	_, err = store.Get(keychain.KeyAccessToken)
	require.Error(t, err)

	_, err = store.Get(keychain.KeyRefreshToken)
	require.Error(t, err)

	_, err = store.Get(keychain.KeyTokenExpiry)
	require.Error(t, err)
}

// TestInMemoryStore_GetMissing verifies that Get on a missing key returns a descriptive error.
func TestInMemoryStore_GetMissing(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	_, err := store.Get(keychain.KeyAccessToken)
	require.Error(t, err)
	assert.Contains(t, err.Error(), keychain.KeyAccessToken)
}

// TestIsExpiringSoon_True verifies that IsExpiringSoon returns true when
// the token expires within 5 minutes.
func TestIsExpiringSoon_True(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()

	// Set expiry to 4 minutes from now (within the 5-minute threshold).
	expiry := time.Now().Add(4 * time.Minute)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, formatUnix(expiry)))

	soon, err := store.IsExpiringSoon()
	require.NoError(t, err)
	assert.True(t, soon)
}

// TestIsExpiringSoon_False verifies that IsExpiringSoon returns false when
// the token expires more than 5 minutes from now.
func TestIsExpiringSoon_False(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()

	// Set expiry to 10 minutes from now (outside the 5-minute threshold).
	expiry := time.Now().Add(10 * time.Minute)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, formatUnix(expiry)))

	soon, err := store.IsExpiringSoon()
	require.NoError(t, err)
	assert.False(t, soon)
}

// TestIsExpiringSoon_AlreadyExpired verifies that IsExpiringSoon returns true
// for a timestamp in the past.
func TestIsExpiringSoon_AlreadyExpired(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()

	// Set expiry to 1 hour ago.
	expiry := time.Now().Add(-1 * time.Hour)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, formatUnix(expiry)))

	soon, err := store.IsExpiringSoon()
	require.NoError(t, err)
	assert.True(t, soon)
}

// TestGetExpiry_ValidTimestamp verifies that GetExpiry parses a Unix timestamp
// string into the correct time.Time.
func TestGetExpiry_ValidTimestamp(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()

	expected := time.Unix(1735689600, 0)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "1735689600"))

	got, err := store.GetExpiry()
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

// TestGetExpiry_InvalidTimestamp verifies that GetExpiry returns an error
// for a non-numeric string.
func TestGetExpiry_InvalidTimestamp(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "not-a-number"))

	_, err := store.GetExpiry()
	require.Error(t, err)
}

// TestKeychain_ImplementsInterface is a compile-time interface check.
// It verifies that both KeychainTokenStore and InMemoryTokenStore implement TokenStore.
func TestKeychain_ImplementsInterface(t *testing.T) {
	t.Helper()
	// These assignments fail at compile time if the interfaces are not satisfied.
	var _ keychain.TokenStore = (*keychain.KeychainTokenStore)(nil)
	var _ keychain.TokenStore = (*keychain.InMemoryTokenStore)(nil)
}

// formatUnix converts a time.Time to its Unix timestamp as a string.
func formatUnix(t time.Time) string {
	return fmt.Sprintf("%d", t.Unix())
}
