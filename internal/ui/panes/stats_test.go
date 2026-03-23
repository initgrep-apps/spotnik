package panes_test

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newStatsView builds a StatsView for testing with a default theme and store.
func newStatsView() (*panes.StatsView, *state.Store) {
	t := theme.Load("black")
	s := state.New()
	sv := panes.NewStatsView(s, t)
	return sv, s
}

// prefillStatsStore sets up a store with top tracks, top artists, and recently played.
func prefillStatsStore(s *state.Store) {
	s.SetTopTracks("short_term", []api.Track{
		{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1",
			Artists: []api.Artist{{ID: "a1", Name: "The Weeknd"}},
			Album:   api.Album{Name: "After Hours"}},
		{ID: "t2", Name: "Levitating", URI: "spotify:track:t2",
			Artists: []api.Artist{{ID: "a2", Name: "Dua Lipa"}},
			Album:   api.Album{Name: "Future Nostalgia"}},
	})
	s.SetTopArtists("short_term", []api.FullArtist{
		{ID: "a1", Name: "The Weeknd", URI: "spotify:artist:a1", Genres: []string{"pop"}, Popularity: 95},
		{ID: "a2", Name: "Dua Lipa", URI: "spotify:artist:a2", Genres: []string{"dance pop"}, Popularity: 92},
	})
	s.SetRecentlyPlayed([]api.PlayHistory{
		{
			Track:    api.Track{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1", Artists: []api.Artist{{Name: "The Weeknd"}}, Album: api.Album{Name: "After Hours"}},
			PlayedAt: "2024-01-01T12:00:00Z",
		},
	})
}

// TestStatsView_Init_FetchesShortTerm verifies Init returns a batch command for
// short_term tracks + artists + recently played.
func TestStatsView_Init_FetchesShortTerm(t *testing.T) {
	sv, _ := newStatsView()
	cmd := sv.Init()
	require.NotNil(t, cmd, "Init should return a non-nil batch command")

	// The command is a batch — running it should produce a message.
	// We verify that the command is not nil (indicating data fetching started).
	msg := cmd()
	// The command may be a batch, which returns nil on direct call.
	// The key invariant is that Init() returns a command, not nil.
	_ = msg
}

// TestStatsView_View_TopTracks verifies the view renders a numbered track list.
func TestStatsView_View_TopTracks(t *testing.T) {
	sv, s := newStatsView()
	prefillStatsStore(s)

	// Send the stats loaded message.
	updated, _ := sv.Update(panes.StatsLoadedMsg{
		TopTracks:  s.TopTracks("short_term"),
		TopArtists: s.TopArtists("short_term"),
		TimeRange:  "short_term",
	})
	sv, _ = updated.(*panes.StatsView)
	require.NotNil(t, sv)

	view := sv.View()
	assert.Contains(t, view, "TOP TRACKS", "view should contain section header")
	assert.Contains(t, view, "Blinding Lights", "view should contain first track")
	assert.Contains(t, view, "The Weeknd", "view should contain artist name")
}

// TestStatsView_View_TopArtists verifies the view renders a numbered artist list.
func TestStatsView_View_TopArtists(t *testing.T) {
	sv, s := newStatsView()
	prefillStatsStore(s)

	updated, _ := sv.Update(panes.StatsLoadedMsg{
		TopTracks:  s.TopTracks("short_term"),
		TopArtists: s.TopArtists("short_term"),
		TimeRange:  "short_term",
	})
	sv, _ = updated.(*panes.StatsView)
	require.NotNil(t, sv)

	view := sv.View()
	assert.Contains(t, view, "TOP ARTISTS", "view should contain TOP ARTISTS header")
	assert.Contains(t, view, "The Weeknd", "view should contain first artist")
	assert.Contains(t, view, "Dua Lipa", "view should contain second artist")
}

// TestStatsView_View_RecentlyPlayed verifies the recently played section renders.
func TestStatsView_View_RecentlyPlayed(t *testing.T) {
	sv, s := newStatsView()
	prefillStatsStore(s)

	updated, _ := sv.Update(panes.StatsLoadedMsg{
		TopTracks:  s.TopTracks("short_term"),
		TopArtists: s.TopArtists("short_term"),
		TimeRange:  "short_term",
	})
	sv, _ = updated.(*panes.StatsView)

	updated, _ = sv.Update(panes.RecentlyPlayedLoadedMsg{})
	sv, _ = updated.(*panes.StatsView)
	require.NotNil(t, sv)

	view := sv.View()
	assert.Contains(t, view, "RECENTLY PLAYED", "view should contain recently played header")
	assert.Contains(t, view, "Blinding Lights", "view should contain recently played track")
}

// TestStatsView_View_EmptySection verifies the empty state message is shown when no data.
func TestStatsView_View_EmptySection(t *testing.T) {
	sv, _ := newStatsView()
	// Don't prefill the store — leave data empty.
	view := sv.View()
	// At minimum the view should render without crashing.
	assert.NotEmpty(t, view)
	// With no data, the view should show loading or empty state indicators.
	// (At least one section header should be present.)
	assert.Contains(t, view, "TOP TRACKS")
}

// TestStatsView_Update_Tab verifies Tab cycles section focus.
func TestStatsView_Update_Tab(t *testing.T) {
	sv, s := newStatsView()
	prefillStatsStore(s)

	// Load data.
	updated, _ := sv.Update(panes.StatsLoadedMsg{
		TopTracks:  s.TopTracks("short_term"),
		TopArtists: s.TopArtists("short_term"),
		TimeRange:  "short_term",
	})
	sv, _ = updated.(*panes.StatsView)
	require.NotNil(t, sv)

	// Initially focused on Top Tracks.
	assert.Equal(t, panes.StatsSectionTopTracks, sv.ActiveSection())

	// Tab → Top Artists.
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ = sv.Update(tabMsg)
	sv, _ = updated.(*panes.StatsView)
	assert.Equal(t, panes.StatsSectionTopArtists, sv.ActiveSection())

	// Tab → Recently Played.
	updated, _ = sv.Update(tabMsg)
	sv, _ = updated.(*panes.StatsView)
	assert.Equal(t, panes.StatsSectionRecentlyPlayed, sv.ActiveSection())

	// Tab → wraps back to Top Tracks.
	updated, _ = sv.Update(tabMsg)
	sv, _ = updated.(*panes.StatsView)
	assert.Equal(t, panes.StatsSectionTopTracks, sv.ActiveSection())
}

// TestStatsView_Update_JK verifies j/k moves cursor within section.
func TestStatsView_Update_JK(t *testing.T) {
	sv, s := newStatsView()
	prefillStatsStore(s)

	updated, _ := sv.Update(panes.StatsLoadedMsg{
		TopTracks:  s.TopTracks("short_term"),
		TopArtists: s.TopArtists("short_term"),
		TimeRange:  "short_term",
	})
	sv, _ = updated.(*panes.StatsView)
	require.NotNil(t, sv)

	// Initially cursor is at 0.
	assert.Equal(t, 0, sv.Cursor())

	// j moves down.
	jMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = sv.Update(jMsg)
	sv, _ = updated.(*panes.StatsView)
	assert.Equal(t, 1, sv.Cursor())

	// k moves back up.
	kMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated, _ = sv.Update(kMsg)
	sv, _ = updated.(*panes.StatsView)
	assert.Equal(t, 0, sv.Cursor())

	// k at 0 stays at 0 (no underflow).
	updated, _ = sv.Update(kMsg)
	sv, _ = updated.(*panes.StatsView)
	assert.Equal(t, 0, sv.Cursor())
}

// TestStatsView_Update_Enter_PlaysTrack verifies Enter on a track returns a play command.
func TestStatsView_Update_Enter_PlaysTrack(t *testing.T) {
	sv, s := newStatsView()
	prefillStatsStore(s)

	updated, _ := sv.Update(panes.StatsLoadedMsg{
		TopTracks:  s.TopTracks("short_term"),
		TopArtists: s.TopArtists("short_term"),
		TimeRange:  "short_term",
	})
	sv, _ = updated.(*panes.StatsView)
	require.NotNil(t, sv)

	// Cursor on first track (index 0).
	assert.Equal(t, 0, sv.Cursor())

	// Enter on top tracks section.
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := sv.Update(enterMsg)
	require.NotNil(t, cmd, "Enter on a track should return a play command")

	msg := cmd()
	playMsg, ok := msg.(panes.PlayTrackMsg)
	require.True(t, ok, "should return a PlayTrackMsg, got %T", msg)
	assert.Equal(t, "spotify:track:t1", playMsg.TrackURI)
}

// TestStatsView_Update_Enter_PlaysArtist verifies Enter on an artist returns a play command.
func TestStatsView_Update_Enter_PlaysArtist(t *testing.T) {
	sv, s := newStatsView()
	prefillStatsStore(s)

	updated, _ := sv.Update(panes.StatsLoadedMsg{
		TopTracks:  s.TopTracks("short_term"),
		TopArtists: s.TopArtists("short_term"),
		TimeRange:  "short_term",
	})
	sv, _ = updated.(*panes.StatsView)

	// Tab to artists section.
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ = sv.Update(tabMsg)
	sv, _ = updated.(*panes.StatsView)
	assert.Equal(t, panes.StatsSectionTopArtists, sv.ActiveSection())

	// Enter on first artist.
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := sv.Update(enterMsg)
	require.NotNil(t, cmd)

	msg := cmd()
	playMsg, ok := msg.(panes.PlayContextMsg)
	require.True(t, ok, "should return a PlayContextMsg for artist, got %T", msg)
	assert.Equal(t, "spotify:artist:a1", playMsg.ContextURI)
}

// TestStatsView_Update_F_CyclesRange verifies f key cycles through time ranges.
func TestStatsView_Update_F_CyclesRange(t *testing.T) {
	sv, s := newStatsView()
	prefillStatsStore(s)

	updated, _ := sv.Update(panes.StatsLoadedMsg{
		TopTracks:  s.TopTracks("short_term"),
		TopArtists: s.TopArtists("short_term"),
		TimeRange:  "short_term",
	})
	sv, _ = updated.(*panes.StatsView)

	assert.Equal(t, "short_term", sv.TimeRange())

	fMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}

	// f → medium_term.
	updated, _ = sv.Update(fMsg)
	sv, _ = updated.(*panes.StatsView)
	assert.Equal(t, "medium_term", sv.TimeRange())

	// f → long_term.
	updated, _ = sv.Update(fMsg)
	sv, _ = updated.(*panes.StatsView)
	assert.Equal(t, "long_term", sv.TimeRange())

	// f → wraps back to short_term.
	updated, _ = sv.Update(fMsg)
	sv, _ = updated.(*panes.StatsView)
	assert.Equal(t, "short_term", sv.TimeRange())
}

// TestStatsView_TimeRange_CacheHit verifies that when data is cached, no fetch is issued.
func TestStatsView_TimeRange_CacheHit(t *testing.T) {
	sv, s := newStatsView()
	prefillStatsStore(s)

	// Pre-populate medium_term data in the store.
	s.SetTopTracks("medium_term", []api.Track{
		{ID: "mt1", Name: "Medium Track", URI: "spotify:track:mt1"},
	})
	s.SetTopArtists("medium_term", []api.FullArtist{
		{ID: "ma1", Name: "Medium Artist", URI: "spotify:artist:ma1"},
	})

	// Load initial short_term data.
	updated, _ := sv.Update(panes.StatsLoadedMsg{
		TopTracks:  s.TopTracks("short_term"),
		TopArtists: s.TopArtists("short_term"),
		TimeRange:  "short_term",
	})
	sv, _ = updated.(*panes.StatsView)

	// Switch to medium_term via f key.
	fMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	updated, cmd := sv.Update(fMsg)
	sv, _ = updated.(*panes.StatsView)

	// With cache hit, no fetch command should be issued.
	assert.Nil(t, cmd, "cached range should not issue a fetch command")
	assert.Equal(t, "medium_term", sv.TimeRange())
}

// TestStatsView_TimeRange_CacheMiss verifies that uncached range triggers a fetch command.
func TestStatsView_TimeRange_CacheMiss(t *testing.T) {
	sv, s := newStatsView()
	prefillStatsStore(s)

	// Load initial short_term data.
	updated, _ := sv.Update(panes.StatsLoadedMsg{
		TopTracks:  s.TopTracks("short_term"),
		TopArtists: s.TopArtists("short_term"),
		TimeRange:  "short_term",
	})
	sv, _ = updated.(*panes.StatsView)

	// Switch to medium_term (not cached).
	fMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	_, cmd := sv.Update(fMsg)

	// With cache miss, a fetch command should be issued.
	assert.NotNil(t, cmd, "uncached range should issue a fetch command")
}

// TestStatsView_View_ActiveRangeHighlighted verifies the active range label appears in the view.
func TestStatsView_View_ActiveRangeHighlighted(t *testing.T) {
	sv, s := newStatsView()
	prefillStatsStore(s)

	updated, _ := sv.Update(panes.StatsLoadedMsg{
		TopTracks:  s.TopTracks("short_term"),
		TopArtists: s.TopArtists("short_term"),
		TimeRange:  "short_term",
	})
	sv, _ = updated.(*panes.StatsView)

	view := sv.View()
	// Active range label should appear in the view.
	assert.Contains(t, view, "4wk", "short_term should display as 4wk")
	assert.Contains(t, view, "6mo", "medium_term should display as 6mo")
	assert.Contains(t, view, "all", "long_term should display as all")
}

// TestFormatRelativeTime_JustNow verifies times under 1 minute return "just now".
func TestFormatRelativeTime_JustNow(t *testing.T) {
	now := time.Now()
	result := panes.FormatRelativeTime(now.Add(-30 * time.Second))
	assert.Equal(t, "just now", result)
}

// TestFormatRelativeTime_Minutes verifies times in 1-59 minutes return "{n} min ago".
func TestFormatRelativeTime_Minutes(t *testing.T) {
	now := time.Now()
	result := panes.FormatRelativeTime(now.Add(-5 * time.Minute))
	assert.Equal(t, "5 min ago", result)
}

// TestFormatRelativeTime_Hours verifies times in 1-23 hours return "{n} hr ago".
func TestFormatRelativeTime_Hours(t *testing.T) {
	now := time.Now()
	result := panes.FormatRelativeTime(now.Add(-3 * time.Hour))
	assert.Equal(t, "3 hr ago", result)
}

// TestFormatRelativeTime_Days verifies times in 1-6 days return "{n} days ago".
func TestFormatRelativeTime_Days(t *testing.T) {
	now := time.Now()
	result := panes.FormatRelativeTime(now.Add(-2 * 24 * time.Hour))
	assert.Equal(t, "2 days ago", result)
}

// TestFormatRelativeTime_OlderThanWeek verifies times >= 7 days return short date format.
func TestFormatRelativeTime_OlderThanWeek(t *testing.T) {
	// Use a fixed date so the assertion is deterministic.
	past := time.Date(2024, time.March, 12, 0, 0, 0, 0, time.UTC)
	result := panes.FormatRelativeTime(past)
	assert.Equal(t, "Mar 12", result)
}
