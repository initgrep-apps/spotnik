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

var _ layout.Pane = &SavedEpisodesPane{}

func TestSavedEpisodesPane_ID(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewSavedEpisodesPane(s, th, false)
	assert.Equal(t, layout.PaneSavedEpisodes, p.ID())
}

func TestSavedEpisodesPane_ToggleKey(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewSavedEpisodesPane(s, th, false)
	assert.Equal(t, 4, p.ToggleKey())
}

func TestSavedEpisodesPane_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewSavedEpisodesPane(s, th, true)
	p.SetSize(80, 20)
	output := p.View()
	assert.Contains(t, output, "No saved episodes")
}

func TestSavedEpisodesPane_EnterPlayable(t *testing.T) {
	s := state.New()
	s.SetSavedEpisodes([]domain.SavedEpisode{
		{
			AddedAt: "2024-05-29T12:00:00Z",
			Episode: domain.Episode{
				ID: "ep1", Name: "Episode 1", URI: "spotify:episode:ep1",
				IsPlayable: true, DurationMs: 1800000,
				Show: &domain.Show{ID: "show1", Name: "Test Show"},
			},
		},
	})
	th := theme.Load("black")
	p := NewSavedEpisodesPane(s, th, true)
	p.SetSize(80, 20)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	playMsg, ok := msg.(PlayEpisodeMsg)
	require.True(t, ok)
	assert.Equal(t, "spotify:episode:ep1", playMsg.EpisodeURI)
	assert.Equal(t, "spotify:show:show1", playMsg.PlaylistURI)
}

func TestSavedEpisodesPane_DurationZero(t *testing.T) {
	s := state.New()
	s.SetSavedEpisodes([]domain.SavedEpisode{
		{
			AddedAt: "2024-05-29T12:00:00Z",
			Episode: domain.Episode{
				ID: "ep1", Name: "Episode 1", IsPlayable: true, DurationMs: 0,
			},
		},
	})
	th := theme.Load("black")
	p := NewSavedEpisodesPane(s, th, true)
	p.SetSize(80, 20)
	output := p.View()
	assert.Contains(t, output, "\u2014")
}
