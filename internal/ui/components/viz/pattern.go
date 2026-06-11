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
// Pattern 0: GaussianRenderer (Pixel Spectrum)
// Pattern 1: DotRenderer (Standing Wave)
// Pattern 2: FloorRenderer (Floor Spectrum)
// Pattern 3: BrailleMirrorRenderer (Braille Mirror)
func Patterns() []Pattern {
	return []Pattern{
		{
			Name:       "Pixel Spectrum",
			Renderer:   GaussianRenderer{},
			HeightFunc: heightPixelSpectrum,
		},
		{
			Name:       "Standing Wave",
			Renderer:   DotRenderer{},
			HeightFunc: heightStandingWave,
		},
		{
			Name:       "Floor Spectrum",
			Renderer:   FloorRenderer{},
			HeightFunc: heightFloorSpectrum,
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

// heightPulseRipple — Pattern 1 (Braille): narrow Gaussian peak traveling left-to-right
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

// heightBlockSparse — Pattern 2 (Block): low overall height with occasional
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

// heightWinampEQ — Pattern 3 (DensityRenderer): dense equalizer bars with
// small per-column variation. Base height near max (0.75) with ±0.25 sine
// oscillation per column. Creates the classic 90s equalizer look where every
// bar is active with visible variation.
func heightWinampEQ(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 2 * math.Pi
		val := 0.75 + 0.25*math.Sin(x*3+phase)*math.Cos(x+phase*0.5)
		val = clamp01(val)
		out[col] = int(val * float64(maxHeight))
	}
	return out
}

// heightMatrixRain — Pattern 4 (CharRenderer): column-staggered binary rain.
// Each column has an independent fall phase offset: colPhase = (phase + col*0.3) % (2*π).
// Stream height oscillates between 0.3 and 0.8 of maxHeight.
// Produces heights per column that create falling streaks.
func heightMatrixRain(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	for col := 0; col < width; col++ {
		colPhase := math.Mod(phase+float64(col)*0.3, 2*math.Pi)
		// Height oscillates between 0.3 and 0.8 of maxHeight
		h := 0.3 + 0.5*(math.Sin(colPhase)+1)/2
		out[col] = int(h * float64(maxHeight))
		if out[col] < 1 {
			out[col] = 1
		}
	}
	return out
}

// heightPixelSpectrum — Pattern 0 (GaussianRenderer): centered Gaussian density
// wave. Multi-frequency sine composition for smooth organic motion.
func heightPixelSpectrum(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 2 * math.Pi
		val := 0.5 + 0.3*math.Sin(x+phase) + 0.15*math.Sin(2*x+phase*0.7) + 0.05*math.Sin(5*x)
		val = clamp01(val)
		out[col] = int(val * float64(maxHeight))
	}
	return out
}

// heightStandingWave — Pattern 1 (DotRenderer): standing wave with 2 antinodes.
// Column heights serve as phase offsets for horizontal wave.
func heightStandingWave(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 2 * math.Pi
		val := 0.5 + 0.5*math.Sin(x+phase) + 0.3*math.Sin(2*x+phase*0.5)
		val = clamp01(val)
		// Store phase-like value (0-100) for renderer to use
		out[col] = int(val * 100)
	}
	return out
}

// heightFloorSpectrum — Pattern 2 (FloorRenderer): bars with thick baseline floor.
// Bottom quarter is floor, upper portion varies via sine.
func heightFloorSpectrum(width, maxHeight, frameIdx int) []int {
	out := make([]int, width)
	phase := phaseFor(frameIdx)
	floor := maxHeight / 4
	if floor < 1 {
		floor = 1
	}
	for col := 0; col < width; col++ {
		x := float64(col) / float64(width) * 2 * math.Pi
		val := 0.3 + 0.7*(0.5+0.5*math.Sin(x+phase)+0.3*math.Sin(2*x+phase*0.6))
		val = clamp01(val)
		out[col] = floor + int(val*float64(maxHeight-floor))
	}
	return out
}

// heightBrailleMirror — Pattern 3 (BrailleMirrorRenderer): double-lobe standing wave.
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
