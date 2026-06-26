package panes_test

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// fixedGatewayTime is a deterministic timestamp used for gateway live event
// snapshots so the rendered "HH:MM:SS" prefixes are stable across runs.
var fixedGatewayTime = time.Date(2025, 3, 14, 9, 26, 53, 0, time.UTC)

// TestGatewayLivePane_View_WithEntries verifies the golden snapshot of
// GatewayLivePane with 10 recent gateway events shown in reverse-chronological
// order (newest first). Timestamps use a fixed UTC time so the rendered
// "HH:MM:SS" prefixes are deterministic.
func TestGatewayLivePane_View_WithEntries(t *testing.T) {
	store := state.New()
	kinds := []domain.EventKind{
		domain.EventRequestAllowed,
		domain.EventRequestBlocked,
		domain.EventTokenConsumed,
		domain.EventTokenRefilled,
		domain.EventSemaphoreAcquired,
		domain.EventSemaphoreReleased,
		domain.EventDedupJoined,
		domain.EventDedupResolved,
		domain.EventBackoffStarted,
		domain.EventHttpCompleted,
	}
	for i, k := range kinds {
		e := domain.GatewayEvent{
			Timestamp:  fixedGatewayTime,
			Kind:       k,
			Method:     "GET",
			Path:       "/v1/me/player",
			Priority:   domain.PriorityBackground,
			StatusCode: 200,
			DurationMs: int64(50 + i),
			Snapshot: domain.GatewayStateSnapshot{
				TokensAvailable:  9 - i,
				TokensMax:        10,
				ConcurrentActive: i % 5,
				ConcurrentMax:    5,
				BackoffRemaining: float64(i),
				DedupWaiters:     i,
			},
		}
		store.RecordEvent(e)
	}

	pane := panes.NewGatewayLivePane(store, theme.Load("black"))
	pane.SetSize(78, 14)
	pane.Update(panes.TickMsg{}) //nolint:errcheck

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestGatewayLivePane_View_Empty verifies the golden snapshot of
// GatewayLivePane when no gateway events have been recorded yet.
func TestGatewayLivePane_View_Empty(t *testing.T) {
	store := state.New()

	pane := panes.NewGatewayLivePane(store, theme.Load("black"))
	pane.SetSize(78, 14)
	pane.Update(panes.TickMsg{}) //nolint:errcheck

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
