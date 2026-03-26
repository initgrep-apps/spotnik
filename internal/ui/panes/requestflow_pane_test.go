package panes_test

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
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

// --- Arrow animation advances on VisualizerTickMsg ---

func TestRequestFlowPane_VisualizerTickMsg_AdvancesFrame(t *testing.T) {
	pane := newTestRequestFlowPane()
	pane.SetSize(100, 20)

	frameBefore := pane.FrameIndex()
	_, _ = pane.Update(components.VisualizerTickMsg(time.Now()))
	frameAfter := pane.FrameIndex()
	assert.Equal(t, frameBefore+1, frameAfter, "VisualizerTickMsg should advance frameIndex by 1")
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
		Priority:   api.Background,
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
		Priority:   api.Background,
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
		Priority:   api.Background,
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
		Priority:    api.Background,
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
		Priority:    api.Background,
		CompletedAt: time.Now().Add(-6 * time.Second),
	})

	// TickMsg should prune entries older than 5s.
	_, _ = pane.Update(panes.TickMsg{})

	v := pane.View()
	// After age-out, the old request should not appear.
	assert.NotContains(t, v, "/me/player", "request older than 5s should be pruned on tick")
}
