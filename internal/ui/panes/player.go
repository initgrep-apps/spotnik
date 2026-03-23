// Package panes contains the Bubble Tea pane models for the Spotnik TUI.
// Each pane reads from the central Store and returns tea.Cmds for side effects.
// Panes never call the API directly or import api/ — data flows via messages.
package panes

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TickMsg is sent every second by the polling tick loop.
// It drives both progress interpolation and playback state refresh.
type TickMsg struct{}

// PlaybackStateFetchedMsg carries the result of a GET /me/player poll.
// State is nil when Spotify returned 204 (nothing playing).
type PlaybackStateFetchedMsg struct {
	State *api.PlaybackState
}

// PlaybackCmdSentMsg is returned after any playback control command completes.
// Err is non-nil if the API returned an error.
type PlaybackCmdSentMsg struct {
	Err error
}

// playerAction identifies what kind of playback action was sent.
// Used internally to build the correct command.
type playerAction int

const (
	actionPause playerAction = iota
	actionPlay
	actionNext
	actionPrevious
	actionVolumeUp
	actionVolumeDown
	actionToggleShuffle
	actionCycleRepeat
)

// PlayerPane is the center pane Bubble Tea model.
// It renders the currently playing track and handles playback key events.
// It reads all state from the Store; it never stores API data in its own fields.
type PlayerPane struct {
	store  *state.Store
	theme  theme.Theme
	player *api.Player

	focused bool
	width   int
	height  int

	// localProgressMs is pane-local state (not in Store). It increments by 1000ms
	// on each tick when playing, for smooth seek bar updates between polls.
	localProgressMs int
}

// NewPlayerPane creates a PlayerPane with the given store and theme.
// The player field is nil initially — it is set by the root app via SetPlayer.
func NewPlayerPane(s *state.Store, t theme.Theme, focused bool) *PlayerPane {
	return &PlayerPane{
		store:   s,
		theme:   t,
		focused: focused,
	}
}

// SetPlayer injects the API player client into the pane.
// This is called by the root app model after construction.
func (p *PlayerPane) SetPlayer(player *api.Player) {
	p.player = player
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

// Init starts the tick loop.
// NOTE: The root app model is responsible for starting the tick; PlayerPane's
// Init() is a no-op so it can be used standalone in tests.
func (p *PlayerPane) Init() tea.Cmd {
	return nil
}

// Update handles all messages for the PlayerPane.
func (p *PlayerPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case TickMsg:
		return p.handleTick()

	case PlaybackStateFetchedMsg:
		return p.handlePlaybackFetched(m)

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
	state := p.store.PlaybackState()
	if state == nil || state.Item == nil {
		return p.renderEmpty()
	}
	return p.renderNowPlaying(state)
}

// renderEmpty shows the "Nothing playing" empty state.
func (p *PlayerPane) renderEmpty() string {
	headerStyle := lipgloss.NewStyle().Foreground(p.theme.SectionHeader()).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	header := headerStyle.Render("NOW PLAYING")
	divider := strings.Repeat("─", max(p.width-4, 20))

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

// renderNowPlaying renders the full player UI for an active track.
func (p *PlayerPane) renderNowPlaying(ps *api.PlaybackState) string {
	t := ps.Item

	headerStyle := lipgloss.NewStyle().Foreground(p.theme.SectionHeader()).Bold(true)
	primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary()).Bold(true)
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	header := headerStyle.Render("NOW PLAYING")
	divider := strings.Repeat("─", max(p.width-4, 20))

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

// handleTick processes a TickMsg: increments local progress when playing.
func (p *PlayerPane) handleTick() (*PlayerPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil && ps.IsPlaying {
		p.localProgressMs += 1000
	}
	return p, nil
}

// handlePlaybackFetched processes a new playback state from the API.
// It updates the store and resets localProgressMs to the server value.
func (p *PlayerPane) handlePlaybackFetched(msg PlaybackStateFetchedMsg) (*PlayerPane, tea.Cmd) {
	p.store.SetPlaybackState(msg.State)
	if msg.State != nil {
		p.localProgressMs = msg.State.ProgressMs
	} else {
		p.localProgressMs = 0
	}
	return p, nil
}

// handleKey dispatches key events to the appropriate playback command.
func (p *PlayerPane) handleKey(msg tea.KeyMsg) (*PlayerPane, tea.Cmd) {
	ps := p.store.PlaybackState()

	switch {
	case msg.Type == tea.KeyRunes && string(msg.Runes) == " ":
		if ps != nil && ps.IsPlaying {
			return p, p.buildPlaybackCmd(actionPause)
		}
		return p, p.buildPlaybackCmd(actionPlay)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "n",
		msg.Type == tea.KeyRight:
		return p, p.buildPlaybackCmd(actionNext)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "p",
		msg.Type == tea.KeyLeft:
		return p, p.buildPlaybackCmd(actionPrevious)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "+":
		return p, p.buildPlaybackCmd(actionVolumeUp)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "-":
		return p, p.buildPlaybackCmd(actionVolumeDown)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "s":
		return p, p.buildPlaybackCmd(actionToggleShuffle)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "r":
		return p, p.buildPlaybackCmd(actionCycleRepeat)
	}

	return p, nil
}

// buildPlaybackCmd returns a tea.Cmd that calls the Spotify API for the given action.
// The command returns a PlaybackCmdSentMsg with any error.
// NOTE: If the player is nil (not wired in tests), the command returns a no-op message.
func (p *PlayerPane) buildPlaybackCmd(action playerAction) tea.Cmd {
	player := p.player
	store := p.store

	return func() tea.Msg {
		if player == nil {
			// NOTE: player not injected — return success msg so tests can verify routing.
			return PlaybackCmdSentMsg{}
		}

		ctx := context.Background()
		var err error

		switch action {
		case actionPause:
			err = player.Pause(ctx)

		case actionPlay:
			err = player.Play(ctx, api.PlayOptions{})

		case actionNext:
			err = player.Next(ctx)

		case actionPrevious:
			err = player.Previous(ctx)

		case actionVolumeUp:
			ps := store.PlaybackState()
			vol := 65 // default
			if ps != nil && ps.Device != nil {
				vol = ps.Device.VolumePercent
			}
			err = player.SetVolume(ctx, vol+5)

		case actionVolumeDown:
			ps := store.PlaybackState()
			vol := 65
			if ps != nil && ps.Device != nil {
				vol = ps.Device.VolumePercent
			}
			err = player.SetVolume(ctx, vol-5)

		case actionToggleShuffle:
			ps := store.PlaybackState()
			shuffle := false
			if ps != nil {
				shuffle = !ps.ShuffleState
			}
			err = player.SetShuffle(ctx, shuffle)

		case actionCycleRepeat:
			ps := store.PlaybackState()
			mode := "off"
			if ps != nil {
				mode = nextRepeatMode(ps.RepeatState)
			}
			err = player.SetRepeat(ctx, mode)
		}

		return PlaybackCmdSentMsg{Err: err}
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

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// fetchPlaybackStateCmd creates a command that fetches the current playback state.
// It is called by the root app model's tick handler.
func fetchPlaybackStateCmd(player *api.Player, store *state.Store) tea.Cmd {
	return func() tea.Msg {
		if player == nil {
			return PlaybackStateFetchedMsg{State: nil}
		}
		state, err := player.GetPlaybackState(context.Background())
		if err != nil {
			// NOTE: On error, we return nil state — the pane will show empty state.
			// Error logging / status bar update handled by the root model.
			return PlaybackStateFetchedMsg{State: nil}
		}
		store.SetPlaybackState(state)
		return PlaybackStateFetchedMsg{State: state}
	}
}

// FetchPlaybackStateCmd is the exported version for use by the root app model.
func FetchPlaybackStateCmd(player *api.Player, store *state.Store) tea.Cmd {
	return fetchPlaybackStateCmd(player, store)
}

// renderDeviceName returns the active device name, or empty string if none.
func renderDeviceName(store *state.Store) string {
	device := store.ActiveDevice()
	if device == nil {
		return ""
	}
	return fmt.Sprintf("  %s", device.Name)
}

// DeviceName returns the currently active device name from the store.
// Used by the root app's header bar.
func DeviceName(store *state.Store) string {
	return renderDeviceName(store)
}
