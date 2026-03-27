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
//
// Layout: horizontal split with InfoBox (left, ~1/4 width) and Visualizer (right, ~3/4 width),
// and a gradient seek bar spanning full width at the bottom.
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

	// visualizer is the animated braille audio spectrum (right side of the split).
	visualizer *components.Visualizer

	// seekBar is the gradient seek bar spanning full width at the bottom.
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
		store:      s,
		theme:      t,
		focused:    focused,
		infoBox:    components.NewInfoBox(t),
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
// The content area is divided: InfoBox takes ~1/4 of width (min 28 chars) on the left,
// Visualizer takes the remaining ~3/4 on the right, and the seek bar spans the full width
// at the bottom.
func (p *NowPlayingPane) SetSize(width, height int) {
	p.width = width
	p.height = height

	contentWidth := paneMax(width-4, 10)

	// Split layout dimensions.
	infoWidth := paneMax(contentWidth/4, 28) // minimum 28 chars for controls
	vizWidth := contentWidth - infoWidth - 1 // -1 for gap between regions

	progressHeight := 1
	bodyHeight := paneMax(height-4, 4) - progressHeight // subtract border + seek bar

	p.infoBox.SetSize(infoWidth, bodyHeight)
	p.visualizer.SetSize(vizWidth, bodyHeight)
	p.seekBar.SetWidth(contentWidth)
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
// Renders a horizontal split: InfoBox (track/artist/album/controls/volume) on the left,
// Visualizer on the right, with a gradient seek bar spanning the full width at the bottom.
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

	infoLines := []string{
		primaryStyle.Render(t.Name),
		secondaryStyle.Render(strings.Join(artistNames, ", ")),
		mutedStyle.Render(t.Album.Name),
		"",
		ctrl.Render(),
		p.volumeBar.Render(volume),
	}

	// Render InfoBox (left) and Visualizer (right) side by side.
	infoView := p.infoBox.Render("Track Info", infoLines, p.focused)
	vizView := p.visualizer.View()
	body := lipgloss.JoinHorizontal(lipgloss.Top, infoView, " ", vizView)

	// Seek bar spans full width at the bottom.
	seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)

	return lipgloss.JoinVertical(lipgloss.Left, body, seekBar)
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

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "v":
		// Cycle visualizer animation pattern locally — no API call needed.
		p.visualizer.CyclePattern()
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
