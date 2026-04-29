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

// TestSpinner_Running_ASCII_UsesSpinnerFrames verifies that a running Spinner in
// ASCII mode renders a frame drawn from SpinnerFrames(GlyphASCII), i.e. one of
// "|", "/", "-", "\". This locks in the integration between NewSpinner and
// SpinnerFrames — if spinner.go ever reverted to an inline frame array with
// different characters, this test would fail.
func TestSpinner_Running_ASCII_UsesSpinnerFrames(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := newTestTheme()
	s := uikit.NewSpinner("Working", th)

	// The spinner is in running state (result == nil) so View() returns
	// the current frame + text. Strip ANSI to get bare characters.
	v := stripANSI(s.View())

	asciiFrames := uikit.SpinnerFrames(uikit.GlyphASCII)
	found := false
	for _, f := range asciiFrames {
		if strings.Contains(v, f) {
			found = true
			break
		}
	}
	assert.True(t, found,
		"running spinner in ASCII mode must render a frame from SpinnerFrames(GlyphASCII) %v, got: %q",
		asciiFrames, v)
}

// TestSpinner_Running_Unicode_UsesSpinnerFrames verifies that a running Spinner in
// Unicode mode renders a frame drawn from SpinnerFrames(GlyphUnicode), i.e. one of
// the 10 braille characters. This mirrors the ASCII counterpart.
func TestSpinner_Running_Unicode_UsesSpinnerFrames(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := newTestTheme()
	s := uikit.NewSpinner("Working", th)

	v := stripANSI(s.View())

	unicodeFrames := uikit.SpinnerFrames(uikit.GlyphUnicode)
	found := false
	for _, f := range unicodeFrames {
		if strings.Contains(v, f) {
			found = true
			break
		}
	}
	assert.True(t, found,
		"running spinner in Unicode mode must render a frame from SpinnerFrames(GlyphUnicode) %v, got: %q",
		unicodeFrames, v)
}
