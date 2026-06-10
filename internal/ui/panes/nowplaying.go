// Package panes contains the Bubble Tea pane models for the Spotnik TUI.
// Each pane reads from the central Store and emits request messages for side effects.
// Panes never call the API directly or import api/ — data flows through messages and store only.
package panes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/components/viz"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// NowPlaying overlay layout constants.
//
// The InfoBox overlays the left ~25% of the visualizer; its solid
// OverlayBackground fill (story 221) hides the visualizer behind the box.
const (
	npPadV       = 1  // rows of vertical padding top + bottom
	npInfoMult   = 1  // infoWidth = vizRows * npInfoMult
	npInfoMin    = 18 // minimum InfoBox width for readability
	npGap        = 1  // column gap between InfoBox and visualizer
	npMinViz     = 10 // minimum viz width; below this, drop InfoBox
	npMaxInfoPct = 4  // cap infoWidth at contentWidth / npMaxInfoPct (~25%)
)

// NowPlayingPane is the center pane Bubble Tea model.
// It renders the currently playing track info overlay on top of a full-pane
// visualizer background. It reads all state from the Store; it never stores
// API data in its own fields.
// It implements the layout.Pane interface for integration with the layout manager.
//
// Layout: visualizer fills the full content area; InfoBox overlays the left
// ~25% with a solid OverlayBackground fill. Seek bar lives in the visualizer
// column only (right of the gap). When width is too narrow, the InfoBox is
// dropped and the visualizer fills the full content area. When height < 8,
// Title() embeds compact track info in the pane title bar instead.
type NowPlayingPane struct {
	BasePane

	// localProgressMs is pane-local state (not in Store). It increments by 1000ms
	// on each tick when playing, for smooth seek bar updates between polls.
	localProgressMs int

	// infoBox is the bordered sub-pane on the left showing track/artist/album/controls.
	infoBox *components.InfoBox

	// engine is the animated visualization engine (right side of the overlay).
	engine *viz.Engine

	// seekBar is the gradient seek bar rendered inside the visualizer column.
	seekBar *components.GradientSeekBar

	// volumeBar is the gradient volume bar rendered inside the InfoBox.
	volumeBar *components.GradientVolumeBar
}

// Compile-time check: NowPlayingPane implements layout.Pane.
var _ layout.Pane = &NowPlayingPane{}

// NewNowPlayingPane creates a NowPlayingPane with the given store and theme.
// localProgressMs is initialized from the store's current playback state so that
// constructing a pane after setting state shows the correct position immediately.
func NewNowPlayingPane(s state.StateReader, t theme.Theme, focused bool) *NowPlayingPane {
	p := &NowPlayingPane{
		BasePane:  BasePane{store: s, theme: t, focused: focused},
		infoBox:   components.NewInfoBox(t),
		engine:    viz.NewEngine(t),
		seekBar:   components.NewGradientSeekBar(t),
		volumeBar: components.NewGradientVolumeBar(t),
	}
	if ps := s.PlaybackState(); ps != nil {
		p.localProgressMs = ps.ProgressMs
		p.engine.SetPlaying(ps.IsPlaying)
		if ps.Device != nil {
			p.volumeBar.SetConfirmed(ps.Device.VolumePercent)
		}
	}
	return p
}

// ID returns the PaneID for the NowPlaying slot.
func (p *NowPlayingPane) ID() layout.PaneID {
	return layout.PaneNowPlaying
}

// Title returns the display title for the border.
// When height < 8 (pane too small for the overlay body), the title embeds track info
// so the user can still see what's playing without any content area.
func (p *NowPlayingPane) Title() string {
	if p.height < 8 {
		ps := p.store.PlaybackState()
		if ps != nil && ps.Item != nil {
			t := ps.Item
			artistNames := make([]string, len(t.Artists))
			for i, a := range t.Artists {
				artistNames[i] = a.Name
			}
			m := uikit.ActiveMode()
			var stateGlyph string
			if ps.IsPlaying {
				stateGlyph = uikit.GlyphFor(uikit.GlyphPaused, m)
			} else {
				stateGlyph = uikit.GlyphFor(uikit.GlyphPlaying, m)
			}
			sep := uikit.GlyphFor(uikit.GlyphHRule, m)
			midDot := uikit.GlyphFor(uikit.GlyphSeparator, m)
			current := formatDurationMs(p.localProgressMs)
			total := formatDurationMs(t.DurationMs)
			return fmt.Sprintf("Now Playing %s %s %s %s %s %s %s/%s",
				sep, t.Name, midDot, strings.Join(artistNames, ", "), sep, stateGlyph, current, total)
		}
	}
	return "Now Playing"
}

// ToggleKey returns the number key for btop-style pane toggling (key 1).
func (p *NowPlayingPane) ToggleKey() int {
	return 1
}

// Actions returns the pane-specific shortcuts shown in the border.
// NOTE: layout.RenderPaneBorder drops all actions atomically when the pane is too
// narrow to fit title + actions within cfg.Width (dashCount < 0 after computing
// fixedWidth). At narrow widths none of these five actions will appear; once the
// pane is wide enough they all appear. This is expected graceful degradation, not a bug.
func (p *NowPlayingPane) Actions() []layout.Action {
	return []layout.Action{
		{Key: "s", Label: "shfl"},
		{Key: "r", Label: "rpt"},
		{Key: "space", Label: "play"},
		{Key: "+/-", Label: "vol"},
		{Key: "v", Label: "viz"},
	}
}

// SetSize updates the pane's dimensions and recomputes the overlay layout geometry.
// infoWidth is derived from vizRows, then capped at ~25% of content width.
// When the remaining viz width would be too narrow, the InfoBox is dropped and
// the visualizer fills the full content area.
func (p *NowPlayingPane) SetSize(width, height int) {
	p.BasePane.SetSize(width, height)

	cw := p.contentWidth()
	x := p.vizRows()

	infoWidth := paneMax(x*npInfoMult, npInfoMin)
	if cap := cw / npMaxInfoPct; infoWidth > cap {
		infoWidth = cap
	}
	vizWidth := cw - infoWidth - npGap
	if vizWidth < npMinViz {
		infoWidth = 0
		vizWidth = cw
	}

	vizHeight := paneMax(x-1, 1)
	if infoWidth > 0 {
		p.infoBox.SetSize(infoWidth, x)
	}
	p.engine.SetSize(vizWidth, vizHeight)
	p.seekBar.SetWidth(vizWidth)
	p.volumeBar.SetWidth(paneMax(infoWidth-4, 1))
}

// SetFocused updates the focused state.
func (p *NowPlayingPane) SetFocused(focused bool) {
	p.BasePane.SetFocused(focused)
}

// Init starts the viz engine animation tick loop. The album art fetch was
// removed in story 222 — the visualizer fills the full content area.
func (p *NowPlayingPane) Init() tea.Cmd {
	return p.engine.Init()
}

// Update handles all messages for the NowPlayingPane.
func (p *NowPlayingPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case TickMsg:
		return p.handleTick()

	case PlaybackStateFetchedMsg:
		return p.handlePlaybackFetched(m)

	case components.VolumeDebounceTickMsg:
		if matched, vol, seq := p.volumeBar.HandleDebounce(m); matched {
			return p, func() tea.Msg { return VolumeIntentMsg{TargetVol: vol, Seq: seq} }
		}
		return p, nil

	case VolumeAppliedMsg:
		if m.Err != nil {
			p.volumeBar.CancelPending(m.Seq, confirmedVolume(p.store))
		} else {
			p.volumeBar.ConfirmFromAPI(m.Seq, m.Vol)
		}
		return p, nil

	case viz.TickMsg:
		// Advance the animation frame, then re-arm the tick.
		p.engine.Advance()
		cmd := p.engine.Update(m)
		return p, cmd

	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		return p.handleKey(m)
	}

	return p, nil
}

// View renders the NowPlaying pane. It reads from the store and never calls the API.
// Dispatches to renderEmpty when nothing is playing, otherwise renderOverlay.
func (p *NowPlayingPane) View() string {
	ps := p.store.PlaybackState()
	if ps == nil || ps.Item == nil {
		return p.renderEmpty()
	}
	return p.renderOverlay()
}

// renderOverlay composes the visualizer (full content area) with the InfoBox
// composited on top of its leading edge. The InfoBox's OverlayBackground
// fill (from story 221) hides the visualizer behind it. When the content
// width is too narrow, the InfoBox is dropped and only the visualizer shows.
// The output is padded to one blank row on top and one on the bottom.
func (p *NowPlayingPane) renderOverlay() string {
	ps := p.store.PlaybackState()
	t := ps.Item
	cw := p.contentWidth()
	x := p.vizRows()

	infoWidth := paneMax(x*npInfoMult, npInfoMin)
	if cap := cw / npMaxInfoPct; infoWidth > cap {
		infoWidth = cap
	}
	vizWidth := cw - infoWidth - npGap
	// If the remaining viz width is below the readability threshold, drop the
	// InfoBox and let the visualizer fill the full content area.
	if vizWidth < npMinViz {
		infoWidth = 0
	}

	frame := p.engine.CurrentFrame()
	topRows, bottomRows := splitFrame(frame)
	seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)
	vizPanel := lipgloss.JoinVertical(lipgloss.Left,
		renderStyledLines(topRows), seekBar, renderStyledLines(bottomRows))

	paddedViz := strings.Repeat(" ", infoWidth+npGap) + vizPanel

	var composite string
	if infoWidth > 0 {
		infoLines := p.buildInfoLines(x)
		infoView := p.infoBox.Render("Track Info", infoLines, p.focused)
		composite = lipgloss.JoinHorizontal(lipgloss.Top, infoView, paddedViz)
	} else {
		composite = vizPanel
	}

	// Equal 1-row top + 1-row bottom padding.
	contentH := lipgloss.Height(composite)
	if contentH < p.height {
		pad := p.height - contentH
		topPad := 1
		bottomPad := pad - topPad
		if bottomPad > 1 {
			bottomPad = 1
		}
		if bottomPad < 0 {
			bottomPad = 0
			if topPad > 1 {
				topPad = 1
			}
		}
		composite = strings.Repeat("\n", topPad) + composite
		if bottomPad > 0 {
			composite += strings.Repeat("\n", bottomPad)
		}
	}
	return composite
}

// buildInfoLines builds the 5-line InfoBox content (track, artists, album,
// controls, volume) and pads with trailing blank strings so the InfoBox
// vertically centres to top alignment.
func (p *NowPlayingPane) buildInfoLines(bodyHeight int) []string {
	ps := p.store.PlaybackState()
	if ps == nil || ps.Item == nil {
		return nil
	}
	t := ps.Item
	primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary()).Bold(true)
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	artistNames := make([]string, len(t.Artists))
	for i, a := range t.Artists {
		artistNames[i] = a.Name
	}

	ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)

	lines := []string{
		primaryStyle.Render(t.Name),
		secondaryStyle.Render(strings.Join(artistNames, ", ")),
		mutedStyle.Render(t.Album.Name),
		ctrl.Render(),
		p.volumeBar.Render(),
	}

	innerH := bodyHeight - 2
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	return lines
}

// splitFrame divides a frame into top and bottom halves for display around the seek bar.
// The engine receives vizHeight = bodyHeight - 1 (seek bar row excluded),
// so len(f) == vizHeight. We split evenly: topRows = len/2, bottomRows = len - len/2.
// For odd lengths (e.g. 5), bottom gets the extra row (top=2, bottom=3).
func splitFrame(f viz.Frame) (top, bottom viz.Frame) {
	if len(f) == 0 {
		return nil, nil
	}
	mid := len(f) / 2
	return f[:mid], f[mid:]
}

// renderStyledLines joins StyledLines into a single string with per-line coloring.
func renderStyledLines(lines viz.Frame) string {
	if len(lines) == 0 {
		return ""
	}
	rows := make([]string, len(lines))
	for i, line := range lines {
		style := lipgloss.NewStyle().Foreground(line.Color)
		rows[i] = style.Render(line.Text)
	}
	return strings.Join(rows, "\n")
}

// renderEmpty shows the "Nothing playing" empty state, centered in the pane.
func (p *NowPlayingPane) renderEmpty() string {
	return uikit.EmptyState{
		Text:   "Nothing playing",
		Hint:   "Open Spotify on a device and start playing music",
		Width:  p.width,
		Height: p.height,
		Theme:  p.theme,
	}.Render()
}

// handleTick processes a TickMsg: increments local progress when playing.
// localProgressMs is clamped to DurationMs so the seek bar never overflows.
func (p *NowPlayingPane) handleTick() (*NowPlayingPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil && ps.IsPlaying {
		p.localProgressMs += 1000
		if ps.Item != nil && p.localProgressMs > ps.Item.DurationMs {
			p.localProgressMs = ps.Item.DurationMs
		}
	}
	return p, nil
}

// handlePlaybackFetched processes notification that the store has fresh playback state.
// It resets localProgressMs to the server value and syncs the engine playing state.
// Album art dispatch was removed in story 222.
func (p *NowPlayingPane) handlePlaybackFetched(_ PlaybackStateFetchedMsg) (*NowPlayingPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil {
		p.localProgressMs = ps.ProgressMs
		p.engine.SetPlaying(ps.IsPlaying)
		if ps.Device != nil {
			p.volumeBar.SetConfirmed(ps.Device.VolumePercent)
		}
	} else {
		p.localProgressMs = 0
		p.engine.SetPlaying(false)
	}
	return p, nil
}

// handleKey dispatches key events to playback request messages.
// The root app model receives these and dispatches the corresponding Spotify API calls.
func (p *NowPlayingPane) handleKey(msg tea.KeyMsg) (*NowPlayingPane, tea.Cmd) {
	switch {
	// NOTE: Bubbletea v0.27 delivers Space as tea.KeySpace (Type field), not as a rune.
	// The rune " " branch has been removed — it was dead code and bypassed the
	// premium gate in routing.go which only checks tea.KeySpace.
	case msg.Type == tea.KeySpace:
		ps := p.store.PlaybackState()
		if ps != nil && ps.IsPlaying {
			return p, emitPlaybackRequest(ActionPause)
		}
		return p, emitPlaybackRequest(ActionPlay)

	case msg.Type == tea.KeyRight:
		return p, emitPlaybackRequest(ActionNext)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "p",
		msg.Type == tea.KeyLeft:
		return p, emitPlaybackRequest(ActionPrevious)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "+":
		return p, p.volumeBar.HandleKey(+1, confirmedVolume(p.store))

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "-":
		return p, p.volumeBar.HandleKey(-1, confirmedVolume(p.store))

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "s":
		return p, emitPlaybackRequest(ActionToggleShuffle)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "r":
		return p, emitPlaybackRequest(ActionCycleRepeat)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "v":
		// Cycle engine animation pattern and emit a message so the app can persist
		// the new index via PreferenceStore.
		p.engine.CyclePattern()
		return p, func() tea.Msg {
			return VisualizerPatternChangedMsg{PatternIndex: p.engine.Pattern()}
		}
	}

	return p, nil
}

// formatDurationMs formats milliseconds as "m:ss".
func formatDurationMs(ms int) string {
	totalSec := ms / 1000
	minutes := totalSec / 60
	seconds := totalSec % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// emitPlaybackRequest returns a command that immediately emits a PlaybackRequestMsg.
// The root app model receives this and dispatches the corresponding Spotify API call.
func emitPlaybackRequest(action PlaybackAction) tea.Cmd {
	return func() tea.Msg {
		return PlaybackRequestMsg{Action: action}
	}
}

// VisualizerPatternChangedMsg is emitted when the user cycles the visualizer
// pattern via the 'v' key. The root app handles this to persist the preference
// via PreferenceStore.
type VisualizerPatternChangedMsg struct {
	PatternIndex int
}

// SetVisualizerPattern sets the visualizer engine to a specific pattern index.
// Used at startup to restore the saved preference from config.
// Delegates directly to engine.SetPattern which wraps out-of-range values with modulo.
func (p *NowPlayingPane) SetVisualizerPattern(index int) {
	p.engine.SetPattern(index)
}

// VisualizerPattern returns the current visualizer pattern index.
// Used by tests and the app layer to read back the active pattern.
func (p *NowPlayingPane) VisualizerPattern() int {
	return p.engine.Pattern()
}

// SetTheme updates the theme reference for runtime theme switching.
// NowPlayingPane propagates the new theme to its sub-components.
func (p *NowPlayingPane) SetTheme(th theme.Theme) {
	// Save the current pattern index so theme changes don't reset the user's choice.
	savedPattern := p.engine.Pattern()
	p.theme = th
	p.infoBox = components.NewInfoBox(th)
	p.engine = viz.NewEngine(th)
	p.engine.SetPattern(savedPattern)
	p.seekBar = components.NewGradientSeekBar(th)
	p.volumeBar = components.NewGradientVolumeBar(th)
	p.volumeBar.SetConfirmed(confirmedVolume(p.store))
	// Propagate dimensions to newly created sub-components.
	p.SetSize(p.width, p.height)
}

// contentWidth returns the inner content width (pane width minus border chrome).
func (p *NowPlayingPane) contentWidth() int { return paneMax(p.width-4, 10) }

// vizRows returns the number of terminal rows allocated to the overlay body
// (pane height minus 2 padding rows).
func (p *NowPlayingPane) vizRows() int {
	return paneMax(p.height-2*npPadV, 4)
}

// paneMax returns the larger of two ints.
func paneMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// confirmedVolume reads the active device's volume from the store.
// Returns 0 when playback state or device info is unavailable.
func confirmedVolume(s state.StateReader) int {
	if ps := s.PlaybackState(); ps != nil && ps.Device != nil {
		return ps.Device.VolumePercent
	}
	return 0
}

// DeviceName returns the currently active device name from the store.
// Used by the root app's header bar.
func DeviceName(store state.StateReader) string {
	device := store.ActiveDevice()
	if device == nil {
		return ""
	}
	return fmt.Sprintf("  %s", device.Name)
}
