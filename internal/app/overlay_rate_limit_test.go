package app

// overlay_rate_limit_test.go — White-box tests for Story 225: overlay rate-limit
// error delivery. Must be package app (not app_test) because the tests access
// unexported fields profileOverlayOpen, deviceOverlayOpen, profilePane, devicePane.

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// someTime is a fixed non-zero timestamp used to mark data as already fetched.
var someTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// TestRateLimitedMsg_DeliversErrorToProfileOverlay verifies that when a 429 occurs
// and the profile overlay is open with no profile loaded, the RateLimitedMsg handler
// forwards a synthetic UserProfileLoadedMsg{Err} to the profile overlay so it shows
// "Profile unavailable" instead of hanging on "Loading profile...".
func TestRateLimitedMsg_DeliversErrorToProfileOverlay(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	// Open profile overlay — this sets profileOverlayOpen = true and Init() fires
	// FetchCurrentUserRequestMsg. The store has no user profile loaded (ID == "").
	a.profileOverlayOpen = true
	require.Equal(t, "", a.store.UserID(), "profile ID should be empty before rate limit")

	// Send RateLimitedMsg — simulates a 429 from Spotify.
	model, _ := a.Update(panes.RateLimitedMsg{RetryAfterSecs: 5})
	a = model.(*App)

	// The profile overlay should have received the synthetic error.
	assert.NotNil(t, a.ProfilePaneErr(), "profile overlay should have a rate-limit error after RateLimitedMsg")
	assert.Contains(t, a.ProfilePaneErr().Error(), "rate limited",
		"profile overlay error should mention rate limiting")
}

// TestRateLimitedMsg_DeliversErrorToDeviceOverlay verifies that when a 429 occurs
// and the device overlay is open with no devices fetched, the RateLimitedMsg handler
// forwards a synthetic DevicesLoadedMsg{Err} to the device overlay so it shows
// "Failed to load devices" instead of "No devices found".
func TestRateLimitedMsg_DeliversErrorToDeviceOverlay(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	// Open device overlay — this sets deviceOverlayOpen = true.
	// Init() fires FetchDevicesRequestMsg, but the store has no devices (DevicesFetchedAt is zero).
	a.deviceOverlayOpen = true
	require.True(t, a.store.DevicesFetchedAt().IsZero(), "devices should not be fetched before rate limit")

	// Send RateLimitedMsg — simulates a 429 from Spotify.
	model, _ := a.Update(panes.RateLimitedMsg{RetryAfterSecs: 5})
	a = model.(*App)

	// The device overlay should have received the synthetic error.
	assert.NotNil(t, a.devicePane.Err(), "device overlay should have a rate-limit error after RateLimitedMsg")
	assert.Contains(t, a.devicePane.Err().Error(), "rate limited",
		"device overlay error should mention rate limiting")
}

// TestRateLimitedMsg_SkipsProfileOverlayWhenClosed verifies that no error is
// delivered to the profile pane when the profile overlay is not open.
func TestRateLimitedMsg_SkipsProfileOverlayWhenClosed(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	// Profile overlay is NOT open.
	require.False(t, a.ProfileOverlayOpen(), "profile overlay should be closed")

	// Send RateLimitedMsg.
	model, _ := a.Update(panes.RateLimitedMsg{RetryAfterSecs: 5})
	a = model.(*App)

	// Profile pane should NOT have an error.
	assert.Nil(t, a.ProfilePaneErr(), "profile overlay should have no error when overlay is closed")
}

// TestRateLimitedMsg_SkipsDeviceOverlayWhenClosed verifies that no error is
// delivered to the device pane when the device overlay is not open.
func TestRateLimitedMsg_SkipsDeviceOverlayWhenClosed(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	// Device overlay is NOT open.
	require.False(t, a.DeviceOverlayOpen(), "device overlay should be closed")

	// Send RateLimitedMsg.
	model, _ := a.Update(panes.RateLimitedMsg{RetryAfterSecs: 5})
	a = model.(*App)

	// Device pane should NOT have an error.
	assert.Nil(t, a.devicePane.Err(), "device overlay should have no error when overlay is closed")
}

// TestRateLimitedMsg_SkipsProfileOverlayWhenDataLoaded verifies that no error is
// delivered to the profile pane when the user profile is already loaded (ID != "").
func TestRateLimitedMsg_SkipsProfileOverlayWhenDataLoaded(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	// Open profile overlay and simulate a profile already loaded.
	a.profileOverlayOpen = true
	a.store.SetUserProfile(domain.UserProfile{ID: "user-123", DisplayName: "Test", Product: "premium", Country: "US"})
	require.NotEqual(t, "", a.store.UserID(), "profile ID should be non-empty")

	// Send RateLimitedMsg.
	model, _ := a.Update(panes.RateLimitedMsg{RetryAfterSecs: 5})
	a = model.(*App)

	// Profile pane should NOT have an error — it already has data.
	assert.Nil(t, a.ProfilePaneErr(), "profile overlay should have no error when profile is already loaded")
}

// TestRateLimitedMsg_SkipsDeviceOverlayWhenDataLoaded verifies that no error is
// delivered to the device pane when devices have already been fetched.
func TestRateLimitedMsg_SkipsDeviceOverlayWhenDataLoaded(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	// Open device overlay and simulate devices already fetched.
	a.deviceOverlayOpen = true
	a.store.SetDevicesFetchedAt(someTime) // non-zero = devices have been fetched
	require.False(t, a.store.DevicesFetchedAt().IsZero(), "devices should be marked as fetched")

	// Send RateLimitedMsg.
	model, _ := a.Update(panes.RateLimitedMsg{RetryAfterSecs: 5})
	a = model.(*App)

	// Device pane should NOT have an error — it already has data.
	assert.Nil(t, a.devicePane.Err(), "device overlay should have no error when devices are already fetched")
}
