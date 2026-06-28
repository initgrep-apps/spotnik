// Package panes contains the Bubble Tea pane models for the Spotnik TUI.
// Each pane reads from the central Store and emits request messages for side effects.
// Panes never call the API directly or import api/ — data flows through messages and store only.
package panes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/components/viz"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// NowPlaying side-by-side layout constants.
//
// The pane is split into two columns: InfoBox on the left, visualizer on the
// right. The seek bar lives in its own row at the bottom of the right column.
const (
	npPadV         = 1  // vertical padding rows (top + bottom)
	npInfoPctTall  = 3  // tall pane: InfoBox = cw / npInfoPctTall
	npInfoPctShort = 2  // short pane: InfoBox = cw / npInfoPctShort
	npInfoMin      = 28 // minimum InfoBox width for controls + volume
	npGap          = 1  // column gap between InfoBox and visualizer
	npMinViz       = 10 // below this, drop InfoBox entirely
	npMaxContentH  = 24 // cap content height when pane is oversized
	npInfoPadLeft  = 2  // left padding columns for InfoBox text lines
)

// NowPlayingPane is the center pane Bubble Tea model.
// It renders the currently playing track info overlay on top of a full-pane
// visualizer background. It reads all state from the Store; it never stores
// API data in its own fields.
// It implements the layout.Pane interface for integration with the layout manager.
//
// Layout: visualizer fills the full content area; InfoBox overlays the left
// ~25-33%. Seek bar lives in the visualizer column only (right of the gap).
// When width is too narrow, the InfoBox is dropped and the visualizer fills the
// full content area. When height < 8, Title() embeds compact track info in the
// pane title bar instead.
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
		p.seekBar.SetPositionConfirmed(ps.ProgressMs)
		durationMs := p.playingDurationMs(ps)
		if durationMs > 0 {
			p.seekBar.SetTrackDuration(durationMs)
		}
	}
	return p
}

// ID returns the PaneID for the NowPlaying slot.
func (p *NowPlayingPane) ID() layout.PaneID {
	return layout.PaneNowPlaying
}

// Title returns the display title for the border.
// When height < 8 (pane too small for the overlay body), the title embeds track/episode
// info so the user can still see what's playing without any content area.
// When an episode is playing, the title includes a ⏵ Podcast notch.
func (p *NowPlayingPane) Title() string {
	ps := p.store.PlaybackState()
	if ps == nil {
		return "Now Playing"
	}

	switch ps.CurrentlyPlayingType {
	case "episode":
		if ps.Episode == nil {
			return "Now Playing"
		}
		ep := ps.Episode
		podcastNotch := " ⏵ Podcast"
		if p.height < 8 {
			showName := ""
			if ep.Show != nil {
				showName = ep.Show.Name
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
			total := formatDurationMs(ep.DurationMs)
			return fmt.Sprintf("Now Playing%s %s %s %s %s %s %s %s/%s",
				podcastNotch, sep, ep.Name, midDot, showName, sep, stateGlyph, current, total)
		}
		return fmt.Sprintf("Now Playing%s", podcastNotch)

	case "track":
		if ps.Item == nil {
			return "Now Playing"
		}
		t := ps.Item
		if p.height < 8 {
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
		return "Now Playing"

	default:
		return "Now Playing"
	}
}

// ToggleKey returns the number key for btop-style pane toggling (key 1).
func (p *NowPlayingPane) ToggleKey() int {
	return 1
}

// Actions returns the pane-specific shortcuts shown in the border.
// When an episode is playing, includes an {Key: "i", Label: "details"} action
// for the Episode Details overlay.
// NOTE: layout.RenderPaneBorder drops all actions atomically when the pane is too
// narrow to fit title + actions within cfg.Width (dashCount < 0 after computing
// fixedWidth). At narrow widths none of these actions will appear; once the
// pane is wide enough they all appear. This is expected graceful degradation, not a bug.
func (p *NowPlayingPane) Actions() []layout.Action {
	actions := []layout.Action{
		{Key: "s", Label: "shfl"},
		{Key: "r", Label: "rpt"},
		{Key: "space", Label: "play"},
		{Key: "+/-", Label: "vol"},
		{Key: "v", Label: "viz"},
	}
	ps := p.store.PlaybackState()
	if ps != nil && ps.CurrentlyPlayingType == "episode" {
		actions = append(actions, layout.Action{Key: "i", Label: "details"})
	}
	return actions
}

// SetSize updates the pane's dimensions and recomputes the side-by-side
// layout geometry. The InfoBox occupies the left ~33% of the content area;
// the visualizer fills the right column with the seek bar as a single row
// at the bottom. When the right column would be too narrow, the InfoBox is
// dropped and the visualizer fills the full content area.
func (p *NowPlayingPane) SetSize(width, height int) {
	p.BasePane.SetSize(width, height)

	effH := p.effectiveHeight()
	cw := p.contentWidth()

	// --- adaptive InfoBox width (same formula as renderSideBySide) ---
	infoWidth := cw / npInfoPctTall
	if effH <= 8 {
		infoWidth = cw / npInfoPctShort
	}
	if infoWidth < npInfoMin {
		infoWidth = npInfoMin
	}
	vizWidth := cw - infoWidth - npGap
	if vizWidth < npMinViz {
		infoWidth = 0
		vizWidth = cw
	}

	// --- visualizer height: reserve 1 row for seek bar ---
	rightH := effH - 2*npPadV
	if rightH < 1 {
		rightH = 1
	}
	vizHeight := rightH - 1
	if vizHeight < 1 {
		vizHeight = rightH
	}

	if infoWidth > 0 {
		p.infoBox.SetSize(infoWidth, effH)
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

	case components.SeekDebounceTickMsg:
		if matched, targetMs, seq := p.seekBar.HandleDebounce(m); matched {
			return p, func() tea.Msg { return SeekIntentMsg{TargetMs: targetMs, Seq: seq} }
		}
		return p, nil

	case SeekAppliedMsg:
		if m.Err != nil {
			p.seekBar.CancelPending(m.Seq, confirmedProgress(p.store))
		} else {
			p.seekBar.ConfirmFromAPI(m.Seq, m.PosMs)
			p.localProgressMs = m.PosMs
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
// Dispatches based on CurrentlyPlayingType: track → side-by-side, episode → episode
// side-by-side, default/empty → empty state.
func (p *NowPlayingPane) View() string {
	ps := p.store.PlaybackState()
	if ps == nil {
		return p.renderEmpty()
	}
	switch ps.CurrentlyPlayingType {
	case "track":
		if ps.Item != nil {
			return p.renderSideBySide()
		}
		return p.renderEmpty()
	case "episode":
		if ps.Episode != nil {
			return p.renderEpisodeSideBySide()
		}
		return p.renderEmpty()
	default:
		return p.renderEmpty()
	}
}

// renderSideBySide composes the InfoBox on the left and the visualizer on
// the right. The visualizer renders as a single block at the top of the
// right column, with the seek bar as its own row at the bottom. When the
// right column would be too narrow, the InfoBox is dropped.
func (p *NowPlayingPane) renderSideBySide() string {
	ps := p.store.PlaybackState()
	t := ps.Item
	effH := p.effectiveHeight()
	cw := p.contentWidth()

	infoWidth := cw / npInfoPctTall
	if effH <= 8 {
		infoWidth = cw / npInfoPctShort
	}
	if infoWidth < npInfoMin {
		infoWidth = npInfoMin
	}
	vizWidth := cw - infoWidth - npGap
	if vizWidth < npMinViz {
		infoWidth = 0
		vizWidth = cw
	}

	frame := p.engine.CurrentFrame()
	seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)

	// Build right column lines.
	rightLines := make([]string, 0, len(frame)+1)
	for _, line := range frame {
		rightLines = append(rightLines, renderStyledLine(line))
	}
	rightLines = append(rightLines, seekBar)

	// Fit right column into effective height, clipping visualizer (not seek bar) if needed.
	targetH := effH
	contentH := len(rightLines)
	padTotal := targetH - contentH
	if padTotal < 0 {
		keepViz := targetH - 1 // reserve exactly 1 row for seek bar
		if keepViz < 0 {
			keepViz = 0
		}
		if len(rightLines)-1 > keepViz {
			rightLines = append(rightLines[:keepViz], seekBar)
		}
		padTotal = 0
	}
	topPad := padTotal / 2
	botPad := padTotal - topPad
	for i := 0; i < topPad; i++ {
		rightLines = append([]string{""}, rightLines...)
	}
	for i := 0; i < botPad; i++ {
		rightLines = append(rightLines, "")
	}

	// Compose line-by-line.
	var lines []string
	if infoWidth > 0 {
		infoLines := p.buildInfoLines(effH, infoWidth)
		infoView := p.infoBox.Render("Track Info", infoLines, p.focused)
		infoSplit := strings.Split(infoView, "\n")

		// Equalise line count.
		for len(infoSplit) < len(rightLines) {
			infoSplit = append(infoSplit, strings.Repeat(" ", infoWidth))
		}
		for len(rightLines) < len(infoSplit) {
			rightLines = append(rightLines, strings.Repeat(" ", vizWidth))
		}

		gap := strings.Repeat(" ", npGap)
		for i := range infoSplit {
			lines = append(lines, infoSplit[i]+gap+rightLines[i])
		}
	} else {
		lines = rightLines
	}

	// Centre vertically within oversized pane.
	if p.height > effH {
		outerPad := (p.height - effH) / 2
		for i := 0; i < outerPad; i++ {
			lines = append([]string{""}, lines...)
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n")
}

// renderEpisodeSideBySide composes the episode InfoBox on the left and the
// visualizer on the right. Same layout as renderSideBySide but renders episode
// info (episode name, show name, release date) instead of track info.
func (p *NowPlayingPane) renderEpisodeSideBySide() string {
	ps := p.store.PlaybackState()
	ep := ps.Episode
	effH := p.effectiveHeight()
	cw := p.contentWidth()

	infoWidth := cw / npInfoPctTall
	if effH <= 8 {
		infoWidth = cw / npInfoPctShort
	}
	if infoWidth < npInfoMin {
		infoWidth = npInfoMin
	}
	vizWidth := cw - infoWidth - npGap
	if vizWidth < npMinViz {
		infoWidth = 0
		vizWidth = cw
	}

	frame := p.engine.CurrentFrame()
	seekBar := p.seekBar.Render(p.localProgressMs, ep.DurationMs)

	// Build right column lines.
	rightLines := make([]string, 0, len(frame)+1)
	for _, line := range frame {
		rightLines = append(rightLines, renderStyledLine(line))
	}
	rightLines = append(rightLines, seekBar)

	// Fit right column into effective height.
	targetH := effH
	contentH := len(rightLines)
	padTotal := targetH - contentH
	if padTotal < 0 {
		keepViz := targetH - 1
		if keepViz < 0 {
			keepViz = 0
		}
		if len(rightLines)-1 > keepViz {
			rightLines = append(rightLines[:keepViz], seekBar)
		}
		padTotal = 0
	}
	topPad := padTotal / 2
	botPad := padTotal - topPad
	for i := 0; i < topPad; i++ {
		rightLines = append([]string{""}, rightLines...)
	}
	for i := 0; i < botPad; i++ {
		rightLines = append(rightLines, "")
	}

	// Compose line-by-line.
	var lines []string
	if infoWidth > 0 {
		infoLines := p.buildEpisodeInfoLines(effH, infoWidth)
		infoTitle := "Episode Info"
		epiActions := []layout.Action{{Key: "i", Label: "details"}}
		infoView := p.infoBox.Render(infoTitle, infoLines, p.focused, epiActions...)
		infoSplit := strings.Split(infoView, "\n")

		// Equalise line count.
		for len(infoSplit) < len(rightLines) {
			infoSplit = append(infoSplit, strings.Repeat(" ", infoWidth))
		}
		for len(rightLines) < len(infoSplit) {
			rightLines = append(rightLines, strings.Repeat(" ", vizWidth))
		}

		gap := strings.Repeat(" ", npGap)
		for i := range infoSplit {
			lines = append(lines, infoSplit[i]+gap+rightLines[i])
		}
	} else {
		lines = rightLines
	}

	// Centre vertically within oversized pane.
	if p.height > effH {
		outerPad := (p.height - effH) / 2
		for i := 0; i < outerPad; i++ {
			lines = append([]string{""}, lines...)
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n")
}

// buildEpisodeInfoLines builds the InfoBox content for an episode (episode name,
// show name, release date, controls, volume). When tight, release date is dropped.
func (p *NowPlayingPane) buildEpisodeInfoLines(bodyHeight int, infoWidth int) []string {
	ps := p.store.PlaybackState()
	if ps == nil || ps.Episode == nil {
		return nil
	}
	ep := ps.Episode
	primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary()).Bold(true)
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	showName := ""
	if ep.Show != nil {
		showName = ep.Show.Name
	}

	pad := strings.Repeat(" ", npInfoPadLeft)
	ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)
	innerW := infoWidth - 2
	ctrlLine := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(ctrl.Render())

	lines := []string{
		pad + primaryStyle.Render(ep.Name),
		pad + secondaryStyle.Render(showName),
		pad + mutedStyle.Render(ep.ReleaseDate),
		ctrlLine,
		p.volumeBar.Render(),
	}

	innerH := bodyHeight - 2
	if innerH < 1 {
		innerH = 1
	}
	if len(lines) > innerH {
		if innerH >= 4 {
			// Drop release date line so controls + volume remain visible.
			lines = append(lines[:2], lines[3:]...)
		} else {
			lines = lines[:innerH]
		}
	}
	return lines
}
func (p *NowPlayingPane) buildInfoLines(bodyHeight int, infoWidth int) []string {
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

	// Track name with optional ♥ prefix when the track is liked.
	trackName := t.Name
	if p.store.IsTrackLiked(t.ID) {
		trackName = uikit.GlyphFor(uikit.GlyphLiked, uikit.ActiveMode()) + " " + trackName
	}

	pad := strings.Repeat(" ", npInfoPadLeft)
	ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)
	innerW := infoWidth - 2
	ctrlLine := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(ctrl.Render())

	lines := []string{
		pad + primaryStyle.Render(trackName),
		pad + secondaryStyle.Render(strings.Join(artistNames, ", ")),
		pad + mutedStyle.Render(t.Album.Name),
		ctrlLine,
		p.volumeBar.Render(),
	}

	innerH := bodyHeight - 2 // InfoBox borders consume 2 rows
	if innerH < 1 {
		innerH = 1
	}
	if len(lines) > innerH {
		if innerH >= 4 {
			// Drop album line so controls + volume remain visible.
			lines = append(lines[:2], lines[3:]...)
		} else {
			lines = lines[:innerH]
		}
	}
	return lines
}

// renderStyledLine renders a single StyledLine into a string.
// When Segments is populated, each segment gets its own color;
// otherwise the line-level Color field is used (backward compatible).
func renderStyledLine(line viz.StyledLine) string {
	if len(line.Segments) > 0 {
		var sb strings.Builder
		for _, seg := range line.Segments {
			sb.WriteString(lipgloss.NewStyle().Foreground(seg.Color).Render(seg.Text))
		}
		return sb.String()
	}
	return lipgloss.NewStyle().Foreground(line.Color).Render(line.Text)
}

// renderStyledLines joins StyledLines into a single string.
// When Segments is populated, each segment gets its own color;
// otherwise the line-level Color field is used (backward compatible).
func renderStyledLines(lines viz.Frame) string {
	if len(lines) == 0 {
		return ""
	}
	rows := make([]string, len(lines))
	for i, line := range lines {
		rows[i] = renderStyledLine(line)
	}
	return strings.Join(rows, "\n")
}

// currentDurationMs returns the duration of the currently playing item (track or episode).
// Returns 0 if nothing is playing.
func (p *NowPlayingPane) currentDurationMs() int {
	ps := p.store.PlaybackState()
	if ps == nil {
		return 0
	}
	return p.playingDurationMs(ps)
}

// currentProgressMs returns the confirmed playback progress from the store.
// Returns 0 if playback state is unavailable.
func (p *NowPlayingPane) currentProgressMs() int {
	ps := p.store.PlaybackState()
	if ps == nil {
		return 0
	}
	return ps.ProgressMs
}

// renderEmpty shows the "Nothing playing" empty state, centered in the pane.
func (p *NowPlayingPane) renderEmpty() string {
	return uikit.EmptyState{
		Text:   "Nothing playing",
		Hint:   "Press / to search",
		Width:  p.width,
		Height: p.height,
		Theme:  p.theme,
	}.Render()
}

// handleTick processes a TickMsg: increments local progress when playing.
// localProgressMs is clamped to DurationMs so the seek bar never overflows.
// When a seek is pending (user pressed ←/→ but debounce hasn't resolved),
// the tick is skipped so localProgressMs doesn't drift away from the user's
// intended position.
func (p *NowPlayingPane) handleTick() (*NowPlayingPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil && ps.IsPlaying {
		if !p.seekBar.HasPending() {
			p.localProgressMs += 1000
			durationMs := p.playingDurationMs(ps)
			if durationMs > 0 && p.localProgressMs > durationMs {
				p.localProgressMs = durationMs
			}
		}
	}
	return p, nil
}

// playingDurationMs returns the duration in ms of whatever is currently playing.
// Returns 0 if nothing is playing (duration unknown).
func (p *NowPlayingPane) playingDurationMs(ps *domain.PlaybackState) int {
	switch ps.CurrentlyPlayingType {
	case "track":
		if ps.Item != nil {
			return ps.Item.DurationMs
		}
	case "episode":
		if ps.Episode != nil {
			return ps.Episode.DurationMs
		}
	}
	return 0
}

// handlePlaybackFetched processes notification that the store has fresh playback state.
// It resets localProgressMs to the server value and syncs the engine playing state.
// Album art dispatch was removed in story 222.
func (p *NowPlayingPane) handlePlaybackFetched(_ PlaybackStateFetchedMsg) (*NowPlayingPane, tea.Cmd) {
	ps := p.store.PlaybackState()
	if ps != nil {
		// Preserve local progress when a seek is pending so the title bar
		// stays in sync with the optimistic seek bar position.
		if !p.seekBar.HasPending() {
			p.localProgressMs = ps.ProgressMs
		}
		p.engine.SetPlaying(ps.IsPlaying)
		if ps.Device != nil {
			p.volumeBar.SetConfirmed(ps.Device.VolumePercent)
		}
		durationMs := p.playingDurationMs(ps)
		if durationMs > 0 {
			p.seekBar.SetTrackDuration(durationMs)
		}
		p.seekBar.SetPositionConfirmed(ps.ProgressMs)
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

	// Shift+arrows: previous/next track (moved from plain arrows).
	// Bubble Tea delivers Shift+arrows as separate KeyType constants,
	// NOT modifier flags on tea.KeyLeft/tea.KeyRight.
	// Shift variants must be checked before plain arrows (more specific first).
	case msg.Type == tea.KeyShiftLeft:
		return p, emitPlaybackRequest(ActionPrevious)

	case msg.Type == tea.KeyShiftRight:
		return p, emitPlaybackRequest(ActionNext)

	// Plain arrows: seek back/forward 5 seconds. Works for both tracks and episodes.
	case msg.Type == tea.KeyLeft:
		if durationMs := p.currentDurationMs(); durationMs > 0 {
			confirmed := p.currentProgressMs()
			if p.seekBar.HasPending() {
				confirmed = p.seekBar.Current()
			}
			cmd := p.seekBar.HandleKey(-5000, confirmed, durationMs)
			return p, cmd
		}
		return p, nil

	case msg.Type == tea.KeyRight:
		if durationMs := p.currentDurationMs(); durationMs > 0 {
			confirmed := p.currentProgressMs()
			if p.seekBar.HasPending() {
				confirmed = p.seekBar.Current()
			}
			cmd := p.seekBar.HandleKey(+5000, confirmed, durationMs)
			return p, cmd
		}
		return p, nil

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

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "l":
		// Like/unlike the currently playing track. Reads liked status from the
		// store (O(1) lookup) and emits a ToggleLikeRequestMsg for the root app
		// to handle the premium gate, optimistic update, and API dispatch.
		ps := p.store.PlaybackState()
		if ps == nil || ps.Item == nil {
			return p, nil
		}
		track := *ps.Item
		return p, func() tea.Msg {
			return ToggleLikeRequestMsg{
				Track:          track,
				CurrentlyLiked: p.store.IsTrackLiked(track.ID),
			}
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
	// Restore seek bar state from the store so the new bar renders the correct position.
	if ps := p.store.PlaybackState(); ps != nil {
		p.seekBar.SetPositionConfirmed(ps.ProgressMs)
		durationMs := p.playingDurationMs(ps)
		if durationMs > 0 {
			p.seekBar.SetTrackDuration(durationMs)
		}
	}
	// Propagate dimensions to newly created sub-components.
	p.SetSize(p.width, p.height)
}

// contentWidth returns the inner content width (pane width minus border chrome).
func (p *NowPlayingPane) contentWidth() int { return paneMax(p.width, 10) }

// effectiveHeight caps the content height at npMaxContentH so solo panes don't
// stretch the visualizer to absurd heights.
func (p *NowPlayingPane) effectiveHeight() int {
	if p.height > npMaxContentH {
		return npMaxContentH
	}
	return p.height
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

// confirmedProgress reads the playback progress from the store.
// Returns 0 when playback state is unavailable.
func confirmedProgress(s state.StateReader) int {
	if ps := s.PlaybackState(); ps != nil {
		return ps.ProgressMs
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
