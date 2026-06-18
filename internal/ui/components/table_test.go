package components_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTheme returns the black theme which is the Spotnik default.
func testTheme() theme.Theme {
	return theme.Load("black")
}

// makeColumns returns a typical 3-column track list definition (no index column).
func makeColumns() []components.ColumnDef {
	t := testTheme()
	return []components.ColumnDef{
		{Key: "track", Header: "Track", FlexFactor: 4, Color: t.TextPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 3, Color: t.TextSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 2, Color: t.TextMuted()},
	}
}

func TestNewTable_CreatesWithColumns(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	require.NotNil(t, tbl)
}

func TestTable_SetSizeScalesColumns(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	// Should not panic on resize
	tbl.SetSize(80, 20)
	tbl.SetSize(120, 40)
}

func TestTable_SetRows(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	rows := []map[string]string{
		{"track": "Track A", "artist": "Artist A", "duration": "3:00"},
		{"track": "Track B", "artist": "Artist B", "duration": "4:00"},
		{"track": "Track C", "artist": "Artist C", "duration": "2:30"},
		{"track": "Track D", "artist": "Artist D", "duration": "5:10"},
		{"track": "Track E", "artist": "Artist E", "duration": "3:45"},
	}

	// SetRows should not panic
	tbl.SetRows(rows)

	// After setting rows, view should render without panic
	view := tbl.View()
	assert.NotEmpty(t, view)
}

func TestTable_SelectedIndexInitiallyZero(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	rows := []map[string]string{
		{"track": "Track A", "artist": "Artist A", "duration": "3:00"},
		{"track": "Track B", "artist": "Artist B", "duration": "4:00"},
	}
	tbl.SetRows(rows)

	assert.Equal(t, 0, tbl.SelectedIndex())
}

func TestTable_KeyboardNavigationChangesSelection(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	rows := []map[string]string{
		{"track": "Track A", "artist": "Artist A", "duration": "3:00"},
		{"track": "Track B", "artist": "Artist B", "duration": "4:00"},
		{"track": "Track C", "artist": "Artist C", "duration": "2:30"},
	}
	tbl.SetRows(rows)
	tbl.SetFocused(true)

	// Press down key
	tbl.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, tbl.SelectedIndex())

	// Press down again
	tbl.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, tbl.SelectedIndex())

	// Press up key
	tbl.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, tbl.SelectedIndex())
}

func TestTable_EmptyRowsRendersCleanly(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)
	tbl.SetRows([]map[string]string{})

	// Should not panic
	view := tbl.View()
	assert.NotNil(t, view)
}

func TestTable_WidthRecalculatedAfterSetSize(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)

	rows := []map[string]string{
		{"track": "Track A", "artist": "Artist A", "duration": "3:00"},
	}
	tbl.SetRows(rows)

	// Resize to small width
	tbl.SetSize(40, 10)
	view40 := tbl.View()

	// Resize to larger width
	tbl.SetSize(100, 20)
	view100 := tbl.View()

	// Views at different widths should differ
	assert.NotEqual(t, view40, view100)
}

func TestTable_ColumnDefsHaveCorrectColors(t *testing.T) {
	th := testTheme()
	cols := makeColumns()

	// Verify column colors are set from theme
	assert.Equal(t, lipgloss.Color(th.TextPrimary()), cols[0].Color)
	assert.Equal(t, lipgloss.Color(th.TextSecondary()), cols[1].Color)
	assert.Equal(t, lipgloss.Color(th.TextMuted()), cols[2].Color)
}

func TestTable_SetFocused(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	// SetFocused should not panic
	tbl.SetFocused(true)
	tbl.SetFocused(false)
}

func TestTable_GotoTop_ResetsToFirstPage(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	// pageSize = height - 4 (header + borders + padding) with ShowHeader=true.
	// height=11 → pageSize=5; 20 rows across 5-row pages = 4 pages.
	tbl.SetSize(80, 11)
	tbl.SetFocused(true)

	rows := make([]map[string]string, 20)
	for i := range rows {
		rows[i] = map[string]string{
			"track": "T", "artist": "A", "duration": "1:00",
		}
	}
	tbl.SetRows(rows)

	// Scroll 8 rows down to move onto page 2+.
	for i := 0; i < 8; i++ {
		tbl.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	require.Greater(t, tbl.CurrentPage(), 1, "should have scrolled past page 1")

	tbl.GotoTop()
	assert.Equal(t, 1, tbl.CurrentPage(), "GotoTop should reset to page 1")
}

func TestTable_ViewRendersHeader(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	rows := []map[string]string{
		{"track": "Song", "artist": "Artist", "duration": "3:00"},
	}
	tbl.SetRows(rows)

	view := tbl.View()
	// Headers should appear in the view (rendered text may include ANSI codes around the word)
	assert.Contains(t, view, "Track")
	assert.Contains(t, view, "Artist")
}

// makePriorityColumns returns a 3-column layout with mixed priorities:
//
//	trk: Priority 1 (always), art: Priority 2 (≥40 cols), dur: Priority 3 (≥60 cols).
func makePriorityColumns() []components.ColumnDef {
	th := testTheme()
	return []components.ColumnDef{
		{Key: "trk", Header: "Track", FlexFactor: 4, Color: th.TextPrimary(), Priority: 1},
		{Key: "art", Header: "Artist", FlexFactor: 3, Color: th.TextSecondary(), Priority: 2},
		{Key: "dur", Header: "Dur", FlexFactor: 2, Color: th.TextMuted(), Priority: 3},
	}
}

func TestTable_PriorityFiltering_HidesColumnsAtNarrowWidth(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makePriorityColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(30, 20) // below 40 — only priority-1 cols

	cols := tbl.Columns()
	assert.Len(t, cols, 1, "only Priority-1 columns should be visible at width 30")
	assert.Equal(t, "trk", cols[0].Key)
}

func TestTable_PriorityFiltering_MediumWidthShowsPriority2(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makePriorityColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(50, 20) // >=40, <60 — priority 1+2

	cols := tbl.Columns()
	assert.Len(t, cols, 2, "Priority-1 and Priority-2 columns should be visible at width 50")
	keys := make([]string, len(cols))
	for i, c := range cols {
		keys[i] = c.Key
	}
	assert.Contains(t, keys, "trk")
	assert.Contains(t, keys, "art")
	assert.NotContains(t, keys, "dur")
}

func TestTable_PriorityFiltering_WideWidthShowsAll(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makePriorityColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20) // >=60 — all columns

	cols := tbl.Columns()
	assert.Len(t, cols, 3, "all columns should be visible at width 80 (>=60)")
	keys := make([]string, len(cols))
	for i, c := range cols {
		keys[i] = c.Key
	}
	assert.Contains(t, keys, "trk")
	assert.Contains(t, keys, "art")
	assert.Contains(t, keys, "dur")
}

func TestTable_PriorityFiltering_WidthThresholdCrossing(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makePriorityColumns(),
		Theme:      testTheme(),
		ShowHeader: true,
	}
	tbl := components.NewTable(cfg)

	// Start narrow: only trk.
	tbl.SetSize(30, 20)
	assert.Len(t, tbl.Columns(), 1)

	// Cross 40: trk + art appear.
	tbl.SetSize(45, 20)
	assert.Len(t, tbl.Columns(), 2)

	// Cross 60: all three appear.
	tbl.SetSize(70, 20)
	assert.Len(t, tbl.Columns(), 3)

	// Drop back below 60: dur hides.
	tbl.SetSize(55, 20)
	assert.Len(t, tbl.Columns(), 2)

	// Drop below 40: art hides too.
	tbl.SetSize(35, 20)
	assert.Len(t, tbl.Columns(), 1)
}

// makeRichColumns returns a two-column layout matching GatewayLivePane's design.
func makeRichColumns() []components.ColumnDef {
	th := testTheme()
	return []components.ColumnDef{
		{Key: "glyph", Header: "", FlexFactor: 1, Color: th.TextPrimary()},
		{Key: "event", Header: "", FlexFactor: 30, Color: th.ColumnPrimary()},
	}
}

// TestTable_SetRichRows_PlainStringCellsRender verifies that a []map[string]any with
// plain string values for both columns renders the expected text in View().
func TestTable_SetRichRows_PlainStringCellsRender(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeRichColumns(),
		Theme:      testTheme(),
		ShowHeader: false,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	rows := []map[string]any{
		{"glyph": "✓", "event": "Request allowed"},
		{"glyph": "✗", "event": "Request blocked"},
	}
	tbl.SetRichRows(rows)

	view := tbl.View()
	assert.Contains(t, view, "Request allowed", "first event label must appear in view")
	assert.Contains(t, view, "Request blocked", "second event label must appear in view")
}

// TestTable_SetRichRows_StyledCellAppliesForeground verifies that a btable.StyledCell
// in the glyph column renders with its foreground ANSI SGR sequence.
// TrueColor is forced per-test so ANSI codes are emitted without breaking other
// tests (e.g. InfoBox centering tests that rely on no-color lipgloss measurements).
func TestTable_SetRichRows_StyledCellAppliesForeground(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	cfg := components.TableConfig{
		Columns:    makeRichColumns(),
		Theme:      testTheme(),
		ShowHeader: false,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	styledGlyph := btable.NewStyledCell("✓", redStyle)

	rows := []map[string]any{
		{"glyph": styledGlyph, "event": "styled event"},
	}
	tbl.SetRichRows(rows)

	view := tbl.View()
	// The red foreground should produce an ANSI 38 (set foreground colour) sequence.
	assert.Contains(t, view, "\x1b[38", "styled cell must produce ANSI foreground escape in rendered output")
	assert.Contains(t, view, "styled event", "event column label must appear in view")
}

// TestTable_SetRichRows_DoesNotAffectExistingSetRows verifies that calling SetRichRows
// after SetRows replaces the data and no leftover string rows appear.
func TestTable_SetRichRows_DoesNotAffectExistingSetRows(t *testing.T) {
	cfg := components.TableConfig{
		Columns:    makeRichColumns(),
		Theme:      testTheme(),
		ShowHeader: false,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	// First load plain string rows via SetRows.
	tbl.SetRows([]map[string]string{
		{"glyph": "A", "event": "old plain row"},
	})
	viewBefore := tbl.View()
	assert.Contains(t, viewBefore, "old plain row", "SetRows data must appear before SetRichRows")

	// Now replace with rich rows via SetRichRows.
	tbl.SetRichRows([]map[string]any{
		{"glyph": "B", "event": "new rich row"},
	})
	viewAfter := tbl.View()
	assert.Contains(t, viewAfter, "new rich row", "SetRichRows data must appear after call")
	assert.NotContains(t, viewAfter, "old plain row", "SetRows data must not appear after SetRichRows replaces it")
}
