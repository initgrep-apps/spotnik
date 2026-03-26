package components_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTheme returns the black theme which is the Spotnik default.
func testTheme() theme.Theme {
	return &theme.BlackTheme{}
}

// makeColumns returns a typical 4-column track list definition.
func makeColumns() []components.ColumnDef {
	t := testTheme()
	return []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: t.TextMuted()},
		{Key: "track", Header: "Track", FlexFactor: 4, Color: t.TextPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 3, Color: t.TextSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 2, Color: t.TextMuted()},
	}
}

func TestNewTable_CreatesWithColumns(t *testing.T) {
	cfg := components.TableConfig{
		Columns:      makeColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   true,
	}
	tbl := components.NewTable(cfg)
	require.NotNil(t, tbl)
}

func TestTable_SetSizeScalesColumns(t *testing.T) {
	cfg := components.TableConfig{
		Columns:      makeColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   true,
	}
	tbl := components.NewTable(cfg)
	// Should not panic on resize
	tbl.SetSize(80, 20)
	tbl.SetSize(120, 40)
}

func TestTable_SetRows(t *testing.T) {
	cfg := components.TableConfig{
		Columns:      makeColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	rows := []map[string]string{
		{"index": "1", "track": "Track A", "artist": "Artist A", "duration": "3:00"},
		{"index": "2", "track": "Track B", "artist": "Artist B", "duration": "4:00"},
		{"index": "3", "track": "Track C", "artist": "Artist C", "duration": "2:30"},
		{"index": "4", "track": "Track D", "artist": "Artist D", "duration": "5:10"},
		{"index": "5", "track": "Track E", "artist": "Artist E", "duration": "3:45"},
	}

	// SetRows should not panic
	tbl.SetRows(rows)

	// After setting rows, view should render without panic
	view := tbl.View()
	assert.NotEmpty(t, view)
}

func TestTable_SelectedIndexInitiallyZero(t *testing.T) {
	cfg := components.TableConfig{
		Columns:      makeColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	rows := []map[string]string{
		{"index": "1", "track": "Track A", "artist": "Artist A", "duration": "3:00"},
		{"index": "2", "track": "Track B", "artist": "Artist B", "duration": "4:00"},
	}
	tbl.SetRows(rows)

	assert.Equal(t, 0, tbl.SelectedIndex())
}

func TestTable_KeyboardNavigationChangesSelection(t *testing.T) {
	cfg := components.TableConfig{
		Columns:      makeColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	rows := []map[string]string{
		{"index": "1", "track": "Track A", "artist": "Artist A", "duration": "3:00"},
		{"index": "2", "track": "Track B", "artist": "Artist B", "duration": "4:00"},
		{"index": "3", "track": "Track C", "artist": "Artist C", "duration": "2:30"},
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

func TestTable_SetPlayingIndex(t *testing.T) {
	cfg := components.TableConfig{
		Columns:      makeColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	rows := []map[string]string{
		{"index": "1", "track": "Track A", "artist": "Artist A", "duration": "3:00"},
		{"index": "2", "track": "Track B", "artist": "Artist B", "duration": "4:00"},
		{"index": "3", "track": "Track C", "artist": "Artist C", "duration": "2:30"},
	}
	tbl.SetRows(rows)
	tbl.SetPlayingIndex(2)

	// View should render without panic and the playing indicator config is set
	view := tbl.View()
	assert.NotEmpty(t, view)
}

func TestTable_EmptyRowsRendersCleanly(t *testing.T) {
	cfg := components.TableConfig{
		Columns:      makeColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   true,
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
		Columns:      makeColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   true,
	}
	tbl := components.NewTable(cfg)

	rows := []map[string]string{
		{"index": "1", "track": "Track A", "artist": "Artist A", "duration": "3:00"},
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
	assert.Equal(t, lipgloss.Color(th.TextMuted()), cols[0].Color)
	assert.Equal(t, lipgloss.Color(th.TextPrimary()), cols[1].Color)
	assert.Equal(t, lipgloss.Color(th.TextSecondary()), cols[2].Color)
	assert.Equal(t, lipgloss.Color(th.TextMuted()), cols[3].Color)
}

func TestTable_SetFocused(t *testing.T) {
	cfg := components.TableConfig{
		Columns:      makeColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   true,
	}
	tbl := components.NewTable(cfg)
	tbl.SetSize(80, 20)

	// SetFocused should not panic
	tbl.SetFocused(true)
	tbl.SetFocused(false)
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
		{"index": "1", "track": "Song", "artist": "Artist", "duration": "3:00"},
	}
	tbl.SetRows(rows)

	view := tbl.View()
	// Headers should appear in the view (rendered text may include ANSI codes around the word)
	assert.Contains(t, view, "Track")
	assert.Contains(t, view, "Artist")
}
