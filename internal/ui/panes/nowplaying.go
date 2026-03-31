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
)

// NowPlayingPane is the center pane Bubble Tea model.
// It renders the currently playing track and handles playback key events.
// It reads all state from the Store; it never stores API data in its own fields.
// It implements the layout.Pane interface for integration with the layout manager.
//
// Layout: horizontal split with InfoBox (left, ~1/3 width) and viz engine (right, ~2/3 width).
// The right panel contains: top viz rows, seek bar, bottom viz rows.
// When height < 8, Title() embeds compact track info in the pane title bar instead.
type NowPlayingPane struct {
	store *state.Store
	theme theme.Theme

	focused bool
	width   int
	height  int

	// localProgressMs is pane-local state (not in Store). It increments by 1000ms
	// on each tick when playing, for smooth seek bar updates between polls.
	localProgressMs int

	// infoBox is the bordered sub-pane on the left showing track/artist/album/controls.
	infoBox *components.InfoBox

	// engine is the animated visualization engine (right side of the split).
	engine *viz.Engine

	// seekBar is the gradient seek bar rendered inside the right panel.
	seekBar *components.GradientSeekBar

	// volumeBar is the gradient volume bar rendered inside the InfoBox.
	volumeBar *components.GradientVolumeBar
}

// Compile-time check: NowPlayingPane implements layout.Pane.
var _ layout.Pane = &NowPlayingPane{}

// NewNowPlayingPane creates a NowPlayingPane with the given store and theme.
// localProgressMs is initialized from the store's current playback state so that
// constructing a pane after setting state shows the correct position immediately.
func NewNowPlayingPane(s *state.Store, t theme.Theme, focused bool) *NowPlayingPane {
	p := &NowPlayingPane{
		store:     s,
		theme:     t,
		focused:   focused,
		infoBox:   components.NewInfoBox(t),
		engine:    viz.NewEngine(t),
		seekBar:   components.NewGradientSeekBar(t),
		volumeBar: components.NewGradientVolumeBar(t),
	}
	if ps := s.PlaybackState(); ps != nil {
		p.localProgressMs = ps.ProgressMs
		p.engine.SetPlaying(ps.IsPlaying)
	}
	return p
}

// ID returns the PaneID for the NowPlaying slot.
func (p *NowPlayingPane) ID() layout.PaneID {
	return layout.PaneNowPlaying
}

// Title returns the display title for the border.
// When height < 8 (pane too small for the split body), the title embeds track info
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
			playSymbol := "▶"
			if !ps.IsPlaying {
				playSymbol = "⏸"
			}
			current := formatDurationMs(p.localProgressMs)
			total := formatDurationMs(t.DurationMs)
			return fmt.Sprintf("Now Playing \u2500\u2500 %s \u00b7 %s \u2500\u2500 %s %s/%s",
				t.Name, strings.Join(artistNames, ", "), playSymbol, current, total)
		}
	}
	return "Now Playing"
}

// ToggleKey returns the number key for btop-style pane toggling (key 1).
func (p *NowPlayingPane) ToggleKey() int {
	return 1
}

// Actions returns the pane-specific shortcuts shown in the border.
func (p *NowPlayingPane) Actions() []layout.Action {
	return []layout.Action{
		{Key: "s", Label: "shfl"},
		{Key: "r", Label: "rpt"},
		{Key: "space", Label: "play"},
		{Key: "+/-", Label: "vol"},
		{Key: "v", Label: "viz"},
	}
}

// SetSize updates the pane's dimensions and recomputes the split layout geometry.
// Two-column split: InfoBox (~1/3 width, min 28 chars) on the left,
// viz engine (~2/3 width) on the right.
// The right panel shows: top viz rows, seek bar (1 row), bottom viz rows.
func (p *NowPlayingPane) SetSize(width, height int) {
	p.width = width
	p.height = height

	contentWidth := paneMax(width-4, 10)

	// Two-column split: ~1/3 left (min 28), ~2/3 right.
	infoWidth := paneMax(contentWidth/3, 28)
	vizWidth := contentWidth - infoWidth - 1 // -1 for gap between regions
	if vizWidth < 1 {
		vizWidth = 1
	}

	bodyHeight := paneMax(height-4, 4)

	p.infoBox.SetSize(infoWidth, bodyHeight)

	// Viz rows split around the seek bar (1 row).
	vizHeight := paneMax(bodyHeight-1, 1)
	p.engine.SetSize(vizWidth, vizHeight)

	// Seek bar sized to right panel width (not full content width).
	p.seekBar.SetWidth(vizWidth)
	p.volumeBar.SetWidth(infoWidth - 4) // fits inside InfoBox with border padding
}

// SetFocused updates the focused state.
func (p *NowPlayingPane) SetFocused(focused bool) {
	p.focused = focused
}

// IsFocused returns whether the pane currently has focus.
func (p *NowPlayingPane) IsFocused() bool {
	return p.focused
}

// Init starts the viz engine animation tick loop.
func (p *NowPlayingPane) Init() tea.Cmd {
	return p.engine.Init()
}

// Update handles all messages for the NowPlayingPane.
func (p *NowPlayingPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case TickMsg:
		return p.handleTick()

	case PlaybackStateFetchedMsg:
		return p.handlePlaybackFetched()

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
// Layout: InfoBox (left, ~1/3 width) and viz+seekbar (right, ~2/3 width).
// Right panel: top viz rows, seek bar (1 row), bottom viz rows.
// In expanded mode (height > content), the composite is vertically centered.
func (p *NowPlayingPane) View() string {
	ps := p.store.PlaybackState()
	if ps == nil || ps.Item == nil {
		return p.renderEmpty()
	}

	t := ps.Item

	primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary()).Bold(true)
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	artistNames := make([]string, len(t.Artists))
	for i, a := range t.Artists {
		artistNames[i] = a.Name
	}

	volume := 0
	if ps.Device != nil {
		volume = ps.Device.VolumePercent
	}
	ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)

	// Compute available inner height to decide which info lines to include.
	// This mirrors SetSize: bodyHeight = max(height-4, 4), innerH = bodyHeight-2.
	bodyHeight := paneMax(p.height-4, 4)
	innerH := bodyHeight - 2 // InfoBox border rows consume 2

	// Priority: always show track name, controls, and volume bar (essential).
	// Add artists, album, spacer as space allows.
	var infoLines []string
	switch {
	case innerH >= 6:
		// Full layout: track, artists, album, spacer, controls, volume.
		infoLines = []string{
			primaryStyle.Render(t.Name),
			secondaryStyle.Render(strings.Join(artistNames, ", ")),
			mutedStyle.Render(t.Album.Name),
			"",
			ctrl.Render(),
			p.volumeBar.Render(volume),
		}
	case innerH >= 5:
		// Drop spacer: track, artists, album, controls, volume.
		infoLines = []string{
			primaryStyle.Render(t.Name),
			secondaryStyle.Render(strings.Join(artistNames, ", ")),
			mutedStyle.Render(t.Album.Name),
			ctrl.Render(),
			p.volumeBar.Render(volume),
		}
	case innerH >= 4:
		// Drop album: track, artists, controls, volume.
		infoLines = []string{
			primaryStyle.Render(t.Name),
			secondaryStyle.Render(strings.Join(artistNames, ", ")),
			ctrl.Render(),
			p.volumeBar.Render(volume),
		}
	case innerH >= 3:
		// Drop artists: track, controls, volume.
		infoLines = []string{
			primaryStyle.Render(t.Name),
			ctrl.Render(),
			p.volumeBar.Render(volume),
		}
	default:
		// Minimal: track and controls only (no room for volume bar).
		infoLines = []string{
			primaryStyle.Render(t.Name),
			ctrl.Render(),
		}
	}

	infoView := p.infoBox.Render("Track Info", infoLines, p.focused)

	// Right panel: viz top rows + seek bar + viz bottom rows.
	frame := p.engine.CurrentFrame()
	topRows, bottomRows := splitFrame(frame)
	topView := renderStyledLines(topRows)
	bottomView := renderStyledLines(bottomRows)
	seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)

	rightPanel := lipgloss.JoinVertical(lipgloss.Left, topView, seekBar, bottomView)

	composite := lipgloss.JoinHorizontal(lipgloss.Top, infoView, " ", rightPanel)

	// If pane is taller than the content, vertically center the block.
	contentHeight := lipgloss.Height(composite)
	availableHeight := paneMax(p.height-2, 1) // subtract border chrome
	if contentHeight < availableHeight {
		contentWidth := paneMax(p.width-4, 10)
		composite = lipgloss.Place(contentWidth, availableHeight,
			lipgloss.Center, lipgloss.Center, composite)
	}

	return composite
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
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	content := lipgloss.JoinVertical(lipgloss.Center,
		mutedStyle.Render("Nothing playing"),
		"",
		mutedStyle.Render("Open Spotify on a device"),
		mutedStyle.Render("and start playing music"),
	)

	// Center the content block within the available pane dimensions.
	bodyHeight := paneMax(p.height-4, 6) // subtract header + padding
	centered := lipgloss.Place(paneMax(p.width-4, 20), bodyHeight,
		lipgloss.Center, lipgloss.Center, content)

	return centered
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
// It resets localProgressMs to the server value and syncs engine playing state.
func (p *NowPlayingPane) handlePlaybackFetched() (*NowPlayingPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil {
		p.localProgressMs = ps.ProgressMs
		p.engine.SetPlaying(ps.IsPlaying)
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
	case msg.Type == tea.KeyRunes && string(msg.Runes) == " ":
		ps := p.store.PlaybackState()
		if ps != nil && ps.IsPlaying {
			return p, emitPlaybackRequest(ActionPause)
		}
		return p, emitPlaybackRequest(ActionPlay)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "n",
		msg.Type == tea.KeyRight:
		return p, emitPlaybackRequest(ActionNext)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "p",
		msg.Type == tea.KeyLeft:
		return p, emitPlaybackRequest(ActionPrevious)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "+":
		return p, emitPlaybackRequest(ActionVolumeUp)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "-":
		return p, emitPlaybackRequest(ActionVolumeDown)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "s":
		return p, emitPlaybackRequest(ActionToggleShuffle)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "r":
		return p, emitPlaybackRequest(ActionCycleRepeat)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "v":
		// Cycle engine animation pattern locally — no API call needed.
		p.engine.CyclePattern()
		return p, nil
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

// SetTheme updates the theme reference for runtime theme switching.
// NowPlayingPane propagates the new theme to its sub-components.
func (p *NowPlayingPane) SetTheme(th theme.Theme) {
	p.theme = th
	p.infoBox = components.NewInfoBox(th)
	p.engine = viz.NewEngine(th)
	p.seekBar = components.NewGradientSeekBar(th)
	p.volumeBar = components.NewGradientVolumeBar(th)
	// Propagate dimensions to newly created sub-components.
	p.SetSize(p.width, p.height)
}

// paneMax returns the larger of two ints.
func paneMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DeviceName returns the currently active device name from the store.
// Used by the root app's header bar.
func DeviceName(store *state.Store) string {
	device := store.ActiveDevice()
	if device == nil {
		return ""
	}
	return fmt.Sprintf("  %s", device.Name)
}
