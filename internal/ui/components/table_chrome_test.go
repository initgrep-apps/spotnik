package components_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

// TestTableChrome_WrapsComponentsTable verifies that Inner() returns a non-nil
// *Table both on first call (nil-inner path) and on subsequent calls
// (primed-inner path), keeping 100% coverage of the lazy-construction branches.
func TestTableChrome_WrapsComponentsTable(t *testing.T) {
	th := theme.Load("black")
	cols := []components.ColumnDef{
		{Key: "name", Header: "Name", FlexFactor: 5, Color: th.ColumnPrimary()},
		{Key: "artist", Header: "Artist", FlexFactor: 3, Color: th.ColumnSecondary()},
	}
	tbl := components.TableChrome{Columns: cols, Theme: th}

	// Nil-inner path: Inner() must construct and return a non-nil table.
	first := tbl.Inner()
	assert.NotNil(t, first, "Inner() must return a non-nil *Table on first call")

	// Primed-inner path: subsequent calls must return the same instance.
	second := tbl.Inner()
	assert.Same(t, first, second, "Inner() must return the same *Table on subsequent calls")
}
