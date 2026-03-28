package panes

// renderSubBox_test.go — internal tests for the boxed layout helpers.
// These tests are in the `panes` package (not `panes_test`) so they can
// directly call unexported methods on *RequestFlowPane.

import (
	"strings"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

// newInternalTestPane creates a RequestFlowPane for internal helper testing.
func newInternalTestPane() *RequestFlowPane {
	gw := api.NewGateway()
	s := state.New()
	t := theme.Load("black")
	return NewRequestFlowPane(gw, s, t)
}

// --- Task 1: renderSubBox ---

func TestRenderSubBox_ContainsRoundedCorners(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("APP", []string{"line1", "line2"}, 20)
	assert.Contains(t, out, "╭", "top-left rounded corner must be present")
	assert.Contains(t, out, "╮", "top-right rounded corner must be present")
	assert.Contains(t, out, "╰", "bottom-left rounded corner must be present")
	assert.Contains(t, out, "╯", "bottom-right rounded corner must be present")
}

func TestRenderSubBox_ContainsTitle(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("APP", []string{"line1"}, 20)
	assert.Contains(t, out, "APP", "title must appear in top border")
}

func TestRenderSubBox_ContainsSideBorders(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("APP", []string{"line1", "line2"}, 20)
	assert.Contains(t, out, "│", "side border character must be present")
}

func TestRenderSubBox_ContentLinesPresent(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("GW", []string{"hello", "world"}, 20)
	assert.Contains(t, out, "hello", "first content line should appear in box")
	assert.Contains(t, out, "world", "second content line should appear in box")
}

func TestRenderSubBox_LongLineTruncated(t *testing.T) {
	p := newInternalTestPane()
	longLine := strings.Repeat("x", 50)
	out := p.renderSubBox("T", []string{longLine}, 20)
	// Box should not overflow its width — truncation must occur.
	assert.Contains(t, out, "…", "long content lines must be truncated with ellipsis")
}

func TestRenderSubBox_TooNarrowReturnsEmpty(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("APP", []string{"hi"}, 7)
	assert.Empty(t, out, "width < 8 should return empty string")
}

func TestRenderSubBox_EmptyLinesSlice(t *testing.T) {
	p := newInternalTestPane()
	out := p.renderSubBox("APP", []string{}, 20)
	// Box with no content: only top + bottom border rows.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 2, "empty content slice should produce 2-line box (top+bottom border)")
}

// --- Task 2: renderRightArrow ---

func TestRenderRightArrow_2xx_ContainsAnimatedArrow(t *testing.T) {
	p := newInternalTestPane()
	r := reqDisplay{statusCode: 200}
	out := p.renderRightArrow(r, 12)
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
	r := reqDisplay{statusCode: 429}
	out := p.renderRightArrow(r, 12)
	assert.Contains(t, out, "╳", "429 response should render ╳ symbol")
}

func TestRenderRightArrow_500_ContainsArrow(t *testing.T) {
	p := newInternalTestPane()
	r := reqDisplay{statusCode: 500}
	out := p.renderRightArrow(r, 12)
	animatedFrames := []string{"──→──", "───→─", "────→"}
	found := false
	for _, f := range animatedFrames {
		if strings.Contains(out, f) {
			found = true
			break
		}
	}
	assert.True(t, found, "5xx status should render an animated arrow frame (Error color)")
}

func TestRenderRightArrow_StatusZero_ContainsX(t *testing.T) {
	p := newInternalTestPane()
	r := reqDisplay{statusCode: 0}
	out := p.renderRightArrow(r, 12)
	assert.Contains(t, out, "╳", "blocked request (status 0) should render ╳ symbol")
}

// --- Task 3: buildAppBoxLines ---

func TestBuildAppBoxLines_PadsToMaxRows(t *testing.T) {
	p := newInternalTestPane()
	// 2 requests, maxRows=4 → 2 content lines + 2 empty
	_, _ = p.Update(RequestCompletedMsg{
		Endpoint:    "/me/player",
		StatusCode:  200,
		CompletedAt: time.Now(),
	})
	_, _ = p.Update(RequestCompletedMsg{
		Endpoint:    "/me/queue",
		StatusCode:  200,
		CompletedAt: time.Now(),
	})
	lines := p.buildAppBoxLines(4)
	assert.Len(t, lines, 4, "buildAppBoxLines must return exactly maxRows lines")
}

func TestBuildAppBoxLines_CapsAtMaxRows(t *testing.T) {
	p := newInternalTestPane()
	// 6 requests, maxRows=3 → only 3 lines returned
	for i := 0; i < 6; i++ {
		_, _ = p.Update(RequestCompletedMsg{
			Endpoint:    "/ep",
			StatusCode:  200,
			CompletedAt: time.Now(),
		})
	}
	lines := p.buildAppBoxLines(3)
	assert.Len(t, lines, 3, "buildAppBoxLines must cap at maxRows")
}

func TestBuildAppBoxLines_ZeroMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildAppBoxLines(0)
	assert.Len(t, lines, 0, "buildAppBoxLines(0) must return empty slice")
}

// --- Task 3: buildGatewayBoxLines ---

func TestBuildGatewayBoxLines_AlwaysIncludesTokenAndSemaphore(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildGatewayBoxLines(5)
	// Join all lines and check for token/semaphore markers.
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "●", "token bucket line must be included")
	assert.Contains(t, combined, "□", "semaphore line must be included (empty squares = idle)")
}

func TestBuildGatewayBoxLines_ThrottleShowsBackoff(t *testing.T) {
	s := state.New()
	s.SetThrottle(true, 30, time.Now())
	gw := api.NewGateway()
	th := theme.Load("black")
	p := NewRequestFlowPane(gw, s, th)
	lines := p.buildGatewayBoxLines(6)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "backoff", "throttled state must include backoff line")
}

func TestBuildGatewayBoxLines_NoThrottleNoBackoff(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildGatewayBoxLines(5)
	combined := strings.Join(lines, "\n")
	assert.NotContains(t, combined, "backoff", "non-throttled state must not include backoff")
}

func TestBuildGatewayBoxLines_PadsToMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildGatewayBoxLines(8)
	assert.Len(t, lines, 8, "buildGatewayBoxLines must return exactly maxRows lines")
}

func TestBuildGatewayBoxLines_ZeroMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildGatewayBoxLines(0)
	assert.Len(t, lines, 0, "buildGatewayBoxLines(0) must return empty slice")
}

// --- Task 3: buildSpotifyBoxLines ---

func TestBuildSpotifyBoxLines_429ContainsWarning(t *testing.T) {
	p := newInternalTestPane()
	_, _ = p.Update(RequestCompletedMsg{
		Endpoint:    "/me/player",
		StatusCode:  429,
		CompletedAt: time.Now(),
	})
	lines := p.buildSpotifyBoxLines(3)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "⚠", "429 response should include warning suffix")
}

func TestBuildSpotifyBoxLines_StatusZero(t *testing.T) {
	p := newInternalTestPane()
	_, _ = p.Update(RequestCompletedMsg{
		Endpoint:    "/blocked",
		StatusCode:  0,
		CompletedAt: time.Now(),
	})
	lines := p.buildSpotifyBoxLines(3)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "0", "blocked request (status 0) must show status in spotify box")
}

func TestBuildSpotifyBoxLines_PadsToMaxRows(t *testing.T) {
	p := newInternalTestPane()
	// 1 request, maxRows=3 → 1 content + 2 padding
	_, _ = p.Update(RequestCompletedMsg{
		Endpoint:    "/me/player",
		StatusCode:  200,
		CompletedAt: time.Now(),
	})
	lines := p.buildSpotifyBoxLines(3)
	assert.Len(t, lines, 3, "buildSpotifyBoxLines must return exactly maxRows lines")
}

func TestBuildSpotifyBoxLines_ZeroMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildSpotifyBoxLines(0)
	assert.Len(t, lines, 0, "buildSpotifyBoxLines(0) must return empty slice")
}

// --- Task 6: gatewayStateLines ---

func TestGatewayStateLines_ReturnsSlice(t *testing.T) {
	p := newInternalTestPane()
	lines := p.gatewayStateLines()
	// Always has at least token + semaphore lines.
	assert.GreaterOrEqual(t, len(lines), 2, "gatewayStateLines must return at least 2 lines")
}

func TestGatewayStateLines_ThrottledAddsBackoff(t *testing.T) {
	s := state.New()
	s.SetThrottle(true, 30, time.Now())
	gw := api.NewGateway()
	th := theme.Load("black")
	p := NewRequestFlowPane(gw, s, th)
	linesNoThrottle := newInternalTestPane().gatewayStateLines()
	linesThrottled := p.gatewayStateLines()
	assert.Greater(t, len(linesThrottled), len(linesNoThrottle),
		"throttled state must produce more lines than non-throttled")
}

func TestRenderGatewayState_BackwardCompat(t *testing.T) {
	p := newInternalTestPane()
	// renderGatewayState() must still work and produce non-empty output.
	out := p.renderGatewayState()
	assert.Contains(t, out, "●", "renderGatewayState must include token bucket bar")
}

// --- Arrow alignment: buildLeftArrowLines / buildRightArrowLines ---

func TestBuildLeftArrowLines_LengthMatchesMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildLeftArrowLines(4, 12)
	assert.Len(t, lines, 4, "buildLeftArrowLines must return exactly maxRows lines")
}

func TestBuildRightArrowLines_LengthMatchesMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildRightArrowLines(4, 12)
	assert.Len(t, lines, 4, "buildRightArrowLines must return exactly maxRows lines")
}

func TestBuildLeftArrowLines_RequestRowHasArrow(t *testing.T) {
	p := newInternalTestPane()
	_, _ = p.Update(RequestCompletedMsg{
		Endpoint:        "/me/player",
		StatusCode:      200,
		GatewayDecision: domain.DecisionAllowed,
		CompletedAt:     time.Now(),
	})
	lines := p.buildLeftArrowLines(4, 12)
	// First row should contain arrow content (non-blank).
	assert.True(t, strings.TrimSpace(lines[0]) != "" ||
		strings.TrimSpace(lines[1]) != "" ||
		strings.TrimSpace(lines[2]) != "",
		"at least one arrow line must be non-blank with a request injected")
}

func TestBuildRightArrowLines_RequestRowHasArrow(t *testing.T) {
	p := newInternalTestPane()
	_, _ = p.Update(RequestCompletedMsg{
		Endpoint:    "/me/player",
		StatusCode:  200,
		CompletedAt: time.Now(),
	})
	lines := p.buildRightArrowLines(4, 12)
	combined := strings.Join(lines, "")
	assert.True(t, strings.TrimSpace(combined) != "",
		"right arrow lines must be non-blank when a request is present")
}

// --- Task 3: maxRows <= 0 guard for arrow builders ---

// TestBuildLeftArrowLines_ZeroMaxRows verifies that buildLeftArrowLines returns
// nil (not a zero-length slice backed by a non-nil array) when maxRows <= 0.
func TestBuildLeftArrowLines_ZeroMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildLeftArrowLines(0, 10)
	assert.Nil(t, lines, "buildLeftArrowLines(0, w) must return nil")
}

// TestBuildRightArrowLines_ZeroMaxRows verifies that buildRightArrowLines returns
// nil (not a zero-length slice backed by a non-nil array) when maxRows <= 0.
func TestBuildRightArrowLines_ZeroMaxRows(t *testing.T) {
	p := newInternalTestPane()
	lines := p.buildRightArrowLines(0, 10)
	assert.Nil(t, lines, "buildRightArrowLines(0, w) must return nil")
}
