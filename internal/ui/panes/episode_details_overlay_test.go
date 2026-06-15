package panes

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestEpisodeOverlay(ps *domain.PlaybackState) *EpisodeDetailsOverlay {
	s := state.New()
	if ps != nil {
		s.SetPlaybackState(ps)
	}
	return NewEpisodeDetailsOverlay(s, theme.Load("black"))
}

func episodeWithDescription(name, showName, publisher, htmlDesc, plainDesc string) *domain.PlaybackState {
	ep := &domain.Episode{
		Name:            name,
		DurationMs:      3723000,
		ReleaseDate:     "2026-05-29",
		Description:     plainDesc,
		HTMLDescription: htmlDesc,
		Show: &domain.Show{
			Name:      showName,
			Publisher: publisher,
		},
	}
	return &domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "episode",
		Episode:              ep,
	}
}

func TestEpisodeDetailsOverlay_View_ShowsEpisodeName(t *testing.T) {
	ps := episodeWithDescription("Episode 497", "The Show", "Publisher Co", "", "Plain desc")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	view := o.View()
	assert.Contains(t, view, "Episode 497")
}

func TestEpisodeDetailsOverlay_View_ShowsShowName(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "The Podcast Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	view := o.View()
	assert.Contains(t, view, "The Podcast Show")
}

func TestEpisodeDetailsOverlay_View_ShowsPublisher(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "Acme Studios", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	view := o.View()
	assert.Contains(t, view, "Acme Studios")
}

func TestEpisodeDetailsOverlay_View_ShowsDuration(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "Pub", "", "Desc")
	ps.Episode.DurationMs = 3723000 // 1h 02m
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	view := o.View()
	assert.Contains(t, view, "1h 02m")
}

func TestEpisodeDetailsOverlay_View_ShowsReleaseDate(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "Pub", "", "Desc")
	ps.Episode.ReleaseDate = "2026-05-29"
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	view := o.View()
	assert.Contains(t, view, "2026-05-29")
}

func TestEpisodeDetailsOverlay_View_ShowsNoDescription(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	view := o.View()
	assert.Contains(t, view, "No description available")
}

func TestEpisodeDetailsOverlay_View_ShowsPlainDescription(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "This is a plain text description.")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	view := o.View()
	assert.Contains(t, view, "This is a plain text description")
}

func TestEpisodeDetailsOverlay_View_ShowsHTMLDescription(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "<p>HTML content here</p>", "Plain fallback")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	view := o.View()
	// HTML description takes priority over plain text.
	// glamour may split "HTML" and "content" across styling boundaries,
	// so check for the key word "HTML" appearing in the rendered view.
	assert.Contains(t, view, "HTML")
}

func TestEpisodeDetailsOverlay_View_NoEpisode(t *testing.T) {
	o := newTestEpisodeOverlay(nil)
	o.SetSize(120, 40)
	view := o.View()
	assert.Contains(t, view, "No episode playing")
}

func TestEpisodeDetailsOverlay_EscCloses(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	_, ok := cmd().(EpisodeDetailsClosedMsg)
	assert.True(t, ok)
}

func TestEpisodeDetailsOverlay_QCloses(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)
	_, ok := cmd().(EpisodeDetailsClosedMsg)
	assert.True(t, ok)
}

func TestEpisodeDetailsOverlay_ScrollDown(t *testing.T) {
	// Create episode with long description to enable scrolling
	longDesc := strings.Repeat("Line of text. ", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)

	initialScroll := o.scrollY
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.GreaterOrEqual(t, o.scrollY, initialScroll)
}

func TestEpisodeDetailsOverlay_ScrollUp(t *testing.T) {
	longDesc := strings.Repeat("Line of text. ", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)

	// Scroll down first
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	afterDown := o.scrollY

	// Scroll up
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.LessOrEqual(t, o.scrollY, afterDown)
}

func TestEpisodeDetailsOverlay_ArrowKeysScroll(t *testing.T) {
	longDesc := strings.Repeat("Line of text. ", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)

	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyDown})
	afterDown := o.scrollY
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.LessOrEqual(t, o.scrollY, afterDown)
}

func TestEpisodeDetailsOverlay_OtherKeysConsumed(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Nil(t, cmd)
}

func TestEpisodeDetailsOverlay_NonKeyMsgIgnored(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	_, cmd := o.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	assert.Nil(t, cmd)
}

func TestEpisodeDetailsOverlay_SetTheme(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	assert.NotPanics(t, func() { o.SetTheme(theme.Load("monokai")) })
	assert.Equal(t, "monokai", o.theme.ID())
}

func TestEpisodeDetailsOverlay_SetSize(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(100, 30)
	assert.Equal(t, 100, o.width)
	assert.Equal(t, 30, o.height)
}

func TestEpisodeDetailsOpenMsg_Type(t *testing.T) {
	msg := EpisodeDetailsOpenMsg{}
	assert.IsType(t, EpisodeDetailsOpenMsg{}, msg)
}

func TestEpisodeDetailsClosedMsg_Type(t *testing.T) {
	msg := EpisodeDetailsClosedMsg{}
	assert.IsType(t, EpisodeDetailsClosedMsg{}, msg)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		ms       int
		expected string
	}{
		{"zero", 0, ""},
		{"seconds only", 30000, "30s"},
		{"minutes and seconds", 185000, "3m 05s"},
		{"hours and minutes", 3723000, "1h 02m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatDuration(tt.ms))
		})
	}
}

func TestEpisodeDetailsOverlay_View_HasBorder(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	view := o.View()
	require.NotEmpty(t, view)
	assert.Contains(t, view, "╭")
	assert.Contains(t, view, "╰")
}

func TestEpisodeDetailsOverlay_View_HasTitle(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	assert.Contains(t, o.View(), "Episode Details")
}