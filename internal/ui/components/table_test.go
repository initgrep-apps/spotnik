package components_test

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTheme returns the black theme which is the Spotnik default.
func testTheme() theme.Theme {
	return theme.Load("black")
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

func TestTable_GotoTop_ResetsToFirstPage(t *testing.T) {
	cfg := components.TableConfig{
		Columns:      makeColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   true,
	}
	tbl := components.NewTable(cfg)
	// pageSize = height - 6 (header + borders + padding) with ShowHeader=true.
	// height=11 → pageSize=5; 20 rows across 5-row pages = 4 pages.
	tbl.SetSize(80, 11)
	tbl.SetFocused(true)

	rows := make([]map[string]string, 20)
	for i := range rows {
		rows[i] = map[string]string{
			"index": fmt.Sprintf("%d", i+1), "track": "T", "artist": "A", "duration": "1:00",
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
		{"index": "1", "track": "Song", "artist": "Artist", "duration": "3:00"},
	}
	tbl.SetRows(rows)

	view := tbl.View()
	// Headers should appear in the view (rendered text may include ANSI codes around the word)
	assert.Contains(t, view, "Track")
	assert.Contains(t, view, "Artist")
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
		Columns:      makeRichColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   false,
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
		Columns:      makeRichColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   false,
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
		Columns:      makeRichColumns(),
		Theme:        testTheme(),
		PlayingIndex: -1,
		ShowHeader:   false,
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

// TestTable_PlayingSymbol_AsciiMode verifies that when the uikit glyph mode is
// ASCII the playing indicator renders as ">" (GlyphPlaying ASCII form) and NOT
// as the unicode "▶". Both the plain-rows and rich-rows paths in applyRows call
// playingSymbol(), so we exercise them here.
func TestTable_PlayingSymbol_AsciiMode(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	cols := makeColumns() // uses "index", "track", "artist", "duration"

	t.Run("plain rows", func(t *testing.T) {
		cfg := components.TableConfig{
			Columns:      cols,
			Theme:        testTheme(),
			PlayingIndex: 0, // first row is playing
			ShowHeader:   false,
		}
		tbl := components.NewTable(cfg)
		tbl.SetSize(120, 20)
		tbl.SetRows([]map[string]string{
			{"index": "1", "track": "Song A", "artist": "Artist A", "duration": "3:30"},
			{"index": "2", "track": "Song B", "artist": "Artist B", "duration": "4:00"},
		})

		out := tbl.View()
		assert.NotContains(t, out, "▶", "unicode playing glyph ▶ must not appear in ASCII mode")
		assert.True(t, strings.Contains(out, ">"), "ASCII playing glyph '>' must appear in ASCII mode")
	})

	t.Run("rich rows", func(t *testing.T) {
		richCols := []components.ColumnDef{
			{Key: "index", Header: "#", FlexFactor: 1},
			{Key: "track", Header: "Track", FlexFactor: 4},
		}
		cfg := components.TableConfig{
			Columns:      richCols,
			Theme:        testTheme(),
			PlayingIndex: 0,
			ShowHeader:   false,
		}
		tbl := components.NewTable(cfg)
		tbl.SetSize(120, 20)
		tbl.SetRichRows([]map[string]any{
			{"index": "1", "track": "Rich Song A"},
			{"index": "2", "track": "Rich Song B"},
		})

		out := tbl.View()
		assert.NotContains(t, out, "▶", "unicode playing glyph ▶ must not appear in ASCII mode (rich rows)")
		assert.True(t, strings.Contains(out, ">"), "ASCII playing glyph '>' must appear in ASCII mode (rich rows)")
	})
}
