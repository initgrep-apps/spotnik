package app

// user_profile_test.go — Internal (white-box) tests for the userProfileLoadedMsg
// routing handler. Uses package app (not app_test) so it can access the unexported
// message type.

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestUserProfileLoadedMsg_StoresUserIDInStore verifies that the routing handler
// for userProfileLoadedMsg writes the user ID to the store.
func TestUserProfileLoadedMsg_StoresUserIDInStore(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	assert.Equal(t, "", a.store.UserID(), "user ID starts empty")

	a.Update(userProfileLoadedMsg{userID: "user-abc"})

	assert.Equal(t, "user-abc", a.store.UserID(), "routing handler must write userID to store")
}

// TestUserProfileLoadedMsg_IgnoresEmptyUserID verifies that an empty userID
// does not overwrite an existing user ID (guard for duplicate or partial messages).
func TestUserProfileLoadedMsg_IgnoresEmptyUserID(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})
	a.store.SetUserID("existing-id")

	a.Update(userProfileLoadedMsg{userID: ""})

	assert.Equal(t, "existing-id", a.store.UserID(), "empty userID must not overwrite existing ID")
}

// TestUserProfileLoadedMsg_ErrorDoesNotSetUserID verifies that a failed profile
// fetch does not write anything to the store.
func TestUserProfileLoadedMsg_ErrorDoesNotSetUserID(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	a.Update(userProfileLoadedMsg{err: errNilClient})

	assert.Equal(t, "", a.store.UserID(), "error result must not set a user ID")
}
