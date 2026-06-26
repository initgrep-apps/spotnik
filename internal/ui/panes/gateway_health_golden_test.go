package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestGatewayHealthPane_View_AllHealthy verifies the golden snapshot of
// GatewayHealthPane when all 4 health rows are green/healthy: full token
// bucket (10/10), slots below capacity (0/5), no backoff, no dedup waiters.
func TestGatewayHealthPane_View_AllHealthy(t *testing.T) {
	store := state.New()
	store.RecordEvent(domain.GatewayEvent{
		Kind: domain.EventTokenRefilled,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable:  10,
			TokensMax:        10,
			ConcurrentActive: 0,
			ConcurrentMax:    5,
			BackoffRemaining: 0,
			DedupWaiters:     0,
		},
	})

	pane := panes.NewGatewayHealthPane(store, theme.Load("black"))
	pane.SetSize(78, 10)
	pane.Update(panes.TickMsg{}) //nolint:errcheck

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestGatewayHealthPane_View_MixedHealth verifies the golden snapshot of
// GatewayHealthPane with mixed health states: tokens near empty (warning),
// slots at capacity (warning), active backoff (error), and dedup waiters.
func TestGatewayHealthPane_View_MixedHealth(t *testing.T) {
	store := state.New()
	store.RecordEvent(domain.GatewayEvent{
		Kind: domain.EventBackoffStarted,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable:  2,
			TokensMax:        10,
			ConcurrentActive: 5,
			ConcurrentMax:    5,
			BackoffRemaining: 3.5,
			DedupWaiters:     4,
		},
	})

	pane := panes.NewGatewayHealthPane(store, theme.Load("black"))
	pane.SetSize(78, 10)
	pane.Update(panes.TickMsg{}) //nolint:errcheck

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}