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

// SetSize updates the pane's dimensions and recomputes the adaptive layout
// matching NowPlaying's InfoBox sizing pattern.
func (p *PodcastPlaybackPane) SetSize(width, height int) {
	p.BasePane.SetSize(width, height)
	cw := width
	if cw < 10 {
		cw = 10
	}
	infoWidth := cw / 3
	if height <= 8 {
		infoWidth = cw / 2
	}
	if infoWidth < 28 {
		infoWidth = 28
	}
	detailsWidth := cw - infoWidth - 1
	if detailsWidth < 10 {
		infoWidth = 0
		detailsWidth = cw
	}
	p.infoWidth = infoWidth
	p.detailsWidth = detailsWidth
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
			p.localProgressMs = target
			p.seekBar.SetPositionConfirmed(target)
		}
		return p, nil

	case msg.Type == tea.KeyRight:
		ps := p.store.PlaybackState()
		if ps != nil && ps.Episode != nil {
			target := p.localProgressMs + 5000
			if target > ps.Episode.DurationMs {
				target = ps.Episode.DurationMs
			}
			p.localProgressMs = target
			p.seekBar.SetPositionConfirmed(target)
		}
		return p, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "+":
		return p, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "-":
		return p, nil

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
	lines := []string{episode.Name}
	if show != nil {
		lines = append(lines, show.Name)
	}
	lines = append(lines, "")
	ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)
	lines = append(lines, ctrl.Render())
	p.volumeBar.SetWidth(p.infoWidth - 4)
	lines = append(lines, p.volumeBar.Render())

	b := components.NewInfoBox(p.theme)
	b.SetSize(p.infoWidth, p.height)
	return b.Render("Episode Info", lines, p.focused)
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
	if episode.Description != "" && descMaxLines > 0 {
		descLines := strings.Split(episode.Description, "\n")
		for i, dl := range descLines {
			if i >= descMaxLines {
				descLines = descLines[:descMaxLines]
				ell := uikit.GlyphFor(uikit.GlyphEllipsis, uikit.ActiveMode())
				descLines[len(descLines)-1] = truncateStr(descLines[len(descLines)-1], p.detailsWidth-len([]rune(ell))) + ell
				break
			}
			descLines[i] = truncateStr(dl, p.detailsWidth)
		}
		for _, l := range descLines {
			lines = append(lines, mutedStyle.Width(p.detailsWidth).Render(l))
		}
	}

	for len(lines) < p.height-1 {
		lines = append(lines, "")
	}

	p.seekBar.SetWidth(p.detailsWidth)
	progressLine := p.seekBar.Render(p.localProgressMs, episode.DurationMs)

	if len(lines) > 0 {
		lines[len(lines)-1] = mutedStyle.Width(p.detailsWidth).Render(progressLine)
	} else {
		lines = append(lines, mutedStyle.Width(p.detailsWidth).Render(progressLine))
	}

	return strings.Join(lines, "\n")
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
