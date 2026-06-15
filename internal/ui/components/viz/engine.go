package viz

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// TickMsg is sent on the visualizer's animation tick.
// Replaces components.VisualizerTickMsg.
type TickMsg time.Time

// defaultInterval is the time between animation frames.
const defaultInterval = 200 * time.Millisecond

// Engine manages the pattern registry, current pattern, precomputed frame table,
// frame index advancement, and playing/paused state.
//
// Usage:
//
//	e := NewEngine(th)
//	e.SetSize(width, height)
//	e.SetPlaying(true)
//	cmd := e.Init()   // start tick loop
//	// in pane Update:
//	cmd = e.Update(msg)  // re-arms tick, call e.Advance() on TickMsg
//	// in pane View:
//	frame := e.CurrentFrame()
type Engine struct {
	theme      theme.Theme
	patterns   []Pattern
	patternIdx int
	frames     []Frame // precomputed frame table indexed by frameIdx
	frameIdx   int
	playing    bool
	width      int
	height     int
	interval   time.Duration
}

// NewEngine creates an Engine with all registered patterns.
// The theme is used to resolve gradient colors during frame precomputation.
// If th is nil, the "black" theme is used as a safe default.
// Call SetSize before calling CurrentFrame or Init.
func NewEngine(th theme.Theme) *Engine {
	if th == nil {
		th = theme.Load("black")
	}
	return &Engine{
		theme:    th,
		patterns: Patterns(),
		interval: defaultInterval,
	}
}

// SetSize updates dimensions and regenerates the precomputed frame table.
// No height cap — the engine accepts any height passed by the pane.
// Resets frameIdx to 0 when dimensions change.
func (e *Engine) SetSize(width, height int) {
	if width < 1 {
		width = 1
	}
	if height < 0 {
		height = 0
	}
	if e.width == width && e.height == height {
		return
	}
	e.width = width
	e.height = height
	e.frameIdx = 0
	e.frames = e.generateFrames()
}

// SetPlaying controls animation state.
// When playing, Advance() increments the frame index.
// When paused, CurrentFrame() returns the last frame frozen in place.
func (e *Engine) SetPlaying(playing bool) {
	e.playing = playing
}

// Advance increments frameIdx when playing. Called by the pane on each tick.
// Safe to call before SetSize — no-ops when no frames are precomputed.
func (e *Engine) Advance() {
	if !e.playing || len(e.frames) == 0 {
		return
	}
	e.frameIdx = (e.frameIdx + 1) % len(e.frames)
}

// CurrentFrame returns the current precomputed frame.
// When paused, returns the last frame frozen in place (not blank).
// Returns an empty Frame before SetSize is called.
func (e *Engine) CurrentFrame() Frame {
	if len(e.frames) == 0 {
		return Frame{}
	}
	return e.frames[e.frameIdx]
}

// CyclePattern advances to the next pattern (wraps around) and regenerates frames.
// Resets frameIdx to 0.
func (e *Engine) CyclePattern() {
	if len(e.patterns) == 0 {
		return
	}
	e.patternIdx = (e.patternIdx + 1) % len(e.patterns)
	e.frameIdx = 0
	if e.width > 0 && e.height > 0 {
		e.frames = e.generateFrames()
	}
}

// SetPattern sets the active pattern to the given index.
// If index is out of range, it wraps with modulo (same as CyclePattern).
// Negative values are clamped to 0.
// Resets frameIdx to 0 and regenerates frames if the engine has been sized.
func (e *Engine) SetPattern(index int) {
	if len(e.patterns) == 0 {
		return
	}
	e.patternIdx = index % len(e.patterns)
	if e.patternIdx < 0 {
		e.patternIdx = 0
	}
	e.frameIdx = 0
	if e.width > 0 && e.height > 0 {
		e.frames = e.generateFrames()
	}
}

// Pattern returns the current pattern index.
func (e *Engine) Pattern() int {
	return e.patternIdx
}

// PatternCount returns the total number of registered patterns.
func (e *Engine) PatternCount() int {
	return len(e.patterns)
}

// FrameIndex returns the current frame index (for testing and debugging).
func (e *Engine) FrameIndex() int {
	return e.frameIdx
}

// Init returns the initial tick command to start the animation loop.
func (e *Engine) Init() tea.Cmd {
	return e.tickCmd()
}

// Update handles TickMsg to re-arm the tick loop.
// Returns a re-armed tick command on tick messages; nil for all other messages.
func (e *Engine) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case TickMsg:
		return e.tickCmd()
	}
	return nil
}

// tickCmd returns a tea.Tick for the next animation frame.
func (e *Engine) tickCmd() tea.Cmd {
	return tea.Tick(e.interval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// selectRenderer returns the Renderer to use for the current pattern.
// When uikit.ActiveMode() reports GlyphASCII, AsciiBarsRenderer is returned
// regardless of the configured pattern so the visualizer remains present
// (at reduced resolution) on non-UTF-8 terminals. In unicode mode the
// pattern's own renderer is used.
func (e *Engine) selectRenderer() Renderer {
	if len(e.patterns) == 0 {
		return NewAsciiBarsRenderer()
	}
	if uikit.ActiveMode() == uikit.GlyphASCII {
		return NewAsciiBarsRenderer()
	}
	return e.patterns[e.patternIdx].Renderer
}

// generateFrames builds the precomputed frame table for the current pattern.
// Precomputes numFrames frames. If the renderer implements FrameAwareRenderer,
// RenderFrameAt is called with the frame index directly; otherwise the
// pattern's HeightFunc is used and RenderFrame receives column heights.
// Per-row colors are assigned using the 7-zone gradient (VizGradient7 top, VizGradient1 bottom).
func (e *Engine) generateFrames() []Frame {
	if e.height <= 0 || e.width <= 0 || len(e.patterns) == 0 {
		return nil
	}

	p := e.patterns[e.patternIdx]
	colors := e.buildColors(e.height)

	// selectRenderer chooses AsciiBarsRenderer in ASCII mode; the pattern's own
	// renderer otherwise. MaxHeight is renderer-specific:
	// braille renderers use height*4 (dot rows), block renderers use height or 4.
	r := e.selectRenderer()
	maxHeight := r.MaxHeight(e.height)

	frames := make([]Frame, numFrames)
	if fr, ok := r.(FrameAwareRenderer); ok {
		for f := 0; f < numFrames; f++ {
			frames[f] = fr.RenderFrameAt(e.width, e.height, f, colors)
		}
	} else {
		for f := 0; f < numFrames; f++ {
			colHeights := p.HeightFunc(e.width, maxHeight, f)
			frames[f] = r.RenderFrame(e.width, e.height, colHeights, colors)
		}
	}
	return frames
}

// buildColors constructs the per-row color slice for a given height using the
// 7-zone visualizer gradient (VizGradient1 base through VizGradient7 peaks).
//
// Row assignment divides height into 7 zones:
//   - Zone 1 (bottom): VizGradient1
//   - Zone 2: VizGradient2
//   - Zone 3: VizGradient3
//   - Zone 4 (center): VizGradient4
//   - Zone 5: VizGradient5
//   - Zone 6: VizGradient6
//   - Zone 7 (top): VizGradient7
//
// For heights not evenly divisible by 7, extra rows go to the bottom zone.
func (e *Engine) buildColors(height int) []lipgloss.Color {
	colors := make([]lipgloss.Color, height)
	zoneSize := height / 7
	remainder := height % 7

	gradients := []lipgloss.Color{
		e.theme.VizGradient1(),
		e.theme.VizGradient2(),
		e.theme.VizGradient3(),
		e.theme.VizGradient4(),
		e.theme.VizGradient5(),
		e.theme.VizGradient6(),
		e.theme.VizGradient7(),
	}

	row := height
	for zone := 0; zone < 7; zone++ {
		size := zoneSize
		if zone < remainder {
			size++
		}
		start := row - size
		if start < 0 {
			start = 0
		}
		for i := start; i < row; i++ {
			colors[i] = gradients[zone]
		}
		row = start
	}
	return colors
}
