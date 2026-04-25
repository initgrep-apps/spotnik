package panes

// requestflow_boxed_test.go — internal tests for the boxed layout helpers.
// These tests are in the `panes` package (not `panes_test`) so they can
// directly call unexported methods on *RequestFlowPane.

import (
	"strings"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components/viz"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newInternalTestPane creates a RequestFlowPane for internal helper testing.
func newInternalTestPane() *RequestFlowPane {
	s := state.New()
	t := theme.Load("black")
	return NewRequestFlowPane(s, t)
}

// newInternalTestPaneWithStore creates a RequestFlowPane sharing the given store.
func newInternalTestPaneWithStore(s *state.Store) *RequestFlowPane {
	return NewRequestFlowPane(s, theme.Load("black"))
}

// injectEventInternal records an event into the store and processes one tick.
func injectEventInternal(p *RequestFlowPane, s *state.Store, event domain.GatewayEvent) {
	s.RecordEvent(event)
	_, _ = p.Update(viz.TickMsg(time.Now()))
}

// --- Task 3: buildAppBoxLines ---

func TestBuildAppBoxLines_PadsToMaxRows(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	// 2 requests, maxRows=4 → 2 content lines + 2 empty
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventRequestEntered, RequestID: 1, Method: "GET", Path: "/me/player"})
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventRequestEntered, RequestID: 2, Method: "GET", Path: "/me/queue"})
	lines := p.buildAppBoxLines(4)
	assert.Len(t, lines, 4)
}

func TestBuildAppBoxLines_CapsAtMaxRows(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	for i := 0; i < 6; i++ {
		injectEventInternal(p, s, domain.GatewayEvent{
			Kind:      domain.EventRequestEntered,
			RequestID: uint64(i + 1),
			Method:    "GET",
			Path:      "/ep",
		})
	}
	lines := p.buildAppBoxLines(3)
	assert.Len(t, lines, 3)
}

func TestBuildAppBoxLines_ZeroMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildAppBoxLines(0)
	assert.Len(t, lines, 0)
}

// --- Task 3: buildGatewayBoxLines ---

func TestBuildGatewayBoxLines_PadsToMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildGatewayBoxLines(8)
	assert.Len(t, lines, 8)
}

func TestBuildGatewayBoxLines_ZeroMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildGatewayBoxLines(0)
	assert.Len(t, lines, 0)
}

func TestBuildGatewayBoxLines_ShowsDecisionLog(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind:      domain.EventRequestAllowed,
		RequestID: 1,
		Method:    "GET",
		Path:      "/me/player",
	})
	lines := p.buildGatewayBoxLines(10)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "allowed", "decision log should include 'allowed' entry")
}

// --- Task 3: buildSpotifyBoxLines ---

func TestBuildSpotifyBoxLines_ZeroMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildSpotifyBoxLines(0)
	assert.Len(t, lines, 0)
}

// --- Task 7: buildSpotifyBoxLines new format ---

func TestBuildSpotifyBoxLines_ShowsStatusMethodPath(t *testing.T) {
	p := newInternalTestPane()
	const reqID uint64 = 1
	if p.displayState.requests == nil {
		p.displayState.requests = make(map[uint64]*requestAnimation)
	}
	p.displayState.requests[reqID] = &requestAnimation{
		requestID:  reqID,
		method:     "GET",
		path:       "/v1/me/player",
		phase:      phaseCompleted,
		decision:   domain.EventRequestAllowed,
		statusCode: 200,
		durationMs: 43,
	}
	lines := p.buildSpotifyBoxLines(10)
	require.NotEmpty(t, lines)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "200")
	assert.Contains(t, combined, "GET")
	assert.Contains(t, combined, "/player") // /v1/me stripped
	assert.Contains(t, combined, "43ms")
}

func TestBuildSpotifyBoxLines_OmitsBlockedRequests(t *testing.T) {
	p := newInternalTestPane()
	const reqID uint64 = 2
	if p.displayState.requests == nil {
		p.displayState.requests = make(map[uint64]*requestAnimation)
	}
	p.displayState.requests[reqID] = &requestAnimation{
		requestID: reqID,
		method:    "PUT",
		path:      "/v1/me/player/volume",
		phase:     phaseCompleted,
		decision:  domain.EventRequestBlocked,
	}
	lines := p.buildSpotifyBoxLines(10)
	combined := strings.Join(lines, "\n")
	assert.NotContains(t, combined, "PUT", "blocked request must not appear in SPOTIFY box")
}

func TestBuildSpotifyBoxLines_OmitsDedupJoined(t *testing.T) {
	p := newInternalTestPane()
	const reqID uint64 = 3
	if p.displayState.requests == nil {
		p.displayState.requests = make(map[uint64]*requestAnimation)
	}
	p.displayState.requests[reqID] = &requestAnimation{
		requestID: reqID,
		method:    "GET",
		path:      "/v1/me/player",
		phase:     phaseCompleted,
		decision:  domain.EventDedupJoined,
	}
	lines := p.buildSpotifyBoxLines(10)
	combined := strings.Join(lines, "\n")
	assert.Empty(t, strings.TrimSpace(combined), "dedup-joined request must not appear in SPOTIFY box")
}

func TestBuildSpotifyBoxLines_InFlightShowsPlaceholder(t *testing.T) {
	p := newInternalTestPane()
	const reqID uint64 = 4
	if p.displayState.requests == nil {
		p.displayState.requests = make(map[uint64]*requestAnimation)
	}
	p.displayState.requests[reqID] = &requestAnimation{
		requestID: reqID,
		method:    "GET",
		path:      "/v1/me/player",
		phase:     phaseInFlight, // in-flight = HTTP call in progress, no response yet
	}
	lines := p.buildSpotifyBoxLines(10)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "···", "in-flight request must show placeholder")
}

// --- Task 6: buildGatewayBoxLines pure event log ---

func TestBuildGatewayBoxLines_NoStateMetrics(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	// Push a decision event so there's something in the log.
	p.displayState.decisions = append(p.displayState.decisions, decisionEntry{
		kind:  domain.EventRequestAllowed,
		label: "✓ GET /player  allowed",
	})
	lines := p.buildGatewayBoxLines(10)
	for _, l := range lines {
		assert.NotContains(t, l, "tokens", "gateway log box must not contain token metric header")
		assert.NotContains(t, l, "concurrent", "gateway log box must not contain semaphore metric header")
	}
}

func TestBuildGatewayBoxLines_ContainsDecisionEntry(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	p.displayState.decisions = append(p.displayState.decisions, decisionEntry{
		kind:  domain.EventRequestAllowed,
		label: "✓ GET /player  allowed",
	})
	lines := p.buildGatewayBoxLines(10)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "allowed")
}

func TestRenderGatewayState_BackwardCompat(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind:     domain.EventTokenConsumed,
		Snapshot: domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
	})
	out := p.renderGatewayState()
	assert.Contains(t, out, "●")
}

// --- Task 5: renderGatewayBanner ---

func TestRenderGatewayBanner_ContainsTokens(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{
		TokensAvailable:  8,
		TokensMax:        10,
		ConcurrentActive: 0,
		ConcurrentMax:    5,
	}
	out := p.renderGatewayBanner(80)
	assert.Contains(t, out, "TOKENS")
	assert.Contains(t, out, "8/10")
}

func TestRenderGatewayBanner_ContainsSlots(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{
		TokensMax: 10, ConcurrentActive: 2, ConcurrentMax: 5,
	}
	out := p.renderGatewayBanner(80)
	assert.Contains(t, out, "SLOTS")
	assert.Contains(t, out, "2/5")
}

func TestRenderGatewayBanner_BackoffNone_WhenNotThrottled(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5}
	out := p.renderGatewayBanner(80)
	assert.Contains(t, out, "BACKOFF")
	assert.Contains(t, out, "none")
}

func TestRenderGatewayBanner_DedupNone_WhenZeroWaiters(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5, DedupWaiters: 0}
	out := p.renderGatewayBanner(80)
	assert.Contains(t, out, "DEDUP")
	assert.Contains(t, out, "none")
}

func TestRenderGatewayBanner_DedupWaiters(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5, DedupWaiters: 3}
	// Use a wide enough box so the dedup segment is not truncated.
	out := p.renderGatewayBanner(200)
	assert.Contains(t, out, "3 waiting")
}

func TestRenderGatewayBanner_HasSectionLabel(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5}
	out := p.renderGatewayBanner(80)
	// renderGatewayBanner now uses SectionLabel: label line + rule line.
	assert.Contains(t, out, "GATEWAY")
	assert.Contains(t, out, "─")
}

// --- Task 8: renderAutoTrafficStrip ---

func TestRenderAutoTrafficStrip_HasSectionLabel(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	out := p.renderAutoTrafficStrip(120)
	// renderAutoTrafficStrip now uses SectionLabel: label line + rule line.
	assert.Contains(t, out, "AUTO-TRAFFIC")
	assert.Contains(t, out, "─")
}

func TestRenderAutoTrafficStrip_ShowsPollingRunning(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	_, _ = p.Update(PollingSnapshotMsg{TickIntervalMs: 1000, IsIdle: false})
	out := p.renderAutoTrafficStrip(120)
	assert.Contains(t, out, "▶")
	assert.Contains(t, out, "1s")
	assert.Contains(t, out, "running")
}

func TestRenderAutoTrafficStrip_ShowsPollingIdle(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	_, _ = p.Update(PollingSnapshotMsg{TickIntervalMs: 3000, IsIdle: true, IdleSecs: 60})
	out := p.renderAutoTrafficStrip(120)
	assert.Contains(t, out, "⏸")
	assert.Contains(t, out, "60s")
}

func TestRenderAutoTrafficStrip_NeverFetchedCache(t *testing.T) {
	s := state.New() // no FetchedAt stamps — all zero
	p := newInternalTestPaneWithStore(s)
	out := p.renderAutoTrafficStrip(120)
	assert.Contains(t, out, "never fetched")
}

func TestRenderAutoTrafficStrip_FreshCache(t *testing.T) {
	s := state.New()
	s.SetPlaylistsFetchedAt(time.Now().Add(-1 * time.Minute))
	p := newInternalTestPaneWithStore(s)
	out := p.renderAutoTrafficStrip(120)
	assert.Contains(t, out, "fresh")
}

func TestRenderAutoTrafficStrip_StaleCache(t *testing.T) {
	s := state.New()
	s.SetPlaylistsFetchedAt(time.Now().Add(-10 * time.Minute))
	p := newInternalTestPaneWithStore(s)
	out := p.renderAutoTrafficStrip(120)
	assert.Contains(t, out, "◬")
	assert.Contains(t, out, "playlists")
}

func TestRenderAutoTrafficStrip_NilStore_NoPanic(t *testing.T) {
	p := NewRequestFlowPane(nil, theme.Load("black"))
	assert.NotPanics(t, func() {
		out := p.renderAutoTrafficStrip(120)
		assert.Contains(t, out, "AUTO-TRAFFIC")
	})
}

// --- Story 159: renderSectionColumn uses accent color ---

// TestRenderSectionColumn_UsesAccentColor verifies that renderSectionColumn() uses
// PaneBorderRequestFlow() (orange accent) rather than TextSecondary() (grey)
// for the PaneChrome border.
//
// The black theme maps:
//
//	PaneBorderRequestFlow() → #ffb86c → ANSI "38;2;255;184;108"
//	TextSecondary()         → #888888 → ANSI "38;2;136;136;136"
func TestRenderSectionColumn_UsesAccentColor(t *testing.T) {
	p := newInternalTestPane()

	// Rendered ANSI RGB codes for the black theme.
	const accentANSI = "38;2;255;184;108" // #ffb86c — PaneBorderRequestFlow
	const greyANSI = "38;2;136;136;136"   // #888888 — TextSecondary

	out := renderSectionColumn("APP", []string{"line1", "line2"}, 30,
		p.theme.PaneBorderRequestFlow(), p.theme)

	// Output must contain the accent color ANSI escape.
	assert.Contains(t, out, accentANSI,
		"renderSectionColumn should use PaneBorderRequestFlow() accent color for label")

	// Output must NOT contain TextSecondary color code anywhere.
	assert.NotContains(t, out, greyANSI,
		"renderSectionColumn should NOT use TextSecondary() for label")
}

// --- Story 170: renderSectionColumn renders bordered boxes ---

// TestRenderSectionColumn_BorderedOutput verifies that renderSectionColumn() now
// produces a fully bordered PaneChrome box (╭ top border, │ side borders, ╰ bottom
// border) instead of the former SectionLabel header-only output.
func TestRenderSectionColumn_BorderedOutput(t *testing.T) {
	p := newInternalTestPane()

	out := renderSectionColumn("APP", []string{"line1", "line2"}, 30,
		p.theme.PaneBorderRequestFlow(), p.theme)

	assert.Contains(t, out, "╭", "top border glyph ╭ must be present")
	assert.Contains(t, out, "│", "side border glyph │ must be present")
	assert.Contains(t, out, "╰", "bottom border glyph ╰ must be present")
	assert.Contains(t, out, "APP", "label must appear in the border title")
}
