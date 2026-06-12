package panes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// PodcastPlaybackPane displays the currently playing podcast episode in a
// 30/70 vertical split. The left panel shows episode info, transport controls,
// and volume. The right panel shows metadata, description, and a progress bar.
type PodcastPlaybackPane struct {
	BasePane
	store           state.StateReader
	theme           theme.Theme
	localProgressMs int
	seekBar         *components.GradientSeekBar
	volumeBar       *components.GradientVolumeBar
	infoWidth       int
	detailsWidth    int
}

var _ layout.Pane = &PodcastPlaybackPane{}

// NewPodcastPlaybackPane creates a PodcastPlaybackPane with the given store and theme.
// localProgressMs is initialized from the store's current playback state.
func NewPodcastPlaybackPane(store state.StateReader, th theme.Theme, focused bool) *PodcastPlaybackPane {
	p := &PodcastPlaybackPane{
		BasePane:  BasePane{store: store, theme: th, focused: focused},
		store:     store,
		theme:     th,
		seekBar:   components.NewGradientSeekBar(th),
		volumeBar: components.NewGradientVolumeBar(th),
	}
	if ps := store.PlaybackState(); ps != nil {
		p.localProgressMs = ps.ProgressMs
		p.seekBar.SetPositionConfirmed(ps.ProgressMs)
		if ps.Episode != nil {
			p.seekBar.SetTrackDuration(ps.Episode.DurationMs)
		}
		if ps.Device != nil {
			p.volumeBar.SetConfirmed(ps.Device.VolumePercent)
		}
	}
	return p
}

// ID returns the PaneID for the PodcastPlayback slot.
func (p *PodcastPlaybackPane) ID() layout.PaneID { return layout.PanePodcastPlayback }

// Title returns the display title for the border.
func (p *PodcastPlaybackPane) Title() string { return "Now Playing" }

// ToggleKey returns the number key for btop-style pane toggling (key 1).
func (p *PodcastPlaybackPane) ToggleKey() int { return 1 }

// Actions returns pane-specific shortcuts shown in the border.
func (p *PodcastPlaybackPane) Actions() []layout.Action { return nil }

// SetTheme updates the theme reference for runtime theme switching.
func (p *PodcastPlaybackPane) SetTheme(th theme.Theme) {
	p.theme = th
	p.seekBar = components.NewGradientSeekBar(th)
	p.volumeBar = components.NewGradientVolumeBar(th)
	if ps := p.store.PlaybackState(); ps != nil {
		p.seekBar.SetPositionConfirmed(ps.ProgressMs)
		if ps.Episode != nil {
			p.seekBar.SetTrackDuration(ps.Episode.DurationMs)
		}
		if ps.Device != nil {
			p.volumeBar.SetConfirmed(ps.Device.VolumePercent)
		}
	}
	p.SetSize(p.width, p.height)
}

// SetSize updates the pane's dimensions and recomputes the 30/70 layout.
func (p *PodcastPlaybackPane) SetSize(width, height int) {
	p.BasePane.SetSize(width, height)
	cw := width
	if cw < 10 {
		cw = 10
	}
	p.infoWidth = cw * 30 / 100
	if p.infoWidth < 24 {
		p.infoWidth = 24
	}
	p.detailsWidth = cw - p.infoWidth - 1
}

// Init returns nil — no initial commands for this pane.
func (p *PodcastPlaybackPane) Init() tea.Cmd { return nil }

// Update handles messages for the PodcastPlaybackPane.
func (p *PodcastPlaybackPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
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

func (p *PodcastPlaybackPane) handlePlaybackFetched(_ PlaybackStateFetchedMsg) (*PodcastPlaybackPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil {
		p.localProgressMs = ps.ProgressMs
		p.seekBar.SetPositionConfirmed(ps.ProgressMs)
		if ps.Episode != nil {
			p.seekBar.SetTrackDuration(ps.Episode.DurationMs)
		}
		if ps.Device != nil {
			p.volumeBar.SetConfirmed(ps.Device.VolumePercent)
		}
	} else {
		p.localProgressMs = 0
	}
	return p, nil
}

func (p *PodcastPlaybackPane) handleKey(msg tea.KeyMsg) (*PodcastPlaybackPane, tea.Cmd) {
	switch {
	case msg.Type == tea.KeySpace:
		ps := p.store.PlaybackState()
		if ps != nil && ps.IsPlaying {
			return p, emitPlaybackRequest(ActionPause)
		}
		return p, emitPlaybackRequest(ActionPlay)

	case msg.Type == tea.KeyShiftLeft:
		return p, emitPlaybackRequest(ActionPrevious)

	case msg.Type == tea.KeyShiftRight:
		return p, emitPlaybackRequest(ActionNext)

	case msg.Type == tea.KeyLeft:
		ps := p.store.PlaybackState()
		if ps != nil && ps.Episode != nil {
			target := p.localProgressMs - 5000
			if target < 0 {
				target = 0
			}
			return p, func() tea.Msg {
				return SeekIntentMsg{TargetMs: target, Seq: 0}
			}
		}
		return p, nil

	case msg.Type == tea.KeyRight:
		ps := p.store.PlaybackState()
		if ps != nil && ps.Episode != nil {
			target := p.localProgressMs + 5000
			if target > ps.Episode.DurationMs {
				target = ps.Episode.DurationMs
			}
			return p, func() tea.Msg {
				return SeekIntentMsg{TargetMs: target, Seq: 0}
			}
		}
		return p, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "+":
		ps := p.store.PlaybackState()
		vol := 0
		if ps != nil && ps.Device != nil {
			vol = ps.Device.VolumePercent
		}
		vol += 5
		if vol > 100 {
			vol = 100
		}
		return p, func() tea.Msg {
			return VolumeIntentMsg{TargetVol: vol, Seq: 0}
		}

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "-":
		ps := p.store.PlaybackState()
		vol := 0
		if ps != nil && ps.Device != nil {
			vol = ps.Device.VolumePercent
		}
		vol -= 5
		if vol < 0 {
			vol = 0
		}
		return p, func() tea.Msg {
			return VolumeIntentMsg{TargetVol: vol, Seq: 0}
		}

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "s":
		return p, emitPlaybackRequest(ActionToggleShuffle)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "r":
		return p, emitPlaybackRequest(ActionCycleRepeat)
	}

	return p, nil
}

// View renders the PodcastPlaybackPane. It reads from the store and never
// calls the API. Dispatches to renderEmpty when nothing is playing or the
// current item is not an episode, otherwise renderEpisode.
func (p *PodcastPlaybackPane) View() string {
	ps := p.store.PlaybackState()
	if ps == nil || ps.CurrentlyPlayingType != "episode" || ps.Episode == nil {
		return p.renderEmpty()
	}
	return p.renderEpisode(ps.Episode, ps)
}

func (p *PodcastPlaybackPane) renderEmpty() string {
	return uikit.EmptyState{
		Text:   "No podcast playing",
		Hint:   "Press / to search for shows\nOr select a show from Followed Shows",
		Width:  p.width,
		Height: p.height,
		Theme:  p.theme,
	}.Render()
}

func (p *PodcastPlaybackPane) renderEpisode(episode *domain.Episode, ps *domain.PlaybackState) string {
	show := episode.Show

	left := p.renderLeftPanel(episode, show, ps)
	right := p.renderRightPanel(episode, show)

	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, strings.Repeat(" ", p.infoWidth))
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, strings.Repeat(" ", p.detailsWidth))
	}

	if len(leftLines) > p.height {
		leftLines = leftLines[:p.height]
	}
	if len(rightLines) > p.height {
		rightLines = rightLines[:p.height]
	}
	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, strings.Repeat(" ", p.infoWidth))
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, strings.Repeat(" ", p.detailsWidth))
	}

	gap := " "
	out := make([]string, len(leftLines))
	for i := range leftLines {
		out[i] = leftLines[i] + gap + rightLines[i]
	}
	return strings.Join(out, "\n")
}

func (p *PodcastPlaybackPane) renderLeftPanel(episode *domain.Episode, show *domain.Show, ps *domain.PlaybackState) string {
	innerW := p.infoWidth - 2

	primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary()).Bold(true)
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())

	titleLine := primaryStyle.Render(truncateStr(episode.Name, innerW))
	showLine := ""
	if show != nil {
		showLine = secondaryStyle.Render(truncateStr(show.Name, innerW))
	}

	ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)
	ctrlLine := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(ctrl.Render())

	p.volumeBar.SetWidth(p.infoWidth - 4)
	volLine := p.volumeBar.Render()

	content := lipgloss.JoinVertical(lipgloss.Left,
		"Episode Info",
		"",
		titleLine,
		showLine,
		"",
		ctrlLine,
		volLine,
	)

	borderColor := p.theme.ActiveBorder()
	if !p.focused {
		borderColor = p.theme.InactiveBorder()
	}
	borderStyle := lipgloss.NewStyle().
		Border(uikit.RoundedBorder()).
		BorderForeground(borderColor).
		Width(p.infoWidth)

	return borderStyle.Render(content)
}

func (p *PodcastPlaybackPane) renderRightPanel(episode *domain.Episode, show *domain.Show) string {
	var lines []string
	primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())

	durationStr := fmt.Sprintf("%dm", episode.DurationMs/60000)
	sep := uikit.GlyphFor(uikit.GlyphSeparator, uikit.ActiveMode())
	metaLine := fmt.Sprintf("Released: %s %s Duration: %s", episode.ReleaseDate, sep, durationStr)
	lines = append(lines, primaryStyle.Width(p.detailsWidth).Render(metaLine))

	if show != nil && show.Publisher != "" {
		pubLine := fmt.Sprintf("Publisher: %s", show.Publisher)
		lines = append(lines, secondaryStyle.Width(p.detailsWidth).Render(pubLine))
	}

	lines = append(lines, "")

	descMaxLines := p.height - len(lines) - 2
	if descMaxLines < 1 {
		descMaxLines = 1
	}
	if episode.Description != "" {
		rendered := mutedStyle.Width(p.detailsWidth).Render(episode.Description)
		descLines := strings.Split(rendered, "\n")
		if len(descLines) > descMaxLines {
			descLines = descLines[:descMaxLines]
		}
		lines = append(lines, descLines...)
	}

	for len(lines) < p.height-1 {
		lines = append(lines, "")
	}

	current := formatDurationMs(p.localProgressMs)
	total := formatDurationMs(episode.DurationMs)
	barWidth := p.detailsWidth - len(current) - len(total) - 8
	if barWidth < 3 {
		barWidth = 3
	}
	bar := renderProgressBar(p.localProgressMs, episode.DurationMs, barWidth)
	progressLine := fmt.Sprintf("-- %s %s%s %s %s%s %s --", current, sep, sep, bar, sep, sep, total)

	if len(lines) > 0 {
		lines[len(lines)-1] = mutedStyle.Width(p.detailsWidth).Render(progressLine)
	} else {
		lines = append(lines, mutedStyle.Width(p.detailsWidth).Render(progressLine))
	}

	return strings.Join(lines, "\n")
}

// renderProgressBar renders a simple progress bar with filled and empty blocks.
func renderProgressBar(progressMs, durationMs int, width int) string {
	var ratio float64
	if durationMs > 0 {
		ratio = float64(progressMs) / float64(durationMs)
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1.0 {
		ratio = 1.0
	}
	filled := int(ratio * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", width-filled)
}

func truncateStr(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width < 1 {
		return ""
	}
	ellipsis := uikit.GlyphFor(uikit.GlyphEllipsis, uikit.ActiveMode())
	ellipsisLen := len([]rune(ellipsis))
	if width <= ellipsisLen {
		return ellipsis
	}
	runes := []rune(s)
	keep := width - ellipsisLen
	return string(runes[:keep]) + ellipsis
}
