package panes

// renderSubBox_test.go — internal tests for the boxed layout helpers.
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

// --- Task 1: renderSubBox ---

func TestRenderSubBox_ContainsRoundedCorners(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("APP", []string{"line1", "line2"}, 20)
	assert.Contains(t, out, "╭")
	assert.Contains(t, out, "╮")
	assert.Contains(t, out, "╰")
	assert.Contains(t, out, "╯")
}

func TestRenderSubBox_ContainsTitle(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("APP", []string{"line1"}, 20)
	assert.Contains(t, out, "APP")
}

func TestRenderSubBox_ContainsSideBorders(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("APP", []string{"line1", "line2"}, 20)
	assert.Contains(t, out, "│")
}

func TestRenderSubBox_ContentLinesPresent(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("GW", []string{"hello", "world"}, 20)
	assert.Contains(t, out, "hello")
	assert.Contains(t, out, "world")
}

func TestRenderSubBox_LongLineTruncated(t *testing.T) {
	p := newInternalTestPane()
	longLine := strings.Repeat("x", 50)
	out := p.renderSubBox("T", []string{longLine}, 20)
	assert.Contains(t, out, "…")
}

func TestRenderSubBox_TooNarrowReturnsEmpty(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("APP", []string{"hi"}, 7)
	assert.Empty(t, out)
}

func TestRenderSubBox_EmptyLinesSlice(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("APP", []string{}, 20)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 2, "empty content slice should produce 2-line box")
}

// --- Task 2: renderRightArrow ---

func TestRenderRightArrow_2xx_ContainsAnimatedArrow(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)

	const reqID uint64 = 1
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventRequestEntered, RequestID: reqID, Method: "GET", Path: "/ep"})
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventHttpCompleted, RequestID: reqID, Method: "GET", Path: "/ep", StatusCode: 200, DurationMs: 30})

	anim := &requestAnimation{statusCode: 200}
	out := p.renderRightArrow(anim, 12)
	animatedFrames := []string{"──→──", "───→─", "────→"}
	found := false
	for _, f := range animatedFrames {
		if strings.Contains(out, f) {
			found = true
			break
		}
	}
	assert.True(t, found, "2xx status should render an animated arrow frame")
}

func TestRenderRightArrow_429_ContainsX(t *testing.T) {
	p := newInternalTestPane()
	anim := &requestAnimation{statusCode: 429}
	out := p.renderRightArrow(anim, 12)
	assert.Contains(t, out, "╳")
}

func TestRenderRightArrow_500_ContainsArrow(t *testing.T) {
	p := newInternalTestPane()
	anim := &requestAnimation{statusCode: 500}
	out := p.renderRightArrow(anim, 12)
	animatedFrames := []string{"──→──", "───→─", "────→"}
	found := false
	for _, f := range animatedFrames {
		if strings.Contains(out, f) {
			found = true
			break
		}
	}
	assert.True(t, found, "5xx status should render an animated arrow frame")
}

func TestRenderRightArrow_StatusZero_ContainsX(t *testing.T) {
	p := newInternalTestPane()
	anim := &requestAnimation{statusCode: 0}
	out := p.renderRightArrow(anim, 12)
	assert.Contains(t, out, "╳")
}

func TestRenderRightArrow_Blocked_ContainsX(t *testing.T) {
	p := newInternalTestPane()
	anim := &requestAnimation{decision: domain.EventRequestBlocked, statusCode: 0}
	out := p.renderRightArrow(anim, 12)
	assert.Contains(t, out, "╳", "blocked request should render ╳ symbol")
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

func TestBuildGatewayBoxLines_AlwaysIncludesTokenAndSemaphore(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	// Inject snapshot with full bucket.
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventSemaphoreReleased,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable:  10,
			TokensMax:        10,
			ConcurrentActive: 0,
			ConcurrentMax:    5,
		},
	})
	lines := p.buildGatewayBoxLines(5)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "●")
	assert.Contains(t, combined, "□")
}

func TestBuildGatewayBoxLines_ThrottleShowsBackoff(t *testing.T) {
	s := state.New()
	s.SetThrottle(true, 30, time.Now())
	p := newInternalTestPaneWithStore(s)
	lines := p.buildGatewayBoxLines(6)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "backoff")
}

func TestBuildGatewayBoxLines_NoThrottleNoBackoff(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildGatewayBoxLines(5)
	combined := strings.Join(lines, "\n")
	assert.NotContains(t, combined, "backoff")
}

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

func TestBuildSpotifyBoxLines_429ContainsWarning(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	const reqID uint64 = 5
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventRequestEntered, RequestID: reqID, Method: "GET", Path: "/me/player"})
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventHttpCompleted, RequestID: reqID, Method: "GET", Path: "/me/player", StatusCode: 429, DurationMs: 5})
	lines := p.buildSpotifyBoxLines(3)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "⚠")
}

func TestBuildSpotifyBoxLines_BeforeHttpCompleted_EmptyEntry(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	// Only RequestEntered — no HTTP response yet.
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventRequestEntered, RequestID: 1, Method: "GET", Path: "/ep"})
	lines := p.buildSpotifyBoxLines(3)
	// The request is in phaseEntered, so SPOTIFY entry should be empty.
	assert.Len(t, lines, 3)
}

func TestBuildSpotifyBoxLines_PadsToMaxRows(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	const reqID uint64 = 10
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventRequestEntered, RequestID: reqID, Method: "GET", Path: "/ep"})
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventHttpCompleted, RequestID: reqID, Method: "GET", Path: "/ep", StatusCode: 200, DurationMs: 30})
	lines := p.buildSpotifyBoxLines(3)
	assert.Len(t, lines, 3)
}

func TestBuildSpotifyBoxLines_ZeroMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildSpotifyBoxLines(0)
	assert.Len(t, lines, 0)
}

// --- Task 6: gatewayStateLines ---

func TestGatewayStateLines_ReturnsSlice(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventTokenConsumed,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable: 10,
			TokensMax:       10,
			ConcurrentMax:   5,
		},
	})
	lines := p.gatewayStateLines()
	assert.GreaterOrEqual(t, len(lines), 2)
}

func TestGatewayStateLines_ThrottledAddsBackoff(t *testing.T) {
	s1 := state.New()
	p1 := newInternalTestPaneWithStore(s1)
	linesNoThrottle := p1.gatewayStateLines()

	s2 := state.New()
	s2.SetThrottle(true, 30, time.Now())
	p2 := newInternalTestPaneWithStore(s2)
	linesThrottled := p2.gatewayStateLines()

	assert.Greater(t, len(linesThrottled), len(linesNoThrottle))
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

// --- Arrow alignment: buildLeftArrowLines / buildRightArrowLines ---

func TestBuildLeftArrowLines_LengthMatchesMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildLeftArrowLines(4, 12)
	assert.Len(t, lines, 4)
}

func TestBuildRightArrowLines_LengthMatchesMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildRightArrowLines(4, 12)
	assert.Len(t, lines, 4)
}

func TestBuildLeftArrowLines_RequestRowHasArrow(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind:      domain.EventRequestEntered,
		RequestID: 1,
		Method:    "GET",
		Path:      "/me/player",
	})
	lines := p.buildLeftArrowLines(4, 12)
	combined := strings.Join(lines, "")
	assert.True(t, strings.TrimSpace(combined) != "",
		"at least one arrow line must be non-blank with a request injected")
}

func TestBuildRightArrowLines_RequestRowHasArrow(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	const reqID uint64 = 20
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventRequestEntered, RequestID: reqID, Method: "GET", Path: "/ep"})
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventHttpCompleted, RequestID: reqID, Method: "GET", Path: "/ep", StatusCode: 200, DurationMs: 20})
	lines := p.buildRightArrowLines(4, 12)
	combined := strings.Join(lines, "")
	assert.True(t, strings.TrimSpace(combined) != "", "right arrow lines must be non-blank with a request present")
}

func TestBuildLeftArrowLines_ZeroMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildLeftArrowLines(0, 10)
	assert.Nil(t, lines)
}

func TestBuildRightArrowLines_ZeroMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildRightArrowLines(0, 10)
	assert.Nil(t, lines)
}

// --- Decision log: EventDedupJoined arrow labels ---

func TestBuildLeftArrowLines_DedupDecision(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	const reqID uint64 = 31
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventRequestEntered, RequestID: reqID, Method: "GET", Path: "/ep"})
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventDedupJoined, RequestID: reqID, Method: "GET", Path: "/ep"})

	lines := p.buildLeftArrowLines(4, 12)
	combined := strings.Join(lines, "")
	assert.Contains(t, combined, "dedup", "EventDedupJoined should render 'dedup' in left arrow")
}

func TestBuildLeftArrowLines_BlockedDecision(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	const reqID uint64 = 32
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventRequestEntered, RequestID: reqID, Method: "GET", Path: "/ep"})
	injectEventInternal(p, s, domain.GatewayEvent{Kind: domain.EventRequestBlocked, RequestID: reqID, Method: "GET", Path: "/ep"})

	lines := p.buildLeftArrowLines(4, 12)
	combined := strings.Join(lines, "")
	assert.Contains(t, combined, "╳", "EventRequestBlocked should render ╳ in left arrow")
}

// --- Story 74 Task 1: renderSubBox uses accent color ---

// TestRenderSubBox_UsesAccentColor verifies that renderSubBox() uses
// PaneBorderRequestFlow() (orange accent) rather than TextSecondary() (grey)
// for border color and title styling.
//
// The black theme maps:
//
//	PaneBorderRequestFlow() → #ffb86c → ANSI "38;2;255;184;108"
//	TextSecondary()         → #888888 → ANSI "38;2;136;136;136"
func TestRenderSubBox_UsesAccentColor(t *testing.T) {
	p := newInternalTestPane()

	// Rendered ANSI RGB codes for the black theme.
	const accentANSI = "38;2;255;184;108" // #ffb86c — PaneBorderRequestFlow
	const greyANSI = "38;2;136;136;136"   // #888888 — TextSecondary

	out := p.renderSubBox("APP", []string{"line1", "line2"}, 30)

	// Output must contain the accent color ANSI escape.
	assert.Contains(t, out, accentANSI,
		"renderSubBox should use PaneBorderRequestFlow() accent color for borders/title")

	// Output must NOT contain TextSecondary color code anywhere in the box.
	assert.NotContains(t, out, greyANSI,
		"renderSubBox should NOT use TextSecondary() for borders/title")
}
