package app

// user_profile_test.go — Internal (white-box) tests for the userProfileLoadedMsg
// routing handler and buildFetchCurrentUserCmd command factory.
// Uses package app (not app_test) so it can access the unexported message type,
// errNilClient sentinel, and inject a.userAPI directly.

import (
	"errors"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/api/apitest"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
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
	assert.Equal(t, domain.UserProfile{ID: "existing-id"}, a.store.UserProfile(), "existing profile must not be overwritten by empty msg")
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

// --- Premium gate tests ---

// isPlaybackRequestMsg returns true if cmd (possibly a batch) produces a panes.PlaybackRequestMsg.
// Used to distinguish "gate blocked → toast" from "gate passed → playback dispatched".
func isPlaybackRequestMsg(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	if _, ok := msg.(panes.PlaybackRequestMsg); ok {
		return true
	}
	// Handle batch commands.
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			inner := c()
			if _, ok := inner.(panes.PlaybackRequestMsg); ok {
				return true
			}
		}
	}
	return false
}

// TestPremiumGate_FreeUser_PlaybackKeyEmitsToast verifies that pressing a Premium-only playback key
// while the store has a free-tier profile returns a non-nil cmd that does NOT contain
// a PlaybackRequestMsg — it should be a warning toast cmd instead.
func TestPremiumGate_FreeUser_PlaybackKeyEmitsToast(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := New(cfg, AppOptions{})
	// Resize so panes exist.
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// Mark user as free tier.
	a.store.SetUserProfile(domain.UserProfile{ID: "user-free", Product: "free"})

	playbackKeys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{' '}},
		{Type: tea.KeyRunes, Runes: []rune{'n'}},
		{Type: tea.KeyLeft},
		{Type: tea.KeyRight},
		{Type: tea.KeyRunes, Runes: []rune{'+'}},
		{Type: tea.KeyRunes, Runes: []rune{'-'}},
		{Type: tea.KeyRunes, Runes: []rune{'s'}},
		{Type: tea.KeyRunes, Runes: []rune{'r'}},
	}

	for _, key := range playbackKeys {
		_, cmd := a.Update(key)
		require.NotNil(t, cmd, "free user: playback key must return a non-nil cmd (warning toast)")
		assert.False(t, isPlaybackRequestMsg(cmd),
			"free user: cmd must NOT be a PlaybackRequestMsg (gate should block before dispatching to NowPlayingPane)")
	}
}

// TestPremiumGate_FreeUser_VisualizerKeyNotBlocked verifies that 'v' (visualizer cycle)
// is NOT blocked by the premium gate — it is a local UI action with no API call.
// A free user pressing 'v' should get the same result as a premium user pressing 'v'.
func TestPremiumGate_FreeUser_VisualizerKeyNotBlocked(t *testing.T) {
	vKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}

	// Premium user result.
	cfgP := &config.Config{}
	cfgP.Preferences.Theme = "black"
	ap := New(cfgP, AppOptions{})
	ap.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	ap.store.SetUserProfile(domain.UserProfile{ID: "premium", Product: "premium"})
	_, premiumCmd := ap.Update(vKey)

	// Free user result.
	cfgF := &config.Config{}
	cfgF.Preferences.Theme = "black"
	af := New(cfgF, AppOptions{})
	af.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	af.store.SetUserProfile(domain.UserProfile{ID: "free", Product: "free"})
	_, freeCmd := af.Update(vKey)

	// Both should produce the same (non-nil) result: VisualizerPatternChangedMsg.
	// The gate must NOT block 'v' for free users.
	require.NotNil(t, freeCmd, "free user: 'v' must not be blocked (gate exempt)")
	require.NotNil(t, premiumCmd, "premium user: 'v' must return cmd")

	// Both cmds should produce VisualizerPatternChangedMsg (same type for both tiers).
	freeMsg := freeCmd()
	premiumMsg := premiumCmd()
	assert.Equal(t, fmt.Sprintf("%T", premiumMsg), fmt.Sprintf("%T", freeMsg),
		"'v' key should produce same message type for free and premium users")
}

// TestPremiumGate_PremiumUser_PlaybackKeyDispatches verifies that pressing a playback key
// while the store has a premium profile dispatches to NowPlayingPane — the cmd contains
// a PlaybackRequestMsg (not short-circuited by the gate).
func TestPremiumGate_PremiumUser_PlaybackKeyDispatches(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := New(cfg, AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// Mark user as premium.
	a.store.SetUserProfile(domain.UserProfile{ID: "user-premium", Product: "premium"})

	// Space (play/pause) dispatches to NowPlayingPane which emits PlaybackRequestMsg.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	require.NotNil(t, cmd, "premium user: playback key must return a non-nil cmd")
	assert.True(t, isPlaybackRequestMsg(cmd),
		"premium user: cmd must contain a PlaybackRequestMsg (gate passed, dispatched to NowPlayingPane)")
}

// TestPremiumGate_FreeUser_TransferPlaybackEmitsToast verifies that when a free-tier user
// selects a device to transfer playback, the app emits a "Spotify Premium required" toast
// and does NOT batch a buildTransferPlaybackCmd.
// Before the gate: returns tea.Batch(buildTransferPlaybackCmd, infoToast) — a BatchMsg.
// After the gate:  returns only the warningToast — NOT a BatchMsg.
func TestPremiumGate_FreeUser_TransferPlaybackEmitsToast(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := New(cfg, AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// Mark user as free tier.
	a.store.SetUserProfile(domain.UserProfile{ID: "user-free", Product: "free"})

	_, cmd := a.Update(panes.TransferPlaybackMsg{DeviceID: "dev-1", DeviceName: "My Speaker"})
	require.NotNil(t, cmd, "free user: TransferPlaybackMsg must return a non-nil cmd (warning toast)")

	// Without the gate the handler returns tea.Batch(buildTransferPlaybackCmd, infoToast).
	// tea.Batch() returns a Cmd that produces a tea.BatchMsg when called.
	// The gate should short-circuit and return a single toast cmd — NOT a BatchMsg.
	msg := cmd()
	_, isBatch := msg.(tea.BatchMsg)
	assert.False(t, isBatch,
		"free user: cmd must NOT be a BatchMsg (gate should return single toast, not batch with transfer cmd)")
}

// --- AddToQueue premium gate tests ---

// isAddToQueueResultMsg returns true if cmd produces a panes.AddToQueueResultMsg.
// Used to distinguish "gate blocked → toast" from "gate passed → API cmd dispatched".
func isAddToQueueResultMsg(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	if _, ok := msg.(panes.AddToQueueResultMsg); ok {
		return true
	}
	// Handle batch commands.
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			inner := c()
			if _, ok := inner.(panes.AddToQueueResultMsg); ok {
				return true
			}
		}
	}
	return false
}

// TestPremiumGate_FreeUser_AddToQueueEmitsToast verifies that a free-tier user
// sending AddToQueueMsg receives a warning toast and NOT an AddToQueueResultMsg.
// The premium gate in the AddToQueueMsg handler must intercept before buildAddToQueueCmd.
func TestPremiumGate_FreeUser_AddToQueueEmitsToast(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := New(cfg, AppOptions{})
	// Mark user as free tier.
	a.store.SetUserProfile(domain.UserProfile{ID: "user-free", Product: "free"})

	_, cmd := a.Update(panes.AddToQueueMsg{TrackURI: "spotify:track:abc", TrackName: "Test Song"})

	require.NotNil(t, cmd, "free user: AddToQueueMsg must return a non-nil cmd (warning toast)")
	assert.False(t, isAddToQueueResultMsg(cmd),
		"free user: cmd must NOT produce AddToQueueResultMsg (gate should block before API call)")
}

// TestPremiumGate_PremiumUser_AddToQueueDispatches verifies that a premium-tier user
// sending AddToQueueMsg bypasses the gate and dispatches a buildAddToQueueCmd.
// With no player configured the result is AddToQueueResultMsg (errNilClient) — proving
// the API cmd was dispatched rather than a plain toast.
func TestPremiumGate_PremiumUser_AddToQueueDispatches(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := New(cfg, AppOptions{})
	// Mark user as premium.
	a.store.SetUserProfile(domain.UserProfile{ID: "user-premium", Product: "premium"})

	_, cmd := a.Update(panes.AddToQueueMsg{TrackURI: "spotify:track:abc", TrackName: "Test Song"})

	require.NotNil(t, cmd, "premium user: AddToQueueMsg must return a non-nil cmd")
	assert.True(t, isAddToQueueResultMsg(cmd),
		"premium user: cmd must produce AddToQueueResultMsg (gate passed, buildAddToQueueCmd dispatched)")
}

// TestPlaybackCmdSentMsg_ForbiddenEmitsWarningToast verifies that a 403 ForbiddenError
// wrapped in PlaybackCmdSentMsg emits "Spotify Premium required" in the toast overlay
// and does NOT show a generic "not available" message that would confuse users.
func TestPlaybackCmdSentMsg_ForbiddenEmitsWarningToast(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := New(cfg, AppOptions{})
	// Resize so the view renders properly.
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	forbiddenErr := &api.ForbiddenError{Message: "Spotify Premium required"}
	_, cmd := a.Update(panes.PlaybackCmdSentMsg{Err: forbiddenErr})

	require.NotNil(t, cmd, "403 PlaybackCmdSentMsg must return a non-nil cmd (toast + fetch)")

	// The handler returns tea.Batch(fetchPlaybackStateCmd, alertCmd).
	// Execute the batch to get the BatchMsg, then feed each cmd to Update.
	batchMsg := cmd()
	batchCmds, ok := batchMsg.(tea.BatchMsg)
	require.True(t, ok, "PlaybackCmdSentMsg 403 handler must return a BatchMsg, got %T", batchMsg)

	// Feed each sub-cmd result to Update to activate the alert overlay.
	for _, c := range batchCmds {
		if c == nil {
			continue
		}
		innerMsg := c()
		updatedModel, _ := a.Update(innerMsg)
		if app, ok := updatedModel.(*App); ok {
			a = app
		}
	}

	output := a.View()
	assert.Contains(t, output, "Spotify Premium required",
		"403 PlaybackCmdSentMsg toast must show 'Spotify Premium required'")
	assert.NotContains(t, output, "not available",
		"403 PlaybackCmdSentMsg toast must NOT show generic 'not available' text")
}

// TestBuildFetchCurrentUserCmd covers the command closure for all key error paths
// and the happy path. It injects MockUser directly into a.userAPI so no HTTP server
// is needed.
func TestBuildFetchCurrentUserCmd(t *testing.T) {
	tests := []struct {
		name        string
		mockProfile domain.UserProfile
		mockErr     error
		wantMsgType interface{}
		check       func(t *testing.T, msg interface{})
	}{
		{
			name:        "success returns userProfileLoadedMsg with profile",
			mockProfile: domain.UserProfile{ID: "user-1", DisplayName: "Bob", Product: "premium"},
			mockErr:     nil,
			check: func(t *testing.T, msg interface{}) {
				t.Helper()
				m, ok := msg.(userProfileLoadedMsg)
				require.True(t, ok, "expected userProfileLoadedMsg, got %T", msg)
				assert.Nil(t, m.err)
				assert.Equal(t, "user-1", m.profile.ID)
				assert.Equal(t, "premium", m.profile.Product)
			},
		},
		{
			name:    "generic error returns userProfileLoadedMsg with err",
			mockErr: errors.New("connection refused"),
			check: func(t *testing.T, msg interface{}) {
				t.Helper()
				m, ok := msg.(userProfileLoadedMsg)
				require.True(t, ok, "expected userProfileLoadedMsg, got %T", msg)
				require.Error(t, m.err)
				assert.Contains(t, m.err.Error(), "connection refused")
			},
		},
		{
			name:    "403 ForbiddenError returns userProfileLoadedMsg with ForbiddenError",
			mockErr: &api.ForbiddenError{Message: "Premium required"},
			check: func(t *testing.T, msg interface{}) {
				t.Helper()
				m, ok := msg.(userProfileLoadedMsg)
				require.True(t, ok, "expected userProfileLoadedMsg, got %T", msg)
				require.Error(t, m.err)
				var forbErr *api.ForbiddenError
				assert.True(t, errors.As(m.err, &forbErr), "err should be *api.ForbiddenError")
			},
		},
		{
			name:    "429 RateLimitError returns panes.RateLimitedMsg",
			mockErr: &api.RateLimitError{RetryAfter: 5},
			check: func(t *testing.T, msg interface{}) {
				t.Helper()
				_, ok := msg.(panes.RateLimitedMsg)
				assert.True(t, ok, "expected panes.RateLimitedMsg for 429, got %T", msg)
			},
		},
		{
			name:    "401 UnauthorizedError returns unauthorizedMsg",
			mockErr: &api.UnauthorizedError{},
			check: func(t *testing.T, msg interface{}) {
				t.Helper()
				_, ok := msg.(unauthorizedMsg)
				assert.True(t, ok, "expected unauthorizedMsg for 401, got %T", msg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			a := New(cfg, AppOptions{})
			a.userAPI = &apitest.MockUser{
				ProfileResult: tt.mockProfile,
				ProfileErr:    tt.mockErr,
			}
			cmd := a.buildFetchCurrentUserCmd()
			require.NotNil(t, cmd, "buildFetchCurrentUserCmd must return a non-nil Cmd")
			msg := cmd()
			tt.check(t, msg)
		})
	}
}
