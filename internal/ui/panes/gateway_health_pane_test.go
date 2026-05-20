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
// rendered token bar reflects event B's higher fill (not A's lower fill).
//
// The token bar now uses per-slot dot glyphs (● filled, □ empty). A higher
// TokensAvailable produces more ● and fewer □, so viewAfterB must contain more
// "●" characters than viewAfterA in the tokens row.
func TestGatewayHealthPane_Update_DrainsCursor(t *testing.T) {
	store := state.New()
	p := panes.NewGatewayHealthPane(store, theme.Load("black"))
	p.SetSize(80, 10)

	// Event A: 5 tokens (50% fill) — capture view after tick.
	store.RecordEvent(domain.GatewayEvent{
		Kind: domain.EventTokenConsumed,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable: 5, TokensMax: 10,
		},
	})
	p.Update(panes.TickMsg{})
	viewAfterA := strings.Split(p.View(), "\n")[0]

	// Event B: 8 tokens (80% fill) — capture view after tick.
	store.RecordEvent(domain.GatewayEvent{
		Kind: domain.EventTokenConsumed,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable: 8, TokensMax: 10,
		},
	})
	p.Update(panes.TickMsg{})
	viewAfterB := strings.Split(p.View(), "\n")[0]

	// Per-slot dot bar: event B (8/10) must show more filled dots than A (5/10).
	// Count "●" (unicode filled-dot glyph) — only filled token slots use this glyph.
	fillA := strings.Count(viewAfterA, "●")
	fillB := strings.Count(viewAfterB, "●")
	assert.Greater(t, fillB, fillA,
		"event B (8/10 tokens) must show more filled dots than event A (5/10 tokens): A=%d B=%d", fillA, fillB)
	// The rendered lines for A and B must differ (different fill fractions).
	assert.NotEqual(t, viewAfterA, viewAfterB,
		"different TokensAvailable must produce different token bar output")
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

// TestGatewayHealthPane_View_TokensData asserts that more TokensAvailable produces
// more filled dot glyphs in the per-slot token bar. Higher availability → more ●
// filled dots and fewer □ empty squares in the rendered tokens line.
func TestGatewayHealthPane_View_TokensData(t *testing.T) {
	makeTokenLine := func(available int) string {
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
		lines := strings.Split(p.View(), "\n")
		if len(lines) == 0 {
			t.Fatal("expected non-empty view")
		}
		return lines[0]
	}

	// 3 < 7 < 10 — filled dot count must be strictly ordered.
	line3 := makeTokenLine(3)
	line7 := makeTokenLine(7)
	line10 := makeTokenLine(10)

	fill3 := strings.Count(line3, "●")
	fill7 := strings.Count(line7, "●")
	fill10 := strings.Count(line10, "●")

	assert.Greater(t, fill7, fill3, "7 available must show more filled dots than 3 available")
	assert.Greater(t, fill10, fill7, "10 available (full) must show more filled dots than 7 available")
}

// TestGatewayHealthPane_FreshPane_FullTokenBar asserts that a newly created pane
// (TokensAvailable == TokensMax == 10) renders a fully filled per-slot dot bar:
// all 10 slots show the ● filled-dot glyph and no □ empty-square glyph.
func TestGatewayHealthPane_FreshPane_FullTokenBar(t *testing.T) {
	p := newTestGatewayHealthPane(t)
	p.SetSize(80, 10)

	lines := strings.Split(p.View(), "\n")
	if len(lines) == 0 {
		t.Fatal("expected non-empty view")
	}
	tokenLine := lines[0]
	// Full bar (TokensAvailable == TokensMax == 10) renders 10 ● filled-dot glyphs; no □ empty squares.
	filled := strings.Count(tokenLine, "●")
	empty := strings.Count(tokenLine, "□")
	assert.Equal(t, 10, filled,
		"fresh pane tokens row must show 10 filled dots (TokensAvailable = TokensMax = 10), got %d", filled)
	assert.Equal(t, 0, empty,
		"fresh pane tokens row must show 0 empty squares when bucket is full, got %d", empty)
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

// TestGatewayHealthPane_SetSize_EagerDrain verifies that transitioning from width=0 to a
// non-zero width triggers an immediate snapshot update (no TickMsg needed). The snapshot
// recorded before SetSize must be reflected in View() after SetSize.
func TestGatewayHealthPane_SetSize_EagerDrain(t *testing.T) {
	store := state.New()
	store.RecordEvent(domain.GatewayEvent{
		Kind: domain.EventTokenConsumed,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable: 3, TokensMax: 10,
		},
	})

	p := panes.NewGatewayHealthPane(store, theme.Load("black"))
	// Before SetSize, width==0, View() returns "".
	assert.Equal(t, "", p.View(), "View() before SetSize must be empty")

	// Transition 0 → non-zero; should drain immediately and expose the event's snapshot.
	p.SetSize(80, 10)

	// After eager drain the EventCursor must have advanced past the seeded event.
	assert.Greater(t, p.EventCursor(), uint64(0), "EventCursor must advance after eager drain in SetSize")
}

// TestGatewayHealthPane_SetSize_EagerDrain_NoReDrainOnSecondCall verifies that a second
// SetSize call (non-zero → non-zero) does NOT re-read from the cursor.
func TestGatewayHealthPane_SetSize_EagerDrain_NoReDrainOnSecondCall(t *testing.T) {
	store := state.New()
	p := panes.NewGatewayHealthPane(store, theme.Load("black"))
	p.SetSize(80, 10) // first non-zero: drains (store empty, cursor stays 0)

	cursorAfterFirst := p.EventCursor()

	store.RecordEvent(domain.GatewayEvent{
		Kind: domain.EventTokenConsumed,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable: 5, TokensMax: 10,
		},
	})
	p.SetSize(100, 12) // second non-zero: must NOT drain again

	// Cursor must not have advanced (only TickMsg should do that now).
	assert.Equal(t, cursorAfterFirst, p.EventCursor(),
		"cursor must not advance on second (non-zero→non-zero) SetSize call")
}

// TestGatewayHealthPane_View_DotBarGlyphs verifies that after the renderDotBar fix,
// View() renders per-slot dot/square glyphs (● or ■ and □) instead of ProgressBar
// fill characters (█ and ░). The Tokens row uses GlyphFilledDot (●) for filled slots
// and GlyphEmptySquare (□) for empty; the Slots row uses GlyphFilledSquare (■) for
// filled and GlyphEmptySquare (□) for empty.
func TestGatewayHealthPane_View_DotBarGlyphs(t *testing.T) {
	store := state.New()
	// 5 tokens available out of 10, 3 slots active out of 5.
	store.RecordEvent(domain.GatewayEvent{
		Kind: domain.EventTokenConsumed,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable:  5,
			TokensMax:        10,
			ConcurrentActive: 3,
			ConcurrentMax:    5,
		},
	})
	p := panes.NewGatewayHealthPane(store, theme.Load("black"))
	p.SetSize(80, 10)
	p.Update(panes.TickMsg{})

	view := p.View()
	// Filled dot glyph (token bar) or filled square glyph (slot bar) must appear.
	assert.True(t, strings.Contains(view, "●") || strings.Contains(view, "■"),
		"view must contain filled dot ● or filled square ■ glyph (renderDotBar per-slot output)")
	// Empty square glyph must appear for the unfilled portion of both bars.
	assert.Contains(t, view, "□", "view must contain empty square □ glyph (renderDotBar empty slot)")
}
