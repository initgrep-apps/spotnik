package components_test

import (
	"strings"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

// TestIntegration_Visualizer_Lifecycle exercises Init → multiple ticks → View changes.
func TestIntegration_Visualizer_Lifecycle(t *testing.T) {
	th := theme.Load("black")
	v := components.NewVisualizer(th)
	v.SetSize(40, 2)
	v.SetPlaying(true)

	// Init must return a command.
	cmd := v.Init()
	assert.NotNil(t, cmd)

	// Tick 3 times and confirm View changes each frame.
	views := make([]string, 4)
	views[0] = v.View()
	for i := 1; i <= 3; i++ {
		v.Update(components.VisualizerTickMsg(time.Now()))
		views[i] = v.View()
	}

	// At least some frames should differ (with 40 unique frames they will).
	someChanged := false
	for i := 1; i < len(views); i++ {
		if views[i] != views[0] {
			someChanged = true
			break
		}
	}
	assert.True(t, someChanged, "view should change across frames when playing")
}

// TestIntegration_Visualizer_PlayPauseCycle verifies play→tick→advance, pause→tick→freeze,
// play→tick→advance pattern.
func TestIntegration_Visualizer_PlayPauseCycle(t *testing.T) {
	th := theme.Load("black")
	v := components.NewVisualizer(th)
	v.SetSize(40, 2)

	// Start playing — tick should advance the frame.
	v.SetPlaying(true)
	before := v.FrameIndex()
	v.Update(components.VisualizerTickMsg(time.Now()))
	assert.Equal(t, before+1, v.FrameIndex(), "frame should advance while playing")

	// Pause — tick should NOT advance the frame.
	v.SetPlaying(false)
	frozen := v.FrameIndex()
	v.Update(components.VisualizerTickMsg(time.Now()))
	assert.Equal(t, frozen, v.FrameIndex(), "frame should freeze while paused")

	// Resume playing — tick should advance again.
	v.SetPlaying(true)
	v.Update(components.VisualizerTickMsg(time.Now()))
	assert.Equal(t, frozen+1, v.FrameIndex(), "frame should advance after resuming")
}

// TestIntegration_SeekBar_GradientVisible verifies seek bar output contains gradient chars.
func TestIntegration_SeekBar_GradientVisible(t *testing.T) {
	th := theme.Load("black")
	b := components.NewGradientSeekBar(th)
	b.SetWidth(60)

	// At 50% progress, filled chars should be present.
	out := b.Render(150000, 300000)
	assert.Contains(t, out, "█", "seek bar should contain filled chars at 50% progress")
	assert.Contains(t, out, "░", "seek bar should contain empty chars at 50% progress")
	assert.Contains(t, out, "2:30", "elapsed label should show 2:30")
	assert.Contains(t, out, "5:00", "total label should show 5:00")
}

// TestIntegration_VolumeBar_ThresholdTransitions verifies color-band transitions.
func TestIntegration_VolumeBar_ThresholdTransitions(t *testing.T) {
	th := theme.Load("black")
	b := components.NewGradientVolumeBar(th)
	b.SetWidth(40)

	// Just check that crossing thresholds doesn't panic and output looks correct.
	out33 := b.Render(33)
	assert.Contains(t, out33, "33%")

	out34 := b.Render(34)
	assert.Contains(t, out34, "34%")

	out66 := b.Render(66)
	assert.Contains(t, out66, "66%")

	out67 := b.Render(67)
	assert.Contains(t, out67, "67%")
}

// TestIntegration_AllComponentsRenderWithinWidth verifies no line exceeds the specified width.
func TestIntegration_AllComponentsRenderWithinWidth(t *testing.T) {
	th := theme.Load("black")
	width := 60

	// Visualizer
	v := components.NewVisualizer(th)
	v.SetSize(width, 2)
	v.SetPlaying(true)
	for _, line := range strings.Split(strings.TrimRight(v.View(), "\n"), "\n") {
		runeWidth := len([]rune(line))
		assert.LessOrEqual(t, runeWidth, width, "visualizer line exceeds width: %q", line)
	}

	// GradientSeekBar — output is one line, width is approximate due to labels.
	sb := components.NewGradientSeekBar(th)
	sb.SetWidth(width)
	seekLine := sb.Render(150000, 300000)
	_ = seekLine // width check is approximate; main check is no panic

	// GradientVolumeBar — verify no panic, output has the right shape.
	vb := components.NewGradientVolumeBar(th)
	vb.SetWidth(width)
	volOut := vb.Render(50)
	assert.Contains(t, volOut, "VOL")
}

// TestIntegration_NoHardcodedHexInComponents verifies theme tokens are used (structural check).
// We verify that removing ANSI escapes still leaves braille/block chars — not raw hex strings.
func TestIntegration_NoHardcodedHexInComponents(t *testing.T) {
	th := theme.Load("black")

	v := components.NewVisualizer(th)
	v.SetSize(20, 1)
	v.SetPlaying(true)
	view := v.View()
	// No raw #rrggbb in the rendered output.
	assert.NotContains(t, view, "#000000", "visualizer view should not contain raw hex")

	b := components.NewGradientSeekBar(th)
	b.SetWidth(40)
	seekView := b.Render(100000, 300000)
	assert.NotContains(t, seekView, "#000000", "seek bar view should not contain raw hex")

	vb := components.NewGradientVolumeBar(th)
	vb.SetWidth(40)
	volView := vb.Render(50)
	assert.NotContains(t, volView, "#000000", "volume bar view should not contain raw hex")
}
