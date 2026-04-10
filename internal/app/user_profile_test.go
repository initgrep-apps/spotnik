package app

// user_profile_test.go — Internal (white-box) tests for the userProfileLoadedMsg
// routing handler. Uses package app (not app_test) so it can access the unexported
// message type and errNilClient sentinel.

import (
	"errors"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserProfileLoadedMsg_StoresUserProfileInStore verifies that the routing handler
// for userProfileLoadedMsg writes the full profile to the store.
func TestUserProfileLoadedMsg_StoresUserProfileInStore(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	assert.Equal(t, "", a.store.UserID(), "user ID starts empty")

	a.Update(userProfileLoadedMsg{profile: domain.UserProfile{
		ID:          "user-abc",
		DisplayName: "Alice",
		Product:     "premium",
		Country:     "US",
	}})

	assert.Equal(t, "user-abc", a.store.UserID(), "routing handler must write profile ID to store")
	assert.Equal(t, "Alice", a.store.UserProfile().DisplayName)
	assert.Equal(t, "premium", a.store.UserProfile().Product)
	assert.Equal(t, "US", a.store.UserProfile().Country)
}

// TestUserProfileLoadedMsg_IgnoresEmptyUserID verifies that an empty profile ID
// does not overwrite an existing profile (guard for duplicate or partial messages).
func TestUserProfileLoadedMsg_IgnoresEmptyUserID(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})
	a.store.SetUserProfile(domain.UserProfile{ID: "existing-id"})

	a.Update(userProfileLoadedMsg{profile: domain.UserProfile{}})

	assert.Equal(t, "existing-id", a.store.UserID(), "empty profile.ID must not overwrite existing profile")
}

// TestUserProfileLoadedMsg_ErrorDoesNotSetUserID verifies that a failed profile
// fetch does not write anything to the store, and emits a warning toast.
func TestUserProfileLoadedMsg_ErrorDoesNotSetUserID(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	_, cmd := a.Update(userProfileLoadedMsg{err: errNilClient})

	assert.Equal(t, "", a.store.UserID(), "errNilClient must not set a user ID")
	// errNilClient is a programming error — no user-visible toast expected.
	assert.Nil(t, cmd, "errNilClient should return nil cmd (no toast)")
}

// TestUserProfileLoadedMsg_NetworkErrorEmitsWarningToast verifies that a real
// network error (not errNilClient) surfaces a warning toast to the user.
func TestUserProfileLoadedMsg_NetworkErrorEmitsWarningToast(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	_, cmd := a.Update(userProfileLoadedMsg{err: errors.New("connection refused")})

	assert.Equal(t, "", a.store.UserID(), "network error must not set a user ID")
	require.NotNil(t, cmd, "network error must emit a warning toast command")
}
