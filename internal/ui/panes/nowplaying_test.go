package panes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestNowPlayingPane creates a NowPlayingPane with a fresh store and black theme.
func newTestNowPlayingPane(focused bool) *NowPlayingPane {
	s := state.New()
	t := theme.Load("black")
	return NewNowPlayingPane(s, t, focused)
}

// newTestNowPlayingPaneWithState creates a NowPlayingPane pre-loaded with playback state.
func newTestNowPlayingPaneWithState(isPlaying bool, focused bool) *NowPlayingPane {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying:    isPlaying,
		ProgressMs:   30000,
		ShuffleState: false,
		RepeatState:  "off",
		Item: &api.Track{
			ID:         "track-1",
			Name:       "Blinding Lights",
			DurationMs: 252000,
			Artists:    []api.Artist{{ID: "a1", Name: "The Weeknd"}},
			Album:      api.Album{ID: "alb1", Name: "After Hours"},
		},
		Device: &api.Device{
			ID:            "dev-1",
			Name:          "MacBook Pro",
			VolumePercent: 65,
		},
	})
	t := theme.Load("black")
	return NewNowPlayingPane(s, t, focused)
}

// ── Task 1: Rename tests ─────────────────────────────────────────────────────

func TestNowPlayingPane_View_NowPlaying(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	output := pane.View()

	assert.Contains(t, output, "Blinding Lights", "should show track name")
	assert.Contains(t, output, "The Weeknd", "should show artist name")
	assert.Contains(t, output, "After Hours", "should show album name")
}

func TestNowPlayingPane_View_EmptyState(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)
	output := pane.View()

	assert.Contains(t, output, "Nothing playing", "should show empty state message")
}

func TestNowPlayingPane_Update_Space_WhenPlaying(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd, "space when playing should return a command")
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok, "space cmd should return PlaybackRequestMsg, got %T", msg)
	assert.Equal(t, ActionPause, req.Action, "playing → space should request pause")
}

func TestNowPlayingPane_Update_Space_WhenPaused(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(false, true)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd, "space when paused should return a command")
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok, "space cmd should return PlaybackRequestMsg, got %T", msg)
	assert.Equal(t, ActionPlay, req.Action, "paused → space should request play")
}

func TestNowPlayingPane_Update_N_SkipsNext(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	_, cmd := pane.Update(nMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionNext, req.Action)
}

func TestNowPlayingPane_Update_P_SkipsPrev(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	pMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	_, cmd := pane.Update(pMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionPrevious, req.Action)
}

func TestNowPlayingPane_Update_Plus_VolUp(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	_, cmd := pane.Update(plusMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionVolumeUp, req.Action)
}

func TestNowPlayingPane_Update_Minus_VolDown(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	minusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}
	_, cmd := pane.Update(minusMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionVolumeDown, req.Action)
}

func TestNowPlayingPane_Update_S_TogglesShuffle(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	_, cmd := pane.Update(sMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionToggleShuffle, req.Action)
}

func TestNowPlayingPane_Update_R_CyclesRepeat(t *testing.T) {
	tests := []struct {
		name        string
		startRepeat string
	}{
		{"off", "off"},
		{"context", "context"},
		{"track", "track"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := state.New()
			s.SetPlaybackState(&api.PlaybackState{
				IsPlaying:   true,
				RepeatState: tt.startRepeat,
				Item: &api.Track{
					ID:         "t1",
					Name:       "Track",
					DurationMs: 200000,
					Artists:    []api.Artist{{Name: "Artist"}},
				},
				Device: &api.Device{VolumePercent: 50},
			})
			th := theme.Load("black")
			pane := NewNowPlayingPane(s, th, true)

			rMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
			_, cmd := pane.Update(rMsg)

			require.NotNil(t, cmd)
			msg := cmd()
			req, ok := msg.(PlaybackRequestMsg)
			assert.True(t, ok)
			assert.Equal(t, ActionCycleRepeat, req.Action)
		})
	}
}

func TestNowPlayingPane_Update_PlaybackFetched(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	// Simulate app.go updating the store and sending PlaybackStateFetchedMsg.
	newState := &api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 90000,
		Item: &api.Track{
			ID:         "track-2",
			Name:       "Save Your Tears",
			DurationMs: 215000,
			Artists:    []api.Artist{{Name: "The Weeknd"}},
			Album:      api.Album{Name: "After Hours"},
		},
		Device: &api.Device{VolumePercent: 70},
	}
	pane.store.SetPlaybackState(newState)

	fetchedMsg := PlaybackStateFetchedMsg{}
	updatedModel, _ := pane.Update(fetchedMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	// localProgressMs should be reset to server value.
	assert.Equal(t, 90000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_Update_PlaybackFetched_NilState(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.localProgressMs = 60000

	// Nil state (nothing playing) — store is cleared, then notification sent.
	pane.store.SetPlaybackState(nil)

	fetchedMsg := PlaybackStateFetchedMsg{}
	updatedModel, _ := pane.Update(fetchedMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 0, updatedPane.localProgressMs)
	assert.Nil(t, updatedPane.store.PlaybackState())
}

func TestNowPlayingPane_Update_IgnoresKeysWhenNotFocused(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, false) // not focused

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	assert.Nil(t, cmd, "unfocused pane should return nil cmd for key events")
}

func TestNowPlayingPane_Update_TickIncrements(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.localProgressMs = 30000

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 31000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_Update_TickNoIncrement_WhenPaused(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(false, true)
	pane.localProgressMs = 30000

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 30000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_SetFocused(t *testing.T) {
	pane := newTestNowPlayingPane(false)
	assert.False(t, pane.focused)

	pane.SetFocused(true)
	assert.True(t, pane.focused)
}

func TestNowPlayingPane_IsFocused(t *testing.T) {
	pane := newTestNowPlayingPane(false)
	assert.False(t, pane.IsFocused())

	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
}

func TestNowPlayingPane_Init_ReturnsVisualizerCmd(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)
	cmd := pane.Init()
	// Init should return a command (visualizer tick).
	assert.NotNil(t, cmd, "Init should return visualizer tick command")
}

func TestDeviceName_NoDevice(t *testing.T) {
	s := state.New()
	name := DeviceName(s)
	assert.Empty(t, name)
}

func TestDeviceName_WithDevice(t *testing.T) {
	s := state.New()
	s.SetActiveDevice(&api.Device{Name: "MacBook Pro"})
	name := DeviceName(s)
	assert.Contains(t, name, "MacBook Pro")
}

// TestNowPlayingPane_ArrowKeys tests left/right arrow keys for navigation.
func TestNowPlayingPane_ArrowKeys(t *testing.T) {
	tests := []struct {
		name       string
		keyType    tea.KeyType
		wantAction PlaybackAction
	}{
		{"right arrow skips next", tea.KeyRight, ActionNext},
		{"left arrow skips prev", tea.KeyLeft, ActionPrevious},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newTestNowPlayingPaneWithState(true, true)

			keyMsg := tea.KeyMsg{Type: tt.keyType}
			_, cmd := pane.Update(keyMsg)

			require.NotNil(t, cmd)
			msg := cmd()
			req, ok := msg.(PlaybackRequestMsg)
			assert.True(t, ok)
			assert.Equal(t, tt.wantAction, req.Action)
		})
	}
}

// TestNowPlayingPane_Space_NilState tests that pressing space with no state still returns cmd.
func TestNowPlayingPane_Space_NilState(t *testing.T) {
	pane := newTestNowPlayingPane(true)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionPlay, req.Action, "nil state → space should request play")
}

// ── Task 2: layout.Pane interface tests ─────────────────────────────────────

// TestNowPlayingPane_ImplementsLayoutPane verifies compile-time interface satisfaction.
func TestNowPlayingPane_ImplementsLayoutPane(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	var _ layout.Pane = pane // compile-time check
	assert.NotNil(t, pane)
}

func TestNowPlayingPane_ID(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	assert.Equal(t, layout.PaneNowPlaying, pane.ID())
}

func TestNowPlayingPane_Title_DefaultIsNowPlaying(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24) // full mode
	assert.Equal(t, "Now Playing", pane.Title())
}

func TestNowPlayingPane_ToggleKey(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	assert.Equal(t, 1, pane.ToggleKey())
}

func TestNowPlayingPane_Actions(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	actions := pane.Actions()
	require.Len(t, actions, 2)
	assert.Equal(t, "s", actions[0].Key)
	assert.Equal(t, "shuffle", actions[0].Label)
	assert.Equal(t, "r", actions[1].Key)
	assert.Equal(t, "repeat", actions[1].Label)
}

// ── Task 3: Visualizer tests ─────────────────────────────────────────────────

func TestNowPlayingPane_VisualizerTickMsg_AdvancesFrame(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	// Set visualizer to playing.
	pane.visualizer.SetPlaying(true)
	initialFrame := pane.visualizer.FrameIndex()

	// Send VisualizerTickMsg.
	_, _ = pane.Update(components.VisualizerTickMsg{})

	// Frame should have advanced (visualizer was playing).
	assert.Equal(t, (initialFrame+1)%40, pane.visualizer.FrameIndex(),
		"VisualizerTickMsg should advance frame when playing")
}

func TestNowPlayingPane_PlaybackFetched_SetsVisualizerPlaying(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(false, true)
	pane.SetSize(80, 24)

	// Update store to playing=true, then send PlaybackStateFetchedMsg.
	pane.store.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 5000,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
	})
	_, _ = pane.Update(PlaybackStateFetchedMsg{})

	// Visualizer should now be in playing state.
	// Verify by sending a tick and checking frame advances.
	before := pane.visualizer.FrameIndex()
	_, _ = pane.Update(components.VisualizerTickMsg{})
	assert.Equal(t, (before+1)%40, pane.visualizer.FrameIndex(),
		"visualizer should animate when playing=true")
}

func TestNowPlayingPane_PlaybackFetched_PausesVisualizer(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	pane.visualizer.SetPlaying(true)

	// Update store to paused, send PlaybackStateFetchedMsg.
	pane.store.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  false,
		ProgressMs: 5000,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
	})
	_, _ = pane.Update(PlaybackStateFetchedMsg{})

	// Visualizer should be paused — frame should not advance on tick.
	before := pane.visualizer.FrameIndex()
	_, _ = pane.Update(components.VisualizerTickMsg{})
	assert.Equal(t, before, pane.visualizer.FrameIndex(),
		"visualizer should not animate when playing=false")
}

func TestNowPlayingPane_FullView_ContainsBrailleChars(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	pane.visualizer.SetPlaying(true)

	output := pane.View()

	// Braille chars used by the visualizer (any of the 5 codepoints).
	hasBraille := false
	for _, r := range output {
		if r >= '\u2800' && r <= '\u28FF' {
			hasBraille = true
			break
		}
	}
	assert.True(t, hasBraille, "full mode View() should contain braille characters from visualizer")
}

// ── Task 4: Gradient bars tests ──────────────────────────────────────────────

func TestNowPlayingPane_SeekBar_Renders(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	// The gradient seek bar renders time stamps like "0:30" for 30000ms.
	assert.Contains(t, output, "0:30", "seek bar should show current time")
}

func TestNowPlayingPane_VolumeBar_Renders(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	// GradientVolumeBar always renders "VOL" prefix.
	assert.Contains(t, output, "VOL", "volume bar should be rendered in full mode")
}

func TestNowPlayingPane_BarsResize_WithSetSize(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	pane.SetSize(40, 24)
	small := pane.View()

	pane.SetSize(80, 24)
	large := pane.View()

	assert.NotEqual(t, small, large, "different sizes should produce different output")
}

func TestNowPlayingPane_FullView_ContainsTrackAndAlbum(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	assert.Contains(t, output, "Blinding Lights")
	assert.Contains(t, output, "The Weeknd")
	assert.Contains(t, output, "After Hours")
}

// TestNowPlayingPane_ZeroDuration verifies seek bar handles zero duration gracefully.
func TestNowPlayingPane_ZeroDuration(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 0,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 0, // zero duration edge case
			Artists:    []api.Artist{{Name: "Artist"}},
		},
	})
	th := theme.Load("black")
	pane := NewNowPlayingPane(s, th, true)
	pane.SetSize(80, 24)

	// Should not panic.
	output := pane.View()
	assert.NotEmpty(t, output)
}

// TestNowPlayingPane_VisualizerTickMsg_ReturnsCmd verifies the tick returns a re-arm command.
func TestNowPlayingPane_VisualizerTickMsg_ReturnsCmd(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)

	_, cmd := pane.Update(components.VisualizerTickMsg{})
	assert.NotNil(t, cmd, "VisualizerTickMsg should return a re-arm tick command")
}

// ── Task 2: v key cycles visualizer pattern ──────────────────────────────────

// TestNowPlayingPane_V_CyclesVisualizerPattern verifies that pressing 'v' while
// focused advances the visualizer's pattern index, wrapping at NumPatterns.
func TestNowPlayingPane_V_CyclesVisualizerPattern(t *testing.T) {
	tests := []struct {
		name        string
		startPat    int
		wantPattern int
	}{
		{"pattern 0 → 1", 0, 1},
		{"pattern 1 → 2", 1, 2},
		{"pattern 2 wraps → 0", 2, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newTestNowPlayingPaneWithState(true, true)
			pane.SetSize(80, 24)
			pane.visualizer.SetPattern(tt.startPat)

			vMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}
			_, cmd := pane.Update(vMsg)

			assert.Nil(t, cmd, "v key should return nil cmd (local state change only)")
			assert.Equal(t, tt.wantPattern, pane.visualizer.Pattern(),
				"visualizer pattern should cycle from %d to %d", tt.startPat, tt.wantPattern)
		})
	}
}

// TestNowPlayingPane_V_IgnoredWhenNotFocused verifies that 'v' is ignored when
// the pane is not focused (routing.go handles global routing; pane itself guards focus).
func TestNowPlayingPane_V_IgnoredWhenNotFocused(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, false) // not focused
	pane.SetSize(80, 24)
	pane.visualizer.SetPattern(0)

	vMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}
	_, cmd := pane.Update(vMsg)

	assert.Nil(t, cmd, "unfocused pane should return nil cmd")
	assert.Equal(t, 0, pane.visualizer.Pattern(), "pattern should not change when not focused")
}

// ── Feature 58: Split layout tests ──────────────────────────────────────────

// TestNowPlayingPane_SplitLayout_ContainsInfoBoxBorders verifies that View() at
// 80x24 contains the rounded-corner border characters produced by the InfoBox.
func TestNowPlayingPane_SplitLayout_ContainsInfoBoxBorders(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	assert.Contains(t, output, "╭", "split layout should contain InfoBox top-left corner")
	assert.Contains(t, output, "╰", "split layout should contain InfoBox bottom-left corner")
}

// TestNowPlayingPane_SplitLayout_ContainsBraille verifies that View() contains
// braille characters from the visualizer rendered on the right side.
func TestNowPlayingPane_SplitLayout_ContainsBraille(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	pane.visualizer.SetPlaying(true)

	output := pane.View()

	hasBraille := false
	for _, r := range output {
		if r >= '\u2800' && r <= '\u28FF' {
			hasBraille = true
			break
		}
	}
	assert.True(t, hasBraille, "split layout should contain braille characters from visualizer")
}

// TestNowPlayingPane_SplitLayout_ContainsSeekBar verifies that View() contains
// the seek bar time stamps rendered at the bottom.
func TestNowPlayingPane_SplitLayout_ContainsSeekBar(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	pane.localProgressMs = 30000

	output := pane.View()
	// GradientSeekBar renders current/total times.
	assert.Contains(t, output, "0:30", "split layout should contain seek bar current time")
}

// TestNowPlayingPane_SplitLayout_ContainsVolumeInInfoBox verifies that View()
// contains "VOL" from the volume bar rendered inside the InfoBox.
func TestNowPlayingPane_SplitLayout_ContainsVolumeInInfoBox(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	assert.Contains(t, output, "VOL", "split layout InfoBox should contain volume bar")
}

// TestNowPlayingPane_SplitLayout_ContainsControls verifies that View() contains
// the playback control characters from the Controls component.
func TestNowPlayingPane_SplitLayout_ContainsControls(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	// Controls renders Unicode glyphs — shuffle ⇄ and repeat ↻.
	assert.Contains(t, output, "⇄", "split layout InfoBox should contain shuffle control")
	assert.Contains(t, output, "↻", "split layout InfoBox should contain repeat control")
}

// TestNowPlayingPane_Title_ShowsTrackInfoWhenSmall verifies that Title() includes
// track name when the pane height is below 8 (the new compact-title threshold).
func TestNowPlayingPane_Title_ShowsTrackInfoWhenSmall(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 6) // height < 8
	pane.localProgressMs = 30000

	title := pane.Title()
	assert.Contains(t, title, "Blinding Lights", "title at height<8 should include track name")
	assert.Contains(t, title, "The Weeknd", "title at height<8 should include artist name")
	assert.Contains(t, title, "0:30", "title at height<8 should include current time")
}

// TestNowPlayingPane_Title_DefaultWhenTall verifies that Title() returns the
// default "Now Playing" string when the pane height is >= 8.
func TestNowPlayingPane_Title_DefaultWhenTall(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24) // height >= 8

	title := pane.Title()
	assert.Equal(t, "Now Playing", title, "title at height>=8 should be 'Now Playing'")
}

// TestNowPlayingPane_SplitLayout_AdaptsToDifferentSizes verifies that
// different pane sizes produce different view output (layout is proportional).
func TestNowPlayingPane_SplitLayout_AdaptsToDifferentSizes(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	pane.SetSize(60, 20)
	small := pane.View()

	pane.SetSize(120, 36)
	large := pane.View()

	assert.NotEqual(t, small, large, "different sizes should produce different split layout output")
}

// splitLines splits a string by newline.
// Shared helper used by multiple pane test files in the same package.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
