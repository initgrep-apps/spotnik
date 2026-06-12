package panes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ layout.Pane = &ShowEpisodesPane{}

func newTestShowEpisodesPaneWithData(show *domain.Show, episodes []domain.Episode) *ShowEpisodesPane {
	s := state.New()
	if show != nil {
		s.SetSelectedShow(show)
		s.SetSelectedShowID(show.ID)
		s.SetShowEpisodes(episodes)
	}
	th := theme.Load("black")
	p := NewShowEpisodesPane(s, th, true)
	p.SetSize(80, 20)
	return p
}

func TestShowEpisodesPane_ID(t *testing.T) {
	p := newTestShowEpisodesPaneWithData(nil, nil)
	assert.Equal(t, layout.PaneShowEpisodes, p.ID())
}

func TestShowEpisodesPane_ToggleKey(t *testing.T) {
	p := newTestShowEpisodesPaneWithData(nil, nil)
	assert.Equal(t, 2, p.ToggleKey())
}

func TestShowEpisodesPane_Title_Dynamic(t *testing.T) {
	show := &domain.Show{ID: "show1", Name: "My Favorite Show", TotalEpisodes: 42}
	episodes := []domain.Episode{
		{ID: "ep1", Name: "Episode 1", IsPlayable: true},
	}
	p := newTestShowEpisodesPaneWithData(show, episodes)
	title := p.Title()
	assert.Contains(t, title, "My Favorite Show")
	assert.Contains(t, title, "42 eps")
}

func TestShowEpisodesPane_Title_Default(t *testing.T) {
	p := NewShowEpisodesPane(state.New(), theme.Load("black"), false)
	assert.Equal(t, "Show Episodes", p.Title())
}

func TestShowEpisodesPane_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewShowEpisodesPane(s, th, true)
	p.SetSize(80, 20)
	output := p.View()
	assert.Contains(t, output, "No show selected")
}

func TestShowEpisodesPane_EnterPlayable(t *testing.T) {
	show := &domain.Show{ID: "show1", Name: "Test Show"}
	episodes := []domain.Episode{
		{ID: "ep1", Name: "Episode 1", URI: "spotify:episode:ep1", IsPlayable: true, DurationMs: 1800000},
	}
	p := newTestShowEpisodesPaneWithData(show, episodes)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	playMsg, ok := msg.(PlayEpisodeMsg)
	require.True(t, ok)
	assert.Equal(t, "spotify:episode:ep1", playMsg.EpisodeURI)
	assert.Equal(t, "spotify:show:show1", playMsg.PlaylistURI)
}

func TestShowEpisodesPane_DurationZero(t *testing.T) {
	show := &domain.Show{ID: "show1", Name: "Test Show"}
	episodes := []domain.Episode{
		{ID: "ep1", Name: "Episode 1", IsPlayable: true, DurationMs: 0},
	}
	p := newTestShowEpisodesPaneWithData(show, episodes)
	output := p.View()
	assert.Contains(t, output, "\u2014")
}
