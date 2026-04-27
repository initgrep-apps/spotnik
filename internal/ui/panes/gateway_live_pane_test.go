package panes_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// newTestGatewayLivePane creates a GatewayLivePane using a real store and the
// black theme. Use SetSize to enable non-empty View() output.
func newTestGatewayLivePane(t *testing.T) (*panes.GatewayLivePane, *state.Store) {
	t.Helper()
	store := state.New()
	th := theme.Load("black")
	p := panes.NewGatewayLivePane(store, th)
	return p, store
}

// makeGatewayLiveEvent creates a minimal GatewayEvent for test use.
func makeGatewayLiveEvent(kind domain.EventKind) domain.GatewayEvent {
	return domain.GatewayEvent{
		Timestamp: time.Now(),
		Kind:      kind,
		Method:    "GET",
		Path:      "/v1/me/player",
		Priority:  domain.PriorityBackground,
		Snapshot:  domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
	}
}

// TestGatewayLivePane_ImplementsLayoutPane verifies the compile-time interface contract.
func TestGatewayLivePane_ImplementsLayoutPane(t *testing.T) {
	p, _ := newTestGatewayLivePane(t)
	var _ layout.Pane = p
	var _ layout.FilterablePane = p
}

// TestGatewayLivePane_ID verifies the pane returns PaneGatewayLive.
func TestGatewayLivePane_ID(t *testing.T) {
	p, _ := newTestGatewayLivePane(t)
	if got := p.ID(); got != layout.PaneGatewayLive {
		t.Errorf("ID() = %d, want %d (PaneGatewayLive)", got, layout.PaneGatewayLive)
	}
}

// TestGatewayLivePane_Title verifies the display title.
func TestGatewayLivePane_Title(t *testing.T) {
	p, _ := newTestGatewayLivePane(t)
	if got := p.Title(); got != "Gateway Live" {
		t.Errorf("Title() = %q, want %q", got, "Gateway Live")
	}
}

// TestGatewayLivePane_ToggleKey verifies the toggle key is 4.
func TestGatewayLivePane_ToggleKey(t *testing.T) {
	p, _ := newTestGatewayLivePane(t)
	if got := p.ToggleKey(); got != 4 {
		t.Errorf("ToggleKey() = %d, want 4", got)
	}
}

// TestGatewayLivePane_View_EmptyBeforeResize verifies that View() returns "" when
// dimensions are zero (before SetSize is called).
func TestGatewayLivePane_View_EmptyBeforeResize(t *testing.T) {
	p, _ := newTestGatewayLivePane(t)
	if got := p.View(); got != "" {
		t.Errorf("View() before SetSize = %q, want empty string", got)
	}
}

// TestGatewayLivePane_Update_DrainsCursorOnTick verifies that a TickMsg causes the
// pane to read new events from the store, advance the event cursor, and that a
// second tick with no new events does not duplicate the existing buffer entry.
func TestGatewayLivePane_Update_DrainsCursorOnTick(t *testing.T) {
	p, store := newTestGatewayLivePane(t)
	p.SetSize(80, 20)

	store.RecordEvent(makeGatewayLiveEvent(domain.EventRequestAllowed))

	p.Update(panes.TickMsg{})

	if got := p.BufferedEventCount(); got != 1 {
		t.Errorf("BufferedEventCount() = %d after first tick, want 1", got)
	}

	// Second tick with no new events — cursor was advanced, nothing new to drain.
	p.Update(panes.TickMsg{})

	if got := p.BufferedEventCount(); got != 1 {
		t.Errorf("BufferedEventCount() = %d after second tick (no new events), want 1 (cursor not re-read)", got)
	}
}

// TestGatewayLivePane_Buffer_CapsAt500 verifies that the buffer never grows beyond
// 500 entries even when more events are recorded, and that the OLDEST events are
// evicted (newest events are retained at the top of the view).
func TestGatewayLivePane_Buffer_CapsAt500(t *testing.T) {
	p, store := newTestGatewayLivePane(t)
	p.SetSize(80, 20)

	// Emit 510 events in a single tick to keep ordering unambiguous.
	// Events use zero-padded 4-digit IDs (e.g. "0000", "0001", …, "0509") so that
	// no ID is a prefix of another — avoiding false-positive substring matches.
	for i := 0; i < 510; i++ {
		store.RecordEvent(domain.GatewayEvent{
			Timestamp: time.Now(),
			Kind:      domain.EventRequestAllowed,
			Method:    "GET",
			Path:      fmt.Sprintf("/v1/me/e%04d", i),
			Snapshot:  domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
		})
	}
	p.Update(panes.TickMsg{})

	if got := p.BufferedEventCount(); got > 500 {
		t.Errorf("BufferedEventCount() = %d, want <= 500 (capped)", got)
	}

	// The oldest events (e0000 through e0009) must have been evicted.
	// The newest event (e0509) must still be present in the view.
	view := p.View()
	for i := 0; i < 10; i++ {
		tag := fmt.Sprintf("e%04d", i)
		if strings.Contains(view, tag) {
			t.Errorf("oldest event tag %q still visible in View() — eviction direction wrong", tag)
		}
	}
	newestTag := "e0509"
	if !strings.Contains(view, newestTag) {
		t.Errorf("newest event tag %q missing from View() — buffer must retain newest events", newestTag)
	}
}

// TestGatewayLivePane_Esc_ResetsScrollWhenFilterInactive verifies Esc mode 3:
// when the filter is not open and no committed query is set, Esc resets scroll
// to the first page. Mode 1 (cancel active filter) and Mode 2 (clear committed
// query) are covered by TestGatewayLivePane_CommittedFilter_ClearedByEsc and
// TestGatewayLivePane_HasActiveFilter.
func TestGatewayLivePane_Esc_ResetsScrollWhenFilterInactive(t *testing.T) {
	p, store := newTestGatewayLivePane(t)
	p.SetSize(80, 20)

	// Fill the buffer with enough events to trigger pagination (>pageSize rows).
	for i := 0; i < 60; i++ {
		store.RecordEvent(domain.GatewayEvent{
			Timestamp: time.Now(),
			Kind:      domain.EventRequestAllowed,
			Method:    "GET",
			Path:      fmt.Sprintf("/v1/me/player/%d", i),
			Snapshot:  domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
		})
	}
	p.Update(panes.TickMsg{})

	// Navigate forward using 'j' to advance page.
	for i := 0; i < 8; i++ {
		p.SetFocused(true)
		p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}

	// Esc with filter inactive and no committed query → should reset scroll.
	p.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if got := p.TableCurrentPage(); got != 1 {
		t.Errorf("TableCurrentPage() = %d after Esc, want 1 (scroll reset)", got)
	}
}

// TestGatewayLivePane_HasActiveFilter verifies filter activation via 'f' key.
func TestGatewayLivePane_HasActiveFilter(t *testing.T) {
	p, _ := newTestGatewayLivePane(t)
	p.SetSize(80, 20)
	p.SetFocused(true)

	if p.HasActiveFilter() {
		t.Fatal("HasActiveFilter() = true before 'f' key, want false")
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})

	if !p.HasActiveFilter() {
		t.Error("HasActiveFilter() = false after 'f' key, want true")
	}
}

// TestGatewayLivePane_BackoffExpired_Skipped verifies that EventBackoffExpired events
// are silently skipped and not added to the buffer.
func TestGatewayLivePane_BackoffExpired_Skipped(t *testing.T) {
	p, store := newTestGatewayLivePane(t)
	p.SetSize(80, 20)

	store.RecordEvent(makeGatewayLiveEvent(domain.EventBackoffExpired))
	p.Update(panes.TickMsg{})

	if got := p.BufferedEventCount(); got != 0 {
		t.Errorf("BufferedEventCount() = %d after EventBackoffExpired, want 0 (skipped)", got)
	}
}

// TestGatewayLivePane_CommittedFilter_ClearedByEsc verifies the second mode of the
// Esc state machine: Esc when filter is inactive but committed query is set clears
// the committed query without resetting scroll. The filter uses discriminating paths
// so that filtering actually narrows the result set.
func TestGatewayLivePane_CommittedFilter_ClearedByEsc(t *testing.T) {
	p, store := newTestGatewayLivePane(t)
	p.SetSize(80, 20)
	p.SetFocused(true)

	// Record events with two distinct path segments so filtering is meaningful.
	// Half match "player", half match "tracks".
	for i := 0; i < 3; i++ {
		store.RecordEvent(domain.GatewayEvent{
			Timestamp: time.Now(),
			Kind:      domain.EventRequestAllowed,
			Method:    "GET",
			Path:      fmt.Sprintf("/v1/me/player/%d", i),
			Snapshot:  domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
		})
		store.RecordEvent(domain.GatewayEvent{
			Timestamp: time.Now(),
			Kind:      domain.EventRequestAllowed,
			Method:    "GET",
			Path:      fmt.Sprintf("/v1/me/tracks/%d", i),
			Snapshot:  domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
		})
	}
	p.Update(panes.TickMsg{})

	// Activate filter, type "player", commit with Enter.
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	for _, ch := range "player" {
		p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Filter should now be inactive (committed).
	if p.HasActiveFilter() {
		t.Fatal("filter still active after Enter, expected committed (inactive)")
	}

	// ActiveFilterQuery must reflect the committed term.
	if got := p.ActiveFilterQuery(); got != "player" {
		t.Errorf("ActiveFilterQuery() = %q after Enter, want %q", got, "player")
	}

	// With active query "player", "player" rows are visible and "tracks" rows are hidden.
	view := p.View()
	if !strings.Contains(view, "player") {
		t.Errorf("View() after committing filter 'player' does not contain 'player' rows; view = %q", view)
	}
	if strings.Contains(view, "tracks") {
		t.Errorf("View() after committing filter 'player' still shows 'tracks' rows; view = %q", view)
	}

	// Esc should clear committed filter (not reset scroll).
	pageBefore := p.TableCurrentPage()
	p.Update(tea.KeyMsg{Type: tea.KeyEscape})

	// After clearing committed filter, ActiveFilterQuery must be empty.
	if got := p.ActiveFilterQuery(); got != "" {
		t.Errorf("ActiveFilterQuery() = %q after Esc (clear committed), want empty", got)
	}
	if p.HasActiveFilter() {
		t.Error("HasActiveFilter() = true after Esc (clear committed), want false")
	}
	if got := p.TableCurrentPage(); got != pageBefore {
		t.Errorf("TableCurrentPage() changed from %d to %d after Esc (clear committed), want unchanged", pageBefore, got)
	}

	// Both path segments must be visible again after filter cleared.
	view = p.View()
	if !strings.Contains(view, "player") {
		t.Errorf("View() after clearing filter does not contain 'player' rows")
	}
	if !strings.Contains(view, "tracks") {
		t.Errorf("View() after clearing filter does not contain 'tracks' rows")
	}
}

// TestGatewayLivePane_EnterOnEmptyPreservesPriorQuery verifies that pressing Enter
// on an empty filter input does NOT wipe a previously committed query.
func TestGatewayLivePane_EnterOnEmptyPreservesPriorQuery(t *testing.T) {
	p, store := newTestGatewayLivePane(t)
	p.SetSize(80, 20)
	p.SetFocused(true)

	// Seed events so there is something to filter.
	store.RecordEvent(domain.GatewayEvent{
		Timestamp: time.Now(),
		Kind:      domain.EventRequestAllowed,
		Method:    "GET",
		Path:      "/v1/me/player/x",
		Snapshot:  domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
	})
	p.Update(panes.TickMsg{})

	// Commit a query "player".
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	for _, ch := range "player" {
		p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if got := p.ActiveFilterQuery(); got != "player" {
		t.Fatalf("ActiveFilterQuery() = %q after first Enter, want %q", got, "player")
	}

	// Open filter again but press Enter immediately without typing.
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	p.Update(tea.KeyMsg{Type: tea.KeyEnter}) // empty input — must not wipe prior query

	if got := p.ActiveFilterQuery(); got != "player" {
		t.Errorf("ActiveFilterQuery() = %q after Enter on empty input, want %q (prior query must be preserved)", got, "player")
	}
}

// TestGatewayLivePane_EscWhileTypingCancels verifies that pressing Esc while the
// filter is open and a draft has been typed cancels without committing anything.
func TestGatewayLivePane_EscWhileTypingCancels(t *testing.T) {
	p, store := newTestGatewayLivePane(t)
	p.SetSize(80, 20)
	p.SetFocused(true)

	// Seed a couple of events.
	for i := 0; i < 3; i++ {
		store.RecordEvent(domain.GatewayEvent{
			Timestamp: time.Now(),
			Kind:      domain.EventRequestAllowed,
			Method:    "GET",
			Path:      fmt.Sprintf("/v1/me/player/%d", i),
			Snapshot:  domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
		})
	}
	p.Update(panes.TickMsg{})

	// Open filter and type a partial query, then press Esc.
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	p.Update(tea.KeyMsg{Type: tea.KeyEscape})

	// Filter must be closed and no committed query must be set.
	if p.HasActiveFilter() {
		t.Error("HasActiveFilter() = true after Esc while typing, want false")
	}
	if got := p.ActiveFilterQuery(); got != "" {
		t.Errorf("ActiveFilterQuery() = %q after Esc while typing, want empty", got)
	}

	// All original rows must still be visible (filter was not committed).
	view := p.View()
	if !strings.Contains(view, "player") {
		t.Errorf("View() after Esc-cancel does not show original rows; view = %q", view)
	}
}

// TestGatewayLivePane_ReverseChronological verifies that events are stored
// newest-first in the buffer.
func TestGatewayLivePane_ReverseChronological(t *testing.T) {
	p, store := newTestGatewayLivePane(t)
	p.SetSize(80, 20)

	// Record events with distinguishable paths.
	store.RecordEvent(domain.GatewayEvent{
		Timestamp: time.Now(),
		Kind:      domain.EventRequestAllowed,
		Method:    "GET",
		Path:      "/v1/me/player/first",
		Snapshot:  domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
	})
	store.RecordEvent(domain.GatewayEvent{
		Timestamp: time.Now(),
		Kind:      domain.EventRequestAllowed,
		Method:    "GET",
		Path:      "/v1/me/player/last",
		Snapshot:  domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
	})

	p.Update(panes.TickMsg{})

	// Buffer should have 2 entries; newest-first means "last" is at index 0.
	if got := p.BufferedEventCount(); got != 2 {
		t.Fatalf("BufferedEventCount() = %d, want 2", got)
	}

	// The view should show "last" before "first" (it renders buffer[0] first).
	view := p.View()
	idxFirst := strings.Index(view, "first")
	idxLast := strings.Index(view, "last")

	if idxFirst == -1 || idxLast == -1 {
		// Paths are trimmed at "/v1/me" so "first" and "last" should still appear.
		t.Fatalf("expected substrings 'first' and 'last' in view — test setup is broken; view = %q", view)
	}
	if idxLast > idxFirst {
		t.Errorf("reverse-chronological: 'last' appears after 'first' in View(), want newest-first")
	}
}

// TestGatewayLivePane_Actions_FilterActive verifies Actions() always returns the
// filter shortcut regardless of active filter state (consistent with TableBasedPane default).
func TestGatewayLivePane_Actions_FilterActive(t *testing.T) {
	p, _ := newTestGatewayLivePane(t)
	p.SetSize(80, 20)
	p.SetFocused(true)

	// Before filter: should show "f filter".
	actions := p.Actions()
	if len(actions) != 1 || actions[0].Key != "f" {
		t.Errorf("Actions() before filter = %v, want [{f filter}]", actions)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})

	// After filter opens: still shows "f filter" (no Esc-cancel variant in the actions bar).
	actions = p.Actions()
	if len(actions) != 1 || actions[0].Key != "f" {
		t.Errorf("Actions() with active filter = %v, want [{f filter}]", actions)
	}
}

// TestGatewayLivePane_BuildTableRows_AllEventKinds verifies that buildGatewayLiveRow
// produces a distinguishing label substring for every supported event kind.
// Coverage of the per-kind label/glyph/role mapping is the contract for this function.
func TestGatewayLivePane_BuildTableRows_AllEventKinds(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		event   domain.GatewayEvent
		wantSub string // substring that must appear in View() for this event
	}{
		{
			name: "EventTokenRefilled",
			event: domain.GatewayEvent{
				Timestamp: now, Kind: domain.EventTokenRefilled, Method: "GET", Path: "/v1/me/player",
				Snapshot: domain.GatewayStateSnapshot{TokensAvailable: 7, TokensMax: 10},
			},
			wantSub: "Tokens refilled",
		},
		{
			name: "EventSemaphoreAcquired",
			event: domain.GatewayEvent{
				Timestamp: now, Kind: domain.EventSemaphoreAcquired, Method: "GET", Path: "/v1/me/player",
				Snapshot: domain.GatewayStateSnapshot{ConcurrentActive: 2, ConcurrentMax: 5},
			},
			wantSub: "Semaphore acquired",
		},
		{
			name: "EventSemaphoreReleased",
			event: domain.GatewayEvent{
				Timestamp: now, Kind: domain.EventSemaphoreReleased, Method: "GET", Path: "/v1/me/player",
				Snapshot: domain.GatewayStateSnapshot{ConcurrentActive: 1, ConcurrentMax: 5},
			},
			wantSub: "Semaphore released",
		},
		{
			name: "EventRequestBlocked",
			event: domain.GatewayEvent{
				Timestamp: now, Kind: domain.EventRequestBlocked, Method: "GET", Path: "/v1/me/player",
				Snapshot: domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
			},
			wantSub: "blocked",
		},
		{
			name: "EventDedupJoined",
			event: domain.GatewayEvent{
				Timestamp: now, Kind: domain.EventDedupJoined, Method: "GET", Path: "/v1/me/player",
				Snapshot: domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
			},
			wantSub: "dedup joined",
		},
		{
			name: "EventDedupResolved",
			event: domain.GatewayEvent{
				Timestamp: now, Kind: domain.EventDedupResolved, Method: "GET", Path: "/v1/me/player",
				Snapshot: domain.GatewayStateSnapshot{DedupWaiters: 3},
			},
			wantSub: "Dedup resolved",
		},
		{
			name: "EventBackoffStarted",
			event: domain.GatewayEvent{
				Timestamp: now, Kind: domain.EventBackoffStarted, Method: "GET", Path: "/v1/me/player",
				Snapshot: domain.GatewayStateSnapshot{BackoffRemaining: 2.5},
			},
			wantSub: "(retry in",
		},
		{
			name: "EventHttpCompleted",
			event: domain.GatewayEvent{
				Timestamp:  now,
				Kind:       domain.EventHttpCompleted,
				Method:     "GET",
				Path:       "/v1/me/player",
				StatusCode: 200,
				DurationMs: 123,
			},
			wantSub: "200",
		},
		{
			name: "EventRequestAllowed",
			event: domain.GatewayEvent{
				Timestamp: now, Kind: domain.EventRequestAllowed, Method: "GET", Path: "/v1/me/player",
				Snapshot: domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
			},
			wantSub: "allowed",
		},
		{
			name: "EventTokenConsumed",
			event: domain.GatewayEvent{
				Timestamp: now, Kind: domain.EventTokenConsumed, Method: "GET", Path: "/v1/me/player",
				Snapshot: domain.GatewayStateSnapshot{TokensAvailable: 9, TokensMax: 10},
			},
			wantSub: "Token consumed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, store := newTestGatewayLivePane(t)
			p.SetSize(80, 20)
			store.RecordEvent(tt.event)
			p.Update(panes.TickMsg{})

			if p.BufferedEventCount() != 1 {
				t.Fatalf("BufferedEventCount() = %d, want 1 — event was not buffered", p.BufferedEventCount())
			}

			view := p.View()
			if !strings.Contains(view, tt.wantSub) {
				t.Errorf("View() does not contain %q for %s; view = %q", tt.wantSub, tt.name, view)
			}
		})
	}

	// EventBackoffExpired must NOT be buffered (silently skipped).
	t.Run("EventBackoffExpired_skipped", func(t *testing.T) {
		p, store := newTestGatewayLivePane(t)
		p.SetSize(80, 20)
		store.RecordEvent(domain.GatewayEvent{
			Timestamp: now, Kind: domain.EventBackoffExpired, Method: "GET", Path: "/v1/me/player",
		})
		p.Update(panes.TickMsg{})
		if got := p.BufferedEventCount(); got != 0 {
			t.Errorf("BufferedEventCount() = %d for EventBackoffExpired, want 0 (silently skipped)", got)
		}
	})

	// EventRequestEntered priority branch: Interactive vs Background must differ.
	t.Run("EventRequestEntered_PriorityDiffers", func(t *testing.T) {
		pInteractive, storeI := newTestGatewayLivePane(t)
		pInteractive.SetSize(80, 20)
		storeI.RecordEvent(domain.GatewayEvent{
			Timestamp: now, Kind: domain.EventRequestEntered,
			Method: "GET", Path: "/v1/me/player", Priority: domain.PriorityInteractive,
			Snapshot: domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
		})
		pInteractive.Update(panes.TickMsg{})

		pBackground, storeB := newTestGatewayLivePane(t)
		pBackground.SetSize(80, 20)
		storeB.RecordEvent(domain.GatewayEvent{
			Timestamp: now, Kind: domain.EventRequestEntered,
			Method: "GET", Path: "/v1/me/player", Priority: domain.PriorityBackground,
			Snapshot: domain.GatewayStateSnapshot{TokensAvailable: 10, TokensMax: 10},
		})
		pBackground.Update(panes.TickMsg{})

		viewI := pInteractive.View()
		viewB := pBackground.View()
		if viewI == viewB {
			t.Errorf("View() for PriorityInteractive and PriorityBackground are identical — glyph/intent branch is not exercised")
		}
	})
}
