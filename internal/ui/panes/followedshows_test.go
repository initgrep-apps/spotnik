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

var _ layout.Pane = &FollowedShowsPane{}

func TestFollowedShowsPane_ID(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, false)
	assert.Equal(t, layout.PaneFollowedShows, p.ID())
}

func TestFollowedShowsPane_ToggleKey(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, false)
	assert.Equal(t, 3, p.ToggleKey())
}

func TestFollowedShowsPane_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)
	output := p.View()
	assert.Contains(t, output, "No followed shows")
}

func TestFollowedShowsPane_EnterSelectsShow(t *testing.T) {
	s := state.New()
	s.SetFollowedShows([]domain.SavedShow{
		{Show: domain.Show{ID: "show1", Name: "Show 1", Publisher: "Pub 1"}},
	})
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	selectMsg, ok := msg.(SelectedShowChangedMsg)
	require.True(t, ok)
	assert.Equal(t, "show1", selectMsg.ShowID)
}

func TestFollowedShowsPane_EnterSameShow(t *testing.T) {
	s := state.New()
	s.SetFollowedShows([]domain.SavedShow{
		{Show: domain.Show{ID: "show1", Name: "Show 1", Publisher: "Pub 1"}},
	})
	s.SetSelectedShowID("show1")
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "Enter on same show should be no-op")
}
