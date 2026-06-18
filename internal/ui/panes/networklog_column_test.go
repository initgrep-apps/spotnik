package panes

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Story 71 Task 5: column color tokens ─────────────────────────────────────

// TestNetworkLogPane_UsesColumnColors verifies that NetworkLogPane column definitions
// use the new column color tokens instead of TextMuted/TextPrimary/TextSecondary.
func TestNetworkLogPane_UsesColumnColors(t *testing.T) {
	th := theme.Load("black")
	p := NewNetworkLogPane(state.New(), th)
	p.SetSize(80, 20)
	cols := p.table.Columns()
	require.Len(t, cols, 7, "NetworkLogPane should have 7 columns")

	// TIME: index-like timestamp → ColumnIndex
	assert.Equal(t, th.ColumnIndex(), cols[0].Color, "TIME column should use ColumnIndex()")
	// METHOD: supporting context → ColumnSecondary
	assert.Equal(t, th.ColumnSecondary(), cols[1].Color, "METHOD column should use ColumnSecondary()")
	// ENDPOINT: main data column → ColumnPrimary
	assert.Equal(t, th.ColumnPrimary(), cols[2].Color, "ENDPOINT column should use ColumnPrimary()")
	// STATUS: metadata → ColumnTertiary
	assert.Equal(t, th.ColumnTertiary(), cols[3].Color, "STATUS column should use ColumnTertiary()")
	// LATENCY: metadata → ColumnTertiary
	assert.Equal(t, th.ColumnTertiary(), cols[4].Color, "LATENCY column should use ColumnTertiary()")
	// PRIORITY: metadata label → ColumnIndex
	assert.Equal(t, th.ColumnIndex(), cols[5].Color, "PRIORITY column should use ColumnIndex()")
	// DECISION: supporting result → ColumnSecondary
	assert.Equal(t, th.ColumnSecondary(), cols[6].Color, "DECISION column should use ColumnSecondary()")
}
