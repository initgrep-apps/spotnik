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

// fixedNetworkLogTime is a deterministic timestamp used for network log
// event snapshots so the rendered "HH:MM:SS" prefixes are stable across runs.
var fixedNetworkLogTime = time.Date(2025, 3, 14, 9, 26, 53, 0, time.UTC)

// TestNetworkLogPane_View_WithEntries verifies the golden snapshot of
// NetworkLogPane with 5 completed requests shown with timestamps, methods,
// endpoints, status codes, latency, priority, and decision columns.
// Timestamps use a fixed UTC time so the rendered "HH:MM:SS" prefixes are
// deterministic.
func TestNetworkLogPane_View_WithEntries(t *testing.T) {
	store := state.New()
	entries := []struct {
		id       uint64
		method   string
		path     string
		status   int
		duration int64
		priority domain.RequestPriority
	}{
		{1, "GET", "/v1/me/player", 200, 120, domain.PriorityInteractive},
		{2, "GET", "/v1/me/playlists", 200, 85, domain.PriorityBackground},
		{3, "PUT", "/v1/me/player/play", 204, 40, domain.PriorityInteractive},
		{4, "GET", "/v1/me/tracks/contains", 200, 60, domain.PriorityBackground},
		{5, "POST", "/v1/me/player/queue", 202, 95, domain.PriorityInteractive},
	}
	for _, e := range entries {
		// Production emission order: EventHttpCompleted first, then
		// EventRequestAllowed. refreshRows backfills the decision.
		store.RecordEvent(domain.GatewayEvent{
			Timestamp:  fixedNetworkLogTime,
			Kind:       domain.EventHttpCompleted,
			RequestID:  e.id,
			Method:     e.method,
			Path:       e.path,
			StatusCode: e.status,
			DurationMs: e.duration,
			Priority:   e.priority,
			Snapshot:   domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
		})
		store.RecordEvent(domain.GatewayEvent{
			Kind:      domain.EventRequestAllowed,
			RequestID: e.id,
			Method:    e.method,
			Path:      e.path,
			Priority:  e.priority,
			Snapshot:  domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
		})
	}

	pane := panes.NewNetworkLogPane(store, theme.Load("black"))
	pane.SetSize(78, 14)
	pane.Update(panes.TickMsg{}) //nolint:errcheck

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestNetworkLogPane_View_Empty verifies the golden snapshot of NetworkLogPane
// when no requests have been recorded yet (only the column header is shown).
func TestNetworkLogPane_View_Empty(t *testing.T) {
	store := state.New()

	pane := panes.NewNetworkLogPane(store, theme.Load("black"))
	pane.SetSize(78, 14)
	pane.Update(panes.TickMsg{}) //nolint:errcheck

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}