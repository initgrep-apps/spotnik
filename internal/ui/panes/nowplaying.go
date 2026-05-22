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

// NowPlayingPane is the center pane Bubble Tea model.
// It renders the currently playing track and handles playback key events.
// It reads all state from the Store; it never stores API data in its own fields.
// It implements the layout.Pane interface for integration with the layout manager.
//
// Layout: horizontal split with InfoBox (left, ~1/3 width) and viz engine (right, ~2/3 width).
// The right panel contains: top viz rows, seek bar, bottom viz rows.
// When height < 8, Title() embeds compact track info in the pane title bar instead.
type NowPlayingPane struct {
	BasePane

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

	// artRenderer caches pixterm-rendered album art rows and tracks loading state.
	artRenderer components.AlbumArtRenderer

	// pendingArtRefresh is set by SetSize when imageRows changes by more than 2.
	// The next WindowSizeMsg handler dispatches a re-fetch with updated dimensions.
	pendingArtRefresh bool
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
			m := uikit.ActiveMode()
			var stateGlyph string
			if ps.IsPlaying {
				stateGlyph = uikit.GlyphFor(uikit.GlyphPaused, m)
			} else {
				stateGlyph = uikit.GlyphFor(uikit.GlyphPlaying, m)
			}
			sep := uikit.GlyphFor(uikit.GlyphHRule, m)
			current := formatDurationMs(p.localProgressMs)
			total := formatDurationMs(t.DurationMs)
			return fmt.Sprintf("Now Playing %s %s \u00b7 %s %s %s %s/%s",
				sep, t.Name, strings.Join(artistNames, ", "), sep, stateGlyph, current, total)
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

// SetSize updates the pane's dimensions and recomputes the split layout geometry.
// Sub-component sizes are tier-aware: base uses a 3-col inline layout, mid/full use a
// 2-col upper section with a full-width InfoBox below. When imageRows changes by
// more than 2, pendingArtRefresh is set so the next WindowSizeMsg handler can
// dispatch a re-fetch with the updated dimensions.
func (p *NowPlayingPane) SetSize(width, height int) {
	prevRows := p.imageRows()
	p.BasePane.SetSize(width, height)

	cw := p.contentWidth()
	bh := p.bodyHeight()

	switch p.renderTier() {
	case tierBase:
		// 3-col inline: image | info | viz
		rows := p.imageRows()
		cols := rows * 2
		remaining := paneMax(cw-cols-2, 10)

		infoWidth := paneMax(remaining/2, 14)
		vizWidth := remaining - infoWidth - 1
		if vizWidth < 1 {
			vizWidth = 1
		}

		p.infoBox.SetSize(infoWidth, rows)
		p.engine.SetSize(vizWidth, paneMax(rows-1, 1))
		p.seekBar.SetWidth(vizWidth)
		p.volumeBar.SetWidth(infoWidth - 4)

	case tierMid, tierFull:
		// 2-col upper: image | viz
		rows := p.imageRows()
		cols := rows * 2
		vizWidth := paneMax(cw-cols-1, 1)

		p.engine.SetSize(vizWidth, paneMax(rows-1, 1))
		p.seekBar.SetWidth(vizWidth)

		// Lower section: full-width InfoBox
		infoHeight := paneMax(bh-rows, 2)
		p.infoBox.SetSize(cw, infoHeight)
		p.volumeBar.SetWidth(cw - 4)
	}

	p.pendingArtRefresh = abs(p.imageRows()-prevRows) > 2
}

// SetFocused updates the focused state.
func (p *NowPlayingPane) SetFocused(focused bool) {
	p.BasePane.SetFocused(focused)
}

// Init starts the viz engine animation tick loop and dispatches an album art fetch
// if playback is active at startup. Image dimensions use conservative defaults
// (8x16) since SetSize() will not have run yet; the art is re-fetched after the
// first resize with correct dimensions.
func (p *NowPlayingPane) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, p.engine.Init())

	ps := p.store.PlaybackState()
	if ps != nil && ps.Item != nil {
		if img := ps.Item.Album.BestImage(100); img != nil {
			cmds = append(cmds, components.FetchAlbumArtCmd(ps.Item.ID, img.URL, 8, 16))
			p.artRenderer.SetLoading(ps.Item.ID)
		}
	}

	return tea.Batch(cmds...)
}

// Update handles all messages for the NowPlayingPane.
func (p *NowPlayingPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case TickMsg:
		return p.handleTick()

	case PlaybackStateFetchedMsg:
		return p.handlePlaybackFetched(m)

	case components.AlbumArtFetchedMsg:
		p.artRenderer.SetResult(m.TrackID, m.Rows)
		return p, nil

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

	case tea.WindowSizeMsg:
		if p.pendingArtRefresh {
			p.pendingArtRefresh = false
			ps := p.store.PlaybackState()
			if ps != nil && ps.Item != nil {
				if img := ps.Item.Album.BestImage(100); img != nil {
					p.artRenderer.SetLoading(ps.Item.ID)
					return p, components.FetchAlbumArtCmd(ps.Item.ID, img.URL, p.imageRows(), p.imageCols())
				}
			}
		}
		return p, nil

	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		return p.handleKey(m)
	}

	return p, nil
}

// View renders the NowPlaying pane. It reads from the store and never calls the API.
// Layout is tier-aware: base (3-col inline), mid (2-col upper + compact InfoBox lower),
// full (2-col upper + rich InfoBox lower). Falls back to pre-feature 2-col when no image.
func (p *NowPlayingPane) View() string {
	ps := p.store.PlaybackState()
	if ps == nil || ps.Item == nil {
		return p.renderEmpty()
	}

	if !p.artRenderer.HasImage() && !p.artRenderer.IsLoading() {
		return p.renderFallback()
	}

	switch p.renderTier() {
	case tierBase:
		return p.renderBase()
	case tierMid:
		return p.renderMid()
	case tierFull:
		return p.renderFull()
	}
	return ""
}

// renderFallback renders the pre-feature 2-col layout: InfoBox left, viz+seekbar right.
// Used when no album art is available and none is loading.
// A local InfoBox is created with fallback dimensions so tier-aware SetSize does not
// leave the infoBox too short or too wide for this path.
func (p *NowPlayingPane) renderFallback() string {
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

	ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)

	contentWidth := paneMax(p.width-4, 10)
	bodyHeight := paneMax(p.height-4, 4)
	innerH := bodyHeight - 2

	var infoLines []string
	switch {
	case innerH >= 6:
		infoLines = []string{
			primaryStyle.Render(t.Name),
			secondaryStyle.Render(strings.Join(artistNames, ", ")),
			mutedStyle.Render(t.Album.Name),
			"",
			ctrl.Render(),
			p.volumeBar.Render(),
		}
	case innerH >= 5:
		infoLines = []string{
			primaryStyle.Render(t.Name),
			secondaryStyle.Render(strings.Join(artistNames, ", ")),
			mutedStyle.Render(t.Album.Name),
			ctrl.Render(),
			p.volumeBar.Render(),
		}
	case innerH >= 4:
		infoLines = []string{
			primaryStyle.Render(t.Name),
			secondaryStyle.Render(strings.Join(artistNames, ", ")),
			ctrl.Render(),
			p.volumeBar.Render(),
		}
	case innerH >= 3:
		infoLines = []string{
			primaryStyle.Render(t.Name),
			ctrl.Render(),
			p.volumeBar.Render(),
		}
	default:
		infoLines = []string{
			primaryStyle.Render(t.Name),
			ctrl.Render(),
		}
	}

	// Fallback layout: InfoBox left (~1/3 width), viz+seekbar right (~2/3 width).
	infoWidth := paneMax(contentWidth/3, 28)

	fbInfoBox := components.NewInfoBox(p.theme)
	fbInfoBox.SetSize(infoWidth, bodyHeight)
	infoView := fbInfoBox.Render("Track Info", infoLines, p.focused)

	frame := p.engine.CurrentFrame()
	topRows, bottomRows := splitFrame(frame)
	topView := renderStyledLines(topRows)
	bottomView := renderStyledLines(bottomRows)
	seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)

	rightPanel := lipgloss.JoinVertical(lipgloss.Left, topView, seekBar, bottomView)
	composite := lipgloss.JoinHorizontal(lipgloss.Top, infoView, " ", rightPanel)

	contentHeight := lipgloss.Height(composite)
	availableHeight := paneMax(p.height-2, 1)
	if contentHeight < availableHeight {
		composite = lipgloss.Place(contentWidth, availableHeight,
			lipgloss.Center, lipgloss.Center, composite)
	}

	return composite
}

// renderBase renders the 3-col inline layout for the base tier.
// Columns: imageBlock · InfoBox · vizBlock.
// Falls back to renderFallback when the remaining width after the image is < 28.
func (p *NowPlayingPane) renderBase() string {
	ps := p.store.PlaybackState()
	if ps == nil || ps.Item == nil {
		return p.renderEmpty()
	}
	t := ps.Item
	bh := p.bodyHeight()
	cw := p.contentWidth()
	rows := p.imageRows()
	cols := p.imageCols()

	remaining := cw - cols - 2
	if remaining < 28 {
		return p.renderFallback()
	}

	imageBlock := p.renderImageBlock(rows, cols)

	infoLines := p.buildInfoLinesBase(bh)
	infoView := p.infoBox.Render("Track Info", infoLines, p.focused)

	frame := p.engine.CurrentFrame()
	topRows, bottomRows := splitFrame(frame)
	topView := renderStyledLines(topRows)
	bottomView := renderStyledLines(bottomRows)
	seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)
	rightPanel := lipgloss.JoinVertical(lipgloss.Left, topView, seekBar, bottomView)

	composite := lipgloss.JoinHorizontal(lipgloss.Top, imageBlock, " ", infoView, " ", rightPanel)

	contentHeight := lipgloss.Height(composite)
	availableHeight := paneMax(p.height-2, 1)
	if contentHeight < availableHeight {
		composite = lipgloss.Place(cw, availableHeight,
			lipgloss.Center, lipgloss.Center, composite)
	}

	return composite
}

// buildInfoLinesBase builds the 5-line InfoBox content for the base tier and pads
// with trailing blank strings so the InfoBox vertically centres to top alignment.
func (p *NowPlayingPane) buildInfoLinesBase(bodyHeight int) []string {
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

// renderMid renders the mid-tier layout: image+viz side-by-side above a full-width
// 2-line InfoBox with no title.
func (p *NowPlayingPane) renderMid() string {
	ps := p.store.PlaybackState()
	if ps == nil || ps.Item == nil {
		return p.renderEmpty()
	}
	t := ps.Item
	rows := p.imageRows()
	cols := p.imageCols()

	imageBlock := p.renderImageBlock(rows, cols)

	frame := p.engine.CurrentFrame()
	topRows, bottomRows := splitFrame(frame)
	topView := renderStyledLines(topRows)
	bottomView := renderStyledLines(bottomRows)
	seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)
	vizBlock := lipgloss.JoinVertical(lipgloss.Left, topView, seekBar, bottomView)

	upperSection := lipgloss.JoinHorizontal(lipgloss.Top, imageBlock, " ", vizBlock)

	infoLines := p.buildInfoLinesMid()
	infoView := p.infoBox.Render("", infoLines, p.focused)

	return lipgloss.JoinVertical(lipgloss.Left, upperSection, infoView)
}

// buildInfoLinesMid builds the compact 2-line InfoBox content for the mid tier.
func (p *NowPlayingPane) buildInfoLinesMid() []string {
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

	line1 := primaryStyle.Render(t.Name) + " · " + secondaryStyle.Render(strings.Join(artistNames, ", ")) + " · " + mutedStyle.Render(t.Album.Name)
	line2 := ctrl.Render() + "   " + p.volumeBar.Render()

	return []string{line1, line2}
}

// renderFull renders the full-tier layout: image+viz side-by-side above a full-width
// 5-row InfoBox titled "Track Info" with 3 content lines.
func (p *NowPlayingPane) renderFull() string {
	ps := p.store.PlaybackState()
	if ps == nil || ps.Item == nil {
		return p.renderEmpty()
	}
	t := ps.Item
	cw := p.contentWidth()
	rows := p.imageRows()
	cols := p.imageCols()

	imageBlock := p.renderImageBlock(rows, cols)

	frame := p.engine.CurrentFrame()
	topRows, bottomRows := splitFrame(frame)
	topView := renderStyledLines(topRows)
	bottomView := renderStyledLines(bottomRows)
	seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)
	vizBlock := lipgloss.JoinVertical(lipgloss.Left, topView, seekBar, bottomView)

	upperSection := lipgloss.JoinHorizontal(lipgloss.Top, imageBlock, " ", vizBlock)

	infoLines := p.buildInfoLinesFull(cw)
	infoView := p.infoBox.Render("Track Info", infoLines, p.focused)

	return lipgloss.JoinVertical(lipgloss.Left, upperSection, infoView)
}

// buildInfoLinesFull builds the rich 3-line InfoBox content for the full tier.
func (p *NowPlayingPane) buildInfoLinesFull(contentWidth int) []string {
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

	innerW := contentWidth - 2

	line1 := primaryStyle.Render(t.Name)

	left := secondaryStyle.Render(strings.Join(artistNames, ", ")) + " · " + mutedStyle.Render(t.Album.Name)
	right := ctrl.Render() + "  " + p.volumeBar.Render()

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := innerW - leftW - rightW
	if gap < 1 {
		gap = 1
	}
	line2 := left + strings.Repeat(" ", gap) + right

	return []string{line1, line2, ""}
}

// renderImageBlock returns the album art image as a rows×cols block, a muted
// placeholder when loading, or an empty block as last resort.
func (p *NowPlayingPane) renderImageBlock(rows, cols int) string {
	if p.artRenderer.HasImage() {
		imgRows := p.artRenderer.Rows()
		if len(imgRows) > rows {
			imgRows = imgRows[:rows]
		}
		for len(imgRows) < rows {
			imgRows = append(imgRows, strings.Repeat(" ", cols))
		}
		for i := range imgRows {
			imgRows[i] = layout.TruncateOrPad(imgRows[i], cols)
		}
		return strings.Join(imgRows, "\n")
	}

	if p.artRenderer.IsLoading() {
		placeholder := lipgloss.NewStyle().Background(p.theme.TextMuted()).Render(strings.Repeat(" ", cols))
		lines := make([]string, rows)
		for i := range lines {
			lines[i] = placeholder
		}
		return strings.Join(lines, "\n")
	}

	return strings.Repeat("\n", rows-1) + strings.Repeat(" ", cols)
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
// It resets localProgressMs to the server value, syncs engine playing state, and
// dispatches an album art fetch when the track has changed.
func (p *NowPlayingPane) handlePlaybackFetched(msg PlaybackStateFetchedMsg) (*NowPlayingPane, tea.Cmd) {
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

	if msg.State != nil && msg.State.Item != nil {
		track := msg.State.Item
		if p.artRenderer.NeedsRefresh(track.ID) {
			if img := track.Album.BestImage(100); img != nil {
				p.artRenderer.SetLoading(track.ID)
				return p, components.FetchAlbumArtCmd(track.ID, img.URL, p.imageRows(), p.imageCols())
			}
		}
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

// renderTier identifies which responsive layout tier the pane should render.
type renderTier int

const (
	tierBase renderTier = iota
	tierMid
	tierFull
)

// renderTier selects the layout tier based on bodyHeight.
//   base: bodyHeight ≤ 18
//   mid:  19 – 30
//   full: > 30
func (p *NowPlayingPane) renderTier() renderTier {
	switch {
	case p.bodyHeight() > 30:
		return tierFull
	case p.bodyHeight() > 18:
		return tierMid
	default:
		return tierBase
	}
}

// bodyHeight returns the inner content height (pane height minus border chrome).
func (p *NowPlayingPane) bodyHeight() int { return paneMax(p.height-4, 0) }

// contentWidth returns the inner content width (pane width minus border chrome).
func (p *NowPlayingPane) contentWidth() int { return paneMax(p.width-4, 10) }

// imageRows returns the number of terminal rows allocated to the album art block.
// The formula is tier-aware: base uses the full body height, mid/full cap rows so
// the viz column never falls below 10 chars and reserve space for the InfoBox.
func (p *NowPlayingPane) imageRows() int {
	bh := p.bodyHeight()
	cw := p.contentWidth()
	switch p.renderTier() {
	case tierMid:
		return paneMax(paneMin(bh-4, (cw-11)/2), 4)
	case tierFull:
		return paneMax(paneMin(bh-5, (cw-11)/2), 4)
	default:
		return paneMax(bh, 4)
	}
}

// imageCols returns the number of terminal columns allocated to the album art block.
// Terminal chars are ~2:1 height:width, so cols = rows*2 produces a square image.
func (p *NowPlayingPane) imageCols() int { return p.imageRows() * 2 }

// paneMax returns the larger of two ints.
func paneMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// paneMin returns the smaller of two ints.
func paneMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// abs returns the absolute value of n.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
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
