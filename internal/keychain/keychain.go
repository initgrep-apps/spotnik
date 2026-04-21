// Package keychain provides an abstraction for storing and retrieving
// OAuth tokens. It supports two implementations: one backed by the OS
// keychain (for production) and one in-memory (for tests).
package keychain

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	gokeyring "github.com/zalando/go-keyring"
)

// Service is the keyring service name used for all spotnik credentials.
const Service = "spotnik"

// Key constants for the three keychain entries.
const (
	KeyAccessToken  = "spotnik:access_token"
	KeyRefreshToken = "spotnik:refresh_token"
	KeyTokenExpiry  = "spotnik:token_expiry"
)

// expirySoonThreshold is how soon a token must expire before we consider
// it "expiring soon" and trigger a proactive refresh.
const expirySoonThreshold = 5 * time.Minute

// TokenStore defines the interface for storing and retrieving OAuth tokens.
// Both the production OS keychain and the in-memory test store implement this.
type TokenStore interface {
	// Get retrieves the value for the given key.
	Get(key string) (string, error)
	// Set stores a value for the given key.
	Set(key, value string) error
	// Delete removes all three token keys from the store.
	Delete() error
	// GetExpiry returns the stored token expiry as a time.Time.
	GetExpiry() (time.Time, error)
	// IsExpiringSoon returns true if the token expires within 5 minutes.
	IsExpiringSoon() (bool, error)
}

// KeychainTokenStore is the production implementation of TokenStore.
// It persists tokens in the OS-native keychain via go-keyring.
type KeychainTokenStore struct{}

// NewKeychainTokenStore creates a new KeychainTokenStore.
func NewKeychainTokenStore() *KeychainTokenStore {
	return &KeychainTokenStore{}
}

// Get retrieves the value for the given key from the OS keychain.
func (s *KeychainTokenStore) Get(key string) (string, error) {
	val, err := gokeyring.Get(Service, key)
	if err != nil {
		return "", fmt.Errorf("getting %s from keychain: %w", key, err)
	}
	return val, nil
}

// Set stores a value for the given key in the OS keychain.
func (s *KeychainTokenStore) Set(key, value string) error {
	if err := gokeyring.Set(Service, key, value); err != nil {
		return fmt.Errorf("setting %s in keychain: %w", key, err)
	}
	return nil
}

// Delete removes all three token keys from the OS keychain.
// Keys that are already absent (ErrNotFound) are skipped silently — the end
// state is the same whether or not the key existed, so not-found is not an
// error from the caller's perspective. Other errors are collected and returned.
func (s *KeychainTokenStore) Delete() error {
	keys := []string{KeyAccessToken, KeyRefreshToken, KeyTokenExpiry}
	var errs []error
	for _, key := range keys {
		if err := gokeyring.Delete(Service, key); err != nil {
			if errors.Is(err, gokeyring.ErrNotFound) {
				continue // key absent — nothing to delete, not an error
			}
			errs = append(errs, fmt.Errorf("deleting %s: %w", key, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("deleting keychain tokens: %v", errs)
	}
	return nil
}

// GetExpiry retrieves the stored Unix timestamp string and parses it to time.Time.
func (s *KeychainTokenStore) GetExpiry() (time.Time, error) {
	return getExpiry(s)
}

// IsExpiringSoon returns true if the access token expires within 5 minutes.
func (s *KeychainTokenStore) IsExpiringSoon() (bool, error) {
	return isExpiringSoon(s)
}

// InMemoryTokenStore is a test-only in-memory implementation of TokenStore.
// It satisfies the same interface as KeychainTokenStore without touching the OS keychain.
type InMemoryTokenStore struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewInMemoryTokenStore creates a new empty InMemoryTokenStore.
func NewInMemoryTokenStore() *InMemoryTokenStore {
	return &InMemoryTokenStore{
		data: make(map[string]string),
	}
}

// Get retrieves the value for the given key from the in-memory store.
// Returns a descriptive error if the key is not found.
func (s *InMemoryTokenStore) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in token store", key)
	}
	return val, nil
}

// Set stores a value for the given key in the in-memory store.
func (s *InMemoryTokenStore) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

// Delete removes all three token keys from the in-memory store.
func (s *InMemoryTokenStore) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, KeyAccessToken)
	delete(s.data, KeyRefreshToken)
	delete(s.data, KeyTokenExpiry)
	return nil
}

// GetExpiry retrieves and parses the stored Unix timestamp string.
func (s *InMemoryTokenStore) GetExpiry() (time.Time, error) {
	return getExpiry(s)
}

// IsExpiringSoon returns true if the access token expires within 5 minutes.
func (s *InMemoryTokenStore) IsExpiringSoon() (bool, error) {
	return isExpiringSoon(s)
}

// getExpiry is a shared helper that reads the expiry key and parses the Unix timestamp.
func getExpiry(s TokenStore) (time.Time, error) {
	raw, err := s.Get(KeyTokenExpiry)
	if err != nil {
		return time.Time{}, fmt.Errorf("getting token expiry: %w", err)
	}
	ts, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing token expiry %q: %w", raw, err)
	}
	return time.Unix(ts, 0), nil
}

// isExpiringSoon is a shared helper that checks if the token expires within
// the configured threshold.
func isExpiringSoon(s TokenStore) (bool, error) {
	expiry, err := getExpiry(s)
	if err != nil {
		return false, err
	}
	return time.Until(expiry) < expirySoonThreshold, nil
}
