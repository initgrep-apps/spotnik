package panes_test

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tea_keyMsg builds a tea.KeyMsg for a single string key (e.g. "f", "j", "k").
func tea_keyMsg(key string) tea.KeyMsg {
	switch key {
	case "Esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "Enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	default:
		if len(key) == 1 {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

// tea_keyMsgRune builds a tea.KeyMsg for a single rune character.
func tea_keyMsgRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// newTestNetworkLogPane creates a NetworkLogPane with an empty store for testing.
func newTestNetworkLogPane() *panes.NetworkLogPane {
	s := state.New()
	t := theme.Load("black")
	return panes.NewNetworkLogPane(s, t)
}

// recordHttpCompleted adds gateway events to the store in production emission order:
// EventHttpCompleted fires FIRST (gateway.go line 513), then EventRequestAllowed
// (gateway.go line 524). Using production order here ensures tests match real
// gateway behavior and validate the backfill sweep in refreshRows.
func recordHttpCompleted(s *state.Store, requestID uint64, method, path string, statusCode int, durationMs int64, priority domain.RequestPriority) {
	s.RecordEvent(domain.GatewayEvent{
		Timestamp:  time.Now(),
		Kind:       domain.EventHttpCompleted,
		RequestID:  requestID,
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		DurationMs: durationMs,
		Priority:   priority,
		Snapshot:   domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})
	s.RecordEvent(domain.GatewayEvent{
		Kind:      domain.EventRequestAllowed,
		RequestID: requestID,
		Method:    method,
		Path:      path,
		Priority:  priority,
		Snapshot:  domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})
}

// recordBlocked adds an EventRequestBlocked event to the store for a background request.
func recordBlocked(s *state.Store, requestID uint64, method, path string) {
	s.RecordEvent(domain.GatewayEvent{
		Timestamp: time.Now(),
		Kind:      domain.EventRequestBlocked,
		RequestID: requestID,
		Method:    method,
		Path:      path,
		Priority:  domain.PriorityBackground,
		Snapshot:  domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})
}

// --- Interface satisfaction ---

func TestNetworkLogPane_ImplementsLayoutPane(t *testing.T) {
	var _ layout.Pane = &panes.NetworkLogPane{}
}

// --- ID / Title / ToggleKey / Actions ---

func TestNetworkLogPane_ID(t *testing.T) {
	pane := newTestNetworkLogPane()
	assert.Equal(t, layout.PaneNetworkLog, pane.ID())
}

func TestNetworkLogPane_Title(t *testing.T) {
	pane := newTestNetworkLogPane()
	assert.Equal(t, "Network Log", pane.Title())
}

func TestNetworkLogPane_ToggleKey(t *testing.T) {
	pane := newTestNetworkLogPane()
	assert.Equal(t, 5, pane.ToggleKey())
}

func TestNetworkLogPane_Actions_NoFilter(t *testing.T) {
	pane := newTestNetworkLogPane()
	pane.SetFocused(true)
	actions := pane.Actions()
	require.NotNil(t, actions)
	keys := make([]string, len(actions))
	for i, a := range actions {
		keys[i] = a.Key
	}
	assert.Contains(t, keys, "f", "f key should appear in actions when filter is not active")
	assert.NotContains(t, keys, "j/k", "j/k implicit scroll hint must not appear in actions (5D)")
}

func TestNetworkLogPane_Actions_WithFilter(t *testing.T) {
	pane := newTestNetworkLogPane()
	pane.SetFocused(true)
	pane.SetSize(80, 20)
	// Activate filter via 'f' key.
	_, _ = pane.Update(tea_keyMsg("f"))
	actions := pane.Actions()
	require.NotNil(t, actions)
	assert.Equal(t, "Esc", actions[0].Key, "Esc should be the only action when filter is open")
}

// --- SetSize / SetFocused / IsFocused ---

func TestNetworkLogPane_SetSize(t *testing.T) {
	pane := newTestNetworkLogPane()
	pane.SetSize(80, 20)
	_ = pane.View()
}

func TestNetworkLogPane_FocusRoundtrip(t *testing.T) {
	pane := newTestNetworkLogPane()
	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
	pane.SetFocused(false)
	assert.False(t, pane.IsFocused())
}

// --- Init ---

func TestNetworkLogPane_Init_ReturnsNil(t *testing.T) {
	pane := newTestNetworkLogPane()
	assert.Nil(t, pane.Init())
}

// --- Table shows columns with correct headers ---

func TestNetworkLogPane_View_ShowsAllColumns(t *testing.T) {
	pane := newTestNetworkLogPane()
	pane.SetSize(160, 20)
	v := pane.View()
	// Title Case headers (not all-caps).
	assert.Contains(t, v, "Time", "Time column header")
	assert.Contains(t, v, "Method", "Method column header")
	assert.Contains(t, v, "Endpoint", "Endpoint column header")
	assert.Contains(t, v, "Status", "Status column header")
	assert.Contains(t, v, "Latency", "Latency column header")
	assert.Contains(t, v, "Priority", "Priority column header")
	assert.Contains(t, v, "Decision", "Decision column header")
	// Verify old all-caps headers are gone.
	assert.NotContains(t, v, "TIME", "TIME header must not appear — Title Case required")
	assert.NotContains(t, v, "METHOD", "METHOD header must not appear — Title Case required")
	assert.NotContains(t, v, "STATUS", "STATUS header must not appear — Title Case required")
	assert.NotContains(t, v, "LATENCY", "LATENCY header must not appear — Title Case required")
	assert.NotContains(t, v, "PRIORITY", "PRIORITY header must not appear — Title Case required")
	assert.NotContains(t, v, "DECISION", "DECISION header must not appear — Title Case required")
}

// --- Cursor-based reads ---

func TestNetworkLogPane_RefreshRows_CursorAdvances(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	// Before adding events, cursor should be at 0 (no events).
	cursorBefore := pane.EventCursor()

	// Add an event.
	recordHttpCompleted(s, 1, "GET", "/me/player", 200, 45, domain.PriorityBackground)

	// TickMsg triggers refreshRows which should advance the cursor.
	_, _ = pane.Update(panes.TickMsg{})

	cursorAfter := pane.EventCursor()
	assert.Greater(t, cursorAfter, cursorBefore, "cursor should advance after reading events")
}

func TestNetworkLogPane_RefreshRows_IncrementalDrain(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	// Add 2 http-completed events (each has 2 sub-events: allowed + completed).
	recordHttpCompleted(s, 1, "GET", "/me/player", 200, 45, domain.PriorityBackground)
	recordHttpCompleted(s, 2, "GET", "/me/playlists", 200, 88, domain.PriorityBackground)

	// First refresh — drain both.
	_, _ = pane.Update(panes.TickMsg{})
	v1 := pane.View()
	assert.Contains(t, v1, "/me/player")
	assert.Contains(t, v1, "/me/playlists")

	cursor1 := pane.EventCursor()

	// Add one more event.
	recordHttpCompleted(s, 3, "GET", "/me/queue", 200, 22, domain.PriorityBackground)

	// Second refresh — should only see the new event.
	_, _ = pane.Update(panes.TickMsg{})
	cursor2 := pane.EventCursor()
	assert.Greater(t, cursor2, cursor1, "cursor must advance after second drain")

	v2 := pane.View()
	assert.Contains(t, v2, "/me/queue", "third request should be visible after second refresh")
}

func TestNetworkLogPane_RefreshRows_HttpCompletedAppearsInTable(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	recordHttpCompleted(s, 1, "GET", "/me/player", 200, 55, domain.PriorityBackground)

	_, _ = pane.Update(panes.TickMsg{})

	v := pane.View()
	assert.Contains(t, v, "/me/player", "HTTP completed event should appear as table row")
	assert.Contains(t, v, "200")
}

func TestNetworkLogPane_RefreshRows_BlockedRequestAppearsInTable(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	recordBlocked(s, 1, "GET", "/me/player")

	_, _ = pane.Update(panes.TickMsg{})

	v := pane.View()
	assert.Contains(t, v, "/me/player", "blocked event should appear as table row")
	assert.Contains(t, v, "0", "blocked request should show status 0")
}

func TestNetworkLogPane_RefreshRows_CapsAt200(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	// Add 250 http completed events.
	for i := 0; i < 250; i++ {
		recordHttpCompleted(s, uint64(i+1), "GET", "/me/player", 200, int64(i+10), domain.PriorityBackground)
	}

	_, _ = pane.Update(panes.TickMsg{})

	// CompletedRequests should be capped at 200.
	assert.LessOrEqual(t, pane.CompletedRequestsLen(), 200, "completed requests must be capped at 200")
}

// --- PRIORITY and DECISION columns ---

func TestNetworkLogPane_View_ShowsPriorityColumn(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	// Interactive request.
	recordHttpCompleted(s, 1, "GET", "/me/player", 200, 45, domain.PriorityInteractive)
	// Background request.
	recordHttpCompleted(s, 2, "GET", "/me/playlists", 200, 88, domain.PriorityBackground)

	_, _ = pane.Update(panes.TickMsg{})

	v := pane.View()
	assert.Contains(t, v, "interactive", "interactive request should show '⚡ interactive' in PRIORITY column")
	assert.Contains(t, v, "background", "background request should show '◷ background' in PRIORITY column")
}

func TestNetworkLogPane_View_ShowsDecisionColumn(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	// Add an allowed request.
	s.RecordEvent(domain.GatewayEvent{
		Kind:      domain.EventRequestAllowed,
		RequestID: 1,
		Method:    "GET",
		Path:      "/me/player",
		Priority:  domain.PriorityBackground,
	})
	s.RecordEvent(domain.GatewayEvent{
		Timestamp:  time.Now(),
		Kind:       domain.EventHttpCompleted,
		RequestID:  1,
		Method:     "GET",
		Path:       "/me/player",
		StatusCode: 200,
		DurationMs: 45,
		Priority:   domain.PriorityBackground,
	})

	// Add a blocked request.
	recordBlocked(s, 2, "GET", "/me/queue")

	_, _ = pane.Update(panes.TickMsg{})

	v := pane.View()
	assert.Contains(t, v, "allowed", "allowed request should show 'allowed' in DECISION column")
	assert.Contains(t, v, "blocked", "blocked request should show 'blocked' in DECISION column")
}

func TestNetworkLogPane_View_BlockedNotesColumn(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	recordBlocked(s, 1, "GET", "/me/player")

	_, _ = pane.Update(panes.TickMsg{})

	v := pane.View()
	assert.Contains(t, v, "blocked", "blocked request decision column should show 'blocked'")
}

// --- Filter extended for priority and decision ---

func TestNetworkLogPane_Filter_MatchesPriority(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)
	pane.SetFocused(true)

	recordHttpCompleted(s, 1, "GET", "/me/player", 200, 45, domain.PriorityInteractive)
	recordHttpCompleted(s, 2, "GET", "/me/playlists", 200, 88, domain.PriorityBackground)

	_, _ = pane.Update(panes.TickMsg{})

	// Filter by "interactive" to show only interactive requests.
	_, _ = pane.Update(tea_keyMsg("f"))
	for _, ch := range "interactive" {
		_, _ = pane.Update(tea_keyMsgRune(ch))
	}

	v := pane.View()
	assert.Contains(t, v, "/me/player", "interactive endpoint should be visible with 'interactive' filter")
	assert.NotContains(t, v, "/me/playlists", "background endpoint should be hidden with 'interactive' filter")
}

func TestNetworkLogPane_Filter_MatchesDecision(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)
	pane.SetFocused(true)

	recordHttpCompleted(s, 1, "GET", "/me/player", 200, 45, domain.PriorityBackground)
	recordBlocked(s, 2, "GET", "/me/queue")

	_, _ = pane.Update(panes.TickMsg{})

	// Filter by "blocked" to show only blocked.
	_, _ = pane.Update(tea_keyMsg("f"))
	for _, ch := range "blocked" {
		_, _ = pane.Update(tea_keyMsgRune(ch))
	}

	v := pane.View()
	assert.Contains(t, v, "/me/queue", "blocked endpoint should be visible with 'blocked' filter")
	assert.NotContains(t, v, "/me/player", "allowed endpoint should be hidden with 'blocked' filter")
}

func TestNetworkLogPane_Filter_MatchesEndpoint(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)
	pane.SetFocused(true)

	recordHttpCompleted(s, 1, "GET", "/me/player", 200, 45, domain.PriorityBackground)
	recordHttpCompleted(s, 2, "GET", "/me/playlists", 200, 88, domain.PriorityBackground)

	_, _ = pane.Update(panes.TickMsg{})

	// Open filter and type "/me/player" to filter by endpoint.
	_, _ = pane.Update(tea_keyMsg("f"))
	for _, ch := range "/me/player" {
		_, _ = pane.Update(tea_keyMsgRune(ch))
	}

	v := pane.View()
	assert.Contains(t, v, "/me/player", "filtered endpoint should be visible")
	assert.NotContains(t, v, "/me/playlists", "non-matching endpoint should be hidden")
}

func TestNetworkLogPane_Filter_ByStatusCode(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)
	pane.SetFocused(true)

	recordHttpCompleted(s, 1, "GET", "/me/player", 200, 45, domain.PriorityBackground)
	recordHttpCompleted(s, 2, "GET", "/me/player/queue", 429, 12, domain.PriorityBackground)

	_, _ = pane.Update(panes.TickMsg{})

	// Open filter and type "429".
	_, _ = pane.Update(tea_keyMsg("f"))
	for _, ch := range "429" {
		_, _ = pane.Update(tea_keyMsgRune(ch))
	}

	v := pane.View()
	assert.Contains(t, v, "429", "429 entries should appear when filtered by status code")
}

// --- Color coding ---

func TestNetworkLogPane_View_Status200_InView(t *testing.T) {
	s := state.New()
	recordHttpCompleted(s, 1, "GET", "/me/player", 200, 45, domain.PriorityBackground)
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(160, 20)
	_, _ = pane.Update(panes.TickMsg{})
	v := pane.View()
	assert.Contains(t, v, "200")
	assert.Contains(t, v, "/me/player")
}

func TestNetworkLogPane_View_Status429_ShowsWarningMarker(t *testing.T) {
	s := state.New()
	recordHttpCompleted(s, 1, "GET", "/me/player", 429, 12, domain.PriorityBackground)
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(160, 20)
	_, _ = pane.Update(panes.TickMsg{})
	v := pane.View()
	assert.Contains(t, v, "429")
}

func TestNetworkLogPane_View_Status500_InView(t *testing.T) {
	s := state.New()
	recordHttpCompleted(s, 1, "GET", "/me/player", 500, 100, domain.PriorityBackground)
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(160, 20)
	_, _ = pane.Update(panes.TickMsg{})
	v := pane.View()
	assert.Contains(t, v, "500")
}

// --- Latency bar ---

// --- Entries sorted newest-first ---

func TestNetworkLogPane_View_NewestFirst(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Add older entry first (requestID 1).
	s.RecordEvent(domain.GatewayEvent{
		Kind:      domain.EventRequestAllowed,
		RequestID: 1,
		Path:      "/me/player/queue",
	})
	s.RecordEvent(domain.GatewayEvent{
		Timestamp:  time.Now().Add(-2 * time.Second),
		Kind:       domain.EventHttpCompleted,
		RequestID:  1,
		Method:     "GET",
		Path:       "/me/player/queue",
		StatusCode: 200,
		DurationMs: 62,
	})

	// Add newer entry second (requestID 2).
	s.RecordEvent(domain.GatewayEvent{
		Kind:      domain.EventRequestAllowed,
		RequestID: 2,
		Path:      "/me/player",
	})
	s.RecordEvent(domain.GatewayEvent{
		Timestamp:  time.Now(),
		Kind:       domain.EventHttpCompleted,
		RequestID:  2,
		Method:     "GET",
		Path:       "/me/player",
		StatusCode: 200,
		DurationMs: 45,
	})

	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)
	v := pane.View()

	// Newest entry (/me/player) should appear before older (/me/player/queue).
	playerIdx := indexInStr(v, "/me/player")
	queueIdx := indexInStr(v, "/me/player/queue")
	require.NotEqual(t, -1, playerIdx, "/me/player should be in view")
	require.NotEqual(t, -1, queueIdx, "/me/player/queue should be in view")
	assert.Less(t, playerIdx, queueIdx, "newest entry should appear first (smaller offset)")
}

// --- Scrolling ---

func TestNetworkLogPane_Scrolling_JKey(t *testing.T) {
	s := state.New()
	// Add enough entries to require scrolling.
	for i := 0; i < 30; i++ {
		recordHttpCompleted(s, uint64(i+1), "GET", "/me/player", 200, int64(i+10), domain.PriorityBackground)
	}
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(160, 15)
	pane.SetFocused(true)

	idxBefore := pane.SelectedIndex()
	_, _ = pane.Update(tea_keyMsg("j"))
	idxAfter := pane.SelectedIndex()
	assert.Greater(t, idxAfter, idxBefore, "j key should move cursor down")
}

func TestNetworkLogPane_Scrolling_KKey(t *testing.T) {
	s := state.New()
	for i := 0; i < 30; i++ {
		recordHttpCompleted(s, uint64(i+1), "GET", "/me/player", 200, int64(i+10), domain.PriorityBackground)
	}
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(160, 15)
	pane.SetFocused(true)

	// Move down first.
	_, _ = pane.Update(tea_keyMsg("j"))
	_, _ = pane.Update(tea_keyMsg("j"))
	idxBefore := pane.SelectedIndex()
	_, _ = pane.Update(tea_keyMsg("k"))
	idxAfter := pane.SelectedIndex()
	assert.Less(t, idxAfter, idxBefore, "k key should move cursor up")
}

// --- Empty log ---

func TestNetworkLogPane_EmptyLog_CleanState(t *testing.T) {
	pane := newTestNetworkLogPane()
	pane.SetSize(160, 20)
	v := pane.View()
	// Should not panic and should show the header columns (Title Case since story 177).
	assert.Contains(t, v, "Time", "empty log should still show column headers")
}

// --- Full 200-entry buffer ---

func TestNetworkLogPane_FullBuffer_Scrollable(t *testing.T) {
	s := state.New()
	// Fill the 200-entry buffer.
	for i := 0; i < 200; i++ {
		recordHttpCompleted(s, uint64(i+1), "GET", "/me/player", 200, int64(i+10), domain.PriorityBackground)
	}
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(160, 20)
	pane.SetFocused(true)

	// View must not panic with a full buffer.
	v := pane.View()
	assert.Contains(t, v, "200", "full-buffer view should render without panic")

	// Scrolling should work.
	_, _ = pane.Update(tea_keyMsg("j"))
	_ = pane.View()
}

// --- TickMsg refreshes rows ---

func TestNetworkLogPane_TickMsg_RefreshesRows(t *testing.T) {
	s := state.New()
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(160, 20)

	// Initially empty.
	v1 := pane.View()
	assert.NotContains(t, v1, "/me/player")

	// Add an event.
	recordHttpCompleted(s, 1, "GET", "/me/player", 200, 45, domain.PriorityBackground)

	// TickMsg should trigger row refresh.
	_, _ = pane.Update(panes.TickMsg{})

	v2 := pane.View()
	assert.Contains(t, v2, "/me/player", "after TickMsg, new log entry should appear")
}

// --- Helpers ---

// indexInStr returns the index of substr in s, or -1 if not found.
func indexInStr(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// ── Story 173: Esc scroll-reset ───────────────────────────────────────────────

// TestNetworkLogPane_Esc_ResetsScrollToPage1 verifies that pressing Esc when no
// filter is active resets the table scroll position back to page 1.
func TestNetworkLogPane_Esc_ResetsScrollToPage1(t *testing.T) {
	s := state.New()
	// Record 20 completed requests to fill multiple pages.
	for i := 0; i < 20; i++ {
		recordHttpCompleted(s, uint64(i+1), "GET", fmt.Sprintf("/v1/track/%d", i), 200, 50, domain.PriorityBackground)
	}
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetFocused(true)
	// height=11 → pageSize=5 with ShowHeader=true (pageSize = height - 6).
	pane.SetSize(80, 11)

	// Trigger a tick to load events into the pane.
	m, _ := pane.Update(panes.TickMsg{})
	pane = m.(*panes.NetworkLogPane)

	// Scroll 8 rows down to advance past page 1.
	for i := 0; i < 8; i++ {
		m, _ = pane.Update(tea_keyMsg("j"))
		pane = m.(*panes.NetworkLogPane)
	}
	require.Greater(t, pane.TableCurrentPage(), 1, "should have scrolled past page 1")

	// Press Esc with no active filter — should reset to page 1.
	m, _ = pane.Update(tea_keyMsg("Esc"))
	pane = m.(*panes.NetworkLogPane)
	assert.Equal(t, 1, pane.TableCurrentPage(), "Esc should reset table to page 1")
}

// ── Story 177: Decision persisted across ticks ────────────────────────────────

// TestNetworkLogPane_Decision_PersistedAcrossTicks verifies that the Decision
// column shows the correct value when EventRequestAllowed arrives on tick N and
// EventHttpCompleted arrives on tick N+1 for the same RequestID.
//
// With the old method-local decisions map this fails: the allowed decision from
// tick N is lost, and the row built on tick N+1 shows an empty Decision cell.
func TestNetworkLogPane_Decision_PersistedAcrossTicks(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	// Tick 1: only EventRequestAllowed is recorded.
	s.RecordEvent(domain.GatewayEvent{
		Kind:      domain.EventRequestAllowed,
		RequestID: 42,
		Method:    "GET",
		Path:      "/cross-tick/path",
		Priority:  domain.PriorityBackground,
		Snapshot:  domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})
	m, _ := pane.Update(panes.TickMsg{})
	pane = m.(*panes.NetworkLogPane)

	// No completed event yet — pane should show nothing for this request.
	v1 := pane.View()
	assert.NotContains(t, v1, "/cross-tick/path", "no completed row yet on tick 1")

	// Tick 2: EventHttpCompleted arrives for the SAME RequestID.
	s.RecordEvent(domain.GatewayEvent{
		Kind:       domain.EventHttpCompleted,
		RequestID:  42,
		Method:     "GET",
		Path:       "/cross-tick/path",
		StatusCode: 200,
		DurationMs: 55,
		Priority:   domain.PriorityBackground,
		Snapshot:   domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})
	m, _ = pane.Update(panes.TickMsg{})
	pane = m.(*panes.NetworkLogPane)

	v2 := pane.View()
	assert.Contains(t, v2, "/cross-tick/path", "completed row must appear after tick 2")
	assert.Contains(t, v2, "allowed", "Decision column must show 'allowed' — decision from tick 1 must persist into tick 2")
}

// TestNetworkLogPane_Decision_ProductionOrder_CrossTick verifies the Decision column
// shows "allowed" when EventHttpCompleted arrives BEFORE EventRequestAllowed (the
// real gateway emission order: HttpCompleted on line 513, Allowed on line 524).
// Split across ticks:
//
//	Tick N: EventHttpCompleted arrives (row created with zero decision)
//	Tick N+1: EventRequestAllowed arrives (backfill sweep fills in the decision)
func TestNetworkLogPane_Decision_ProductionOrder_CrossTick(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	// Tick 1: only EventHttpCompleted is recorded (production order: HttpCompleted first).
	s.RecordEvent(domain.GatewayEvent{
		Timestamp:  time.Now(),
		Kind:       domain.EventHttpCompleted,
		RequestID:  99,
		Method:     "GET",
		Path:       "/production-order/path",
		StatusCode: 200,
		DurationMs: 30,
		Priority:   domain.PriorityBackground,
		Snapshot:   domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})
	m, _ := pane.Update(panes.TickMsg{})
	pane = m.(*panes.NetworkLogPane)

	// Row should appear with blank decision (Allowed not yet received).
	v1 := pane.View()
	assert.Contains(t, v1, "/production-order/path", "row must appear after HttpCompleted")

	// Tick 2: EventRequestAllowed arrives for the SAME RequestID.
	s.RecordEvent(domain.GatewayEvent{
		Kind:      domain.EventRequestAllowed,
		RequestID: 99,
		Method:    "GET",
		Path:      "/production-order/path",
		Priority:  domain.PriorityBackground,
		Snapshot:  domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})
	m, _ = pane.Update(panes.TickMsg{})
	pane = m.(*panes.NetworkLogPane)

	v2 := pane.View()
	assert.Contains(t, v2, "/production-order/path", "row still present after tick 2")
	assert.Contains(t, v2, "allowed", "Decision column must show 'allowed' after backfill sweep")
}

// TestNetworkLogPane_DedupJoined_NoPendingDecisionsLeak verifies that
// EventDedupJoined entries do not accumulate indefinitely in pendingDecisions.
// Dedup waiters emit EventDedupJoined then EventDedupResolved and never emit
// EventHttpCompleted. Without handling EventDedupResolved the map grows without bound.
func TestNetworkLogPane_DedupJoined_NoPendingDecisionsLeak(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	// Emit 250 DedupJoined+DedupResolved pairs (simulating 250 background waiter requests).
	for i := uint64(1); i <= 250; i++ {
		s.RecordEvent(domain.GatewayEvent{
			Kind:      domain.EventDedupJoined,
			RequestID: i,
			Method:    "GET",
			Path:      "/me/player",
			Priority:  domain.PriorityBackground,
		})
		s.RecordEvent(domain.GatewayEvent{
			Kind:      domain.EventDedupResolved,
			RequestID: i,
			Method:    "GET",
			Path:      "/me/player",
			Priority:  domain.PriorityBackground,
		})
	}

	m, _ := pane.Update(panes.TickMsg{})
	pane = m.(*panes.NetworkLogPane)

	// pendingDecisions must not grow — DedupResolved should clear each entry.
	assert.Equal(t, 0, pane.PendingDecisionsLen(),
		"pendingDecisions must be empty after DedupResolved clears each DedupJoined entry")
}

// TestNetworkLogPane_Decision_DedupJoined_PersistedAcrossTicks verifies the Decision
// column shows "dedup" for a request that joined a dedup group, even when
// EventDedupJoined arrives on tick N and EventHttpCompleted (for the primary's ID, not
// used here) arrives separately. This test drives via the primary path to confirm the
// dedup waiter row (EventDedupJoined → EventDedupResolved) does not show in the log
// (only primary's HttpCompleted rows appear). The key regression guard is that a
// hypothetical dedup-primary row created with a pending DedupJoined decision is backfilled.
//
// Concretely: emit EventDedupJoined for request X on tick N, then emit EventHttpCompleted
// for a SEPARATE primary requestID Y on the same tick (so the primary gets a row). The
// primary's pending decision should come from EventRequestAllowed on tick N+1.
// Meanwhile request X (the waiter) never creates a row (no HttpCompleted for it).
// Emit EventDedupResolved for X on tick N+1 to clear the map entry.
func TestNetworkLogPane_Decision_DedupJoined_PersistedAcrossTicks(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 20)

	// Tick 1: primary's HttpCompleted (ID=10) and waiter's DedupJoined (ID=11).
	s.RecordEvent(domain.GatewayEvent{
		Timestamp:  time.Now(),
		Kind:       domain.EventHttpCompleted,
		RequestID:  10,
		Method:     "GET",
		Path:       "/dedup-primary/path",
		StatusCode: 200,
		DurationMs: 40,
		Priority:   domain.PriorityBackground,
		Snapshot:   domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})
	s.RecordEvent(domain.GatewayEvent{
		Kind:      domain.EventDedupJoined,
		RequestID: 11,
		Method:    "GET",
		Path:      "/dedup-primary/path",
		Priority:  domain.PriorityBackground,
	})
	m, _ := pane.Update(panes.TickMsg{})
	pane = m.(*panes.NetworkLogPane)

	// Primary row appears with blank decision (Allowed not yet received).
	v1 := pane.View()
	assert.Contains(t, v1, "/dedup-primary/path", "primary row must appear")
	// Waiter has no HttpCompleted so no second row.
	assert.Equal(t, 1, pane.CompletedRequestsLen(), "only one row — the primary")

	// Tick 2: Allowed arrives for primary (ID=10) and DedupResolved for waiter (ID=11).
	s.RecordEvent(domain.GatewayEvent{
		Kind:      domain.EventRequestAllowed,
		RequestID: 10,
		Method:    "GET",
		Path:      "/dedup-primary/path",
		Priority:  domain.PriorityBackground,
		Snapshot:  domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})
	s.RecordEvent(domain.GatewayEvent{
		Kind:      domain.EventDedupResolved,
		RequestID: 11,
		Method:    "GET",
		Path:      "/dedup-primary/path",
		Priority:  domain.PriorityBackground,
	})
	m, _ = pane.Update(panes.TickMsg{})
	pane = m.(*panes.NetworkLogPane)

	v2 := pane.View()
	assert.Contains(t, v2, "/dedup-primary/path", "row still present")
	assert.Contains(t, v2, "allowed", "Decision column must show 'allowed' after backfill on tick 2")
	assert.Equal(t, 0, pane.PendingDecisionsLen(),
		"pendingDecisions must be empty — DedupResolved clears waiter, Allowed consumed by backfill")
}

// ── Story 174: Filter_EscCloses ───────────────────────────────────────────────

// TestNetworkLogPane_Filter_EscCloses verifies that Esc while the filter is active
// closes the filter and does NOT reset scroll position.
func TestNetworkLogPane_Filter_EscCloses(t *testing.T) {
	s := state.New()
	for i := 0; i < 20; i++ {
		recordHttpCompleted(s, uint64(i+1), "GET", fmt.Sprintf("/v1/track/%d", i), 200, 50, domain.PriorityBackground)
	}
	th := theme.Load("black")
	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(160, 11) // pageSize=5
	pane.SetFocused(true)

	// Trigger tick to load rows.
	m, _ := pane.Update(panes.TickMsg{})
	pane = m.(*panes.NetworkLogPane)

	// Scroll to page 2 before activating the filter.
	for i := 0; i < 8; i++ {
		m, _ = pane.Update(tea_keyMsg("j"))
		pane = m.(*panes.NetworkLogPane)
	}
	pageBeforeFilter := pane.TableCurrentPage()
	require.Greater(t, pageBeforeFilter, 1, "pre-condition: should be past page 1")

	// Activate filter.
	m, _ = pane.Update(tea_keyMsg("f"))
	pane = m.(*panes.NetworkLogPane)
	require.True(t, pane.HasActiveFilter(), "filter should be active after pressing f")

	// Press Esc — filter should close without resetting scroll.
	m, _ = pane.Update(tea_keyMsg("Esc"))
	pane = m.(*panes.NetworkLogPane)
	assert.False(t, pane.HasActiveFilter(), "Esc should close the filter")
	assert.Equal(t, pageBeforeFilter, pane.TableCurrentPage(), "Esc should NOT reset scroll when closing the filter")
	assert.Contains(t, pane.View(), "/v1/track/", "full log should be visible after filter close")
}
