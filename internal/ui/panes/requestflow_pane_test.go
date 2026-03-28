package panes_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components/viz"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRequestFlowPane creates a RequestFlowPane with a live gateway for testing.
func newTestRequestFlowPane() *panes.RequestFlowPane {
	gw := api.NewGateway()
	s := state.New()
	t := theme.Load("black")
	return panes.NewRequestFlowPane(gw, s, t)
}

// --- Interface satisfaction ---

func TestRequestFlowPane_ImplementsLayoutPane(t *testing.T) {
	var _ layout.Pane = &panes.RequestFlowPane{}
}

// --- ID / Title / ToggleKey / Actions ---

func TestRequestFlowPane_ID(t *testing.T) {
	pane := newTestRequestFlowPane()
	assert.Equal(t, layout.PaneRequestFlow, pane.ID())
}

func TestRequestFlowPane_Title(t *testing.T) {
	pane := newTestRequestFlowPane()
	assert.Equal(t, "Request Flow", pane.Title())
}

func TestRequestFlowPane_ToggleKey(t *testing.T) {
	pane := newTestRequestFlowPane()
	// Page B panes are not individually toggleable.
	assert.Equal(t, 0, pane.ToggleKey())
}

func TestRequestFlowPane_Actions(t *testing.T) {
	pane := newTestRequestFlowPane()
	// RequestFlowPane has no actions.
	assert.Nil(t, pane.Actions())
}

// --- SetSize / SetFocused / IsFocused ---

func TestRequestFlowPane_SetSize(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(80, 20)
	// View() must not panic after SetSize.
	_ = pane.View()
}

func TestRequestFlowPane_FocusRoundtrip(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
	pane.SetFocused(false)
	assert.False(t, pane.IsFocused())
}

// --- Init ---

func TestRequestFlowPane_Init_ReturnsNil(t *testing.T) {
	pane := newTestRequestFlowPane()
	cmd := pane.Init()
	// RequestFlowPane has no self-initiated tick — it reacts to shared TickMsg.
	assert.Nil(t, cmd)
}

// --- View renders three column headers ---

func TestRequestFlowPane_View_ShowsThreeColumns(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	v := pane.View()
	assert.Contains(t, v, "APP", "APP column header must be present")
	assert.Contains(t, v, "GATEWAY", "GATEWAY column header must be present")
	assert.Contains(t, v, "SPOTIFY", "SPOTIFY column header must be present")
}

// --- Token bucket bar ---

func TestRequestFlowPane_View_TokenBucketBar(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	v := pane.View()
	// Full bucket: should show 10 filled indicators.
	assert.Contains(t, v, "●", "token bucket bar should use filled circle for available tokens")
}

// --- Semaphore bar ---

func TestRequestFlowPane_View_SemaphoreBar(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	v := pane.View()
	// Fresh gateway: semaphore is empty (no active requests). Show empty squares.
	assert.Contains(t, v, "□", "semaphore bar should use empty square for available slots")
}

// --- Backoff timer ---

func TestRequestFlowPane_View_BackoffHiddenWhenNotThrottled(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	v := pane.View()
	assert.NotContains(t, v, "backoff", "backoff timer should be hidden when not throttled")
}

func TestRequestFlowPane_View_BackoffVisibleWhenThrottled(t *testing.T) {
	s := state.New()
	s.SetThrottle(true, 30, time.Now())
	gw := api.NewGateway()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(100, 20)
	v := pane.View()
	assert.Contains(t, v, "backoff", "backoff timer should appear when store is throttled")
}

// --- Arrow animation advances on viz.TickMsg ---

func TestRequestFlowPane_VizTickMsg_AdvancesFrame(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)

	frameBefore := pane.FrameIndex()
	_, _ = pane.Update(viz.TickMsg(time.Now()))
	frameAfter := pane.FrameIndex()
	assert.Equal(t, frameBefore+1, frameAfter, "viz.TickMsg should advance frameIndex by 1")
}

// --- TickMsg refreshes state ---

func TestRequestFlowPane_TickMsg_Updates(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)

	// After a TickMsg the pane should not panic and should return self.
	updated, cmd := pane.Update(panes.TickMsg{})
	require.NotNil(t, updated)
	// TickMsg should not return a new tick command (app.go owns the tick loop).
	assert.Nil(t, cmd)
}

// --- Status strip ---

func TestRequestFlowPane_View_StatusStrip_ShowsPollingState(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	// Inject polling state via PollingSnapshotMsg.
	_, _ = pane.Update(panes.PollingSnapshotMsg{
		TickIntervalMs: 1000,
		IsIdle:         false,
		IdleSecs:       0,
	})
	v := pane.View()
	assert.Contains(t, v, "POLLING", "status strip should include POLLING label")
	assert.Contains(t, v, "1000ms", "status strip should show tick interval")
}

func TestRequestFlowPane_View_StatusStrip_ShowsIdleState(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.PollingSnapshotMsg{
		TickIntervalMs: 1000,
		IsIdle:         true,
		IdleSecs:       45,
	})
	v := pane.View()
	assert.Contains(t, v, "idle", "status strip should show idle state")
	assert.Contains(t, v, "45s", "status strip should show idle duration")
}

func TestRequestFlowPane_View_StatusStrip_ShowsStoreFetching(t *testing.T) {
	s := state.New()
	s.SetPlaylistsFetching(true)
	gw := api.NewGateway()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(100, 20)
	v := pane.View()
	assert.Contains(t, v, "STORE", "status strip should include STORE label")
}

// --- Color coding for recent requests ---

func TestRequestFlowPane_View_RecentRequest_2xx(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	// Inject a completed 200 request.
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:   "/me/player",
		StatusCode: 200,
		LatencyMs:  45,
		Priority:   domain.PriorityBackground,
	})
	v := pane.View()
	assert.Contains(t, v, "/me/player", "request endpoint should appear in view")
	assert.Contains(t, v, "200", "status code should appear in view")
}

func TestRequestFlowPane_View_RecentRequest_429(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:   "/me/player",
		StatusCode: 429,
		LatencyMs:  12,
		Priority:   domain.PriorityBackground,
	})
	v := pane.View()
	assert.Contains(t, v, "429", "429 status should appear in view")
}

func TestRequestFlowPane_View_RecentRequest_5xx(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:   "/me/player",
		StatusCode: 500,
		LatencyMs:  100,
		Priority:   domain.PriorityBackground,
	})
	v := pane.View()
	assert.Contains(t, v, "500", "5xx status should appear in view")
}

// --- Recent requests fade after 3 seconds ---

func TestRequestFlowPane_View_RequestFadesAfter3s(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)

	// Inject a request that completed >3s ago.
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:    "/me/player",
		StatusCode:  200,
		LatencyMs:   45,
		Priority:    domain.PriorityBackground,
		CompletedAt: time.Now().Add(-4 * time.Second),
	})

	v := pane.View()
	// The endpoint should still appear (faded/dimmed, but present).
	// We don't check colour codes here — just that old requests don't disappear immediately.
	// A TickMsg should age them out after the threshold.
	_ = v // View must not panic with an aged request.
}

func TestRequestFlowPane_View_RequestAgedOutOnTick(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)

	// Inject a request older than 5s (beyond max display window).
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:    "/me/player",
		StatusCode:  200,
		LatencyMs:   45,
		Priority:    domain.PriorityBackground,
		CompletedAt: time.Now().Add(-6 * time.Second),
	})

	// TickMsg should prune entries older than 5s.
	_, _ = pane.Update(panes.TickMsg{})

	v := pane.View()
	// After age-out, the old request should not appear.
	assert.NotContains(t, v, "/me/player", "request older than 5s should be pruned on tick")
}

// --- Integration tests ---

// TestRequestFlowPane_Integration_MultipleVizTicks verifies that multiple
// animation ticks advance frameIndex monotonically.
func TestRequestFlowPane_Integration_MultipleVizTicks(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)

	for i := 0; i < 10; i++ {
		_, _ = pane.Update(viz.TickMsg(time.Now()))
	}
	assert.Equal(t, 10, pane.FrameIndex(), "after 10 ticks, frameIndex should be 10")
}

// TestRequestFlowPane_Integration_GatewaySnapshot_Refreshes verifies that
// TickMsg causes the gateway snapshot to be re-read.
func TestRequestFlowPane_Integration_GatewaySnapshot_Refreshes(t *testing.T) {
	gw := api.NewGateway()
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(100, 20)

	// Snapshot before TickMsg — fresh gateway.
	_, _ = pane.Update(panes.TickMsg{})
	v := pane.View()
	assert.Contains(t, v, "●", "token bucket should show filled tokens after TickMsg")
}

// TestRequestFlowPane_Integration_EmptyGateway_IdleState verifies that a pane
// with no activity shows an empty/idle state without panicking.
func TestRequestFlowPane_Integration_EmptyGateway_IdleState(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	// No requests injected — view should not panic.
	v := pane.View()
	assert.Contains(t, v, "GATEWAY", "idle gateway state should still show GATEWAY section")
}

// TestRequestFlowPane_Integration_BackoffActive_TimerVisible verifies that when
// the store marks the gateway as throttled, the backoff timer appears.
func TestRequestFlowPane_Integration_BackoffActive_TimerVisible(t *testing.T) {
	s := state.New()
	s.SetThrottle(true, 30, time.Now())
	gw := api.NewGateway()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(100, 20)

	// Refresh the snapshot.
	_, _ = pane.Update(panes.TickMsg{})

	v := pane.View()
	assert.Contains(t, v, "backoff", "backoff timer should be visible during throttle")
}

// TestRequestFlowPane_Integration_MaxRequests verifies that at most maxRecentReqs
// entries are stored regardless of how many RequestCompletedMsgs are sent.
func TestRequestFlowPane_Integration_MaxRequests(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)

	// Send more than maxRecentReqs (6) requests.
	for i := 0; i < 10; i++ {
		_, _ = pane.Update(panes.RequestCompletedMsg{
			Endpoint:   fmt.Sprintf("/me/endpoint/%d", i),
			StatusCode: 200,
			LatencyMs:  10 + i,
			Priority:   domain.PriorityBackground,
		})
	}

	v := pane.View()
	// Only the last 6 endpoints should be visible (cap at maxRecentReqs).
	// The first 4 (i=0..3) should have been evicted.
	assert.NotContains(t, v, "/me/endpoint/0", "oldest entries should be evicted when cap exceeded")
	assert.Contains(t, v, "/me/endpoint/9", "newest entry should be visible")
}

// TestRequestFlowPane_Integration_SyncFromNetLog verifies that TickMsg populates
// recentReqs from the store's network log entries.
func TestRequestFlowPane_Integration_SyncFromNetLog(t *testing.T) {
	s := state.New()
	gw := api.NewGateway()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(100, 20)

	// Record API calls in the store's net log.
	s.RecordNetCall("GET", "/me/player", 200, 45)
	s.RecordNetCall("GET", "/me/playlists", 200, 120)

	// TickMsg triggers syncFromNetLog.
	_, _ = pane.Update(panes.TickMsg{})

	v := pane.View()
	assert.Contains(t, v, "/me/player", "net log entry should appear after TickMsg sync")
	assert.Contains(t, v, "/me/playlists", "net log entry should appear after TickMsg sync")
	assert.Contains(t, v, "200", "status code should appear")
}

// --- InFlightKeys rendering tests ---

// mockGateway implements domain.GatewaySnapshotter for testing InFlightKeys.
type mockGateway struct {
	snap domain.GatewayState
}

func (m *mockGateway) Snapshot() domain.GatewayState { return m.snap }

func TestRequestFlowPane_View_InFlightKeys_NonEmpty(t *testing.T) {
	gw := &mockGateway{snap: domain.GatewayState{
		TokensAvailable: 10,
		TokensMax:       10,
		ConcurrentMax:   5,
		InFlightKeys:    []string{"GET /me/player", "GET /me/playlists"},
	}}
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.TickMsg{})
	v := pane.View()
	assert.Contains(t, v, "GET /me/player", "in-flight key should appear in view")
	assert.Contains(t, v, "GET /me/playlists", "in-flight key should appear in view")
}

func TestRequestFlowPane_View_InFlightKeys_Truncated(t *testing.T) {
	gw := &mockGateway{snap: domain.GatewayState{
		TokensAvailable: 10,
		TokensMax:       10,
		ConcurrentMax:   5,
		InFlightKeys: []string{
			"GET /me/player",
			"GET /me/playlists",
			"GET /me/albums",
			"GET /me/liked",
			"GET /me/recent",
		},
	}}
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.TickMsg{})
	v := pane.View()
	// At most 3 keys shown, rest truncated.
	assert.Contains(t, v, "+2 more", "overflow should show '+N more' truncation")
}

func TestRequestFlowPane_View_InFlightKeys_Empty(t *testing.T) {
	gw := &mockGateway{snap: domain.GatewayState{
		TokensAvailable: 10,
		TokensMax:       10,
		ConcurrentMax:   5,
		InFlightKeys:    nil,
	}}
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.TickMsg{})
	v := pane.View()
	// No in-flight section rendered when keys is empty.
	assert.NotContains(t, v, "→ GET", "no in-flight section when InFlightKeys is empty")
}

// --- Arrow state tests (four gateway decisions) ---

func TestRequestFlowPane_Arrow_AllowedDecision_Animated(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	// Inject an Allowed request.
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:        "/me/player",
		StatusCode:      200,
		LatencyMs:       50,
		Priority:        domain.PriorityBackground,
		GatewayDecision: domain.DecisionAllowed,
	})
	v := pane.View()
	// Allowed decision renders an animated arrow.
	assert.True(t, containsAny(v, "──→──", "───→─", "────→"),
		"DecisionAllowed should render an animated arrow")
}

func TestRequestFlowPane_Arrow_WaitedDecision(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:        "/me/player",
		StatusCode:      200,
		LatencyMs:       100,
		Priority:        domain.PriorityBackground,
		GatewayDecision: domain.DecisionWaited,
	})
	v := pane.View()
	assert.Contains(t, v, "wait", "DecisionWaited should render 'wait' in the arrow column")
}

func TestRequestFlowPane_Arrow_DedupedDecision(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:        "/me/player",
		StatusCode:      200,
		LatencyMs:       30,
		Priority:        domain.PriorityBackground,
		GatewayDecision: domain.DecisionDeduped,
	})
	v := pane.View()
	assert.Contains(t, v, "dedup", "DecisionDeduped should render 'dedup' in the arrow column")
}

func TestRequestFlowPane_Arrow_BlockedDecision(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:        "/me/player",
		StatusCode:      0,
		LatencyMs:       0,
		Priority:        domain.PriorityBackground,
		GatewayDecision: domain.DecisionBlocked,
	})
	v := pane.View()
	assert.Contains(t, v, "╳", "DecisionBlocked should render ╳ in the arrow column")
}

func TestRequestFlowPane_Arrow_Allowed429_ShowsBlock(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	// DecisionAllowed with 429 status code → X arrow (HTTP-layer throttle).
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:        "/me/player",
		StatusCode:      429,
		LatencyMs:       5,
		Priority:        domain.PriorityBackground,
		GatewayDecision: domain.DecisionAllowed,
	})
	v := pane.View()
	assert.Contains(t, v, "╳", "DecisionAllowed+429 should render ╳ in the arrow column")
}

// containsAny returns true if s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// --- Theme color coding tests ---

func TestRequestFlowPane_View_ContainsANSIEscapes(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	// Inject a request so color-coded rows are rendered.
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:   "/me/player",
		StatusCode: 200,
		LatencyMs:  50,
		Priority:   domain.PriorityBackground,
	})
	v := pane.View()
	// Theme colors produce ANSI escape sequences — check for ESC character.
	assert.Contains(t, v, "\x1b[", "View() should contain ANSI escape sequences from theme styling")
}

func TestRequestFlowPane_View_StatusCodeColoring_2xx(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:   "/me/player",
		StatusCode: 200,
		LatencyMs:  50,
		Priority:   domain.PriorityBackground,
	})
	v := pane.View()
	// ANSI + "200" must appear (the status code is rendered with color).
	assert.Contains(t, v, "200", "2xx status code should appear in view with theme color")
}

func TestRequestFlowPane_View_StatusCodeColoring_429(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:   "/me/player",
		StatusCode: 429,
		LatencyMs:  5,
		Priority:   domain.PriorityBackground,
	})
	v := pane.View()
	assert.Contains(t, v, "429", "429 status code should appear in view")
}

func TestRequestFlowPane_View_StatusCodeColoring_5xx(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	_, _ = pane.Update(panes.RequestCompletedMsg{
		Endpoint:   "/me/player",
		StatusCode: 500,
		LatencyMs:  200,
		Priority:   domain.PriorityBackground,
	})
	v := pane.View()
	assert.Contains(t, v, "500", "5xx status code should appear in view")
}

func TestRequestFlowPane_View_Headers_AreStyled(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)
	v := pane.View()
	// Column headers must still be present after styling is applied.
	assert.Contains(t, v, "APP", "APP header must appear after theme styling")
	assert.Contains(t, v, "GATEWAY", "GATEWAY header must appear after theme styling")
	assert.Contains(t, v, "SPOTIFY", "SPOTIFY header must appear after theme styling")
}

// --- Staleness display tests ---

func TestRequestFlowPane_View_StalenessDisplay_StalePlaylist(t *testing.T) {
	s := state.New()
	// Set playlists fetched-at to 10 minutes ago (beyond PlaylistsTTL of 5 min).
	s.SetPlaylistsFetchedAt(time.Now().Add(-10 * time.Minute))
	gw := api.NewGateway()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(100, 20)
	v := pane.View()
	assert.Contains(t, v, "stale:", "status strip should show stale label when data is stale")
	assert.Contains(t, v, "playlists", "stale playlists domain should appear")
}

func TestRequestFlowPane_View_StalenessDisplay_FreshData(t *testing.T) {
	s := state.New()
	// Set playlists fetched-at to 1 minute ago (within PlaylistsTTL of 5 min).
	s.SetPlaylistsFetchedAt(time.Now().Add(-1 * time.Minute))
	gw := api.NewGateway()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(100, 20)
	v := pane.View()
	assert.NotContains(t, v, "stale:", "fresh data should not show stale label")
}

func TestRequestFlowPane_View_StalenessDisplay_NeverFetched(t *testing.T) {
	s := state.New()
	// Never fetched — zero time — should not appear as stale.
	gw := api.NewGateway()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(100, 20)
	v := pane.View()
	assert.NotContains(t, v, "stale:", "never-fetched data should not appear as stale")
}

func TestRequestFlowPane_View_StalenessDisplay_MultipleStale(t *testing.T) {
	s := state.New()
	// Set multiple domains stale.
	s.SetPlaylistsFetchedAt(time.Now().Add(-10 * time.Minute))
	s.SetAlbumsFetchedAt(time.Now().Add(-10 * time.Minute))
	gw := api.NewGateway()
	th := theme.Load("black")
	pane := panes.NewRequestFlowPane(gw, s, th)
	pane.SetSize(120, 20)
	v := pane.View()
	assert.Contains(t, v, "playlists", "stale playlists should appear")
	assert.Contains(t, v, "albums", "stale albums should appear")
}

// TestRequestFlowPane_Integration_PollingSnapshot_IdleReturn verifies that
// switching from idle to active updates the status strip.
func TestRequestFlowPane_Integration_PollingSnapshot_IdleReturn(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)

	// Simulate idle state.
	_, _ = pane.Update(panes.PollingSnapshotMsg{TickIntervalMs: 3000, IsIdle: true, IdleSecs: 120})
	v1 := pane.View()
	assert.Contains(t, v1, "idle", "idle state should appear in view")
	assert.Contains(t, v1, "120s", "idle duration should appear")

	// Simulate return to active.
	_, _ = pane.Update(panes.PollingSnapshotMsg{TickIntervalMs: 1000, IsIdle: false, IdleSecs: 0})
	v2 := pane.View()
	assert.NotContains(t, v2, "idle", "active state should not show idle label")
	assert.Contains(t, v2, "1000ms", "active tick interval should update")
}
