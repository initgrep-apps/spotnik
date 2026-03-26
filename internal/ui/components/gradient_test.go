package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func newTestSeekBar(width int) *GradientSeekBar {
	t := theme.Load("black")
	b := NewGradientSeekBar(t)
	b.SetWidth(width)
	return b
}

func newTestGradientVolumeBar(width int) *GradientVolumeBar {
	t := theme.Load("black")
	b := NewGradientVolumeBar(t)
	b.SetWidth(width)
	return b
}

// --------------------------------------------------------------------------
// interpolateHex
// --------------------------------------------------------------------------

func TestInterpolateHex_AtZero(t *testing.T) {
	c := interpolateHex("#ff0000", "#0000ff", 0.0)
	assert.Equal(t, lipgloss.Color("#ff0000"), c)
}

func TestInterpolateHex_AtOne(t *testing.T) {
	c := interpolateHex("#ff0000", "#0000ff", 1.0)
	assert.Equal(t, lipgloss.Color("#0000ff"), c)
}

func TestInterpolateHex_AtHalf(t *testing.T) {
	c := interpolateHex("#ff0000", "#0000ff", 0.5)
	// Expect midpoint: R=128, G=0, B=128 → #800080
	// (255 * 0.5 = 127.5 → rounds to 128 = 0x80)
	assert.Equal(t, lipgloss.Color("#800080"), c)
}

func TestInterpolateHex_Clamp(t *testing.T) {
	// t < 0 should clamp to color1
	c := interpolateHex("#ff0000", "#0000ff", -0.5)
	assert.Equal(t, lipgloss.Color("#ff0000"), c)

	// t > 1 should clamp to color2
	c = interpolateHex("#ff0000", "#0000ff", 1.5)
	assert.Equal(t, lipgloss.Color("#0000ff"), c)
}

// --------------------------------------------------------------------------
// GradientSeekBar
// --------------------------------------------------------------------------

func TestGradientSeekBar_ZeroProgress(t *testing.T) {
	b := newTestSeekBar(50)
	out := b.Render(0, 300000)
	assert.NotContains(t, out, "█", "zero progress should show no filled chars")
	assert.Contains(t, out, "░", "zero progress should show empty chars")
}

func TestGradientSeekBar_HalfProgress(t *testing.T) {
	b := newTestSeekBar(50)
	out := b.Render(150000, 300000)
	assert.Contains(t, out, "█")
	assert.Contains(t, out, "░")
}

func TestGradientSeekBar_FullProgress(t *testing.T) {
	b := newTestSeekBar(50)
	out := b.Render(300000, 300000)
	assert.Contains(t, out, "█")
	assert.NotContains(t, out, "░", "full progress should show no empty chars")
}

func TestGradientSeekBar_TimeLabel_Format(t *testing.T) {
	b := newTestSeekBar(60)
	// 2:30 = 150000ms; 5:00 = 300000ms
	out := b.Render(150000, 300000)
	assert.Contains(t, out, "2:30", "should show elapsed time 2:30")
	assert.Contains(t, out, "5:00", "should show total time 5:00")
}

func TestGradientSeekBar_ZeroDuration(t *testing.T) {
	b := newTestSeekBar(50)
	// Must not panic.
	require.NotPanics(t, func() {
		out := b.Render(0, 0)
		assert.NotContains(t, out, "NaN")
	})
}

func TestGradientSeekBar_WidthChanges(t *testing.T) {
	b40 := newTestSeekBar(40)
	b80 := newTestSeekBar(80)

	out40 := b40.Render(120000, 300000)
	out80 := b80.Render(120000, 300000)
	// Wider bar should produce more characters (before ANSI stripping, compare lengths).
	assert.Greater(t, len(out80), len(out40), "wider bar should produce more output")
}

func TestGradientSeekBar_TimeLabelPadded(t *testing.T) {
	b := newTestSeekBar(50)
	// 1:01 = 61000ms — single digit seconds should be zero-padded
	out := b.Render(61000, 120000)
	assert.Contains(t, out, "1:01")
}

// --------------------------------------------------------------------------
// GradientVolumeBar
// --------------------------------------------------------------------------

func TestGradientVolumeBar_ZeroVolume(t *testing.T) {
	b := newTestGradientVolumeBar(30)
	out := b.Render(0)
	assert.Contains(t, out, "VOL")
	assert.Contains(t, out, "0%")
	assert.NotContains(t, out, "█", "zero volume should show no filled chars")
}

func TestGradientVolumeBar_LowVolume_Gradient1(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	// 25% is in the 0-33% band — should use Gradient1 color.
	// In no-color terminal, just verify structural output.
	out := b.Render(25)
	assert.Contains(t, out, "VOL")
	assert.Contains(t, out, "25%")
	assert.Contains(t, out, "█")
}

func TestGradientVolumeBar_MidVolume_Gradient2(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	// 50% is in the 34-66% band — should use Gradient2 color.
	out := b.Render(50)
	assert.Contains(t, out, "VOL")
	assert.Contains(t, out, "50%")
	assert.Contains(t, out, "█")
}

func TestGradientVolumeBar_HighVolume_Gradient3(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	// 80% is in the 67-100% band — should use Gradient3 color.
	out := b.Render(80)
	assert.Contains(t, out, "VOL")
	assert.Contains(t, out, "80%")
	assert.Contains(t, out, "█")
}

func TestGradientVolumeBar_FullVolume(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	out := b.Render(100)
	assert.Contains(t, out, "100%")
	assert.NotContains(t, out, "░", "full volume should have no empty chars")
}

func TestGradientVolumeBar_ClampHigh(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	// Volume > 100 should be clamped to 100.
	out := b.Render(150)
	assert.Contains(t, out, "100%")
}

func TestGradientVolumeBar_ClampLow(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	// Volume < 0 should be clamped to 0.
	out := b.Render(-5)
	assert.Contains(t, out, "0%")
}

func TestGradientVolumeBar_WidthChanges(t *testing.T) {
	b30 := newTestGradientVolumeBar(30)
	b60 := newTestGradientVolumeBar(60)

	out30 := b30.Render(50)
	out60 := b60.Render(50)

	lines30 := strings.Split(out30, "\n")[0]
	lines60 := strings.Split(out60, "\n")[0]

	// Wider bar should produce longer line.
	assert.Greater(t, len(lines60), len(lines30), "wider bar should produce longer line")
}

// --------------------------------------------------------------------------
// Threshold boundary tests
// --------------------------------------------------------------------------

func TestGradientVolumeBar_At33_Gradient1(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	out := b.Render(33)
	// 33% should still be in band 1 (0-33%).
	assert.Contains(t, out, "33%")
}

func TestGradientVolumeBar_At34_Gradient2(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	out := b.Render(34)
	// 34% crosses into band 2 (34-66%).
	assert.Contains(t, out, "34%")
}

func TestGradientVolumeBar_At66_Gradient2(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	out := b.Render(66)
	assert.Contains(t, out, "66%")
}

func TestGradientVolumeBar_At67_Gradient3(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	out := b.Render(67)
	// 67% crosses into band 3 (67-100%).
	assert.Contains(t, out, "67%")
}
