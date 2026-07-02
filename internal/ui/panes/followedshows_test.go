package panes

import (
	"fmt"
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

func TestFollowedShowsPane_InitialState_Level1(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	assert.False(t, p.inEpisodeView, "should start in show list view")
	assert.Equal(t, "", p.selectedShowID)
	assert.Equal(t, "Followed Shows", p.Title())
	assert.Len(t, p.Actions(), 1)
	assert.Equal(t, "filter", p.Actions()[0].Label)
}

// ── Level 1: Enter show enters episode view ──────────────────────────────

func TestFollowedShows_EnterShow_EntersEpisodeView(t *testing.T) {
	s := state.New()
	s.SetFollowedShows([]domain.SavedShow{
		{Show: domain.Show{ID: "show1", Name: "Show 1", Publisher: "Pub 1", TotalEpisodes: 42}},
	})
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	reqMsg, ok := msg.(FetchShowEpisodesRequestMsg)
	require.True(t, ok, "should emit FetchShowEpisodesRequestMsg")
	assert.Equal(t, "show1", reqMsg.ShowID)
	assert.Equal(t, 0, reqMsg.Offset)
	assert.True(t, p.inEpisodeView)
	assert.Equal(t, "show1", p.selectedShowID)
	assert.Equal(t, "Show 1", p.selectedShowName)
	assert.True(t, p.episodesFetching)
}

func TestFollowedShows_EnterSameShow_NoOp(t *testing.T) {
	s := state.New()
	s.SetFollowedShows([]domain.SavedShow{
		{Show: domain.Show{ID: "show1", Name: "Show 1", Publisher: "Pub 1"}},
	})
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	// First Enter: enters episode view
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	assert.True(t, p.inEpisodeView)
	assert.Equal(t, "show1", p.selectedShowID)

	// Second Enter on same show in Level 1: no-op
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	_, _ = p.Update(escMsg)
	assert.False(t, p.inEpisodeView, "Esc should return to show list")

	_, cmd2 := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd2, "Enter on same show in Level 1 should be no-op")
}

// ── Level 1: Enter different show resets episode state ───────────────────

func TestFollowedShows_EnterDifferentShow_ResetsEpisodes(t *testing.T) {
	s := state.New()
	s.SetFollowedShows([]domain.SavedShow{
		{Show: domain.Show{ID: "show1", Name: "Show 1", Publisher: "Pub 1", TotalEpisodes: 10}},
		{Show: domain.Show{ID: "show2", Name: "Show 2", Publisher: "Pub 2", TotalEpisodes: 20}},
	})
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	// Enter show1
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, p.inEpisodeView)
	assert.Equal(t, "show1", p.selectedShowID)

	// Feed episode data to show1 so we have state to reset
	p.loadedEpisodes = []domain.Episode{{ID: "ep1", Name: "Ep 1", IsPlayable: true, URI: "spotify:episode:ep1"}}
	p.episodesOffset = 1
	p.episodesTotal = 10
	p.hasMoreEpisodes = true
	p.episodesFetching = false

	// Return to show list (Esc in episode view)
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	_, _ = p.Update(escMsg)
	assert.False(t, p.inEpisodeView)

	// Navigate to show2 (row index 1) — press 'j' to move down
	p.SetFocused(true)
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Enter show2
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	reqMsg, ok := msg.(FetchShowEpisodesRequestMsg)
	require.True(t, ok)
	assert.Equal(t, "show2", reqMsg.ShowID)
	assert.Equal(t, 0, reqMsg.Offset)

	// Episode state should be reset
	assert.True(t, p.inEpisodeView)
	assert.Equal(t, "show2", p.selectedShowID)
	assert.Equal(t, "Show 2", p.selectedShowName)
	assert.Nil(t, p.loadedEpisodes)
	assert.Equal(t, 0, p.episodesOffset)
	assert.Equal(t, 0, p.episodesTotal)
	assert.False(t, p.hasMoreEpisodes)
	assert.True(t, p.episodesFetching)
}

// ── Level 2: Episode view rendering ─────────────────────────────────────

func TestFollowedShows_EpisodeView_ColumnHeaders(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	// Manually enter episode view
	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.selectedShowName = "Show 1"
	p.buildEpisodeRows()

	output := p.View()
	assert.Contains(t, output, "Title")
	assert.Contains(t, output, "Released")
	assert.Contains(t, output, "Dur")
}

func TestFollowedShows_EpisodeView_ShowsEpisodeData(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.selectedShowName = "Show 1"
	p.loadedEpisodes = []domain.Episode{
		{ID: "ep1", Name: "Episode One", ReleaseDate: "2024-01-15", DurationMs: 1800000, IsPlayable: true, URI: "spotify:episode:ep1"},
		{ID: "ep2", Name: "Episode Two", ReleaseDate: "2024-02-20", DurationMs: 3600000, IsPlayable: false, URI: "spotify:episode:ep2"},
	}
	p.buildEpisodeRows()

	output := p.View()
	assert.Contains(t, output, "Episode One")
	assert.Contains(t, output, "Episode Two")
	assert.Contains(t, output, "30:00")   // 1800000ms = 30min
	assert.Contains(t, output, "1:00:00") // 3600000ms = 1h
}

// ── Level 2: Enter playable episode ─────────────────────────────────────

func TestFollowedShows_EnterPlayableEpisode_Plays(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.selectedShowName = "Show 1"
	p.loadedEpisodes = []domain.Episode{
		{ID: "ep1", Name: "Episode One", IsPlayable: true, URI: "spotify:episode:ep1"},
	}
	p.buildEpisodeRows()
	// Cursor defaults to row 0

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	playMsg, ok := msg.(PlayEpisodeMsg)
	require.True(t, ok)
	assert.Equal(t, "spotify:episode:ep1", playMsg.EpisodeURI)
	assert.Equal(t, "spotify:show:show1", playMsg.PlaylistURI)
}

func TestFollowedShows_EnterUnplayableEpisode_Toasts(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.selectedShowName = "Show 1"
	p.loadedEpisodes = []domain.Episode{
		{ID: "ep1", Name: "Locked Episode", IsPlayable: false, URI: "spotify:episode:ep1"},
	}
	p.buildEpisodeRows()
	// Cursor defaults to row 0

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "unplayable episode should return nil cmd")
}

// ── Level 2: Esc returns to Level 1 ─────────────────────────────────────

func TestFollowedShows_Esc_ReturnsToLevel1(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.selectedShowName = "Show 1"
	p.episodesFetching = true

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(FollowedShowsViewClosedMsg)
	assert.True(t, ok, "Esc should emit FollowedShowsViewClosedMsg")
	assert.False(t, p.inEpisodeView)
}

// ── Title / Actions dynamic rendering ────────────────────────────────────

func TestFollowedShows_Title_Level1(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	assert.Equal(t, "Followed Shows", p.Title())
}

func TestFollowedShows_Title_Level2(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)

	p.inEpisodeView = true
	p.selectedShowName = "Test Show"
	p.episodesTotal = 42

	title := p.Title()
	assert.Contains(t, title, "Test Show")
	assert.Contains(t, title, "Followed Shows")
}

func TestFollowedShows_Actions_Level1(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	actions := p.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "filter", actions[0].Label)
}

func TestFollowedShows_Actions_Level2(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)

	p.inEpisodeView = true
	actions := p.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "back", actions[0].Label)
}

// ── ShowEpisodesLoadedMsg handling ──────────────────────────────────────

func TestFollowedShows_ShowEpisodesLoaded_UpdatesState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.selectedShowName = "Show 1"
	p.episodesFetching = true

	msg := ShowEpisodesLoadedMsg{
		ShowID: "show1",
		Items: []domain.Episode{
			{ID: "ep1", Name: "Episode One", DurationMs: 1800000, ReleaseDate: "2024-01-15", IsPlayable: true, URI: "spotify:episode:ep1"},
			{ID: "ep2", Name: "Episode Two", DurationMs: 3600000, ReleaseDate: "2024-02-20", IsPlayable: true, URI: "spotify:episode:ep2"},
		},
		Total:   42,
		HasNext: true,
		Offset:  0,
	}

	_, cmd := p.Update(msg)
	assert.Nil(t, cmd)
	assert.False(t, p.episodesFetching)
	assert.Len(t, p.loadedEpisodes, 2)
	assert.Equal(t, 42, p.episodesTotal)
	assert.Equal(t, 2, p.episodesOffset)
	assert.True(t, p.hasMoreEpisodes)
}

func TestFollowedShows_ShowEpisodesLoaded_StaleDiscards(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)

	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.episodesFetching = true
	p.loadedEpisodes = []domain.Episode{{ID: "old", Name: "Old Episode"}}
	p.episodesOffset = 1

	msg := ShowEpisodesLoadedMsg{
		ShowID: "show2",
		Items:  []domain.Episode{{ID: "new", Name: "New Episode"}},
		Total:  5,
	}

	_, cmd := p.Update(msg)
	assert.Nil(t, cmd)
	assert.True(t, p.episodesFetching, "fetching state should not be cleared for stale response")
	assert.Len(t, p.loadedEpisodes, 1, "episodes should not be updated for stale response")
	assert.Equal(t, "Old Episode", p.loadedEpisodes[0].Name)
}

func TestFollowedShows_ShowEpisodesLoaded_Append(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)

	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.episodesFetching = true
	p.loadedEpisodes = []domain.Episode{
		{ID: "ep1", Name: "Episode One"},
		{ID: "ep2", Name: "Episode Two"},
	}
	p.episodesOffset = 2

	msg := ShowEpisodesLoadedMsg{
		ShowID: "show1",
		Items: []domain.Episode{
			{ID: "ep3", Name: "Episode Three"},
		},
		Total:   42,
		HasNext: false,
		Offset:  2,
	}

	_, _ = p.Update(msg)
	assert.False(t, p.episodesFetching)
	assert.Len(t, p.loadedEpisodes, 3)
	assert.Equal(t, "Episode Three", p.loadedEpisodes[2].Name)
	assert.Equal(t, 42, p.episodesTotal)
	assert.Equal(t, 3, p.episodesOffset)
	assert.False(t, p.hasMoreEpisodes)
}

// ── Pagination ──────────────────────────────────────────────────────────

func TestFollowedShows_Pagination_PrefetchNearEnd(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.selectedShowName = "Show 1"
	p.hasMoreEpisodes = true
	p.episodesFetching = false
	p.episodesOffset = 7

	eps := make([]domain.Episode, 7)
	for i := 0; i < 7; i++ {
		eps[i] = domain.Episode{ID: fmt.Sprintf("ep%d", i), Name: fmt.Sprintf("Ep %d", i), IsPlayable: true}
	}
	p.loadedEpisodes = eps
	p.buildEpisodeRows()

	// First Down press: cursor is 0, threshold = 7-5=2.
	// 0 < 2 so the guard "cursor < len-5" returns nil. No prefetch.
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Second Down press: cursor is still 0 (table's SelectedIndex doesn't advance in unit tests).
	// But the threshold check uses SelectedIndex() which is 0, so no prefetch.
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Since the bubble-table's cursor doesn't advance in unit tests due to viewport
	// constraints, we test the prefetch logic by directly setting conditions that
	// satisfy the threshold: cursor must be >= len(loadedEpisodes)-5.
	// With len=2, threshold=-3, cursor=0 >= -3 → prefetch always fires for tiny sets.
	p.loadedEpisodes = []domain.Episode{
		{ID: "ep1", Name: "Ep 1", IsPlayable: true},
		{ID: "ep2", Name: "Ep 2", IsPlayable: true},
	}
	p.episodesFetching = false
	p.buildEpisodeRows()

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyDown})
	require.NotNil(t, cmd, "should trigger prefetch on small set where cursor >= len-5")

	msg := cmd()
	_, ok := msg.(FetchShowEpisodesRequestMsg)
	require.True(t, ok)
	assert.True(t, p.episodesFetching)
}

func TestFollowedShows_Pagination_NoPrefetchWhenAllLoaded(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.selectedShowName = "Show 1"
	p.hasMoreEpisodes = false // no more pages
	p.episodesFetching = false
	p.episodesOffset = 50

	eps := make([]domain.Episode, 50)
	for i := 0; i < 50; i++ {
		eps[i] = domain.Episode{ID: fmt.Sprintf("ep%d", i), Name: fmt.Sprintf("Ep %d", i), IsPlayable: true}
	}
	p.loadedEpisodes = eps
	p.buildEpisodeRows()

	for i := 0; i < 48; i++ {
		_, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Nil(t, cmd, "should not prefetch when hasMoreEpisodes is false")
}

func TestFollowedShows_Pagination_NoPrefetchWhileFetching(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.selectedShowName = "Show 1"
	p.hasMoreEpisodes = true
	p.episodesFetching = true // already fetching
	p.episodesOffset = 50

	eps := make([]domain.Episode, 50)
	for i := 0; i < 50; i++ {
		eps[i] = domain.Episode{ID: fmt.Sprintf("ep%d", i), Name: fmt.Sprintf("Ep %d", i), IsPlayable: true}
	}
	p.loadedEpisodes = eps
	p.buildEpisodeRows()

	for i := 0; i < 48; i++ {
		_, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Nil(t, cmd, "should not prefetch when already fetching")
}

// ── Drill-down state persists across SetSize/SetFocused ─────────────────

func TestFollowedShows_DrillDownState_Persists(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	p.inEpisodeView = true
	p.selectedShowID = "show1"
	p.selectedShowName = "Show 1"
	p.episodesTotal = 42
	p.loadedEpisodes = []domain.Episode{{ID: "ep1", Name: "Test", IsPlayable: true, URI: "spotify:episode:ep1"}}
	p.buildEpisodeRows()

	p.SetFocused(false)
	p.SetSize(40, 10)
	p.SetFocused(true)

	assert.True(t, p.inEpisodeView)
	assert.Equal(t, "show1", p.selectedShowID)
	assert.Equal(t, "Show 1", p.selectedShowName)
	assert.Equal(t, 42, p.episodesTotal)
	assert.Len(t, p.loadedEpisodes, 1)
}

// ── View renders empty state only in Level 1 ────────────────────────────

func TestFollowedShows_View_Level2_NoEmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewFollowedShowsPane(s, th, true)
	p.SetSize(80, 20)

	p.inEpisodeView = true
	p.selectedShowName = "Show 1"
	p.loadedEpisodes = nil
	p.buildEpisodeRows()

	output := p.View()
	assert.NotContains(t, output, "No followed shows")
}
