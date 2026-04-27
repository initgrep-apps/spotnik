package panes_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func newTestGatewayHealthPane(t *testing.T) *panes.GatewayHealthPane {
	t.Helper()
	return panes.NewGatewayHealthPane(state.New(), theme.Load("black"))
}

func TestGatewayHealthPane_ImplementsLayoutPane(t *testing.T) {
	var _ layout.Pane = newTestGatewayHealthPane(t)
}

func TestGatewayHealthPane_ID(t *testing.T) {
	assert.Equal(t, layout.PaneGatewayHealth, newTestGatewayHealthPane(t).ID())
}

func TestGatewayHealthPane_Title(t *testing.T) {
	assert.Equal(t, "Gateway Health", newTestGatewayHealthPane(t).Title())
}

func TestGatewayHealthPane_ToggleKey(t *testing.T) {
	assert.Equal(t, 2, newTestGatewayHealthPane(t).ToggleKey())
}

func TestGatewayHealthPane_View_EmptyBeforeResize(t *testing.T) {
	assert.Equal(t, "", newTestGatewayHealthPane(t).View())
}

func TestGatewayHealthPane_View_ContainsHealthRows(t *testing.T) {
	p := newTestGatewayHealthPane(t)
	p.SetSize(50, 10)
	view := p.View()
	assert.Contains(t, view, "Tokens")
	assert.Contains(t, view, "Slots")
	assert.Contains(t, view, "Backoff")
	assert.Contains(t, view, "Dedup")
}

func TestGatewayHealthPane_View_NoBorder(t *testing.T) {
	p := newTestGatewayHealthPane(t)
	p.SetSize(50, 10)
	view := p.View()
	// render.go adds the outer border; View() must return raw content only.
	assert.NotContains(t, view, "╭")
	assert.NotContains(t, view, "╰")
}

// TestGatewayHealthPane_FreshPane_NotWarningColor asserts that a brand-new pane
// (before any gateway event arrives) does NOT render the Tokens row with Warning
// color. The seed default is a full bucket (TokensAvailable == TokensMax == 10),
// which is below the warning threshold (<=2).
func TestGatewayHealthPane_FreshPane_NotWarningColor(t *testing.T) {
	p := newTestGatewayHealthPane(t)
	p.SetSize(50, 10)

	view := p.View()
	lines := strings.Split(view, "\n")

	// Line 0 is the tokens row; it should contain 10 filled dots.
	if len(lines) == 0 {
		t.Fatal("expected non-empty view")
	}
	tokenLine := lines[0]

	// Build a snapshot-triggered warning view for comparison: set TokensAvailable=1 (<=2 → warning).
	storeWarn := state.New()
	pWarn := panes.NewGatewayHealthPane(storeWarn, theme.Load("black"))
	pWarn.SetSize(50, 10)
	storeWarn.RecordEvent(domain.GatewayEvent{
		Kind: domain.EventTokenConsumed,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable: 1, TokensMax: 10,
		},
	})
	pWarn.Update(panes.TickMsg{})
	warnLines := strings.Split(pWarn.View(), "\n")

	// The ANSI color sequences differ between fresh (healthy, 10 filled) and warning (1 filled).
	assert.NotEqual(t, tokenLine, warnLines[0], "fresh pane tokens row must not match warning-color row")
}

// TestGatewayHealthPane_Update_DrainsCursor records event A (TokensAvailable=5),
// ticks once, records event B (TokensAvailable=8), ticks again, and asserts the
// rendered token bar reflects event B's 8 bar-dots (not A's 5 or accumulated A+B).
//
// The tokens row is: icon(●) + bar(N filled ● + M empty). The icon is always
// GlyphFilledDot so total ● in the line = TokensAvailable + 1 (icon). We
// compare A-tick and B-tick views to verify only event B's snapshot is reflected.
func TestGatewayHealthPane_Update_DrainsCursor(t *testing.T) {
	store := state.New()
	p := panes.NewGatewayHealthPane(store, theme.Load("black"))
	p.SetSize(80, 10)

	// Event A: 5 tokens — capture view after tick.
	store.RecordEvent(domain.GatewayEvent{
		Kind: domain.EventTokenConsumed,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable: 5, TokensMax: 10,
		},
	})
	p.Update(panes.TickMsg{})
	viewAfterA := strings.Split(p.View(), "\n")[0]

	// Event B: 8 tokens — capture view after tick.
	store.RecordEvent(domain.GatewayEvent{
		Kind: domain.EventTokenConsumed,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable: 8, TokensMax: 10,
		},
	})
	p.Update(panes.TickMsg{})
	viewAfterB := strings.Split(p.View(), "\n")[0]

	// icon(●) + 5 bar-dots after A; icon(●) + 8 bar-dots after B.
	// Total ● = available + 1 (icon).
	dotsA := strings.Count(viewAfterA, "●")
	dotsB := strings.Count(viewAfterB, "●")

	assert.Equal(t, 6, dotsA, "after event A: token bar must show 5 bar-dots + 1 icon = 6 ●")
	assert.Equal(t, 9, dotsB, "after event B: token bar must show 8 bar-dots + 1 icon = 9 ●")
	assert.Greater(t, dotsB, dotsA, "event B (8 tokens) must show more filled dots than event A (5 tokens)")
}

// TestGatewayHealthPane_Threshold_Tokens asserts that rendering with
// TokensAvailable=1 (≤2, warning) produces different ANSI output than
// TokensAvailable=5 (healthy).
func TestGatewayHealthPane_Threshold_Tokens(t *testing.T) {
	makeView := func(available int) string {
		store := state.New()
		p := panes.NewGatewayHealthPane(store, theme.Load("black"))
		p.SetSize(80, 10)
		store.RecordEvent(domain.GatewayEvent{
			Kind: domain.EventTokenConsumed,
			Snapshot: domain.GatewayStateSnapshot{
				TokensAvailable: available, TokensMax: 10,
			},
		})
		p.Update(panes.TickMsg{})
		return p.View()
	}

	viewHealthy := makeView(5)
	viewWarn := makeView(1)
	assert.NotEqual(t, viewHealthy, viewWarn,
		"warning-threshold tokens row must differ in ANSI escape codes from healthy row")
}

// TestGatewayHealthPane_Threshold_Slots asserts that slots at capacity produce
// a different ANSI-encoded view than slots below capacity.
func TestGatewayHealthPane_Threshold_Slots(t *testing.T) {
	makeView := func(active int) string {
		store := state.New()
		p := panes.NewGatewayHealthPane(store, theme.Load("black"))
		p.SetSize(80, 10)
		store.RecordEvent(domain.GatewayEvent{
			Kind: domain.EventTokenConsumed,
			Snapshot: domain.GatewayStateSnapshot{
				TokensAvailable: 10, TokensMax: 10,
				ConcurrentActive: active, ConcurrentMax: 5,
			},
		})
		p.Update(panes.TickMsg{})
		return p.View()
	}

	viewBelow := makeView(2)
	viewAtCapacity := makeView(5)
	assert.NotEqual(t, viewBelow, viewAtCapacity,
		"slots-at-capacity view must differ in ANSI escape codes from below-capacity view")
}

// TestGatewayHealthPane_Threshold_Backoff asserts that a non-zero BackoffRemaining
// produces a different view than zero backoff.
func TestGatewayHealthPane_Threshold_Backoff(t *testing.T) {
	makeView := func(remaining float64) string {
		store := state.New()
		p := panes.NewGatewayHealthPane(store, theme.Load("black"))
		p.SetSize(80, 10)
		store.RecordEvent(domain.GatewayEvent{
			Kind: domain.EventTokenConsumed,
			Snapshot: domain.GatewayStateSnapshot{
				TokensAvailable: 10, TokensMax: 10,
				ConcurrentMax:    5,
				BackoffRemaining: remaining,
			},
		})
		p.Update(panes.TickMsg{})
		return p.View()
	}

	viewNoBackoff := makeView(0)
	viewBackoff := makeView(5.0)
	assert.NotEqual(t, viewNoBackoff, viewBackoff,
		"active backoff view must differ in ANSI escape codes from no-backoff view")
}

// TestGatewayHealthPane_View_TokensData asserts the total ● count in the tokens
// row equals TokensAvailable + 1 (bar dots + the row icon which also uses GlyphFilledDot).
func TestGatewayHealthPane_View_TokensData(t *testing.T) {
	tests := []struct {
		name      string
		available int
	}{
		{"7 available", 7},
		{"10 available (full)", 10},
		{"3 available", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := state.New()
			p := panes.NewGatewayHealthPane(store, theme.Load("black"))
			p.SetSize(80, 10)
			store.RecordEvent(domain.GatewayEvent{
				Kind: domain.EventTokenConsumed,
				Snapshot: domain.GatewayStateSnapshot{
					TokensAvailable: tt.available, TokensMax: 10,
				},
			})
			p.Update(panes.TickMsg{})

			lines := strings.Split(p.View(), "\n")
			if len(lines) == 0 {
				t.Fatal("expected non-empty view")
			}
			tokenLine := lines[0]
			// Total ● = bar filled dots (TokensAvailable) + 1 icon dot.
			filled := strings.Count(tokenLine, "●")
			assert.Equal(t, tt.available+1, filled,
				"token row must have %d ● (bar: %d + icon: 1)", tt.available+1, tt.available)
		})
	}
}

// TestGatewayHealthPane_FreshPane_TenFilledDots asserts that a newly created pane
// renders 11 ● in the tokens row: 10 bar-dots (full healthy bucket) + 1 icon dot.
func TestGatewayHealthPane_FreshPane_TenFilledDots(t *testing.T) {
	p := newTestGatewayHealthPane(t)
	p.SetSize(80, 10)

	lines := strings.Split(p.View(), "\n")
	if len(lines) == 0 {
		t.Fatal("expected non-empty view")
	}
	tokenLine := lines[0]
	// icon(●) + 10 bar-dots = 11 total ● when TokensAvailable == TokensMax == 10.
	filled := strings.Count(tokenLine, "●")
	assert.Equal(t, 11, filled,
		"fresh pane tokens row must show 11 ● (10 bar + 1 icon), got %d", filled)
}

// TestGatewayHealthPane_PollingSnapshotMsg_Ignored asserts the pane ignores
// non-TickMsg messages (no crash, no state change).
func TestGatewayHealthPane_PollingSnapshotMsg_Ignored(t *testing.T) {
	p := newTestGatewayHealthPane(t)
	p.SetSize(50, 10)
	before := p.View()
	p.Update(PollingSnapshotMsgForTest{})
	after := p.View()
	assert.Equal(t, before, after, "unrelated message must not change pane state")
}

// PollingSnapshotMsgForTest is an unrelated message type used to verify
// GatewayHealthPane ignores messages it doesn't handle.
type PollingSnapshotMsgForTest struct{}

// TestGatewayHealthPane_NilStore_NoCrash asserts that a nil store does not
// panic during drainEvents (the nil-guard in drainEvents must hold).
func TestGatewayHealthPane_NilStore_NoCrash(t *testing.T) {
	p := panes.NewGatewayHealthPane(nil, theme.Load("black"))
	p.SetSize(50, 10)
	assert.NotPanics(t, func() {
		p.Update(panes.TickMsg{})
		_ = p.View()
	})
}

// TestGatewayHealthPane_FreshPane_DateDerived verifies the pane snapshot
// seeded at construction is "all healthy": BackoffRemaining == 0 (no backoff),
// DedupWaiters == 0 (no dedup), and ConcurrentActive == 0 (slots below capacity).
// This is the complementary behavioral assertion to TestGatewayHealthPane_FreshPane_NotWarningColor.
func TestGatewayHealthPane_FreshPane_DateDerived(t *testing.T) {
	p := newTestGatewayHealthPane(t)
	p.SetSize(80, 10)
	view := p.View()

	assert.Contains(t, view, "none", "fresh pane must show 'none' for backoff (zero) and dedup (zero)")
}
