package uikit

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// SpinnerDoneMsg is emitted after Spinner.Done's hold period expires.
type SpinnerDoneMsg struct{ Text string }

// SpinnerFailMsg is emitted after Spinner.Fail's hold period expires.
type SpinnerFailMsg struct{ Err string }

// SpinnerCancelledMsg is emitted immediately on Spinner.Cancel.
type SpinnerCancelledMsg struct{}

// spinnerResolution holds the terminal state of a resolved spinner.
type spinnerResolution struct {
	glyph GlyphRole
	text  string
	role  Role
}

// Spinner is the TUI spinner primitive. It wraps bubbles/spinner with terminal
// states for Done, Fail, and Cancel — matching cliout.Spinner's contract.
// While running the bubbles spinner frame animates; once resolved the frame is
// replaced by a static glyph (✓ / ✗) and the hold timer emits the final message.
type Spinner struct {
	s      spinner.Model
	text   string
	result *spinnerResolution
	theme  theme.Theme
}

// NewSpinner creates a running Spinner that displays text while waiting.
// ASCII mode is resolved via ActiveMode(); use SetModeForTest in tests.
func NewSpinner(text string, th theme.Theme) *Spinner {
	m := spinner.New()
	// Use rotating-bar frames in ASCII mode; Dot (⣾) in unicode.
	if ActiveMode() == GlyphASCII {
		m.Spinner = spinner.Line // |/-\ frames
	} else {
		m.Spinner = spinner.Dot
	}
	m.Style = lipgloss.NewStyle().Foreground(th.Accent())
	return &Spinner{s: m, text: text, theme: th}
}

// Init returns the first tick cmd to start the embedded bubbles spinner.
func (s *Spinner) Init() tea.Cmd { return s.s.Tick }

// Update advances the spinner frame. Once the spinner has been resolved via
// Done, Fail, or Cancel, all further messages are silently dropped.
func (s *Spinner) Update(msg tea.Msg) (*Spinner, tea.Cmd) {
	if s.result != nil {
		return s, nil
	}
	var cmd tea.Cmd
	s.s, cmd = s.s.Update(msg)
	return s, cmd
}

// View renders the spinner. While running: animated frame + muted text.
// After Done: ✓ (Success colour) + muted text.
// After Fail: ✗ (Error colour) + muted text.
// After Cancel: empty string.
func (s *Spinner) View() string {
	if s.result != nil {
		if s.result.text == "" && s.result.glyph == "" {
			// Cancel: cleared immediately.
			return ""
		}
		g := GlyphFor(s.result.glyph, ActiveMode())
		gl := lipgloss.NewStyle().Foreground(ColourFor(s.result.role, s.theme)).Render(g)
		tx := lipgloss.NewStyle().Foreground(s.theme.TextMuted()).Render(s.result.text)
		return gl + " " + tx
	}
	frame := s.s.View()
	tx := lipgloss.NewStyle().Foreground(s.theme.TextMuted()).Render(s.text)
	return frame + " " + tx
}

// Done resolves the spinner to a ✓ (Success) glyph; after 1.2 s the returned
// tea.Cmd fires SpinnerDoneMsg. The cmd uses tea.Tick so it runs through Bubble
// Tea's scheduler — time.Sleep is never used.
func (s *Spinner) Done(text string) (*Spinner, tea.Cmd) {
	s.result = &spinnerResolution{glyph: GlyphSuccess, text: text, role: RoleSuccess}
	return s, tea.Tick(1200*time.Millisecond, func(time.Time) tea.Msg {
		return SpinnerDoneMsg{Text: text}
	})
}

// Fail resolves the spinner to a ✗ (Error) glyph; after 2 s the returned
// tea.Cmd fires SpinnerFailMsg. Uses tea.Tick — no time.Sleep.
func (s *Spinner) Fail(text string) (*Spinner, tea.Cmd) {
	s.result = &spinnerResolution{glyph: GlyphError, text: text, role: RoleError}
	return s, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return SpinnerFailMsg{Err: text}
	})
}

// Cancel clears the spinner immediately without displaying a final glyph, then
// fires SpinnerCancelledMsg synchronously.
func (s *Spinner) Cancel() (*Spinner, tea.Cmd) {
	// Empty glyph + empty text signals "cleared" to View().
	s.result = &spinnerResolution{}
	s.text = ""
	return s, func() tea.Msg { return SpinnerCancelledMsg{} }
}
