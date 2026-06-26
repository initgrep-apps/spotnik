package app_test

// error_flow_test.go — Story 264: Cross-cutting error→toast integration tests.
//
// Verifies the full error-resilience flows through the root App model:
//   - 401 triggers a token refresh (and "Session expired" when refresh fails).
//   - 429 shows a rate-limit toast and activates backoff.
//   - 403 on a premium-only playback key shows "Spotify Premium required".
//   - Repeated playback poll failures surface a toast only on the 3rd consecutive
//     failure (toast throttling).

import (
	"net/http"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorToToast_401_TriggersRefresh verifies that a 401 from the API triggers
// the token-refresh command. When the token store has no refresh token, the refresh
// fails and the "Session expired" toast is shown — proving the full 401→refresh→toast
// chain is wired through the App.
func TestErrorToToast_401_TriggersRefresh(t *testing.T) {
	srv := unauthorizedServer()
	defer srv.Close()

	cfg := &config.Config{ClientID: "test-client-id"}
	store := keychain.NewInMemoryTokenStore() // no refresh token → refresh fails

	a := app.New(cfg, app.AppOptions{TokenStore: store})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))

	// Step 1: fetch playlists → command dispatched.
	model, fetchCmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	a = model.(*app.App)
	require.NotNil(t, fetchCmd)

	// Step 2: execute fetch → 401 → unauthorizedMsg.
	unauthorizedMsg := fetchCmd()

	// Step 3: feed unauthorizedMsg → refresh command dispatched.
	model, refreshCmd := a.Update(unauthorizedMsg)
	a = model.(*app.App)
	require.NotNil(t, refreshCmd, "401 must dispatch a token-refresh command")

	// Step 4: execute refresh → fails (no refresh token) → tokenRefreshedMsg(err).
	refreshResult := refreshCmd()

	// Step 5: feed tokenRefreshedMsg(err) → "Session expired" toast alert cmd.
	model, alertCmd := a.Update(refreshResult)
	a = model.(*app.App)
	require.NotNil(t, alertCmd, "failed refresh must emit a session-expired toast cmd")

	// Step 6: activate the toast and assert the message appears in View().
	activateAlertCmd(t, a, alertCmd)
	assert.Contains(t, a.View(), "Session expired",
		"401 with no refresh token should show the session-expired toast")
}

// TestErrorToToast_429_ShowsRateLimitToast verifies that a 429 response produces a
// RateLimitedMsg which, when fed back to the App, activates the backoff mechanism
// and renders a rate-limit toast.
func TestErrorToToast_429_ShowsRateLimitToast(t *testing.T) {
	srv := rateLimitServer("30")
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetLibrary(api.NewLibraryClient(srv.URL, "test-token"))
	// Initialize layout and dismiss splash so the grid view (and its toast overlay) render.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	a.Update(app.SplashDismissMsgForTest{})

	// Step 1: fetch playlists → 429 → RateLimitedMsg.
	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	require.NotNil(t, cmd)
	msg := cmd()
	rateLimitMsg, ok := msg.(panes.RateLimitedMsg)
	require.True(t, ok, "429 should produce RateLimitedMsg, got %T", msg)
	assert.Equal(t, 30, rateLimitMsg.RetryAfterSecs, "RetryAfterSecs must match Retry-After header")

	// Step 2: feed RateLimitedMsg → backoff activated + ratelimit toast cmd.
	model, alertCmd := a.Update(msg)
	a = model.(*app.App)
	assert.Greater(t, a.BackoffTicks(), 0, "RateLimitedMsg must activate backoff")
	require.NotNil(t, alertCmd, "429 must emit a ratelimit toast cmd")

	// Step 3: activate the toast. The cmd may be a tea.BatchMsg (toast + alert init),
	// so we execute every sub-command and feed each result back until the toast renders.
	activateAlertCmd(t, a, alertCmd)
	view := a.View()
	assert.Contains(t, view, "Rate-limited", "toast should mention rate limiting")
}

// activateAlertCmd executes a toast/alert command (possibly batched) and feeds the
// resulting alert message back to the App. The toast command is often a tea.BatchMsg
// containing [alertMsg, tea.Tick(throttleExpired), ...]. We execute ONLY the first
// sub-cmd (the alertMsg) and feed it; subsequent sub-cmds include real-time ticks
// (tea.Tick with backoff seconds) that would block the test, so we skip them.
func activateAlertCmd(t *testing.T, a *app.App, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	switch m := msg.(type) {
	case tea.BatchMsg:
		// Feed only the first sub-cmd — it is the alertMsg that renders the toast.
		// Other sub-cmds (tea.Tick for throttle expiry) would block on real timers.
		if len(m) > 0 && m[0] != nil {
			if subMsg := m[0](); subMsg != nil {
				a.Update(subMsg)
			}
		}
	default:
		if msg != nil {
			a.Update(msg)
		}
	}
}

// TestErrorToToast_403_ShowsPremiumRequired verifies that a free-tier user pressing
// a premium-only playback key (Space) gets a "Spotify Premium required" toast.
func TestErrorToToast_403_ShowsPremiumRequired(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// Free-tier user — IsPremium() returns false.
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "free"})
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	a.Update(app.SplashDismissMsgForTest{})

	// Space is a premium-only playback key — routing gates free-tier users.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeySpace})
	require.NotNil(t, cmd, "free-tier Space must emit a Premium-required toast cmd")

	// Execute the toast cmd and feed the alert message back to render the toast.
	activateAlertCmd(t, a, cmd)
	assert.Contains(t, a.View(), "Spotify Premium required",
		"free-tier Space should show the Premium-required toast")
}

// TestErrorToToast_PollingFailure_Throttled verifies that consecutive playback poll
// failures surface a warning toast only on the 3rd failure, not on the 1st or 2nd.
// A 4th failure must not produce a duplicate toast (counter stays > 3).
func TestErrorToToast_PollingFailure_Throttled(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	a.Update(app.SplashDismissMsgForTest{})

	pollErr := http.StatusText(http.StatusBadGateway) // any non-nil error
	failure := panes.PlaybackStateFetchedMsg{Err: &flowErr{msg: pollErr}}

	// 1st failure — no toast.
	_, cmd := a.Update(failure)
	assert.Nil(t, cmd, "1st consecutive playback failure must not emit a toast")

	// 2nd failure — no toast.
	_, cmd = a.Update(failure)
	assert.Nil(t, cmd, "2nd consecutive playback failure must not emit a toast")

	// 3rd failure — toast emitted.
	m, cmd = a.Update(failure)
	a = m.(*app.App)
	require.NotNil(t, cmd, "3rd consecutive playback failure must emit a warning toast")

	// Activate the toast (cmd may be batched) and assert the message in View().
	activateAlertCmd(t, a, cmd)
	view := a.View()
	assert.Contains(t, view, "Playback updates failing",
		"3rd failure should show the playback-failing toast")

	// 4th failure — no duplicate toast (counter > 3). The Update may still return a
	// cmd (the existing alert's own tick), but it must not produce a SECOND toast.
	// We do NOT drive the tick cmd (that would tick the existing toast). Instead we
	// assert the view still shows exactly one "Playback updates failing" toast.
	updated, _ := a.Update(failure)
	a = updated.(*app.App)
	view2 := a.View()
	assert.Equal(t, countStr(view2, "Playback updates failing"), 1,
		"4th consecutive failure must not emit a duplicate toast")
}

// countStr returns the number of non-overlapping occurrences of sub in s.
func countStr(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	n := 0
	for {
		i := indexOf(s, sub)
		if i < 0 {
			break
		}
		n++
		s = s[i+len(sub):]
	}
	return n
}

// indexOf returns the index of the first occurrence of sub in s, or -1.
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// flowErr is a lightweight error type for playback poll-failure tests.
type flowErr struct{ msg string }

func (e *flowErr) Error() string { return e.msg }
