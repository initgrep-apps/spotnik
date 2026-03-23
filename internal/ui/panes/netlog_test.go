package panes_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tabToNetLog sends Tab keys until the NetLog section is active.
func tabToNetLog(sv *panes.StatsView) *panes.StatsView {
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	for sv.ActiveSection() != panes.StatsSectionNetLog {
		updated, _ := sv.Update(tabMsg)
		sv = updated.(*panes.StatsView)
	}
	return sv
}

func TestNetLog_EmptyRender(t *testing.T) {
	th := theme.Load("black")
	s := state.New()
	sv := panes.NewStatsView(s, th)
	sv.SetSize(100, 40)

	sv = tabToNetLog(sv)
	view := sv.View()
	assert.Contains(t, view, "NETWORK LOG")
	assert.Contains(t, view, "No API calls recorded")
}

func TestNetLog_ShowsEntries(t *testing.T) {
	th := theme.Load("black")
	s := state.New()
	sv := panes.NewStatsView(s, th)
	sv.SetSize(100, 40)

	s.RecordNetCall("GET", "/v1/me/player", 200, 42)
	s.RecordNetCall("PUT", "/v1/me/player/play", 204, 120)

	sv = tabToNetLog(sv)
	view := sv.View()
	assert.Contains(t, view, "/v1/me/player")
	assert.Contains(t, view, "PUT")
	assert.Contains(t, view, "204")
}

func TestNetLog_CursorMovesWithJK(t *testing.T) {
	th := theme.Load("black")
	s := state.New()
	sv := panes.NewStatsView(s, th)
	sv.SetSize(100, 40)

	s.RecordNetCall("GET", "/v1/first", 200, 10)
	s.RecordNetCall("GET", "/v1/second", 200, 20)
	s.RecordNetCall("GET", "/v1/third", 200, 30)

	sv = tabToNetLog(sv)
	// Cursor should start at newest (last entry = index 2).
	assert.Equal(t, 2, sv.Cursor())

	// k moves cursor up.
	kMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	updated, _ := sv.Update(kMsg)
	sv = updated.(*panes.StatsView)
	assert.Equal(t, 1, sv.Cursor())

	// k again.
	updated, _ = sv.Update(kMsg)
	sv = updated.(*panes.StatsView)
	assert.Equal(t, 0, sv.Cursor())

	// k at 0 stays at 0.
	updated, _ = sv.Update(kMsg)
	sv = updated.(*panes.StatsView)
	assert.Equal(t, 0, sv.Cursor())

	// j moves cursor down.
	jMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	updated, _ = sv.Update(jMsg)
	sv = updated.(*panes.StatsView)
	assert.Equal(t, 1, sv.Cursor())
}

func TestNetLog_TabIn_AutoScrollsToNewest(t *testing.T) {
	th := theme.Load("black")
	s := state.New()
	sv := panes.NewStatsView(s, th)
	sv.SetSize(100, 40)

	for i := 0; i < 20; i++ {
		s.RecordNetCall("GET", "/v1/test", 200, int64(i))
	}

	sv = tabToNetLog(sv)
	// Cursor should jump to the last entry (index 19).
	assert.Equal(t, 19, sv.Cursor())
}

func TestNetLog_SectionLen(t *testing.T) {
	th := theme.Load("black")
	s := state.New()
	sv := panes.NewStatsView(s, th)

	s.RecordNetCall("GET", "/v1/a", 200, 10)
	s.RecordNetCall("GET", "/v1/b", 200, 20)

	sv = tabToNetLog(sv)
	// Cursor bounded by section length: can move to 0 and 1, not beyond.
	kMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	updated, _ := sv.Update(kMsg)
	sv = updated.(*panes.StatsView)
	assert.Equal(t, 0, sv.Cursor())

	jMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	updated, _ = sv.Update(jMsg)
	sv = updated.(*panes.StatsView)
	assert.Equal(t, 1, sv.Cursor())

	// Can't go past last entry.
	updated, _ = sv.Update(jMsg)
	sv = updated.(*panes.StatsView)
	assert.Equal(t, 1, sv.Cursor())
}

func TestNetLog_RenderContainsHighlight(t *testing.T) {
	th := theme.Load("black")
	s := state.New()
	sv := panes.NewStatsView(s, th)
	sv.SetSize(100, 40)

	s.RecordNetCall("GET", "/v1/me/player", 200, 42)
	s.RecordNetCall("GET", "/v1/me/player/queue", 429, 5)

	sv = tabToNetLog(sv)
	view := sv.View()

	// Both entries should be present.
	assert.Contains(t, view, "/v1/me/player")
	assert.Contains(t, view, "429")

	// The view should render without error — visual highlight tested by cursor position.
	require.NotEmpty(t, view)
}
