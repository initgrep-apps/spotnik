package components

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func newTestProgressBar(width int) ProgressBar {
	t := theme.Load("black")
	return NewProgressBar(width, t)
}

func TestProgressBar_ZeroProgress(t *testing.T) {
	pb := newTestProgressBar(40)
	output := pb.Render(0, 240000)

	// At zero progress, bar should contain no filled chars.
	assert.NotContains(t, output, "█", "zero progress should show no filled chars")
}

func TestProgressBar_HalfProgress(t *testing.T) {
	pb := newTestProgressBar(40)
	// 120 seconds out of 240 seconds = exactly half.
	output := pb.Render(120000, 240000)

	// Should contain both filled and empty characters.
	assert.Contains(t, output, "█")
	assert.Contains(t, output, "░")
}

func TestProgressBar_FullProgress(t *testing.T) {
	pb := newTestProgressBar(40)
	output := pb.Render(240000, 240000)

	assert.NotContains(t, output, "░", "full progress should show no empty chars")
}

func TestProgressBar_TimeLabels(t *testing.T) {
	pb := newTestProgressBar(50)
	// 2:34 = 154000ms; 4:12 = 252000ms
	output := pb.Render(154000, 252000)

	assert.Contains(t, output, "2:34", "should show elapsed time")
	assert.Contains(t, output, "4:12", "should show total time")
}

func TestProgressBar_ZeroDuration(t *testing.T) {
	pb := newTestProgressBar(40)
	// Should not panic on zero duration.
	output := pb.Render(0, 0)

	assert.NotContains(t, output, "NaN")
	// Time labels should show 0:00 for both.
	assert.Contains(t, output, "0:00")
}

func TestProgressBar_WidthAdapts(t *testing.T) {
	t40 := theme.Load("black")
	t80 := theme.Load("black")

	pb40 := NewProgressBar(40, t40)
	pb80 := NewProgressBar(80, t80)

	out40 := pb40.Render(120000, 240000)
	out80 := pb80.Render(120000, 240000)

	// Wider bar should produce longer output.
	assert.Greater(t, len(out80), len(out40), "wider bar should produce more characters")
}

func TestProgressBar_TimeLabelFormat(t *testing.T) {
	pb := newTestProgressBar(40)
	// 61 seconds = 1:01 (single-digit seconds should be zero-padded)
	output := pb.Render(61000, 120000)
	assert.Contains(t, output, "1:01")
}

// TestProgressBar_OutputContainsBothLines checks that Render returns
// both the bar line and the time label line.
func TestProgressBar_OutputContainsBothLines(t *testing.T) {
	pb := newTestProgressBar(40)
	output := pb.Render(60000, 120000)

	lines := strings.Split(output, "\n")
	assert.GreaterOrEqual(t, len(lines), 2, "output should have at least 2 lines")
}
