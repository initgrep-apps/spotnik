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

func TestEpisodeDetailsOverlay_View_NarrowTerminal(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "Publisher Co", "", "A plain description")
	o := newTestEpisodeOverlay(ps)

	var view string
	assert.NotPanics(t, func() {
		o.SetSize(50, 40)
		view = o.View()
	}, "View() on narrow terminal should not panic")

	assert.NotEmpty(t, view, "narrow terminal view should not be empty")
	assert.Contains(t, view, "Episode Details", "narrow terminal view should contain title")
	assert.Contains(t, view, "╭", "narrow terminal view should have top border")
	assert.Contains(t, view, "╰", "narrow terminal view should have bottom border")
}

func TestEpisodeDetailsOverlay_View_HasTitle(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	assert.Contains(t, o.View(), "Episode Details")
}

func TestEpisodeDetailsOverlay_ViewportTinyTerminal(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "Pub", "", "Desc")
	o := newTestEpisodeOverlay(ps)

	assert.NotPanics(t, func() {
		o.SetSize(3, 10)
		o.View()
	}, "tiny terminal (3x10) should not panic")
}

// --- Viewport-specific tests ---

func TestEpisodeDetailsOverlay_ViewportInitialized(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	o.View()
	assert.Greater(t, o.viewport.Width, 0, "viewport width should be set after View()")
	assert.Greater(t, o.viewport.Height, 0, "viewport height should be set after View()")
	assert.True(t, o.viewport.MouseWheelEnabled, "mouse wheel should be enabled")
}

func TestEpisodeDetailsOverlay_ViewportScrollsDown(t *testing.T) {
	longDesc := strings.Repeat("Line of text.\n", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	o.View() // initializes viewport with content

	beforeY := o.viewport.YOffset
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Greater(t, o.viewport.YOffset, beforeY, "viewport should scroll down on 'j'")
}

func TestEpisodeDetailsOverlay_ViewportScrollsUp(t *testing.T) {
	longDesc := strings.Repeat("Line of text.\n", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	o.View()

	// Scroll down first
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	afterDown := o.viewport.YOffset
	require.Greater(t, afterDown, 0, "should have scrolled down")

	// Scroll up
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Less(t, o.viewport.YOffset, afterDown, "viewport should scroll up on 'k'")
}

func TestEpisodeDetailsOverlay_ArrowKeysScroll(t *testing.T) {
	longDesc := strings.Repeat("Line of text.\n", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	o.View()

	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyDown})
	afterDown := o.viewport.YOffset
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Less(t, o.viewport.YOffset, afterDown)
}

func TestEpisodeDetailsOverlay_PageUpDownScrolls(t *testing.T) {
	longDesc := strings.Repeat("Line of text.\n", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	o.View()

	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Greater(t, o.viewport.YOffset, 0, "PgDn should scroll down")

	afterDown := o.viewport.YOffset
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	assert.Less(t, o.viewport.YOffset, afterDown, "PgUp should scroll up")
}

func TestEpisodeDetailsOverlay_HomeJumpsToTop(t *testing.T) {
	longDesc := strings.Repeat("Line of text.\n", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	o.View()

	// Scroll down a bunch
	for i := 0; i < 20; i++ {
		_, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	require.Greater(t, o.viewport.YOffset, 0, "should have scrolled down")

	// Home
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyHome})
	assert.Equal(t, 0, o.viewport.YOffset, "Home should jump to top")
}

func TestEpisodeDetailsOverlay_EndJumpsToBottom(t *testing.T) {
	longDesc := strings.Repeat("Line of text.\n", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	o.View()

	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyEnd})
	assert.True(t, o.viewport.AtBottom(), "End should jump to bottom")
}

func TestEpisodeDetailsOverlay_MouseWheelScrolls(t *testing.T) {
	longDesc := strings.Repeat("Line of text.\n", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	o.View()

	beforeY := o.viewport.YOffset
	_, _ = o.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	assert.Greater(t, o.viewport.YOffset, beforeY, "mouse wheel down should scroll down")

	afterDown := o.viewport.YOffset
	_, _ = o.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	assert.Less(t, o.viewport.YOffset, afterDown, "mouse wheel up should scroll up")
}

func TestEpisodeDetailsOverlay_ViewportScrollSurvivesView(t *testing.T) {
	longDesc := strings.Repeat("Line of text.\n", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	o.View()

	// Scroll down 2 lines
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	afterScroll := o.viewport.YOffset
	require.Greater(t, afterScroll, 0)

	// View() must not reset scroll position
	o.View()
	assert.Equal(t, afterScroll, o.viewport.YOffset, "scroll position should survive View() calls")
}

func TestEpisodeDetailsOverlay_View_ShowsScrollPercent(t *testing.T) {
	longDesc := strings.Repeat("Line of text.\n", 200)
	ps := episodeWithDescription("Ep 1", "Show", "", "", longDesc)
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	o.View()

	// At top, should show 0%
	view := o.View()
	assert.Contains(t, view, "0%", "should show 0% at top")

	// Jump to bottom, should show 100%
	_, _ = o.Update(tea.KeyMsg{Type: tea.KeyEnd})
	view = o.View()
	assert.Contains(t, view, "100%", "should show 100% at bottom")
}

func TestEpisodeDetailsOverlay_KeybarShowsEscClose(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	view := o.View()
	assert.Contains(t, view, "Esc", "keybar should show Esc hint")
	assert.Contains(t, view, "close", "keybar should show close hint")
}

func TestEpisodeDetailsOverlay_SetSize_InitializesViewport(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)

	// resizeViewport is called in SetSize — viewport should get dimensions
	assert.Greater(t, o.viewport.Width, 0, "SetSize should set viewport width")
	assert.Greater(t, o.viewport.Height, 0, "SetSize should set viewport height")
}

func TestEpisodeDetailsOverlay_SetSize_ResizesViewport(t *testing.T) {
	ps := episodeWithDescription("Ep 1", "Show", "", "", "Desc")
	o := newTestEpisodeOverlay(ps)
	o.SetSize(120, 40)
	firstW := o.viewport.Width

	o.SetSize(60, 20)
	// Narrow terminal: overlayWidth becomes min(80, 60) = 60, viewport width = 60-2 = 58
	assert.NotEqual(t, firstW, o.viewport.Width, "viewport should resize on SetSize")
}
