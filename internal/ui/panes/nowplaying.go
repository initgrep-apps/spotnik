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
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// NowPlayingPane is the center pane Bubble Tea model.
// It renders the currently playing track and handles playback key events.
// It reads all state from the Store; it never stores API data in its own fields.
// It implements the layout.Pane interface for integration with the layout manager.
type NowPlayingPane struct {
	store *state.Store
	theme theme.Theme

	focused bool
	width   int
	height  int
	compact bool // true when height <= 3 (compact single-line strip mode)

	// localProgressMs is pane-local state (not in Store). It increments by 1000ms
	// on each tick when playing, for smooth seek bar updates between polls.
	localProgressMs int

	// visualizer is the animated braille audio spectrum.
	visualizer *components.Visualizer

	// seekBar is the gradient seek bar (replaces monochrome ProgressBar).
	seekBar *components.GradientSeekBar

	// volumeBar is the gradient volume bar (replaces monochrome VolumeBar).
	volumeBar *components.GradientVolumeBar
}

// Compile-time check: NowPlayingPane implements layout.Pane.
var _ layout.Pane = &NowPlayingPane{}

// NewNowPlayingPane creates a NowPlayingPane with the given store and theme.
// localProgressMs is initialized from the store's current playback state so that
// constructing a pane after setting state shows the correct position immediately.
func NewNowPlayingPane(s *state.Store, t theme.Theme, focused bool) *NowPlayingPane {
	p := &NowPlayingPane{
		store:      s,
		theme:      t,
		focused:    focused,
		visualizer: components.NewVisualizer(t),
		seekBar:    components.NewGradientSeekBar(t),
		volumeBar:  components.NewGradientVolumeBar(t),
	}
	if ps := s.PlaybackState(); ps != nil {
		p.localProgressMs = ps.ProgressMs
		p.visualizer.SetPlaying(ps.IsPlaying)
	}
	return p
}

// ID returns the PaneID for the NowPlaying slot.
func (p *NowPlayingPane) ID() layout.PaneID {
	return layout.PaneNowPlaying
}

// Title returns the display title for the border, including track info in compact mode.
func (p *NowPlayingPane) Title() string {
	if p.compact {
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
		{Key: "s", Label: "shuffle"},
		{Key: "r", Label: "repeat"},
	}
}

// SetSize updates the pane's dimensions (called by the root model on WindowSizeMsg).
// Compact mode is enabled when height <= 3 (border + 1 content line).
func (p *NowPlayingPane) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.compact = height <= 3

	contentWidth := paneMax(width-4, 10)
	p.seekBar.SetWidth(contentWidth)
	p.volumeBar.SetWidth(contentWidth)

	// Visualizer height depends on mode:
	//   compact: 1 line (kept for sizing but not shown)
	//   full: 2 lines
	//   expanded (height >= 10): 4 lines
	vizHeight := 2
	if p.compact {
		vizHeight = 1
	} else if height >= 10 {
		vizHeight = 4
	}
	p.visualizer.SetSize(contentWidth, vizHeight)
}

// SetFocused updates the focused state.
func (p *NowPlayingPane) SetFocused(focused bool) {
	p.focused = focused
}

// IsFocused returns whether the pane currently has focus.
func (p *NowPlayingPane) IsFocused() bool {
	return p.focused
}

// Init starts the visualizer animation tick loop.
func (p *NowPlayingPane) Init() tea.Cmd {
	return p.visualizer.Init()
}

// Update handles all messages for the NowPlayingPane.
func (p *NowPlayingPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case TickMsg:
		return p.handleTick()

	case PlaybackStateFetchedMsg:
		return p.handlePlaybackFetched()

	case components.VisualizerTickMsg:
		cmd := p.visualizer.Update(m)
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
// In compact mode (height <= 3), renders a single content line with controls.
// In full mode, renders visualizer + track info + seek bar + controls + volume bar.
func (p *NowPlayingPane) View() string {
	if p.compact {
		return p.renderCompact()
	}
	return p.renderFull()
}

// renderFull renders the full-mode view with visualizer, track info, bars, and controls.
func (p *NowPlayingPane) renderFull() string {
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

	trackName := primaryStyle.Render(t.Name)
	artist := secondaryStyle.Render(strings.Join(artistNames, ", "))
	album := mutedStyle.Render(t.Album.Name)

	// Visualizer.
	vizView := p.visualizer.View()

	// Seek bar.
	seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)

	// Transport controls.
	volume := 0
	if ps.Device != nil {
		volume = ps.Device.VolumePercent
	}
	ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)
	volBar := p.volumeBar.Render(volume)

	lines := []string{
		vizView,
		"",
		trackName,
		artist,
		album,
		"",
		seekBar,
		"",
		ctrl.Render(),
		volBar,
		"",
	}

	return strings.Join(lines, "\n")
}

// renderCompact renders a single-line strip for small presets.
// Content line: seek bar gradient + controls + volume bar inline.
func (p *NowPlayingPane) renderCompact() string {
	actualPS := p.store.PlaybackState()
	if actualPS == nil || actualPS.Item == nil {
		mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
		return mutedStyle.Render("Nothing playing")
	}

	// Single line: seek bar + controls + volume.
	volume := 0
	if actualPS.Device != nil {
		volume = actualPS.Device.VolumePercent
	}

	// Compact seek bar: just the fill bar without time labels.
	barWidth := paneMax(p.width/3, 10)
	ratio := 0.0
	if actualPS.Item.DurationMs > 0 {
		ratio = float64(p.localProgressMs) / float64(actualPS.Item.DurationMs)
		if ratio > 1.0 {
			ratio = 1.0
		}
	}
	fillCount := int(ratio * float64(barWidth))
	emptyCount := barWidth - fillCount

	g1 := string(p.theme.Gradient1())
	g2 := string(p.theme.Gradient2())
	emptyStyle := lipgloss.NewStyle().Foreground(p.theme.Surface())
	var seekSB strings.Builder
	for i := 0; i < fillCount; i++ {
		var tt float64
		if fillCount > 1 {
			tt = float64(i) / float64(fillCount-1)
		}
		col := interpolateHexCompact(g1, g2, tt)
		seekSB.WriteString(lipgloss.NewStyle().Foreground(col).Render("█"))
	}
	seekSB.WriteString(emptyStyle.Render(strings.Repeat("░", emptyCount)))

	ctrl := components.NewControls(p.theme, actualPS.IsPlaying, actualPS.ShuffleState, actualPS.RepeatState)
	volBar := p.volumeBar.Render(volume)

	return seekSB.String() + "  " + ctrl.Render() + "   " + volBar
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
func (p *NowPlayingPane) handleTick() (*NowPlayingPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil && ps.IsPlaying {
		p.localProgressMs += 1000
	}
	return p, nil
}

// handlePlaybackFetched processes notification that the store has fresh playback state.
// It resets localProgressMs to the server value and syncs visualizer playing state.
func (p *NowPlayingPane) handlePlaybackFetched() (*NowPlayingPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil {
		p.localProgressMs = ps.ProgressMs
		p.visualizer.SetPlaying(ps.IsPlaying)
	} else {
		p.localProgressMs = 0
		p.visualizer.SetPlaying(false)
	}
	return p, nil
}

// handleKey dispatches key events to playback request messages.
// The root app model receives these and dispatches the actual API calls.
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

// interpolateHexCompact is a thin wrapper around the gradient interpolation for compact mode.
// Defined here to avoid importing components for this internal function.
func interpolateHexCompact(g1, g2 string, t float64) lipgloss.Color {
	// Use the gradient.go helper via the GradientSeekBar's Render indirectly.
	// Since interpolateHex is package-private in components, we duplicate the
	// minimal logic here for the compact seek bar.
	if t <= 0 {
		return lipgloss.Color(g1)
	}
	if t >= 1 {
		return lipgloss.Color(g2)
	}
	r1, g1r, b1 := parseHexParts(g1)
	r2, g2r, b2 := parseHexParts(g2)
	r := lerpByte(r1, r2, t)
	g := lerpByte(g1r, g2r, t)
	b := lerpByte(b1, b2, t)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

// parseHexParts parses a "#rrggbb" hex string into r, g, b uint8 components.
func parseHexParts(hex string) (r, g, b uint8) {
	s := strings.TrimPrefix(hex, "#")
	if len(s) != 6 {
		return 0, 0, 0
	}
	parse := func(sub string) uint8 {
		var v uint64
		for _, c := range sub {
			v <<= 4
			switch {
			case c >= '0' && c <= '9':
				v |= uint64(c - '0')
			case c >= 'a' && c <= 'f':
				v |= uint64(c-'a') + 10
			case c >= 'A' && c <= 'F':
				v |= uint64(c-'A') + 10
			}
		}
		return uint8(v)
	}
	return parse(s[0:2]), parse(s[2:4]), parse(s[4:6])
}

// lerpByte linearly interpolates between two uint8 values.
func lerpByte(a, b uint8, t float64) uint8 {
	result := float64(a) + t*(float64(b)-float64(a))
	if result < 0 {
		return 0
	}
	if result > 255 {
		return 255
	}
	return uint8(result + 0.5)
}

// emitPlaybackRequest returns a command that immediately emits a PlaybackRequestMsg.
// The root app model receives this and dispatches the corresponding Spotify API call.
func emitPlaybackRequest(action PlaybackAction) tea.Cmd {
	return func() tea.Msg {
		return PlaybackRequestMsg{Action: action}
	}
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
