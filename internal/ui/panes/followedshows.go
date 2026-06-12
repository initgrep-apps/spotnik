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

var _ layout.Pane = &FollowedShowsPane{}
var _ layout.FilterablePane = &FollowedShowsPane{}
var _ layout.FilterQueryPane = &FollowedShowsPane{}

type FollowedShowsPane struct {
	*TableBasedPane
}

func NewFollowedShowsPane(store state.StateReader, th theme.Theme, focused bool) *FollowedShowsPane {
	columns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "show", Header: "Show", FlexFactor: 10, Color: th.ColumnPrimary()},
		{Key: "publisher", Header: "Publisher", FlexFactor: 6, Color: th.ColumnSecondary()},
		{Key: "episodes", Header: "Eps", FlexFactor: 3, Color: th.ColumnTertiary()},
		{Key: "media", Header: "", FlexFactor: 1, Color: th.ColumnSecondary()},
	}
	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})
	f := components.NewFilter(th)
	f.SetPlaceholder("filter shows...")
	p := &FollowedShowsPane{
		TableBasedPane: NewTableBasedPane(store, th, focused, t, f),
	}
	t.SetFocused(focused)
	p.buildRows()
	return p
}

func (p *FollowedShowsPane) ID() layout.PaneID { return layout.PaneFollowedShows }

func (p *FollowedShowsPane) Title() string { return "Followed Shows" }

func (p *FollowedShowsPane) ToggleKey() int { return 3 }

func (p *FollowedShowsPane) Actions() []layout.Action {
	return []layout.Action{p.BaseFilterAction()}
}

func (p *FollowedShowsPane) Init() tea.Cmd { return nil }

func (p *FollowedShowsPane) SetFocused(focused bool) {
	p.TableBasedPane.SetFocused(focused)
	p.Table().SetFocused(focused && !p.Filter().IsActive())
}

func (p *FollowedShowsPane) SetSize(width, height int) {
	p.TableBasedPane.SetSize(width, height)
	p.Filter().SetWidth(width)
	p.resizeTable()
}

func (p *FollowedShowsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case FollowedShowsLoadedMsg:
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
		shows := p.filteredShows()
		idx := p.Table().SelectedIndex()
		if idx >= 0 && idx < len(shows) {
			show := shows[idx].Show
			if show.ID == p.store.SelectedShowID() {
				return p, nil
			}
			return p, func() tea.Msg {
				return SelectedShowChangedMsg{ShowID: show.ID}
			}
		}
		return p, nil
	}

	cmd := p.Table().Update(keyMsg)
	return p, cmd
}

func (p *FollowedShowsPane) View() string {
	if len(p.store.FollowedShows()) == 0 && !p.Filter().IsActive() {
		return uikit.EmptyState{
			Text:   "No followed shows",
			Hint:   "Search for shows with /",
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

func (p *FollowedShowsPane) RefreshRows() { p.buildRows() }

func (p *FollowedShowsPane) buildRows() {
	shows := p.filteredShows()
	if shows == nil {
		shows = []domain.SavedShow{}
	}

	rows := make([]map[string]string, len(shows))
	mode := uikit.ActiveMode()
	for i, ss := range shows {
		show := ss.Show

		var mediaGlyph string
		switch show.MediaType {
		case "audio":
			mediaGlyph = uikit.GlyphFor(uikit.GlyphMusicNote, mode)
		case "mixed":
			mediaGlyph = uikit.GlyphFor(uikit.GlyphDoubleNote, mode)
		case "video":
			mediaGlyph = "\U0001F3AC"
		}

		rows[i] = map[string]string{
			"index":     fmt.Sprintf("%d", i+1),
			"show":      show.Name,
			"publisher": show.Publisher,
			"episodes":  fmt.Sprintf("%d", show.TotalEpisodes),
			"media":     mediaGlyph,
		}
	}
	p.Table().SetRows(rows)
}

func (p *FollowedShowsPane) filteredShows() []domain.SavedShow {
	all := p.store.FollowedShows()
	if p.Filter().Query() == "" {
		return all
	}
	result := make([]domain.SavedShow, 0, len(all))
	for _, ss := range all {
		if p.Filter().MatchesAny(ss.Show.Name, ss.Show.Publisher) {
			result = append(result, ss)
		}
	}
	return result
}

func (p *FollowedShowsPane) resizeTable() {
	tableHeight := p.height
	if p.Filter().IsActive() {
		tableHeight--
	}
	if tableHeight < 0 {
		tableHeight = 0
	}
	p.Table().SetSize(p.width, tableHeight)
}

func (p *FollowedShowsPane) SetTheme(th theme.Theme) {
	p.theme = th
	cols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "show", Header: "Show", FlexFactor: 10, Color: th.ColumnPrimary()},
		{Key: "publisher", Header: "Publisher", FlexFactor: 6, Color: th.ColumnSecondary()},
		{Key: "episodes", Header: "Eps", FlexFactor: 3, Color: th.ColumnTertiary()},
		{Key: "media", Header: "", FlexFactor: 1, Color: th.ColumnSecondary()},
	}
	newTable, newFilter := components.RebuildTableTheme(th, cols, p.Table().Rows(), p.focused)
	p.SwapTableAndFilter(newTable, newFilter)
	p.resizeTable()
	p.buildRows()
}
