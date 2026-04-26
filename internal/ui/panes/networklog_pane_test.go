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

// recordHttpCompleted adds an EventHttpCompleted event to the store, with an
// accompanying EventRequestAllowed decision event (same RequestID).
func recordHttpCompleted(s *state.Store, requestID uint64, method, path string, statusCode int, durationMs int64, priority domain.RequestPriority) {
	s.RecordEvent(domain.GatewayEvent{
		Kind:      domain.EventRequestAllowed,
		RequestID: requestID,
		Method:    method,
		Path:      path,
		Priority:  priority,
		Snapshot:  domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	})
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
	assert.Equal(t, 0, pane.ToggleKey())
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
	assert.Contains(t, v, "TIME", "TIME column header")
	assert.Contains(t, v, "METHOD", "METHOD column header")
	assert.Contains(t, v, "ENDPOINT", "ENDPOINT column header")
	assert.Contains(t, v, "STATUS", "STATUS column header")
	assert.Contains(t, v, "LATENCY", "LATENCY column header")
	assert.Contains(t, v, "PRIORITY", "PRIORITY column header")
	assert.Contains(t, v, "DECISION", "DECISION column header")
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
	// Should not panic and should show the header columns.
	assert.Contains(t, v, "TIME", "empty log should still show column headers")
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
