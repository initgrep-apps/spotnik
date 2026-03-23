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
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// PlayerPane is the center pane Bubble Tea model.
// It renders the currently playing track and handles playback key events.
// It reads all state from the Store; it never stores API data in its own fields.
type PlayerPane struct {
	store *state.Store
	theme theme.Theme

	focused bool
	width   int
	height  int

	// localProgressMs is pane-local state (not in Store). It increments by 1000ms
	// on each tick when playing, for smooth seek bar updates between polls.
	localProgressMs int
}

// NewPlayerPane creates a PlayerPane with the given store and theme.
func NewPlayerPane(s *state.Store, t theme.Theme, focused bool) *PlayerPane {
	return &PlayerPane{
		store:   s,
		theme:   t,
		focused: focused,
	}
}

// SetSize updates the pane's dimensions (called by the root model on WindowSizeMsg).
func (p *PlayerPane) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetFocused updates the focused state.
func (p *PlayerPane) SetFocused(focused bool) {
	p.focused = focused
}

// IsFocused returns whether the pane currently has focus.
func (p *PlayerPane) IsFocused() bool {
	return p.focused
}

// Init is a no-op for PlayerPane; the root app model manages the tick loop.
func (p *PlayerPane) Init() tea.Cmd {
	return nil
}

// Update handles all messages for the PlayerPane.
func (p *PlayerPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case TickMsg:
		return p.handleTick()

	case PlaybackStateFetchedMsg:
		return p.handlePlaybackFetched()

	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		return p.handleKey(m)
	}

	return p, nil
}

// View renders the player pane. It reads from the store and never calls the API.
func (p *PlayerPane) View() string {
	ps := p.store.PlaybackState()
	if ps == nil || ps.Item == nil {
		return p.renderEmpty()
	}

	t := ps.Item

	headerStyle := lipgloss.NewStyle().Foreground(p.theme.SectionHeader()).Bold(true)
	primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary()).Bold(true)
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	header := headerStyle.Render("NOW PLAYING")
	divider := strings.Repeat("─", paneMax(p.width-4, 20))

	artistNames := make([]string, len(t.Artists))
	for i, a := range t.Artists {
		artistNames[i] = a.Name
	}

	trackName := primaryStyle.Render(t.Name)
	artist := secondaryStyle.Render(strings.Join(artistNames, ", "))
	album := mutedStyle.Render(t.Album.Name)

	// Seek bar uses localProgressMs for smooth interpolation between polls.
	barWidth := p.width - 4
	if barWidth < 10 {
		barWidth = 10
	}
	pb := components.NewProgressBar(barWidth, p.theme)
	seekBar := pb.Render(p.localProgressMs, t.DurationMs)

	// Transport controls.
	volume := 0
	if ps.Device != nil {
		volume = ps.Device.VolumePercent
	}
	ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)
	vb := components.NewVolumeBar(p.theme)

	lines := []string{
		header,
		divider,
		"",
		trackName,
		artist,
		album,
		"",
		seekBar,
		"",
		ctrl.Render(),
		divider,
		vb.Render(volume),
		"",
	}

	return strings.Join(lines, "\n")
}

// renderEmpty shows the "Nothing playing" empty state.
func (p *PlayerPane) renderEmpty() string {
	headerStyle := lipgloss.NewStyle().Foreground(p.theme.SectionHeader()).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	header := headerStyle.Render("NOW PLAYING")
	divider := strings.Repeat("─", paneMax(p.width-4, 20))

	lines := []string{
		header,
		divider,
		"",
		mutedStyle.Render("        Nothing playing"),
		"",
		mutedStyle.Render("   Open Spotify on a device"),
		mutedStyle.Render("   and start playing music"),
		"",
	}

	return strings.Join(lines, "\n")
}

// handleTick processes a TickMsg: increments local progress when playing.
func (p *PlayerPane) handleTick() (*PlayerPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil && ps.IsPlaying {
		p.localProgressMs += 1000
	}
	return p, nil
}

// handlePlaybackFetched processes notification that the store has fresh playback state.
// It resets localProgressMs to the server value read from the store.
func (p *PlayerPane) handlePlaybackFetched() (*PlayerPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil {
		p.localProgressMs = ps.ProgressMs
	} else {
		p.localProgressMs = 0
	}
	return p, nil
}

// handleKey dispatches key events to playback request messages.
// The root app model receives these and dispatches the actual API calls.
func (p *PlayerPane) handleKey(msg tea.KeyMsg) (*PlayerPane, tea.Cmd) {
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

// emitPlaybackRequest returns a command that immediately emits a PlaybackRequestMsg.
// The root app model receives this and dispatches the corresponding Spotify API call.
func emitPlaybackRequest(action PlaybackAction) tea.Cmd {
	return func() tea.Msg {
		return PlaybackRequestMsg{Action: action}
	}
}

// nextRepeatMode returns the next repeat mode in the cycle off→context→track→off.
func nextRepeatMode(current string) string {
	switch current {
	case "off":
		return "context"
	case "context":
		return "track"
	default:
		return "off"
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
