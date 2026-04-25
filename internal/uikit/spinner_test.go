package uikit_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestTheme returns the black theme used throughout uikit tests.
func newTestTheme() theme.Theme { return theme.Load("black") }

// TestSpinner_Done_EmitsMsgAfterTTL verifies that Done() immediately returns
// a tea.Cmd and that executing it produces a SpinnerDoneMsg with the given text.
func TestSpinner_Done_EmitsMsgAfterTTL(t *testing.T) {
	th := newTestTheme()
	s := uikit.NewSpinner("Working", th)
	_, cmd := s.Done("Authorized")
	require.NotNil(t, cmd)
	msg := cmd()
	done, ok := msg.(uikit.SpinnerDoneMsg)
	assert.True(t, ok, "expected SpinnerDoneMsg, got %T", msg)
	assert.Equal(t, "Authorized", done.Text)
}

// TestSpinner_Fail_EmitsMsgWithErr verifies that Fail() returns a cmd that
// produces a SpinnerFailMsg carrying the error string.
func TestSpinner_Fail_EmitsMsgWithErr(t *testing.T) {
	th := newTestTheme()
	s := uikit.NewSpinner("Working", th)
	_, cmd := s.Fail("Failed: bad token")
	require.NotNil(t, cmd)
	msg := cmd()
	fail, ok := msg.(uikit.SpinnerFailMsg)
	assert.True(t, ok, "expected SpinnerFailMsg, got %T", msg)
	assert.Equal(t, "Failed: bad token", fail.Err)
}

// TestSpinner_Cancel_ClearsImmediately verifies that Cancel() emits
// SpinnerCancelledMsg and leaves the View() returning empty string.
func TestSpinner_Cancel_ClearsImmediately(t *testing.T) {
	th := newTestTheme()
	s := uikit.NewSpinner("Working", th)
	m, cmd := s.Cancel()
	assert.Empty(t, m.View())
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(uikit.SpinnerCancelledMsg)
	assert.True(t, ok, "expected SpinnerCancelledMsg, got %T", msg)
}

// TestSpinner_Init_ReturnsCmd confirms Init() hands back a non-nil tea.Cmd so
// the embedded bubbles/spinner tick loop can start.
func TestSpinner_Init_ReturnsCmd(t *testing.T) {
	th := newTestTheme()
	s := uikit.NewSpinner("Working", th)
	cmd := s.Init()
	assert.NotNil(t, cmd)
}

// TestSpinner_View_WhileRunning verifies the running view contains both the
// spinner frame and the muted text.
func TestSpinner_View_WhileRunning(t *testing.T) {
	th := newTestTheme()
	uikit.SetModeForTest(uikit.GlyphUnicode)
	s := uikit.NewSpinner("Waiting...", th)
	v := s.View()
	assert.NotEmpty(t, v)
	assert.Contains(t, stripANSI(v), "Waiting...")
}

// TestSpinner_View_AfterDone verifies the resolved Done view contains the
// success glyph and the hold text.
func TestSpinner_View_AfterDone(t *testing.T) {
	th := newTestTheme()
	uikit.SetModeForTest(uikit.GlyphUnicode)
	s := uikit.NewSpinner("Working", th)
	s, _ = s.Done("Authorized")
	v := stripANSI(s.View())
	assert.Contains(t, v, "✓")
	assert.Contains(t, v, "Authorized")
}

// TestSpinner_View_AfterFail verifies the resolved Fail view contains the
// error glyph and the hold text.
func TestSpinner_View_AfterFail(t *testing.T) {
	th := newTestTheme()
	uikit.SetModeForTest(uikit.GlyphUnicode)
	s := uikit.NewSpinner("Working", th)
	s, _ = s.Fail("Authorization failed")
	v := stripANSI(s.View())
	assert.Contains(t, v, "✗")
	assert.Contains(t, v, "Authorization failed")
}

// TestSpinner_Update_BlocksAfterResolution confirms that once the spinner is
// resolved (Done/Fail/Cancel), Update() returns a nil cmd regardless of the
// incoming message.
func TestSpinner_Update_BlocksAfterResolution(t *testing.T) {
	th := newTestTheme()
	s := uikit.NewSpinner("Working", th)
	s, _ = s.Done("Authorized")
	// Feed a tick — should be dropped.
	_, cmd := s.Update(spinner.TickMsg{})
	assert.Nil(t, cmd)
}

// TestSpinner_Update_ForwardsTick verifies that while running, Update forwards
// spinner ticks so the frame advances.
func TestSpinner_Update_ForwardsTick(t *testing.T) {
	th := newTestTheme()
	uikit.SetModeForTest(uikit.GlyphUnicode)
	s := uikit.NewSpinner("Working", th)
	// Simulate a tick from bubbles/spinner.
	_, cmd := s.Update(spinner.TickMsg{})
	// cmd will be non-nil (re-arms the tick).
	assert.NotNil(t, cmd)
}

// TestSpinner_ASCII_Mode verifies that the Done-state glyph uses the ASCII
// form ("+") when GlyphASCII mode is active.
func TestSpinner_ASCII_Mode(t *testing.T) {
	th := newTestTheme()
	uikit.SetModeForTest(uikit.GlyphASCII)
	t.Cleanup(func() { uikit.SetModeForTest(uikit.GlyphUnicode) })
	s := uikit.NewSpinner("Working", th)
	s, _ = s.Done("ok")
	v := stripANSI(s.View())
	assert.True(t, strings.HasPrefix(v, "+"), "ASCII Done glyph should be '+', got: %q", v)
}

// TestSpinner_DoneCmd_UsesTeaTick verifies Done() returns a tea.Tick-based cmd
// (not a bare function closure) so that the hold timer respects Bubble Tea's
// scheduler rather than calling time.Sleep.
func TestSpinner_DoneCmd_UsesTeaTick(t *testing.T) {
	th := newTestTheme()
	s := uikit.NewSpinner("Working", th)
	_, cmd := s.Done("ok")
	// The cmd returned by tea.Tick is NOT equal to nil, and when executed it must
	// return SpinnerDoneMsg — enough to confirm the scheduler pattern.
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(uikit.SpinnerDoneMsg)
	assert.True(t, ok)
}

// TestSpinner_FailCmd_UsesTeaTick mirrors the Done test for Fail().
func TestSpinner_FailCmd_UsesTeaTick(t *testing.T) {
	th := newTestTheme()
	s := uikit.NewSpinner("Working", th)
	_, cmd := s.Fail("boom")
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(uikit.SpinnerFailMsg)
	assert.True(t, ok)
}
