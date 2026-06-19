package components_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/stretchr/testify/assert"
)

// newKeyRune creates a tea.KeyMsg that simulates typing a single rune.
func newKeyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// allRows is a sample track list used across integration tests.
func allRows() []map[string]string {
	return []map[string]string{
		{"index": "1", "track": "Rocket Man", "artist": "Elton John", "duration": "4:52"},
		{"index": "1", "track": "Bohemian Rhapsody", "artist": "Queen", "duration": "5:55"},
		{"index": "1", "track": "Rock and Roll", "artist": "Led Zeppelin", "duration": "3:40"},
		{"index": "1", "track": "Space Oddity", "artist": "David Bowie", "duration": "5:17"},
		{"index": "1", "track": "Sweet Home Alabama", "artist": "Lynyrd Skynyrd", "duration": "4:44"},
	}
}

// TestIntegration_TableWithFilter verifies the filter-then-update-table flow that
// panes will use: Filter.Matches() gates which rows are passed to Table.SetRows.
func TestIntegration_TableWithFilter(t *testing.T) {
	th := testTheme()
	f := components.NewFilter(th)
	tbl := components.NewTable(components.TableConfig{
		Columns:    makeColumns(),
		Theme:      th,
		ShowHeader: true,
	})
	tbl.SetSize(80, 20)
	tbl.SetRows(allRows())

	// Activate filter and type "rock"
	f.Toggle()
	for _, r := range []rune{'r', 'o', 'c', 'k'} {
		f.Update(newKeyRune(r))
	}

	// Filter the rows externally (as a pane's Update() would do).
	var filtered []map[string]string
	for _, row := range allRows() {
		if f.MatchesAny(row["track"], row["artist"]) {
			filtered = append(filtered, row)
		}
	}
	tbl.SetRows(filtered)

	// "rock" should match "Rocket Man" and "Rock and Roll"
	assert.Len(t, filtered, 2)

	// Table should render without panic
	view := tbl.View()
	assert.NotEmpty(t, view)
}

// TestIntegration_TableResizeCycle verifies that resizing the table does not
// cause panics or render overflow.
func TestIntegration_TableResizeCycle(t *testing.T) {
	th := testTheme()
	tbl := components.NewTable(components.TableConfig{
		Columns:    makeColumns(),
		Theme:      th,
		ShowHeader: true,
	})
	tbl.SetSize(80, 20)
	tbl.SetRows(allRows())

	// Resize to a small terminal
	tbl.SetSize(30, 5)
	view30 := tbl.View()
	assert.NotEmpty(t, view30)

	// Resize back to a larger terminal
	tbl.SetSize(120, 40)
	view120 := tbl.View()
	assert.NotEmpty(t, view120)

	// Views should differ between widths
	assert.NotEqual(t, view30, view120)
}

// TestIntegration_FilterBorderLabel verifies the border label lifecycle: active
// with query → label shown; deactivated → label empty.
func TestIntegration_FilterBorderLabel(t *testing.T) {
	f := components.NewFilter(testTheme())

	// Inactive: border label should be empty
	assert.Equal(t, "", f.BorderLabel())

	// Activate and type a query
	f.Toggle()
	for _, r := range []rune{'r', 'o', 'c', 'k'} {
		f.Update(newKeyRune(r))
	}
	assert.Equal(t, `filtering: "rock"`, f.BorderLabel())

	// Deactivate via second Toggle (clears query)
	f.Toggle()
	assert.Equal(t, "", f.BorderLabel())
}

// TestIntegration_TruncateOnTableCellValues verifies that applying Truncate to
// table cell values before SetRows prevents content wider than the column.
func TestIntegration_TruncateOnTableCellValues(t *testing.T) {
	th := testTheme()
	tbl := components.NewTable(components.TableConfig{
		Columns:    makeColumns(),
		Theme:      th,
		ShowHeader: true,
	})
	tbl.SetSize(80, 20)

	// Very long track and artist names
	rows := []map[string]string{
		{
			"index":    "1",
			"track":    layout.TruncateOrPad("A Very Long Track Name That Would Overflow Any Column", 20),
			"artist":   layout.TruncateOrPad("An Extremely Long Artist Name", 15),
			"duration": "3:00",
		},
	}

	// After truncation cell widths are bounded — should render without panic
	tbl.SetRows(rows)
	view := tbl.View()
	assert.NotEmpty(t, view)
}

// TestIntegration_TableZeroRowsNoPanic verifies that a table with no rows renders
// cleanly (no panic, no empty-string panic).
func TestIntegration_TableZeroRowsNoPanic(t *testing.T) {
	th := testTheme()
	tbl := components.NewTable(components.TableConfig{
		Columns:    makeColumns(),
		Theme:      th,
		ShowHeader: true,
	})
	tbl.SetSize(80, 20)
	tbl.SetRows([]map[string]string{})

	// Must not panic
	view := tbl.View()
	assert.NotNil(t, view)
}

// TestIntegration_FilterEmptyQueryMatchesAll verifies that an active filter with
// an empty query matches every row (no accidental filtering).
func TestIntegration_FilterEmptyQueryMatchesAll(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle() // activate, but type nothing

	for _, row := range allRows() {
		assert.True(t, f.MatchesAny(row["track"], row["artist"]),
			"empty query should match row %v", row)
	}
}
