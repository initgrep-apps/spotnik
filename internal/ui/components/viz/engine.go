package viz

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
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
// When paused, CurrentFrame() returns a blank frame.
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
// Returns a blank Frame (all empty StyledLines) when paused.
// Returns an empty Frame before SetSize is called.
func (e *Engine) CurrentFrame() Frame {
	if len(e.frames) == 0 {
		return Frame{}
	}
	if !e.playing {
		// Return a blank frame with the correct dimensions and colors,
		// but empty text.
		blank := make(Frame, e.height)
		colors := e.buildColors(e.height)
		for i := range blank {
			blank[i] = StyledLine{Text: "", Color: colors[i]}
		}
		return blank
	}
	return e.frames[e.frameIdx]
}

// CyclePattern advances to the next pattern (wraps around) and regenerates frames.
// Resets frameIdx to 0.
func (e *Engine) CyclePattern() {
	e.patternIdx = (e.patternIdx + 1) % len(e.patterns)
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

// generateFrames builds the precomputed frame table for the current pattern.
// Precomputes numFrames frames using the current pattern's HeightFunc and Renderer.
// Per-row colors are assigned using the gradient (Gradient3 top, Gradient1 bottom).
func (e *Engine) generateFrames() []Frame {
	if e.height <= 0 || e.width <= 0 {
		return nil
	}

	p := e.patterns[e.patternIdx]
	colors := e.buildColors(e.height)

	// MaxHeight is renderer-specific: braille uses height*4 (dot rows),
	// block uses height (display rows). Delegating avoids type assertions here.
	maxHeight := p.Renderer.MaxHeight(e.height)

	frames := make([]Frame, numFrames)
	for f := 0; f < numFrames; f++ {
		colHeights := p.HeightFunc(e.width, maxHeight, f)
		frames[f] = p.Renderer.RenderFrame(e.width, e.height, colHeights, colors)
	}
	return frames
}

// buildColors constructs the per-row color slice for a given height.
// Row assignment:
//   - Top 1/3: Gradient3 (red/hot, peaks)
//   - Middle 1/3: Gradient2 (yellow/warm)
//   - Bottom 1/3: Gradient1 (green/cool, base)
//
// For heights not evenly divisible by 3, extra rows go to the bottom third.
func (e *Engine) buildColors(height int) []lipgloss.Color {
	colors := make([]lipgloss.Color, height)
	third := height / 3

	for i := 0; i < height; i++ {
		switch {
		case i < third:
			colors[i] = e.theme.Gradient3()
		case i < 2*third:
			colors[i] = e.theme.Gradient2()
		default:
			colors[i] = e.theme.Gradient1()
		}
	}
	return colors
}
