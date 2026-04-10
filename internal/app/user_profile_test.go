package app

// user_profile_test.go — Internal (white-box) tests for the userProfileLoadedMsg
// routing handler and buildFetchCurrentUserCmd command factory.
// Uses package app (not app_test) so it can access the unexported message type,
// errNilClient sentinel, and inject a.userAPI directly.

import (
	"errors"
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

// TestPremiumGate_FreeUser_PlaybackKeyEmitsToast verifies that pressing a playback key
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
