package viz

import "math"

// numFrames is the number of animation frames in each precomputed frame table.
const numFrames = 40

// HeightFunc computes column heights for a given frame index.
// Parameters:
//   - width: number of columns
//   - maxHeight: maximum height value (dot rows for braille, display rows for block)
//   - frameIdx: current frame index in [0, numFrames)
//
// Returns a slice of length width with values in [0, maxHeight].
// Must be deterministic — same inputs always produce the same output.
// Must not use math/rand.
type HeightFunc func(width, maxHeight, frameIdx int) []int

// Pattern combines a name, a Renderer, and a HeightFunc.
// Adding a new pattern requires only defining a HeightFunc and choosing a Renderer.
type Pattern struct {
	// Name is a human-readable label for the pattern.
	Name string
	// Renderer produces frames from column heights.
	Renderer Renderer
	// HeightFunc computes column heights for each animation frame.
	HeightFunc HeightFunc
}

// Patterns returns the ordered list of available animation patterns.
// Index is stable — callers cycle by incrementing modulo len(Patterns()).
//
// Pattern 0: DotRenderer (Standing Wave)
// Pattern 1: BrailleMirrorRenderer (Braille Mirror)
func Patterns() []Pattern {
	return []Pattern{
		{
			Name:       "Standing Wave",
			Renderer:   DotRenderer{},
			HeightFunc: heightStandingWave,
		},
		{
			Name:       "Braille Mirror",
			Renderer:   BrailleMirrorRenderer{},
			HeightFunc: heightBrailleMirror,
		},
	}
}

// clamp01 clamps a value to [0, 1].
func clamp01(v float64) float64 {
	if v > 1.0 {
		return 1.0
	}
	if v < 0 {
		return 0
	}
	return v
}

// phaseFor returns the phase shift in radians for a given frame index.
// Completes one full cycle over numFrames frames.
func phaseFor(frameIdx int) float64 {
	return float64(frameIdx) * (2 * math.Pi / float64(numFrames))
}

// heightStandingWave — Pattern 0 (DotRenderer): standing wave with 2 antinodes.
// Column heights serve as phase offsets for horizontal wave.
func heightStandingWave(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 2 * math.Pi
		val := 0.5 + 0.5*math.Sin(x+phase) + 0.3*math.Sin(2*x+phase*0.5)
		val = clamp01(val)
		// Map sine to height range for DotRenderer
		out[col] = int(val * float64(maxHeight))
	}
	return out
}

// heightBrailleMirror — Pattern 1 (BrailleMirrorRenderer): double-lobe standing wave.
// Column heights determine lobe thickness (distance from center to lobe edge).
func heightBrailleMirror(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	center := maxHeight / 2
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 2 * math.Pi
		val := 0.4 + 0.6*(0.5+0.5*math.Sin(x+phase)+0.3*math.Sin(2*x+phase*0.5))
		val = clamp01(val)
		// lobeThickness = distance from center to edge; add frame-based micro-variation
		lobeThickness := int(val*float64(center-1)) + int(2*math.Sin(phase*1.3))
		if lobeThickness < 1 {
			lobeThickness = 1
		}
		if lobeThickness > maxHeight {
			lobeThickness = maxHeight
		}
		out[col] = lobeThickness
	}
	return out
}
