package panes_test

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

// addTestEntry adds a NetLogEntry to the store's net log via the Store.RecordRequest helper.
func addTestEntry(s *state.Store, method, path string, statusCode int, durationMs int64) {
	// Use the public API: store.RecordRequest (implements api.NetLogRecorder).
	// NetLog is populated via store.RecordAPICall which is called by the API layer.
	// For tests we write entries via the raw NetLog.
	s.NetLog().Add(state.NetLogEntry{
		Timestamp:  time.Now(),
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		DurationMs: durationMs,
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
	assert.Contains(t, keys, "j/k", "j/k should appear as scroll hint")
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

// --- Table shows 6 columns with correct headers ---

func TestNetworkLogPane_View_ShowsSixColumns(t *testing.T) {
	pane := newTestNetworkLogPane()
	pane.SetSize(120, 20)
	v := pane.View()
	assert.Contains(t, v, "TIME", "TIME column header")
	assert.Contains(t, v, "METHOD", "METHOD column header")
	assert.Contains(t, v, "ENDPOINT", "ENDPOINT column header")
	assert.Contains(t, v, "STATUS", "STATUS column header")
	assert.Contains(t, v, "LATENCY", "LATENCY column header")
	assert.Contains(t, v, "NOTES", "NOTES column header")
}

// --- Entries sorted newest-first ---

func TestNetworkLogPane_View_NewestFirst(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	// Add older entry first.
	s.NetLog().Add(state.NetLogEntry{
		Timestamp:  time.Now().Add(-2 * time.Second),
		Method:     "GET",
		Path:       "/me/player/queue",
		StatusCode: 200,
		DurationMs: 62,
	})
	// Add newer entry second.
	s.NetLog().Add(state.NetLogEntry{
		Timestamp:  time.Now(),
		Method:     "GET",
		Path:       "/me/player",
		StatusCode: 200,
		DurationMs: 45,
	})

	pane := panes.NewNetworkLogPane(s, th)
	pane.SetSize(120, 20)
	v := pane.View()

	// Newest entry (/me/player) should appear before older (/me/player/queue).
	playerIdx := indexInStr(v, "/me/player")
	queueIdx := indexInStr(v, "/me/player/queue")
	require.NotEqual(t, -1, playerIdx, "/me/player should be in view")
	require.NotEqual(t, -1, queueIdx, "/me/player/queue should be in view")
	assert.Less(t, playerIdx, queueIdx, "newest entry should appear first (smaller offset)")
}

// --- Color coding ---

func TestNetworkLogPane_View_Status200_InView(t *testing.T) {
	s := state.New()
	addTestEntry(s, "GET", "/me/player", 200, 45)
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(120, 20)
	v := pane.View()
	assert.Contains(t, v, "200")
	assert.Contains(t, v, "/me/player")
}

func TestNetworkLogPane_View_Status429_ShowsWarningMarker(t *testing.T) {
	s := state.New()
	addTestEntry(s, "GET", "/me/player", 429, 12)
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(120, 20)
	v := pane.View()
	assert.Contains(t, v, "429")
	assert.Contains(t, v, "⚠", "429 entries should show ⚠ in NOTES column")
}

func TestNetworkLogPane_View_Status500_InView(t *testing.T) {
	s := state.New()
	addTestEntry(s, "GET", "/me/player", 500, 100)
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(120, 20)
	v := pane.View()
	assert.Contains(t, v, "500")
}

// --- Latency bar ---

func TestNetworkLogPane_View_LatencyBar_Proportional(t *testing.T) {
	tests := []struct {
		name       string
		durationMs int64
		minBars    int
	}{
		{name: "fast (45ms)", durationMs: 45, minBars: 1},
		{name: "medium (100ms)", durationMs: 100, minBars: 4},
		{name: "max (200ms+)", durationMs: 250, minBars: 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := state.New()
			addTestEntry(s, "GET", "/me/player", 200, tt.durationMs)
			pane := panes.NewNetworkLogPane(s, theme.Load("black"))
			pane.SetSize(120, 20)
			v := pane.View()
			// Count █ characters in the view.
			count := countRune(v, '█')
			assert.GreaterOrEqual(t, count, tt.minBars, "latency bar should have at least %d bars", tt.minBars)
		})
	}
}

// --- Filter ---

func TestNetworkLogPane_Filter_ByEndpoint(t *testing.T) {
	s := state.New()
	addTestEntry(s, "GET", "/me/player", 200, 45)
	addTestEntry(s, "GET", "/me/playlists", 200, 128)
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(120, 20)
	pane.SetFocused(true)

	// Open filter.
	_, _ = pane.Update(tea_keyMsg("f"))
	// Type "/me/player" into filter.
	for _, ch := range "/me/player" {
		_, _ = pane.Update(tea_keyMsgRune(ch))
	}

	v := pane.View()
	assert.Contains(t, v, "/me/player", "filtered endpoint should be visible")
	assert.NotContains(t, v, "/me/playlists", "non-matching endpoint should be hidden")
}

func TestNetworkLogPane_Filter_ByStatusCode(t *testing.T) {
	s := state.New()
	addTestEntry(s, "GET", "/me/player", 200, 45)
	addTestEntry(s, "GET", "/me/player/queue", 429, 12)
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(120, 20)
	pane.SetFocused(true)

	// Open filter and type "429".
	_, _ = pane.Update(tea_keyMsg("f"))
	for _, ch := range "429" {
		_, _ = pane.Update(tea_keyMsgRune(ch))
	}

	v := pane.View()
	assert.Contains(t, v, "429", "429 entries should appear when filtered by status code")
}

// --- Scrolling ---

func TestNetworkLogPane_Scrolling_JKey(t *testing.T) {
	s := state.New()
	// Add enough entries to require scrolling.
	for i := 0; i < 30; i++ {
		addTestEntry(s, "GET", "/me/player", 200, int64(i+10))
	}
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(120, 15)
	pane.SetFocused(true)

	idxBefore := pane.SelectedIndex()
	_, _ = pane.Update(tea_keyMsg("j"))
	idxAfter := pane.SelectedIndex()
	assert.Greater(t, idxAfter, idxBefore, "j key should move cursor down")
}

func TestNetworkLogPane_Scrolling_KKey(t *testing.T) {
	s := state.New()
	for i := 0; i < 30; i++ {
		addTestEntry(s, "GET", "/me/player", 200, int64(i+10))
	}
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(120, 15)
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
	pane.SetSize(120, 20)
	v := pane.View()
	// Should not panic and should show the header columns.
	assert.Contains(t, v, "TIME", "empty log should still show column headers")
}

// --- Full 200-entry ring buffer ---

func TestNetworkLogPane_FullBuffer_Scrollable(t *testing.T) {
	s := state.New()
	// Fill the 200-entry ring buffer.
	for i := 0; i < 200; i++ {
		addTestEntry(s, "GET", "/me/player", 200, int64(i+10))
	}
	pane := panes.NewNetworkLogPane(s, theme.Load("black"))
	pane.SetSize(120, 20)
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
	pane.SetSize(120, 20)

	// Initially empty.
	v1 := pane.View()
	assert.NotContains(t, v1, "/me/player")

	// Add an entry.
	addTestEntry(s, "GET", "/me/player", 200, 45)

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

// countRune counts occurrences of r in s.
func countRune(s string, r rune) int {
	count := 0
	for _, c := range s {
		if c == r {
			count++
		}
	}
	return count
}
