package components

import (
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// VisualizerTickMsg is sent on the visualizer's animation tick.
type VisualizerTickMsg time.Time

// numFrames is the number of animation frames in the precomputed frame table.
const numFrames = 40

// NumPatterns is the number of available animation patterns.
const NumPatterns = 3

// Visualizer renders an animated braille-dot audio spectrum.
// It maintains a precomputed frame table for deterministic, allocation-free animation.
//
// The frame table is generated once at construction time and regenerated when
// SetSize() or SetPattern() changes dimensions or pattern. View() simply indexes
// into the table — no work per frame during animation.
type Visualizer struct {
	th         theme.Theme
	playing    bool
	frameIndex int
	width      int
	height     int // number of display lines (1-4)
	pattern    int // animation pattern index (0..NumPatterns-1)
	interval   time.Duration
	frames     [][]string // frames[frameIndex][lineIndex] = styled braille string
}

// NewVisualizer creates a Visualizer with a default 200ms animation interval.
// Call SetSize before calling View or Init.
func NewVisualizer(t theme.Theme) *Visualizer {
	v := &Visualizer{
		th:       t,
		interval: 200 * time.Millisecond,
	}
	return v
}

// SetSize updates the visualizer dimensions and regenerates the frame table.
// width is the number of braille columns. height is the number of display lines (1-4).
func (v *Visualizer) SetSize(width, height int) {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	if height > 4 {
		height = 4
	}
	if v.width == width && v.height == height {
		return
	}
	v.width = width
	v.height = height
	v.frames = generateFrames(width, height, v.pattern)
}

// Pattern returns the current animation pattern index (0..NumPatterns-1).
func (v *Visualizer) Pattern() int {
	return v.pattern
}

// SetPattern selects an animation pattern and regenerates the frame table.
// Values outside 0..NumPatterns-1 are clamped.
func (v *Visualizer) SetPattern(p int) {
	if p < 0 {
		p = 0
	}
	if p >= NumPatterns {
		p = NumPatterns - 1
	}
	if v.pattern == p {
		return
	}
	v.pattern = p
	if v.width > 0 && v.height > 0 {
		v.frames = generateFrames(v.width, v.height, v.pattern)
	}
}

// CyclePattern advances the animation pattern: 0→1→2→0.
func (v *Visualizer) CyclePattern() {
	v.SetPattern((v.pattern + 1) % NumPatterns)
}

// SetPlaying controls animation state.
// When playing, frameIndex advances on each VisualizerTickMsg.
// When paused, frameIndex is frozen and a flat-line pattern is shown.
func (v *Visualizer) SetPlaying(playing bool) {
	v.playing = playing
}

// FrameIndex returns the current animation frame index.
// Exported for integration testing and pane state inspection.
func (v *Visualizer) FrameIndex() int {
	return v.frameIndex
}

// Init returns the initial tick command to start the animation loop.
func (v *Visualizer) Init() tea.Cmd {
	return v.tickCmd()
}

// Update handles VisualizerTickMsg to advance the animation.
// Returns a re-armed tick command on tick messages; nil for all other messages.
// Safe to call before SetSize — no-ops gracefully when frames is empty.
func (v *Visualizer) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case VisualizerTickMsg:
		if v.playing && len(v.frames) > 0 {
			v.frameIndex = (v.frameIndex + 1) % len(v.frames)
		}
		return v.tickCmd()
	}
	return nil
}

// View renders the current frame. Pure function — reads frameIndex, returns string.
// Returns a flat-line pattern when paused.
func (v *Visualizer) View() string {
	if len(v.frames) == 0 {
		return ""
	}

	style := lipgloss.NewStyle().Foreground(v.th.VisualizerFg())

	if !v.playing {
		// Flat-line: render the blank row (all ⠀ chars) for all lines.
		lines := make([]string, v.height)
		blank := strings.Repeat("⠀", v.width)
		for i := range lines {
			lines[i] = style.Render(blank)
		}
		return strings.Join(lines, "\n")
	}

	frame := v.frames[v.frameIndex]
	lines := make([]string, len(frame))
	for i, line := range frame {
		lines[i] = style.Render(line)
	}
	return strings.Join(lines, "\n")
}

// tickCmd returns a tea.Tick for the next animation frame.
func (v *Visualizer) tickCmd() tea.Cmd {
	return tea.Tick(v.interval, func(t time.Time) tea.Msg {
		return VisualizerTickMsg(t)
	})
}

// generateFrames builds the precomputed frame table for the given pattern.
// Each frame is a []string — one string per line of display height.
// Bar heights are derived from deterministic math functions with per-frame phase offsets,
// ensuring identical output for the same frameIndex across different Visualizer instances.
//
// Pattern 0: dual sine wave — two sine waves at different frequencies create a flowing wave.
// Pattern 1: standing wave — interference of two counter-propagating waves creates nodes/antinodes.
// Pattern 2: pulse/ripple — a narrow peak travels left-to-right and wraps, like a sonar ping.
func generateFrames(width, height, pattern int) [][]string {
	result := make([][]string, numFrames)
	totalDotRows := height * 4

	for f := 0; f < numFrames; f++ {
		colHeights := computeColumnHeights(width, totalDotRows, f, pattern)
		result[f] = renderFrame(width, height, colHeights)
	}
	return result
}

// computeColumnHeights returns the bar height (in dot rows) for each column
// in a single frame, based on the selected animation pattern.
func computeColumnHeights(width, totalDotRows, frame, pattern int) []int {
	colHeights := make([]int, width)
	phaseShift := float64(frame) * (2 * math.Pi / float64(numFrames))

	switch pattern {
	case 1:
		// Standing wave: interference of two counter-propagating sine waves.
		// Creates stationary nodes (zero amplitude) and antinodes (max amplitude).
		for col := 0; col < width; col++ {
			x := float64(col) / float64(width) * 2 * math.Pi
			// Two waves traveling in opposite directions sum to a standing wave.
			wave1 := math.Sin(x*2 + phaseShift)
			wave2 := math.Sin(x*2 - phaseShift)
			val := (wave1 + wave2 + 2.0) / 4.0 // normalize to [0, 1]
			val = clamp01(val)
			colHeights[col] = int(val * float64(totalDotRows))
		}
	case 2:
		// Pulse/ripple: a narrow Gaussian peak that travels left-to-right and wraps.
		// The peak position advances by one full width over the frame cycle.
		peakPos := float64(frame) / float64(numFrames) // 0..1 position across width
		for col := 0; col < width; col++ {
			x := float64(col) / float64(width)
			// Distance from peak, wrapping around edges.
			dist := math.Abs(x - peakPos)
			if dist > 0.5 {
				dist = 1.0 - dist
			}
			// Gaussian envelope: sigma controls pulse width.
			sigma := 0.08
			val := math.Exp(-(dist * dist) / (2 * sigma * sigma))
			// Add a small trailing ripple for visual interest.
			ripple := 0.15 * math.Exp(-(dist*dist)/(2*0.15*0.15)) * math.Sin(dist*30)
			val = clamp01(val + ripple)
			colHeights[col] = int(val * float64(totalDotRows))
		}
	default:
		// Pattern 0: dual sine wave (original behavior).
		for col := 0; col < width; col++ {
			x := float64(col) / float64(width) * 2 * math.Pi
			val := 0.5*(math.Sin(x+phaseShift)+1) +
				0.3*(math.Sin(2*x+phaseShift*1.3)+1)*0.5 +
				0.2*(math.Sin(3*x+phaseShift*0.7)+1)*0.5
			val = clamp01(val)
			colHeights[col] = int(val * float64(totalDotRows))
		}
	}
	return colHeights
}

// renderFrame converts column heights into braille display lines.
func renderFrame(width, height int, colHeights []int) []string {
	frameLines := make([]string, height)
	for lineIdx := 0; lineIdx < height; lineIdx++ {
		lineBottom := (height - 1 - lineIdx) * 4
		var sb strings.Builder
		for col := 0; col < width; col++ {
			h := colHeights[col]
			filledInChar := h - lineBottom
			if filledInChar < 0 {
				filledInChar = 0
			}
			if filledInChar > 4 {
				filledInChar = 4
			}
			sb.WriteRune(brailleChar(filledInChar))
		}
		frameLines[lineIdx] = sb.String()
	}
	return frameLines
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

// brailleChar returns a Unicode braille character for a given fill level (0-4).
// The characters match the spec's height mapping exactly and produce a bar
// appearance when rendered in a terminal at each fill level.
//
// Braille dot layout: 1 4 / 2 5 / 3 6 / 7 8 (left|right per row).
// Bits: dot1=0x01, dot2=0x02, dot3=0x04, dot4=0x08, dot5=0x10, dot6=0x20, dot7=0x40, dot8=0x80.
//
//	0 filled: ⠀ (U+2800, offset 0x00) — blank
//	1 filled: ⡀ (U+2840, offset 0x40) — dot7 (left col, row 4)
//	2 filled: ⡠ (U+2860, offset 0x60) — dot6+dot7 (right col row 3 + left col row 4)
//	3 filled: ⡰ (U+2870, offset 0x70) — dot5+dot6+dot7 (right col rows 2-3 + left col row 4)
//	4 filled: ⣰ (U+28F0, offset 0xF0) — dot5+dot6+dot7+dot8 (bottom two rows filled)
//
// NOTE: These codepoints match the spec exactly. The fill pattern spans both
// columns for a wider visual bar effect rather than a single-column fill.
func brailleChar(filledDots int) rune {
	switch filledDots {
	case 0:
		return '\u2800' // ⠀ blank
	case 1:
		return '\u2840' // ⡀ dot7 (0x40)
	case 2:
		return '\u2860' // ⡠ dot6+dot7 (0x20|0x40=0x60)
	case 3:
		return '\u2870' // ⡰ dot5+dot6+dot7 (0x10|0x20|0x40=0x70)
	default: // 4+
		return '\u28F0' // ⣰ dot5+dot6+dot7+dot8 (0x10|0x20|0x40|0x80=0xF0)
	}
}
