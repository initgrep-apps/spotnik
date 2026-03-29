package domain_test

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestEventKind_ZeroValue(t *testing.T) {
	// EventRequestEntered must be iota zero so a zero-value GatewayEvent has meaningful kind.
	assert.Equal(t, domain.EventKind(0), domain.EventRequestEntered)
}

func TestEventKind_Distinct(t *testing.T) {
	kinds := []domain.EventKind{
		domain.EventRequestEntered,
		domain.EventTokenConsumed,
		domain.EventTokenRefilled,
		domain.EventSemaphoreAcquired,
		domain.EventSemaphoreReleased,
		domain.EventBackoffStarted,
		domain.EventBackoffExpired,
		domain.EventRequestAllowed,
		domain.EventRequestWaited,
		domain.EventRequestBlocked,
		domain.EventDedupJoined,
		domain.EventDedupResolved,
		domain.EventHttpCompleted,
	}
	assert.Len(t, kinds, 13, "EventKind enum must have exactly 13 constants")
	seen := make(map[domain.EventKind]bool)
	for _, k := range kinds {
		assert.False(t, seen[k], "EventKind values must be distinct")
		seen[k] = true
	}
}

func TestGatewayStateSnapshot_ZeroValue(t *testing.T) {
	var snap domain.GatewayStateSnapshot
	assert.Equal(t, 0, snap.TokensAvailable)
	assert.Equal(t, 0, snap.TokensMax)
	assert.Equal(t, 0, snap.ConcurrentActive)
	assert.Equal(t, 0, snap.ConcurrentMax)
	assert.Equal(t, 0.0, snap.BackoffRemaining)
	assert.Equal(t, 0, snap.DedupWaiters)
	assert.Nil(t, snap.InFlightKeys)
}

func TestGatewayEvent_ZeroValue(t *testing.T) {
	var ev domain.GatewayEvent
	assert.Equal(t, domain.EventRequestEntered, ev.Kind, "zero-value GatewayEvent must have EventRequestEntered kind")
	assert.Equal(t, uint64(0), ev.RequestID)
}

func TestGatewayEvent_RoundTrip(t *testing.T) {
	now := time.Now()
	snap := domain.GatewayStateSnapshot{
		TokensAvailable:  7,
		TokensMax:        10,
		ConcurrentActive: 3,
		ConcurrentMax:    5,
		BackoffRemaining: 1.5,
		DedupWaiters:     2,
		InFlightKeys:     []string{"/me/player", "/me/tracks"},
	}
	ev := domain.GatewayEvent{
		Timestamp:  now,
		Kind:       domain.EventHttpCompleted,
		RequestID:  42,
		Method:     "GET",
		Path:       "/me/player",
		Priority:   domain.PriorityInteractive,
		StatusCode: 200,
		DurationMs: 150,
		Snapshot:   snap,
	}

	assert.Equal(t, now, ev.Timestamp)
	assert.Equal(t, domain.EventHttpCompleted, ev.Kind)
	assert.Equal(t, uint64(42), ev.RequestID)
	assert.Equal(t, "GET", ev.Method)
	assert.Equal(t, "/me/player", ev.Path)
	assert.Equal(t, domain.PriorityInteractive, ev.Priority)
	assert.Equal(t, 200, ev.StatusCode)
	assert.Equal(t, int64(150), ev.DurationMs)
	assert.Equal(t, snap, ev.Snapshot)
}
