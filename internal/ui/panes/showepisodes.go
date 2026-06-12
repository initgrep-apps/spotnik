package panes

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

var _ layout.Pane = &ShowEpisodesPane{}
var _ layout.FilterablePane = &ShowEpisodesPane{}
var _ layout.FilterQueryPane = &ShowEpisodesPane{}

type ShowEpisodesPane struct {
	*TableBasedPane
}

func NewShowEpisodesPane(store state.StateReader, th theme.Theme, focused bool) *ShowEpisodesPane {
	columns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "title", Header: "Title", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "released", Header: "Released", FlexFactor: 4, Color: th.ColumnSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
		{Key: "icon", Header: "", FlexFactor: 1, Color: th.ColumnSecondary()},
	}
	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})
	f := components.NewFilter(th)
	f.SetPlaceholder("filter episodes...")
	p := &ShowEpisodesPane{
		TableBasedPane: NewTableBasedPane(store, th, focused, t, f),
	}
	t.SetFocused(focused)
	p.buildRows()
	return p
}

func (p *ShowEpisodesPane) ID() layout.PaneID { return layout.PaneShowEpisodes }

func (p *ShowEpisodesPane) Title() string {
	show := p.store.SelectedShow()
	if show != nil {
		return fmt.Sprintf("%s (%d eps)", show.Name, show.TotalEpisodes)
	}
	return "Show Episodes"
}

func (p *ShowEpisodesPane) ToggleKey() int { return 2 }

func (p *ShowEpisodesPane) Actions() []layout.Action {
	return []layout.Action{p.BaseFilterAction()}
}

func (p *ShowEpisodesPane) Init() tea.Cmd { return nil }

func (p *ShowEpisodesPane) SetFocused(focused bool) {
	p.TableBasedPane.SetFocused(focused)
	p.Table().SetFocused(focused && !p.Filter().IsActive())
}

func (p *ShowEpisodesPane) SetSize(width, height int) {
	p.TableBasedPane.SetSize(width, height)
	p.Filter().SetWidth(width)
	p.resizeTable()
}

func (p *ShowEpisodesPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case ShowEpisodesLoadedMsg:
		p.buildRows()
		return p, nil
	}

	if !p.focused {
		return p, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	if consumed, cmd := p.HandleFilterKey(keyMsg, p.buildRows, p.resizeTable); consumed {
		return p, cmd
	}

	if keyMsg.Type == tea.KeyEnter {
		episodes := p.filteredEpisodes()
		idx := p.Table().SelectedIndex()
		if idx >= 0 && idx < len(episodes) {
			ep := episodes[idx]
			if !ep.IsPlayable {
				return p, nil
			}
			show := p.store.SelectedShow()
			playlistURI := ""
			if show != nil {
				playlistURI = "spotify:show:" + show.ID
			}
			return p, func() tea.Msg {
				return PlayEpisodeMsg{
					EpisodeURI:  ep.URI,
					PlaylistURI: playlistURI,
				}
			}
		}
		return p, nil
	}

	cmd := p.Table().Update(keyMsg)
	return p, cmd
}

func (p *ShowEpisodesPane) View() string {
	if p.store.SelectedShow() == nil {
		return uikit.EmptyState{
			Text:   "No show selected",
			Hint:   "Select a show from Followed Shows",
			Width:  p.width,
			Height: p.height,
			Theme:  p.theme,
		}.Render()
	}
	var parts []string
	if p.Filter().IsActive() {
		parts = append(parts, p.Filter().View(p.width))
	}
	parts = append(parts, p.Table().View())
	return strings.Join(parts, "\n")
}

func (p *ShowEpisodesPane) RefreshRows() { p.buildRows() }

func (p *ShowEpisodesPane) buildRows() {
	episodes := p.filteredEpisodes()
	if episodes == nil {
		episodes = []domain.Episode{}
	}

	rows := make([]map[string]string, len(episodes))
	mode := uikit.ActiveMode()
	for i, ep := range episodes {
		var iconGlyph string
		if !ep.IsPlayable {
			iconGlyph = uikit.GlyphFor(uikit.GlyphBlocked, mode)
		} else if ep.ResumePoint.ResumePositionMs > 0 && !ep.ResumePoint.FullyPlayed {
			iconGlyph = uikit.GlyphFor(uikit.GlyphPlaying, mode)
		}

		rows[i] = map[string]string{
			"index":    fmt.Sprintf("%d", i+1),
			"title":    ep.Name,
			"released": formatReleaseDate(ep.ReleaseDate),
			"duration": formatDurationMsH(ep.DurationMs),
			"icon":     iconGlyph,
		}
	}

	playingIdx := -1
	if ps := p.store.PlaybackState(); ps != nil && ps.Episode != nil {
		for i, ep := range episodes {
			if ep.ID == ps.Episode.ID {
				playingIdx = i
				break
			}
		}
	}
	p.Table().SetPlayingIndex(playingIdx)
	p.Table().SetRows(rows)
}

func (p *ShowEpisodesPane) filteredEpisodes() []domain.Episode {
	all := p.store.ShowEpisodes()
	if p.Filter().Query() == "" {
		return all
	}
	result := make([]domain.Episode, 0, len(all))
	for _, ep := range all {
		if p.Filter().Matches(ep.Name) {
			result = append(result, ep)
		}
	}
	return result
}

func (p *ShowEpisodesPane) resizeTable() {
	tableHeight := p.height
	if p.Filter().IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	p.Table().SetSize(p.width, tableHeight)
}

func (p *ShowEpisodesPane) SetTheme(th theme.Theme) {
	p.theme = th
	cols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "title", Header: "Title", FlexFactor: 9, Color: th.ColumnPrimary()},
		{Key: "released", Header: "Released", FlexFactor: 4, Color: th.ColumnSecondary()},
		{Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
		{Key: "icon", Header: "", FlexFactor: 1, Color: th.ColumnSecondary()},
	}
	newTable, newFilter := components.RebuildTableTheme(th, cols, p.Table().Rows(), p.focused)
	p.SwapTableAndFilter(newTable, newFilter)
	p.resizeTable()
	p.buildRows()
}

func formatDurationMsH(ms int) string {
	if ms <= 0 {
		return "\u2014"
	}
	totalSec := ms / 1000
	hours := totalSec / 3600
	minutes := (totalSec % 3600) / 60
	seconds := totalSec % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func formatReleaseDate(dateStr string) string {
	if len(dateStr) < 10 {
		return dateStr
	}
	t, err := time.Parse("2006-01-02", dateStr[:10])
	if err != nil {
		return dateStr
	}
	return t.Format("Jan 2")
}
