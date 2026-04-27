package panes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestTableBasedPane builds a minimal TableBasedPane with a real Filter and
// a real Table (no columns — sufficient for filter-routing tests).
func newTestTableBasedPane(t *testing.T) *TableBasedPane {
	t.Helper()
	s := state.New()
	th := theme.Load("black")
	tbl := components.NewTable(components.TableConfig{
		Columns:    nil,
		Theme:      th,
		ShowHeader: false,
	})
	// Prime the table with enough rows so pagination works for GotoPage tests.
	tbl.SetSize(80, 5)
	rows := make([]map[string]string, 20)
	for i := range rows {
		rows[i] = map[string]string{}
	}
	tbl.SetRows(rows)

	f := components.NewFilter(th)
	return NewTableBasedPane(s, th, true, tbl, f)
}

func TestTableBasedPane_HandleFilterKey_NotConsumedForOtherKeys(t *testing.T) {
	b := newTestTableBasedPane(t)
	consumed, cmd := b.HandleFilterKey(
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")},
		func() {}, func() {},
	)
	assert.False(t, consumed)
	assert.Nil(t, cmd)
}

func TestTableBasedPane_HandleFilterKey_FActivatesFilter(t *testing.T) {
	b := newTestTableBasedPane(t)
	var resized int
	consumed, _ := b.HandleFilterKey(
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")},
		func() {}, func() { resized++ },
	)
	assert.True(t, consumed)
	assert.True(t, b.HasActiveFilter())
	assert.Equal(t, 1, resized)
}

func TestTableBasedPane_HandleFilterKey_ForwardsToFilterWhenActive(t *testing.T) {
	b := newTestTableBasedPane(t)
	b.Filter().Toggle() // activate
	var refreshed int
	b.HandleFilterKey(
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")},
		func() { refreshed++ }, func() {},
	)
	assert.Equal(t, "r", b.Filter().Query())
	assert.Equal(t, 1, refreshed)
}

func TestTableBasedPane_HandleFilterKey_EnterClosesFilterPreservesQuery(t *testing.T) {
	b := newTestTableBasedPane(t)
	b.Filter().Toggle()
	b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}, func() {}, func() {})
	b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}, func() {}, func() {})
	consumed, _ := b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyEnter}, func() {}, func() {})
	assert.True(t, consumed)
	assert.False(t, b.HasActiveFilter())
	assert.Equal(t, "ro", b.Filter().Query())
}

func TestTableBasedPane_HandleFilterKey_EscWhileActiveCancels(t *testing.T) {
	b := newTestTableBasedPane(t)
	b.Filter().Toggle()
	b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}, func() {}, func() {})
	consumed, _ := b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyEscape}, func() {}, func() {})
	assert.True(t, consumed)
	assert.False(t, b.HasActiveFilter())
	assert.Equal(t, "", b.Filter().Query())
}

func TestTableBasedPane_HandleFilterKey_EscWhenClosedClearsCommittedQuery(t *testing.T) {
	b := newTestTableBasedPane(t)
	b.Filter().Toggle()
	b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}, func() {}, func() {})
	b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyEnter}, func() {}, func() {})
	require.Equal(t, "r", b.Filter().Query())

	var refreshed int
	consumed, _ := b.HandleFilterKey(
		tea.KeyMsg{Type: tea.KeyEscape},
		func() { refreshed++ }, func() {},
	)
	assert.True(t, consumed)
	assert.Equal(t, "", b.Filter().Query())
	assert.Equal(t, 1, refreshed)
}

func TestTableBasedPane_HandleFilterKey_EscWhenClosedAndNoQueryGotoTop(t *testing.T) {
	b := newTestTableBasedPane(t)
	// Navigate to a non-first page, then verify GotoTop resets it.
	b.Table().GotoPage(2)
	consumed, _ := b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyEscape}, func() {}, func() {})
	assert.True(t, consumed)
	assert.Equal(t, 1, b.Table().CurrentPage())
}

func TestTableBasedPane_HandleFilterKey_PanicsOnNilHooks(t *testing.T) {
	b := newTestTableBasedPane(t)
	assert.Panics(t, func() {
		b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyEscape}, nil, func() {})
	})
}

func TestTableBasedPane_ActiveFilterQuery_LiveValueWhileTyping(t *testing.T) {
	b := newTestTableBasedPane(t)
	b.Filter().Toggle()
	b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}, func() {}, func() {})
	assert.Equal(t, "r", b.ActiveFilterQuery())
	b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}, func() {}, func() {})
	assert.Equal(t, "ro", b.ActiveFilterQuery())
}

// Pin the design decision that Actions() does NOT return nil while the filter
// is active with an empty query. Rationale: the border renderer evaluates
// FilterQuery before Actions; once the user types a character the filter
// label takes over the right segment. The brief active+empty-query window
// before the first keystroke harmlessly continues to render the {f, filter}
// hint — special-casing it adds branching without UX value.
func TestTableBasedPane_Actions_NotNilWhenFilterActiveWithEmptyQuery(t *testing.T) {
	b := newTestTableBasedPane(t)
	b.Filter().Toggle()
	require.True(t, b.HasActiveFilter())
	require.Equal(t, "", b.Filter().Query())

	actions := b.Actions()
	require.Len(t, actions, 1, "default Actions() returns the {f, filter} hint regardless of filter state")
	assert.Equal(t, "f", actions[0].Key)
	assert.Equal(t, "filter", actions[0].Label)
}
