package components

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func newTestVolumeBar() VolumeBar {
	t := theme.Load("black")
	return NewVolumeBar(t)
}

func TestVolumeBar_Zero(t *testing.T) {
	vb := newTestVolumeBar()
	output := vb.Render(0)

	assert.Contains(t, output, "VOL")
	assert.Contains(t, output, "0%")
	assert.NotContains(t, output, "█", "zero volume should show no filled chars")
}

func TestVolumeBar_Fifty(t *testing.T) {
	vb := newTestVolumeBar()
	output := vb.Render(50)

	assert.Contains(t, output, "50%")
	assert.Contains(t, output, "█")
	assert.Contains(t, output, "░")
}

func TestVolumeBar_Hundred(t *testing.T) {
	vb := newTestVolumeBar()
	output := vb.Render(100)

	assert.Contains(t, output, "100%")
	assert.NotContains(t, output, "░", "full volume should show no empty chars")
}

func TestVolumeBar_OverHundred(t *testing.T) {
	vb := newTestVolumeBar()
	output := vb.Render(150)

	// Should be clamped to 100.
	assert.Contains(t, output, "100%")
	assert.NotContains(t, output, "░")
}

func TestVolumeBar_Negative(t *testing.T) {
	vb := newTestVolumeBar()
	output := vb.Render(-10)

	// Should be clamped to 0.
	assert.Contains(t, output, "0%")
	assert.NotContains(t, output, "█")
}
