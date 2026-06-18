package panes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

var _ layout.Pane = &SavedEpisodesPane{}
var _ layout.FilterablePane = &SavedEpisodesPane{}
var _ layout.FilterQueryPane = &SavedEpisodesPane{}

type SavedEpisodesPane struct {
	*TableBasedPane
}

func NewSavedEpisodesPane(store state.StateReader, th theme.Theme, focused bool) *SavedEpisodesPane {
	columns := []components.ColumnDef{
		{Key: "icon", Header: "", FlexFactor: 1, Color: th.ColumnSecondary(), Priority: 1},
		{Key: "episode", Header: "Episode", FlexFactor: 9, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "show", Header: "Show", FlexFactor: 6, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "duration", Header: "Dur", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
	}
	t := components.NewTable(components.TableConfig{
		Columns:    columns,
		Theme:      th,
		ShowHeader: true,
	})
	f := components.NewFilter(th)
	f.SetPlaceholder("filter episodes...")
	p := &SavedEpisodesPane{
		TableBasedPane: NewTableBasedPane(store, th, focused, t, f),
	}
	t.SetFocused(focused)
	p.buildRows()
	return p
}

func (p *SavedEpisodesPane) ID() layout.PaneID { return layout.PaneSavedEpisodes }

func (p *SavedEpisodesPane) Title() string { return "Saved Episodes" }

func (p *SavedEpisodesPane) ToggleKey() int { return 4 }

func (p *SavedEpisodesPane) Actions() []layout.Action {
	return []layout.Action{p.BaseFilterAction()}
}

func (p *SavedEpisodesPane) Init() tea.Cmd { return nil }

func (p *SavedEpisodesPane) SetFocused(focused bool) {
	p.TableBasedPane.SetFocused(focused)
	p.Table().SetFocused(focused && !p.Filter().IsActive())
}

func (p *SavedEpisodesPane) SetSize(width, height int) {
	p.TableBasedPane.SetSize(width, height)
	p.Filter().SetWidth(width)
	p.resizeTable()
}

func (p *SavedEpisodesPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case SavedEpisodesLoadedMsg:
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
		episodes := p.filteredSavedEpisodes()
		idx := p.Table().SelectedIndex()
		if idx >= 0 && idx < len(episodes) {
			se := episodes[idx]
			if !se.Episode.IsPlayable {
				return p, nil
			}
			showURI := ""
			if se.Episode.Show != nil {
				showURI = "spotify:show:" + se.Episode.Show.ID
			}
			return p, func() tea.Msg {
				return PlayEpisodeMsg{
					EpisodeURI:  se.Episode.URI,
					PlaylistURI: showURI,
				}
			}
		}
		return p, nil
	}

	cmd := p.Table().Update(keyMsg)
	return p, cmd
}

func (p *SavedEpisodesPane) View() string {
	if len(p.store.SavedEpisodes()) == 0 && !p.Filter().IsActive() {
		return uikit.EmptyState{
			Text:   "No saved episodes",
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

func (p *SavedEpisodesPane) RefreshRows() { p.buildRows() }

func (p *SavedEpisodesPane) buildRows() {
	saved := p.filteredSavedEpisodes()
	if saved == nil {
		saved = []domain.SavedEpisode{}
	}

	rows := make([]map[string]string, len(saved))
	mode := uikit.ActiveMode()
	for i, se := range saved {
		ep := se.Episode
		var iconGlyph string
		if !ep.IsPlayable {
			iconGlyph = uikit.GlyphFor(uikit.GlyphBlocked, mode)
		} else if ep.ResumePoint.ResumePositionMs > 0 && !ep.ResumePoint.FullyPlayed {
			iconGlyph = uikit.GlyphFor(uikit.GlyphPlaying, mode)
		}

		showName := ""
		if ep.Show != nil {
			showName = ep.Show.Name
		}

		rows[i] = map[string]string{
			"icon":     iconGlyph,
			"episode":  ep.Name,
			"show":     showName,
			"duration": formatDurationMsH(ep.DurationMs),
		}
	}

	p.Table().SetRows(rows)
}

func (p *SavedEpisodesPane) filteredSavedEpisodes() []domain.SavedEpisode {
	all := p.store.SavedEpisodes()
	if p.Filter().Query() == "" {
		return all
	}
	result := make([]domain.SavedEpisode, 0, len(all))
	for _, se := range all {
		showName := ""
		if se.Episode.Show != nil {
			showName = se.Episode.Show.Name
		}
		if p.Filter().MatchesAny(se.Episode.Name, showName) {
			result = append(result, se)
		}
	}
	return result
}

func (p *SavedEpisodesPane) resizeTable() {
	tableHeight := p.height
	if p.Filter().IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	p.Table().SetSize(p.width, tableHeight)
}

func (p *SavedEpisodesPane) SetTheme(th theme.Theme) {
	p.theme = th
	cols := []components.ColumnDef{
		{Key: "icon", Header: "", FlexFactor: 1, Color: th.ColumnSecondary(), Priority: 1},
		{Key: "episode", Header: "Episode", FlexFactor: 9, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "show", Header: "Show", FlexFactor: 6, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "duration", Header: "Dur", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
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
