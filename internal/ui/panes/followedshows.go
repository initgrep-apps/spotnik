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

var _ layout.Pane = &FollowedShowsPane{}
var _ layout.FilterablePane = &FollowedShowsPane{}
var _ layout.FilterQueryPane = &FollowedShowsPane{}

// FollowedShowsPane renders the user's followed shows in a dense table
// and supports a two-level drill-down: Level 1 is the show list, Level 2
// shows episodes for the selected show.
type FollowedShowsPane struct {
	*TableBasedPane

	// episodeTable renders the episode list when inEpisodeView is true.
	episodeTable *components.Table

	// Drill-down identity
	inEpisodeView    bool
	selectedShowID   string
	selectedShowName string

	// Episode sub-view data (pane-owned, NOT in global store)
	loadedEpisodes   []domain.Episode
	episodesTotal    int
	episodesOffset   int
	hasMoreEpisodes  bool
	episodesFetching bool
}

func NewFollowedShowsPane(store state.StateReader, th theme.Theme, focused bool) *FollowedShowsPane {
	showColumns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "media", Header: "", FlexFactor: 1, Color: th.ColumnSecondary(), Priority: 1},
		{Key: "show", Header: "Show", FlexFactor: 10, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "publisher", Header: "Pub", FlexFactor: 6, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "episodes", Header: "Eps", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
	}
	t := components.NewTable(components.TableConfig{
		Columns:    showColumns,
		Theme:      th,
		ShowHeader: true,
	})

	episodeColumns := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "icon", Header: "", FlexFactor: 1, Color: th.ColumnSecondary(), Priority: 1},
		{Key: "title", Header: "Title", FlexFactor: 9, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "released", Header: "Released", FlexFactor: 4, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "duration", Header: "Dur", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
	}
	et := components.NewTable(components.TableConfig{
		Columns:    episodeColumns,
		Theme:      th,
		ShowHeader: true,
	})

	f := components.NewFilter(th)
	f.SetPlaceholder("filter shows...")
	p := &FollowedShowsPane{
		TableBasedPane: NewTableBasedPane(store, th, focused, t, f),
		episodeTable:   et,
	}
	t.SetFocused(focused)
	p.buildShowRows()
	return p
}

func (p *FollowedShowsPane) ID() layout.PaneID { return layout.PaneFollowedShows }

func (p *FollowedShowsPane) Title() string {
	if p.inEpisodeView {
		hrule := uikit.GlyphFor(uikit.GlyphHRule, uikit.ActiveMode())
		return fmt.Sprintf("Followed Shows %s%s %s (%d eps)", hrule, hrule, p.selectedShowName, p.episodesTotal)
	}
	return "Followed Shows"
}

func (p *FollowedShowsPane) ToggleKey() int { return 3 }

func (p *FollowedShowsPane) Actions() []layout.Action {
	if p.inEpisodeView {
		return []layout.Action{{Key: "Esc", Label: "back"}}
	}
	return []layout.Action{p.BaseFilterAction()}
}

func (p *FollowedShowsPane) Init() tea.Cmd { return nil }

func (p *FollowedShowsPane) SetFocused(focused bool) {
	p.TableBasedPane.SetFocused(focused)
	if p.inEpisodeView {
		p.episodeTable.SetFocused(focused)
		p.Table().SetFocused(false)
	} else {
		p.Table().SetFocused(focused && !p.Filter().IsActive())
		p.episodeTable.SetFocused(false)
	}
}

func (p *FollowedShowsPane) SetSize(width, height int) {
	p.TableBasedPane.SetSize(width, height)
	p.Filter().SetWidth(width)
	p.episodeTable.SetSize(width, height)
	p.resizeTable()
}

func (p *FollowedShowsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case FollowedShowsLoadedMsg:
		p.buildShowRows()
		return p, nil

	case ShowEpisodesLoadedMsg:
		if m.ShowID != p.selectedShowID {
			return p, nil
		}
		p.episodesFetching = false
		if m.Err != nil {
			return p, nil
		}
		if m.Offset == 0 {
			p.loadedEpisodes = m.Items
		} else {
			p.loadedEpisodes = append(p.loadedEpisodes, m.Items...)
		}
		p.episodesOffset = len(p.loadedEpisodes)
		p.episodesTotal = m.Total
		p.hasMoreEpisodes = m.HasNext
		p.buildEpisodeRows()
		return p, nil
	}

	if !p.focused {
		return p, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	// In episode sub-view: handle Enter (play), Esc (back), navigation.
	if p.inEpisodeView {
		return p.handleEpisodeViewKey(keyMsg)
	}

	// In show list: handle keys.
	return p.handleListViewKey(keyMsg)
}

func (p *FollowedShowsPane) handleListViewKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if consumed, cmd := p.HandleFilterKey(key, p.buildShowRows, p.resizeTable); consumed {
		return p, cmd
	}

	if key.Type == tea.KeyEnter {
		shows := p.filteredShows()
		idx := p.Table().SelectedIndex()
		if idx >= 0 && idx < len(shows) {
			show := shows[idx].Show
			// Prevent re-entering same show.
			if show.ID == p.selectedShowID {
				return p, nil
			}
			// Enter episode sub-view.
			p.selectedShowID = show.ID
			p.selectedShowName = show.Name
			p.inEpisodeView = true
			p.loadedEpisodes = nil
			p.episodesOffset = 0
			p.episodesTotal = 0
			p.hasMoreEpisodes = false
			p.episodesFetching = true
			p.Table().SetFocused(false)
			p.episodeTable.SetFocused(true)
			p.resizeTable()
			p.buildEpisodeRows()
			showID := show.ID
			return p, func() tea.Msg {
				return FetchShowEpisodesRequestMsg{ShowID: showID, Offset: 0}
			}
		}
		return p, nil
	}

	cmd := p.Table().Update(key)
	return p, cmd
}

func (p *FollowedShowsPane) handleEpisodeViewKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		p.inEpisodeView = false
		p.episodeTable.SetFocused(false)
		p.Table().SetFocused(true)
		p.resizeTable()
		return p, func() tea.Msg { return FollowedShowsViewClosedMsg{} }

	case tea.KeyEnter:
		idx := p.episodeTable.SelectedIndex()
		if idx >= 0 && idx < len(p.loadedEpisodes) {
			ep := p.loadedEpisodes[idx]
			if !ep.IsPlayable {
				return p, nil
			}
			episodeURI := ep.URI
			showID := p.selectedShowID
			return p, func() tea.Msg {
				return PlayEpisodeMsg{
					EpisodeURI:  episodeURI,
					PlaylistURI: "spotify:show:" + showID,
				}
			}
		}
		return p, nil
	}

	cmd := p.episodeTable.Update(key)
	prefetchCmd := p.checkEpisodePrefetch()
	return p, tea.Batch(cmd, prefetchCmd)
}

func (p *FollowedShowsPane) checkEpisodePrefetch() tea.Cmd {
	if !p.hasMoreEpisodes || p.episodesFetching {
		return nil
	}
	if len(p.loadedEpisodes) == 0 {
		return nil
	}
	cursor := p.episodeTable.SelectedIndex()
	if cursor < len(p.loadedEpisodes)-5 {
		return nil
	}
	p.episodesFetching = true
	showID := p.selectedShowID
	offset := p.episodesOffset
	return func() tea.Msg {
		return FetchShowEpisodesRequestMsg{ShowID: showID, Offset: offset}
	}
}

func (p *FollowedShowsPane) View() string {
	if !p.inEpisodeView && len(p.store.FollowedShows()) == 0 && !p.Filter().IsActive() {
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
	if p.inEpisodeView {
		parts = append(parts, p.episodeTable.View())
	} else {
		parts = append(parts, p.Table().View())
	}
	return strings.Join(parts, "\n")
}

func (p *FollowedShowsPane) RefreshRows() {
	if p.inEpisodeView {
		p.buildEpisodeRows()
	} else {
		p.buildShowRows()
	}
}

func (p *FollowedShowsPane) buildShowRows() {
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

func (p *FollowedShowsPane) buildEpisodeRows() {
	rows := make([]map[string]string, len(p.loadedEpisodes))
	mode := uikit.ActiveMode()
	for i, ep := range p.loadedEpisodes {
		var iconGlyph string
		if !ep.IsPlayable {
			iconGlyph = uikit.GlyphFor(uikit.GlyphLocked, mode)
		}
		rows[i] = map[string]string{
			"index":    fmt.Sprintf("%d", i+1),
			"title":    ep.Name,
			"released": formatReleaseDate(ep.ReleaseDate),
			"duration": formatDurationMsH(ep.DurationMs),
			"icon":     iconGlyph,
		}
	}
	p.episodeTable.SetRows(rows)
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
	if p.inEpisodeView {
		p.episodeTable.SetSize(p.width, tableHeight)
	} else {
		p.Table().SetSize(p.width, tableHeight)
	}
}

func (p *FollowedShowsPane) SetTheme(th theme.Theme) {
	p.theme = th
	showCols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "media", Header: "", FlexFactor: 1, Color: th.ColumnSecondary(), Priority: 1},
		{Key: "show", Header: "Show", FlexFactor: 10, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "publisher", Header: "Pub", FlexFactor: 6, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "episodes", Header: "Eps", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
	}
	newTable, newFilter := components.RebuildTableTheme(th, showCols, p.Table().Rows(), p.focused && !p.inEpisodeView)
	p.SwapTableAndFilter(newTable, newFilter)

	episodeCols := []components.ColumnDef{
		{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
		{Key: "icon", Header: "", FlexFactor: 1, Color: th.ColumnSecondary(), Priority: 1},
		{Key: "title", Header: "Title", FlexFactor: 9, Color: th.ColumnPrimary(), Priority: 1},
		{Key: "released", Header: "Released", FlexFactor: 4, Color: th.ColumnSecondary(), Priority: 2},
		{Key: "duration", Header: "Dur", FlexFactor: 3, Color: th.ColumnTertiary(), Priority: 3},
	}
	p.episodeTable = components.NewTable(components.TableConfig{
		Columns:    episodeCols,
		Theme:      th,
		ShowHeader: true,
	})
	p.episodeTable.SetSize(p.width, p.height)
	if p.inEpisodeView {
		p.episodeTable.SetFocused(p.focused)
		p.buildEpisodeRows()
	}

	p.resizeTable()
	p.buildShowRows()
}

// formatReleaseDate formats an ISO date string as a short date (e.g. "Jan 15").
func formatReleaseDate(isoDate string) string {
	if isoDate == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", isoDate)
	if err != nil {
		return isoDate
	}
	return t.Format("Jan 2")
}
