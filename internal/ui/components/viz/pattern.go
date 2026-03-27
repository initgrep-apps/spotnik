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

// Patterns returns the full ordered list of available animation patterns.
// Index is stable — callers cycle by incrementing modulo len(Patterns()).
//
// Indices 0-2, 6: BrailleRenderer patterns.
// Indices 3-5: BlockRenderer patterns.
func Patterns() []Pattern {
	return []Pattern{
		{
			Name:       "Braille Dual Sine",
			Renderer:   BrailleRenderer{},
			HeightFunc: heightDualSine,
		},
		{
			Name:       "Braille Standing Wave",
			Renderer:   BrailleRenderer{},
			HeightFunc: heightStandingWave,
		},
		{
			Name:       "Braille Pulse Ripple",
			Renderer:   BrailleRenderer{},
			HeightFunc: heightPulseRipple,
		},
		{
			Name:       "Block Dense Equalizer",
			Renderer:   BlockRenderer{},
			HeightFunc: heightBlockDenseEqualizer,
		},
		{
			Name:       "Block Waveform",
			Renderer:   BlockRenderer{},
			HeightFunc: heightBlockWaveform,
		},
		{
			Name:       "Block Sparse",
			Renderer:   BlockRenderer{},
			HeightFunc: heightBlockSparse,
		},
		{
			Name:       "Braille Organic",
			Renderer:   BrailleRenderer{},
			HeightFunc: heightBrailleOrganic,
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

// heightDualSine — Pattern 0 (Braille): two sine waves at different frequencies.
// Ported directly from the existing visualizer.go pattern 0.
func heightDualSine(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 2 * math.Pi
		val := 0.5*(math.Sin(x+phase)+1) +
			0.3*(math.Sin(2*x+phase*1.3)+1)*0.5 +
			0.2*(math.Sin(3*x+phase*0.7)+1)*0.5
		val = clamp01(val)
		out[col] = int(val * float64(maxHeight))
	}
	return out
}

// heightStandingWave — Pattern 1 (Braille): interference of two counter-propagating
// sine waves creating stationary nodes and antinodes.
// Ported directly from the existing visualizer.go pattern 1.
func heightStandingWave(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 2 * math.Pi
		wave1 := math.Sin(x*2 + phase)
		wave2 := math.Sin(x*2 - phase)
		val := (wave1 + wave2 + 2.0) / 4.0
		val = clamp01(val)
		out[col] = int(val * float64(maxHeight))
	}
	return out
}

// heightPulseRipple — Pattern 2 (Braille): narrow Gaussian peak traveling left-to-right
// with a trailing ripple, wrapping around the edges.
// Ported directly from the existing visualizer.go pattern 2.
func heightPulseRipple(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	peakPos := float64(frameIdx) / float64(numFrames)
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width)
		dist := math.Abs(x - peakPos)
		if dist > 0.5 {
			dist = 1.0 - dist
		}
		sigma := 0.08
		val := math.Exp(-(dist * dist) / (2 * sigma * sigma))
		ripple := 0.15 * math.Exp(-(dist*dist)/(2*0.15*0.15)) * math.Sin(dist*30)
		val = clamp01(val + ripple)
		out[col] = int(val * float64(maxHeight))
	}
	return out
}

// heightBlockDenseEqualizer — Pattern 3 (Block): full-height bars with small
// deterministic variation per column. Dense, heavy look.
func heightBlockDenseEqualizer(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 2 * math.Pi
		// Base height near max with small oscillation
		val := 0.75 + 0.25*math.Sin(x*3+phase)*math.Cos(x+phase*0.5)
		val = clamp01(val)
		out[col] = int(val * float64(maxHeight))
	}
	return out
}

// heightBlockWaveform — Pattern 4 (Block): smooth sine-based heights.
// Clean, flowing appearance.
func heightBlockWaveform(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 2 * math.Pi
		val := 0.5*(math.Sin(x+phase)+1) + 0.2*(math.Sin(2*x-phase*0.8)+1)*0.5
		val = clamp01(val)
		out[col] = int(val * float64(maxHeight))
	}
	return out
}

// heightBlockSparse — Pattern 5 (Block): low overall height with occasional
// deterministic spikes. Ambient, minimal feel.
func heightBlockSparse(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 4 * math.Pi
		// Mostly low with occasional narrow peaks
		base := 0.1 * (math.Sin(x+phase) + 1)
		// Gaussian spike at a single phase-dependent position per frame
		spikePos := math.Mod(phase/(2*math.Pi), 1.0)
		colPos := float64(col) / float64(width)
		dist := math.Abs(colPos - spikePos)
		if dist > 0.5 {
			dist = 1.0 - dist
		}
		spike := 0.8 * math.Exp(-(dist*dist)/(2*0.05*0.05))
		val := clamp01(base + spike)
		out[col] = int(val * float64(maxHeight))
	}
	return out
}

// heightBrailleOrganic — Pattern 6 (Braille): multi-frequency sine composition
// producing natural, organic movement without external randomness.
func heightBrailleOrganic(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 2 * math.Pi
		// Superposition of four frequencies with different phase offsets
		v1 := 0.35 * (math.Sin(x+phase) + 1)
		v2 := 0.25 * (math.Sin(2.3*x+phase*1.1) + 1)
		v3 := 0.25 * (math.Sin(3.7*x-phase*0.9) + 1)
		v4 := 0.15 * (math.Sin(5.1*x+phase*1.4) + 1)
		val := clamp01((v1 + v2 + v3 + v4) / 2.0)
		out[col] = int(val * float64(maxHeight))
	}
	return out
}
