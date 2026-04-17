package panes

import (
	"fmt"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/stretchr/testify/assert"
)

// --- Task 1: replayDisplayState zero value and animationPhase ordering ---

func TestReplayDisplayState_ZeroValue(t *testing.T) {
	var ds replayDisplayState
	assert.Nil(t, ds.requests, "zero-value replayDisplayState should have nil requests map")
	assert.Empty(t, ds.decisions, "zero-value replayDisplayState should have empty decisions")
}

func TestAnimationPhase_ConstantOrdering(t *testing.T) {
	// Phase constants must be in ascending order so phase comparisons work.
	assert.True(t, phaseEntered < phaseAtGateway, "phaseEntered must be < phaseAtGateway")
	assert.True(t, phaseAtGateway < phaseInFlight, "phaseAtGateway must be < phaseInFlight")
	assert.True(t, phaseInFlight < phaseCompleted, "phaseInFlight must be < phaseCompleted")
	assert.True(t, phaseCompleted < phaseDone, "phaseCompleted must be < phaseDone")
}

// --- Task 2: stripAPIPrefix and formatDecisionLabel ---

func TestStripAPIPrefix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/v1/me/player", "/player"},
		{"/v1/me/player/volume", "/player/volume"},
		{"/v1/me/playlists", "/playlists"},
		{"/other/path", "/other/path"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, stripAPIPrefix(tt.in))
		})
	}
}

func TestFormatDecisionLabel_EventRequestEntered_Background(t *testing.T) {
	e := domain.GatewayEvent{
		Kind:     domain.EventRequestEntered,
		Method:   "GET",
		Path:     "/v1/me/player",
		Priority: domain.PriorityBackground,
	}
	assert.Equal(t, "◷ GET /player", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventRequestEntered_Interactive(t *testing.T) {
	e := domain.GatewayEvent{
		Kind:     domain.EventRequestEntered,
		Method:   "PUT",
		Path:     "/v1/me/player/volume",
		Priority: domain.PriorityInteractive,
	}
	assert.Equal(t, "⚡ PUT /player/volume", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventRequestAllowed(t *testing.T) {
	e := domain.GatewayEvent{Kind: domain.EventRequestAllowed, Method: "GET", Path: "/v1/me/player"}
	assert.Equal(t, "✓ GET /player  allowed", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventRequestBlocked(t *testing.T) {
	e := domain.GatewayEvent{Kind: domain.EventRequestBlocked, Method: "PUT", Path: "/v1/me/player/volume"}
	assert.Equal(t, "✗ PUT /player/volume  blocked", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventDedupJoined(t *testing.T) {
	e := domain.GatewayEvent{Kind: domain.EventDedupJoined, Method: "GET", Path: "/v1/me/player"}
	assert.Equal(t, "⧖ GET /player  dedup joined", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventHttpCompleted(t *testing.T) {
	e := domain.GatewayEvent{Kind: domain.EventHttpCompleted, StatusCode: 200, DurationMs: 43}
	assert.Equal(t, "✓ 200  43ms", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventBackoffStarted(t *testing.T) {
	e := domain.GatewayEvent{
		Kind:     domain.EventBackoffStarted,
		Snapshot: domain.GatewayStateSnapshot{BackoffRemaining: 10.0},
	}
	assert.Equal(t, "⏳ backoff started  10s", formatDecisionLabel(e))
}

// --- Task 3: humanInterval and humanAge helpers ---

func TestHumanInterval(t *testing.T) {
	tests := []struct {
		ms   int
		want string
	}{
		{1000, "1s"},
		{3000, "3s"},
		{500, "500ms"},
		{999, "999ms"},
		{0, "?"},
		{-1, "?"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%dms", tt.ms), func(t *testing.T) {
			assert.Equal(t, tt.want, humanInterval(tt.ms))
		})
	}
}

func TestHumanAge_JustNow(t *testing.T) {
	fa := time.Now().Add(-30 * time.Second)
	assert.Equal(t, "just now", humanAge(fa))
}

func TestHumanAge_Minutes(t *testing.T) {
	fa := time.Now().Add(-21 * time.Minute)
	assert.Equal(t, "21m ago", humanAge(fa))
}

func TestHumanAge_Hours(t *testing.T) {
	fa := time.Now().Add(-2 * time.Hour)
	assert.Equal(t, "2h ago", humanAge(fa))
}

func TestHumanAge_HoursAndMinutes(t *testing.T) {
	fa := time.Now().Add(-(1*time.Hour + 2*time.Minute))
	assert.Equal(t, "1h 2m ago", humanAge(fa))
}
