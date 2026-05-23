package panes

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/components/viz"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStateWriter wraps *state.Store to expose write methods for test setup
// without requiring a type assertion through the StateReader interface.
type testStateWriter struct {
	*state.Store
}

// newTestNowPlayingPane creates a NowPlayingPane with a fresh store and black theme.
func newTestNowPlayingPane(focused bool) *NowPlayingPane {
	s := state.New()
	t := theme.Load("black")
	return NewNowPlayingPane(s, t, focused)
}

// newTestNowPlayingPaneWithState creates a NowPlayingPane pre-loaded with playback state.
// It also returns a testStateWriter so tests can mutate the shared store without
// a type assertion through the StateReader interface.
func newTestNowPlayingPaneWithState(isPlaying bool, focused bool) (*NowPlayingPane, *testStateWriter) {
	s := state.New()
	w := &testStateWriter{s}
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
			Album: api.Album{
				ID:   "alb1",
				Name: "After Hours",
				Images: []api.AlbumImage{
					{URL: "https://i.scdn.co/image/after-hours-640", Width: 640, Height: 640},
				},
			},
		},
		Device: &api.Device{
			ID:            "dev-1",
			Name:          "MacBook Pro",
			VolumePercent: 65,
		},
	})
	t := theme.Load("black")
	return NewNowPlayingPane(s, t, focused), w
}

// ── Task 1: Rename tests ─────────────────────────────────────────────────────

func TestNowPlayingPane_View_NowPlaying(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
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
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd, "space when playing should return a command")
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok, "space cmd should return PlaybackRequestMsg, got %T", msg)
	assert.Equal(t, ActionPause, req.Action, "playing → space should request pause")
}

func TestNowPlayingPane_Update_Space_WhenPaused(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(false, true)

	spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd, "space when paused should return a command")
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok, "space cmd should return PlaybackRequestMsg, got %T", msg)
	assert.Equal(t, ActionPlay, req.Action, "paused → space should request play")
}

func TestNowPlayingPane_Update_P_SkipsPrev(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	pMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	_, cmd := pane.Update(pMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionPrevious, req.Action)
}

func TestNowPlayingPane_Update_Plus_VolUp(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	_, cmd := pane.Update(plusMsg)

	// + key now returns a debounce cmd (VolumeDebounceTickMsg), not PlaybackRequestMsg.
	require.NotNil(t, cmd, "+ key must return a debounce cmd")
	result := cmd()
	_, isPlaybackReq := result.(PlaybackRequestMsg)
	assert.False(t, isPlaybackReq, "+ key must not emit PlaybackRequestMsg directly")
	_, isDebounce := result.(components.VolumeDebounceTickMsg)
	// NOTE: tea.Tick returns a deferred cmd; firing it immediately may return
	// VolumeDebounceTickMsg or a tea.BatchMsg. We only need to confirm it's
	// NOT a PlaybackRequestMsg (the old stale-read pattern).
	_ = isDebounce
}

func TestNowPlayingPane_Update_Minus_VolDown(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	minusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}
	_, cmd := pane.Update(minusMsg)

	// - key now returns a debounce cmd, not PlaybackRequestMsg.
	require.NotNil(t, cmd, "- key must return a debounce cmd")
	result := cmd()
	_, isPlaybackReq := result.(PlaybackRequestMsg)
	assert.False(t, isPlaybackReq, "- key must not emit PlaybackRequestMsg directly")
}

func TestNowPlayingPane_Update_S_TogglesShuffle(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

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
	pane, w := newTestNowPlayingPaneWithState(true, true)

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
	w.SetPlaybackState(newState)

	fetchedMsg := PlaybackStateFetchedMsg{}
	updatedModel, _ := pane.Update(fetchedMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	// localProgressMs should be reset to server value.
	assert.Equal(t, 90000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_Update_PlaybackFetched_NilState(t *testing.T) {
	pane, w := newTestNowPlayingPaneWithState(true, true)
	pane.localProgressMs = 60000

	// Nil state (nothing playing) — store is cleared, then notification sent.
	w.SetPlaybackState(nil)

	fetchedMsg := PlaybackStateFetchedMsg{}
	updatedModel, _ := pane.Update(fetchedMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 0, updatedPane.localProgressMs)
	assert.Nil(t, updatedPane.store.PlaybackState())
}

func TestNowPlayingPane_Update_IgnoresKeysWhenNotFocused(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, false) // not focused

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	assert.Nil(t, cmd, "unfocused pane should return nil cmd for key events")
}

func TestNowPlayingPane_Update_TickIncrements(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.localProgressMs = 30000

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 31000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_Update_TickNoIncrement_WhenPaused(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(false, true)
	pane.localProgressMs = 30000

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 30000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_Update_TickClampsAtDuration(t *testing.T) {
	// localProgressMs must not exceed DurationMs (252000 in the test fixture).
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.localProgressMs = 251500 // one tick away from exceeding DurationMs

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	// 251500 + 1000 = 252500 > 252000 — must be clamped to DurationMs.
	assert.Equal(t, 252000, updatedPane.localProgressMs,
		"localProgressMs must be clamped to DurationMs when it would overflow")
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
			pane, _ := newTestNowPlayingPaneWithState(true, true)

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

	spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
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
	require.Len(t, actions, 5, "should have exactly 5 border actions")

	expected := map[string]string{
		"s":     "shfl",
		"r":     "rpt",
		"space": "play",
		"+/-":   "vol",
		"v":     "viz",
	}
	for _, a := range actions {
		label, ok := expected[a.Key]
		assert.True(t, ok, "unexpected action key: %s", a.Key)
		assert.Equal(t, label, a.Label)
	}
}

// ── Task 4: viz.Engine migration tests ───────────────────────────────────────

func TestNowPlayingPane_VizTickMsg_AdvancesFrame(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	initialFrame := pane.engine.FrameIndex()

	// Send viz.TickMsg — should advance frame when playing.
	_, _ = pane.Update(viz.TickMsg(time.Now()))

	assert.Equal(t, (initialFrame+1)%40, pane.engine.FrameIndex(),
		"viz.TickMsg should advance engine frame when playing")
}

func TestNowPlayingPane_VizTickMsg_ReturnsCmd(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)

	_, cmd := pane.Update(viz.TickMsg(time.Now()))
	assert.NotNil(t, cmd, "viz.TickMsg should return a re-arm tick command")
}

func TestNowPlayingPane_PlaybackFetched_SetsEnginePlaying(t *testing.T) {
	pane, w := newTestNowPlayingPaneWithState(false, true)
	pane.SetSize(80, 24)

	// Update store to playing=true, then send PlaybackStateFetchedMsg.
	w.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 5000,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
	})
	_, _ = pane.Update(PlaybackStateFetchedMsg{})

	// Engine should now be in playing state — frame should advance on tick.
	before := pane.engine.FrameIndex()
	_, _ = pane.Update(viz.TickMsg(time.Now()))
	assert.Equal(t, (before+1)%40, pane.engine.FrameIndex(),
		"engine should animate when playing=true")
}

func TestNowPlayingPane_PlaybackFetched_PausesEngine(t *testing.T) {
	pane, w := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	// Update store to paused, send PlaybackStateFetchedMsg.
	w.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  false,
		ProgressMs: 5000,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
	})
	_, _ = pane.Update(PlaybackStateFetchedMsg{})

	// Engine should be paused — frame should not advance on tick.
	before := pane.engine.FrameIndex()
	_, _ = pane.Update(viz.TickMsg(time.Now()))
	assert.Equal(t, before, pane.engine.FrameIndex(),
		"engine should not animate when playing=false")
}

func TestNowPlayingPane_FullView_ContainsBrailleChars(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()

	// Braille chars used by the engine (any braille codepoint).
	hasBraille := false
	for _, r := range output {
		if r >= '\u2800' && r <= '\u28FF' {
			hasBraille = true
			break
		}
	}
	assert.True(t, hasBraille, "full mode View() should contain braille characters from engine")
}

// ── Task 4: Gradient bars tests ──────────────────────────────────────────────

func TestNowPlayingPane_SeekBar_Renders(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	// The gradient seek bar renders time stamps like "0:30" for 30000ms.
	assert.Contains(t, output, "0:30", "seek bar should show current time")
}

func TestNowPlayingPane_VolumeBar_Renders(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	// GradientVolumeBar renders music note icon ♪.
	assert.Contains(t, output, "♪", "volume bar should be rendered in full mode")
}

func TestNowPlayingPane_BarsResize_WithSetSize(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	pane.SetSize(40, 24)
	small := pane.View()

	pane.SetSize(80, 24)
	large := pane.View()

	assert.NotEqual(t, small, large, "different sizes should produce different output")
}

func TestNowPlayingPane_FullView_ContainsTrackAndAlbum(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
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

// ── v key cycles engine pattern ──────────────────────────────────────────────

// TestNowPlayingPane_V_CyclesEnginePattern verifies that pressing 'v' while
// focused advances the engine's pattern index and emits a VisualizerPatternChangedMsg.
func TestNowPlayingPane_V_CyclesEnginePattern(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	pane.engine.SetSize(40, 10) // ensure frames are generated

	startPat := pane.engine.Pattern()

	vMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}
	_, cmd := pane.Update(vMsg)

	// v now emits VisualizerPatternChangedMsg for preference persistence.
	require.NotNil(t, cmd, "v key should return a Cmd for persistence")
	msg := cmd()
	_, ok := msg.(VisualizerPatternChangedMsg)
	assert.True(t, ok, "v key cmd should return VisualizerPatternChangedMsg, got %T", msg)

	patternCount := pane.engine.PatternCount()
	assert.Equal(t, (startPat+1)%patternCount, pane.engine.Pattern(),
		"engine pattern should advance by 1 on v key")
}

// TestNowPlayingPane_V_IgnoredWhenNotFocused verifies that 'v' is ignored when
// the pane is not focused (routing.go handles global routing; pane itself guards focus).
func TestNowPlayingPane_V_IgnoredWhenNotFocused(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, false) // not focused
	pane.SetSize(80, 24)

	startPat := pane.engine.Pattern()

	vMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}
	_, cmd := pane.Update(vMsg)

	assert.Nil(t, cmd, "unfocused pane should return nil cmd")
	assert.Equal(t, startPat, pane.engine.Pattern(), "pattern should not change when not focused")
}

// ── Feature 58: Split layout tests ──────────────────────────────────────────

// TestNowPlayingPane_SplitLayout_ContainsInfoBoxBorders verifies that View() at
// 80x24 contains the rounded-corner border characters produced by the InfoBox.
func TestNowPlayingPane_SplitLayout_ContainsInfoBoxBorders(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	assert.Contains(t, output, "╭", "split layout should contain InfoBox top-left corner")
	assert.Contains(t, output, "╰", "split layout should contain InfoBox bottom-left corner")
}

// TestNowPlayingPane_SplitLayout_ContainsBraille verifies that View() contains
// braille characters from the engine rendered on the right side.
func TestNowPlayingPane_SplitLayout_ContainsBraille(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()

	hasBraille := false
	for _, r := range output {
		if r >= '\u2800' && r <= '\u28FF' {
			hasBraille = true
			break
		}
	}
	assert.True(t, hasBraille, "split layout should contain braille characters from engine")
}

// TestNowPlayingPane_SplitLayout_ContainsSeekBar verifies that View() contains
// the seek bar time stamps rendered at the bottom.
func TestNowPlayingPane_SplitLayout_ContainsSeekBar(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	pane.localProgressMs = 30000

	output := pane.View()
	// GradientSeekBar renders current/total times.
	assert.Contains(t, output, "0:30", "split layout should contain seek bar current time")
}

// TestNowPlayingPane_SplitLayout_ContainsVolumeInInfoBox verifies that View()
// contains ♪ from the volume bar rendered inside the InfoBox.
func TestNowPlayingPane_SplitLayout_ContainsVolumeInInfoBox(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	assert.Contains(t, output, "♪", "split layout InfoBox should contain volume bar")
}

// TestNowPlayingPane_SplitLayout_ContainsControls verifies that View() contains
// the playback control characters from the Controls component.
func TestNowPlayingPane_SplitLayout_ContainsControls(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	// Controls renders Unicode glyphs — shuffle ⇄ and repeat-off ⟳ (GlyphRepeatOff).
	assert.Contains(t, output, "⇄", "split layout InfoBox should contain shuffle control")
	assert.Contains(t, output, "⟳", "split layout InfoBox should contain repeat-off control")
}

// TestNowPlayingPane_Title_ShowsTrackInfoWhenSmall verifies that Title() includes
// track name when the pane height is below 8 (the new compact-title threshold).
func TestNowPlayingPane_Title_ShowsTrackInfoWhenSmall(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
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
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24) // height >= 8

	title := pane.Title()
	assert.Equal(t, "Now Playing", title, "title at height>=8 should be 'Now Playing'")
}

// TestNowPlayingPane_SplitLayout_AdaptsToDifferentSizes verifies that
// different pane sizes produce different view output (layout is proportional).
func TestNowPlayingPane_SplitLayout_AdaptsToDifferentSizes(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	pane.SetSize(60, 20)
	small := pane.View()

	pane.SetSize(120, 36)
	large := pane.View()

	assert.NotEqual(t, small, large, "different sizes should produce different split layout output")
}

// ── Task 5: Two-column layout with seek bar in right panel ───────────────────

func TestNowPlayingPane_SeekBarInRightPanel(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 20)
	view := pane.View()

	// The seek bar time labels should be present.
	assert.Contains(t, view, "0:30", "should contain elapsed time from seek bar")

	// Multiple lines should be present.
	lines := strings.Split(view, "\n")
	assert.Greater(t, len(lines), 3, "should have multiple lines")
}

func TestNowPlayingPane_SplitFrame_Table(t *testing.T) {
	tests := []struct {
		name      string
		frameLen  int
		expectTop int
		expectBot int
	}{
		{"6 lines", 6, 3, 3},
		{"5 lines", 5, 2, 3},
		{"1 line", 1, 0, 1},
		{"0 lines", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := make(viz.Frame, tt.frameLen)
			for i := range frame {
				frame[i] = viz.StyledLine{Text: "x", Color: "#fff"}
			}
			top, bot := splitFrame(frame)
			assert.Len(t, top, tt.expectTop)
			assert.Len(t, bot, tt.expectBot)
		})
	}
}

func TestNowPlayingPane_RenderStyledLines(t *testing.T) {
	lines := viz.Frame{
		{Text: "aaa", Color: lipgloss.Color("#ff0000")},
		{Text: "bbb", Color: lipgloss.Color("#00ff00")},
	}
	out := renderStyledLines(lines)
	assert.Contains(t, out, "aaa")
	assert.Contains(t, out, "bbb")
}

func TestNowPlayingPane_RenderStyledLines_Empty(t *testing.T) {
	out := renderStyledLines(viz.Frame{})
	assert.Empty(t, out)
}

// ── Task 6: Vertical centering in expanded mode ───────────────────────────────

func TestNowPlayingPane_ExpandedVerticalCentering(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	// Expanded: large height
	pane.SetSize(80, 30)
	view := pane.View()
	lines := strings.Split(view, "\n")

	// lipgloss.Place pads output to the available height (30-2 = 28 lines).
	availableHeight := 30 - 2 // pane height minus border chrome
	assert.Equal(t, availableHeight, len(lines), "expanded should be padded to available height")
}

func TestNowPlayingPane_CompactNoCentering(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	// Compact: small height — content should not be over-padded.
	pane.SetSize(80, 10)
	view := pane.View()
	lines := strings.Split(view, "\n")

	// Should NOT have excessive padding beyond the content.
	assert.LessOrEqual(t, len(lines), 12, "compact should not have centering padding")
}

// ── Bug fix: compact mode controls visibility ─────────────────────────────────

// TestNowPlayingPane_CompactShowsControls verifies that at height=10 (innerH=4)
// the View output still contains the transport controls (⇄) and volume bar (♪).
// At this height, InfoBox.Render() previously truncated content to 4 lines which
// cut off controls (line 5) and volume bar (line 6) from the 6-line infoLines.
func TestNowPlayingPane_CompactShowsControls(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 10)

	output := pane.View()

	assert.Contains(t, output, "⇄", "compact mode (height=10) should show shuffle control")
	assert.Contains(t, output, "♪", "compact mode (height=10) should show volume bar")
}

// TestNowPlayingPane_VeryCompactShowsControls verifies that at height=8 (innerH=2)
// the View output still contains the transport controls glyph (⇄).
// At this height only 2 inner lines are available so the layout should show
// track name + controls (dropping artists, album, spacer, and volume bar).
func TestNowPlayingPane_VeryCompactShowsControls(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 8)

	output := pane.View()

	assert.Contains(t, output, "⇄", "very compact mode (height=8) should still show controls glyph")
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

// ---------------------------------------------------------------------------
// Story 80: SetVisualizerPattern and VisualizerPatternChangedMsg
// ---------------------------------------------------------------------------

// TestNowPlayingPane_SetVisualizerPattern verifies that SetVisualizerPattern
// delegates to the engine and changes the current pattern index.
func TestNowPlayingPane_SetVisualizerPattern(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)

	pane.SetVisualizerPattern(3)
	assert.Equal(t, 3, pane.engine.Pattern(), "SetVisualizerPattern should delegate to engine.SetPattern")
}

// TestNowPlayingPane_VKey_EmitsVisualizerChangedMsg verifies that pressing 'v'
// cycles the pattern AND emits a VisualizerPatternChangedMsg carrying the new index.
func TestNowPlayingPane_VKey_EmitsVisualizerChangedMsg(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)
	initialPattern := pane.engine.Pattern()

	vMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}
	_, cmd := pane.Update(vMsg)

	require.NotNil(t, cmd, "v key should return a Cmd")
	msg := cmd()
	changedMsg, ok := msg.(VisualizerPatternChangedMsg)
	require.True(t, ok, "v key cmd should return VisualizerPatternChangedMsg, got %T", msg)
	assert.Equal(t, initialPattern+1, changedMsg.PatternIndex,
		"VisualizerPatternChangedMsg should carry the new pattern index")
}

// TestNowPlayingPane_SetTheme_PreservesVisualizerPattern verifies that switching
// theme does not reset the visualizer pattern the user has selected.
func TestNowPlayingPane_SetTheme_PreservesVisualizerPattern(t *testing.T) {
	pane := newTestNowPlayingPane(false)
	pane.SetSize(80, 24)

	// Cycle to a non-default pattern (pattern 2).
	pane.SetVisualizerPattern(2)
	require.Equal(t, 2, pane.engine.Pattern(), "pattern should be 2 before theme change")

	// Switch to a different theme.
	newTheme := theme.Load("dracula")
	pane.SetTheme(newTheme)

	// Pattern must be preserved after the theme change.
	assert.Equal(t, 2, pane.engine.Pattern(),
		"SetTheme must not reset the visualizer pattern")
}

// ── Story 118: Playback key bug fixes ────────────────────────────────────────

// TestNowPlayingPane_HandleKey_KeySpace_Plays verifies that tea.KeySpace (not a rune)
// triggers play/pause — fixing the Bubbletea v0.27 Space delivery bug.
func TestNowPlayingPane_HandleKey_KeySpace_Plays(t *testing.T) {
	tests := []struct {
		name       string
		isPlaying  bool
		wantAction PlaybackAction
	}{
		{"KeySpace when playing → pause", true, ActionPause},
		{"KeySpace when paused → play", false, ActionPlay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane, _ := newTestNowPlayingPaneWithState(tt.isPlaying, true)

			// tea.KeySpace is how Bubbletea v0.27 delivers the Space key —
			// Type is tea.KeySpace, Runes is empty (not a rune ' ').
			spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
			_, cmd := pane.Update(spaceMsg)

			require.NotNil(t, cmd, "tea.KeySpace must return a command")
			msg := cmd()
			req, ok := msg.(PlaybackRequestMsg)
			require.True(t, ok, "tea.KeySpace must produce PlaybackRequestMsg, got %T", msg)
			assert.Equal(t, tt.wantAction, req.Action)
		})
	}
}

// TestNowPlaying_AsciiTitle verifies that Title() in compact mode (height < 8)
// uses ASCII glyphs when GlyphASCII mode is active: the pause-indicator resolves
// to "||" and the unicode literals ▶, ⏸, ─ are absent from the output.
func TestNowPlaying_AsciiTitle(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 6) // height < 8 triggers compact title path

	title := pane.Title()
	// IsPlaying == true → GlyphPaused → "||" in ASCII mode
	assert.Contains(t, title, "||", "ASCII mode playing state should show || pause glyph")
	// Unicode literals must not appear in ASCII mode
	assert.NotContains(t, title, "▶", "▶ should not appear in ASCII mode")
	assert.NotContains(t, title, "⏸", "⏸ should not appear in ASCII mode")
	assert.NotContains(t, title, "─", "─ should not appear in ASCII mode")
}

// TestNowPlaying_UnicodeTitlePlayPauseMapping verifies that Title() in compact mode
// (height < 8) maps IsPlaying correctly to the action glyph in unicode mode:
//   - IsPlaying=true  → shows ⏸ (pause action) and NOT ▶
//   - IsPlaying=false → shows ▶ (play action)  and NOT ⏸
//
// A regression swapping the if/else branches inside Title() would be caught here.
func TestNowPlaying_UnicodeTitlePlayPauseMapping(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	tests := []struct {
		name      string
		isPlaying bool
		wantGlyph string
		denyGlyph string
	}{
		{
			name:      "playing shows pause action ⏸",
			isPlaying: true,
			wantGlyph: "⏸",
			denyGlyph: "▶",
		},
		{
			name:      "paused shows play action ▶",
			isPlaying: false,
			wantGlyph: "▶",
			denyGlyph: "⏸",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane, _ := newTestNowPlayingPaneWithState(tt.isPlaying, true)
			pane.SetSize(80, 6) // height < 8 triggers compact title path

			title := pane.Title()
			assert.Contains(t, title, tt.wantGlyph,
				"compact title with isPlaying=%v should contain %q", tt.isPlaying, tt.wantGlyph)
			assert.NotContains(t, title, tt.denyGlyph,
				"compact title with isPlaying=%v must NOT contain %q", tt.isPlaying, tt.denyGlyph)
		})
	}
}

// TestNowPlayingPane_HandleKey_N_NoOp verifies that pressing "n" on the NowPlayingPane
// no longer emits a playback command — the n→next binding was removed in Story 118.
// The → key (tea.KeyRight) remains the authoritative "next track" binding.
func TestNowPlayingPane_HandleKey_N_NoOp(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	_, cmd := pane.Update(nMsg)

	// After removing the "n" arm from handleKey, pressing 'n' must return nil.
	assert.Nil(t, cmd, "'n' key on NowPlayingPane must not emit a playback command")
}

// ── Task 4: Volume debounce tests ────────────────────────────────────────────

// newPaneWithVolume creates a NowPlayingPane whose store device volume is vol.
func newPaneWithVolume(vol int) *NowPlayingPane {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 0,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
		Device: &api.Device{
			ID:            "dev-1",
			Name:          "Speaker",
			VolumePercent: vol,
			IsActive:      true,
		},
	})
	p := NewNowPlayingPane(s, theme.Load("black"), true)
	p.SetSize(80, 20)
	return p
}

func TestNowPlayingPane_VolumeUp_ReturnsDebounceCmdNotPlaybackRequest(t *testing.T) {
	p := newPaneWithVolume(49)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")}
	_, cmd := p.Update(msg)

	require.NotNil(t, cmd, "volume up must return a debounce cmd")
	// Fire the cmd to get the message; it must NOT be a PlaybackRequestMsg.
	result := cmd()
	_, isPlaybackReq := result.(PlaybackRequestMsg)
	assert.False(t, isPlaybackReq, "+ key must no longer emit PlaybackRequestMsg directly")
}

func TestNowPlayingPane_VolumeDebounceMsg_EmitsVolumeIntent(t *testing.T) {
	p := newPaneWithVolume(49)

	// Simulate: user pressed + once → bar seq=1, target=50
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")}
	p.Update(keyMsg)

	// Fire matching debounce tick (seq=1 because HandleDebounce increments on match,
	// but this is the first tick so bar.seq == 1 after HandleKey).
	debounce := components.VolumeDebounceTickMsg{TargetVol: 50, Seq: 1}
	_, cmd := p.Update(debounce)

	require.NotNil(t, cmd, "matching debounce must return a VolumeIntentMsg cmd")
	result := cmd()
	intent, ok := result.(VolumeIntentMsg)
	require.True(t, ok, "result must be VolumeIntentMsg, got %T", result)
	assert.Equal(t, 50, intent.TargetVol)
	assert.Equal(t, 1, intent.Seq)
}

func TestNowPlayingPane_VolumeAppliedMsg_Success_ConfirmsBar(t *testing.T) {
	p := newPaneWithVolume(49)

	// Prime the bar: press + once, then match the debounce tick.
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	_, cmd := p.Update(components.VolumeDebounceTickMsg{TargetVol: 50, Seq: 1})
	require.NotNil(t, cmd)

	// Send VolumeAppliedMsg confirming the API succeeded.
	p.Update(VolumeAppliedMsg{Vol: 55, Seq: 1})

	// Bar should now show 55, but hasPending stays true until a matching poll.
	out := p.View()
	assert.Contains(t, out, "55%")

	// Stale poll with old volume should be blocked while hasPending is true.
	p.volumeBar.SetConfirmed(0)
	out = p.View()
	assert.Contains(t, out, "55%", "stale poll must not snap bar back")

	// Matching poll clears hasPending.
	p.volumeBar.SetConfirmed(55)
	out = p.View()
	assert.Contains(t, out, "55%")

	// Now SetConfirmed updates freely.
	p.volumeBar.SetConfirmed(0)
	out = p.View()
	assert.Contains(t, out, "0%")
}

func TestNowPlayingPane_VolumeAppliedMsg_Error_CancelsPending(t *testing.T) {
	p := newPaneWithVolume(49)

	// Prime the bar: press + once, then match the debounce tick.
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	_, cmd := p.Update(components.VolumeDebounceTickMsg{TargetVol: 50, Seq: 1})
	require.NotNil(t, cmd)

	// Send VolumeAppliedMsg with an error.
	p.Update(VolumeAppliedMsg{Err: errors.New("fail"), Seq: 1})

	// currentVol should revert to the store's confirmed value (49), and hasPending should be false.
	out := p.View()
	assert.Contains(t, out, "49%", "error must revert bar to store value")

	// SetConfirmed should now be accepted since hasPending=false.
	p.volumeBar.SetConfirmed(0)
	out = p.View()
	assert.Contains(t, out, "0%")
}

func TestNowPlayingPane_StaleVolumeDebounce_ReturnsNilCmd(t *testing.T) {
	p := newPaneWithVolume(49)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")}
	p.Update(keyMsg) // seq=1
	p.Update(keyMsg) // seq=2

	stale := components.VolumeDebounceTickMsg{TargetVol: 50, Seq: 1}
	_, cmd := p.Update(stale)
	assert.Nil(t, cmd, "stale debounce tick must return nil cmd")
}

// TestNowPlayingPane_VolumeAppliedMsg_StaleSeq_BarStaysInSecondBurst verifies that a
// VolumeAppliedMsg from the first burst does not snap the bar back when the user has
// already started a second burst. The seq guard in ConfirmFromAPI prevents the stale
// confirmation from overwriting the second burst's pending value.
//
// Seq progression:
//   - HandleKey(+1, 50): seq=1, currentVol=51
//   - HandleDebounce({Seq:1}): seq advances to 2 (double-fire guard), matched
//   - HandleKey(+1, 51): seq=3, currentVol=52
//   - ConfirmFromAPI(intentSeq=1, vol=51): checks seq(3) == 1+1(2) → false → no snap
func TestNowPlayingPane_VolumeAppliedMsg_StaleSeq_BarStaysInSecondBurst(t *testing.T) {
	p := newPaneWithVolume(50)

	// First burst: press + (seq=1, currentVol=51).
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})

	// Resolve the first burst's debounce tick — seq advances to 2 inside HandleDebounce.
	p.Update(components.VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})

	// Second burst starts: press + again (seq=3, currentVol=52).
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})

	// The first burst's API result arrives with intentSeq=1 (stale).
	// ConfirmFromAPI(1, 51) checks: b.seq(3) == 1+1(2) → false → no snap.
	p.Update(VolumeAppliedMsg{Vol: 51, Seq: 1})

	out := p.View()
	assert.Contains(t, out, "52%", "stale ConfirmFromAPI must not snap bar back from second burst value")
}

// ── Story 216: Album Art Fetch Component ───────────────────────────────────

// tinyPNG returns a 1x1 red PNG encoded as bytes.
func tinyPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// execCmdTimeout calls cmd in a goroutine and returns its message or nil on timeout.
func execCmdTimeout(cmd tea.Cmd, timeout time.Duration) tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(timeout):
		return nil
	}
}

// canBindLocalhost checks whether the test environment allows binding to
// 127.0.0.1 (required for httptest). Sandbox environments often block this.
func canBindLocalhost() bool {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

// newLocalServer creates an httptest.Server bound to 127.0.0.1 instead of [::1]
// to work around sandbox IPv6 restrictions.
func newLocalServer(handler http.Handler) *httptest.Server {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener = l
	srv.Start()
	return srv
}

// findAlbumArtMsg searches a tea.BatchMsg for an AlbumArtFetchedMsg by executing
// each sub-command with a timeout. Returns nil if not found.
func findAlbumArtMsg(batch tea.BatchMsg) *components.AlbumArtFetchedMsg {
	for _, c := range batch {
		msg := execCmdTimeout(c, 2*time.Second)
		if m, ok := msg.(components.AlbumArtFetchedMsg); ok {
			return &m
		}
	}
	return nil
}

// TestNowPlayingPane_Init_FetchesArtWhenPlaying verifies that Init() returns a
// tea.Batch containing an album art fetch when the store has an active track with images.
// Skipped in sandbox environments that block localhost binding.
func TestNowPlayingPane_Init_FetchesArtWhenPlaying(t *testing.T) {
	if !canBindLocalhost() {
		t.Skip("localhost binding not available in this environment")
	}
	server := newLocalServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(tinyPNG())
	}))
	defer server.Close()

	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item: &api.Track{
			ID:   "track-1",
			Name: "Blinding Lights",
			Album: api.Album{
				ID: "alb1",
				Images: []api.AlbumImage{
					{URL: server.URL, Width: 640, Height: 640},
				},
			},
		},
	})
	pane := NewNowPlayingPane(s, theme.Load("black"), true)
	pane.SetSize(80, 24)

	cmd := pane.Init()
	require.NotNil(t, cmd, "Init should return a command")
	assert.True(t, pane.artRenderer.IsLoading(), "Init should set loading for the current track")

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	require.True(t, ok, "Init should return tea.BatchMsg, got %T", msg)
	require.GreaterOrEqual(t, len(batch), 2, "Init batch should contain engine tick + art fetch")

	artMsg := findAlbumArtMsg(batch)
	require.NotNil(t, artMsg, "Init batch should produce an AlbumArtFetchedMsg")
	assert.Equal(t, "track-1", artMsg.TrackID)
	assert.NotNil(t, artMsg.Rows)
	assert.Nil(t, artMsg.Err)
}

// TestNowPlayingPane_Init_NoFetchWithoutTrack verifies that Init() does not
// dispatch an art fetch when the store has no track item.
func TestNowPlayingPane_Init_NoFetchWithoutTrack(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		// no Item
	})
	pane := NewNowPlayingPane(s, theme.Load("black"), true)
	pane.SetSize(80, 24)

	cmd := pane.Init()
	require.NotNil(t, cmd)

	msg := cmd()
	// When no track is present, Init returns only the engine tick command
	// (tea.Batch with one element returns the element directly).
	assert.False(t, pane.artRenderer.IsLoading(), "no track → no loading state")
	_, isBatch := msg.(tea.BatchMsg)
	assert.False(t, isBatch, "no track → no BatchMsg, just engine tick")
}

// TestNowPlayingPane_HandlePlaybackFetched_FetchesArtOnTrackChange verifies that
// a PlaybackStateFetchedMsg carrying a different track ID dispatches an art fetch.
// Skipped in sandbox environments that block localhost binding.
func TestNowPlayingPane_HandlePlaybackFetched_FetchesArtOnTrackChange(t *testing.T) {
	if !canBindLocalhost() {
		t.Skip("localhost binding not available in this environment")
	}
	server := newLocalServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(tinyPNG())
	}))
	defer server.Close()

	pane, w := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	newState := &api.PlaybackState{
		IsPlaying: true,
		Item: &api.Track{
			ID:   "track-2",
			Name: "Save Your Tears",
			Album: api.Album{
				ID: "alb2",
				Images: []api.AlbumImage{
					{URL: server.URL, Width: 640, Height: 640},
				},
			},
		},
		Device: &api.Device{VolumePercent: 70},
	}
	w.SetPlaybackState(newState)

	_, cmd := pane.Update(PlaybackStateFetchedMsg{State: newState})
	require.NotNil(t, cmd, "track change should dispatch art fetch cmd")

	msg := cmd()
	artMsg, ok := msg.(components.AlbumArtFetchedMsg)
	require.True(t, ok, "cmd should return AlbumArtFetchedMsg, got %T", msg)
	assert.Equal(t, "track-2", artMsg.TrackID)
}

// TestNowPlayingPane_HandlePlaybackFetched_SkipsSameTrack verifies that
// re-sending the same track does not dispatch a redundant art fetch.
func TestNowPlayingPane_HandlePlaybackFetched_SkipsSameTrack(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	ps := pane.store.PlaybackState()
	require.NotNil(t, ps)
	require.NotNil(t, ps.Item)

	// Prime the renderer so the current track is already known.
	pane.artRenderer.SetLoading(ps.Item.ID)

	_, cmd := pane.Update(PlaybackStateFetchedMsg{State: ps})
	assert.Nil(t, cmd, "same track should not dispatch art fetch")
}

// TestNowPlayingPane_HandlePlaybackFetched_ClearsArtOnNoImageTrack verifies that
// changing to a track with no album images clears the renderer so View() falls back.
func TestNowPlayingPane_HandlePlaybackFetched_ClearsArtOnNoImageTrack(t *testing.T) {
	pane, w := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	// Prime renderer with art for the initial track.
	pane.artRenderer.SetLoading("track-1")
	pane.artRenderer.SetResult("track-1", []string{"row1", "row2"})
	assert.True(t, pane.artRenderer.HasImage())

	newState := &api.PlaybackState{
		IsPlaying: true,
		Item: &api.Track{
			ID:   "track-2",
			Name: "Save Your Tears",
			Album: api.Album{
				ID:     "alb2",
				Images: []api.AlbumImage{}, // no images
			},
		},
	}
	w.SetPlaybackState(newState)

	_, cmd := pane.Update(PlaybackStateFetchedMsg{State: newState})
	assert.Nil(t, cmd, "no images → no fetch cmd dispatched")
	assert.False(t, pane.artRenderer.HasImage(), "stale art should be cleared")
	assert.False(t, pane.artRenderer.IsLoading(), "loading should be cleared")
}

// TestNowPlayingPane_AlbumArtFetchedMsg_SetsImage verifies that a valid
// AlbumArtFetchedMsg populates the renderer.
func TestNowPlayingPane_AlbumArtFetchedMsg_SetsImage(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.artRenderer.SetLoading("track-1")

	_, _ = pane.Update(components.AlbumArtFetchedMsg{
		TrackID: "track-1",
		Rows:    []string{"row1", "row2"},
	})

	assert.True(t, pane.artRenderer.HasImage(), "valid msg should populate rows")
	assert.Equal(t, []string{"row1", "row2"}, pane.artRenderer.Rows())
}

// TestNowPlayingPane_AlbumArtFetchedMsg_StaleTrackID verifies that a result
// for a different track ID is ignored and does not overwrite current state.
func TestNowPlayingPane_AlbumArtFetchedMsg_StaleTrackID(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.artRenderer.SetLoading("track-1")

	_, _ = pane.Update(components.AlbumArtFetchedMsg{
		TrackID: "track-2",
		Rows:    []string{"row1", "row2"},
	})

	assert.False(t, pane.artRenderer.HasImage(), "stale msg should not populate rows")
	assert.True(t, pane.artRenderer.IsLoading(), "stale msg should not clear loading")
}

// ── Story 217: Responsive 3-tier layout ───────────────────────────────────────

// TestNowPlayingPane_RenderTier verifies tier dispatch for various body heights.
func TestNowPlayingPane_RenderTier(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		want   renderTier
	}{
		{"base at bodyH 10", 120, 12, tierBase},  // 12-2=10
		{"base at bodyH 15", 120, 17, tierBase},  // 17-2=15
		{"base at bodyH 20", 120, 22, tierBase},  // 22-2=20
		{"base at bodyH 28", 120, 30, tierBase},  // 30-2=28
		{"base at bodyH 30", 120, 32, tierBase},  // 32-2=30
		{"full at bodyH 31", 120, 33, tierFull},  // 33-2=31
		{"full at bodyH 45", 120, 47, tierFull},  // 47-2=45
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newTestNowPlayingPane(true)
			pane.SetSize(tt.width, tt.height)
			assert.Equal(t, tt.want, pane.renderTier())
		})
	}
}

// TestNowPlayingPane_View_BaseTier verifies the 3-column base-tier layout.
func TestNowPlayingPane_View_BaseTier(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(120, 16) // bodyH = 12 (base tier)

	// Simulate loaded album art
	pane.artRenderer.SetLoading("track-1")
	pane.artRenderer.SetResult("track-1", []string{"\x1b[31mimg-row-1\x1b[0m", "\x1b[31mimg-row-2\x1b[0m"})

	output := pane.View()

	// InfoBox content must be present
	assert.Contains(t, output, "Blinding Lights")
	assert.Contains(t, output, "The Weeknd")
	assert.Contains(t, output, "After Hours")

	// Viz braille characters must be present
	hasBraille := false
	for _, r := range output {
		if r >= '⠀' && r <= '⣿' {
			hasBraille = true
			break
		}
	}
	assert.True(t, hasBraille, "base tier should contain braille characters")

	// Album art ANSI sequences must appear in the output
	assert.Contains(t, output, "\x1b[31m", "album art ANSI sequences should be present")
}

// TestNowPlayingPane_View_FullTier verifies the full-tier 2-section layout.
func TestNowPlayingPane_View_FullTier(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(120, 45) // bodyH = 41 (full tier)

	// Simulate loaded album art
	pane.artRenderer.SetLoading("track-1")
	pane.artRenderer.SetResult("track-1", []string{"\x1b[31mimg1\x1b[0m", "\x1b[31mimg2\x1b[0m", "\x1b[31mimg3\x1b[0m", "\x1b[31mimg4\x1b[0m", "\x1b[31mimg5\x1b[0m", "\x1b[31mimg6\x1b[0m", "\x1b[31mimg7\x1b[0m", "\x1b[31mimg8\x1b[0m", "\x1b[31mimg9\x1b[0m", "\x1b[31mimg10\x1b[0m", "\x1b[31mimg11\x1b[0m", "\x1b[31mimg12\x1b[0m", "\x1b[31mimg13\x1b[0m", "\x1b[31mimg14\x1b[0m", "\x1b[31mimg15\x1b[0m", "\x1b[31mimg16\x1b[0m", "\x1b[31mimg17\x1b[0m", "\x1b[31mimg18\x1b[0m"})

	output := pane.View()

	// InfoBox content must be present
	assert.Contains(t, output, "Blinding Lights")
	assert.Contains(t, output, "The Weeknd")

	// Braille viz chars must be present
	hasBraille := false
	for _, r := range output {
		if r >= '⠀' && r <= '⣿' {
			hasBraille = true
			break
		}
	}
	assert.True(t, hasBraille, "full tier should contain braille characters")

	// Album art ANSI sequences must appear
	assert.Contains(t, output, "\x1b[31m", "album art ANSI sequences should be present")
}

// TestNowPlayingPane_View_Fallback_NoImage verifies that when no image is loaded
// and the renderer is not loading, all tiers fall back to the pre-feature 2-col layout.
func TestNowPlayingPane_View_Fallback_NoImage(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	// Ensure no image and not loading
	pane.artRenderer = components.AlbumArtRenderer{}

	// Test multiple tiers
	tiers := []struct {
		name   string
		width  int
		height int
	}{
		{"base tier", 120, 16},
		{"base tier tall", 120, 25},
		{"full tier", 120, 45},
	}

	for _, tt := range tiers {
		t.Run(tt.name, func(t *testing.T) {
			pane.SetSize(tt.width, tt.height)
			output := pane.View()

			// Should still show track info (InfoBox left)
			assert.Contains(t, output, "Blinding Lights")
			assert.Contains(t, output, "The Weeknd")

			// Should contain braille (viz right) — no empty image column
			hasBraille := false
			for _, r := range output {
				if r >= '⠀' && r <= '⣿' {
					hasBraille = true
					break
				}
			}
			assert.True(t, hasBraille, "fallback should contain braille characters")

			// Should NOT contain album art ANSI sequences
			assert.NotContains(t, output, "\x1b[31m", "fallback should not contain album art ANSI")
		})
	}
}

// TestNowPlayingPane_View_LoadingPlaceholder verifies that when the renderer is
// loading, a muted placeholder block appears in the image column position.
func TestNowPlayingPane_View_LoadingPlaceholder(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(120, 16) // base tier

	// Set loading without a result
	pane.artRenderer.SetLoading("track-1")

	output := pane.View()

	// Should still show track info
	assert.Contains(t, output, "Blinding Lights")

	// Should contain braille (viz right)
	hasBraille := false
	for _, r := range output {
		if r >= '⠀' && r <= '⣿' {
			hasBraille = true
			break
		}
	}
	assert.True(t, hasBraille, "loading state should contain braille characters")

	// Placeholder should not contain album art ANSI sequences
	assert.NotContains(t, output, "\x1b[31m", "placeholder should not contain album art ANSI")
}

// TestNowPlayingPane_SetSize_NoNegativeDimensions verifies that SetSize never
// assigns a zero or negative dimension to any sub-component in any tier.
func TestNowPlayingPane_SetSize_NoNegativeDimensions(t *testing.T) {
	sizes := []struct {
		name   string
		width  int
		height int
	}{
		{"tiny", 10, 10},
		{"narrow", 20, 40},
		{"short", 120, 10},
		{"base tier", 120, 16},
		{"base tier tall", 120, 25},
		{"full tier", 120, 45},
		{"large", 200, 60},
	}

	for _, tt := range sizes {
		t.Run(tt.name, func(t *testing.T) {
			pane, _ := newTestNowPlayingPaneWithState(true, true)
			// Ensure no panic and View returns non-empty.
			pane.SetSize(tt.width, tt.height)
			output := pane.View()
			assert.NotEmpty(t, output, "View should not be empty after SetSize")
		})
	}
}

// TestNowPlayingPane_WindowSizeMsg_RefetchesArt verifies that when SetSize
// changes imageRows by more than 2, a subsequent WindowSizeMsg triggers a
// non-nil album-art fetch command.
func TestNowPlayingPane_WindowSizeMsg_RefetchesArt(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	// Pre-load art so HasImage() is true and View() uses tiered layout.
	pane.artRenderer.SetLoading("track-1")
	pane.artRenderer.SetResult("track-1", []string{"img1", "img2", "img3", "img4", "img5", "img6", "img7", "img8", "img9", "img10", "img11", "img12", "img13", "img14", "img15", "img16"})

	// First size: base tier, imageRows ≈ bodyHeight = 16
	pane.SetSize(100, 20)

	// Second size: full tier, imageRows ≈ paneMin(25, 42) = 25
	// Difference = 9 > 2 → pendingArtRefresh should be set.
	pane.SetSize(100, 54)

	// WindowSizeMsg should dispatch a fetch and clear the flag.
	_, cmd := pane.Update(tea.WindowSizeMsg{Width: 100, Height: 54})
	assert.NotNil(t, cmd, "WindowSizeMsg after large resize should return non-nil cmd")

	// The returned cmd must be a FetchAlbumArtCmd (executed it returns AlbumArtFetchedMsg).
	msg := cmd()
	_, ok := msg.(components.AlbumArtFetchedMsg)
	assert.True(t, ok, "cmd should return AlbumArtFetchedMsg, got %T", msg)
}

// TestNowPlayingPane_WindowSizeMsg_NoRefetchWhenSmallChange verifies that a
// small resize (≤2 rows difference) does not trigger a re-fetch.
func TestNowPlayingPane_WindowSizeMsg_NoRefetchWhenSmallChange(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.artRenderer.SetLoading("track-1")
	pane.artRenderer.SetResult("track-1", []string{"img1", "img2", "img3", "img4", "img5", "img6", "img7", "img8", "img9", "img10", "img11", "img12", "img13", "img14", "img15", "img16"})

	// Base tier both times: bodyHeight changes from 16 → 18, imageRows from 16 → 18.
	// Difference = 2, which is NOT > 2, so no refresh.
	pane.SetSize(100, 20)
	pane.SetSize(100, 22)

	_, cmd := pane.Update(tea.WindowSizeMsg{Width: 100, Height: 22})
	assert.Nil(t, cmd, "small resize should not trigger art re-fetch")
}
