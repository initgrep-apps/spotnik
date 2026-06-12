package panes

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodcastPlaybackPane_ID(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, false)
	assert.Equal(t, layout.PanePodcastPlayback, p.ID())
}

func TestPodcastPlaybackPane_Title(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, false)
	assert.Equal(t, "Now Playing", p.Title())
}

func TestPodcastPlaybackPane_ToggleKey(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, false)
	assert.Equal(t, 1, p.ToggleKey())
}

func TestPodcastPlaybackPane_Actions(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, false)
	assert.Nil(t, p.Actions())
}

func TestPodcastPlaybackPane_EmptyState(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)
	p.SetSize(80, 24)
	output := p.View()

	assert.Contains(t, output, "No podcast playing", "empty state should show message")
	assert.Contains(t, output, "Press / to search for shows", "empty state should show hint")
}

func TestPodcastPlaybackPane_EmptyState_WhenTrack(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           30000,
		CurrentlyPlayingType: "track",
		Item: &domain.Track{
			ID:         "track-1",
			Name:       "Song",
			DurationMs: 200000,
		},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)
	p.SetSize(80, 24)
	output := p.View()

	assert.Contains(t, output, "No podcast playing", "empty state when item is track")
}

func TestPodcastPlaybackPane_EpisodeView(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           60000,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:          "ep-1",
			Name:        "Test Episode Title",
			Description: "This is a test episode description for the podcast playback pane.",
			DurationMs:  1800000,
			ReleaseDate: "2024-01-15",
			Show: &domain.Show{
				Name:      "Test Show",
				Publisher: "Test Publisher",
			},
		},
		Device: &domain.Device{
			ID:            "dev-1",
			Name:          "Test Device",
			VolumePercent: 70,
		},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)
	p.SetSize(80, 24)
	output := p.View()

	assert.Contains(t, output, "Test Episode Title", "should show episode title")
	assert.Contains(t, output, "Test Show", "should show show name")
	assert.Contains(t, output, "Released: 2024-01-15", "should show release date")
	assert.Contains(t, output, "· Duration: 30m", "should show duration")
	assert.Contains(t, output, "Publisher: Test Publisher", "should show publisher")
	assert.Contains(t, output, "test episode description", "should show description")
}

func TestPodcastPlaybackPane_ProgressBar(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           60000,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:          "ep-1",
			Name:        "Episode",
			Description: "Description",
			DurationMs:  1800000,
			ReleaseDate: "2024-01-15",
		},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)
	p.SetSize(80, 24)
	output := p.View()

	assert.Contains(t, output, "1:00", "should show current time")
	assert.Contains(t, output, "30:00", "should show total time")
}

func TestPodcastPlaybackPane_KeyPlayPause(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           30000,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:         "ep-1",
			Name:       "Episode",
			DurationMs: 1800000,
		},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)

	msg := tea.KeyMsg{Type: tea.KeySpace}
	_, cmd := p.Update(msg)
	require.NotNil(t, cmd, "space when playing should return a command")
	result := cmd()
	req, ok := result.(PlaybackRequestMsg)
	require.True(t, ok, "space should produce PlaybackRequestMsg, got %T", result)
	assert.Equal(t, ActionPause, req.Action, "playing → space should request pause")

	// Paused
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            false,
		ProgressMs:           30000,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:         "ep-1",
			Name:       "Episode",
			DurationMs: 1800000,
		},
	})
	p = NewPodcastPlaybackPane(s, th, true)
	_, cmd = p.Update(msg)
	require.NotNil(t, cmd, "space when paused should return a command")
	result = cmd()
	req, ok = result.(PlaybackRequestMsg)
	require.True(t, ok, "space should produce PlaybackRequestMsg, got %T", result)
	assert.Equal(t, ActionPlay, req.Action, "paused → space should request play")
}

func TestPodcastPlaybackPane_KeySeekBackward(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           120000,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:         "ep-1",
			Name:       "Episode",
			DurationMs: 1800000,
		},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)
	p.localProgressMs = 120000

	msg := tea.KeyMsg{Type: tea.KeyLeft}
	_, cmd := p.Update(msg)
	require.NotNil(t, cmd, "left arrow should return a command")
	result := cmd()
	seekMsg, ok := result.(SeekIntentMsg)
	require.True(t, ok, "left should produce SeekIntentMsg, got %T", result)
	assert.Equal(t, 115000, seekMsg.TargetMs, "should seek backward 5s from 120000")
}

func TestPodcastPlaybackPane_KeySeekForward(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           120000,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:         "ep-1",
			Name:       "Episode",
			DurationMs: 1800000,
		},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)
	p.localProgressMs = 120000

	msg := tea.KeyMsg{Type: tea.KeyRight}
	_, cmd := p.Update(msg)
	require.NotNil(t, cmd, "right arrow should return a command")
	result := cmd()
	seekMsg, ok := result.(SeekIntentMsg)
	require.True(t, ok, "right should produce SeekIntentMsg, got %T", result)
	assert.Equal(t, 125000, seekMsg.TargetMs, "should seek forward 5s from 120000")
}

func TestPodcastPlaybackPane_KeySeekBackwardClamp(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           2000,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:         "ep-1",
			Name:       "Episode",
			DurationMs: 1800000,
		},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)
	p.localProgressMs = 2000

	msg := tea.KeyMsg{Type: tea.KeyLeft}
	_, cmd := p.Update(msg)
	require.NotNil(t, cmd)
	result := cmd()
	seekMsg, ok := result.(SeekIntentMsg)
	require.True(t, ok)
	assert.Equal(t, 0, seekMsg.TargetMs, "should clamp to 0")
}

func TestPodcastPlaybackPane_KeySeekForwardClamp(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           1795000,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:         "ep-1",
			Name:       "Episode",
			DurationMs: 1800000,
		},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)
	p.localProgressMs = 1795000

	msg := tea.KeyMsg{Type: tea.KeyRight}
	_, cmd := p.Update(msg)
	require.NotNil(t, cmd)
	result := cmd()
	seekMsg, ok := result.(SeekIntentMsg)
	require.True(t, ok)
	assert.Equal(t, 1800000, seekMsg.TargetMs, "should clamp to duration")
}

func TestPodcastPlaybackPane_KeyPreviousNext(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           30000,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{ID: "ep-1", Name: "Episode", DurationMs: 1800000},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)

	tests := []struct {
		name     string
		key      tea.KeyType
		want     PlaybackAction
	}{
		{"shift+left", tea.KeyShiftLeft, ActionPrevious},
		{"shift+right", tea.KeyShiftRight, ActionNext},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tt.key}
			_, cmd := p.Update(msg)
			require.NotNil(t, cmd)
			result := cmd()
			req, ok := result.(PlaybackRequestMsg)
			require.True(t, ok, "got %T", result)
			assert.Equal(t, tt.want, req.Action)
		})
	}
}

func TestPodcastPlaybackPane_KeyVolume(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           30000,
		CurrentlyPlayingType: "episode",
		Episode:              &domain.Episode{ID: "ep-1", Name: "Episode", DurationMs: 1800000},
		Device:               &domain.Device{ID: "dev-1", VolumePercent: 70},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)

	tests := []struct {
		name string
		key  string
		want int
	}{
		{"volume up", "+", 75},
		{"volume down", "-", 65},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			_, cmd := p.Update(msg)
			require.NotNil(t, cmd)
			result := cmd()
			volMsg, ok := result.(VolumeIntentMsg)
			require.True(t, ok, "got %T", result)
			assert.Equal(t, tt.want, volMsg.TargetVol)
		})
	}
}

func TestPodcastPlaybackPane_KeyShuffleRepeat(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           30000,
		CurrentlyPlayingType: "episode",
		Episode:              &domain.Episode{ID: "ep-1", Name: "Episode", DurationMs: 1800000},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)

	tests := []struct {
		name string
		key  string
		want PlaybackAction
	}{
		{"toggle shuffle", "s", ActionToggleShuffle},
		{"cycle repeat", "r", ActionCycleRepeat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			_, cmd := p.Update(msg)
			require.NotNil(t, cmd)
			result := cmd()
			req, ok := result.(PlaybackRequestMsg)
			require.True(t, ok, "got %T", result)
			assert.Equal(t, tt.want, req.Action)
		})
	}
}

func TestPodcastPlaybackPane_UpdateNotFocused(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           30000,
		CurrentlyPlayingType: "episode",
		Episode:              &domain.Episode{ID: "ep-1", Name: "Episode", DurationMs: 1800000},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, false)

	msg := tea.KeyMsg{Type: tea.KeySpace}
	_, cmd := p.Update(msg)
	assert.Nil(t, cmd, "unfocused pane should not handle keys")
}

func TestPodcastPlaybackPane_PlaybackStateFetchedMsg(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)
	p.SetSize(80, 24)

	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           120000,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:         "ep-1",
			Name:       "Updated Episode",
			DurationMs: 1800000,
			ReleaseDate: "2024-01-15",
		},
		Device: &domain.Device{ID: "dev-1", VolumePercent: 50},
	})
	_, cmd := p.Update(PlaybackStateFetchedMsg{})
	assert.Nil(t, cmd, "PlaybackStateFetchedMsg should not produce a command")

	assert.Equal(t, 120000, p.localProgressMs, "should update local progress")
	output := p.View()
	assert.Contains(t, output, "Updated Episode")
}

func TestPodcastPlaybackPane_SetSize(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)

	p.SetSize(100, 30)
	assert.Equal(t, 100, p.width)
	assert.Equal(t, 30, p.height)
	assert.Equal(t, 30, p.infoWidth)
	assert.Equal(t, 69, p.detailsWidth)

	// Minimum infoWidth 24
	p.SetSize(30, 10)
	assert.Equal(t, 24, p.infoWidth)
	assert.Equal(t, 5, p.detailsWidth)

	// Smallest possible
	p.SetSize(5, 10)
	assert.Equal(t, 24, p.infoWidth, "min 24 even when pane is narrow")
}

func TestPodcastPlaybackPane_ProgressBarFormat(t *testing.T) {
	bar := renderProgressBar(0, 100, 10)
	assert.Equal(t, strings.Repeat("\u2588", 0)+strings.Repeat("\u2591", 10), bar,
		"0% progress should be all empty")

	bar = renderProgressBar(100, 100, 10)
	assert.Equal(t, strings.Repeat("\u2588", 10)+strings.Repeat("\u2591", 0), bar,
		"100% progress should be all full")

	bar = renderProgressBar(50, 100, 10)
	assert.Equal(t, strings.Repeat("\u2588", 5)+strings.Repeat("\u2591", 5), bar,
		"50% progress should be half full")

	// Zero duration
	bar = renderProgressBar(50, 0, 10)
	assert.Equal(t, strings.Repeat("\u2588", 0)+strings.Repeat("\u2591", 10), bar,
		"zero duration should produce empty bar")
}

func TestPodcastPlaybackPane_TruncateStr(t *testing.T) {
	assert.Equal(t, "hello", truncateStr("hello", 10), "no truncation needed")
	assert.Equal(t, "h\u2026", truncateStr("hello", 2), "truncation with ellipsis")
	assert.Equal(t, "", truncateStr("hello", 0), "zero width should be empty")
	assert.Equal(t, "\u2026", truncateStr("hello", 1), "one char width should be ellipsis")
}

func TestPodcastPlaybackPane_EpisodeViewWithoutShow(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		ProgressMs:           60000,
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:          "ep-1",
			Name:        "Test Episode",
			Description: "Description text",
			DurationMs:  1800000,
			ReleaseDate: "2024-01-15",
			Show:        nil,
		},
	})
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, true)
	p.SetSize(80, 24)
	output := p.View()

	assert.NotContains(t, output, "Publisher:", "no publisher when show is nil")
	assert.Contains(t, output, "Released:", "should still show release date")
}

func TestPodcastPlaybackPane_KeyNotFocused(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	p := NewPodcastPlaybackPane(s, th, false)

	msg := tea.KeyMsg{Type: tea.KeySpace}
	_, cmd := p.Update(msg)
	assert.Nil(t, cmd, "should ignore keys when not focused")
}
