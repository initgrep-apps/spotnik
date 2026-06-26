package panes_test

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestPollingTrafficPane_View_Fresh verifies the golden snapshot of
// PollingTrafficPane when the playback poller is running at 1s interval and
// all library caches are fresh (recently fetched, within TTL).
func TestPollingTrafficPane_View_Fresh(t *testing.T) {
	store := state.New()
	// Stamp all caches as freshly fetched so each row renders "fresh".
	now := time.Now()
	store.SetPlaylistsFetchedAt(now)
	store.SetAlbumsFetchedAt(now)
	store.SetLikedTracksFetchedAt(now)
	store.SetRecentPlayedFetchedAt(now)
	store.StampStatsFetchedAt("short_term")

	pane := panes.NewPollingTrafficPane(store, theme.Load("black"))
	pane.SetSize(78, 10)
	// Running playback poller at 1s interval.
	pane.Update(panes.PollingSnapshotMsg{TickIntervalMs: 1000, IsIdle: false}) //nolint:errcheck

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestPollingTrafficPane_View_Stale verifies the golden snapshot of
// PollingTrafficPane when the playback poller is idle and library caches are
// stale, showing the warning glyph and age. Cache ages use a fixed offset
// from now (10 minutes) so the rendered "10m stale" text is stable across
// runs: any sub-minute drift stays within the "10m" minute bucket.
func TestPollingTrafficPane_View_Stale(t *testing.T) {
	store := state.New()
	// Set every cache to 10 minutes ago — past the 5-minute TTL so it is stale,
	// but within the "Xm" age bucket (< 1h) so the warning (not error) glyph
	// renders and the age text "10m" is stable for ~50 seconds of drift.
	staleAt := time.Now().Add(-10 * time.Minute)
	store.SetPlaylistsFetchedAt(staleAt)
	store.SetAlbumsFetchedAt(staleAt)
	store.SetLikedTracksFetchedAt(staleAt)
	store.SetRecentPlayedFetchedAt(staleAt)
	// Stats: no stamp → "never fetched" (deterministic muted text).

	pane := panes.NewPollingTrafficPane(store, theme.Load("black"))
	pane.SetSize(78, 10)
	// Idle playback poller at 10s interval.
	pane.Update(panes.PollingSnapshotMsg{TickIntervalMs: 10000, IsIdle: true, IdleSecs: 120}) //nolint:errcheck

	tm := goldentest.NewPaneTest(t, pane, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}